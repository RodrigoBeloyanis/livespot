package state

import (
	"fmt"
	"math"
	"math/big"
	"sort"
	"sync"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/hash"
)

type BookTickerStore struct {
	mu      sync.Mutex
	symbols map[string]*symbolBookState
}

type symbolBookState struct {
	samples []bookSample
	drops   int
	lastTs  int64
}

type bookSample struct {
	tsMs     int64
	bid      string
	ask      string
	bidQty   string
	askQty   string
	spreadBps int
	imbalanceX10000 int
}

func NewBookTickerStore() *BookTickerStore {
	return &BookTickerStore{symbols: map[string]*symbolBookState{}}
}

func (s *BookTickerStore) Add(symbol string, eventTsMs int64, bid string, ask string, bidQty string, askQty string) {
	if symbol == "" || eventTsMs <= 0 || bid == "" || ask == "" {
		return
	}
	spreadBps, imbalance, err := computeSpreadImbalance(bid, ask, bidQty, askQty)
	if err != nil {
		return
	}
	s.mu.Lock()
	state := s.symbols[symbol]
	if state == nil {
		state = &symbolBookState{}
		s.symbols[symbol] = state
	}
	if state.lastTs > 0 && eventTsMs < state.lastTs-5000 {
		state.drops++
		s.mu.Unlock()
		return
	}
	state.lastTs = eventTsMs
	state.samples = append(state.samples, bookSample{
		tsMs:     eventTsMs,
		bid:      bid,
		ask:      ask,
		bidQty:   bidQty,
		askQty:   askQty,
		spreadBps: spreadBps,
		imbalanceX10000: imbalance,
	})
	s.prune(state, eventTsMs)
	s.mu.Unlock()
}

func (s *BookTickerStore) Snapshot(symbol string, nowMs int64) (contracts.Microstructure60s, contracts.PricesSnapshot, string, int64) {
	s.mu.Lock()
	state := s.symbols[symbol]
	if state == nil || len(state.samples) == 0 {
		s.mu.Unlock()
		return contracts.Microstructure60s{}, contracts.PricesSnapshot{}, "", 0
	}
	s.prune(state, nowMs)
	samples := append([]bookSample{}, state.samples...)
	drops := state.drops
	last := samples[len(samples)-1]
	s.mu.Unlock()

	spreads60 := filterByWindow(samples, nowMs-60000, nowMs, func(s bookSample) int { return s.spreadBps })
	imb10 := filterByWindow(samples, nowMs-10000, nowMs, func(s bookSample) int { return s.imbalanceX10000 })
	p90Now := percentile(spreads60, 0.9)
	p50Now := percentile(spreads60, 0.5)
	p90Prev := percentile(filterByWindow(samples, nowMs-20000, nowMs-10000, func(s bookSample) int { return s.spreadBps }), 0.9)
	delta := p90Now - p90Prev
	if p90Prev == 0 {
		delta = 0
	}
	imbMedian := percentile(imb10, 0.5)

	micro := contracts.Microstructure60s{
		SpreadBpsP50_60s:             p50Now,
		SpreadBpsP90_60s:             p90Now,
		SpreadCurrentBps:             last.spreadBps,
		DeltaSpreadBpsP90_10s:        delta,
		BidAskImbalanceP50_10sX10000: imbMedian,
		OutOfOrderDrops:              drops,
	}
	prices := contracts.PricesSnapshot{
		BestBid:   last.bid,
		BestAsk:   last.ask,
		MidPrice:  midPrice(last.bid, last.ask),
		LastPrice: last.bid,
	}
	bookHash := hashBook(last)
	return micro, prices, bookHash, last.tsMs
}

func (s *BookTickerStore) prune(state *symbolBookState, nowMs int64) {
	if len(state.samples) == 0 {
		return
	}
	cutoff := nowMs - 60000
	idx := 0
	for idx < len(state.samples) && state.samples[idx].tsMs < cutoff {
		idx++
	}
	if idx > 0 {
		state.samples = append([]bookSample{}, state.samples[idx:]...)
	}
}

func computeSpreadImbalance(bid string, ask string, bidQty string, askQty string) (int, int, error) {
	bidRat, ok := new(big.Rat).SetString(bid)
	if !ok {
		return 0, 0, fmt.Errorf("bid")
	}
	askRat, ok := new(big.Rat).SetString(ask)
	if !ok {
		return 0, 0, fmt.Errorf("ask")
	}
	if bidRat.Sign() <= 0 || askRat.Sign() <= 0 {
		return 0, 0, fmt.Errorf("price")
	}
	mid := new(big.Rat).Quo(new(big.Rat).Add(bidRat, askRat), big.NewRat(2, 1))
	diff := new(big.Rat).Sub(askRat, bidRat)
	spread := new(big.Rat).Mul(new(big.Rat).Quo(diff, mid), big.NewRat(10000, 1))
	spreadF, _ := spread.Float64()
	spreadBps := int(math.RoundToEven(spreadF))
	bq, ok := new(big.Rat).SetString(bidQty)
	if !ok {
		return 0, 0, fmt.Errorf("bidQty")
	}
	aq, ok := new(big.Rat).SetString(askQty)
	if !ok {
		return 0, 0, fmt.Errorf("askQty")
	}
	sum := new(big.Rat).Add(bq, aq)
	if sum.Sign() == 0 {
		return spreadBps, 0, nil
	}
	imb := new(big.Rat).Quo(bq, sum)
	imbF, _ := imb.Float64()
	imbX10000 := int(math.RoundToEven(imbF * 10000.0))
	if imbX10000 < 0 {
		imbX10000 = 0
	}
	if imbX10000 > 10000 {
		imbX10000 = 10000
	}
	return spreadBps, imbX10000, nil
}

func filterByWindow(samples []bookSample, startMs int64, endMs int64, pick func(bookSample) int) []int {
	out := make([]int, 0, len(samples))
	for _, s := range samples {
		if s.tsMs >= startMs && s.tsMs <= endMs {
			out = append(out, pick(s))
		}
	}
	return out
}

func percentile(values []int, pct float64) int {
	if len(values) == 0 {
		return 0
	}
	cp := append([]int{}, values...)
	sort.Ints(cp)
	if pct <= 0 {
		return cp[0]
	}
	if pct >= 1 {
		return cp[len(cp)-1]
	}
	pos := int(math.Ceil(float64(len(cp))*pct)) - 1
	if pos < 0 {
		pos = 0
	}
	if pos >= len(cp) {
		pos = len(cp) - 1
	}
	return cp[pos]
}

func midPrice(bid string, ask string) string {
	bidRat, ok := new(big.Rat).SetString(bid)
	if !ok {
		return ""
	}
	askRat, ok := new(big.Rat).SetString(ask)
	if !ok {
		return ""
	}
	mid := new(big.Rat).Quo(new(big.Rat).Add(bidRat, askRat), big.NewRat(2, 1))
	return mid.FloatString(8)
}

func hashBook(sample bookSample) string {
	payload := map[string]any{
		"ts_ms":  sample.tsMs,
		"bid":    sample.bid,
		"ask":    sample.ask,
		"bid_qty": sample.bidQty,
		"ask_qty": sample.askQty,
	}
	return hashFromAny(payload)
}

func hashFromAny(payload map[string]any) string {
	b, err := hash.CanonicalJSON(payload)
	if err != nil {
		return ""
	}
	return hash.HashSHA256Hex(b)
}
