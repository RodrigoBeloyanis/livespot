package health

import (
	"math"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
)

type SysMode string

const (
	SysModeNormal  SysMode = "NORMAL"
	SysModeDegrade SysMode = "DEGRADE"
	SysModePause   SysMode = "PAUSE"
	SysModeExit    SysMode = "EXIT"
)

type Signals struct {
	NowMs              int64
	LastProgressMs     int64
	WsLastMsgMs        int64
	RestLastSuccessMs  int64
	DiskFreeBytes      int64
	AuditQueuePct      int
	AuditWriterLagMs   int
	ForceExitRequested bool
}

type Result struct {
	Mode    SysMode
	Reasons []reasoncodes.ReasonCode
}

type Evaluator struct {
	cfg config.Config
}

func NewEvaluator(cfg config.Config) Evaluator {
	return Evaluator{cfg: cfg}
}

func (e Evaluator) Evaluate(current SysMode, signals Signals) Result {
	if current == SysModeExit || signals.ForceExitRequested {
		return Result{
			Mode:    SysModeExit,
			Reasons: []reasoncodes.ReasonCode{reasoncodes.ENTER_EXIT},
		}
	}

	desired := SysModeNormal
	reasons := []reasoncodes.ReasonCode{}

	addReason := func(code reasoncodes.ReasonCode) {
		for _, existing := range reasons {
			if existing == code {
				return
			}
		}
		reasons = append(reasons, code)
	}

	if shouldPause, reason := loopStuck(signals, e.cfg, true); shouldPause {
		desired = SysModePause
		addReason(reason)
	} else if shouldDegrade, reason := loopStuck(signals, e.cfg, false); shouldDegrade {
		desired = SysModeDegrade
		addReason(reason)
	}

	if shouldPause, reason := wsStale(signals, e.cfg, true); shouldPause {
		desired = SysModePause
		addReason(reason)
	} else if shouldDegrade, reason := wsStale(signals, e.cfg, false); shouldDegrade {
		if desired != SysModePause {
			desired = SysModeDegrade
		}
		addReason(reason)
	}

	if shouldPause, reason := restStale(signals, e.cfg, true); shouldPause {
		desired = SysModePause
		addReason(reason)
	} else if shouldDegrade, reason := restStale(signals, e.cfg, false); shouldDegrade {
		if desired != SysModePause {
			desired = SysModeDegrade
		}
		addReason(reason)
	}

	if shouldPause, reason := diskLow(signals, e.cfg, true); shouldPause {
		desired = SysModePause
		addReason(reason)
	} else if shouldDegrade, reason := diskLow(signals, e.cfg, false); shouldDegrade {
		if desired != SysModePause {
			desired = SysModeDegrade
		}
		addReason(reason)
	}

	if shouldPause, reason := writerPressure(signals, e.cfg, true); shouldPause {
		desired = SysModePause
		addReason(reason)
	} else if shouldDegrade, reason := writerPressure(signals, e.cfg, false); shouldDegrade {
		if desired != SysModePause {
			desired = SysModeDegrade
		}
		addReason(reason)
	}

	return Result{Mode: desired, Reasons: reasons}
}

func loopStuck(signals Signals, cfg config.Config, pause bool) (bool, reasoncodes.ReasonCode) {
	delta := deltaMs(signals.NowMs, signals.LastProgressMs)
	if pause {
		if delta >= int64(cfg.LoopStuckMsPause) {
			return true, reasoncodes.LOOP_STUCK_PAUSE
		}
		return false, ""
	}
	if delta >= int64(cfg.LoopStuckMsDegrade) {
		return true, reasoncodes.LOOP_STUCK_DEGRADE
	}
	return false, ""
}

func wsStale(signals Signals, cfg config.Config, pause bool) (bool, reasoncodes.ReasonCode) {
	delta := deltaMs(signals.NowMs, signals.WsLastMsgMs)
	if pause {
		if delta >= int64(cfg.WsStaleMsPause) {
			return true, reasoncodes.WS_STALE_PAUSE
		}
		return false, ""
	}
	if delta >= int64(cfg.WsStaleMsDegrade) {
		return true, reasoncodes.WS_STALE_DEGRADE
	}
	return false, ""
}

func restStale(signals Signals, cfg config.Config, pause bool) (bool, reasoncodes.ReasonCode) {
	delta := deltaMs(signals.NowMs, signals.RestLastSuccessMs)
	if pause {
		if delta >= int64(cfg.RestStaleMsPause) {
			return true, reasoncodes.REST_STALE_PAUSE
		}
		return false, ""
	}
	if delta >= int64(cfg.RestStaleMsDegrade) {
		return true, reasoncodes.REST_STALE_DEGRADE
	}
	return false, ""
}

func diskLow(signals Signals, cfg config.Config, pause bool) (bool, reasoncodes.ReasonCode) {
	if pause {
		if signals.DiskFreeBytes <= cfg.DiskFreePauseBytes {
			return true, reasoncodes.DISK_LOW_PAUSE
		}
		return false, ""
	}
	if signals.DiskFreeBytes <= cfg.DiskFreeDegradeBytes {
		return true, reasoncodes.DISK_LOW_DEGRADE
	}
	return false, ""
}

func writerPressure(signals Signals, cfg config.Config, pause bool) (bool, reasoncodes.ReasonCode) {
	if pause {
		if signals.AuditQueuePct >= cfg.AuditWriterQueueFull {
			return true, reasoncodes.DB_WRITER_QUEUE_FULL
		}
		return false, ""
	}
	if signals.AuditQueuePct >= cfg.AuditWriterQueueHiWatermark || signals.AuditWriterLagMs >= cfg.AuditWriterMaxLagMs {
		return true, reasoncodes.DB_WRITER_QUEUE_HIGH
	}
	return false, ""
}

func deltaMs(nowMs int64, lastMs int64) int64 {
	if lastMs <= 0 {
		return math.MaxInt64
	}
	if nowMs <= lastMs {
		return 0
	}
	return nowMs - lastMs
}
