# Bead B02 — Spotify API Client

**Phase:** 1 (task 1.3)  
**Model:** qwen3.6-27b  
**Depends on:** B01  

---

## Preamble

Read `v1-preamble.md` in the repo root before starting. All constraints
and directory structure requirements are defined there. Work in `v1/` only.

---

## Task

Implement the Spotify API client in `v1/internal/spotify/client.go`.

The client uses the **Client Credentials OAuth flow** — no user login
required. It authenticates with a client ID and secret and gets a bearer
token for read-only access to public Spotify data.

### Types

Define these types in `v1/internal/spotify/client.go`:

```go
type Playlist struct {
    ID   string
    Name string
}

type Track struct {
    ID         string
    Title      string
    Artist     string
    DurationMs int
}
```

Note: this is a Spotify-specific Track type used only within this package.
It is distinct from `core.Track` in `internal/stego/core/types.go`.

### Client

```go
type Client struct { /* unexported fields */ }

func NewClient(clientID, clientSecret string) *Client
```

`NewClient` does not make any network calls. Authentication is lazy —
the token is fetched on the first API call and refreshed when expired.

### Methods

```go
// GetPopularPlaylistsByGenre returns Spotify's featured playlists for a
// given genre/category ID (e.g. "indie", "edm", "pop").
// Returns up to 20 playlists.
func (c *Client) GetPopularPlaylistsByGenre(genre string) ([]Playlist, error)

// GetPlaylistTracks returns all tracks in a playlist, handling Spotify's
// pagination (100 tracks per page). Returns only tracks where both title
// and artist are non-empty.
func (c *Client) GetPlaylistTracks(playlistID string) ([]Track, error)
```

### Authentication

Use the Client Credentials flow:
- POST to `https://accounts.spotify.com/api/token`
- Body: `grant_type=client_credentials`
- Authorization: Basic base64(clientID:clientSecret)
- Response contains `access_token` and `expires_in`
- Cache the token; refresh automatically when within 60 seconds of expiry

### Rate Limiting

Spotify allows ~150 requests/minute on client credentials. Add a log line
(using the standard `log` package) if a 429 response is received. On 429,
wait the number of seconds specified in the `Retry-After` header (default
to 1 second if header is absent) and retry once.

### Spotify API Endpoints

- Playlists by category:
  `GET https://api.spotify.com/v1/browse/categories/{genre}/playlists?limit=20`
- Playlist tracks (paginated):
  `GET https://api.spotify.com/v1/playlists/{playlistID}/tracks?limit=100&offset={offset}`

From the playlist tracks response, extract:
- `items[].track.id` → Track.ID
- `items[].track.name` → Track.Title  
- `items[].track.artists[0].name` → Track.Artist
- `items[].track.duration_ms` → Track.DurationMs

Skip items where `items[].track` is null (Spotify includes null entries for
local files and unavailable tracks).

### Error Handling

- Wrap all errors with context: `fmt.Errorf("get playlists: %w", err)`
- Return a descriptive error if the genre/category returns no playlists
- Do not panic on any input

### Credentials

Read `SPOTIFY_CLIENT_ID` and `SPOTIFY_CLIENT_SECRET` from environment
variables. The client itself takes them as constructor arguments — the
binary (not this package) is responsible for reading the environment.

### Test

Write `v1/internal/spotify/client_test.go` with at least one test that
verifies the token parsing logic using a mock HTTP server (use
`net/http/httptest`). The test must not make real network calls.

Example: create an httptest server that returns a valid token response,
point the client at it, call a method, verify it uses the token correctly.

## Exit Criteria

- [ ] `v1/internal/spotify/client.go` exists with `Client`, `Playlist`,
      `Track` types and `NewClient`, `GetPopularPlaylistsByGenre`,
      `GetPlaylistTracks`
- [ ] Token is fetched lazily and refreshed before expiry
- [ ] Pagination handled in `GetPlaylistTracks`
- [ ] 429 rate limit handling with `Retry-After` respect
- [ ] Test uses `httptest` and makes no real network calls
- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
