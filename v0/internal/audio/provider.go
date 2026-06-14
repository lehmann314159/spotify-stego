// Package audio defines the interface for track audio analysis (BPM and key).
package audio

// AudioData holds BPM and key information for a track.
type AudioData struct {
	BPM         float64 // beats per minute as float64
	KeyOf       string  // musical key in standard notation, e.g. "Em", "C"
	CamelotCode string  // Camelot wheel code in "NL" format (1–12 + A/B), e.g. "7A", "1B"
}

// AudioDataProvider abstracts BPM and musical key lookups.
// Implementations include StubProvider (always returns placeholder values)
// and GetSongBPMProvider (calls the GetSongBPM API).
type AudioDataProvider interface {
	// GetAudioData returns BPM and key data for the given track title and artist.
	GetAudioData(title, artist string) (AudioData, error)
}
