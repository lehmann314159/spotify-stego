package audio

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type GetSongBPMProvider struct {
	APIKey     string
	httpClient *http.Client
	baseURL    string
}

func NewGetSongBPMProvider(apiKey string) *GetSongBPMProvider {
	return &GetSongBPMProvider{
		APIKey:  apiKey,
		baseURL: "https://api.getsongbpm.com",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SetHTTPClient overrides the HTTP client (useful for testing).
func (g *GetSongBPMProvider) SetHTTPClient(httpClient *http.Client) {
	g.httpClient = httpClient
}

// SetBaseURL overrides the base URL (useful for testing).
func (g *GetSongBPMProvider) SetBaseURL(baseURL string) {
	g.baseURL = baseURL
}

type searchHit struct {
	ID      string  `json:"id"`
	Title   string  `json:"title"`
	Artist  struct {
		Name string `json:"name"`
	} `json:"artist"`
	Tempo   *string `json:"tempo"`
	KeyOf   *string `json:"key_of"`
	OpenKey *string `json:"open_key"`
}

type searchResponse struct {
	GenericSearch []searchHit `json:"search"`
}

type songDetailInner struct {
	ID      string  `json:"id"`
	Tempo   *string `json:"tempo"`
	KeyOf   *string `json:"key_of"`
	OpenKey *string `json:"open_key"`
}

func (g *GetSongBPMProvider) GetAudioData(title, artist string) (AudioData, error) {
	searchURL := fmt.Sprintf("%s/search/?api_key=%s&type=song&lookup=%s",
		g.baseURL, url.QueryEscape(g.APIKey),
		url.QueryEscape(title+" "+artist))

	resp, err := g.httpClient.Get(searchURL)
	if err != nil {
		return AudioData{}, fmt.Errorf("GetSongBPM search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return AudioData{}, fmt.Errorf("GetSongBPM search returned status %d", resp.StatusCode)
	}

	var body searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return AudioData{}, fmt.Errorf("failed to decode GetSongBPM search response: %w", err)
	}

	if len(body.GenericSearch) == 0 {
		return AudioData{}, ErrNotFound
	}

	hit := &body.GenericSearch[0]

	bpm, keyOf, camelotCode, err := g.parseSearchHit(hit)
	if err != nil {
		return AudioData{}, fmt.Errorf("parsing search result: %w", err)
	}

	if keyOf == "" {
		detail, err := g.fetchSongDetail(hit.ID)
		if err != nil {
			return AudioData{}, fmt.Errorf("fetching song detail: %w", err)
		}

		keyOf = stringPtr(detail.KeyOf)
		camelotCode = openKeyToCamelot(stringPtr(detail.OpenKey))
		if detail.Tempo != nil {
			bpm, _ = strconv.ParseFloat(*detail.Tempo, 64)
		}
	}

	return AudioData{
		BPM:         bpm,
		KeyOf:       keyOf,
		CamelotCode: camelotCode,
	}, nil
}

func (g *GetSongBPMProvider) fetchSongDetail(id string) (*songDetailInner, error) {
	detailURL := fmt.Sprintf("%s/song/?api_key=%s&id=%s",
		g.baseURL, url.QueryEscape(g.APIKey), url.QueryEscape(id))

	resp, err := g.httpClient.Get(detailURL)
	if err != nil {
		return nil, fmt.Errorf("GetSongBPM detail request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GetSongBPM detail returned status %d", resp.StatusCode)
	}

	var body struct {
		GenericSong struct {
			Song songDetailInner `json:"song"`
		} `json:"song"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("failed to decode GetSongBPM detail response: %w", err)
	}

	detail := body.GenericSong.Song
	return &detail, nil
}

func (g *GetSongBPMProvider) parseSearchHit(hit *searchHit) (float64, string, string, error) {
	var bpm float64
	if hit.Tempo != nil {
		var err error
		bpm, err = strconv.ParseFloat(*hit.Tempo, 64)
		if err != nil {
			return 0, "", "", fmt.Errorf("invalid tempo value %q: %w", *hit.Tempo, err)
		}
	}

	keyOf := ""
	if hit.KeyOf != nil {
		keyOf = *hit.KeyOf
	}

	camelotCode := openKeyToCamelot(stringPtr(hit.OpenKey))

	return bpm, keyOf, camelotCode, nil
}

func openKeyToCamelot(openKey string) string {
	openKey = strings.TrimSpace(openKey)
	if len(openKey) < 2 {
		return ""
	}

	last := openKey[len(openKey)-1]
	num := openKey[:len(openKey)-1]

	switch last {
	case 'm':
		return num + "A"
	case 'd':
		return num + "B"
	default:
		return ""
	}
}

func stringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}