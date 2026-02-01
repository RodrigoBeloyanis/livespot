package audit

import (
	"encoding/json"
	"fmt"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
)

type Record struct {
	Event audit.AuditEvent
	Data  map[string]any
}

var reservedKeys = map[string]struct{}{
	"ts_ms":             {},
	"run_id":            {},
	"cycle_id":          {},
	"mode":              {},
	"stage":             {},
	"event_type":        {},
	"reasons":           {},
	"snapshot_id":       {},
	"decision_id":       {},
	"order_intent_id":   {},
	"exchange_time_ms":  {},
	"local_received_ms": {},
}

func (r Record) Validate() error {
	if err := r.Event.Validate(); err != nil {
		return err
	}
	for key := range r.Data {
		if _, exists := reservedKeys[key]; exists {
			return fmt.Errorf("audit record data key conflicts with envelope: %s", key)
		}
	}
	return nil
}

func (r Record) JSONLine() ([]byte, error) {
	line := map[string]any{
		"ts_ms":             r.Event.TsMs,
		"run_id":            r.Event.RunID,
		"cycle_id":          r.Event.CycleID,
		"mode":              r.Event.Mode,
		"stage":             r.Event.Stage,
		"event_type":        r.Event.EventType,
		"reasons":           reasonStrings(r.Event.Reasons),
		"snapshot_id":       r.Event.SnapshotID,
		"decision_id":       r.Event.DecisionID,
		"order_intent_id":   r.Event.OrderIntentID,
		"exchange_time_ms":  r.Event.ExchangeTimeMs,
		"local_received_ms": r.Event.LocalReceivedMs,
	}
	for key, value := range r.Data {
		line[key] = value
	}
	return json.Marshal(line)
}

func (r Record) DataJSON() (string, error) {
	data := r.Data
	if data == nil {
		data = map[string]any{}
	}
	buf, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(buf), nil
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
