package binance

import "encoding/json"

type Ticker24h struct {
	Symbol             string `json:"symbol"`
	LastPrice          string `json:"lastPrice"`
	QuoteVolume        string `json:"quoteVolume"`
	Count              int    `json:"count"`
	PriceChangePercent string `json:"priceChangePercent"`
}

func ParseTicker24hAll(body []byte) ([]Ticker24h, error) {
	var out []Ticker24h
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

