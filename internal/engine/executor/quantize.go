package executor

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
)

func QuantizePrice(desired string, tickSize string) (string, error) {
	return quantizeToStep(desired, tickSize)
}

func QuantizeQty(desired string, stepSize string, minQty string, maxQty string, minNotional string, maxNotional string, price string) (string, error) {
	qty, err := quantizeToStep(desired, stepSize)
	if err != nil {
		return "", err
	}
	if err := ensureDecimalRange(qty, minQty, maxQty); err != nil {
		return "", err
	}
	if minNotional != "" {
		ok, err := notionalAtLeast(qty, price, minNotional)
		if err != nil {
			return "", err
		}
		if !ok {
			return "", fmt.Errorf("quantized notional below minimum")
		}
	}
	if maxNotional != "" {
		ok, err := notionalAtMost(qty, price, maxNotional)
		if err != nil {
			return "", err
		}
		if !ok {
			return "", fmt.Errorf("quantized notional above maximum")
		}
	}
	return qty, nil
}

func QuantizeEntry(desiredPrice string, desiredQty string, constraints contracts.DecisionConstraints) (string, string, error) {
	price, err := QuantizePrice(desiredPrice, constraints.TickSize)
	if err != nil {
		return "", "", err
	}
	qty, err := QuantizeQty(desiredQty, constraints.StepSize, constraints.MinQty, constraints.MaxQty, constraints.MinNotional, constraints.MaxNotional, price)
	if err != nil {
		return "", "", err
	}
	return price, qty, nil
}

func quantizeToStep(value string, step string) (string, error) {
	scale, err := decimalScale(step)
	if err != nil {
		return "", err
	}
	stepInt, err := decimalToScaledInt(step, scale)
	if err != nil {
		return "", err
	}
	if stepInt.Sign() <= 0 {
		return "", fmt.Errorf("step size invalid")
	}
	valueInt, err := decimalToScaledIntFloor(value, scale)
	if err != nil {
		return "", err
	}
	quotient := new(big.Int).Div(valueInt, stepInt)
	quantized := new(big.Int).Mul(quotient, stepInt)
	return formatScaledInt(quantized, scale), nil
}

func ensureDecimalRange(value string, min string, max string) error {
	if min != "" {
		cmp, err := compareDecimal(value, min)
		if err != nil {
			return err
		}
		if cmp < 0 {
			return fmt.Errorf("value below min")
		}
	}
	if max != "" {
		cmp, err := compareDecimal(value, max)
		if err != nil {
			return err
		}
		if cmp > 0 {
			return fmt.Errorf("value above max")
		}
	}
	return nil
}

func notionalAtLeast(qty string, price string, minNotional string) (bool, error) {
	n, err := multiplyDecimal(qty, price)
	if err != nil {
		return false, err
	}
	cmp, err := compareDecimal(n, minNotional)
	if err != nil {
		return false, err
	}
	return cmp >= 0, nil
}

func notionalAtMost(qty string, price string, maxNotional string) (bool, error) {
	n, err := multiplyDecimal(qty, price)
	if err != nil {
		return false, err
	}
	cmp, err := compareDecimal(n, maxNotional)
	if err != nil {
		return false, err
	}
	return cmp <= 0, nil
}

func decimalScale(value string) (int, error) {
	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid decimal")
	}
	if len(parts) == 1 {
		return 0, nil
	}
	return len(parts[1]), nil
}

func decimalToScaledInt(value string, scale int) (*big.Int, error) {
	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid decimal")
	}
	whole := parts[0]
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}
	if len(frac) > scale {
		return nil, fmt.Errorf("decimal scale exceeds step")
	}
	for len(frac) < scale {
		frac += "0"
	}
	raw := whole + frac
	out := new(big.Int)
	if raw == "" {
		return nil, fmt.Errorf("invalid decimal")
	}
	_, ok := out.SetString(raw, 10)
	if !ok {
		return nil, fmt.Errorf("invalid decimal")
	}
	return out, nil
}

func decimalToScaledIntFloor(value string, scale int) (*big.Int, error) {
	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return nil, fmt.Errorf("invalid decimal")
	}
	whole := parts[0]
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}
	if len(frac) > scale {
		frac = frac[:scale]
	}
	for len(frac) < scale {
		frac += "0"
	}
	raw := whole + frac
	out := new(big.Int)
	if raw == "" {
		return nil, fmt.Errorf("invalid decimal")
	}
	_, ok := out.SetString(raw, 10)
	if !ok {
		return nil, fmt.Errorf("invalid decimal")
	}
	return out, nil
}

func formatScaledInt(value *big.Int, scale int) string {
	if scale == 0 {
		return value.String()
	}
	s := value.String()
	if len(s) <= scale {
		s = strings.Repeat("0", scale-len(s)+1) + s
	}
	idx := len(s) - scale
	return s[:idx] + "." + s[idx:]
}

func compareDecimal(a string, b string) (int, error) {
	ra, ok := new(big.Rat).SetString(a)
	if !ok {
		return 0, fmt.Errorf("invalid decimal")
	}
	rb, ok := new(big.Rat).SetString(b)
	if !ok {
		return 0, fmt.Errorf("invalid decimal")
	}
	return ra.Cmp(rb), nil
}

func multiplyDecimal(a string, b string) (string, error) {
	ra, ok := new(big.Rat).SetString(a)
	if !ok {
		return "", fmt.Errorf("invalid decimal")
	}
	rb, ok := new(big.Rat).SetString(b)
	if !ok {
		return "", fmt.Errorf("invalid decimal")
	}
	out := new(big.Rat).Mul(ra, rb)
	return out.FloatString(8), nil
}
