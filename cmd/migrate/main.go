package main

import (
	"log"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/infra/sqlite"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}
	db, err := sqlite.Open(audit.DefaultSQLitePath, cfg)
	if err != nil {
		log.Fatalf("sqlite open failed: %v", err)
	}
	defer func() {
		_ = db.Close()
	}()
	if err := audit.EnsureSchema(db, time.Now()); err != nil {
		log.Fatalf("migrate failed: %v", err)
	}
}
