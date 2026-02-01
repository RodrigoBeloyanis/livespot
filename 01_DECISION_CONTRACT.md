01_DECISION_CONTRACT (CONTRACTS AND INVARIANTS)

GOAL
Define the contracts (Decision, Snapshot refs, RiskVerdict, AI Gate Result) and invariants.
This file is law. Any change here requires:
- schema updates
- validation updates
- migrations if needed
- an entry in 12_DECISIONS_LOG.md

MAIN ENTITIES (SUMMARY)
FORMAL SEPARATION: StageName vs AuditEventType vs ReasonCode

StageName
- Enum of the 24/7 cycle (loop stages). Source: 08_SYSTEM_ARCHITECTURE.md.
- Used in Decision.stage and in stage-change events.

AuditEventType
- Enum of auditable event types (e.g., AIGATE_CALL, ORDER_SUBMIT, RECONCILE_DIFF, ALERT_RAISED).
- Used only in the audit trail (SQLite/JSONL). Does not replace stage.

ReasonCode
- Enum of atomic reasons (single catalog in this document).
- Used in Decision.reasons, RiskVerdict.reasons, and AIGateResult.reasons.

Validations
- stage accepts only StageName.
- event_type accepts only AuditEventType.
- reasons accepts only ReasonCode.


REASONS REPRESENTATION (CANONICAL)
Rules:
- Reasons are serialized as a JSON array of ReasonCode strings (e.g., ["STRAT_EDGE_OK"]).
- When reasons are not applicable, serialize an empty array [] (never use 0, empty string, null, or missing).
- Numeric IDs shown in this document are documentation-only and MUST NOT appear in persisted JSON.
- The ReasonCode "OK" MUST NOT be used as a placeholder for "no reasons"; use [] instead.


- Snapshot: reference to deterministic data (see 02_DATA_SNAPSHOT_SPEC.md)
- Strategy: deterministic rules for signals/edge/entry/exit (see 03_STRATEGY.md)
- Decision: proposed action (entry/exit) with reasons and a complete plan.
- RiskVerdict: final deterministic decision (ALLOW/BLOCK) with reasons.
- AIGateResult: structured return ALLOW/BLOCK/MODIFY (MODIFY is conservative).

GLOBAL SELECTION -> TOPN -> TOPK (CONTRACT)
The pipeline includes deterministic symbol selection steps:
- universe_scan: filters the global universe to eligible symbols.
- rank_topn: ranks by liquidity/volume + radar momentum.
- deep_scan: analyzes candidates with signals and regime.
- select_topk: chooses finalists for watchlist/execution.

Invariants:
- selection must be deterministic given snapshot + config.
- every cut/filter must emit reason_codes (auditable).
- every intermediate list must be auditable (SQLite + JSONL).

DECISION (CONTRACT)
Decision is the canonical object representing a proposed action.

Required fields:
- mode: LIVE
- ts_ms: local timestamp when generated
- symbol: string
- side: BUY | SELL
- intent: ENTRY | EXIT | MANAGE
- entry_plan: EntryPlan (when intent=ENTRY)
- exit_plan: ExitPlan (when intent=ENTRY or MANAGE)
- edge_score_x10000: int (0..10000, normalized score; no float in the contract)
- edge_bps_expected: int (expected bps, net of modeled fees/slippage)
- reasons: []ReasonCode (fixed catalog)
- snapshot_id: string
- decision_id: string (deterministic)
- cycle_id: string
- stage: StageName (see 08_SYSTEM_ARCHITECTURE.md)
- constraints: DecisionConstraints (quantization, filters, and limits)
- ai_gate: AIGateResult (when used)
- risk_verdict: RiskVerdict (always at the end)

Invariants:
- decision_id must be deterministic from DecisionHashPayload (RFC 8785 + SHA-256).
- decision must always include snapshot_id and cycle_id.
- reasons are never empty when intent=ENTRY.
- edge_bps_expected must be integer and in bps.
- entry_plan and exit_plan must not conflict (e.g., stop above entry for BUY).
- ai_gate is populated only after stage AIGATE_CALL.
- risk_verdict can only be populated after stage RISK_VERDICT.

DECISIONCONSTRAINTS (CONTRACT)
Fields:
- tick_size: string (decimal, e.g., "0.01000000")
- step_size: string (decimal, e.g., "0.00001000")
- min_qty: string (decimal, e.g., "0.00100000")
- min_notional: string (decimal, e.g., "10.00000000")
- price_precision: int (e.g., 2)
- qty_precision: int (e.g., 6)
- max_qty: string (decimal, e.g., "1000.00000000")
- max_num_orders: int (e.g., 200)
- max_algo_orders: int (e.g., 5)
- max_notional: string (decimal, optional)
- quantization_policy: enum (ENFORCED | KNOWN_NON_ENFORCED | UNKNOWN)

Invariants:
- values must be decimal strings (no float, no scientific notation).
- all decimal strings use '.' as separator.
- quantization_policy must be present and auditable.
- ENFORCED means the filter is validated before order submission.
- KNOWN_NON_ENFORCED means the filter exists but is not validated.
- UNKNOWN means the filter was not found in exchangeInfo.

ENTRYPLAN (CONTRACT)
Fields:
- kind: MAKER_FIRST | TAKER | MARKET
- desired_price: string (decimal, target price before quantization)
- limit_price: string (decimal, quantized price for LIMIT orders)
- qty: string (decimal, quantized quantity)
- time_in_force: GTC | IOC | FOK
- ttl_ms: int (milliseconds, 0 means no TTL)
- reprice_ms: int (milliseconds between repricing attempts)
- max_reprices: int (maximum repricing attempts, 0 means no repricing)
- fallback: FallbackPlan
- client_order_id: string (exactly 36 characters)

Invariants:
- qty and prices must be quantized per constraints.
- client_order_id must be deterministic and unique until terminal state.
- MAKER_FIRST must have TTL > 0 and max_reprices >= 1.
- fallback can only reduce risk (never increase qty or aggressiveness beyond what is allowed).
- For LIMIT_MAKER orders: time_in_force must be GTC.

FALLBACKPLAN (CONTRACT)
Fields:
- enabled: bool
- kind: CANCEL_AND_REPLACE | MARKET_IF_ALLOWED | IOC_LIMIT
- max_slippage_bps: int (maximum allowed slippage in basis points)
- deadline_ms: int (milliseconds from entry attempt start)

Invariants:
- fallback cannot exist without explicit limits.
- MARKET_IF_ALLOWED is allowed only if policy allows and with max_slippage_bps > 0.
- any triggered fallback must be audited with a reason_code.
- deadline_ms must be > 0 if enabled=true.

EXITPLAN (CONTRACT)
Fields:
- tp_price: string (decimal, take profit price)
- sl_price: string (decimal, stop loss price)
- protection_kind: OCO | TP_SL_SEPARATE
- trailing_mode: OFF | VIRTUAL | NATIVE
- trailing_trigger_price: string (decimal, price at which trailing activates)
- trailing_delta_bips: int (distance in BIPS, 1 BIPS = 0.01%)
- client_order_id_tp: string (exactly 36 characters)
- client_order_id_sl: string (exactly 36 characters)

Definitions:
- trailing_delta_bips follows the Spot trailingDelta standard (BIPS). When trailing_mode=NATIVE, trailing_delta_bips is sent as trailingDelta and must pass the symbol's TRAILING_DELTA filter.
- stopPrice is optional for trailing NATIVE; if provided, the order starts trailing only after the stopPrice condition is satisfied.
- stopPrice rules vs market: stopPrice above market for STOP_LOSS BUY and TAKE_PROFIT SELL; stopPrice below market for STOP_LOSS SELL and TAKE_PROFIT BUY.
- trailing NATIVE is allowed only with Spot types: STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT, TAKE_PROFIT_LIMIT.

Invariants:
- for BUY: sl_price < entry_price and tp_price > entry_price.
- for SELL: sl_price > entry_price and tp_price < entry_price.
- trailing_trigger_price must be coherent with tp/sl (for BUY: trigger >= entry_price).
- protection must not be left uncovered during swaps (see 05_EXECUTION_AND_FAILSAFE.md).
- protection order IDs must not collide.
- If trailing_mode=NATIVE, trailing_delta_bips must be integer >= 1.
- If trailing_mode=OFF, trailing_trigger_price and trailing_delta_bips must be empty/zero.

AIGATERESULT (CONTRACT)
AIGateResult is the structured gate return. It never replaces Risk; it only restricts.

Required fields:
- enabled: bool
- verdict: ALLOW | BLOCK | MODIFY | ERROR
- reasons: []ReasonCode
- model: string (e.g., "gpt-4", "claude-3-opus")
- latency_ms: int (milliseconds for API call)
- raw_hash: string (SHA-256 hex of raw model response)
- input_hash: string (SHA-256 hex of canonical input payload)
- snapshot_hash: string (SHA-256 hex of canonical snapshot)
- modified_decision: Decision (only if verdict=MODIFY and valid)

Hash definitions:
- input_hash: SHA-256 hex of the canonical payload sent to the AI Gate (RFC 8785).
- snapshot_hash: SHA-256 hex of the canonical snapshot used as the payload base (RFC 8785). It may be derived from snapshot_id, but must be recorded as an explicit hash in the event.
- raw_hash: SHA-256 hex of the raw model response body (before normalization), stored/derived in redacted form per 07_SECURITY.md.

General invariants:
- reasons are never empty when enabled=true.
- enabled=false implies verdict=ALLOW and absence of modified_decision.
- enabled=true implies input_hash and snapshot_hash are non-empty.
- verdict=ALLOW implies modified_decision is absent.
- verdict=BLOCK implies modified_decision is absent.
- verdict=MODIFY implies modified_decision is present and valid.
- verdict=ERROR implies error occurred and modified_decision is absent.
- modified_decision is always revalidated locally and its decision_id is recomputed by the system after application (the AI has no authority to define decision_id).
- the AI Gate cannot define risk_verdict nor modify risk_verdict.
- the AI Gate cannot create new reason codes outside the catalog. Unknown reason codes make the result invalid.

FORMAL MODIFY RULES (CONSERVATIVE)
MODIFY can only reduce risk and/or aggressiveness. Any violation invalidates the result and must lead to local rejection of MODIFY.

Fields that MUST NOT change in modified_decision (immutable):
- mode
- symbol
- side
- intent
- snapshot_id
- cycle_id
- stage
- constraints (values and policy)
- edge_score_x10000
- edge_bps_expected
- original reasons may be extended, but never removed
- entry/exit plans cannot be removed when required by intent

MODIFY may:
- reduce entry_plan.qty (strictly smaller decimal string, after quantization)
- change kind to a less aggressive option:
  - MARKET -> TAKER -> MAKER_FIRST (only in this direction)
- reduce maker-first aggressiveness:
  - reduce max_reprices
  - reduce ttl_ms
  - increase reprice_ms (less churn)
- tighten protection:
  - for BUY: raise sl_price (closer to entry) without crossing entry
  - for BUY: reduce tp_price (more conservative) without crossing entry
  - trailing: reduce trailing_delta_bips (tighter) and/or move trigger earlier (more conservative), preserving coherence with tp/sl
- reduce or disable fallback:
  - disable fallback
  - reduce deadline_ms
  - reduce max_slippage_bps
  - change kind to a less aggressive option (MARKET_IF_ALLOWED -> IOC_LIMIT -> CANCEL_AND_REPLACE, only in this direction)
- suggest BLOCK (via verdict=BLOCK) instead of modifying

MODIFY must NOT (always forbidden):
- increase qty
- loosen stop (for BUY: reduce sl_price)
- increase tp in a way that raises execution/management risk (for BUY: raising tp_price is forbidden)
- remove stop/protection
- increase max_reprices, increase ttl_ms, reduce reprice_ms (more churn)
- enable fallback where it was previously disabled
- change fallback kind to a more aggressive option
- change client_order_id (any client_order_id is the system's responsibility and must be deterministic)
- change protection_kind to reduce coverage or create windows without protection
- introduce fields not recognized by the schema
- change risk_verdict or ai_gate fields

Numeric validations:
- all monetary/qty values in modified_decision must remain decimal strings (no float).
- modified_decision must remain quantized per DecisionConstraints.
- any missing numeric field where required makes MODIFY invalid.

Rejection policy:
- if any rule above is violated, MODIFY is rejected locally with reason_code AIGATE_MODIFY_INVALID.
- the system must treat rejected MODIFY as BLOCK in the most conservative manner applicable to the configured AI_DEC and fail policy.

RISKVERDICT (CONTRACT)
Fields:
- verdict: ALLOW | BLOCK
- reasons: []ReasonCode
- risk_limits: RiskLimitsSnapshot (captured limits used by Risk for this verdict)
- costs: CostsSnapshot (fees, slippage, spread)
RISKLIMITSSNAPSHOT (CONTRACT)
Purpose:
- Record the effective limits and counters applied by Risk for this decision.
- Enable post-mortem reconstruction of why a decision was allowed or blocked.

Fields (minimum; closed list):
- max_exposure_symbol_usdt: string decimal (maximum allowed notional exposure for this symbol)
- max_exposure_total_usdt: string decimal (maximum allowed total notional exposure across all symbols)
- max_daily_loss_usdt: string decimal (maximum daily loss before pausing/exiting)
- max_drawdown_usdt: string decimal (maximum drawdown before pausing/exiting)
- max_open_orders: int (maximum open orders allowed in the account/session)
- max_trades_day: int (maximum trades per day)
- max_trades_window: int (maximum trades per window)
- trades_window_seconds: int (window length for max_trades_window)
- post_stop_cooldown_seconds: int (cooldown after a stop-loss fill)

Serialization rules:
- All monetary values are decimal strings (no float, no scientific notation).
- Integers are base-10, no separators.
- This snapshot must match the config values and derived counters used by Risk for the current cycle.


Invariants:
- Risk has the final word (overrides AI Gate).
- reasons are never empty in BLOCK.
- verdict=ALLOW means the decision can proceed to execution.
- verdict=BLOCK means the decision is rejected and no execution occurs.

CANONICALIZATION, NORMALIZATION, AND HASHES
General rules:
- Canonical JSON per RFC 8785 (lexicographic key ordering).
- SHA-256 hex hashes (lowercase).
- decimal strings: no exponent, '.' separator, no trailing zeros beyond precision.
- quantization always before serialization (tick/step/minNotional).
- price/qty values that enter a hash never use float.

DecisionHashPayload:
- Canonical payload for decision_id and auditability.
- May include operational fields (cycle_id, stage, ts_ms) for auditing and correlation.
- Must include all fields that affect the decision outcome.
- Example structure:
  {
    "cycle_id": "cyc_abc123",
    "stage": "STRATEGY_PROPOSE",
    "ts_ms": 1706700000000,
    "symbol": "BTCUSDT",
    "side": "BUY",
    "intent": "ENTRY",
    "edge_score_x10000": 4500,
    "edge_bps_expected": 25,
    "snapshot_hash": "a1b2c3...",
    "constraints": {...},
    "entry_plan": {...},
    "exit_plan": {...}
  }

OrderIntentHashPayload:
- Canonical payload for order_intent_id and clientOrderId.
- MUST NOT include: run_id, cycle_id, stage, ts_ms, any timestamp, any value derived from local clock, or any volatile network field.
- Must include only: mode, symbol, side, intent, quantized plans (entry/exit), relevant constraints, and snapshot_hash (or deterministic snapshot_id + equivalent proof).
- Example structure:
  {
    "mode": "LIVE",
    "symbol": "BTCUSDT",
    "side": "BUY",
    "intent": "ENTRY",
    "entry_plan_qty": "0.01000000",
    "entry_plan_price": "41250.75",
    "exit_plan_tp": "41800.00",
    "exit_plan_sl": "40800.00",
    "snapshot_hash": "a1b2c3...",
    "constraints_hash": "d4e5f6..."
  }

Float prohibition in hashes:
- Any field participating in DecisionHashPayload or OrderIntentHashPayload must not be float.
- Scores must be scaled integers (e.g., *_x10000).
- Percentages must be in basis points (integer).
- All numeric values must be integers or decimal strings.

DETERMINISTIC IDS (CONTRACT)
IDs:
- run_id: string (generated at boot, e.g., "run_20260131_100000_abc123")
- cycle_id: string (deterministic per cycle, e.g., "cyc_20260131_100500_def456")
- snapshot_id: string (deterministic per snapshot, e.g., "snap_btcusdt_1706700000_ghi789")
- decision_id: string (deterministic per canonical Decision, e.g., "dec_btcusdt_1706700000_jkl012")
- order_intent_id: string (deterministic per canonical intent, e.g., "oi_btcusdt_entry_1706700000_mno345")

Rules:
- canonical JSON per RFC 8785.
- SHA-256 hex of the canonical payload.
- fixed prefixes for readability.
- truncation allowed in logs (first 12 chars), but full ID stored in DB.
- IDs must be URL-safe (A-Za-z0-9_-).

clientOrderId:
- len 36 characters exactly
- derived from order_intent_id (base32 upper case, no padding)
- format: "X_" + base32(sha256(order_intent_id))[0:34]
- never reuse until a confirmed terminal state (FILLED, CANCELLED, EXPIRED, REJECTED)
- must respect Binance limits: A-Za-z0-9_- only, no special characters

INTENTS LEDGER (EXACTLY-ONCE)
Goal:
- Prevent duplicate orders under restart/timeouts and guarantee reconcile/auditing.

Minimum states:
- CREATED: intent recorded locally, not yet sent.
- SENT_UNKNOWN: submit was attempted, but the result was not confirmed locally (timeout/connection break).
- CONFIRMED: Binance confirmed the order (ACK/FULL) and/or it was found via REST lookup.
- NOT_FOUND: REST lookup did not find the order after defined windows and attempts; terminal for retry.
- FAILED_TERMINAL: unrecoverable failure (e.g., rejection by a non-recoverable filter) with reason_code.

Critical rule:
- In SENT_UNKNOWN, blind resubmission is forbidden.
- Before any retry, the executor must query REST by clientOrderId and only then transition to CONFIRMED or NOT_FOUND.
- Query REST up to 3 times with 5-second timeout each.
- If still unknown after queries, keep SENT_UNKNOWN and enter DEGRADE.

State transition rules:
- CREATED -> SENT_UNKNOWN: when order submission attempted
- SENT_UNKNOWN -> CONFIRMED: when REST confirms order exists
- SENT_UNKNOWN -> NOT_FOUND: when REST confirms order does not exist
- Any state -> FAILED_TERMINAL: when rejection or permanent failure
- CONFIRMED, NOT_FOUND, FAILED_TERMINAL: terminal states (no further transitions)

STAGES CATALOG (CONTRACT)
Stages are enumerated and fixed (see 08_SYSTEM_ARCHITECTURE.md).
Any new stage requires:
- update enum/const in internal/observability/stage.go
- update STAGE audit trail logic
- record the decision in 12_DECISIONS_LOG.md

Complete StageName catalog:
BOOT, DOCTOR_CHECKS, STARTUP_RECOVER, UNIVERSE_SCAN, RANK_TOPN, DEEP_SCAN, WATCHLIST_ATTACH, STATE_UPDATE, STRATEGY_PROPOSE, AIGATE_CALL, RISK_VERDICT, EXECUTE_INTENT, POSITION_MANAGE, RECONCILE_REST, REPORT_DAILY_SUMMARY, DEGRADE, PAUSE, SHUTDOWN

SINGLE REASON_CODES CATALOG (SOURCE OF TRUTH)
Rules:
- Unknown ReasonCode makes Decision/RiskVerdict/AIGateResult invalid.
- New codes require updating here, validations, tests, and an entry in 12_DECISIONS_LOG.md.
- All codes must be defined in internal/domain/reasoncodes/codes.go.

Complete ReasonCode catalog:

OK (0) - No error, operation successful

OPS (Operational):
- ALERT_RAISED (1000)
- CLOSE_SAFELY_DUST (1001)
- CLOSE_SAFELY_FAILED (1002)
- DB_WRITER_BACKPRESSURE (1003)
- DB_WRITER_QUEUE_HIGH (1004)
- DB_WRITER_QUEUE_FULL (1005)
- DISK_LOW_DEGRADE (1006)
- DISK_LOW_PAUSE (1007)
- DRIFT_LIMIT_EXCEEDED (1008)
- ENTER_DEGRADE (1009)
- ENTER_EXIT (1010)
- ENTER_PAUSE (1011)
- LOOP_STUCK_DEGRADE (1012)
- LOOP_STUCK_PAUSE (1013)
- PAUSE_NEEDS_MANUAL_PROTECTION (1014)
- PAUSE_NEEDS_MANUAL (1019)
- REST_STALE_DEGRADE (1015)
- REST_STALE_PAUSE (1016)
- WS_STALE_DEGRADE (1017)
- WS_STALE_PAUSE (1018)

AIGATE (AI Gate):
- AIGATE_MODIFY_INVALID (2000)
- AIGATE_PARSE_FAIL (2001)
- AIGATE_REASON_UNKNOWN (2002)
- AIGATE_SCHEMA_INVALID (2003)
- AIGATE_TIMEOUT (2004)

BINANCE_LIMITS (Exchange):
- BINANCE_TIMESTAMP_REJECTED (3000)
- CLIENT_ORDER_ID_INVALID (3001)
- CLIENT_ORDER_ID_REUSE_BLOCK (3002)
- FILTERS_DRIFT_DETECTED (3003)
- FILTERS_REFRESHED (3004)
- INTENT_CONFIRMED_BY_REST (3005)
- INTENT_NOT_FOUND_BY_REST (3006)
- INTENT_SENT_UNKNOWN (3007)
- ORDER_CANCEL_REJECTED (3008)
- ORDER_SUBMIT_REJECTED (3009)
- ORDER_SUBMIT_TIMEOUT (3010)
- PROTECTION_INSTALL_FAILED (3011)
- PROTECTION_INVALID_FILTER (3012)
- PROTECTION_INVALID_MIN_NOTIONAL (3013)
- RATE_LIMIT_418 (3014)
- RATE_LIMIT_429 (3015)
- RECONCILE_DIFF_DETECTED (3016)
- RETRY_AFTER_APPLIED (3017)

STATE (Market State):
- CLOCK_DRIFT_WARN (4000)
- IMBALANCE_AGAINST (4001)
- MICROVOL_SPIKE (4002)
- REGIME_MISMATCH (4003)
- REGIME_NO_TRADE (4004)
- REGIME_WEAK (4005)
- SPREAD_BAD_VS_NORMAL (4006)
- SPREAD_OPENING (4007)
- SYMBOL_QUARANTINED (4008)
- TIME_SYNC_FAIL (4009)
- WS_OOO_EVENT (4010)

RISK (Risk Engine):
- RISK_CANCEL_REPLACE_LIMIT_HIT (5000)
- RISK_CHURN_LIMIT_HIT (5001)
- RISK_COOLDOWN_ACTIVE (5002)
- RISK_CORRELATION_TOO_HIGH (5003)
- RISK_DAILY_LOSS_LIMIT (5004)
- RISK_DIVERSIFY_APPLIED (5005)
- RISK_DRAWDOWN_LIMIT (5006)
- RISK_ENTRY_ALREADY_PENDING (5007)
- RISK_EXPOSURE_LIMIT (5008)
- RISK_INSUFFICIENT_FREE_BALANCE (5009)
- RISK_MAX_OPEN_ORDERS (5010)
- RISK_MAX_TRADES_DAY (5011)
- RISK_MAX_TRADES_WINDOW (5012)
- RISK_POSITION_ALREADY_OPEN (5013)
- RISK_SIZE_INVALID (5014)
- RISK_SYMBOL_LOSS_STREAK (5015)
- RISK_UNFILLED_ORDER_COUNT_RISK (5016)

STRATEGY (Strategy Engine):
- STRAT_CONFIG_INVALID (6000)
- STRAT_EDGE_BELOW_MIN (6001)
- STRAT_EDGE_OK (6002)
- STRAT_ENTRY_ABORTED_COST (6003)
- STRAT_ENTRY_MAKER_TTL (6004)
- STRAT_ENTRY_TIMEOUT (6005)
- STRAT_EXIT_ATR_TP_SL (6006)
- STRAT_EXIT_INVALID (6007)
- STRAT_FALLBACK_ALLOWED (6008)
- STRAT_FALLBACK_BLOCKED (6009)
- STRAT_FALLBACK_IOC (6010)
- STRAT_FALLBACK_MARKET (6011)
- STRAT_IMBALANCE_AGAINST (6012)
- STRAT_IMBALANCE_OK (6013)
- STRAT_INPUT_INVALID (6014)
- STRAT_MAKER_REJECTED_CROSSES_BOOK (6015)
- STRAT_MAKER_REPRICE (6016)
- STRAT_MICROSTRUCT_OK (6017)
- STRAT_MISSING_FIELD (6018)
- STRAT_OK (6019)
- STRAT_PULLBACK_FAIL (6020)
- STRAT_PULLBACK_OK (6021)
- STRAT_REGIME_RANGE_OK (6022)
- STRAT_REGIME_TREND_OK (6023)
- STRAT_REGIME_WEAK (6024)
- STRAT_SPREAD_OPENING (6025)
- STRAT_SPREAD_TOO_WIDE (6026)
- STRAT_TRAILING_ARM_ALLOWED (6027)
- STRAT_TRAILING_ARM_BLOCKED (6028)
- STRAT_VOLUME_LOW (6029)
- STRAT_VOLUME_OK (6030)

MINIMUM VALIDATIONS (TESTS)
Required validation tests:
- deterministic decision: same Snapshot + same Config -> same decision_id
- intent golden test: same Snapshot + same Config -> same order_intent_id and same clientOrderId
- deterministic client_order_id: same intent -> same id
- quantization: tick/step and minNotional applied correctly
- sl/tp/trailing invariants: for BUY, sl < entry < tp
- conservative AI modify: never increases risk, never increases qty, never removes stop
- minimum AI result: input_hash and snapshot_hash present when enabled=true
- AI raw_hash always present when enabled=true
- reason code validation: all codes in catalog, no unknown codes
- hash determinism: same input produces same hash across runs

Implementation validation:
- Decision validation on creation (all required fields, valid values)
- Decision validation before persistence (invariants hold)
- Decision validation before execution (quantization applied)
- AI Gate validation before application (MODIFY rules)
- Risk validation before final verdict (limits checked)

CHANGES TO THIS CONTRACT
When changing this contract:
1. Always update validations and unit tests
2. Update schema files (prompts/schemas/*.schema.json)
3. Update Go structs (internal/domain/contracts/*.go)
4. Update migrations if DB schema changes
5. Always record in 12_DECISIONS_LOG.md with:
   - Date of change
   - What changed and why
   - Impact on existing data
   - Migration strategy

Compatibility rules:
- New fields may be added (must be optional with defaults)
- Existing fields may not be removed (deprecate instead)
- Enum values may be added but not removed
- Type changes require migration plan
- Hash algorithms must remain SHA-256 (cannot change)

This contract version: 1.0.0 (2026-01-31)