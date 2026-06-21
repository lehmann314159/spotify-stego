# Multi-Letter Encoding Per Track

## The problem with the current attempt

`ExtractFromTrack` determines how many letters to extract using the shared PRNG:

```go
count := 1 + rng.Intn(cap3)  // 1, 2, or 3
```

During constrained search, all candidates are tested with the same cloned PRNG state, so they all get the same `count`. If the PRNG says 3, every candidate must match 3 payload chars simultaneously — probability 1/17,576 per track. With 2400 tracks, expected matches ≈ 0.14. The encoder reliably fails.

## Proposed fix

Derive `count` from the track title itself instead of the PRNG:

```go
count := 1 + (len(titleLetters(title)) % cap3)
```

Different titles have different letter counts, so candidates yield different `count` values. The encoder can fall back to 1-letter tracks when 2- or 3-letter tracks aren't available. The decoder stays in sync because it calls the same `ExtractFromTrack` with the same title and gets the same count.

## What needs to change

- `ExtractFromTrack` in `v0/internal/encoder/extract.go` — replace `rng.Intn(cap3)` with `len(titleLetters(title)) % cap3`
- The PRNG no longer advances for the count step, only for the position samples — verify encoder and decoder still advance identically
- `EncodeMessage` in `v0/internal/encoder/encode.go` — restore `payloadPos` loop that advances by `consumed` (the number of letters the chosen track yields)
- `FindConstrained` in `v0/internal/encoder/constrained.go` — restore multi-letter `lettersMatch` and return consumed count
- `DecodePlaylist` in `v0/internal/decoder/decode.go` — restore `allLetters = append(allLetters, letters...)` (all letters, not just first)
- Integration tests — add a realistic multi-word title pool to catch the count-collision bug that the single-char-word pool masked

## Risk

The `count` derivation must be stable — if `titleLetters` ever changes behavior, old playlists become undecodable. Pin the formula carefully.
