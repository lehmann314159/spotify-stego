# Bead B01 — Scaffold + SQLite Schema

**Phase:** 1 (tasks 1.1 + 1.2)  
**Model:** qwen3.6-27b  
**Depends on:** nothing  

---

## Preamble

Read `v1-preamble.md` in the repo root before starting. All constraints
and directory structure requirements are defined there. Work in `v1/` only.

---

## Task

Set up the Go project scaffold and SQLite schema for the spotify-stego V1
implementation.

### 1. Go Module + Directory Structure

Initialize the Go module at `v1/`:

```
cd v1
go mod init github.com/user/spotify-stego-v1
```

Create the following directory structure. Directories must exist (add
`.gitkeep` files where needed to ensure they are tracked):

```
v1/
  cmd/
    server/
    indexer/
    encode-cli/
    decode-cli/
  internal/
    stego/
      core/
    spotify/
    audio/
    camelot/
    database/
    encoder/
    decoder/
  templates/
  static/
```

Create a `v1/.gitignore` containing:
```
*.db
*.db-shm
*.db-wal
vendor/
```

### 2. SQLite Schema + Migrations

Use `modernc.org/sqlite` (pure Go, no CGo). Add it to go.mod:

```
go get modernc.org/sqlite
```

Create `v1/internal/database/schema.go` containing the schema SQL as a
constant and a `Migrate(db *sql.DB) error` function that creates the
`tracks` table and indexes if they do not exist.

Schema:

```sql
CREATE TABLE IF NOT EXISTS tracks (
    id           TEXT PRIMARY KEY,
    title        TEXT NOT NULL,
    artist       TEXT NOT NULL,
    genre        TEXT NOT NULL,
    duration_ms  INTEGER,
    tempo        REAL,
    key_of       TEXT,
    camelot_code TEXT
);

CREATE INDEX IF NOT EXISTS idx_genre_camelot ON tracks (genre, camelot_code);
CREATE INDEX IF NOT EXISTS idx_genre_tempo   ON tracks (genre, tempo);
```

Create `v1/internal/database/db.go` with:
- `Open(path string) (*sql.DB, error)` — opens the SQLite file and calls
  `Migrate`
- `UpsertTrack(db *sql.DB, t Track) error` — inserts or replaces a track
- `GetTracksByGenre(db *sql.DB, genre string) ([]Track, error)` — returns
  all tracks for a genre

The `Track` struct belongs in `v1/internal/stego/core/types.go` (shared
package, imported by database, encoder, decoder, and everything else):

```go
package core

type Track struct {
    ID          string
    Title       string
    Artist      string
    Genre       string
    DurationMs  int
    Tempo       float64
    KeyOf       string
    CamelotCode string
}
```

### 3. Verify

- `cd v1 && go build ./...` must succeed with no errors
- `go vet ./...` must produce no warnings
- Write a test in `v1/internal/database/db_test.go` that:
  1. Opens an in-memory SQLite database (`:memory:`)
  2. Calls `Migrate`
  3. Upserts one track
  4. Calls `GetTracksByGenre` and verifies the track is returned
- `go test ./internal/database/...` must pass

## Exit Criteria

- [ ] `v1/go.mod` exists with module name and `modernc.org/sqlite` dependency
- [ ] All directories exist
- [ ] `v1/internal/stego/core/types.go` defines `Track`
- [ ] `v1/internal/database/schema.go` defines schema SQL and `Migrate`
- [ ] `v1/internal/database/db.go` defines `Open`, `UpsertTrack`,
      `GetTracksByGenre`
- [ ] `go build ./...` passes
- [ ] `go test ./internal/database/...` passes
