package camelot

import "testing"

var expectedCodes = map[string]string{
	"Am":  "1A",
	"Em":  "2A",
	"Bm":  "3A",
	"F#m": "4A",
	"C#m": "5A",
	"G#m": "6A",
	"D#m": "7A",
	"A#m": "8A",
	"Fm":  "9A",
	"Cm":  "10A",
	"Gm":  "11A",
	"Dm":  "12A",
	"C":   "8B",
	"G":   "9B",
	"D":   "10B",
	"A":   "11B",
	"E":   "12B",
	"B":   "1B",
	"F#":  "2B",
	"Db":  "3B",
	"Ab":  "4B",
	"Eb":  "5B",
	"Bb":  "6B",
	"F":   "7B",
}

func TestAllKeyMappings(t *testing.T) {
	for key, want := range expectedCodes {
		got, err := KeyToCode(key)
		if err != nil {
			t.Errorf("KeyToCode(%q): unexpected error: %v", key, err)
			continue
		}
		if got != want {
			t.Errorf("KeyToCode(%q): got %q, want %q", key, got, want)
		}
	}
}

func TestUnknownKey(t *testing.T) {
	_, err := KeyToCode("Xm")
	if err == nil {
		t.Error("expected error for unknown key, got nil")
	}
}

func TestScoreSameKey(t *testing.T) {
	if s := Score("8B", "8B"); s != ScoreSameKey {
		t.Errorf("same key: got %d, want %d", s, ScoreSameKey)
	}
}

func TestScoreRelativeMajorMinor(t *testing.T) {
	// 8A and 8B share the same number — relative major/minor
	if s := Score("8A", "8B"); s != ScoreRelative {
		t.Errorf("relative: got %d, want %d", s, ScoreRelative)
	}
}

func TestScoreNeighbor(t *testing.T) {
	// 8B → 9B: +1 on wheel, same letter
	if s := Score("8B", "9B"); s != ScoreNeighbor {
		t.Errorf("neighbor: got %d, want %d", s, ScoreNeighbor)
	}
}

func TestScoreWrapAround(t *testing.T) {
	// 1A and 12A are neighbors (wrap)
	if s := Score("1A", "12A"); s != ScoreNeighbor {
		t.Errorf("wrap-around neighbor: got %d, want %d", s, ScoreNeighbor)
	}
}

func TestScoreNoRelationship(t *testing.T) {
	if s := Score("1A", "6B"); s != ScoreNoRelationship {
		t.Errorf("no relation: got %d, want %d", s, ScoreNoRelationship)
	}
}
