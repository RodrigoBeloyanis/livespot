package risk

import (
	"fmt"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
)

type AdaptiveResult struct {
	AdjustedMinEdgeBps int
	MultiplierX10000   int
}

func ComputeAdaptiveMinEdgeBps(baseMinEdgeBps int, snapshot contracts.Snapshot, cfg config.Config) (AdaptiveResult, error) {
	multiplier := 10000
	spreadThreshold := roundDiv(snapshot.Microstructure60s.SpreadBpsP50_60s*cfg.RiskAdaptiveSpreadFactorX10000, 10000)
	if snapshot.Microstructure60s.SpreadCurrentBps > spreadThreshold {
		multiplier = maxInt(multiplier, 12000)
	}
	atrThreshold := roundDiv(cfg.RiskAdaptiveNormalATR5mBps*cfg.RiskAdaptiveVolatilityFactorX10000, 10000)
	if snapshot.Volatility.ATR14_5mBps > atrThreshold {
		multiplier = maxInt(multiplier, 13000)
	}
	if snapshot.Microstructure60s.BidAskImbalanceP50_10sX10000 < cfg.RiskAdaptiveLiquidityFloorX10000 {
		multiplier = maxInt(multiplier, 14000)
	}
	if multiplier > cfg.RiskAdaptiveMaxMultiplierX10000 {
		multiplier = cfg.RiskAdaptiveMaxMultiplierX10000
	}
	adjusted := roundDiv(baseMinEdgeBps*multiplier, 10000)
	return AdaptiveResult{AdjustedMinEdgeBps: adjusted, MultiplierX10000: multiplier}, nil
}

func ValidateCosts(snapshot contracts.Snapshot) error {
	if snapshot.CostInputs.MakerFeeBps < 0 || snapshot.CostInputs.TakerFeeBps < 0 {
		return fmt.Errorf("costs invalid")
	}
	if snapshot.CostInputs.SlippageEntryMakerBps < 0 || snapshot.CostInputs.SlippageEntryTakerBps < 0 || snapshot.CostInputs.SlippageExitTakerBps < 0 {
		return fmt.Errorf("costs invalid")
	}
	if snapshot.Microstructure60s.SpreadCurrentBps < 0 || snapshot.Microstructure60s.DeltaSpreadBpsP90_10s < 0 {
		return fmt.Errorf("costs invalid")
	}
	return nil
}

func roundDiv(a int, b int) int {
	if b <= 0 {
		return 0
	}
	return (a + (b / 2)) / b
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
