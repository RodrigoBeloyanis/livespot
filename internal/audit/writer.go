package audit

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/infra/sqlite"
)

type Writer struct {
	queue         chan writeRequest
	queueCapacity int
	jsonl         *jsonlWriter
	db            *sql.DB
	closed        chan struct{}
	wg            sync.WaitGroup
}

type writeRequest struct {
	record Record
	result chan error
}

type WriterOptions struct {
	DBPath   string
	JSONLDir string
	Now      func() time.Time
}

func NewWriter(cfg config.Config, opts WriterOptions) (*Writer, error) {
	if opts.DBPath == "" {
		opts.DBPath = DefaultSQLitePath
	}
	if opts.JSONLDir == "" {
		opts.JSONLDir = DefaultJSONLDir
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if cfg.AuditWriterQueueCapacity <= 0 {
		return nil, fmt.Errorf("audit writer queue capacity invalid")
	}
	jsonl, err := newJSONLWriter(opts.JSONLDir)
	if err != nil {
		return nil, err
	}
	db, err := sqlite.Open(opts.DBPath, cfg)
	if err != nil {
		_ = jsonl.Close()
		return nil, err
	}
	if err := sqlite.Migrate(db, opts.Now()); err != nil {
		_ = db.Close()
		_ = jsonl.Close()
		return nil, err
	}
	writer := &Writer{
		queue:         make(chan writeRequest, cfg.AuditWriterQueueCapacity),
		queueCapacity: cfg.AuditWriterQueueCapacity,
		jsonl:         jsonl,
		db:            db,
		closed:        make(chan struct{}),
	}
	writer.wg.Add(1)
	go writer.run(opts.Now)
	return writer, nil
}

func (w *Writer) Write(record Record) error {
	if err := record.Validate(); err != nil {
		return err
	}
	select {
	case <-w.closed:
		return errors.New("audit writer closed")
	default:
	}
	req := writeRequest{record: record, result: make(chan error, 1)}
	select {
	case w.queue <- req:
		return <-req.result
	default:
		return errors.New("audit writer queue full")
	}
}

func (w *Writer) QueueStats() (int, int) {
	return len(w.queue), w.queueCapacity
}

func (w *Writer) Close() error {
	select {
	case <-w.closed:
		return nil
	default:
		close(w.closed)
	}
	close(w.queue)
	w.wg.Wait()
	var err error
	if w.jsonl != nil {
		if closeErr := w.jsonl.Close(); closeErr != nil {
			err = closeErr
		}
	}
	if w.db != nil {
		if closeErr := w.db.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}
	return err
}

func (w *Writer) run(now func() time.Time) {
	defer w.wg.Done()
	for req := range w.queue {
		req.result <- w.writeOnce(req.record)
		close(req.result)
	}
}

func (w *Writer) writeOnce(record Record) error {
	dataJSON, err := record.DataJSON()
	if err != nil {
		return fmt.Errorf("audit data json: %w", err)
	}
	reasonsJSON, err := recordReasonsJSON(record)
	if err != nil {
		return err
	}
	if err := w.writeSQLite(record, reasonsJSON, dataJSON); err != nil {
		return err
	}
	line, err := record.JSONLine()
	if err != nil {
		return fmt.Errorf("audit jsonl: %w", err)
	}
	ts := time.UnixMilli(record.Event.TsMs)
	if err := w.jsonl.Write(ts, line); err != nil {
		return err
	}
	return nil
}

func recordReasonsJSON(record Record) (string, error) {
	reasons := reasonStrings(record.Event.Reasons)
	buf, err := json.Marshal(reasons)
	if err != nil {
		return "", fmt.Errorf("audit reasons json: %w", err)
	}
	return string(buf), nil
}

func (w *Writer) writeSQLite(record Record, reasonsJSON string, dataJSON string) error {
	_, err := w.db.Exec(`INSERT INTO audit_events (
  ts_ms, run_id, cycle_id, mode, stage, event_type, reasons_json, snapshot_id,
  decision_id, order_intent_id, exchange_time_ms, local_received_ms, data_json, created_at_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.Event.TsMs,
		record.Event.RunID,
		record.Event.CycleID,
		record.Event.Mode,
		record.Event.Stage,
		record.Event.EventType,
		reasonsJSON,
		record.Event.SnapshotID,
		record.Event.DecisionID,
		record.Event.OrderIntentID,
		record.Event.ExchangeTimeMs,
		record.Event.LocalReceivedMs,
		dataJSON,
		record.Event.TsMs,
	)
	if err != nil {
		return fmt.Errorf("audit sqlite insert: %w", err)
	}
	return nil
}
