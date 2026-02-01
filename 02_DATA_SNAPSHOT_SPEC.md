02_DATA_SNAPSHOT_SPEC (SNAPSHOT SPECIFICATION)

GOAL
Define the minimum Snapshot contents so that:
- Strategy and Risk are reproducible (determinism)
- Auditing, post-mortem reconstruction, and deterministic auditability are supported
- the AI Gate receives enough information to act conservatively

TIMEFRAMES
Project decision (editable in config.go):
- 1m: execution/management (triggers, trailing, microstructure)
- 5m: short confirmation and noise filters
- 15m: context and higher-level direction
- 1h: regime filter (trend vs range) and day volatility

Implementation:
- internal\config\config.go: timeframes and weights per stage (rank/deepscan/strategy/risk).
- internal\domain\timeframes.go: enum/supported timeframes and helpers.
- internal\engine\state\: candle aggregation per timeframe.
- internal\engine\state\candle_store.go: applies candle persistence policy.

REGIME DETECTION (FIRST CLASS)
Regime is used to:
- filter trades (no-trade by regime)
- adjust entry (maker-first vs aggressive)
- adjust TP/SL (ATR k/m by regime)
- adjust trailing (activation and distance by ATR)

Minimum rules:
- regime is computed deterministically multi-timeframe (1m/5m/15m/1h).
- a trade is considered only if there is minimum coherence (e.g., aligned trend or stable range, per strategy defined in 03_STRATEGY.md).
- regime and scores enter the Snapshot, with timestamps.

Regime computation algorithm:
1. For each timeframe (1m, 5m, 15m, 1h):
   - Compute ADX (14 period)
   - Compute +DI and -DI (14 period)
   - Determine trend strength: ADX >= 25 = strong trend
   - Determine direction: +DI > -DI = bullish, -DI > +DI = bearish
2. Aggregate across timeframes with weights:
   - 1h: weight 0.4
   - 15m: weight 0.3
   - 5m: weight 0.2
   - 1m: weight 0.1
3. Calculate composite scores:
   - trend_score_x10000 = round(aggregated_trend_strength * 10000)
   - range_score_x10000 = round((100 - aggregated_trend_strength) * 100)
4. Determine regime_label:
   - If trend_score_x10000 >= 7000: "TREND"
   - If range_score_x10000 >= 6000: "RANGE"
   - Else: "UNCLEAR"

Files:
- internal\engine\state\regime.go : computes regime and scores.
- internal\domain\contracts\snapshot.go: adds regime/trend_score_x10000/range_score_x10000 fields.
- internal\engine\strategy\strategy.go: uses regime as a filter and for parametrization.
- internal\engine\risk\regime_rules.go : final deterministic regime blocks.

Reason codes: see the single catalog in 01_DECISION_CONTRACT.md (prefixes REGIME_*, SPREAD_*, IMBALANCE_*, MICROVOL_*, WS_*, CLOCK_* as applicable).

MICROSTRUCTURE
Deterministic and cheap signals (via bookTicker and short windows):
- spread_bps_p50 and spread_bps_p90 (last 60s): trade only if the current spread is good vs normal.
- approximate imbalance: bidQty/(bidQty+askQty) (derive bid_ask_imbalance_*_x10000).
- spread delta in 5–10s: avoid entering when the spread is opening.
- optional: mid microvol and "spread shock" (spike vs normal).

Microstructure computation:
1. Spread calculation (per bookTicker event):
   - spread_bps = ((ask - bid) / mid) * 10000
   - Store in a 60-second window sampled every 500ms (exactly 120 samples once warmed up)
2. Statistics (over 60s window):
   - spread_bps_p50_60s: median spread
   - spread_bps_p90_60s: 90th percentile spread
   - spread_current_bps: most recent spread
3. Delta calculation:
   - delta_spread_bps_p90_10s = current_p90 - p90_10s_ago
4. Imbalance calculation (10s window):
   - bid_ask_imbalance_p50_10s_x10000 = median(bidQty/(bidQty+askQty)) * 10000
5. Out-of-order tracking:
   - Count events where exchange_time_ms < previous_exchange_time_ms
   - Window: 60 seconds

Files:
- internal\engine\state\microstructure.go : computes median/percentile/deltas/imbalance.
- internal\engine\state\microvol.go : microvol based on mid.
- internal\domain\contracts\snapshot.go: adds microstructure fields.
- internal\domain\reasoncodes\codes.go: adds reason_codes.

Reason codes: see the single catalog in 01_DECISION_CONTRACT.md (prefixes REGIME_*, SPREAD_*, IMBALANCE_*, MICROVOL_*, WS_*, CLOCK_* as applicable).

TIMESTAMPS AND EVENT ORDER
Rule:
- always record exchange_time_ms and local_received_ms.
- tolerate out-of-order events (short window) and audit drops.
- abstract the clock for deterministic tests.

Out-of-order window: 5000ms (5 seconds)
Events outside this window are discarded with audit event WS_OOO_EVENT.

Files:
- internal\infra\clock\real.go and fake.go : real and fake clock.
- internal\engine\state\event_order.go : out-of-order window.
- internal\audit\writer.go: persist dual timestamps.

Reason codes: see the single catalog in 01_DECISION_CONTRACT.md (prefixes REGIME_*, SPREAD_*, IMBALANCE_*, MICROVOL_*, WS_*, CLOCK_* as applicable).

CANDLES AND PERSISTENCE POLICY
Default policy: persist candles for watchlist and traded symbols (not the whole universe).

Candle specifications:
- Source: Binance klines (REST)
- Timeframes: 1m, 5m, 15m, 1h
- Fields per candle: open, high, low, close, volume (all decimal strings)
- Retention in memory: 1000 candles per timeframe per symbol
- Persistence to SQLite: all candles for watchlist symbols; no persistence for non-watchlist symbols
- Timestamp alignment: use exchange kline open time, not local time

Candle validation:
- Open <= High, Low <= Close, Low <= Open, Low <= Close
- Volume >= 0
- Timestamps monotonically increasing
- Invalid candles are discarded with audit

Files:
- internal\engine\state\candle_store.go: candle aggregation and persistence.
- internal\domain\contracts\candle.go: candle structure.

GLOBAL -> TOPN -> TOPK (MINIMUM DATA FOR SELECTION)
Goal:
- enable deterministic and reproducible TopN ranking and deep scan
- reduce chasing and churn with stable metrics
- enable correlation filtering (TopK diversification) without depending on external state

Rules:
- any metric used in eligibility/ranking/deep must be present in the Snapshot (or referenced by persisted-entity id in SQLite)
- any REST data affecting selection (e.g., 24h ticker) must be persisted for audit reconstruction / verification
- the Snapshot must include the effective thresholds used in the cycle (or a reference to config_hash)

SNAPSHOT STRUCTURE (PER CANDIDATE SYMBOL)

Required fields for Strategy (03_STRATEGY.md) - closed list:
A) regime
   - regime_label: "TREND" | "RANGE" | "UNCLEAR"
   - trend_score_x10000: int [0..10000]
   - range_score_x10000: int [0..10000]

B) microstructure_60s
   - spread_bps_p50_60s: int
   - spread_bps_p90_60s: int
   - spread_current_bps: int
   - delta_spread_bps_p90_10s: int
   - bid_ask_imbalance_p50_10s_x10000: int [0..10000]
   - out_of_order_drops: int

C) volatility
   - atr14_5m_bps: int
   - atr14_15m_bps: int (optional, default 0)

D) market price inputs
   - best_bid: string decimal (e.g., "41250.75")
   - best_ask: string decimal
   - mid_price: string decimal ((bid+ask)/2)
   - last_price: string decimal

E) candles_5m
   - Array of 40 candles (5-minute)
   - Each candle: {
       "ts_ms": int64 (open time),
       "open": string decimal,
       "high": string decimal,
       "low": string decimal,
       "close": string decimal,
       "volume": string decimal
     }
   - Ascending temporal order (oldest first)

F) cost_inputs
   - maker_fee_bps: int (e.g., 2 for 0.02%)
   - taker_fee_bps: int (e.g., 4 for 0.04%)
   - slippage_est_entry_maker_bps: int
   - slippage_est_entry_taker_bps: int
   - slippage_est_exit_taker_bps: int

Additional fields for Universe/Rank/DeepScan:

G) market_24h (from REST 24h ticker)
   - quote_volume_24h_usdt: string decimal
   - trades_24h: int
   - price_change_24h_bps: int
   - source_ts_ms: int64 (when data was fetched)

H) health_flags
   - filters_ok: bool (exchangeInfo filters loaded and valid)
   - ws_ok: bool (WS connection stable for this symbol)
   - recent_rejects_window_count: int (rejections in last 1 hour)
   - quarantined_until_ms: int64 (0 if not quarantined)
   - symbol_status: string ("TRADING", "HALT", "BREAK")

I) returns_series (for correlation)
   - timeframe: "5m" (default)
   - window_points: 72 (6 hours at 5m)
   - log_return_bps: []int32 (length = window_points)
   - missing_count: int (number of missing points)
   - computed_ts_ms: int64 (when series was computed)

J) configuration_reference
   - config_hash: string (SHA-256 of config.go relevant sections)
   - thresholds_hash: string (SHA-256 of thresholds used in ranking)
   - cycle_config_version: string (e.g., "20260131_1")

K) metadata
   - snapshot_id: string (deterministic, e.g., "snap_BTCUSDT_1706700000_abc123")
   - created_ts_ms: int64
   - exchange_time_ms: int64 (most recent exchange timestamp in snapshot)
   - local_received_ms: int64 (when snapshot was finalized)
   - source_hashes: {
       "candles_hash": string,
       "book_hash": string,
       "ticker_hash": string
     }

RETURNS SERIES FOR CORRELATION (TOPK DIVERSIFICATION)

Purpose: Enable deterministic pairwise correlation computation in-cycle.

Specification:
- Timeframe: 5m (default)
- Window: 72 points (6 hours)
- Representation: log_return_bps int32 (return in bps, rounded)
- Calculation:
  For each point i (0 = oldest, 71 = most recent):
  return_bps_i = round(ln(close_i / close_{i-1}) * 10000)
- Missing data policy:
  If missing_count > (window_points * 0.10) (10%): symbol not eligible for TopK
  Missing points represented as 0 (neutral return)

Storage:
- Only computed for symbols in TopN (configurable, default 20)
- Persisted in Snapshot for audit reconstruction
- Not computed for entire universe (performance)

Correlation calculation (Pearson):
corr = Σ((x_i - x̄)(y_i - ȳ)) / √(Σ(x_i - x̄)² * Σ(y_i - ȳ)²)
Where x_i, y_i are log_return_bps values.
Implemented with integer arithmetic (scaled by 10000).

UNIT CONVENTIONS (FOR DETERMINISM)

Basic units:
- bps: basis points (1 bps = 0.01%), integer
- x10000: scaled integer (real_value * 10000), int32
- decimal strings: no exponent, '.' separator, trailing zeros allowed
- timestamps: milliseconds since Unix epoch, int64

Conversion rules:
- Percentage to bps: round(pct * 100)
- Float to x10000: round(float_value * 10000)
- Price to string: format with required precision, no scientific notation
- Volume to string: format with 8 decimal places minimum

SNAPSHOT_HASH_PAYLOAD (CANONICAL, NO FLOAT)

Rule:
- snapshot_hash is computed over a canonical SnapshotHashPayload (RFC 8785) using only deterministic fields.
- SnapshotHashPayload must not contain float (including float in JSON).
- price/qty must be decimal strings with no exponent and with "." as separator.
- "pct" fields may exist for display/telemetry, but never enter SnapshotHashPayload.

Minimum required fields in SnapshotHashPayload (per symbol in TopN):

{
  "symbol": "BTCUSDT",
  "symbol_status": "TRADING",
  "filters_hash": "abc123...",
  "config_hash": "def456...",
  "thresholds_hash": "ghi789...",
  
  "regime": {
    "label": "TREND",
    "trend_score_x10000": 7500,
    "range_score_x10000": 2500
  },
  
  "microstructure_60s": {
    "spread_bps_p50_60s": 15,
    "spread_bps_p90_60s": 25,
    "delta_spread_bps_p90_10s": 3,
    "bid_ask_imbalance_p50_10s_x10000": 6000,
    "out_of_order_drops": 2
  },
  
  "volatility": {
    "atr14_5m_bps": 45
  },
  
  "prices": {
    "best_bid": "41250.75",
    "best_ask": "41251.00",
    "mid_price": "41250.875",
    "last_price": "41250.80"
  },
  
  "cost_inputs": {
    "maker_fee_bps": 2,
    "taker_fee_bps": 4,
    "slippage_est_entry_maker_bps": 1,
    "slippage_est_entry_taker_bps": 3,
    "slippage_est_exit_taker_bps": 2
  },
  
  "returns_series": {
    "timeframe": "5m",
    "window": 72,
    "log_return_bps": [15, -8, 22, 5, -12, ...] // 72 integers
  }
}

Excluded from hash (volatile or derived):
- candles_5m (too large, referenced by hash)
- market_24h (volatile, referenced by timestamp)
- health_flags (volatile state)
- metadata fields (timestamps, IDs)
- Any debug or telemetry fields

Implementation:
- Use RFC 8785 canonical JSON (lexicographic key ordering)
- SHA-256 hex (lowercase)
- Hash computed when Snapshot is finalized
- Stored in Snapshot.metadata.snapshot_hash

SOURCE OF TRUTH (WS VS REST)

Goal: Eliminate ambiguities and reduce drift and bugs.

Rules:
- WS is the source for:
  - Real-time market data (bookTicker, trades)
  - Signals and state (ticks, microstructure, microvol)
  - Immediate price updates for execution
  
- REST is the source for:
  - Execution and confirmation (order status, fills, balances)
  - Historical data (candles, 24h ticker)
  - Exchange info (filters, limits)
  - Reconcile (final truth in Live)
  - Account state (balances, positions)

- Auditing persists both:
  - exchange_time_ms (from exchange)
  - local_received_ms (when data was processed)
  - source annotation (WS or REST)

Conflict resolution:
- For market prices: WS takes precedence (more recent)
- For order state: REST is final truth
- For account balances: REST is final truth
- On reconcile mismatch: trust REST, audit discrepancy

Files:
- internal\domain\source_of_truth.go: enum/helpers for data origin.
- internal\engine\position\reconcile.go: applies REST as the final truth.

BOOK SLICES AND DEPTH

Purpose: Estimate impact/slippage for execution planning.

Minimum depth data:
- Source: REST /api/v3/depth (limit=20)
- Frequency: Every 30 seconds for watchlist symbols, on-demand for execution
- Fields:
  - bids: [["price", "quantity"], ...] (top 20)
  - asks: [["price", "quantity"], ...] (top 20)
  - timestamp_ms: int64
  - symbol: string

Storage in Snapshot:
- Include top 5 levels for slippage estimation
- Or include depth hash for reference
- Record origin (REST) and timestamp

Slippage estimation:
- For MARKET orders: estimate fill across order book
- For LIMIT orders: check if price crosses spread
- Model: linear impact based on order size vs depth

COSTS AND INPUTS FOR EDGE

Record assumed costs for edge calculation:

1. Fees (from Binance fee schedule):
   - maker_fee_bps: integer (e.g., 2 for 0.02%)
   - taker_fee_bps: integer (e.g., 4 for 0.04%)

2. Spread costs:
   - spread_current_bps: from microstructure
   - spread_roundtrip_bps = spread_current_bps * 2 (entry + exit)

3. Slippage estimates:
   - Based on order size vs depth
   - Entry maker: slippage_est_entry_maker_bps (int bps; deterministic input captured in Snapshot)
   - Entry taker: slippage_est_entry_taker_bps (int bps; deterministic input captured in Snapshot)
   - Exit taker: slippage_est_exit_taker_bps (int bps; deterministic input captured in Snapshot)

Deterministic defaults (used only if no explicit slippage estimates are available):
- slippage_est_entry_maker_bps = 1
- slippage_est_entry_taker_bps = 3
- slippage_est_exit_taker_bps = 2

If the system cannot provide explicit slippage estimates, it MUST use the defaults above. Defaults MUST NOT be disabled in LIVE. If explicit estimates are missing, Snapshot MUST use the defaults above.


4. Delta spread penalty:
   - delta_spread_bps_p90_10s: if > 0, add to costs

All costs in integer bps for deterministic calculation.

VALIDATION AND COMPLETENESS

Snapshot validation rules:
1. Required fields present (Strategy fields)
2. Timestamps within deterministic bounds:
   - Let now_ms = clock.now_ms() (monotonic wall clock abstraction).
   - For every timestamp field ts_ms in the Snapshot:
     - ts_ms MUST be <= (now_ms + time_sync_recv_window_ms)
     - ts_ms MUST be >= (now_ms - rest_stale_ms_pause)

3. Prices positive and bid <= ask
4. Scores within valid ranges (0..10000)
5. Decimal strings properly formatted
6. Hashes valid (64 hex chars)
7. No NaN or Infinity values

If validation fails:
- Discard snapshot
- Emit audit event UNIVERSE_ELIGIBILITY with eligible=false and reasons=[STRAT_INPUT_INVALID]
- Skip symbol for current cycle

Completeness guarantee:
The Snapshot must contain sufficient data to:
1. Reproduce Strategy decision (edge_score, EntryPlan, ExitPlan)
2. Reproduce Risk decision (limits, exposure)
3. Reproduce AI Gate decision (conservative evaluation)
4. Audit and reconstruct post-mortem
5. Compute correlation for diversification

PERFORMANCE AND SIZE CONSIDERATIONS

Size limits:
- TopN symbols: 20 (configurable)
- Candles per symbol: 40 (5m) = ~2KB
- Returns series: 72 integers = ~288 bytes
- Total per snapshot: ~50KB
- Memory footprint: < 10MB

Retention:
- In memory: current cycle only
- SQLite: all snapshots for audit
- JSONL: sampled (every 10th cycle)

Optimizations:
- Lazy loading of candles
- Incremental updates to returns series
- Hash-based change detection
- Compression for storage

IMPLEMENTATION NOTES

Determinism requirements:
1. Same inputs → same snapshot_hash
2. Same snapshot_hash → same decision (via Strategy)
3. Rounding and calculations must be reproducible
4. Timestamps affect hashes only via snapshot_id
5. Volatile fields excluded from hash

Testing:
- Golden tests with known inputs/outputs
- Hash consistency across runs
- Snapshot reconstruction from audit
- Performance benchmarks

Files to create/update:
- internal/domain/contracts/snapshot.go (main structure)
- internal/engine/state/snapshot_builder.go (construction)
- internal/engine/state/snapshot_validator.go (validation)
- migrations/000x_snapshots.sql (DB schema)

NON-NEGOTIABLE
- The snapshot must be sufficient to explain "why" and "how" the decision was made.
- The snapshot must carry exchange_time_ms and local_received_ms when WS data exists.
- The snapshot must be deterministic (same inputs → same snapshot_hash).
- The snapshot must support audit reconstruction.
- Missing required fields must cause validation failure, not silent defaults.