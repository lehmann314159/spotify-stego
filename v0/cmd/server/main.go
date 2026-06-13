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
	"time"

	"github.com/joho/godotenv"

	"spotifystego/internal/camelot"
	"spotifystego/internal/database"
	"spotifystego/internal/decoder"
	"spotifystego/internal/encoder"
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

	srv := &server{db: db}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", srv.handleIndex)
	mux.HandleFunc("/encode", srv.handleEncode)
	mux.HandleFunc("/decode", srv.handleDecode)

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("Listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, logging(mux)))
}

type server struct {
	db *sql.DB
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

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	renderTemplate(w, "base.html", nil)
}

type encodeResult struct {
	Tracks        []encoder.Track
	Message       string
	MsgLen        int
	PlaylistLen   int
	TotalDuration string
	MusicalScore  float64
	WheelSVG      template.HTML
	BPMSVG        template.HTML
	Error         string
}

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
	result.PlaylistLen = len(playlist)
	result.TotalDuration = formatDuration(playlist)
	result.MusicalScore = computeMusicalScore(playlist)
	result.WheelSVG = template.HTML(camelot.RenderWheelSVG(toCalWheelTracks(playlist)))
	result.BPMSVG = template.HTML(encoder.RenderBPMGraphSVG(playlist))

	renderTemplate(w, "encode-results.html", result)
}

type decodeResult struct {
	Message     string
	Extractions []decoder.TrackExtraction
	Error       string
}

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

func computeMusicalScore(tracks []encoder.Track) float64 {
	if len(tracks) < 2 {
		return 0
	}
	total := 0.0
	for i := 1; i < len(tracks); i++ {
		prev := &tracks[i-1]
		cur := &tracks[i]
		cs := float64(camelot.Score(prev.CamelotCode, cur.CamelotCode))
		bs := bpmScoreLocal(prev.BPM, cur.BPM)
		total += (cs + bs) / 15.0
	}
	return total / float64(len(tracks)-1) * 100
}

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
