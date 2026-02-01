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

func TestValidateLiveOKFileMissing(t *testing.T) {
	cfg := Default()
	cfg.LiveRequireOKFile = true
	cfg.LiveOKFilePath = filepath.Join(t.TempDir(), "LIVE.ok")
	if err := Validate(cfg, os.Stat); err == nil {
		t.Fatalf("expected error for missing live ok file")
	}
}

func TestValidateRiskPerTradeBounds(t *testing.T) {
	cfg := Default()
	cfg.RiskPerTradeUSDT = "5.00"
	if err := Validate(cfg, os.Stat); err == nil {
		t.Fatalf("expected error for risk_per_trade_usdt below min")
	}
}

func TestValidateStrategyWeightsSum(t *testing.T) {
	cfg := Default()
	cfg.StrategyWeightTrend = 0.9
	cfg.StrategyWeightPullback = 0.2
	cfg.StrategyWeightMicrostruct = 0.0
	cfg.StrategyWeightVolume = 0.0
	if err := Validate(cfg, os.Stat); err == nil {
		t.Fatalf("expected error for strategy weights sum")
	}
}
