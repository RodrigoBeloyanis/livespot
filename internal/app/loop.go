package app

import (
	"context"
	"fmt"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	auditdomain "github.com/RodrigoBeloyanis/livespot/internal/domain/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/health"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"
)

type Loop struct {
	cfg      config.Config
	writer   *audit.Writer
	reporter observability.StageReporter
	now      func() time.Time
	sysEval  health.Evaluator
	sysMode  health.SysMode

	sysModeSince   time.Time
	sysModeReasons []reasoncodes.ReasonCode

	lastProgressMs    int64
	wsLastMsgMs       int64
	restLastSuccessMs int64
	diskFreeBytes     int64
	auditWriterLagMs  int
	forceExit         bool
}

func NewLoop(cfg config.Config, writer *audit.Writer, reporter observability.StageReporter, now func() time.Time) (*Loop, error) {
	if writer == nil {
		return nil, fmt.Errorf("audit writer missing")
	}
	if now == nil {
		now = time.Now
	}
	return &Loop{
		cfg:          cfg,
		writer:       writer,
		reporter:     reporter,
		now:          now,
		sysEval:      health.NewEvaluator(cfg),
		sysMode:      health.SysModeNormal,
		sysModeSince: now(),
	}, nil
}

func (l *Loop) RunDryRun() error {
	runID, err := observability.NewRunID(l.now())
	if err != nil {
		return err
	}
	cycleID, err := observability.NewCycleID(l.now())
	if err != nil {
		return err
	}
	for _, stage := range l.stageSequence() {
		if err := l.emitStage(runID, cycleID, stage, "", "dry-run"); err != nil {
			return err
		}
	}
	return nil
}

func (l *Loop) Run(ctx context.Context) error {
	runID, err := observability.NewRunID(l.now())
	if err != nil {
		return err
	}
	l.lastProgressMs = l.now().UnixMilli()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		cycleID, err := observability.NewCycleID(l.now())
		if err != nil {
			return err
		}
		if err := l.runCycle(ctx, runID, cycleID); err != nil {
			return err
		}
	}
}

func (l *Loop) runCycle(ctx context.Context, runID string, cycleID string) error {
	for _, stage := range l.stageSequence() {
		if err := l.refreshSysMode(ctx, runID, cycleID); err != nil {
			return err
		}
		if l.sysMode == health.SysModePause {
			if err := l.emitStage(runID, cycleID, observability.PAUSE, "", "paused"); err != nil {
				return err
			}
			if err := sleepCtx(ctx, 500*time.Millisecond); err != nil {
				return err
			}
			continue
		}
		summary := "not implemented"
		if l.sysMode == health.SysModeDegrade {
			summary = "degraded: entries blocked"
		}
		if err := l.emitStage(runID, cycleID, stage, "", summary); err != nil {
			return err
		}
	}
	return nil
}

func (l *Loop) stageSequence() []observability.StageName {
	return []observability.StageName{
		observability.BOOT,
		observability.DOCTOR_CHECKS,
		observability.STARTUP_RECOVER,
		observability.UNIVERSE_SCAN,
		observability.RANK_TOPN,
		observability.DEEP_SCAN,
		observability.WATCHLIST_ATTACH,
		observability.STATE_UPDATE,
		observability.STRATEGY_PROPOSE,
		observability.AIGATE_CALL,
		observability.RISK_VERDICT,
		observability.EXECUTE_INTENT,
		observability.POSITION_MANAGE,
		observability.RECONCILE_REST,
		observability.REPORT_DAILY_SUMMARY,
		observability.SHUTDOWN,
	}
}

func (l *Loop) UpdateWSLastMsg(ts time.Time) {
	l.wsLastMsgMs = ts.UnixMilli()
}

func (l *Loop) UpdateRESTLastSuccess(ts time.Time) {
	l.restLastSuccessMs = ts.UnixMilli()
}

func (l *Loop) UpdateDiskFreeBytes(bytes int64) {
	l.diskFreeBytes = bytes
}

func (l *Loop) UpdateAuditWriterLagMs(ms int) {
	l.auditWriterLagMs = ms
}

func (l *Loop) RequestExit() {
	l.forceExit = true
}

func (l *Loop) emitStage(runID string, cycleID string, stage observability.StageName, symbol string, summary string) error {
	now := l.now()
	l.lastProgressMs = now.UnixMilli()
	if l.reporter != nil {
		l.reporter.StageChanged(now, stage, cycleID, "", symbol, summary)
	}
	record := audit.Record{
		Event: auditdomain.AuditEvent{
			TsMs:            now.UnixMilli(),
			RunID:           runID,
			CycleID:         cycleID,
			Mode:            l.cfg.Mode,
			Stage:           stage,
			EventType:       auditdomain.STAGE_CHANGED,
			Reasons:         []reasoncodes.ReasonCode{},
			SnapshotID:      "",
			DecisionID:      "",
			OrderIntentID:   "",
			ExchangeTimeMs:  0,
			LocalReceivedMs: now.UnixMilli(),
		},
		Data: map[string]any{
			"symbol":  symbol,
			"summary": summary,
		},
	}
	if err := l.writer.Write(record); err != nil {
		return fmt.Errorf("audit stage write: %w", err)
	}
	return nil
}

func (l *Loop) refreshSysMode(ctx context.Context, runID string, cycleID string) error {
	if err := l.sampleDiskFree(); err != nil {
		return err
	}
	queueLen, queueCap := l.writer.QueueStats()
	queuePct := 0
	if queueCap > 0 {
		queuePct = int(float64(queueLen) / float64(queueCap) * 100)
	}
	signals := health.Signals{
		NowMs:              l.now().UnixMilli(),
		LastProgressMs:     l.lastProgressMs,
		WsLastMsgMs:        l.wsLastMsgMs,
		RestLastSuccessMs:  l.restLastSuccessMs,
		DiskFreeBytes:      l.diskFreeBytes,
		AuditQueuePct:      queuePct,
		AuditWriterLagMs:   l.auditWriterLagMs,
		ForceExitRequested: l.forceExit,
	}
	result := l.sysEval.Evaluate(l.sysMode, signals)
	if result.Mode == l.sysMode {
		return nil
	}
	l.sysMode = result.Mode
	l.sysModeSince = l.now()
	l.sysModeReasons = result.Reasons
	if l.sysMode == health.SysModeDegrade {
		return l.emitSysModeChange(runID, cycleID, observability.DEGRADE, reasoncodes.ENTER_DEGRADE, result.Reasons, signals)
	}
	if l.sysMode == health.SysModePause {
		return l.emitSysModeChange(runID, cycleID, observability.PAUSE, reasoncodes.ENTER_PAUSE, result.Reasons, signals)
	}
	if l.sysMode == health.SysModeExit {
		if err := l.emitSysModeChange(runID, cycleID, observability.SHUTDOWN, reasoncodes.ENTER_EXIT, result.Reasons, signals); err != nil {
			return err
		}
		return fmt.Errorf("exit requested")
	}
	return nil
}

func (l *Loop) emitSysModeChange(runID string, cycleID string, stage observability.StageName, enterReason reasoncodes.ReasonCode, reasons []reasoncodes.ReasonCode, signals health.Signals) error {
	alertReasons := append([]reasoncodes.ReasonCode{reasoncodes.ALERT_RAISED, enterReason}, reasons...)
	alertData := map[string]any{
		"last_progress_ts_ms":     signals.LastProgressMs,
		"ws_last_msg_ts_ms":       signals.WsLastMsgMs,
		"rest_last_success_ts_ms": signals.RestLastSuccessMs,
		"disk_free_bytes":         signals.DiskFreeBytes,
		"audit_queue_pct":         signals.AuditQueuePct,
		"audit_writer_lag_ms":     signals.AuditWriterLagMs,
	}
	if err := l.emitAlert(runID, cycleID, stage, alertReasons, alertData); err != nil {
		return err
	}
	stageReasons := append([]reasoncodes.ReasonCode{enterReason}, reasons...)
	return l.emitStageWithReasons(runID, cycleID, stage, "", "sys_mode change", stageReasons)
}

func (l *Loop) emitAlert(runID string, cycleID string, stage observability.StageName, reasons []reasoncodes.ReasonCode, data map[string]any) error {
	now := l.now()
	if l.reporter != nil {
		l.reporter.StageChanged(now, stage, cycleID, "", "", "ALERT")
	}
	record := audit.Record{
		Event: auditdomain.AuditEvent{
			TsMs:            now.UnixMilli(),
			RunID:           runID,
			CycleID:         cycleID,
			Mode:            l.cfg.Mode,
			Stage:           stage,
			EventType:       auditdomain.ALERT_RAISED,
			Reasons:         reasons,
			SnapshotID:      "",
			DecisionID:      "",
			OrderIntentID:   "",
			ExchangeTimeMs:  0,
			LocalReceivedMs: now.UnixMilli(),
		},
		Data: data,
	}
	if err := l.writer.Write(record); err != nil {
		return fmt.Errorf("audit alert write: %w", err)
	}
	fmt.Printf("ALERT stage=%s reasons=%v\n", stage, reasons)
	return nil
}

func (l *Loop) emitStageWithReasons(runID string, cycleID string, stage observability.StageName, symbol string, summary string, reasons []reasoncodes.ReasonCode) error {
	now := l.now()
	l.lastProgressMs = now.UnixMilli()
	if l.reporter != nil {
		l.reporter.StageChanged(now, stage, cycleID, "", symbol, summary)
	}
	record := audit.Record{
		Event: auditdomain.AuditEvent{
			TsMs:            now.UnixMilli(),
			RunID:           runID,
			CycleID:         cycleID,
			Mode:            l.cfg.Mode,
			Stage:           stage,
			EventType:       auditdomain.STAGE_CHANGED,
			Reasons:         reasons,
			SnapshotID:      "",
			DecisionID:      "",
			OrderIntentID:   "",
			ExchangeTimeMs:  0,
			LocalReceivedMs: now.UnixMilli(),
		},
		Data: map[string]any{
			"symbol":  symbol,
			"summary": summary,
		},
	}
	if err := l.writer.Write(record); err != nil {
		return fmt.Errorf("audit stage write: %w", err)
	}
	return nil
}

func (l *Loop) sampleDiskFree() error {
	freeBytes, err := health.FreeBytes(audit.DefaultSQLitePath)
	if err != nil {
		l.diskFreeBytes = 0
		return nil
	}
	l.diskFreeBytes = freeBytes
	return nil
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
