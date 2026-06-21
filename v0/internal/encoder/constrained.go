package encoder

import (
	"fmt"

	"spotifystego/internal/camelot"
)

// lettersMatch returns true if got[0] == needed[0].
// Only the first extracted letter carries payload — the encoder selects
// tracks by their first letter, and ExtractFromTrack advances the PRNG
// consistently regardless of how many letters a title yields.
func lettersMatch(got, needed []byte) bool {
	return len(got) > 0 && len(needed) > 0 && got[0] == needed[0]
}

// FindConstrained finds the best track from pool whose first extracted letter
// matches needed[0]. Returns the track and 1 (always one payload byte consumed).
// Does NOT advance rng — caller must call ExtractFromTrack to commit.
func FindConstrained(prev *Track, needed []byte, pool []Track, used map[string]bool, rng *RNG) (*Track, int, error) {
	bestScore := -1.0
	var best *Track

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
		}
	}
	if best == nil {
		return nil, 0, fmt.Errorf("constrained: no track in pool of %d yields %q", len(pool), needed[:1])
	}
	return best, 1, nil
}
