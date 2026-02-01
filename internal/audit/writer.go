package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	domainaudit "github.com/RodrigoBeloyanis/livespot/internal/domain/audit"
	"github.com/RodrigoBeloyanis/livespot/internal/infra/sqlite"
)

type Writer struct {
	cfg      config.Config
	db       *sql.DB
	jsonl    *JSONLWriter
	writeCh  chan writeRequest
	fatalErr error
	mu       sync.Mutex
	closed   bool
}

type writeRequest struct {
	event   domainaudit.AuditEvent
	payload []byte
	result  chan error
}

func NewWriter(cfg config.Config) (*Writer, error) {
	if err := config.Validate(cfg, nil); err != nil {
		return nil, err
	}
	db, err := sqlite.Open(cfg.AuditSQLitePath, cfg.AuditSQLiteBusyTimeoutMs)
	if err != nil {
		return nil, err
	}
	if err := sqlite.ApplySchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	jsonl, err := NewJSONLWriter(cfg.AuditJSONLDir)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	w := &Writer{
		cfg:     cfg,
		db:      db,
		jsonl:   jsonl,
		writeCh: make(chan writeRequest),
	}
	go w.loop()
	return w, nil
}

func (w *Writer) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	w.mu.Unlock()

	close(w.writeCh)
	if w.jsonl != nil {
		_ = w.jsonl.Close()
	}
	if w.db != nil {
		return w.db.Close()
	}
	return nil
}

func (w *Writer) WriteEvent(ctx context.Context, event domainaudit.AuditEvent, payload any) error {
	if err := w.checkFatal(); err != nil {
		return err
	}
	if err := event.Validate(); err != nil {
		return err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("audit payload marshal failed: %w", err)
	}
	res := make(chan error, 1)
	req := writeRequest{event: event, payload: payloadJSON, result: res}

	select {
	case w.writeCh <- req:
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case err := <-res:
		if err != nil {
			w.setFatal(err)
		}
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Writer) loop() {
	for req := range w.writeCh {
		req.result <- w.write(req)
	}
}

func (w *Writer) write(req writeRequest) error {
	if err := w.writeSQLite(req.event, req.payload); err != nil {
		return err
	}
	jsonEvent := map[string]any{
		"ts_ms":             req.event.TsMs,
		"run_id":            req.event.RunID,
		"cycle_id":          req.event.CycleID,
		"mode":              req.event.Mode,
		"stage":             req.event.Stage,
		"event_type":        req.event.EventType,
		"reasons":           req.event.Reasons,
		"snapshot_id":       req.event.SnapshotID,
		"decision_id":       req.event.DecisionID,
		"order_intent_id":   req.event.OrderIntentID,
		"exchange_time_ms":  req.event.ExchangeTimeMs,
		"local_received_ms": req.event.LocalReceivedMs,
		"payload":           json.RawMessage(req.payload),
	}
	if err := w.jsonl.WriteEvent(jsonEvent); err != nil {
		return err
	}
	return nil
}

func (w *Writer) writeSQLite(event domainaudit.AuditEvent, payload []byte) error {
	if w.db == nil {
		return fmt.Errorf("audit sqlite not initialized")
	}
	reasonsJSON, err := json.Marshal(event.Reasons)
	if err != nil {
		return fmt.Errorf("audit reasons marshal failed: %w", err)
	}
	_, err = w.db.Exec(`
		INSERT INTO audit_events (
			ts_ms, run_id, cycle_id, mode, stage, event_type, reasons_json,
			snapshot_id, decision_id, order_intent_id, exchange_time_ms, local_received_ms,
			payload_json, created_at_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		event.TsMs,
		event.RunID,
		event.CycleID,
		event.Mode,
		event.Stage,
		event.EventType,
		string(reasonsJSON),
		event.SnapshotID,
		event.DecisionID,
		event.OrderIntentID,
		event.ExchangeTimeMs,
		event.LocalReceivedMs,
		string(payload),
		time.Now().UTC().UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("audit sqlite insert failed: %w", err)
	}
	return nil
}

func (w *Writer) checkFatal() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.fatalErr != nil {
		return w.fatalErr
	}
	if w.closed {
		return errors.New("audit writer closed")
	}
	return nil
}

func (w *Writer) setFatal(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.fatalErr == nil {
		w.fatalErr = err
	}
}
