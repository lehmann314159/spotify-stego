package encoder

import (
	"fmt"
	"log"
)

// findWithFallback tries the constrained search with progressively relaxed
// Camelot constraints. Returns the chosen track or an error with diagnostics.
//
// Strategy:
//   1. Strict: Camelot-compatible candidates only
//   2. Relax Camelot: any key, letters still required
//   3. Error with diagnostics
func findWithFallback(prev *Track, needed []byte, pool []Track, used map[string]bool, rng *RNG) (*Track, error) {
	// Pass 1: strict (Camelot filter applied inside scoring)
	t, err := FindConstrained(prev, needed, pool, used, rng)
	if err == nil {
		return t, nil
	}

	// Pass 2: relax Camelot by passing nil as prev (no key scoring pressure)
	log.Printf("fallback: relaxing Camelot constraint for needed=%q (pool=%d)", needed, len(pool))
	t, err = FindConstrained(nil, needed, pool, used, rng)
	if err == nil {
		return t, nil
	}

	// Pass 3: total failure with diagnostics
	return nil, fmt.Errorf(
		"encode failed: no track yields letters %q — pool=%d, used=%d, tried both strict and relaxed Camelot",
		needed, len(pool), len(used),
	)
}
