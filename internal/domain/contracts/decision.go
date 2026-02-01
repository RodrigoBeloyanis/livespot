package contracts

import (
	"fmt"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/hash"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
	"github.com/RodrigoBeloyanis/livespot/internal/observability"
)

type Side string

type Intent string

type EntryKind string

type TimeInForce string

type FallbackKind string

type ProtectionKind string

type TrailingMode string

type QuantizationPolicy string

type AIGateVerdict string

type RiskVerdictType string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"

	IntentEntry  Intent = "ENTRY"
	IntentExit   Intent = "EXIT"
	IntentManage Intent = "MANAGE"

	EntryMakerFirst EntryKind = "MAKER_FIRST"
	EntryTaker      EntryKind = "TAKER"
	EntryMarket     EntryKind = "MARKET"

	TIFGTC TimeInForce = "GTC"
	TIFIOC TimeInForce = "IOC"
	TIFFOK TimeInForce = "FOK"

	FallbackCancelReplace   FallbackKind = "CANCEL_AND_REPLACE"
	FallbackMarketIfAllowed FallbackKind = "MARKET_IF_ALLOWED"
	FallbackIOCLimit        FallbackKind = "IOC_LIMIT"

	ProtectionOCO          ProtectionKind = "OCO"
	ProtectionTPSLSeparate ProtectionKind = "TP_SL_SEPARATE"

	TrailingOff     TrailingMode = "OFF"
	TrailingVirtual TrailingMode = "VIRTUAL"
	TrailingNative  TrailingMode = "NATIVE"

	QuantizationEnforced         QuantizationPolicy = "ENFORCED"
	QuantizationKnownNonEnforced QuantizationPolicy = "KNOWN_NON_ENFORCED"
	QuantizationUnknown          QuantizationPolicy = "UNKNOWN"

	AIGateAllow  AIGateVerdict = "ALLOW"
	AIGateBlock  AIGateVerdict = "BLOCK"
	AIGateModify AIGateVerdict = "MODIFY"
	AIGateError  AIGateVerdict = "ERROR"

	RiskAllow RiskVerdictType = "ALLOW"
	RiskBlock RiskVerdictType = "BLOCK"
)

type Decision struct {
	Mode            string                   `json:"mode"`
	TsMs            int64                    `json:"ts_ms"`
	Symbol          string                   `json:"symbol"`
	Side            Side                     `json:"side"`
	Intent          Intent                   `json:"intent"`
	EntryPlan       *EntryPlan               `json:"entry_plan"`
	ExitPlan        *ExitPlan                `json:"exit_plan"`
	EdgeScoreX10000 int                      `json:"edge_score_x10000"`
	EdgeBpsExpected int                      `json:"edge_bps_expected"`
	Reasons         []reasoncodes.ReasonCode `json:"reasons"`
	SnapshotID      string                   `json:"snapshot_id"`
	DecisionID      string                   `json:"decision_id"`
	CycleID         string                   `json:"cycle_id"`
	Stage           observability.StageName  `json:"stage"`
	Constraints     DecisionConstraints      `json:"constraints"`
	AIGate          *AIGateResult            `json:"ai_gate"`
	RiskVerdict     *RiskVerdict             `json:"risk_verdict"`
}

type DecisionConstraints struct {
	TickSize           string             `json:"tick_size"`
	StepSize           string             `json:"step_size"`
	MinQty             string             `json:"min_qty"`
	MinNotional        string             `json:"min_notional"`
	PricePrecision     int                `json:"price_precision"`
	QtyPrecision       int                `json:"qty_precision"`
	MaxQty             string             `json:"max_qty"`
	MaxNumOrders       int                `json:"max_num_orders"`
	MaxAlgoOrders      int                `json:"max_algo_orders"`
	MaxNotional        string             `json:"max_notional"`
	QuantizationPolicy QuantizationPolicy `json:"quantization_policy"`
}

type EntryPlan struct {
	Kind          EntryKind    `json:"kind"`
	DesiredPrice  string       `json:"desired_price"`
	LimitPrice    string       `json:"limit_price"`
	Qty           string       `json:"qty"`
	TimeInForce   TimeInForce  `json:"time_in_force"`
	TTLMS         int          `json:"ttl_ms"`
	RepriceMS     int          `json:"reprice_ms"`
	MaxReprices   int          `json:"max_reprices"`
	Fallback      FallbackPlan `json:"fallback"`
	ClientOrderID string       `json:"client_order_id"`
}

type FallbackPlan struct {
	Enabled        bool         `json:"enabled"`
	Kind           FallbackKind `json:"kind"`
	MaxSlippageBps int          `json:"max_slippage_bps"`
	DeadlineMS     int          `json:"deadline_ms"`
}

type ExitPlan struct {
	TPPrice              string         `json:"tp_price"`
	SLPrice              string         `json:"sl_price"`
	ProtectionKind       ProtectionKind `json:"protection_kind"`
	TrailingMode         TrailingMode   `json:"trailing_mode"`
	TrailingTriggerPrice string         `json:"trailing_trigger_price"`
	TrailingDeltaBips    int            `json:"trailing_delta_bips"`
	ClientOrderIDTP      string         `json:"client_order_id_tp"`
	ClientOrderIDSL      string         `json:"client_order_id_sl"`
}

type AIGateResult struct {
	Enabled          bool                     `json:"enabled"`
	Verdict          AIGateVerdict            `json:"verdict"`
	Reasons          []reasoncodes.ReasonCode `json:"reasons"`
	Model            string                   `json:"model"`
	LatencyMs        int                      `json:"latency_ms"`
	RawHash          string                   `json:"raw_hash"`
	InputHash        string                   `json:"input_hash"`
	SnapshotHash     string                   `json:"snapshot_hash"`
	ModifiedDecision *Decision                `json:"modified_decision"`
}

type RiskVerdict struct {
	Verdict    RiskVerdictType          `json:"verdict"`
	Reasons    []reasoncodes.ReasonCode `json:"reasons"`
	RiskLimits RiskLimitsSnapshot       `json:"risk_limits"`
	Costs      CostsSnapshot            `json:"costs"`
}

type RiskLimitsSnapshot struct {
	MaxExposureSymbolUSDT   string `json:"max_exposure_symbol_usdt"`
	MaxExposureTotalUSDT    string `json:"max_exposure_total_usdt"`
	MaxDailyLossUSDT        string `json:"max_daily_loss_usdt"`
	MaxDrawdownUSDT         string `json:"max_drawdown_usdt"`
	MaxOpenOrders           int    `json:"max_open_orders"`
	MaxTradesDay            int    `json:"max_trades_day"`
	MaxTradesWindow         int    `json:"max_trades_window"`
	TradesWindowSeconds     int    `json:"trades_window_seconds"`
	PostStopCooldownSeconds int    `json:"post_stop_cooldown_seconds"`
}

type CostsSnapshot struct {
	MakerFeeBps           int `json:"maker_fee_bps"`
	TakerFeeBps           int `json:"taker_fee_bps"`
	SlippageEntryMakerBps int `json:"slippage_est_entry_maker_bps"`
	SlippageEntryTakerBps int `json:"slippage_est_entry_taker_bps"`
	SlippageExitTakerBps  int `json:"slippage_est_exit_taker_bps"`
	SpreadCurrentBps      int `json:"spread_current_bps"`
	DeltaSpreadBpsP90_10s int `json:"delta_spread_bps_p90_10s"`
}

type DecisionHashPayload struct {
	CycleID         string                  `json:"cycle_id"`
	Stage           observability.StageName `json:"stage"`
	TsMs            int64                   `json:"ts_ms"`
	Symbol          string                  `json:"symbol"`
	Side            Side                    `json:"side"`
	Intent          Intent                  `json:"intent"`
	EdgeScoreX10000 int                     `json:"edge_score_x10000"`
	EdgeBpsExpected int                     `json:"edge_bps_expected"`
	SnapshotHash    string                  `json:"snapshot_hash"`
	Constraints     DecisionConstraints     `json:"constraints"`
	EntryPlan       *EntryPlan              `json:"entry_plan"`
	ExitPlan        *ExitPlan               `json:"exit_plan"`
}

func (d Decision) Validate() error {
	if d.Mode != "LIVE" {
		return fmt.Errorf("decision mode must be LIVE")
	}
	if d.TsMs <= 0 {
		return fmt.Errorf("decision ts_ms missing")
	}
	if d.Symbol == "" {
		return fmt.Errorf("decision symbol missing")
	}
	if d.Side != SideBuy && d.Side != SideSell {
		return fmt.Errorf("decision side invalid")
	}
	if d.Intent != IntentEntry && d.Intent != IntentExit && d.Intent != IntentManage {
		return fmt.Errorf("decision intent invalid")
	}
	if d.EdgeScoreX10000 < 0 || d.EdgeScoreX10000 > 10000 {
		return fmt.Errorf("decision edge_score_x10000 invalid")
	}
	if d.SnapshotID == "" {
		return fmt.Errorf("decision snapshot_id missing")
	}
	if d.DecisionID == "" {
		return fmt.Errorf("decision decision_id missing")
	}
	if d.CycleID == "" {
		return fmt.Errorf("decision cycle_id missing")
	}
	if !observability.IsValidStage(d.Stage) {
		return fmt.Errorf("decision stage invalid")
	}
	if d.Reasons == nil {
		return fmt.Errorf("decision reasons missing")
	}
	if !reasoncodes.ValidateList(d.Reasons) {
		return fmt.Errorf("decision reasons invalid")
	}
	if d.Intent == IntentEntry && len(d.Reasons) == 0 {
		return fmt.Errorf("decision reasons empty for entry")
	}
	if err := d.Constraints.Validate(); err != nil {
		return err
	}
	if d.Intent == IntentEntry {
		if d.EntryPlan == nil {
			return fmt.Errorf("decision entry_plan missing")
		}
		if d.ExitPlan == nil {
			return fmt.Errorf("decision exit_plan missing")
		}
	}
	if d.Intent == IntentManage && d.ExitPlan == nil {
		return fmt.Errorf("decision exit_plan missing for manage")
	}
	if d.EntryPlan != nil {
		if err := d.EntryPlan.Validate(); err != nil {
			return err
		}
	}
	if d.ExitPlan != nil {
		entryPrice := ""
		if d.EntryPlan != nil {
			if d.EntryPlan.LimitPrice != "" {
				entryPrice = d.EntryPlan.LimitPrice
			} else {
				entryPrice = d.EntryPlan.DesiredPrice
			}
		}
		if err := d.ExitPlan.Validate(d.Side, entryPrice); err != nil {
			return err
		}
	}
	if d.AIGate != nil {
		if err := d.AIGate.Validate(); err != nil {
			return err
		}
	}
	if d.RiskVerdict != nil {
		if err := d.RiskVerdict.Validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c DecisionConstraints) Validate() error {
	if !isDecimalString(c.TickSize) {
		return fmt.Errorf("constraints tick_size invalid")
	}
	if !isDecimalString(c.StepSize) {
		return fmt.Errorf("constraints step_size invalid")
	}
	if !isDecimalString(c.MinQty) {
		return fmt.Errorf("constraints min_qty invalid")
	}
	if !isDecimalString(c.MinNotional) {
		return fmt.Errorf("constraints min_notional invalid")
	}
	if c.PricePrecision < 0 || c.QtyPrecision < 0 {
		return fmt.Errorf("constraints precision invalid")
	}
	if !isDecimalString(c.MaxQty) {
		return fmt.Errorf("constraints max_qty invalid")
	}
	if c.MaxNumOrders < 0 || c.MaxAlgoOrders < 0 {
		return fmt.Errorf("constraints max orders invalid")
	}
	if c.MaxNotional != "" && !isDecimalString(c.MaxNotional) {
		return fmt.Errorf("constraints max_notional invalid")
	}
	if c.QuantizationPolicy != QuantizationEnforced && c.QuantizationPolicy != QuantizationKnownNonEnforced && c.QuantizationPolicy != QuantizationUnknown {
		return fmt.Errorf("constraints quantization_policy invalid")
	}
	return nil
}

func (p EntryPlan) Validate() error {
	if p.Kind != EntryMakerFirst && p.Kind != EntryTaker && p.Kind != EntryMarket {
		return fmt.Errorf("entry_plan kind invalid")
	}
	if !isDecimalString(p.DesiredPrice) {
		return fmt.Errorf("entry_plan desired_price invalid")
	}
	if !isDecimalString(p.LimitPrice) {
		return fmt.Errorf("entry_plan limit_price invalid")
	}
	if !isDecimalString(p.Qty) {
		return fmt.Errorf("entry_plan qty invalid")
	}
	if p.TimeInForce != TIFGTC && p.TimeInForce != TIFIOC && p.TimeInForce != TIFFOK {
		return fmt.Errorf("entry_plan time_in_force invalid")
	}
	if p.TTLMS < 0 || p.RepriceMS < 0 || p.MaxReprices < 0 {
		return fmt.Errorf("entry_plan ttl/reprice invalid")
	}
	if p.Kind == EntryMakerFirst {
		if p.TTLMS <= 0 || p.MaxReprices < 1 {
			return fmt.Errorf("entry_plan maker-first requires ttl and reprices")
		}
		if p.TimeInForce != TIFGTC {
			return fmt.Errorf("entry_plan maker-first requires GTC")
		}
	}
	if p.ClientOrderID == "" || len(p.ClientOrderID) != 36 {
		return fmt.Errorf("entry_plan client_order_id invalid")
	}
	if err := p.Fallback.Validate(); err != nil {
		return err
	}
	return nil
}

func (f FallbackPlan) Validate() error {
	if !f.Enabled {
		return nil
	}
	if f.Kind != FallbackCancelReplace && f.Kind != FallbackMarketIfAllowed && f.Kind != FallbackIOCLimit {
		return fmt.Errorf("fallback kind invalid")
	}
	if f.MaxSlippageBps <= 0 {
		return fmt.Errorf("fallback max_slippage_bps invalid")
	}
	if f.DeadlineMS <= 0 {
		return fmt.Errorf("fallback deadline_ms invalid")
	}
	return nil
}

func (p ExitPlan) Validate(side Side, entryPrice string) error {
	if !isDecimalString(p.TPPrice) {
		return fmt.Errorf("exit_plan tp_price invalid")
	}
	if !isDecimalString(p.SLPrice) {
		return fmt.Errorf("exit_plan sl_price invalid")
	}
	if p.ProtectionKind != ProtectionOCO && p.ProtectionKind != ProtectionTPSLSeparate {
		return fmt.Errorf("exit_plan protection_kind invalid")
	}
	if p.TrailingMode != TrailingOff && p.TrailingMode != TrailingVirtual && p.TrailingMode != TrailingNative {
		return fmt.Errorf("exit_plan trailing_mode invalid")
	}
	if p.ClientOrderIDTP == "" || len(p.ClientOrderIDTP) != 36 {
		return fmt.Errorf("exit_plan client_order_id_tp invalid")
	}
	if p.ClientOrderIDSL == "" || len(p.ClientOrderIDSL) != 36 {
		return fmt.Errorf("exit_plan client_order_id_sl invalid")
	}
	if p.TrailingMode == TrailingOff {
		if p.TrailingTriggerPrice != "" && p.TrailingTriggerPrice != "0" {
			return fmt.Errorf("exit_plan trailing_trigger_price must be empty or 0 when trailing off")
		}
		if p.TrailingDeltaBips != 0 {
			return fmt.Errorf("exit_plan trailing_delta_bips must be 0 when trailing off")
		}
	}
	if p.TrailingMode == TrailingNative && p.TrailingDeltaBips < 1 {
		return fmt.Errorf("exit_plan trailing_delta_bips invalid for native")
	}
	if entryPrice != "" {
		if !isDecimalString(entryPrice) {
			return fmt.Errorf("exit_plan entry_price invalid")
		}
		cmpTP, err := compareDecimal(p.TPPrice, entryPrice)
		if err != nil {
			return fmt.Errorf("exit_plan tp_price invalid")
		}
		cmpSL, err := compareDecimal(p.SLPrice, entryPrice)
		if err != nil {
			return fmt.Errorf("exit_plan sl_price invalid")
		}
		if side == SideBuy {
			if cmpSL >= 0 || cmpTP <= 0 {
				return fmt.Errorf("exit_plan sl/tp invalid for buy")
			}
		}
		if side == SideSell {
			if cmpSL <= 0 || cmpTP >= 0 {
				return fmt.Errorf("exit_plan sl/tp invalid for sell")
			}
		}
	}
	return nil
}

func (a AIGateResult) Validate() error {
	if !a.Enabled {
		if a.Verdict != AIGateAllow {
			return fmt.Errorf("ai_gate verdict must be ALLOW when disabled")
		}
		if a.ModifiedDecision != nil {
			return fmt.Errorf("ai_gate modified_decision must be empty when disabled")
		}
		return nil
	}
	if a.Verdict != AIGateAllow && a.Verdict != AIGateBlock && a.Verdict != AIGateModify && a.Verdict != AIGateError {
		return fmt.Errorf("ai_gate verdict invalid")
	}
	if len(a.Reasons) == 0 {
		return fmt.Errorf("ai_gate reasons missing")
	}
	if !reasoncodes.ValidateList(a.Reasons) {
		return fmt.Errorf("ai_gate reasons invalid")
	}
	if a.InputHash == "" || a.SnapshotHash == "" {
		return fmt.Errorf("ai_gate hashes missing")
	}
	if !isLowerHex64(a.InputHash) || !isLowerHex64(a.SnapshotHash) {
		return fmt.Errorf("ai_gate hashes invalid")
	}
	if a.RawHash == "" || !isLowerHex64(a.RawHash) {
		return fmt.Errorf("ai_gate raw_hash invalid")
	}
	if a.Verdict == AIGateModify && a.ModifiedDecision == nil {
		return fmt.Errorf("ai_gate modified_decision missing")
	}
	if (a.Verdict == AIGateAllow || a.Verdict == AIGateBlock || a.Verdict == AIGateError) && a.ModifiedDecision != nil {
		return fmt.Errorf("ai_gate modified_decision invalid")
	}
	return nil
}

func (r RiskVerdict) Validate() error {
	if r.Verdict != RiskAllow && r.Verdict != RiskBlock {
		return fmt.Errorf("risk_verdict invalid")
	}
	if r.Reasons == nil {
		return fmt.Errorf("risk_verdict reasons missing")
	}
	if !reasoncodes.ValidateList(r.Reasons) {
		return fmt.Errorf("risk_verdict reasons invalid")
	}
	if r.Verdict == RiskBlock && len(r.Reasons) == 0 {
		return fmt.Errorf("risk_verdict reasons empty for block")
	}
	if err := r.RiskLimits.Validate(); err != nil {
		return err
	}
	if err := r.Costs.Validate(); err != nil {
		return err
	}
	return nil
}

func (r RiskLimitsSnapshot) Validate() error {
	if !isDecimalString(r.MaxExposureSymbolUSDT) || !isDecimalString(r.MaxExposureTotalUSDT) || !isDecimalString(r.MaxDailyLossUSDT) || !isDecimalString(r.MaxDrawdownUSDT) {
		return fmt.Errorf("risk_limits decimals invalid")
	}
	if r.MaxOpenOrders < 0 || r.MaxTradesDay < 0 || r.MaxTradesWindow < 0 || r.TradesWindowSeconds < 0 || r.PostStopCooldownSeconds < 0 {
		return fmt.Errorf("risk_limits integers invalid")
	}
	return nil
}

func (c CostsSnapshot) Validate() error {
	if c.MakerFeeBps < 0 || c.TakerFeeBps < 0 || c.SlippageEntryMakerBps < 0 || c.SlippageEntryTakerBps < 0 || c.SlippageExitTakerBps < 0 || c.SpreadCurrentBps < 0 || c.DeltaSpreadBpsP90_10s < 0 {
		return fmt.Errorf("costs invalid")
	}
	return nil
}

func (d Decision) HashPayload(snapshotHash string) (DecisionHashPayload, error) {
	if snapshotHash == "" {
		return DecisionHashPayload{}, fmt.Errorf("snapshot_hash missing")
	}
	payload := DecisionHashPayload{
		CycleID:         d.CycleID,
		Stage:           d.Stage,
		TsMs:            d.TsMs,
		Symbol:          d.Symbol,
		Side:            d.Side,
		Intent:          d.Intent,
		EdgeScoreX10000: d.EdgeScoreX10000,
		EdgeBpsExpected: d.EdgeBpsExpected,
		SnapshotHash:    snapshotHash,
		Constraints:     d.Constraints,
		EntryPlan:       d.EntryPlan,
		ExitPlan:        d.ExitPlan,
	}
	return payload, nil
}

func (d Decision) Hash(snapshotHash string) (string, error) {
	payload, err := d.HashPayload(snapshotHash)
	if err != nil {
		return "", err
	}
	return hash.CanonicalHash(payload)
}
