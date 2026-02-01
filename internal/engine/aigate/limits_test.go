package aigate

import (
	"testing"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"
)

func TestApplyModifyReducesQty(t *testing.T) {
	orig := baseDecision()
	mod := baseDecision()
	mod.EntryPlan.Qty = "0.005"
	updated, err := ApplyModify(orig, mod)
	if err != nil {
		t.Fatalf("expected modify to apply: %v", err)
	}
	if updated.EntryPlan.Qty != "0.005" {
		t.Fatalf("expected qty updated")
	}
}

func TestApplyModifyRejectsIncreaseQty(t *testing.T) {
	orig := baseDecision()
	mod := baseDecision()
	mod.EntryPlan.Qty = "0.050"
	if _, err := ApplyModify(orig, mod); err == nil {
		t.Fatalf("expected error for qty increase")
	}
}

func TestApplyModifyRejectsSymbolChange(t *testing.T) {
	orig := baseDecision()
	mod := baseDecision()
	mod.Symbol = "ETHUSDT"
	if _, err := ApplyModify(orig, mod); err == nil {
		t.Fatalf("expected error for symbol change")
	}
}

func baseDecision() contracts.Decision {
	return contracts.Decision{
		Mode:   "LIVE",
		TsMs:   1706700000000,
		Symbol: "BTCUSDT",
		Side:   contracts.SideBuy,
		Intent: contracts.IntentEntry,
		EntryPlan: &contracts.EntryPlan{
			Kind:         contracts.EntryMakerFirst,
			DesiredPrice: "100.00",
			LimitPrice:   "100.00",
			Qty:          "0.010",
			TimeInForce:  contracts.TIFGTC,
			TTLMS:        30000,
			RepriceMS:    30000,
			MaxReprices:  2,
			Fallback: contracts.FallbackPlan{
				Enabled:        true,
				Kind:           contracts.FallbackIOCLimit,
				MaxSlippageBps: 25,
				DeadlineMS:     90000,
			},
			ClientOrderID: "X_0000000000000000000000000000000000",
		},
		ExitPlan: &contracts.ExitPlan{
			TPPrice:              "110.00",
			SLPrice:              "95.00",
			ProtectionKind:       contracts.ProtectionOCO,
			TrailingMode:         contracts.TrailingOff,
			TrailingTriggerPrice: "0",
			TrailingDeltaBips:    0,
			ClientOrderIDTP:      "X_1111111111111111111111111111111111",
			ClientOrderIDSL:      "X_2222222222222222222222222222222222",
		},
		EdgeScoreX10000: 4500,
		EdgeBpsExpected: 25,
		Reasons:         []reasoncodes.ReasonCode{reasoncodes.STRAT_OK},
		SnapshotID:      "snap_btc_1",
		DecisionID:      "dec_btc_1",
		CycleID:         "cyc_1",
		Stage:           observability.STRATEGY_PROPOSE,
		Constraints: contracts.DecisionConstraints{
			TickSize:           "0.01",
			StepSize:           "0.0001",
			MinQty:             "0.001",
			MinNotional:        "10.00",
			PricePrecision:     2,
			QtyPrecision:       4,
			MaxQty:             "100.0000",
			MaxNumOrders:       200,
			MaxAlgoOrders:      5,
			MaxNotional:        "100000.00",
			QuantizationPolicy: contracts.QuantizationEnforced,
		},
	}
}
