package webui

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/config"
	auditdomain "github.com/RodrigoBeloyanis/livespot/internal/domain/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
)

type LatestStage struct {
	TsMs       int64
	Stage      string
	RunID      string
	CycleID    string
	DecisionID string
}

func QueryLatestStage(ctx context.Context, db *sql.DB) (LatestStage, error) {
	row := db.QueryRowContext(ctx, `SELECT ts_ms, stage, run_id, cycle_id, decision_id
FROM audit_events WHERE event_type = ? ORDER BY ts_ms DESC LIMIT 1`, auditdomain.STAGE_CHANGED)
	var out LatestStage
	if err := row.Scan(&out.TsMs, &out.Stage, &out.RunID, &out.CycleID, &out.DecisionID); err != nil {
		if err == sql.ErrNoRows {
			return LatestStage{}, nil
		}
		return LatestStage{}, err
	}
	return out, nil
}

func QueryStageHistory(ctx context.Context, db *sql.DB, limit int) ([]StageHistoryRow, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := db.QueryContext(ctx, `SELECT ts_ms, stage, cycle_id, decision_id, data_json
FROM audit_events WHERE event_type = ? ORDER BY ts_ms DESC LIMIT ?`, auditdomain.STAGE_CHANGED, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []StageHistoryRow
	for rows.Next() {
		var tsMs int64
		var stage string
		var cycleID string
		var decisionID string
		var dataJSON string
		if err := rows.Scan(&tsMs, &stage, &cycleID, &decisionID, &dataJSON); err != nil {
			return nil, err
		}
		var payload map[string]any
		_ = json.Unmarshal([]byte(dataJSON), &payload)
		symbol, _ := payload["symbol"].(string)
		summary, _ := payload["summary"].(string)
		out = append(out, StageHistoryRow{
			TsMs:       tsMs,
			Stage:      stage,
			CycleID:    cycleID,
			DecisionID: decisionID,
			Symbol:     symbol,
			Summary:    summary,
		})
	}
	return out, nil
}

func QueryAlerts(ctx context.Context, db *sql.DB, limit int) ([]AlertAggregateRow, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := db.QueryContext(ctx, `SELECT ts_ms, stage, cycle_id, decision_id, order_intent_id, reasons_json
FROM audit_events WHERE event_type = ? ORDER BY ts_ms DESC LIMIT ?`, auditdomain.ALERT_RAISED, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AlertAggregateRow
	for rows.Next() {
		var tsMs int64
		var stage string
		var cycleID string
		var decisionID string
		var orderIntentID string
		var reasonsJSON string
		if err := rows.Scan(&tsMs, &stage, &cycleID, &decisionID, &orderIntentID, &reasonsJSON); err != nil {
			return nil, err
		}
		reason := ""
		var reasons []string
		_ = json.Unmarshal([]byte(reasonsJSON), &reasons)
		if len(reasons) > 0 {
			reason = reasons[0]
		}
		out = append(out, AlertAggregateRow{
			Severity:      "WARN",
			ReasonCode:    reason,
			Count:         1,
			FirstTsMs:     tsMs,
			LastTsMs:      tsMs,
			Stage:         stage,
			CycleID:       cycleID,
			DecisionID:    decisionID,
			OrderIntentID: orderIntentID,
		})
	}
	return out, nil
}

func QueryIntents(ctx context.Context, db *sql.DB, limit int) (IntentsSnapshot, error) {
	if limit <= 0 {
		limit = 50
	}
	pending := map[string]int64{
		"CREATED":         0,
		"SENT_UNKNOWN":    0,
		"CONFIRMED":       0,
		"NOT_FOUND":       0,
		"FAILED_TERMINAL": 0,
	}
	rows, err := db.QueryContext(ctx, `SELECT state, COUNT(*) FROM order_intents GROUP BY state`)
	if err == nil {
		for rows.Next() {
			var state string
			var count int64
			if err := rows.Scan(&state, &count); err == nil {
				pending[state] = count
			}
		}
		_ = rows.Close()
	}
	recentRows, err := db.QueryContext(ctx, `SELECT order_intent_id, symbol, action, state, updated_at_ms, cycle_id, decision_id
FROM order_intents ORDER BY updated_at_ms DESC LIMIT ?`, limit)
	if err != nil {
		return IntentsSnapshot{PendingByState: pending, Recent: []IntentRow{}}, nil
	}
	defer recentRows.Close()
	var recent []IntentRow
	for recentRows.Next() {
		var id, symbol, action, state, cycleID, decisionID string
		var ts int64
		if err := recentRows.Scan(&id, &symbol, &action, &state, &ts, &cycleID, &decisionID); err != nil {
			return IntentsSnapshot{}, err
		}
		recent = append(recent, IntentRow{
			OrderIntentID: id,
			Symbol:        symbol,
			IntentKind:    action,
			State:         state,
			LastTsMs:      ts,
			CycleID:       cycleID,
			DecisionID:    decisionID,
			Summary:       "",
		})
	}
	return IntentsSnapshot{PendingByState: pending, Recent: recent}, nil
}

func QueryReconcile(ctx context.Context, db *sql.DB, limit int) (ReconcileSnapshot, error) {
	if limit <= 0 {
		limit = 50
	}
	recentRows, err := db.QueryContext(ctx, `SELECT ts_ms, cycle_id, decision_id, data_json
FROM audit_events WHERE event_type = ? ORDER BY ts_ms DESC LIMIT ?`, auditdomain.RECONCILE_DIFF, limit)
	if err != nil {
		return ReconcileSnapshot{RecentDiffs: []ReconcileDiffRow{}, DriftLabel: "OK"}, nil
	}
	defer recentRows.Close()
	var recent []ReconcileDiffRow
	var driftScore int32
	for recentRows.Next() {
		var ts int64
		var cycleID string
		var decisionID string
		var dataJSON string
		if err := recentRows.Scan(&ts, &cycleID, &decisionID, &dataJSON); err != nil {
			return ReconcileSnapshot{}, err
		}
		var payload map[string]any
		_ = json.Unmarshal([]byte(dataJSON), &payload)
		if val, ok := payload["drift_score_x10000"].(float64); ok {
			driftScore = int32(val)
		}
		recent = append(recent, ReconcileDiffRow{
			TsMs:       ts,
			Symbol:     "",
			CycleID:    cycleID,
			DecisionID: decisionID,
			Summary:    "",
		})
	}
	label := "OK"
	if len(recent) > 0 {
		label = "WARN"
	}
	alertRows, err := db.QueryContext(ctx, `SELECT reasons_json FROM audit_events WHERE event_type = ? ORDER BY ts_ms DESC LIMIT 50`, auditdomain.ALERT_RAISED)
	if err == nil {
		for alertRows.Next() {
			var reasonsJSON string
			if err := alertRows.Scan(&reasonsJSON); err == nil {
				var reasons []string
				_ = json.Unmarshal([]byte(reasonsJSON), &reasons)
				for _, r := range reasons {
					if r == string(reasoncodes.DRIFT_LIMIT_EXCEEDED) {
						label = "BAD"
						break
					}
				}
			}
		}
		_ = alertRows.Close()
	}
	return ReconcileSnapshot{
		LastReconcileRestTsMs:  0,
		LastReconcileRestDurMs: 0,
		DriftScoreX10000:       driftScore,
		DriftLabel:             label,
		RecentDiffs:            recent,
	}, nil
}

func QueryAIGate(ctx context.Context, db *sql.DB) (AIGateSnapshot, error) {
	row := db.QueryRowContext(ctx, `SELECT created_at_ms, latency_ms, model, verdict, reasons_json, input_hash, snapshot_hash, raw_hash, modify_applied, error_code
FROM ai_gate_events ORDER BY created_at_ms DESC LIMIT 1`)
	var ts int64
	var latency sql.NullInt64
	var model sql.NullString
	var verdict sql.NullString
	var reasonsJSON sql.NullString
	var inputHash sql.NullString
	var snapshotHash sql.NullString
	var rawHash sql.NullString
	var modifyApplied sql.NullInt64
	var errorCode sql.NullString
	if err := row.Scan(&ts, &latency, &model, &verdict, &reasonsJSON, &inputHash, &snapshotHash, &rawHash, &modifyApplied, &errorCode); err != nil {
		if err == sql.ErrNoRows {
			return AIGateSnapshot{ReasonCodes: []string{}}, nil
		}
		return AIGateSnapshot{}, err
	}
	reasons := []string{}
	if reasonsJSON.Valid {
		_ = json.Unmarshal([]byte(reasonsJSON.String), &reasons)
	}
	return AIGateSnapshot{
		LastCallTsMs:  ts,
		LatencyMs:     int(latency.Int64),
		Model:         model.String,
		Verdict:       verdict.String,
		ReasonCodes:   reasons,
		InputHash:     inputHash.String,
		SnapshotHash:  snapshotHash.String,
		RawHash:       rawHash.String,
		ModifyApplied: modifyApplied.Int64 == 1,
		ErrorKind:     errorCode.String,
	}, nil
}

type OrderRow struct {
	TsMs   int64  `json:"ts_ms"`
	Symbol string `json:"symbol"`
	Side   string `json:"side"`
	Qty    string `json:"qty"`
	Price  string `json:"price"`
	Status string `json:"status"`
}

func QueryRecentOrders(ctx context.Context, db *sql.DB, limit int) ([]OrderRow, error) {
	return []OrderRow{}, nil
}

func DeriveSysMode(alerts []AlertAggregateRow) (string, int64, []string) {
	if len(alerts) == 0 {
		return "NORMAL", 0, []string{}
	}
	for _, alert := range alerts {
		if alert.ReasonCode == string(reasoncodes.ENTER_EXIT) {
			return "EXIT", alert.LastTsMs, []string{alert.ReasonCode}
		}
		if alert.ReasonCode == string(reasoncodes.ENTER_PAUSE) {
			return "PAUSE", alert.LastTsMs, []string{alert.ReasonCode}
		}
		if alert.ReasonCode == string(reasoncodes.ENTER_DEGRADE) {
			return "DEGRADE", alert.LastTsMs, []string{alert.ReasonCode}
		}
	}
	return "NORMAL", 0, []string{}
}

func BuildHealthSnapshot(cfg config.Config, start time.Time, writer *audit.Writer) HealthSnapshot {
	queueLen, queueCap := 0, 1
	if writer != nil {
		queueLen, queueCap = writer.QueueStats()
	}
	queuePct := int32(0)
	if queueCap > 0 {
		queuePct = int32((queueLen * 10000) / queueCap)
	}
	pressure := "OK"
	if queuePct >= int32(cfg.AuditWriterQueueFull*100) {
		pressure = "FULL"
	} else if queuePct >= int32(cfg.AuditWriterQueueHiWatermark*100) {
		pressure = "HIGH"
	}
	return HealthSnapshot{
		DiskFreeBytes:             0,
		SqliteBytes:               0,
		WalBytes:                  0,
		ProcessRSSBytes:           0,
		ProcessCPUPctX10000:       0,
		AuditWriterQueuePctX10000: queuePct,
		AuditWriterQueueLen:       int64(queueLen),
		AuditWriterLagMs:          0,
		AuditWriterPressure:       pressure,
		EventsPerMin:              0,
		DropsPerMin:               0,
		LastCycleTsMs:             0,
		LastWriterCommitTsMs:      0,
	}
}

func DiskStats() (sqliteBytes int64, walBytes int64, diskFreeBytes int64) {
	sqliteBytes = fileSize(audit.DefaultSQLitePath)
	walBytes = fileSize(audit.DefaultSQLitePath + "-wal")
	diskFreeBytes = diskFree(filepath.Dir(audit.DefaultSQLitePath))
	return
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func diskFree(path string) int64 {
	if runtime.GOOS != "windows" {
		return 0
	}
	free, err := diskFreeWindows(path)
	if err != nil {
		return 0
	}
	return free
}

func diskFreeWindows(path string) (int64, error) {
	vol := filepath.VolumeName(path)
	if vol == "" {
		return 0, fmt.Errorf("volume missing")
	}
	var freeBytes int64
	err := getDiskFreeSpaceEx(vol+"\\", &freeBytes)
	return freeBytes, err
}
