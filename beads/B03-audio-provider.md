# Bead B03 — Audio Provider Interface + Stub + GetSongBPM

**Phase:** 1 (task 1.4)  
**Model:** qwen3.6-27b  
**Depends on:** B01  

---

## Preamble

Read `v1-preamble.md` in the repo root before starting. All constraints
and directory structure requirements are defined there. Work in `v1/` only.

---

## Task

Implement the audio data provider in `v1/internal/audio/`. This package
defines the interface, the stub implementation, and the real GetSongBPM
implementation. Business logic elsewhere in the codebase imports only the
interface — never the concrete types directly.

### Types and Interface

Create `v1/internal/audio/provider.go`:

```go
package audio

// AudioData holds BPM and key information for a track.
type AudioData struct {
    BPM         float64
    KeyOf       string // e.g. "Em", "C", "F#"
    CamelotCode string // e.g. "7A", "1B"
}

// Provider is the interface all audio data sources must implement.
type Provider interface {
    GetAudioData(title, artist string) (AudioData, error)
}
```

### Stub Implementation

Create `v1/internal/audio/stub.go`:

```go
package audio

// StubProvider returns fixed placeholder data for any input.
// It is the default implementation used until a real provider is wired in.
type StubProvider struct{}

func (s StubProvider) GetAudioData(title, artist string) (AudioData, error) {
    return AudioData{
        BPM:         120.0,
        KeyOf:       "C",
        CamelotCode: "8B",
    }, nil
}
```

### GetSongBPM Implementation

Create `v1/internal/audio/getsongbpm.go`.

GetSongBPM is a REST API for track BPM and key data. It does not accept
Spotify IDs — you must search by title and artist.

**API base URL:** `https://api.getsongbpm.com`

**Search endpoint:**
```
GET /search/?api_key={key}&type=song&lookup={title artist}
```
Returns a list of song matches. Pick the first result. Response shape:
```json
{
  "search": [
    {
      "id": "abc123",
      "title": "Master of Puppets",
      "artist": { "name": "Metallica" },
      "tempo": "220",
      "key_of": "Em",
      "open_key": "7m"
    }
  ]
}
```

**Song detail endpoint (if search result lacks tempo/key):**
```
GET /song/?api_key={key}&id={id}
```
Returns:
```json
{
  "song": {
    "id": "abc123",
    "tempo": "220",
    "key_of": "Em",
    "open_key": "7m"
  }
}
```

**Implementation requirements:**

```go
type GetSongBPMProvider struct {
    APIKey     string
    httpClient *http.Client
    baseURL    string // configurable for testing
}

func NewGetSongBPMProvider(apiKey string) *GetSongBPMProvider

func (g *GetSongBPMProvider) GetAudioData(title, artist string) (AudioData, error)
```

- Search by `title + " " + artist` as the lookup string
- If search returns no results, return `ErrNotFound`
- Parse `tempo` as float64 (it comes back as a string in the API)
- If `key_of` is empty in the search result, call the song detail endpoint
- The `open_key` field from GetSongBPM maps to Camelot notation:
  - Format is `{number}m` for minor (A keys) or `{number}d` for major (B keys)
  - e.g. `7m` → `7A`, `1d` → `1B`
  - Implement `openKeyToCamelot(openKey string) string` as an unexported helper
- Add a 10-second HTTP timeout
- Wrap all errors with context

**Sentinel error:**
```go
var ErrNotFound = errors.New("track not found in GetSongBPM")
```

### Test

Create `v1/internal/audio/audio_test.go`:

1. Test `StubProvider.GetAudioData` returns the expected fixed values
2. Test `GetSongBPMProvider.GetAudioData` using `httptest`:
   - Mock server returns a valid search response
   - Verify the returned `AudioData` has correct BPM, KeyOf, CamelotCode
3. Test the `openKeyToCamelot` helper directly:
   - `"7m"` → `"7A"`
   - `"1d"` → `"1B"`
   - `"12m"` → `"12A"`
   - empty string → `""`
4. Test that a search returning no results returns `ErrNotFound`

No real network calls. All tests use `httptest`.

## Exit Criteria

- [ ] `v1/internal/audio/provider.go` defines `AudioData`, `Provider`,
      `ErrNotFound`
- [ ] `v1/internal/audio/stub.go` defines `StubProvider` satisfying
      `Provider`
- [ ] `v1/internal/audio/getsongbpm.go` defines `GetSongBPMProvider`
      satisfying `Provider`
- [ ] `openKeyToCamelot` correctly maps `{n}m` → `{n}A` and `{n}d` → `{n}B`
- [ ] `go build ./...` passes
- [ ] `go test ./internal/audio/...` passes
