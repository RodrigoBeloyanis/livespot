00_SOURCE_OF_TRUTH (SYSTEM LAW)

PURPOSE
Define the authority hierarchy, non-negotiable principles, and conflict-resolution rules.
If any instruction is ambiguous or conflicts, this file decides what is valid.

AUTHORITY HIERARCHY (STRONGEST TO WEAKEST)
A) 00_SOURCE_OF_TRUTH.md
B) 01_DECISION_CONTRACT.md (contracts and invariants)
C) 02_DATA_SNAPSHOT_SPEC.md (what enters the snapshot and how to measure)
D) 03_STRATEGY.md (deterministic signals, edge, entry/exit rules)
E) 04_RISK_ENGINE_RULES.md (final deterministic rules)
F) 05_EXECUTION_AND_FAILSAFE.md (how to execute safely and idempotently)
G) 06_AUDIT_RULES.md (what to record and how correlation works)
H) 07_SECURITY.md (secrets, redaction, safety locks)
I) 08_SYSTEM_ARCHITECTURE.md (overview and components)
J) 09_CODE_STRUCTURE.md (code organization)
K) 10_OPERATIONS_RULES.md (ops runbook and incidents)
L) 11_INVARIANTS_MAP.md (derived invariants map; does not create new rules)
M) 12_DECISIONS_LOG.md (history and rationale; does not replace rules)

AUXILIARY DOCUMENTS (NO NORMATIVE AUTHORITY)
- README.md (operational usage manual; does not create rules, does not alter contracts, and never takes precedence over 00–12)

CONFLICT RULE
- If a weaker document contradicts a stronger one: the stronger one prevails.
- If there is silence (not defined): choose the most conservative option and record the decision in 12_DECISIONS_LOG.md.

NON-NEGOTIABLE PRINCIPLES
- Determinism first (Strategy and Risk must be reproducible with snapshot + config).
- Mandatory auditing for any relevant action (SQLite + JSONL).
- No secrets in logs/audit (redaction is mandatory).
- True idempotency: clientOrderId + intents ledger.
- REST is the final truth in LIVE (reconcile is mandatory).
- Explicit fail policy: PAUSE | DEGRADE | RETRY | EXIT.
- LIVE requires safety locks (script --live and optional var\LIVE.ok) + mandatory checklist.
- 1 position per symbol (no pyramiding) in the initial version.
- Active observability is mandatory: failed or degraded => notify locally (console + web panel + sound when enabled) and audit ALERT_RAISED; silent failure is forbidden in 24/7.
- Disk health is mandatory: monitor free space on the volume + bytes of audit.sqlite + bytes of audit.sqlite-wal; thresholds => DEGRADE and then PAUSE before DB_BUSY/corruption.
- When in doubt, do not resubmit: any submit with an uncertain response must become state SENT_UNKNOWN and require a REST lookup by clientOrderId before any retry.

DEFINITIONS (OPERATIONAL VOCABULARY)
- Snapshot: deterministic data package (WS+REST) sufficient for auditing and post-mortem reconstruction.
- Decision: proposed/accepted decision for a symbol, including EntryPlan, ExitPlan, and reason_codes.
- EntryPlan: deterministic entry plan (maker-first TTL + fallback; IOC/FOK/market when allowed).
- ExitPlan: deterministic exit plan (TP/SL by ATR, OCO, and trailing with a defined mode).
- RiskVerdict: final deterministic ALLOW/BLOCK, with reasons.
- AI Gate: conservative external validation (ALLOW/BLOCK/MODIFY conservatively).
- DEGRADE: safe mode that blocks new entries and keeps only management/reconcile/auditing.

MAP OF WHAT BELONGS IN EACH DOCUMENT
- 08_SYSTEM_ARCHITECTURE.md: 24/7 loop, components, and flows.
- 01_DECISION_CONTRACT.md: contracts and invariants (fields, IDs, reason_codes).
- 02_DATA_SNAPSHOT_SPEC.md: snapshot fields, timings, windows, microstructure, and regime.
- 03_STRATEGY.md: deterministic signals, edge calculation, entry/exit rules, parameters, and filters.
- 04_RISK_ENGINE_RULES.md: sizing, adaptive thresholds, kill switch, quarantine, anti-overtrading.
- 05_EXECUTION_AND_FAILSAFE.md: quantization/filters, execution, intents, reconcile, failsafe/degrade.
- 06_AUDIT_RULES.md: tables, events, JSONL, correlation.
- 07_SECURITY.md: secrets, redaction, LIVE safety locks, Git practices.
- 09_CODE_STRUCTURE.md: code organization, boundaries, folders, and conventions.
- 10_OPERATIONS_RULES.md: runbook (startup recovery, incidents, backup, retention).
- 11_INVARIANTS_MAP.md: consolidated invariants map for review and anti-regression.
- 12_DECISIONS_LOG.md: change log and decisions (simple ADR style).

CONFIG (OPERATIONAL SOURCE OF TRUTH)
- internal\config\config.go defines all operational behavior (except secrets).
- Only 3 env vars exist (keys): BINANCE_API_KEY, BINANCE_API_SECRET, OPENAI_API_KEY.

REFERENCE DATE FOR THIS ORGANIZATION
2026-01-26

OPERATIONAL OBSERVABILITY
- The system must provide stage status in the console and optionally a local web panel for orders and health (RAM/SQLite).
- ALERT_RAISED is a mandatory auditable event for: entering DEGRADE/PAUSE/EXIT, drift above limit, DB writer queue at critical level, AI Gate failure, and low disk.
- The local web panel must display operational state (OK/DEGRADE/PAUSE) and active alerts; the console must always print stage + short reason when in failure/degradation.
- Disk health checking is continuous and has priority over non-critical operations (on low disk, reduce polling and avoid excessive writes; if it persists, PAUSE to avoid audit loss).

CANONICAL OPERATIONAL SIGNALS (PANEL + ALERTS)
Objective:
- standardize names, conditions, and severity of signals displayed in the panel and emitted as ALERT_RAISED.
- avoid "each screen invents a different stuck condition".

General rules:
- every signal below, when active, must appear in the local web panel under "Active reason_codes".
- every signal below, when crossing a severity threshold, must generate an auditable ALERT_RAISED (06_AUDIT_RULES.md).
- critical signals that threaten consistency/auditing must result in DEGRADE or PAUSE as specified below (05_EXECUTION_AND_FAILSAFE.md).

Configuration fields (source: internal\config\config.go):
- all constants below must exist in config.go with explicit units.
- values are deterministic and must not be "auto".

Canonical naming rule:
- The following liveness/feed threshold field names are the only valid names for loop and feed health in this repository:
  - loop_stuck_ms_degrade, loop_stuck_ms_pause
  - ws_stale_ms_degrade, ws_stale_ms_pause
  - rest_stale_ms_degrade, rest_stale_ms_pause
- Any weaker document that introduces alternative names (including aliases) is invalid and must be corrected.

1) LOOP_STUCK (loop liveness)
Definition:
- last_progress_ts_ms: last local timestamp when observable loop progress occurred.
  Observable progress = at least one STAGE_CHANGED event in the current cycle.
Condition and action:
- if now_ms - last_progress_ts_ms >= loop_stuck_ms_degrade:
  - enter DEGRADE and raise reason_code LOOP_STUCK_DEGRADE.
- if now_ms - last_progress_ts_ms >= loop_stuck_ms_pause:
  - enter PAUSE and raise reason_code LOOP_STUCK_PAUSE.
Expected result:
- operator sees "STUCK" in the panel and the system stops taking new entries before operating blind.
Audited:
- ALERT_RAISED with reason_code, last_progress_ts_ms, and delta_ms.

2) FEED_STALE (WS and REST)
Definitions:
- ws_last_msg_ts_ms: local timestamp of the last consumed market event (bookTicker or equivalent).
- rest_last_success_ts_ms: local timestamp of the last successful REST call used to reconcile/confirm state.
Conditions and actions (per source):
- if now_ms - ws_last_msg_ts_ms >= ws_stale_ms_degrade:
  - enter DEGRADE and raise reason_code WS_STALE_DEGRADE.
- if now_ms - ws_last_msg_ts_ms >= ws_stale_ms_pause:
  - enter PAUSE and raise reason_code WS_STALE_PAUSE.
- if now_ms - rest_last_success_ts_ms >= rest_stale_ms_degrade:
  - enter DEGRADE and raise reason_code REST_STALE_DEGRADE.
- if now_ms - rest_last_success_ts_ms >= rest_stale_ms_pause:
  - enter PAUSE and raise reason_code REST_STALE_PAUSE.
Expected result:
- prevents decisions on stale data and prevents execution without REST truth-source.
Audited:
- ALERT_RAISED with source=WS/REST, last_ts_ms, and delta_ms.

3) DISK_LOW (SQLite volume health)
Definitions:
- disk_free_bytes: free bytes on the volume where audit.sqlite resides.
Conditions and actions:
- if disk_free_bytes <= disk_free_degrade_bytes:
  - enter DEGRADE and raise reason_code DISK_LOW_DEGRADE.
- if disk_free_bytes <= disk_free_pause_bytes OR there is a write failure due to ENOSPC:
  - enter PAUSE and raise reason_code DISK_LOW_PAUSE.
Expected result:
- prevents DB_BUSY/loops and prevents audit loss due to full disk.
Audited:
- DISK_HEALTH_SAMPLE and ALERT_RAISED with disk_free_bytes, sqlite_bytes, and wal_bytes.

4) DB_WRITER_PRESSURE (audit writer queue)
Definitions:
- audit_writer_queue_pct: percentage occupancy of the writer queue.
- audit_writer_lag_ms: measured delay between producing and persisting events.
Conditions and actions:
- if audit_writer_queue_pct >= audit_writer_queue_hi_watermark_pct OR audit_writer_lag_ms >= audit_writer_max_lag_ms:
  - enter DEGRADE and raise reason_code DB_WRITER_QUEUE_HIGH.
- if audit_writer_queue_pct >= audit_writer_queue_full_pct:
  - enter PAUSE and raise reason_code DB_WRITER_QUEUE_FULL.
Expected result:
- prevents the system from operating when auditing cannot keep up.
Audited:
- DB_WRITER_BACKPRESSURE and ALERT_RAISED with queue_pct and lag_ms.

5) RECONCILE_DRIFT (REST truth, x10000 score)
Definitions:
- drift_score_x10000: int32 fixed-point score in x10000 units computed by reconcile policy (05_EXECUTION_AND_FAILSAFE.md).
Conditions and actions:
- if drift_score_x10000 >= reconcile_drift_degrade_score_x10000:
  - enter DEGRADE and raise reason_code DRIFT_LIMIT_EXCEEDED.
- if drift_score_x10000 >= reconcile_drift_pause_score_x10000:
  - enter PAUSE and raise reason_code DRIFT_LIMIT_EXCEEDED.
Expected result:
- operator can see "drift" in the panel and the system stops opening new entries.
Audited:
- RECONCILE_REST + RECONCILE_DIFF + ALERT_RAISED correlated by cycle_id/decision_id.

MISSING INFORMATION (RESOLVED - EXPLICIT DEFAULTS)
Contract: internal\config\config.go MUST define the following fields (fixed units) with the exact default values listed below.
Verification source: internal\config\config.go (Config struct / defaults).

- loop_stuck_ms_degrade: 5000 (5 seconds)
- loop_stuck_ms_pause: 15000 (15 seconds)
- ws_stale_ms_degrade: 2000 (2 seconds)
- ws_stale_ms_pause: 10000 (10 seconds)
- rest_stale_ms_degrade: 10000 (10 seconds)
- rest_stale_ms_pause: 60000 (60 seconds)
- disk_free_degrade_bytes: 1073741824 (1 GB)
- disk_free_pause_bytes: 536870912 (512 MB)
- audit_writer_queue_hi_watermark_pct: 80
- audit_writer_queue_full_pct: 95
- audit_writer_queue_capacity: 1024 (events)
- audit_writer_max_lag_ms: 5000 (5 seconds)
- reconcile_rest_interval_ms: 5000 (5 seconds)
- reconcile_drift_degrade_score_x10000: 20000 (x10000 scale; 2.0000)
- reconcile_drift_pause_score_x10000: 50000 (x10000 scale; 5.0000)

- webui_port: 8787 (TCP port; local-only)
- webui_stream_snapshot_interval_ms: 1000 (1 second)
- webui_intents_recent_limit: 50
- webui_reconcile_diffs_recent_limit: 50
- webui_market_symbols_limit: 50
- time_sync_recv_window_ms: 5000 (5 seconds; Binance signed calls)
- time_sync_interval_ms: 300000 (5 minutes)
- clock_drift_max_ms_live: 500 (0.5 seconds)
- clock_drift_max_ms_paper: 2000 (2 seconds)
- disk_health_sample_interval_ms: 5000 (5 seconds)
- audit_redacted_json_max_bytes: 4096 (4 KB)

Rule: If any of these fields is missing at boot, the system must enter PAUSE with reason_code STRAT_CONFIG_INVALID.

IMPLEMENTATION VALIDATION
On system boot:
1. Load config.go
2. Validate all fields above exist and have valid values
3. If any field is missing or invalid (e.g., negative value where positive required):
   - Enter PAUSE immediately
   - Emit ALERT_RAISED with reason_code STRAT_CONFIG_INVALID
   - Record details in audit log
4. Only proceed to NORMAL mode after all validation passes

SIGNAL PRIORITIZATION AND ESCALATION
When multiple signals trigger simultaneously, apply this priority order (highest to lowest):
1. DISK_LOW_PAUSE (prevents data loss)
2. DB_WRITER_QUEUE_FULL (prevents audit loss)
3. LOOP_STUCK_PAUSE (system not functioning)
4. WS_STALE_PAUSE / REST_STALE_PAUSE (no data)
5. DRIFT_LIMIT_EXCEEDED (reconcile failure)
6. All DEGRADE-level signals

Escalation rules:
- DEGRADE can escalate to PAUSE if condition worsens
- PAUSE never auto-reverts to NORMAL without explicit recovery
- All transitions must be audited with ALERT_RAISED

CONFIGURATION MANAGEMENT
- config.go is the single source of truth for operational parameters
- No hardcoded values allowed outside config.go
- All changes to config.go require:
  1. Update to default values in this document if changed
  2. Entry in 12_DECISIONS_LOG.md
  3. Validation tests updated
- config.go must be versioned and changes tracked

AUDIT REQUIREMENTS FOR SIGNALS
Each operational signal must generate:
1. ALERT_RAISED event with:
   - reason_code specific to the signal
   - current value (e.g., disk_free_bytes, queue_pct)
   - threshold value that was crossed
   - timestamp of detection
2. Correlation with cycle_id and decision_id when applicable
3. Clear human-readable message in console
4. Persistent alert in web panel until manually acknowledged

DETERMINISM GUARANTEE
The system guarantees that:
- Same config + same inputs = same behavior
- All thresholds are explicit and never "auto-adjusted"
- Rounding and calculations follow deterministic rules
- State transitions are predictable and auditable
- Recovery procedures are deterministic

COMPLIANCE VERIFICATION
To verify the system follows this source of truth:
1. Check that config.go contains all required fields with correct types
2. Verify that signal thresholds are not being bypassed or ignored
3. Confirm that all ALERT_RAISED events correspond to defined signals
4. Validate that conflict resolution follows the hierarchy
5. Ensure that silent failures do not occur (all failures generate alerts)

MAINTENANCE AND EVOLUTION
When updating this document:
1. Record all changes in 12_DECISIONS_LOG.md
2. Update config.go defaults if parameters change
3. Update all dependent documents (01-12) if needed
4. Run validation tests to ensure no regression
5. Update implementation to match new requirements

The system is not operational until all requirements in this document are fully implemented and validated.
