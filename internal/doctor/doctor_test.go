package doctor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/config"
)

func TestRunAllReturnsResults(t *testing.T) {
	cfg := config.Default()
	cfg.AuditWriterQueueCapacity = 4
	cfg.LiveRequireOKFile = false

	tmp := t.TempDir()
	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()
	_ = os.Chdir(tmp)

	_ = os.MkdirAll(filepath.Join(tmp, filepath.Dir(audit.DefaultSQLitePath)), 0o750)
	_ = os.MkdirAll(filepath.Join(tmp, audit.DefaultJSONLDir), 0o750)

	results := RunAll(Runner{Cfg: cfg, Now: time.Now, Stat: os.Stat})
	if len(results) == 0 {
		t.Fatalf("expected results")
	}
	required := map[string]bool{
		"config_validate": false,
		"mode":            false,
		"ai_dec":          false,
		"live_ok_file":    false,
		"data_dir":        false,
		"audit_sqlite":    false,
		"audit_jsonl":     false,
	}
	for _, result := range results {
		if _, ok := required[result.Name]; ok {
			required[result.Name] = true
		}
	}
	for name, seen := range required {
		if !seen {
			t.Fatalf("missing check result: %s", name)
		}
	}
}
