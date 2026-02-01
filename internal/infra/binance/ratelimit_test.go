package binance

import (
	"net/http"
	"testing"
	"time"
)

func TestRateLimiterDelayFromHeaders(t *testing.T) {
	limiter := NewRateLimiter()
	info := ExchangeInfo{
		RateLimits: []RateLimit{
			{RateLimitType: "REQUEST_WEIGHT", Interval: "MINUTE", IntervalNum: 1, Limit: 1200},
		},
	}
	limiter.UpdateFromExchangeInfo(info)

	now := time.Date(2026, 2, 1, 12, 0, 30, 0, time.UTC)
	h := http.Header{}
	h.Set("X-MBX-USED-WEIGHT-1M", "1200")
	limiter.UpdateFromHeaders(h, now)

	delay := limiter.Wait(now, 1)
	if delay <= 0 {
		t.Fatalf("expected delay, got %v", delay)
	}
}
