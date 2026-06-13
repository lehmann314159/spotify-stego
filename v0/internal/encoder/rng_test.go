package encoder

import "testing"

func TestDeriveRNGReproducibility(t *testing.T) {
	kw := [3]string{"alpha", "beta", "gamma"}
	r1 := DeriveRNG(kw)
	r2 := DeriveRNG(kw)
	for i := 0; i < 100; i++ {
		a, b := r1.Intn(1000), r2.Intn(1000)
		if a != b {
			t.Fatalf("step %d: got %d and %d, want equal", i, a, b)
		}
	}
}

func TestDeriveRNGDifferentKeywords(t *testing.T) {
	r1 := DeriveRNG([3]string{"a", "b", "c"})
	r2 := DeriveRNG([3]string{"x", "y", "z"})
	same := 0
	for i := 0; i < 20; i++ {
		if r1.Intn(1000000) == r2.Intn(1000000) {
			same++
		}
	}
	if same > 2 {
		t.Fatalf("different keywords produced nearly identical sequences (%d/20 same)", same)
	}
}

func TestClone(t *testing.T) {
	r := DeriveRNG([3]string{"a", "b", "c"})
	r.Intn(100) // advance once
	clone := r.Clone()
	for i := 0; i < 50; i++ {
		a, b := r.Intn(1000), clone.Intn(1000)
		if a != b {
			t.Fatalf("clone diverged at step %d: %d vs %d", i, a, b)
		}
	}
}
