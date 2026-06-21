package decoder

import (
	"fmt"

	"spotifystego/internal/encoder"
)

// Track is the decoder's view of a playlist entry.
type Track = encoder.Track

// TrackExtraction records which letters were pulled from a track.
type TrackExtraction struct {
	Track   Track
	Letters []byte
}

// DecodePlaylist reconstructs the hidden message from a playlist.
// keywords must match those used during encoding.
// Returns the decoded message (lowercase letters only) and per-track extractions.
func DecodePlaylist(tracks []Track, keywords [3]string) (string, []TrackExtraction, error) {
	rng := encoder.DeriveRNG(keywords)

	var allLetters []byte
	extractions := make([]TrackExtraction, 0, len(tracks))

	for _, t := range tracks {
		letters := encoder.ExtractFromTrack(rng, t.Title)
		extractions = append(extractions, TrackExtraction{Track: t, Letters: letters})
		if len(letters) > 0 {
			allLetters = append(allLetters, letters[0])
		}
	}

	if len(allLetters) < 3 {
		return "", extractions, fmt.Errorf("decode: not enough letters extracted (got %d, need ≥3)", len(allLetters))
	}

	msgLen, err := encoder.DecodeLengthPrefix(string(allLetters[:3]))
	if err != nil {
		return "", extractions, fmt.Errorf("decode: invalid length prefix %q: %w", allLetters[:3], err)
	}
	if msgLen < 0 {
		return "", extractions, fmt.Errorf("decode: negative message length %d", msgLen)
	}
	if len(allLetters) < 3+msgLen {
		return "", extractions, fmt.Errorf(
			"decode: need %d message letters but only %d available after prefix",
			msgLen, len(allLetters)-3,
		)
	}

	message := string(allLetters[3 : 3+msgLen])
	return message, extractions, nil
}
