package risk

import (
	"fmt"
	"math/big"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
)

type Input struct {
	NowMs                 int64
	Snapshot              contracts.Snapshot
	Decision              contracts.Decision
	ExposureSymbolUSDT    string
	ExposureTotalUSDT     string
	OpenOrdersSymbol      int
	OpenOrdersTotal       int
	TradesToday           int
	TradesWindowCount     int
	CooldownUntilMs       int64
	ConsecutiveLosses     int
	WSLatencyMs           int
	HasOpenPosition       bool
	HasPendingEntry       bool
	HasPendingOCO         bool
	RealizedPnLUSDT       string
	UnrealizedPnLUSDT     string
	EquityPeakUSDT        string
	EquityStartUSDT       string
	FreeBalanceUSDT       string
	LockedBalanceUSDT     string
	PendingReserveUSDT    string
	UnfilledOrderCountPct int
	CancelReplaceCount10s int
	CancelCount10s        int
	NewOrdersCount10s     int
}

func Evaluate(cfg config.Config, input Input) (contracts.RiskVerdict, error) {
	if input.Decision.Intent != contracts.IntentEntry {
		return allowVerdict(cfg, input.Snapshot), nil
	}
	if err := ValidateCosts(input.Snapshot); err != nil {
		return blockVerdict(cfg, input.Snapshot, []reasoncodes.ReasonCode{reasoncodes.STRAT_INPUT_INVALID}), nil
	}

	adaptive, err := ComputeAdaptiveMinEdgeBps(cfg.StrategyMinEdgeBps, input.Snapshot, cfg)
	if err != nil {
		return blockVerdict(cfg, input.Snapshot, []reasoncodes.ReasonCode{reasoncodes.STRAT_INPUT_INVALID}), nil
	}
	if input.Decision.EdgeBpsExpected < adaptive.AdjustedMinEdgeBps {
		return blockVerdict(cfg, input.Snapshot, []reasoncodes.ReasonCode{reasoncodes.STRAT_EDGE_BELOW_MIN}), nil
	}

	reasons := []reasoncodes.ReasonCode{}
	reasons = append(reasons, positionPolicyReasons(input.HasOpenPosition, input.HasPendingEntry, input.HasPendingOCO)...)
	if reason := quarantineReason(cfg, input.Snapshot, input.NowMs); reason != "" {
		reasons = append(reasons, reason)
	}
	reasons = append(reasons, churnReasons(cfg, input)...)
	reasons = append(reasons, antiOvertradingReasons(cfg, input)...)
	if input.OpenOrdersSymbol >= cfg.RiskMaxOpenOrdersPerSymbol || input.OpenOrdersTotal >= cfg.RiskMaxOpenOrdersTotal {
		reasons = append(reasons, reasoncodes.RISK_MAX_OPEN_ORDERS)
	}
	if input.TradesToday >= cfg.RiskMaxTradesPerDay {
		reasons = append(reasons, reasoncodes.RISK_MAX_TRADES_DAY)
	}
	if hit, err := dailyLossLimit(cfg, input); err != nil {
		return blockVerdict(cfg, input.Snapshot, []reasoncodes.ReasonCode{reasoncodes.STRAT_INPUT_INVALID}), nil
	} else if hit {
		reasons = append(reasons, reasoncodes.RISK_DAILY_LOSS_LIMIT)
	}
	if hit, err := drawdownLimit(cfg, input); err != nil {
		return blockVerdict(cfg, input.Snapshot, []reasoncodes.ReasonCode{reasoncodes.STRAT_INPUT_INVALID}), nil
	} else if hit {
		reasons = append(reasons, reasoncodes.RISK_DRAWDOWN_LIMIT)
	}
	entryNotional, err := entryNotionalUSDT(input.Decision)
	if err != nil {
		return blockVerdict(cfg, input.Snapshot, []reasoncodes.ReasonCode{reasoncodes.STRAT_INPUT_INVALID}), nil
	}
	if hit, err := exposureLimit(cfg, input, entryNotional); err != nil {
		return blockVerdict(cfg, input.Snapshot, []reasoncodes.ReasonCode{reasoncodes.STRAT_INPUT_INVALID}), nil
	} else if hit {
		reasons = append(reasons, reasoncodes.RISK_EXPOSURE_LIMIT)
	}
	if reason, _, err := ValidatePositionSizing(cfg, input.Decision); err != nil {
		return blockVerdict(cfg, input.Snapshot, []reasoncodes.ReasonCode{reasoncodes.STRAT_INPUT_INVALID}), nil
	} else if reason != "" {
		reasons = append(reasons, reason)
	}
	if reason, err := budgetReason(entryNotional, input.FreeBalanceUSDT, input.LockedBalanceUSDT, input.PendingReserveUSDT); err != nil {
		return blockVerdict(cfg, input.Snapshot, []reasoncodes.ReasonCode{reasoncodes.STRAT_INPUT_INVALID}), nil
	} else if reason != "" {
		reasons = append(reasons, reason)
	}

	if len(reasons) > 0 {
		return blockVerdict(cfg, input.Snapshot, reasons), nil
	}
	return allowVerdict(cfg, input.Snapshot), nil
}

func allowVerdict(cfg config.Config, snapshot contracts.Snapshot) contracts.RiskVerdict {
	return contracts.RiskVerdict{
		Verdict:    contracts.RiskAllow,
		Reasons:    []reasoncodes.ReasonCode{},
		RiskLimits: riskLimitsFromConfig(cfg),
		Costs:      costsFromSnapshot(snapshot),
	}
}

func blockVerdict(cfg config.Config, snapshot contracts.Snapshot, reasons []reasoncodes.ReasonCode) contracts.RiskVerdict {
	return contracts.RiskVerdict{
		Verdict:    contracts.RiskBlock,
		Reasons:    reasons,
		RiskLimits: riskLimitsFromConfig(cfg),
		Costs:      costsFromSnapshot(snapshot),
	}
}

func riskLimitsFromConfig(cfg config.Config) contracts.RiskLimitsSnapshot {
	return contracts.RiskLimitsSnapshot{
		MaxExposureSymbolUSDT:   cfg.RiskMaxExposureSymbolUSDT,
		MaxExposureTotalUSDT:    cfg.RiskMaxExposureTotalUSDT,
		MaxDailyLossUSDT:        cfg.RiskMaxDailyLossUSDT,
		MaxDrawdownUSDT:         cfg.RiskMaxDrawdownUSDT,
		MaxOpenOrders:           cfg.RiskMaxOpenOrdersTotal,
		MaxTradesDay:            cfg.RiskMaxTradesPerDay,
		MaxTradesWindow:         cfg.RiskMaxTradesPerWindow,
		TradesWindowSeconds:     cfg.RiskTradesWindowSeconds,
		PostStopCooldownSeconds: cfg.RiskCooldownSeconds,
	}
}

func dailyLossLimit(cfg config.Config, input Input) (bool, error) {
	realized, err := parseDecimal(input.RealizedPnLUSDT)
	if err != nil {
		return false, err
	}
	unrealized, err := parseDecimal(input.UnrealizedPnLUSDT)
	if err != nil {
		return false, err
	}
	limit, err := parseDecimal(cfg.RiskMaxDailyLossUSDT)
	if err != nil {
		return false, err
	}
	total := new(big.Rat).Add(realized, unrealized)
	return total.Cmp(limit) <= 0, nil
}

func drawdownLimit(cfg config.Config, input Input) (bool, error) {
	peak, err := parseDecimal(input.EquityPeakUSDT)
	if err != nil {
		return false, err
	}
	start, err := parseDecimal(input.EquityStartUSDT)
	if err != nil {
		return false, err
	}
	realized, err := parseDecimal(input.RealizedPnLUSDT)
	if err != nil {
		return false, err
	}
	unrealized, err := parseDecimal(input.UnrealizedPnLUSDT)
	if err != nil {
		return false, err
	}
	current := new(big.Rat).Add(start, new(big.Rat).Add(realized, unrealized))
	limit, err := parseDecimal(cfg.RiskMaxDrawdownUSDT)
	if err != nil {
		return false, err
	}
	drawdown := new(big.Rat).Sub(peak, current)
	return drawdown.Cmp(limit) >= 0, nil
}

func exposureLimit(cfg config.Config, input Input, entryNotional *big.Rat) (bool, error) {
	if entryNotional == nil {
		return false, fmt.Errorf("entry notional missing")
	}
	symbolExposure, err := parseDecimal(input.ExposureSymbolUSDT)
	if err != nil {
		return false, err
	}
	totalExposure, err := parseDecimal(input.ExposureTotalUSDT)
	if err != nil {
		return false, err
	}
	maxSymbol, err := parseDecimal(cfg.RiskMaxExposureSymbolUSDT)
	if err != nil {
		return false, err
	}
	maxTotal, err := parseDecimal(cfg.RiskMaxExposureTotalUSDT)
	if err != nil {
		return false, err
	}
	if new(big.Rat).Add(symbolExposure, entryNotional).Cmp(maxSymbol) > 0 {
		return true, nil
	}
	if new(big.Rat).Add(totalExposure, entryNotional).Cmp(maxTotal) > 0 {
		return true, nil
	}
	return false, nil
}

func parseDecimal(value string) (*big.Rat, error) {
	if value == "" {
		return nil, fmt.Errorf("decimal missing")
	}
	r := new(big.Rat)
	if _, ok := r.SetString(value); !ok {
		return nil, fmt.Errorf("invalid decimal")
	}
	return r, nil
}

func multiplyDecimal(a string, b string) (string, error) {
	ra, err := parseDecimal(a)
	if err != nil {
		return "", err
	}
	rb, err := parseDecimal(b)
	if err != nil {
		return "", err
	}
	out := new(big.Rat).Mul(ra, rb)
	return out.FloatString(8), nil
}

func NowUTC() time.Time {
	return time.Now().UTC()
}

func entryNotionalUSDT(decision contracts.Decision) (*big.Rat, error) {
	if decision.EntryPlan == nil {
		return nil, fmt.Errorf("entry plan missing")
	}
	price := decision.EntryPlan.LimitPrice
	if price == "" {
		price = decision.EntryPlan.DesiredPrice
	}
	if price == "" {
		return nil, fmt.Errorf("entry price missing")
	}
	qty, err := parseDecimalStrict(decision.EntryPlan.Qty)
	if err != nil {
		return nil, err
	}
	priceRat, err := parseDecimalStrict(price)
	if err != nil {
		return nil, err
	}
	return new(big.Rat).Mul(qty, priceRat), nil
}

func costsFromSnapshot(snapshot contracts.Snapshot) contracts.CostsSnapshot {
	return contracts.CostsSnapshot{
		MakerFeeBps:           snapshot.CostInputs.MakerFeeBps,
		TakerFeeBps:           snapshot.CostInputs.TakerFeeBps,
		SlippageEntryMakerBps: snapshot.CostInputs.SlippageEntryMakerBps,
		SlippageEntryTakerBps: snapshot.CostInputs.SlippageEntryTakerBps,
		SlippageExitTakerBps:  snapshot.CostInputs.SlippageExitTakerBps,
		SpreadCurrentBps:      snapshot.Microstructure60s.SpreadCurrentBps,
		DeltaSpreadBpsP90_10s: snapshot.Microstructure60s.DeltaSpreadBpsP90_10s,
	}
}
