package spotify

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const authorizeEndpoint = "https://accounts.spotify.com/authorize"

// AuthorizeURL generates the Spotify authorization URL for PKCE flow.
// Stores state and code verifier for validation in ExchangeCode.
func (c *Client) AuthorizeURL(redirectURI string) (string, error) {
	// TODO: set SPOTIFY_REDIRECT_URI in .env before testing OAuth

	// Step 1: Generate code verifier — 32 random bytes, base64url, no padding.
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return "", fmt.Errorf("pkce verifier: %w", err)
	}
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// Step 2: Derive code challenge — SHA256 of verifier, base64url, no padding, method S256.
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	// Step 3: Generate state — 16 random bytes, base64url.
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", fmt.Errorf("pkce state: %w", err)
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	// Step 4: Store state → verifier so ExchangeCode can validate.
	c.pendingMu.Lock()
	if c.pending == nil {
		c.pending = make(map[string]string)
	}
	c.pending[state] = verifier
	c.pendingMu.Unlock()

	scope := "playlist-modify-public"
	if os.Getenv("SPOTIFY_PLAYLIST_PRIVATE") == "true" {
		scope = "playlist-modify-private"
	}

	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {c.clientID},
		"scope":                 {scope},
		"redirect_uri":          {redirectURI},
		"state":                 {state},
		"code_challenge_method": {"S256"},
		"code_challenge":        {challenge},
	}
	return authorizeEndpoint + "?" + params.Encode(), nil
}

// ExchangeCode validates state and exchanges the code for tokens.
// Returns an error if state does not match.
func (c *Client) ExchangeCode(state, code, redirectURI string) error {
	// TODO: set SPOTIFY_REDIRECT_URI in .env before testing OAuth

	c.pendingMu.Lock()
	verifier, ok := c.pending[state]
	if ok {
		delete(c.pending, state)
	}
	c.pendingMu.Unlock()

	if !ok {
		return fmt.Errorf("oauth: invalid or unknown state %q", state)
	}

	params := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {redirectURI},
		"client_id":     {c.clientID},
		"code_verifier": {verifier},
	}
	req, _ := http.NewRequest("POST", c.authBaseURL, strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("oauth token exchange: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var tok struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        string `json:"error"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return fmt.Errorf("oauth token parse: %w", err)
	}
	if tok.Error != "" || tok.AccessToken == "" {
		return fmt.Errorf("oauth token error: %s", body)
	}

	c.mu.Lock()
	c.userAccessToken = tok.AccessToken
	c.userRefreshToken = tok.RefreshToken
	c.userTokenExpiry = time.Now().Add(time.Duration(tok.ExpiresIn-30) * time.Second)
	c.mu.Unlock()

	// TODO: OAuth token not persisted across restarts.
	return nil
}

// IsAuthenticated reports whether a valid user token is stored.
func (c *Client) IsAuthenticated() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.userAccessToken != "" && time.Now().Before(c.userTokenExpiry)
}

// userToken returns the user access token, refreshing lazily if expired.
func (c *Client) userToken() (string, error) {
	c.mu.Lock()
	tok := c.userAccessToken
	exp := c.userTokenExpiry
	refresh := c.userRefreshToken
	c.mu.Unlock()

	if tok != "" && time.Now().Before(exp) {
		return tok, nil
	}
	if refresh == "" {
		return "", fmt.Errorf("oauth: not authenticated")
	}
	return c.refreshUserToken(refresh)
}

// refreshUserToken exchanges a refresh token for a new access token.
func (c *Client) refreshUserToken(refreshToken string) (string, error) {
	params := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {c.clientID},
	}
	req, _ := http.NewRequest("POST", c.authBaseURL, strings.NewReader(params.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("oauth refresh: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var tok struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Error        string `json:"error"`
	}
	if err := json.Unmarshal(body, &tok); err != nil {
		return "", fmt.Errorf("oauth refresh parse: %w", err)
	}
	if tok.Error != "" || tok.AccessToken == "" {
		return "", fmt.Errorf("oauth refresh error: %s", body)
	}

	c.mu.Lock()
	c.userAccessToken = tok.AccessToken
	if tok.RefreshToken != "" {
		c.userRefreshToken = tok.RefreshToken
	}
	c.userTokenExpiry = time.Now().Add(time.Duration(tok.ExpiresIn-30) * time.Second)
	c.mu.Unlock()

	return tok.AccessToken, nil
}

// execUserRequest builds and executes a single authenticated user request.
func (c *Client) execUserRequest(method, path string, bodyBytes []byte, tok string) (*http.Response, error) {
	var r io.Reader
	if bodyBytes != nil {
		r = bytes.NewReader(bodyBytes)
	}
	req, _ := http.NewRequest(method, c.apiBaseURL+path, r)
	req.Header.Set("Authorization", "Bearer "+tok)
	if bodyBytes != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("spotify %s %s: %w", method, path, err)
	}
	return resp, nil
}

// doUser performs an authenticated API call with the user token.
// Refreshes lazily on 401 and retries once.
func (c *Client) doUser(method, path string, bodyBytes []byte, out interface{}) error {
	tok, err := c.userToken()
	if err != nil {
		return err
	}

	resp, err := c.execUserRequest(method, path, bodyBytes, tok)
	if err != nil {
		return err
	}

	if resp.StatusCode == 401 {
		resp.Body.Close()
		// Expire the cached token so userToken() fetches a fresh one.
		c.mu.Lock()
		c.userTokenExpiry = time.Time{}
		c.mu.Unlock()
		tok, err = c.userToken()
		if err != nil {
			return fmt.Errorf("oauth refresh on 401: %w", err)
		}
		resp, err = c.execUserRequest(method, path, bodyBytes, tok)
		if err != nil {
			return err
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("spotify %s %s: %d %s", method, path, resp.StatusCode, b)
	}
	if out != nil {
		b, _ := io.ReadAll(resp.Body)
		return json.Unmarshal(b, out)
	}
	return nil
}

// GetCurrentUserID calls GET /v1/me.
func (c *Client) GetCurrentUserID() (string, error) {
	var result struct {
		ID string `json:"id"`
	}
	if err := c.doUser("GET", "/me", nil, &result); err != nil {
		return "", err
	}
	if result.ID == "" {
		return "", fmt.Errorf("spotify: empty user ID")
	}
	return result.ID, nil
}

// CreatePlaylist calls POST /v1/me/playlists.
// Returns the new playlist Spotify ID.
func (c *Client) CreatePlaylist(userID, name string, public bool) (string, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"name":   name,
		"public": public,
	})
	var result struct {
		ID string `json:"id"`
	}
	if err := c.doUser("POST", "/me/playlists", body, &result); err != nil {
		return "", err
	}
	return result.ID, nil
}

// AddTracksToPlaylist calls POST /v1/playlists/{playlistID}/tracks.
// Batches in groups of 100 (Spotify API limit).
// trackIDs are bare Spotify IDs; method prepends spotify:track:.
func (c *Client) AddTracksToPlaylist(playlistID string, trackIDs []string) error {
	for i := 0; i < len(trackIDs); i += 100 {
		end := i + 100
		if end > len(trackIDs) {
			end = len(trackIDs)
		}
		uris := make([]string, end-i)
		for j, id := range trackIDs[i:end] {
			uris[j] = "spotify:track:" + id
		}
		body, _ := json.Marshal(map[string]interface{}{"uris": uris})
		if err := c.doUser("POST", fmt.Sprintf("/playlists/%s/tracks", playlistID), body, nil); err != nil {
			return err
		}
	}
	return nil
}
