package state

import (
	"fmt"
	"math"
)

func ReturnsSeriesBps(closes []float64, window int) ([]int32, int, error) {
	if window <= 0 {
		return nil, 0, fmt.Errorf("window")
	}
	if len(closes) < window+1 {
		return nil, 0, fmt.Errorf("closes")
	}
	start := len(closes) - (window + 1)
	out := make([]int32, 0, window)
	missing := 0
	for i := start + 1; i < len(closes); i++ {
		prev := closes[i-1]
		cur := closes[i]
		if prev <= 0 || cur <= 0 {
			out = append(out, 0)
			missing++
			continue
		}
		ret := math.Log(cur/prev) * 10000.0
		val := int32(math.RoundToEven(ret))
		out = append(out, val)
	}
	if len(out) != window {
		return nil, 0, fmt.Errorf("returns size")
	}
	return out, missing, nil
}

