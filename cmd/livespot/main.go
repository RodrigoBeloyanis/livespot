package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/app"
	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"
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

	if err := loop.Run(context.Background()); err != nil {
		log.Fatalf("loop failed: %v", err)
	}
}
