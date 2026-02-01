05_EXECUTION_AND_FAILSAFE (EXECUTION, IDEMPOTENCY, AND FAILURES)

GOAL
Define how to execute safely and predictably in Live.
Includes: quantization/filters, true idempotency, intents, reconcile, failsafe, and degrade.

CHANGE RULE
Any change that alters execution behavior, quantization, reconcile, failsafe, or boot recovery states requires:
- updates to validations and tests
- audit impact review (06_AUDIT_RULES.md)
- an entry in 12_DECISIONS_LOG.md

LOCAL ALERTING (MANDATORY, NO INTERNET)
Goal:
- prevent silent failure in 24/7 when entering DEGRADE/PAUSE/EXIT or when there is immediate operational risk.

Rules:
- every critical event must raise an immediate local alert (console + web panel) and persist an auditable ALERT_RAISED event.
- local alerting must not require internet and must not depend on the AI Gate.
- alert content must be redacted (never include headers, tokens, signatures, full URLs with sensitive querystrings, or env vars).

Minimum triggers (always):
- entering DEGRADE, PAUSE, EXIT.
- reconcile drift above the limit.
- DB writer queue above threshold or writer lag above threshold.
- AI Gate failure (timeout, invalid schema, network error) when AI is mandatory.
- low disk / abnormal growth of audit.sqlite-wal.
- failure to refresh exchangeInfo/filters when required to operate.

Minimum local alert behavior:
- Console: a highlighted line starting with "ALERT" with stage, event_type, and reason_codes.
- Web panel: a persistent toast until manual ack, with counter and last occurrence.
- Optional: local sound (beep), enabled by local config.

Audit:
- always write ALERT_RAISED with correlation ids (run_id, cycle_id, snapshot_id when present, decision_id when present, order_intent_id when present).

Reason codes (minimum):
- ALERT_RAISED
- ENTER_DEGRADE
- ENTER_PAUSE
- ENTER_EXIT
- DRIFT_LIMIT_EXCEEDED
- DB_WRITER_QUEUE_HIGH
- DB_WRITER_BACKPRESSURE
- AIGATE_TIMEOUT
- AIGATE_SCHEMA_INVALID
- AIGATE_PARSE_FAIL
- DISK_LOW_DEGRADE
- DISK_LOW_PAUSE
- FILTERS_DRIFT_DETECTED
- RATE_LIMIT_429
- RATE_LIMIT_418
- RECONCILE_DIFF_DETECTED
BINANCE FILTERS AND QUANTIZATION
Rule:
- every order (Live) must pass a single QUANTIZER before sending.
- the QUANTIZER applies exchangeInfo+symbol filters and guarantees valid orders for the order types the system actually uses.

Minimum filters to ENFORCE (when they exist for the symbol and when applicable to the order type):
- PRICE_FILTER: minPrice, maxPrice, tickSize (price)
- LOT_SIZE: minQty, maxQty, stepSize (base quantity)
- MIN_NOTIONAL or NOTIONAL: minimum quote value (quote)
- MARKET_LOT_SIZE (only if using MARKET): minQty, maxQty, stepSize (base quantity)
- TRAILING_DELTA (only if using native trailing): bounds for trailingDelta (BIPS)
  - conversion (integers): trailing_delta_bips = trailing_distance_bps  // 1 bps = 1 bips = 0.01%
  - validation: the QUANTIZER must validate trailing_delta_bips against TRAILING_DELTA min/max/step before sending.

Filter policy (avoid drift and avoid unintended mass blocking):
- the QUANTIZER must explicitly know the ENFORCED filters above.
- additional known-but-not-ENFORCED filters (e.g., ICEBERG_PARTS, MAX_NUM_ORDERS, MAX_NUM_ALGO_ORDERS, PERCENT_PRICE) must be recorded in the per-symbol cache and in audit, but must not block by default.
- unknown filters must generate audit (FILTERS_DRIFT_DETECTED) and mark the symbol as "requires review", without blocking by default.
- if an order is rejected by the exchange with "Filter failure: X" (or equivalent), the system must:
  - record the filter name in audit
  - quarantine the symbol
  - block new entries on the symbol until quarantine TTL expires and the filter is reviewed
- if exchangeInfo/filters are missing or inconsistent (e.g., stepSize=0, tickSize=0), it must block and quarantine the symbol.

Drift detection (exchangeInfo changed) - mandatory:
- keep a per-symbol cache with filters_hash (hash of normalized JSON of the ENFORCED filters and symbol_status).
- boot: load exchangeInfo and populate filters_hash for all eligible symbols.
- periodic refresh: reload exchangeInfo at a configured interval and update the cache.
- refresh on rejection: when receiving a rejection with "Filter failure" (or equivalent) or a TRAILING_DELTA failure, reload exchangeInfo immediately.
- refresh on status: if symbol_status != TRADING (e.g., HALT/PAUSE), block new entries on the symbol and record audit.
- on update: emit an auditable FILTERS_REFRESHED event with old hash, new hash, and cause (PERIODIC|REJECT|STATUS).

Cache usage rules:
- the executor must use the current cache to quantize/validate every order.
- if filters_hash is missing for an eligible symbol, block execution and require refresh.

Behavior:
- always round down to valid values (conservative) and never "round up" to pass minQty/minNotional.
- validate price/qty min/max after quantization.
- if after quantization the order becomes invalid (notional below minimum, qty=0, outside minQty/maxQty/stepSize/tickSize/minNotional), it must BLOCK with reason_code.
- quantization must be deterministic and identical across all call sites.

Files:
- internal\infra\binance\filters.go : loads and caches per-symbol filters and exposes the raw list.
- internal\engine\executor\quantize.go : applies ENFORCED filters and validates.
- internal\domain\reasoncodes\codes.go: reason codes for invalidity by filters.

Reason codes:
- FILTERS_DRIFT_DETECTED
- FILTERS_REFRESHED
- PROTECTION_INVALID_FILTER
- PROTECTION_INVALID_MIN_NOTIONAL
- SYMBOL_QUARANTINED
- ORDER_SUBMIT_REJECTED
- ORDER_SUBMIT_TIMEOUT

TIME SYNC, TIMESTAMP, AND RECVWINDOW (LIVE CRITICAL)
Goal:
- avoid timestamp rejections and make signed REST calls predictable and recoverable.

Rules:
- maintain a clock_offset_ms computed via GET /api/v3/time.
- apply timestamp = now_ms + clock_offset_ms on EVERY signed call.
- use a fixed, deterministic recvWindow (time_sync_recv_window_ms) defined in internal\config\config.go.
  Default (00_SOURCE_OF_TRUTH.md): 5000 ms.
- on timestamp error:
  1) re-sync clock_offset_ms once
  2) retry the call once only when the endpoint supports explicit idempotency (idempotency key).
  3) if it fails again, enter PAUSE (Live) with audit and no new entries

Periodicity:
- periodic re-sync every time_sync_interval_ms and also on network reconnect events.
  Default (00_SOURCE_OF_TRUTH.md): 300000 ms (5 minutes).

Reason codes:
- TIME_SYNC_FAIL
- BINANCE_TIMESTAMP_REJECTED
- CLOCK_DRIFT_WARN

ENTRY (ENTRYPLAN) AND MAKER-FIRST
Entry is not fixed. It is a deterministic plan (EntryPlan) chosen by expected cost and conditions.

Compatibility note (Binance Spot):
- IOC and FOK are timeInForce for LIMIT orders (they are not separate "types").
- LIMIT_MAKER is post-only: if it crosses the book, Binance may reject the order.
  Maker rejection for crossing the book is an expected event and must generate audited fallback.

Maker-first entry (recommended default):
1) Try LIMIT_MAKER with a short TTL.
2) If it does not fill:
   - if the signal is still valid and cost acceptable: reprice (idempotent) and try again.
   - otherwise: abort (not worth paying expensive fills).
3) If the market accelerated and cost is still within the cap:
   - use an aggressive LIMIT IOC/FOK or MARKET per policy.

Limit vs Market (WS + REST):
- entry chooses between:
  - LIMIT_MAKER (post-only) with TTL
  - aggressive LIMIT with IOC/FOK
  - MARKET (when the risk of not filling is worse than slippage)
- always limited by MAX_SLIPPAGE_BPS and volatility/edge rules.

Data used:
- WS bookTicker: best bid/ask.
- REST depth: book for estimated impact.
- REST exchangeInfo+filters: tickSize/stepSize/minNotional/minQty/maxQty.

Files:
- internal\engine\strategy\entry_plan.go : generates EntryPlan (maker-first + fallback).
- internal\engine\risk\entry_limits.go : validates cost/volatility limits to allow fallback.
- internal\infra\binance\rest_depth.go (as part of rest.go or dedicated file): depth/exchangeInfo.
- internal\engine\executor\orders.go : idempotent create/cancel/replace with TTL.
- internal\engine\executor\quantize.go : quantization by filters.
- internal\domain\contracts\decision.go: persist EntryPlan and parameters.
- internal\domain\reasoncodes\codes.go: entry reason codes.

Reason codes:
- STRAT_ENTRY_MAKER_TTL
- STRAT_MAKER_REPRICE
- STRAT_MAKER_REJECTED_CROSSES_BOOK
- STRAT_ENTRY_TIMEOUT
- STRAT_FALLBACK_IOC
- STRAT_FALLBACK_MARKET
- STRAT_ENTRY_ABORTED_COST

CANCEL/REPLACE (REPRICE) AND OPERATIONAL LIMITS
Goal:
- allow maker-first repricing without generating unnecessary unfilled-order penalties and rate limiting.

Rules:
- the repricing loop must have a strict per-intent limit (e.g., max_reprices per TTL).
- when supported and allowed, prefer /api/v3/order/cancelReplace for repricing LIMIT orders with deterministic clientOrderId.
- conservative default: cancelReplaceMode=STOP_ON_FAILURE.
- if cancelReplace returns NOT_ATTEMPTED, consider that order count was consumed and audit the event.
- if the symbol is near unfilled order count limits or throttle indicates pressure, disable repricing and operate with safe fallback (IOC/MARKET or abort) per EntryPlan and Risk.
- never run repricing in a tight loop; always respect backoff and a central throttle.

Audit:
- record repricing events with event_type ORDER_CANCEL_REPLACE and an outcome field (ATTEMPTED|NOT_ATTEMPTED|DONE), plus associated reason_codes.

EXIT (EXITPLAN), OCO, AND TRAILING
TP/SL by volatility:
- SL = entry - k * ATR
- TP = entry + m * ATR
- k/m vary by regime (TREND vs RANGE).
- Risk validates sanity: SL cannot be too tight vs ATR and tickSize.

Coherence rules for stopPrice (Spot):
- for LONG position protection:
  - STOP_LOSS / STOP_LOSS_LIMIT: stopPrice must be below the reference price (below market).
  - TAKE_PROFIT / TAKE_PROFIT_LIMIT: stopPrice must be above the reference price (above market).
- there is no SHORT protection in Spot (this project is Spot without short). Any attempt must be blocked.

Rule:
- if coherence cannot be satisfied after tickSize quantization, block protection and follow the no-gap policy (fallback or PAUSE_NEEDS_MANUAL).

Smart trailing:
- enable only after profit_bps >= threshold AND regime/trend confirmation.
- trailing distance proportional to ATR (t * ATR).
- trailing mode (config.go):
  - NATIVE: use trailingDelta (BIPS) when the symbol exposes TRAILING_DELTA and when protection uses a compatible stop/take-profit order type; validate TRAILING_DELTA before sending; if not supported, BLOCK NATIVE.
    - fill trailingDelta with an integer: trailing_delta_bips = ExitPlan.trailing_distance_bps
    - validate TRAILING_DELTA in the QUANTIZER (min/max/step); if invalid or absent, BLOCK NATIVE.
  - VIRTUAL: simulate trailing via a state machine (cancel/replace), keeping the protection invariants with no gap; never trust the exchange to adjust trailing.
  - AUTO: try NATIVE; if rejected by filter/compatibility error, record and fall back to VIRTUAL.

OCO -> Trailing (Live recommended, no protection gap window)
Definitions:
- "swap trigger" is when the position already has sufficient profit and regime confirms TREND, per deterministic rules:
  - profit_bps >= trailing_enable_profit_bps
  - regime=TREND (or trend_score >= threshold)
  - microstructure not degraded (spread/impact within tolerance)
- "active protection" means there exists exchange-side protection verifiable by REST (stop/oco/native trailing) OR an active virtual protection managed locally.

Safe sequence (Live):
1) Create OCO immediately on entry (TP + SL) and confirm via REST.
2) When swap trigger occurs:
   2.1) Prepare trailing and record swap start in the audit trail.
   2.2) Create the new protection (native trailing OR an equivalent stop/limit order for virtual) and confirm via REST.
   2.3) Only after confirming the new protection, cancel the previous OCO and confirm via REST.
   2.4) Record swap confirmation in the audit trail.
3) If any step fails, do not cancel existing protection. Enter PAUSE (Live) and require human intervention.

- simulate the same state machine and record the same events (no protection gaps).

Additional reason codes:
- PROTECTION_INSTALL_FAILED
- PAUSE_NEEDS_MANUAL_PROTECTION

Additional rule (Live critical):
- after any critical action (create/cancel OCO, activate trailing), execute an immediate confirmatory REST reconcile.
  (the periodic reconcile still exists; this is a post-action "confirmatory" reconcile)

USER DATA STREAM (LISTENKEY) - OPTIONAL, DO NOT TRUST AS TRUTH
Rules:
- if using UserDataStream for order/fill events:
  - maintain listenKey with periodic keepalive.
  - if keepalive fails or the stream expires/disconnects repeatedly:
    - enter DEGRADE (no new entries) and operate only with REST reconcile.
- never assume final fill only via WS; REST is confirmatory in Live.
- record UDS health events with audit.

Reason codes:
- WS_OOO_EVENT
- ENTER_DEGRADE

Files:
- internal\engine\strategy\exit_plan.go : creates ExitPlan (ATR TP/SL and rules).
- internal\engine\risk\exit_sanity.go : validates TP/SL/trailing coherence vs ATR and regime.
- internal\engine\position\oco.go : OCO lifecycle.
- internal\engine\position\trailing.go : trailing lifecycle (native/virtual).
- internal\engine\position\reconcile.go : REST reconcile of orders/positions.

Reason codes:
- STRAT_EXIT_ATR_TP_SL
- STRAT_EXIT_INVALID
- STRAT_TRAILING_ARM_ALLOWED
- STRAT_TRAILING_ARM_BLOCKED
- PROTECTION_INSTALL_FAILED
- PROTECTION_INVALID_FILTER
- PAUSE_NEEDS_MANUAL_PROTECTION

PARTIAL FILLS (OPERATIONAL POLICY)
Problem:
- partial fills create a smaller-than-planned position.
- OCO/ExitPlan must be recalculated for the actual size, and protection must exist even under failures.

Rules:
- if entry remains partial and the remainder expires/cancels:
  - normalize the position to the actual size
  - recalculate ExitPlan (TP/SL/Trailing) proportionally
  - create minimum protection immediately (OCO if possible; otherwise safe fallback)
- if OCO fails in Live:
  - apply failsafe: PAUSE + attempt alternative protection defined in local policy and audit.

Files:
- internal\engine\position\partial_fill_policy.go : normalization and recalculation.
- internal\engine\position\oco.go: supports proportional creation for actual size.
- internal\engine\failsafe\handlers.go: protection fallback and pause.

Reason codes:
- PROTECTION_INSTALL_FAILED
- PAUSE_NEEDS_MANUAL_PROTECTION

DUST (RESIDUALS)
Problem:
- after sells/buys, residuals below minNotional/stepSize (dust) may remain.
- attempting to sell dust usually fails due to filters.

Rules:
- when closing a position, if the remainder is below minNotional:
  - record as DUST
  - do not attempt to sell/adjust automatically
  - include in daily_summary as dust_value_estimated (optional)

Files:
- internal\engine\position\dust.go : dust detection and policy.
- internal\engine\reports\daily_summary.go: dust aggregation (optional).

Reason codes:
- CLOSE_SAFELY_DUST

TRUE IDEMPOTENCY (ORDER INTENT LEDGER)
Problem:
- clientOrderId helps, but does not solve restart/idempotency by itself.
- an intents ledger is required to guarantee operational "exactly-once" (in practice).

Rules:
- before sending an order (or cancel/replace), record an intent in SQLite:
  - order_intent_id, decision_id, action type, quantized payload, ts
- after REST response, update the intent with status and real ids.

Minimum intent states (SQLite):
- CREATED: intent recorded locally and not yet sent.
- SENT_UNKNOWN: mutation was triggered, but REST confirmation was not received (timeout, drop, disconnect).
- CONFIRMED: mutation confirmed via REST (orderId / status) or confirmed via reconcile.
- NOT_FOUND: after confirmatory lookup, there is no evidence of the mutation on Binance.
- FAILED_TERMINAL: terminal failure (rejection, invalid filter, permission).

"when in doubt, do not resubmit" (Live rule):
- if a submit/cancel/replace returns network error/timeout after send, mark SENT_UNKNOWN.
- in SENT_UNKNOWN, before any retry/resend, query REST to decide:
  - mandatory parameters (internal\config\config.go):
    - intent_rest_query_timeout_ms (int) default 5000
    - intent_max_rest_queries (int) default 3
  - deterministic query algorithm:
    - perform up to intent_max_rest_queries REST queries (each with timeout intent_rest_query_timeout_ms).
    - if any query confirms the mutation: transition to CONFIRMED and continue normal management.
    - if all queries succeed and there is no evidence: transition to NOT_FOUND and apply the conservative action below.
    - if intent_max_rest_queries is exceeded without confirmation due to timeout/error: keep status SENT_UNKNOWN, emit ALERT_RAISED with reason_code=INTENT_SENT_UNKNOWN and detail=REST_QUERY_EXHAUSTED, and apply:
      - for entry: block new entry in the current cycle; in MODE=LIVE, escalate to DEGRADE immediately.
      - for protection: in MODE=LIVE, enter PAUSE_NEEDS_MANUAL (fail-closed) until manual resolution/reconcile confirms valid protection.
  - lookup by clientOrderId (when present) or scan openOrders/allOrders and reconcile.
  - if found, transition to CONFIRMED and continue normal management.
  - if confirmed not found, transition to NOT_FOUND and then decide the conservative action:
    - apply the conservative action per intent type as defined above (entry vs protection).
- never blindly resubmit a mutation in Live without the REST lookup step above.

- on restart, reprocess pending intents (idempotently) before new entries.

Files:
- internal\engine\executor\intent_ledger.go : write/update intents.
- migrations\00xx_order_intents.sql : order_intents table and indexes.
- internal\infra\sqlite\queries.go: queries for pending intents.

Reason codes:
- INTENT_SENT_UNKNOWN
- INTENT_CONFIRMED_BY_REST
- INTENT_NOT_FOUND_BY_REST

REST RECONCILIATION (FINAL TRUTH) + DRIFT (FAIL-CLOSED)

GOAL
- REST is the final truth in LIVE.
- Reconcile detects divergence between what the bot believes (derived state + intents) and what exists on the exchange.

WHAT IS DRIFT
- drift is an integer score (drift_score_x10000) computed from normalized diffs of:
  - open orders (missing/extra/status diverged)
  - position (qty/side diverged)
  - protection (OCO/stop/trailing diverged)
- drift_score_x10000 = 0 means "no drift".

WHEN TO RUN
- periodic REST reconcile (interval reconcile_rest_interval_ms).
- immediate reconcile after critical actions:
  - submit/cancel/replace
  - install/modify protection (OCO/stop/trailing)
  - close position

THRESHOLDS (MANDATORY CONFIG)
Defaults:
- reconcile_rest_interval_ms: 5000   (5 seconds)
- reconcile_drift_degrade_score_x10000: 20000   (score >= 2.0000 => DEGRADE)
- reconcile_drift_pause_score_x10000:   50000   (score >= 5.0000 => PAUSE in LIVE)

ACTIONS (MANDATORY)
- If drift_score_x10000 >= reconcile_drift_degrade_score_x10000:
  - enter DEGRADE.
  - block new entries.
  - allow only reconcile + protection management.
- If drift_score_x10000 >= reconcile_drift_pause_score_x10000:
  - in LIVE: enter PAUSE (fail-closed) and require manual action.


Audit (mandatory):
- each relevant diff must emit a RECONCILE_DIFF event (for drilldown and panel).
- every mode transition must emit ALERT_RAISED with:
  - reason_codes: [DRIFT_LIMIT_EXCEEDED, ENTER_DEGRADE] or [DRIFT_LIMIT_EXCEEDED, ENTER_PAUSE]
  - minimum details: {"drift_score_x10000":<int32>,"dur_ms":<int64>,"last_reconcile_rest_ts_ms":<int64>}

Reason codes:
- DRIFT_LIMIT_EXCEEDED
- RECONCILE_DIFF_DETECTED

Files:
- internal\engine\position\reconcile.go : REST reconcile and diff generation.
- internal\engine\risk\reconcile_policy.go : thresholds and actions.
- internal\app\loop.go: schedules reconcile.

AIGATE_CALL (AI GATE) - OPERATIONAL BEHAVIOR AND FAIL-CLOSED IN LIVE
Goal:
- evaluate the Decision proposed by Strategy before Risk, with ALLOW/BLOCK/MODIFY (MODIFY is conservative only).
- guarantee predictability, determinism, and auditing on timeout/error/invalid schema.

Preconditions (mandatory before the call):
- decision_id already computed deterministically.
- snapshot_id present and snapshot_hash available (canonical hash of the snapshot used in the payload).
- input_hash computed (canonical hash of the payload sent to the AI Gate).
- payload redacted: never include secrets, headers, keys, full URLs, or data outside the Snapshot.
- in LIVE, if AI_DEC=2, the system must be in NORMAL to open new entries (DEGRADE/PAUSE block entries).

Call:
- max timeout defined in internal\config\config.go.
- response must be parseable by strict schema (prompts\schemas\ai_gate_result.schema.json).

Return validation (always local, no blind trust):
- verdict must be ALLOW | BLOCK | MODIFY.
- reasons never empty.
- enabled=true requires non-empty input_hash and snapshot_hash.
- verdict=MODIFY requires modified_decision present and valid.
- modified_decision must follow the conservative contract rules (01_DECISION_CONTRACT.md); otherwise MODIFY is rejected locally.

Failure policy (timeout/error/invalid schema):
- if AI_DEC=0 (disabled): skip AI Gate and proceed to Risk (ai_gate.enabled=false) with audit.
- if AI_DEC=1 (optional): AI failure results in a conservative BLOCK of the entry and continues management/reconcile; never open an entry with ambiguous AI.
- if AI_DEC=2 (mandatory):
  - in LIVE: timeout/error/invalid schema => FAIL-CLOSED: PAUSE the system and BLOCK the current decision, with audit.


Reasons and audit (minimum):
- always write an AIGATE_CALL event in SQLite and JSONL, including on failure.
- minimum event fields: decision_id, snapshot_id, snapshot_hash, input_hash, verdict (if present), model (if present), latency_ms (if present), raw_hash (if present), reasons.
- all raw model content must be treated as redacted; store only raw_hash and a redacted payload.

Reason codes (AIGATE_CALL):
- AIGATE_TIMEOUT
- AIGATE_SCHEMA_INVALID
- AIGATE_PARSE_FAIL
- AIGATE_MODIFY_INVALID (when verdict=MODIFY violates local conservative rules)
- AIGATE_REASON_UNKNOWN (when the return includes a reason code outside the catalog and the result is invalidated)

Files:
- internal\engine\aigate\client.go : call/timeout/schema.
- internal\engine\aigate\enforce.go : local validations and rejection of non-conservative MODIFY.
- internal\audit\writer.go + migrations : persistence of ai_gate_events.
- internal\domain\reasoncodes\codes.go : reason codes above.

FAIL POLICY (PAUSE/DEGRADE/RETRY/EXIT)
Common problems:
- WS down/disconnected
- rate limit / 429
- DB lock/corrupt
- OpenAI unavailable
- reconcile drift
- clock drift / out-of-order events
- rejection by filters/invalid quantization

Rule:
- explicit Fail Policy per category:
  PAUSE | DEGRADE | RETRY | EXIT
- in Live:
  - audit failure (DB) blocks immediately
  - AI_DEC=2 requires AI; failure -> PAUSE (no new entries) and BLOCK current decision with reason_code
- rate-limit:
  - exponential backoff
  - degrade: stop opening new positions; manage/close and reconcile only
- always record reason_code and the action taken.

Files:
- internal\engine\failsafe\policy.go : failure->action matrix.
- internal\engine\failsafe\handlers.go : executes action and audits.
- internal\app\loop.go: integrates failures and degrade/pause.
- internal\domain\reasoncodes\codes.go: failure reason codes.

Reason codes:
- ENTER_PAUSE
- ENTER_DEGRADE
- ENTER_EXIT
- RATE_LIMIT_429
- RATE_LIMIT_418
- DB_WRITER_QUEUE_HIGH
- DB_WRITER_BACKPRESSURE
- DISK_LOW_DEGRADE
- DISK_LOW_PAUSE
- FILTERS_DRIFT_DETECTED
- WS_OOO_EVENT
- CLOCK_DRIFT_WARN
- TIME_SYNC_FAIL
- BINANCE_TIMESTAMP_REJECTED

RATE LIMIT AND CENTRAL THROTTLE
Problem:
- per-symbol depth/exchangeInfo can blow rate limits.
- 429 in Live must become degrade, not "blind persistence".

Rules:
- central rate limiter per REST endpoint.
- depth snapshots must have throttle and per-symbol cache (TTL).
- on rate limit:
  - exponential backoff
  - DEGRADE: block new entries, keep only management/reconcile

Dynamic rate limits (mandatory, no hardcode):
- boot: read exchangeInfo.rateLimits and store in in-memory state (for REQUEST_WEIGHT, ORDERS, and applicable limits).
- runtime: observe response headers and adjust throttle:
  - X-MBX-USED-WEIGHT-*
  - X-MBX-ORDER-COUNT-*
- on 429/418:
  - honor Retry-After when present
  - apply backoff and raise mode to DEGRADE per fail policy
  - write audit for the rate-limit event and raise ALERT_RAISED when it affects the loop

Files:
- internal\infra\binance\ratelimit.go : rate limiter/backoff.
- internal\infra\binance\rest_depth.go: per-symbol TTL cache and throttle.
- internal\engine\failsafe\policy.go: action for 429.

Reason codes:
- RATE_LIMIT_418
- RATE_LIMIT_429
- RETRY_AFTER_APPLIED

DEGRADE MODE
Goal:
- when WS/REST/AI degrades, do not open new positions, but continue protecting and closing if needed.

Rules:
- DEGRADE blocks Universe/Rank/DeepScan/entries.
- DEGRADE allows:
  - reconcile
  - position management (OCO/trailing/close) per safe policy
  - auditing and logs
- exit from DEGRADE only after stability for degrade_exit_stable_window_ms.

Files:
- internal\app\mode_degrade.go : degrade state and loop gates.
- internal\engine\failsafe\policy.go: decides entering/exiting degrade.

Reason codes:
- ENTER_DEGRADE

SQLITE IN 24/7 (WAL, BUSY TIMEOUT, 1 WRITER)
Problem:
- SQLite concurrency causes DB_LOCK/DB_BUSY.
- 24/7 requires predictable writing and recovery.

Rules:
- enable WAL mode and busy_timeout in SQLite.
- use 1 writer goroutine for audit/events (queue).
- if the queue fills: degrade/pause (never silently lose audit).

Writer saturation policy (no silent loss):
- in Live, writer pressure must immediately block production of non-critical events (backpressure).
- critical events (orders, intents, reconcile, failsafe, protection) must have priority (fast lane) and must not be dropped.
- event dropping may exist only for non-critical events and only if explicitly allowed; every drop must be audited and must raise ALERT_RAISED.

Minimum thresholds (config.go):
- audit_writer_queue_hi_watermark_pct: when exceeded, enter DEGRADE and raise ALERT_RAISED.
- audit_writer_queue_full: when reached, block new entries and raise PAUSE (Live) until recovered.
- audit_writer_max_lag_ms: if writer lag exceeds the limit, treat as an operational failure (DEGRADE/PAUSE depending on context).

DISK HEALTH AND LOW DISK MODE (WAL/DB)
Problem:
- SQLite in WAL can grow, and on full disk it can enter DB_BUSY/DB_LOCK loops or fail writes.

Rules:
- continuous health check must collect:
  - bytes of audit.sqlite
  - bytes of audit.sqlite-wal
  - free space of the volume where SQLite resides
- thresholds:
  - crossing threshold 1: enter DEGRADE (block new entries) and raise ALERT_RAISED.
  - crossing threshold 2 or write failure due to disk: enter PAUSE (Live) and require manual action.
- each sample must be auditable via DISK_HEALTH_SAMPLE event.
- Emit DISK_HEALTH_SAMPLE at a fixed interval: disk_health_sample_interval_ms.
  Default (00_SOURCE_OF_TRUTH.md): 5000 ms.

LOOP LIVENESS AND FEED HEALTH (FAIL-CLOSED)

GOAL
Avoid "operating blind" due to a stuck loop or stale data sources.
This section must be consistent with 00_SOURCE_OF_TRUTH.md canonical operational signals.

CANONICAL CONFIG NAMES (NO ALIASES)
The following config field names are the only valid names for liveness/feed thresholds:
- loop_stuck_ms_degrade, loop_stuck_ms_pause
- ws_stale_ms_degrade, ws_stale_ms_pause
- rest_stale_ms_degrade, rest_stale_ms_pause

If any of the fields above is missing at boot, the system must enter PAUSE (fail-closed) and raise ALERT_RAISED with reason_code STRAT_CONFIG_INVALID.

DEFINITIONS (MANDATORY)
- last_progress_ts_ms: last local timestamp when observable loop progress occurred.
  Observable progress = at least one STAGE_CHANGED event in the current cycle.
- ws_last_msg_ts_ms: local timestamp of the last consumed market-data message used for state (bookTicker or equivalent).
- rest_last_success_ts_ms: local timestamp of the last successful REST call used to reconcile/confirm state.

Derived (computed locally; integers only):
- loop_stuck_delta_ms = now_ms - last_progress_ts_ms
- ws_stale_delta_ms   = now_ms - ws_last_msg_ts_ms
- rest_stale_delta_ms = now_ms - rest_last_success_ts_ms

ACTIONS (MANDATORY)

1) LOOP_STUCK
- if loop_stuck_delta_ms >= loop_stuck_ms_degrade:
  - enter DEGRADE (block new entries; allow manage/reconcile only).
  - raise reason_code LOOP_STUCK_DEGRADE and emit ALERT_RAISED with details:
    {"loop_stuck_delta_ms":<int64>,"last_progress_ts_ms":<int64>,"threshold_ms":<int64>}.
- if loop_stuck_delta_ms >= loop_stuck_ms_pause:
  - enter PAUSE.
  - raise reason_code LOOP_STUCK_PAUSE and emit ALERT_RAISED with the same detail keys.

2) WS_STALE
- if ws_stale_delta_ms >= ws_stale_ms_degrade:
  - enter DEGRADE and raise reason_code WS_STALE_DEGRADE.
- if ws_stale_delta_ms >= ws_stale_ms_pause:
  - enter PAUSE and raise reason_code WS_STALE_PAUSE.
- Audit details must include:
  {"source":"WS","delta_ms":<int64>,"last_ts_ms":<int64>,"threshold_ms":<int64>}.

3) REST_STALE
- if rest_stale_delta_ms >= rest_stale_ms_degrade:
  - enter DEGRADE and raise reason_code REST_STALE_DEGRADE.
- if rest_stale_delta_ms >= rest_stale_ms_pause:
  - enter PAUSE and raise reason_code REST_STALE_PAUSE.
- Audit details must include:
  {"source":"REST","delta_ms":<int64>,"last_ts_ms":<int64>,"threshold_ms":<int64>}.

RECOVERY RULES (MANDATORY)
- Exit from DEGRADE is allowed only when ALL deltas below are below their DEGRADE thresholds for a continuous window:
  - loop_stuck_delta_ms < loop_stuck_ms_degrade
  - ws_stale_delta_ms < ws_stale_ms_degrade
  - rest_stale_delta_ms < rest_stale_ms_degrade
- Stability window: degrade_exit_stable_window_ms (default 60000).

IMPLEMENTATION NOTES
- last_progress_ts_ms must be updated in a single place in the loop and must not depend on external IO.
- WS and REST timestamps use local monotonic time where possible; persist as epoch ms for audit.
- PAUSE never auto-reverts to NORMAL without explicit recovery action, per 00_SOURCE_OF_TRUTH.md.

RECOVERY AFTER CRASH (BOOT)
Problem:
- a 24/7 bot may crash mid-position, mid-OCO, or with pending orders.
- on restart, it must not "assume" state; it must reconstruct and choose safe behavior.

Rules:
- on startup, run mandatory STARTUP RECONCILE before any new entry.
- reconstruct state using REST (final truth) + SQLite (latest intents/decisions) and then choose a mode:
  - RESUME_MANAGE_ONLY: position/orders exist; manage and protect, no new entries.
  - CLOSE_SAFELY: critical inconsistency; try to close using safe policy (or install protection) and pause.
  - PAUSE_NEEDS_MANUAL: high risk/ambiguous state; pause and require intervention (auditable).

Files:
- internal\app\startup_recover.go : boot recovery flow.
- internal\engine\position\recover.go : reconstructs position/orders and normalizes state.
- internal\engine\executor\intent_ledger.go : safe reprocessing of pending intents.

Reason codes:
- ENTER_PAUSE
- PAUSE_NEEDS_MANUAL_PROTECTION
- RECONCILE_DIFF_DETECTED
- INTENT_SENT_UNKNOWN
- SYMBOL_QUARANTINED

IMPLEMENTATION NOTES (NON-NEGOTIABLE)
- Executor must not send anything without filter-based quantization.
- In Live, every critical action requires confirmatory reconcile.
- On audit failure (SQLite writer) in Live: block/pause immediately.

WEB PANEL INTEGRATION
- the panel consumes execution data (orders/fills/positions) via SQLite queries.
- any destructive action must be blocked in Live.

REFERENCES
- Contracts: 01_DECISION_CONTRACT.md
- Snapshot: 02_DATA_SNAPSHOT_SPEC.md
- Strategy: 03_STRATEGY.md
- Risk: 04_RISK_ENGINE_RULES.md
- Audit: 06_AUDIT_RULES.md
- Security: 07_SECURITY.md
- Architecture: 08_SYSTEM_ARCHITECTURE.md
- Operations: 10_OPERATIONS_RULES.md
- Decisions log: 12_DECISIONS_LOG.md
