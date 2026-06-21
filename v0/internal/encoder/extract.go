package encoder

import (
	"strings"
	"unicode"
)

// titleLetters returns all lowercase letters from a title, concatenated across words.
func titleLetters(title string) []byte {
	var out []byte
	for _, w := range strings.Fields(title) {
		for _, r := range w {
			if unicode.IsLetter(r) {
				out = append(out, byte(unicode.ToLower(r)))
			}
		}
	}
	return out
}

// maxWordLen returns the length of the longest word in the title, capped at 3.
func maxWordLen(title string) int {
	max := 1
	for _, w := range strings.Fields(title) {
		wordLen := 0
		for _, r := range w {
			if unicode.IsLetter(r) {
				wordLen++
			}
		}
		if wordLen > max {
			max = wordLen
		}
	}
	if max > 3 {
		return 3
	}
	return max
}

// ExtractFromTrack extracts letters from a track title using rng for position
// sampling. The count is derived from the title itself (not the PRNG) so
// different tracks yield different counts, enabling the constrained encoder to
// find tracks that carry 1 or 2 payload bytes in a single slot.
// Always advances rng for every position draw so encoder and decoder stay in sync.
func ExtractFromTrack(rng *RNG, title string) []byte {
	letters := titleLetters(title)
	cap3 := maxWordLen(title)
	// Derive count from title length so candidates get distinct counts.
	// Range: [1, cap3].
	count := 1 + (len(letters) % cap3)
	if len(letters) == 0 {
		for i := 0; i < count; i++ {
			rng.Intn(1) // advance PRNG for each position slot
		}
		return nil
	}
	result := make([]byte, count)
	for i := 0; i < count; i++ {
		pos := rng.Intn(len(letters))
		result[i] = letters[pos]
	}
	return result
}
