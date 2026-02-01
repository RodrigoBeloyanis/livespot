package risk

import (
	"fmt"
	"math/big"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
)

type PositionSize struct {
	Base  *big.Rat
	Quote *big.Rat
}

func ValidatePositionSizing(cfg config.Config, decision contracts.Decision) (reasoncodes.ReasonCode, *PositionSize, error) {
	if decision.EntryPlan == nil || decision.ExitPlan == nil {
		return reasoncodes.STRAT_INPUT_INVALID, nil, fmt.Errorf("entry/exit plan missing")
	}
	entryPrice := decision.EntryPlan.LimitPrice
	if entryPrice == "" {
		entryPrice = decision.EntryPlan.DesiredPrice
	}
	stopPrice := decision.ExitPlan.SLPrice
	if entryPrice == "" || stopPrice == "" {
		return reasoncodes.STRAT_INPUT_INVALID, nil, fmt.Errorf("entry/stop price missing")
	}
	riskValue, err := truncateDecimalString(cfg.RiskPerTradeUSDT, 2)
	if err != nil {
		return reasoncodes.STRAT_INPUT_INVALID, nil, err
	}
	size, err := computePositionSize(entryPrice, stopPrice, riskValue, decision.Constraints)
	if err != nil {
		switch err.Error() {
		case "min_notional":
			return reasoncodes.PROTECTION_INVALID_MIN_NOTIONAL, nil, nil
		case "filters":
			return reasoncodes.PROTECTION_INVALID_FILTER, nil, nil
		case "size_invalid":
			return reasoncodes.RISK_SIZE_INVALID, nil, nil
		default:
			return reasoncodes.STRAT_INPUT_INVALID, nil, err
		}
	}
	entryQty, err := parseDecimalStrict(decision.EntryPlan.Qty)
	if err != nil {
		return reasoncodes.STRAT_INPUT_INVALID, nil, err
	}
	if entryQty.Sign() <= 0 {
		return reasoncodes.RISK_SIZE_INVALID, nil, nil
	}
	if entryQty.Cmp(size.Base) > 0 {
		return reasoncodes.RISK_SIZE_INVALID, nil, nil
	}
	entryPriceRat, err := parseDecimalStrict(entryPrice)
	if err != nil {
		return reasoncodes.STRAT_INPUT_INVALID, nil, err
	}
	minNotional, err := parseDecimalStrict(decision.Constraints.MinNotional)
	if err != nil {
		return reasoncodes.STRAT_INPUT_INVALID, nil, err
	}
	entryNotional := new(big.Rat).Mul(entryQty, entryPriceRat)
	if entryNotional.Cmp(minNotional) < 0 {
		return reasoncodes.PROTECTION_INVALID_MIN_NOTIONAL, nil, nil
	}
	return "", size, nil
}

func computePositionSize(entryPrice string, stopPrice string, riskPerTrade string, constraints contracts.DecisionConstraints) (*PositionSize, error) {
	entry, err := parseDecimalStrict(entryPrice)
	if err != nil {
		return nil, fmt.Errorf("filters")
	}
	stop, err := parseDecimalStrict(stopPrice)
	if err != nil {
		return nil, fmt.Errorf("filters")
	}
	risk, err := parseDecimalStrict(riskPerTrade)
	if err != nil {
		return nil, fmt.Errorf("filters")
	}
	step, err := parseDecimalStrict(constraints.StepSize)
	if err != nil {
		return nil, fmt.Errorf("filters")
	}
	minQty, err := parseDecimalStrict(constraints.MinQty)
	if err != nil {
		return nil, fmt.Errorf("filters")
	}
	maxQty, err := parseDecimalStrict(constraints.MaxQty)
	if err != nil {
		return nil, fmt.Errorf("filters")
	}
	minNotional, err := parseDecimalStrict(constraints.MinNotional)
	if err != nil {
		return nil, fmt.Errorf("filters")
	}
	var maxNotional *big.Rat
	if constraints.MaxNotional != "" {
		maxNotional, err = parseDecimalStrict(constraints.MaxNotional)
		if err != nil {
			return nil, fmt.Errorf("filters")
		}
	}
	if entry.Sign() <= 0 || risk.Sign() <= 0 {
		return nil, fmt.Errorf("size_invalid")
	}
	stopDistance := new(big.Rat).Sub(entry, stop)
	if stopDistance.Sign() < 0 {
		stopDistance.Neg(stopDistance)
	}
	if stopDistance.Sign() == 0 {
		return nil, fmt.Errorf("size_invalid")
	}
	stopDistance = new(big.Rat).Quo(stopDistance, entry)
	stopDistance.Mul(stopDistance, big.NewRat(10000, 1))
	if stopDistance.Sign() == 0 {
		return nil, fmt.Errorf("size_invalid")
	}
	sizeQuote := new(big.Rat).Mul(risk, big.NewRat(10000, 1))
	sizeQuote.Quo(sizeQuote, stopDistance)
	if sizeQuote.Sign() <= 0 {
		return nil, fmt.Errorf("size_invalid")
	}
	sizeBase := new(big.Rat).Quo(sizeQuote, entry)
	quantized, err := quantizeDown(sizeBase, step)
	if err != nil {
		return nil, fmt.Errorf("filters")
	}
	if quantized.Sign() <= 0 {
		return nil, fmt.Errorf("size_invalid")
	}
	if quantized.Cmp(minQty) < 0 || quantized.Cmp(maxQty) > 0 {
		return nil, fmt.Errorf("filters")
	}
	sizeQuote = new(big.Rat).Mul(quantized, entry)
	if sizeQuote.Cmp(minNotional) < 0 {
		return nil, fmt.Errorf("min_notional")
	}
	if maxNotional != nil && sizeQuote.Cmp(maxNotional) > 0 {
		return nil, fmt.Errorf("filters")
	}
	return &PositionSize{Base: quantized, Quote: sizeQuote}, nil
}
