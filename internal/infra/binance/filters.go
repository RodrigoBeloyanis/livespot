package binance

import (
	"fmt"
	"strings"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/hash"
)

type SymbolFilters struct {
	Symbol       string
	Status       string
	QuoteAsset   string
	Price        *Filter
	LotSize      *Filter
	MinNotional  *Filter
	MarketLot    *Filter
	TrailingDelta *Filter
	MaxNumOrders int
	MaxAlgoOrders int
}

func BuildFiltersMap(info ExchangeInfo) map[string]SymbolFilters {
	out := map[string]SymbolFilters{}
	for _, symbol := range info.Symbols {
		filters := SymbolFilters{
			Symbol:     symbol.Symbol,
			Status:     symbol.Status,
			QuoteAsset: symbol.QuoteAsset,
		}
		for _, f := range symbol.Filters {
			switch f.FilterType {
			case "PRICE_FILTER":
				filters.Price = &f
			case "LOT_SIZE":
				filters.LotSize = &f
			case "MIN_NOTIONAL":
				filters.MinNotional = &f
			case "MARKET_LOT_SIZE":
				filters.MarketLot = &f
			case "TRAILING_DELTA":
				filters.TrailingDelta = &f
			case "MAX_NUM_ORDERS":
				filters.MaxNumOrders = f.MaxNumOrders
			case "MAX_NUM_ALGO_ORDERS":
				filters.MaxAlgoOrders = f.MaxNumAlgoOrders
			}
		}
		out[symbol.Symbol] = filters
	}
	return out
}

func ConstraintsFromFilters(filters SymbolFilters) (contracts.DecisionConstraints, error) {
	if filters.Price == nil || filters.LotSize == nil || filters.MinNotional == nil {
		return contracts.DecisionConstraints{}, fmt.Errorf("filters missing")
	}
	pricePrecision := decimalPlaces(filters.Price.TickSize)
	qtyPrecision := decimalPlaces(filters.LotSize.StepSize)
	constraints := contracts.DecisionConstraints{
		TickSize:           filters.Price.TickSize,
		StepSize:           filters.LotSize.StepSize,
		MinQty:             filters.LotSize.MinQty,
		MinNotional:        filters.MinNotional.MinNotional,
		PricePrecision:     pricePrecision,
		QtyPrecision:       qtyPrecision,
		MaxQty:             filters.LotSize.MaxQty,
		MaxNumOrders:       filters.MaxNumOrders,
		MaxAlgoOrders:      filters.MaxAlgoOrders,
		MaxNotional:        "",
		QuantizationPolicy: contracts.QuantizationEnforced,
	}
	return constraints, nil
}

func FiltersHash(filters SymbolFilters) (string, error) {
	payload := map[string]any{
		"symbol_status":   filters.Status,
		"filters": map[string]any{
			"tick_size":    filters.Price.TickSize,
			"step_size":    filters.LotSize.StepSize,
			"min_qty":      filters.LotSize.MinQty,
			"max_qty":      filters.LotSize.MaxQty,
			"min_notional": filters.MinNotional.MinNotional,
			"max_num_orders": filters.MaxNumOrders,
			"max_algo_orders": filters.MaxAlgoOrders,
		},
	}
	if filters.MarketLot != nil {
		payload["filters"].(map[string]any)["market_step_size"] = filters.MarketLot.StepSize
	}
	if filters.TrailingDelta != nil {
		payload["filters"].(map[string]any)["trailing_min_bips"] = filters.TrailingDelta.TrailingDeltaMin
		payload["filters"].(map[string]any)["trailing_max_bips"] = filters.TrailingDelta.TrailingDeltaMax
		payload["filters"].(map[string]any)["trailing_step_bips"] = filters.TrailingDelta.TrailingDeltaStep
	}
	return hash.CanonicalHash(payload)
}

func decimalPlaces(value string) int {
	if value == "" {
		return 0
	}
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 {
		return 0
	}
	return len(parts[1])
}
