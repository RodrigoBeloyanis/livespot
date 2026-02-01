package config

import (
	"fmt"
	"io/fs"
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
	if err := requirePositiveInt("ai_gate_timeout_ms", cfg.AIGateTimeoutMs); err != nil {
		return err
	}
	if err := requireNonEmpty("ai_gate_model", cfg.AIGateModel); err != nil {
		return err
	}
	if err := requireNonEmpty("openai_base_url", cfg.OpenAIBaseURL); err != nil {
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

func requireNonEmpty(field string, v string) error {
	if v == "" {
		return ValidationError{Field: field, Message: "missing"}
	}
	return nil
}
