package rank

import (
	"testing"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/universe"
)

func TestRankTopNDeterministic(t *testing.T) {
	cfg := config.Default()
	snapshots := []contracts.Snapshot{
		mockSnapshot("AAA", "6000000", 12000, 50, 10),
		mockSnapshot("BBB", "9000000", 15000, 80, 20),
		mockSnapshot("CCC", "7000000", 11000, 40, 15),
	}
	universeResults := []universe.ScanResult{
		{Symbol: "AAA", Eligible: true},
		{Symbol: "BBB", Eligible: true},
		{Symbol: "CCC", Eligible: true},
	}

	first, err := RankTopN(cfg, snapshots, universeResults)
	if err != nil {
		t.Fatalf("rank first: %v", err)
	}
	second, err := RankTopN(cfg, snapshots, universeResults)
	if err != nil {
		t.Fatalf("rank second: %v", err)
	}
	if len(first) != len(second) {
		t.Fatalf("expected same length")
	}
	for i := range first {
		if first[i].Symbol != second[i].Symbol || first[i].ScoreX10000 != second[i].ScoreX10000 {
			t.Fatalf("expected deterministic ranking")
		}
	}
}

func mockSnapshot(symbol string, volume string, trades int, priceChange int, spread int) contracts.Snapshot {
	snap := contracts.Snapshot{
		Symbol: symbol,
		Market24h: contracts.Market24hSnapshot{
			QuoteVolume24hUSDT: volume,
			Trades24h:          trades,
			PriceChange24hBps:  priceChange,
		},
		Microstructure60s: contracts.Microstructure60s{
			SpreadBpsP50_60s: spread,
		},
	}
	return snap
}
