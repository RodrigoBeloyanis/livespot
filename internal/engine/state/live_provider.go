package state

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/hash"
	"github.com/RodrigoBeloyanis/livespot/internal/engine/selection"
	"github.com/RodrigoBeloyanis/livespot/internal/infra/binance"
)

type LiveSnapshotProvider struct {
	cfg     config.Config
	client  *binance.Client
	store   *BookTickerStore
	filters map[string]binance.SymbolFilters
	lastAccount binance.AccountInfo
	lastAccountTs int64
}

type SnapshotBundle struct {
	Snapshot    contracts.Snapshot
	Constraints contracts.DecisionConstraints
}

func NewLiveSnapshotProvider(cfg config.Config, client *binance.Client, store *BookTickerStore, filters map[string]binance.SymbolFilters) *LiveSnapshotProvider {
	return &LiveSnapshotProvider{cfg: cfg, client: client, store: store, filters: filters}
}

func (p *LiveSnapshotProvider) AccountInfo() (binance.AccountInfo, int64, bool) {
	if p.lastAccountTs == 0 {
		return binance.AccountInfo{}, 0, false
	}
	return p.lastAccount, p.lastAccountTs, true
}

func (p *LiveSnapshotProvider) BuildSnapshots(ctx context.Context, now time.Time) ([]SnapshotBundle, error) {
	if p.client == nil || p.store == nil {
		return nil, fmt.Errorf("provider missing deps")
	}
	tickers, err := p.client.Ticker24hAll(ctx)
	if err != nil {
		return nil, err
	}
	candidates := filterCandidates(tickers, p.filters)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates")
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].QuoteVolume == candidates[j].QuoteVolume {
			return candidates[i].Symbol < candidates[j].Symbol
		}
		return candidates[i].QuoteVolume > candidates[j].QuoteVolume
	})
	if len(candidates) > p.cfg.TopNSize {
		candidates = candidates[:p.cfg.TopNSize]
	}

	configHash, err := selection.ConfigHash(p.cfg)
	if err != nil {
		return nil, err
	}
	thresholdsHash, err := selection.ThresholdsHash(p.cfg)
	if err != nil {
		return nil, err
	}
	nowMs := now.UnixMilli()
	out := make([]SnapshotBundle, 0, len(candidates))
	for _, cand := range candidates {
		f := p.filters[cand.Symbol]
		constraints, err := binance.ConstraintsFromFilters(f)
		if err != nil {
			continue
		}
		filtersHash, err := binance.FiltersHash(f)
		if err != nil {
			continue
		}
		micro, prices, bookHash, bookTs := p.store.Snapshot(cand.Symbol, nowMs)
		if prices.BestBid == "" || prices.BestAsk == "" {
			continue
		}
		wsOK := nowMs-bookTs <= int64(p.cfg.WsStaleMsDegrade)
		prices.LastPrice = cand.LastPrice

		k5m, err := p.client.Klines(ctx, cand.Symbol, "5m", 80)
		if err != nil {
			continue
		}
		k1m, err := p.client.Klines(ctx, cand.Symbol, "1m", 50)
		if err != nil {
			continue
		}
		k15m, err := p.client.Klines(ctx, cand.Symbol, "15m", 50)
		if err != nil {
			continue
		}
		k1h, err := p.client.Klines(ctx, cand.Symbol, "1h", 50)
		if err != nil {
			continue
		}
		candles5m, err := lastCandles(k5m, 40)
		if err != nil {
			continue
		}
		atr5m, err := atrFromKlines(k5m)
		if err != nil {
			continue
		}
		atr15m, err := atrFromKlines(k15m)
		if err != nil {
			continue
		}
		regime, err := regimeFromKlines(k1m, k5m, k15m, k1h)
		if err != nil {
			continue
		}
		returns, missing, err := returnsFromKlines(k5m, 72)
		if err != nil {
			continue
		}
		account, err := p.client.Account(ctx)
		if err != nil {
			continue
		}
		accountInfo, err := binance.ParseAccountInfo(account.Body)
		if err != nil {
			continue
		}
		p.lastAccount = accountInfo
		p.lastAccountTs = nowMs
		snapshot := contracts.Snapshot{
			Symbol: cand.Symbol,
			Regime: regime,
			Microstructure60s: micro,
			Volatility: contracts.VolatilitySnapshot{
				ATR14_5mBps:  atr5m,
				ATR14_15mBps: atr15m,
			},
			Prices: prices,
			Candles5m: candles5m,
			CostInputs: contracts.CostInputs{
				MakerFeeBps:           accountInfo.MakerCommission,
				TakerFeeBps:           accountInfo.TakerCommission,
				SlippageEntryMakerBps: 0,
				SlippageEntryTakerBps: 0,
				SlippageExitTakerBps:  0,
			},
			Market24h: contracts.Market24hSnapshot{
				QuoteVolume24hUSDT: cand.QuoteVolume,
				Trades24h:          cand.Count,
				PriceChange24hBps:  cand.PriceChangeBps,
				SourceTsMs:         nowMs,
			},
			HealthFlags: contracts.HealthFlagsSnapshot{
				FiltersOK:                true,
				WSOK:                     wsOK,
				RecentRejectsWindowCount: 0,
				QuarantinedUntilMs:       0,
				SymbolStatus:             f.Status,
			},
			ReturnsSeries: contracts.ReturnsSeries{
				Timeframe:    "5m",
				WindowPoints: 72,
				LogReturnBps: returns,
				MissingCount: missing,
				ComputedTsMs: nowMs,
			},
			ConfigReference: contracts.ConfigurationReference{
				ConfigHash:         configHash,
				ThresholdsHash:     thresholdsHash,
				CycleConfigVersion: cycleConfigVersion(now),
				FiltersHash:        filtersHash,
			},
			Metadata: contracts.SnapshotMetadata{
				SnapshotID:      snapshotID(cand.Symbol, bookTs),
				CreatedTsMs:     nowMs,
				ExchangeTimeMs:  bookTs,
				LocalReceivedMs: nowMs,
				SourceHashes: contracts.SourceHashes{
					CandlesHash: hashKlines(k5m),
					BookHash:    bookHash,
					TickerHash:  cand.TickerHash,
				},
				SnapshotHash: "",
			},
		}
		built, err := BuildSnapshot(p.cfg, snapshot, nowMs)
		if err != nil {
			continue
		}
		out = append(out, SnapshotBundle{Snapshot: built, Constraints: constraints})
	}
	return out, nil
}

type tickerCandidate struct {
	Symbol          string
	LastPrice       string
	QuoteVolume     string
	Count           int
	PriceChangeBps  int
	TickerHash      string
}

func filterCandidates(tickers []binance.Ticker24h, filters map[string]binance.SymbolFilters) []tickerCandidate {
	out := make([]tickerCandidate, 0, len(tickers))
	for _, t := range tickers {
		f, ok := filters[t.Symbol]
		if !ok || f.Status != "TRADING" || f.QuoteAsset != "USDT" {
			continue
		}
		priceChangeBps := percentToBps(t.PriceChangePercent)
		tickerHash := hashTicker(t)
		out = append(out, tickerCandidate{
			Symbol:         t.Symbol,
			LastPrice:      t.LastPrice,
			QuoteVolume:    t.QuoteVolume,
			Count:          t.Count,
			PriceChangeBps: priceChangeBps,
			TickerHash:     tickerHash,
		})
	}
	return out
}

func CandidateSymbols(cfg config.Config, tickers []binance.Ticker24h, filters map[string]binance.SymbolFilters) []string {
	candidates := filterCandidates(tickers, filters)
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].QuoteVolume == candidates[j].QuoteVolume {
			return candidates[i].Symbol < candidates[j].Symbol
		}
		return candidates[i].QuoteVolume > candidates[j].QuoteVolume
	})
	if len(candidates) > cfg.TopNSize {
		candidates = candidates[:cfg.TopNSize]
	}
	out := make([]string, 0, len(candidates))
	for _, cand := range candidates {
		out = append(out, cand.Symbol)
	}
	return out
}

func percentToBps(value string) int {
	r, ok := new(big.Rat).SetString(value)
	if !ok {
		return 0
	}
	r.Mul(r, big.NewRat(100, 1))
	f, _ := r.Float64()
	return int(math.RoundToEven(f))
}

func lastCandles(klines []binance.Kline, count int) ([]contracts.Candle, error) {
	if len(klines) < count {
		return nil, fmt.Errorf("candles")
	}
	start := len(klines) - count
	out := make([]contracts.Candle, 0, count)
	for _, k := range klines[start:] {
		out = append(out, contracts.Candle{
			TsMs:   k.OpenTime,
			Open:   k.Open,
			High:   k.High,
			Low:    k.Low,
			Close:  k.Close,
			Volume: k.Volume,
		})
	}
	return out, nil
}

func atrFromKlines(klines []binance.Kline) (int, error) {
	high, low, close, err := klineToFloats(klines)
	if err != nil {
		return 0, err
	}
	return ATR14Bps(high, low, close)
}

func regimeFromKlines(k1m []binance.Kline, k5m []binance.Kline, k15m []binance.Kline, k1h []binance.Kline) (contracts.RegimeSnapshot, error) {
	adx1m, err := adxFromKlines(k1m)
	if err != nil {
		return contracts.RegimeSnapshot{}, err
	}
	adx5m, err := adxFromKlines(k5m)
	if err != nil {
		return contracts.RegimeSnapshot{}, err
	}
	adx15m, err := adxFromKlines(k15m)
	if err != nil {
		return contracts.RegimeSnapshot{}, err
	}
	adx1h, err := adxFromKlines(k1h)
	if err != nil {
		return contracts.RegimeSnapshot{}, err
	}
	trend, rng, label := RegimeScores(adx1m, adx5m, adx15m, adx1h)
	return contracts.RegimeSnapshot{
		Label:            label,
		TrendScoreX10000: trend,
		RangeScoreX10000: rng,
	}, nil
}

func adxFromKlines(klines []binance.Kline) (adxResult, error) {
	high, low, close, err := klineToFloats(klines)
	if err != nil {
		return adxResult{}, err
	}
	return ComputeADX(high, low, close)
}

func returnsFromKlines(klines []binance.Kline, window int) ([]int32, int, error) {
	_, _, close, err := klineToFloats(klines)
	if err != nil {
		return nil, 0, err
	}
	return ReturnsSeriesBps(close, window)
}

func klineToFloats(klines []binance.Kline) ([]float64, []float64, []float64, error) {
	high := make([]float64, 0, len(klines))
	low := make([]float64, 0, len(klines))
	close := make([]float64, 0, len(klines))
	for _, k := range klines {
		h, ok := new(big.Rat).SetString(k.High)
		if !ok {
			return nil, nil, nil, fmt.Errorf("high")
		}
		l, ok := new(big.Rat).SetString(k.Low)
		if !ok {
			return nil, nil, nil, fmt.Errorf("low")
		}
		c, ok := new(big.Rat).SetString(k.Close)
		if !ok {
			return nil, nil, nil, fmt.Errorf("close")
		}
		hf, _ := h.Float64()
		lf, _ := l.Float64()
		cf, _ := c.Float64()
		high = append(high, hf)
		low = append(low, lf)
		close = append(close, cf)
	}
	return high, low, close, nil
}

func hashKlines(klines []binance.Kline) string {
	buf, err := hash.CanonicalJSON(klines)
	if err != nil {
		return ""
	}
	return hash.HashSHA256Hex(buf)
}

func hashTicker(t binance.Ticker24h) string {
	buf, err := hash.CanonicalJSON(t)
	if err != nil {
		return ""
	}
	return hash.HashSHA256Hex(buf)
}

func snapshotID(symbol string, exchangeTimeMs int64) string {
	return fmt.Sprintf("snap_%s_%d", strings.ToUpper(symbol), exchangeTimeMs)
}

func cycleConfigVersion(now time.Time) string {
	return now.UTC().Format("20060102") + "_1"
}
