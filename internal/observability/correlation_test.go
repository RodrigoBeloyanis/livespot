package observability

import (
	"regexp"
	"testing"
	"time"
)

func TestNewRunIDFormat(t *testing.T) {
	now := time.Date(2026, 2, 1, 12, 30, 45, 0, time.UTC)
	id, err := NewRunID(now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantPrefix := "run_20260201_123045_"
	if len(id) <= len(wantPrefix) || id[:len(wantPrefix)] != wantPrefix {
		t.Fatalf("run_id prefix mismatch: %s", id)
	}
	if !regexp.MustCompile(`^run_\d{8}_\d{6}_[0-9a-f]{6}$`).MatchString(id) {
		t.Fatalf("run_id format invalid: %s", id)
	}
}

func TestNewCycleIDFormat(t *testing.T) {
	now := time.Date(2026, 2, 1, 12, 30, 45, 0, time.UTC)
	id, err := NewCycleID(now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !regexp.MustCompile(`^cyc_\d{8}_\d{6}_[0-9a-f]{6}$`).MatchString(id) {
		t.Fatalf("cycle_id format invalid: %s", id)
	}
}
