package encoder

import (
	"fmt"

	"spotifystego/internal/camelot"
)

// lettersMatch checks whether got is a byte-wise prefix of needed up to
// min(len(got), len(needed)). Returns (matched, consumed) where consumed is
// the number of payload bytes this track covers. Both slices must be non-empty.
func lettersMatch(got, needed []byte) (bool, int) {
	if len(got) == 0 || len(needed) == 0 {
		return false, 0
	}
	n := len(got)
	if len(needed) < n {
		n = len(needed)
	}
	for i := 0; i < n; i++ {
		if got[i] != needed[i] {
			return false, 0
		}
	}
	return true, n
}

// FindConstrained finds the best track from pool whose extracted letters match
// as a prefix of needed. Among matches, prefers the track that consumes the
// most payload bytes; breaks ties by musicality score.
// Does NOT advance rng — caller must call ExtractFromTrack to commit.
func FindConstrained(prev *Track, needed []byte, pool []Track, used map[string]bool, rng *RNG) (*Track, int, error) {
	bestScore := -1.0
	bestConsumed := 0
	var best *Track

	for i := range pool {
		c := &pool[i]
		if used[c.ID] {
			continue
		}
		trial := rng.Clone()
		letters := ExtractFromTrack(trial, c.Title)
		ok, consumed := lettersMatch(letters, needed)
		if !ok {
			continue
		}
		var score float64
		if prev != nil {
			score = float64(camelot.Score(prev.CamelotCode, c.CamelotCode)) + bpmScore(prev.BPM, c.BPM)
		}
		if consumed > bestConsumed || (consumed == bestConsumed && score > bestScore) {
			bestScore = score
			bestConsumed = consumed
			best = c
		}
	}
	if best == nil {
		show := needed
		if len(show) > 4 {
			show = show[:4]
		}
		return nil, 0, fmt.Errorf("constrained: no track in pool of %d yields %q", len(pool), show)
	}
	return best, bestConsumed, nil
}
