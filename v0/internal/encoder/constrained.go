package encoder

import (
	"fmt"

	"spotifystego/internal/camelot"
)

// lettersMatch returns true if got starts with all bytes in needed.
func lettersMatch(got, needed []byte) bool {
	if len(got) < len(needed) {
		return false
	}
	for i, b := range needed {
		if got[i] != b {
			return false
		}
	}
	return true
}

// FindConstrained finds the best track from pool whose title yields `needed`
// letters when ExtractFromTrack is called with the current rng state.
// Uses Clone() to test each candidate without advancing the real rng.
// Returns the winning track and does NOT advance rng — the caller must call
// ExtractFromTrack(rng, winner.Title) to commit.
func FindConstrained(prev *Track, needed []byte, pool []Track, used map[string]bool, rng *RNG) (*Track, error) {
	bestScore := -1.0
	var best *Track

	for i := range pool {
		c := &pool[i]
		if used[c.ID] {
			continue
		}
		// Clone rng so we don't advance the real state
		trial := rng.Clone()
		letters := ExtractFromTrack(trial, c.Title)
		if !lettersMatch(letters, needed) {
			continue
		}
		var score float64
		if prev != nil {
			score = float64(camelot.Score(prev.CamelotCode, c.CamelotCode)) + bpmScore(prev.BPM, c.BPM)
		}
		if score > bestScore {
			bestScore = score
			best = c
		}
	}
	if best == nil {
		return nil, fmt.Errorf("constrained: no track in pool of %d yields %q", len(pool), needed)
	}
	return best, nil
}
