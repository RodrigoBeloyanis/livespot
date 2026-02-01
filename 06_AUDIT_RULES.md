06_AUDIT_RULES (AUDITING AND EVENTS)

GOAL
Guarantee that any relevant action is traceable, queryable, and verifiable.
SQLite is the source of truth and JSONL is the human trail.

PRINCIPLES
- SQLite is the source of truth for queries and reconciliation.
- JSONL is a human trail for fast reading, debugging, and operational auditing.
- Every action must have full correlation (run_id, cycle_id, snapshot_id, decision_id, order_intent_id).
- Any payload that may contain secrets must be REDACTED before persistence.
- No silent audit loss: a full queue => DEGRADE/PAUSE with an event.

FORMAL VOCABULARY (DO NOT MIX FIELDS)
Definitions:
- StageName: 24/7 loop stage (e.g., BOOT, UNIVERSE_SCAN, STRATEGY_PROPOSE, AIGATE_CALL, RISK_VERDICT, EXECUTE_INTENT, RECONCILE_REST).
- AuditEventType: audited event type (e.g., STAGE_CHANGED, ALERT_RAISED, ORDER_SUBMIT, ORDER_CANCEL, ORDER_CANCEL_REPLACE, RECONCILE_DIFF, FILTERS_REFRESHED).
- ReasonCode: atomic reason (canonical enum in the contract), used to explain decisions, blocks, and failures.

Rules:
- stage accepts only StageName.
- event_type accepts only AuditEventType.
- reasons accepts only ReasonCode.
- do not use stage to describe an error and do not use a reason code as a stage name.

STANDARD ENVELOPE (JSONL AND EVENTS)
Minimum fields per event:
- ts_ms
- run_id
- cycle_id
- mode (LIVE)
- stage (StageName)
- event_type (AuditEventType)
- reasons (list of ReasonCode; may be empty when not applicable)
- snapshot_id (when present)
- decision_id (when present)
- order_intent_id (when present; otherwise empty string)
- exchange_time_ms (0 when exchange timestamp is not present)
- local_received_ms

AUDIT (SQLITE + JSONL)
Project decision:
- SQLite: source of truth (querying and incident reconstruction)
- JSONL: human trail (fast debugging and reading by Codex)

MINIMUM TABLES FOR GLOBAL -> TOPN -> TOPK
Goal:
- make symbol selection explainable and auditable
- allow auditing of churn guard and correlation diversification

Rules:
- each cycle must persist (at minimum) the TopN list and the final TopK, with scores and features used
- features must be persisted as normalized JSON (no secrets) and with a reference to config_hash

Suggested tables (SQLite):
- cycle_rankings
  - run_id TEXT NOT NULL
  - cycle_id TEXT NOT NULL
  - ranking_stage TEXT NOT NULL (TOPN|DEEP)
  - symbol TEXT NOT NULL
  - rank_index INTEGER NOT NULL
  - score REAL NOT NULL
  - features_json TEXT NOT NULL
  - config_hash TEXT NOT NULL
  - created_at_ms INTEGER NOT NULL
  - PRIMARY KEY (run_id, cycle_id, ranking_stage, symbol)
  - INDEX (run_id, cycle_id, ranking_stage, rank_index)

- cycle_selections
  - run_id TEXT NOT NULL
  - cycle_id TEXT NOT NULL
  - topn_symbols_json TEXT NOT NULL
  - topk_pre_symbols_json TEXT NOT NULL
  - topk_final_symbols_json TEXT NOT NULL
  - churn_guard_applied INTEGER NOT NULL (0|1)
  - correlation_applied INTEGER NOT NULL (0|1)
  - max_pairwise_corr REAL NULL
  - corr_pairs_json TEXT NULL
  - config_hash TEXT NOT NULL
  - created_at_ms INTEGER NOT NULL
  - PRIMARY KEY (run_id, cycle_id)

- symbol_health (optional, recommended)
  - symbol TEXT NOT NULL
  - quarantined_until_ms INTEGER NOT NULL
  - recent_rejects_count INTEGER NOT NULL
  - last_error_code TEXT NULL
  - updated_at_ms INTEGER NOT NULL
  - PRIMARY KEY (symbol)

AuditEventType (recommended for JSONL and auditing):
- UNIVERSE_ELIGIBILITY: per-symbol result (eligible=true/false, reasons)
- RANK_TOPN: TopN list + aggregated scores
- DEEP_SCAN: per-symbol scores and features in TopN (emitted for every TopN symbol; no sampling)
- TOPK_SELECTION: topk_pre, topk_final, churn_guard, correlation (max_pairwise_corr and pairs above the limit)

- STAGE_CHANGED: loop stage transition, emitted to JSONL and optionally as an SSE stream.
- AIGATE_CALL: AI gate call (always REDACTED + hashes), with input_hash, snapshot_hash, raw_hash and the final decision (ALLOW/BLOCK/MODIFY).
- WEBUI_REQUEST: request received by the local panel (route, method, status), with no secrets. Emitted for every request; no sampling.

- ALERT_RAISED: local alert raised (critical failure, degrade/pause/exit, low disk, drift, AI failure, writer pressure, rate limit).
- DISK_HEALTH_SAMPLE: disk and SQLite health sample (sqlite_bytes, wal_bytes, free_bytes).
- DB_WRITER_BACKPRESSURE: write pressure (queue_pct, lag_ms, action taken).
- FILTERS_REFRESHED: exchangeInfo/filters refresh with old/new hash and cause.
- INTENT_STATE_CHANGED: state transition of order_intent_id (CREATED, SENT_UNKNOWN, CONFIRMED, NOT_FOUND, FAILED_TERMINAL).
- RECONCILE_DIFF: divergence between local state and REST state.
- ORDER_SUBMIT, ORDER_CANCEL, ORDER_CANCEL_REPLACE: executed mutations (payload redacted/hashed).

EVENTS AND SAMPLES THAT FEED THE OPERATIONAL PANEL
Goal: define the auditable source for each panel card and avoid “phantom” metrics.

General rule
- If a panel card indicates an operational condition (DEGRADE/PAUSE/EXIT), there must be a corresponding ALERT_RAISED with reasons.
- Cards may use event aggregation. Aggregation is read-only and does not replace auditing.

Per-card mapping (panel -> audit)

1) SYS MODE (NORMAL|DEGRADE|PAUSE|EXIT), since_ms, and active reasons
- Source: ALERT_RAISED + OPS reasons (01_DECISION_CONTRACT.md).
- Computation rule:
  - current sys_mode is the last transition reason_code: ENTER_DEGRADE, ENTER_PAUSE, ENTER_EXIT; absence indicates NORMAL.
  - since_ms = now_ms - ts_ms of the last event that changed sys_mode.
  - active reasons = reasons from the last ALERT_RAISED relevant to the current sys_mode.

2) Active alerts (toast + list)
- Source: ALERT_RAISED.
- Minimum displayable fields: severity, reasons, ts_ms, stage, cycle_id, decision_id, order_intent_id.
- Ack (UI) is not an auditable event and does not alter ALERT_RAISED.

3) Current stage + history (last 20)
- Source: STAGE_CHANGED.
- Minimum fields: ts_ms, stage, cycle_id, decision_id, snapshot_id.

4) Disk and SQLite health (Disk/WAL/DB file)
- Source: DISK_HEALTH_SAMPLE.
- Minimum fields: free_bytes, sqlite_bytes, wal_bytes.
- The panel must display WAL separately (audit.sqlite-wal).

5) DB writer pressure
- Source: DB_WRITER_BACKPRESSURE.
- Minimum fields: queue_pct, lag_ms, action (e.g., NONE|DEGRADE|PAUSE).
- Optional fields (may be missing; panel shows N/A): queue_len, drops_total, drops_delta.

6) Event rate (events/min)
- Source: local aggregation of auditable JSONL/SQLite events.
- Rule: count events per minute in a fixed 60s window, excluding WEBUI_REQUEST.

7) Pending intents and SENT_UNKNOWN highlight
- Source: INTENT_STATE_CHANGED.
- Minimum fields: order_intent_id, state_from, state_to, ts_ms, cycle_id, decision_id, client_order_id (0 or empty string when not applicable).
- The panel must group by state_to and highlight state_to=SENT_UNKNOWN.

8) Reconcile status and diffs
- Primary source: RECONCILE_DIFF (divergences) and STAGE_CHANGED (timing of RECONCILE_REST).
- Reconcile duration:
  - start_ts_ms = ts_ms of STAGE_CHANGED for stage=RECONCILE_REST.
  - end_ts_ms = ts_ms of the next STAGE_CHANGED after start, in the same cycle_id.
  - duration_ms = end_ts_ms - start_ts_ms.
- Reconcile drift status (OK|WARN|BAD):
  - OK: there is no RECONCILE_DIFF in the cycle_id.
  - WARN: there is RECONCILE_DIFF in the cycle_id and none includes reason_code DRIFT_LIMIT_EXCEEDED.
  - BAD: there is RECONCILE_DIFF in the cycle_id with reason_code DRIFT_LIMIT_EXCEEDED.

9) AI Gate (observability without leaking content)
- Source: AIGATE_CALL (always REDACTED).
- Allowed fields: verdict, reasons, model, latency_ms, input_hash, snapshot_hash, raw_hash, modify_applied, error_code.
- Forbidden to persist: raw prompt, raw response, headers, keys, full URLs with secrets, and any non-redacted content.

10) Operational heartbeat (liveness)
- Source: audit aggregation.
- Rule:
  - audit_heartbeat_ts_ms = max(ts_ms) across all auditable events.
  - stage_heartbeat_ts_ms = max(ts_ms) across STAGE_CHANGED events.

SQLITE PERSISTS (ENTITIES)
- universe_scans, rank_runs, deep_scans
- candles (policy: watchlist and traded symbols)
- snapshots, decisions (with reasons)
- ai_gate_events (REDACTED)
- orders, fills, positions
- daily_summary (gross/net PnL, fees, slippage, hit rate, trades)
- experiments and results (offline only; never Live)

ADDITIONAL TABLES (MANDATORY FOR 24/7)
order_intents (idempotency and recovery):
- order_intent_id TEXT NOT NULL (PK or UNIQUE)
- run_id TEXT NOT NULL
- cycle_id TEXT NOT NULL
- mode TEXT NOT NULL (LIVE)
- decision_id TEXT NOT NULL
- symbol TEXT NOT NULL
- action TEXT NOT NULL (SUBMIT|CANCEL|CANCEL_REPLACE|OCO_CREATE|OCO_CANCEL|TRAILING_ACTIVATE)
- client_order_id TEXT NOT NULL
- intent_payload_json TEXT NOT NULL (canonical and deterministic; no volatile fields)
- state TEXT NOT NULL (CREATED|SENT_UNKNOWN|CONFIRMED|NOT_FOUND|FAILED_TERMINAL)
- exchange_order_id TEXT NULL
- exchange_oco_id TEXT NULL
- last_error_code TEXT NULL
- last_error_detail_redacted TEXT NULL
- created_at_ms INTEGER NOT NULL
- updated_at_ms INTEGER NOT NULL
Rules:
- order_intent_id must not collide; duplication must be impossible via constraints.
- state transitions must generate AuditEventType INTENT_STATE_CHANGED.

health_samples (disk/db/writer/process):
- sample_id TEXT NOT NULL (PK)
- run_id TEXT NOT NULL
- cycle_id TEXT NOT NULL
- mode TEXT NOT NULL (LIVE)
- sqlite_file_bytes INTEGER NOT NULL
- sqlite_wal_bytes INTEGER NOT NULL
- disk_free_bytes INTEGER NOT NULL
- db_writer_queue_pct INTEGER NOT NULL
- db_writer_lag_ms INTEGER NOT NULL
- ram_bytes INTEGER NOT NULL
- created_at_ms INTEGER NOT NULL
Rules:
- DISK_HEALTH_SAMPLE is derived from the health_samples collection.

JSONL (LOCAL)
- var\logs\audit-YYYY-MM-DD.jsonl
- every line must include: run_id, cycle_id, snapshot_id, decision_id, order_intent_id
- every line must include: exchange_time_ms 0 or empty string when not applicable and local_received_ms 0 or empty string when not applicable
- every line must include: mode (LIVE) and stage 0 or empty string when not applicable

FILES (REFERENCE)
- internal\audit\writer.go
- internal\audit\schema.go
- internal\observability\correlation.go
- internal\observability\logger.go
- internal\engine\reports\daily_summary.go
- internal\engine\executor\orders.go (clientOrderId builder)
- migrations\000x_daily_summary.sql
- migrations\000x_experiments.sql

OBSERVABILITY BY CORRELATION
Rule:
- generate and propagate IDs:
  - run_id (program execution)
  - cycle_id (24/7 cycle)
  - snapshot_id and decision_id
  - order_intent_id and clientOrderId
- every log line and audit event must carry IDs.

ADDITIONAL RULE (clientOrderId)
- clientOrderId must be deterministic and derived from auditable data (e.g., symbol + order_intent_id).
- clientOrderId must respect limits and validations:
  - len <= 36
  - recommended charset: [A-Za-z0-9-_]
  - never contain secrets, never contain env var values, never contain tokens.
- reuse:
  - it is forbidden to reuse a clientOrderId while there is an active/pending order with the same id.
  - reuse is allowed only after a terminal state is confirmed via reconcile (FILLED|CANCELED|EXPIRED|REJECTED), and it must be audited.
- recommendation: short prefix + truncated hash(symbol|order_intent_id).

Reason codes (minimum):
- CLIENT_ORDER_ID_INVALID
- CLIENT_ORDER_ID_REUSE_BLOCK

AIGATE_CALL (AUDITING AND REDACTION) - MANDATORY
Goal:
- make the AI Gate auditable, reproducible, and fail-closed when required
- enable diagnosis of timeout, invalid schema, rejected modify, and applied modify
- preserve privacy/secrets via redaction and hashes

SQLITE: TABLE ai_gate_events (REDACTED)
Rules:
- every AI Gate attempt must generate 1 row, even on failure
- payload and response must never be persisted without redaction
- hashes must enable correlation and offline verification without leaking sensitive content

Suggested DDL (SQLite):
- ai_gate_events
  - run_id TEXT NOT NULL
  - cycle_id TEXT NOT NULL
  - mode TEXT NOT NULL (LIVE)
  - stage TEXT NOT NULL (AIGATE_CALL)
  - event_type TEXT NOT NULL (AIGATE_CALL)
  - snapshot_id TEXT NOT NULL
  - snapshot_hash TEXT NOT NULL
  - decision_id TEXT NOT NULL
  - input_hash TEXT NOT NULL
  - enabled INTEGER NOT NULL (0|1)
  - verdict TEXT NOT NULL (ALLOW|BLOCK|MODIFY|ERROR)
  - reasons_json TEXT NOT NULL
  - model TEXT NULL
  - latency_ms INTEGER NULL
  - raw_hash TEXT NULL
  - request_json_redacted TEXT NULL
  - response_json_redacted TEXT NULL
  - modified_decision_json_redacted TEXT NULL
  - modify_applied INTEGER NOT NULL (0|1)
  - error_code TEXT NULL
  - error_detail_redacted TEXT NULL
  - exchange_time_ms INTEGER NULL
  - local_received_ms INTEGER NOT NULL
  - created_at_ms INTEGER NOT NULL
  - PRIMARY KEY (run_id, cycle_id, decision_id)
  - INDEX (run_id, cycle_id, created_at_ms)
  - INDEX (decision_id)
  - INDEX (snapshot_id)
  - INDEX (verdict)

Field semantics:
- verdict:
  - ALLOW/BLOCK/MODIFY: valid gate return
  - ERROR: call failure, timeout, invalid schema, parse failure, unknown reason_code, invalid MODIFY
- reasons_json:
  - list of gate reasons, always non-empty when enabled=1
- request_json_redacted:
  - gate payload already redacted, optional (may be NULL)
- response_json_redacted:
  - model raw return redacted, optional (may be NULL)
- modified_decision_json_redacted:
  - only when verdict=MODIFY
  - content: JSON object with ONLY the fields changed (patch) relative to the original Decision
  - forbidden: full Decision, unchanged fields, secrets/credentials, headers, full prompts, or raw exchange data
  - validation: the patch must be applicable on the original Decision and remain valid against 01_DECISION_CONTRACT.md
  - audit reconstruction: the verifier applies the patch over the original Decision and revalidates (schema + reasons + local invariants)

- modify_applied:
  - 1 only if the system applied MODIFY after local validations
- error_code:
  - short enumerated error code (e.g., AIGATE_TIMEOUT, AIGATE_SCHEMA_INVALID)
- error_detail_redacted:
  - short redacted detail (no full stack trace if it contains paths/secrets)

Minimum mandatory fields (always):
- run_id, cycle_id, mode, stage, snapshot_id, snapshot_hash, decision_id, input_hash
- enabled, verdict, reasons_json, local_received_ms, created_at_ms
- in case of ERROR: error_code is mandatory

JSONL: EVENT TYPE AIGATE_CALL
Each AI Gate JSONL line must follow the same standard envelope (correlation) and include:

Mandatory envelope:
- ts_ms
- run_id
- cycle_id
- snapshot_id
- decision_id
- order_intent_id (when present; otherwise empty string)
- mode (LIVE)
- stage (AIGATE_CALL)
- event_type (AIGATE_CALL)

Event fields (minimum):
- ai_enabled: bool
- ai_verdict: string (ALLOW|BLOCK|MODIFY|ERROR)
- ai_reasons: []string (reasons)
- ai_model: string (if known)
- ai_latency_ms: int
- ai_raw_hash: string
- ai_input_hash: string
- ai_snapshot_hash: string
- ai_modify_applied: bool
- ai_error_code: string (if ERROR)
- ai_error_detail_redacted: string (short)

Redaction in JSONL:
- never persist the full prompt
- never persist the full response without redaction
- if including redacted request/response, limit size (e.g., truncate to N bytes) and remove volatile fields
- do not include headers, tokens, keys, full URLs, sensitive querystrings, or full stack traces

Reason codes (AI Gate minimum):
- AIGATE_TIMEOUT
- AIGATE_SCHEMA_INVALID
- AIGATE_PARSE_FAIL
- AIGATE_MODIFY_INVALID
- AIGATE_REASON_UNKNOWN

GLOBAL REDACTION POLICY
- anything from env vars or that may contain secrets must be removed
- any credential identifier must be removed/masked
- logs must be safe-by-default: prefer hashes + short summaries to full payloads
- when in doubt, persist only:
  - input_hash, snapshot_hash, raw_hash
  - verdict, reasons, error_code
  - and optionally truncated redacted request/response

SQLITE IN 24/7
- WAL mode and busy_timeout are mandatory.
- 1 writer goroutine for audit/events.
- a full queue must DEGRADE/PAUSE and be audited (never lose audit silently).

EVENTS (STANDARD)
- every JSONL line includes: run_id, cycle_id, snapshot_id, decision_id, order_intent_id
- every persisted entity has timestamps and reasons as a JSON array of ReasonCode strings; use [] when not applicable
- any payload that may contain secrets must be REDACTED before persistence

RETENTION AND BACKUPS
- daily JSONL rotation and configurable retention
- daily SQLite backup and optional periodic VACUUM

WEB PANEL EVENTS
- STAGE_CHANGED: loop stage transition, emitted to JSONL and optionally as an SSE stream.
- AIGATE_CALL: AI gate call (always REDACTED + hashes), with input_hash, snapshot_hash, raw_hash and the final decision (ALLOW/BLOCK/MODIFY).
- WEBUI_REQUEST: request received by the panel (emitted for every request; no sampling).
- ALERT_RAISED: local alert shown in the panel (persistent toast) and recorded in JSONL.

OPERATIONAL MEASUREMENTS
- sqlite_file_bytes: size of var\data\audit.sqlite.
- ram_bytes: real-time process memory (Go).
- these measurements must be continuously collected and persisted in health_samples at disk_health_sample_interval_ms (canonical config; see 00_SOURCE_OF_TRUTH.md), and must also feed the web panel.
