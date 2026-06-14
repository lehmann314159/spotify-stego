package audio

import "errors"

// AudioData holds BPM and key information for a track.
type AudioData struct {
	BPM         float64
	KeyOf       string // e.g. "Em", "C", "F#"
	CamelotCode string // e.g. "7A", "1B"
}

// Provider is the interface all audio data sources must implement.
type Provider interface {
	GetAudioData(title, artist string) (AudioData, error)
}

var ErrNotFound = errors.New("track not found in GetSongBPM")
