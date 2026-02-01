09_CODE_STRUCTURE (CODE ORGANIZATION)

GOAL
Define folder structure, responsibilities, and conventions.
This file guides refactors and helps Codex navigate without confusion.

AUTHORITY REFERENCES
- 00_SOURCE_OF_TRUTH.md
- 01_DECISION_CONTRACT.md
- 02_DATA_SNAPSHOT_SPEC.md
- 03_STRATEGY.md
- 04_RISK_ENGINE_RULES.md
- 05_EXECUTION_AND_FAILSAFE.md
- 06_AUDIT_RULES.md
- 07_SECURITY.md
- 08_SYSTEM_ARCHITECTURE.md
- 10_OPERATIONS_RULES.md
- 11_INVARIANTS_MAP.md
- 12_DECISIONS_LOG.md

PROFESSIONAL FOLDER AND FILE STRUCTURE (WITH RESPONSIBILITIES)
Root: C:\go\livespot\

TOP-LEVEL FILES
- README.md
  Responsibility: usage manual and execution guide; initial read.
- go.mod / go.sum
  Responsibility: dependencies and Go module version.
- .gitignore
  Responsibility: block secrets and runtime/build files from Git.
- .env.example
  Responsibility: template with names of the 3 variables only (no real secrets).

CMD (BINARIES / TOOLS)
- cmd\livespot\main.go
  Responsibility: 24/7 bot entrypoint; loads config; starts infra; starts the online loop.
- cmd\migrate\main.go
  Responsibility: apply SQLite migrations; create schema and update version.
- cmd\doctor\main.go
  Responsibility: diagnostics (keys, WS/REST, permissions, DB, filters, clock).
- cmd\experiments\main.go
  Responsibility: run parameter grids, measure metrics, and persist results.

INTERNAL\APP (ORCHESTRATION)
- internal\app\app.go
  Responsibility: application composition; builds dependencies and injects them into the loop.
- internal\app\loop.go
  Responsibility: 24/7 main loop; schedules pipeline, reconcile, reports, and failsafes.
- internal\app\shutdown.go
  Responsibility: safe shutdown (flush audit/logs, close WS, close DB, security actions).
- internal\app\live_checklist.go
  Responsibility: checklist and auditable lock to allow Live.
- internal\app\run_ids.go
  Responsibility: initialize run_id and cycle_id and inject into logger/audit.
- internal\app\startup_recover.go
  Responsibility: boot recovery flow (reconcile + intent recovery).

INTERNAL\CONFIG (SOURCE OF TRUTH)
- internal\config\config.go
  Responsibility: ALL operational configuration (except secrets) in one place.
- internal\config\validate.go
  Responsibility: config validation and sanitization.

INTERNAL\DOMAIN (SYSTEM LAW: CONTRACTS/SCHEMAS)
- internal\domain\contracts\decision.go
  Responsibility: Decision + validation; includes EntryPlan, ExitPlan, costs, edge, and reason_codes.
- internal\domain\contracts\snapshot.go
  Responsibility: Snapshot + validation; minimum audit reconstruction data (regime, microstructure, book slice, candles, costs).
- internal\domain\contracts\risk.go
  Responsibility: RiskVerdict + validation; includes portfolio limits and deterministic reasons.
- internal\domain\contracts\ai_gate.go
  Responsibility: AI Gate return + validation; conservative MODIFY.
- internal\domain\audit\event_types.go
  Responsibility: enum/const for AuditEventType (separate from StageName and ReasonCode).
- internal\domain\reasoncodes\codes.go
  Responsibility: central and single reason_codes catalog (failures, market, execution, AI).
- internal\domain\timeframes.go
  Responsibility: supported timeframe enum and helpers.

INTERNAL\ENGINE (DETERMINISTIC PIPELINE)
- internal\engine\universe\...
  Responsibility: eligible universe and persistence of universe_scans.
- internal\engine\rank\...
  Responsibility: TopN ranking and persistence of rank_runs.
- internal\engine\deepscan\...
  Responsibility: deep scan and persistence of deep_scans.
- internal\engine\watchlist\...
  Responsibility: 2–3 symbol watchlist; multi-symbol WS.
- internal\engine\state\...
  Responsibility: state store (ticks, candles, indicators, book summaries).
- internal\engine\health\...
  Responsibility: continuous health checks (disk free, sqlite/wal bytes, writer queue, DB busy) and emission of auditable alerts.
- internal\engine\state\regime.go
  Responsibility: multi-timeframe regime detection.
- internal\engine\state\microstructure.go
  Responsibility: spread median/percentile/delta, imbalance.
- internal\engine\state\microvol.go
  Responsibility: microvol and volatility flags.
- internal\engine\state\event_order.go
  Responsibility: tolerance and auditing of out-of-order events.
- internal\engine\state\candle_store.go
  Responsibility: candle persistence policy.
- internal\engine\strategy\...
  Responsibility: deterministic proposal (with regime/microstructure/edge).
- internal\engine\strategy\edge_score.go
  Responsibility: edge_bps_expected and deterministic score.
- internal\engine\strategy\entry_plan.go
  Responsibility: maker-first TTL + fallback; limit vs market choice.
- internal\engine\strategy\exit_plan.go
  Responsibility: ATR TP/SL and trailing criteria.
- internal\engine\aigate\...
  Responsibility: OpenAI gate and validation.
- internal\engine\aigate\limits.go
  Responsibility: apply MODIFY only when conservative.
- internal\engine\risk\...
  Responsibility: deterministic risk (final word).
- internal\engine\risk\cost_model.go
  Responsibility: shared costs (fees/spread/slippage) for Strategy and Risk.
- internal\engine\risk\entry_limits.go
  Responsibility: limits for fallback and max cost.
- internal\engine\risk\exit_sanity.go
  Responsibility: sanity of TP/SL/trailing vs ATR and regime.
- internal\engine\risk\position_sizing.go
  Responsibility: sizing by risk-per-trade.
- internal\engine\risk\adaptive_thresholds.go
  Responsibility: adaptive min edge by conditions.
- internal\engine\risk\anti_overtrading.go
  Responsibility: cooldowns/limits/blocks per symbol.
- internal\engine\risk\portfolio.go
  Responsibility: exposure limits and kill switch.
- internal\engine\risk\daily_limits.go
  Responsibility: daily limits and counts.
- internal\engine\risk\reconcile_policy.go
  Responsibility: drift thresholds and actions.
- internal\engine\risk\regime_rules.go
  Responsibility: final deterministic regime rules.
- internal\engine\executor\...
  Responsibility: idempotent Live execution.
- internal\engine\executor\orders.go
  Responsibility: create/cancel/replace and OCO; clientOrderId builder.
- internal\engine\executor\quantize.go
  Responsibility: quantization by tick/step/minNotional.
- internal\engine\executor\intent_ledger.go
  Responsibility: intents ledger and idempotent recovery at boot.
- internal\engine\position\...
  Responsibility: position lifecycle and reconcile.
- internal\engine\position\oco.go
  Responsibility: OCO lifecycle.
- internal\engine\position\trailing.go
  Responsibility: trailing lifecycle.
- internal\engine\position\reconcile.go
  Responsibility: REST reconcile (final truth).
- internal\engine\position\recover.go
  Responsibility: position/order reconstruction in startup recover.
- internal\engine\failsafe\policy.go
  Responsibility: failure->action matrix.
- internal\engine\failsafe\handlers.go
  Responsibility: executes failsafe and audits.
- internal\engine\reports\daily_summary.go
  Responsibility: compute daily summary.
- internal\engine\reports\metrics.go
  Responsibility: metrics for experiments and walk-forward.

INTERNAL\INFRA (INTEGRATIONS)
- internal\infra\binance\rest.go / ws.go / models.go
  Responsibility: REST/WS, reconnection, and rate limit handling.
- internal\infra\binance\filters.go
  Responsibility: exchangeInfo+filters per symbol.
- internal\infra\binance\rest_depth.go
  Responsibility: depth snapshots and helpers.
- internal\infra\binance\ratelimit.go
  Responsibility: throttle/backoff and 429 handling.
- internal\infra\openai\client.go / prompt.go
  Responsibility: OpenAI client, prompt builder, structured parsing, redaction.
- internal\infra\sqlite\db.go / migrations.go / queries.go
  Responsibility: SQLite connection, migrations, queries, and integrity checks.
- internal\infra\clock\real.go / fake.go
  Responsibility: real and fake clock.

INTERNAL\AUDIT (AUDITABLE PERSISTENCE)
- internal\audit\writer.go
  Responsibility: consistent writing of events and entities with timestamps and IDs.
- internal\audit\redact.go
  Responsibility: mandatory redaction.
- internal\audit\schema.go
  Responsibility: schema/tables version control.

INTERNAL\OBSERVABILITY
- internal\observability\alerts\alerts.go
  Responsibility: alerts aggregator; routes to console, webui, and optional sound; never drops critical alerts.
- internal\observability\alerts\sinks_console.go
  Responsibility: emit highlighted alerts in the console.
- internal\observability\alerts\sinks_webui.go
  Responsibility: emit toasts/alerts via the panel stream.
- internal\observability\alerts\sinks_sound.go
  Responsibility: optional beep, enabled by local config; no internet.
- internal\observability\logger.go
  Responsibility: file logs and format with IDs.
- internal\observability\correlation.go
  Responsibility: generation/propagation of correlation IDs.
- internal\observability\stage.go
  Responsibility: fixed loop stage catalog and helpers for console/JSONL.
- internal\observability\metrics.go
  Responsibility: basic process metrics for the web panel.

PROMPTS
- prompts\ai_gate_system.txt
  Responsibility: conservative instructions and AI Gate format.
- prompts\ai_gate_user_template.txt
  Responsibility: payload template (costs, edge, regime, microstructure, and ExitPlan).
- prompts\schemas\decision.schema.json
  Responsibility: Decision schema.
- prompts\schemas\snapshot.schema.json
  Responsibility: Snapshot schema.
- prompts\schemas\ai_gate_result.schema.json
  Responsibility: AI Gate result schema (constraints).

MIGRATIONS
- migrations\0001_init.sql
  Responsibility: initial schema.
- migrations\000x_daily_summary.sql
  Responsibility: daily_summary.
- migrations\000x_experiments.sql
  Responsibility: experiments table and results.
- migrations\00xx_order_intents.sql
  Responsibility: order_intents table for idempotency.

SCRIPTS
- scripts\run.ps1
  Responsibility: run the bot.
- scripts\live.ps1
  Responsibility: run with safety locks (require --live and checklist).
- scripts\test.ps1
  Responsibility: tests.
- scripts\doctor.ps1
  Responsibility: diagnostics.

VAR (DO NOT VERSION)
- var\data\audit.sqlite
- var\logs\*.log
- var\logs\audit-YYYY-MM-DD.jsonl
- var\LIVE.ok (optional: external lock for Live)

Cleanup and retention (mandatory in 24/7):
- JSONL: daily rotation already defined; retention defined in local config and automatic deletion.
- .log logs: size-based rotation and retention defined in local config.
- SQLite: daily file backup and optional periodic VACUUM (e.g., weekly), controlled in config.go.

TESTDATA
- testdata\...

CONVENTIONS (NO COMMENTS)
- small functions with names that explain intent
- avoid gigantic files: split by responsibility
- reason_codes are the official explanation of decisions (not comments)
- validate contracts at the boundary (parse/input) and before persisting

TESTS
- unit tests for contracts, cost model, regime/microstructure, quantization, idempotency
- integration tests: partial fills, slippage, OCO/trailing, TTL + fallback, and idempotency edge-cases.
- end-to-end regression tests using deterministic fixtures

NEW FILES MUST
- have a single responsibility
- emit auditable events when they affect execution/decisions
- include tests when they alter contracts/rules

WEB PANEL (WEBUI) - CODE MAP (DASHBOARD)

GOAL
Define where these live:
- dashboard models/DTOs (JSON and SSE).
- aggregated queries (alerts/stages/intents/reconcile/health/market/risk/aigate/topk).
- WebUI handlers/endpoints (HTTP + SSE).
The data contract spec is in 08_SYSTEM_ARCHITECTURE.md (WEB PANEL - DATA CONTRACT V1).

MODE=LIVE (READ-ONLY POLICY)
Rules:
- In MODE=LIVE, the WebUI is for observability only and MUST be read-only.
- The WebUI allowlist in MODE=LIVE is:
  - GET /dashboard
  - GET /api/dashboard
  - GET /api/orders?limit=N
  - GET /api/stream
- Any method other than GET is forbidden in MODE=LIVE and must be rejected (HTTP 403).
- Unknown paths must be rejected (HTTP 404).
- /api/orders is required by the dashboard to show recent orders and MUST remain available in MODE=LIVE.
- WebUI must serve only local persisted data (SQLite) and local process metrics (no remote side effects).


FOLDERS AND FILES (CANONICAL)
- internal\webui\server.go
  Responsibility:
  - start local HTTP server (bind 127.0.0.1 by default).
  - serve static dashboard files.
  - register routes /dashboard, /api/*.

- internal\webui\api.go
  Responsibility:
  - JSON HTTP handlers (read-only):
    - GET /api/dashboard
    - GET /api/orders?limit=N
  - SSE handler (read-only):
    - GET /api/stream
  - mandatory gates (fail-closed):
    - in MODE=LIVE, enforce the WebUI allowlist (see 08_SYSTEM_ARCHITECTURE.md and 07_SECURITY.md).
    - reject any method != GET with HTTP 403.
    - reject unknown paths with HTTP 404.
    - never call Binance REST/WS from WebUI handlers.
    - audit rejected requests with AuditEventType=WEBUI_REQUEST (allowed requests may be sampled).
  - DTOs / JSON contract structs:
    - DashboardSnapshot
    - AlertAggregateRow
    - StageHistoryRow
    - HealthSnapshot
    - IntentsSnapshot + IntentRow
    - ReconcileSnapshot + ReconcileDiffRow
    - MarketSnapshot + MarketSymbolRow
    - RiskSnapshot
    - AIGateSnapshot (redacted)
    - TopKSnapshot + TopKItemRow
  Rules:
  - field names must use json tags per the V1 contract.
  - numeric values must be integers (x10000/bps) to avoid floats.

- internal\webui\queries.go
  Responsibility:
  - aggregated functions to compose DashboardSnapshot in 1 call:
    - QueryAlertsActive()
    - QueryStageHistory(limit=20)
    - QueryHealthSnapshot()
    - QueryIntentsSnapshot(limit=webui_intents_recent_limit)
    - QueryReconcileSnapshot(limit=webui_reconcile_diffs_recent_limit)
    - QueryMarketSnapshot(limit=webui_market_symbols_limit)
    - QueryRiskSnapshot()
    - QueryAIGateSnapshot()
    - QueryTopKSnapshot()
  Rules:
  - queries must be deterministic (explicit ORDER BY).
  - queries must bound size (LIMIT is mandatory).
  - queries must return "zero values" (empty lists, empty strings "") instead of null.

- internal\infra\sqlite\queries.go
  Responsibility:
  - reusable raw SQL (SELECTs) for audit.sqlite.
  - helpers to scan into structs.
  Rules:
  - each query must have a unique name and a regression test when it changes.

- internal\webui\static\dashboard.html
  Responsibility: Bootstrap 5.3 layout (cards/tabs/drawer).

- internal\webui\static\app.js
  Responsibility:
  - connect to /api/stream (SSE).
  - apply local ACK merge (LocalStorage) onto alerts.
  - fallback: poll /api/dashboard if SSE fails.

- internal\webui\static\app.css
  Responsibility: visual tweaks (first fold, badges, drawer).

ENDPOINTS (PER V1 CONTRACT)
- GET /dashboard
  Returns the page.

- GET /api/dashboard
  Returns DashboardSnapshot (JSON).

- GET /api/orders?limit=N
  Returns OrderRow[] (JSON).
  Rules:
  - N default 100.
  - N range 1..500.

- GET /api/stream
  SSE.
  Rules:
  - snapshot event (DashboardSnapshot) every webui_stream_snapshot_interval_ms (source of truth: 00_SOURCE_OF_TRUTH.md).
  - ping event every 15000 ms.

COMPATIBILITY RULES (V1)
- New fields may be added to DashboardSnapshot, but NEVER remove or rename existing V1 fields.
- New enums may be added only if forward-compatible (UI must treat unknown values as "UNKNOWN" and display them).

CHANGES AND RECORDING
- changes to contracts, reason_codes, migrations, and operational policies must be recorded in 12_DECISIONS_LOG.md