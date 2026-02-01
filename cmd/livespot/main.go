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
	"github.com/RodrigoBeloyanis/livespot/internal/infra/binance"
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	report, err := app.RunStartupChecks(ctx, cfg, restClient, wsClient)
	if err != nil {
		log.Fatalf("startup checks failed: %v", err)
	}
	log.Printf("startup check ok: openai model=%s latency_ms=%d", cfg.AIGateModel, report.OpenAITestLatencyMs)
	log.Printf("startup check ok: binance rest latency_ms=%d", report.BinanceRESTLatencyMs)
	log.Printf("startup check ok: binance ws symbol=%s latency_ms=%d", report.BinanceWSSymbol, report.BinanceWSLatencyMs)
	log.Printf("startup check ok: balance usdt free=%s locked=%s", report.USDTFree, report.USDTLocked)
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

	reporter := observability.NewThrottledStageReporter(15 * time.Second)
	loop, err := app.NewLoop(cfg, writer, reporter, time.Now)
	if err != nil {
		log.Fatalf("loop init failed: %v", err)
	}
	if report.BinanceWSEventTimeMs > 0 {
		loop.UpdateWSLastMsg(time.UnixMilli(report.BinanceWSEventTimeMs))
	}
	loop.UpdateRESTLastSuccess(time.Now())
	app.StartRESTHeartbeat(ctx, cfg, loop, restClient)
	app.StartWSFeed(ctx, loop, wsClient, report.BinanceWSSymbol)
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
