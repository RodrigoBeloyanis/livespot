package config

import (
	"fmt"
	"io/fs"
	"math/big"
)

type StatFunc func(string) (fs.FileInfo, error)

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("config invalid: %s: %s", e.Field, e.Message)
}

func Validate(cfg Config, stat StatFunc) error {
	if cfg.Mode == "" {
		return ValidationError{Field: "mode", Message: "missing"}
	}
	if cfg.Mode != "LIVE" {
		return ValidationError{Field: "mode", Message: "only LIVE is supported"}
	}
	if cfg.AiDec < 0 || cfg.AiDec > 2 {
		return ValidationError{Field: "ai_dec", Message: "must be in [0..2]"}
	}
	if cfg.LiveRequireOKFile {
		if cfg.LiveOKFilePath == "" {
			return ValidationError{Field: "live_ok_file_path", Message: "missing"}
		}
		if stat == nil {
			return ValidationError{Field: "live_ok_file_path", Message: "stat function missing"}
		}
		fi, err := stat(cfg.LiveOKFilePath)
		if err != nil {
			return ValidationError{Field: "live_ok_file_path", Message: "file not found"}
		}
		if !fi.Mode().IsRegular() {
			return ValidationError{Field: "live_ok_file_path", Message: "not a regular file"}
		}
	}
	if err := requirePositiveInt("loop_stuck_ms_degrade", cfg.LoopStuckMsDegrade); err != nil {
		return err
	}
	if err := requirePositiveInt("loop_stuck_ms_pause", cfg.LoopStuckMsPause); err != nil {
		return err
	}
	if err := requirePositiveInt("ws_stale_ms_degrade", cfg.WsStaleMsDegrade); err != nil {
		return err
	}
	if err := requirePositiveInt("ws_stale_ms_pause", cfg.WsStaleMsPause); err != nil {
		return err
	}
	if err := requirePositiveInt("rest_stale_ms_degrade", cfg.RestStaleMsDegrade); err != nil {
		return err
	}
	if err := requirePositiveInt("rest_stale_ms_pause", cfg.RestStaleMsPause); err != nil {
		return err
	}
	if err := requirePositiveInt64("disk_free_degrade_bytes", cfg.DiskFreeDegradeBytes); err != nil {
		return err
	}
	if err := requirePositiveInt64("disk_free_pause_bytes", cfg.DiskFreePauseBytes); err != nil {
		return err
	}
	if err := requirePct("audit_writer_queue_hi_watermark_pct", cfg.AuditWriterQueueHiWatermark); err != nil {
		return err
	}
	if err := requirePct("audit_writer_queue_full_pct", cfg.AuditWriterQueueFull); err != nil {
		return err
	}
	if err := requirePositiveInt("audit_writer_queue_capacity", cfg.AuditWriterQueueCapacity); err != nil {
		return err
	}
	if err := requirePositiveInt("audit_writer_max_lag_ms", cfg.AuditWriterMaxLagMs); err != nil {
		return err
	}
	if err := requirePositiveInt("reconcile_rest_interval_ms", cfg.ReconcileRestIntervalMs); err != nil {
		return err
	}
	if err := requirePositiveInt("reconcile_drift_degrade_score_x10000", cfg.ReconcileDriftDegradeX10000); err != nil {
		return err
	}
	if err := requirePositiveInt("reconcile_drift_pause_score_x10000", cfg.ReconcileDriftPauseX10000); err != nil {
		return err
	}
	if err := requirePort("webui_port", cfg.WebuiPort); err != nil {
		return err
	}
	if err := requirePositiveInt("webui_stream_snapshot_interval_ms", cfg.WebuiStreamSnapshotIntervalMs); err != nil {
		return err
	}
	if err := requirePositiveInt("time_sync_recv_window_ms", cfg.TimeSyncRecvWindowMs); err != nil {
		return err
	}
	if err := requirePositiveInt("time_sync_interval_ms", cfg.TimeSyncIntervalMs); err != nil {
		return err
	}
	if err := requirePositiveInt("clock_drift_max_ms_live", cfg.ClockDriftMaxMsLive); err != nil {
		return err
	}
	if err := requirePositiveInt("clock_drift_max_ms_paper", cfg.ClockDriftMaxMsPaper); err != nil {
		return err
	}
	if err := requirePositiveInt("disk_health_sample_interval_ms", cfg.DiskHealthSampleIntervalMs); err != nil {
		return err
	}
	if err := requirePositiveInt("audit_redacted_json_max_bytes", cfg.AuditRedactedJSONMaxBytes); err != nil {
		return err
	}
	if err := requirePositiveInt("strategy_min_edge_bps", cfg.StrategyMinEdgeBps); err != nil {
		return err
	}
	if err := requirePositiveInt("strategy_min_edge_bps_fallback", cfg.StrategyMinEdgeBpsFallback); err != nil {
		return err
	}
	if cfg.StrategyMinEdgeBpsFallback < cfg.StrategyMinEdgeBps {
		return ValidationError{Field: "strategy_min_edge_bps_fallback", Message: "must be >= strategy_min_edge_bps"}
	}
	if err := requireDecimalString("risk_per_trade_usdt", cfg.RiskPerTradeUSDT); err != nil {
		return err
	}
	if err := requireDecimalString("risk_per_trade_min_usdt", cfg.RiskPerTradeMinUSDT); err != nil {
		return err
	}
	if err := requireDecimalString("risk_per_trade_max_usdt", cfg.RiskPerTradeMaxUSDT); err != nil {
		return err
	}
	if err := requireDecimalMinMax("risk_per_trade_usdt", cfg.RiskPerTradeUSDT, cfg.RiskPerTradeMinUSDT, cfg.RiskPerTradeMaxUSDT); err != nil {
		return err
	}
	if err := requireDecimalPositive("risk_max_exposure_symbol_usdt", cfg.RiskMaxExposureSymbolUSDT); err != nil {
		return err
	}
	if err := requireDecimalPositive("risk_max_exposure_total_usdt", cfg.RiskMaxExposureTotalUSDT); err != nil {
		return err
	}
	if err := requireDecimalNonPositive("risk_max_daily_loss_usdt", cfg.RiskMaxDailyLossUSDT); err != nil {
		return err
	}
	if err := requireDecimalPositive("risk_max_drawdown_usdt", cfg.RiskMaxDrawdownUSDT); err != nil {
		return err
	}
	if err := requirePositiveInt("risk_max_open_orders_per_symbol", cfg.RiskMaxOpenOrdersPerSymbol); err != nil {
		return err
	}
	if err := requirePositiveInt("risk_max_open_orders_total", cfg.RiskMaxOpenOrdersTotal); err != nil {
		return err
	}
	if err := requirePositiveInt("risk_max_trades_per_day", cfg.RiskMaxTradesPerDay); err != nil {
		return err
	}
	if err := requirePositiveInt("risk_trades_window_seconds", cfg.RiskTradesWindowSeconds); err != nil {
		return err
	}
	if err := requirePositiveInt("risk_max_trades_per_window", cfg.RiskMaxTradesPerWindow); err != nil {
		return err
	}
	if cfg.RiskCooldownSeconds < 0 {
		return ValidationError{Field: "risk_cooldown_seconds", Message: "must be >= 0"}
	}
	if err := requirePositiveInt("risk_max_consecutive_losses", cfg.RiskMaxConsecutiveLosses); err != nil {
		return err
	}
	if err := requirePositiveInt("risk_ws_latency_threshold_ms", cfg.RiskWSLatencyThresholdMs); err != nil {
		return err
	}
	if err := requirePositiveInt("risk_adaptive_spread_factor_x10000", cfg.RiskAdaptiveSpreadFactorX10000); err != nil {
		return err
	}
	if err := requirePositiveInt("risk_adaptive_volatility_factor_x10000", cfg.RiskAdaptiveVolatilityFactorX10000); err != nil {
		return err
	}
	if err := requirePositiveInt("risk_adaptive_liquidity_floor_x10000", cfg.RiskAdaptiveLiquidityFloorX10000); err != nil {
		return err
	}
	if err := requirePositiveInt("risk_adaptive_max_multiplier_x10000", cfg.RiskAdaptiveMaxMultiplierX10000); err != nil {
		return err
	}
	if err := requirePositiveInt("risk_adaptive_normal_atr_5m_bps", cfg.RiskAdaptiveNormalATR5mBps); err != nil {
		return err
	}
	if err := requirePositiveInt("risk_churn_max_cancel_replace_10s", cfg.RiskChurnMaxCancelReplace10s); err != nil {
		return err
	}
	if err := requirePositiveInt("risk_churn_max_cancel_10s", cfg.RiskChurnMaxCancel10s); err != nil {
		return err
	}
	if err := requirePositiveInt("risk_churn_max_new_orders_10s", cfg.RiskChurnMaxNewOrders10s); err != nil {
		return err
	}
	if err := requirePositiveInt("risk_churn_cooldown_seconds", cfg.RiskChurnCooldownSeconds); err != nil {
		return err
	}
	if err := requirePct("risk_churn_unfilled_order_warning_pct", cfg.RiskChurnUnfilledOrderWarningPct); err != nil {
		return err
	}
	if err := requirePct("risk_churn_unfilled_order_critical_pct", cfg.RiskChurnUnfilledOrderCriticalPct); err != nil {
		return err
	}
	if err := requirePositiveInt("risk_quarantine_max_rejects_per_hour", cfg.RiskQuarantineMaxRejectsPerHour); err != nil {
		return err
	}
	if err := requirePositiveInt("risk_quarantine_max_ws_disconnects_per_10min", cfg.RiskQuarantineMaxWSDisconnectsPer10Min); err != nil {
		return err
	}
	if err := requirePositiveInt("risk_quarantine_max_timeouts_consecutive", cfg.RiskQuarantineMaxTimeoutsConsecutive); err != nil {
		return err
	}
	if err := requirePositiveInt("risk_quarantine_ttl_seconds", cfg.RiskQuarantineTTLSeconds); err != nil {
		return err
	}
	if err := requireX10000("corr_max_x10000", cfg.CorrMaxX10000); err != nil {
		return err
	}
	if err := requirePositiveInt("corr_window_points", cfg.CorrWindowPoints); err != nil {
		return err
	}
	if err := requirePct("corr_missing_max_pct", cfg.CorrMissingMaxPct); err != nil {
		return err
	}
	if err := requirePositiveInt("corr_min_symbols_for_check", cfg.CorrMinSymbolsForCheck); err != nil {
		return err
	}
	return nil
}

func requirePositiveInt(field string, v int) error {
	if v <= 0 {
		return ValidationError{Field: field, Message: "must be > 0"}
	}
	return nil
}

func requirePositiveInt64(field string, v int64) error {
	if v <= 0 {
		return ValidationError{Field: field, Message: "must be > 0"}
	}
	return nil
}

func requirePct(field string, v int) error {
	if v <= 0 || v > 100 {
		return ValidationError{Field: field, Message: "must be in [1..100]"}
	}
	return nil
}

func requirePort(field string, v int) error {
	if v < 1 || v > 65535 {
		return ValidationError{Field: field, Message: "must be in [1..65535]"}
	}
	return nil
}

func requireX10000(field string, v int) error {
	if v <= 0 || v > 10000 {
		return ValidationError{Field: field, Message: "must be in [1..10000]"}
	}
	return nil
}

func requireDecimalString(field string, v string) error {
	if v == "" {
		return ValidationError{Field: field, Message: "missing"}
	}
	if _, ok := parseDecimal(v); !ok {
		return ValidationError{Field: field, Message: "invalid decimal"}
	}
	return nil
}

func requireDecimalMinMax(field string, v string, min string, max string) error {
	val, ok := parseDecimal(v)
	if !ok {
		return ValidationError{Field: field, Message: "invalid decimal"}
	}
	minVal, ok := parseDecimal(min)
	if !ok {
		return ValidationError{Field: field, Message: "invalid min"}
	}
	maxVal, ok := parseDecimal(max)
	if !ok {
		return ValidationError{Field: field, Message: "invalid max"}
	}
	if val.Cmp(minVal) < 0 || val.Cmp(maxVal) > 0 {
		return ValidationError{Field: field, Message: "out of bounds"}
	}
	return nil
}

func requireDecimalPositive(field string, v string) error {
	val, ok := parseDecimal(v)
	if !ok {
		return ValidationError{Field: field, Message: "invalid decimal"}
	}
	if val.Sign() <= 0 {
		return ValidationError{Field: field, Message: "must be > 0"}
	}
	return nil
}

func requireDecimalNonPositive(field string, v string) error {
	val, ok := parseDecimal(v)
	if !ok {
		return ValidationError{Field: field, Message: "invalid decimal"}
	}
	if val.Sign() > 0 {
		return ValidationError{Field: field, Message: "must be <= 0"}
	}
	return nil
}

func parseDecimal(value string) (*big.Rat, bool) {
	r := new(big.Rat)
	if _, ok := r.SetString(value); !ok {
		return nil, false
	}
	return r, true
}
