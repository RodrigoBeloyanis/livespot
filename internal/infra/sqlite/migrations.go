package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

func ApplySchema(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("sqlite db missing")
	}
	path, err := findSchemaPath()
	if err != nil {
		return err
	}
	buf, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("sqlite schema read failed: %w", err)
	}
	if len(buf) == 0 {
		return fmt.Errorf("sqlite schema empty")
	}
	if _, err := db.Exec(string(buf)); err != nil {
		return fmt.Errorf("sqlite schema apply failed: %w", err)
	}
	return nil
}

func findSchemaPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("sqlite schema cwd failed: %w", err)
	}
	for i := 0; i < 8; i++ {
		candidate := filepath.Join(wd, "migrations", "0001_init.sql")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return "", fmt.Errorf("sqlite schema not found: migrations/0001_init.sql")
}
