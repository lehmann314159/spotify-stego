package audio

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const getsongbpmBase = "https://api.getsongbpm.com"

// GetSongBPMProvider is the real AudioDataProvider backed by the GetSongBPM API.
type GetSongBPMProvider struct {
	APIKey string
}

type searchResponse struct {
	Search []struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Artist struct {
			Name string `json:"name"`
		} `json:"artist"`
	} `json:"search"`
}

type songResponse struct {
	Song struct {
		Tempo string `json:"tempo"`
		KeyOf string `json:"key_of"`
	} `json:"song"`
}

func (p *GetSongBPMProvider) SearchSong(title, artist string) (string, error) {
	q := url.QueryEscape(title + " " + artist)
	u := fmt.Sprintf("%s/search/?api_key=%s&type=song&lookup=%s", getsongbpmBase, p.APIKey, q)
	resp, err := http.Get(u) //nolint:gosec
	if err != nil {
		return "", fmt.Errorf("getsongbpm search: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var sr searchResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		return "", fmt.Errorf("getsongbpm search decode: %w", err)
	}
	if len(sr.Search) == 0 {
		return "", fmt.Errorf("getsongbpm: no results for %q by %q", title, artist)
	}
	return sr.Search[0].ID, nil
}

func (p *GetSongBPMProvider) GetSong(id string) (AudioData, error) {
	u := fmt.Sprintf("%s/song/?api_key=%s&id=%s", getsongbpmBase, p.APIKey, id)
	resp, err := http.Get(u) //nolint:gosec
	if err != nil {
		return AudioData{}, fmt.Errorf("getsongbpm get: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var sr songResponse
	if err := json.Unmarshal(body, &sr); err != nil {
		return AudioData{}, fmt.Errorf("getsongbpm get decode: %w", err)
	}
	var bpm float64
	fmt.Sscanf(sr.Song.Tempo, "%f", &bpm)
	keyOf := sr.Song.KeyOf
	return AudioData{BPM: bpm, KeyOf: keyOf}, nil
}

func (p *GetSongBPMProvider) GetAudioData(title, artist string) (AudioData, error) {
	id, err := p.SearchSong(title, artist)
	if err != nil {
		return AudioData{}, err
	}
	data, err := p.GetSong(id)
	if err != nil {
		return AudioData{}, err
	}
	return data, nil
}
