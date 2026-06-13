package main

import (
	"flag"
	"fmt"
	"log"

	"spotifystego/internal/database"
	"spotifystego/internal/encoder"
)

func main() {
	message := flag.String("message", "", "Message to encode (required)")
	genre := flag.String("genre", "pop", "Genre pool to draw from")
	k1 := flag.String("k1", "", "Keyword 1 (required)")
	k2 := flag.String("k2", "", "Keyword 2 (required)")
	k3 := flag.String("k3", "", "Keyword 3 (required)")
	dbPath := flag.String("db", "tracks.db", "SQLite database path")
	length := flag.Int("length", 30, "Target playlist length")
	flag.Parse()

	if *message == "" || *k1 == "" || *k2 == "" || *k3 == "" {
		log.Fatal("--message, --k1, --k2, --k3 are required")
	}

	db, err := database.Open(*dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	dbTracks, err := database.GetTracksByGenre(db, *genre)
	if err != nil {
		log.Fatalf("get tracks: %v", err)
	}

	pool := make([]encoder.Track, len(dbTracks))
	for i, t := range dbTracks {
		pool[i] = encoder.Track{
			ID: t.ID, Title: t.Title, Artist: t.Artist, Genre: t.Genre,
			DurationMS: t.DurationMS, BPM: t.Tempo, KeyOf: t.KeyOf, CamelotCode: t.CamelotCode,
		}
	}

	keywords := [3]string{*k1, *k2, *k3}
	playlist, err := encoder.EncodeMessage(pool, *message, keywords, *length)
	if err != nil {
		log.Fatalf("encode: %v", err)
	}

	fmt.Printf("%-4s %-40s %-25s %-6s %s\n", "#", "Title", "Artist", "Camelot", "BPM")
	fmt.Println("---------------------------------------------------------------------------------------------")
	for i, t := range playlist {
		fmt.Printf("%-4d %-40s %-25s %-6s %.0f\n", i+1, truncate(t.Title, 40), truncate(t.Artist, 25), t.CamelotCode, t.BPM)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
