package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateValidConfig(t *testing.T) {
	cfg := Default()
	if err := Validate(cfg, os.Stat); err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
}

func TestValidateMissingRequiredField(t *testing.T) {
	cfg := Default()
	cfg.LoopStuckMsDegrade = 0
	if err := Validate(cfg, os.Stat); err == nil {
		t.Fatalf("expected error for missing loop_stuck_ms_degrade")
	}
}

func TestValidateInvalidValue(t *testing.T) {
	cfg := Default()
	cfg.WebuiPort = 0
	if err := Validate(cfg, os.Stat); err == nil {
		t.Fatalf("expected error for invalid webui_port")
	}
}

func TestValidateInvalidQueueCapacity(t *testing.T) {
	cfg := Default()
	cfg.AuditWriterQueueCapacity = 0
	if err := Validate(cfg, os.Stat); err == nil {
		t.Fatalf("expected error for invalid audit_writer_queue_capacity")
	}
}

func TestValidateStrategyMinEdgeFallback(t *testing.T) {
	cfg := Default()
	cfg.StrategyMinEdgeBpsFallback = cfg.StrategyMinEdgeBps - 1
	if err := Validate(cfg, os.Stat); err == nil {
		t.Fatalf("expected error for strategy_min_edge_bps_fallback")
	}
}

func TestValidateCorrMaxX10000(t *testing.T) {
	cfg := Default()
	cfg.CorrMaxX10000 = 0
	if err := Validate(cfg, os.Stat); err == nil {
		t.Fatalf("expected error for corr_max_x10000")
	}
}

func TestValidateLiveOKFileMissing(t *testing.T) {
	cfg := Default()
	cfg.LiveRequireOKFile = true
	cfg.LiveOKFilePath = filepath.Join(t.TempDir(), "LIVE.ok")
	if err := Validate(cfg, os.Stat); err == nil {
		t.Fatalf("expected error for missing live ok file")
	}
}
