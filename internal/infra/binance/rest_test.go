package binance

import (
	"net/url"
	"strings"
	"testing"
)

func TestBuildSignedQueryDoesNotLeakSecret(t *testing.T) {
	secret := "supersecret"
	params := url.Values{}
	params.Set("symbol", "BTCUSDT")
	query, err := buildSignedQuery(params.Encode(), secret, 1700000000000, 5000)
	if err != nil {
		t.Fatalf("build signed query: %v", err)
	}
	if query == "" {
		t.Fatalf("expected query")
	}
	if strings.Contains(query, secret) {
		t.Fatalf("secret leaked in query")
	}
}
