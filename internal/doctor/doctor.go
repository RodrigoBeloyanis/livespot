package doctor

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/infra/sqlite"
)

type CheckResult struct {
	Name    string
	OK      bool
	Details string
}

type Runner struct {
	Cfg  config.Config
	Now  func() time.Time
	Stat func(string) (fs.FileInfo, error)
}

func RunAll(r Runner) []CheckResult {
	if r.Now == nil {
		r.Now = time.Now
	}
	if r.Stat == nil {
		r.Stat = os.Stat
	}
	results := []CheckResult{
		checkMode(r.Cfg),
		checkAiDec(r.Cfg),
		checkLiveOKFile(r.Cfg, r.Stat),
		checkSQLite(r.Cfg, r.Now),
		checkJSONLWritable(r.Now),
	}
	return results
}

func checkMode(cfg config.Config) CheckResult {
	if cfg.Mode != "LIVE" {
		return CheckResult{Name: "mode", OK: false, Details: "mode must be LIVE"}
	}
	return CheckResult{Name: "mode", OK: true, Details: "LIVE"}
}

func checkAiDec(cfg config.Config) CheckResult {
	if cfg.AiDec != 2 {
		return CheckResult{Name: "ai_dec", OK: false, Details: "ai_dec must be 2 in LIVE"}
	}
	return CheckResult{Name: "ai_dec", OK: true, Details: "2"}
}

func checkLiveOKFile(cfg config.Config, stat func(string) (fs.FileInfo, error)) CheckResult {
	if !cfg.LiveRequireOKFile {
		return CheckResult{Name: "live_ok_file", OK: true, Details: "not required"}
	}
	if cfg.LiveOKFilePath == "" {
		return CheckResult{Name: "live_ok_file", OK: false, Details: "path missing"}
	}
	if _, err := stat(cfg.LiveOKFilePath); err != nil {
		return CheckResult{Name: "live_ok_file", OK: false, Details: "missing"}
	}
	return CheckResult{Name: "live_ok_file", OK: true, Details: "present"}
}

func checkSQLite(cfg config.Config, now func() time.Time) CheckResult {
	db, err := sqlite.Open(audit.DefaultSQLitePath, cfg)
	if err != nil {
		return CheckResult{Name: "audit_sqlite", OK: false, Details: err.Error()}
	}
	defer db.Close()
	if err := audit.EnsureSchema(db, now()); err != nil {
		return CheckResult{Name: "audit_sqlite", OK: false, Details: err.Error()}
	}
	return CheckResult{Name: "audit_sqlite", OK: true, Details: audit.DefaultSQLitePath}
}

func checkJSONLWritable(now func() time.Time) CheckResult {
	if err := os.MkdirAll(audit.DefaultJSONLDir, 0o750); err != nil {
		return CheckResult{Name: "audit_jsonl", OK: false, Details: err.Error()}
	}
	path := filepath.Join(audit.DefaultJSONLDir, fmt.Sprintf("doctor-%d.tmp", now().UnixNano()))
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return CheckResult{Name: "audit_jsonl", OK: false, Details: err.Error()}
	}
	_, writeErr := file.WriteString("ok\n")
	closeErr := file.Close()
	_ = os.Remove(path)
	if writeErr != nil {
		return CheckResult{Name: "audit_jsonl", OK: false, Details: writeErr.Error()}
	}
	if closeErr != nil {
		return CheckResult{Name: "audit_jsonl", OK: false, Details: closeErr.Error()}
	}
	return CheckResult{Name: "audit_jsonl", OK: true, Details: audit.DefaultJSONLDir}
}
