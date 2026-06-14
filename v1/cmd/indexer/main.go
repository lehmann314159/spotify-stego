package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/user/spotify-stego-v1/internal/audio"
	"github.com/user/spotify-stego-v1/internal/camelot"
	"github.com/user/spotify-stego-v1/internal/database"
	"github.com/user/spotify-stego-v1/internal/stego/core"
	"github.com/user/spotify-stego-v1/internal/spotify"
)

type indexerConfig struct {
	DBPath string
	Genre  string
	Limit  int
}

func runIndexer(cfg indexerConfig, sc *spotify.Client, provider audio.Provider, dbPath string) (indexed, skipped, errorCount int, err error) {
	db, err := database.Open(dbPath)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	playlists, err := sc.GetPopularPlaylistsByGenre(cfg.Genre)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("get playlists: %w", err)
	}

	limit := cfg.Limit
	if len(playlists) < limit {
		limit = len(playlists)
	}

	for _, pl := range playlists[:limit] {
		tracks, err := sc.GetPlaylistTracks(pl.ID)
		if err != nil {
			log.Printf("error getting tracks for playlist %s: %v", pl.Name, err)
			continue
		}

		for i, st := range tracks {
			ad, err := provider.GetAudioData(st.Title, st.Artist)
			if err == audio.ErrNotFound {
				skipped++
				continue
			}
			if err != nil {
				log.Printf("error getting audio data for %s - %s: %v", st.Title, st.Artist, err)
				errorCount++
				continue
			}

			camelotCode := ""
			if code, ok := camelot.KeyToCode(ad.KeyOf); ok {
				camelotCode = code
			}

			tk := core.Track{
				ID:          st.ID,
				Title:       st.Title,
				Artist:      st.Artist,
				Genre:       cfg.Genre,
				DurationMs:  st.DurationMs,
				Tempo:       ad.BPM,
				KeyOf:       ad.KeyOf,
				CamelotCode: camelotCode,
			}

			if err := database.UpsertTrack(db, tk); err != nil {
				log.Printf("error upserting track %s: %v", st.ID, err)
				errorCount++
				continue
			}

			indexed++

			if (i+1)%50 == 0 {
				log.Printf("indexed %d tracks so far...\n", indexed)
			}
		}
	}

	return indexed, skipped, errorCount, nil
}

func main() {
	dbPath := flag.String("db", "spotify-stego.db", "Path to SQLite database file")
	genre := flag.String("genre", "indie", "Spotify genre/category ID to index")
	limit := flag.Int("limit", 5, "Maximum number of playlists to process")
	flag.Parse()

	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		log.Fatalln("SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set")
	}

	sc := spotify.NewClient(clientID, clientSecret)

	var provider audio.Provider
	getSongBPMKey := os.Getenv("GETSONGBPM_API_KEY")
	if getSongBPMKey != "" {
		provider = audio.NewGetSongBPMProvider(getSongBPMKey)
	} else {
		provider = audio.StubProvider{}
		log.Println("GETSONGBPM_API_KEY not set, using StubProvider")
	}

	cfg := indexerConfig{
		DBPath: *dbPath,
		Genre:  *genre,
		Limit:  *limit,
	}

	indexed, skipped, errs, err := runIndexer(cfg, sc, provider, *dbPath)
	if err != nil {
		log.Fatalf("indexer failed: %v", err)
	}

	fmt.Printf("Done. Indexed: %d | Skipped: %d | Errors: %d\n", indexed, skipped, errs)
}
