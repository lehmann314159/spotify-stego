package crypto

import (
	"crypto/sha256"
	"encoding/binary"
	"math/rand"
	"strings"
)

func longestWordLen(words []string) int {
	max := 0
	for _, word := range words {
		if len(word) > max {
			max = len(word)
		}
	}
	return max
}

func DeriveExtractor(keywords [3]string) *rand.Rand {
	combined := strings.Join(keywords[:], "|")
	hash := sha256.Sum256([]byte(combined))
	seed := binary.BigEndian.Uint64(hash[0:8])
	return rand.New(rand.NewSource(int64(seed)))
}

func ExtractFromTrack(rng *rand.Rand, title string, isMessage bool) []byte {
	words := strings.Fields(title)
	if len(words) == 0 {
		return nil
	}

	// Always draw count from full range — keeps PRNG in sync on both sides
	maxLetters := min(3, longestWordLen(words))
	count := rng.Intn(maxLetters) + 1 // 1 to maxLetters

	// Message tracks cap at 2 to keep constrained search viable
	useCount := count
	if isMessage && useCount > 2 {
		useCount = 2
	}

	var letters []byte
	for i := 0; i < count; i++ { // always iterate full count for PRNG sync
		wordIdx := rng.Intn(len(words))
		charIdx := rng.Intn(len(words[wordIdx]))
		if i < useCount {
			letters = append(letters, words[wordIdx][charIdx])
		}
		// else: PRNG advanced but letter discarded
	}
	return letters
}
