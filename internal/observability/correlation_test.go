package observability

import "testing"

func TestClientOrderIDDeterministic(t *testing.T) {
	id, err := ClientOrderID("oi_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatalf("client order id failed: %v", err)
	}
	if len(id) != 36 {
		t.Fatalf("expected len 36, got %d", len(id))
	}
	id2, err := ClientOrderID("oi_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != nil {
		t.Fatalf("client order id failed: %v", err)
	}
	if id != id2 {
		t.Fatalf("client order id not deterministic")
	}
}
