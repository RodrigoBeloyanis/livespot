package contracts

import "testing"

func TestSnapshotValidateAndHashDeterministic(t *testing.T) {
	candles := make([]Candle, 40)
	baseTs := int64(1706700000000)
	for i := 0; i < 40; i++ {
		candles[i] = Candle{
			TsMs:   baseTs + int64(i*300000),
			Open:   "100.00",
			High:   "101.00",
			Low:    "99.00",
			Close:  "100.50",
			Volume: "1.00000000",
		}
	}
	returns := make([]int32, 72)
	snapshot := Snapshot{
		Symbol: "BTCUSDT",
		Regime: RegimeSnapshot{
			Label:            "TREND",
			TrendScoreX10000: 7000,
			RangeScoreX10000: 3000,
		},
		Microstructure60s: Microstructure60s{
			SpreadBpsP50_60s:             10,
			SpreadBpsP90_60s:             20,
			SpreadCurrentBps:             12,
			DeltaSpreadBpsP90_10s:        2,
			BidAskImbalanceP50_10sX10000: 6000,
			OutOfOrderDrops:              0,
		},
		Volatility: VolatilitySnapshot{
			ATR14_5mBps:  45,
			ATR14_15mBps: 0,
		},
		Prices: PricesSnapshot{
			BestBid:   "100.00",
			BestAsk:   "100.10",
			MidPrice:  "100.05",
			LastPrice: "100.02",
		},
		Candles5m: candles,
		CostInputs: CostInputs{
			MakerFeeBps:           2,
			TakerFeeBps:           4,
			SlippageEntryMakerBps: 1,
			SlippageEntryTakerBps: 3,
			SlippageExitTakerBps:  2,
		},
		Market24h: Market24hSnapshot{
			QuoteVolume24hUSDT: "100000.00",
			Trades24h:          1000,
			PriceChange24hBps:  100,
			SourceTsMs:         baseTs,
		},
		HealthFlags: HealthFlagsSnapshot{
			FiltersOK:                true,
			WSOK:                     true,
			RecentRejectsWindowCount: 0,
			QuarantinedUntilMs:       0,
			SymbolStatus:             "TRADING",
		},
		ReturnsSeries: ReturnsSeries{
			Timeframe:    "5m",
			WindowPoints: 72,
			LogReturnBps: returns,
			MissingCount: 0,
			ComputedTsMs: baseTs,
		},
		ConfigReference: ConfigurationReference{
			ConfigHash:         "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			ThresholdsHash:     "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
			CycleConfigVersion: "20260131_1",
			FiltersHash:        "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		},
		Metadata: SnapshotMetadata{
			SnapshotID:      "snap_btc_1",
			CreatedTsMs:     baseTs,
			ExchangeTimeMs:  baseTs,
			LocalReceivedMs: baseTs + 10,
			SourceHashes: SourceHashes{
				CandlesHash: "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
				BookHash:    "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
				TickerHash:  "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			},
		},
	}
	if err := snapshot.Validate(); err != nil {
		t.Fatalf("snapshot validate failed: %v", err)
	}
	h1, err := snapshot.Hash()
	if err != nil {
		t.Fatalf("snapshot hash failed: %v", err)
	}
	h2, err := snapshot.Hash()
	if err != nil {
		t.Fatalf("snapshot hash failed: %v", err)
	}
	if h1 != h2 {
		t.Fatalf("snapshot hash mismatch")
	}
}

func TestSnapshotValidateMissingField(t *testing.T) {
	snapshot := Snapshot{}
	if err := snapshot.Validate(); err == nil {
		t.Fatalf("expected snapshot validation error")
	}
}
