package binance

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RateLimiter struct {
	mu sync.Mutex

	requestWeightLimit int
	orderLimit         int
	orderWindow        time.Duration

	weightWindowStart time.Time
	usedWeight1m      int

	orderWindowStart time.Time
	usedOrdersWindow int
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{}
}

func (r *RateLimiter) UpdateFromExchangeInfo(info ExchangeInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, limit := range info.RateLimits {
		switch strings.ToUpper(limit.RateLimitType) {
		case "REQUEST_WEIGHT":
			if strings.ToUpper(limit.Interval) == "MINUTE" && limit.IntervalNum == 1 {
				r.requestWeightLimit = limit.Limit
			}
		case "ORDERS":
			if strings.ToUpper(limit.Interval) == "SECOND" && limit.IntervalNum > 0 {
				r.orderLimit = limit.Limit
				r.orderWindow = time.Duration(limit.IntervalNum) * time.Second
			}
		}
	}
}

func (r *RateLimiter) UpdateFromHeaders(h http.Header, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if used := headerInt(h, "X-MBX-USED-WEIGHT-1M"); used >= 0 {
		r.usedWeight1m = used
		r.weightWindowStart = now.Truncate(time.Minute)
	}
	if used := headerInt(h, "X-MBX-ORDER-COUNT-10S"); used >= 0 {
		r.usedOrdersWindow = used
		r.orderWindowStart = now.Truncate(10 * time.Second)
		if r.orderWindow == 0 {
			r.orderWindow = 10 * time.Second
		}
	}
}

func (r *RateLimiter) Wait(now time.Time, weight int) time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	delay := time.Duration(0)
	if r.requestWeightLimit > 0 {
		windowStart := now.Truncate(time.Minute)
		if r.weightWindowStart.IsZero() || r.weightWindowStart.Before(windowStart) {
			r.weightWindowStart = windowStart
			r.usedWeight1m = 0
		}
		if r.usedWeight1m+weight > r.requestWeightLimit {
			delay = windowStart.Add(time.Minute).Sub(now)
		} else {
			r.usedWeight1m += weight
		}
	}

	if r.orderLimit > 0 && r.orderWindow > 0 {
		windowStart := now.Truncate(r.orderWindow)
		if r.orderWindowStart.IsZero() || r.orderWindowStart.Before(windowStart) {
			r.orderWindowStart = windowStart
			r.usedOrdersWindow = 0
		}
		if r.usedOrdersWindow+1 > r.orderLimit {
			windowDelay := windowStart.Add(r.orderWindow).Sub(now)
			if windowDelay > delay {
				delay = windowDelay
			}
		} else {
			r.usedOrdersWindow++
		}
	}

	if delay < 0 {
		return 0
	}
	return delay
}

func headerInt(h http.Header, key string) int {
	raw := h.Get(key)
	if raw == "" {
		return -1
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return -1
	}
	return value
}
