package executor

import (
	"context"
	"errors"

	"github.com/RodrigoBeloyanis/livespot/internal/infra/sqlite"
)

var ErrTimeout = errors.New("rest timeout")

type OrderRestClient interface {
	SubmitOrder(ctx context.Context, req OrderRequest) (OrderResponse, error)
	CancelOrder(ctx context.Context, req CancelRequest) (OrderResponse, error)
	CancelReplaceOrder(ctx context.Context, req CancelReplaceRequest) (OrderResponse, error)
}

func SubmitWithIntent(ctx context.Context, ledger *LedgerService, rest OrderRestClient, intent sqlite.OrderIntentRecord, req OrderRequest) (OrderResponse, error) {
	if err := ledger.CreateIntent(ctx, intent); err != nil {
		return OrderResponse{}, err
	}
	resp, err := rest.SubmitOrder(ctx, req)
	if err != nil {
		if errors.Is(err, ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
			_ = ledger.MarkSentUnknown(ctx, intent.OrderIntentID, "TIMEOUT", "submit timeout")
			return OrderResponse{}, ErrSentUnknown
		}
		_ = ledger.MarkFailed(ctx, intent.OrderIntentID, "REJECTED", "submit failed")
		return OrderResponse{}, err
	}
	if resp.Rejected {
		_ = ledger.MarkFailed(ctx, intent.OrderIntentID, "REJECTED", "submit rejected")
		return resp, nil
	}
	if err := ledger.MarkConfirmed(ctx, intent.OrderIntentID, resp.OrderID, ""); err != nil {
		return resp, err
	}
	return resp, nil
}

func CancelWithIntent(ctx context.Context, ledger *LedgerService, rest OrderRestClient, intent sqlite.OrderIntentRecord, req CancelRequest) (OrderResponse, error) {
	if err := ledger.CreateIntent(ctx, intent); err != nil {
		return OrderResponse{}, err
	}
	resp, err := rest.CancelOrder(ctx, req)
	if err != nil {
		if errors.Is(err, ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
			_ = ledger.MarkSentUnknown(ctx, intent.OrderIntentID, "TIMEOUT", "cancel timeout")
			return OrderResponse{}, ErrSentUnknown
		}
		_ = ledger.MarkFailed(ctx, intent.OrderIntentID, "REJECTED", "cancel failed")
		return OrderResponse{}, err
	}
	if resp.Rejected {
		_ = ledger.MarkFailed(ctx, intent.OrderIntentID, "REJECTED", "cancel rejected")
		return resp, nil
	}
	if err := ledger.MarkConfirmed(ctx, intent.OrderIntentID, resp.OrderID, ""); err != nil {
		return resp, err
	}
	return resp, nil
}

func CancelReplaceWithIntent(ctx context.Context, ledger *LedgerService, rest OrderRestClient, intent sqlite.OrderIntentRecord, req CancelReplaceRequest) (OrderResponse, error) {
	if err := ledger.CreateIntent(ctx, intent); err != nil {
		return OrderResponse{}, err
	}
	resp, err := rest.CancelReplaceOrder(ctx, req)
	if err != nil {
		if errors.Is(err, ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
			_ = ledger.MarkSentUnknown(ctx, intent.OrderIntentID, "TIMEOUT", "cancel_replace timeout")
			return OrderResponse{}, ErrSentUnknown
		}
		_ = ledger.MarkFailed(ctx, intent.OrderIntentID, "REJECTED", "cancel_replace failed")
		return OrderResponse{}, err
	}
	if resp.Rejected {
		_ = ledger.MarkFailed(ctx, intent.OrderIntentID, "REJECTED", "cancel_replace rejected")
		return resp, nil
	}
	if err := ledger.MarkConfirmed(ctx, intent.OrderIntentID, resp.OrderID, ""); err != nil {
		return resp, err
	}
	return resp, nil
}
