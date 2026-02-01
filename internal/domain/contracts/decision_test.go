package contracts

import (
	"testing"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"
)

func TestDecisionValidateAndHashDeterministic(t *testing.T) {
	decision := Decision{
		Mode:   "LIVE",
		TsMs:   1706700000000,
		Symbol: "BTCUSDT",
		Side:   SideBuy,
		Intent: IntentEntry,
		EntryPlan: &EntryPlan{
			Kind:         EntryMakerFirst,
			DesiredPrice: "100.00",
			LimitPrice:   "100.00",
			Qty:          "0.01000000",
			TimeInForce:  TIFGTC,
			TTLMS:        30000,
			RepriceMS:    30000,
			MaxReprices:  2,
			Fallback: FallbackPlan{
				Enabled:        true,
				Kind:           FallbackIOCLimit,
				MaxSlippageBps: 25,
				DeadlineMS:     90000,
			},
			ClientOrderID: "X_0000000000000000000000000000000000",
		},
		ExitPlan: &ExitPlan{
			TPPrice:              "110.00",
			SLPrice:              "95.00",
			ProtectionKind:       ProtectionOCO,
			TrailingMode:         TrailingOff,
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
		Constraints: DecisionConstraints{
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
			QuantizationPolicy: QuantizationEnforced,
		},
	}
	if err := decision.Validate(); err != nil {
		t.Fatalf("decision validate failed: %v", err)
	}
	snapshotHash := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	h1, err := decision.Hash(snapshotHash)
	if err != nil {
		t.Fatalf("decision hash failed: %v", err)
	}
	h2, err := decision.Hash(snapshotHash)
	if err != nil {
		t.Fatalf("decision hash failed: %v", err)
	}
	if h1 != h2 {
		t.Fatalf("decision hash mismatch")
	}
}

func TestDecisionRejectsUnknownReason(t *testing.T) {
	decision := Decision{
		Mode:   "LIVE",
		TsMs:   1706700000000,
		Symbol: "BTCUSDT",
		Side:   SideBuy,
		Intent: IntentEntry,
		EntryPlan: &EntryPlan{
			Kind:         EntryMakerFirst,
			DesiredPrice: "100.00",
			LimitPrice:   "100.00",
			Qty:          "0.01000000",
			TimeInForce:  TIFGTC,
			TTLMS:        30000,
			RepriceMS:    30000,
			MaxReprices:  2,
			Fallback: FallbackPlan{
				Enabled:        true,
				Kind:           FallbackIOCLimit,
				MaxSlippageBps: 25,
				DeadlineMS:     90000,
			},
			ClientOrderID: "X_0000000000000000000000000000000000",
		},
		ExitPlan: &ExitPlan{
			TPPrice:              "110.00",
			SLPrice:              "95.00",
			ProtectionKind:       ProtectionOCO,
			TrailingMode:         TrailingOff,
			TrailingTriggerPrice: "0",
			TrailingDeltaBips:    0,
			ClientOrderIDTP:      "X_1111111111111111111111111111111111",
			ClientOrderIDSL:      "X_2222222222222222222222222222222222",
		},
		EdgeScoreX10000: 4500,
		EdgeBpsExpected: 25,
		Reasons:         []reasoncodes.ReasonCode{"UNKNOWN_CODE"},
		SnapshotID:      "snap_btc_1",
		DecisionID:      "dec_btc_1",
		CycleID:         "cyc_1",
		Stage:           observability.STRATEGY_PROPOSE,
		Constraints: DecisionConstraints{
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
			QuantizationPolicy: QuantizationEnforced,
		},
	}
	if err := decision.Validate(); err == nil {
		t.Fatalf("expected decision validation error")
	}
}
