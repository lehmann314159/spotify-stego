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

func TestRoundTripMaxCapacity(t *testing.T) {
	pool := deterministicPool()
	kw := [3]string{"cat", "dog", "bird"}
	// 17-char message → prefix "aar" + "abcdefghijklmnopq" = 20 chars = targetLength exactly.
	message := "abcdefghijklmnopq"

	playlist, err := encoder.EncodeMessage(pool, message, kw, 20)
	if err != nil {
		t.Fatalf("EncodeMessage: %v", err)
	}
	if len(playlist) != 20 {
		t.Fatalf("expected playlist length 20, got %d", len(playlist))
	}

	decoded, _, err := decoder.DecodePlaylist(playlist, kw)
	if err != nil {
		t.Fatalf("DecodePlaylist: %v", err)
	}
	if decoded != message {
		t.Fatalf("round-trip failed: want %q, got %q", message, decoded)
	}
}

// rotatedPool creates 260 tracks (10 per letter a-z) whose titles are
// 10-letter rotations of the alphabet (cap3=1, count=1 always).
// Unlike deterministicPool, different PRNG positions yield different letters —
// so wrong keywords extract different content, making TestWrongKeywordsDecode reliable.
func rotatedPool() []encoder.Track {
	camelotCodes := []string{"1A", "2A", "3A", "4A", "5A", "6A", "7A", "8A", "9A", "10A"}
	var pool []encoder.Track
	for i := 0; i < 26; i++ {
		// Title: 10 single-letter words starting at letter i, wrapping around.
		words := make([]string, 10)
		for k := 0; k < 10; k++ {
			words[k] = string(rune('a' + (i+k)%26))
		}
		title := strings.Join(words, " ")
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

func TestWrongKeywordsDecode(t *testing.T) {
	pool := rotatedPool()
	rightKW := [3]string{"cat", "dog", "bird"}
	wrongKW := [3]string{"x", "y", "z"}
	message := "hello"

	playlist, err := encoder.EncodeMessage(pool, message, rightKW, 20)
	if err != nil {
		t.Fatalf("EncodeMessage: %v", err)
	}

	decoded, _, err := decoder.DecodePlaylist(playlist, wrongKW)
	// Either an error occurs OR the decoded content differs from the original.
	if err == nil && decoded == message {
		t.Fatal("wrong keywords decoded to the correct message — pool or key derivation may be broken")
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
