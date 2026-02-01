package e2e

import (
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
)

type ExchangeClient interface {
	Time(ctx Context) (int64, error)
	ExchangeInfo(ctx Context) ([]byte, error)
	Depth(ctx Context, symbol string, limit int) ([]byte, error)
	NewOrder(ctx Context, req OrderRequest) (OrderResponse, error)
	QueryOrder(ctx Context, symbol string, clientOrderID string) (OrderResponse, error)
	CancelOrder(ctx Context, symbol string, clientOrderID string) (OrderResponse, error)
	CancelReplace(ctx Context, req CancelReplaceRequest) (OrderResponse, error)
	OpenOrders(ctx Context, symbol string) ([]OrderResponse, error)
	AllOrders(ctx Context, symbol string, limit int) ([]OrderResponse, error)
	Account(ctx Context) ([]byte, error)
}

type Context interface {
	Done() <-chan struct{}
}

type OrderRequest struct {
	Symbol        string
	Side          string
	Price         string
	Qty           string
	ClientOrderID string
}

type CancelReplaceRequest struct {
	Symbol               string
	Side                 string
	Price                string
	Qty                  string
	CancelClientOrderID  string
	NewClientOrderID     string
	NewClientOrderIDHint string
}

type OrderResponse struct {
	Symbol        string
	ClientOrderID string
	Status        string
	Price         string
	Qty           string
	Side          string
	CreatedTsMs   int64
	UpdatedTsMs   int64
}

type PipelineInput struct {
	RunID          string
	CycleID        string
	Mode           string
	Snapshot       contracts.Snapshot
	SnapshotHash   string
	Decision       contracts.Decision
	DecisionID     string
	Exchange       ExchangeClient
	Now            func() time.Time
	DisableEntries bool
	Signals        SoakSignals
}

type PipelineResult struct {
	RunID          string
	CycleID        string
	StartTsMs      int64
	EndTsMs        int64
	Stages         []string
	ExchangeTimeMs int64
}

type SoakSignals struct {
	LastProgressTsMs     int64
	WsLastMsgTsMs        int64
	RestLastOkTsMs       int64
	DBWriterQueuePct     int
	ReconcileDriftX10000 int
}

type SoakViolation struct {
	Reason reasoncodes.ReasonCode
	AtTsMs int64
	Detail string
}

type SoakResult struct {
	StartTsMs  int64
	EndTsMs    int64
	TickCount  int
	Violations []SoakViolation
	Pass       bool
}

type ReadinessCheck struct {
	Name   string
	Pass   bool
	Detail string
}

type ReadinessReport struct {
	RunID      string
	StartTsMs  int64
	EndTsMs    int64
	DurationMs int64
	Checks     []ReadinessCheck
	Ready      bool
}
