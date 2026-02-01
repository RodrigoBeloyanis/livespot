package executor

import (
	"fmt"
	"math/big"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
)

func QuantizeOrder(req OrderRequest, filters SymbolFilters) (OrderRequest, reasoncodes.ReasonCode, error) {
	out := req
	if filters.LotSize == nil && filters.MarketLotSize == nil {
		return out, reasoncodes.PROTECTION_INVALID_FILTER, fmt.Errorf("lot size missing")
	}
	if req.Type != OrderTypeMarket && filters.Price == nil {
		return out, reasoncodes.PROTECTION_INVALID_FILTER, fmt.Errorf("price filter missing")
	}
	if req.Type == OrderTypeMarket && req.Price == "" {
		return out, reasoncodes.PROTECTION_INVALID_FILTER, fmt.Errorf("market price missing")
	}

	pricePrecision := 0
	if filters.Price != nil {
		pricePrecision = decimalPlaces(filters.Price.TickSize)
		priceRat, err := parseDecimalStrict(req.Price)
		if err != nil {
			return out, reasoncodes.PROTECTION_INVALID_FILTER, err
		}
		tick, err := parseDecimalStrict(filters.Price.TickSize)
		if err != nil {
			return out, reasoncodes.PROTECTION_INVALID_FILTER, err
		}
		quantizedPrice, err := quantizeDown(priceRat, tick)
		if err != nil {
			return out, reasoncodes.PROTECTION_INVALID_FILTER, err
		}
		if filters.Price.MinPrice != "" {
			minPrice, err := parseDecimalStrict(filters.Price.MinPrice)
			if err != nil {
				return out, reasoncodes.PROTECTION_INVALID_FILTER, err
			}
			if quantizedPrice.Cmp(minPrice) < 0 {
				return out, reasoncodes.PROTECTION_INVALID_FILTER, fmt.Errorf("price below min")
			}
		}
		if filters.Price.MaxPrice != "" {
			maxPrice, err := parseDecimalStrict(filters.Price.MaxPrice)
			if err != nil {
				return out, reasoncodes.PROTECTION_INVALID_FILTER, err
			}
			if quantizedPrice.Cmp(maxPrice) > 0 {
				return out, reasoncodes.PROTECTION_INVALID_FILTER, fmt.Errorf("price above max")
			}
		}
		out.Price = ratToString(quantizedPrice, pricePrecision)
	}

	lot := filters.LotSize
	if req.Type == OrderTypeMarket && filters.MarketLotSize != nil {
		lot = filters.MarketLotSize
	}
	qtyPrecision := decimalPlaces(lot.StepSize)
	qtyRat, err := parseDecimalStrict(req.Qty)
	if err != nil {
		return out, reasoncodes.PROTECTION_INVALID_FILTER, err
	}
	step, err := parseDecimalStrict(lot.StepSize)
	if err != nil {
		return out, reasoncodes.PROTECTION_INVALID_FILTER, err
	}
	quantizedQty, err := quantizeDown(qtyRat, step)
	if err != nil {
		return out, reasoncodes.PROTECTION_INVALID_FILTER, err
	}
	if lot.MinQty != "" {
		minQty, err := parseDecimalStrict(lot.MinQty)
		if err != nil {
			return out, reasoncodes.PROTECTION_INVALID_FILTER, err
		}
		if quantizedQty.Cmp(minQty) < 0 {
			return out, reasoncodes.PROTECTION_INVALID_FILTER, fmt.Errorf("qty below min")
		}
	}
	if lot.MaxQty != "" {
		maxQty, err := parseDecimalStrict(lot.MaxQty)
		if err != nil {
			return out, reasoncodes.PROTECTION_INVALID_FILTER, err
		}
		if quantizedQty.Cmp(maxQty) > 0 {
			return out, reasoncodes.PROTECTION_INVALID_FILTER, fmt.Errorf("qty above max")
		}
	}
	out.Qty = ratToString(quantizedQty, qtyPrecision)

	if filters.MinNotional != nil && filters.MinNotional.MinNotional != "" {
		minNotional, err := parseDecimalStrict(filters.MinNotional.MinNotional)
		if err != nil {
			return out, reasoncodes.PROTECTION_INVALID_FILTER, err
		}
		priceRat, err := parseDecimalStrict(out.Price)
		if err != nil {
			return out, reasoncodes.PROTECTION_INVALID_FILTER, err
		}
		notional := new(big.Rat).Mul(priceRat, quantizedQty)
		if notional.Cmp(minNotional) < 0 {
			return out, reasoncodes.PROTECTION_INVALID_MIN_NOTIONAL, fmt.Errorf("notional below min")
		}
	}

	if req.TrailingDeltaBips > 0 {
		if filters.TrailingDelta == nil {
			return out, reasoncodes.PROTECTION_INVALID_FILTER, fmt.Errorf("trailing delta missing")
		}
		td := filters.TrailingDelta
		if req.TrailingDeltaBips < td.MinTrailingDeltaBips || req.TrailingDeltaBips > td.MaxTrailingDeltaBips {
			return out, reasoncodes.PROTECTION_INVALID_FILTER, fmt.Errorf("trailing delta out of bounds")
		}
		if !isStepAligned(req.TrailingDeltaBips, td.StepBips) {
			return out, reasoncodes.PROTECTION_INVALID_FILTER, fmt.Errorf("trailing delta not aligned")
		}
	}

	return out, "", nil
}
