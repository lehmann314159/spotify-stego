package audio

// AudioData holds BPM and key information for a track.
type AudioData struct {
	BPM         float64
	KeyOf       string // e.g. "Em", "C"
	CamelotCode string // e.g. "7A", "1B"
}

// AudioDataProvider abstracts BPM and musical key lookups.
type AudioDataProvider interface {
	GetAudioData(title, artist string) (AudioData, error)
}
