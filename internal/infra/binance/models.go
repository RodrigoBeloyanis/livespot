package binance

type RateLimit struct {
	RateLimitType string `json:"rateLimitType"`
	Interval      string `json:"interval"`
	IntervalNum   int    `json:"intervalNum"`
	Limit         int    `json:"limit"`
}

type ExchangeInfo struct {
	Timezone   string               `json:"timezone"`
	ServerTime int64                `json:"serverTime"`
	RateLimits []RateLimit          `json:"rateLimits"`
	Symbols    []ExchangeInfoSymbol `json:"symbols"`
}

type ExchangeInfoSymbol struct {
	Symbol     string `json:"symbol"`
	Status     string `json:"status"`
	QuoteAsset string `json:"quoteAsset"`
}

type DepthResponse struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

type TimeResponse struct {
	ServerTime int64 `json:"serverTime"`
}

type JSONResponse struct {
	Status  int
	Body    []byte
	Headers map[string][]string
}

type BinanceError struct {
	Status int
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
}

func (e BinanceError) Error() string {
	if e.Code != 0 {
		return e.Msg
	}
	return "binance error"
}
