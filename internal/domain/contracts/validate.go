package contracts

import (
	"fmt"
	"math/big"
	"regexp"
)

var decimalRe = regexp.MustCompile(`^[0-9]+(\.[0-9]+)?$`)

func isDecimalString(s string) bool {
	if s == "" {
		return false
	}
	return decimalRe.MatchString(s)
}

func parseDecimal(s string) (*big.Rat, error) {
	if !isDecimalString(s) {
		return nil, fmt.Errorf("invalid decimal string")
	}
	r := new(big.Rat)
	if _, ok := r.SetString(s); !ok {
		return nil, fmt.Errorf("invalid decimal string")
	}
	return r, nil
}

func compareDecimal(a, b string) (int, error) {
	ra, err := parseDecimal(a)
	if err != nil {
		return 0, err
	}
	rb, err := parseDecimal(b)
	if err != nil {
		return 0, err
	}
	return ra.Cmp(rb), nil
}

func isLowerHex64(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			continue
		}
		return false
	}
	return true
}
