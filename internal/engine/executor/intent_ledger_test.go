package executor

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/infra/sqlite"
)

type fakeRestClient struct {
	calls          int
	foundOnCall    int
	alwaysNotFound bool
}

func (f *fakeRestClient) GetOrderByClientID(ctx context.Context, symbol string, clientOrderID string) (OrderResponse, error) {
	f.calls++
	if f.alwaysNotFound {
		return OrderResponse{Found: false}, nil
	}
	if f.calls >= f.foundOnCall {
		return OrderResponse{Found: true, OrderID: "123"}, nil
	}
	return OrderResponse{Found: false}, nil
}

func TestResolveSentUnknownConfirmed(t *testing.T) {
	db := openTestDB(t)
	ledger := NewLedger(db, func() time.Time { return time.UnixMilli(1706700000000) })
	rec := sqlite.OrderIntentRecord{
		OrderIntentID:     "intent_1",
		RunID:             "run_1",
		CycleID:           "cyc_1",
		Mode:              "LIVE",
		DecisionID:        "dec_1",
		Symbol:            "BTCUSDT",
		Action:            string(IntentActionNewOrder),
		ClientOrderID:     "client_1",
		IntentPayloadJSON: "{}",
	}
	ctx := context.Background()
	if err := ledger.CreateIntent(ctx, rec); err != nil {
		t.Fatalf("create intent: %v", err)
	}
	if err := ledger.MarkSentUnknown(ctx, rec.OrderIntentID, "TIMEOUT", "timeout"); err != nil {
		t.Fatalf("mark sent unknown: %v", err)
	}
	fake := &fakeRestClient{foundOnCall: 2}
	cfg := config.Default()
	if err := ResolveSentUnknown(ctx, cfg, ledger, fake, rec); err != nil {
		t.Fatalf("resolve sent unknown: %v", err)
	}
	out, err := sqlite.GetOrderIntent(ctx, db, rec.OrderIntentID)
	if err != nil {
		t.Fatalf("get intent: %v", err)
	}
	if out.State != string(IntentConfirmed) {
		t.Fatalf("expected confirmed, got %s", out.State)
	}
}

func TestResolveSentUnknownNotFound(t *testing.T) {
	db := openTestDB(t)
	ledger := NewLedger(db, func() time.Time { return time.UnixMilli(1706700000000) })
	rec := sqlite.OrderIntentRecord{
		OrderIntentID:     "intent_2",
		RunID:             "run_1",
		CycleID:           "cyc_1",
		Mode:              "LIVE",
		DecisionID:        "dec_1",
		Symbol:            "BTCUSDT",
		Action:            string(IntentActionNewOrder),
		ClientOrderID:     "client_2",
		IntentPayloadJSON: "{}",
	}
	ctx := context.Background()
	if err := ledger.CreateIntent(ctx, rec); err != nil {
		t.Fatalf("create intent: %v", err)
	}
	if err := ledger.MarkSentUnknown(ctx, rec.OrderIntentID, "TIMEOUT", "timeout"); err != nil {
		t.Fatalf("mark sent unknown: %v", err)
	}
	fake := &fakeRestClient{alwaysNotFound: true}
	cfg := config.Default()
	if err := ResolveSentUnknown(ctx, cfg, ledger, fake, rec); err != nil {
		t.Fatalf("resolve sent unknown: %v", err)
	}
	out, err := sqlite.GetOrderIntent(ctx, db, rec.OrderIntentID)
	if err != nil {
		t.Fatalf("get intent: %v", err)
	}
	if out.State != string(IntentNotFound) {
		t.Fatalf("expected not found, got %s", out.State)
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	cfg := config.Default()
	path := t.TempDir() + "/test.sqlite"
	db, err := sqlite.Open(path, cfg)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	if err := sqlite.Migrate(db, time.UnixMilli(1706700000000)); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	return db
}
