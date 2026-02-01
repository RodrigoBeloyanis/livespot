package observability

import (
	"fmt"
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
