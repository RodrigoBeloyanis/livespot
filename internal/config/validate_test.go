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

func TestValidateRetentionDefaults(t *testing.T) {
	cfg := Default()
	cfg.JSONLRetentionDays = 0
	if err := Validate(cfg, os.Stat); err == nil {
		t.Fatalf("expected error for jsonl_retention_days")
	}
	cfg = Default()
	cfg.LogRetentionDays = 0
	if err := Validate(cfg, os.Stat); err == nil {
		t.Fatalf("expected error for log_retention_days")
	}
	cfg = Default()
	cfg.SQLiteBackupRetentionDays = 0
	if err := Validate(cfg, os.Stat); err == nil {
		t.Fatalf("expected error for sqlite_backup_retention_days")
	}
	cfg = Default()
	cfg.SQLiteBackupDir = ""
	if err := Validate(cfg, os.Stat); err == nil {
		t.Fatalf("expected error for sqlite_backup_dir")
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
