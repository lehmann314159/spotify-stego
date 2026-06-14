package audio

// StubProvider returns fixed placeholder data for any input.
// It is the default implementation used until a real provider is wired in.
type StubProvider struct{}

func (s StubProvider) GetAudioData(title, artist string) (AudioData, error) {
	return AudioData{
		BPM:         120.0,
		KeyOf:       "C",
		CamelotCode: "8B",
	}, nil
}
