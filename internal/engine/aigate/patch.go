package aigate

import "github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"

func BuildDecisionPatch(original contracts.Decision, modified contracts.Decision) map[string]any {
	patch := make(map[string]any)
	if original.EntryPlan != nil && modified.EntryPlan != nil {
		entryPatch := map[string]any{}
		if original.EntryPlan.Kind != modified.EntryPlan.Kind {
			entryPatch["kind"] = modified.EntryPlan.Kind
		}
		if original.EntryPlan.DesiredPrice != modified.EntryPlan.DesiredPrice {
			entryPatch["desired_price"] = modified.EntryPlan.DesiredPrice
		}
		if original.EntryPlan.LimitPrice != modified.EntryPlan.LimitPrice {
			entryPatch["limit_price"] = modified.EntryPlan.LimitPrice
		}
		if original.EntryPlan.Qty != modified.EntryPlan.Qty {
			entryPatch["qty"] = modified.EntryPlan.Qty
		}
		if original.EntryPlan.TimeInForce != modified.EntryPlan.TimeInForce {
			entryPatch["time_in_force"] = modified.EntryPlan.TimeInForce
		}
		if original.EntryPlan.TTLMS != modified.EntryPlan.TTLMS {
			entryPatch["ttl_ms"] = modified.EntryPlan.TTLMS
		}
		if original.EntryPlan.RepriceMS != modified.EntryPlan.RepriceMS {
			entryPatch["reprice_ms"] = modified.EntryPlan.RepriceMS
		}
		if original.EntryPlan.MaxReprices != modified.EntryPlan.MaxReprices {
			entryPatch["max_reprices"] = modified.EntryPlan.MaxReprices
		}
		if original.EntryPlan.Fallback != modified.EntryPlan.Fallback {
			entryPatch["fallback"] = modified.EntryPlan.Fallback
		}
		if len(entryPatch) > 0 {
			patch["entry_plan"] = entryPatch
		}
	}
	if original.ExitPlan != nil && modified.ExitPlan != nil {
		exitPatch := map[string]any{}
		if original.ExitPlan.TPPrice != modified.ExitPlan.TPPrice {
			exitPatch["tp_price"] = modified.ExitPlan.TPPrice
		}
		if original.ExitPlan.SLPrice != modified.ExitPlan.SLPrice {
			exitPatch["sl_price"] = modified.ExitPlan.SLPrice
		}
		if original.ExitPlan.ProtectionKind != modified.ExitPlan.ProtectionKind {
			exitPatch["protection_kind"] = modified.ExitPlan.ProtectionKind
		}
		if original.ExitPlan.TrailingMode != modified.ExitPlan.TrailingMode {
			exitPatch["trailing_mode"] = modified.ExitPlan.TrailingMode
		}
		if original.ExitPlan.TrailingTriggerPrice != modified.ExitPlan.TrailingTriggerPrice {
			exitPatch["trailing_trigger_price"] = modified.ExitPlan.TrailingTriggerPrice
		}
		if original.ExitPlan.TrailingDeltaBips != modified.ExitPlan.TrailingDeltaBips {
			exitPatch["trailing_delta_bips"] = modified.ExitPlan.TrailingDeltaBips
		}
		if len(exitPatch) > 0 {
			patch["exit_plan"] = exitPatch
		}
	}
	return patch
}
