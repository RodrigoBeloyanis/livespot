package risk

import (
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
)

func churnReasons(cfg config.Config, input Input) []reasoncodes.ReasonCode {
	reasons := []reasoncodes.ReasonCode{}
	if input.CancelReplaceCount10s >= cfg.RiskChurnMaxCancelReplace10s {
		reasons = append(reasons, reasoncodes.RISK_CANCEL_REPLACE_LIMIT_HIT)
	}
	if input.CancelCount10s >= cfg.RiskChurnMaxCancel10s || input.NewOrdersCount10s >= cfg.RiskChurnMaxNewOrders10s {
		reasons = append(reasons, reasoncodes.RISK_CHURN_LIMIT_HIT)
	}
	if input.UnfilledOrderCountPct >= cfg.RiskChurnUnfilledOrderCriticalPct {
		reasons = append(reasons, reasoncodes.RISK_UNFILLED_ORDER_COUNT_RISK)
	}
	return reasons
}
