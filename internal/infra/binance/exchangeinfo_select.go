package binance

import "fmt"

func SelectFirstTradingUSDT(info ExchangeInfo) (string, error) {
	for _, symbol := range info.Symbols {
		if symbol.Status == "TRADING" && symbol.QuoteAsset == "USDT" && symbol.Symbol != "" {
			return symbol.Symbol, nil
		}
	}
	return "", fmt.Errorf("no trading USDT symbols found")
}

