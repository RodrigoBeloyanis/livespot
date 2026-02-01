package app

import (
	"fmt"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	auditdomain "github.com/RodrigoBeloyanis/livespot/internal/domain/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"
)

type Loop struct {
	cfg      config.Config
	writer   *audit.Writer
	reporter observability.StageReporter
	now      func() time.Time
}

func NewLoop(cfg config.Config, writer *audit.Writer, reporter observability.StageReporter, now func() time.Time) (*Loop, error) {
	if writer == nil {
		return nil, fmt.Errorf("audit writer missing")
	}
	if now == nil {
		now = time.Now
	}
	return &Loop{cfg: cfg, writer: writer, reporter: reporter, now: now}, nil
}

func (l *Loop) RunDryRun() error {
	sequence := DefaultStageSequence()
	if err := sequence.Validate(); err != nil {
		return err
	}
	runID, err := observability.NewRunID(l.now())
	if err != nil {
		return err
	}
	cycleID, err := observability.NewCycleID(l.now())
	if err != nil {
		return err
	}
	for _, stage := range sequence.Stages {
		if err := l.emitStage(runID, cycleID, stage, "", "dry-run"); err != nil {
			return err
		}
	}
	return nil
}

func (l *Loop) emitStage(runID string, cycleID string, stage observability.StageName, symbol string, summary string) error {
	now := l.now()
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
