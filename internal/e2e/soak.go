package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
)

type SoakRunner struct {
	Pipeline *Pipeline
	Config   config.Config
	Input    PipelineInput
	Duration time.Duration
	Interval time.Duration
}

func (r *SoakRunner) Run(ctx context.Context) (SoakResult, error) {
	if r.Pipeline == nil {
		return SoakResult{}, fmt.Errorf("pipeline missing")
	}
	if r.Input.Now == nil {
		return SoakResult{}, fmt.Errorf("clock missing")
	}
	if r.Duration <= 0 {
		return SoakResult{}, fmt.Errorf("duration invalid")
	}
	if r.Interval <= 0 {
		return SoakResult{}, fmt.Errorf("interval invalid")
	}
	start := r.Input.Now().UnixMilli()
	deadline := time.After(r.Duration)
	ticker := time.NewTicker(r.Interval)
	defer ticker.Stop()

	monitor := soakMonitor{
		cfg:        r.Config,
		violations: map[reasoncodes.ReasonCode]SoakViolation{},
	}
	ticks := 0
	for {
		select {
		case <-ctx.Done():
			return SoakResult{}, ctx.Err()
		case <-deadline:
			end := r.Input.Now().UnixMilli()
			violations := monitor.List()
			return SoakResult{
				StartTsMs:  start,
				EndTsMs:    end,
				TickCount:  ticks,
				Violations: violations,
				Pass:       len(violations) == 0,
			}, nil
		case <-ticker.C:
			if _, err := r.Pipeline.RunSoakTick(ctx, r.Input); err != nil {
				return SoakResult{}, err
			}
			now := r.Input.Now().UnixMilli()
			monitor.Observe(now, r.Input.Signals)
			ticks++
		}
	}
}

func BuildReadinessReport(runID string, soak SoakResult, configOK bool, auditOK bool) ReadinessReport {
	checks := make([]ReadinessCheck, 0, 3)
	checks = append(checks, ReadinessCheck{
		Name:   "config_validated",
		Pass:   configOK,
		Detail: "",
	})
	checks = append(checks, ReadinessCheck{
		Name:   "audit_writer_ok",
		Pass:   auditOK,
		Detail: "",
	})
	soakDetail := ""
	if !soak.Pass {
		buf, _ := json.Marshal(soak.Violations)
		soakDetail = string(buf)
	}
	checks = append(checks, ReadinessCheck{
		Name:   "soak_pass",
		Pass:   soak.Pass,
		Detail: soakDetail,
	})
	ready := true
	for _, check := range checks {
		if !check.Pass {
			ready = false
			break
		}
	}
	return ReadinessReport{
		RunID:      runID,
		StartTsMs:  soak.StartTsMs,
		EndTsMs:    soak.EndTsMs,
		DurationMs: soak.EndTsMs - soak.StartTsMs,
		Checks:     checks,
		Ready:      ready,
	}
}

type soakMonitor struct {
	cfg            config.Config
	violations     map[reasoncodes.ReasonCode]SoakViolation
	lastProgressMs int64
}

func (m *soakMonitor) Observe(nowMs int64, signals SoakSignals) {
	lastProgress := signals.LastProgressTsMs
	if lastProgress == 0 {
		lastProgress = m.lastProgressMs
	}
	if lastProgress == 0 {
		lastProgress = nowMs
	}
	if nowMs-lastProgress >= int64(m.cfg.LoopStuckMsDegrade) {
		m.add(reasoncodes.LOOP_STUCK_DEGRADE, nowMs, "loop stuck")
	}
	m.lastProgressMs = nowMs

	wsLast := signals.WsLastMsgTsMs
	if wsLast == 0 {
		wsLast = nowMs
	}
	if nowMs-wsLast >= int64(m.cfg.WsStaleMsDegrade) {
		m.add(reasoncodes.WS_STALE_DEGRADE, nowMs, "ws stale")
	}
	restLast := signals.RestLastOkTsMs
	if restLast == 0 {
		restLast = nowMs
	}
	if nowMs-restLast >= int64(m.cfg.RestStaleMsDegrade) {
		m.add(reasoncodes.REST_STALE_DEGRADE, nowMs, "rest stale")
	}
	if signals.DBWriterQueuePct >= m.cfg.AuditWriterQueueFull {
		m.add(reasoncodes.DB_WRITER_QUEUE_FULL, nowMs, "db writer queue full")
	}
	if signals.ReconcileDriftX10000 >= m.cfg.ReconcileDriftDegradeX10000 {
		m.add(reasoncodes.DRIFT_LIMIT_EXCEEDED, nowMs, "reconcile drift")
	}
}

func (m *soakMonitor) add(reason reasoncodes.ReasonCode, atMs int64, detail string) {
	if _, exists := m.violations[reason]; exists {
		return
	}
	m.violations[reason] = SoakViolation{
		Reason: reason,
		AtTsMs: atMs,
		Detail: detail,
	}
}

func (m *soakMonitor) List() []SoakViolation {
	out := make([]SoakViolation, 0, len(m.violations))
	for _, violation := range m.violations {
		out = append(out, violation)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Reason < out[j].Reason
	})
	return out
}
