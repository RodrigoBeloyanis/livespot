package risk

import "github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"

func positionPolicyReasons(hasOpenPosition bool, hasPendingEntry bool, hasPendingOCO bool) []reasoncodes.ReasonCode {
	reasons := []reasoncodes.ReasonCode{}
	if hasOpenPosition {
		reasons = append(reasons, reasoncodes.RISK_POSITION_ALREADY_OPEN)
	}
	if hasPendingEntry || hasPendingOCO {
		reasons = append(reasons, reasoncodes.RISK_ENTRY_ALREADY_PENDING)
	}
	return reasons
}
