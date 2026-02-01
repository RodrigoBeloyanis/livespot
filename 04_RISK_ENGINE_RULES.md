04_RISK_ENGINE_RULES (DETERMINISTIC RULES)

GOAL
Define the final deterministic rules to allow/block operations and protect capital.
Risk always has the final word, even if AI allows.

HIERARCHY AND AUTHORITY
- Risk executes after AI Gate (if enabled) and before execution.
- Risk can BLOCK even if AI Gate returns ALLOW.
- Risk can tighten parameters beyond Strategy suggestions.
- Risk decisions are final and auditable.

MINIMUM EDGE AND COST
Risk validates cost_model and edge_bps_expected/edge_score.
Adaptive thresholds may increase min_edge under bad conditions.

Rules:
1. Base minimum edge: strategy.min_edge_bps (from config)
2. Validate edge_bps_expected >= strategy.min_edge_bps
3. If edge_bps_expected < strategy.min_edge_bps: BLOCK with STRAT_EDGE_BELOW_MIN
4. Validate cost components:
   - maker_fee_bps >= 0, taker_fee_bps >= 0
   - slippage estimates >= 0
   - spread_current_bps >= 0
5. If any cost component invalid: BLOCK with STRAT_INPUT_INVALID

Adaptive edge adjustment (deterministic, fixed-point):
- multiplier_x10000 is computed using the MAX model described in ADAPTIVE THRESHOLDS.
- adjusted_min_edge_bps = round_div(strategy.min_edge_bps * multiplier_x10000, 10000).
- If edge_bps_expected < adjusted_min_edge_bps: BLOCK with STRAT_EDGE_BELOW_MIN.

Files:
- internal\engine\risk\cost_model.go: validates costs and computes adaptive thresholds.
- internal\engine\risk\edge_validation.go: checks edge against minimums.
- internal\config\config.go: parameters for adaptive thresholds.

Reason codes:
- STRAT_EDGE_BELOW_MIN
- STRAT_INPUT_INVALID

POSITION SIZING
Goal: risk equal per trade, reduce oversized trades under high volatility.

Base rule:
- risk_per_trade_usdt = 100.00 (USDT, decimal with 2 dp; canonical default)
- risk_per_trade_min_usdt = 10.00 (USDT; hard floor)
- risk_per_trade_max_usdt = 500.00 (USDT; hard cap)
- Rounding: parse as decimal; if more than 2 dp, ROUND DOWN to 2 dp (conservative).
- stop_distance_bps = abs(entry_price - stop_price) / entry_price * 10000
- size_quote = (risk_per_trade_usdt * 10000) / stop_distance_bps
- size_base = size_quote / entry_price
- Apply quantization to size_base per LOT_SIZE filter
- Apply min_notional check after quantization

Limits applied (before sending to execution):
1. Per-symbol exposure: size_quote <= max_exposure_symbol
2. Total exposure: current_total_exposure + size_quote <= max_exposure_total
3. PRICE_FILTER: entry_price within [minPrice, maxPrice], rounded to tickSize
4. LOT_SIZE: size_base within [minQty, maxQty], rounded to stepSize
5. MIN_NOTIONAL: size_quote >= minNotional
6. MARKET_LOT_SIZE (if using MARKET): same as LOT_SIZE for market orders
7. If any required filter missing/invalid/unknown: block and consider symbol quarantine

Size calculation algorithm:
func calculatePositionSize(entry_price decimal, stop_price decimal, risk_usdt decimal) (size_base decimal, size_quote decimal, ok bool) {
    // 1. Calculate stop distance in bps
    stop_distance_bps = abs((entry_price - stop_price) / entry_price) * 10000
    
    // 2. Calculate raw size in quote
    size_quote_raw = (risk_usdt * 10000) / stop_distance_bps
    
    // 3. Convert to base
    size_base_raw = size_quote_raw / entry_price
    
    // 4. Quantize base to stepSize (round down)
    size_base = quantizeToStepSize(size_base_raw, stepSize)
    
    // 5. Recalculate quote value
    size_quote = size_base * entry_price
    
    // 6. Check min_notional
    if size_quote < minNotional {
        return 0, 0, false
    }
    
    // 7. Check max limits
    if size_base > maxQty || size_quote > max_exposure_symbol {
        return 0, 0, false
    }
    
    return size_base, size_quote, true
}

Rounding rule: Always round DOWN to nearest stepSize for conservative sizing.

Config contract:
- risk_per_trade_usdt MUST be present in internal\config\config.go and MUST match the canonical default unless explicitly overridden.
- The canonical default and bounds (risk_per_trade_min_usdt, risk_per_trade_max_usdt) MUST be mirrored in 00_SOURCE_OF_TRUTH.md (CONFIG DEFAULTS section).
- If config differs from this file and 00_SOURCE_OF_TRUTH.md, the system MUST fail-closed at boot (MODE=LIVE: PAUSE_NEEDS_MANUAL).

Files:
- internal\engine\risk\position_sizing.go: computes allowed size.
- internal\engine\executor\quantize.go: applies filters and guarantees validity.
- internal\domain\contracts\decision.go: persist risk_per_trade, stop_distance, and size.

Reason codes:
- RISK_SIZE_INVALID (quantization fails)
- RISK_EXPOSURE_LIMIT (exceeds symbol or total limits)
- PROTECTION_INVALID_FILTER (filter validation fails)
- PROTECTION_INVALID_MIN_NOTIONAL (below minimum after quantization)
- FILTERS_DRIFT_DETECTED (filters changed during sizing)
- SYMBOL_QUARANTINED (symbol in quarantine)
- RISK_INSUFFICIENT_FREE_BALANCE (insufficient funds)

PROTECTION ON PARTIAL FILLS + PROTECTION FAILURE (NO GAP)
Problem: after a partial fill, the remaining size may fall below filters (MIN_NOTIONAL, LOT_SIZE/STEP, MARKET_LOT_SIZE). An OCO/SL/TP attempt may fail, leaving the position unprotected.

Deterministic (conservative) rule:
1. When detecting a position without valid protection (or protection rejected):
   - Block new entries immediately for the symbol.
   - Start the "protection_recovery" flow for the symbol.
   - Emit ALERT_RAISED with reasons: [PAUSE_NEEDS_MANUAL_PROTECTION] (and/or PROTECTION_INSTALL_FAILED when applicable).

2. Evaluate current notional/qty (already quantized):
   - position_notional = position_qty * current_price
   - If position_notional < minNotional OR position_qty < minQty:
     - Attempt immediate CLOSE_SAFELY (the safest allowed exit order that is valid per filters).
     - If CLOSE_SAFELY is also not possible due to filters/market: PAUSE_NEEDS_MANUAL.
   - Else (protectable size):
     - Attempt to reinstall minimum valid protection (STOP_LOSS with minimum distance).
     - If protection installation fails: CLOSE_SAFELY.
     - If CLOSE_SAFELY fails: PAUSE_NEEDS_MANUAL.

Priorities:
- Never stay in a loop attempting to create invalid protection (max 3 attempts).
- If it is not possible to maintain valid protection, prefer exiting (CLOSE_SAFELY) over operating degraded with an exposed position.
- If exiting fails, require manual intervention (PAUSE_NEEDS_MANUAL).

CLOSE_SAFELY algorithm:
1. Try LIMIT order at current bid (for long position)
2. If limit fails or times out: try MARKET order
3. If market fails: mark as CLOSE_SAFELY_FAILED
4. If resulting size below minNotional: mark as CLOSE_SAFELY_DUST

Protection recovery flow:
- Timeout: 30 seconds maximum
- Retries: 2 attempts with 5-second backoff
- Fallback: CLOSE_SAFELY after retries exhausted
- Audit: every attempt and outcome

Files:
- internal\engine\risk\protection_gap.go: detects and handles protection gaps.
- internal\engine\position\protection_recovery.go: implements recovery flow.
- internal\engine\executor\close_safely.go: safe exit implementation.

Reason codes:
- PROTECTION_INSTALL_FAILED
- PROTECTION_INVALID_MIN_NOTIONAL
- PROTECTION_INVALID_FILTER
- CLOSE_SAFELY_DUST
- CLOSE_SAFELY_FAILED
- PAUSE_NEEDS_MANUAL_PROTECTION

ADAPTIVE THRESHOLDS
Goal: require higher edge when conditions worsen (bad spread, high ATR, low liquidity), while remaining fully deterministic.

Fixed-point model (MAX model; no float):
- multiplier_x10000 starts at 10000 (x10000 scale).
- Each condition may raise multiplier_x10000 by taking the maximum with a defined step.
- multiplier_x10000 is capped by risk.adaptive.max_multiplier_x10000.
- adjusted_min_edge_bps = round_div(base_min_edge_bps * multiplier_x10000, 10000).

Condition steps (deterministic constants; x10000):
- Spread step: 12000
- Volatility step: 13000
- Liquidity step: 14000
- Cap: 30000

Conditions (inputs come from Snapshot, see 02_DATA_SNAPSHOT_SPEC.md):
1) Spread condition
   - Trigger when spread_current_bps > round_div(spread_bps_p50_60s * risk.adaptive.spread_factor_x10000, 10000).
   - Default spread_factor_x10000 = 15000.
   - Action: multiplier_x10000 = max(multiplier_x10000, 12000).

2) Volatility condition
   - Let normal_atr_5m_bps be a config constant (default 50 bps).
   - Trigger when atr14_5m_bps > round_div(normal_atr_5m_bps * risk.adaptive.volatility_factor_x10000, 10000).
   - Default volatility_factor_x10000 = 15000.
   - Action: multiplier_x10000 = max(multiplier_x10000, 13000).

3) Liquidity condition (via imbalance; no depth fields required)
   - Trigger when bid_ask_imbalance_p50_10s_x10000 < risk.adaptive.liquidity_floor_x10000.
   - Default liquidity_floor_x10000 = 5000.
   - Action: multiplier_x10000 = max(multiplier_x10000, 14000).

Config parameters (config.go; all integers; explicit units):
- risk.adaptive.spread_factor_x10000: 15000
- risk.adaptive.volatility_factor_x10000: 15000
- risk.adaptive.liquidity_floor_x10000: 5000
- risk.adaptive.max_multiplier_x10000: 30000
- risk.adaptive.normal_atr_5m_bps: 50

Algorithm (Go-style pseudocode; integer only):
func computeAdaptiveMinEdgeBps(base_min_edge_bps int, snapshot Snapshot, cfg Config) (adjusted_min_edge_bps int, multiplier_x10000 int) {
    multiplier_x10000 = 10000

    // 1) Spread condition
    spread_threshold_bps := round_div(snapshot.spread_bps_p50_60s*cfg.Risk.Adaptive.SpreadFactorX10000, 10000)
    if snapshot.spread_current_bps > spread_threshold_bps {
        multiplier_x10000 = max(multiplier_x10000, 12000)
    }

    // 2) Volatility condition
    atr_threshold_bps := round_div(cfg.Risk.Adaptive.NormalATR5mBps*cfg.Risk.Adaptive.VolatilityFactorX10000, 10000)
    if snapshot.atr14_5m_bps > atr_threshold_bps {
        multiplier_x10000 = max(multiplier_x10000, 13000)
    }

    // 3) Liquidity condition (imbalance)
    if snapshot.bid_ask_imbalance_p50_10s_x10000 < cfg.Risk.Adaptive.LiquidityFloorX10000 {
        multiplier_x10000 = max(multiplier_x10000, 14000)
    }

    // Cap
    if multiplier_x10000 > cfg.Risk.Adaptive.MaxMultiplierX10000 {
        multiplier_x10000 = cfg.Risk.Adaptive.MaxMultiplierX10000
    }

    adjusted_min_edge_bps = round_div(base_min_edge_bps*multiplier_x10000, 10000)
    return adjusted_min_edge_bps, multiplier_x10000
}

Rounding rule (deterministic):
- round_div(a, b) for a>=0 and b>0 is (a + (b/2)) / b (integer division).

ANTI-OVERTRADING
Minimum rules:
1. Post-stop cooldown per symbol: after SL hit, block new entries for cooldown_seconds
2. Max trades per symbol per window: 3 per hour (default)
3. Block symbol after N consecutive losses: 3 consecutive losses (default)
4. No-trade when WS/latency degrades (per Fail Policy)

Implementation:
1. Cooldown tracking:
   - Store symbol -> cooldown_until_ms in memory and SQLite
   - On SL fill: set cooldown_until_ms = now_ms + cooldown_seconds * 1000
   - Check before entry: if now_ms < cooldown_until_ms: BLOCK

2. Trade counting:
   - Window: 1 hour (configurable)
   - Count fills (both entry and exit) for symbol
   - If count >= max_trades_per_hour: BLOCK until count resets

3. Loss streak:
   - Track consecutive losses per symbol
   - Loss defined as exit with realized PnL < 0
   - If consecutive_losses >= max_consecutive_losses: BLOCK symbol
   - Reset streak on profit or after cooldown period

4. Latency/WS degradation:
   - Monitor WS message age and REST response times
   - If degradation detected: block new entries per Fail Policy

Parameters (config.go):
- risk.anti_overtrading.cooldown_seconds: 300 (5 minutes)
- risk.anti_overtrading.max_trades_per_hour: 3
- risk.anti_overtrading.max_consecutive_losses: 3
- risk.anti_overtrading.ws_latency_threshold_ms: 1000

Files:
- internal\engine\risk\anti_overtrading.go: applies limits.
- internal\infra\sqlite\queries.go: reads history and counts.
- internal\engine\failsafe\policy.go: provides degrade/pause signal due to WS/latency.

Reason codes:
- RISK_COOLDOWN_ACTIVE
- RISK_MAX_TRADES_WINDOW
- RISK_SYMBOL_LOSS_STREAK
- WS_OOO_EVENT

CHURN, CANCEL/REPLACE, AND OPERATIONAL LIMITS
Goal: avoid unfilled order count penalties and bans due to excessive cancel/replace.
Ensure maker-first repricing does not become a count/order generator.

Deterministic rules:
1. Limit reprices and cancel/replace per symbol per short window (config):
   - max_cancel_replace_10s: 3 per 10 seconds
   - max_cancel_10s: 5 per 10 seconds
   - max_new_orders_10s: 5 per 10 seconds

2. If any limit is exceeded:
   - Block new entries for the symbol for churn_cooldown_seconds (default 60).
   - Allow only management/safe close.
   - Emit audit event with counters and symbol.

3. If the executor reports unfilled order count risk (X-MBX-ORDER-COUNT-* headers):
   - Force DEGRADE (reduce polling/actions) and apply backoff.
   - Block new entries system-wide until order count resets.

Counting algorithm:
- Maintain rolling windows: 10s, 60s, 300s
- Increment counters on: order create, cancel, cancel/replace
- Reset counters at window boundaries
- Store counters in memory with timestamp

Parameters (config.go):
- risk.churn.max_cancel_replace_10s: 3
- risk.churn.max_cancel_10s: 5
- risk.churn.max_new_orders_10s: 5
- risk.churn.cooldown_seconds: 60
- risk.churn.unfilled_order_warning: 80% of limit
- risk.churn.unfilled_order_critical: 95% of limit

Files:
- internal\engine\risk\churn_limits.go: tracks and enforces churn limits.
- internal\engine\executor\order_counter.go: counts orders and cancellations.
- internal\infra\binance\ratelimit.go: monitors X-MBX-ORDER-COUNT-* headers.

Reason codes:
- RISK_CHURN_LIMIT_HIT
- RISK_CANCEL_REPLACE_LIMIT_HIT
- RISK_UNFILLED_ORDER_COUNT_RISK

SYMBOL QUARANTINE (HEALTH / ANOMALY)
Problem: some symbols have unstable WS, "fake" liquidity, strange filters, or repeated errors.
Operating them increases risk and cost.

Rules:
1. Quarantine triggers (any of):
   - N filter rejections in window (default: 3 in 1 hour)
   - WS disconnections exceeding threshold (default: 5 in 10 minutes)
   - Inconsistent filters detected (FILTERS_DRIFT_DETECTED)
   - Symbol status not TRADING
   - Repeated order submission timeouts (default: 3 consecutive)

2. Quarantine action:
   - Add symbol to quarantine with TTL (default: 1 hour)
   - Universe/Rank/DeepScan must exclude quarantined symbols
   - Block new entries for the symbol
   - Allow only management/close of existing positions

3. Quarantine release:
   - Automatically after TTL expires
   - Manual release via audit/operation (requires reason)
   - On release, clear counters and reset state

Parameters (config.go):
- risk.quarantine.max_rejects_per_hour: 3
- risk.quarantine.max_ws_disconnects_per_10min: 5
- risk.quarantine.max_timeouts_consecutive: 3
- risk.quarantine.ttl_seconds: 3600 (1 hour)
- risk.quarantine.auto_release: true

Implementation:
- Maintain quarantine map: symbol -> quarantine_until_ms
- Check before any symbol operation
- Quarantine MUST be auditable via: (a) symbol_health table updates + (b) UNIVERSE_ELIGIBILITY reasons (SYMBOL_QUARANTINED).
- Persist quarantine state in SQLite for recovery

Files:
- internal\engine\universe\health_filters.go: excludes symbols by health/quarantine.
- internal\engine\risk\symbol_quarantine.go: controls counters and quarantine TTL.
- internal\domain\models\quarantine.go: quarantine state structure.

Reason codes:
- SYMBOL_QUARANTINED

DAILY AND SESSION LIMITS (FULL KILL SWITCH)
Complement to the kill switch (config.go):
1. Daily loss limit: considers realized PnL + unrealized PnL (mark-to-market)
2. Max intraday drawdown (optional): peak-to-trough decline
3. Max open orders (per symbol and total)
4. Max new trades per day (optional)

Rules:
1. Daily loss limit:
   - Track: realized_pnl + unrealized_pnl (mark-to-market)
   - If (realized_pnl + unrealized_pnl) <= daily_loss_limit: BLOCK
   - Reset at UTC 00:00 daily

2. Max intraday drawdown:
   - Track equity curve: starting_equity + realized_pnl + unrealized_pnl
   - Peak = max(equity_curve)
   - Drawdown = (peak - current_equity) / peak
   - If drawdown >= max_drawdown_pct: BLOCK

3. Max open orders:
   - Per symbol: cannot open new order if existing open order
   - Total: cannot exceed max_open_orders_total
   - Counts: LIMIT, STOP_LOSS, TAKE_PROFIT, OCO count as orders

4. Max trades per day:
   - Count successful fills (entry+exit pairs)
   - If count >= max_trades_per_day: BLOCK

When any kill switch triggers:
- Block new entries immediately
- Keep only management/safe close (or PAUSE per policy)
- Audit with reason_code and snapshot
- Emit ALERT_RAISED with severity CRIT

Parameters (config.go):
- risk.daily.loss_limit_usdt: -500.00
- risk.daily.max_drawdown_pct: 0.05 (5%)
- risk.daily.max_open_orders_per_symbol: 1
- risk.daily.max_open_orders_total: 10
- risk.daily.max_trades_per_day: 20

Files:
- internal\engine\risk\daily_limits.go: calculations and checks.
- internal\engine\risk\portfolio.go: enforcement of limits.
- internal\engine\reports\daily_summary.go: aggregates kill switch events.

Reason codes:
- RISK_DAILY_LOSS_LIMIT
- RISK_DRAWDOWN_LIMIT
- RISK_MAX_OPEN_ORDERS
- RISK_MAX_TRADES_DAY

POSITION POLICY (1 POSITION PER SYMBOL)
Project decision (safe to start): 1 position per symbol (no pyramiding).

Rules:
1. Do not open a new position if:
   - There is already an open position for the symbol (qty > 0)
   - There is a pending entry order for the symbol (order status = NEW/PARTIALLY_FILLED)
   - There is a pending OCO entry for the symbol

2. All sizing/exit/oco/trailing rules must assume this policy:
   - Only one active position management at a time
   - Protection applies to the single position
   - Trailing applies to the single position
   - Close applies to the entire position

3. Conflict resolution:
   - If conflict detected: BLOCK new entry
   - Audit conflict with existing position/order details
   - Allow existing position to be managed/closed normally

Implementation:
- Check position table for open positions (qty > 0)
- Check orders table for pending entry orders (side=BUY, status in [NEW, PARTIALLY_FILLED])
- Check OCO table for pending OCO entries
- Cache results per symbol to avoid repeated DB queries

Files:
- internal\engine\position\policy.go: conflict rules and "one-position-per-symbol".
- internal\engine\position\reconcile.go: verifies policy during reconcile.

Reason codes:
- RISK_POSITION_ALREADY_OPEN
- RISK_ENTRY_ALREADY_PENDING

CORRELATION AND CONCENTRATION (TOPK DIVERSIFY)
Goal: reduce indirect exposure when TopK selects highly correlated symbols.
Avoid trading 3 assets with the same risk factor in the same regime.
Maintain determinism and auditability.

Definitions:
- Correlation computed over returns_series in Snapshot (see 02_DATA_SNAPSHOT_SPEC.md).
- Window and timeframe defined in config.go and recorded in Snapshot.
- Returns series: 72 points of 5m log returns (6 hours).

Rules (deterministic):
1. Compute pairwise correlation (Pearson) among proposed TopK candidates (before executing any entry).
2. If any pair has corr > corr_max_pairwise (default 0.85):
   a. Attempt to replace the last candidate (lowest score in TopK) with next best from TopN that:
      - Is eligible (health, filters, spread, min edge/cost)
      - Reduces set max_pairwise_corr to <= corr_max_pairwise
   b. If no viable replacement:
      - Block additional new entries that would increase concentration
      - Keep at most 2 correlated symbols
      - If current entry would be third and raise max_pairwise_corr above limit: BLOCK

Conservative policy:
- Diversification may generate MODIFY only to swap candidate (at selection time), never to increase risk.
- RiskVerdict must record: max_pairwise_corr, pairs above limit, and action taken.

Correlation calculation:
func computeCorrelation(seriesA, seriesB []int32) float64 {
    // Pearson correlation on log_return_bps
    // Use integer arithmetic scaled by 10000
    // Returns value in range [-1.0, 1.0] * 10000 as int
}

Algorithm:
1. Get TopK candidates with scores
2. Compute all pairwise correlations (k*(k-1)/2 pairs)
3. Find max_correlation and pair
4. If max_correlation > threshold:
   - Sort candidates by score (ascending)
   - For each lower-scored candidate (starting with lowest):
     - Find replacement from TopN\TopK
     - Test if replacement reduces max correlation
     - If found: replace and recompute
5. If still above threshold after replacements:
   - If k <= 2: allow (minimum diversification)
   - If k > 2: block new entries that worsen correlation

Parameters (config):
- corr_max_pairwise: 0.85 (85%)
- corr_window_points: 72 (at 5m = 6h)
- corr_missing_max_pct: 10% (max missing data allowed)
- corr_min_symbols_for_check: 3 (only check if TopK >= 3)

Files:
- internal\engine\risk\correlation_diversify.go: correlation computation and diversification.
- internal\engine\universe\correlation.go: returns series management.
- internal\config\config.go: correlation parameters.

Reason codes:
- RISK_DIVERSIFY_APPLIED (replacement made)
- RISK_CORRELATION_TOO_HIGH (blocked due to correlation)

BUDGET / FUNDS LOCKED
Problem: in Spot, funds can be locked in open orders; using only "free" avoids failures and double-spend.
The system needs a deterministic budget allocator per symbol and total.

Rules:
1. Always compute availability using:
   - Balances: free/locked per asset
   - Open orders: reserves per symbol (price * qty)
   - Exposure limits: per symbol and total
   - Pending executions: intents in SENT_UNKNOWN state

2. Do not submit an order if:
   - Required quote > free_balance (even if total balance sufficient)
   - New exposure > max_exposure_symbol
   - Total exposure > max_exposure_total
   - Order would exceed position limits

3. Budget allocation algorithm:
   func canAfford(symbol string, required_quote decimal) bool {
       free = getFreeBalance(quote_asset)
       locked_in_orders = sum(reserves for symbol)
       pending = sum(intents in SENT_UNKNOWN for symbol)
       
       available = free - locked_in_orders - pending
       return required_quote <= available
   }

4. Reserve management:
   - On order intent CREATED: reserve funds
   - On order CONFIRMED: convert reserve to position
   - On order CANCELLED/REJECTED: release reserve
   - On reconcile: adjust reserves to match REST state

Implementation details:
- Track per-symbol reserves in memory and SQLite
- Update reserves atomically with order intent state changes
- Reconcile reserves on boot and periodically
- Handle partial fills: adjust reserve proportionally

Files:
- internal\engine\risk\budget_allocator.go: computes budgets and per-symbol reserves.
- internal\engine\position\reconcile.go: uses balances and orders to reconcile reserves.
- internal\domain\models\reserves.go: reserve tracking structure.

Reason codes:
- RISK_INSUFFICIENT_FREE_BALANCE

RECONCILE POLICY (DRIFT)
Drift thresholds and actions defined here and applied in reconcile.
Drift above tolerance must PAUSE/DEGRADE per policy (see 05_EXECUTION_AND_FAILSAFE.md).

Drift definition:
- Difference between local state (SQLite + intents) and REST state (exchange truth)
- Measured as integer score 0-100 (drift_score)
- Components:
  - Orders: missing, extra, status mismatch (weight: 40)
  - Positions: qty mismatch, side mismatch (weight: 30)
  - Balances: free/locked mismatch (weight: 20)
  - Protection: OCO/trailing state mismatch (weight: 10)

Thresholds (config.go):
- reconcile_drift_warn_score: 10 (investigate)
- reconcile_drift_degrade_score: 20 (block new entries)
- reconcile_drift_pause_score: 50 (pause system)

Actions:
1. drift_score < warn: OK, continue normal operation
2. warn <= drift_score < degrade: log warning, continue
3. degrade <= drift_score < pause: enter DEGRADE, block new entries
4. drift_score >= pause: enter PAUSE, require manual intervention

Drift computation algorithm:
func computeDriftScore(local, remote State) int {
    score = 0
    
    // Orders component (40 max)
    order_diff = compareOrders(local.orders, remote.orders)
    score += min(order_diff * 4, 40)
    
    // Positions component (30 max)
    position_diff = comparePositions(local.positions, remote.positions)
    score += min(position_diff * 6, 30)
    
    // Balances component (20 max)
    balance_diff = compareBalances(local.balances, remote.balances)
    score += min(balance_diff * 4, 20)
    
    // Protection component (10 max)
    protection_diff = compareProtection(local.protection, remote.protection)
    score += min(protection_diff * 2, 10)
    
    return score
}

Files:
- internal\engine\position\reconcile.go: drift detection and scoring.
- internal\engine\risk\reconcile_policy.go: thresholds and actions.
- internal\config\config.go: drift thresholds.

REASON CODES
Every block must generate clear reason_codes from the single catalog (01_DECISION_CONTRACT.md).

Risk-specific reason codes:
- RISK_SIZE_INVALID
- RISK_EXPOSURE_LIMIT
- RISK_COOLDOWN_ACTIVE
- RISK_MAX_TRADES_WINDOW
- RISK_SYMBOL_LOSS_STREAK
- RISK_CHURN_LIMIT_HIT
- RISK_CANCEL_REPLACE_LIMIT_HIT
- RISK_UNFILLED_ORDER_COUNT_RISK
- RISK_DAILY_LOSS_LIMIT
- RISK_DRAWDOWN_LIMIT
- RISK_MAX_OPEN_ORDERS
- RISK_MAX_TRADES_DAY
- RISK_POSITION_ALREADY_OPEN
- RISK_ENTRY_ALREADY_PENDING
- RISK_DIVERSIFY_APPLIED
- RISK_CORRELATION_TOO_HIGH
- RISK_INSUFFICIENT_FREE_BALANCE

Audit requirements:
- Every BLOCK must include reason_codes in RiskVerdict
- Every reason_code must be from the canonical catalog
- RiskVerdict must be persisted with decision
- Audit trail must show Risk decision separate from AI Gate decision

IMPLEMENTATION ORDER AND PRIORITY
1. Position sizing and exposure limits (HIGHEST - prevents overexposure)
2. Budget/funds check (HIGH - prevents failed orders)
3. Daily limits and kill switch (HIGH - protects capital)
4. Anti-overtrading and cooldowns (MEDIUM - prevents overtrading)
5. Correlation diversification (MEDIUM - reduces concentration)
6. Adaptive thresholds (LOW - fine-tuning)
7. Churn limits (LOW - operational hygiene)

All rules are mandatory and deterministic.
No rule may be bypassed or disabled in LIVE mode.