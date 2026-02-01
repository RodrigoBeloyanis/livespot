livespot (Binance Spot 24/7, LIVE only)

PURPOSE OF THIS FILE
This README is a usage manual: overview, prerequisites, how to run LIVE, and where to debug.
It is NOT the law of the system. Rules and contracts live in files 00â€“12.

PROJECT LOCATION (WINDOWS 11)
C:\go\livespot

DOCUMENTATION (HIERARCHY AND RECOMMENDED READING ORDER)
1) 00_SOURCE_OF_TRUTH.md (wins on conflicts)
2) 01_DECISION_CONTRACT.md
3) 02_DATA_SNAPSHOT_SPEC.md
4) 03_STRATEGY.md
5) 04_RISK_ENGINE_RULES.md
6) 05_EXECUTION_AND_FAILSAFE.md
7) 06_AUDIT_RULES.md
8) 07_SECURITY.md
9) 08_SYSTEM_ARCHITECTURE.md
10) 09_CODE_STRUCTURE.md
11) 10_OPERATIONS_RULES.md
12) 11_INVARIANTS_MAP.md
13) 12_DECISIONS_LOG.md

SYSTEM SUMMARY
livespot is a 24/7 Binance Spot bot with:
- deterministic pair selection (Universe Scan -> Rank -> Deep Scan -> Watchlist)
- regime detection (TREND vs RANGE) affecting entry/exit/no-trade
- deterministic microstructure (spread health, imbalance, spread delta) to improve execution
- explicit edge model (before Risk) to reduce bad trades and late BLOCKs
- smart entry (maker-first with TTL + fallback; or IOC/Market depending on expected cost and policy)
- volatility-based exits (ATR-based TP/SL) and trailing (activation and distance by ATR)
- dynamic position sizing and adaptive thresholds
- anti-overtrading (limits per symbol/window, loss-streak blocks, degrade on stale feeds/latency)
- optional/mandatory AI Gate (ALLOW | BLOCK | MODIFY), where MODIFY is conservative-only
- execution with failsafes, REST reconciliation (final truth), and idempotency (clientOrderId + intents ledger)
- full auditing in SQLite + human trail in JSONL

ENVIRONMENT
- Project directory (Windows 11): C:\go\livespot
- Go: go1.25.6 windows/amd64
- Node (optional, local tooling): v25.4.0
- Git: 2.52.0.windows.1
- No Docker
- No Testnet
- Execution mode: LIVE only
- Comments are forbidden in code

Mandatory compensating rule (because comments are forbidden):
- self-explanatory names, short functions, small files
- reason_codes are mandatory in decisions/blocks/failures
- logs and audit must carry correlation IDs (run_id/cycle_id/snapshot_id/decision_id/order_intent_id)
- contracts/schemas must always be validated

HOW TO RUN (WINDOWS 11)

1) Prepare .env
Create .env with the 3 keys (or set them in the system):
- BINANCE_API_KEY=...
- BINANCE_API_SECRET=...
- OPENAI_API_KEY=...

2) Adjust internal config
Update internal\config\config.go:
- Set AI_DEC for LIVE (AI_DEC=2 is the safe default) and enable LIVE safety locks.
- Configure quote asset (USDT default) and universe filters.
- Configure trailing policy (AUTO is recommended to start). AUTO is a config policy that resolves to OFF|VIRTUAL|NATIVE at runtime.
  - Contract note: ExitPlan.trailing_mode in persisted decisions is ALWAYS one of OFF|VIRTUAL|NATIVE (see 01_DECISION_CONTRACT.md).
  - When AUTO is enabled, the system MUST persist the resolved final mode in ExitPlan.trailing_mode for audit reproducibility.

3) Migrate the database (first run)
- go run .\cmd\migrate

4) Run LIVE (only when approved)
- .\scripts\live.ps1 --live
- optional external lock: create file var\LIVE.ok

LIVE safety rules (normative source: 00_SOURCE_OF_TRUTH.md and 10_OPERATIONS_RULES.md):
- prevent accidental operation (explicit --live and optional var\LIVE.ok)
- checklist is mandatory
- clear reason_codes for any BLOCK/PAUSE/DEGRADE transition

24/7 OPERATIONS: PAUSE, DEGRADE, ALERTS, AND RECOVERY
- DEGRADE: blocks new entries and keeps only MANAGE/RECONCILE/AUDIT per 05_EXECUTION_AND_FAILSAFE.md (FAIL POLICY).
- PAUSE: pauses the execution loop per 05_EXECUTION_AND_FAILSAFE.md (FAIL POLICY).
- Any transition to DEGRADE/PAUSE/EXIT and any critical operational failure must:
  - raise a local alert (console + web panel toast and, when enabled, sound)
  - emit an auditable ALERT_RAISED event

LOW DISK MODE
- The system continuously monitors free space of the volume and growth of:
  - var\data\audit.sqlite
  - var\data\audit.sqlite-wal
- When thresholds are crossed: it enters DEGRADE and, if persistent, enters PAUSE to avoid DB_BUSY/corruption loops.

RATE LIMITS (DYNAMIC)
- On boot, the system loads rateLimits from exchangeInfo and uses those values as reference (no hardcoded fixed numbers).
- At runtime, the system respects headers X-MBX-USED-WEIGHT-* and X-MBX-ORDER-COUNT-* and applies backoff honoring Retry-After when 429/418 occurs.

RECOVERY: SENT_UNKNOWN
- SENT_UNKNOWN means: an order submit may have been accepted by Binance, but the response was not confirmed locally (timeout/connection drop).
- Never blindly resend: query REST by clientOrderId first and only then decide CONFIRMED vs NOT_FOUND.

AI GATE (BEHAVIOR)
The AI Gate is a conservative gate between Strategy and Risk.
It never replaces Risk. Risk can always block.

Verdicts:
- ALLOW: the decision continues as-is
- BLOCK: the decision is blocked (or marked for blocking) and goes to Risk
- MODIFY: accepted only if conservative; the system validates it and may reject it

MODIFY (CONSERVATIVE ONLY)
The system enforces rules locally. MODIFY may only reduce risk/aggressiveness, for example:
- reduce qty
- tighten protection (stop/trailing)
- make entry less aggressive (require maker/limit, reduce churn)
- reduce/disable fallback
Any attempt to increase risk (higher qty, looser stop, more aggressive entry, change immutable fields) is rejected and audited.

AI DECISION POLICY (AI_DEC)
- AI_DEC=0: AI disabled (enabled=false)
- AI_DEC=1: AI enabled, failures remain conservative (ENTRY becomes BLOCK)
- AI_DEC=2: AI is mandatory and fail-closed (no new entries on failure)

WHAT HAPPENS WHEN AI FAILS (OPENAI FAILURE)
Typical failures:
- timeout
- network error
- empty response
- invalid schema
- parse failed
- unknown reason_code
- invalid MODIFY (attempted to increase risk)

Behavior (normative source: 05_EXECUTION_AND_FAILSAFE.md, AIGATE_CALL fail-closed section):
- AI_DEC=2:
  - fail-closed: no new entries
  - the system applies the operational transition defined by the FAIL POLICY in 05 (DEGRADE or PAUSE),
    always with ALERT_RAISED and AIGATE_CALL auditing (error_code + hashes)
  - MANAGE/RECONCILE continue only if FAIL POLICY enters DEGRADE; in PAUSE the loop is paused
- AI_DEC=1:
  - AIGATE_CALL failure makes ENTRY = BLOCK (conservative)
  - MANAGE/RECONCILE proceed (without creating new entries) and everything is audited

Where to confirm state:
- console: current stage and short reason
- JSONL: AIGATE_CALL event and STAGE/PAUSE/DEGRADE events
- SQLite: ai_gate_events and cycle states

LOGS AND DEBUGGING

Main logs:
- var\logs\livespot.log
- var\logs\livespot.error.log
- var\logs\ai_gate.log (redacted)
- var\logs\audit-YYYY-MM-DD.jsonl

Auditable database:
- var\data\audit.sqlite

WHERE TO VIEW AI GATE EVENTS
JSONL (human trail):
- var\logs\audit-YYYY-MM-DD.jsonl
- search for:
  - event_type=AIGATE_CALL
  - event_type=STAGE_CHANGED

Useful fields (redacted):
- ai_verdict (ALLOW|BLOCK|MODIFY|ERROR)
- ai_reasons (reason_codes)
- ai_model, ai_latency_ms
- ai_input_hash, ai_snapshot_hash, ai_raw_hash
- ai_modify_applied
- ai_error_code when present

SQLite (query and truth):
- table ai_gate_events (REDACTED)
- each gate attempt generates 1 row (including failures)

Useful fields:
- snapshot_hash, input_hash, raw_hash
- verdict, reasons_json, modify_applied
- error_code and redacted details

Codex debugging source:
- prompts\ (versioned templates and schemas)

TESTS AND QUALITY

Unit tests:
- contract validations (Decision/Snapshot/Risk/AIGateResult)
- regime/microstructure/microvol computation
- edge score and cost model
- entry_plan (maker-first + fallback) and entry_limits
- exit_plan (ATR) and exit_sanity
- quantization and filters (tick/step/minNotional)
- idempotency (clientOrderId)
- AI limits (conservative MODIFY + strict schema)
- anti-overtrading and adaptive thresholds
- position sizing

Run:
- go test ./... -count=1

IMPORTANT NOTES
- No Docker. No Testnet. LIVE only.
- Secrets only in .env (or system environment variables). Never version .env.
- LIVE requires checklist + external lock (see 10_OPERATIONS_RULES.md).
- AI Gate with AI_DEC=2 is fail-closed: if it fails, do not enter new positions.
- When in doubt: consult 00_SOURCE_OF_TRUTH.md and 12_DECISIONS_LOG.md.


TROUBLESHOOTING BY SYMPTOMS
The goal is deterministic triage: symptom -> detection -> action -> expected result.

Doctor checks
- Run go run .\\cmd\\doctor to verify config, LIVE locks, audit sinks, filesystem permissions, and SQLite availability.
- Any FAIL means the system must remain in PAUSE or DEGRADE until resolved.

1) Symptom: SYS MODE stuck in DEGRADE or PAUSE
- Detect:
  - Web panel header: SYS MODE=DEGRADE|PAUSE and active reason_codes
  - JSONL: last ALERT_RAISED + STAGE_CHANGED events
- Action:
  - Identify the top reason_code (e.g., WS_STALE_DEGRADE, REST_STALE_DEGRADE, DISK_LOW_DEGRADE, DRIFT_LIMIT_EXCEEDED).
  - Apply the corresponding runbook action in 10_OPERATIONS_RULES.md.
  - Do NOT re-enable entries until reason_codes clear for at least 2 consecutive cycles.
- Expected result:
  - Transition back to SYS MODE=NORMAL with STAGE_CHANGED + ALERT_RAISED audit events for the transition.

2) Symptom: Frequent SENT_UNKNOWN intents
- Detect:
  - Web panel 'Execution and reconciliation' shows SENT_UNKNOWN pinned/highlighted.
  - SQLite intents table shows many intents with state=SENT_UNKNOWN in the last 10 minutes.
- Action:
  - Follow the recovery flow: query REST by clientOrderId, then mark CONFIRMED vs NOT_FOUND.
  - If the rate persists, enter DEGRADE (block new entries) and investigate network latency and Binance API errors.
- Expected result:
  - SENT_UNKNOWN count returns to near-zero; reconcile diffs shrink; no duplicate orders are created.

3) Symptom: Repeated ALERT_RAISED for protection (manual action required / install failures)
- Detect:
  - JSONL: event_type=ALERT_RAISED with reason_codes including PAUSE_NEEDS_MANUAL_PROTECTION and/or PROTECTION_INSTALL_FAILED.
  - Web panel alerts: protection-related reason_codes; position drawer shows missing/invalid protection.
- Action:
  - The system MUST block new entries for the symbol and run protection recovery (close safely if needed).
  - If recovery cannot install valid protection within limits, PAUSE_NEEDS_MANUAL_PROTECTION is mandatory.
- Expected result:
  - Either: protection is re-installed successfully; or the position is closed safely; or the system is paused awaiting manual action.

4) Symptom: audit.sqlite errors, DB_BUSY, or writer queue saturation
- Detect:
  - Logs: DB_BUSY/SQLITE_BUSY, WAL growth without checkpoint, or writer lag warnings.
  - Web panel: DB writer pressure queue_pct high, lag_ms increasing, WAL bytes growing.
- Action:
  - Enter DEGRADE immediately (block new entries).
  - Check disk space and WAL growth; ensure audit.sqlite is writable.
  - If not recoverable quickly, PAUSE to prevent corruption loops.
- Expected result:
  - Writer queue stabilizes; lag_ms decreases; WAL growth stops; system returns to NORMAL only after stability.

5) Symptom: AI Gate failures (AI_DEC=2) blocking entries
- Detect:
  - Web panel 'AI Gate' shows failure counts; JSONL has AIGATE_CALL with error_code.
- Action:
  - Keep fail-closed behavior; do not force entries.
  - Fix the underlying cause (API key, connectivity, schema mismatch, prompt version mismatch).
- Expected result:
  - AIGATE_CALL returns valid ALLOW/BLOCK/MODIFY; system transitions back to NORMAL when policy permits.

LICENSE
Private (define later).

QUICK DIAGNOSTICS (CONSOLE AND WEB PANEL)
Goal: enable operational diagnosis in under 60 seconds, without opening logs and without manually opening SQLite.

SCOPE AND SECURITY
- The panel is LOCAL (127.0.0.1 only).
- The panel is READ-ONLY: no button may mutate execution state, intents, orders, or the database.

URL
- http://127.0.0.1:<PORT>/dashboard

DATA SOURCES
- Auditable truth: SQLite + JSONL (see 06_AUDIT_RULES.md).
- Process/WS metrics may exist in memory; when they generate operational transitions (DEGRADE/PAUSE/EXIT),
  there must be a corresponding auditable ALERT_RAISED with reason_codes.

FIRST FOLD (ALWAYS VISIBLE)
1) Header (operational state)
- MODE: LIVE
- SYS MODE: NORMAL|DEGRADE|PAUSE|EXIT
- since_ms: time since the last SYS MODE transition
- active reason_codes: current list (when present)
- current stage + run_id + cycle_id + decision_id (empty string outside decision stages)

2) Active alerts (persistent toast + list)
- Persistent toast by severity (CRIT/WARN/INFO) with counts.
- List with: severity, reason_code, count, last_ts_ms, stage, cycle_id, decision_id, order_intent_id.
- "Ack" is UI-only (does not alter auditing). The truth is the ALERT_RAISED event.

3) Stage timeline (last 20 STAGE_CHANGED)
- Columns: ts_ms, stage, cycle_id, decision_id, symbol (empty string outside symbol stages), short summary.

4) Database and process health
- Disk free bytes (with status OK/WARN/CRIT per system state and reason_codes DISK_LOW_DEGRADE / DISK_LOW_PAUSE).
- audit.sqlite bytes and audit.sqlite-wal bytes (WAL separate) with trend (sparkline).
- DB writer pressure: queue_pct, lag_ms, state (OK|HIGH|FULL).
- Event rate: events/min (auditable event count per fixed window).
- Drops (if any): writer/queue drop counter.

5) Execution and reconciliation
- Pending intents by state: CREATED, SENT_UNKNOWN, CONFIRMED, NOT_FOUND, FAILED_TERMINAL.
  - SENT_UNKNOWN must be highlighted and pinned to the top.
- Reconcile: last RECONCILE_REST (ts_ms), duration (ms), drift (OK/WARN/BAD), with links to recent RECONCILE_DIFF.
- Churn/cancel-replace: counters in 10s and 60s windows and a "near limit" indicator.

TABS (BELOW THE FOLD)
- Market/Signals: regime (TREND|RANGE), scores (x10000), microstructure (spread, p50/p90, spread delta, imbalance), edge and costs (fees/slippage/spread) and the block reason when edge < min_edge.
- Risk/Limits: total and per-symbol exposure vs limits, free vs locked, per-order reserves, PnL (realized/unrealized), remaining loss limit, drawdown, trades today, anti-overtrading (cooldown/trades per window/loss streak).
- AI Gate: last AIGATE_CALL (verdict, reason_codes, latency, model, hashes, modify_applied) and failure rate (timeout/schema/parse) + fail-closed indicator.
- TopN/TopK + Correlation: cycle topN/topK (score + persisted features), churn guard applied (yes/no), max_pairwise_corr and replacements.
- Orders/Fills: orders/fills/positions with filters and drilldown links.

DRILLDOWN (ONE DECISION / ONE POSITION)
When clicking a decision_id (or order_intent_id), the panel must open a drawer with a human-readable trace:
- Snapshot refs + regime/microstructure
- Decision (edge/plan) â†’ AIGateResult â†’ RiskVerdict â†’ intents â†’ orders/fills â†’ reconcile diffs

HOW TO USE THE PANEL TO OPERATE 24/7 (FIXED ORDER)
1) Check header: SYS MODE, active reason_codes, and since_ms.
2) Read active alerts (CRIT/WARN first). Ack only after triage.
3) Check stage timeline: confirm cycle progression and absence of visible "stuck".
4) Check health: Disk/WAL/Writer (queue_pct and lag_ms) and events/min.
5) Check intents: if SENT_UNKNOWN exists, follow the recovery procedure (10_OPERATIONS_RULES.md).
6) Check reconcile: drift and diffs; if drift BAD (reason DRIFT_LIMIT_EXCEEDED), treat as an incident.
7) Check churn/cancel-replace: if near the limit, reduce aggressiveness via config (offline change) and keep LIVE without entries until stable.
