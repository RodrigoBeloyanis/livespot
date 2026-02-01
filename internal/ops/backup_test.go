package ops

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBackupSQLiteCopiesFiles(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "audit.sqlite")
	if err := os.WriteFile(dbPath, []byte("db"), 0o640); err != nil {
		t.Fatalf("write db: %v", err)
	}
	if err := os.WriteFile(dbPath+"-wal", []byte("wal"), 0o640); err != nil {
		t.Fatalf("write wal: %v", err)
	}
	backupDir := filepath.Join(dir, "backups")
	now := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	res, err := BackupSQLite(dbPath, backupDir, now)
	if err != nil {
		t.Fatalf("backup: %v", err)
	}
	if len(res.Paths) != 2 {
		t.Fatalf("expected 2 files, got %d", len(res.Paths))
	}
	for _, path := range res.Paths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("backup missing: %v", err)
		}
	}
}
