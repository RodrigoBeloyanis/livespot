CREATE TABLE IF NOT EXISTS snapshots (
  snapshot_id TEXT NOT NULL PRIMARY KEY,
  symbol TEXT NOT NULL,
  snapshot_hash TEXT NOT NULL,
  exchange_time_ms INTEGER NOT NULL,
  local_received_ms INTEGER NOT NULL,
  snapshot_json TEXT NOT NULL,
  created_at_ms INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_snapshots_symbol_created
  ON snapshots (symbol, created_at_ms);
