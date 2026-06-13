package camelot

import "fmt"

// keyToCode maps standard key notation to Camelot wheel codes.
// A-minor = 1A, A♭-major = 1B, etc.
var keyToCode = map[string]string{
	// Minor keys (A codes)
	"Am":  "1A",
	"Em":  "2A",
	"Bm":  "3A",
	"F#m": "4A",
	"Gbm": "4A",
	"Dbm": "5A",
	"C#m": "5A",
	"Abm": "6A",
	"G#m": "6A",
	"Ebm": "7A",
	"D#m": "7A",
	"Bbm": "8A",
	"A#m": "8A",
	"Fm":  "9A",
	"Cm":  "10A",
	"Gm":  "11A",
	"Dm":  "12A",

	// Major keys (B codes)
	"C":  "8B",
	"G":  "9B",
	"D":  "10B",
	"A":  "11B",
	"E":  "12B",
	"B":  "1B",
	"F#": "2B",
	"Gb": "2B",
	"Db": "3B",
	"C#": "3B",
	"Ab": "4B",
	"G#": "4B",
	"Eb": "5B",
	"D#": "5B",
	"Bb": "6B",
	"A#": "6B",
	"F":  "7B",
}

// codeToNumber maps a Camelot code like "1A" to its wheel position (1-12).
func codeToNumber(code string) (int, string, error) {
	if len(code) < 2 {
		return 0, "", fmt.Errorf("invalid camelot code: %q", code)
	}
	letter := string(code[len(code)-1])
	var num int
	if _, err := fmt.Sscanf(code[:len(code)-1], "%d", &num); err != nil {
		return 0, "", fmt.Errorf("invalid camelot code: %q", code)
	}
	return num, letter, nil
}

// KeyToCode converts a key name (e.g. "Em", "C") to a Camelot code (e.g. "2A", "8B").
func KeyToCode(keyOf string) (string, error) {
	code, ok := keyToCode[keyOf]
	if !ok {
		return "", fmt.Errorf("unknown key: %q", keyOf)
	}
	return code, nil
}

// Compatibility scores between two Camelot codes.
const (
	ScoreSameKey       = 10
	ScoreRelative      = 8  // same number, different letter (e.g. 8A ↔ 8B)
	ScoreNeighbor      = 6  // ±1 on the wheel, same letter
	ScoreDiagonal      = 3  // ±1 on the wheel, different letter
	ScoreNoRelationship = 0
)

// Score returns the Camelot compatibility score between two codes.
func Score(a, b string) int {
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return ScoreSameKey
	}
	numA, letA, err := codeToNumber(a)
	if err != nil {
		return 0
	}
	numB, letB, err := codeToNumber(b)
	if err != nil {
		return 0
	}
	sameLetter := letA == letB
	diff := numA - numB
	if diff < 0 {
		diff = -diff
	}
	// Wrap around the 12-position wheel
	if diff > 6 {
		diff = 12 - diff
	}

	if diff == 0 {
		// Same number, different letter = relative major/minor
		return ScoreRelative
	}
	if diff == 1 && sameLetter {
		return ScoreNeighbor
	}
	if diff == 1 && !sameLetter {
		return ScoreDiagonal
	}
	return ScoreNoRelationship
}
