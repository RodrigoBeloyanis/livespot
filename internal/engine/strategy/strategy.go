package strategy

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/executor"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/risk"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"
)

func ProposeEntry(cfg config.Config, snapshot contracts.Snapshot, constraints contracts.DecisionConstraints, cycleID string, now time.Time) (contracts.Decision, error) {
	if snapshot.Metadata.SnapshotHash == "" {
		return contracts.Decision{}, fmt.Errorf("snapshot hash missing")
	}
	if err := snapshot.Validate(); err != nil {
		return contracts.Decision{}, err
	}

	regimeComponent, activeRegime, err := selectRegime(cfg, snapshot.Regime)
	if err != nil {
		return contracts.Decision{}, err
	}

	pullbackBps, err := pullbackBps(cfg, snapshot)
	if err != nil {
		return contracts.Decision{}, err
	}
	if pullbackBps < float64(cfg.StrategyPullbackMinBpsEMA20) || pullbackBps > float64(cfg.StrategyPullbackMaxBpsEMA20) {
		return contracts.Decision{}, fmt.Errorf(string(reasoncodes.STRAT_PULLBACK_FAIL))
	}

	volumeRatio, err := volumeRatio(snapshot, cfg.StrategyVolumeRatioWindow5m)
	if err != nil {
		return contracts.Decision{}, err
	}

	if snapshot.Microstructure60s.SpreadCurrentBps <= 0 {
		return contracts.Decision{}, fmt.Errorf(string(reasoncodes.STRAT_INPUT_INVALID))
	}
	if snapshot.Microstructure60s.SpreadCurrentBps > cfg.StrategyMaxSpreadEntryBps {
		return contracts.Decision{}, fmt.Errorf(string(reasoncodes.STRAT_SPREAD_TOO_WIDE))
	}
	if snapshot.Microstructure60s.DeltaSpreadBpsP90_10s > cfg.StrategyMaxDeltaSpreadBps10s {
		return contracts.Decision{}, fmt.Errorf(string(reasoncodes.STRAT_SPREAD_OPENING))
	}
	if snapshot.Microstructure60s.BidAskImbalanceP50_10sX10000 < cfg.StrategyMinImbalanceBuyX10000 {
		return contracts.Decision{}, fmt.Errorf(string(reasoncodes.STRAT_IMBALANCE_AGAINST))
	}
	if volumeRatio < cfg.StrategyMinVolumeRatio5m {
		return contracts.Decision{}, fmt.Errorf(string(reasoncodes.STRAT_VOLUME_LOW))
	}

	microstructComponent := microstructureComponent(snapshot, cfg)
	volumeComponent := clamp01((volumeRatio - cfg.StrategyMinVolumeRatio5m) / (math.Max(2.0, cfg.StrategyMinVolumeRatio5m+0.8) - cfg.StrategyMinVolumeRatio5m))
	pullbackComponent := clamp01((pullbackBps - float64(cfg.StrategyPullbackMinBpsEMA20)) / float64(cfg.StrategyPullbackMaxBpsEMA20-cfg.StrategyPullbackMinBpsEMA20))
	edgeScore := clamp01((cfg.StrategyWeightTrend * regimeComponent) + (cfg.StrategyWeightPullback * pullbackComponent) + (cfg.StrategyWeightMicrostruct * microstructComponent) + (cfg.StrategyWeightVolume * volumeComponent))
	edgeScoreX10000 := int(math.RoundToEven(edgeScore * 10000.0))

	slDistanceBps, tpDistanceBps, trailingDistanceBps := exitDistances(cfg, snapshot, activeRegime)
	if tpDistanceBps <= slDistanceBps || slDistanceBps <= 0 || tpDistanceBps <= 0 {
		return contracts.Decision{}, fmt.Errorf(string(reasoncodes.STRAT_EXIT_INVALID))
	}

	costTotalBps := snapshot.CostInputs.MakerFeeBps + snapshot.CostInputs.TakerFeeBps +
		snapshot.CostInputs.SlippageEntryMakerBps + snapshot.CostInputs.SlippageExitTakerBps +
		snapshot.Microstructure60s.SpreadCurrentBps + maxInt(0, snapshot.Microstructure60s.DeltaSpreadBpsP90_10s)
	edgeBpsExpected := tpDistanceBps - slDistanceBps - costTotalBps
	if edgeBpsExpected < cfg.StrategyMinEdgeBps {
		return contracts.Decision{}, fmt.Errorf(string(reasoncodes.STRAT_EDGE_BELOW_MIN))
	}
	if activeRegime == "RANGE" && edgeBpsExpected < int(float64(cfg.StrategyMinEdgeBps)*1.5) {
		return contracts.Decision{}, fmt.Errorf(string(reasoncodes.STRAT_EDGE_BELOW_MIN))
	}

	desiredPrice := snapshot.Prices.BestBid
	limitPrice, err := executor.QuantizePrice(desiredPrice, constraints.TickSize)
	if err != nil {
		return contracts.Decision{}, err
	}

	slPrice, err := priceFromBps(limitPrice, -slDistanceBps)
	if err != nil {
		return contracts.Decision{}, err
	}
	tpPrice, err := priceFromBps(limitPrice, tpDistanceBps)
	if err != nil {
		return contracts.Decision{}, err
	}
	slPrice, err = executor.QuantizePrice(slPrice, constraints.TickSize)
	if err != nil {
		return contracts.Decision{}, err
	}
	tpPrice, err = executor.QuantizePrice(tpPrice, constraints.TickSize)
	if err != nil {
		return contracts.Decision{}, err
	}

	riskPerTrade, err := normalizeRisk(cfg.RiskPerTradeUSDT)
	if err != nil {
		return contracts.Decision{}, err
	}
	sizeResult, err := risk.CalculatePositionSize(limitPrice, slPrice, riskPerTrade, constraints)
	if err != nil {
		return contracts.Decision{}, err
	}

	entryPlan := &contracts.EntryPlan{
		Kind:         contracts.EntryMakerFirst,
		DesiredPrice: desiredPrice,
		LimitPrice:   limitPrice,
		Qty:          sizeResult.QtyBase,
		TimeInForce:  contracts.TIFGTC,
		TTLMS:        cfg.StrategyEntryMakerTTLSeconds * 1000,
		RepriceMS:    cfg.StrategyEntryMakerTTLSeconds * 1000,
		MaxReprices:  cfg.StrategyEntryMakerRepriceMax,
		Fallback: contracts.FallbackPlan{
			Enabled:        false,
			Kind:           contracts.FallbackIOCLimit,
			MaxSlippageBps: 0,
			DeadlineMS:     0,
		},
		ClientOrderID: "",
	}

	fallbackAllowed := snapshot.Microstructure60s.SpreadCurrentBps <= cfg.StrategyEntryFallbackMaxSpreadBps &&
		snapshot.CostInputs.SlippageEntryTakerBps <= cfg.StrategyEntryMaxSlippageBps &&
		edgeBpsExpected >= cfg.StrategyMinEdgeBpsFallback
	reasons := []reasoncodes.ReasonCode{reasoncodes.STRAT_OK}
	if fallbackAllowed {
		entryPlan.Fallback.Enabled = true
		entryPlan.Fallback.Kind = fallbackKind(cfg.StrategyEntryFallbackKind)
		entryPlan.Fallback.MaxSlippageBps = cfg.StrategyEntryMaxSlippageBps
		entryPlan.Fallback.DeadlineMS = entryPlan.TTLMS * (entryPlan.MaxReprices + 1)
		reasons = append(reasons, reasoncodes.STRAT_FALLBACK_ALLOWED)
		if entryPlan.Fallback.Kind == contracts.FallbackIOCLimit {
			reasons = append(reasons, reasoncodes.STRAT_FALLBACK_IOC)
		} else {
			reasons = append(reasons, reasoncodes.STRAT_FALLBACK_MARKET)
		}
	} else {
		reasons = append(reasons, reasoncodes.STRAT_FALLBACK_BLOCKED)
	}

	trailingMode := contracts.TrailingOff
	trailingTrigger := "0"
	trailingDelta := 0
	if activeRegime == "TREND" && snapshot.Regime.TrendScoreX10000 >= cfg.StrategyExitTrailingTrendMinX10000 &&
		snapshot.Microstructure60s.SpreadCurrentBps <= cfg.StrategyExitTrailingMaxSpreadBps {
		trigger, err := priceFromBps(limitPrice, cfg.StrategyExitTrailingEnableProfitBps)
		if err == nil {
			trailingMode = contracts.TrailingVirtual
			trailingTrigger = trigger
			trailingDelta = trailingDistanceBps
			reasons = append(reasons, reasoncodes.STRAT_TRAILING_ARM_ALLOWED)
		}
	} else {
		reasons = append(reasons, reasoncodes.STRAT_TRAILING_ARM_BLOCKED)
	}

	exitPlan := &contracts.ExitPlan{
		TPPrice:              tpPrice,
		SLPrice:              slPrice,
		ProtectionKind:       contracts.ProtectionOCO,
		TrailingMode:         trailingMode,
		TrailingTriggerPrice: trailingTrigger,
		TrailingDeltaBips:    trailingDelta,
		ClientOrderIDTP:      "",
		ClientOrderIDSL:      "",
	}

	decision := contracts.Decision{
		Mode:            "LIVE",
		TsMs:            now.UnixMilli(),
		Symbol:          snapshot.Symbol,
		Side:            contracts.SideBuy,
		Intent:          contracts.IntentEntry,
		EntryPlan:       entryPlan,
		ExitPlan:        exitPlan,
		EdgeScoreX10000: edgeScoreX10000,
		EdgeBpsExpected: edgeBpsExpected,
		Reasons:         reasons,
		SnapshotID:      snapshot.Metadata.SnapshotID,
		DecisionID:      "",
		CycleID:         cycleID,
		Stage:           observability.STRATEGY_PROPOSE,
		Constraints:     constraints,
		AIGate:          nil,
		RiskVerdict:     nil,
	}

	orderIntentID, err := executor.OrderIntentID(decision, snapshot.Metadata.SnapshotHash)
	if err != nil {
		return contracts.Decision{}, err
	}
	entryPlan.ClientOrderID = executor.ClientOrderID(orderIntentID)
	exitPlan.ClientOrderIDTP = executor.ClientOrderID(orderIntentID + "_TP")
	exitPlan.ClientOrderIDSL = executor.ClientOrderID(orderIntentID + "_SL")

	decisionID, err := decision.Hash(snapshot.Metadata.SnapshotHash)
	if err != nil {
		return contracts.Decision{}, err
	}
	decision.DecisionID = "dec_" + decisionID
	if err := decision.Validate(); err != nil {
		return contracts.Decision{}, err
	}
	return decision, nil
}

func selectRegime(cfg config.Config, regime contracts.RegimeSnapshot) (float64, string, error) {
	if regime.Label == "TREND" && regime.TrendScoreX10000 >= cfg.StrategyTrendThresholdX10000 {
		component := clamp01(float64(regime.TrendScoreX10000-cfg.StrategyTrendThresholdX10000) / float64(10000-cfg.StrategyTrendThresholdX10000))
		return component, "TREND", nil
	}
	if regime.Label == "RANGE" && regime.RangeScoreX10000 >= cfg.StrategyRangeThresholdX10000 {
		component := clamp01(float64(regime.RangeScoreX10000-cfg.StrategyRangeThresholdX10000) / float64(10000-cfg.StrategyRangeThresholdX10000))
		return component * 0.7, "RANGE", nil
	}
	return 0, "NONE", fmt.Errorf(string(reasoncodes.STRAT_REGIME_WEAK))
}

func microstructureComponent(snapshot contracts.Snapshot, cfg config.Config) float64 {
	scoreSpread := clamp01(1.0 - (float64(snapshot.Microstructure60s.SpreadCurrentBps) / float64(cfg.StrategyMaxSpreadEntryBps)))
	scoreDelta := clamp01(1.0 - (float64(snapshot.Microstructure60s.DeltaSpreadBpsP90_10s) / float64(cfg.StrategyMaxDeltaSpreadBps10s)))
	denImb := 10000 - cfg.StrategyMinImbalanceBuyX10000
	scoreImb := 0.0
	if denImb > 0 {
		scoreImb = clamp01(float64(snapshot.Microstructure60s.BidAskImbalanceP50_10sX10000-cfg.StrategyMinImbalanceBuyX10000) / float64(denImb))
	}
	return (scoreSpread * 0.5) + (scoreDelta * 0.25) + (scoreImb * 0.25)
}

func pullbackBps(cfg config.Config, snapshot contracts.Snapshot) (float64, error) {
	ema, err := ema20(snapshot)
	if err != nil {
		return 0, err
	}
	last, ok := new(big.Rat).SetString(snapshot.Prices.LastPrice)
	if !ok {
		return 0, fmt.Errorf("invalid last price")
	}
	if ema.Sign() <= 0 {
		return 0, fmt.Errorf(string(reasoncodes.STRAT_INPUT_INVALID))
	}
	diff := new(big.Rat).Sub(ema, last)
	pullback := new(big.Rat).Mul(new(big.Rat).Quo(diff, ema), big.NewRat(10000, 1))
	val, _ := pullback.Float64()
	return val, nil
}

func ema20(snapshot contracts.Snapshot) (*big.Rat, error) {
	if len(snapshot.Candles5m) < 20 {
		return nil, fmt.Errorf(string(reasoncodes.STRAT_MISSING_FIELD))
	}
	alpha := new(big.Rat).SetFrac(big.NewInt(2), big.NewInt(21))
	oneMinus := new(big.Rat).Sub(big.NewRat(1, 1), alpha)

	sum := new(big.Rat)
	for i := 0; i < 20; i++ {
		closeRat, ok := new(big.Rat).SetString(snapshot.Candles5m[i].Close)
		if !ok {
			return nil, fmt.Errorf(string(reasoncodes.STRAT_INPUT_INVALID))
		}
		sum.Add(sum, closeRat)
	}
	ema := new(big.Rat).Quo(sum, big.NewRat(20, 1))
	for i := 20; i < len(snapshot.Candles5m); i++ {
		closeRat, ok := new(big.Rat).SetString(snapshot.Candles5m[i].Close)
		if !ok {
			return nil, fmt.Errorf(string(reasoncodes.STRAT_INPUT_INVALID))
		}
		ema = new(big.Rat).Add(new(big.Rat).Mul(closeRat, alpha), new(big.Rat).Mul(ema, oneMinus))
	}
	return ema, nil
}

func volumeRatio(snapshot contracts.Snapshot, window int) (float64, error) {
	if len(snapshot.Candles5m) < window {
		return 0, fmt.Errorf(string(reasoncodes.STRAT_MISSING_FIELD))
	}
	baseSum := new(big.Rat)
	for i := len(snapshot.Candles5m) - window; i < len(snapshot.Candles5m)-1; i++ {
		v, ok := new(big.Rat).SetString(snapshot.Candles5m[i].Volume)
		if !ok {
			return 0, fmt.Errorf(string(reasoncodes.STRAT_INPUT_INVALID))
		}
		baseSum.Add(baseSum, v)
	}
	base := new(big.Rat).Quo(baseSum, big.NewRat(int64(window-1), 1))
	if base.Sign() <= 0 {
		return 0, fmt.Errorf(string(reasoncodes.STRAT_INPUT_INVALID))
	}
	last, ok := new(big.Rat).SetString(snapshot.Candles5m[len(snapshot.Candles5m)-1].Volume)
	if !ok {
		return 0, fmt.Errorf(string(reasoncodes.STRAT_INPUT_INVALID))
	}
	ratio := new(big.Rat).Quo(last, base)
	val, _ := ratio.Float64()
	return val, nil
}

func exitDistances(cfg config.Config, snapshot contracts.Snapshot, activeRegime string) (int, int, int) {
	k := cfg.StrategyExitKATRTrend
	m := cfg.StrategyExitMATRTrend
	t := cfg.StrategyExitTrailingTATRTrend
	if activeRegime == "RANGE" {
		k = cfg.StrategyExitKATRRange
		m = cfg.StrategyExitMATRRange
		t = cfg.StrategyExitTrailingTATRRange
	}
	sl := int(math.RoundToEven(k * float64(snapshot.Volatility.ATR14_5mBps)))
	tp := int(math.RoundToEven(m * float64(snapshot.Volatility.ATR14_5mBps)))
	trailing := int(math.RoundToEven(t * float64(snapshot.Volatility.ATR14_5mBps)))
	return sl, tp, trailing
}

func priceFromBps(price string, deltaBps int) (string, error) {
	r, ok := new(big.Rat).SetString(price)
	if !ok {
		return "", fmt.Errorf("invalid price")
	}
	mult := new(big.Rat).Add(big.NewRat(1, 1), new(big.Rat).SetFrac(big.NewInt(int64(deltaBps)), big.NewInt(10000)))
	out := new(big.Rat).Mul(r, mult)
	return out.FloatString(8), nil
}

func normalizeRisk(value string) (string, error) {
	r, ok := new(big.Rat).SetString(value)
	if !ok {
		return "", fmt.Errorf("invalid risk_per_trade_usdt")
	}
	scale := big.NewInt(100)
	intVal := new(big.Int).Quo(new(big.Int).Mul(r.Num(), scale), r.Denom())
	return formatFixed(intVal, 2), nil
}

func formatFixed(value *big.Int, scale int) string {
	s := value.String()
	if scale == 0 {
		return s
	}
	if len(s) <= scale {
		s = strings.Repeat("0", scale-len(s)+1) + s
	}
	idx := len(s) - scale
	return s[:idx] + "." + s[idx:]
}

func fallbackKind(kind string) contracts.FallbackKind {
	if kind == "MARKET_IF_ALLOWED" {
		return contracts.FallbackMarketIfAllowed
	}
	return contracts.FallbackIOCLimit
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

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
