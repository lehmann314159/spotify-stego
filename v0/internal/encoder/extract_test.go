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

func TestExtractCountDeterministic(t *testing.T) {
	// Count is now derived from the title, not the PRNG, so it must be stable
	// regardless of RNG state.
	cases := []struct {
		title     string
		wantCount int
	}{
		// "a": 1 letter, cap3=1 → count = 1+(1%1) = 1
		{"a", 1},
		// "Hello World": 10 letters, cap3=3 → count = 1+(10%3) = 2
		{"Hello World", 2},
		// "abc def ghi": 9 letters, cap3=3 → count = 1+(9%3) = 1
		{"abc def ghi", 1},
	}
	for _, tc := range cases {
		for seed := 0; seed < 20; seed++ {
			rng := DeriveRNG([3]string{"x", "y", "z"})
			for j := 0; j < seed; j++ {
				rng.Intn(100) // vary PRNG state
			}
			got := ExtractFromTrack(rng, tc.title)
			if len(got) != tc.wantCount {
				t.Errorf("title %q seed %d: want count %d, got %d", tc.title, seed, tc.wantCount, len(got))
			}
		}
	}
}
