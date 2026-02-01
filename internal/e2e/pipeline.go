package e2e

import (
	"fmt"

	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	auditdomain "github.com/RodrigoBeloyanis/livespot/internal/domain/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"
)

type Pipeline struct {
	cfg      config.Config
	writer   *audit.Writer
	reporter observability.StageReporter
}

func NewPipeline(cfg config.Config, writer *audit.Writer, reporter observability.StageReporter) (*Pipeline, error) {
	if writer == nil {
		return nil, fmt.Errorf("audit writer missing")
	}
	return &Pipeline{cfg: cfg, writer: writer, reporter: reporter}, nil
}

func (p *Pipeline) RunOnce(ctx Context, input PipelineInput) (PipelineResult, error) {
	if err := p.validateInput(input); err != nil {
		return PipelineResult{}, err
	}
	if err := input.Snapshot.Validate(); err != nil {
		return PipelineResult{}, err
	}
	if err := input.Decision.Validate(); err != nil {
		return PipelineResult{}, err
	}
	if input.DecisionID != input.Decision.DecisionID {
		return PipelineResult{}, fmt.Errorf("decision_id mismatch")
	}
	if input.Decision.SnapshotID != input.Snapshot.Metadata.SnapshotID {
		return PipelineResult{}, fmt.Errorf("decision snapshot_id mismatch")
	}
	if input.Decision.CycleID != input.CycleID {
		return PipelineResult{}, fmt.Errorf("decision cycle_id mismatch")
	}
	snapshotHash, err := p.snapshotHash(input)
	if err != nil {
		return PipelineResult{}, err
	}
	decisionHash, err := input.Decision.Hash(snapshotHash)
	if err != nil {
		return PipelineResult{}, err
	}
	if decisionHash != input.Decision.DecisionID {
		return PipelineResult{}, fmt.Errorf("decision hash mismatch")
	}
	exchangeTimeMs, err := input.Exchange.Time(ctx)
	if err != nil {
		return PipelineResult{}, err
	}
	if _, err := input.Exchange.ExchangeInfo(ctx); err != nil {
		return PipelineResult{}, err
	}
	if _, err := input.Exchange.Depth(ctx, input.Decision.Symbol, 5); err != nil {
		return PipelineResult{}, err
	}
	if _, err := input.Exchange.Account(ctx); err != nil {
		return PipelineResult{}, err
	}
	now := input.Now().UnixMilli()
	stages := []observability.StageName{
		observability.BOOT,
		observability.STRATEGY_PROPOSE,
		observability.RISK_VERDICT,
		observability.EXECUTE_INTENT,
		observability.SHUTDOWN,
	}
	stageStrings := make([]string, 0, len(stages))
	for _, stage := range stages {
		if err := p.emitStage(input, stage, exchangeTimeMs, "e2e"); err != nil {
			return PipelineResult{}, err
		}
		stageStrings = append(stageStrings, string(stage))
	}
	end := input.Now().UnixMilli()
	return PipelineResult{
		RunID:          input.RunID,
		CycleID:        input.CycleID,
		StartTsMs:      now,
		EndTsMs:        end,
		Stages:         stageStrings,
		ExchangeTimeMs: exchangeTimeMs,
	}, nil
}

func (p *Pipeline) RunSoakTick(ctx Context, input PipelineInput) (PipelineResult, error) {
	if !input.DisableEntries {
		return PipelineResult{}, fmt.Errorf("soak requires entries disabled")
	}
	if err := p.validateInput(input); err != nil {
		return PipelineResult{}, err
	}
	if err := input.Snapshot.Validate(); err != nil {
		return PipelineResult{}, err
	}
	if err := input.Decision.Validate(); err != nil {
		return PipelineResult{}, err
	}
	snapshotHash, err := p.snapshotHash(input)
	if err != nil {
		return PipelineResult{}, err
	}
	if _, err := input.Decision.Hash(snapshotHash); err != nil {
		return PipelineResult{}, err
	}
	exchangeTimeMs, err := input.Exchange.Time(ctx)
	if err != nil {
		return PipelineResult{}, err
	}
	now := input.Now().UnixMilli()
	stages := []observability.StageName{
		observability.STATE_UPDATE,
		observability.RECONCILE_REST,
	}
	stageStrings := make([]string, 0, len(stages))
	for _, stage := range stages {
		if err := p.emitStage(input, stage, exchangeTimeMs, "soak"); err != nil {
			return PipelineResult{}, err
		}
		stageStrings = append(stageStrings, string(stage))
	}
	end := input.Now().UnixMilli()
	return PipelineResult{
		RunID:          input.RunID,
		CycleID:        input.CycleID,
		StartTsMs:      now,
		EndTsMs:        end,
		Stages:         stageStrings,
		ExchangeTimeMs: exchangeTimeMs,
	}, nil
}

func (p *Pipeline) snapshotHash(input PipelineInput) (string, error) {
	snapshotHash := input.SnapshotHash
	if snapshotHash == "" {
		var err error
		snapshotHash, err = input.Snapshot.Hash()
		if err != nil {
			return "", err
		}
	}
	if input.Snapshot.Metadata.SnapshotHash != "" && input.Snapshot.Metadata.SnapshotHash != snapshotHash {
		return "", fmt.Errorf("snapshot hash mismatch")
	}
	return snapshotHash, nil
}

func (p *Pipeline) validateInput(input PipelineInput) error {
	if input.RunID == "" {
		return fmt.Errorf("run_id missing")
	}
	if input.CycleID == "" {
		return fmt.Errorf("cycle_id missing")
	}
	if input.Mode != p.cfg.Mode {
		return fmt.Errorf("mode mismatch")
	}
	if input.Exchange == nil {
		return fmt.Errorf("exchange client missing")
	}
	if input.Now == nil {
		return fmt.Errorf("clock missing")
	}
	return nil
}

func (p *Pipeline) emitStage(input PipelineInput, stage observability.StageName, exchangeTimeMs int64, summary string) error {
	now := input.Now()
	if p.reporter != nil {
		p.reporter.StageChanged(now, stage, input.CycleID, input.Decision.DecisionID, input.Decision.Symbol, summary)
	}
	record := audit.Record{
		Event: auditdomain.AuditEvent{
			TsMs:            now.UnixMilli(),
			RunID:           input.RunID,
			CycleID:         input.CycleID,
			Mode:            input.Mode,
			Stage:           stage,
			EventType:       auditdomain.STAGE_CHANGED,
			Reasons:         []reasoncodes.ReasonCode{},
			SnapshotID:      input.Snapshot.Metadata.SnapshotID,
			DecisionID:      input.Decision.DecisionID,
			OrderIntentID:   "",
			ExchangeTimeMs:  exchangeTimeMs,
			LocalReceivedMs: now.UnixMilli(),
		},
		Data: map[string]any{
			"symbol":  input.Decision.Symbol,
			"summary": summary,
		},
	}
	if err := p.writer.Write(record); err != nil {
		return fmt.Errorf("audit stage write: %w", err)
	}
	return nil
}
