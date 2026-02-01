package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type jsonlWriter struct {
	dir        string
	currentDay string
	file       *os.File
}

func newJSONLWriter(dir string) (*jsonlWriter, error) {
	if err := ensureDir(dir); err != nil {
		return nil, err
	}
	return &jsonlWriter{dir: dir}, nil
}

func (w *jsonlWriter) Write(ts time.Time, line []byte) error {
	if err := w.rotateIfNeeded(ts); err != nil {
		return err
	}
	if _, err := w.file.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("jsonl write: %w", err)
	}
	return nil
}

func (w *jsonlWriter) Close() error {
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}

func (w *jsonlWriter) rotateIfNeeded(ts time.Time) error {
	day := ts.UTC().Format("2006-01-02")
	if day == w.currentDay && w.file != nil {
		return nil
	}
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return fmt.Errorf("jsonl close: %w", err)
		}
	}
	path := filepath.Join(w.dir, fmt.Sprintf("audit-%s.jsonl", day))
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return fmt.Errorf("jsonl open: %w", err)
	}
	w.file = file
	w.currentDay = day
	return nil
}

func ensureDir(path string) error {
	if fi, err := os.Stat(path); err == nil {
		if fi.IsDir() {
			return nil
		}
		return fmt.Errorf("path is not a directory: %s", path)
	}
	if err := os.MkdirAll(path, 0o750); err != nil {
		return fmt.Errorf("mkdir %s: %w", path, err)
	}
	return nil
}
