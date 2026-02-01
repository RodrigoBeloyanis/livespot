package hash

import "testing"

func TestCanonicalHashDeterministic(t *testing.T) {
	a := map[string]any{"b": 1, "a": 2}
	b := map[string]any{"a": 2, "b": 1}
	ha, err := CanonicalHash(a)
	if err != nil {
		t.Fatalf("hash a failed: %v", err)
	}
	hb, err := CanonicalHash(b)
	if err != nil {
		t.Fatalf("hash b failed: %v", err)
	}
	if ha != hb {
		t.Fatalf("hash mismatch: %s vs %s", ha, hb)
	}
}
