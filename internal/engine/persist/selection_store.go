package persist

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/engine/deepscan"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/rank"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/topk"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/universe"
)

type SelectionStore struct {
	db *sql.DB
}

func NewSelectionStore(db *sql.DB) *SelectionStore {
	return &SelectionStore{db: db}
}

func (s *SelectionStore) InsertUniverseScans(runID string, cycleID string, results []universe.ScanResult, now time.Time) error {
	for _, result := range results {
		reasons, err := json.Marshal(result.Reasons)
		if err != nil {
			return fmt.Errorf("universe reasons json: %w", err)
		}
		_, err = s.db.Exec(`INSERT INTO universe_scans (
  run_id, cycle_id, symbol, eligible, reasons_json, config_hash, created_at_ms
) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			runID,
			cycleID,
			result.Symbol,
			boolToInt(result.Eligible),
			string(reasons),
			result.ConfigHash,
			now.UnixMilli(),
		)
		if err != nil {
			return fmt.Errorf("universe insert: %w", err)
		}
	}
	return nil
}

func (s *SelectionStore) InsertRankings(runID string, cycleID string, stage string, items []rank.RankedSymbol, now time.Time) error {
	for idx, item := range items {
		features, err := json.Marshal(item.Features)
		if err != nil {
			return fmt.Errorf("ranking features json: %w", err)
		}
		_, err = s.db.Exec(`INSERT INTO cycle_rankings (
  run_id, cycle_id, ranking_stage, symbol, rank_index, score, features_json, config_hash, created_at_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			runID,
			cycleID,
			stage,
			item.Symbol,
			idx,
			item.ScoreX10000,
			string(features),
			item.ConfigHash,
			now.UnixMilli(),
		)
		if err != nil {
			return fmt.Errorf("ranking insert: %w", err)
		}
	}
	return nil
}

func (s *SelectionStore) InsertDeepScan(runID string, cycleID string, items []deepscan.DeepResult, now time.Time) error {
	for idx, item := range items {
		features, err := json.Marshal(item.Features)
		if err != nil {
			return fmt.Errorf("deepscan features json: %w", err)
		}
		_, err = s.db.Exec(`INSERT INTO cycle_rankings (
  run_id, cycle_id, ranking_stage, symbol, rank_index, score, features_json, config_hash, created_at_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			runID,
			cycleID,
			"DEEP",
			item.Symbol,
			idx,
			item.ScoreX10000,
			string(features),
			item.ConfigHash,
			now.UnixMilli(),
		)
		if err != nil {
			return fmt.Errorf("deepscan insert: %w", err)
		}
	}
	return nil
}

func (s *SelectionStore) InsertSelection(runID string, cycleID string, topn []string, topkPre []string, topkFinal []string, churnApplied bool, corrApplied bool, maxCorr int, pairs []topk.PairOverLimit, configHash string, now time.Time) error {
	topnJSON, err := json.Marshal(topn)
	if err != nil {
		return fmt.Errorf("topn json: %w", err)
	}
	topkPreJSON, err := json.Marshal(topkPre)
	if err != nil {
		return fmt.Errorf("topk pre json: %w", err)
	}
	topkFinalJSON, err := json.Marshal(topkFinal)
	if err != nil {
		return fmt.Errorf("topk final json: %w", err)
	}
	pairsJSON, err := json.Marshal(pairs)
	if err != nil {
		return fmt.Errorf("corr pairs json: %w", err)
	}
	_, err = s.db.Exec(`INSERT INTO cycle_selections (
  run_id, cycle_id, topn_symbols_json, topk_pre_symbols_json, topk_final_symbols_json,
  churn_guard_applied, correlation_applied, max_pairwise_corr_x10000, corr_pairs_json,
  config_hash, created_at_ms
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		runID,
		cycleID,
		string(topnJSON),
		string(topkPreJSON),
		string(topkFinalJSON),
		boolToInt(churnApplied),
		boolToInt(corrApplied),
		maxCorr,
		string(pairsJSON),
		configHash,
		now.UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("selection insert: %w", err)
	}
	return nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
