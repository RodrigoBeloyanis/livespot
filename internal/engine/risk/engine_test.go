package risk

import (
	"testing"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"
)

func TestEvaluateAllowsBaseline(t *testing.T) {
	cfg := config.Default()
	input := baseInput()
	verdict, err := Evaluate(cfg, input)
	if err != nil {
		t.Fatalf("evaluate failed: %v", err)
	}
	if verdict.Verdict != contracts.RiskAllow {
		t.Fatalf("expected allow verdict")
	}
	if len(verdict.Reasons) != 0 {
		t.Fatalf("expected no reasons")
	}
}

func TestEvaluateBlocksEdgeBelowMin(t *testing.T) {
	cfg := config.Default()
	input := baseInput()
	input.Decision.EdgeBpsExpected = 10
	verdict, _ := Evaluate(cfg, input)
	assertReason(t, verdict, reasoncodes.STRAT_EDGE_BELOW_MIN)
}

func TestEvaluateBlocksInvalidCosts(t *testing.T) {
	cfg := config.Default()
	input := baseInput()
	input.Snapshot.CostInputs.MakerFeeBps = -1
	verdict, _ := Evaluate(cfg, input)
	assertReason(t, verdict, reasoncodes.STRAT_INPUT_INVALID)
}

func TestEvaluateBlocksCooldown(t *testing.T) {
	cfg := config.Default()
	input := baseInput()
	input.CooldownUntilMs = input.NowMs + 1000
	verdict, _ := Evaluate(cfg, input)
	assertReason(t, verdict, reasoncodes.RISK_COOLDOWN_ACTIVE)
}

func TestEvaluateBlocksMaxTradesWindow(t *testing.T) {
	cfg := config.Default()
	input := baseInput()
	input.TradesWindowCount = cfg.RiskMaxTradesPerWindow
	verdict, _ := Evaluate(cfg, input)
	assertReason(t, verdict, reasoncodes.RISK_MAX_TRADES_WINDOW)
}

func TestEvaluateBlocksLossStreak(t *testing.T) {
	cfg := config.Default()
	input := baseInput()
	input.ConsecutiveLosses = cfg.RiskMaxConsecutiveLosses
	verdict, _ := Evaluate(cfg, input)
	assertReason(t, verdict, reasoncodes.RISK_SYMBOL_LOSS_STREAK)
}

func TestEvaluateBlocksLatency(t *testing.T) {
	cfg := config.Default()
	input := baseInput()
	input.WSLatencyMs = cfg.RiskWSLatencyThresholdMs
	verdict, _ := Evaluate(cfg, input)
	assertReason(t, verdict, reasoncodes.WS_OOO_EVENT)
}

func TestEvaluateBlocksOpenOrders(t *testing.T) {
	cfg := config.Default()
	input := baseInput()
	input.OpenOrdersSymbol = cfg.RiskMaxOpenOrdersPerSymbol
	verdict, _ := Evaluate(cfg, input)
	assertReason(t, verdict, reasoncodes.RISK_MAX_OPEN_ORDERS)
}

func TestEvaluateBlocksTradesPerDay(t *testing.T) {
	cfg := config.Default()
	input := baseInput()
	input.TradesToday = cfg.RiskMaxTradesPerDay
	verdict, _ := Evaluate(cfg, input)
	assertReason(t, verdict, reasoncodes.RISK_MAX_TRADES_DAY)
}

func TestEvaluateBlocksDailyLoss(t *testing.T) {
	cfg := config.Default()
	input := baseInput()
	input.RealizedPnLUSDT = "-600.00"
	verdict, _ := Evaluate(cfg, input)
	assertReason(t, verdict, reasoncodes.RISK_DAILY_LOSS_LIMIT)
}

func TestEvaluateBlocksDrawdown(t *testing.T) {
	cfg := config.Default()
	input := baseInput()
	input.RealizedPnLUSDT = "-600.00"
	input.EquityPeakUSDT = "10000.00"
	input.EquityStartUSDT = "10000.00"
	verdict, _ := Evaluate(cfg, input)
	assertReason(t, verdict, reasoncodes.RISK_DRAWDOWN_LIMIT)
}

func TestEvaluateBlocksExposureLimit(t *testing.T) {
	cfg := config.Default()
	input := baseInput()
	input.ExposureSymbolUSDT = "190.00"
	verdict, _ := Evaluate(cfg, input)
	assertReason(t, verdict, reasoncodes.RISK_EXPOSURE_LIMIT)
}

func TestEvaluateBlocksPositionPolicy(t *testing.T) {
	cfg := config.Default()
	input := baseInput()
	input.HasOpenPosition = true
	verdict, _ := Evaluate(cfg, input)
	assertReason(t, verdict, reasoncodes.RISK_POSITION_ALREADY_OPEN)
}

func TestEvaluateBlocksPendingEntry(t *testing.T) {
	cfg := config.Default()
	input := baseInput()
	input.HasPendingEntry = true
	verdict, _ := Evaluate(cfg, input)
	assertReason(t, verdict, reasoncodes.RISK_ENTRY_ALREADY_PENDING)
}

func TestEvaluateBlocksQuarantine(t *testing.T) {
	cfg := config.Default()
	input := baseInput()
	input.Snapshot.HealthFlags.QuarantinedUntilMs = input.NowMs + 1000
	verdict, _ := Evaluate(cfg, input)
	assertReason(t, verdict, reasoncodes.SYMBOL_QUARANTINED)
}

func TestEvaluateBlocksChurnLimit(t *testing.T) {
	cfg := config.Default()
	input := baseInput()
	input.CancelReplaceCount10s = cfg.RiskChurnMaxCancelReplace10s
	verdict, _ := Evaluate(cfg, input)
	assertReason(t, verdict, reasoncodes.RISK_CANCEL_REPLACE_LIMIT_HIT)
}

func TestEvaluateBlocksUnfilledOrderCritical(t *testing.T) {
	cfg := config.Default()
	input := baseInput()
	input.UnfilledOrderCountPct = cfg.RiskChurnUnfilledOrderCriticalPct
	verdict, _ := Evaluate(cfg, input)
	assertReason(t, verdict, reasoncodes.RISK_UNFILLED_ORDER_COUNT_RISK)
}

func TestEvaluateBlocksBudgetInsufficient(t *testing.T) {
	cfg := config.Default()
	input := baseInput()
	input.FreeBalanceUSDT = "5.00"
	verdict, _ := Evaluate(cfg, input)
	assertReason(t, verdict, reasoncodes.RISK_INSUFFICIENT_FREE_BALANCE)
}

func TestEvaluateBlocksMinNotional(t *testing.T) {
	cfg := config.Default()
	input := baseInput()
	input.Decision.EntryPlan.Qty = "0.05"
	verdict, _ := Evaluate(cfg, input)
	assertReason(t, verdict, reasoncodes.PROTECTION_INVALID_MIN_NOTIONAL)
}

func baseInput() Input {
	return Input{
		NowMs:                 1706700000000,
		Snapshot:              baseSnapshot(),
		Decision:              baseDecision(),
		ExposureSymbolUSDT:    "0.00",
		ExposureTotalUSDT:     "0.00",
		OpenOrdersSymbol:      0,
		OpenOrdersTotal:       0,
		TradesToday:           0,
		TradesWindowCount:     0,
		CooldownUntilMs:       0,
		ConsecutiveLosses:     0,
		WSLatencyMs:           0,
		HasOpenPosition:       false,
		HasPendingEntry:       false,
		HasPendingOCO:         false,
		RealizedPnLUSDT:       "0.00",
		UnrealizedPnLUSDT:     "0.00",
		EquityPeakUSDT:        "10000.00",
		EquityStartUSDT:       "10000.00",
		FreeBalanceUSDT:       "10000.00",
		LockedBalanceUSDT:     "0.00",
		PendingReserveUSDT:    "0.00",
		UnfilledOrderCountPct: 0,
		CancelReplaceCount10s: 0,
		CancelCount10s:        0,
		NewOrdersCount10s:     0,
	}
}

func baseSnapshot() contracts.Snapshot {
	candles := make([]contracts.Candle, 40)
	baseTs := int64(1706700000000)
	for i := 0; i < 40; i++ {
		candles[i] = contracts.Candle{
			TsMs:   baseTs + int64(i*300000),
			Open:   "100.00",
			High:   "101.00",
			Low:    "99.00",
			Close:  "100.50",
			Volume: "1.00000000",
		}
	}
	returns := make([]int32, 72)
	return contracts.Snapshot{
		Symbol: "BTCUSDT",
		Regime: contracts.RegimeSnapshot{
			Label:            "TREND",
			TrendScoreX10000: 7000,
			RangeScoreX10000: 3000,
		},
		Microstructure60s: contracts.Microstructure60s{
			SpreadBpsP50_60s:             10,
			SpreadBpsP90_60s:             20,
			SpreadCurrentBps:             12,
			DeltaSpreadBpsP90_10s:        2,
			BidAskImbalanceP50_10sX10000: 6000,
			OutOfOrderDrops:              0,
		},
		Volatility: contracts.VolatilitySnapshot{
			ATR14_5mBps:  45,
			ATR14_15mBps: 0,
		},
		Prices: contracts.PricesSnapshot{
			BestBid:   "100.00",
			BestAsk:   "100.10",
			MidPrice:  "100.05",
			LastPrice: "100.02",
		},
		Candles5m: candles,
		CostInputs: contracts.CostInputs{
			MakerFeeBps:           2,
			TakerFeeBps:           4,
			SlippageEntryMakerBps: 1,
			SlippageEntryTakerBps: 3,
			SlippageExitTakerBps:  2,
		},
		Market24h: contracts.Market24hSnapshot{
			QuoteVolume24hUSDT: "100000.00",
			Trades24h:          1000,
			PriceChange24hBps:  100,
			SourceTsMs:         baseTs,
		},
		HealthFlags: contracts.HealthFlagsSnapshot{
			FiltersOK:                true,
			WSOK:                     true,
			RecentRejectsWindowCount: 0,
			QuarantinedUntilMs:       0,
			SymbolStatus:             "TRADING",
		},
		ReturnsSeries: contracts.ReturnsSeries{
			Timeframe:    "5m",
			WindowPoints: 72,
			LogReturnBps: returns,
			MissingCount: 0,
			ComputedTsMs: baseTs,
		},
		ConfigReference: contracts.ConfigurationReference{
			ConfigHash:         "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			ThresholdsHash:     "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			CycleConfigVersion: "20260131_1",
			FiltersHash:        "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		},
		Metadata: contracts.SnapshotMetadata{
			SnapshotID:      "snap_btc_1",
			CreatedTsMs:     baseTs,
			ExchangeTimeMs:  baseTs,
			LocalReceivedMs: baseTs + 10,
			SourceHashes: contracts.SourceHashes{
				CandlesHash: "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
				BookHash:    "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
				TickerHash:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
		},
	}
}

func baseDecision() contracts.Decision {
	return contracts.Decision{
		Mode:   "LIVE",
		TsMs:   1706700000000,
		Symbol: "BTCUSDT",
		Side:   contracts.SideBuy,
		Intent: contracts.IntentEntry,
		EntryPlan: &contracts.EntryPlan{
			Kind:         contracts.EntryMakerFirst,
			DesiredPrice: "100.00",
			LimitPrice:   "100.00",
			Qty:          "0.20",
			TimeInForce:  contracts.TIFGTC,
			TTLMS:        30000,
			RepriceMS:    30000,
			MaxReprices:  2,
			Fallback: contracts.FallbackPlan{
				Enabled:        true,
				Kind:           contracts.FallbackIOCLimit,
				MaxSlippageBps: 25,
				DeadlineMS:     90000,
			},
			ClientOrderID: "X_0000000000000000000000000000000000",
		},
		ExitPlan: &contracts.ExitPlan{
			TPPrice:              "110.00",
			SLPrice:              "95.00",
			ProtectionKind:       contracts.ProtectionOCO,
			TrailingMode:         contracts.TrailingOff,
			TrailingTriggerPrice: "0",
			TrailingDeltaBips:    0,
			ClientOrderIDTP:      "X_1111111111111111111111111111111111",
			ClientOrderIDSL:      "X_2222222222222222222222222222222222",
		},
		EdgeScoreX10000: 4500,
		EdgeBpsExpected: 25,
		Reasons:         []reasoncodes.ReasonCode{reasoncodes.STRAT_OK},
		SnapshotID:      "snap_btc_1",
		DecisionID:      "dec_btc_1",
		CycleID:         "cyc_1",
		Stage:           observability.STRATEGY_PROPOSE,
		Constraints: contracts.DecisionConstraints{
			TickSize:           "0.01",
			StepSize:           "0.0001",
			MinQty:             "0.001",
			MinNotional:        "10.00",
			PricePrecision:     2,
			QtyPrecision:       4,
			MaxQty:             "100.0000",
			MaxNumOrders:       200,
			MaxAlgoOrders:      5,
			MaxNotional:        "100000.00",
			QuantizationPolicy: contracts.QuantizationEnforced,
		},
	}
}

func assertReason(t *testing.T, verdict contracts.RiskVerdict, reason reasoncodes.ReasonCode) {
	t.Helper()
	if verdict.Verdict != contracts.RiskBlock {
		t.Fatalf("expected block verdict")
	}
	for _, r := range verdict.Reasons {
		if r == reason {
			return
		}
	}
	t.Fatalf("missing reason %s", reason)
}
