package integration

import (
	"strings"
	"testing"

	"spotifystego/internal/decoder"
	"spotifystego/internal/encoder"
)

// deterministicPool creates 260 tracks — 10 per letter a-z.
// Each track's title contains only that letter repeated, so ExtractFromTrack
// always yields that letter regardless of RNG position.
func deterministicPool() []encoder.Track {
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

func TestEncodeDecodeRoundTrip(t *testing.T) {
	pool := deterministicPool()
	keywords := [3]string{"cat", "dog", "bird"}
	message := "helloworld"

	playlist, err := encoder.EncodeMessage(pool, message, keywords, 20)
	if err != nil {
		t.Fatalf("EncodeMessage: %v", err)
	}
	if len(playlist) != 20 {
		t.Fatalf("expected playlist length 20, got %d", len(playlist))
	}

	decoded, _, err := decoder.DecodePlaylist(playlist, keywords)
	if err != nil {
		t.Fatalf("DecodePlaylist: %v", err)
	}
	if decoded != message {
		t.Fatalf("round-trip failed: encoded %q, decoded %q", message, decoded)
	}
}

func TestEncodeDecodeNormalization(t *testing.T) {
	pool := deterministicPool()
	keywords := [3]string{"cat", "dog", "bird"}
	// "Hello World" normalizes to "helloworld"
	playlist, err := encoder.EncodeMessage(pool, "Hello World", keywords, 20)
	if err != nil {
		t.Fatalf("EncodeMessage: %v", err)
	}
	decoded, _, err := decoder.DecodePlaylist(playlist, keywords)
	if err != nil {
		t.Fatalf("DecodePlaylist: %v", err)
	}
	if decoded != "helloworld" {
		t.Fatalf("expected 'helloworld', got %q", decoded)
	}
}

// TestKeywordSensitivity verifies that different keywords produce different PRNG
// sequences, causing different tracks to be selected for the same message.
// (Wrong-keyword decoding is not tested against a deterministic pool because
// that pool always yields the same letter regardless of PRNG state — by design.)
func TestKeywordSensitivity(t *testing.T) {
	kw1 := [3]string{"a", "b", "c"}
	kw2 := [3]string{"x", "y", "z"}
	rng1 := encoder.DeriveRNG(kw1)
	rng2 := encoder.DeriveRNG(kw2)
	// Two different keyword sets must produce different PRNG sequences
	if rng1.SameState(rng2) {
		t.Fatal("different keywords produced the same PRNG state")
	}
}
