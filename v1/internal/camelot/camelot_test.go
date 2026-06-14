package camelot

import "testing"

func TestKeyToCode(t *testing.T) {
	cases := []struct {
		key    string
		code   string
		found  bool
	}{
		{"Am", "1A", true},
		{"C", "1B", true},
		{"Em", "2A", true},
		{"F#m", "4A", true},
		{"Gbm", "4A", true},
		{"Ebm", "7A", true},
		{"D#m", "7A", true},
		{"C#", "8B", true},
		{"Db", "8B", true},
		{"unknown", "", false},
	}

	for _, tc := range cases {
		code, ok := KeyToCode(tc.key)
		if code != tc.code || ok != tc.found {
			t.Errorf("KeyToCode(%q) = (%q, %v); want (%q, %v)", tc.key, code, ok, tc.code, tc.found)
		}
	}
}

func TestScore(t *testing.T) {
	cases := []struct {
		a, b   string
		score  int
	}{
		{"7A", "7A", 10}, // same code
		{"7A", "7B", 8},  // relative major/minor
		{"7A", "8A", 6},  // circle neighbor up
		{"7A", "6A", 6},  // circle neighbor down
		{"7A", "8B", 3},  // diagonal up
		{"7A", "6B", 3},  // diagonal down
		{"7A", "1A", 0},  // no relation
		{"12A", "1A", 6}, // wrap around
		{"12A", "1B", 3}, // wrap diagonal
		{"", "7A", 0},    // empty input
	}

	for _, tc := range cases {
		got := Score(tc.a, tc.b)
		if got != tc.score {
			t.Errorf("Score(%q, %q) = %d; want %d", tc.a, tc.b, got, tc.score)
		}
	}
}
