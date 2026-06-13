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

## V1 Log

| Bead | Phase | Model | Prompt Summary | Result | Failure Mode | Notes |
|------|-------|-------|----------------|--------|--------------|-------|
