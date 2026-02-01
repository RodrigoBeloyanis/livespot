package deepscan

import (
	"math"
	"sort"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/selection"
)

type DeepResult struct {
	Symbol      string
	ScoreX10000 int
	Features    map[string]any
	ConfigHash  string
}

func DeepScan(cfg config.Config, snapshots []contracts.Snapshot) ([]DeepResult, error) {
	configHash, err := selection.ConfigHash(cfg)
	if err != nil {
		return nil, err
	}
	results := make([]DeepResult, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if snapshot.Microstructure60s.SpreadCurrentBps > cfg.DeepMaxSpreadBps {
			continue
		}
		if snapshot.Microstructure60s.BidAskImbalanceP50_10sX10000 < cfg.DeepMinImbalanceX10000 {
			continue
		}
		edgeScore := estimateEdgeBps(snapshot)
		if edgeScore < cfg.DeepMinEdgeBps {
			continue
		}
		regimeScore := regimeComponent(snapshot.Regime)
		microScore := microstructureComponent(snapshot.Microstructure60s, cfg.DeepMaxSpreadBps, cfg.DeepMinImbalanceX10000)
		volScore := volatilityComponent(snapshot.Volatility)

		edgeDenom := float64(cfg.DeepMinEdgeBps * 2)
		if edgeDenom <= 0 {
			edgeDenom = 1
		}
		score := (cfg.DeepWeightEdge * clamp01(float64(edgeScore)/edgeDenom)) +
			(cfg.DeepWeightRegime * regimeScore) +
			(cfg.DeepWeightMicrostructure * microScore) +
			(cfg.DeepWeightVolatility * volScore)
		scoreX10000 := int(math.RoundToEven(clamp01(score) * 10000.0))
		features := map[string]any{
			"edge_bps_estimate":                edgeScore,
			"regime_label":                     snapshot.Regime.Label,
			"trend_score_x10000":               snapshot.Regime.TrendScoreX10000,
			"range_score_x10000":               snapshot.Regime.RangeScoreX10000,
			"spread_current_bps":               snapshot.Microstructure60s.SpreadCurrentBps,
			"bid_ask_imbalance_p50_10s_x10000": snapshot.Microstructure60s.BidAskImbalanceP50_10sX10000,
			"atr14_5m_bps":                     snapshot.Volatility.ATR14_5mBps,
		}
		results = append(results, DeepResult{
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
	return results, nil
}

func estimateEdgeBps(snapshot contracts.Snapshot) int {
	spread := snapshot.Microstructure60s.SpreadCurrentBps
	fees := snapshot.CostInputs.MakerFeeBps + snapshot.CostInputs.TakerFeeBps
	slippage := snapshot.CostInputs.SlippageEntryMakerBps + snapshot.CostInputs.SlippageExitTakerBps
	return snapshot.Volatility.ATR14_5mBps - (spread + fees + slippage)
}

func regimeComponent(regime contracts.RegimeSnapshot) float64 {
	if regime.Label == "TREND" {
		return clamp01(float64(regime.TrendScoreX10000) / 10000.0)
	}
	if regime.Label == "RANGE" {
		return clamp01(float64(regime.RangeScoreX10000) / 10000.0)
	}
	return 0
}

func microstructureComponent(micro contracts.Microstructure60s, maxSpread int, minImbalance int) float64 {
	spreadScore := 1.0 - (float64(micro.SpreadCurrentBps) / float64(maxSpread))
	if spreadScore < 0 {
		spreadScore = 0
	}
	imbalanceDen := 10000 - minImbalance
	imbalanceScore := 0.0
	if imbalanceDen > 0 {
		imbalanceScore = clamp01(float64(micro.BidAskImbalanceP50_10sX10000-minImbalance) / float64(imbalanceDen))
	}
	return clamp01((spreadScore * 0.6) + (imbalanceScore * 0.4))
}

func volatilityComponent(vol contracts.VolatilitySnapshot) float64 {
	return clamp01(float64(vol.ATR14_5mBps) / 200.0)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
