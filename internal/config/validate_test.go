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

func TestValidateTopKSizeInvalid(t *testing.T) {
	cfg := Default()
	cfg.TopKSize = cfg.TopNSize + 1
	if err := Validate(cfg, os.Stat); err == nil {
		t.Fatalf("expected error for topk_size > topn_size")
	}
}

func TestValidateRankWeightsSum(t *testing.T) {
	cfg := Default()
	cfg.RankWeightLiquidity = 0.9
	cfg.RankWeightMomentum = 0.2
	cfg.RankWeightSpread = 0.0
	if err := Validate(cfg, os.Stat); err == nil {
		t.Fatalf("expected error for rank weights sum")
	}
}

func TestValidateDeepWeightsSum(t *testing.T) {
	cfg := Default()
	cfg.DeepWeightEdge = 0.7
	cfg.DeepWeightRegime = 0.2
	cfg.DeepWeightMicrostructure = 0.2
	cfg.DeepWeightVolatility = 0.0
	if err := Validate(cfg, os.Stat); err == nil {
		t.Fatalf("expected error for deep weights sum")
	}
}
