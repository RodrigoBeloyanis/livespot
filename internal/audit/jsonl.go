package audit

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type JSONLWriter struct {
	dir      string
	file     *os.File
	writer   *bufio.Writer
	currDate string
	mu       sync.Mutex
	now      func() time.Time
}

func NewJSONLWriter(dir string) (*JSONLWriter, error) {
	return newJSONLWriter(dir, time.Now)
}

func newJSONLWriter(dir string, now func() time.Time) (*JSONLWriter, error) {
	if dir == "" {
		return nil, fmt.Errorf("jsonl dir missing")
	}
	w := &JSONLWriter{dir: dir, now: now}
	if err := w.rotateIfNeeded(); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *JSONLWriter) WriteEvent(event any) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.rotateIfNeeded(); err != nil {
		return err
	}
	buf, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("jsonl marshal failed: %w", err)
	}
	if _, err := w.writer.Write(buf); err != nil {
		return fmt.Errorf("jsonl write failed: %w", err)
	}
	if err := w.writer.WriteByte('\n'); err != nil {
		return fmt.Errorf("jsonl write newline failed: %w", err)
	}
	if err := w.writer.Flush(); err != nil {
		return fmt.Errorf("jsonl flush failed: %w", err)
	}
	return nil
}

func (w *JSONLWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.writer != nil {
		if err := w.writer.Flush(); err != nil {
			return err
		}
	}
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

func (w *JSONLWriter) rotateIfNeeded() error {
	if err := os.MkdirAll(w.dir, 0o755); err != nil {
		return fmt.Errorf("jsonl mkdir failed: %w", err)
	}
	date := w.now().UTC().Format("2006-01-02")
	if w.currDate == date && w.file != nil {
		return nil
	}
	if w.file != nil {
		_ = w.file.Close()
	}
	path := filepath.Join(w.dir, fmt.Sprintf("audit-%s.jsonl", date))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("jsonl open failed: %w", err)
	}
	w.file = file
	w.writer = bufio.NewWriter(file)
	w.currDate = date
	return nil
}
