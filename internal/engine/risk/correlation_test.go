package risk

import "testing"

func TestCorrelationX10000PerfectPositive(t *testing.T) {
	seriesA := []int32{1, 2, 3, 4, 5}
	seriesB := []int32{1, 2, 3, 4, 5}
	corr, err := CorrelationX10000(seriesA, seriesB)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if corr != 10000 {
		t.Fatalf("expected 10000, got %d", corr)
	}
}

func TestCorrelationX10000PerfectNegative(t *testing.T) {
	seriesA := []int32{1, 2, 3, 4, 5}
	seriesB := []int32{5, 4, 3, 2, 1}
	corr, err := CorrelationX10000(seriesA, seriesB)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if corr != -10000 {
		t.Fatalf("expected -10000, got %d", corr)
	}
}
