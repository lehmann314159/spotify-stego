package encoder

import (
	"fmt"
	"log"
)

// findWithFallback tries the constrained search with progressively relaxed
// Camelot constraints. Returns the chosen track, how many payload bytes it
// consumes, or an error with diagnostics.
//
// Strategy:
//  1. Strict: Camelot-compatible candidates only
//  2. Relax Camelot: any key, letters still required
//  3. Error with diagnostics
func findWithFallback(prev *Track, needed []byte, pool []Track, used map[string]bool, rng *RNG) (*Track, int, error) {
	t, n, err := FindConstrained(prev, needed, pool, used, rng)
	if err == nil {
		return t, n, nil
	}

	show := needed
	if len(show) > 4 {
		show = needed[:4]
	}
	log.Printf("fallback: relaxing Camelot constraint for needed=%q (pool=%d)", show, len(pool))
	t, n, err = FindConstrained(nil, needed, pool, used, rng)
	if err == nil {
		return t, n, nil
	}

	return nil, 0, fmt.Errorf(
		"encode failed: no track yields letters %q — pool=%d, used=%d, tried both strict and relaxed Camelot",
		show, len(pool), len(used),
	)
}
