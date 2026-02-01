package position

import (
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	auditdomain "github.com/RodrigoBeloyanis/livespot/internal/domain/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"
)

type DriftDiff struct {
	OrdersMissing         int
	OrdersExtra           int
	OrdersStatusMismatch  int
	PositionsQtyMismatch  int
	PositionsSideMismatch int
	BalancesMismatch      int
	ProtectionMismatch    int
}

type DriftAction string

const (
	DriftNone    DriftAction = "NONE"
	DriftDegrade DriftAction = "DEGRADE"
	DriftPause   DriftAction = "PAUSE"
)

type ReconcileContext struct {
	RunID          string
	CycleID        string
	Mode           string
	SnapshotID     string
	DecisionID     string
	ExchangeTimeMs int64
}

func (d DriftDiff) DriftScoreX10000() int {
	orderDiff := d.OrdersMissing + d.OrdersExtra + d.OrdersStatusMismatch
	positionDiff := d.PositionsQtyMismatch + d.PositionsSideMismatch
	balanceDiff := d.BalancesMismatch
	protectionDiff := d.ProtectionMismatch
	score := minInt(orderDiff*4, 40) +
		minInt(positionDiff*6, 30) +
		minInt(balanceDiff*4, 20) +
		minInt(protectionDiff*2, 10)
	return score * 10000
}

func EvaluateDriftAction(cfg config.Config, driftScoreX10000 int) DriftAction {
	if driftScoreX10000 >= cfg.ReconcileDriftPauseX10000 {
		return DriftPause
	}
	if driftScoreX10000 >= cfg.ReconcileDriftDegradeX10000 {
		return DriftDegrade
	}
	return DriftNone
}

func BuildReconcileDiffRecord(now time.Time, ctx ReconcileContext, diff DriftDiff, driftScoreX10000 int) audit.Record {
	return audit.Record{
		Event: auditdomain.AuditEvent{
			TsMs:            now.UnixMilli(),
			RunID:           ctx.RunID,
			CycleID:         ctx.CycleID,
			Mode:            ctx.Mode,
			Stage:           observability.RECONCILE_REST,
			EventType:       auditdomain.RECONCILE_DIFF,
			Reasons:         []reasoncodes.ReasonCode{reasoncodes.RECONCILE_DIFF_DETECTED},
			SnapshotID:      ctx.SnapshotID,
			DecisionID:      ctx.DecisionID,
			OrderIntentID:   "",
			ExchangeTimeMs:  ctx.ExchangeTimeMs,
			LocalReceivedMs: now.UnixMilli(),
		},
		Data: map[string]any{
			"drift_score_x10000":      driftScoreX10000,
			"orders_missing":          diff.OrdersMissing,
			"orders_extra":            diff.OrdersExtra,
			"orders_status_mismatch":  diff.OrdersStatusMismatch,
			"positions_qty_mismatch":  diff.PositionsQtyMismatch,
			"positions_side_mismatch": diff.PositionsSideMismatch,
			"balances_mismatch":       diff.BalancesMismatch,
			"protection_mismatch":     diff.ProtectionMismatch,
		},
	}
}

func BuildReconcileAlertRecord(now time.Time, ctx ReconcileContext, driftScoreX10000 int, action DriftAction) audit.Record {
	reasons := []reasoncodes.ReasonCode{reasoncodes.DRIFT_LIMIT_EXCEEDED}
	switch action {
	case DriftPause:
		reasons = append(reasons, reasoncodes.ENTER_PAUSE)
	case DriftDegrade:
		reasons = append(reasons, reasoncodes.ENTER_DEGRADE)
	}
	return audit.Record{
		Event: auditdomain.AuditEvent{
			TsMs:            now.UnixMilli(),
			RunID:           ctx.RunID,
			CycleID:         ctx.CycleID,
			Mode:            ctx.Mode,
			Stage:           observability.RECONCILE_REST,
			EventType:       auditdomain.ALERT_RAISED,
			Reasons:         reasons,
			SnapshotID:      ctx.SnapshotID,
			DecisionID:      ctx.DecisionID,
			OrderIntentID:   "",
			ExchangeTimeMs:  ctx.ExchangeTimeMs,
			LocalReceivedMs: now.UnixMilli(),
		},
		Data: map[string]any{
			"drift_score_x10000": driftScoreX10000,
			"action":             string(action),
		},
	}
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
