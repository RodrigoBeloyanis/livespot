CREATE TABLE IF NOT EXISTS decisions (
  decision_id TEXT NOT NULL PRIMARY KEY,
  run_id TEXT NOT NULL,
  cycle_id TEXT NOT NULL,
  snapshot_id TEXT NOT NULL,
  symbol TEXT NOT NULL,
  stage TEXT NOT NULL,
  intent TEXT NOT NULL,
  risk_verdict TEXT NOT NULL,
  reasons_json TEXT NOT NULL,
  risk_reasons_json TEXT NOT NULL,
  ai_verdict TEXT NOT NULL,
  ai_reasons_json TEXT NOT NULL,
  decision_json TEXT NOT NULL,
  created_at_ms INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_decisions_cycle
  ON decisions (cycle_id, created_at_ms);
