# Bead B05 — Indexer + Integration Test

**Phase:** 1 (tasks 1.6 + 1.7)  
**Model:** qwen3.6-27b  
**Depends on:** B01, B02, B03, B04  

---

## Preamble

Read `v1-preamble.md` in the repo root before starting. All constraints
and directory structure requirements are defined there. Work in `v1/` only.

---

## Task

Implement the track indexer binary and its integration test. This is the
first Bead that wires multiple packages together: Spotify client, audio
provider, Camelot mapping, and database.

### Binary: `v1/cmd/indexer/main.go`

The indexer fetches tracks from Spotify, looks up their BPM and key via
the audio provider, maps the key to a Camelot code, and stores everything
in SQLite.

**Command-line flags:**

```
-db      string   Path to SQLite database file (default: "spotify-stego.db")
-genre   string   Spotify genre/category ID to index (default: "indie")
-limit   int      Maximum number of playlists to process (default: 5)
```

**Credentials** (read from environment):
- `SPOTIFY_CLIENT_ID`
- `SPOTIFY_CLIENT_SECRET`
- `GETSONGBPM_API_KEY` (optional — uses stub if empty)

**Flow:**

```
1. Open (or create) the SQLite database
2. Create Spotify client from env credentials
3. If GETSONGBPM_API_KEY is set, use GetSongBPMProvider; otherwise use StubProvider
4. Fetch up to -limit playlists for the given genre
5. For each playlist:
   a. Fetch all tracks
   b. For each track:
      - Call provider.GetAudioData(title, artist)
      - If ErrNotFound: skip, increment skipped counter
      - If other error: log and increment error counter, continue
      - Map KeyOf to CamelotCode via camelot.KeyToCode
      - If key not recognized: use empty string for CamelotCode (don't skip)
      - Build core.Track and upsert into database
      - Increment indexed counter
   c. Log progress every 50 tracks: "indexed N tracks so far..."
6. Print summary: "Done. Indexed: X | Skipped: Y | Errors: Z"
```

**Notes:**
- Use `log` package for progress and error logging
- Do not crash on individual track errors — log and continue
- The indexer uses whatever audio provider is wired in; stub is fine for
  testing without real API keys

### Integration Test: `v1/cmd/indexer/main_test.go`

Write a test that exercises the full indexer pipeline end-to-end using
mock servers for both Spotify and GetSongBPM (no real network calls).

The test should:
1. Start an `httptest.Server` that mocks the Spotify token endpoint,
   the genre playlists endpoint, and the playlist tracks endpoint
2. Start a second `httptest.Server` that mocks the GetSongBPM search
   endpoint
3. Create a temporary SQLite database (`t.TempDir()`)
4. Run the indexer logic (extract the core pipeline into a testable
   function, or call `main` with appropriate env/flags)
5. Verify the database contains the expected tracks with correct fields

**Suggested approach:** Extract the indexer pipeline into a function:

```go
func runIndexer(cfg indexerConfig, spotifyClient *spotify.Client,
    provider audio.Provider, db *sql.DB) (indexed, skipped, errors int, err error)
```

Then `main` just parses flags, builds the dependencies, and calls
`runIndexer`. The test calls `runIndexer` directly with mock dependencies.

**Test assertions:**
- At least one track is inserted into the database
- The inserted track has non-empty ID, Title, Artist, Genre
- Tempo and CamelotCode are populated (from mock GetSongBPM response)
- Tracks returning ErrNotFound are skipped (skipped counter increments)
- The indexed counter matches the number of rows in the database

## Exit Criteria

- [ ] `v1/cmd/indexer/main.go` exists with flag parsing and `runIndexer`
      function
- [ ] Provider selected based on `GETSONGBPM_API_KEY` env var
- [ ] Progress logged every 50 tracks
- [ ] Summary printed on completion
- [ ] `v1/cmd/indexer/main_test.go` exercises full pipeline with mock
      servers, no real network calls
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
