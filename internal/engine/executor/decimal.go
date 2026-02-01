package executor

import (
	"fmt"
	"math/big"
	"strings"
)

func parseDecimalStrict(value string) (*big.Rat, error) {
	if value == "" {
		return nil, fmt.Errorf("decimal missing")
	}
	r := new(big.Rat)
	if _, ok := r.SetString(value); !ok {
		return nil, fmt.Errorf("invalid decimal")
	}
	return r, nil
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

func quantizeDown(value *big.Rat, step *big.Rat) (*big.Rat, error) {
	if value == nil || step == nil {
		return nil, fmt.Errorf("quantize input missing")
	}
	if value.Sign() < 0 {
		return nil, fmt.Errorf("quantize value must be >= 0")
	}
	if step.Sign() <= 0 {
		return nil, fmt.Errorf("quantize step must be > 0")
	}
	q := new(big.Rat).Quo(value, step)
	n := new(big.Int).Quo(q.Num(), q.Denom())
	return new(big.Rat).Mul(new(big.Rat).SetInt(n), step), nil
}

func ratToString(r *big.Rat, precision int) string {
	if r == nil {
		return ""
	}
	return r.FloatString(precision)
}

func QuantizePrice(price string, tickSize string) (string, error) {
	priceRat, err := parseDecimalStrict(price)
	if err != nil {
		return "", err
	}
	tick, err := parseDecimalStrict(tickSize)
	if err != nil {
		return "", err
	}
	quantized, err := quantizeDown(priceRat, tick)
	if err != nil {
		return "", err
	}
	return ratToString(quantized, decimalPlaces(tickSize)), nil
}

func isStepAligned(value int, step int) bool {
	if step <= 0 {
		return false
	}
	return value%step == 0
}
