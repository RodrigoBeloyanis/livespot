package ops

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type BackupResult struct {
	Paths []string
}

func BackupSQLite(dbPath string, backupDir string, now time.Time) (BackupResult, error) {
	if dbPath == "" || backupDir == "" {
		return BackupResult{}, fmt.Errorf("backup path missing")
	}
	if err := os.MkdirAll(backupDir, 0o750); err != nil {
		return BackupResult{}, err
	}
	ts := now.UTC().Format("20060102-150405")
	base := filepath.Base(dbPath)
	destBase := fmt.Sprintf("%s.%s", base, ts)
	var copied []string
	mainDest := filepath.Join(backupDir, destBase)
	if err := copyFile(dbPath, mainDest); err != nil {
		return BackupResult{}, err
	}
	copied = append(copied, mainDest)
	for _, suffix := range []string{"-wal", "-shm"} {
		src := dbPath + suffix
		if _, err := os.Stat(src); err == nil {
			dst := filepath.Join(backupDir, destBase+suffix)
			if err := copyFile(src, dst); err != nil {
				return BackupResult{}, err
			}
			copied = append(copied, dst)
		}
	}
	return BackupResult{Paths: copied}, nil
}

func CleanupSQLiteBackups(backupDir string, keepDays int, now time.Time) ([]string, error) {
	if keepDays <= 0 {
		return nil, fmt.Errorf("keepDays must be > 0")
	}
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	cutoff := now.AddDate(0, 0, -keepDays)
	var deleted []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		parts := strings.Split(name, ".")
		if len(parts) < 2 {
			continue
		}
		tsPart := parts[len(parts)-1]
		if strings.HasSuffix(name, ".sqlite-wal") || strings.HasSuffix(name, ".sqlite-shm") {
			tsPart = parts[len(parts)-2]
		}
		t, err := time.Parse("20060102-150405", tsPart)
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			path := filepath.Join(backupDir, name)
			if err := os.Remove(path); err != nil {
				return deleted, err
			}
			deleted = append(deleted, path)
		}
	}
	return deleted, nil
}

func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
