package ops

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func CleanupJSONL(dir string, keepDays int, now time.Time) ([]string, error) {
	if keepDays <= 0 {
		return nil, fmt.Errorf("keepDays must be > 0")
	}
	pattern := filepath.Join(dir, "audit-*.jsonl")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	var deleted []string
	cutoff := now.AddDate(0, 0, -keepDays)
	for _, path := range files {
		base := filepath.Base(path)
		if len(base) < len("audit-2006-01-02.jsonl") {
			continue
		}
		datePart := base[len("audit-") : len("audit-")+10]
		day, err := time.Parse("2006-01-02", datePart)
		if err != nil {
			continue
		}
		if day.Before(cutoff) {
			if err := os.Remove(path); err != nil {
				return deleted, err
			}
			deleted = append(deleted, path)
		}
	}
	return deleted, nil
}

func CleanupLogs(dir string, keepDays int, now time.Time) ([]string, error) {
	if keepDays <= 0 {
		return nil, fmt.Errorf("keepDays must be > 0")
	}
	entries, err := os.ReadDir(dir)
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
		if filepath.Ext(entry.Name()) != ".log" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return deleted, err
		}
		if info.ModTime().Before(cutoff) {
			path := filepath.Join(dir, entry.Name())
			if err := os.Remove(path); err != nil {
				return deleted, err
			}
			deleted = append(deleted, path)
		}
	}
	return deleted, nil
}
