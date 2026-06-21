package encoder

import (
	"fmt"
	"log"
	"strings"
	"unicode"
)

// NormalizeMessage lowercases message and strips non-letter characters.
// Only a-z are supported in the steganographic payload.
func NormalizeMessage(msg string) string {
	var b strings.Builder
	for _, r := range msg {
		if unicode.IsLetter(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// encodeLength encodes a non-negative integer as exactly 3 base-26 lowercase
// letters ('a'=0 .. 'z'=25). Supports lengths 0..17575 (26^3 - 1).
func encodeLength(n int) (string, error) {
	if n < 0 || n > 17575 {
		return "", fmt.Errorf("message length %d out of range [0, 17575]", n)
	}
	c2 := byte('a' + n%26)
	n /= 26
	c1 := byte('a' + n%26)
	n /= 26
	c0 := byte('a' + n%26)
	return string([]byte{c0, c1, c2}), nil
}

// decodeLength reverses encodeLength.
func decodeLength(s string) (int, error) {
	if len(s) != 3 {
		return 0, fmt.Errorf("length prefix must be 3 chars, got %d", len(s))
	}
	for _, c := range []byte(s) {
		if c < 'a' || c > 'z' {
			return 0, fmt.Errorf("invalid length prefix char %q", c)
		}
	}
	n := int(s[0]-'a')*676 + int(s[1]-'a')*26 + int(s[2]-'a')
	return n, nil
}

// DecodeLengthPrefix is exported so the decoder can call it.
var DecodeLengthPrefix = decodeLength

// EncodeMessage encodes message into a playlist drawn from pool.
// The first 3 tracks carry the base-26 length prefix; subsequent tracks carry
// message bytes (one lowercase letter per track). The remainder of targetLength
// is filled with greedy unconstrained tracks.
func EncodeMessage(pool []Track, message string, keywords [3]string, targetLength int) ([]Track, error) {
	if len(pool) == 0 {
		return nil, fmt.Errorf("encode: empty pool")
	}
	msg := NormalizeMessage(message)
	prefix, err := encodeLength(len(msg))
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}
	payload := prefix + msg // e.g. "aabhelloworld"
	if len(payload) > targetLength {
		return nil, fmt.Errorf("encode: payload length %d exceeds target %d", len(payload), targetLength)
	}

	rng := DeriveRNG(keywords)
	used := make(map[string]bool, targetLength)
	playlist := make([]Track, 0, targetLength)

	var prev *Track
	payloadPos := 0
	for payloadPos < len(payload) {
		needed := []byte(payload[payloadPos:])
		t, consumed, err := findWithFallback(prev, needed, pool, used, rng)
		if err != nil {
			show := needed
			if len(show) > 4 {
				show = show[:4]
			}
			return nil, fmt.Errorf("encode position %d (needed %q): %w", payloadPos, show, err)
		}
		ExtractFromTrack(rng, t.Title) // commit: advance rng
		used[t.ID] = true
		playlist = append(playlist, *t)
		prev = &playlist[len(playlist)-1]
		payloadPos += consumed
	}

	// Fill remainder with greedy unconstrained tracks
	remaining := targetLength - len(playlist)
	if remaining > 0 {
		var freePool []Track
		for _, t := range pool {
			if !used[t.ID] {
				freePool = append(freePool, t)
			}
		}
		if len(freePool) < remaining {
			log.Printf("encode: only %d free tracks for %d greedy slots", len(freePool), remaining)
			remaining = len(freePool)
		}
		greedySeed := int64(rng.Clone().next())
		greedy, err := BuildPlaylistGreedy(freePool, remaining, greedySeed)
		if err != nil {
			return nil, fmt.Errorf("encode greedy fill: %w", err)
		}
		playlist = append(playlist, greedy...)
	}

	return playlist, nil
}
