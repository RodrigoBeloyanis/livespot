package config

import (
	"os"
)

type Config struct {
	Mode                                string
	AiDec                               int
	LiveRequireOKFile                   bool
	LiveOKFilePath                      string
	LoopStuckMsDegrade                  int
	LoopStuckMsPause                    int
	WsStaleMsDegrade                    int
	WsStaleMsPause                      int
	RestStaleMsDegrade                  int
	RestStaleMsPause                    int
	DiskFreeDegradeBytes                int64
	DiskFreePauseBytes                  int64
	AuditWriterQueueHiWatermark         int
	AuditWriterQueueFull                int
	AuditWriterQueueCapacity            int
	AuditWriterMaxLagMs                 int
	ReconcileRestIntervalMs             int
	ReconcileDriftDegradeX10000         int
	ReconcileDriftPauseX10000           int
	WebuiPort                           int
	WebuiStreamSnapshotIntervalMs       int
	TimeSyncRecvWindowMs                int
	TimeSyncIntervalMs                  int
	ClockDriftMaxMsLive                 int
	ClockDriftMaxMsPaper                int
	DiskHealthSampleIntervalMs          int
	AuditRedactedJSONMaxBytes           int
	RiskPerTradeUSDT                    string
	RiskPerTradeMinUSDT                 string
	RiskPerTradeMaxUSDT                 string
	TopNSize                            int
	TopKSize                            int
	RankWeightLiquidity                 float64
	RankWeightMomentum                  float64
	RankWeightSpread                    float64
	RankMinQuoteVolume24hUSDT           int64
	RankMinTrades24h                    int
	RankMinPriceChangeBps               int
	DeepWeightEdge                      float64
	DeepWeightRegime                    float64
	DeepWeightMicrostructure            float64
	DeepWeightVolatility                float64
	DeepMinEdgeBps                      int
	DeepMaxSpreadBps                    int
	DeepMinImbalanceX10000              int
	CorrMaxX10000                       int
	CorrWindowPoints                    int
	ChurnGuardEnabled                   bool
	ChurnGuardMinCycles                 int
	ChurnGuardMinScoreDeltaX10000       int
	StrategyID                          string
	StrategyVersion                     string
	StrategyTrendThresholdX10000        int
	StrategyRangeThresholdX10000        int
	StrategyMinEdgeBps                  int
	StrategyMinEdgeBpsFallback          int
	StrategyMaxSpreadEntryBps           int
	StrategyMaxDeltaSpreadBps10s        int
	StrategyMinImbalanceBuyX10000       int
	StrategyPullbackMinBpsEMA20         int
	StrategyPullbackMaxBpsEMA20         int
	StrategyVolumeRatioWindow5m         int
	StrategyMinVolumeRatio5m            float64
	StrategyWeightTrend                 float64
	StrategyWeightPullback              float64
	StrategyWeightMicrostruct           float64
	StrategyWeightVolume                float64
	StrategyEntryMakerTTLSeconds        int
	StrategyEntryMakerRepriceMax        int
	StrategyEntryFallbackMaxSpreadBps   int
	StrategyEntryMaxSlippageBps         int
	StrategyEntryFallbackKind           string
	StrategyExitKATRTrend               float64
	StrategyExitMATRTrend               float64
	StrategyExitKATRRange               float64
	StrategyExitMATRRange               float64
	StrategyExitTrailingEnableProfitBps int
	StrategyExitTrailingTrendMinX10000  int
	StrategyExitTrailingTATRTrend       float64
	StrategyExitTrailingTATRRange       float64
	StrategyExitTrailingMaxSpreadBps    int
}

func Default() Config {
	return Config{
		Mode:                                "LIVE",
		AiDec:                               2,
		LiveRequireOKFile:                   false,
		LiveOKFilePath:                      "var/LIVE.ok",
		LoopStuckMsDegrade:                  5000,
		LoopStuckMsPause:                    15000,
		WsStaleMsDegrade:                    2000,
		WsStaleMsPause:                      10000,
		RestStaleMsDegrade:                  10000,
		RestStaleMsPause:                    60000,
		DiskFreeDegradeBytes:                1073741824,
		DiskFreePauseBytes:                  536870912,
		AuditWriterQueueHiWatermark:         80,
		AuditWriterQueueFull:                95,
		AuditWriterQueueCapacity:            1024,
		AuditWriterMaxLagMs:                 5000,
		ReconcileRestIntervalMs:             5000,
		ReconcileDriftDegradeX10000:         20000,
		ReconcileDriftPauseX10000:           50000,
		WebuiPort:                           8787,
		WebuiStreamSnapshotIntervalMs:       1000,
		TimeSyncRecvWindowMs:                5000,
		TimeSyncIntervalMs:                  300000,
		ClockDriftMaxMsLive:                 500,
		ClockDriftMaxMsPaper:                2000,
		DiskHealthSampleIntervalMs:          5000,
		AuditRedactedJSONMaxBytes:           4096,
		RiskPerTradeUSDT:                    "100.00",
		RiskPerTradeMinUSDT:                 "10.00",
		RiskPerTradeMaxUSDT:                 "500.00",
		TopNSize:                            20,
		TopKSize:                            3,
		RankWeightLiquidity:                 0.55,
		RankWeightMomentum:                  0.30,
		RankWeightSpread:                    0.15,
		RankMinQuoteVolume24hUSDT:           5000000,
		RankMinTrades24h:                    10000,
		RankMinPriceChangeBps:               30,
		DeepWeightEdge:                      0.45,
		DeepWeightRegime:                    0.20,
		DeepWeightMicrostructure:            0.20,
		DeepWeightVolatility:                0.15,
		DeepMinEdgeBps:                      20,
		DeepMaxSpreadBps:                    20,
		DeepMinImbalanceX10000:              5500,
		CorrMaxX10000:                       8000,
		CorrWindowPoints:                    72,
		ChurnGuardEnabled:                   true,
		ChurnGuardMinCycles:                 6,
		ChurnGuardMinScoreDeltaX10000:       1000,
		StrategyID:                          "REGIME_PULLBACK_MAKER_V1",
		StrategyVersion:                     "1.0.0",
		StrategyTrendThresholdX10000:        7000,
		StrategyRangeThresholdX10000:        6000,
		StrategyMinEdgeBps:                  15,
		StrategyMinEdgeBpsFallback:          20,
		StrategyMaxSpreadEntryBps:           20,
		StrategyMaxDeltaSpreadBps10s:        5,
		StrategyMinImbalanceBuyX10000:       5500,
		StrategyPullbackMinBpsEMA20:         10,
		StrategyPullbackMaxBpsEMA20:         80,
		StrategyVolumeRatioWindow5m:         12,
		StrategyMinVolumeRatio5m:            1.20,
		StrategyWeightTrend:                 0.55,
		StrategyWeightPullback:              0.20,
		StrategyWeightMicrostruct:           0.15,
		StrategyWeightVolume:                0.10,
		StrategyEntryMakerTTLSeconds:        30,
		StrategyEntryMakerRepriceMax:        2,
		StrategyEntryFallbackMaxSpreadBps:   25,
		StrategyEntryMaxSlippageBps:         25,
		StrategyEntryFallbackKind:           "IOC_LIMIT",
		StrategyExitKATRTrend:               1.50,
		StrategyExitMATRTrend:               2.00,
		StrategyExitKATRRange:               1.00,
		StrategyExitMATRRange:               1.50,
		StrategyExitTrailingEnableProfitBps: 50,
		StrategyExitTrailingTrendMinX10000:  6000,
		StrategyExitTrailingTATRTrend:       1.00,
		StrategyExitTrailingTATRRange:       0.80,
		StrategyExitTrailingMaxSpreadBps:    25,
	}
}

func Load() (Config, error) {
	cfg := Default()
	if err := Validate(cfg, os.Stat); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
