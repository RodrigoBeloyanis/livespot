package health

import (
	"testing"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
)

func TestEvaluatorLoopStuckTransitions(t *testing.T) {
	cfg := config.Default()
	eval := NewEvaluator(cfg)
	now := int64(100000)

	signals := baseSignals(cfg, now)
	signals.LastProgressMs = now - int64(cfg.LoopStuckMsDegrade)
	result := eval.Evaluate(SysModeNormal, signals)
	if result.Mode != SysModeDegrade {
		t.Fatalf("expected degrade, got %s", result.Mode)
	}
	if !containsReason(result.Reasons, reasoncodes.LOOP_STUCK_DEGRADE) {
		t.Fatalf("expected loop stuck degrade reason")
	}

	signals = baseSignals(cfg, now)
	signals.LastProgressMs = now - int64(cfg.LoopStuckMsPause)
	result = eval.Evaluate(SysModeNormal, signals)
	if result.Mode != SysModePause {
		t.Fatalf("expected pause, got %s", result.Mode)
	}
	if !containsReason(result.Reasons, reasoncodes.LOOP_STUCK_PAUSE) {
		t.Fatalf("expected loop stuck pause reason")
	}
}

func TestEvaluatorFeedStaleTransitions(t *testing.T) {
	cfg := config.Default()
	eval := NewEvaluator(cfg)
	now := int64(200000)

	signals := baseSignals(cfg, now)
	signals.WsLastMsgMs = now - int64(cfg.WsStaleMsDegrade)
	result := eval.Evaluate(SysModeNormal, signals)
	if result.Mode != SysModeDegrade {
		t.Fatalf("expected degrade, got %s", result.Mode)
	}
	if !containsReason(result.Reasons, reasoncodes.WS_STALE_DEGRADE) {
		t.Fatalf("expected ws stale degrade reason")
	}

	signals = baseSignals(cfg, now)
	signals.WsLastMsgMs = now - int64(cfg.WsStaleMsPause)
	result = eval.Evaluate(SysModeNormal, signals)
	if result.Mode != SysModePause {
		t.Fatalf("expected pause, got %s", result.Mode)
	}
	if !containsReason(result.Reasons, reasoncodes.WS_STALE_PAUSE) {
		t.Fatalf("expected ws stale pause reason")
	}

	signals = baseSignals(cfg, now)
	signals.RestLastSuccessMs = now - int64(cfg.RestStaleMsDegrade)
	result = eval.Evaluate(SysModeNormal, signals)
	if result.Mode != SysModeDegrade {
		t.Fatalf("expected degrade, got %s", result.Mode)
	}
	if !containsReason(result.Reasons, reasoncodes.REST_STALE_DEGRADE) {
		t.Fatalf("expected rest stale degrade reason")
	}

	signals = baseSignals(cfg, now)
	signals.RestLastSuccessMs = now - int64(cfg.RestStaleMsPause)
	result = eval.Evaluate(SysModeNormal, signals)
	if result.Mode != SysModePause {
		t.Fatalf("expected pause, got %s", result.Mode)
	}
	if !containsReason(result.Reasons, reasoncodes.REST_STALE_PAUSE) {
		t.Fatalf("expected rest stale pause reason")
	}
}

func TestEvaluatorDiskLowTransitions(t *testing.T) {
	cfg := config.Default()
	eval := NewEvaluator(cfg)
	now := int64(300000)

	signals := baseSignals(cfg, now)
	signals.DiskFreeBytes = cfg.DiskFreeDegradeBytes
	result := eval.Evaluate(SysModeNormal, signals)
	if result.Mode != SysModeDegrade {
		t.Fatalf("expected degrade, got %s", result.Mode)
	}
	if !containsReason(result.Reasons, reasoncodes.DISK_LOW_DEGRADE) {
		t.Fatalf("expected disk low degrade reason")
	}

	signals = baseSignals(cfg, now)
	signals.DiskFreeBytes = cfg.DiskFreePauseBytes
	result = eval.Evaluate(SysModeNormal, signals)
	if result.Mode != SysModePause {
		t.Fatalf("expected pause, got %s", result.Mode)
	}
	if !containsReason(result.Reasons, reasoncodes.DISK_LOW_PAUSE) {
		t.Fatalf("expected disk low pause reason")
	}
}

func TestEvaluatorWriterPressureTransitions(t *testing.T) {
	cfg := config.Default()
	eval := NewEvaluator(cfg)
	now := int64(400000)

	signals := baseSignals(cfg, now)
	signals.AuditQueuePct = cfg.AuditWriterQueueHiWatermark
	result := eval.Evaluate(SysModeNormal, signals)
	if result.Mode != SysModeDegrade {
		t.Fatalf("expected degrade, got %s", result.Mode)
	}
	if !containsReason(result.Reasons, reasoncodes.DB_WRITER_QUEUE_HIGH) {
		t.Fatalf("expected writer queue high reason")
	}

	signals = baseSignals(cfg, now)
	signals.AuditQueuePct = cfg.AuditWriterQueueFull
	result = eval.Evaluate(SysModeNormal, signals)
	if result.Mode != SysModePause {
		t.Fatalf("expected pause, got %s", result.Mode)
	}
	if !containsReason(result.Reasons, reasoncodes.DB_WRITER_QUEUE_FULL) {
		t.Fatalf("expected writer queue full reason")
	}
}

func baseSignals(cfg config.Config, now int64) Signals {
	return Signals{
		NowMs:             now,
		LastProgressMs:    now,
		WsLastMsgMs:       now,
		RestLastSuccessMs: now,
		DiskFreeBytes:     cfg.DiskFreeDegradeBytes + 1,
		AuditQueuePct:     0,
		AuditWriterLagMs:  0,
	}
}

func containsReason(codes []reasoncodes.ReasonCode, want reasoncodes.ReasonCode) bool {
	for _, code := range codes {
		if code == want {
			return true
		}
	}
	return false
}
