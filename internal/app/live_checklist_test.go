package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
)

func TestValidateLiveChecklistOK(t *testing.T) {
	cfg := testConfig()
	status := LiveRuntimeStatus{
		DBOK:               true,
		FiltersLoaded:      true,
		ClockOK:            true,
		WSOK:               true,
		InitialReconcileOK: true,
	}
	result := ValidateLiveChecklist(cfg, status, os.Stat)
	if !result.OK {
		t.Fatalf("expected ok, got missing=%v", result.Missing)
	}
}

func TestValidateLiveChecklistMissing(t *testing.T) {
	cfg := testConfig()
	cfg.AiDec = 1
	status := LiveRuntimeStatus{}
	result := ValidateLiveChecklist(cfg, status, nil)
	if result.OK {
		t.Fatalf("expected failure")
	}
	if len(result.Reasons) == 0 {
		t.Fatalf("expected reasons")
	}
}

func TestValidateLiveChecklistOKFileRequired(t *testing.T) {
	cfg := testConfig()
	cfg.LiveRequireOKFile = true
	cfg.LiveOKFilePath = filepath.Join(t.TempDir(), "LIVE.ok")
	status := LiveRuntimeStatus{
		DBOK:               true,
		FiltersLoaded:      true,
		ClockOK:            true,
		WSOK:               true,
		InitialReconcileOK: true,
	}
	result := ValidateLiveChecklist(cfg, status, os.Stat)
	if result.OK {
		t.Fatalf("expected failure when ok file missing")
	}
}

func testConfig() config.Config {
	cfg := config.Default()
	cfg.AiDec = 2
	return cfg
}
