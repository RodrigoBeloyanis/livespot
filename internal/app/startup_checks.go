package app

import (
	"context"
	"fmt"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/aigate"
	"github.com/RodrigoBeloyanis/livespot/internal/infra/binance"
	"github.com/RodrigoBeloyanis/livespot/internal/infra/openai"
)

type StartupCheckReport struct {
	OpenAITestLatencyMs   int64
	BinanceRESTLatencyMs  int64
	BinanceWSLatencyMs    int64
	BinanceWSEventTimeMs  int64
	BinanceWSSymbol       string
	USDTFree              string
	USDTLocked            string
}

func RunStartupChecks(ctx context.Context, cfg config.Config, restClient *binance.Client, wsClient *binance.WSClient) (StartupCheckReport, error) {
	if restClient == nil {
		return StartupCheckReport{}, fmt.Errorf("binance rest client missing")
	}
	if wsClient == nil {
		return StartupCheckReport{}, fmt.Errorf("binance ws client missing")
	}
	openAIReport, err := checkOpenAI(ctx, cfg)
	if err != nil {
		return StartupCheckReport{}, err
	}
	restReport, exchangeInfo, err := checkBinanceREST(ctx, restClient)
	if err != nil {
		return StartupCheckReport{}, err
	}
	wsSymbol, err := binance.SelectFirstTradingUSDT(exchangeInfo)
	if err != nil {
		return StartupCheckReport{}, err
	}
	wsReport, err := checkBinanceWS(ctx, wsClient, wsSymbol)
	if err != nil {
		return StartupCheckReport{}, err
	}
	usdtFree, usdtLocked, err := checkUSDTBalance(ctx, restClient)
	if err != nil {
		return StartupCheckReport{}, err
	}
	return StartupCheckReport{
		OpenAITestLatencyMs:   openAIReport,
		BinanceRESTLatencyMs:  restReport,
		BinanceWSLatencyMs:    wsReport.latencyMs,
		BinanceWSEventTimeMs:  wsReport.eventTimeMs,
		BinanceWSSymbol:       wsSymbol,
		USDTFree:              usdtFree,
		USDTLocked:            usdtLocked,
	}, nil
}

func checkOpenAI(ctx context.Context, cfg config.Config) (int64, error) {
	client, err := aigate.NewOpenAIClient(cfg)
	if err != nil {
		return 0, err
	}
	timeout := time.Duration(cfg.AIGateTimeoutMs) * time.Millisecond
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	temperature := 0.0
	_, _, err = client.ChatCompletion(callCtx, openai.ChatCompletionRequest{
		Model: cfg.AIGateModel,
		Messages: []openai.Message{
			{Role: "user", Content: "ping"},
		},
		Temperature: &temperature,
	})
	if err != nil {
		return 0, err
	}
	return time.Since(start).Milliseconds(), nil
}

func checkBinanceREST(ctx context.Context, restClient *binance.Client) (int64, binance.ExchangeInfo, error) {
	start := time.Now()
	if _, err := restClient.Time(ctx); err != nil {
		return 0, binance.ExchangeInfo{}, err
	}
	info, err := restClient.ExchangeInfo(ctx)
	if err != nil {
		return 0, binance.ExchangeInfo{}, err
	}
	return time.Since(start).Milliseconds(), info, nil
}

type wsCheckReport struct {
	latencyMs  int64
	eventTimeMs int64
}

func checkBinanceWS(ctx context.Context, wsClient *binance.WSClient, symbol string) (wsCheckReport, error) {
	timeout := 15 * time.Second
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	eventCh := make(chan binance.BookTickerEvent, 1)
	errCh := make(chan error, 1)
	go func() {
		errCh <- wsClient.Run(callCtx, []string{symbol}, func(ev binance.BookTickerEvent) {
			select {
			case eventCh <- ev:
			default:
			}
		})
	}()
	select {
	case ev := <-eventCh:
		cancel()
		return wsCheckReport{
			latencyMs:  time.Since(start).Milliseconds(),
			eventTimeMs: ev.EventTime,
		}, nil
	case err := <-errCh:
		return wsCheckReport{}, err
	case <-callCtx.Done():
		return wsCheckReport{}, callCtx.Err()
	}
}

func checkUSDTBalance(ctx context.Context, restClient *binance.Client) (string, string, error) {
	resp, err := restClient.Account(ctx)
	if err != nil {
		return "", "", err
	}
	info, err := binance.ParseAccountInfo(resp.Body)
	if err != nil {
		return "", "", err
	}
	balance, ok := binance.FindBalance(info, "USDT")
	if !ok {
		return "", "", fmt.Errorf("usdt balance missing")
	}
	return balance.Free, balance.Locked, nil
}

