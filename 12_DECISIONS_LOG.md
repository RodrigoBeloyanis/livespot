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
TOPIC: Stage 12 ranking normalization defaults
DECISION: Normalize ranking components by max value in the eligible set (volume, momentum) and invert spread by max spread in the set for score calculations.
MOTIVATION: Stage 12 required weights and thresholds but did not specify normalization; this provides deterministic, scale-free scoring.
IMPACT: internal\engine\rank\topn.go
RISKS / MITIGATIONS: If a fixed scale or alternative normalization is required, update the ranking function and record the change here.

DATE: 2026-02-01
TOPIC: Stage 12 eligibility rules default
DECISION: Eligibility requires symbol_status=TRADING, filters_ok=true, ws_ok=true, not quarantined, and Market24h thresholds (volume, trades, price_change).
MOTIVATION: Health flags are present in the Snapshot spec but eligibility rules were not explicit; these are conservative safety defaults.
IMPACT: internal\engine\universe\scan.go
RISKS / MITIGATIONS: If eligibility should be looser or based on different signals, update rules and record the change here.

DATE: 2026-02-01
TOPIC: Stage 12 deep-scan edge estimate
DECISION: Estimate edge_bps using atr14_5m_bps minus (spread_current_bps + maker+taker fees + entry/exit slippage).
MOTIVATION: Deep-scan weights included an edge component but no deterministic formula was specified for Stage 12.
IMPACT: internal\engine\deepscan\deepscan.go
RISKS / MITIGATIONS: Replace with the Strategy edge_bps_expected when available and record the change here.
