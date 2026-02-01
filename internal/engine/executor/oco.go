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

type OCOClient interface {
	SubmitOCO(ctx context.Context, req OCORequest) (OCOResponse, error)
}

func BuildOCORequest(decision contracts.Decision) (OCORequest, error) {
	if decision.EntryPlan == nil || decision.ExitPlan == nil {
		return OCORequest{}, fmt.Errorf("plans missing")
	}
	if decision.ExitPlan.TPPrice == "" || decision.ExitPlan.SLPrice == "" {
		return OCORequest{}, fmt.Errorf("tp/sl missing")
	}
	side := contracts.SideSell
	req := OCORequest{
		Symbol:             decision.Symbol,
		Side:               side,
		Qty:                decision.EntryPlan.Qty,
		TPPrice:            decision.ExitPlan.TPPrice,
		SLStopPrice:        decision.ExitPlan.SLPrice,
		SLStopLimitPrice:   decision.ExitPlan.SLPrice,
		SLStopLimitTIF:     contracts.TIFGTC,
		ListClientOrderID:  "",
		LimitClientOrderID: decision.ExitPlan.ClientOrderIDTP,
		StopClientOrderID:  decision.ExitPlan.ClientOrderIDSL,
	}
	return req, nil
}

func BuildOCOIntent(decision contracts.Decision, intentID string, runID string, cycleID string, req OCORequest, now time.Time) (sqlite.OrderIntentRecord, error) {
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
		Action:            string(IntentActionOCOCreate),
		ClientOrderID:     req.ListClientOrderID,
		IntentPayloadJSON: string(payload),
		CreatedAtMs:       now.UnixMilli(),
		UpdatedAtMs:       now.UnixMilli(),
	}, nil
}

func ExecuteOCO(ctx context.Context, ledger *LedgerService, rest OCOClient, decision contracts.Decision, snapshotHash string, runID string, cycleID string, now time.Time) (OCOResponse, string, reasoncodes.ReasonCode, error) {
	intentID, err := OrderIntentID(decision, snapshotHash)
	if err != nil {
		return OCOResponse{}, "", reasoncodes.PROTECTION_INSTALL_FAILED, err
	}
	intentID = intentID + "_OCO"
	req, err := BuildOCORequest(decision)
	if err != nil {
		return OCOResponse{}, intentID, reasoncodes.PROTECTION_INSTALL_FAILED, err
	}
	filters := FiltersFromConstraints(decision.Constraints)
	tpReq := OrderRequest{
		Symbol:      req.Symbol,
		Side:        contracts.SideSell,
		Type:        OrderTypeLimit,
		TimeInForce: contracts.TIFGTC,
		Price:       req.TPPrice,
		Qty:         req.Qty,
	}
	tpQuant, reason, err := QuantizeOrder(tpReq, filters)
	if err != nil {
		return OCOResponse{}, intentID, reason, err
	}
	slReq := OrderRequest{
		Symbol:      req.Symbol,
		Side:        contracts.SideSell,
		Type:        OrderTypeLimit,
		TimeInForce: contracts.TIFGTC,
		Price:       req.SLStopLimitPrice,
		Qty:         req.Qty,
	}
	slQuant, reason, err := QuantizeOrder(slReq, filters)
	if err != nil {
		return OCOResponse{}, intentID, reason, err
	}
	stopPrice, err := QuantizePrice(req.SLStopPrice, decision.Constraints.TickSize)
	if err != nil {
		return OCOResponse{}, intentID, reasoncodes.PROTECTION_INVALID_FILTER, err
	}
	req.Qty = tpQuant.Qty
	req.TPPrice = tpQuant.Price
	req.SLStopLimitPrice = slQuant.Price
	req.SLStopPrice = stopPrice
	req.ListClientOrderID = intentID
	intent, err := BuildOCOIntent(decision, intentID, runID, cycleID, req, now)
	if err != nil {
		return OCOResponse{}, intentID, reasoncodes.PROTECTION_INSTALL_FAILED, err
	}
	if err := ledger.CreateIntent(ctx, intent); err != nil {
		return OCOResponse{}, intentID, reasoncodes.PROTECTION_INSTALL_FAILED, err
	}
	resp, err := rest.SubmitOCO(ctx, req)
	if err != nil {
		_ = ledger.MarkFailed(ctx, intentID, "REJECTED", "oco submit failed")
		return OCOResponse{}, intentID, reasoncodes.PROTECTION_INSTALL_FAILED, err
	}
	if resp.Rejected {
		_ = ledger.MarkFailed(ctx, intentID, "REJECTED", "oco submit rejected")
		return resp, intentID, reasoncodes.PROTECTION_INSTALL_FAILED, nil
	}
	if err := ledger.MarkConfirmed(ctx, intentID, "", resp.OrderListID); err != nil {
		return resp, intentID, reasoncodes.PROTECTION_INSTALL_FAILED, err
	}
	return resp, intentID, "", nil
}
