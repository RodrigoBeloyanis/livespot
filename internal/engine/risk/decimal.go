package risk

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

func parseDecimalNonNegative(value string) (*big.Rat, error) {
	r, err := parseDecimalStrict(value)
	if err != nil {
		return nil, err
	}
	if r.Sign() < 0 {
		return nil, fmt.Errorf("decimal must be >= 0")
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

func ratToString(r *big.Rat, precision int) string {
	if r == nil {
		return ""
	}
	return r.FloatString(precision)
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

func truncateDecimalString(value string, dp int) (string, error) {
	if dp < 0 {
		return "", fmt.Errorf("invalid dp")
	}
	if value == "" {
		return "", fmt.Errorf("decimal missing")
	}
	if strings.Contains(value, ".") {
		parts := strings.SplitN(value, ".", 2)
		if len(parts[1]) <= dp {
			return value, nil
		}
		return parts[0] + "." + parts[1][:dp], nil
	}
	if dp == 0 {
		return value, nil
	}
	return value + "." + strings.Repeat("0", dp), nil
}
