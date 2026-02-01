package position

import (
	"testing"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
)

func TestDriftScoreX10000(t *testing.T) {
	diff := DriftDiff{
		OrdersMissing:         3,
		OrdersExtra:           2,
		OrdersStatusMismatch:  1,
		PositionsQtyMismatch:  2,
		PositionsSideMismatch: 1,
		BalancesMismatch:      3,
		ProtectionMismatch:    4,
	}
	score := diff.DriftScoreX10000()
	expected := (minInt((3+2+1)*4, 40) +
		minInt((2+1)*6, 30) +
		minInt(3*4, 20) +
		minInt(4*2, 10)) * 10000
	if score != expected {
		t.Fatalf("expected %d, got %d", expected, score)
	}
}

func TestEvaluateDriftAction(t *testing.T) {
	cfg := config.Default()
	diff := DriftDiff{OrdersMissing: 1}
	score := diff.DriftScoreX10000()
	if action := EvaluateDriftAction(cfg, score); action != DriftDegrade {
		t.Fatalf("expected degrade, got %s", action)
	}
	if action := EvaluateDriftAction(cfg, cfg.ReconcileDriftPauseX10000); action != DriftPause {
		t.Fatalf("expected pause, got %s", action)
	}
	if action := EvaluateDriftAction(cfg, 0); action != DriftNone {
		t.Fatalf("expected none, got %s", action)
	}
}

func TestBuildReconcileRecords(t *testing.T) {
	now := time.UnixMilli(1706700000000)
	ctx := ReconcileContext{
		RunID:          "run_1",
		CycleID:        "cyc_1",
		Mode:           "LIVE",
		SnapshotID:     "snap_1",
		DecisionID:     "dec_1",
		ExchangeTimeMs: 1706700000000,
	}
	diff := DriftDiff{OrdersMissing: 1}
	score := diff.DriftScoreX10000()
	diffRecord := BuildReconcileDiffRecord(now, ctx, diff, score)
	if diffRecord.Event.RunID != ctx.RunID || diffRecord.Event.CycleID != ctx.CycleID {
		t.Fatalf("reconcile diff record missing ids")
	}
	alertRecord := BuildReconcileAlertRecord(now, ctx, score, DriftDegrade)
	if alertRecord.Event.RunID != ctx.RunID || alertRecord.Event.CycleID != ctx.CycleID {
		t.Fatalf("reconcile alert record missing ids")
	}
}
