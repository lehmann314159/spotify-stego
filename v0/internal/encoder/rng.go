package encoder

import (
	"crypto/sha256"
	"encoding/binary"
	"strings"
)

// RNG is a cloneable Xorshift64 pseudorandom number generator.
// Being cloneable lets constrained search test candidate tracks without
// permanently advancing the sequence.
type RNG struct {
	state uint64
}

// DeriveRNG seeds an RNG from the SHA256 hash of the joined keywords.
// Same keywords always produce the same sequence.
func DeriveRNG(keywords [3]string) *RNG {
	joined := strings.Join(keywords[:], "|")
	hash := sha256.Sum256([]byte(joined))
	seed := binary.LittleEndian.Uint64(hash[:8])
	if seed == 0 {
		seed = 1 // xorshift64 must not have state 0
	}
	return &RNG{state: seed}
}

// Clone returns a copy of this RNG at the same position in the sequence.
func (r *RNG) Clone() *RNG {
	return &RNG{state: r.state}
}

// SameState reports whether r and other are at the same position.
func (r *RNG) SameState(other *RNG) bool {
	return r.state == other.state
}

// next advances the state and returns the new value.
func (r *RNG) next() uint64 {
	x := r.state
	x ^= x << 13
	x ^= x >> 7
	x ^= x << 17
	r.state = x
	return x
}

// Intn returns a non-negative integer in [0, n).
// Panics if n <= 0.
func (r *RNG) Intn(n int) int {
	if n <= 1 {
		r.next() // always advance even for n==1
		return 0
	}
	return int(r.next() % uint64(n))
}
