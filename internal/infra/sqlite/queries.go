package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

type OrderIntentRecord struct {
	OrderIntentID           string
	RunID                   string
	CycleID                 string
	Mode                    string
	DecisionID              string
	Symbol                  string
	Action                  string
	ClientOrderID           string
	IntentPayloadJSON       string
	State                   string
	ExchangeOrderID         string
	ExchangeOCOID           string
	LastErrorCode           string
	LastErrorDetailRedacted string
	CreatedAtMs             int64
	UpdatedAtMs             int64
}

func InsertOrderIntent(ctx context.Context, db *sql.DB, rec OrderIntentRecord) error {
	_, err := db.ExecContext(ctx, `INSERT INTO order_intents (
  order_intent_id, run_id, cycle_id, mode, decision_id, symbol, action, client_order_id,
  intent_payload_json, state, exchange_order_id, exchange_oco_id, last_error_code, last_error_detail_redacted,
  created_at_ms, updated_at_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.OrderIntentID,
		rec.RunID,
		rec.CycleID,
		rec.Mode,
		rec.DecisionID,
		rec.Symbol,
		rec.Action,
		rec.ClientOrderID,
		rec.IntentPayloadJSON,
		rec.State,
		nullIfEmpty(rec.ExchangeOrderID),
		nullIfEmpty(rec.ExchangeOCOID),
		nullIfEmpty(rec.LastErrorCode),
		nullIfEmpty(rec.LastErrorDetailRedacted),
		rec.CreatedAtMs,
		rec.UpdatedAtMs,
	)
	if err != nil {
		return fmt.Errorf("insert order_intent: %w", err)
	}
	return nil
}

func UpdateOrderIntentState(ctx context.Context, db *sql.DB, id string, state string, exchangeOrderID string, exchangeOCOID string, lastErrorCode string, lastErrorDetailRedacted string, updatedAtMs int64) error {
	_, err := db.ExecContext(ctx, `UPDATE order_intents
SET state = ?, exchange_order_id = ?, exchange_oco_id = ?, last_error_code = ?, last_error_detail_redacted = ?, updated_at_ms = ?
WHERE order_intent_id = ?`,
		state,
		nullIfEmpty(exchangeOrderID),
		nullIfEmpty(exchangeOCOID),
		nullIfEmpty(lastErrorCode),
		nullIfEmpty(lastErrorDetailRedacted),
		updatedAtMs,
		id,
	)
	if err != nil {
		return fmt.Errorf("update order_intent: %w", err)
	}
	return nil
}

func GetOrderIntent(ctx context.Context, db *sql.DB, id string) (OrderIntentRecord, error) {
	row := db.QueryRowContext(ctx, `SELECT order_intent_id, run_id, cycle_id, mode, decision_id, symbol, action, client_order_id,
  intent_payload_json, state, exchange_order_id, exchange_oco_id, last_error_code, last_error_detail_redacted,
  created_at_ms, updated_at_ms
FROM order_intents WHERE order_intent_id = ?`, id)
	return scanOrderIntent(row)
}

func ListOrderIntentsByState(ctx context.Context, db *sql.DB, state string, limit int) ([]OrderIntentRecord, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := db.QueryContext(ctx, `SELECT order_intent_id, run_id, cycle_id, mode, decision_id, symbol, action, client_order_id,
  intent_payload_json, state, exchange_order_id, exchange_oco_id, last_error_code, last_error_detail_redacted,
  created_at_ms, updated_at_ms
FROM order_intents WHERE state = ? ORDER BY updated_at_ms ASC LIMIT ?`, state, limit)
	if err != nil {
		return nil, fmt.Errorf("list order_intents: %w", err)
	}
	defer rows.Close()
	var out []OrderIntentRecord
	for rows.Next() {
		rec, err := scanOrderIntent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list order_intents rows: %w", err)
	}
	return out, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOrderIntent(row rowScanner) (OrderIntentRecord, error) {
	var rec OrderIntentRecord
	var exchangeOrderID sql.NullString
	var exchangeOCOID sql.NullString
	var lastErrorCode sql.NullString
	var lastErrorDetail sql.NullString
	if err := row.Scan(
		&rec.OrderIntentID,
		&rec.RunID,
		&rec.CycleID,
		&rec.Mode,
		&rec.DecisionID,
		&rec.Symbol,
		&rec.Action,
		&rec.ClientOrderID,
		&rec.IntentPayloadJSON,
		&rec.State,
		&exchangeOrderID,
		&exchangeOCOID,
		&lastErrorCode,
		&lastErrorDetail,
		&rec.CreatedAtMs,
		&rec.UpdatedAtMs,
	); err != nil {
		return OrderIntentRecord{}, fmt.Errorf("scan order_intent: %w", err)
	}
	rec.ExchangeOrderID = exchangeOrderID.String
	rec.ExchangeOCOID = exchangeOCOID.String
	rec.LastErrorCode = lastErrorCode.String
	rec.LastErrorDetailRedacted = lastErrorDetail.String
	return rec, nil
}

func nullIfEmpty(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}
