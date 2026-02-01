package executor

import "github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"

type OrderType string

const (
	OrderTypeLimit      OrderType = "LIMIT"
	OrderTypeLimitMaker OrderType = "LIMIT_MAKER"
	OrderTypeMarket     OrderType = "MARKET"
)

type OrderRequest struct {
	Symbol            string
	Side              contracts.Side
	Type              OrderType
	TimeInForce       contracts.TimeInForce
	Price             string
	Qty               string
	StopPrice         string
	TrailingDeltaBips int
	ClientOrderID     string
}

type OrderResponse struct {
	Found         bool
	Rejected      bool
	OrderID       string
	ClientOrderID string
	Status        string
}

type CancelRequest struct {
	Symbol        string
	ClientOrderID string
}

type CancelReplaceRequest struct {
	Symbol        string
	ClientOrderID string
	NewClientID   string
	NewPrice      string
	NewQty        string
}

type PriceFilter struct {
	MinPrice string
	MaxPrice string
	TickSize string
}

type LotSizeFilter struct {
	MinQty   string
	MaxQty   string
	StepSize string
}

type MinNotionalFilter struct {
	MinNotional string
}

type MarketLotSizeFilter = LotSizeFilter

type TrailingDeltaFilter struct {
	MinTrailingDeltaBips int
	MaxTrailingDeltaBips int
	StepBips             int
}

type SymbolFilters struct {
	Price         *PriceFilter
	LotSize       *LotSizeFilter
	MinNotional   *MinNotionalFilter
	MarketLotSize *MarketLotSizeFilter
	TrailingDelta *TrailingDeltaFilter
}
