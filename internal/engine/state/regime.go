package state

import (
	"fmt"
	"math"
)

type adxResult struct {
	ADX float64
	PlusDI float64
	MinusDI float64
}

func RegimeScores(adx1m adxResult, adx5m adxResult, adx15m adxResult, adx1h adxResult) (int, int, string) {
	agg := (adx1h.ADX*0.4 + adx15m.ADX*0.3 + adx5m.ADX*0.2 + adx1m.ADX*0.1) / 100.0
	trend := int(math.RoundToEven(agg * 10000.0))
	if trend < 0 {
		trend = 0
	}
	if trend > 10000 {
		trend = 10000
	}
	rangeScore := 10000 - trend
	label := "UNCLEAR"
	if trend >= 7000 {
		label = "TREND"
	} else if rangeScore >= 6000 {
		label = "RANGE"
	}
	return trend, rangeScore, label
}

func ComputeADX(high []float64, low []float64, close []float64) (adxResult, error) {
	if len(high) < 15 || len(low) < 15 || len(close) < 15 {
		return adxResult{}, fmt.Errorf("adx samples")
	}
	tr := make([]float64, 0, len(high)-1)
	plusDM := make([]float64, 0, len(high)-1)
	minusDM := make([]float64, 0, len(high)-1)
	for i := 1; i < len(high); i++ {
		upMove := high[i] - high[i-1]
		downMove := low[i-1] - low[i]
		pdm := 0.0
		mdm := 0.0
		if upMove > downMove && upMove > 0 {
			pdm = upMove
		}
		if downMove > upMove && downMove > 0 {
			mdm = downMove
		}
		hl := high[i] - low[i]
		hc := math.Abs(high[i] - close[i-1])
		lc := math.Abs(low[i] - close[i-1])
		tr = append(tr, math.Max(hl, math.Max(hc, lc)))
		plusDM = append(plusDM, pdm)
		minusDM = append(minusDM, mdm)
	}
	period := 14
	tr14 := sum(tr[:period])
	plus14 := sum(plusDM[:period])
	minus14 := sum(minusDM[:period])
	plusDI := 100.0 * (plus14 / tr14)
	minusDI := 100.0 * (minus14 / tr14)
	dx := make([]float64, 0, len(tr)-period+1)
	dx = append(dx, dxVal(plusDI, minusDI))
	for i := period; i < len(tr); i++ {
		tr14 = tr14 - (tr14 / float64(period)) + tr[i]
		plus14 = plus14 - (plus14 / float64(period)) + plusDM[i]
		minus14 = minus14 - (minus14 / float64(period)) + minusDM[i]
		plusDI = 100.0 * (plus14 / tr14)
		minusDI = 100.0 * (minus14 / tr14)
		dx = append(dx, dxVal(plusDI, minusDI))
	}
	if len(dx) < period {
		return adxResult{}, fmt.Errorf("adx dx")
	}
	adx := sum(dx[:period]) / float64(period)
	for i := period; i < len(dx); i++ {
		adx = ((adx * float64(period-1)) + dx[i]) / float64(period)
	}
	return adxResult{ADX: adx, PlusDI: plusDI, MinusDI: minusDI}, nil
}

func dxVal(plusDI float64, minusDI float64) float64 {
	den := plusDI + minusDI
	if den == 0 {
		return 0
	}
	return math.Abs(plusDI-minusDI) / den * 100.0
}

func sum(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total
}

