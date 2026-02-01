package app

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"

	_ "modernc.org/sqlite"
)

func TestRunDryRunEmitsStageEvents(t *testing.T) {
	cfg := config.Default()
	cfg.AuditWriterQueueCapacity = 8
	tmp := t.TempDir()
	writer, err := audit.NewWriter(cfg, audit.WriterOptions{DBPath: filepath.Join(tmp, "data", "audit.sqlite"), JSONLDir: filepath.Join(tmp, "logs"), Now: time.Now})
	if err != nil {
		t.Fatalf("writer create: %v", err)
	}
	loop, err := NewLoop(cfg, writer, observability.ConsoleStageReporter{}, time.Now, nil, nil, nil)
	if err != nil {
		t.Fatalf("new loop: %v", err)
	}
	if err := loop.RunDryRun(); err != nil {
		t.Fatalf("dry run: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer close: %v", err)
	}

	db, err := sql.Open("sqlite", filepath.Join(tmp, "data", "audit.sqlite"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM audit_events WHERE event_type='STAGE_CHANGED'").Scan(&count); err != nil {
		t.Fatalf("count stage events: %v", err)
	}
	if count == 0 {
		t.Fatalf("expected stage events")
	}
}
