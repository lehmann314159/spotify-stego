package decoder

import (
	"strings"
	"testing"

	"spotifystego/internal/encoder"
)

// deterministicPool creates 260 tracks — 10 per letter a-z.
// Each track's title contains only that letter repeated, so ExtractFromTrack
// always yields that letter regardless of RNG position. Copied from encode_test.go.
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

func TestDecodeNilPlaylist(t *testing.T) {
	kw := [3]string{"a", "b", "c"}
	_, _, err := DecodePlaylist(nil, kw)
	if err == nil {
		t.Fatal("expected error for nil playlist, got nil")
	}
}

func TestDecodeTooFewLetters(t *testing.T) {
	kw := [3]string{"a", "b", "c"}
	// Two single-letter-word tracks each yield exactly 1 letter (cap3=1), total 2 < 3.
	tracks := []Track{
		{ID: "t1", Title: "a"},
		{ID: "t2", Title: "b"},
	}
	_, _, err := DecodePlaylist(tracks, kw)
	if err == nil {
		t.Fatal("expected error for too few letters, got nil")
	}
}

func TestDecodeCorruptLengthPrefix(t *testing.T) {
	kw := [3]string{"a", "b", "c"}
	// Title "z z z z z z z z z z" (10 single-letter 'z' words):
	// cap3=1 → always 1 letter per track → 'z'. Three tracks → allLetters="zzz".
	// decodeLength("zzz") = 17575; only 0 letters remain after prefix → error.
	zTitle := strings.Repeat("z ", 10)
	tracks := []Track{
		{ID: "z0", Title: zTitle},
		{ID: "z1", Title: zTitle},
		{ID: "z2", Title: zTitle},
	}
	_, _, err := DecodePlaylist(tracks, kw)
	if err == nil {
		t.Fatal("expected error for corrupt length prefix, got nil")
	}
	errStr := err.Error()
	// Error should mention both the expected count and the available count.
	if !strings.Contains(errStr, "17575") {
		t.Errorf("error %q should mention the decoded length 17575", errStr)
	}
}

func TestDecodeEmptyMessage(t *testing.T) {
	pool := deterministicPool()
	kw := [3]string{"cat", "dog", "bird"}

	playlist, err := encoder.EncodeMessage(pool, "", kw, 10)
	if err != nil {
		t.Fatalf("EncodeMessage: %v", err)
	}

	decoded, _, err := DecodePlaylist(playlist, kw)
	if err != nil {
		t.Fatalf("DecodePlaylist: %v", err)
	}
	if decoded != "" {
		t.Fatalf("expected empty message, got %q", decoded)
	}
}
