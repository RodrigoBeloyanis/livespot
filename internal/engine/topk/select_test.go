package topk

import (
	"testing"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
)

func TestSelectTopKCorrelation(t *testing.T) {
	cfg := config.Default()
	cfg.TopKSize = 2
	cfg.CorrMaxX10000 = 8000
	cfg.CorrWindowPoints = 6

	snapshots := map[string]contracts.Snapshot{
		"AAA": mockReturns("AAA", []int32{1, 2, 3, 4, 5, 6}),
		"BBB": mockReturns("BBB", []int32{2, 4, 6, 8, 10, 12}),
		"CCC": mockReturns("CCC", []int32{1, -1, 1, -1, 1, -1}),
	}
	ranked := []Selection{
		{Symbol: "AAA", ScoreX10000: 9000},
		{Symbol: "BBB", ScoreX10000: 8000},
		{Symbol: "CCC", ScoreX10000: 7000},
	}
	result := SelectTopK(cfg, ranked, snapshots, nil, 0)
	if len(result.TopK) != 2 {
		t.Fatalf("expected topk size 2")
	}
	if result.TopK[0].Symbol != "AAA" {
		t.Fatalf("expected AAA first")
	}
	if result.TopK[1].Symbol != "CCC" {
		t.Fatalf("expected CCC second due to correlation filter")
	}
	if result.MaxPairwiseCorr == 0 {
		t.Fatalf("expected corr computed")
	}
}

func mockReturns(symbol string, returns []int32) contracts.Snapshot {
	return contracts.Snapshot{
		Symbol: symbol,
		ReturnsSeries: contracts.ReturnsSeries{
			Timeframe:    "5m",
			WindowPoints: len(returns),
			LogReturnBps: returns,
			MissingCount: 0,
		},
	}
}
