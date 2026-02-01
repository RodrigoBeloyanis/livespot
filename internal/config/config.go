package config

import (
	"os"
)

type Config struct {
	Mode                          string
	AiDec                         int
	LiveRequireOKFile             bool
	LiveOKFilePath                string
	LoopStuckMsDegrade            int
	LoopStuckMsPause              int
	WsStaleMsDegrade              int
	WsStaleMsPause                int
	RestStaleMsDegrade            int
	RestStaleMsPause              int
	DiskFreeDegradeBytes          int64
	DiskFreePauseBytes            int64
	AuditWriterQueueHiWatermark   int
	AuditWriterQueueFull          int
	AuditWriterQueueCapacity      int
	AuditWriterMaxLagMs           int
	ReconcileRestIntervalMs       int
	ReconcileDriftDegradeX10000   int
	ReconcileDriftPauseX10000     int
	WebuiPort                     int
	WebuiStreamSnapshotIntervalMs int
	TimeSyncRecvWindowMs          int
	TimeSyncIntervalMs            int
	ClockDriftMaxMsLive           int
	ClockDriftMaxMsPaper          int
	DiskHealthSampleIntervalMs    int
	AuditRedactedJSONMaxBytes     int
}

func Default() Config {
	return Config{
		Mode:                          "LIVE",
		AiDec:                         2,
		LiveRequireOKFile:             false,
		LiveOKFilePath:                "var/LIVE.ok",
		LoopStuckMsDegrade:            5000,
		LoopStuckMsPause:              15000,
		WsStaleMsDegrade:              2000,
		WsStaleMsPause:                10000,
		RestStaleMsDegrade:            10000,
		RestStaleMsPause:              60000,
		DiskFreeDegradeBytes:          1073741824,
		DiskFreePauseBytes:            536870912,
		AuditWriterQueueHiWatermark:   80,
		AuditWriterQueueFull:          95,
		AuditWriterQueueCapacity:      1024,
		AuditWriterMaxLagMs:           5000,
		ReconcileRestIntervalMs:       5000,
		ReconcileDriftDegradeX10000:   20000,
		ReconcileDriftPauseX10000:     50000,
		WebuiPort:                     8787,
		WebuiStreamSnapshotIntervalMs: 1000,
		TimeSyncRecvWindowMs:          5000,
		TimeSyncIntervalMs:            300000,
		ClockDriftMaxMsLive:           500,
		ClockDriftMaxMsPaper:          2000,
		DiskHealthSampleIntervalMs:    5000,
		AuditRedactedJSONMaxBytes:     4096,
	}
}

func Load() (Config, error) {
	cfg := Default()
	if err := Validate(cfg, os.Stat); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
