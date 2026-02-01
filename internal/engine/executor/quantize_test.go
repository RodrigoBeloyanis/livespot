package executor

import "testing"

func TestQuantizePrice(t *testing.T) {
	price, err := QuantizePrice("100.129", "0.01")
	if err != nil {
		t.Fatalf("quantize price: %v", err)
	}
	if price != "100.12" {
		t.Fatalf("expected 100.12, got %s", price)
	}
}

func TestQuantizeQtyMinNotional(t *testing.T) {
	qty, err := QuantizeQty("0.0109", "0.001", "0.001", "10", "1.00000000", "", "100.0")
	if err != nil {
		t.Fatalf("quantize qty: %v", err)
	}
	if qty != "0.010" {
		t.Fatalf("expected 0.010, got %s", qty)
	}
}
