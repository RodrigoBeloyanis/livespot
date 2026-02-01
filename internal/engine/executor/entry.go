package executor

import (
	"context"
	"fmt"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/hash"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
	"github.com/RodrigoBeloyanis/livespot/internal/infra/sqlite"
)

func FiltersFromConstraints(c contracts.DecisionConstraints) SymbolFilters {
	return SymbolFilters{
		Price: &PriceFilter{
			TickSize: c.TickSize,
		},
		LotSize: &LotSizeFilter{
			MinQty:   c.MinQty,
			MaxQty:   c.MaxQty,
			StepSize: c.StepSize,
		},
		MinNotional: &MinNotionalFilter{
			MinNotional: c.MinNotional,
		},
		MarketLotSize: &MarketLotSizeFilter{
			MinQty:   c.MinQty,
			MaxQty:   c.MaxQty,
			StepSize: c.StepSize,
		},
	}
}

func BuildEntryOrder(decision contracts.Decision) (OrderRequest, error) {
	if decision.EntryPlan == nil {
		return OrderRequest{}, fmt.Errorf("entry plan missing")
	}
	req := OrderRequest{
		Symbol:        decision.Symbol,
		Side:          decision.Side,
		Type:          OrderTypeLimitMaker,
		TimeInForce:   decision.EntryPlan.TimeInForce,
		Price:         decision.EntryPlan.LimitPrice,
		Qty:           decision.EntryPlan.Qty,
		ClientOrderID: decision.EntryPlan.ClientOrderID,
	}
	return req, nil
}

func BuildEntryIntent(decision contracts.Decision, intentID string, runID string, cycleID string, req OrderRequest, now time.Time) (sqlite.OrderIntentRecord, error) {
	payload, err := hash.CanonicalJSON(req)
	if err != nil {
		return sqlite.OrderIntentRecord{}, err
	}
	return sqlite.OrderIntentRecord{
		OrderIntentID:     intentID,
		RunID:             runID,
		CycleID:           cycleID,
		Mode:              "LIVE",
		DecisionID:        decision.DecisionID,
		Symbol:            decision.Symbol,
		Action:            string(IntentActionNewOrder),
		ClientOrderID:     req.ClientOrderID,
		IntentPayloadJSON: string(payload),
		CreatedAtMs:       now.UnixMilli(),
		UpdatedAtMs:       now.UnixMilli(),
	}, nil
}

func ExecuteEntry(ctx context.Context, ledger *LedgerService, rest OrderRestClient, decision contracts.Decision, snapshotHash string, runID string, cycleID string, now time.Time) (OrderResponse, string, reasoncodes.ReasonCode, error) {
	intentID, err := OrderIntentID(decision, snapshotHash)
	if err != nil {
		return OrderResponse{}, "", reasoncodes.STRAT_ENTRY_ABORTED_COST, err
	}
	req, err := BuildEntryOrder(decision)
	if err != nil {
		return OrderResponse{}, intentID, reasoncodes.STRAT_ENTRY_ABORTED_COST, err
	}
	filters := FiltersFromConstraints(decision.Constraints)
	quantized, reason, err := QuantizeOrder(req, filters)
	if err != nil {
		return OrderResponse{}, intentID, reason, err
	}
	intent, err := BuildEntryIntent(decision, intentID, runID, cycleID, quantized, now)
	if err != nil {
		return OrderResponse{}, intentID, reasoncodes.STRAT_ENTRY_ABORTED_COST, err
	}
	resp, err := SubmitWithIntent(ctx, ledger, rest, intent, quantized)
	if err != nil {
		if err == ErrSentUnknown {
			return OrderResponse{}, intentID, reasoncodes.INTENT_SENT_UNKNOWN, err
		}
		return OrderResponse{}, intentID, reasoncodes.ORDER_SUBMIT_REJECTED, err
	}
	if resp.Rejected {
		return resp, intentID, reasoncodes.ORDER_SUBMIT_REJECTED, nil
	}
	return resp, intentID, "", nil
}
