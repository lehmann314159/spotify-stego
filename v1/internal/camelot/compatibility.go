package camelot

import (
	"strconv"
	"strings"
)

// parseCode extracts the numeric position (1-12) and letter (A or B) from a
// Camelot code such as "7A" or "12B". Returns false if the code is invalid.
func parseCode(code string) (int, string, bool) {
	if code == "" || len(code) < 2 {
		return 0, "", false
	}

	letter := string(code[len(code)-1])
	numStr := strings.TrimSuffix(code, letter)

	num, err := strconv.Atoi(numStr)
	if err != nil || num < 1 || num > 12 {
		return 0, "", false
	}

	return num, letter, true
}

// Score returns a compatibility score between two Camelot codes.
// Higher is better. Returns 0 if either code is empty or unrecognized.
//
// Scoring:
//
//	10 — same code (e.g. "7A" → "7A")
//	 8 — relative major/minor (same number, different letter: "7A" → "7B")
//	 6 — circle of fifths neighbor (±1 on wheel, same letter: "7A" → "6A" or "8A")
//	 3 — diagonal (±1 on wheel, different letter: "7A" → "6B" or "8B")
//	 0 — no relationship
func Score(current, candidate string) int {
	n1, l1, ok1 := parseCode(current)
	n2, l2, ok2 := parseCode(candidate)

	if !ok1 || !ok2 {
		return 0
	}

	// Same code
	if n1 == n2 && l1 == l2 {
		return 10
	}

	// Relative major/minor (same number, different letter)
	if n1 == n2 {
		return 8
	}

	// Distance on the wheel (circular)
	diff := abs(n1 - n2)
	if diff > 6 {
		diff = 12 - diff
	}

	// Neighbor (distance == 1)
	if diff == 1 {
		if l1 == l2 {
			return 6 // circle of fifths neighbor
		}
		return 3 // diagonal
	}

	return 0
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
