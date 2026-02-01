package app

import (
	"io/fs"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
)

type LiveRuntimeStatus struct {
	DBOK               bool
	FiltersLoaded      bool
	ClockOK            bool
	WSOK               bool
	InitialReconcileOK bool
}

type LiveChecklistResult struct {
	OK      bool
	Reasons []reasoncodes.ReasonCode
	Missing []string
}

type StatFunc func(string) (fs.FileInfo, error)

func ValidateLiveChecklist(cfg config.Config, status LiveRuntimeStatus, stat StatFunc) LiveChecklistResult {
	missing := make([]string, 0, 6)
	if cfg.Mode != "LIVE" {
		missing = append(missing, "mode")
	}
	if cfg.AiDec != 2 {
		missing = append(missing, "ai_dec")
	}
	if cfg.LiveRequireOKFile {
		if stat == nil {
			missing = append(missing, "live_ok_file_stat")
		} else if _, err := stat(cfg.LiveOKFilePath); err != nil {
			missing = append(missing, "live_ok_file")
		}
	}
	if !status.DBOK {
		missing = append(missing, "db_ok")
	}
	if !status.FiltersLoaded {
		missing = append(missing, "filters_loaded")
	}
	if !status.ClockOK {
		missing = append(missing, "clock_ok")
	}
	if !status.WSOK {
		missing = append(missing, "ws_ok")
	}
	if !status.InitialReconcileOK {
		missing = append(missing, "initial_reconcile_ok")
	}
	if len(missing) == 0 {
		return LiveChecklistResult{OK: true}
	}
	return LiveChecklistResult{
		OK:      false,
		Reasons: []reasoncodes.ReasonCode{reasoncodes.ENTER_PAUSE},
		Missing: missing,
	}
}
