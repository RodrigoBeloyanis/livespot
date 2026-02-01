package observability

import (
	"fmt"
	"sync"
	"time"
)

type StageReporter interface {
	StageChanged(ts time.Time, stage StageName, cycleID string, decisionID string, symbol string, summary string)
}

type ConsoleStageReporter struct{}

func (ConsoleStageReporter) StageChanged(ts time.Time, stage StageName, cycleID string, decisionID string, symbol string, summary string) {
	stamp := ts.UTC().Format(time.RFC3339)
	fmt.Printf("%s stage=%s cycle_id=%s decision_id=%s symbol=%s %s\n", stamp, stage, cycleID, decisionID, symbol, summary)
}

type ThrottledStageReporter struct {
	interval time.Duration
	mu       sync.Mutex
	last     stageEvent
	hasLast  bool
}

type stageEvent struct {
	ts         time.Time
	stage      StageName
	cycleID    string
	decisionID string
	symbol     string
	summary    string
}

func NewThrottledStageReporter(interval time.Duration) *ThrottledStageReporter {
	if interval <= 0 {
		interval = 15 * time.Second
	}
	reporter := &ThrottledStageReporter{interval: interval}
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			reporter.mu.Lock()
			if !reporter.hasLast {
				reporter.mu.Unlock()
				continue
			}
			ev := reporter.last
			reporter.mu.Unlock()
			stamp := time.Now().UTC().Format(time.RFC3339)
			fmt.Printf("%s summary last_stage=%s cycle_id=%s decision_id=%s symbol=%s %s\n", stamp, ev.stage, ev.cycleID, ev.decisionID, ev.symbol, ev.summary)
		}
	}()
	return reporter
}

func (r *ThrottledStageReporter) StageChanged(ts time.Time, stage StageName, cycleID string, decisionID string, symbol string, summary string) {
	r.mu.Lock()
	r.last = stageEvent{
		ts:         ts,
		stage:      stage,
		cycleID:    cycleID,
		decisionID: decisionID,
		symbol:     symbol,
		summary:    summary,
	}
	r.hasLast = true
	r.mu.Unlock()
}
