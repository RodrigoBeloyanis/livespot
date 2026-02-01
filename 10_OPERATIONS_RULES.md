10_OPERATIONS_RULES (OPERATIONS, RUNBOOK, AND INCIDENTS)

GOAL
Rules and procedures to operate the bot 24/7 safely, especially in Live.

PRINCIPLES
- Live is fail-closed when there is relevant uncertainty.
- No audit => no operation: DB unavailable => PAUSE (Live).
- The system is Live-only; all execution is real and reconciled against REST.
- In Live, REST is the final truth and reconcile is mandatory.
- AI Gate is mandatory in Live when AI_DEC=2 and is fail-closed.

LIVE CHECKLIST (BEFORE ENABLING ENTRIES)
Precondition: starting the bot in Live always begins in STARTUP_RECOVER.

1) Mode and safety locks
- Confirm MODE=LIVE.
- Confirm AI_DEC=2.
- Confirm external lock:
  - run with --live flag
  - and, if enabled, file var\LIVE.ok exists.
- Confirm the system is in NORMAL (not DEGRADE/PAUSE).

2) Database and auditing
- Confirm SQLite OK and writable:
  - WAL enabled
  - busy_timeout configured
  - 1 writer active
- Confirm disk health and DB growth (low disk / WAL):
  - free space on the volume above disk_free_pause_bytes in Live (source of truth: 00_SOURCE_OF_TRUTH.md)
  - var\data\audit.sqlite is writable (source of truth: 00_SOURCE_OF_TRUTH.md)
  - var\data\audit.sqlite-wal is writable (source of truth: 00_SOURCE_OF_TRUTH.md)
  - DB writer pressure is not FULL: queue_pct < audit_writer_queue_full_pct (source of truth: 00_SOURCE_OF_TRUTH.md)
  - last_writer_commit_ts_ms is recent: now_ms - last_writer_commit_ts_ms <= (2 * audit_writer_max_lag_ms) (source of truth: 00_SOURCE_OF_TRUTH.md)
- Confirm audit writer is not in FULL state:
  - reason_code DB_WRITER_QUEUE_FULL is NOT active
- Confirm JSONL is active and writing.

3) Binance: filters and clock
- Confirm filters are loaded and cache is valid.
- Confirm clock OK:
  - clock drift must be <= clock_drift_max_ms_live (source of truth: 00_SOURCE_OF_TRUTH.md)
- Confirm time sync OK:
  - clock_offset_ms updated via /api/v3/time
  - time_sync_recv_window_ms configured (source of truth: 00_SOURCE_OF_TRUTH.md)
  - time_sync_interval_ms configured (source of truth: 00_SOURCE_OF_TRUTH.md)

4) WS/REST and reconcile
- Confirm dynamic rate limits are loaded from exchangeInfo and headers are being observed (X-MBX-USED-WEIGHT-* / X-MBX-ORDER-COUNT-* / Retry-After).
- Confirm WS is connected and stable.
- If UserDataStream is used, confirm keepalive and events arriving.
- Confirm STARTUP_RECONCILE executed successfully:
  - position/orders reconstructed
  - no critical inconsistency
  - execution state defined (manage-only vs normal)

5) Limits and operational security
- Confirm local alerting is working:
  - ensure alerts appear in console and the local panel when raised.
- Confirm daily limits and exposure are configured.
- Confirm kill switch and symbol quarantine are active.
- Confirm there is no duplicate position per symbol (1 position per symbol).

Criteria to release entries:
- All items above OK.
- Otherwise: remain in DEGRADE or PAUSE with an auditable reason_code.

STARTUP RECOVERY (ON BOOT)
Goal: reconstruct state and avoid operating with drift.

Sequence:
- Run STARTUP_RECONCILE before any new entry.
- If there is an intent in state SENT_UNKNOWN:
  - never blindly resend
  - query REST by clientOrderId and only then advance (confirm vs not_found)
- If open position/orders are found:
  - enter RESUME_MANAGE_ONLY until stabilized and reconciled.
- If a critical inconsistency is found:
  - CLOSE_SAFELY or PAUSE_NEEDS_MANUAL (auditable).

Expected recovery outputs:
- known inventory of orders/position (REST)
- protected position (no protection gap window)
- defined state: NORMAL | DEGRADE | PAUSE


OPERATIONAL PANEL (24/7)
Goal: local diagnosis in less than 60 seconds, with a fixed reading order, without manually opening logs or SQLite.

Security rules
- The panel is local (127.0.0.1).
- In MODE=LIVE the panel is READ-ONLY: no button can create intents, submit orders, cancel orders, or alter engine state.
- Alert "Ack" is UI-only; auditable truth is the ALERT_RAISED event.

Fixed reading order (always in this order)
1) SYS MODE + reason_codes + since_ms
- NORMAL: operation allowed per policy (and Live locks).
- DEGRADE: the system must block new entries and keep manage/reconcile/audit per 05_EXECUTION_AND_FAILSAFE.md.
- PAUSE: the system must stop new execution actions; operate only recovery/manual per reason_codes.
- EXIT: the process ended; treat as an incident and consult the auditable trail of the run_id.

Operator action:
- If SYS MODE != NORMAL: read active reason_codes and follow the incident runbook (section "INCIDENTS AND RECOVERY" of this file).

2) Active alerts (toast + list)
- Prioritize severity CRIT, then WARN.
- Read: severity, reason_code, count, last_ts_ms, stage, cycle_id, decision_id, order_intent_id.
- After triage, Ack (UI) to reduce visual noise. Ack does not close the incident.

3) Current stage + short history (last 20 STAGE_CHANGED)
- Confirm the cycle advances: ts_ms and cycle_id must progress.
- If the stage does not change, use the header since_ms and treat as a possible loop stuck incident.

Operator action:
- If stuck is visible: collect run_id, current stage, last_ts_ms, and reason_codes; restart only after recording the cause in audit (or keep PAUSE).

4) DB and process health (Disk/WAL/Writer/Rate)
- Disk free bytes:
  - If reason_code DISK_LOW_DEGRADE is active: free disk space and monitor WAL trend.
  - If reason_code DISK_LOW_PAUSE is active (Live): free space and resume only after the system exits PAUSE.
- WAL bytes (audit.sqlite-wal): persistent growth indicates writer pressure or excessive ingestion.
- DB writer pressure:
  - queue_pct and lag_ms are the primary metrics.
  - state: OK|HIGH|FULL.
- Events/min: detect abnormal loop behavior (high rate) and regressions.
- Drops (if any): treat as an audit incident; without audit the system does not operate.

Operator action:
- If DB_WRITER_BACKPRESSURE or DB_WRITER_QUEUE_HIGH: keep or enter DEGRADE/PAUSE as already decided by the engine and investigate cause (event burst, slow disk, large DB).
- Never operate Live if auditing is degraded.

5) Pending intents (priority SENT_UNKNOWN)
- Show counts by state: CREATED, SENT_UNKNOWN, CONFIRMED, NOT_FOUND, FAILED_TERMINAL.
- SENT_UNKNOWN is priority 1 in Live.

Operator action:
- For each SENT_UNKNOWN:
  - Do not resend.
  - Query REST by clientOrderId.
  - Record transition via INTENT_STATE_CHANGED to CONFIRMED or NOT_FOUND.
  - Only then can the system return to normal flow.

6) Reconcile drift/diffs
- Reconcile status for the last cycle: ts_ms, duration (ms), drift (OK/WARN/BAD).
- Recent diffs: list of RECONCILE_DIFF with links by cycle_id/decision_id/order_intent_id.

Operator action:
- If drift is BAD (reason DRIFT_LIMIT_EXCEEDED present): treat as an incident and keep manage-only/PAUSE until resolved.
- If drift is WARN: inspect diffs and confirm the correction was applied and audited.

7) Churn / cancel-replace (10s/60s)
- Show counters in fixed windows of 10 seconds and 60 seconds.
- Show a "near limit" indicator and the symbol with the highest churn.

Operator action:
- If reason_code RISK_CHURN_LIMIT_HIT or RISK_CANCEL_REPLACE_LIMIT_HIT appears:
  - Do not "force" execution in Live.
  - Adjust parameters offline (config) to reduce churn and re-apply with Live locks.

OPERATIONAL CHECKLIST FOR STAGE AIGATE_CALL (EXECUTABLE)
Goal: make AI Gate usage predictable, auditable, and conservative.

Before (preconditions)
- Confirm the previous stage STRATEGY_PROPOSE completed and produced a valid Decision.
- Confirm Decision is valid against the contract (fields + invariants).
- Confirm DecisionConstraints are present and quantization applied.
- Confirm IDs and hashes are ready:
  - run_id, cycle_id, snapshot_id, decision_id
  - snapshot_hash computed
  - input_hash computed (canonical payload)
- Confirm redaction is ready:
  - redacted payload available (optional) and truncatable
  - no secrets/headers/keys in the payload
- Confirm operational policy:
  - If MODE=LIVE and AI_DEC=2: gate is mandatory and fail-closed.
  - If the system is in DEGRADE/PAUSE: do not call AI Gate for ENTRY.

During (execution)
- Call the model with strict timeout and limits.
- Parse only via strict schema.
- Validate AIGateResult invariants.
- If verdict=MODIFY:
  - apply local enforcement "only reduces risk"
  - reject if:
    - increases qty
    - loosens stop
    - increases aggressiveness (churn)
    - changes immutable fields (symbol/side/intent/constraints/ids)
  - if applied: recompute decision_id locally

After (postconditions)
- Always emit AI Gate auditing:
  - SQLite ai_gate_events (REDACTED) with hashes, verdict, reasons, error_code when present
  - JSONL event_type=AIGATE_CALL (REDACTED) with correlation envelope
- Forward to RISK_VERDICT:
  - ALLOW: original Decision
  - MODIFY applied: modified Decision
  - BLOCK: Decision marked
  - ERROR: apply the policy below

Failures and policy (operational)
- MODE=LIVE and AI_DEC=2:
  - timeout/error/invalid schema/parse failed/unknown reason_code/invalid MODIFY =>
    - PAUSE (no new entries) and record reason_code
    - manage-only per execution policy

Minimum reason codes (AI Gate)
- AIGATE_MODIFY_INVALID
- AIGATE_PARSE_FAIL
- AIGATE_REASON_UNKNOWN
- AIGATE_SCHEMA_INVALID
- AIGATE_TIMEOUT

DEGRADE MODE (WHEN THINGS WORSEN)
Goal: reduce risk and maintain control.

Rules:
- Block new entries.
- Allow only:
  - position management (protection/close)
  - REST reconcile
  - auditing/logs
- Exit DEGRADE only after stability for the configured window.
- DEGRADE due to unstable DB, unstable WS, persistent 429, reconcile drift.

INCIDENT RESPONSE (FAST RULES)
General rules:
- Always record an event with reason_code.
- In Live, prefer PAUSE over operating in an unknown state.

1) Timestamp / signature rejected
- Re-sync /api/v3/time once.
- Retry the call once.
- If it persists: PAUSE (Live).

2) UserDataStream unstable (if used)
- Enter DEGRADE.
- Use only REST reconcile until stabilized.
- Record event and reason_code.

3) Protection swap failed (OCO -> trailing or protection swap)
- Do not cancel existing protection.
- PAUSE (Live) and require intervention.
- Record the reason and the current order state.

4) WS down or reconnecting too often
- Enter DEGRADE.
- Keep REST reconcile if possible.
- If there is also loss of state: PAUSE.

5) Rate limit 429
- Exponential backoff.
- Enter DEGRADE.
- Reduce REST calls and increase reconcile intervals.

6) DB lock/busy or full writer queue
- If persistent: PAUSE (Live) and require action.
- Never operate without audit.
- If transient: DEGRADE and re-evaluate.

7) OpenAI failed with AI_DEC=2 (Live)
- PAUSE (no new entries) and BLOCK the current decision.
- Emit ai_gate_events (ERROR) and JSONL AIGATE_CALL.
- Require recovery before returning to NORMAL.

8) Reconcile drift above tolerance
- PAUSE and investigate.
- Record delta and inventory of orders/position.

9) Low disk / abnormal SQLite/WAL growth
- Enter DEGRADE immediately (block new entries).
- If persistent or writes fail: PAUSE (Live).
- Avoid VACUUM/backup on low disk; prioritize freeing space.
- Record bytes, free_space, and thresholds.

10) Filters/symbol changed (exchangeInfo drift, symbol_status not TRADING)
- Refresh exchangeInfo/filters and record FILTERS_REFRESHED.
- If persistent filter rejects or status not TRADING: quarantine the symbol and block new entries.
- Keep only management/close per policy.

11) Excessive churn (reprice/cancelReplace)
- Reduce aggressiveness (increase TTL, reduce requotes) and enter DEGRADE if needed.
- Avoid cancelReplace loops; prefer cancel+new when policy indicates.
- Monitor order-count limits (X-MBX-ORDER-COUNT-*) and apply backoff.

BACKUP AND RETENTION
- JSONL: daily rotation; configurable retention; automatic delete.
- Logs: size-based rotation and configurable retention.
- SQLite: daily backup and optional periodic VACUUM.
- Never version var\ or backups in Git.

OPERATION TOOLS
- cmd\doctor: diagnose WS/REST/DB/filters/clock.
- cmd\experiments: offline analysis and parameter evaluation; MUST NOT run while Live execution is enabled.

HEALTH SIGNALS (WHAT TO WATCH)
- WS reconnect rate
- DISK_FREE_BYTES low / SQLITE_WAL_BYTES growing
- FILTERS_REFRESHED and quarantined symbols
- intents in SENT_UNKNOWN (should be rare and always resolved via REST)
- number of RATE_LIMIT and REST_THROTTLED
- DB_WRITER_BACKPRESSURE / DB_WRITER_QUEUE_HIGH / DB_WRITER_QUEUE_FULL
- RECONCILE_DRIFT
- volume of BLOCKs by EDGE_TOO_LOW/COST_TOO_HIGH
- volume of AIGATE_TIMEOUT / AIGATE_SCHEMA_INVALID / AIGATE_PARSE_FAIL / AIGATE_MODIFY_INVALID / AIGATE_REASON_UNKNOWN
- average AIGATE_CALL time (latency) and variance

WEB PANEL (OPERATIONAL USE)
- by default, the bot exposes a local panel at:
  http://127.0.0.1:<PORT>/dashboard
- the panel shows:
  - current stage (stage)
  - orders and status (Live)
  - gain/loss cards
  - SQLite disk usage
  - process RAM usage

CHANGES AND RECORDING
- Changes to the runbook, Live checklist, degrade/pause policies, and incident response must be recorded in 12_DECISIONS_LOG.md when they alter operational behavior.
