package encoder

import (
	"fmt"

	"spotifystego/internal/camelot"
)

// lettersMatch returns true if the first min(len(got), len(needed)) bytes agree.
// The consumed count is min(len(got), len(needed)) — the track covers that many
// payload bytes. Tracks that yield more letters than remain in the payload are
// accepted but only consume what's left (decoder ignores the tail).
func lettersMatch(got, needed []byte) bool {
	n := len(got)
	if n > len(needed) {
		n = len(needed)
	}
	if n == 0 {
		return false
	}
	for i := 0; i < n; i++ {
		if got[i] != needed[i] {
			return false
		}
	}
	return true
}

// consumed returns how many payload bytes got covers given the remaining needed.
func consumed(got, needed []byte) int {
	if len(got) < len(needed) {
		return len(got)
	}
	return len(needed)
}

// FindConstrained finds the best track from pool whose extracted letters cover
// the next bytes of needed. Returns the track and the number of payload bytes
// it consumes. Does NOT advance rng — caller must call ExtractFromTrack to commit.
func FindConstrained(prev *Track, needed []byte, pool []Track, used map[string]bool, rng *RNG) (*Track, int, error) {
	bestScore := -1.0
	var best *Track
	bestConsumed := 0

	for i := range pool {
		c := &pool[i]
		if used[c.ID] {
			continue
		}
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
			bestConsumed = consumed(letters, needed)
		}
	}
	if best == nil {
		show := needed
		if len(show) > 4 {
			show = needed[:4]
		}
		return nil, 0, fmt.Errorf("constrained: no track in pool of %d yields %q", len(pool), show)
	}
	return best, bestConsumed, nil
}
