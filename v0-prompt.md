# V0 Prompt — Claude Code Baseline

This is the exact prompt handed to Claude Code for the V0 implementation.
It is preserved verbatim as part of the research record.

---

You are implementing a Spotify playlist steganography system in Go. This is
a complete, production-quality implementation. Work inside the `v0/`
directory of this repository.

## What You Are Building

A system that hides text messages inside Spotify playlists. The encoding
works as follows:

- A shared secret consisting of three keywords is used to seed a PRNG
- The PRNG determines which letters to extract from each track title
- The encoder selects tracks whose titles yield the correct letters at the
  correct positions, while also optimizing for musical coherence using the
  Camelot wheel (a DJ mixing system based on musical key compatibility)
- The decoder, given the same keywords and the playlist, reconstructs the
  message deterministically

The playlist sounds like a normal, musically coherent playlist to anyone
who doesn't know the keywords.

## Tech Stack

- Go (single binary)
- SQLite via `modernc.org/sqlite` (pure Go, no CGo)
- HTMX for frontend interactivity (CDN, no build step)
- Go `html/template` for server-side rendering
- Spotify Web API (client credentials flow) for track data
- BPM/key data provider behind an interface (see constraints below)

## Cross-Version Constraints

These constraints exist to keep this implementation comparable to future
versions built by open model Polecats. Do not deviate from them:

1. **Provider interface required.** BPM and musical key data must be
   accessed through a Go interface, not via direct API calls in business
   logic. Define an interface such as:

   ```go
   type AudioDataProvider interface {
       GetAudioData(title, artist string) (AudioData, error)
   }

   type AudioData struct {
       BPM       float64
       KeyOf     string // e.g. "Em", "C"
       CamelotCode string // e.g. "7A", "1B"
   }
   ```

   The concrete implementation (GetSongBPM client or otherwise) satisfies
   this interface. Business logic imports the interface only.

2. **Stub is the default implementation.** Implement a stub that satisfies
   the AudioDataProvider interface and returns deterministic placeholder
   data. Wire the stub by default. The real provider is a drop-in
   replacement. This allows the full pipeline to be exercised without
   live API access.

3. **Scope:** Implement Phases 1 through 4 of the spec below. Phase 5
   (deploy, OAuth, multi-genre indexing) is explicitly out of scope for V0.

## Full Specification

### Directory Structure

Produce a standard Go project layout inside `v0/`. Use `cmd/` for binaries
and `internal/` for packages. Suggested structure:

```
v0/
  cmd/
    server/       HTTP server (Phase 4)
    indexer/      Track indexer (Phase 1)
    encode-cli/   CLI encoder (Phase 3)
    decode-cli/   CLI decoder (Phase 3)
  internal/
    spotify/      Spotify API client
    audio/        AudioDataProvider interface + stub + real implementation
    camelot/      Camelot wheel mapping and SVG renderer
    database/     SQLite schema and migrations
    encoder/      PRNG, letter extraction, greedy builder, constrained search
    decoder/      Playlist decoder
  templates/      Go HTML templates
  static/         CSS
```

### Phase 1: Foundation

**SQLite schema** (`tracks` table):
- `id` TEXT PRIMARY KEY (Spotify track ID)
- `title` TEXT NOT NULL
- `artist` TEXT NOT NULL
- `genre` TEXT NOT NULL
- `duration_ms` INTEGER
- `tempo` REAL
- `key_of` TEXT
- `camelot_code` TEXT

Indexes: `idx_genre_camelot` on (genre, camelot_code), `idx_genre_tempo`
on (genre, tempo).

**Spotify client** (`internal/spotify/client.go`):
- Client credentials OAuth flow (no user login required)
- `GetPopularPlaylistsByGenre(genre string) ([]Playlist, error)`
- `GetPlaylistTracks(playlistID string) ([]Track, error)`
- Extracts: ID, Title, Artist, Duration
- Rate limit logging if approaching 150 req/min

**AudioDataProvider interface** (`internal/audio/`):
- Interface as defined above
- Stub implementation: returns BPM=120.0, KeyOf="C", CamelotCode="8B" for
  any input. Deterministic, no network calls.
- Real implementation: GetSongBPM API client
  - `SearchSong(title, artist string) (id string, err error)`
  - `GetSong(id string) (AudioData, error)`
  - Match by title+artist, pick highest-confidence result

**Camelot mapping** (`internal/camelot/mapping.go`):
- Complete lookup table: all 24 keys (12 major, 12 minor) to Camelot codes
- `KeyToCode(keyOf string) (string, error)`
- Standard DJ notation: A minor = 1A, A♭ major = 1B, etc.

**Indexer** (`cmd/indexer/`):
- Fetches playlists by genre from Spotify
- For each track, calls AudioDataProvider.GetAudioData
- Inserts into SQLite; skips on not-found, logs errors
- Progress logging every 50 tracks
- Summary on completion: indexed / skipped / errors

### Phase 2: Core Algorithm

**PRNG and key derivation** (`internal/encoder/prng.go`):
- `DeriveExtractor(keywords [3]string) *rand.Rand`
- SHA256 hash the joined keywords, seed from first 8 bytes as uint64
- Same keywords must produce identical sequences across calls and restarts

**Letter extraction** (`internal/encoder/extract.go`):
- `ExtractFromTrack(rng *rand.Rand, title string) []byte`
- PRNG determines count (1 to max word length in title, capped at 3)
- PRNG determines which positions to extract
- Always advance PRNG fully even if fewer letters are needed (keeps
  encoder and decoder in sync)

**Message length encoding**:
- 3-digit zero-padded decimal: message of length 25 → "025"
- Encoded as the first letters extracted from the playlist
- Decoder reads first 3 extracted letters as the length prefix

**Decoder** (`internal/decoder/decode.go`):
- `DecodePlaylist(tracks []Track, keywords [3]string) (string, error)`
- Seeds PRNG identically to encoder
- Walks tracks, extracts letters
- First 3 letters = message length (parse as decimal)
- Next N letters = message content
- Returns decoded string

**Greedy playlist builder** (`internal/encoder/greedy.go`):
- `BuildPlaylistGreedy(pool []Track, targetLength int, seed int64) ([]Track, error)`
- Starts with a random seed track
- Each step: scores all remaining candidates by Camelot compatibility and
  BPM smoothness, picks best
- Camelot scoring:
  - Same key: 10 points
  - Relative major/minor: 8 points
  - Circle of fifths neighbor (±1 on wheel): 6 points
  - Diagonal (same number, opposite letter): 3 points
  - No relationship: 0 points
- BPM scoring: full points within 6% of current BPM, falling off beyond
- No track repeats

### Phase 3: Constrained Encoding

**Constrained track search** (`internal/encoder/constrained.go`):
- `FindConstrained(prev *Track, needed []byte, pool []Track, used map[string]bool, rng *rand.Rand) (*Track, error)`
- Filters pool to tracks whose titles yield the required letters at the
  required PRNG-determined positions
- Among valid candidates, scores by Camelot + BPM compatibility
- Returns best match or error

**Fallback chain** (`internal/encoder/fallback.go`):
1. Strict: Camelot compatibility + exact letters required
2. Relax Camelot: any key in pool, letters still required
3. Error with diagnostics: which letter failed, pool size, candidates tried

**Full encoder** (`internal/encoder/encode.go`):
- `EncodeMessage(pool []Track, message string, keywords [3]string, targetLength int) ([]Track, error)`
- Encodes length prefix first (3 characters)
- For each message character: find constrained track
- Fills remainder with greedy unconstrained tracks
- Logs fallback usage

**CLI tools**:
- `cmd/encode-cli/`: accepts message, genre, keywords; outputs track list
- `cmd/decode-cli/`: accepts track list, keywords; outputs message

### Phase 4: Frontend

**HTTP server** (`cmd/server/main.go`):
- Routes: `GET /`, `POST /encode`, `POST /decode`, `GET /static/`
- Logging middleware
- Reads `.env` for credentials (SPOTIFY_CLIENT_ID, SPOTIFY_CLIENT_SECRET)

**Templates** (`templates/`):
- `base.html`: HTML5 shell, HTMX from CDN, tab structure (Encode / Decode)
- `encode.html`: form with message textarea, genre select, three keyword inputs
- `decode.html`: form with playlist track list input and keyword inputs
- `encode-results.html`: partial returned by POST /encode
  - Track list table (title, artist, Camelot code, BPM)
  - Stats: message length, playlist length, musicality score, total duration
  - Camelot wheel SVG (server-rendered, highlights visited nodes and
    transitions)
  - BPM graph SVG (server-rendered polyline)
- `decode-results.html`: partial returned by POST /decode
  - Decoded message (prominent)
  - Per-track breakdown: title → letters extracted

**Camelot wheel SVG** (`internal/camelot/svg.go`):
- `RenderWheelSVG(tracks []Track) string`
- Two concentric rings, 12 segments each (outer = A keys, inner = B keys)
- Segments labeled 1A-12A and 1B-12B
- Visited segments highlighted
- Transition arrows between consecutive tracks, color-coded by compatibility

**BPM graph SVG** (`internal/encoder/bpmgraph.go`):
- `RenderBPMGraphSVG(tracks []Track) string`
- viewBox 800×200
- Polyline connecting BPM values, Y-axis scaled to playlist range
- Grid lines and axis labels
- Track titles at each point

**CSS** (`static/style.css`):
- System font stack
- CSS custom properties for colors
- Clean, minimal design
- Responsive (flexbox)
- SVGs inherit CSS color variables

## Credentials

Read from `.env` in the working directory (or environment variables):
- `SPOTIFY_CLIENT_ID`
- `SPOTIFY_CLIENT_SECRET`
- `GETSONGBPM_API_KEY` (only needed when wiring real AudioDataProvider)

Use `github.com/joho/godotenv` or equivalent to load `.env`.

## Testing

Write tests for:
- PRNG reproducibility (same keywords → same sequence)
- Letter extraction determinism
- Camelot compatibility functions (all 24 keys)
- Encode/decode round-trip: encode "hello world", decode it back
- Constrained search: verify a found track actually yields the required letters
- Fallback triggering: construct a scenario that forces the fallback chain

## Quality Bar

This is the baseline all future versions are measured against. Write code
you would be proud to ship: clear package boundaries, meaningful error
messages, no panics on bad input, idiomatic Go. The architecture decisions
you make here become the reference for evaluating V1 output quality.
