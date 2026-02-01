package selection

import (
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/hash"
)

type thresholdsPayload struct {
	RankMinQuoteVolume24hUSDT int64 `json:"rank_min_quote_volume_24h_usdt"`
	RankMinTrades24h          int   `json:"rank_min_trades_24h"`
	RankMinPriceChangeBps     int   `json:"rank_min_price_change_bps"`
	DeepMinEdgeBps            int   `json:"deep_min_edge_bps"`
	DeepMaxSpreadBps          int   `json:"deep_max_spread_bps"`
	DeepMinImbalanceX10000    int   `json:"deep_min_imbalance_x10000"`
}

func ThresholdsHash(cfg config.Config) (string, error) {
	payload := thresholdsPayload{
		RankMinQuoteVolume24hUSDT: cfg.RankMinQuoteVolume24hUSDT,
		RankMinTrades24h:          cfg.RankMinTrades24h,
		RankMinPriceChangeBps:     cfg.RankMinPriceChangeBps,
		DeepMinEdgeBps:            cfg.DeepMinEdgeBps,
		DeepMaxSpreadBps:          cfg.DeepMaxSpreadBps,
		DeepMinImbalanceX10000:    cfg.DeepMinImbalanceX10000,
	}
	return hash.CanonicalHash(payload)
}

