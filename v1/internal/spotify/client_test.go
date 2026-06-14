package spotify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestNewClientNoNetwork(t *testing.T) {
	c := NewClient("test-id", "test-secret")
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
	if c.clientID != "test-id" {
		t.Errorf("expected clientID test-id, got %s", c.clientID)
	}
	if c.clientSecret != "test-secret" {
		t.Errorf("expected clientSecret test-secret, got %s", c.clientSecret)
	}
}

func TestTokenFetchAndAPIUsage(t *testing.T) {
	tokenCalls := atomic.Int32{}
	apiCalls := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/token" && r.Method == http.MethodPost:
			tokenCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "mock-access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})

		case r.URL.Path == "/v1/browse/categories/indie/playlists" && r.Method == http.MethodGet:
			apiCalls.Add(1)
			if r.Header.Get("Authorization") != "Bearer mock-access-token" {
				t.Errorf("missing or wrong bearer token on API call")
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"playlists": map[string]interface{}{
					"items": []map[string]interface{}{
						{"id": "pl1", "name": "Indie Vibes"},
						{"id": "pl2", "name": "Chill Indie"},
					},
					"total": 2,
				},
			})

		case r.URL.Path == "/v1/playlists/pl1/tracks" && r.Method == http.MethodGet:
			apiCalls.Add(1)
			if r.Header.Get("Authorization") != "Bearer mock-access-token" {
				t.Errorf("missing or wrong bearer token on tracks call")
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"track": map[string]interface{}{
							"id":          "t1",
							"name":        "Sunset Lover",
							"duration_ms": 246000,
							"artists":     []map[string]interface{}{{"name": "Petit Biscuit"}},
						},
					},
					{
						"track": nil,
					},
					{
						"track": map[string]interface{}{
							"id":          "",
							"name":        "",
							"duration_ms": 0,
							"artists":     []map[string]interface{}{{"name": "Nobody"}},
						},
					},
				},
				"total": 3,
			})

		case r.URL.Path == "/v1/playlists/empty/tracks" && r.Method == http.MethodGet:
			apiCalls.Add(1)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]interface{}{},
				"total": 0,
			})

		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := NewClient("test-id", "test-secret")
	c.tokenURL = server.URL + "/api/token"
	c.apiBaseURL = server.URL + "/v1"

	t.Run("GetPopularPlaylistsByGenre", func(t *testing.T) {
		playlists, err := c.GetPopularPlaylistsByGenre("indie")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(playlists) != 2 {
			t.Fatalf("expected 2 playlists, got %d", len(playlists))
		}
		if playlists[0].ID != "pl1" || playlists[0].Name != "Indie Vibes" {
			t.Errorf("unexpected playlist: %+v", playlists[0])
		}
	})

	t.Run("GetPlaylistTracks", func(t *testing.T) {
		tracks, err := c.GetPlaylistTracks("pl1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(tracks) != 1 {
			t.Fatalf("expected 1 valid track (skip null and empty), got %d", len(tracks))
		}
		tr := tracks[0]
		if tr.ID != "t1" || tr.Title != "Sunset Lover" || tr.Artist != "Petit Biscuit" || tr.DurationMs != 246000 {
			t.Errorf("unexpected track: %+v", tr)
		}
	})

	t.Run("EmptyPlaylistReturnsError", func(t *testing.T) {
		_, err := c.GetPlaylistTracks("empty")
		if err == nil {
			t.Fatal("expected error for empty playlist, got nil")
		}
	})

	t.Run("TokenIsCached", func(t *testing.T) {
		callsBefore := tokenCalls.Load()

		_, _ = c.GetPopularPlaylistsByGenre("indie")
		_, _ = c.GetPlaylistTracks("pl1")

		if tokenCalls.Load() != callsBefore {
			t.Error("token was re-fetched, expected it to be cached")
		}
	})
}

func TestRateLimitRetry(t *testing.T) {
	callCount := atomic.Int32{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/token" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "rate-limited-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})

		case r.URL.Path == "/v1/browse/categories/jazz/playlists":
			callCount.Add(1)
			if callCount.Load() == 1 {
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"playlists": map[string]interface{}{
					"items": []map[string]interface{}{
						{"id": "j1", "name": "Smooth Jazz"},
					},
					"total": 1,
				},
			})

		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	c := NewClient("test-id", "test-secret")
	c.tokenURL = server.URL + "/api/token"
	c.apiBaseURL = server.URL + "/v1"

	playlists, err := c.GetPopularPlaylistsByGenre("jazz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(playlists) != 1 || playlists[0].Name != "Smooth Jazz" {
		t.Errorf("unexpected playlists after retry: %+v", playlists)
	}
	if callCount.Load() != 2 {
		t.Errorf("expected 2 calls (initial + retry), got %d", callCount.Load())
	}
}

func TestRateLimitDoubleFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/token" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "fail-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})

		default:
			w.WriteHeader(http.StatusTooManyRequests)
		}
	}))
	defer server.Close()

	c := NewClient("test-id", "test-secret")
	c.tokenURL = server.URL + "/api/token"
	c.apiBaseURL = server.URL + "/v1"

	_, err := c.GetPopularPlaylistsByGenre("rock")
	if err == nil {
		t.Fatal("expected error after double rate limit, got nil")
	}
}

func TestNoPlaylistsForGenre(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/token" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "ok",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})

		default:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"playlists": map[string]interface{}{
					"items": []map[string]interface{}{},
					"total": 0,
				},
			})
		}
	}))
	defer server.Close()

	c := NewClient("test-id", "test-secret")
	c.tokenURL = server.URL + "/api/token"
	c.apiBaseURL = server.URL + "/v1"

	_, err := c.GetPopularPlaylistsByGenre("unknown_genre")
	if err == nil {
		t.Fatal("expected error for genre with no playlists")
	}
}
