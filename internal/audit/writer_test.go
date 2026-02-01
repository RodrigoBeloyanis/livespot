package audit

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	domainaudit "github.com/RodrigoBeloyanis/livespot/internal/domain/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"
	_ "modernc.org/sqlite"
)

func TestWriterWriteEventSQLiteJSONL(t *testing.T) {
	baseDir := t.TempDir()
	cfg := config.Default()
	cfg.AuditSQLitePath = filepath.Join(baseDir, "data", "audit.sqlite")
	cfg.AuditJSONLDir = filepath.Join(baseDir, "logs")

	writer, err := NewWriter(cfg)
	if err != nil {
		t.Fatalf("new writer failed: %v", err)
	}
	defer func() {
		_ = writer.Close()
	}()

	now := time.Now().UTC()
	event := domainaudit.AuditEvent{
		TsMs:            now.UnixMilli(),
		RunID:           "run_test",
		CycleID:         "cyc_test",
		Mode:            "LIVE",
		Stage:           observability.BOOT,
		EventType:       domainaudit.STAGE_CHANGED,
		Reasons:         []reasoncodes.ReasonCode{},
		SnapshotID:      "",
		DecisionID:      "",
		OrderIntentID:   "",
		ExchangeTimeMs:  0,
		LocalReceivedMs: now.UnixMilli(),
	}

	payload := map[string]any{"msg": "ok"}
	if err := writer.WriteEvent(context.Background(), event, payload); err != nil {
		t.Fatalf("write event failed: %v", err)
	}

	date := now.Format("2006-01-02")
	jsonlPath := filepath.Join(cfg.AuditJSONLDir, "audit-"+date+".jsonl")
	if _, err := os.Stat(jsonlPath); err != nil {
		t.Fatalf("jsonl file missing: %v", err)
	}

	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(cfg.AuditSQLitePath))
	if err != nil {
		t.Fatalf("sqlite open failed: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()
	var count int
	if err := db.QueryRow("SELECT COUNT(1) FROM audit_events").Scan(&count); err != nil {
		t.Fatalf("sqlite query failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 audit_events row, got %d", count)
	}
}
