package risk

import (
	"fmt"
	"math/big"
)

func CorrelationX10000(seriesA []int32, seriesB []int32) (int, error) {
	if len(seriesA) == 0 || len(seriesA) != len(seriesB) {
		return 0, fmt.Errorf("series length mismatch")
	}
	var sumX, sumY, sumX2, sumY2, sumXY int64
	for i := 0; i < len(seriesA); i++ {
		x := int64(seriesA[i])
		y := int64(seriesB[i])
		sumX += x
		sumY += y
		sumX2 += x * x
		sumY2 += y * y
		sumXY += x * y
	}
	n := int64(len(seriesA))
	num := n*sumXY - sumX*sumY
	denA := n*sumX2 - sumX*sumX
	denB := n*sumY2 - sumY*sumY
	if denA <= 0 || denB <= 0 {
		return 0, fmt.Errorf("zero variance")
	}
	den := new(big.Int).Mul(big.NewInt(denA), big.NewInt(denB))
	den.Sqrt(den)
	if den.Sign() == 0 {
		return 0, fmt.Errorf("zero variance")
	}
	ratio := new(big.Rat).SetFrac(big.NewInt(num), den)
	ratio.Mul(ratio, big.NewRat(10000, 1))
	return roundRatToInt(ratio), nil
}

func roundRatToInt(r *big.Rat) int {
	if r == nil {
		return 0
	}
	num := new(big.Int).Set(r.Num())
	den := new(big.Int).Set(r.Denom())
	q, rem := new(big.Int).QuoRem(num, den, new(big.Int))
	if rem.Sign() == 0 {
		return int(q.Int64())
	}
	absRem := new(big.Int).Abs(rem)
	twiceRem := new(big.Int).Mul(absRem, big.NewInt(2))
	if twiceRem.Cmp(den) >= 0 {
		if r.Sign() >= 0 {
			q.Add(q, big.NewInt(1))
		} else {
			q.Sub(q, big.NewInt(1))
		}
	}
	return int(q.Int64())
}
