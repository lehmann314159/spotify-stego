# Spotify Stego - Learning Progress

## Current Phase
Phase 1: PRNG seeding + deterministic sequences

## Status
Concept explained, ready to implement

## Completed
- [x] Read design document
- [x] Decided on learning approach (guided writing)
- [ ] Implement deriveExtractor function
- [ ] Test deterministic sequence generation
- [ ] Understand why this matters for encode/decode sync

## Current Task
Write deriveExtractor function that:
- Takes [3]string
- Combines with strings.Join using "|" separator
- Hashes with sha256.Sum256
- Converts first 8 bytes to int64 via binary.BigEndian.Uint64
- Returns *rand.Rand seeded with that value

## Notes
- Using math/rand (not crypto/rand) - determinism matters more than security here

## Phase Overview
1. PRNG seeding + deterministic sequences ← current
2. Letter extraction algorithm
3. Encode/decode round-trip (CLI)
4. Camelot wheel + BPM scoring
5. Constrained search
6. Spotify/GetSongBPM APIs
7. SQLite track cache
8. Web UI (Go templates + HTMX)
