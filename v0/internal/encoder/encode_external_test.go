package encoder_test

import (
	"fmt"
	"strings"
	"testing"

	"spotifystego/internal/decoder"
	"spotifystego/internal/encoder"
)

// deterministicPoolExt mirrors the pool helper in encode_test.go for external tests.
func deterministicPoolExt() []encoder.Track {
	camelotCodes := []string{"1A", "2A", "3A", "4A", "5A", "6A", "7A", "8A", "9A", "10A"}
	var pool []encoder.Track
	for i := 0; i < 26; i++ {
		letter := string(rune('a' + i))
		title := strings.Repeat(letter+" ", 10)
		for j := 0; j < 10; j++ {
			pool = append(pool, encoder.Track{
				ID:          string(rune('a'+i)) + string(rune('0'+j)),
				Title:       title,
				Artist:      "Test",
				CamelotCode: camelotCodes[j],
				BPM:         float64(80 + j*10),
			})
		}
	}
	return pool
}

// multiWordPool builds 260 tracks covering every letter a-z with two title
// patterns per letter to exercise both count=1 and count=2 extraction:
//
//   count=1 tracks (j<5): "a a a a a a a a a a " — 10 single-letter words,
//     cap3=1, count=1+(10%1)=1. Always yields the target letter.
//
//   count=2 tracks (j>=5): "aaa a" — one 3-letter word + one 1-letter word,
//     cap3=3, count=1+(4%3)=2. Both position draws pick from 4 identical
//     letters, always yielding the target letter twice.
//
// This ensures the constrained encoder can find 2-byte consumers when two
// consecutive payload bytes are the same (e.g. "aa" or "ll").
func multiWordPool() []encoder.Track {
	camelotCodes := []string{"1A", "2A", "3A", "4A", "5A", "6A", "7A", "8A", "9A", "10A"}
	var pool []encoder.Track
	for i := 0; i < 26; i++ {
		ltr := string(rune('a' + i))
		for j := 0; j < 10; j++ {
			var title string
			if j < 5 {
				title = strings.Repeat(ltr+" ", 10) // count=1
			} else {
				title = ltr + ltr + ltr + " " + ltr // count=2
			}
			pool = append(pool, encoder.Track{
				ID:          fmt.Sprintf("%s%d", ltr, j),
				Title:       title,
				Artist:      "Test",
				CamelotCode: camelotCodes[j%len(camelotCodes)],
				BPM:         float64(80 + j*10),
			})
		}
	}
	return pool
}

func TestRoundTripMultiLetter(t *testing.T) {
	pool := multiWordPool()
	kw := [3]string{"red", "green", "blue"}

	playlist, err := encoder.EncodeMessage(pool, "hello", kw, 20)
	if err != nil {
		t.Fatalf("EncodeMessage: %v", err)
	}
	decoded, _, err := decoder.DecodePlaylist(playlist, kw)
	if err != nil {
		t.Fatalf("DecodePlaylist: %v", err)
	}
	if decoded != "hello" {
		t.Errorf("got %q, want %q", decoded, "hello")
	}
}

func TestEncodeEmptyMessage(t *testing.T) {
	pool := deterministicPoolExt()
	kw := [3]string{"x", "y", "z"}

	playlist, err := encoder.EncodeMessage(pool, "", kw, 10)
	if err != nil {
		t.Fatalf("EncodeMessage: %v", err)
	}
	if len(playlist) != 10 {
		t.Fatalf("expected playlist length 10, got %d", len(playlist))
	}

	decoded, _, err := decoder.DecodePlaylist(playlist, kw)
	if err != nil {
		t.Fatalf("DecodePlaylist: %v", err)
	}
	if decoded != "" {
		t.Fatalf("expected empty message, got %q", decoded)
	}
}
