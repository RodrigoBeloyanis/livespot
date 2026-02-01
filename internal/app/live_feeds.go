package app

import (
	"context"
	"log"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/infra/binance"
)

func StartRESTHeartbeat(ctx context.Context, cfg config.Config, loop *Loop, restClient *binance.Client) {
	if loop == nil || restClient == nil {
		return
	}
	interval := time.Duration(cfg.RestStaleMsDegrade) * time.Millisecond / 2
	if interval <= 0 {
		interval = 2 * time.Second
	}
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := restClient.Time(ctx); err != nil {
					log.Printf("rest heartbeat failed: %v", err)
					continue
				}
				loop.UpdateRESTLastSuccess(time.Now())
			}
		}
	}()
}

func StartWSFeed(ctx context.Context, loop *Loop, wsClient *binance.WSClient, symbol string) {
	if loop == nil || wsClient == nil || symbol == "" {
		return
	}
	go func() {
		err := wsClient.Run(ctx, []string{symbol}, func(ev binance.BookTickerEvent) {
			if ev.EventTime > 0 {
				loop.UpdateWSLastMsg(time.UnixMilli(ev.EventTime))
				return
			}
			loop.UpdateWSLastMsg(time.Now())
		})
		if err != nil {
			log.Printf("ws feed stopped: %v", err)
		}
	}()
}

