package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"

	"spotifystego/internal/audio"
	"spotifystego/internal/database"
	"spotifystego/internal/spotify"
)

func main() {
	genre := flag.String("genre", "pop", "Genre to index")
	dbPath := flag.String("db", "tracks.db", "SQLite database path")
	flag.Parse()

	_ = godotenv.Load()

	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		log.Fatal("SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set")
	}

	db, err := database.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	sp := spotify.New(clientID, clientSecret)
	if authURL, apiBase := os.Getenv("SPOTIFY_AUTH_URL"), os.Getenv("SPOTIFY_API_BASE"); authURL != "" && apiBase != "" {
		sp.SetBaseURLs(authURL, apiBase)
		log.Printf("Using mock backend: auth=%s api=%s", authURL, apiBase)
	}
	provider := audio.StubProvider{}

	log.Printf("Searching tracks for genre %q...", *genre)
	tracks, err := sp.GetTracksByGenre(*genre)
	if err != nil {
		log.Fatalf("get tracks: %v", err)
	}
	log.Printf("Found %d tracks for genre %q", len(tracks), *genre)

	indexed, skipped, errCount := 0, 0, 0
	for _, t := range tracks {
		if t.ID == "" || t.Title == "" {
			skipped++
			continue
		}
		audioData, err := provider.GetAudioData(t.Title, t.Artist)
		if err != nil {
			log.Printf("  audio data error for %q: %v", t.Title, err)
			errCount++
			continue
		}
		err = database.UpsertTrack(db, database.Track{
			ID:          t.ID,
			Title:       t.Title,
			Artist:      t.Artist,
			Genre:       *genre,
			DurationMS:  t.DurationMS,
			Tempo:       audioData.BPM,
			KeyOf:       audioData.KeyOf,
			CamelotCode: audioData.CamelotCode,
		})
		if err != nil {
			log.Printf("  upsert error for %q: %v", t.Title, err)
			errCount++
			continue
		}
		indexed++
		if indexed%50 == 0 {
			fmt.Printf("Progress: %d indexed, %d skipped, %d errors\n", indexed, skipped, errCount)
		}
	}
	fmt.Printf("Done: %d indexed, %d skipped, %d errors\n", indexed, skipped, errCount)
}
