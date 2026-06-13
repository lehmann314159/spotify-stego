package encoder

import (
	"strings"
	"testing"
)

// deterministicPool creates 260 tracks — 10 per letter a-z.
// Each track's title contains only that letter repeated, so ExtractFromTrack
// always yields that letter regardless of RNG position.
// Multiple tracks per letter provide Camelot scoring diversity.
func deterministicPool() []Track {
	camelotCodes := []string{"1A", "2A", "3A", "4A", "5A", "6A", "7A", "8A", "9A", "10A"}
	var pool []Track
	for i := 0; i < 26; i++ {
		letter := string(rune('a' + i))
		// Title of 10 single-letter "words" so maxWordLen=1 → count always=1
		title := strings.Repeat(letter+" ", 10)
		for j := 0; j < 10; j++ {
			pool = append(pool, Track{
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

func TestConstrainedTrackYieldsNeeded(t *testing.T) {
	pool := deterministicPool()
	rng := DeriveRNG([3]string{"test", "one", "two"})

	needed := []byte{'h'}
	found, err := FindConstrained(nil, needed, pool, map[string]bool{}, rng)
	if err != nil {
		t.Fatalf("FindConstrained: %v", err)
	}

	// Commit rng for this track, then verify it yields 'h'
	verifyRng := rng.Clone()
	letters := ExtractFromTrack(verifyRng, found.Title)
	if len(letters) == 0 || letters[0] != 'h' {
		t.Fatalf("found track %q: expected first letter 'h', got %q", found.Title, letters)
	}
}

func TestFallbackToRelaxedCamelot(t *testing.T) {
	// Single-letter titles ensure 'z' is always reachable regardless of RNG.
	zTitle := strings.Repeat("z ", 10)
	pool := []Track{
		{ID: "z0", Title: zTitle, Artist: "A", CamelotCode: "1A", BPM: 120},
		{ID: "a0", Title: strings.Repeat("a ", 10), Artist: "B", CamelotCode: "1A", BPM: 120},
	}
	prev := &Track{CamelotCode: "12B", BPM: 200}
	rng := DeriveRNG([3]string{"f", "g", "h"})
	needed := []byte{'z'}

	found, err := FindConstrained(prev, needed, pool, map[string]bool{}, rng)
	if err != nil {
		t.Fatalf("expected to find 'z' track: %v", err)
	}
	if found == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestEncodeMessageLength(t *testing.T) {
	pool := deterministicPool()
	keywords := [3]string{"x", "y", "z"}
	message := "hi"
	playlist, err := EncodeMessage(pool, message, keywords, 15)
	if err != nil {
		t.Fatalf("EncodeMessage: %v", err)
	}
	if len(playlist) != 15 {
		t.Fatalf("expected 15 tracks, got %d", len(playlist))
	}
}
