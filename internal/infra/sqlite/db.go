package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	_ "modernc.org/sqlite"
)

func Open(path string, busyTimeoutMs int) (*sql.DB, error) {
	if path == "" {
		return nil, fmt.Errorf("sqlite path missing")
	}
	if busyTimeoutMs <= 0 {
		return nil, fmt.Errorf("sqlite busy_timeout invalid")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("sqlite mkdir failed: %w", err)
	}
	cfg := fmt.Sprintf("file:%s?_pragma=foreign_keys(ON)", filepath.ToSlash(path))
	db, err := sql.Open("sqlite", cfg)
	if err != nil {
		return nil, fmt.Errorf("sqlite open failed: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite wal failed: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout = " + strconv.Itoa(busyTimeoutMs)); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite busy_timeout failed: %w", err)
	}
	return db, nil
}
