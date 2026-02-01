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
}
