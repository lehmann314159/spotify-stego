package audio

// StubProvider satisfies AudioDataProvider with deterministic placeholder data.
// No network calls are made. BPM=120.0, KeyOf="C", CamelotCode="8B" for any input.
type StubProvider struct{}

func (StubProvider) GetAudioData(title, artist string) (AudioData, error) {
	return AudioData{
		BPM:         120.0,
		KeyOf:       "C",
		CamelotCode: "8B",
	}, nil
}
