package executor

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/infra/sqlite"
)

type IntentState string

const (
	IntentCreated     IntentState = "CREATED"
	IntentSentUnknown IntentState = "SENT_UNKNOWN"
	IntentConfirmed   IntentState = "CONFIRMED"
	IntentNotFound    IntentState = "NOT_FOUND"
	IntentFailed      IntentState = "FAILED_TERMINAL"
)

type IntentAction string

const (
	IntentActionNewOrder      IntentAction = "NEW_ORDER"
	IntentActionCancelOrder   IntentAction = "CANCEL_ORDER"
	IntentActionCancelReplace IntentAction = "CANCEL_REPLACE"
)

type LedgerService struct {
	DB    *sql.DB
	Clock func() time.Time
}

type RestClient interface {
	GetOrderByClientID(ctx context.Context, symbol string, clientOrderID string) (OrderResponse, error)
}

var ErrSentUnknown = errors.New("intent sent unknown")

func NewLedger(db *sql.DB, clock func() time.Time) *LedgerService {
	if clock == nil {
		clock = time.Now
	}
	return &LedgerService{DB: db, Clock: clock}
}

func (l *LedgerService) CreateIntent(ctx context.Context, rec sqlite.OrderIntentRecord) error {
	now := l.Clock().UnixMilli()
	rec.CreatedAtMs = now
	rec.UpdatedAtMs = now
	rec.State = string(IntentCreated)
	return sqlite.InsertOrderIntent(ctx, l.DB, rec)
}

func (l *LedgerService) MarkSentUnknown(ctx context.Context, id string, errCode string, errDetail string) error {
	return sqlite.UpdateOrderIntentState(ctx, l.DB, id, string(IntentSentUnknown), "", "", errCode, errDetail, l.Clock().UnixMilli())
}

func (l *LedgerService) MarkConfirmed(ctx context.Context, id string, orderID string, ocoID string) error {
	return sqlite.UpdateOrderIntentState(ctx, l.DB, id, string(IntentConfirmed), orderID, ocoID, "", "", l.Clock().UnixMilli())
}

func (l *LedgerService) MarkNotFound(ctx context.Context, id string) error {
	return sqlite.UpdateOrderIntentState(ctx, l.DB, id, string(IntentNotFound), "", "", "", "", l.Clock().UnixMilli())
}

func (l *LedgerService) MarkFailed(ctx context.Context, id string, errCode string, errDetail string) error {
	return sqlite.UpdateOrderIntentState(ctx, l.DB, id, string(IntentFailed), "", "", errCode, errDetail, l.Clock().UnixMilli())
}

func (l *LedgerService) PendingSentUnknown(ctx context.Context, limit int) ([]sqlite.OrderIntentRecord, error) {
	return sqlite.ListOrderIntentsByState(ctx, l.DB, string(IntentSentUnknown), limit)
}

func ResolveSentUnknown(ctx context.Context, cfg config.Config, ledger *LedgerService, rest RestClient, intent sqlite.OrderIntentRecord) error {
	allNotFound := true
	for i := 0; i < cfg.IntentMaxRestQueries; i++ {
		qctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.IntentRestQueryTimeoutMs)*time.Millisecond)
		resp, err := rest.GetOrderByClientID(qctx, intent.Symbol, intent.ClientOrderID)
		cancel()
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				allNotFound = false
				continue
			}
			allNotFound = false
			continue
		}
		if resp.Found {
			return ledger.MarkConfirmed(ctx, intent.OrderIntentID, resp.OrderID, "")
		}
	}
	if allNotFound {
		return ledger.MarkNotFound(ctx, intent.OrderIntentID)
	}
	return ErrSentUnknown
}
