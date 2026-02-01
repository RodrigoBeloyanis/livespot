package risk

import (
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
)

func antiOvertradingReasons(cfg config.Config, input Input) []reasoncodes.ReasonCode {
	reasons := []reasoncodes.ReasonCode{}
	if input.NowMs < input.CooldownUntilMs {
		reasons = append(reasons, reasoncodes.RISK_COOLDOWN_ACTIVE)
	}
	if input.TradesWindowCount >= cfg.RiskMaxTradesPerWindow {
		reasons = append(reasons, reasoncodes.RISK_MAX_TRADES_WINDOW)
	}
	if input.ConsecutiveLosses >= cfg.RiskMaxConsecutiveLosses {
		reasons = append(reasons, reasoncodes.RISK_SYMBOL_LOSS_STREAK)
	}
	if input.WSLatencyMs >= cfg.RiskWSLatencyThresholdMs {
		reasons = append(reasons, reasoncodes.WS_OOO_EVENT)
	}
	return reasons
}
