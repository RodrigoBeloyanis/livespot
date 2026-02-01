package risk

import (
	"fmt"
	"math/big"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/executor"
)

type SizeResult struct {
	QtyBase   string
	QuoteUSDT string
}

func CalculatePositionSize(entryPrice string, stopPrice string, riskPerTrade string, constraints contracts.DecisionConstraints) (SizeResult, error) {
	entry, err := parseDecimal(entryPrice)
	if err != nil {
		return SizeResult{}, err
	}
	stop, err := parseDecimal(stopPrice)
	if err != nil {
		return SizeResult{}, err
	}
	risk, err := parseDecimal(riskPerTrade)
	if err != nil {
		return SizeResult{}, err
	}
	if entry.Sign() <= 0 {
		return SizeResult{}, fmt.Errorf("entry price invalid")
	}
	diff := new(big.Rat).Sub(entry, stop)
	if diff.Sign() < 0 {
		diff.Neg(diff)
	}
	stopDistanceBps := new(big.Rat).Mul(new(big.Rat).Quo(diff, entry), big.NewRat(10000, 1))
	if stopDistanceBps.Sign() <= 0 {
		return SizeResult{}, fmt.Errorf("stop distance invalid")
	}
	sizeQuoteRaw := new(big.Rat).Quo(new(big.Rat).Mul(risk, big.NewRat(10000, 1)), stopDistanceBps)
	sizeBaseRaw := new(big.Rat).Quo(sizeQuoteRaw, entry)
	sizeBaseStr := sizeBaseRaw.FloatString(8)
	priceQuantized, qtyQuantized, err := executor.QuantizeEntry(entryPrice, sizeBaseStr, constraints)
	if err != nil {
		return SizeResult{}, err
	}
	quote, err := multiplyDecimal(qtyQuantized, priceQuantized)
	if err != nil {
		return SizeResult{}, err
	}
	return SizeResult{
		QtyBase:   qtyQuantized,
		QuoteUSDT: quote,
	}, nil
}

func parseDecimal(value string) (*big.Rat, error) {
	r := new(big.Rat)
	if _, ok := r.SetString(value); !ok {
		return nil, fmt.Errorf("invalid decimal")
	}
	return r, nil
}

func multiplyDecimal(a string, b string) (string, error) {
	ra, err := parseDecimal(a)
	if err != nil {
		return "", err
	}
	rb, err := parseDecimal(b)
	if err != nil {
		return "", err
	}
	out := new(big.Rat).Mul(ra, rb)
	return out.FloatString(8), nil
}
