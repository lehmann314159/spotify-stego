# GasTown Execution Log — spotify-stego

Per-Bead record of Polecat dispatch, results, and failure classification.
See `gastown_spotify_stego_pilot.md` in AiHomeLab for research design.

## Failure Mode Taxonomy
- **M** — Mayor failure: Bead was underspecified; model made silent choices
- **P** — Polecat failure: Model hit a genuine capability ceiling
- **I** — Infrastructure failure: OpenCode, Ollama, or tooling issue

## Escalation Policy (V1)
1. Primary: qwen3.6-27b
2. After 2 M-classified retries + Bead rewrite: qwen3.6-35b-a3b
3. After 2 more failures, or Mayor judgment: Claude Code

---

## V0 — Claude Code Baseline

**Completed:** June 2026  
**Elapsed:** 18m 26s  
**Files:** 33  
**Tests:** 16, all passing  
**Prompt:** `v0-prompt.md`  

### Spec Defects Discovered

| # | Description | Resolution |
|---|-------------|------------|
| 1 | Decimal length prefix impossible — extraction only yields a-z, digits unreachable | Fixed: base-26 3-char prefix. Added to cross-version constraints. |
| 2 | Implicit character set assumption — messages must normalize to lowercase a-z | Added to cross-version constraints. |

### Architectural Decisions (Reference for V1 Comparison)

| Decision | Choice | Rationale |
|----------|--------|-----------|
| PRNG | Custom Xorshift64 with Clone() | math/rand.Rand not cloneable; speculative candidate testing requires it |
| Length prefix | Base-26, 3 chars (a-z) | Decimal impossible given extraction constraints |
| Provider package | interface + stub + real in internal/audio/ | Clean separation; stub default |
| SVG rendering | Server-side, embedded in HTMX partials | No JS dependency |
| Template loading | CWD-relative | Limitation: binary must run from v0/ |

### Post-Mortem Observations

- Pool size is more critical than spec implied. Small pools (26 tracks)
  routinely leave some letters unreachable at specific PRNG states. Real
  deployment needs thousands of indexed tracks.
- Import cycle between encoder and decoder is a design smell. Both share
  PRNG and extraction logic that should live in a neutral package
  (internal/stego/core). Flagged for V1 correction.
- PRNG cloneability gap is a silent Go footgun — math/rand limitation not
  documented; only discovered when speculative testing is attempted.

### Deployment Gaps (Phase 5 — Expected)

- Indexer hardcodes StubProvider; no flag to switch to real provider
- Template loading is CWD-relative
- Spotify client makes redundant unauthenticated request before real call
- No multi-genre loop in indexer

### V1 Watchpoints

1. Will qwen3.6-27b independently discover the PRNG cloneability issue,
   or will it use math/rand and produce a subtly broken constrained search?
2. Will V1 independently arrive at the import cycle fix (internal/stego/core)?

---

## V1 Phase 2 Post-Mortem

*Completed: June 2026. B06–B09. All beads passing.*

### Watchpoint Resolution

| # | Watchpoint | Outcome |
|---|------------|---------|
| 1 | Will qwen3.6-27b independently discover the PRNG cloneability requirement, or reach for math/rand? | **Not testable.** Constraint 5 in v1-preamble.md pre-empted independent discovery by naming the requirement explicitly. Polecat complied correctly. |
| 2 | Will V1 independently arrive at the import cycle fix (internal/stego/core)? | **Not testable.** Constraint 6 in v1-preamble.md named the package explicitly. Polecat complied correctly. |

Both watchpoints were defused by the preamble before Phase 2 started. For V2, consider withholding these constraints from the decomposer to test independent discovery.

### Mayor Lessons

**Bead prompt size discipline.** The initial B06 draft produced in this conversation was substantially larger than the B01–B05 convention (full standing context repeated inline, rather than delegated to v1-preamble.md). Corrected before dispatch. The right pattern: Bead files are task-only; v1-preamble.md carries all standing context. Keeps Beads readable and avoids redundancy drift between the preamble and individual Beads.

**Bead ordering risk: decoder before encoder.** Placing the decoder (B09) before the constrained encoder (Phase 3) was architecturally correct — decoder has no dependency on encoder — but made B09's round-trip test the hardest part of Phase 2. Without a working encoder, the Polecat had to invent a brute-force keyword search to find PRNG states where the length prefix decoded to a manageable value. The test is correct and reproducible, but cost 52k tokens and multiple self-correction loops. If the test requirement had been deferred to B12 (end-to-end, after the encoder exists), B09 would have been straightforward.

**Sharpness principle confirmed.** B09's self-correction spiral was caused by ambiguity in test construction, not algorithmic difficulty. The Polecat's PRNG reasoning was sound; the problem was that the Bead left "build a playlist where the letters are known" as an exercise. B10 responds directly: speculative probing approach is chosen explicitly (approach 2), prefix matching semantics are stated precisely, and tests are pre-scoped to avoid equivalent ambiguity.

**Token cost as a signal.** B06–B08 were cheap (estimated 5–15k tokens each). B09 was 52k. The jump was caused by the test construction problem, not the algorithm itself. B10 and B11 are the most algorithmically complex Beads in the project — if the Bead is sharp enough, token cost should stay reasonable. If B10 or B11 approach B09's cost, that's a signal to re-examine Bead sharpness before B12.

### Phase 2 Architectural Decisions (V1)

| Decision | Choice | Notes |
|----------|--------|-------|
| PRNG | Xorshift64, constants (13,7,17), Clone() via struct copy | Matches V0. Cross-version compatible. |
| Key derivation | SHA-256 of space-joined keywords → big-endian uint64 | V0 used pipe-joined; V1 uses space-joined. Not cross-version compatible for encode, but decoder only needs keyword→seed to match within a version. |
| Letter extraction | ExtractLetters in stego/core, isMessage cap at 2 | PRNG sync verified by test. |
| Length encoding | Base-26, 3 chars, EncodeLength/DecodeLength in stego/core | Matches cross-version constraint 4. |
| Decoder | Treats all tracks as isMessage=true, stops at 3+msgLen letters | Correct: decoder doesn't know message/padding boundary, relies on length prefix. |

**Note on key derivation divergence:** V0 joined keywords with `"|"` as separator; V1 joins with `" "`. This means a playlist encoded by V0 cannot be decoded by V1 and vice versa. Not a problem for the pilot (cross-version decode was never a stated requirement) but worth documenting for the article.

---

## V1 Log

| B01 | 1.1+1.2 | qwen3.6-27b | Scaffold + SQLite schema | ✅ Pass | — | Clean. Track in stego/core, modernc.org/sqlite, idiomatic error wrapping. Polecat self-updated preamble tracker. |
| B02 | 1.3 | qwen3.6-27b | Spotify API client | ✅ Pass | — | Lazy auth, token caching with mutex, pagination, 429 retry. Strong httptest suite (5 test cases). Pagination termination logic slightly convoluted but correct. |
| B03 | 1.4 | qwen3.6-27b | Audio provider + stub + GetSongBPM | ✅ Pass | — | Clean interface/stub/real separation. openKeyToCamelot correct. Possible detail endpoint JSON shape mismatch ({"song":{"song":{}}} vs {"song":{}}); needs verification when real API wired. |
| B04 | 1.5 | qwen3.6-27b | Camelot mapping + compatibility | ✅ Pass | — | All 24 keys + enharmonics correct. Wrap-around handled cleanly. Score logic concise and correct. All 10 test cases pass. |
| B05 | 1.6+1.7 | qwen3.6-27b | Indexer + integration test | ✅ Pass | — | runIndexer extracted cleanly. Provider selection via env var correct. Hit compilation error (missing setters on Spotify/audio clients); self-diagnosed and fixed by adding SetHTTPClient/SetEndpoints/SetBaseURL methods to upstream packages without prompting. Strong G3-level recovery. |
| B06 | 2.1 | qwen3.6-27b | PRNG + key derivation (stego/core) | ✅ Pass | — | Correct xorshift64 constants (13,7,17). No math/rand. Clone() independence tested via direct state access (package-internal test). DeriveKey uses binary.BigEndian.Uint64 correctly. Zero-seed guard in both New() and DeriveKey(). All 7 test cases present plus unprompted Intn panic test. Clean. |
| B07 | 2.2 | qwen3.6-27b | Letter extraction | ✅ Pass | — | PRNG sync test correct — verifies identical state after message and padding tracks via direct state comparison. Message cap and padding uncapped both tested across large trial counts (1000/10000 seeds). Minor: byte-indexing into UTF-8 strings is technically unsound for non-ASCII titles, but matches V0 behavior and is acceptable for scope. |
| B08 | 2.4 | qwen3.6-27b | Greedy playlist builder | ✅ Pass | — | Used camelot.Score correctly, no reimplementation. Scoring logic matches spec exactly. Tie-breaking by pool index is stable. Camelot preference test has minor fragility (doesn't control which track becomes seed), but passes reliably on the small test pool. All 6 test cases present. |
| B09 | 2.3+2.5+2.6 | qwen3.6-27b | Decoder + length encoding + round-trip test | ✅ Pass | — | 52k tokens, extensive self-correction. Root cause: round-trip test without a working encoder required inventing a brute-force keyword search to find PRNG states where the prefix decodes to a manageable length. Polecat correctly identified and worked around the problem. Multiple PRNG-sync reasoning errors self-corrected mid-session. Final solution (pre-searched keywords kw815→7-char message, kw1906→empty message) is correct and reproducible. "In parallel" language in output is thinking-mode leakage, not a correctness issue. Lesson: decoder-before-encoder ordering made the round-trip test the hardest part; Phase 3 encoder would have made this trivial. |
| B10 | 3.1 | qwen3.6-27b | Constrained search (internal/encoder) | ✅ Pass | — | Clean and fast. Correct clone discipline: rng.Clone() in satSatisfied, real rng only advanced after winner committed. PRNG sync test verifies via next-value comparison. Closure-based filter approach (pickBest taking func) hit a Go syntax issue on first attempt; self-corrected to explicit loops cleanly. Read builder_test.go before writing tests and renamed helper to avoid collision with existing makePool — good repo awareness. scoreTrack reused correctly, not reimplemented. All 6 tests present. |
| B11 | 3.2 | qwen3.6-27b | Constrained encoder (internal/encoder) | ⚠️ Partial — declared done | M | Three attempts total. Attempt 1 (~90 min): two algorithmic gaps in original spec. Attempt 2 (~40 min): partial file contamination caused reconciliation spiral. Attempt 3: Polecat implemented correct advance logic but round-trip tests failed silently (decoded wrong message, no error). Mayor investigation revealed two distinct problems: (1) FindBestTrack fallback silently commits unverified bytes when pool has no satisfying track — fixed by adding post-selection invariant check in encode.go; (2) diversePool() in the Bead spec uses modular arithmetic that clusters character distribution, leaving specific letter-pair combinations unreachable at certain PRNG states even with 1400 tracks. TestEmptyMessageRoundTrip passes (no message-phase constraint); TestEncodeDecodeRoundTrip fails at payload position 2 with "fh" unreachable. Declared done: encoder algorithm is correct and verified; fixture gap does not represent a production problem (real pool is the full Spotify library). Round-trip correctness gate deferred to B12 end-to-end test with a real pool from the database. Primary V1 finding already captured: human-courier Mayor/Polecat workflow is extremely labor-intensive on hard problems; failure detection and automated recovery are prerequisite for V2 viability. |
| B12 | 3.3 | qwen3.6-27b | CLI tools + end-to-end test | ❌ Killed | M | Immediately hit the same pool exhaustion error as B11 attempt 3. Polecat correctly identified the symptom (FindBestTrack fallback bypassing letter constraint) but began debug spiral rather than surfacing root cause. Killed early. Root cause identified by Mayor via V0 source comparison: V0 encodes one byte per track using a flat letter string (all a-z chars concatenated, position-indexed); V1 encodes two bytes per track using word-index/char-index draws that silently drop non-alpha characters. The 2-bytes-per-track design was a Mayor optimization introduced in B11 spec without methodological justification — it makes V1 a different algorithm from V0, not the same algorithm built differently. V1 closed as methodologically invalid. |

---

## V1 Post-Mortem

*Closed: June 2026. B01–B10 complete, B11 partial, B12 killed.*

### Conclusion

V1 cannot serve as a valid comparison to V0. The Mayor introduced two algorithmic deviations that made V1 a different system:

1. **2 bytes per track** (V0: 1 byte per track). Created a constraint satisfaction problem that no pool can reliably solve at arbitrary PRNG states.
2. **Indexed extraction** (V0: flat letter string). V1's `ExtractLetters` draws word index then char index, filtering non-alpha silently — result can be 0, 1, or 2 bytes even when count=2. V0's `ExtractFromTrack` draws position into a flat a-z string, always returning exactly `count` bytes.

Both deviations were Mayor-introduced in Bead specs. The Polecat implemented what it was told. All B01–B10 failures were M-classified.

### Polecat capability (within correctly specified Beads)

qwen3.6-27b performed well on B01–B10: idiomatic Go, correct package boundaries, self-correction on compilation errors, repo awareness, strong on well-defined algorithmic tasks. The capability ceiling was not reached. V1's failures were specification errors, not model failures.

### Primary finding

**Mayor discipline is a first-class requirement of the GasTown workflow.** The Mayor must verify algorithm-by-algorithm against the reference implementation before writing Beads. Without that discipline, the Mayor can invalidate the experiment by improving the design — and the Polecat cannot push back.

**Secondary finding:** The human-courier model is extremely labor-intensive on hard problems. Every M failure sits unresolved until human re-engagement. Partial file contamination, debug spirals, and cross-session context loss are real hazards with no automated mitigation in V1's architecture.

### Path forward

V2 reruns V1 with the methodological error corrected: same workflow, same Polecat, but with cross-version constraints 7 and 8 (flat letter string, one byte per track) explicit in the spec. V2's outcome determines whether V3 (automated recovery) is a throughput improvement or a capability requirement.

---

## V0 Phase 5

**Date:** 2026-06-14
**Elapsed:** ~45m
**Files changed:** 14
**Tests:** 16 → 32
**Notes:** One spec inconsistency found: TestWrongKeywordsDecode cannot be satisfied with the deterministic pool (cap3=1 always gives same letter regardless of PRNG state). Resolved by introducing rotatedPool() — 10-word rotated-alphabet titles, still cap3=1 but different letters at different positions. The pool supports deterministic encoding while guaranteeing wrong-keyword decoding extracts different content. The spec's round-trip encoder test (TestEncodeEmptyMessage) was placed in encode_external_test.go with package encoder_test to avoid the encoder↔decoder import cycle noted in V0 baseline. All 32 tests pass.
