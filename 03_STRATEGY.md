03_STRATEGY (DETERMINISTIC STRATEGY)

HIERARCHY (ORDER OF AUTHORITY)
00_SOURCE_OF_TRUTH.md
01_DECISION_CONTRACT.md
02_DATA_SNAPSHOT_SPEC.md
03_STRATEGY.md
04_RISK_ENGINE_RULES.md
05_EXECUTION_AND_FAILSAFE.md
06_AUDIT_RULES.md
07_SECURITY.md
08_SYSTEM_ARCHITECTURE.md
09_CODE_STRUCTURE.md
10_OPERATIONS_RULES.md
11_INVARIANTS_MAP.md
12_DECISIONS_LOG.md
README.md

OBJECTIVE
To deterministically and reproducibly define how the Strategy produces:
edge_score_x10000
edge_bps_expected
EntryPlan (maker-first, TTL, reprice, fallback)
ExitPlan (TP, SL, trailing triggers)

OUT OF SCOPE
Strategy does not execute orders.
Strategy does not ignore or relax limits; Risk can always block and can tighten limits.
Strategy assumes MODE=LIVE and never mutates execution mode. Strategy does not use settings outside of internal\config\config.go.

DEFINITIONS AND UNITS
bps = basis points (1 bps = 0.01%)
Values in percent may only exist as derived for display/telemetry; they do not enter hashes.
When necessary for deterministic calculation, the preferred representation is integer bps.
Conversion: pct_to_bps(pct) = round(pct * 100)

DETERMINISTIC ROUNDING SPECIFICATION

Scope:
- Floating point is allowed only for in-memory intermediate calculations that do NOT enter hashes.
- Any persisted numeric score must be an integer (bps or x10000).
- Decimal prices/qty in any hashed object MUST be decimal strings and MUST NOT use float.

Function Definitions (Go standard library):
- clamp01(x float64) float64:
  - clamp to [0, 1] using math.Max(0.0, math.Min(1.0, x))
- max(a float64, b float64) float64:
  - returns the greater of a and b (math.Max(a, b))
- round_to_even(x float64) int:
  - ties-to-even rounding using math.RoundToEven(x)
- round_x10000(x float64) int:
  - int(math.RoundToEven(clamp01(x) * 10000.0))
- round_bps(x float64) int:
  - int(math.RoundToEven(x))

Implementation Guarantee:
- float64 is IEEE-754 and deterministic on amd64 for the same Go toolchain.
- Round-to-even tie breaking is explicit via math.RoundToEven.
- Rounding occurs only at the final step before conversion to int.

DECIMAL STRINGS AND QUANTIZATION (HASHED FIELDS)

Contract rule (01_DECISION_CONTRACT.md):
- price/qty that enter hashes must be decimal strings ('.' separator, no exponent).
- quantization always occurs before serialization and hashing.

Quantization policy:
- Unquantized math uses exact rational arithmetic (no fixed scale).
- Quantization sets the final scale to the precision implied by:
  - tick_size (prices)
  - step_size (qty)
- Quantization rounding mode: ROUND_DOWN (floor), as required by 05_EXECUTION_AND_FAILSAFE.md.

Canonical module (single source of truth):
- internal\engine\executor\quantize.go (the QUANTIZER)

Rule:
- Strategy MUST call the QUANTIZER before computing decision_id and before persisting any Decision that contains prices/qty.

LONG-ONLY (SPOT)
This strategy is long-only:
ENTRY is always BUY.
EXIT is always SELL (TP/SL/trailing).
SELL to open a position does not exist.

ENTRY REQUIREMENTS (SNAPSHOT) AND MANDATORY FIELDS
All fields below must be present in the cycle's Snapshot; if any is missing, Strategy must BLOCK with reason_code STRAT_MISSING_FIELD.

Minimum fields (per candidate symbol, for Strategy/Entry/Manage):
A) regime
regime_label (TREND or RANGE)
trend_score_x10000 (int in [0..10000])
range_score_x10000 (int in [0..10000])

B) microstructure_60s
spread_bps_p50_60s (int)
spread_bps_p90_60s (int)
spread_current_bps (int)
delta_spread_bps_p90_10s (int)
bid_ask_imbalance_p50_10s_x10000 (int in [0..10000])

C) volatility
atr14_5m_bps (int)

D) market price inputs (for EntryPlan and costs)
best_bid (string decimal)
best_ask (string decimal)
mid_price (string decimal)
last_price (string decimal)

E) candles_5m (to derive volume_ratio_5m and EMA20_5m)
The last 40 5m candles containing at minimum:
open, high, low, close, volume
Ascending temporal order with coherent timestamps.

F) cost_inputs (for edge_bps_expected)
maker_fee_bps (int)
taker_fee_bps (int)
slippage_est_entry_maker_bps (int)
slippage_est_entry_taker_bps (int)
slippage_est_exit_taker_bps (int)

CONTRACT NOTE
If any of the above fields do not yet exist in the Snapshot as per 02_DATA_SNAPSHOT_SPEC.md and the schema, they must be added to the Snapshot and audited.
Reason codes for absence/invalidity: see the unified catalog in 01_DECISION_CONTRACT.md (STRAT_MISSING_FIELD, STRAT_INPUT_INVALID).

PARAMETERS (ONLY internal\config\config.go)
strategy.id (string) e.g., "REGIME_PULLBACK_MAKER_V1"
strategy.version (string) e.g., "1.0.0"

strategy.trend_threshold_x10000 (int) default 7000
strategy.range_threshold_x10000 (int) default 6000

strategy.min_edge_bps (int) default 15
strategy.min_edge_bps_fallback (int) default 20

strategy.max_spread_entry_bps (int) default 20
strategy.max_delta_spread_bps_10s (int) default 5
strategy.min_imbalance_buy_x10000 (int) default 5500

strategy.pullback_min_bps_from_ema20_5m (int) default 10
strategy.pullback_max_bps_from_ema20_5m (int) default 80

strategy.volume_ratio_window_5m (int) default 12
strategy.min_volume_ratio_5m (float64) default 1.20

strategy.weights.w_trend (float64) default 0.55
strategy.weights.w_pullback (float64) default 0.20
strategy.weights.w_microstruct (float64) default 0.15
strategy.weights.w_volume (float64) default 0.10
Rule: the sum must be 1.0; if not, BLOCK with STRAT_CONFIG_INVALID.

strategy.entry.maker_ttl_seconds (int) default 30
strategy.entry.maker_reprice_max (int) default 2

strategy.entry.fallback_max_spread_bps (int) default 25
strategy.entry.max_slippage_bps (int) default 25
strategy.entry.fallback_kind (string enum) values: "IOC_LIMIT" or "MARKET_IF_ALLOWED"
default "IOC_LIMIT"

strategy.exit.k_atr_trend (float64) default 1.50
strategy.exit.m_atr_trend (float64) default 2.00
strategy.exit.k_atr_range (float64) default 1.00
strategy.exit.m_atr_range (float64) default 1.50

strategy.exit.trailing_enable_profit_bps (int) default 50
strategy.exit.trailing_trend_min_x10000 (int) default 6000
strategy.exit.trailing_t_atr_trend (float64) default 1.00
strategy.exit.trailing_t_atr_range (float64) default 0.80
strategy.exit.trailing_max_spread_bps (int) default 25

CONFIGURATION VALIDATION
Strategy must validate ALL parameters on initialization:

1. Weights sum to 1.0: abs((w_trend + w_pullback + w_micro + w_vol) - 1.0) < 0.000001
2. All weights >= 0
3. strategy.trend_threshold_x10000 in [0..10000]
4. strategy.range_threshold_x10000 in [0..10000]
5. strategy.pullback_max_bps_from_ema20_5m > strategy.pullback_min_bps_from_ema20_5m
6. strategy.min_edge_bps > 0
7. strategy.min_edge_bps_fallback >= strategy.min_edge_bps
8. strategy.max_spread_entry_bps > 0
9. strategy.volume_ratio_window_5m >= 2

If any validation fails: BLOCK with STRAT_CONFIG_INVALID and enter PAUSE.

DETERMINISTIC INDICATORS AND CALCULATIONS

1) EMA20_5m
Input: closes of the last 20 5m candles.
alpha = 2.0 / (20.0 + 1.0)  // explicitly float: 2/21 = 0.09523809523809523
seed: SMA of the first 20 closes used (SMA20).
ema20_5m = (close_last * alpha) + (ema_prev * (1.0 - alpha))

Implementation note: Use decimal arithmetic or fixed-point for deterministic results.

2) volume_ratio_5m
window = strategy.volume_ratio_window_5m (default 12)
Input: volumes of the last "window" 5m candles.
base = simple average of volumes for candles [N-window .. N-2] (excludes the most recent candle).
last = volume of the most recent candle (N-1).
If base <= 0, BLOCK with STRAT_INPUT_INVALID.
volume_ratio_5m = last / base.

3) spread_current_bps
Input: spread_current_bps (int) from Snapshot (calculated by state engine).
Rule: if spread_current_bps <= 0, BLOCK with STRAT_INPUT_INVALID.

4) atr14_5m_bps
Input: atr14_5m_bps (int) from Snapshot.
Rule: if atr14_5m_bps <= 0, BLOCK with STRAT_INPUT_INVALID.

ENTRY SIGNALS (BUY)

Eligibility pre-filter (deterministic local Strategy blocks)
A) spread_current_bps <= strategy.max_spread_entry_bps
B) delta_spread_bps_p90_10s <= strategy.max_delta_spread_bps_10s
C) bid_ask_imbalance_p50_10s_x10000 >= strategy.min_imbalance_buy_x10000
D) volume_ratio_5m >= strategy.min_volume_ratio_5m

REGIME PROCESSING - SINGLE ACTIVE REGIME

Rule: Exactly one regime is active per symbol. Selection algorithm:

Active regime selection:
if regime_label == "TREND" AND trend_score_x10000 >= strategy.trend_threshold_x10000:
    active_regime = "TREND"
    regime_component = trend_component_calculated()
elif regime_label == "RANGE" AND range_score_x10000 >= strategy.range_threshold_x10000:
    active_regime = "RANGE"
    regime_component = range_component_calculated() * 0.7  // RANGE penalty (30% reduction)
else:
    active_regime = "NONE"
    regime_component = 0.0
    BLOCK with STRAT_REGIME_WEAK

RANGE REGIME - CONSERVATIVE RULES
If active_regime == "RANGE":
    Additional requirements:
    1. spread_current_bps <= (strategy.max_spread_entry_bps / 2)
    2. edge_bps_expected >= (strategy.min_edge_bps * 1.5)  // verified later
    3. volume_ratio_5m >= (strategy.min_volume_ratio_5m * 1.1)
    
    If any fails: BLOCK with STRAT_REGIME_WEAK

Pullback (zone of interest) based on EMA20_5m
ema20_5m must exist.
pullback_bps = ((ema20_5m - last_price) / ema20_5m) * 10000
Rules:
pullback_bps must be between [strategy.pullback_min_bps_from_ema20_5m .. strategy.pullback_max_bps_from_ema20_5m]
If pullback_bps < min => price "too high" (no pullback), fail with STRAT_PULLBACK_FAIL.
If pullback_bps > max => pullback "too deep" (risk of breakdown), fail with STRAT_PULLBACK_FAIL.

Note: A positive pullback_bps means last_price is below EMA20_5m.
If ema20_5m <= 0, BLOCK with STRAT_INPUT_INVALID.

CALCULATION OF edge_score_x10000 (0..10000)

Rule:
- edge_score is calculated as a float64 in memory.
- edge_score_x10000 is the persisted representation (Decision/Audit) and the only one allowed in hashes.

Normalized components:
All components below are calculated as float64 in [0..1] only in memory.
When persisting to Decision/Audit, serialize as int_x10000 via round_x10000.

trend_component (only if active_regime == "TREND"):
trend_component = clamp01(float64(trend_score_x10000 - strategy.trend_threshold_x10000) / float64(10000 - strategy.trend_threshold_x10000))

range_component (only if active_regime == "RANGE"):
range_component = clamp01(float64(range_score_x10000 - strategy.range_threshold_x10000) / float64(10000 - strategy.range_threshold_x10000))

microstruct_component:
score_spread = clamp01(1.0 - (float64(spread_current_bps) / float64(strategy.max_spread_entry_bps)))
score_delta = clamp01(1.0 - (float64(delta_spread_bps_p90_10s) / float64(strategy.max_delta_spread_bps_10s)))
score_imb:
- den_imb = 10000 - strategy.min_imbalance_buy_x10000
- if den_imb <= 0: BLOCK with STRAT_CONFIG_INVALID (min_imbalance_buy_x10000 must be in [0..9999])
- else:
  score_imb = clamp01(float64(bid_ask_imbalance_p50_10s_x10000 - strategy.min_imbalance_buy_x10000) / float64(den_imb))
microstruct_component = (score_spread * 0.5) + (score_delta * 0.25) + (score_imb * 0.25)

volume_component:
volume_component = clamp01((volume_ratio_5m - strategy.min_volume_ratio_5m) /
(max(2.0, strategy.min_volume_ratio_5m + 0.8) - strategy.min_volume_ratio_5m))

pullback_component:
- pullback_component measures how "pulled down" the price is vs EMA20_5m, in bps.
- pre-conditions (config):
  - strategy.pullback_max_bps_from_ema20_5m > strategy.pullback_min_bps_from_ema20_5m
  - if pre-condition fails: BLOCK with STRAT_CONFIG_INVALID
- deterministic calculation:
  pullback_component = clamp01(
    (pullback_bps - strategy.pullback_min_bps_from_ema20_5m) /
    (strategy.pullback_max_bps_from_ema20_5m - strategy.pullback_min_bps_from_ema20_5m)
  )

Weights:
w_trend = strategy.weights.w_trend
w_pullback = strategy.weights.w_pullback
w_micro = strategy.weights.w_microstruct
w_vol = strategy.weights.w_volume

edge_score = clamp01((w_trend * regime_component) + (w_pullback * pullback_component) +
(w_micro * microstruct_component) + (w_vol * volume_component))
edge_score_x10000 = round_x10000(edge_score)

CALCULATION OF ExitPlan (TP/SL/Trailing) AND DISTANCES IN bps

Multiplier selection (by active regime):
If active_regime == "TREND":
k = strategy.exit.k_atr_trend
m = strategy.exit.m_atr_trend
t = strategy.exit.trailing_t_atr_trend
If active_regime == "RANGE":
k = strategy.exit.k_atr_range
m = strategy.exit.m_atr_range
t = strategy.exit.trailing_t_atr_range

sl_distance_bps = round_to_even(k * float64(atr14_5m_bps))
tp_distance_bps = round_to_even(m * float64(atr14_5m_bps))
trailing_distance_bps = round_to_even(t * float64(atr14_5m_bps))

Conversion rule:
- round_to_even is round() ties-to-even (deterministic) and the result is int in bps before persisting and before calculating edge_bps_expected.

Minimum local sanity check (Strategy):
sl_distance_bps > 0
tp_distance_bps > 0
tp_distance_bps > sl_distance_bps
If fails, BLOCK with STRAT_EXIT_INVALID.

COSTS (cost_breakdown) AND edge_bps_expected

Conservative base estimates:
fee_roundtrip_bps = maker_fee_bps + taker_fee_bps
slippage_roundtrip_bps_entry = slippage_est_entry_maker_bps
slippage_roundtrip_bps_exit = slippage_est_exit_taker_bps
slippage_roundtrip_bps = slippage_roundtrip_bps_entry + slippage_roundtrip_bps_exit

spread_roundtrip_bps = spread_current_bps
delta_spread_penalty_bps = max(0, delta_spread_bps_p90_10s)

cost_total_bps = fee_roundtrip_bps + slippage_roundtrip_bps + spread_roundtrip_bps + delta_spread_penalty_bps

edge_bps_expected = tp_distance_bps - sl_distance_bps - cost_total_bps

Note:
- edge_bps_expected is integer in bps.

EDGE BPS VALIDATION RULES
Validation rules (in order):
1. If tp_distance_bps <= sl_distance_bps: BLOCK with STRAT_EXIT_INVALID
2. If edge_bps_expected < 0: BLOCK with STRAT_EDGE_BELOW_MIN
3. If edge_bps_expected < strategy.min_edge_bps: BLOCK with STRAT_EDGE_BELOW_MIN
4. If active_regime == "RANGE" and edge_bps_expected < (strategy.min_edge_bps * 1.5): BLOCK with STRAT_EDGE_BELOW_MIN

Mandatory note: Risk can raise the effective min_edge and can block even with Strategy OK.

ENTRYPLAN (contract-shaped: EntryPlan + FallbackPlan)

Inputs:
best_bid, best_ask, mid_price, spread_current_bps
ttl_seconds = strategy.entry.maker_ttl_seconds
ttl_ms = ttl_seconds * 1000
reprice_max = strategy.entry.maker_reprice_max
reprice_ms = ttl_ms  // deterministic: reprice occurs when TTL expires

Base plan (attempt 1):
EntryPlan.kind = MAKER_FIRST
EntryPlan.time_in_force = GTC
EntryPlan.desired_price = best_bid

Quantization (mandatory before hashing):
EntryPlan.limit_price = QUANTIZER.quantize_price(desired_price, constraints.tick_size)
EntryPlan.qty = QUANTIZER.quantize_qty(desired_qty, constraints.step_size, constraints.min_qty, constraints.min_notional)
If quantization fails: BLOCK with STRAT_INPUT_INVALID.

Maker-first TTL and reprice:
- EntryPlan.ttl_ms = ttl_ms
- EntryPlan.reprice_ms = reprice_ms
- EntryPlan.max_reprices = reprice_max
- If no fill until TTL:
  - if signals still valid and edge_bps_expected still >= strategy.min_edge_bps:
    - attempt reprice (up to reprice_max)
  - else abort (STRAT_ENTRY_ABORTED_COST)

Reprice (attempts 2..N):
reprice uses the current book:
new desired_price = best_bid
maintains EntryPlan.kind = MAKER_FIRST and time_in_force = GTC and full TTL per attempt.
Each reprice increments intent_seq and emits STRAT_MAKER_REPRICE.

Fallback (after exhausting reprice_max or maker rejected for crossing the book)
Fallback is only permitted if ALL the following conditions are true:
1) spread_current_bps <= strategy.entry.fallback_max_spread_bps
2) slippage_est_entry_taker_bps <= strategy.entry.max_slippage_bps
3) edge_bps_expected >= strategy.min_edge_bps_fallback

If permitted:
EntryPlan.fallback.enabled = true
EntryPlan.fallback.kind = strategy.entry.fallback_kind  // IOC_LIMIT or MARKET_IF_ALLOWED
EntryPlan.fallback.max_slippage_bps = strategy.entry.max_slippage_bps
EntryPlan.fallback.deadline_ms = ttl_ms * (reprice_max + 1)  // from maker attempt start

Semantics (execution mapping, informational):
If fallback.kind == IOC_LIMIT:
- Execution submits an aggressive LIMIT with time_in_force=IOC at the current best_ask (for BUY), subject to max_slippage_bps.
If fallback.kind == MARKET_IF_ALLOWED:
- Execution submits a MARKET order only if allowed by global policy and subject to max_slippage_bps.

If not permitted:
abort entry and emit STRAT_FALLBACK_BLOCKED.

EXITPLAN (protection on entry and trailing)

Immediate protection on entry:
ExitPlan must always include TP and SL (OCO recommended in Live per 05_EXECUTION_AND_FAILSAFE.md).
TP:
tp_price_intent = entry_price * (1 + (tp_distance_bps / 10000)) using decimal arithmetic (no float)
SL:
sl_price_intent = entry_price * (1 - (sl_distance_bps / 10000)) using decimal arithmetic (no float)

Trailing (trigger and distance):
trailing_enable_profit_bps = strategy.exit.trailing_enable_profit_bps
trailing_trend_min_x10000 = strategy.exit.trailing_trend_min_x10000
trailing_max_spread_bps = strategy.exit.trailing_max_spread_bps

Trigger (conservative):
- trailing can only arm when active_regime == "TREND".
- if active_regime != "TREND": do not arm and emit STRAT_TRAILING_ARM_BLOCKED.
- if active_regime == "TREND": allow trailing arm when ALL conditions below are true:
  - profit_bps >= trailing_enable_profit_bps
  - trend_score_x10000 >= trailing_trend_min_x10000
  - spread_current_bps <= trailing_max_spread_bps
  - otherwise: STRAT_TRAILING_ARM_BLOCKED

TrailingDistance:
- trailing_distance_bps was already calculated above (round_to_even(t * atr14_5m_bps)) and is the distance in bps (integer).
- Strategy always populates ExitPlan.trailing_distance_bps (bps).
- When execution selects trailing_mode=NATIVE, map 1:1 to the exchange's field (BIPS):
  - trailing_delta_bips = trailing_distance_bps  // 1 bps = 1 bips = 0.01%
NOTE: The trailing mode (NATIVE/VIRTUAL/AUTO) is an execution policy and compatibility matter (05).
When execution selects NATIVE, it must use trailingDelta (Spot) and respect the TRAILING_DELTA filter; supported types on Spot: STOP_LOSS, STOP_LOSS_LIMIT, TAKE_PROFIT, TAKE_PROFIT_LIMIT.
Strategy only defines the trigger and expected distance.

NUMERIC VERIFICATION EXAMPLE

Test Case (TREND regime):
Inputs:
- trend_score_x10000 = 8000, trend_threshold = 7000
- spread_current_bps = 15, max_spread = 20
- delta_spread_bps = 3, max_delta = 5
- imbalance_x10000 = 6000, min_imbalance = 5500
- volume_ratio_5m = 1.5, min_ratio = 1.2
- pullback_bps = 30, min=10, max=80
- weights: w_trend=0.55, w_pullback=0.20, w_micro=0.15, w_vol=0.10

Calculations:
trend_component = (8000-7000)/(10000-7000) = 0.3333
score_spread = 1 - (15/20) = 0.25
score_delta = 1 - (3/5) = 0.4
score_imb = (6000-5500)/(10000-5500) = 0.1111
microstruct_component = (0.25*0.5)+(0.4*0.25)+(0.1111*0.25) = 0.125+0.1+0.0278 = 0.2528
volume_component = (1.5-1.2)/(2.0-1.2) = 0.375
pullback_component = (30-10)/(80-10) = 0.2857

edge_score = (0.55*0.3333)+(0.20*0.2857)+(0.15*0.2528)+(0.10*0.375) = 0.1833+0.0571+0.0379+0.0375 = 0.3158
edge_score_x10000 = round(0.3158 * 10000) = 3158

IMPLEMENTATION NOTES

EMA20 Calculation:
Use decimal arithmetic or fixed-point to avoid floating-point non-determinism.

Deterministic Guarantee:
Same snapshot + same config → same Decision every time.
Test via golden tests with known inputs/outputs.

Error Handling:
- Missing field → STRAT_MISSING_FIELD
- Invalid value → STRAT_INPUT_INVALID  
- Config error → STRAT_CONFIG_INVALID
- Regime weak → STRAT_REGIME_WEAK
- Edge below minimum → STRAT_EDGE_BELOW_MIN
- Exit invalid → STRAT_EXIT_INVALID
- Pullback fail → STRAT_PULLBACK_FAIL
- Fallback blocked → STRAT_FALLBACK_BLOCKED
- Trailing blocked → STRAT_TRAILING_ARM_BLOCKED

REASON CODES (STRATEGY)
Strategy must emit ReasonCodes from the unified catalog in 01_DECISION_CONTRACT.md.
Convention: Strategy uses prefix STRAT_* for local decisions and blocks (e.g., STRAT_OK, STRAT_EDGE_BELOW_MIN, STRAT_TRAILING_ARM_BLOCKED).

EXAMPLES (CONTRACT-SHAPED, REDACTED)

Example A: ENTRY (MAKER_FIRST) with IOC_LIMIT fallback and OCO exit
{
  "mode": "LIVE",
  "symbol": "BTCUSDT",
  "side": "BUY",
  "intent": "ENTRY",
  "edge_score_x10000": 3158,
  "edge_bps_expected": 22,
  "entry_plan": {
    "kind": "MAKER_FIRST",
    "desired_price": "41250.75",
    "limit_price": "41250.75",
    "qty": "0.01",
    "time_in_force": "GTC",
    "ttl_ms": 30000,
    "reprice_ms": 30000,
    "max_reprices": 2,
    "fallback": {
      "enabled": true,
      "kind": "IOC_LIMIT",
      "max_slippage_bps": 25,
      "deadline_ms": 90000
    },
    "client_order_id": "X_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  },
  "exit_plan": {
    "tp_price": "41900.01",
    "sl_price": "40800.22",
    "protection_kind": "OCO",
    "trailing_mode": "OFF",
    "trailing_trigger_price": "0",
    "trailing_delta_bips": 0,
    "client_order_id_tp": "X_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    "client_order_id_sl": "X_cccccccccccccccccccccccccccccccccc"
  },
  "reasons": ["STRAT_OK"]
}

Example B: ENTRY (MAKER_FIRST) with MARKET_IF_ALLOWED fallback
{
  "entry_plan": {
    "kind": "MAKER_FIRST",
    "time_in_force": "GTC",
    "ttl_ms": 30000,
    "max_reprices": 2,
    "fallback": {
      "enabled": true,
      "kind": "MARKET_IF_ALLOWED",
      "max_slippage_bps": 25,
      "deadline_ms": 90000
    }
  },
  "reasons": ["STRAT_FALLBACK_ALLOWED", "STRAT_FALLBACK_MARKET"]
}
