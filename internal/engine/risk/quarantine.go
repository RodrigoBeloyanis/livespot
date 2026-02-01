package risk

import (
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
)

func quarantineReason(cfg config.Config, snapshot contracts.Snapshot, nowMs int64) reasoncodes.ReasonCode {
	if snapshot.HealthFlags.QuarantinedUntilMs > nowMs {
		return reasoncodes.SYMBOL_QUARANTINED
	}
	if snapshot.HealthFlags.SymbolStatus != "TRADING" {
		return reasoncodes.SYMBOL_QUARANTINED
	}
	if snapshot.HealthFlags.RecentRejectsWindowCount >= cfg.RiskQuarantineMaxRejectsPerHour {
		return reasoncodes.SYMBOL_QUARANTINED
	}
	if !snapshot.HealthFlags.FiltersOK {
		return reasoncodes.SYMBOL_QUARANTINED
	}
	return ""
}
