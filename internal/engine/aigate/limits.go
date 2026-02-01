package aigate

import (
	"fmt"
	"math/big"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
)

func ApplyModify(original contracts.Decision, modified contracts.Decision) (contracts.Decision, error) {
	if err := validateImmutable(original, modified); err != nil {
		return contracts.Decision{}, err
	}
	if original.EntryPlan == nil || original.ExitPlan == nil {
		return contracts.Decision{}, fmt.Errorf("original plans missing")
	}
	if modified.EntryPlan == nil || modified.ExitPlan == nil {
		return contracts.Decision{}, fmt.Errorf("modified plans missing")
	}
	out := original
	out.DecisionID = original.DecisionID

	if err := applyEntryModify(original, modified, &out); err != nil {
		return contracts.Decision{}, err
	}
	if err := applyExitModify(original, modified, &out); err != nil {
		return contracts.Decision{}, err
	}
	if err := out.Validate(); err != nil {
		return contracts.Decision{}, err
	}
	return out, nil
}

func validateImmutable(original contracts.Decision, modified contracts.Decision) error {
	if original.Mode != modified.Mode ||
		original.Symbol != modified.Symbol ||
		original.Side != modified.Side ||
		original.Intent != modified.Intent ||
		original.SnapshotID != modified.SnapshotID ||
		original.CycleID != modified.CycleID {
		return fmt.Errorf("immutable fields changed")
	}
	if original.Constraints != modified.Constraints {
		return fmt.Errorf("constraints changed")
	}
	return nil
}

func applyEntryModify(original contracts.Decision, modified contracts.Decision, out *contracts.Decision) error {
	if original.EntryPlan == nil || modified.EntryPlan == nil {
		return fmt.Errorf("entry plan missing")
	}
	if original.EntryPlan.ClientOrderID != modified.EntryPlan.ClientOrderID {
		return fmt.Errorf("client_order_id changed")
	}
	if err := validateEntryKind(original.EntryPlan.Kind, modified.EntryPlan.Kind); err != nil {
		return err
	}
	if err := validateQtyReduction(original.EntryPlan.Qty, modified.EntryPlan.Qty); err != nil {
		return err
	}
	if err := validateEntryPrice(original.EntryPlan.LimitPrice, modified.EntryPlan.LimitPrice, original.Side); err != nil {
		return err
	}
	if err := validateEntryPrice(original.EntryPlan.DesiredPrice, modified.EntryPlan.DesiredPrice, original.Side); err != nil {
		return err
	}
	if modified.EntryPlan.MaxReprices > original.EntryPlan.MaxReprices {
		return fmt.Errorf("max_reprices increased")
	}
	if modified.EntryPlan.TTLMS > original.EntryPlan.TTLMS {
		return fmt.Errorf("ttl increased")
	}
	if modified.EntryPlan.RepriceMS < original.EntryPlan.RepriceMS {
		return fmt.Errorf("reprice interval reduced")
	}
	if err := validateFallback(original.EntryPlan.Fallback, modified.EntryPlan.Fallback); err != nil {
		return err
	}
	out.EntryPlan = modified.EntryPlan
	return nil
}

func applyExitModify(original contracts.Decision, modified contracts.Decision, out *contracts.Decision) error {
	if original.ExitPlan == nil || modified.ExitPlan == nil {
		return fmt.Errorf("exit plan missing")
	}
	if original.ExitPlan.ClientOrderIDTP != modified.ExitPlan.ClientOrderIDTP ||
		original.ExitPlan.ClientOrderIDSL != modified.ExitPlan.ClientOrderIDSL {
		return fmt.Errorf("exit client_order_id changed")
	}
	if err := validateStopTighten(original.ExitPlan.SLPrice, modified.ExitPlan.SLPrice, original.Side); err != nil {
		return err
	}
	if err := validateTakeProfit(original.ExitPlan.TPPrice, modified.ExitPlan.TPPrice, original.Side); err != nil {
		return err
	}
	if err := validateTrailing(original.ExitPlan, modified.ExitPlan); err != nil {
		return err
	}
	out.ExitPlan = modified.ExitPlan
	return nil
}

func validateEntryKind(original contracts.EntryKind, modified contracts.EntryKind) error {
	if aggressiveness(modified) > aggressiveness(original) {
		return fmt.Errorf("entry kind more aggressive")
	}
	return nil
}

func aggressiveness(kind contracts.EntryKind) int {
	switch kind {
	case contracts.EntryMarket:
		return 3
	case contracts.EntryTaker:
		return 2
	case contracts.EntryMakerFirst:
		return 1
	default:
		return 3
	}
}

func validateQtyReduction(original string, modified string) error {
	cmp, err := compareDecimal(modified, original)
	if err != nil {
		return err
	}
	if cmp > 0 {
		return fmt.Errorf("qty increased")
	}
	return nil
}

func validateEntryPrice(original string, modified string, side contracts.Side) error {
	if original == "" || modified == "" {
		return fmt.Errorf("price missing")
	}
	cmp, err := compareDecimal(modified, original)
	if err != nil {
		return err
	}
	if side == contracts.SideBuy && cmp > 0 {
		return fmt.Errorf("price more aggressive")
	}
	if side == contracts.SideSell && cmp < 0 {
		return fmt.Errorf("price more aggressive")
	}
	return nil
}

func validateStopTighten(original string, modified string, side contracts.Side) error {
	cmp, err := compareDecimal(modified, original)
	if err != nil {
		return err
	}
	if side == contracts.SideBuy && cmp < 0 {
		return fmt.Errorf("stop loosened")
	}
	if side == contracts.SideSell && cmp > 0 {
		return fmt.Errorf("stop loosened")
	}
	return nil
}

func validateTakeProfit(original string, modified string, side contracts.Side) error {
	cmp, err := compareDecimal(modified, original)
	if err != nil {
		return err
	}
	if side == contracts.SideBuy && cmp > 0 {
		return fmt.Errorf("tp increased")
	}
	if side == contracts.SideSell && cmp < 0 {
		return fmt.Errorf("tp increased")
	}
	return nil
}

func validateTrailing(original *contracts.ExitPlan, modified *contracts.ExitPlan) error {
	if modified.TrailingMode == contracts.TrailingOff {
		if modified.TrailingDeltaBips != 0 {
			return fmt.Errorf("trailing delta invalid")
		}
		return nil
	}
	if aggressivenessTrailing(modified.TrailingMode) > aggressivenessTrailing(original.TrailingMode) {
		return fmt.Errorf("trailing mode more aggressive")
	}
	if modified.TrailingDeltaBips > original.TrailingDeltaBips {
		return fmt.Errorf("trailing delta increased")
	}
	return nil
}

func aggressivenessTrailing(mode contracts.TrailingMode) int {
	switch mode {
	case contracts.TrailingNative:
		return 3
	case contracts.TrailingVirtual:
		return 2
	case contracts.TrailingOff:
		return 1
	default:
		return 3
	}
}

func validateFallback(original contracts.FallbackPlan, modified contracts.FallbackPlan) error {
	if !original.Enabled && modified.Enabled {
		return fmt.Errorf("fallback enabled")
	}
	if !modified.Enabled {
		return nil
	}
	if modified.MaxSlippageBps > original.MaxSlippageBps {
		return fmt.Errorf("slippage increased")
	}
	if modified.DeadlineMS > original.DeadlineMS {
		return fmt.Errorf("deadline increased")
	}
	if fallbackAggressiveness(modified.Kind) > fallbackAggressiveness(original.Kind) {
		return fmt.Errorf("fallback more aggressive")
	}
	return nil
}

func fallbackAggressiveness(kind contracts.FallbackKind) int {
	switch kind {
	case contracts.FallbackMarketIfAllowed:
		return 3
	case contracts.FallbackIOCLimit:
		return 2
	case contracts.FallbackCancelReplace:
		return 1
	default:
		return 3
	}
}

func compareDecimal(a string, b string) (int, error) {
	ra, err := parseDecimal(a)
	if err != nil {
		return 0, err
	}
	rb, err := parseDecimal(b)
	if err != nil {
		return 0, err
	}
	return ra.Cmp(rb), nil
}

func parseDecimal(s string) (*big.Rat, error) {
	if s == "" {
		return nil, fmt.Errorf("decimal missing")
	}
	r := new(big.Rat)
	if _, ok := r.SetString(s); !ok {
		return nil, fmt.Errorf("invalid decimal")
	}
	return r, nil
}
