package ops

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupJSONL(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "audit-2020-01-01.jsonl")
	newer := filepath.Join(dir, "audit-2026-01-30.jsonl")
	if err := os.WriteFile(old, []byte("x"), 0o640); err != nil {
		t.Fatalf("write old: %v", err)
	}
	if err := os.WriteFile(newer, []byte("y"), 0o640); err != nil {
		t.Fatalf("write new: %v", err)
	}
	now := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	deleted, err := CleanupJSONL(dir, 7, now)
	if err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if len(deleted) != 1 {
		t.Fatalf("expected 1 delete, got %d", len(deleted))
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("old file should be removed")
	}
	if _, err := os.Stat(newer); err != nil {
		t.Fatalf("newer file should remain")
	}
}

func TestCleanupLogs(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "old.log")
	if err := os.WriteFile(old, []byte("x"), 0o640); err != nil {
		t.Fatalf("write old: %v", err)
	}
	oldTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(old, oldTime, oldTime); err != nil {
		t.Fatalf("chtimes: %v", err)
	}
	now := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	deleted, err := CleanupLogs(dir, 7, now)
	if err != nil {
		t.Fatalf("cleanup logs: %v", err)
	}
	if len(deleted) != 1 {
		t.Fatalf("expected 1 delete, got %d", len(deleted))
	}
}
