package spotify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	tokenEndpoint      = "https://accounts.spotify.com/api/token"
	apiBase            = "https://api.spotify.com/v1"
	rateLimitRetrySecs = 1
	authBufferSeconds  = 60
)

// Playlist represents a Spotify playlist.
type Playlist struct {
	ID   string
	Name string
}

// Track represents a Spotify track (Spotify-specific, distinct from core.Track).
type Track struct {
	ID         string
	Title      string
	Artist     string
	DurationMs int
}

// Client communicates with the Spotify API using client credentials flow.
type Client struct {
	clientID     string
	clientSecret string
	httpClient   *http.Client
	tokenURL     string
	apiBaseURL   string

	mu          sync.Mutex
	token       string
	tokenExpiry time.Time
}

// authResponse is the JSON response from Spotify's token endpoint.
type authResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// browsePlaylistsResponse is the JSON response from the browse categories playlists endpoint.
type browsePlaylistsResponse struct {
	Playlists struct {
		Items []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
		} `json:"items"`
		Total int `json:"total"`
	} `json:"playlists"`
}

// playlistTracksResponse is the JSON response from the playlist tracks endpoint.
type playlistTracksResponse struct {
	Items []struct {
		Track *struct {
			ID          string   `json:"id"`
			Name        string   `json:"name"`
			DurationMs  int      `json:"duration_ms"`
			Artists     []struct {
				Name string `json:"name"`
			} `json:"artists"`
		} `json:"track"`
	} `json:"items"`
	Total int `json:"total"`
}

// SetHTTPClient overrides the HTTP client (useful for testing).
func (c *Client) SetHTTPClient(httpClient *http.Client) {
	c.httpClient = httpClient
}

// SetEndpoints overrides the token and API base URLs (useful for testing).
func (c *Client) SetEndpoints(tokenURL, apiBaseURL string) {
	c.tokenURL = tokenURL
	c.apiBaseURL = apiBaseURL
}

// NewClient creates a new Spotify API client. No network calls are made here —
// authentication is lazy and happens on the first API call.
func NewClient(clientID, clientSecret string) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
		tokenURL:     tokenEndpoint,
		apiBaseURL:   apiBase,
	}
}

// ensureToken fetches a new access token if none is cached or if the current
// token is within authBufferSeconds of expiry.
func (c *Client) ensureToken(ctx context.Context) error {
	c.mu.Lock()
	if time.Until(c.tokenExpiry) > authBufferSeconds*time.Second && c.token != "" {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	authBody := url.Values{"grant_type": {"client_credentials"}}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.tokenURL, strings.NewReader(authBody))
	if err != nil {
		return fmt.Errorf("create token request: %w", err)
	}

	creds := base64.StdEncoding.EncodeToString([]byte(c.clientID + ":" + c.clientSecret))
	req.Header.Set("Authorization", "Basic "+creds)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token request failed with status %d", resp.StatusCode)
	}

	var ar authResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return fmt.Errorf("decode token response: %w", err)
	}

	c.mu.Lock()
	c.token = ar.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(ar.ExpiresIn) * time.Second)
	c.mu.Unlock()

	return nil
}

// authToken returns the current access token.
func (c *Client) authToken() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.token
}

// doRequest performs an HTTP request with auth and 429 retry logic.
func (c *Client) doRequest(ctx context.Context, req *http.Request, into any) error {
	req.Header.Set("Authorization", "Bearer "+c.authToken())

	for attempt := 0; attempt < 2; attempt++ {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := rateLimitRetrySecs
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if n, parseErr := fmt.Sscanf(ra, "%d", &retryAfter); n == 0 || parseErr != nil {
					retryAfter = rateLimitRetrySecs
				}
			}
			log.Printf("Spotify API rate limited (429), waiting %d seconds before retry\n", retryAfter)

			if attempt == 0 {
				time.Sleep(time.Duration(retryAfter) * time.Second)
				continue
			}
			return fmt.Errorf("rate limited by Spotify API (429)")
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("request failed with status %d", resp.StatusCode)
		}

		if into != nil {
			if err := json.NewDecoder(resp.Body).Decode(into); err != nil {
				return fmt.Errorf("decode response: %w", err)
			}
		}
		return nil
	}
	return nil
}

// GetPopularPlaylistsByGenre returns Spotify's featured playlists for a given
// genre/category ID. Returns up to 20 playlists.
func (c *Client) GetPopularPlaylistsByGenre(genre string) ([]Playlist, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := c.ensureToken(ctx); err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	u := fmt.Sprintf("%s/browse/categories/%s/playlists?limit=20", c.apiBaseURL, url.PathEscape(genre))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build playlists request: %w", err)
	}

	var resp browsePlaylistsResponse
	if err := c.doRequest(ctx, req, &resp); err != nil {
		return nil, fmt.Errorf("get playlists for genre %q: %w", genre, err)
	}

	if len(resp.Playlists.Items) == 0 {
		return nil, fmt.Errorf("no playlists found for genre %q", genre)
	}

	playlists := make([]Playlist, 0, len(resp.Playlists.Items))
	for _, item := range resp.Playlists.Items {
		playlists = append(playlists, Playlist{
			ID:   item.ID,
			Name: item.Name,
		})
	}
	return playlists, nil
}

// GetPlaylistTracks returns all tracks in a playlist, handling pagination.
// Only tracks with non-empty title and artist are returned.
func (c *Client) GetPlaylistTracks(playlistID string) ([]Track, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := c.ensureToken(ctx); err != nil {
		return nil, fmt.Errorf("authenticate: %w", err)
	}

	var allTracks []Track
	offset := 0
	pageSize := 100

	for {
		u := fmt.Sprintf("%s/playlists/%s/tracks?limit=%d&offset=%d",
			c.apiBaseURL, url.PathEscape(playlistID), pageSize, offset)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, fmt.Errorf("build tracks request: %w", err)
		}

		var resp playlistTracksResponse
		if err := c.doRequest(ctx, req, &resp); err != nil {
			return nil, fmt.Errorf("get tracks for playlist %s: %w", playlistID, err)
		}

		for _, item := range resp.Items {
			if item.Track == nil {
				continue
			}

			title := item.Track.Name
			artist := ""
			if len(item.Track.Artists) > 0 {
				artist = item.Track.Artists[0].Name
			}

			if title == "" || artist == "" {
				continue
			}

			allTracks = append(allTracks, Track{
				ID:         item.Track.ID,
				Title:      title,
				Artist:     artist,
				DurationMs: item.Track.DurationMs,
			})
		}

		totalPages := float64(resp.Total) / float64(pageSize)
		if float64(offset)+float64(pageSize) >= math.Ceil(totalPages)*float64(pageSize) {
			break
		}
		offset += pageSize
	}

	if len(allTracks) == 0 {
		return nil, fmt.Errorf("no valid tracks found in playlist %q", playlistID)
	}

	return allTracks, nil
}
