package crypto

import "testing"

func TestDeriveExtractorDeterministic(t *testing.T) {
	rand1 := DeriveExtractor([3]string{"a", "b", "c"})
	rand2 := DeriveExtractor([3]string{"a", "b", "c"})

	a1 := rand1.Intn(100)
	b1 := rand1.Intn(100)
	c1 := rand1.Intn(100)

	a2 := rand2.Intn(100)
	b2 := rand2.Intn(100)
	c2 := rand2.Intn(100)

	if (a1 != a2) || (b1 != b2) || (c1 != c2) {
		t.Error("random numbers should be equal")
	}
}

func TestDeriveExtractorDifferentKeys(t *testing.T) {
	rand1 := DeriveExtractor([3]string{"a", "b", "c"})
	rand2 := DeriveExtractor([3]string{"d", "e", "f"})

	a1 := rand1.Intn(100)
	b1 := rand1.Intn(100)
	c1 := rand1.Intn(100)

	a2 := rand2.Intn(100)
	b2 := rand2.Intn(100)
	c2 := rand2.Intn(100)

	if (a1 == a2) && (b1 == b2) && (c1 == c2) {
		t.Error("random numbers should be different")
	}
}
