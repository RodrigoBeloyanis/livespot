package executor

import (
	"testing"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
)

func TestQuantizeOrderDeterministic(t *testing.T) {
	filters := SymbolFilters{
		Price: &PriceFilter{
			MinPrice: "0.01",
			MaxPrice: "100000.00",
			TickSize: "0.01",
		},
		LotSize: &LotSizeFilter{
			MinQty:   "0.001",
			MaxQty:   "100.000",
			StepSize: "0.001",
		},
		MinNotional: &MinNotionalFilter{
			MinNotional: "10.00",
		},
	}
	req := OrderRequest{
		Symbol:        "BTCUSDT",
		Side:          contracts.SideBuy,
		Type:          OrderTypeLimit,
		TimeInForce:   contracts.TIFGTC,
		Price:         "100.009",
		Qty:           "0.1234",
		ClientOrderID: "id1",
	}
	out1, reason, err := QuantizeOrder(req, filters)
	if err != nil || reason != "" {
		t.Fatalf("unexpected error: %v reason=%s", err, reason)
	}
	out2, reason, err := QuantizeOrder(req, filters)
	if err != nil || reason != "" {
		t.Fatalf("unexpected error: %v reason=%s", err, reason)
	}
	if out1.Price != "100.00" || out1.Qty != "0.123" {
		t.Fatalf("unexpected quantization: price=%s qty=%s", out1.Price, out1.Qty)
	}
	if out1.Price != out2.Price || out1.Qty != out2.Qty {
		t.Fatalf("non-deterministic quantization")
	}
}

func TestQuantizeOrderMinNotional(t *testing.T) {
	filters := SymbolFilters{
		Price: &PriceFilter{
			MinPrice: "0.01",
			MaxPrice: "100000.00",
			TickSize: "0.01",
		},
		LotSize: &LotSizeFilter{
			MinQty:   "0.001",
			MaxQty:   "100.000",
			StepSize: "0.001",
		},
		MinNotional: &MinNotionalFilter{
			MinNotional: "10.00",
		},
	}
	req := OrderRequest{
		Symbol:        "BTCUSDT",
		Side:          contracts.SideBuy,
		Type:          OrderTypeLimit,
		TimeInForce:   contracts.TIFGTC,
		Price:         "100.00",
		Qty:           "0.05",
		ClientOrderID: "id2",
	}
	_, reason, err := QuantizeOrder(req, filters)
	if err == nil {
		t.Fatalf("expected min notional error")
	}
	if reason != reasoncodes.PROTECTION_INVALID_MIN_NOTIONAL {
		t.Fatalf("expected min notional reason, got %s", reason)
	}
}
