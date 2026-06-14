package encoder_test

import (
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
