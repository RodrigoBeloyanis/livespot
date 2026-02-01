package aigate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/hash"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
	"github.com/RodrigoBeloyanis/livespot/internal/infra/openai"
)

type Gate struct {
	cfg          config.Config
	client       *openai.Client
	recorder     *Recorder
	systemPrompt string
	userTemplate string
	schema       openai.JSONSchema
	now          func() time.Time
}

type CallContext struct {
	RunID          string
	CycleID        string
	ExchangeTimeMs int64
}

func NewGate(cfg config.Config, client *openai.Client, recorder *Recorder, now func() time.Time) (*Gate, error) {
	if now == nil {
		now = time.Now
	}
	systemPrompt, err := loadText("prompts/ai_gate_system.txt")
	if err != nil {
		return nil, err
	}
	userTemplate, err := loadText("prompts/ai_gate_user_template.txt")
	if err != nil {
		return nil, err
	}
	schemaBytes, err := os.ReadFile("prompts/schemas/ai_gate_result.schema.json")
	if err != nil {
		return nil, fmt.Errorf("read ai_gate schema: %w", err)
	}
	return &Gate{
		cfg:          cfg,
		client:       client,
		recorder:     recorder,
		systemPrompt: systemPrompt,
		userTemplate: userTemplate,
		schema: openai.JSONSchema{
			Name:   "ai_gate_result",
			Schema: json.RawMessage(schemaBytes),
			Strict: true,
		},
		now: now,
	}, nil
}

func (g *Gate) Evaluate(ctx context.Context, callCtx CallContext, decision contracts.Decision, snapshot contracts.Snapshot) (contracts.AIGateResult, *contracts.Decision, error) {
	start := g.now()
	snapshotHash, err := snapshot.Hash()
	if err != nil {
		return g.fail(callCtx, decision, snapshotHash, reasoncodes.AIGATE_SCHEMA_INVALID, "snapshot_hash"), nil, err
	}
	payload, err := BuildPayload(decision, snapshot, snapshotHash)
	if err != nil {
		return g.fail(callCtx, decision, snapshotHash, reasoncodes.AIGATE_SCHEMA_INVALID, "payload"), nil, err
	}
	inputHash, err := InputHash(payload)
	if err != nil {
		return g.fail(callCtx, decision, snapshotHash, reasoncodes.AIGATE_SCHEMA_INVALID, "input_hash"), nil, err
	}
	payload.InputHash = inputHash
	payloadJSON, err := hash.CanonicalJSON(payload)
	if err != nil {
		return g.fail(callCtx, decision, snapshotHash, reasoncodes.AIGATE_SCHEMA_INVALID, "payload_json"), nil, err
	}
	userPrompt := strings.ReplaceAll(g.userTemplate, "{{payload_json}}", string(payloadJSON))

	req := openai.ChatCompletionRequest{
		Model: g.cfg.AIGateModel,
		Messages: []openai.Message{
			{Role: "system", Content: g.systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		ResponseFormat: &openai.ResponseFormat{
			Type:       "json_schema",
			JSONSchema: &g.schema,
		},
	}
	temp := 0.0
	req.Temperature = &temp

	ctx, cancel := context.WithTimeout(ctx, time.Duration(g.cfg.AIGateTimeoutMs)*time.Millisecond)
	defer cancel()

	resp, _, err := g.client.ChatCompletion(ctx, req)
	latency := int(g.now().Sub(start).Milliseconds())
	if err != nil {
		reason := reasoncodes.AIGATE_PARSE_FAIL
		if ctx.Err() == context.DeadlineExceeded {
			reason = reasoncodes.AIGATE_TIMEOUT
		}
		result := g.fail(callCtx, decision, snapshotHash, reason, "request_failed")
		result.InputHash = inputHash
		result.LatencyMs = latency
		g.record(callCtx, decision, result, payloadJSON, nil, nil, false, reason, "request_failed")
		return result, nil, err
	}
	if len(resp.Choices) == 0 {
		return g.fail(callCtx, decision, snapshotHash, reasoncodes.AIGATE_PARSE_FAIL, "empty_choices"), nil, fmt.Errorf("ai gate empty response")
	}
	rawContent := resp.Choices[0].Message.Content
	rawHash := hash.HashSHA256Hex([]byte(rawContent))

	verdict, reasons, modifiedDecision, parseErr := ParseModelResponse(rawContent)
	if parseErr != nil {
		reason := reasoncodes.AIGATE_PARSE_FAIL
		if parseErr == ErrSchemaInvalid {
			reason = reasoncodes.AIGATE_SCHEMA_INVALID
		}
		if parseErr == ErrReasonsInvalid {
			reason = reasoncodes.AIGATE_REASON_UNKNOWN
		}
		result := g.fail(callCtx, decision, snapshotHash, reason, "parse_failed")
		result.InputHash = inputHash
		result.RawHash = rawHash
		result.LatencyMs = latency
		g.record(callCtx, decision, result, payloadJSON, []byte(rawContent), nil, false, reason, "parse_failed")
		return result, nil, parseErr
	}

	result := contracts.AIGateResult{
		Enabled:      true,
		Verdict:      verdict,
		Reasons:      reasons,
		Model:        g.cfg.AIGateModel,
		LatencyMs:    latency,
		RawHash:      rawHash,
		InputHash:    inputHash,
		SnapshotHash: snapshotHash,
	}

	var applied *contracts.Decision
	modifyApplied := false
	if verdict == contracts.AIGateModify {
		if modifiedDecision == nil {
			result = g.fail(callCtx, decision, snapshotHash, reasoncodes.AIGATE_SCHEMA_INVALID, "modified_missing")
		} else {
			mod, err := ApplyModify(decision, *modifiedDecision)
			if err != nil {
				result = g.fail(callCtx, decision, snapshotHash, reasoncodes.AIGATE_MODIFY_INVALID, "modify_invalid")
			} else {
				result.ModifiedDecision = &mod
				applied = &mod
				modifyApplied = true
			}
		}
	}

	g.record(callCtx, decision, result, payloadJSON, []byte(rawContent), decisionPatch(decision, applied), modifyApplied, "", "")
	return result, applied, nil
}

func (g *Gate) fail(callCtx CallContext, decision contracts.Decision, snapshotHash string, reason reasoncodes.ReasonCode, detail string) contracts.AIGateResult {
	return contracts.AIGateResult{
		Enabled:          true,
		Verdict:          contracts.AIGateError,
		Reasons:          []reasoncodes.ReasonCode{reason},
		Model:            g.cfg.AIGateModel,
		LatencyMs:        0,
		RawHash:          hash.HashSHA256Hex([]byte{}),
		InputHash:        "",
		SnapshotHash:     snapshotHash,
		ModifiedDecision: nil,
	}
}

func (g *Gate) record(callCtx CallContext, decision contracts.Decision, result contracts.AIGateResult, requestJSON []byte, responseJSON []byte, patch map[string]any, modifyApplied bool, errCode reasoncodes.ReasonCode, errDetail string) {
	if g.recorder == nil {
		return
	}
	errorCode := ""
	if result.Verdict == contracts.AIGateError && len(result.Reasons) > 0 {
		errorCode = string(result.Reasons[0])
	}
	if errCode != "" {
		errorCode = string(errCode)
	}
	evt := Event{
		RunID:                 callCtx.RunID,
		CycleID:               callCtx.CycleID,
		Mode:                  decision.Mode,
		SnapshotID:            decision.SnapshotID,
		SnapshotHash:          result.SnapshotHash,
		DecisionID:            decision.DecisionID,
		InputHash:             result.InputHash,
		Enabled:               result.Enabled,
		Verdict:               string(result.Verdict),
		Reasons:               result.Reasons,
		Model:                 result.Model,
		LatencyMs:             result.LatencyMs,
		RawHash:               result.RawHash,
		RequestJSON:           requestJSON,
		ResponseJSON:          responseJSON,
		ModifiedDecisionPatch: patch,
		ModifyApplied:         modifyApplied,
		ErrorCode:             errorCode,
		ErrorDetailRedacted:   errDetail,
		ExchangeTimeMs:        callCtx.ExchangeTimeMs,
		LocalReceivedMs:       g.now().UnixMilli(),
	}
	_ = g.recorder.Record(context.Background(), evt)
}

func decisionPatch(original contracts.Decision, modified *contracts.Decision) map[string]any {
	if modified == nil {
		return nil
	}
	return BuildDecisionPatch(original, *modified)
}

func loadText(path string) (string, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(buf), nil
}
