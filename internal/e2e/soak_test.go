package e2e

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
)

func TestSoakRunnerPass(t *testing.T) {
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

	now := time.Now()
	exchange := NewMockExchange(time.Now)
	exchange.FiltersJSON = []byte(`{}`)
	exchange.DepthBySymbol["BTCUSDT"] = []byte(`{}`)
	exchange.BalancesJSON = []byte(`{}`)

	snapshot, err := SampleSnapshot(now)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	decision, err := SampleDecision(now, snapshot, snapshot.Metadata.SnapshotHash, "cyc_soak")
	if err != nil {
		t.Fatalf("decision: %v", err)
	}
	pipeline, err := NewPipeline(cfg, writer, nil)
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}

	runner := SoakRunner{
		Pipeline: pipeline,
		Config:   cfg,
		Input: PipelineInput{
			RunID:          "run_soak",
			CycleID:        "cyc_soak",
			Mode:           cfg.Mode,
			Snapshot:       snapshot,
			SnapshotHash:   snapshot.Metadata.SnapshotHash,
			Decision:       decision,
			DecisionID:     decision.DecisionID,
			Exchange:       exchange,
			Now:            time.Now,
			DisableEntries: true,
		},
		Duration: 50 * time.Millisecond,
		Interval: 10 * time.Millisecond,
	}
	result, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("soak run: %v", err)
	}
	if !result.Pass {
		t.Fatalf("expected soak pass")
	}
}

func TestSoakRunnerDetectsStale(t *testing.T) {
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

	now := time.Now()
	exchange := NewMockExchange(time.Now)
	exchange.FiltersJSON = []byte(`{}`)
	exchange.DepthBySymbol["BTCUSDT"] = []byte(`{}`)
	exchange.BalancesJSON = []byte(`{}`)

	snapshot, err := SampleSnapshot(now)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	decision, err := SampleDecision(now, snapshot, snapshot.Metadata.SnapshotHash, "cyc_stale")
	if err != nil {
		t.Fatalf("decision: %v", err)
	}
	pipeline, err := NewPipeline(cfg, writer, nil)
	if err != nil {
		t.Fatalf("pipeline: %v", err)
	}

	runner := SoakRunner{
		Pipeline: pipeline,
		Config:   cfg,
		Input: PipelineInput{
			RunID:          "run_stale",
			CycleID:        "cyc_stale",
			Mode:           cfg.Mode,
			Snapshot:       snapshot,
			SnapshotHash:   snapshot.Metadata.SnapshotHash,
			Decision:       decision,
			DecisionID:     decision.DecisionID,
			Exchange:       exchange,
			Now:            time.Now,
			DisableEntries: true,
			Signals: SoakSignals{
				WsLastMsgTsMs:  now.Add(-time.Duration(cfg.WsStaleMsDegrade) * time.Millisecond).Add(-time.Second).UnixMilli(),
				RestLastOkTsMs: now.UnixMilli(),
			},
		},
		Duration: 20 * time.Millisecond,
		Interval: 10 * time.Millisecond,
	}
	result, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("soak run: %v", err)
	}
	if result.Pass {
		t.Fatalf("expected soak failure")
	}
}

func TestReadinessReport(t *testing.T) {
	report := BuildReadinessReport("run_ready", SoakResult{
		StartTsMs: 1,
		EndTsMs:   2,
		Pass:      false,
		Violations: []SoakViolation{
			{Reason: reasoncodes.WS_STALE_DEGRADE, AtTsMs: 2, Detail: "ws stale"},
		},
	}, true, true)
	if report.Ready {
		t.Fatalf("expected readiness false")
	}
	if len(report.Checks) != 3 {
		t.Fatalf("expected 3 checks")
	}
}
