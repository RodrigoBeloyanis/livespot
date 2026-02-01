11_INVARIANTS_MAP (INVARIANTS MAP)

NATURE
This document is derived and operational. It does not create new rules.
In case of conflict, the hierarchy defined in 00_SOURCE_OF_TRUTH.md prevails.

GOAL
Maintain a short checklist of non-negotiable promises to prevent regressions and loss of auditability.

HOW TO CHANGE AN INVARIANT
Rule: change the applicable authority document first (00 to 10) and record the decision in 12_DECISIONS_LOG.md when it is a rule/contract change.
Then update this map to reflect the change.

DEFINITION
An invariant is something that must remain true in any PR, refactor, new feature, or optimization.
If an item is not explicitly defined in the hierarchy, the default must be the most conservative option. Ref: 00_SOURCE_OF_TRUTH.md

SECTION A CONFIGURATION AND MODES

INV-001 MODE: only LIVE exists. Ref: 08_SYSTEM_ARCHITECTURE.md, README.md, 00_SOURCE_OF_TRUTH.md
INV-002 CONFIG: all operational configuration lives only in `internal\config\config.go`. Ref: 00_SOURCE_OF_TRUTH.md
INV-003 SECRETS: only 3 environment variables exist: BINANCE_API_KEY, BINANCE_API_SECRET, OPENAI_API_KEY. Ref: 00_SOURCE_OF_TRUTH.md, 07_SECURITY.md
INV-004 SECRETS DO NOT PERSIST: never persist secrets in logs, JSONL, or SQLite. Ref: 07_SECURITY.md
INV-005 NO TESTNET: there is no testnet mode. Live uses production market data and live execution. Ref: 08_SYSTEM_ARCHITECTURE.md, README.md
INV-006 ONE PIPELINE: Strategy, Risk, and Execution share a single deterministic pipeline; there are no alternate execution modes. Ref: 08_SYSTEM_ARCHITECTURE.md, 05_EXECUTION_AND_FAILSAFE.md
INV-007 NO MUTATION ENDPOINTS: the local web panel must not expose any administrative or destructive HTTP actions in LIVE. Ref: 07_SECURITY.md, 08_SYSTEM_ARCHITECTURE.md, 10_OPERATIONS_RULES.md
INV-008 LIVE LOCKS: operating Live requires external locks and an approved checklist before starting the entry loop. Ref: 07_SECURITY.md, 10_OPERATIONS_RULES.md

SECTION B DETERMINISM AND SNAPSHOT

INV-010 DETERMINISM: Strategy and Risk must be reproducible given config + snapshot. Ref: 00_SOURCE_OF_TRUTH.md, 02_DATA_SNAPSHOT_SPEC.md, 03_STRATEGY.md
INV-011 SUFFICIENT SNAPSHOT: the snapshot must contain enough data to explain why and how the decision was made (audit reconstruction). Ref: 02_DATA_SNAPSHOT_SPEC.md
INV-012 TIMEFRAMES: timeframes and weights are defined in config.go and enter the deterministic pipeline. Ref: 02_DATA_SNAPSHOT_SPEC.md
INV-013 REGIME FIRST CLASS: regime and scores enter the snapshot and are used by Strategy and Risk. Ref: 02_DATA_SNAPSHOT_SPEC.md, 03_STRATEGY.md, 04_RISK_ENGINE_RULES.md
INV-014 MICROSTRUCTURE: deterministic spread metrics and short signals enter the snapshot when used for decisions. Ref: 02_DATA_SNAPSHOT_SPEC.md, 03_STRATEGY.md
INV-015 DUAL TIMESTAMPS: always record exchange_time_ms and local_received_ms when applicable. Ref: 01_DECISION_CONTRACT.md, 02_DATA_SNAPSHOT_SPEC.md
INV-016 EVENT ORDER: tolerate out-of-order events within a short window and audit discards. Ref: 02_DATA_SNAPSHOT_SPEC.md
INV-017 SOURCE OF TRUTH: WS is the source for signals and state; REST is the source for execution, confirmation, and reconcile. Ref: 02_DATA_SNAPSHOT_SPEC.md
INV-018 REST FINAL TRUTH IN LIVE: any critical operational state must be confirmed by REST. Ref: 00_SOURCE_OF_TRUTH.md, 05_EXECUTION_AND_FAILSAFE.md

SECTION C CONTRACTS, IDS, CANONICAL JSON, AND REASON CODES

INV-019 SEPARATE TERMINOLOGY: StageName (loop stages), AuditEventType (audit event types), and ReasonCode (reasons) are distinct concepts and must not be mixed. stage uses only StageName; event_type uses only AuditEventType; reasons uses only ReasonCode. Ref: 01_DECISION_CONTRACT.md, 06_AUDIT_RULES.md, 08_SYSTEM_ARCHITECTURE.md

INV-020 CONTRACT IS LAW: changing 01_DECISION_CONTRACT requires updating schemas, validations, migrations if needed, and recording in 12_DECISIONS_LOG.md. Ref: 01_DECISION_CONTRACT.md
INV-021 REQUIRED FIELDS: run_id, cycle_id, snapshot_id, decision_id, symbol, quote, timestamps, reason_codes, and plans (EntryPlan and ExitPlan) when intent exists. Ref: 01_DECISION_CONTRACT.md
INV-022 CORRELATION: run_id per execution; cycle_id per cycle; decision_id derived; order_intent_id per atomic action; deterministic and short clientOrderId. Ref: 01_DECISION_CONTRACT.md, 06_AUDIT_RULES.md
INV-023 REASON CODES ALWAYS: any ALLOW, BLOCK, MODIFY, failure, fallback, degrade, pause, and administrative event must carry reason_codes. Ref: 01_DECISION_CONTRACT.md, 06_AUDIT_RULES.md
INV-024 SINGLE CATALOG: the reason_codes catalog lives in internal\domain\reasoncodes\codes.go and must not be duplicated. Ref: 01_DECISION_CONTRACT.md
INV-025 VALIDATE AT THE BOUNDARY: contracts must be validated by schema and validations before persisting and before executing. Ref: 01_DECISION_CONTRACT.md, 09_CODE_STRUCTURE.md
INV-026 CANONICAL JSON: every contractual hash (decision_id, snapshot_hash, input_hash, raw_hash) derives from canonical JSON (RFC 8785) + SHA-256 hex. Ref: 01_DECISION_CONTRACT.md, 06_AUDIT_RULES.md
INV-027 STABLE HASHES: the same canonical input must always generate the same hash (no volatile fields outside the snapshot). Ref: 01_DECISION_CONTRACT.md, 08_SYSTEM_ARCHITECTURE.md
INV-028 SEPARATE HASH PAYLOADS: DecisionHashPayload may include cycle_id, stage, and ts_ms for audit; OrderIntentHashPayload must not include run_id, cycle_id, stage, ts_ms, or fields derived from the local clock. Ref: 01_DECISION_CONTRACT.md, 05_EXECUTION_AND_FAILSAFE.md, 06_AUDIT_RULES.md
INV-029 NO FLOAT IN HASHED OBJECTS: any field participating in DecisionHashPayload, OrderIntentHashPayload, or snapshot_hash must be int or normalized decimal string (no exponent), never float. Ref: 01_DECISION_CONTRACT.md, 02_DATA_SNAPSHOT_SPEC.md, 03_STRATEGY.md

SECTION D AI GATE (CONSERVATIVE, DETERMINISTIC, AND AUDITABLE)

INV-030 AIGATE VERDICTS: only ALLOW, BLOCK, MODIFY (and ERROR as an operational representation of failure when persisted in audit). Ref: 01_DECISION_CONTRACT.md, 06_AUDIT_RULES.md
INV-031 STRICT SCHEMA: the AI Gate return is accepted only if it validates against a strict schema; invalid schema => failure. Ref: 07_SECURITY.md, 01_DECISION_CONTRACT.md
INV-032 DETERMINISTIC PAYLOAD: the payload sent to the AI Gate must be deterministic and canonical, producing a stable input_hash; the base snapshot must produce a stable snapshot_hash. Ref: 01_DECISION_CONTRACT.md, 08_SYSTEM_ARCHITECTURE.md, 07_SECURITY.md
INV-033 REDACTION: AI payloads and responses must be redacted before any log/audit; when in doubt, persist only hashes + a short summary. Ref: 07_SECURITY.md, 06_AUDIT_RULES.md
INV-034 CONSERVATIVE MODIFY: MODIFY may only reduce risk (reduce qty, tighten stop, require less aggressive entry, reduce/disable fallback). Never increase qty, loosen stop, increase aggressiveness, change symbol/timeframe, or touch ids. Ref: 01_DECISION_CONTRACT.md, 07_SECURITY.md
INV-035 LOCAL ENFORCEMENT: every MODIFY must be validated locally; any attempt to increase risk => reject MODIFY and audit (AIGATE_MODIFY_INVALID). Ref: 01_DECISION_CONTRACT.md, 07_SECURITY.md, 10_OPERATIONS_RULES.md
INV-036 RISK HAS FINAL WORD: the AI Gate never replaces Risk; Risk may block even with ALLOW. Ref: 04_RISK_ENGINE_RULES.md, 07_SECURITY.md
INV-037 AI_DEC=2 IN LIVE FAIL-CLOSED: in MODE=LIVE with AI_DEC=2, AI Gate failure (timeout/error/invalid schema/parse failed/unknown reason/invalid MODIFY) => PAUSE or auditable BLOCK and no new entries. Ref: 05_EXECUTION_AND_FAILSAFE.md, 07_SECURITY.md, 10_OPERATIONS_RULES.md
INV-038 ALWAYS AUDIT AIGATE: every AI Gate attempt (including failure) must generate a SQLite event (ai_gate_events) and a JSONL AIGATE_CALL event with full correlation. Ref: 06_AUDIT_RULES.md, 08_SYSTEM_ARCHITECTURE.md

SECTION E RISK ENGINE (DETERMINISTIC)

INV-040 RISK FINAL: RiskVerdict is deterministic and final (ALLOW or BLOCK) and must record applied limits and reasons. Ref: 01_DECISION_CONTRACT.md, 04_RISK_ENGINE_RULES.md
INV-041 EDGE AND COST: Risk validates cost_model and edge_bps_expected or edge_score, and may require adaptive min_edge under bad conditions. Ref: 04_RISK_ENGINE_RULES.md, 03_STRATEGY.md
INV-042 SIZING: position sizing uses risk_per_trade and stop_distance, respecting exposure limits and filters (min_notional and stepSize). Ref: 04_RISK_ENGINE_RULES.md, 05_EXECUTION_AND_FAILSAFE.md
INV-043 ANTI-OVERTRADING: cooldown, max trades per window, and loss streak block new entries. Ref: 04_RISK_ENGINE_RULES.md
INV-044 QUARANTINE: symbols with repeated failures enter quarantine for TTL and are excluded from the universe. Ref: 04_RISK_ENGINE_RULES.md
INV-045 KILL SWITCH: daily and session limits block new entries and allow only management or pause per policy. Ref: 04_RISK_ENGINE_RULES.md
INV-046 ONE POSITION PER SYMBOL: no pyramiding in the initial version; do not open if a position or a pending entry already exists. Ref: 00_SOURCE_OF_TRUTH.md, 04_RISK_ENGINE_RULES.md, 12_DECISIONS_LOG.md
INV-047 BUDGET BY FREE: use free balances and per-order reserves; do not spend locked; avoid double spend. Ref: 04_RISK_ENGINE_RULES.md

SECTION F EXECUTION, QUANTIZATION, IDEMPOTENCY, RECONCILE

INV-050 SINGLE QUANTIZER: every order goes through a single quantizer before sending to the exchange. Ref: 05_EXECUTION_AND_FAILSAFE.md
INV-051 EXPLICIT FILTERS: when they exist on the symbol and are required for the order type, support PRICE_FILTER (minPrice/maxPrice/tickSize), LOT_SIZE (minQty/maxQty/stepSize), MIN_NOTIONAL or NOTIONAL, MARKET_LOT_SIZE when using MARKET, and TRAILING_DELTA when using trailingDelta (Spot) in STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT, TAKE_PROFIT_LIMIT. Ref: 05_EXECUTION_AND_FAILSAFE.md
INV-052 FILTERS POLICY (ENFORCED): maintain an allowlist of enforced filters by order type. Any enforced filter that is absent, inconsistent, or unknown must result in BLOCK with audit and may lead to symbol quarantine. Filters outside the enforced allowlist must not BLOCK by default. Ref: 05_EXECUTION_AND_FAILSAFE.md, 04_RISK_ENGINE_RULES.md
INV-053 DETERMINISTIC QUANTIZATION: round down conservatively and identically on every run; if it becomes invalid after quantization, it must BLOCK with reason_code. Ref: 05_EXECUTION_AND_FAILSAFE.md
INV-054 MAKER FIRST: the default EntryPlan is maker-first with short TTL and fallback only when allowed by Risk and cost. Ref: 01_DECISION_CONTRACT.md, 05_EXECUTION_AND_FAILSAFE.md, 03_STRATEGY.md
INV-055 TIME SYNC: signed REST calls use corrected timestamp (clock_offset_ms via /api/v3/time) and fixed recvWindow from config.go; a persistent timestamp error must PAUSE in Live. Ref: 05_EXECUTION_AND_FAILSAFE.md, 10_OPERATIONS_RULES.md
INV-056 NO PROTECTION GAP: any protection swap (OCO to trailing) follows a sequence with no unprotected window, with REST confirmations between steps. Ref: 05_EXECUTION_AND_FAILSAFE.md
INV-057 OPTIONAL UDS: if UserDataStream is used, maintain keepalive; if unstable, DEGRADE and rely only on REST reconcile. Ref: 08_SYSTEM_ARCHITECTURE.md, 05_EXECUTION_AND_FAILSAFE.md
INV-058 PARTIAL FILLS: normalize actual size and recalculate ExitPlan proportionally; minimum protection must exist even on failures. Ref: 05_EXECUTION_AND_FAILSAFE.md
INV-059 DUST: residuals below minNotional must not be sold automatically; record as dust and include in summary when applicable. Ref: 05_EXECUTION_AND_FAILSAFE.md
INV-060 INTENTS LEDGER: before create, cancel, or replace, record the intent in SQLite; on restart, reprocess pending intents idempotently. Ref: 00_SOURCE_OF_TRUTH.md, 05_EXECUTION_AND_FAILSAFE.md
INV-061 CLIENTORDERID: clientOrderId must be deterministic, len<=36, recommended charset [A-Za-z0-9-_], within Binance limits, contain no secrets, and must not be reused while an active/pending order with the same id exists. Ref: 01_DECISION_CONTRACT.md, 06_AUDIT_RULES.md
INV-062 PERIODIC RECONCILE: in Live, periodic REST reconcile is mandatory for orders, balances, OCO, and fills. Ref: 05_EXECUTION_AND_FAILSAFE.md
INV-063 POST-ACTION RECONCILE: after critical actions (create/cancel OCO, activate trailing, close), execute immediate reconcile and block progress if it fails. Ref: 05_EXECUTION_AND_FAILSAFE.md
INV-064 DRIFT REACTS: drift above tolerance requires PAUSE or DEGRADE with auditing. Ref: 05_EXECUTION_AND_FAILSAFE.md, 04_RISK_ENGINE_RULES.md
INV-065 RATE LIMIT: central rate limiter, exponential backoff, and degrade on 429/418; limits are not hardcoded and must be read from exchangeInfo (rateLimits) at boot and observed via headers X-MBX-USED-WEIGHT-*, X-MBX-ORDER-COUNT-*, and Retry-After. Ref: 05_EXECUTION_AND_FAILSAFE.md, 10_OPERATIONS_RULES.md
INV-066 DEGRADE: in DEGRADE, block new entries and allow only management, reconcile, and auditing; exit only after stability for a window. Ref: 05_EXECUTION_AND_FAILSAFE.md, 10_OPERATIONS_RULES.md
INV-067 SQLITE 24/7: WAL mode, busy_timeout, and 1 audit writer are mandatory; a full queue and/or lagged writer requires explicit backpressure policy (block non-critical producers and prioritize critical events) and must generate DB_WRITER_BACKPRESSURE and ALERT_RAISED; never lose auditing silently. Ref: 05_EXECUTION_AND_FAILSAFE.md, 06_AUDIT_RULES.md
INV-068 BOOT RECOVERY: on startup, run startup reconcile before new entries; choose safe mode RESUME_MANAGE_ONLY, CLOSE_SAFELY, or PAUSE_NEEDS_MANUAL. Ref: 05_EXECUTION_AND_FAILSAFE.md, 10_OPERATIONS_RULES.md
INV-069 MAKER FAILURES ARE EXPECTED: LIMIT_MAKER rejection for crossing the book must be treated as an expected event and must generate an auditable fallback or abort. Ref: 05_EXECUTION_AND_FAILSAFE.md
INV-075 EXITPLAN ALWAYS: when a position is planned, ExitPlan exists and includes TP, SL, and trailing policy with a defined mode. Ref: 01_DECISION_CONTRACT.md, 05_EXECUTION_AND_FAILSAFE.md, 03_STRATEGY.md
INV-076 IMMEDIATE OCO IN LIVE: recommendation is to create OCO on entry; any critical action requires immediate confirmatory reconcile. Ref: 05_EXECUTION_AND_FAILSAFE.md

INV-077 INTENT SENT_UNKNOWN: in Live, if a mutable call is left in an unknown state (timeout, drop, lost response), mark SENT_UNKNOWN and query REST by clientOrderId before any retry or resubmission. Ref: 01_DECISION_CONTRACT.md, 05_EXECUTION_AND_FAILSAFE.md
INV-078 FILTERS HASH DRIFT: keep exchangeInfo hash/version per symbol; on drift detection or reject by filter failure, refresh and audit FILTERS_REFRESHED (old_hash, new_hash, reason). Ref: 05_EXECUTION_AND_FAILSAFE.md, 06_AUDIT_RULES.md, 10_OPERATIONS_RULES.md
INV-079 CANCEL REPLACE: use cancelReplace only when policy allows; conservative default is STOP_ON_FAILURE and must consider unfilled order count and churn; when near penalty risk, apply backoff and/or prefer cancel+new per rule. Ref: 05_EXECUTION_AND_FAILSAFE.md, 10_OPERATIONS_RULES.md

SECTION G FAILSAFE AND LIVE OPERATION

INV-070 EXPLICIT FAIL POLICY: for failures (WS, rate limit, DB, AI, drift, clock) there is an action PAUSE, DEGRADE, RETRY, or EXIT, always audited. Ref: 00_SOURCE_OF_TRUTH.md, 05_EXECUTION_AND_FAILSAFE.md
INV-071 LIVE LOCKS: Live requires an external lock (scripts\live.ps1 with --live and optional var\LIVE.ok) plus checklist. Ref: 00_SOURCE_OF_TRUTH.md, 07_SECURITY.md, 10_OPERATIONS_RULES.md
INV-072 LIVE CHECKLIST: before operating Live, validate MODE, AI_DEC, DB, filters, WS, clock, external lock, NORMAL state, and configured limits. Ref: 07_SECURITY.md, 10_OPERATIONS_RULES.md
INV-073 AUDIT FAILURE IN LIVE: any failure that prevents consistent auditing in Live must block immediately (PAUSE or EXIT). Ref: 05_EXECUTION_AND_FAILSAFE.md, 06_AUDIT_RULES.md, 10_OPERATIONS_RULES.md
INV-074 NO DESTRUCTIVE ACTION IN LIVE: administrative endpoints or cleanup are forbidden in Live. Ref: 08_SYSTEM_ARCHITECTURE.md, 05_EXECUTION_AND_FAILSAFE.md, 07_SECURITY.md, 06_AUDIT_RULES.md

INV-090 CANONICAL SIGNALS: LOOP_STUCK, WS_STALE, REST_STALE, DISK_LOW, DB_WRITER_PRESSURE, and RECONCILE_DRIFT signals must exist with unique definitions in 00_SOURCE_OF_TRUTH.md and must generate ALERT_RAISED when crossing thresholds. Ref: 00_SOURCE_OF_TRUTH.md, 05_EXECUTION_AND_FAILSAFE.md, 06_AUDIT_RULES.md

INV-091 OPERATIONAL PANEL: the local web panel must display (first fold): SYS MODE (NORMAL|DEGRADE|PAUSE|EXIT) + since + active reason_codes; active alerts; current stage and short history (>=20); health (disk/sqlite/wal/writer/events/min/drops); pending intents highlighting SENT_UNKNOWN; reconcile drift and diffs. Ref: 10_OPERATIONS_RULES.md, README.md, 00_SOURCE_OF_TRUTH.md

INV-092 LIVE READ-ONLY PANEL: in MODE=LIVE, the panel is strictly read-only; any administrative action is forbidden and must be disabled in the UI. Ref: 07_SECURITY.md, 08_SYSTEM_ARCHITECTURE.md, 10_OPERATIONS_RULES.md

SECTION H AUDIT, JSONL, AND SQLITE

INV-080 SQLITE SOURCE: SQLite is the source of truth for querying, reconciliation analysis, and incident investigation. Ref: 06_AUDIT_RULES.md
INV-081 JSONL HUMAN TRAIL: JSONL is the human trail for debug and fast reading; it never replaces SQLite. Ref: 06_AUDIT_RULES.md
INV-082 IDS ON EVERY LINE: every JSONL line must include run_id, cycle_id, snapshot_id, decision_id, and order_intent_id when applicable. Ref: 06_AUDIT_RULES.md
INV-083 MINIMUM PERSISTENCE: SQLite persists snapshots, decisions with reason_codes, redacted ai_gate_events, orders, fills, positions, and daily_summary. Ref: 06_AUDIT_RULES.md
INV-084 AUDIT ALWAYS EMITTED: call failures, schema failures, parsing failures, timeouts, MODIFY rejects, and critical operational failures must generate auditable events in SQLite and JSONL. Ref: 06_AUDIT_RULES.md, 10_OPERATIONS_RULES.md, 07_SECURITY.md
INV-085 OFFLINE EXPERIMENTS: experiments and tuning are offline only; never run Live. Ref: 06_AUDIT_RULES.md, 10_OPERATIONS_RULES.md
INV-086 RETENTION AND BACKUP: JSONL daily rotation and retention; SQLite daily backup and optional vacuum; everything controlled via config.go; never version var\. Ref: 09_CODE_STRUCTURE.md, 06_AUDIT_RULES.md, 10_OPERATIONS_RULES.md, 07_SECURITY.md

SECTION I OBSERVABILITY AND LOCAL WEBUI

INV-090 STAGE IN CONSOLE: console shows the current loop stage in a short and auditable way (optional STAGE event). Ref: 08_SYSTEM_ARCHITECTURE.md, 09_CODE_STRUCTURE.md, 06_AUDIT_RULES.md
INV-091 LOCAL WEBUI: local web panel in Bootstrap 5.3, bind 127.0.0.1 by default, no authentication by default because it is local. Ref: 08_SYSTEM_ARCHITECTURE.md, 07_SECURITY.md, 10_OPERATIONS_RULES.md
INV-092 WEBUI METRICS: the panel shows orders and status, gain/loss, SQLite disk usage, and process RAM usage. Ref: 08_SYSTEM_ARCHITECTURE.md, 10_OPERATIONS_RULES.md
INV-095 LOCAL ALERTING REQUIRED: any entry into PAUSE, DEGRADE, or EXIT, drift above limit, low disk, DB writer saturation, AI Gate failure, and critical operational failure must fire a local alert (console and WebUI) and generate an auditable ALERT_RAISED event. Ref: 05_EXECUTION_AND_FAILSAFE.md, 06_AUDIT_RULES.md, 10_OPERATIONS_RULES.md
INV-096 LOW DISK DEGRADE/PAUSE: monitor free space of the volume and growth of audit.sqlite and audit.sqlite-wal; thresholds must lead to DEGRADE and then PAUSE in Live, with DISK_HEALTH_SAMPLE and ALERT_RAISED. Ref: 05_EXECUTION_AND_FAILSAFE.md, 06_AUDIT_RULES.md, 10_OPERATIONS_RULES.md

SECTION J CODE CONVENTIONS AND QUALITY

INV-100 NO COMMENTS: code does not use comments; names and small files explain intent; reason_codes explain decisions. Ref: 09_CODE_STRUCTURE.md
INV-101 SMALL FILES: split by single responsibility; avoid giant files. Ref: 09_CODE_STRUCTURE.md
INV-102 MINIMUM TESTS: unit tests for contracts, cost model, regime/microstructure, quantization, and idempotency; integration tests for partial fills, OCO/trailing flows, cancel/replace policy, and SENT_UNKNOWN recovery paths. Ref: 09_CODE_STRUCTURE.md, 06_AUDIT_RULES.md
INV-103 MIGRATIONS: any schema change requires a SQL migration and recording in 12_DECISIONS_LOG.md. Ref: 01_DECISION_CONTRACT.md, 09_CODE_STRUCTURE.md, 06_AUDIT_RULES.md

END
