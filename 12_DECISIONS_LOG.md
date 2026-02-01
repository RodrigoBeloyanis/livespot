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
DATE: 2026-02-01
TOPIC: Binance WS base URL default
DECISION: Use wss://stream.binance.com:9443 as the default WS base URL when none is provided.
MOTIVATION: Required to implement Stage 10 WS subscriptions; no higher-priority document specifies a base URL.
IMPACT: internal\\infra\\binance\\ws.go
RISKS / MITIGATIONS: If the official WS endpoint changes or regional endpoints are required, pass a different BaseURL via client options and record the change here.

DATE: 2026-02-01
TOPIC: Binance REST base URL default
DECISION: Use https://api.binance.com as the default REST base URL when none is provided.
MOTIVATION: Required to implement the Stage 09 REST client; no higher-priority document specifies a base URL.
IMPACT: internal\\infra\\binance\\rest.go
RISKS / MITIGATIONS: If the official base URL changes or regional endpoints are required, pass a different BaseURL via client options and record the change here.

DATE: 2026-02-01
TOPIC: Snapshot persistence table
DECISION: Add snapshots table with snapshot_json storage keyed by snapshot_id.
MOTIVATION: Stage 11 requires snapshot persistence for audit reconstruction; schema details were not specified.
IMPACT: migrations/0002_snapshots.sql
RISKS / MITIGATIONS: If query needs expand (cycle_id/run_id) or payload size becomes an issue, evolve schema with a new migration and record the change here.

DATE: 2026-02-01
TOPIC: Snapshot timestamp bounds scope
DECISION: Apply rest_stale_ms_pause/time_sync_recv_window_ms bounds to snapshot-level timestamps (metadata, market_24h, returns_series), not historical candle timestamps.
MOTIVATION: Candle timestamps are intentionally historical and would always violate a 60s staleness window; the spec does not clarify scope.
IMPACT: internal\\engine\\state\\snapshot_validator.go
RISKS / MITIGATIONS: If a stricter policy is required, update validator to include candles and adjust config; log the change here.

DATE: 2026-02-01
TOPIC: Stage 12 ranking normalization defaults
DECISION: Normalize ranking components by max value in the eligible set (volume, momentum) and invert spread by max spread in the set for score calculations.
MOTIVATION: Stage 12 required weights and thresholds but did not specify normalization; this provides deterministic, scale-free scoring.
IMPACT: internal\\engine\\rank\\topn.go
RISKS / MITIGATIONS: If a fixed scale or alternative normalization is required, update the ranking function and record the change here.

DATE: 2026-02-01
TOPIC: Stage 12 eligibility rules default
DECISION: Eligibility requires symbol_status=TRADING, filters_ok=true, ws_ok=true, not quarantined, and Market24h thresholds (volume, trades, price_change).
MOTIVATION: Health flags are present in the Snapshot spec but eligibility rules were not explicit; these are conservative safety defaults.
IMPACT: internal\\engine\\universe\\scan.go
RISKS / MITIGATIONS: If eligibility should be looser or based on different signals, update rules and record the change here.

DATE: 2026-02-01
TOPIC: Stage 12 deep-scan edge estimate
DECISION: Estimate edge_bps using atr14_5m_bps minus (spread_current_bps + maker+taker fees + entry/exit slippage).
MOTIVATION: Deep-scan weights included an edge component but no deterministic formula was specified for Stage 12.
IMPACT: internal\\engine\\deepscan\\deepscan.go
RISKS / MITIGATIONS: Replace with the Strategy edge_bps_expected when available and record the change here.
