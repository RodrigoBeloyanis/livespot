package binance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWSClientReceivesBookTicker(t *testing.T) {
	upgrader := websocket.Upgrader{}
	var wg sync.WaitGroup
	wg.Add(1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade: %v", err)
		}
		defer conn.Close()
		env := streamEnvelope{
			Stream: "btcusdt@bookTicker",
			Data:   mustJSON(BookTickerEvent{EventTime: 1700000000000, Symbol: "BTCUSDT"}),
		}
		payload, err := json.Marshal(env)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
			t.Fatalf("write: %v", err)
		}
		wg.Done()
		time.Sleep(10 * time.Millisecond)
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	wsBase := "ws://" + u.Host

	client := NewWSClient(WSOptions{BaseURL: wsBase})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	got := make(chan BookTickerEvent, 1)
	go func() {
		_ = client.Run(ctx, []string{"BTCUSDT"}, func(ev BookTickerEvent) {
			select {
			case got <- ev:
			default:
			}
			cancel()
		})
	}()

	wg.Wait()
	select {
	case ev := <-got:
		if ev.Symbol != "BTCUSDT" {
			t.Fatalf("unexpected symbol: %s", ev.Symbol)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for event")
	}
}

func mustJSON(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
