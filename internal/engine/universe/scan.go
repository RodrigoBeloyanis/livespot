package universe

import (
	"fmt"
	"math/big"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/selection"
)

type ScanResult struct {
	Symbol      string
	Eligible    bool
	Reasons     []reasoncodes.ReasonCode
	ScoreX10000 int
	ConfigHash  string
}

func Scan(cfg config.Config, snapshots []contracts.Snapshot) ([]ScanResult, error) {
	configHash, err := selection.ConfigHash(cfg)
	if err != nil {
		return nil, err
	}
	results := make([]ScanResult, 0, len(snapshots))
	for _, snapshot := range snapshots {
		result := ScanResult{
			Symbol:      snapshot.Symbol,
			Eligible:    true,
			Reasons:     []reasoncodes.ReasonCode{},
			ScoreX10000: 0,
			ConfigHash:  configHash,
		}
		if snapshot.Symbol == "" {
			result.Eligible = false
			result.Reasons = append(result.Reasons, reasoncodes.STRAT_INPUT_INVALID)
			results = append(results, result)
			continue
		}
		if snapshot.HealthFlags.SymbolStatus != "TRADING" {
			result.Eligible = false
			result.Reasons = append(result.Reasons, reasoncodes.SYMBOL_QUARANTINED)
		}
		if !snapshot.HealthFlags.FiltersOK || !snapshot.HealthFlags.WSOK {
			result.Eligible = false
			result.Reasons = append(result.Reasons, reasoncodes.STRAT_INPUT_INVALID)
		}
		if snapshot.HealthFlags.QuarantinedUntilMs > 0 {
			result.Eligible = false
			result.Reasons = append(result.Reasons, reasoncodes.SYMBOL_QUARANTINED)
		}
		if snapshot.Market24h.QuoteVolume24hUSDT == "" {
			result.Eligible = false
			result.Reasons = append(result.Reasons, reasoncodes.STRAT_MISSING_FIELD)
		} else {
			quoteVolume, err := parseDecimalFloat(snapshot.Market24h.QuoteVolume24hUSDT)
			if err != nil {
				return nil, fmt.Errorf("quote volume parse: %w", err)
			}
			if quoteVolume < float64(cfg.RankMinQuoteVolume24hUSDT) {
				result.Eligible = false
			}
		}
		if snapshot.Market24h.Trades24h < cfg.RankMinTrades24h {
			result.Eligible = false
		}
		if snapshot.Market24h.PriceChange24hBps < cfg.RankMinPriceChangeBps {
			result.Eligible = false
		}
		results = append(results, result)
	}
	return results, nil
}

func parseDecimalFloat(value string) (float64, error) {
	r := new(big.Rat)
	if _, ok := r.SetString(value); !ok {
		return 0, fmt.Errorf("invalid decimal string")
	}
	f, _ := r.Float64()
	return f, nil
}
