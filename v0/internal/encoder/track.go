package encoder

// Track is the encoder's view of a song — fields drawn from both Spotify and
// the audio provider. The database and CLI layers convert to this type.
type Track struct {
	ID          string
	Title       string
	Artist      string
	Genre       string
	DurationMS  int
	BPM         float64
	KeyOf       string
	CamelotCode string
}
