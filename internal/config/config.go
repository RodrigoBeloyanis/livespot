package config

import (
	"os"
)

type Config struct {
	Mode                                   string
	AiDec                                  int
	LiveRequireOKFile                      bool
	LiveOKFilePath                         string
	LoopStuckMsDegrade                     int
	LoopStuckMsPause                       int
	WsStaleMsDegrade                       int
	WsStaleMsPause                         int
	RestStaleMsDegrade                     int
	RestStaleMsPause                       int
	DiskFreeDegradeBytes                   int64
	DiskFreePauseBytes                     int64
	AuditWriterQueueHiWatermark            int
	AuditWriterQueueFull                   int
	AuditWriterQueueCapacity               int
	AuditWriterMaxLagMs                    int
	ReconcileRestIntervalMs                int
	ReconcileDriftDegradeX10000            int
	ReconcileDriftPauseX10000              int
	WebuiPort                              int
	WebuiStreamSnapshotIntervalMs          int
	TimeSyncRecvWindowMs                   int
	TimeSyncIntervalMs                     int
	ClockDriftMaxMsLive                    int
	ClockDriftMaxMsPaper                   int
	DiskHealthSampleIntervalMs             int
	AuditRedactedJSONMaxBytes              int
	StrategyMinEdgeBps                     int
	StrategyMinEdgeBpsFallback             int
	RiskPerTradeUSDT                       string
	RiskPerTradeMinUSDT                    string
	RiskPerTradeMaxUSDT                    string
	RiskMaxExposureSymbolUSDT              string
	RiskMaxExposureTotalUSDT               string
	RiskMaxDailyLossUSDT                   string
	RiskMaxDrawdownUSDT                    string
	RiskMaxOpenOrdersPerSymbol             int
	RiskMaxOpenOrdersTotal                 int
	RiskMaxTradesPerDay                    int
	RiskTradesWindowSeconds                int
	RiskMaxTradesPerWindow                 int
	RiskCooldownSeconds                    int
	RiskMaxConsecutiveLosses               int
	RiskWSLatencyThresholdMs               int
	RiskAdaptiveSpreadFactorX10000         int
	RiskAdaptiveVolatilityFactorX10000     int
	RiskAdaptiveLiquidityFloorX10000       int
	RiskAdaptiveMaxMultiplierX10000        int
	RiskAdaptiveNormalATR5mBps             int
	RiskChurnMaxCancelReplace10s           int
	RiskChurnMaxCancel10s                  int
	RiskChurnMaxNewOrders10s               int
	RiskChurnCooldownSeconds               int
	RiskChurnUnfilledOrderWarningPct       int
	RiskChurnUnfilledOrderCriticalPct      int
	RiskQuarantineMaxRejectsPerHour        int
	RiskQuarantineMaxWSDisconnectsPer10Min int
	RiskQuarantineMaxTimeoutsConsecutive   int
	RiskQuarantineTTLSeconds               int
	RiskQuarantineAutoRelease              bool
	CorrMaxX10000                          int
	CorrWindowPoints                       int
	CorrMissingMaxPct                      int
	CorrMinSymbolsForCheck                 int
}

func Default() Config {
	return Config{
		Mode:                                   "LIVE",
		AiDec:                                  2,
		LiveRequireOKFile:                      false,
		LiveOKFilePath:                         "var/LIVE.ok",
		LoopStuckMsDegrade:                     5000,
		LoopStuckMsPause:                       15000,
		WsStaleMsDegrade:                       2000,
		WsStaleMsPause:                         10000,
		RestStaleMsDegrade:                     10000,
		RestStaleMsPause:                       60000,
		DiskFreeDegradeBytes:                   1073741824,
		DiskFreePauseBytes:                     536870912,
		AuditWriterQueueHiWatermark:            80,
		AuditWriterQueueFull:                   95,
		AuditWriterQueueCapacity:               1024,
		AuditWriterMaxLagMs:                    5000,
		ReconcileRestIntervalMs:                5000,
		ReconcileDriftDegradeX10000:            20000,
		ReconcileDriftPauseX10000:              50000,
		WebuiPort:                              8787,
		WebuiStreamSnapshotIntervalMs:          1000,
		TimeSyncRecvWindowMs:                   5000,
		TimeSyncIntervalMs:                     300000,
		ClockDriftMaxMsLive:                    500,
		ClockDriftMaxMsPaper:                   2000,
		DiskHealthSampleIntervalMs:             5000,
		AuditRedactedJSONMaxBytes:              4096,
		StrategyMinEdgeBps:                     15,
		StrategyMinEdgeBpsFallback:             20,
		RiskPerTradeUSDT:                       "100.00",
		RiskPerTradeMinUSDT:                    "10.00",
		RiskPerTradeMaxUSDT:                    "500.00",
		RiskMaxExposureSymbolUSDT:              "200.00",
		RiskMaxExposureTotalUSDT:               "500.00",
		RiskMaxDailyLossUSDT:                   "-500.00",
		RiskMaxDrawdownUSDT:                    "500.00",
		RiskMaxOpenOrdersPerSymbol:             1,
		RiskMaxOpenOrdersTotal:                 10,
		RiskMaxTradesPerDay:                    20,
		RiskTradesWindowSeconds:                3600,
		RiskMaxTradesPerWindow:                 3,
		RiskCooldownSeconds:                    300,
		RiskMaxConsecutiveLosses:               3,
		RiskWSLatencyThresholdMs:               1000,
		RiskAdaptiveSpreadFactorX10000:         15000,
		RiskAdaptiveVolatilityFactorX10000:     15000,
		RiskAdaptiveLiquidityFloorX10000:       5000,
		RiskAdaptiveMaxMultiplierX10000:        30000,
		RiskAdaptiveNormalATR5mBps:             50,
		RiskChurnMaxCancelReplace10s:           3,
		RiskChurnMaxCancel10s:                  5,
		RiskChurnMaxNewOrders10s:               5,
		RiskChurnCooldownSeconds:               60,
		RiskChurnUnfilledOrderWarningPct:       80,
		RiskChurnUnfilledOrderCriticalPct:      95,
		RiskQuarantineMaxRejectsPerHour:        3,
		RiskQuarantineMaxWSDisconnectsPer10Min: 5,
		RiskQuarantineMaxTimeoutsConsecutive:   3,
		RiskQuarantineTTLSeconds:               3600,
		RiskQuarantineAutoRelease:              true,
		CorrMaxX10000:                          8500,
		CorrWindowPoints:                       72,
		CorrMissingMaxPct:                      10,
		CorrMinSymbolsForCheck:                 3,
	}
}

func Load() (Config, error) {
	cfg := Default()
	if err := Validate(cfg, os.Stat); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
