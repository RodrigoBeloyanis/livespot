package state

import (
	"testing"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
)

func TestBuildSnapshotHashDeterminism(t *testing.T) {
	cfg := config.Default()
	nowMs := int64(1700000000000)
	input := baseSnapshot(nowMs)

	first, err := BuildSnapshot(cfg, input, nowMs)
	if err != nil {
		t.Fatalf("build snapshot: %v", err)
	}
	second, err := BuildSnapshot(cfg, input, nowMs)
	if err != nil {
		t.Fatalf("build snapshot second: %v", err)
	}
	if first.Metadata.SnapshotHash == "" || second.Metadata.SnapshotHash == "" {
		t.Fatalf("expected snapshot hash")
	}
	if first.Metadata.SnapshotHash != second.Metadata.SnapshotHash {
		t.Fatalf("expected deterministic snapshot hash")
	}
}

func baseSnapshot(nowMs int64) contracts.Snapshot {
	candles := make([]contracts.Candle, 0, 40)
	for i := 0; i < 40; i++ {
		ts := nowMs - int64((40-i)*300000)
		candles = append(candles, contracts.Candle{
			TsMs:   ts,
			Open:   "100.0",
			High:   "110.0",
			Low:    "90.0",
			Close:  "105.0",
			Volume: "12.3456",
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
			SpreadCurrentBps:             12,
			DeltaSpreadBpsP90_10s:        1,
			BidAskImbalanceP50_10sX10000: 6000,
			OutOfOrderDrops:              0,
		},
		Volatility: contracts.VolatilitySnapshot{
			ATR14_5mBps:  45,
			ATR14_15mBps: 30,
		},
		Prices: contracts.PricesSnapshot{
			BestBid:   "100.0",
			BestAsk:   "100.5",
			MidPrice:  "100.25",
			LastPrice: "100.1",
		},
		Candles5m: candles,
		CostInputs: contracts.CostInputs{
			MakerFeeBps:           2,
			TakerFeeBps:           4,
			SlippageEntryMakerBps: 0,
			SlippageEntryTakerBps: 0,
			SlippageExitTakerBps:  0,
		},
		Market24h: contracts.Market24hSnapshot{
			QuoteVolume24hUSDT: "123456.78",
			Trades24h:          100,
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
		},
	}
}
