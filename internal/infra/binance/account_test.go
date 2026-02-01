package binance

import "testing"

func TestParseAccountInfoAndFindBalance(t *testing.T) {
	body := []byte(`{"balances":[{"asset":"BTC","free":"0.1","locked":"0"},{"asset":"USDT","free":"123.45","locked":"6.78"}]}`)
	info, err := ParseAccountInfo(body)
	if err != nil {
		t.Fatalf("parse account: %v", err)
	}
	balance, ok := FindBalance(info, "USDT")
	if !ok {
		t.Fatalf("expected USDT balance")
	}
	if balance.Free != "123.45" || balance.Locked != "6.78" {
		t.Fatalf("unexpected balance values: free=%s locked=%s", balance.Free, balance.Locked)
	}
}

