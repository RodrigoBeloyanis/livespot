PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS schema_migrations (
  name TEXT NOT NULL PRIMARY KEY,
  applied_at_ms INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS audit_events (
  event_id INTEGER PRIMARY KEY AUTOINCREMENT,
  ts_ms INTEGER NOT NULL,
  run_id TEXT NOT NULL,
  cycle_id TEXT NOT NULL,
  mode TEXT NOT NULL,
  stage TEXT NOT NULL,
  event_type TEXT NOT NULL,
  reasons_json TEXT NOT NULL,
  snapshot_id TEXT NOT NULL,
  decision_id TEXT NOT NULL,
  order_intent_id TEXT NOT NULL,
  exchange_time_ms INTEGER NOT NULL,
  local_received_ms INTEGER NOT NULL,
  data_json TEXT NOT NULL,
  created_at_ms INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_events_run_cycle_ts
  ON audit_events (run_id, cycle_id, ts_ms);

CREATE INDEX IF NOT EXISTS idx_audit_events_event_type
  ON audit_events (event_type, ts_ms);

CREATE TABLE IF NOT EXISTS order_intents (
  order_intent_id TEXT NOT NULL PRIMARY KEY,
  run_id TEXT NOT NULL,
  cycle_id TEXT NOT NULL,
  mode TEXT NOT NULL,
  decision_id TEXT NOT NULL,
  symbol TEXT NOT NULL,
  action TEXT NOT NULL,
  client_order_id TEXT NOT NULL,
  intent_payload_json TEXT NOT NULL,
  state TEXT NOT NULL,
  exchange_order_id TEXT NULL,
  exchange_oco_id TEXT NULL,
  last_error_code TEXT NULL,
  last_error_detail_redacted TEXT NULL,
  created_at_ms INTEGER NOT NULL,
  updated_at_ms INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_order_intents_run_cycle
  ON order_intents (run_id, cycle_id);

CREATE INDEX IF NOT EXISTS idx_order_intents_state
  ON order_intents (state, updated_at_ms);

CREATE TABLE IF NOT EXISTS health_samples (
  sample_id TEXT NOT NULL PRIMARY KEY,
  run_id TEXT NOT NULL,
  cycle_id TEXT NOT NULL,
  mode TEXT NOT NULL,
  sqlite_file_bytes INTEGER NOT NULL,
  sqlite_wal_bytes INTEGER NOT NULL,
  disk_free_bytes INTEGER NOT NULL,
  db_writer_queue_pct INTEGER NOT NULL,
  db_writer_lag_ms INTEGER NOT NULL,
  ram_bytes INTEGER NOT NULL,
  created_at_ms INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_health_samples_run_cycle
  ON health_samples (run_id, cycle_id, created_at_ms);

CREATE TABLE IF NOT EXISTS ai_gate_events (
  run_id TEXT NOT NULL,
  cycle_id TEXT NOT NULL,
  mode TEXT NOT NULL,
  stage TEXT NOT NULL,
  event_type TEXT NOT NULL,
  snapshot_id TEXT NOT NULL,
  snapshot_hash TEXT NOT NULL,
  decision_id TEXT NOT NULL,
  input_hash TEXT NOT NULL,
  enabled INTEGER NOT NULL,
  verdict TEXT NOT NULL,
  reasons_json TEXT NOT NULL,
  model TEXT NULL,
  latency_ms INTEGER NULL,
  raw_hash TEXT NULL,
  request_json_redacted TEXT NULL,
  response_json_redacted TEXT NULL,
  modified_decision_json_redacted TEXT NULL,
  modify_applied INTEGER NOT NULL,
  error_code TEXT NULL,
  error_detail_redacted TEXT NULL,
  exchange_time_ms INTEGER NULL,
  local_received_ms INTEGER NOT NULL,
  created_at_ms INTEGER NOT NULL,
  PRIMARY KEY (run_id, cycle_id, decision_id)
);

CREATE INDEX IF NOT EXISTS idx_ai_gate_events_run_cycle
  ON ai_gate_events (run_id, cycle_id, created_at_ms);

CREATE INDEX IF NOT EXISTS idx_ai_gate_events_decision
  ON ai_gate_events (decision_id);

CREATE INDEX IF NOT EXISTS idx_ai_gate_events_snapshot
  ON ai_gate_events (snapshot_id);

CREATE INDEX IF NOT EXISTS idx_ai_gate_events_verdict
  ON ai_gate_events (verdict);
