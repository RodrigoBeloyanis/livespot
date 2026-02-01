package aigate

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	auditdomain "github.com/RodrigoBeloyanis/livespot/internal/domain/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/hash"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"
)

type Event struct {
	RunID                 string
	CycleID               string
	Mode                  string
	SnapshotID            string
	SnapshotHash          string
	DecisionID            string
	InputHash             string
	Enabled               bool
	Verdict               string
	Reasons               []reasoncodes.ReasonCode
	Model                 string
	LatencyMs             int
	RawHash               string
	RequestJSON           []byte
	ResponseJSON          []byte
	ModifiedDecisionPatch map[string]any
	ModifyApplied         bool
	ErrorCode             string
	ErrorDetailRedacted   string
	ExchangeTimeMs        int64
	LocalReceivedMs       int64
}

type Recorder struct {
	cfg    config.Config
	db     *sql.DB
	writer *audit.Writer
	now    func() time.Time
}

func NewRecorder(cfg config.Config, db *sql.DB, writer *audit.Writer, now func() time.Time) *Recorder {
	if now == nil {
		now = time.Now
	}
	return &Recorder{cfg: cfg, db: db, writer: writer, now: now}
}

func (r *Recorder) Record(ctx context.Context, evt Event) error {
	if err := r.insertEvent(ctx, evt); err != nil {
		return err
	}
	if r.writer != nil {
		record := r.auditRecord(evt)
		if err := r.writer.Write(record); err != nil {
			return fmt.Errorf("audit writer: %w", err)
		}
	}
	return nil
}

func (r *Recorder) insertEvent(ctx context.Context, evt Event) error {
	if r.db == nil {
		return fmt.Errorf("ai gate db missing")
	}
	reasonsJSON, err := json.Marshal(reasonStrings(evt.Reasons))
	if err != nil {
		return fmt.Errorf("ai gate reasons json: %w", err)
	}
	requestRedacted, err := redactJSON(evt.RequestJSON, r.cfg.AuditRedactedJSONMaxBytes)
	if err != nil {
		requestRedacted = ""
	}
	responseRedacted, err := redactJSON(evt.ResponseJSON, r.cfg.AuditRedactedJSONMaxBytes)
	if err != nil {
		responseRedacted = ""
	}
	patchRedacted := ""
	if evt.ModifiedDecisionPatch != nil {
		buf, err := hash.CanonicalJSON(evt.ModifiedDecisionPatch)
		if err == nil {
			patchRedacted, _ = redactJSON(buf, r.cfg.AuditRedactedJSONMaxBytes)
		}
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO ai_gate_events (
  run_id, cycle_id, mode, stage, event_type, snapshot_id, snapshot_hash, decision_id, input_hash,
  enabled, verdict, reasons_json, model, latency_ms, raw_hash, request_json_redacted, response_json_redacted,
  modified_decision_json_redacted, modify_applied, error_code, error_detail_redacted, exchange_time_ms, local_received_ms, created_at_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		evt.RunID,
		evt.CycleID,
		evt.Mode,
		observability.AIGATE_CALL,
		auditdomain.AIGATE_CALL,
		evt.SnapshotID,
		evt.SnapshotHash,
		evt.DecisionID,
		evt.InputHash,
		boolToInt(evt.Enabled),
		evt.Verdict,
		string(reasonsJSON),
		nullIfEmpty(evt.Model),
		nullIfZero(evt.LatencyMs),
		nullIfEmpty(evt.RawHash),
		nullIfEmpty(requestRedacted),
		nullIfEmpty(responseRedacted),
		nullIfEmpty(patchRedacted),
		boolToInt(evt.ModifyApplied),
		nullIfEmpty(evt.ErrorCode),
		nullIfEmpty(evt.ErrorDetailRedacted),
		nullIfZero64(evt.ExchangeTimeMs),
		evt.LocalReceivedMs,
		evt.LocalReceivedMs,
	)
	if err != nil {
		return fmt.Errorf("ai gate insert: %w", err)
	}
	return nil
}

func (r *Recorder) auditRecord(evt Event) audit.Record {
	now := r.now()
	return audit.Record{
		Event: auditdomain.AuditEvent{
			TsMs:            now.UnixMilli(),
			RunID:           evt.RunID,
			CycleID:         evt.CycleID,
			Mode:            evt.Mode,
			Stage:           observability.AIGATE_CALL,
			EventType:       auditdomain.AIGATE_CALL,
			Reasons:         evt.Reasons,
			SnapshotID:      evt.SnapshotID,
			DecisionID:      evt.DecisionID,
			OrderIntentID:   "",
			ExchangeTimeMs:  evt.ExchangeTimeMs,
			LocalReceivedMs: evt.LocalReceivedMs,
		},
		Data: map[string]any{
			"ai_enabled":               evt.Enabled,
			"ai_verdict":               evt.Verdict,
			"ai_reasons":               reasonStrings(evt.Reasons),
			"ai_model":                 evt.Model,
			"ai_latency_ms":            evt.LatencyMs,
			"ai_raw_hash":              evt.RawHash,
			"ai_input_hash":            evt.InputHash,
			"ai_snapshot_hash":         evt.SnapshotHash,
			"ai_modify_applied":        evt.ModifyApplied,
			"ai_error_code":            evt.ErrorCode,
			"ai_error_detail_redacted": evt.ErrorDetailRedacted,
		},
	}
}

func redactJSON(raw []byte, maxBytes int) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}
	return audit.RedactAndTruncateJSON(raw, maxBytes, audit.DefaultRedactionPolicy())
}

func reasonStrings(reasons []reasoncodes.ReasonCode) []string {
	if len(reasons) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		out = append(out, string(reason))
	}
	return out
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func nullIfEmpty(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func nullIfZero(value int) interface{} {
	if value == 0 {
		return nil
	}
	return value
}

func nullIfZero64(value int64) interface{} {
	if value == 0 {
		return nil
	}
	return value
}
