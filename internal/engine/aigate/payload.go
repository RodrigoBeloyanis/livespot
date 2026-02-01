package aigate

import (
	"fmt"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/hash"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
)

type Payload struct {
	DecisionID     string                       `json:"decision_id"`
	SnapshotID     string                       `json:"snapshot_id"`
	SnapshotHash   string                       `json:"snapshot_hash"`
	InputHash      string                       `json:"input_hash"`
	Mode           string                       `json:"mode"`
	Symbol         string                       `json:"symbol"`
	Side           contracts.Side               `json:"side"`
	Intent         contracts.Intent             `json:"intent"`
	EdgeScore      int                          `json:"edge_score_x10000"`
	EdgeBps        int                          `json:"edge_bps_expected"`
	Reasons        []reasoncodes.ReasonCode     `json:"reasons"`
	Regime         contracts.RegimeSnapshot     `json:"regime"`
	Microstructure contracts.Microstructure60s  `json:"microstructure_60s"`
	Volatility     contracts.VolatilitySnapshot `json:"volatility"`
	Costs          contracts.CostInputs         `json:"cost_inputs"`
	ExitPlan       contracts.ExitPlan           `json:"exit_plan"`
}

func BuildPayload(decision contracts.Decision, snapshot contracts.Snapshot, snapshotHash string) (Payload, error) {
	if snapshotHash == "" {
		return Payload{}, fmt.Errorf("snapshot hash missing")
	}
	if decision.ExitPlan == nil {
		return Payload{}, fmt.Errorf("exit plan missing")
	}
	return Payload{
		DecisionID:     decision.DecisionID,
		SnapshotID:     decision.SnapshotID,
		SnapshotHash:   snapshotHash,
		InputHash:      "",
		Mode:           decision.Mode,
		Symbol:         decision.Symbol,
		Side:           decision.Side,
		Intent:         decision.Intent,
		EdgeScore:      decision.EdgeScoreX10000,
		EdgeBps:        decision.EdgeBpsExpected,
		Reasons:        decision.Reasons,
		Regime:         snapshot.Regime,
		Microstructure: snapshot.Microstructure60s,
		Volatility:     snapshot.Volatility,
		Costs:          snapshot.CostInputs,
		ExitPlan:       *decision.ExitPlan,
	}, nil
}

func InputHash(payload Payload) (string, error) {
	clone := payload
	clone.InputHash = ""
	return hash.CanonicalHash(clone)
}
