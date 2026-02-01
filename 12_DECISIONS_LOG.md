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

DATE: 2026-02-01
TOPIC: Decision/order intent ID prefixes
DECISION: Use decision_id prefix "dec_" and order_intent_id prefix "oi_" with SHA-256 hex of canonical payloads.
MOTIVATION: Contract requires deterministic IDs but does not specify prefix format; this keeps IDs URL-safe and readable.
IMPACT: internal\\engine\\executor\\intent_id.go, internal\\engine\\strategy\\strategy.go
RISKS / MITIGATIONS: If a different prefix format is mandated later, update builders and record a migration plan.

DATE: 2026-02-01
TOPIC: Deterministic client_order_id derivation for TP/SL
DECISION: Derive TP/SL client_order_id from order_intent_id with suffixes "_TP" and "_SL" before base32 hashing.
MOTIVATION: Contract requires deterministic IDs for protection orders but does not specify a derivation for TP/SL.
IMPACT: internal\\engine\\executor\\intent_id.go, internal\\engine\\strategy\\strategy.go
RISKS / MITIGATIONS: If execution policy mandates a different derivation, update the derivation and record the change.

DATE: 2026-02-01
TOPIC: Strategy/risk defaults and correlation thresholds
DECISION: Add strategy_min_edge_bps=15, strategy_min_edge_bps_fallback=20, and Stage 14 risk defaults (risk_per_trade, exposure, adaptive, churn, quarantine, corr thresholds) to config defaults and 00_SOURCE_OF_TRUTH.md.
MOTIVATION: 03_STRATEGY.md and 04_RISK_ENGINE_RULES.md require explicit deterministic defaults in config.go; 00_SOURCE_OF_TRUTH mandates mirroring defaults there.
IMPACT: internal\\config\\config.go, internal\\config\\validate.go, internal\\config\\validate_test.go, 00_SOURCE_OF_TRUTH.md
RISKS / MITIGATIONS: Defaults may need tuning after live observation; any change must update 00_SOURCE_OF_TRUTH.md and be logged here.

DATE: 2026-02-01
TOPIC: Drawdown unit alignment
DECISION: Implement drawdown limits in USDT (max_drawdown_usdt) to align with RiskLimitsSnapshot contract fields, despite 04_RISK_ENGINE_RULES.md describing percent drawdown.
MOTIVATION: 01_DECISION_CONTRACT.md defines MaxDrawdownUSDT, which is higher authority than 04_RISK_ENGINE_RULES.md; preserving contract avoids schema change in Stage 14.
IMPACT: internal\\engine\\risk\\engine.go, internal\\config\\config.go, 00_SOURCE_OF_TRUTH.md
RISKS / MITIGATIONS: If percentage drawdown is required later, add a new field with a formal contract update and migration; keep USDT field for backward compatibility.
TOPIC: Soak mode and readiness report
DECISION: Add offline soak mode with mocked exchange and a deterministic readiness report (config_validated, audit_writer_ok, soak_pass).
MOTIVATION: Stage 20 requires a long soak that validates liveness/health signals without network calls and a deterministic readiness summary.
IMPACT: cmd\soak\main.go, internal\e2e\*, README.md, 10_OPERATIONS_RULES.md, 09_CODE_STRUCTURE.md
RISKS / MITIGATIONS: Soak uses synthetic fixtures and cannot validate live exchange behavior; keep Live checklist and real audits mandatory for production.

DATE: 2026-02-01
TOPIC: AI Gate defaults
DECISION: Add ai_gate_timeout_ms=8000, ai_gate_model=gpt-4o-mini, and openai_base_url=https://api.openai.com/v1.
MOTIVATION: AI Gate requires deterministic API timeouts/model selection and a single base URL for OpenAI calls.
IMPACT: internal\config\config.go, internal\config\validate.go, 00_SOURCE_OF_TRUTH.md
RISKS / MITIGATIONS: If model or base URL changes, update config defaults and record the change here.
