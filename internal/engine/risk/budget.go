package risk

import (
	"math/big"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
)

func budgetReason(requiredQuote *big.Rat, free string, locked string, pending string) (reasoncodes.ReasonCode, error) {
	freeRat, err := parseDecimalStrict(free)
	if err != nil {
		return reasoncodes.STRAT_INPUT_INVALID, err
	}
	lockedRat, err := parseDecimalNonNegative(locked)
	if err != nil {
		return reasoncodes.STRAT_INPUT_INVALID, err
	}
	pendingRat, err := parseDecimalNonNegative(pending)
	if err != nil {
		return reasoncodes.STRAT_INPUT_INVALID, err
	}
	available := new(big.Rat).Sub(freeRat, lockedRat)
	available.Sub(available, pendingRat)
	if requiredQuote.Cmp(available) > 0 {
		return reasoncodes.RISK_INSUFFICIENT_FREE_BALANCE, nil
	}
	return "", nil
}
