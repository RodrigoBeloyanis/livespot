package e2e

import (
	"fmt"
	"strings"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"
)

func SampleSnapshot(now time.Time) (contracts.Snapshot, error) {
	candles := make([]contracts.Candle, 0, 40)
	base := now.Add(-time.Duration(40) * time.Minute)
	for i := 0; i < 40; i++ {
		ts := base.Add(time.Duration(i) * time.Minute).UnixMilli()
		candles = append(candles, contracts.Candle{
			TsMs:   ts,
			Open:   "100.0",
			High:   "110.0",
			Low:    "90.0",
			Close:  "105.0",
			Volume: "1000.0",
		})
	}
	returns := make([]int32, 0, 10)
	for i := 0; i < 10; i++ {
		returns = append(returns, int32(i))
	}
	snapshot := contracts.Snapshot{
		Symbol: "BTCUSDT",
		Regime: contracts.RegimeSnapshot{
			Label:            "TREND",
			TrendScoreX10000: 7000,
			RangeScoreX10000: 3000,
		},
		Microstructure60s: contracts.Microstructure60s{
			SpreadBpsP50_60s:             3,
			SpreadBpsP90_60s:             6,
			SpreadCurrentBps:             2,
			DeltaSpreadBpsP90_10s:        1,
			BidAskImbalanceP50_10sX10000: 6000,
			OutOfOrderDrops:              0,
		},
		Volatility: contracts.VolatilitySnapshot{
			ATR14_5mBps:  50,
			ATR14_15mBps: 60,
		},
		Prices: contracts.PricesSnapshot{
			BestBid:   "100.0",
			BestAsk:   "100.1",
			MidPrice:  "100.05",
			LastPrice: "100.0",
		},
		Candles5m: candles,
		CostInputs: contracts.CostInputs{
			MakerFeeBps:           1,
			TakerFeeBps:           2,
			SlippageEntryMakerBps: 1,
			SlippageEntryTakerBps: 2,
			SlippageExitTakerBps:  2,
		},
		Market24h: contracts.Market24hSnapshot{
			QuoteVolume24hUSDT: "10000000",
			Trades24h:          20000,
			PriceChange24hBps:  50,
			SourceTsMs:         now.UnixMilli(),
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
			WindowPoints: len(returns),
			LogReturnBps: returns,
			MissingCount: 0,
			ComputedTsMs: now.UnixMilli(),
		},
		ConfigReference: contracts.ConfigurationReference{
			ConfigHash:         strings.Repeat("a", 64),
			ThresholdsHash:     strings.Repeat("b", 64),
			CycleConfigVersion: "v1",
			FiltersHash:        strings.Repeat("c", 64),
		},
		Metadata: contracts.SnapshotMetadata{
			SnapshotID:      "snap_001",
			CreatedTsMs:     now.UnixMilli(),
			ExchangeTimeMs:  now.UnixMilli(),
			LocalReceivedMs: now.UnixMilli(),
			SourceHashes: contracts.SourceHashes{
				CandlesHash: strings.Repeat("d", 64),
				BookHash:    strings.Repeat("e", 64),
				TickerHash:  strings.Repeat("f", 64),
			},
		},
	}
	hash, err := snapshot.Hash()
	if err != nil {
		return contracts.Snapshot{}, err
	}
	snapshot.Metadata.SnapshotHash = hash
	if err := snapshot.Validate(); err != nil {
		return contracts.Snapshot{}, fmt.Errorf("snapshot invalid: %w", err)
	}
	return snapshot, nil
}

func SampleDecision(now time.Time, snapshot contracts.Snapshot, snapshotHash string, cycleID string) (contracts.Decision, error) {
	decision := contracts.Decision{
		Mode:            "LIVE",
		TsMs:            now.UnixMilli(),
		Symbol:          snapshot.Symbol,
		Side:            contracts.SideBuy,
		Intent:          contracts.IntentEntry,
		EdgeScoreX10000: 6000,
		EdgeBpsExpected: 50,
		Reasons:         []reasoncodes.ReasonCode{reasoncodes.STRAT_OK},
		SnapshotID:      snapshot.Metadata.SnapshotID,
		DecisionID:      "",
		CycleID:         cycleID,
		Stage:           observability.STRATEGY_PROPOSE,
		Constraints: contracts.DecisionConstraints{
			TickSize:           "0.01",
			StepSize:           "0.001",
			MinQty:             "0.001",
			MinNotional:        "5",
			PricePrecision:     2,
			QtyPrecision:       3,
			MaxQty:             "1000",
			MaxNumOrders:       200,
			MaxAlgoOrders:      5,
			MaxNotional:        "100000",
			QuantizationPolicy: contracts.QuantizationEnforced,
		},
		EntryPlan: &contracts.EntryPlan{
			Kind:         contracts.EntryMakerFirst,
			DesiredPrice: "100.0",
			LimitPrice:   "100.0",
			Qty:          "0.01",
			TimeInForce:  contracts.TIFGTC,
			TTLMS:        5000,
			RepriceMS:    1000,
			MaxReprices:  3,
			Fallback: contracts.FallbackPlan{
				Enabled:        true,
				Kind:           contracts.FallbackCancelReplace,
				MaxSlippageBps: 10,
				DeadlineMS:     10000,
			},
			ClientOrderID: strings.Repeat("a", 36),
		},
		ExitPlan: &contracts.ExitPlan{
			TPPrice:              "110.0",
			SLPrice:              "90.0",
			ProtectionKind:       contracts.ProtectionOCO,
			TrailingMode:         contracts.TrailingOff,
			TrailingTriggerPrice: "0",
			TrailingDeltaBips:    0,
			ClientOrderIDTP:      strings.Repeat("b", 36),
			ClientOrderIDSL:      strings.Repeat("c", 36),
		},
	}
	decisionHash, err := decision.Hash(snapshotHash)
	if err != nil {
		return contracts.Decision{}, err
	}
	decision.DecisionID = decisionHash
	if err := decision.Validate(); err != nil {
		return contracts.Decision{}, fmt.Errorf("decision invalid: %w", err)
	}
	return decision, nil
}
