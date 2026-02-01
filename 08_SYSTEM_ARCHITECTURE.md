08_SYSTEM_ARCHITECTURE (OVERVIEW AND COMPONENTS)

TERMINOLOGY (FORMAL SEPARATION)
To avoid confusion in implementation and auditing, these three concepts are distinct:

- StageName: loop stage (e.g., BOOT, STRATEGY_PROPOSE, AIGATE_CALL). Used in console, panel, and the envelope field stage.
- AuditEventType: auditable event type (e.g., STAGE_CHANGED, AIGATE_CALL, ORDER_SUBMIT, ALERT_RAISED). Used in the envelope field event_type.
- ReasonCode: atomic reason (e.g., AIGATE_TIMEOUT, DB_WRITER_QUEUE_HIGH, DISK_LOW_DEGRADE). Used in reasons and in decisions/verdicts.

Rules:
- stage accepts only StageName.
- event_type accepts only AuditEventType.
- reasons accepts only ReasonCode.

24/7 VIEW (CYCLE)
The bot runs continuously and follows this cycle:

1) Universe Scan (USDT by default): scans eligible pairs defined in runtime config.
2) Periodic TopN: recomputes ranking by liquidity/volume + radar momentum.
3) Deep Scan: analyzes candidates and selects 2–3 finalists.
4) Operation: tracks finalists in real time:
 WS -> state -> strategy -> proposed decision -> AI Gate -> risk -> executor -> audit
5) Automatic Entry and Exit: on entry, creates OCO (or OCO -> Trailing).
6) Auditing: persists candles, decisions, orders, fills, and positions.
7) Restart: when positions are closed, return to step 1.

ONLINE PIPELINE (LIVE)
Pipeline (LIVE):
WS -> state -> strategy -> proposed decision -> AI Gate -> risk -> executor -> audit

Responsibilities:
- Strategy: proposes a deterministic and auditable decision.
- AI Gate (OpenAI): returns ALLOW | BLOCK | MODIFY (structured and conservative).
- Risk: deterministic, has the final word (may block even if AI allows).
- Executor: executes orders with idempotency and TTL.
- Audit: persists everything (SQLite) and records a human trail (JSONL), including the AI return.

COMPONENTS (BOX VIEW)
- WS ingestion: consumes bookTicker/ticks, updates state and microstructure windows.
- (Optional) UserDataStream: account events (orders/fills) with keepalive; if unstable, degrade and use REST reconcile.
- State Store: multi-timeframe candles, regime detection, microstructure, microvol, event order.
- Strategy: computes edge_score and proposes Decision (EntryPlan + ExitPlan) with reason_codes.
- AI Gate: validates (ALLOW/BLOCK/conservative MODIFY) and produces an auditable event.
- Risk: validates costs, limits, and policies; may block even if AI allowed.
- Executor: executes orders with idempotency and TTL.
- Position Manager: tracks position, OCO, trailing, and reconcile.
- Audit/Observability: persists SQLite + JSONL + logs with correlation.
- Local alerting: mandatory alerting (highlighted console + panel toast + optional sound) on critical events; always generates AuditEventType=ALERT_RAISED and the corresponding reason_code.

SYSTEM STATE MODES
- NORMAL: full pipeline (scans + entries + management).
- DEGRADE: blocks new entries, keeps management/reconcile/auditing.
- PAUSE: does not operate (no entries), only keeps diagnostics and logs; management follows the policy defined in 05_EXECUTION_AND_FAILSAFE.md.
- EXIT: terminates the process with an auditable reason.

RECONCILIATION AND FINAL TRUTH
- In Live, REST is the final truth.
- Periodic reconcile + immediate reconcile after critical actions (create/cancel OCO, protection swap, trailing).
- Protection must never be left uncovered: any protection swap must follow a sequence with no unprotected gap window (see 05_EXECUTION_AND_FAILSAFE.md).

RECOVERY AFTER CRASH (BOOT)
- Mandatory startup reconcile before new entries.
- Reconstruct position/orders and choose safe behavior (manage-only, close-safely, pause).

STAGE STATUS (CONSOLE SUMMARY)
Objective:
- while running the bot, continuously show in the console a short summary of which stage is currently executing.
- this is for human operation and for Codex debugging (without manual reading of long logs).

Rules:
- every stage change must produce:
 - a short console line (stdout) with: ts, stage, symbol (when applicable), cycle_id, decision_id (when applicable) and summary.
 - an equivalent JSONL event (human audit trail) with event_type=STAGE_CHANGED (AuditEventType).
- the console must display only the current stage (e.g., updating the same line) OR print short lines (defined in runtime config).
- do not print secrets and do not log raw payloads.

Recommended stages (fixed catalog):
- BOOT
- DOCTOR_CHECKS
- STARTUP_RECOVER
- UNIVERSE_SCAN
- RANK_TOPN
- DEEP_SCAN
- WATCHLIST_ATTACH
- STATE_UPDATE
- STRATEGY_PROPOSE
- AIGATE_CALL
- RISK_VERDICT
- EXECUTE_INTENT
- POSITION_MANAGE
- RECONCILE_REST
- REPORT_DAILY_SUMMARY
- DEGRADE
- PAUSE
- SHUTDOWN

Files:
- internal\observability\stage.go : stage enum/const + helpers.
- internal\observability\logger.go: print short stage in console.
- internal\audit\writer.go: emit JSONL STAGE_CHANGED event.

Reason codes: always ReasonCode; use an empty list only when an event has no applicable reason.
- ENTER_DEGRADE
- ENTER_PAUSE
- ENTER_EXIT

STAGE AIGATE_CALL (PRE/POST CONDITIONS AND OUTPUTS)
Objective:
- Apply the conservative gate (OpenAI) on a Decision proposed by Strategy before Risk.
- Guarantee determinism, auditability, strict schema, and fail-closed when required.

Expected inputs (stage inputs):
- mode: LIVE
- config.ai_dec (AI Gate policy)
- Snapshot already materialized and referenced by snapshot_id
- Decision proposed by Strategy (intent=ENTRY|EXIT|MANAGE) with:
 - entry_plan and exit_plan are mandatory per intent (see 01_DECISION_CONTRACT.md).
 - constraints already resolved/quantized per policy
 - complete reason_codes
- correlation IDs: run_id, cycle_id, snapshot_id, decision_id (pre-AI)

Preconditions (mandatory):
- proposed Decision must be valid against 01_DECISION_CONTRACT.md (fields and invariants).
- DecisionConstraints must be present and coherent (tick/step/minNotional and policy).
- deterministic hashes must be ready for auditing:
 - snapshot_hash computed from the canonical snapshot (RFC 8785 + SHA-256 hex).
 - input_hash computed from the canonical AI Gate payload (RFC 8785 + SHA-256 hex).
- redaction ready:
 - redacted payload available for audit (optional, truncated), never raw payload.
- system state:
 - if in DEGRADE: AIGATE_CALL must not generate new ENTRY; only MANAGE/EXIT (per policy).
 - if in PAUSE: AIGATE_CALL must not be called for new entry decisions.
- mandatory Live rule:
 - in MODE=LIVE with AI_DEC=2, this stage is mandatory and is fail-closed (see 05 and 07).

Stage action (minimum sequence):
1) Emit STAGE event (console + JSONL) with stage=AIGATE_CALL.
2) Build deterministic payload (canonical) with Decision + reduced Snapshot + costs/edge + reasons.
3) Compute input_hash and snapshot_hash (before calling the AI).
4) Call the model with strict timeout and limits.
5) Parse the return via strict schema.
6) Validate AIGateResult invariants and, if verdict=MODIFY:
 - apply local enforcement: MODIFY can only reduce risk/aggressiveness
 - recompute decision_id for the applied result (the model does not define ids)
7) Produce the final AIGateResult (ALLOW|BLOCK|MODIFY or ERROR) and forward to Risk.

Postconditions (mandatory):
- Always produce an AIGateResult to attach to the flow, including on failure:
 - enabled=0 when gate is disabled
 - verdict=ERROR when there was a call/parse/schema failure
- Always emit AI Gate auditing:
 - 1 SQLite row in ai_gate_events (REDACTED)
 - 1 JSONL AIGATE_CALL event (REDACTED)
- Never leak secrets:
 - request/response only redacted and truncated, or hashes only.

Expected outputs (stage outputs):
- AIGateResult populated (contract structure):
 - enabled, verdict, reasons, model, latency_ms, raw_hash, input_hash, snapshot_hash
 - modified_decision only when verdict=MODIFY (and only if applied after local validation)
- Decision forwarded to the next stage:
 - verdict=ALLOW: original Decision goes to RISK_VERDICT
 - verdict=MODIFY applied: modified Decision goes to RISK_VERDICT
 - verdict=BLOCK: Decision goes to RISK_VERDICT (Risk may block directly or keep BLOCK)
 - verdict=ERROR: determined by mode + AI_DEC (see matrix below)

Failure policy (architectural) - LIVE
- In MODE=LIVE with AI_DEC=2:
 - timeout/error/invalid schema/parse failed/invalid MODIFY => fail-closed:
 - system enters PAUSE (no new entries)
 - manage-only per execution policy (see 05)

Reasons and codes (minimum reason_codes related to the stage):
- AIGATE_TIMEOUT
- AIGATE_SCHEMA_INVALID
- AIGATE_PARSE_FAIL
- AIGATE_MODIFY_INVALID
- AIGATE_REASON_UNKNOWN

Stage audit (minimum required):
- JSONL event_type=AIGATE_CALL:
 - envelope: ts_ms, run_id, cycle_id, snapshot_id, decision_id, mode, stage
 - fields: ai_enabled, ai_verdict, ai_reasons, ai_model, ai_latency_ms, ai_raw_hash, ai_input_hash, ai_snapshot_hash, ai_modify_applied, ai_error_code (when present)
- SQLite ai_gate_events (REDACTED):
 - run_id, cycle_id, mode, snapshot_id, snapshot_hash, decision_id, input_hash
 - verdict, reasons_json, model, latency_ms, raw_hash
 - modify_applied and (optional) redacted/truncated request/response/modified_decision

LOCAL ALERTING (LOCAL ON-CALL)
Objective:
- avoid silent failure in 24/7: whenever the system enters a risky state or loses a guarantee, it must actively alert in the local environment.

Minimum channels:
- console: highlighted line with ts_ms, stage, mode, reason_code, and summary
- web panel: visible toast/alert (Bootstrap) and stream record
- sound: optional (beep), enabled by local config; no internet

Minimum alert sources (additional items require a record in 12_DECISIONS_LOG.md):
- transition to DEGRADE, PAUSE, or EXIT
- reconcile drift above tolerance
- low disk / abnormal SQLite/WAL growth
- audit writer queue near the limit or full
- persistent DB_BUSY / loss of write ability
- AI Gate failures in Live when AI_DEC=2 (fail-closed)
- persistent rate limit 429/418
- filters/symbol changed (filters_hash drift, symbol_status not TRADING)
- intent in state SENT_UNKNOWN requiring REST query

Audit rule:
- every alert generates an AuditEventType=ALERT_RAISED with:
 - stage, reason_code, severity, and minimum metadata (e.g., bytes, percentages); symbol must be "" when there is no symbol context.

WEB PANEL (DASHBOARD) - DATA CONTRACT (V1)

OBJECTIVE
Provide a local panel for 24/7 operation, with:
- fast reading of operational state (MODE, SYS_MODE, reason_codes, stage, and ids).
- visibility into alerts and health (disk/WAL/writer/process).
- visibility into execution (pending intents, reconcile drift/diffs, orders/fills).
- visibility into decisions (market/signal, risk, AI Gate, topk) without exposing sensitive content.

SCOPE AND GUARANTEES

WEBUI CONFIGURATION (CANONICAL DEFAULTS)
Source of truth: 00_SOURCE_OF_TRUTH.md (MISSING INFORMATION - EXPLICIT DEFAULTS).
- webui_port: 8787 (bind 127.0.0.1 by default)
- webui_stream_snapshot_interval_ms: 1000


- Default bind: 127.0.0.1 (local only).
- LIVE is READ-ONLY: no route can execute mutations when MODE=LIVE.
- Alert ACK is UI-only (local browser state); it does NOT generate auditing and does NOT change audit-log truth.
- The panel displays only persisted data (SQLite) and local process metrics (Go runtime + filesystem).
- The panel does NOT display raw AI Gate prompts/responses; only redacted metadata.

LIVE ENDPOINT POLICY (WEBUI, MODE=LIVE)

Definition:
- READ-ONLY endpoint: serves data from local SQLite and local process metrics only.
- MUTATION endpoint: any route that would (directly or indirectly) submit/cancel/replace orders, modify system state, modify configuration, or write operational state.

Rules (fail-closed):
- MODE is LIVE. The panel must remain strictly read-only in Live.
- Allowed endpoints (allowlist, READ-ONLY):
  - GET /dashboard
  - GET /api/dashboard
  - GET /api/orders
  - GET /api/stream
- Blocked by default:
  - Any method other than GET returns HTTP 403.
  - Any unknown/undeclared route returns HTTP 404.
  - Any route that could trigger mutations MUST NOT exist; if present, it must return HTTP 403 and must have zero side effects.
- A READ-ONLY endpoint must never call the exchange and must never create/cancel/replace intents. Data is served from SQLite only.

HTTP ENDPOINTS (MANDATORY)

1) GET /dashboard
- Returns the panel HTML (static).

2) GET /api/dashboard
- Returns a full snapshot (DashboardSnapshot) for UI rendering.
- Must be used by polling and SSE (snapshot event).

3) GET /api/orders?limit=N
- Returns a recent list of orders (OrderRow), ordered by ts_ms desc.
- N must be integer >= 1 and <= 500. If absent, use N=100.

- In MODE=LIVE it is allowed (READ-ONLY): it must read from local SQLite only and must have zero side effects.
- On success it returns HTTP 200. It must never call the exchange and must never create/cancel/replace intents.

STREAM (REAL TIME) - SSE (MANDATORY)
Endpoint:
- GET /api/stream (Content-Type: text/event-stream)

Rules:
- Must emit a "snapshot" event every webui_stream_snapshot_interval_ms.
- Must emit "ping" every 15000 ms to keep the connection alive.
- If the loop is delayed and misses one or more intervals: emit exactly one snapshot on the next loop cycle (no catch-up burst).
- Payloads must be valid JSON (UTF-8), one line per event.

SSE events:
- event: snapshot
 data: DashboardSnapshot
- event: alert
 data: AlertAggregateRow
 Note: optional event to reduce toast latency; truth remains the snapshot and the audit log.
- event: ping
 data: {"ts_ms": <int64>}

JSON CONTRACT - DashboardSnapshot (MANDATORY)

Basic types:
- ts_ms: int64 (epoch milliseconds)
- id: string (ASCII; empty "" when missing)
- bytes: int64
- bps: int32 (integer; bps * 1)
- score_x10000: int32 (integer; real value * 10000)
- pct_x10000: int32 (integer; percent * 10000)

Enums (strings):
- mode: "LIVE"
- sys_mode: "NORMAL" | "DEGRADE" | "PAUSE" | "EXIT"
- severity: "INFO" | "WARN" | "CRIT"
- writer_pressure: "OK" | "HIGH" | "FULL"
- reconcile_drift_label: "OK" | "WARN" | "BAD"
- aigate_verdict: "ALLOW" | "BLOCK" | "MODIFY"

Schema (mandatory fields):
{
 "ts_ms": <int64>,
 "mode": "LIVE",
 "sys_mode": "NORMAL|DEGRADE|PAUSE|EXIT",
 "sys_mode_since_ts_ms": <int64>,
 "active_reason_codes": [<ReasonCode>...],

 "uptime_s": <int64>,
 "run_id": "<string>",
 "cycle_id": "<string>",
 "decision_id": "<string>",
 "stage": "<StageName>",
 "stage_since_ts_ms": <int64>,

 "alerts_active": [AlertAggregateRow...],
 "stage_history": [StageHistoryRow...],

 "health": HealthSnapshot,
 "intents": IntentsSnapshot,
 "reconcile": ReconcileSnapshot,

 "market": MarketSnapshot,
 "risk": RiskSnapshot,
 "aigate": AIGateSnapshot,
 "topk": TopKSnapshot
}

AlertAggregateRow:
{
 "severity": "INFO|WARN|CRIT",
 "reason_code": "<ReasonCode>",
 "count": <int64>,
 "first_ts_ms": <int64>,
 "last_ts_ms": <int64>,
 "stage": "<StageName>",
 "cycle_id": "<string>",
 "decision_id": "<string>",
 "order_intent_id": "<string>"
}

StageHistoryRow (max 20 items; ordered by ts_ms desc):
{
 "ts_ms": <int64>,
 "stage": "<StageName>",
 "cycle_id": "<string>",
 "decision_id": "<string>",
 "symbol": "<string>",
 "summary": "<string>"
}

HealthSnapshot:
{
 "disk_free_bytes": <int64>,
 "sqlite_bytes": <int64>,
 "wal_bytes": <int64>,

 "process_rss_bytes": <int64>,
 "process_cpu_pct_x10000": <int32>,

 "audit_writer_queue_pct_x10000": <int32>,
 "audit_writer_queue_len": <int64>,
 "audit_writer_lag_ms": <int64>,
 "audit_writer_pressure": "OK|HIGH|FULL",

 "events_per_min": <int64>,
 "drops_per_min": <int64>,

 "last_cycle_ts_ms": <int64>,
 "last_writer_commit_ts_ms": <int64>
}

IntentsSnapshot:
{
 "pending_by_state": {
 "CREATED": <int64>,
 "SENT_UNKNOWN": <int64>,
 "CONFIRMED": <int64>,
 "NOT_FOUND": <int64>,
 "FAILED_TERMINAL": <int64>
 },
 "recent": [IntentRow...]
}

IntentRow (max webui_intents_recent_limit items; ordered by last_ts_ms desc):
{
 "order_intent_id": "<string>",
 "symbol": "<string>",
 "intent_kind": "<string>",
 "state": "CREATED|SENT_UNKNOWN|CONFIRMED|NOT_FOUND|FAILED_TERMINAL",
 "last_ts_ms": <int64>,
 "cycle_id": "<string>",
 "decision_id": "<string>",
 "summary": "<string>"
}

ReconcileSnapshot:
{
 "last_reconcile_rest_ts_ms": <int64>,
 "last_reconcile_rest_dur_ms": <int64>,
 "drift_score_x10000": <int32>,
 "drift_label": "OK|WARN|BAD",
 "recent_diffs": [ReconcileDiffRow...]
}

ReconcileDiffRow (max webui_reconcile_diffs_recent_limit items; ordered by ts_ms desc):
{
 "ts_ms": <int64>,
 "symbol": "<string>",
 "cycle_id": "<string>",
 "decision_id": "<string>",
 "summary": "<string>"
}

MarketSnapshot:
{
 "symbols": [MarketSymbolRow...]
}

MarketSymbolRow (for watchlist and/or cycle topk; ordered by internal score desc):
{
 "symbol": "<string>",
 "regime": "TREND|RANGE",
 "trend_score_x10000": <int32>,
 "range_score_x10000": <int32>,

 "spread_current_bps": <int32>,
 "spread_p50_bps": <int32>,
 "spread_p90_bps": <int32>,
 "delta_spread_bps_p90_10s": <int32>,
 "imbalance_x10000": <int32>,

 "edge_bps_expected": <int32>,
 "costs_bps": {"fees_bps": <int32>, "slippage_bps": <int32>, "spread_bps": <int32>},
 "blocked_reason_code": "<ReasonCode>",

 "signal_flags": [<ReasonCode>...]
}

RiskSnapshot (minimum; UI may expand in the future while keeping compatibility):
{
 "exposure_total_quote": "<decimal_string>",
 "exposure_by_symbol_quote": [{"symbol":"<string>","quote":"<decimal_string>"}...],
 "locked_quote": "<decimal_string>",
 "free_quote": "<decimal_string>",

 "daily_pnl_realized_quote": "<decimal_string>",
 "daily_pnl_unrealized_quote": "<decimal_string>",
 "daily_loss_limit_remaining_quote": "<decimal_string>",
 "drawdown_quote": "<decimal_string>",
 "trades_today": <int64>,

 "cooldowns": [{"symbol":"<string>","cooldown_until_ts_ms":<int64>,"trades_per_hour":<int64>,"loss_streak":<int64>}...]
}

AIGateSnapshot (redacted):
{
 "last_call_ts_ms": <int64>,
 "latency_ms": <int64>,
 "model": "<string>",
 "verdict": "ALLOW|BLOCK|MODIFY",
 "reason_codes": [<ReasonCode>...],
 "input_hash": "<hex>",
 "snapshot_hash": "<hex>",
 "raw_hash": "<hex>",
 "modify_applied": <bool>,
 "error_kind": "<string>"
}
Rules:
- Never include prompt, response, headers, tokens, full URLs, or data outside the Snapshot.

TopKSnapshot:
{
 "cycle_id": "<string>",
 "items": [TopKItemRow...],
 "max_pairwise_corr_x10000": <int32>,
 "pairs_over_limit": [{"a":"<string>","b":"<string>","corr_x10000":<int32>,"action":"<string>"}...]
}

TopKItemRow:
{
 "symbol": "<string>",
 "score_x10000": <int32>,
 "features_top": [{"name":"<string>","value_x10000":<int32>}...],
 "churn_guard_applied": <bool>
}

SIZE LIMITS (MANDATORY)
- stage_history: 20 items.
- intents.recent: webui_intents_recent_limit items (default 50; range 10..200).
- reconcile.recent_diffs: webui_reconcile_diffs_recent_limit items (default 50; range 10..200).
- market.symbols: webui_market_symbols_limit items (default 50; range 10..200).
- topk.items: limited by topk configured in Strategy (03_STRATEGY.md).

DATA SOURCES (MANDATORY)
- SQLite (audit.sqlite) is the source for: alerts_active, stage_history, intents, reconcile_diffs, orders/fills, summaries.
- Filesystem (os.Stat) is the source for: sqlite_bytes, wal_bytes, disk_free_bytes.
- Go runtime is the source for: rss, cpu, uptime, and local timestamps.

FILES (MAP)
- cmd\livespot\main.go: starts web server (flag/config).
- internal\webui\server.go : HTTP server and routes.
- internal\webui\api.go : JSON + SSE handlers; applies MODE gates.
- internal\webui\queries.go : aggregated panel queries (calls infra sqlite).
- internal\webui\static\dashboard.html : Bootstrap page.
- internal\webui\static\app.js : SSE/polling consumption and rendering.
- internal\webui\static\app.css : visual adjustments.

REFERENCES
- Contracts: 01_DECISION_CONTRACT.md
- Snapshot: 02_DATA_SNAPSHOT_SPEC.md
- Strategy: 03_STRATEGY.md
- Risk: 04_RISK_ENGINE_RULES.md
- Execution/failsafe: 05_EXECUTION_AND_FAILSAFE.md
- Auditing: 06_AUDIT_RULES.md
- Operations: 10_OPERATIONS_RULES.md
