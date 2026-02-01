package selection

import (
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/hash"
)

type rankConfigPayload struct {
	TopNSize                  int     `json:"topn_size"`
	TopKSize                  int     `json:"topk_size"`
	RankWeightLiquidity       float64 `json:"rank_weight_liquidity"`
	RankWeightMomentum        float64 `json:"rank_weight_momentum"`
	RankWeightSpread          float64 `json:"rank_weight_spread"`
	RankMinQuoteVolume24hUSDT int64   `json:"rank_min_quote_volume_24h_usdt"`
	RankMinTrades24h          int     `json:"rank_min_trades_24h"`
	RankMinPriceChangeBps     int     `json:"rank_min_price_change_bps"`
	DeepWeightEdge            float64 `json:"deep_weight_edge"`
	DeepWeightRegime          float64 `json:"deep_weight_regime"`
	DeepWeightMicrostructure  float64 `json:"deep_weight_microstructure"`
	DeepWeightVolatility      float64 `json:"deep_weight_volatility"`
	DeepMinEdgeBps            int     `json:"deep_min_edge_bps"`
	DeepMaxSpreadBps          int     `json:"deep_max_spread_bps"`
	DeepMinImbalanceX10000    int     `json:"deep_min_imbalance_x10000"`
	CorrMaxX10000             int     `json:"corr_max_x10000"`
	CorrWindowPoints          int     `json:"corr_window_points"`
	ChurnGuardEnabled         bool    `json:"churn_guard_enabled"`
	ChurnGuardMinCycles       int     `json:"churn_guard_min_cycles"`
	ChurnGuardMinScoreDelta   int     `json:"churn_guard_min_score_delta_x10000"`
}

func ConfigHash(cfg config.Config) (string, error) {
	payload := rankConfigPayload{
		TopNSize:                  cfg.TopNSize,
		TopKSize:                  cfg.TopKSize,
		RankWeightLiquidity:       cfg.RankWeightLiquidity,
		RankWeightMomentum:        cfg.RankWeightMomentum,
		RankWeightSpread:          cfg.RankWeightSpread,
		RankMinQuoteVolume24hUSDT: cfg.RankMinQuoteVolume24hUSDT,
		RankMinTrades24h:          cfg.RankMinTrades24h,
		RankMinPriceChangeBps:     cfg.RankMinPriceChangeBps,
		DeepWeightEdge:            cfg.DeepWeightEdge,
		DeepWeightRegime:          cfg.DeepWeightRegime,
		DeepWeightMicrostructure:  cfg.DeepWeightMicrostructure,
		DeepWeightVolatility:      cfg.DeepWeightVolatility,
		DeepMinEdgeBps:            cfg.DeepMinEdgeBps,
		DeepMaxSpreadBps:          cfg.DeepMaxSpreadBps,
		DeepMinImbalanceX10000:    cfg.DeepMinImbalanceX10000,
		CorrMaxX10000:             cfg.CorrMaxX10000,
		CorrWindowPoints:          cfg.CorrWindowPoints,
		ChurnGuardEnabled:         cfg.ChurnGuardEnabled,
		ChurnGuardMinCycles:       cfg.ChurnGuardMinCycles,
		ChurnGuardMinScoreDelta:   cfg.ChurnGuardMinScoreDeltaX10000,
	}
	return hash.CanonicalHash(payload)
}
