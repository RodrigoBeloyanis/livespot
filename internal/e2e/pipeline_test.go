package e2e

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/config"

	_ "modernc.org/sqlite"
)

func TestPipelineRunOnceWritesStages(t *testing.T) {
	cfg := config.Default()
	tmp := t.TempDir()
	writer, err := audit.NewWriter(cfg, audit.WriterOptions{
		DBPath:   filepath.Join(tmp, "data", "audit.sqlite"),
		JSONLDir: filepath.Join(tmp, "logs"),
		Now:      time.Now,
	})
	if err != nil {
		t.Fatalf("writer create: %v", err)
	}
	defer writer.Close()

	fixed := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	exchange := NewMockExchange(func() time.Time { return fixed })
	exchange.FiltersJSON = []byte(`{}`)
	exchange.DepthBySymbol["BTCUSDT"] = []byte(`{}`)
	exchange.BalancesJSON = []byte(`{}`)

	snapshot, err := SampleSnapshot(fixed)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	decision, err := SampleDecision(fixed, snapshot, snapshot.Metadata.SnapshotHash, "cyc_test")
	if err != nil {
		t.Fatalf("decision: %v", err)
	}
	pipeline, err := NewPipeline(cfg, writer, nil)
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}
	_, err = pipeline.RunOnce(nil, PipelineInput{
		RunID:        "run_test",
		CycleID:      "cyc_test",
		Mode:         cfg.Mode,
		Snapshot:     snapshot,
		SnapshotHash: snapshot.Metadata.SnapshotHash,
		Decision:     decision,
		DecisionID:   decision.DecisionID,
		Exchange:     exchange,
		Now:          func() time.Time { return fixed },
	})
	if err != nil {
		t.Fatalf("run once: %v", err)
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
	if count != 5 {
		t.Fatalf("expected 5 stage events, got %d", count)
	}
}
