# Spotify Stego — V0

Steganography via Spotify playlists. A hidden message is encoded into a playlist of real Spotify tracks by exploiting the relationship between a secret set of keywords and the letters that appear in track titles. The Camelot wheel ensures consecutive tracks are harmonically compatible, making the playlist sound intentional.

## How it works

1. Three keywords are joined and hashed with SHA256, seeding a deterministic Xorshift64 PRNG.
2. For each character in the message (preceded by a 3-character base-26 length prefix), the PRNG determines how many letters to extract from a candidate track's title and which positions to pick.
3. The encoder searches the track pool for a track that, at the current PRNG state, yields the required letter — preferring tracks with high Camelot wheel and BPM compatibility with the previous track.
4. The decoder replays the same PRNG sequence against the playlist and reads off the same letters, recovering the message.
5. **Pool size note:** below ~500 tracks per genre, some letters become unreachable at specific PRNG states and encode will fail.

## Prerequisites

- Go 1.26+
- A Spotify Developer App (free) — needed for indexing and OAuth playlist creation
- For OAuth: configure a Redirect URI in your Spotify app dashboard pointing at `/auth/spotify/callback` on wherever the server is hosted

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `SPOTIFY_CLIENT_ID` | required | Spotify app client ID |
| `SPOTIFY_CLIENT_SECRET` | required | Spotify app client secret |
| `SPOTIFY_REDIRECT_URI` | required for OAuth | Full URL to `/auth/spotify/callback` on your host |
| `DB_PATH` | `tracks.db` | SQLite database path |
| `ADDR` | `:8080` | Server listen address |
| `SPOTIFY_PLAYLIST_PRIVATE` | `false` | Set `true` to create private playlists |

## Running the indexer

```
cd v0
go run ./cmd/indexer -genre pop -db tracks.db
```

**Note:** audio data (BPM, key) uses a stub provider returning placeholder values. Real data requires a GetSongBPM API key wired into the indexer.

## Running the server

```
cd v0    # required: templates load relative to CWD
go run ./cmd/server
```

Set `SPOTIFY_REDIRECT_URI` before using the Save to Spotify feature.

## CLI tools

### encode-cli

Encodes a message into a playlist drawn from the local SQLite database.

Flags:
- `-message` — the plaintext message to encode
- `-genre` — genre to search in the database
- `-k1`, `-k2`, `-k3` — three secret keywords
- `-db` — path to SQLite database (default `tracks.db`)
- `-target` — target playlist length (default 20)

Example:
```
go run ./cmd/encode-cli -message "hello world" -genre pop -k1 cat -k2 dog -k3 bird
```

### decode-cli

Decodes a playlist of track titles back to the hidden message.

Flags:
- `-tracks` — newline-separated list of track titles (one per line)
- `-k1`, `-k2`, `-k3` — three secret keywords matching those used to encode

Example:
```
go run ./cmd/decode-cli -k1 cat -k2 dog -k3 bird <<EOF
Track One
Track Two
EOF
```

## Architecture

| Package | Purpose |
|---|---|
| `internal/spotify` | Spotify API client: client credentials + PKCE OAuth |
| `internal/audio` | AudioDataProvider interface, stub, GetSongBPM real impl |
| `internal/database` | SQLite schema and track persistence |
| `internal/camelot` | Camelot wheel mapping, scoring, SVG renderer |
| `internal/encoder` | PRNG, letter extraction, greedy builder, constrained search, encoder |
| `internal/decoder` | Playlist decoder |
| `internal/integration` | End-to-end round-trip tests |
| `cmd/server` | HTMX web server |
| `cmd/indexer` | Genre indexer |
| `cmd/encode-cli` | CLI encoder |
| `cmd/decode-cli` | CLI decoder |

## Known limitations

- Templates load relative to CWD: server must run from `v0/`.
- Audio data (BPM, key) is stub unless GetSongBPM is wired.
- OAuth token is in-memory only — lost on server restart.
- Pool size: encode quality degrades below ~500 tracks per genre.
