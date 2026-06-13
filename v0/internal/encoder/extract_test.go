package encoder

import "testing"

func TestExtractDeterminism(t *testing.T) {
	kw := [3]string{"one", "two", "three"}
	title := "Bohemian Rhapsody"

	r1 := DeriveRNG(kw)
	r2 := DeriveRNG(kw)
	got1 := ExtractFromTrack(r1, title)
	got2 := ExtractFromTrack(r2, title)

	if string(got1) != string(got2) {
		t.Fatalf("non-deterministic extraction: %q vs %q", got1, got2)
	}
}

func TestExtractWithinLetters(t *testing.T) {
	kw := [3]string{"a", "b", "c"}
	title := "Hello World"
	letters := "helloworld"
	for i := 0; i < 50; i++ {
		rng := DeriveRNG(kw)
		for j := 0; j < i; j++ {
			rng.Intn(100) // advance to different positions
		}
		got := ExtractFromTrack(rng, title)
		for _, b := range got {
			found := false
			for _, l := range []byte(letters) {
				if b == l {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("extracted %q not in letters of %q", string(b), title)
			}
		}
	}
}

func TestExtractCountCapped(t *testing.T) {
	// Title with max word length 2 → count should always be 1 or 2
	title := "Hi Go"
	for i := 0; i < 100; i++ {
		rng := DeriveRNG([3]string{"x", "y", "z"})
		for j := 0; j < i; j++ {
			rng.Intn(100)
		}
		got := ExtractFromTrack(rng, title)
		if len(got) < 1 || len(got) > 2 {
			t.Fatalf("expected count 1-2, got %d", len(got))
		}
	}
}
