package executor

import (
	"crypto/sha256"
	"encoding/base32"
	"fmt"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/hash"
)

type OrderIntentHashPayload struct {
	Mode           string                        `json:"mode"`
	Symbol         string                        `json:"symbol"`
	Side           contracts.Side                `json:"side"`
	Intent         contracts.Intent              `json:"intent"`
	EntryPlanQty   string                        `json:"entry_plan_qty"`
	EntryPlanPrice string                        `json:"entry_plan_price"`
	ExitPlanTP     string                        `json:"exit_plan_tp"`
	ExitPlanSL     string                        `json:"exit_plan_sl"`
	SnapshotHash   string                        `json:"snapshot_hash"`
	Constraints    contracts.DecisionConstraints `json:"constraints"`
}

func OrderIntentID(decision contracts.Decision, snapshotHash string) (string, error) {
	if decision.EntryPlan == nil || decision.ExitPlan == nil {
		return "", fmt.Errorf("order intent requires entry and exit plans")
	}
	payload := OrderIntentHashPayload{
		Mode:           decision.Mode,
		Symbol:         decision.Symbol,
		Side:           decision.Side,
		Intent:         decision.Intent,
		EntryPlanQty:   decision.EntryPlan.Qty,
		EntryPlanPrice: decision.EntryPlan.LimitPrice,
		ExitPlanTP:     decision.ExitPlan.TPPrice,
		ExitPlanSL:     decision.ExitPlan.SLPrice,
		SnapshotHash:   snapshotHash,
		Constraints:    decision.Constraints,
	}
	sum, err := hash.CanonicalHash(payload)
	if err != nil {
		return "", err
	}
	return "oi_" + sum, nil
}

func ClientOrderID(orderIntentID string) string {
	sum := sha256.Sum256([]byte(orderIntentID))
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:])
	if len(enc) > 34 {
		enc = enc[:34]
	}
	return "X_" + enc
}
