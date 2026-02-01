CREATE TABLE IF NOT EXISTS universe_scans (
  run_id TEXT NOT NULL,
  cycle_id TEXT NOT NULL,
  symbol TEXT NOT NULL,
  eligible INTEGER NOT NULL,
  reasons_json TEXT NOT NULL,
  config_hash TEXT NOT NULL,
  created_at_ms INTEGER NOT NULL,
  PRIMARY KEY (run_id, cycle_id, symbol)
);

CREATE INDEX IF NOT EXISTS idx_universe_scans_cycle
  ON universe_scans (run_id, cycle_id, created_at_ms);

CREATE TABLE IF NOT EXISTS cycle_rankings (
  run_id TEXT NOT NULL,
  cycle_id TEXT NOT NULL,
  ranking_stage TEXT NOT NULL,
  symbol TEXT NOT NULL,
  rank_index INTEGER NOT NULL,
  score INTEGER NOT NULL,
  features_json TEXT NOT NULL,
  config_hash TEXT NOT NULL,
  created_at_ms INTEGER NOT NULL,
  PRIMARY KEY (run_id, cycle_id, ranking_stage, symbol)
);

CREATE INDEX IF NOT EXISTS idx_cycle_rankings_stage
  ON cycle_rankings (run_id, cycle_id, ranking_stage, rank_index);

CREATE TABLE IF NOT EXISTS cycle_selections (
  run_id TEXT NOT NULL,
  cycle_id TEXT NOT NULL,
  topn_symbols_json TEXT NOT NULL,
  topk_pre_symbols_json TEXT NOT NULL,
  topk_final_symbols_json TEXT NOT NULL,
  churn_guard_applied INTEGER NOT NULL,
  correlation_applied INTEGER NOT NULL,
  max_pairwise_corr_x10000 INTEGER NULL,
  corr_pairs_json TEXT NULL,
  config_hash TEXT NOT NULL,
  created_at_ms INTEGER NOT NULL,
  PRIMARY KEY (run_id, cycle_id)
);
