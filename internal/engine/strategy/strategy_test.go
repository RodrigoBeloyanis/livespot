package strategy

import (
	"testing"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
)

func TestProposeEntryDeterministicID(t *testing.T) {
	cfg := config.Default()
	now := time.UnixMilli(1700000000000)
	snapshot := strategySnapshot(now.UnixMilli())
	constraints := contracts.DecisionConstraints{
		TickSize:           "0.01",
		StepSize:           "0.001",
		MinQty:             "0.001",
		MinNotional:        "10.00",
		PricePrecision:     2,
		QtyPrecision:       3,
		MaxQty:             "1000.000",
		MaxNumOrders:       200,
		MaxAlgoOrders:      5,
		MaxNotional:        "",
		QuantizationPolicy: contracts.QuantizationEnforced,
	}
	first, err := ProposeEntry(cfg, snapshot, constraints, "cyc_test", now)
	if err != nil {
		t.Fatalf("propose entry: %v", err)
	}
	second, err := ProposeEntry(cfg, snapshot, constraints, "cyc_test", now)
	if err != nil {
		t.Fatalf("propose entry second: %v", err)
	}
	if first.DecisionID == "" || second.DecisionID == "" {
		t.Fatalf("expected decision_id")
	}
	if first.DecisionID != second.DecisionID {
		t.Fatalf("expected deterministic decision_id")
	}
	if len(first.EntryPlan.ClientOrderID) != 36 {
		t.Fatalf("expected client_order_id length 36")
	}
}

func strategySnapshot(nowMs int64) contracts.Snapshot {
	candles := make([]contracts.Candle, 0, 40)
	for i := 0; i < 40; i++ {
		ts := nowMs - int64((40-i)*300000)
		volume := "12.3456"
		if i == 39 {
			volume = "20.0000"
		}
		candles = append(candles, contracts.Candle{
			TsMs:   ts,
			Open:   "100.0",
			High:   "110.0",
			Low:    "90.0",
			Close:  "105.0",
			Volume: volume,
		})
	}
	return contracts.Snapshot{
		Symbol: "BTCUSDT",
		Regime: contracts.RegimeSnapshot{
			Label:            "TREND",
			TrendScoreX10000: 8000,
			RangeScoreX10000: 2000,
		},
		Microstructure60s: contracts.Microstructure60s{
			SpreadBpsP50_60s:             10,
			SpreadBpsP90_60s:             20,
			SpreadCurrentBps:             10,
			DeltaSpreadBpsP90_10s:        1,
			BidAskImbalanceP50_10sX10000: 6000,
			OutOfOrderDrops:              0,
		},
		Volatility: contracts.VolatilitySnapshot{
			ATR14_5mBps:  200,
			ATR14_15mBps: 150,
		},
		Prices: contracts.PricesSnapshot{
			BestBid:   "104.0",
			BestAsk:   "104.3",
			MidPrice:  "104.15",
			LastPrice: "104.2",
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
			QuoteVolume24hUSDT: "123456.78",
			Trades24h:          10000,
			PriceChange24hBps:  150,
			SourceTsMs:         nowMs - 1000,
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
			LogReturnBps: make([]int32, 72),
			MissingCount: 0,
			ComputedTsMs: nowMs - 5000,
		},
		ConfigReference: contracts.ConfigurationReference{
			ConfigHash:         "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			ThresholdsHash:     "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			CycleConfigVersion: "20260201_1",
			FiltersHash:        "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		},
		Metadata: contracts.SnapshotMetadata{
			SnapshotID:      "snap_BTCUSDT_1700000000_abc123",
			CreatedTsMs:     nowMs,
			ExchangeTimeMs:  nowMs - 2000,
			LocalReceivedMs: nowMs,
			SourceHashes: contracts.SourceHashes{
				CandlesHash: "1111111111111111111111111111111111111111111111111111111111111111",
				BookHash:    "2222222222222222222222222222222222222222222222222222222222222222",
				TickerHash:  "3333333333333333333333333333333333333333333333333333333333333333",
			},
			SnapshotHash: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		},
	}
}
