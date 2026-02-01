package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/e2e"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"
)

func main() {
	var duration time.Duration
	var interval time.Duration
	var dbPath string
	var jsonlDir string
	var outPath string
	flag.DurationVar(&duration, "duration", 24*time.Hour, "soak duration")
	flag.DurationVar(&interval, "interval", time.Second, "soak tick interval")
	flag.StringVar(&dbPath, "db", filepath.Join("var", "soak", "audit.sqlite"), "sqlite path")
	flag.StringVar(&jsonlDir, "jsonl", filepath.Join("var", "soak", "logs"), "jsonl dir")
	flag.StringVar(&outPath, "out", "", "output report path")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		exitErr(err)
	}
	writer, err := audit.NewWriter(cfg, audit.WriterOptions{
		DBPath:   dbPath,
		JSONLDir: jsonlDir,
		Now:      time.Now,
	})
	if err != nil {
		exitErr(err)
	}
	defer writer.Close()

	now := time.Now()
	runID := fmt.Sprintf("soak_%s", now.UTC().Format("20060102_150405"))
	cycleID := fmt.Sprintf("cyc_%s", now.UTC().Format("20060102_150405"))

	exchange := e2e.NewMockExchange(time.Now)
	exchange.FiltersJSON = []byte(`{"rateLimits":[]}`)
	exchange.DepthBySymbol["BTCUSDT"] = []byte(`{"lastUpdateId":1,"bids":[["100.0","1.0"]],"asks":[["100.1","1.0"]]}`)
	exchange.BalancesJSON = []byte(`{"balances":[{"asset":"USDT","free":"1000","locked":"0"}]}`)

	snapshot, err := e2e.SampleSnapshot(now)
	if err != nil {
		exitErr(err)
	}
	snapshotHash := snapshot.Metadata.SnapshotHash
	decision, err := e2e.SampleDecision(now, snapshot, snapshotHash, cycleID)
	if err != nil {
		exitErr(err)
	}

	pipeline, err := e2e.NewPipeline(cfg, writer, observability.ConsoleStageReporter{})
	if err != nil {
		exitErr(err)
	}

	input := e2e.PipelineInput{
		RunID:          runID,
		CycleID:        cycleID,
		Mode:           cfg.Mode,
		Snapshot:       snapshot,
		SnapshotHash:   snapshotHash,
		Decision:       decision,
		DecisionID:     decision.DecisionID,
		Exchange:       exchange,
		Now:            time.Now,
		DisableEntries: true,
	}
	runner := e2e.SoakRunner{
		Pipeline: pipeline,
		Config:   cfg,
		Input:    input,
		Duration: duration,
		Interval: interval,
	}
	result, err := runner.Run(context.Background())
	if err != nil {
		exitErr(err)
	}
	report := e2e.BuildReadinessReport(runID, result, true, true)
	buf, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		exitErr(err)
	}
	if outPath != "" {
		if err := os.WriteFile(outPath, buf, 0o600); err != nil {
			exitErr(err)
		}
	}
	fmt.Printf("%s\n", string(buf))
}

func exitErr(err error) {
	fmt.Fprintf(os.Stderr, "error: %s\n", err)
	os.Exit(1)
}
