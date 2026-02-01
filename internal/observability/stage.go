package observability

type StageName string

const (
	BOOT                 StageName = "BOOT"
	DOCTOR_CHECKS        StageName = "DOCTOR_CHECKS"
	STARTUP_RECOVER      StageName = "STARTUP_RECOVER"
	UNIVERSE_SCAN        StageName = "UNIVERSE_SCAN"
	RANK_TOPN            StageName = "RANK_TOPN"
	DEEP_SCAN            StageName = "DEEP_SCAN"
	WATCHLIST_ATTACH     StageName = "WATCHLIST_ATTACH"
	STATE_UPDATE         StageName = "STATE_UPDATE"
	STRATEGY_PROPOSE     StageName = "STRATEGY_PROPOSE"
	AIGATE_CALL          StageName = "AIGATE_CALL"
	RISK_VERDICT         StageName = "RISK_VERDICT"
	EXECUTE_INTENT       StageName = "EXECUTE_INTENT"
	POSITION_MANAGE      StageName = "POSITION_MANAGE"
	RECONCILE_REST       StageName = "RECONCILE_REST"
	REPORT_DAILY_SUMMARY StageName = "REPORT_DAILY_SUMMARY"
	DEGRADE              StageName = "DEGRADE"
	PAUSE                StageName = "PAUSE"
	SHUTDOWN             StageName = "SHUTDOWN"
)

var stages = map[StageName]struct{}{
	BOOT:                 {},
	DOCTOR_CHECKS:        {},
	STARTUP_RECOVER:      {},
	UNIVERSE_SCAN:        {},
	RANK_TOPN:            {},
	DEEP_SCAN:            {},
	WATCHLIST_ATTACH:     {},
	STATE_UPDATE:         {},
	STRATEGY_PROPOSE:     {},
	AIGATE_CALL:          {},
	RISK_VERDICT:         {},
	EXECUTE_INTENT:       {},
	POSITION_MANAGE:      {},
	RECONCILE_REST:       {},
	REPORT_DAILY_SUMMARY: {},
	DEGRADE:              {},
	PAUSE:                {},
	SHUTDOWN:             {},
}

func IsValidStage(stage StageName) bool {
	_, ok := stages[stage]
	return ok
}
