package rank

import (
	"math"
	"math/big"
	"sort"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/selection"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/universe"
)

type RankedSymbol struct {
	Symbol      string
	ScoreX10000 int
	Features    map[string]any
	ConfigHash  string
}

func RankTopN(cfg config.Config, snapshots []contracts.Snapshot, universeResults []universe.ScanResult) ([]RankedSymbol, error) {
	configHash, err := selection.ConfigHash(cfg)
	if err != nil {
		return nil, err
	}
	eligible := make([]contracts.Snapshot, 0, len(snapshots))
	eligibleSet := map[string]bool{}
	for _, result := range universeResults {
		if result.Eligible {
			eligibleSet[result.Symbol] = true
		}
	}
	for _, snapshot := range snapshots {
		if eligibleSet[snapshot.Symbol] {
			eligible = append(eligible, snapshot)
		}
	}
	if len(eligible) == 0 {
		return []RankedSymbol{}, nil
	}

	maxVolume := 1.0
	maxMomentum := 1
	maxSpread := 1
	for _, snapshot := range eligible {
		volume := parseVolume(snapshot.Market24h.QuoteVolume24hUSDT)
		if volume > maxVolume {
			maxVolume = volume
		}
		if snapshot.Market24h.PriceChange24hBps > maxMomentum {
			maxMomentum = snapshot.Market24h.PriceChange24hBps
		}
		if snapshot.Microstructure60s.SpreadBpsP50_60s > maxSpread {
			maxSpread = snapshot.Microstructure60s.SpreadBpsP50_60s
		}
	}

	results := make([]RankedSymbol, 0, len(eligible))
	for _, snapshot := range eligible {
		volumeScore := parseVolume(snapshot.Market24h.QuoteVolume24hUSDT) / maxVolume
		momentumScore := float64(snapshot.Market24h.PriceChange24hBps) / float64(maxMomentum)
		spreadScore := 1.0 - (float64(snapshot.Microstructure60s.SpreadBpsP50_60s) / float64(maxSpread))
		if spreadScore < 0 {
			spreadScore = 0
		}
		if spreadScore > 1 {
			spreadScore = 1
		}
		score := (cfg.RankWeightLiquidity * volumeScore) + (cfg.RankWeightMomentum * momentumScore) + (cfg.RankWeightSpread * spreadScore)
		scoreX10000 := int(math.RoundToEven(score * 10000.0))
		features := map[string]any{
			"quote_volume_24h_usdt": snapshot.Market24h.QuoteVolume24hUSDT,
			"price_change_24h_bps":  snapshot.Market24h.PriceChange24hBps,
			"spread_bps_p50_60s":    snapshot.Microstructure60s.SpreadBpsP50_60s,
		}
		results = append(results, RankedSymbol{
			Symbol:      snapshot.Symbol,
			ScoreX10000: scoreX10000,
			Features:    features,
			ConfigHash:  configHash,
		})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].ScoreX10000 == results[j].ScoreX10000 {
			return results[i].Symbol < results[j].Symbol
		}
		return results[i].ScoreX10000 > results[j].ScoreX10000
	})
	if len(results) > cfg.TopNSize {
		results = results[:cfg.TopNSize]
	}
	return results, nil
}

func parseVolume(value string) float64 {
	r := new(big.Rat)
	if _, ok := r.SetString(value); !ok {
		return 0
	}
	f, _ := r.Float64()
	return f
}
