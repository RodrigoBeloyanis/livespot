package binance

import "testing"

func TestSelectFirstTradingUSDT(t *testing.T) {
	info := ExchangeInfo{
		Symbols: []ExchangeInfoSymbol{
			{Symbol: "ABCUSDT", Status: "BREAK", QuoteAsset: "USDT"},
			{Symbol: "BTCUSDT", Status: "TRADING", QuoteAsset: "USDT"},
			{Symbol: "ETHBTC", Status: "TRADING", QuoteAsset: "BTC"},
		},
	}
	symbol, err := SelectFirstTradingUSDT(info)
	if err != nil {
		t.Fatalf("select symbol: %v", err)
	}
	if symbol != "BTCUSDT" {
		t.Fatalf("unexpected symbol: %s", symbol)
	}
}

