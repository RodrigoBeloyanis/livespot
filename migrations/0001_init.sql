-- 0001_init.sql
-- Initial audit schema (SQLite)

PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS audit_events (
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
  payload_json TEXT NOT NULL,
  created_at_ms INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_events_run_cycle ON audit_events(run_id, cycle_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_event_type ON audit_events(event_type, ts_ms);
CREATE INDEX IF NOT EXISTS idx_audit_events_decision ON audit_events(decision_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_snapshot ON audit_events(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_audit_events_order_intent ON audit_events(order_intent_id);

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
CREATE INDEX IF NOT EXISTS idx_ai_gate_events_run_cycle ON ai_gate_events(run_id, cycle_id, created_at_ms);
CREATE INDEX IF NOT EXISTS idx_ai_gate_events_decision ON ai_gate_events(decision_id);
CREATE INDEX IF NOT EXISTS idx_ai_gate_events_snapshot ON ai_gate_events(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_ai_gate_events_verdict ON ai_gate_events(verdict);

CREATE TABLE IF NOT EXISTS order_intents (
  order_intent_id TEXT NOT NULL,
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
  updated_at_ms INTEGER NOT NULL,
  PRIMARY KEY (order_intent_id)
);
CREATE INDEX IF NOT EXISTS idx_order_intents_run_cycle ON order_intents(run_id, cycle_id, created_at_ms);
CREATE INDEX IF NOT EXISTS idx_order_intents_decision ON order_intents(decision_id);

CREATE TABLE IF NOT EXISTS health_samples (
  sample_id TEXT NOT NULL,
  run_id TEXT NOT NULL,
  cycle_id TEXT NOT NULL,
  mode TEXT NOT NULL,
  sqlite_file_bytes INTEGER NOT NULL,
  sqlite_wal_bytes INTEGER NOT NULL,
  disk_free_bytes INTEGER NOT NULL,
  db_writer_queue_pct INTEGER NOT NULL,
  db_writer_lag_ms INTEGER NOT NULL,
  ram_bytes INTEGER NOT NULL,
  created_at_ms INTEGER NOT NULL,
  PRIMARY KEY (sample_id)
);
CREATE INDEX IF NOT EXISTS idx_health_samples_run_cycle ON health_samples(run_id, cycle_id, created_at_ms);

CREATE TABLE IF NOT EXISTS cycle_rankings (
  run_id TEXT NOT NULL,
  cycle_id TEXT NOT NULL,
  ranking_stage TEXT NOT NULL,
  symbol TEXT NOT NULL,
  rank_index INTEGER NOT NULL,
  score REAL NOT NULL,
  features_json TEXT NOT NULL,
  config_hash TEXT NOT NULL,
  created_at_ms INTEGER NOT NULL,
  PRIMARY KEY (run_id, cycle_id, ranking_stage, symbol)
);
CREATE INDEX IF NOT EXISTS idx_cycle_rankings_run_cycle_stage ON cycle_rankings(run_id, cycle_id, ranking_stage, rank_index);

CREATE TABLE IF NOT EXISTS cycle_selections (
  run_id TEXT NOT NULL,
  cycle_id TEXT NOT NULL,
  topn_symbols_json TEXT NOT NULL,
  topk_pre_symbols_json TEXT NOT NULL,
  topk_final_symbols_json TEXT NOT NULL,
  churn_guard_applied INTEGER NOT NULL,
  correlation_applied INTEGER NOT NULL,
  max_pairwise_corr REAL NULL,
  corr_pairs_json TEXT NULL,
  config_hash TEXT NOT NULL,
  created_at_ms INTEGER NOT NULL,
  PRIMARY KEY (run_id, cycle_id)
);

CREATE TABLE IF NOT EXISTS symbol_health (
  symbol TEXT NOT NULL,
  quarantined_until_ms INTEGER NOT NULL,
  recent_rejects_count INTEGER NOT NULL,
  last_error_code TEXT NULL,
  updated_at_ms INTEGER NOT NULL,
  PRIMARY KEY (symbol)
);
