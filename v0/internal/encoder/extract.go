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

// ExtractFromTrack uses rng to decide how many letters to pull and which
// positions to pick from the title's letter string. Always advances rng fully
// so encoder and decoder stay in sync.
func ExtractFromTrack(rng *RNG, title string) []byte {
	letters := titleLetters(title)
	cap3 := maxWordLen(title)
	// count in [1, cap3]
	count := 1 + rng.Intn(cap3)
	if len(letters) == 0 {
		for i := 0; i < count; i++ {
			rng.Intn(1) // advance for each position slot
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
