// Package main is the HTMX web server for the Spotify steganography system.
// It handles encoding, decoding, Spotify OAuth, and playlist creation.
package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"

	"spotifystego/internal/camelot"
	"spotifystego/internal/database"
	"spotifystego/internal/decoder"
	"spotifystego/internal/encoder"
	"spotifystego/internal/spotify"
)

var tmpl *template.Template

func main() {
	_ = godotenv.Load()

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "tracks.db"
	}

	db, err := database.Open(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	tmpl = template.Must(template.New("").Funcs(template.FuncMap{
		"inc":    func(i int) int { return i + 1 },
		"string": func(b []byte) string { return string(b) },
	}).ParseGlob("templates/*.html"))

	var sp *spotify.Client
	if id, secret := os.Getenv("SPOTIFY_CLIENT_ID"), os.Getenv("SPOTIFY_CLIENT_SECRET"); id != "" && secret != "" {
		sp = spotify.New(id, secret)
		if authURL, apiBase := os.Getenv("SPOTIFY_AUTH_URL"), os.Getenv("SPOTIFY_API_BASE"); authURL != "" && apiBase != "" {
			sp.SetBaseURLs(authURL, apiBase)
			log.Printf("Using mock Spotify backend: auth=%s api=%s", authURL, apiBase)
		}
	}

	redirectURI := os.Getenv("SPOTIFY_REDIRECT_URI") // TODO: set SPOTIFY_REDIRECT_URI in .env before testing OAuth

	srv := &server{
		db:          db,
		sp:          sp,
		redirectURI: redirectURI,
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", srv.handleIndex)
	mux.HandleFunc("/encode", srv.handleEncode)
	mux.HandleFunc("/decode", srv.handleDecode)
	mux.HandleFunc("/auth/spotify/login", srv.handleSpotifyLogin)
	mux.HandleFunc("/auth/spotify/callback", srv.handleSpotifyCallback)
	mux.HandleFunc("/encode/save", srv.handleSave)

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, logging(mux)))
}

// server holds shared server state.
type server struct {
	db          *sql.DB
	sp          *spotify.Client
	redirectURI string // TODO: set SPOTIFY_REDIRECT_URI in .env before testing OAuth
	lastEncode  *encodeResult
	lastEncMu   sync.Mutex
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

// handleIndex serves the main page.
func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	renderTemplate(w, "base.html", nil)
}

// encodeResult holds the output of a successful encode operation.
type encodeResult struct {
	Tracks             []encoder.Track
	Message            string
	MsgLen             int
	Genre              string
	PlaylistLen        int
	TotalDuration      string
	AudioDataAvailable bool    // false when pool uses stub BPM/Camelot data
	MusicalScore       float64
	AvgCamelotScore    float64 // mean Camelot score across transitions
	AvgBPMScore        float64 // mean BPM score across transitions
	ScoreLabel         string  // "Excellent" | "Good" | "Fair" | "Needs work"
	ScoreClass         string  // "excellent" | "good" | "fair" | "needs-work"
	WheelSVG           template.HTML
	BPMSVG             template.HTML
	Error              string
}

// handleEncode encodes a message into a Spotify playlist and renders the results.
func (s *server) handleEncode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", 400)
		return
	}

	message := r.FormValue("message")
	genre := r.FormValue("genre")
	k1 := r.FormValue("k1")
	k2 := r.FormValue("k2")
	k3 := r.FormValue("k3")
	keywords := [3]string{k1, k2, k3}

	result := encodeResult{Message: message, MsgLen: len(message)}

	dbTracks, err := database.GetTracksByGenre(s.db, genre)
	if err != nil || len(dbTracks) == 0 {
		result.Error = fmt.Sprintf("no tracks found for genre %q — run the indexer first", genre)
		renderTemplate(w, "encode-results.html", result)
		return
	}

	pool := dbTracksToEncoder(dbTracks)
	playlist, err := encoder.EncodeMessage(pool, message, keywords, 20)
	if err != nil {
		result.Error = err.Error()
		renderTemplate(w, "encode-results.html", result)
		return
	}

	result.Tracks = playlist
	result.Genre = genre
	result.PlaylistLen = len(playlist)
	result.TotalDuration = formatDuration(playlist)
	result.AudioDataAvailable = poolHasAudioData(pool)
	if result.AudioDataAvailable {
		result.MusicalScore, result.AvgCamelotScore, result.AvgBPMScore = computeMusicalScore(playlist)
		result.ScoreLabel, result.ScoreClass = musicalityLabel(result.MusicalScore)
		result.WheelSVG = template.HTML(camelot.RenderWheelSVG(toCalWheelTracks(playlist)))
		result.BPMSVG = template.HTML(encoder.RenderBPMGraphSVG(playlist))
	}

	s.lastEncMu.Lock()
	s.lastEncode = &result
	s.lastEncMu.Unlock()

	renderTemplate(w, "encode-results.html", result)
}

type decodeResult struct {
	Message     string
	Extractions []decoder.TrackExtraction
	Error       string
}

// handleDecode decodes a playlist of track titles and renders the hidden message.
func (s *server) handleDecode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", 400)
		return
	}

	trackList := r.FormValue("tracks")
	k1 := r.FormValue("k1")
	k2 := r.FormValue("k2")
	k3 := r.FormValue("k3")
	keywords := [3]string{k1, k2, k3}

	result := decodeResult{}
	lines := strings.Split(strings.TrimSpace(trackList), "\n")
	var tracks []encoder.Track
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		t := encoder.Track{ID: fmt.Sprintf("dec-%d", i), Title: strings.TrimSpace(parts[0])}
		if len(parts) == 2 {
			t.Artist = strings.TrimSpace(parts[1])
		}
		tracks = append(tracks, t)
	}
	if len(tracks) == 0 {
		result.Error = "no tracks provided"
		renderTemplate(w, "decode-results.html", result)
		return
	}

	message, extractions, err := decoder.DecodePlaylist(tracks, keywords)
	if err != nil {
		result.Error = err.Error()
		renderTemplate(w, "decode-results.html", result)
		return
	}
	result.Message = message
	result.Extractions = extractions
	renderTemplate(w, "decode-results.html", result)
}

// handleSpotifyLogin redirects the user to Spotify's PKCE authorization page.
func (s *server) handleSpotifyLogin(w http.ResponseWriter, r *http.Request) {
	if s.sp == nil {
		http.Error(w, "Spotify not configured", 503)
		return
	}
	// TODO: set SPOTIFY_REDIRECT_URI in .env before testing OAuth
	authURL, err := s.sp.AuthorizeURL(s.redirectURI)
	if err != nil {
		http.Error(w, "auth error: "+err.Error(), 500)
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

// handleSpotifyCallback completes the OAuth exchange after Spotify redirects back.
func (s *server) handleSpotifyCallback(w http.ResponseWriter, r *http.Request) {
	if s.sp == nil {
		http.Error(w, "Spotify not configured", 503)
		return
	}
	state := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	// TODO: set SPOTIFY_REDIRECT_URI in .env before testing OAuth
	if err := s.sp.ExchangeCode(state, code, s.redirectURI); err != nil {
		http.Error(w, "OAuth callback error: "+err.Error(), 400)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// saveResult is the data model for the save-result HTMX partial.
type saveResult struct {
	ConnectSpotify bool
	Error          string
	PlaylistURL    string
}

// handleSave creates a Spotify playlist from the last encode result (HTMX partial).
func (s *server) handleSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}

	if s.sp == nil || !s.sp.IsAuthenticated() {
		renderTemplate(w, "save-result.html", saveResult{ConnectSpotify: true})
		return
	}

	s.lastEncMu.Lock()
	enc := s.lastEncode
	s.lastEncMu.Unlock()

	if enc == nil {
		renderTemplate(w, "save-result.html", saveResult{Error: "No encode result — run Encode first."})
		return
	}

	userID, err := s.sp.GetCurrentUserID()
	if err != nil {
		renderTemplate(w, "save-result.html", saveResult{Error: err.Error()})
		return
	}

	public := os.Getenv("SPOTIFY_PLAYLIST_PRIVATE") != "true"
	name := fmt.Sprintf("Stego: %s %s", enc.Genre, time.Now().Format("2006-01-02"))
	playlistID, err := s.sp.CreatePlaylist(userID, name, public)
	if err != nil {
		renderTemplate(w, "save-result.html", saveResult{Error: err.Error()})
		return
	}

	trackIDs := make([]string, len(enc.Tracks))
	for i, t := range enc.Tracks {
		trackIDs[i] = t.ID
	}
	if err := s.sp.AddTracksToPlaylist(playlistID, trackIDs); err != nil {
		renderTemplate(w, "save-result.html", saveResult{Error: err.Error()})
		return
	}

	renderTemplate(w, "save-result.html", saveResult{
		PlaylistURL: "https://open.spotify.com/playlist/" + playlistID,
	})
}

// poolHasAudioData returns true if the pool contains varied BPM and Camelot data.
// All-identical values indicate a stub provider — scoring would be meaningless.
func poolHasAudioData(pool []encoder.Track) bool {
	if len(pool) < 2 {
		return false
	}
	first := pool[0]
	for _, t := range pool[1:] {
		if t.BPM != first.BPM || t.CamelotCode != first.CamelotCode {
			return true
		}
	}
	return false
}

func dbTracksToEncoder(dbTracks []database.Track) []encoder.Track {
	pool := make([]encoder.Track, len(dbTracks))
	for i, t := range dbTracks {
		pool[i] = encoder.Track{
			ID: t.ID, Title: t.Title, Artist: t.Artist, Genre: t.Genre,
			DurationMS: t.DurationMS, BPM: t.Tempo, KeyOf: t.KeyOf, CamelotCode: t.CamelotCode,
		}
	}
	return pool
}

func toCalWheelTracks(tracks []encoder.Track) []camelot.Track {
	out := make([]camelot.Track, len(tracks))
	for i, t := range tracks {
		out[i] = camelot.Track{CamelotCode: t.CamelotCode}
	}
	return out
}

func formatDuration(tracks []encoder.Track) string {
	total := 0
	for _, t := range tracks {
		total += t.DurationMS
	}
	secs := total / 1000
	return fmt.Sprintf("%d:%02d", secs/60, secs%60)
}

// computeMusicalScore returns the musical coherence score (0–100) for a playlist
// and the per-component averages per transition.
// Formula: mean((camelotScore + bpmScore) / 15) × 100 across consecutive pairs.
func computeMusicalScore(tracks []encoder.Track) (score, avgCamelot, avgBPM float64) {
	if len(tracks) < 2 {
		return 0, 0, 0
	}
	totalCamelot, totalBPM := 0.0, 0.0
	n := float64(len(tracks) - 1)
	for i := 1; i < len(tracks); i++ {
		prev := &tracks[i-1]
		cur := &tracks[i]
		totalCamelot += float64(camelot.Score(prev.CamelotCode, cur.CamelotCode))
		totalBPM += bpmScoreLocal(prev.BPM, cur.BPM)
	}
	avgCamelot = totalCamelot / n
	avgBPM = totalBPM / n
	score = (totalCamelot + totalBPM) / (n * 15.0) * 100
	return
}

// musicalityLabel maps a 0–100 score to a human label and CSS class.
func musicalityLabel(score float64) (label, class string) {
	switch {
	case score >= 80:
		return "Excellent", "excellent"
	case score >= 60:
		return "Good", "good"
	case score >= 40:
		return "Fair", "fair"
	default:
		return "Needs work", "needs-work"
	}
}

// bpmScoreLocal returns a BPM compatibility score in [0,5].
// Duplicates bpmScore in internal/encoder/greedy.go; do not refactor.
func bpmScoreLocal(a, b float64) float64 {
	if a == 0 || b == 0 {
		return 0
	}
	diff := math.Abs(a-b) / a
	if diff <= 0.06 {
		return 5
	}
	if diff >= 0.20 {
		return 0
	}
	return 5 * (1 - (diff-0.06)/0.14)
}
