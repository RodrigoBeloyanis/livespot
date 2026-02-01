package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	auditdomain "github.com/RodrigoBeloyanis/livespot/internal/domain/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/hash"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/aigate"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/deepscan"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/executor"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/health"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/persist"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/rank"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/risk"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/state"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/strategy"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/topk"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/universe"
	"github.com/RodrigoBeloyanis/livespot/internal/infra/sqlite"
	"github.com/RodrigoBeloyanis/livespot/internal/infra/binance"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"
)

type Loop struct {
	cfg      config.Config
	writer   *audit.Writer
	reporter observability.StageReporter
	now      func() time.Time
	sysEval  health.Evaluator
	sysMode  health.SysMode
	db       *sql.DB
	provider *state.LiveSnapshotProvider
	gate     *aigate.Gate
	ledger   *executor.LedgerService
	orderClient executor.OrderRestClient
	intentLookup executor.RestClient
	selectionStore *persist.SelectionStore
	prevTopK []string
	cyclesSincePrev int

	sysModeSince   time.Time
	sysModeReasons []reasoncodes.ReasonCode

	lastProgressMs    int64
	wsLastMsgMs       int64
	restLastSuccessMs int64
	diskFreeBytes     int64
	auditWriterLagMs  int
	forceExit         bool
}

func NewLoop(cfg config.Config, writer *audit.Writer, reporter observability.StageReporter, now func() time.Time, db *sql.DB, provider *state.LiveSnapshotProvider, gate *aigate.Gate, orderClient executor.OrderRestClient, intentLookup executor.RestClient) (*Loop, error) {
	if writer == nil {
		return nil, fmt.Errorf("audit writer missing")
	}
	if now == nil {
		now = time.Now
	}
	var ledger *executor.LedgerService
	if db != nil {
		ledger = executor.NewLedger(db, now)
	}
	return &Loop{
		cfg:          cfg,
		writer:       writer,
		reporter:     reporter,
		now:          now,
		sysEval:      health.NewEvaluator(cfg),
		sysMode:      health.SysModeNormal,
		sysModeSince: now(),
		db:           db,
		provider:     provider,
		gate:         gate,
		ledger:       ledger,
		orderClient:  orderClient,
		intentLookup: intentLookup,
		selectionStore: persist.NewSelectionStore(db),
	}, nil
}

func (l *Loop) RunDryRun() error {
	runID, err := observability.NewRunID(l.now())
	if err != nil {
		return err
	}
	cycleID, err := observability.NewCycleID(l.now())
	if err != nil {
		return err
	}
	for _, stage := range l.stageSequence() {
		if err := l.emitStage(runID, cycleID, stage, "", "dry-run"); err != nil {
			return err
		}
	}
	return nil
}

func (l *Loop) Run(ctx context.Context) error {
	runID, err := observability.NewRunID(l.now())
	if err != nil {
		return err
	}
	l.lastProgressMs = l.now().UnixMilli()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		cycleID, err := observability.NewCycleID(l.now())
		if err != nil {
			return err
		}
		if err := l.runCycle(ctx, runID, cycleID); err != nil {
			return err
		}
	}
}

func (l *Loop) runCycle(ctx context.Context, runID string, cycleID string) error {
	if err := l.refreshSysMode(ctx, runID, cycleID); err != nil {
		return err
	}
	if l.sysMode == health.SysModePause {
		if err := l.emitStage(runID, cycleID, observability.PAUSE, "", "paused"); err != nil {
			return err
		}
		if err := sleepCtx(ctx, 500*time.Millisecond); err != nil {
			return err
		}
		return nil
	}
	if l.provider == nil || l.db == nil {
		return fmt.Errorf("loop deps missing")
	}

	bundles, err := l.provider.BuildSnapshots(ctx, l.now())
	if err != nil {
		return err
	}
	if err := l.persistSnapshots(ctx, bundles); err != nil {
		return err
	}

	snapshotList := make([]state.SnapshotBundle, 0, len(bundles))
	snapshotsOnly := make([]contracts.Snapshot, 0, len(bundles))
	for _, bundle := range bundles {
		snapshotList = append(snapshotList, bundle)
		snapshotsOnly = append(snapshotsOnly, bundle.Snapshot)
	}

	if err := l.emitStage(runID, cycleID, observability.BOOT, "", "cycle start"); err != nil {
		return err
	}

	universeResults, err := universe.Scan(l.cfg, snapshotsOnly)
	if err != nil {
		return err
	}
	if err := l.selectionStore.InsertUniverseScans(runID, cycleID, universeResults, l.now()); err != nil {
		return err
	}
	if err := l.emitStage(runID, cycleID, observability.UNIVERSE_SCAN, "", "universe scanned"); err != nil {
		return err
	}

	ranked, err := rank.RankTopN(l.cfg, snapshotsOnly, universeResults)
	if err != nil {
		return err
	}
	if err := l.selectionStore.InsertRankings(runID, cycleID, "TOPN", ranked, l.now()); err != nil {
		return err
	}
	if err := l.emitStage(runID, cycleID, observability.RANK_TOPN, "", "ranked"); err != nil {
		return err
	}

	topnSnapshots := filterSnapshotsByRank(snapshotsOnly, ranked)
	deepResults, err := deepscan.DeepScan(l.cfg, topnSnapshots)
	if err != nil {
		return err
	}
	if err := l.selectionStore.InsertDeepScan(runID, cycleID, deepResults, l.now()); err != nil {
		return err
	}
	if err := l.emitStage(runID, cycleID, observability.DEEP_SCAN, "", "deep scan"); err != nil {
		return err
	}

	selections := make([]topk.Selection, 0, len(deepResults))
	for _, item := range deepResults {
		selections = append(selections, topk.Selection{
			Symbol:      item.Symbol,
			ScoreX10000: item.ScoreX10000,
			Features:    item.Features,
		})
	}
	snapMap := map[string]contracts.Snapshot{}
	for _, s := range snapshotsOnly {
		snapMap[s.Symbol] = s
	}
	result := topk.SelectTopK(l.cfg, selections, snapMap, l.prevTopK, l.cyclesSincePrev)
	topkSymbols := make([]string, 0, len(result.TopK))
	for _, sel := range result.TopK {
		topkSymbols = append(topkSymbols, sel.Symbol)
	}
	if err := l.selectionStore.InsertSelection(runID, cycleID, symbolsFromRank(ranked), topkSymbols, topkSymbols, result.ChurnGuardApplied, len(result.PairsOverLimit) > 0, result.MaxPairwiseCorr, result.PairsOverLimit, rankedConfigHash(ranked), l.now()); err != nil {
		return err
	}
	if err := l.emitStage(runID, cycleID, observability.WATCHLIST_ATTACH, "", "watchlist"); err != nil {
		return err
	}
	l.prevTopK = topkSymbols
	l.cyclesSincePrev++

	for _, sym := range topkSymbols {
		bundle, ok := findBundle(snapshotList, sym)
		if !ok {
			continue
		}
		decision, err := strategy.ProposeEntry(l.cfg, bundle.Snapshot, bundle.Constraints, cycleID, l.now())
		if err != nil {
			continue
		}
		if err := l.emitStage(runID, cycleID, observability.STRATEGY_PROPOSE, sym, "decision"); err != nil {
			return err
		}
		decisionID, err := decision.Hash(bundle.Snapshot.Metadata.SnapshotHash)
		if err != nil {
			return err
		}
		decision.DecisionID = "dec_" + decisionID

		var aiResult contracts.AIGateResult
		var applied *contracts.Decision
		if l.cfg.AiDec > 0 && l.gate != nil {
			aiResult, applied, err = l.gate.Evaluate(ctx, aigate.CallContext{
				RunID:          runID,
				CycleID:        cycleID,
				ExchangeTimeMs: bundle.Snapshot.Metadata.ExchangeTimeMs,
			}, decision, bundle.Snapshot)
			if err != nil && l.cfg.AiDec == 2 {
				return err
			}
			if applied != nil {
				decision = *applied
			}
			decision.AIGate = &aiResult
		}

		freeBalance, lockedBalance := l.accountBalances()
		riskInput := risk.Input{
			NowMs:             l.now().UnixMilli(),
			Snapshot:          bundle.Snapshot,
			Decision:          decision,
			ExposureSymbolUSDT: "0.00",
			ExposureTotalUSDT:  "0.00",
			OpenOrdersSymbol:   0,
			OpenOrdersTotal:    0,
			TradesToday:        0,
			TradesWindowCount:  0,
			CooldownUntilMs:    0,
			ConsecutiveLosses:  0,
			WSLatencyMs:        0,
			HasOpenPosition:    false,
			HasPendingEntry:    false,
			HasPendingOCO:      false,
			RealizedPnLUSDT:    "0.00",
			UnrealizedPnLUSDT:  "0.00",
			EquityPeakUSDT:     "0.00",
			EquityStartUSDT:    "0.00",
			FreeBalanceUSDT:    freeBalance,
			LockedBalanceUSDT:  lockedBalance,
			PendingReserveUSDT: "0.00",
			UnfilledOrderCountPct: 0,
			CancelReplaceCount10s: 0,
			CancelCount10s:        0,
			NewOrdersCount10s:     0,
		}
		verdict, err := risk.Evaluate(l.cfg, riskInput)
		if err != nil {
			return err
		}
		decision.RiskVerdict = &verdict
		if err := l.emitStage(runID, cycleID, observability.RISK_VERDICT, sym, string(verdict.Verdict)); err != nil {
			return err
		}
		if err := l.persistDecision(ctx, runID, cycleID, decision); err != nil {
			return err
		}
		if decision.Intent == contracts.IntentEntry && decision.RiskVerdict != nil && decision.RiskVerdict.Verdict == contracts.RiskAllow {
			if decision.AIGate != nil && (decision.AIGate.Verdict == contracts.AIGateBlock || decision.AIGate.Verdict == contracts.AIGateError) {
				continue
			}
			if l.orderClient == nil || l.ledger == nil {
				return fmt.Errorf("order execution deps missing")
			}
			if err := l.emitStage(runID, cycleID, observability.EXECUTE_INTENT, sym, "entry submit"); err != nil {
				return err
			}
			resp, intentID, reason, err := executor.ExecuteEntry(ctx, l.ledger, l.orderClient, decision, bundle.Snapshot.Metadata.SnapshotHash, runID, cycleID, l.now())
			if err != nil {
				if reason == reasoncodes.INTENT_SENT_UNKNOWN && l.intentLookup != nil && l.db != nil {
					intent, ierr := sqlite.GetOrderIntent(ctx, l.db, intentID)
					if ierr != nil {
						return ierr
					}
					if rerr := executor.ResolveSentUnknown(ctx, l.cfg, l.ledger, l.intentLookup, intent); rerr != nil {
						return rerr
					}
				}
				_ = l.emitOrderSubmit(runID, cycleID, decision, intentID, resp, reason)
				return err
			}
			if err := l.emitOrderSubmit(runID, cycleID, decision, intentID, resp, reason); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l *Loop) stageSequence() []observability.StageName {
	return []observability.StageName{
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
	}
}

func (l *Loop) UpdateWSLastMsg(ts time.Time) {
	l.wsLastMsgMs = ts.UnixMilli()
}

func (l *Loop) UpdateRESTLastSuccess(ts time.Time) {
	l.restLastSuccessMs = ts.UnixMilli()
}

func (l *Loop) UpdateDiskFreeBytes(bytes int64) {
	l.diskFreeBytes = bytes
}

func (l *Loop) UpdateAuditWriterLagMs(ms int) {
	l.auditWriterLagMs = ms
}

func (l *Loop) RequestExit() {
	l.forceExit = true
}

func (l *Loop) emitStage(runID string, cycleID string, stage observability.StageName, symbol string, summary string) error {
	now := l.now()
	l.lastProgressMs = now.UnixMilli()
	if l.reporter != nil {
		l.reporter.StageChanged(now, stage, cycleID, "", symbol, summary)
	}
	record := audit.Record{
		Event: auditdomain.AuditEvent{
			TsMs:            now.UnixMilli(),
			RunID:           runID,
			CycleID:         cycleID,
			Mode:            l.cfg.Mode,
			Stage:           stage,
			EventType:       auditdomain.STAGE_CHANGED,
			Reasons:         []reasoncodes.ReasonCode{},
			SnapshotID:      "",
			DecisionID:      "",
			OrderIntentID:   "",
			ExchangeTimeMs:  0,
			LocalReceivedMs: now.UnixMilli(),
		},
		Data: map[string]any{
			"symbol":  symbol,
			"summary": summary,
		},
	}
	if err := l.writer.Write(record); err != nil {
		return fmt.Errorf("audit stage write: %w", err)
	}
	return nil
}

func (l *Loop) refreshSysMode(ctx context.Context, runID string, cycleID string) error {
	if err := l.sampleDiskFree(); err != nil {
		return err
	}
	queueLen, queueCap := l.writer.QueueStats()
	queuePct := 0
	if queueCap > 0 {
		queuePct = int(float64(queueLen) / float64(queueCap) * 100)
	}
	signals := health.Signals{
		NowMs:              l.now().UnixMilli(),
		LastProgressMs:     l.lastProgressMs,
		WsLastMsgMs:        l.wsLastMsgMs,
		RestLastSuccessMs:  l.restLastSuccessMs,
		DiskFreeBytes:      l.diskFreeBytes,
		AuditQueuePct:      queuePct,
		AuditWriterLagMs:   l.auditWriterLagMs,
		ForceExitRequested: l.forceExit,
	}
	result := l.sysEval.Evaluate(l.sysMode, signals)
	if result.Mode == l.sysMode {
		return nil
	}
	l.sysMode = result.Mode
	l.sysModeSince = l.now()
	l.sysModeReasons = result.Reasons
	if l.sysMode == health.SysModeDegrade {
		return l.emitSysModeChange(runID, cycleID, observability.DEGRADE, reasoncodes.ENTER_DEGRADE, result.Reasons, signals)
	}
	if l.sysMode == health.SysModePause {
		return l.emitSysModeChange(runID, cycleID, observability.PAUSE, reasoncodes.ENTER_PAUSE, result.Reasons, signals)
	}
	if l.sysMode == health.SysModeExit {
		if err := l.emitSysModeChange(runID, cycleID, observability.SHUTDOWN, reasoncodes.ENTER_EXIT, result.Reasons, signals); err != nil {
			return err
		}
		return fmt.Errorf("exit requested")
	}
	return nil
}

func (l *Loop) emitSysModeChange(runID string, cycleID string, stage observability.StageName, enterReason reasoncodes.ReasonCode, reasons []reasoncodes.ReasonCode, signals health.Signals) error {
	alertReasons := append([]reasoncodes.ReasonCode{reasoncodes.ALERT_RAISED, enterReason}, reasons...)
	alertData := map[string]any{
		"last_progress_ts_ms":     signals.LastProgressMs,
		"ws_last_msg_ts_ms":       signals.WsLastMsgMs,
		"rest_last_success_ts_ms": signals.RestLastSuccessMs,
		"disk_free_bytes":         signals.DiskFreeBytes,
		"audit_queue_pct":         signals.AuditQueuePct,
		"audit_writer_lag_ms":     signals.AuditWriterLagMs,
	}
	if err := l.emitAlert(runID, cycleID, stage, alertReasons, alertData); err != nil {
		return err
	}
	stageReasons := append([]reasoncodes.ReasonCode{enterReason}, reasons...)
	return l.emitStageWithReasons(runID, cycleID, stage, "", "sys_mode change", stageReasons)
}

func (l *Loop) emitAlert(runID string, cycleID string, stage observability.StageName, reasons []reasoncodes.ReasonCode, data map[string]any) error {
	now := l.now()
	if l.reporter != nil {
		l.reporter.StageChanged(now, stage, cycleID, "", "", "ALERT")
	}
	record := audit.Record{
		Event: auditdomain.AuditEvent{
			TsMs:            now.UnixMilli(),
			RunID:           runID,
			CycleID:         cycleID,
			Mode:            l.cfg.Mode,
			Stage:           stage,
			EventType:       auditdomain.ALERT_RAISED,
			Reasons:         reasons,
			SnapshotID:      "",
			DecisionID:      "",
			OrderIntentID:   "",
			ExchangeTimeMs:  0,
			LocalReceivedMs: now.UnixMilli(),
		},
		Data: data,
	}
	if err := l.writer.Write(record); err != nil {
		return fmt.Errorf("audit alert write: %w", err)
	}
	fmt.Printf("ALERT stage=%s reasons=%v\n", stage, reasons)
	return nil
}

func (l *Loop) emitStageWithReasons(runID string, cycleID string, stage observability.StageName, symbol string, summary string, reasons []reasoncodes.ReasonCode) error {
	now := l.now()
	l.lastProgressMs = now.UnixMilli()
	if l.reporter != nil {
		l.reporter.StageChanged(now, stage, cycleID, "", symbol, summary)
	}
	record := audit.Record{
		Event: auditdomain.AuditEvent{
			TsMs:            now.UnixMilli(),
			RunID:           runID,
			CycleID:         cycleID,
			Mode:            l.cfg.Mode,
			Stage:           stage,
			EventType:       auditdomain.STAGE_CHANGED,
			Reasons:         reasons,
			SnapshotID:      "",
			DecisionID:      "",
			OrderIntentID:   "",
			ExchangeTimeMs:  0,
			LocalReceivedMs: now.UnixMilli(),
		},
		Data: map[string]any{
			"symbol":  symbol,
			"summary": summary,
		},
	}
	if err := l.writer.Write(record); err != nil {
		return fmt.Errorf("audit stage write: %w", err)
	}
	return nil
}

func (l *Loop) sampleDiskFree() error {
	freeBytes, err := health.FreeBytes(audit.DefaultSQLitePath)
	if err != nil {
		l.diskFreeBytes = 0
		return nil
	}
	l.diskFreeBytes = freeBytes
	return nil
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (l *Loop) persistSnapshots(ctx context.Context, bundles []state.SnapshotBundle) error {
	for _, bundle := range bundles {
		snapshotJSON, err := hash.CanonicalJSON(bundle.Snapshot)
		if err != nil {
			return err
		}
		rec := sqlite.SnapshotRecord{
			SnapshotID:     bundle.Snapshot.Metadata.SnapshotID,
			Symbol:         bundle.Snapshot.Symbol,
			SnapshotHash:   bundle.Snapshot.Metadata.SnapshotHash,
			ExchangeTimeMs: bundle.Snapshot.Metadata.ExchangeTimeMs,
			LocalReceivedMs: bundle.Snapshot.Metadata.LocalReceivedMs,
			SnapshotJSON:   string(snapshotJSON),
			CreatedAtMs:    bundle.Snapshot.Metadata.LocalReceivedMs,
		}
		if err := sqlite.InsertSnapshot(ctx, l.db, rec); err != nil {
			return err
		}
	}
	return nil
}

func (l *Loop) persistDecision(ctx context.Context, runID string, cycleID string, decision contracts.Decision) error {
	reasons, err := json.Marshal(decision.Reasons)
	if err != nil {
		return err
	}
	riskReasons := []reasoncodes.ReasonCode{}
	riskVerdict := ""
	if decision.RiskVerdict != nil {
		riskReasons = decision.RiskVerdict.Reasons
		riskVerdict = string(decision.RiskVerdict.Verdict)
	}
	riskReasonsJSON, err := json.Marshal(riskReasons)
	if err != nil {
		return err
	}
	aiVerdict := ""
	aiReasons := []reasoncodes.ReasonCode{}
	if decision.AIGate != nil {
		aiVerdict = string(decision.AIGate.Verdict)
		aiReasons = decision.AIGate.Reasons
	}
	aiReasonsJSON, err := json.Marshal(aiReasons)
	if err != nil {
		return err
	}
	decisionJSON, err := hash.CanonicalJSON(decision)
	if err != nil {
		return err
	}
	rec := sqlite.DecisionRecord{
		DecisionID:      decision.DecisionID,
		RunID:           runID,
		CycleID:         cycleID,
		SnapshotID:      decision.SnapshotID,
		Symbol:          decision.Symbol,
		Stage:           string(decision.Stage),
		Intent:          string(decision.Intent),
		RiskVerdict:     riskVerdict,
		ReasonsJSON:     string(reasons),
		RiskReasonsJSON: string(riskReasonsJSON),
		AIVerdict:       aiVerdict,
		AIReasonsJSON:   string(aiReasonsJSON),
		DecisionJSON:    string(decisionJSON),
		CreatedAtMs:     decision.TsMs,
	}
	return sqlite.InsertDecision(ctx, l.db, rec)
}

func (l *Loop) emitOrderSubmit(runID string, cycleID string, decision contracts.Decision, intentID string, resp executor.OrderResponse, reason reasoncodes.ReasonCode) error {
	now := l.now()
	reasons := []reasoncodes.ReasonCode{}
	if reason != "" {
		reasons = append(reasons, reason)
	}
	record := audit.Record{
		Event: auditdomain.AuditEvent{
			TsMs:            now.UnixMilli(),
			RunID:           runID,
			CycleID:         cycleID,
			Mode:            l.cfg.Mode,
			Stage:           observability.EXECUTE_INTENT,
			EventType:       auditdomain.ORDER_SUBMIT,
			Reasons:         reasons,
			SnapshotID:      decision.SnapshotID,
			DecisionID:      decision.DecisionID,
			OrderIntentID:   intentID,
			ExchangeTimeMs:  0,
			LocalReceivedMs: now.UnixMilli(),
		},
		Data: map[string]any{
			"symbol":          decision.Symbol,
			"client_order_id": resp.ClientOrderID,
			"order_id":        resp.OrderID,
			"status":          resp.Status,
			"rejected":        resp.Rejected,
		},
	}
	if err := l.writer.Write(record); err != nil {
		return fmt.Errorf("audit order submit write: %w", err)
	}
	return nil
}

func findBundle(bundles []state.SnapshotBundle, symbol string) (state.SnapshotBundle, bool) {
	for _, bundle := range bundles {
		if bundle.Snapshot.Symbol == symbol {
			return bundle, true
		}
	}
	return state.SnapshotBundle{}, false
}

func filterSnapshotsByRank(snapshots []contracts.Snapshot, ranked []rank.RankedSymbol) []contracts.Snapshot {
	index := map[string]contracts.Snapshot{}
	for _, snap := range snapshots {
		index[snap.Symbol] = snap
	}
	out := make([]contracts.Snapshot, 0, len(ranked))
	for _, item := range ranked {
		if snap, ok := index[item.Symbol]; ok {
			out = append(out, snap)
		}
	}
	return out
}

func symbolsFromRank(ranked []rank.RankedSymbol) []string {
	out := make([]string, 0, len(ranked))
	for _, item := range ranked {
		out = append(out, item.Symbol)
	}
	return out
}

func rankedConfigHash(ranked []rank.RankedSymbol) string {
	if len(ranked) == 0 {
		return ""
	}
	return ranked[0].ConfigHash
}

func (l *Loop) accountBalances() (string, string) {
	if l.provider == nil {
		return "0.00", "0.00"
	}
	info, _, ok := l.provider.AccountInfo()
	if !ok {
		return "0.00", "0.00"
	}
	balance, ok := binance.FindBalance(info, "USDT")
	if !ok {
		return "0.00", "0.00"
	}
	return balance.Free, balance.Locked
}
