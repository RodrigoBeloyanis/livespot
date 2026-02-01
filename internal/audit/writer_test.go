package audit

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	auditdomain "github.com/RodrigoBeloyanis/livespot/internal/domain/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"

	_ "modernc.org/sqlite"
)

func TestWriterWritesSQLiteAndJSONL(t *testing.T) {
	tmp := t.TempDir()
	cfg := testConfig()
	dbPath := filepath.Join(tmp, "data", "audit.sqlite")
	jsonlDir := filepath.Join(tmp, "logs")
	now := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	writer, err := NewWriter(cfg, WriterOptions{DBPath: dbPath, JSONLDir: jsonlDir, Now: func() time.Time { return now }})
	if err != nil {
		t.Fatalf("unexpected writer error: %v", err)
	}

	event := auditdomain.AuditEvent{
		TsMs:            now.UnixMilli(),
		RunID:           "run_20260201_120000_abcdef",
		CycleID:         "cyc_20260201_120000_abcdef",
		Mode:            "LIVE",
		Stage:           observability.BOOT,
		EventType:       auditdomain.ALERT_RAISED,
		Reasons:         []reasoncodes.ReasonCode{reasoncodes.STRAT_CONFIG_INVALID},
		SnapshotID:      "",
		DecisionID:      "",
		OrderIntentID:   "",
		ExchangeTimeMs:  0,
		LocalReceivedMs: now.UnixMilli(),
	}
	record := Record{
		Event: event,
		Data: map[string]any{
			"message": "config invalid",
		},
	}
	if err := writer.Write(record); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM audit_events").Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row, got %d", count)
	}

	jsonlPath := filepath.Join(jsonlDir, "audit-2026-02-01.jsonl")
	file, err := os.Open(jsonlPath)
	if err != nil {
		t.Fatalf("open jsonl: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		t.Fatalf("expected jsonl line")
	}
	var payload map[string]any
	if err := json.Unmarshal(scanner.Bytes(), &payload); err != nil {
		t.Fatalf("json decode: %v", err)
	}
	if payload["run_id"] != event.RunID {
		t.Fatalf("run_id mismatch")
	}
	if payload["event_type"] != string(event.EventType) {
		t.Fatalf("event_type mismatch")
	}
}

func TestWriterRejectsJSONLDirFile(t *testing.T) {
	tmp := t.TempDir()
	cfg := testConfig()
	path := filepath.Join(tmp, "logs")
	if err := os.WriteFile(path, []byte("file"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	_, err := NewWriter(cfg, WriterOptions{DBPath: filepath.Join(tmp, "data", "audit.sqlite"), JSONLDir: path, Now: time.Now})
	if err == nil {
		t.Fatalf("expected error for jsonl dir file")
	}
}

func testConfig() config.Config {
	cfg := config.Default()
	cfg.AuditWriterQueueCapacity = 4
	return cfg
}
