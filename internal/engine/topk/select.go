package topk

import (
	"math"
	"sort"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
)

type Selection struct {
	Symbol      string
	ScoreX10000 int
	Features    map[string]any
}

type SelectionResult struct {
	TopK              []Selection
	MaxPairwiseCorr   int
	PairsOverLimit    []PairOverLimit
	ChurnGuardApplied bool
}

type PairOverLimit struct {
	A          string
	B          string
	CorrX10000 int
	Action     string
}

func SelectTopK(cfg config.Config, ranked []Selection, snapshots map[string]contracts.Snapshot, prevTopK []string, cyclesSincePrev int) SelectionResult {
	ordered := make([]Selection, len(ranked))
	copy(ordered, ranked)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].ScoreX10000 == ordered[j].ScoreX10000 {
			return ordered[i].Symbol < ordered[j].Symbol
		}
		return ordered[i].ScoreX10000 > ordered[j].ScoreX10000
	})

	var appliedChurn bool
	if cfg.ChurnGuardEnabled && len(prevTopK) > 0 && cyclesSincePrev < cfg.ChurnGuardMinCycles {
		ordered, appliedChurn = applyChurnGuard(cfg, ordered, prevTopK)
	}

	selected := []Selection{}
	pairsOver := []PairOverLimit{}
	maxCorr := 0
	for _, candidate := range ordered {
		if len(selected) >= cfg.TopKSize {
			break
		}
		allow := true
		for _, existing := range selected {
			corr := correlationX10000(snapshots[candidate.Symbol], snapshots[existing.Symbol], cfg.CorrWindowPoints)
			if corr > maxCorr {
				maxCorr = corr
			}
			if corr >= cfg.CorrMaxX10000 {
				pairsOver = append(pairsOver, PairOverLimit{
					A:          candidate.Symbol,
					B:          existing.Symbol,
					CorrX10000: corr,
					Action:     "REJECT",
				})
				allow = false
				break
			}
		}
		if allow {
			selected = append(selected, candidate)
		}
	}

	return SelectionResult{
		TopK:              selected,
		MaxPairwiseCorr:   maxCorr,
		PairsOverLimit:    pairsOver,
		ChurnGuardApplied: appliedChurn,
	}
}

func applyChurnGuard(cfg config.Config, ordered []Selection, prevTopK []string) ([]Selection, bool) {
	prevSet := map[string]bool{}
	for _, sym := range prevTopK {
		prevSet[sym] = true
	}
	if cfg.ChurnGuardMinScoreDeltaX10000 <= 0 {
		return ordered, false
	}
	guarded := make([]Selection, 0, len(ordered))
	for _, item := range ordered {
		if prevSet[item.Symbol] {
			guarded = append(guarded, item)
			continue
		}
		if len(guarded) >= cfg.TopKSize {
			continue
		}
		guarded = append(guarded, item)
	}
	if len(guarded) == 0 {
		return ordered, false
	}
	return guarded, true
}

func correlationX10000(a contracts.Snapshot, b contracts.Snapshot, window int) int {
	if window <= 0 {
		return 0
	}
	if a.ReturnsSeries.WindowPoints < window || b.ReturnsSeries.WindowPoints < window {
		return 0
	}
	if a.ReturnsSeries.MissingCount > int(float64(window)*0.10) || b.ReturnsSeries.MissingCount > int(float64(window)*0.10) {
		return 0
	}
	xa := a.ReturnsSeries.LogReturnBps
	xb := b.ReturnsSeries.LogReturnBps
	if len(xa) < window || len(xb) < window {
		return 0
	}
	var sumA, sumB int64
	for i := len(xa) - window; i < len(xa); i++ {
		sumA += int64(xa[i])
		sumB += int64(xb[i])
	}
	meanA := float64(sumA) / float64(window)
	meanB := float64(sumB) / float64(window)
	var num, denomA, denomB float64
	for i := len(xa) - window; i < len(xa); i++ {
		da := float64(xa[i]) - meanA
		db := float64(xb[i]) - meanB
		num += da * db
		denomA += da * da
		denomB += db * db
	}
	if denomA == 0 || denomB == 0 {
		return 0
	}
	corr := num / math.Sqrt(denomA*denomB)
	if corr < 0 {
		corr = -corr
	}
	if corr > 1 {
		corr = 1
	}
	return int(math.RoundToEven(corr * 10000.0))
}
