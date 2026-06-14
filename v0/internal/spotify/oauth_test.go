package spotify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestAddTracksToPlaylistBatching(t *testing.T) {
	var mu sync.Mutex
	var requestCount int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/playlists/pl1/tracks" {
			mu.Lock()
			requestCount++
			mu.Unlock()
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := New("id", "secret")
	c.apiBaseURL = srv.URL
	c.userAccessToken = "test-token"
	c.userTokenExpiry = time.Now().Add(time.Hour)

	ids := make([]string, 150)
	for i := range ids {
		ids[i] = fmt.Sprintf("track%d", i)
	}

	if err := c.AddTracksToPlaylist("pl1", ids); err != nil {
		t.Fatalf("AddTracksToPlaylist: %v", err)
	}
	if requestCount != 2 {
		t.Fatalf("expected 2 POST requests (100+50), got %d", requestCount)
	}
}

func TestOAuthStateValidation(t *testing.T) {
	c := New("id", "secret")

	// No prior AuthorizeURL call — no stored state.
	err := c.ExchangeCode("wrong-state", "code", "http://example.com")
	if err == nil {
		t.Fatal("expected error for unknown state, got nil")
	}
	if c.IsAuthenticated() {
		t.Fatal("expected not authenticated after invalid state")
	}
}

func TestCreatePlaylistRequest(t *testing.T) {
	var gotPublicTrue bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/me":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"uid"}`)
		case "/users/uid/playlists":
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body) //nolint:errcheck
			if v, ok := body["public"].(bool); ok && v {
				gotPublicTrue = true
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"id":"playlist123"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	c := New("id", "secret")
	c.apiBaseURL = srv.URL
	c.userAccessToken = "fake-token"
	c.userTokenExpiry = time.Now().Add(time.Hour)

	userID, err := c.GetCurrentUserID()
	if err != nil {
		t.Fatalf("GetCurrentUserID: %v", err)
	}
	if userID != "uid" {
		t.Fatalf("expected userID %q, got %q", "uid", userID)
	}

	playlistID, err := c.CreatePlaylist(userID, "Test Playlist", true)
	if err != nil {
		t.Fatalf("CreatePlaylist: %v", err)
	}
	if playlistID != "playlist123" {
		t.Fatalf("expected playlist ID %q, got %q", "playlist123", playlistID)
	}
	if !gotPublicTrue {
		t.Fatal(`expected "public":true in CreatePlaylist request body`)
	}
}
