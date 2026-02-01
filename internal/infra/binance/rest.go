package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
)

const defaultBaseURL = "https://api.binance.com"

type Options struct {
	BaseURL    string
	APIKey     string
	APISecret  string
	HTTPClient *http.Client
	Now        func() time.Time
}

type Client struct {
	cfg        config.Config
	baseURL    string
	apiKey     string
	apiSecret  string
	httpClient *http.Client
	now        func() time.Time

	limiter *RateLimiter

	mu             sync.Mutex
	clockOffsetMs  int64
	lastTimeSyncMs int64
}

func NewClient(cfg config.Config, opts Options) (*Client, error) {
	baseURL := strings.TrimSpace(opts.BaseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	return &Client{
		cfg:        cfg,
		baseURL:    baseURL,
		apiKey:     opts.APIKey,
		apiSecret:  opts.APISecret,
		httpClient: client,
		now:        opts.Now,
		limiter:    NewRateLimiter(),
	}, nil
}

func (c *Client) Time(ctx context.Context) (TimeResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v3/time", url.Values{}, false, 1, false)
	if err != nil {
		return TimeResponse{}, err
	}
	var tr TimeResponse
	if err := json.Unmarshal(resp.Body, &tr); err != nil {
		return TimeResponse{}, fmt.Errorf("time decode: %w", err)
	}
	return tr, nil
}

func (c *Client) SyncTime(ctx context.Context) (int64, error) {
	tr, err := c.Time(ctx)
	if err != nil {
		return 0, err
	}
	nowMs := c.now().UnixMilli()
	offset := tr.ServerTime - nowMs
	c.mu.Lock()
	c.clockOffsetMs = offset
	c.lastTimeSyncMs = nowMs
	c.mu.Unlock()
	return offset, nil
}

func (c *Client) ExchangeInfo(ctx context.Context) (ExchangeInfo, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v3/exchangeInfo", url.Values{}, false, 10, false)
	if err != nil {
		return ExchangeInfo{}, err
	}
	var info ExchangeInfo
	if err := json.Unmarshal(resp.Body, &info); err != nil {
		return ExchangeInfo{}, fmt.Errorf("exchangeInfo decode: %w", err)
	}
	c.limiter.UpdateFromExchangeInfo(info)
	return info, nil
}

func (c *Client) Ticker24hAll(ctx context.Context) ([]Ticker24h, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v3/ticker/24hr", url.Values{}, false, 40, false)
	if err != nil {
		return nil, err
	}
	tickers, err := ParseTicker24hAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ticker24h decode: %w", err)
	}
	return tickers, nil
}

func (c *Client) Klines(ctx context.Context, symbol string, interval string, limit int) ([]Kline, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("interval", interval)
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v3/klines", params, false, 2, false)
	if err != nil {
		return nil, err
	}
	klines, err := ParseKlines(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("klines decode: %w", err)
	}
	return klines, nil
}

func (c *Client) Depth(ctx context.Context, symbol string, limit int) (DepthResponse, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v3/depth", params, false, 1, false)
	if err != nil {
		return DepthResponse{}, err
	}
	var depth DepthResponse
	if err := json.Unmarshal(resp.Body, &depth); err != nil {
		return DepthResponse{}, fmt.Errorf("depth decode: %w", err)
	}
	return depth, nil
}

func (c *Client) NewOrder(ctx context.Context, params url.Values) (JSONResponse, error) {
	return c.doRequest(ctx, http.MethodPost, "/api/v3/order", params, true, 1, true)
}

func (c *Client) QueryOrder(ctx context.Context, params url.Values) (JSONResponse, error) {
	return c.doRequest(ctx, http.MethodGet, "/api/v3/order", params, true, 1, false)
}

func (c *Client) CancelOrder(ctx context.Context, params url.Values) (JSONResponse, error) {
	return c.doRequest(ctx, http.MethodDelete, "/api/v3/order", params, true, 1, true)
}

func (c *Client) CancelReplaceOrder(ctx context.Context, params url.Values) (JSONResponse, error) {
	return c.doRequest(ctx, http.MethodPost, "/api/v3/order/cancelReplace", params, true, 1, true)
}

func (c *Client) NewOCO(ctx context.Context, params url.Values) (JSONResponse, error) {
	return c.doRequest(ctx, http.MethodPost, "/api/v3/order/oco", params, true, 1, true)
}

func (c *Client) OpenOrders(ctx context.Context, params url.Values) (JSONResponse, error) {
	return c.doRequest(ctx, http.MethodGet, "/api/v3/openOrders", params, true, 1, false)
}

func (c *Client) AllOrders(ctx context.Context, params url.Values) (JSONResponse, error) {
	return c.doRequest(ctx, http.MethodGet, "/api/v3/allOrders", params, true, 1, false)
}

func (c *Client) Account(ctx context.Context) (JSONResponse, error) {
	return c.doRequest(ctx, http.MethodGet, "/api/v3/account", url.Values{}, true, 1, false)
}

func (c *Client) doRequest(ctx context.Context, method string, path string, params url.Values, signed bool, weight int, idempotent bool) (JSONResponse, error) {
	if signed {
		if err := c.ensureTimeSync(ctx); err != nil {
			return JSONResponse{}, err
		}
	}
	resp, err := c.doOnce(ctx, method, path, params, signed, weight)
	if err == nil {
		return resp, nil
	}
	var bErr BinanceError
	if errors.As(err, &bErr) && bErr.Code == -1021 && signed && idempotent {
		if _, syncErr := c.SyncTime(ctx); syncErr != nil {
			return JSONResponse{}, syncErr
		}
		return c.doOnce(ctx, method, path, params, signed, weight)
	}
	return JSONResponse{}, err
}

func (c *Client) doOnce(ctx context.Context, method string, path string, params url.Values, signed bool, weight int) (JSONResponse, error) {
	if weight <= 0 {
		weight = 1
	}
	if delay := c.limiter.Wait(c.now(), weight); delay > 0 {
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return JSONResponse{}, ctx.Err()
		case <-timer.C:
		}
	}
	query := params.Encode()
	if signed {
		if c.apiSecret == "" || c.apiKey == "" {
			return JSONResponse{}, fmt.Errorf("binance credentials missing")
		}
		timestamp := c.now().UnixMilli() + c.clockOffset()
		signedQuery, err := buildSignedQuery(query, c.apiSecret, timestamp, c.cfg.TimeSyncRecvWindowMs)
		if err != nil {
			return JSONResponse{}, err
		}
		query = signedQuery
	}
	fullURL := c.baseURL + path
	if query != "" {
		fullURL += "?" + query
	}
	req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return JSONResponse{}, err
	}
	if signed {
		req.Header.Set("X-MBX-APIKEY", c.apiKey)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return JSONResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return JSONResponse{}, err
	}
	c.limiter.UpdateFromHeaders(resp.Header, c.now())

	if resp.StatusCode >= 400 {
		bErr := BinanceError{Status: resp.StatusCode}
		_ = json.Unmarshal(body, &bErr)
		if bErr.Msg == "" {
			bErr.Msg = string(body)
		}
		return JSONResponse{}, bErr
	}
	return JSONResponse{
		Status:  resp.StatusCode,
		Body:    body,
		Headers: resp.Header,
	}, nil
}

func (c *Client) ensureTimeSync(ctx context.Context) error {
	nowMs := c.now().UnixMilli()
	c.mu.Lock()
	lastSync := c.lastTimeSyncMs
	c.mu.Unlock()
	if lastSync == 0 || nowMs-lastSync >= int64(c.cfg.TimeSyncIntervalMs) {
		_, err := c.SyncTime(ctx)
		return err
	}
	return nil
}

func (c *Client) clockOffset() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.clockOffsetMs
}

func buildSignedQuery(query string, secret string, timestamp int64, recvWindowMs int) (string, error) {
	values, err := url.ParseQuery(query)
	if err != nil {
		return "", err
	}
	values.Set("timestamp", fmt.Sprintf("%d", timestamp))
	values.Set("recvWindow", fmt.Sprintf("%d", recvWindowMs))
	encoded := values.Encode()
	mac := hmac.New(sha256.New, []byte(secret))
	if _, err := mac.Write([]byte(encoded)); err != nil {
		return "", err
	}
	signature := hex.EncodeToString(mac.Sum(nil))
	values.Set("signature", signature)
	return values.Encode(), nil
}
