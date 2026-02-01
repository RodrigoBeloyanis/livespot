package main

import (
	"flag"
	"log"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/app"
	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/config"
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

	loop, err := app.NewLoop(cfg, writer, observability.ConsoleStageReporter{}, time.Now)
	if err != nil {
		log.Fatalf("loop init failed: %v", err)
	}
	if *dryRun {
		if err := loop.RunDryRun(); err != nil {
			log.Fatalf("dry run failed: %v", err)
		}
		return
	}

	log.Printf("no execution mode implemented yet; use --dry-run")
}
