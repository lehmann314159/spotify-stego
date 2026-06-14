package audio

import (
	"errors"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStubProvider_GetAudioData(t *testing.T) {
	s := StubProvider{}
	data, err := s.GetAudioData("any title", "any artist")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if data.BPM != 120.0 {
		t.Errorf("expected BPM 120.0, got %f", data.BPM)
	}
	if data.KeyOf != "C" {
		t.Errorf("expected KeyOf \"C\", got %q", data.KeyOf)
	}
	if data.CamelotCode != "8B" {
		t.Errorf("expected CamelotCode \"8B\", got %q", data.CamelotCode)
	}
}

func TestGetSongBPMProvider_GetAudioData_ValidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"search": []map[string]interface{}{
				{
					"id":     "abc123",
					"title":  "Master of Puppets",
					"artist": map[string]string{"name": "Metallica"},
					"tempo":  "220",
					"key_of": "Em",
					"open_key": "7m",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewGetSongBPMProvider("test-key")
	provider.baseURL = server.URL
	provider.httpClient = &http.Client{}

	data, err := provider.GetAudioData("Master of Puppets", "Metallica")

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if data.BPM != 220.0 {
		t.Errorf("expected BPM 220.0, got %f", data.BPM)
	}
	if data.KeyOf != "Em" {
		t.Errorf("expected KeyOf \"Em\", got %q", data.KeyOf)
	}
	if data.CamelotCode != "7A" {
		t.Errorf("expected CamelotCode \"7A\", got %q", data.CamelotCode)
	}
}

func TestGetSongBPMProvider_GetAudioData_NoResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"search": []map[string]interface{}{},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewGetSongBPMProvider("test-key")
	provider.baseURL = server.URL
	provider.httpClient = &http.Client{}

	_, err := provider.GetAudioData("nonexistent", "track")

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestOpenKeyToCamelot(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"7m", "7A"},
		{"1d", "1B"},
		{"12m", "12A"},
		{"", ""},
	}

	for _, tt := range tests {
		result := openKeyToCamelot(tt.input)
		if result != tt.expected {
			t.Errorf("openKeyToCamelot(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
