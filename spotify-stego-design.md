# Spotify Playlist Steganography Project

## Core Concept

Hide messages in Spotify playlists by extracting specific letters from track titles, with the extraction pattern determined by a PRNG seeded from three keywords in the playlist name. Playlist track selection uses the Camelot wheel and BPM matching to produce genuinely coherent, listenable results.

---

## The Big Idea

**Encoding Method:** Extract letters from track titles at positions determined by a seeded PRNG. The same seed (three keywords) produces the same extraction sequence on both encoder and decoder sides — no pattern needs to be stored or communicated.

**Key Innovation:** Three unusual words in playlist title serve dual purpose:
1. Make the playlist easy to find/identify
2. Seed a PRNG that determines exactly which letters to extract from each track

**Why letter-level over word-level:**
- Word-level required every message word to exist somewhere in a song title — severely constrained what you could say
- Letter-level decouples track selection from message content entirely
- Any message is now possible regardless of catalog coverage
- Track selection can optimize purely for musicality
- Slightly better security characteristics (harder to reverse without the seed)

**Example:**
- Playlist: "Penguin Saxophone Velociraptor Indie Mix"
- PRNG seeded from "Penguin|Saxophone|Velociraptor"
- Track 1: "Somebody Told Me" → PRNG says: word 2, char 1 → **o**
- Track 2: "Under the Bridge" → PRNG says: word 1, char 4 → **e**
- Track 3: "Black Hole Sun" → PRNG says: word 3, char 2 → **u**
- ...continues until message is fully encoded
- Remaining tracks are padding (PRNG keeps advancing but output is discarded)

---

## Architecture Decisions

### Tech Stack
- **Frontend:** Go templates + HTMX (server-rendered, single binary)
- **Backend:** Go (HTTP server + indexer)
- **Database:** SQLite (simplified track pool cache)
- **Deployment:** AWS Lightsail (existing instance)
- **APIs:** Spotify Web API (track metadata) + GetSongBPM (audio features: BPM, key, mode)

### Why Go templates + HTMX over React?
The UI is fundamentally "forms and results" — two input forms, submit, display output. There's no client-side state to manage, no real-time updates, no complex component trees. The heavy work (constrained search, PRNG, Camelot scoring) all happens server-side anyway. HTMX handles the form submissions and swaps in the server-rendered results with no page reload, and the visualizations (Camelot wheel, BPM graph) are SVG rendered server-side per request. The payoff: single Go binary deployment, no npm, no build pipeline, no reverse proxy. Just `go build`, copy to Lightsail, done.

### Why SQLite (still)?
The database purpose has shifted — it's no longer an index for word lookups. It's now a **track pool cache** that pre-stores tracks and their audio features per genre. This is still worth having because:
- Every encode would otherwise hit Spotify API for tracks AND a separate call for audio features per track
- Rate limits would be a constant problem without caching
- Encode requests become fast local queries
- Background refresh keeps it current

The schema is much simpler now — no word parsing, no word tables. Just tracks with their musical properties pre-computed.

### Why Lightsail (still)?
- Simple deployment: single Go binary (serves routes, templates, and static CSS)
- Cheap: $5-10/month
- You already have one
- No complex orchestration needed

---

## Database Schema

```sql
CREATE TABLE tracks (
    id TEXT PRIMARY KEY,           -- Spotify track ID (22 chars)
    title TEXT NOT NULL,
    artist TEXT,
    genre TEXT,                    -- our genre classification
    duration INTEGER,              -- milliseconds
    tempo REAL,                    -- BPM from GetSongBPM
    key INTEGER,                   -- pitch class 0-11 (derived from key_of)
    mode TEXT,                     -- 'major' or 'minor' (derived from key_of)
    camelot_code TEXT,             -- pre-computed from key_of: '3A', '7B', etc.
    title_length INTEGER,          -- total character count (for extraction planning)
    spotify_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Primary query patterns for playlist building
CREATE INDEX idx_genre_camelot ON tracks(genre, camelot_code);
CREATE INDEX idx_genre_tempo ON tracks(genre, tempo);
```

That's it. No `track_words` table, no complex indexing. Just a well-organized pool of tracks with musical properties pre-computed and ready for playlist assembly.

---

## PRNG-Based Key Derivation

The three keywords are hashed to produce a seed, which drives a deterministic PRNG. Both encoder and decoder generate the exact same extraction sequence independently — nothing needs to be stored or communicated beyond the keywords.

```go
func deriveExtractor(keywords [3]string) *rand.Rand {
    combined := strings.Join(keywords[:], "|")
    hash := sha256.Sum256([]byte(combined))
    seed := binary.BigEndian.Uint64(hash[0:8])
    return rand.New(rand.NewSource(int64(seed)))
}
```

**Why this solves the "not enough values" problem:**
The earlier design tried to stretch hash bytes directly into a fixed repeating pattern (e.g., [2,3,2,4,2,3...]). That's limited by how many bytes you pull from the hash. A PRNG seeded from the hash produces an effectively infinite deterministic sequence. You never run out of extraction instructions, no matter how long the message or playlist.

---

## Letter Extraction Algorithm

Each track contributes a variable number of letters based on its title length. The PRNG determines both *how many* letters a track contributes and *which* letters those are.

**Message tracks vs padding tracks:** The constrained search gets harder exponentially as letters-per-track increases (candidate pool shrinks multiplicatively with each additional constraint). To keep the search viable, message tracks cap at 2 letters. Padding tracks don't care — their output is discarded anyway. The critical detail: the PRNG must advance identically on both encoder and decoder sides, so we always draw the full count and all positions, then discard extras for message tracks. Both sides know which tracks are message vs padding (encoder by construction, decoder by tracking extracted character count against the length prefix).

```go
func extractFromTrack(rng *rand.Rand, title string, isMessage bool) []byte {
    words := strings.Fields(title)
    if len(words) == 0 {
        return nil
    }

    // Always draw count from full range — keeps PRNG in sync on both sides
    maxLetters := min(3, longestWordLen(words))
    count := rng.Intn(maxLetters) + 1  // 1 to maxLetters

    // Message tracks cap at 2 to keep constrained search viable
    useCount := count
    if isMessage && useCount > 2 {
        useCount = 2
    }

    var letters []byte
    for i := 0; i < count; i++ {  // always iterate full count for PRNG sync
        wordIdx := rng.Intn(len(words))
        charIdx := rng.Intn(len(words[wordIdx]))
        if i < useCount {
            letters = append(letters, words[wordIdx][charIdx])
        }
        // else: PRNG advanced but letter discarded
    }
    return letters
}
```

**Important:** The PRNG advances the same way on both encoder and decoder sides regardless of whether letters are used or discarded. This keeps the sequences synchronized even though message and padding tracks extract different effective counts.

**Estimating playlist length for a message:**
- Message tracks: average ~1.5 letters per track (1-2 range, variable)
- 25-30 tracks with ~60-70% message tracks → ~25-30 letters of message capacity
- Longer messages: expand pool size (200k+ tracks) before increasing playlist length
- Short messages get padded with extra tracks to reach natural playlist length

---

## Camelot Wheel + BPM Playlist Coherence

This is the system DJs use to mix tracks seamlessly. Using it gives the playlist genuine musical coherence — not something we invented, but an established practice that makes the result sound intentional and natural.

### Camelot Compatibility Rules

From any position (e.g., 3A), compatible transitions are:

| Transition Type | Example (from 3A) | Score Weight |
|---|---|---|
| Same key+mode | 3A → 3A | Highest |
| Relative major/minor | 3A → 3B | High |
| Circle of fifths neighbor | 3A → 2A or 4A | Good |
| Diagonal (advanced) | 3A → 2B or 4B | Acceptable |
| Anything else | 3A → 8B | Avoid |

### BPM Compatibility

- Within ~6% BPM difference: smooth transition
- Exact doubles/halves work (60 → 120 BPM)
- Genre-dependent ranges: EDM clusters 120-140, indie 100-130, hip-hop 85-105

### GetSongBPM → Camelot Mapping

GetSongBPM returns `key_of` (e.g. "Em") and `open_key` (e.g. "2m"). The `open_key` field uses Open Key notation — similar to Camelot but with different numbering and m/M suffixes instead of A/B — so it's not a drop-in Camelot code. We use `key_of` instead, which gives us the human-readable key name and makes the lookup table straightforward:

```go
var keyToCamelot = map[string]string{
    // Minor keys (A suffix)
    "Am":  "1A",   "Bbm": "2A",   "Bm":  "3A",
    "Cm":  "4A",   "Dm":  "5A",   "Ebm": "6A",
    "Em":  "7A",   "Fm":  "8A",   "F#m": "9A",
    "Gm":  "10A",  "Abm": "11A",  "Dbm": "12A",

    // Major keys (B suffix)
    "C":   "1B",   "Db":  "2B",   "D":   "3B",
    "Eb":  "4B",   "E":   "5B",   "F":   "6B",
    "F#":  "7B",   "G":   "8B",   "Ab":  "9B",
    "A":   "10B",  "Bb":  "11B",  "B":   "12B",
}

func toCamelot(keyOf string) string {
    if code, ok := keyToCamelot[keyOf]; ok {
        return code
    }
    return ""  // unknown key — track skipped during indexing
}
```

This is actually simpler than the previous Spotify approach, which required composing a `"pitchClass-mode"` string key from two separate integer/string fields. GetSongBPM hands us the key name directly.

### Track Selection: Greedy Graph Walk

Playlist assembly is a greedy walk through a compatibility graph. Each track is a node; edges exist between musically compatible tracks. We start from a random seed track and greedily pick the best next track at each step.

```go
func buildPlaylist(pool []Track, targetLength int) []Track {
    playlist := []Track{pickRandom(pool)}  // seed track

    for len(playlist) < targetLength {
        current := playlist[len(playlist)-1]
        best := scoreCandidates(current, pool, playlist)
        playlist = append(playlist, best)
    }
    return playlist
}

func scoreCandidates(current Track, pool []Track, used []Track) Track {
    var bestTrack Track
    bestScore := -1.0

    for _, candidate := range pool {
        if alreadyUsed(candidate, used) { continue }

        score := 0.0

        // --- Camelot compatibility (weighted heavily) ---
        if candidate.CamelotCode == current.CamelotCode {
            score += 10.0   // same key+mode, always safe
        } else if isRelativeMajorMinor(current, candidate) {
            score += 8.0    // relative key swap
        } else if isCircleOfFifthsNeighbor(current, candidate) {
            score += 6.0    // adjacent on Camelot wheel
        } else if isDiagonal(current, candidate) {
            score += 3.0    // advanced DJ move
        }
        // else: score += 0, incompatible — avoid

        // --- BPM compatibility ---
        bpmDiff := math.Abs(current.Tempo - candidate.Tempo)
        bpmRatio := bpmDiff / current.Tempo
        if bpmRatio < 0.06 {
            score += 5.0 - (bpmRatio * 50)  // smooth falloff within 6%
        }

        // --- Prefer longer titles (more extraction potential) ---
        score += float64(candidate.TitleLength) * 0.1

        if score > bestScore {
            bestScore = score
            bestTrack = candidate
        }
    }
    return bestTrack
}
```

**Key insight:** Track selection is now completely independent of the message. The encoder picks the best *musical* playlist first, then the PRNG determines which letters get extracted. This is what makes the playlist sound natural — it's optimized purely for musicality.

---

## System Components

### 1. Track Indexer Service (Go)

Runs periodically (daily or on-demand) to refresh the track pool:

```
- Fetch popular playlists by genre from Spotify API (title, artist, track ID)
- For each track, query GetSongBPM for audio features (tempo, key_of)
- Derive Camelot code from key_of using lookup table
- Store in SQLite; skip tracks GetSongBPM doesn't have
- Target: 50k-200k tracks across supported genres
```

**Genres to prioritize** (diverse titles, good BPM ranges):
- Indie rock / alternative
- Electronic / EDM
- Hip-hop
- Pop rock

**Avoid:**
- Classical (opus numbers, very short "titles")
- Jazz standards (heavily repeated titles)

### 2. Encode Route (Go)

```go
POST /encode
Form data:
  message:  "meet at noon tuesday"
  genre:    "indie-rock"
  keyword1: "penguin"
  keyword2: "saxophone"
  keyword3: "velociraptor"

Response: HTML fragment (swapped into results div by HTMX)
  - Track list table (title, artist, Camelot code, BPM)
  - Stats sidebar (message length, tracks, musicality score, duration)
  - Camelot wheel SVG with playlist path highlighted
  - BPM graph SVG
  - Spotify embed iframe
  - "Copy playlist" button
```

**Encoding flow:**
1. Parse form, seed PRNG from keywords
2. Run constrained greedy search (Camelot + BPM + required letters)
3. Render results into `encode-results.html` template
4. HTMX swaps the fragment into the page — no reload

### 3. Decode Route (Go)

```go
POST /decode
Form data:
  playlistUrl: "https://open.spotify.com/playlist/xyz"
  keyword1:    "penguin"
  keyword2:    "saxophone"
  keyword3:    "velociraptor"

Response: HTML fragment (swapped into results div by HTMX)
  - Decoded message (large, prominent display)
  - Stats (tracks read, letters extracted)
  - Track list showing which letters were pulled from each track
```

**Decoding flow:**
1. Fetch playlist tracks from Spotify API (order matters!)
2. Seed PRNG from keywords (same seed = same sequence)
3. Walk through tracks, extract letters using PRNG
4. Stop at message length (length prefix terminator)
5. Render results into `decode-results.html` template
6. HTMX swaps the fragment into the page

### 4. Templates + HTMX Frontend

**Route structure:**
```go
GET  /              → base layout with Encode tab active
POST /encode        → runs encoder, returns encode-results fragment
POST /decode        → runs decoder, returns decode-results fragment
GET  /static/style.css
```

**Template hierarchy:**
```
base.html                   ← layout shell, nav tabs, loads HTMX CDN
├── encode.html             ← encode form (message, genre, keywords)
├── encode-results.html     ← partial: track list, stats, SVG visualizations
├── decode.html             ← decode form (playlist URL, keywords)
└── decode-results.html     ← partial: decoded message, per-track breakdown
```

**How HTMX wires it together:**

Tab switching — just swap content, no JS framework needed:
```html
<div class="tabs">
  <button hx-get="/encode" hx-target="#main-content" class="active">Encode</button>
  <button hx-get="/decode" hx-target="#main-content">Decode</button>
</div>
<div id="main-content">
  <!-- encode or decode form lives here -->
</div>
```

Form submission with loading indicator:
```html
<form hx-post="/encode" hx-target="#encode-results" hx-swap="innerHTML"
      hx-indicator="#encode-spinner">
  <textarea name="message" placeholder="Your secret message..."></textarea>
  <select name="genre">...</select>
  <input name="keyword1" placeholder="Word 1" />
  <input name="keyword2" placeholder="Word 2" />
  <input name="keyword3" placeholder="Word 3" />
  <button type="submit">Encode</button>
  <span id="encode-spinner" class="htmx-indicator">🔄 Building playlist...</span>
</form>
<div id="encode-results"><!-- server-rendered results swap in here --></div>
```

**Visualizations — server-rendered SVG:**

Camelot wheel: static geometry (two concentric rings of 12 segments each). The Go handler computes which nodes the playlist visited and which transitions were used, then renders the SVG with appropriate highlight classes. No client-side drawing needed.

```html
<!-- camelot-wheel.html (partial template) -->
<svg viewBox="0 0 400 400">
  {{range .Nodes}}
    <path d="..." class="node {{if .Visited}}visited{{end}} {{.Ring}}" />
    <text>{{.Label}}</text>
  {{end}}
  {{range .Transitions}}
    <line class="transition {{.Type}}" x1="..." y1="..." x2="..." x2="..." />
  {{end}}
</svg>
```

BPM graph: a polyline computed server-side from the track list. The Go handler calculates SVG coordinates from BPM values and passes the points string directly to the template.

```html
<!-- bpm-graph.html (partial template) -->
<svg viewBox="0 0 800 200">
  <polyline points="{{.BPMPolyline}}" fill="none" stroke="var(--accent)" />
  {{range .BPMLabels}}
    <text x="{{.X}}" y="{{.Y}}">{{.Value}}</text>
  {{end}}
</svg>
```

**One intentional limitation:**
The "character capacity indicator as you type" nice-to-have would need client-side JS. Not worth adding for a stretch goal — capacity shows up in the encode results after submission, which is fine.

---

## The Encoding Challenge

This is the most interesting technical problem in the project. Track selection is optimized for musicality, but the PRNG extraction is deterministic — it will pull specific positions from specific titles. The encoder can't just pick the best musical playlist and hope the letters work out.

**The solution: constrained search with fallbacks.**

For each track position in the playlist:
1. Determine what letter(s) the PRNG will extract from this position (capped at 2 for message tracks)
2. Find candidate tracks that: (a) are musically compatible with the previous track, AND (b) have the required letters at the required positions
3. Pick the best candidate by musicality score
4. If no candidate satisfies both constraints, relax the Camelot constraint (allow less compatible keys) or suggest a message edit
5. Once the message is fully encoded, remaining tracks are padding — unconstrained, pure musicality optimization

This turns playlist building from a pure greedy walk into a **constrained greedy search** — still tractable, but more interesting algorithmically. The 2-letter cap on message tracks is what keeps it tractable: candidate pools stay large enough that you can usually find a musically compatible track too.

```go
func buildEncodedPlaylist(pool []Track, message string, keywords [3]string, targetLength int) []Track {
    rng := deriveExtractor(keywords)
    playlist := []Track{}
    msgIdx := 0  // current position in message
    messageComplete := false

    for len(playlist) < targetLength {
        var prev *Track
        if len(playlist) > 0 {
            prev = &playlist[len(playlist)-1]
        }

        if !messageComplete {
            // Determine what this track needs to contribute (max 2 letters)
            needed := lettersNeededAtPosition(rng, message, msgIdx)
            
            // Find best track: musically compatible AND has required letters
            best := findBestTrack(pool, prev, needed, playlist)
            playlist = append(playlist, best)
            
            msgIdx += len(needed)
            if msgIdx >= len(message) {
                messageComplete = true
            }
        } else {
            // Padding territory: advance PRNG (to stay in sync) but pick purely on musicality
            advancePRNG(rng)  // keeps sequence synchronized with decoder
            best := findBestTrack(pool, prev, nil, playlist)
            playlist = append(playlist, best)
        }
    }
    return playlist
}
```

---

## Message Termination

The decoder needs to know when the message ends and padding begins. Options:

1. **Fixed length prefix:** First few extracted letters encode the message length as a number
   - Simple, reliable
   - Costs a few characters of capacity

2. **Null terminator:** A specific extracted letter sequence signals end-of-message
   - Derived from the PRNG seed so it's key-dependent
   - Slightly more elegant

3. **Out-of-band:** Message length shared separately (e.g., in playlist description)
   - Simplest to implement
   - Slightly less elegant from a steganography perspective

**Recommendation for MVP:** Option 1. Encode message length as first 3 extracted characters (supports messages up to 999 characters). Decoder reads length first, then extracts exactly that many characters.

---

## Security Model

**Threat Model:** Educational demonstration, NOT covert communication.

**Design choice:** Keywords are public (in playlist title). Security is through obscurity — the algorithm isn't widely known.

**This is a feature, not a bug.** The project demonstrates steganography concepts honestly. You can blog about the exact algorithm and include a live demo.

**Attack vectors (interesting for blog discussion):**
- Without the keywords, attacker must guess which three words seed the PRNG
- Statistical analysis: playlists with unusual title-length distributions or genre consistency might be suspicious
- Brute force: if attacker knows the algorithm, they could try common three-word combos against any playlist
- Playlist tampering: reordering or swapping tracks breaks decoding (order-dependent)

---

## Blog Post Ideas

1. **"I Hid Messages in Spotify Playlists"**
   - The idea, the pivot from words to letters, why it matters
   - How PRNGs make steganography practical
   - Live demo with example playlists

2. **"Making Steganographic Playlists Sound Good: The Camelot Wheel"**
   - What Camelot is and why DJs use it
   - How we integrated it into constrained search
   - Before/after: random track selection vs. Camelot-guided

3. **"The Constrained Search Problem"**
   - The tension between musicality and message encoding
   - How the greedy constrained search works
   - Edge cases and fallback strategies

4. **"Which Music Genre is Best for Hiding Messages?"**
   - Title length and vocabulary diversity analysis
   - Genre comparison: capacity per playlist
   - Character frequency distributions

5. **"How to Detect a Steganographic Playlist"**
   - Statistical anomalies to look for
   - What makes a playlist look "too perfect"
   - The cat-and-mouse game

---

## Project Structure

```
spotify-stego/
├── cmd/
│   ├── server/               # HTTP server (routes, handlers, template rendering)
│   └── indexer/              # Track scraping + audio feature fetcher
├── internal/
│   ├── database/             # SQLite operations (track pool)
│   ├── spotify/              # Spotify API client (track metadata)
│   ├── getsongbpm/           # GetSongBPM API client (audio features)
│   ├── camelot/              # Camelot wheel logic + compatibility scoring + SVG generation
│   ├── encoder/              # Constrained playlist building + letter encoding
│   ├── decoder/              # Letter extraction from playlist
│   └── crypto/               # PRNG seeding from keywords
├── templates/
│   ├── base.html             # Layout shell, tabs, HTMX CDN
│   ├── encode.html           # Encode form
│   ├── encode-results.html   # Partial: track list, stats, Camelot SVG, BPM SVG
│   ├── decode.html           # Decode form
│   └── decode-results.html   # Partial: decoded message, per-track breakdown
├── static/
│   └── style.css             # All styles in one place
├── database/
│   ├── schema.sql
│   └── spotify-stego.db      # SQLite file
├── blog-examples/
│   └── sample-playlists.md
├── go.mod
└── README.md
```

---

## Implementation Plan

### Phase 1: Foundation (Week 1)
- [ ] Project structure: Go modules, basic HTTP server
- [ ] SQLite schema + migrations
- [ ] Spotify API client (app credentials, track fetching by genre)
- [ ] GetSongBPM API client (audio features: tempo, key_of → Camelot)
- [ ] Basic indexer: scrape and cache 10k tracks in one genre
- [ ] Camelot mapping table + compatibility functions

### Phase 2: Core Algorithm (Week 1-2)
- [ ] PRNG key derivation from keywords
- [ ] Letter extraction logic (variable per track)
- [ ] Message termination (length prefix)
- [ ] Decoder: given a playlist + keywords, extract message
- [ ] Greedy playlist builder (musicality only, no message constraints yet)

### Phase 3: Constrained Encoding (Week 2)
- [ ] Constrained greedy search: musicality + required letters
- [ ] Fallback strategies when no perfect candidate exists
- [ ] End-to-end test: encode "hello world", verify decode
- [ ] CLI tools for quick testing

### Phase 4: Frontend (Week 3)
- [ ] Base template with layout, tabs, HTMX CDN
- [ ] Encode form + decode form templates
- [ ] Wire HTMX: form submissions swap in result partials
- [ ] Encode results partial: track list table + stats
- [ ] Decode results partial: decoded message display
- [ ] Camelot wheel SVG rendering (Go computes node highlights + transition paths)
- [ ] BPM graph SVG rendering (Go computes polyline from track tempos)
- [ ] Loading indicator (hx-indicator) while constrained search runs
- [ ] Spotify embed iframe in encode results
- [ ] style.css: clean, readable design

### Phase 5: Polish + Deploy (Week 3-4)
- [ ] Spotify OAuth for auto-creating playlists in user's account
- [ ] Musicality score display
- [ ] BPM graph visualization
- [ ] Deploy to Lightsail
- [ ] Multi-genre support + expanded track database
- [ ] Write blog post(s)

### Stretch Goals
- [ ] "Detect steganography" tool (for demonstration)
- [ ] Substitution cipher layer on extracted letters
- [ ] Mobile-friendly PWA
- [ ] Capacity analyzer: "how many characters can this genre hold?"
- [ ] Easter eggs: hide famous quotes in playlists, see if anyone finds them

---

## Why This Project is Great

✅ Teaches steganography principles hands-on
✅ Teaches music theory (Camelot wheel) as a side effect
✅ Interesting algorithm design (constrained greedy search)
✅ PRNGs and deterministic sequences
✅ Real-world API integration (Spotify)
✅ Fun to demo — actual listenable playlists with hidden messages
✅ Good portfolio piece
✅ Multiple blog post angles
✅ Uses tech you're learning (Go, HTMX, SQLite)
✅ Actually original — haven't seen this done before
✅ Scalable complexity: MVP works, then layers of sophistication on top

---

*Last updated: Day 3. Spotify deprecated audio features API (Nov 2024) — replaced with GetSongBPM as audio features source. GetSongBPM provides tempo, key_of, and open_key for free (backlink required). Camelot wheel fully intact: key_of maps directly to Camelot codes via a simpler lookup table than the previous Spotify pitch-class approach. Added getsongbpm package to project structure. Previous updates: switched frontend from React to Go templates + HTMX, shifted from word-level to letter-level extraction, added PRNG-based key derivation, simplified database to track pool cache, integrated Camelot wheel + BPM for playlist coherence.*
