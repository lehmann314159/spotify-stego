package spotify

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	authURL = "https://accounts.spotify.com/api/token"
	apiBase = "https://api.spotify.com/v1"
)

// Track holds the fields we care about from the Spotify API.
type Track struct {
	ID         string
	Title      string
	Artist     string
	DurationMS int
}

// Playlist holds minimal playlist metadata.
type Playlist struct {
	ID   string
	Name string
}

// Client is a Spotify Web API client using the client credentials flow.
type Client struct {
	clientID     string
	clientSecret string

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
	reqCount    int
	windowStart time.Time
}

func New(clientID, clientSecret string) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		windowStart:  time.Now(),
	}
}

func (c *Client) token() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Now().Before(c.expiresAt) {
		return c.accessToken, nil
	}
	creds := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))
	resp, err := http.PostForm(authURL, url.Values{"grant_type": {"client_credentials"}})
	if err != nil {
		return "", fmt.Errorf("spotify token request: %w", err)
	}
	defer resp.Body.Close()
	// Rebuild with auth header
	req, _ := http.NewRequest("POST", authURL, strings.NewReader("grant_type=client_credentials"))
	req.Header.Set("Authorization", "Basic "+creds)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp2, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("spotify auth: %w", err)
	}
	defer resp2.Body.Close()
	body, _ := io.ReadAll(resp2.Body)
	var tok struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &tok); err != nil || tok.AccessToken == "" {
		return "", fmt.Errorf("spotify auth response: %s", body)
	}
	c.accessToken = tok.AccessToken
	c.expiresAt = time.Now().Add(time.Duration(tok.ExpiresIn-30) * time.Second)
	return c.accessToken, nil
}

func (c *Client) get(path string, out interface{}) error {
	tok, err := c.token()
	if err != nil {
		return err
	}
	c.mu.Lock()
	// Simple rate-limit logging: track requests per minute
	now := time.Now()
	if now.Sub(c.windowStart) > time.Minute {
		c.reqCount = 0
		c.windowStart = now
	}
	c.reqCount++
	if c.reqCount > 130 {
		log.Printf("spotify: approaching rate limit (%d req/min)", c.reqCount)
	}
	c.mu.Unlock()

	req, _ := http.NewRequest("GET", apiBase+path, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("spotify GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 429 {
		return fmt.Errorf("spotify rate limited on %s", path)
	}
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("spotify %s: %d %s", path, resp.StatusCode, body)
	}
	body, _ := io.ReadAll(resp.Body)
	return json.Unmarshal(body, out)
}

// GetPopularPlaylistsByGenre searches for playlists by genre keyword.
func (c *Client) GetPopularPlaylistsByGenre(genre string) ([]Playlist, error) {
	var result struct {
		Playlists struct {
			Items []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"items"`
		} `json:"playlists"`
	}
	path := fmt.Sprintf("/search?q=%s&type=playlist&limit=20", url.QueryEscape(genre))
	if err := c.get(path, &result); err != nil {
		return nil, err
	}
	playlists := make([]Playlist, 0, len(result.Playlists.Items))
	for _, item := range result.Playlists.Items {
		playlists = append(playlists, Playlist{ID: item.ID, Name: item.Name})
	}
	return playlists, nil
}

// GetPlaylistTracks fetches all tracks from a playlist (handles pagination).
func (c *Client) GetPlaylistTracks(playlistID string) ([]Track, error) {
	var tracks []Track
	path := fmt.Sprintf("/playlists/%s/tracks?limit=100&fields=next,items(track(id,name,duration_ms,artists))", playlistID)
	for path != "" {
		var result struct {
			Next  *string `json:"next"`
			Items []struct {
				Track *struct {
					ID         string `json:"id"`
					Name       string `json:"name"`
					DurationMS int    `json:"duration_ms"`
					Artists    []struct {
						Name string `json:"name"`
					} `json:"artists"`
				} `json:"track"`
			} `json:"items"`
		}
		if err := c.get(path, &result); err != nil {
			return nil, err
		}
		for _, item := range result.Items {
			if item.Track == nil || item.Track.ID == "" {
				continue
			}
			artist := ""
			if len(item.Track.Artists) > 0 {
				artist = item.Track.Artists[0].Name
			}
			tracks = append(tracks, Track{
				ID:         item.Track.ID,
				Title:      item.Track.Name,
				Artist:     artist,
				DurationMS: item.Track.DurationMS,
			})
		}
		if result.Next != nil && *result.Next != "" {
			// Strip base URL for our get() helper
			next := strings.TrimPrefix(*result.Next, apiBase)
			path = next
		} else {
			path = ""
		}
	}
	return tracks, nil
}
