package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"spotifystego/internal/decoder"
	"spotifystego/internal/encoder"
)

func main() {
	k1 := flag.String("k1", "", "Keyword 1 (required)")
	k2 := flag.String("k2", "", "Keyword 2 (required)")
	k3 := flag.String("k3", "", "Keyword 3 (required)")
	flag.Parse()

	if *k1 == "" || *k2 == "" || *k3 == "" {
		log.Fatal("--k1, --k2, --k3 are required")
	}

	// Read track titles from stdin (one per line: "title|artist" or just "title")
	var tracks []encoder.Track
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		t := encoder.Track{Title: parts[0]}
		if len(parts) == 2 {
			t.Artist = parts[1]
		}
		t.ID = fmt.Sprintf("cli-%d", len(tracks))
		tracks = append(tracks, t)
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("read stdin: %v", err)
	}
	if len(tracks) == 0 {
		log.Fatal("no tracks provided on stdin")
	}

	keywords := [3]string{*k1, *k2, *k3}
	message, extractions, err := decoder.DecodePlaylist(tracks, keywords)
	if err != nil {
		log.Fatalf("decode: %v", err)
	}

	fmt.Printf("Decoded message: %q\n\n", message)
	fmt.Println("Per-track extraction:")
	for _, e := range extractions {
		fmt.Printf("  %-40s → %q\n", truncate(e.Track.Title, 40), e.Letters)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
