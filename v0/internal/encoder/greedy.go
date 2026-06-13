package encoder

import (
	"fmt"
	"math"
	"math/rand"

	"spotifystego/internal/camelot"
)

// bpmScore returns a BPM compatibility score in [0,5]: full within 6%,
// linearly falling to 0 at 20% difference.
func bpmScore(current, candidate float64) float64 {
	if current == 0 || candidate == 0 {
		return 0
	}
	diff := math.Abs(current-candidate) / current
	if diff <= 0.06 {
		return 5
	}
	if diff >= 0.20 {
		return 0
	}
	return 5 * (1 - (diff-0.06)/0.14)
}

// scoreTransition returns the total musical score between two consecutive tracks.
func scoreTransition(prev, c *Track) float64 {
	cs := float64(camelot.Score(prev.CamelotCode, c.CamelotCode))
	bs := bpmScore(prev.BPM, c.BPM)
	return cs + bs
}

// BuildPlaylistGreedy builds a musically coherent playlist of targetLength from
// pool, starting from a random seed track. No track repeats.
func BuildPlaylistGreedy(pool []Track, targetLength int, seed int64) ([]Track, error) {
	if len(pool) < targetLength {
		return nil, fmt.Errorf("greedy: pool size %d < target %d", len(pool), targetLength)
	}
	//nolint:gosec
	rng := rand.New(rand.NewSource(seed))
	used := make(map[string]bool, targetLength)

	startIdx := rng.Intn(len(pool))
	playlist := []Track{pool[startIdx]}
	used[pool[startIdx].ID] = true

	for len(playlist) < targetLength {
		prev := &playlist[len(playlist)-1]
		bestScore := -1.0
		bestIdx := -1
		for i := range pool {
			c := &pool[i]
			if used[c.ID] {
				continue
			}
			s := scoreTransition(prev, c)
			if s > bestScore {
				bestScore = s
				bestIdx = i
			}
		}
		if bestIdx < 0 {
			return nil, fmt.Errorf("greedy: ran out of candidates at length %d", len(playlist))
		}
		playlist = append(playlist, pool[bestIdx])
		used[pool[bestIdx].ID] = true
	}
	return playlist, nil
}
