12_DECISIONS_LOG (PROJECT DECISIONS LOG)

GOAL
Record important decisions and rule/contract changes.
It does not replace the rule files. It exists for history and traceability.

RECOMMENDED FORMAT (SIMPLE ADR)
- DATE (YYYY-MM-DD)
- TOPIC
- DECISION
- MOTIVATION
- IMPACT (affected files and tables)
- RISKS / MITIGATIONS
DATE: 2026-02-01
TOPIC: Go module path
DECISION: Use module path github.com/RodrigoBeloyanis/livespot
MOTIVATION: Align module path with the configured Git remote to avoid local-only module names.
IMPACT: go.mod
RISKS / MITIGATIONS: If the repository origin changes, update go.mod and record the change here.

DATE: 2026-02-01
TOPIC: Config defaults and validation baseline
DECISION: Implement config defaults per 00_SOURCE_OF_TRUTH and add audit_redacted_json_max_bytes=4096; set live_require_ok_file=false with live_ok_file_path=var/LIVE.ok.
MOTIVATION: Stage 02 requires strict config validation and 07_SECURITY mandates a fixed audit_redacted_json_max_bytes default.
IMPACT: internal\config\config.go, internal\config\validate.go, 00_SOURCE_OF_TRUTH.md
RISKS / MITIGATIONS: If the redacted JSON limit is too small or the LIVE.ok policy changes, adjust defaults and revalidate with a logged decision.


DATE: 2026-02-01
TOPIC: Audit writer queue capacity and SQLite busy_timeout
DECISION: Add audit_writer_queue_capacity=1024 to config defaults and set SQLite busy_timeout to audit_writer_max_lag_ms.
MOTIVATION: Audit sink requires a bounded queue for backpressure and a configured busy_timeout; no explicit values were specified in higher-priority docs.
IMPACT: internal\config\config.go, internal\config\validate.go, internal\config\validate_test.go, 00_SOURCE_OF_TRUTH.md, internal\infra\sqlite\db.go
RISKS / MITIGATIONS: Capacity too small may trigger PAUSE under load; adjust via config defaults with a logged decision if needed.

DATE: 2026-02-01
TOPIC: Strategy/risk defaults and correlation thresholds
DECISION: Add strategy_min_edge_bps=15, strategy_min_edge_bps_fallback=20, and Stage 14 risk defaults (risk_per_trade, exposure, adaptive, churn, quarantine, corr thresholds) to config defaults and 00_SOURCE_OF_TRUTH.md.
MOTIVATION: 03_STRATEGY.md and 04_RISK_ENGINE_RULES.md require explicit deterministic defaults in config.go; 00_SOURCE_OF_TRUTH mandates mirroring defaults there.
IMPACT: internal\config\config.go, internal\config\validate.go, internal\config\validate_test.go, 00_SOURCE_OF_TRUTH.md
RISKS / MITIGATIONS: Defaults may need tuning after live observation; any change must update 00_SOURCE_OF_TRUTH.md and be logged here.

DATE: 2026-02-01
TOPIC: Drawdown unit alignment
DECISION: Implement drawdown limits in USDT (max_drawdown_usdt) to align with RiskLimitsSnapshot contract fields, despite 04_RISK_ENGINE_RULES.md describing percent drawdown.
MOTIVATION: 01_DECISION_CONTRACT.md defines MaxDrawdownUSDT, which is higher authority than 04_RISK_ENGINE_RULES.md; preserving contract avoids schema change in Stage 14.
IMPACT: internal\engine\risk\engine.go, internal\config\config.go, 00_SOURCE_OF_TRUTH.md
RISKS / MITIGATIONS: If percentage drawdown is required later, add a new field with a formal contract update and migration; keep USDT field for backward compatibility.
