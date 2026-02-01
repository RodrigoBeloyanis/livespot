package app

import (
	"fmt"

	"github.com/RodrigoBeloyanis/livespot/internal/observability"
)

type StageSequence struct {
	Stages []observability.StageName
}

func DefaultStageSequence() StageSequence {
	return StageSequence{
		Stages: []observability.StageName{
			observability.BOOT,
			observability.DOCTOR_CHECKS,
			observability.STARTUP_RECOVER,
			observability.UNIVERSE_SCAN,
			observability.RANK_TOPN,
			observability.DEEP_SCAN,
			observability.WATCHLIST_ATTACH,
			observability.STATE_UPDATE,
			observability.STRATEGY_PROPOSE,
			observability.AIGATE_CALL,
			observability.RISK_VERDICT,
			observability.EXECUTE_INTENT,
			observability.POSITION_MANAGE,
			observability.RECONCILE_REST,
			observability.REPORT_DAILY_SUMMARY,
			observability.SHUTDOWN,
		},
	}
}

func (s StageSequence) Validate() error {
	if len(s.Stages) == 0 {
		return fmt.Errorf("stage sequence empty")
	}
	for _, stage := range s.Stages {
		if !observability.IsValidStage(stage) {
			return fmt.Errorf("invalid stage: %s", stage)
		}
	}
	return nil
}
