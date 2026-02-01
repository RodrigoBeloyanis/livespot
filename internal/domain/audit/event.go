package audit

import (
	"fmt"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"
)

type AuditEvent struct {
	TsMs            int64                    `json:"ts_ms"`
	RunID           string                   `json:"run_id"`
	CycleID         string                   `json:"cycle_id"`
	Mode            string                   `json:"mode"`
	Stage           observability.StageName  `json:"stage"`
	EventType       AuditEventType           `json:"event_type"`
	Reasons         []reasoncodes.ReasonCode `json:"reasons"`
	SnapshotID      string                   `json:"snapshot_id"`
	DecisionID      string                   `json:"decision_id"`
	OrderIntentID   string                   `json:"order_intent_id"`
	ExchangeTimeMs  int64                    `json:"exchange_time_ms"`
	LocalReceivedMs int64                    `json:"local_received_ms"`
}

func (e AuditEvent) Validate() error {
	if e.TsMs <= 0 {
		return fmt.Errorf("audit event ts_ms missing")
	}
	if e.RunID == "" {
		return fmt.Errorf("audit event run_id missing")
	}
	if e.CycleID == "" {
		return fmt.Errorf("audit event cycle_id missing")
	}
	if e.Mode != "LIVE" {
		return fmt.Errorf("audit event mode must be LIVE")
	}
	if !observability.IsValidStage(e.Stage) {
		return fmt.Errorf("audit event stage invalid")
	}
	if !IsValidEventType(e.EventType) {
		return fmt.Errorf("audit event type invalid")
	}
	if e.Reasons == nil {
		return fmt.Errorf("audit event reasons missing")
	}
	if !reasoncodes.ValidateList(e.Reasons) {
		return fmt.Errorf("audit event reasons invalid")
	}
	if e.LocalReceivedMs <= 0 {
		return fmt.Errorf("audit event local_received_ms missing")
	}
	return nil
}
