# spotify-stego

A steganography system that encodes messages in Spotify playlists using
track selection, ordering, and harmonic continuity (Camelot wheel) as the
encoding medium.

This repository is also a structured pilot of the GasTown Mayor/Polecat
pattern, comparing Claude Code (V0 baseline) against open model Polecats
(V1+) on a bounded, well-specified Go project.

---

## Structure

```
v0/   Claude Code baseline implementation (frozen after completion)
v1/   Guided Mayor + open model Polecat implementation
```

See `gastown_log.md` for the per-Bead execution record.  
See the AiHomeLab doc `gastown_spotify_stego_pilot.md` for the full
research design, version arc, and article plan.

---

## Design Constraints (all versions)

1. The BPM/key provider (GetSongBPM or equivalent) must be behind a Go
   interface — not called directly. This makes the provider swappable and
   keeps the cross-version comparison fair.
2. The default implementation is a stub returning deterministic placeholder
   data. Replace with a real provider when ready.
3. All versions implement against the same spec (`spotify-stego-plan.md`
   in AiHomeLab). Scope must not diverge between versions.

---

## Credentials

Copy `.env.example` to `.env` and fill in your values. Never commit `.env`.

```
SPOTIFY_CLIENT_ID=
SPOTIFY_CLIENT_SECRET=
GETSONGBPM_API_KEY=
```
