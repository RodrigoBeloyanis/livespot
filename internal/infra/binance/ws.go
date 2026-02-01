package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

const defaultWSBaseURL = "wss://stream.binance.com:9443"

type WSOptions struct {
	BaseURL string
	Dialer  *websocket.Dialer
	Now     func() time.Time
}

type WSClient struct {
	baseURL string
	dialer  *websocket.Dialer
	now     func() time.Time
}

type BookTickerEvent struct {
	EventTime int64  `json:"E"`
	Symbol    string `json:"s"`
	BidPrice  string `json:"b"`
	BidQty    string `json:"B"`
	AskPrice  string `json:"a"`
	AskQty    string `json:"A"`
}

type streamEnvelope struct {
	Stream string          `json:"stream"`
	Data   json.RawMessage `json:"data"`
}

func NewWSClient(opts WSOptions) *WSClient {
	baseURL := strings.TrimSpace(opts.BaseURL)
	if baseURL == "" {
		baseURL = defaultWSBaseURL
	}
	dialer := opts.Dialer
	if dialer == nil {
		dialer = websocket.DefaultDialer
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	return &WSClient{
		baseURL: baseURL,
		dialer:  dialer,
		now:     now,
	}
}

func (c *WSClient) ProbeBookTicker(ctx context.Context, symbol string) (BookTickerEvent, int, error) {
	name := strings.TrimSpace(symbol)
	if name == "" {
		return BookTickerEvent{}, 0, fmt.Errorf("ws symbol missing")
	}
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return BookTickerEvent{}, 0, fmt.Errorf("ws base url: %w", err)
	}
	u.Path = "/stream"
	q := u.Query()
	q.Set("streams", strings.ToLower(name)+"@bookTicker")
	u.RawQuery = q.Encode()

	start := c.now()
	conn, _, err := c.dialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		return BookTickerEvent{}, 0, err
	}
	defer conn.Close()

	_, payload, err := conn.ReadMessage()
	if err != nil {
		return BookTickerEvent{}, 0, err
	}
	var env streamEnvelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return BookTickerEvent{}, 0, err
	}
	var event BookTickerEvent
	if err := json.Unmarshal(env.Data, &event); err != nil {
		return BookTickerEvent{}, 0, err
	}
	if event.EventTime <= 0 {
		event.EventTime = c.now().UnixMilli()
	}
	latency := int(c.now().Sub(start).Milliseconds())
	return event, latency, nil
}

func (c *WSClient) Run(ctx context.Context, symbols []string, onEvent func(BookTickerEvent)) error {
	if len(symbols) == 0 {
		return fmt.Errorf("ws symbols missing")
	}
	streams := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		name := strings.TrimSpace(symbol)
		if name == "" {
			continue
		}
		streams = append(streams, strings.ToLower(name)+"@bookTicker")
	}
	if len(streams) == 0 {
		return fmt.Errorf("ws symbols missing")
	}
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("ws base url: %w", err)
	}
	u.Path = "/stream"
	q := u.Query()
	q.Set("streams", strings.Join(streams, "/"))
	u.RawQuery = q.Encode()

	conn, _, err := c.dialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		return err
	}
	defer conn.Close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		_, payload, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		var env streamEnvelope
		if err := json.Unmarshal(payload, &env); err != nil {
			continue
		}
		var event BookTickerEvent
		if err := json.Unmarshal(env.Data, &event); err != nil {
			continue
		}
		if event.EventTime <= 0 {
			event.EventTime = c.now().UnixMilli()
		}
		if onEvent != nil {
			onEvent(event)
		}
	}
}
