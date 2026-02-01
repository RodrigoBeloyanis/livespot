package sqlite

import (
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/RodrigoBeloyanis/livespot/migrations"
)

func Migrate(db *sql.DB, now time.Time) error {
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}
	applied, err := appliedMigrations(db)
	if err != nil {
		return err
	}
	entries, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		return fmt.Errorf("migrations glob: %w", err)
	}
	sort.Strings(entries)
	for _, name := range entries {
		if applied[name] {
			continue
		}
		if err := applyMigration(db, name, now); err != nil {
			return err
		}
	}
	return nil
}

func ensureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
  name TEXT NOT NULL PRIMARY KEY,
  applied_at_ms INTEGER NOT NULL
);`)
	if err != nil {
		return fmt.Errorf("migrations table: %w", err)
	}
	return nil
}

func appliedMigrations(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT name FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("migrations query: %w", err)
	}
	defer rows.Close()
	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("migrations scan: %w", err)
		}
		applied[name] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("migrations rows: %w", err)
	}
	return applied, nil
}

func applyMigration(db *sql.DB, name string, now time.Time) error {
	buf, err := fs.ReadFile(migrations.FS, name)
	if err != nil {
		return fmt.Errorf("migration read %s: %w", name, err)
	}
	sqlText := strings.TrimSpace(string(buf))
	if sqlText == "" {
		return fmt.Errorf("migration empty: %s", name)
	}
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("migration begin %s: %w", name, err)
	}
	if _, err := tx.Exec(sqlText); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("migration exec %s: %w", name, err)
	}
	if _, err := tx.Exec("INSERT INTO schema_migrations(name, applied_at_ms) VALUES(?, ?)", name, now.UnixMilli()); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("migration mark %s: %w", name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migration commit %s: %w", name, err)
	}
	return nil
}
