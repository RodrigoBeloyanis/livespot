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

func TestValidateLiveOKFileMissing(t *testing.T) {
	cfg := Default()
	cfg.LiveRequireOKFile = true
	cfg.LiveOKFilePath = filepath.Join(t.TempDir(), "LIVE.ok")
	if err := Validate(cfg, os.Stat); err == nil {
		t.Fatalf("expected error for missing live ok file")
	}
}
