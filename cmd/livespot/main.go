package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/app"
	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/aigate"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/executor"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/state"
	"github.com/RodrigoBeloyanis/livespot/internal/infra/binance"
	"github.com/RodrigoBeloyanis/livespot/internal/infra/openai"
	"github.com/RodrigoBeloyanis/livespot/internal/infra/sqlite"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"
	"github.com/RodrigoBeloyanis/livespot/internal/webui"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "run a single dry-run cycle with audit events")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}
	restClient, err := binance.NewClient(cfg, binance.Options{
		APIKey:     os.Getenv("BINANCE_API_KEY"),
		APISecret:  os.Getenv("BINANCE_API_SECRET"),
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
		Now:        time.Now,
	})
	if err != nil {
		log.Fatalf("binance client init failed: %v", err)
	}
	wsClient := binance.NewWSClient(binance.WSOptions{})
	writer, err := audit.NewWriter(cfg, audit.WriterOptions{})
	if err != nil {
		log.Fatalf("audit writer init failed: %v", err)
	}
	defer func() {
		_ = writer.Close()
	}()

	webDB, err := sqlite.Open(audit.DefaultSQLitePath, cfg)
	if err != nil {
		log.Fatalf("webui db open failed: %v", err)
	}
	defer func() {
		_ = webDB.Close()
	}()
	if err := sqlite.Migrate(webDB, time.Now()); err != nil {
		log.Fatalf("webui db migrate failed: %v", err)
	}
	log.Printf("startup check ok: sqlite migrate (%s)", audit.DefaultSQLitePath)

	openaiClient, err := aigate.NewOpenAIClient(cfg)
	if err != nil {
		log.Fatalf("openai client init failed: %v", err)
	}
	openaiStart := time.Now()
	_, _, err = openaiClient.ChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model: cfg.AIGateModel,
		Messages: []openai.Message{
			{Role: "system", Content: "ping"},
		},
	})
	if err != nil {
		log.Fatalf("openai startup check failed: %v", err)
	}
	log.Printf("startup check ok: openai model=%s latency_ms=%d", cfg.AIGateModel, time.Since(openaiStart).Milliseconds())

	restStart := time.Now()
	if _, err := restClient.Time(context.Background()); err != nil {
		log.Fatalf("binance rest startup check failed: %v", err)
	}
	log.Printf("startup check ok: binance rest latency_ms=%d", time.Since(restStart).Milliseconds())

	exchangeInfo, err := restClient.ExchangeInfo(context.Background())
	if err != nil {
		log.Fatalf("exchangeInfo failed: %v", err)
	}
	filters := binance.BuildFiltersMap(exchangeInfo)
	tickers, err := restClient.Ticker24hAll(context.Background())
	if err != nil {
		log.Fatalf("ticker24h failed: %v", err)
	}
	symbols := state.CandidateSymbols(cfg, tickers, filters)
	store := state.NewBookTickerStore()

	webServer, err := webui.NewServer(cfg, webDB, writer, time.Now)
	if err != nil {
		log.Fatalf("webui init failed: %v", err)
	}
	if err := webServer.Start(); err != nil {
		log.Fatalf("webui start failed: %v", err)
	}
	defer func() {
		_ = webServer.Close()
	}()

	provider := state.NewLiveSnapshotProvider(cfg, restClient, store, filters)
	recorder := aigate.NewRecorder(cfg, webDB, writer, time.Now)
	gate, err := aigate.NewGate(cfg, openaiClient, recorder, time.Now)
	if err != nil {
		log.Fatalf("ai gate init failed: %v", err)
	}
	reporter := observability.NewThrottledStageReporter(15 * time.Second)
	orderClient := executor.NewBinanceOrderClient(restClient)
	loop, err := app.NewLoop(cfg, writer, reporter, time.Now, webDB, provider, gate, orderClient, orderClient, orderClient, orderClient)
	if err != nil {
		log.Fatalf("loop init failed: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app.StartRESTHeartbeat(ctx, cfg, loop, restClient)
	if len(symbols) > 0 {
		app.StartWSFeed(ctx, loop, wsClient, symbols, store)
	}
	wsProbeCtx, wsCancel := context.WithTimeout(context.Background(), 8*time.Second)
	wsEvent, wsLatency, err := wsClient.ProbeBookTicker(wsProbeCtx, "BTCUSDT")
	wsCancel()
	if err != nil {
		log.Fatalf("binance ws startup check failed: %v", err)
	}
	log.Printf("startup check ok: binance ws symbol=%s latency_ms=%d", wsEvent.Symbol, wsLatency)

	accountResp, err := restClient.Account(context.Background())
	if err != nil {
		log.Fatalf("binance account startup check failed: %v", err)
	}
	accountInfo, err := binance.ParseAccountInfo(accountResp.Body)
	if err != nil {
		log.Fatalf("binance account decode failed: %v", err)
	}
	if balance, ok := binance.FindBalance(accountInfo, "USDT"); ok {
		log.Printf("startup check ok: balance usdt free=%s locked=%s", balance.Free, balance.Locked)
	} else {
		log.Printf("startup check ok: balance usdt free=0.00000000 locked=0.00000000")
	}

	now := time.Now()
	loop.UpdateRESTLastSuccess(now)
	loop.UpdateWSLastMsg(now)

	if *dryRun {
		if err := loop.RunDryRun(); err != nil {
			log.Fatalf("dry run failed: %v", err)
		}
		return
	}

	if err := loop.Run(ctx); err != nil {
		log.Fatalf("loop failed: %v", err)
	}
}
