# Bead B04 — Camelot Mapping

**Phase:** 1 (task 1.5)  
**Model:** qwen3.6-27b  
**Depends on:** B01  

---

## Preamble

Read `v1-preamble.md` in the repo root before starting. All constraints
and directory structure requirements are defined there. Work in `v1/` only.

---

## Task

Implement the Camelot wheel mapping and compatibility scoring in
`v1/internal/camelot/`.

The Camelot wheel is a DJ mixing tool that represents musical keys as
positions on a clock face. Each position has two codes: an "A" code
(minor key) and a "B" code (major key). Compatible keys are adjacent
on the wheel.

### Camelot Wheel Layout

The 24 codes map to musical keys as follows:

| Code | Key   | Code | Key   |
|------|-------|------|-------|
| 1A   | Am    | 1B   | C     |
| 2A   | Em    | 2B   | G     |
| 3A   | Bm    | 3B   | D     |
| 4A   | F#m   | 4B   | A     |
| 5A   | C#m   | 5B   | E     |
| 6A   | G#m   | 6B   | B     |
| 7A   | Ebm   | 7B   | F#    |
| 8A   | Bbm   | 8B   | C#    |
| 9A   | Fm    | 9B   | Ab    |
| 10A  | Cm    | 10B  | Eb    |
| 11A  | Gm    | 11B  | Bb    |
| 12A  | Dm    | 12B  | F     |

Note: key names from GetSongBPM may use flats or sharps interchangeably
(e.g. "Ebm" and "D#m" refer to the same key). The mapping must handle
common enharmonic equivalents.

### Files

**`v1/internal/camelot/mapping.go`**

```go
package camelot

// KeyToCode maps a musical key string to its Camelot wheel code.
// Returns ("", false) if the key is not recognized.
func KeyToCode(keyOf string) (string, bool)
```

The lookup table must cover all 24 standard keys plus common enharmonic
equivalents:
- "Ebm" and "D#m" both map to "7A"
- "Bbm" and "A#m" both map to "8A"  
- "F#m" and "Gbm" both map to "4A"
- "C#m" and "Dbm" both map to "5A"
- "G#m" and "Abm" both map to "6A"
- "F#" and "Gb" both map to "7B"
- "C#" and "Db" both map to "8B"
- "Ab" and "G#" both map to "9B"
- "Eb" and "D#" both map to "10B"
- "Bb" and "A#" both map to "11B"

**`v1/internal/camelot/compatibility.go`**

Implement Camelot wheel compatibility scoring between two codes. The wheel
has 12 positions numbered 1–12, each with an A (minor) and B (major) variant.

```go
package camelot

// Score returns a compatibility score between two Camelot codes.
// Higher is better. Returns 0 if either code is empty or unrecognized.
//
// Scoring:
//   10 — same code (e.g. "7A" → "7A")
//    8 — relative major/minor (same number, different letter: "7A" → "7B")
//    6 — circle of fifths neighbor (±1 on wheel, same letter: "7A" → "6A" or "8A")
//    3 — diagonal (±1 on wheel, different letter: "7A" → "6B" or "8B")
//    0 — no relationship
func Score(current, candidate string) int
```

Parse each Camelot code into its numeric position (1–12) and letter (A or B).
The wheel wraps: position 12 and position 1 are neighbors.

### Test

Create `v1/internal/camelot/camelot_test.go`:

1. **KeyToCode tests** — verify a representative sample:
   - "Am" → "1A"
   - "C" → "1B"
   - "Em" → "2A"
   - "F#m" → "4A"
   - "Gbm" → "4A" (enharmonic)
   - "Ebm" → "7A"
   - "D#m" → "7A" (enharmonic)
   - "C#" → "8B"
   - "Db" → "8B" (enharmonic)
   - "unknown" → ("", false)

2. **Score tests** — verify all relationship types:
   - Same code: Score("7A", "7A") == 10
   - Relative: Score("7A", "7B") == 8
   - Circle neighbor up: Score("7A", "8A") == 6
   - Circle neighbor down: Score("7A", "6A") == 6
   - Diagonal up: Score("7A", "8B") == 3
   - Diagonal down: Score("7A", "6B") == 3
   - No relation: Score("7A", "1A") == 0
   - Wrap around: Score("12A", "1A") == 6
   - Wrap diagonal: Score("12A", "1B") == 3
   - Empty input: Score("", "7A") == 0

## Exit Criteria

- [ ] `v1/internal/camelot/mapping.go` with `KeyToCode` covering all 24
      keys and common enharmonic equivalents
- [ ] `v1/internal/camelot/compatibility.go` with `Score` implementing
      all four compatibility levels
- [ ] Wrap-around (12 ↔ 1) handled correctly
- [ ] `go build ./...` passes
- [ ] `go test ./internal/camelot/...` passes with all cases above
