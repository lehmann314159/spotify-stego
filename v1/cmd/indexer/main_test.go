package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/user/spotify-stego-v1/internal/audio"
	"github.com/user/spotify-stego-v1/internal/database"
	"github.com/user/spotify-stego-v1/internal/spotify"
)

func TestIndexerPipeline(t *testing.T) {
	spotifyServer, audioServer := startMockServers(t)
	defer spotifyServer.Close()
	defer audioServer.Close()

	sc := spotify.NewClient("test-id", "test-secret")
	sc.SetHTTPClient(spotifyServer.Client())
	sc.SetEndpoints(
		spotifyServer.URL+"/api/token",
		spotifyServer.URL,
	)

	ap := audio.NewGetSongBPMProvider("test-key")
	ap.SetHTTPClient(audioServer.Client())
	ap.SetBaseURL(audioServer.URL)

	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	cfg := indexerConfig{
		Genre: "indie",
		Limit: 2,
	}

	indexed, skipped, errs, err := runIndexer(cfg, sc, ap, dbPath)
	if err != nil {
		t.Fatalf("runIndexer failed: %v", err)
	}

	if indexed == 0 {
		t.Fatal("expected at least one track to be indexed")
	}

	if skipped == 0 {
		t.Fatal("expected at least one track to be skipped (ErrNotFound)")
	}

	db, err := database.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	tracks, err := database.GetTracksByGenre(db, "indie")
	if err != nil {
		t.Fatalf("get tracks by genre: %v", err)
	}

	if len(tracks) == 0 {
		t.Fatal("expected rows in the database")
	}

	for _, tr := range tracks {
		if tr.ID == "" {
			t.Error("track ID is empty")
		}
		if tr.Title == "" {
			t.Error("track Title is empty")
		}
		if tr.Artist == "" {
			t.Error("track Artist is empty")
		}
		if tr.Genre != "indie" {
			t.Errorf("expected genre indie, got %s", tr.Genre)
		}
		if tr.Tempo == 0 {
			t.Error("track Tempo is zero")
		}
		if tr.CamelotCode == "" {
			t.Error("track CamelotCode is empty")
		}
	}

	if indexed != len(tracks) {
		t.Errorf("indexed count %d does not match db row count %d", indexed, len(tracks))
	}

	_ = errs
}

func startMockServers(t *testing.T) (*httptest.Server, *httptest.Server) {
	spotifyMux := http.NewServeMux()

	tokenMu := sync.Mutex{}
	tokenCalled := false

	spotifyMux.HandleFunc("/api/token", func(w http.ResponseWriter, r *http.Request) {
		tokenMu.Lock()
		defer tokenMu.Unlock()

		if !tokenCalled {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "mock-token-123",
				"expires_in":   3600,
				"token_type":   "Bearer",
			})
			tokenCalled = true
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
	})

	spotifyMux.HandleFunc("/browse/categories/indie/playlists", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"playlists": map[string]interface{}{
				"items": []map[string]interface{}{
					{"id": "playlist1", "name": "Indie Vibes", "description": "chill indie"},
					{"id": "playlist2", "name": "Lo-fi Indie", "description": "more chill"},
				},
				"total": 2,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	spotifyMux.HandleFunc("/playlists/playlist1/tracks", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"items": []map[string]interface{}{
				{"track": map[string]interface{}{
					"id": "track1", "name": "Hello World", "duration_ms": 200000,
					"artists": []map[string]interface{}{{"name": "Mock Artist"}},
				}},
				{"track": map[string]interface{}{
					"id": "track2", "name": "Another Song", "duration_ms": 180000,
					"artists": []map[string]interface{}{{"name": "Another Artist"}},
				}},
			},
			"total": 2,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	spotifyMux.HandleFunc("/playlists/playlist2/tracks", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"items": []map[string]interface{}{
				{"track": map[string]interface{}{
					"id": "track3", "name": "Ghost Title", "duration_ms": 210000,
					"artists": []map[string]interface{}{{"name": "Ghost Artist"}},
				}},
			},
			"total": 1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	spotifyServer := httptest.NewServer(spotifyMux)

	audioMux := http.NewServeMux()
	audioMux.HandleFunc("/search/", func(w http.ResponseWriter, r *http.Request) {
		lookup := r.URL.Query().Get("lookup")

		if strings.Contains(lookup, "Ghost") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"search": []interface{}{},
			})
			return
		}

		resp := map[string]interface{}{
			"search": []map[string]interface{}{
				{
					"id":     "999",
					"title":  "",
					"artist": map[string]string{"name": ""},
					"tempo":  "115.5",
					"key_of": "C",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	audioServer := httptest.NewServer(audioMux)

	return spotifyServer, audioServer
}
