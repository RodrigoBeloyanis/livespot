package state

import (
	"fmt"
	"math"
)

func ATR14Bps(high []float64, low []float64, close []float64) (int, error) {
	if len(high) < 15 || len(low) < 15 || len(close) < 15 {
		return 0, fmt.Errorf("atr samples")
	}
	tr := make([]float64, 0, len(high)-1)
	for i := 1; i < len(high); i++ {
		hl := high[i] - low[i]
		hc := math.Abs(high[i] - close[i-1])
		lc := math.Abs(low[i] - close[i-1])
		tr = append(tr, math.Max(hl, math.Max(hc, lc)))
	}
	period := 14
	atr := 0.0
	for i := 0; i < period; i++ {
		atr += tr[i]
	}
	atr = atr / float64(period)
	for i := period; i < len(tr); i++ {
		atr = ((atr * float64(period-1)) + tr[i]) / float64(period)
	}
	lastClose := close[len(close)-1]
	if lastClose <= 0 {
		return 0, fmt.Errorf("atr close")
	}
	atrBps := int(math.RoundToEven((atr / lastClose) * 10000.0))
	if atrBps <= 0 {
		return 0, fmt.Errorf("atr bps")
	}
	return atrBps, nil
}

