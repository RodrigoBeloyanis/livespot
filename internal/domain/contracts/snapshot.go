package contracts

import (
	"fmt"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/hash"
)

type Snapshot struct {
	Symbol            string                 `json:"symbol"`
	Regime            RegimeSnapshot         `json:"regime"`
	Microstructure60s Microstructure60s      `json:"microstructure_60s"`
	Volatility        VolatilitySnapshot     `json:"volatility"`
	Prices            PricesSnapshot         `json:"prices"`
	Candles5m         []Candle               `json:"candles_5m"`
	CostInputs        CostInputs             `json:"cost_inputs"`
	Market24h         Market24hSnapshot      `json:"market_24h"`
	HealthFlags       HealthFlagsSnapshot    `json:"health_flags"`
	ReturnsSeries     ReturnsSeries          `json:"returns_series"`
	ConfigReference   ConfigurationReference `json:"configuration_reference"`
	Metadata          SnapshotMetadata       `json:"metadata"`
}

type RegimeSnapshot struct {
	Label            string `json:"label"`
	TrendScoreX10000 int    `json:"trend_score_x10000"`
	RangeScoreX10000 int    `json:"range_score_x10000"`
}

type Microstructure60s struct {
	SpreadBpsP50_60s             int `json:"spread_bps_p50_60s"`
	SpreadBpsP90_60s             int `json:"spread_bps_p90_60s"`
	SpreadCurrentBps             int `json:"spread_current_bps"`
	DeltaSpreadBpsP90_10s        int `json:"delta_spread_bps_p90_10s"`
	BidAskImbalanceP50_10sX10000 int `json:"bid_ask_imbalance_p50_10s_x10000"`
	OutOfOrderDrops              int `json:"out_of_order_drops"`
}

type VolatilitySnapshot struct {
	ATR14_5mBps  int `json:"atr14_5m_bps"`
	ATR14_15mBps int `json:"atr14_15m_bps"`
}

type PricesSnapshot struct {
	BestBid   string `json:"best_bid"`
	BestAsk   string `json:"best_ask"`
	MidPrice  string `json:"mid_price"`
	LastPrice string `json:"last_price"`
}

type Candle struct {
	TsMs   int64  `json:"ts_ms"`
	Open   string `json:"open"`
	High   string `json:"high"`
	Low    string `json:"low"`
	Close  string `json:"close"`
	Volume string `json:"volume"`
}

type CostInputs struct {
	MakerFeeBps           int `json:"maker_fee_bps"`
	TakerFeeBps           int `json:"taker_fee_bps"`
	SlippageEntryMakerBps int `json:"slippage_est_entry_maker_bps"`
	SlippageEntryTakerBps int `json:"slippage_est_entry_taker_bps"`
	SlippageExitTakerBps  int `json:"slippage_est_exit_taker_bps"`
}

type Market24hSnapshot struct {
	QuoteVolume24hUSDT string `json:"quote_volume_24h_usdt"`
	Trades24h          int    `json:"trades_24h"`
	PriceChange24hBps  int    `json:"price_change_24h_bps"`
	SourceTsMs         int64  `json:"source_ts_ms"`
}

type HealthFlagsSnapshot struct {
	FiltersOK                bool   `json:"filters_ok"`
	WSOK                     bool   `json:"ws_ok"`
	RecentRejectsWindowCount int    `json:"recent_rejects_window_count"`
	QuarantinedUntilMs       int64  `json:"quarantined_until_ms"`
	SymbolStatus             string `json:"symbol_status"`
}

type ReturnsSeries struct {
	Timeframe    string  `json:"timeframe"`
	WindowPoints int     `json:"window_points"`
	LogReturnBps []int32 `json:"log_return_bps"`
	MissingCount int     `json:"missing_count"`
	ComputedTsMs int64   `json:"computed_ts_ms"`
}

type ConfigurationReference struct {
	ConfigHash         string `json:"config_hash"`
	ThresholdsHash     string `json:"thresholds_hash"`
	CycleConfigVersion string `json:"cycle_config_version"`
	FiltersHash        string `json:"filters_hash"`
}

type SnapshotMetadata struct {
	SnapshotID      string       `json:"snapshot_id"`
	CreatedTsMs     int64        `json:"created_ts_ms"`
	ExchangeTimeMs  int64        `json:"exchange_time_ms"`
	LocalReceivedMs int64        `json:"local_received_ms"`
	SourceHashes    SourceHashes `json:"source_hashes"`
	SnapshotHash    string       `json:"snapshot_hash"`
}

type SourceHashes struct {
	CandlesHash string `json:"candles_hash"`
	BookHash    string `json:"book_hash"`
	TickerHash  string `json:"ticker_hash"`
}

type SnapshotHashPayload struct {
	Symbol         string             `json:"symbol"`
	SymbolStatus   string             `json:"symbol_status"`
	FiltersHash    string             `json:"filters_hash"`
	ConfigHash     string             `json:"config_hash"`
	ThresholdsHash string             `json:"thresholds_hash"`
	Regime         RegimeSnapshot     `json:"regime"`
	Microstructure Microstructure60s  `json:"microstructure_60s"`
	Volatility     VolatilitySnapshot `json:"volatility"`
	Prices         PricesSnapshot     `json:"prices"`
	CostInputs     CostInputs         `json:"cost_inputs"`
	ReturnsSeries  ReturnsSeries      `json:"returns_series"`
}

func (s Snapshot) Validate() error {
	if s.Symbol == "" {
		return fmt.Errorf("snapshot symbol missing")
	}
	if s.Regime.Label != "TREND" && s.Regime.Label != "RANGE" && s.Regime.Label != "UNCLEAR" {
		return fmt.Errorf("snapshot regime label invalid")
	}
	if s.Regime.TrendScoreX10000 < 0 || s.Regime.TrendScoreX10000 > 10000 {
		return fmt.Errorf("snapshot trend_score_x10000 invalid")
	}
	if s.Regime.RangeScoreX10000 < 0 || s.Regime.RangeScoreX10000 > 10000 {
		return fmt.Errorf("snapshot range_score_x10000 invalid")
	}
	if s.Microstructure60s.BidAskImbalanceP50_10sX10000 < 0 || s.Microstructure60s.BidAskImbalanceP50_10sX10000 > 10000 {
		return fmt.Errorf("snapshot imbalance invalid")
	}
	if s.Microstructure60s.OutOfOrderDrops < 0 {
		return fmt.Errorf("snapshot out_of_order_drops invalid")
	}
	if s.Volatility.ATR14_5mBps <= 0 {
		return fmt.Errorf("snapshot atr14_5m_bps invalid")
	}
	if !isDecimalString(s.Prices.BestBid) || !isDecimalString(s.Prices.BestAsk) || !isDecimalString(s.Prices.MidPrice) || !isDecimalString(s.Prices.LastPrice) {
		return fmt.Errorf("snapshot prices invalid")
	}
	cmp, err := compareDecimal(s.Prices.BestBid, s.Prices.BestAsk)
	if err != nil {
		return fmt.Errorf("snapshot price compare invalid")
	}
	if cmp > 0 {
		return fmt.Errorf("snapshot best_bid > best_ask")
	}
	if len(s.Candles5m) != 40 {
		return fmt.Errorf("snapshot candles_5m length invalid")
	}
	for i, c := range s.Candles5m {
		if c.TsMs <= 0 {
			return fmt.Errorf("snapshot candle ts_ms invalid")
		}
		if !isDecimalString(c.Open) || !isDecimalString(c.High) || !isDecimalString(c.Low) || !isDecimalString(c.Close) || !isDecimalString(c.Volume) {
			return fmt.Errorf("snapshot candle decimals invalid")
		}
		cmpOL, err := compareDecimal(c.Low, c.Open)
		if err != nil {
			return fmt.Errorf("snapshot candle low/open invalid")
		}
		cmpCL, err := compareDecimal(c.Low, c.Close)
		if err != nil {
			return fmt.Errorf("snapshot candle low/close invalid")
		}
		cmpOH, err := compareDecimal(c.Open, c.High)
		if err != nil {
			return fmt.Errorf("snapshot candle open/high invalid")
		}
		cmpCH, err := compareDecimal(c.Close, c.High)
		if err != nil {
			return fmt.Errorf("snapshot candle close/high invalid")
		}
		if cmpOL > 0 || cmpCL > 0 || cmpOH > 0 || cmpCH > 0 {
			return fmt.Errorf("snapshot candle bounds invalid")
		}
		if i > 0 && c.TsMs <= s.Candles5m[i-1].TsMs {
			return fmt.Errorf("snapshot candles not ascending")
		}
	}
	if s.CostInputs.MakerFeeBps < 0 || s.CostInputs.TakerFeeBps < 0 || s.CostInputs.SlippageEntryMakerBps < 0 || s.CostInputs.SlippageEntryTakerBps < 0 || s.CostInputs.SlippageExitTakerBps < 0 {
		return fmt.Errorf("snapshot cost_inputs invalid")
	}
	if s.Metadata.SnapshotID == "" {
		return fmt.Errorf("snapshot snapshot_id missing")
	}
	if s.Metadata.CreatedTsMs <= 0 || s.Metadata.ExchangeTimeMs <= 0 || s.Metadata.LocalReceivedMs <= 0 {
		return fmt.Errorf("snapshot metadata timestamps invalid")
	}
	if s.Metadata.SnapshotHash != "" && !isLowerHex64(s.Metadata.SnapshotHash) {
		return fmt.Errorf("snapshot metadata snapshot_hash invalid")
	}
	if s.ConfigReference.ConfigHash == "" || s.ConfigReference.ThresholdsHash == "" || s.ConfigReference.FiltersHash == "" {
		return fmt.Errorf("snapshot config hashes missing")
	}
	if !isLowerHex64(s.ConfigReference.ConfigHash) || !isLowerHex64(s.ConfigReference.ThresholdsHash) || !isLowerHex64(s.ConfigReference.FiltersHash) {
		return fmt.Errorf("snapshot config hashes invalid")
	}
	if s.ReturnsSeries.Timeframe == "" || s.ReturnsSeries.WindowPoints <= 0 {
		return fmt.Errorf("snapshot returns_series invalid")
	}
	if len(s.ReturnsSeries.LogReturnBps) != s.ReturnsSeries.WindowPoints {
		return fmt.Errorf("snapshot returns_series length invalid")
	}
	return nil
}

func (s Snapshot) HashPayload() (SnapshotHashPayload, error) {
	if s.Symbol == "" {
		return SnapshotHashPayload{}, fmt.Errorf("snapshot symbol missing")
	}
	if s.ConfigReference.ConfigHash == "" || s.ConfigReference.ThresholdsHash == "" || s.ConfigReference.FiltersHash == "" {
		return SnapshotHashPayload{}, fmt.Errorf("snapshot config hashes missing")
	}
	if s.HealthFlags.SymbolStatus == "" {
		return SnapshotHashPayload{}, fmt.Errorf("snapshot symbol_status missing")
	}
	if s.ReturnsSeries.Timeframe == "" || s.ReturnsSeries.WindowPoints <= 0 {
		return SnapshotHashPayload{}, fmt.Errorf("snapshot returns_series missing")
	}
	payload := SnapshotHashPayload{
		Symbol:         s.Symbol,
		SymbolStatus:   s.HealthFlags.SymbolStatus,
		FiltersHash:    s.ConfigReference.FiltersHash,
		ConfigHash:     s.ConfigReference.ConfigHash,
		ThresholdsHash: s.ConfigReference.ThresholdsHash,
		Regime:         s.Regime,
		Microstructure: s.Microstructure60s,
		Volatility:     s.Volatility,
		Prices:         s.Prices,
		CostInputs:     s.CostInputs,
		ReturnsSeries:  s.ReturnsSeries,
	}
	return payload, nil
}

func (s Snapshot) Hash() (string, error) {
	payload, err := s.HashPayload()
	if err != nil {
		return "", err
	}
	return hash.CanonicalHash(payload)
}
