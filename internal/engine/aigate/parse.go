package aigate

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/contracts"
	"github.com/RodrigoBeloyanis/livespot/internal/domain/reasoncodes"
)

type ModelResponse struct {
	Verdict          contracts.AIGateVerdict `json:"verdict"`
	Reasons          []string                `json:"reasons"`
	ModifiedDecision *contracts.Decision     `json:"modified_decision,omitempty"`
}

var (
	ErrSchemaInvalid  = errors.New("schema invalid")
	ErrReasonsInvalid = errors.New("reasons invalid")
)

func ParseModelResponse(raw string) (contracts.AIGateVerdict, []reasoncodes.ReasonCode, *contracts.Decision, error) {
	var parsed ModelResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return "", nil, nil, fmt.Errorf("%w: %v", ErrSchemaInvalid, err)
	}
	if parsed.Verdict != contracts.AIGateAllow && parsed.Verdict != contracts.AIGateBlock && parsed.Verdict != contracts.AIGateModify {
		return "", nil, nil, ErrSchemaInvalid
	}
	if len(parsed.Reasons) == 0 {
		return "", nil, nil, ErrSchemaInvalid
	}
	reasons := make([]reasoncodes.ReasonCode, 0, len(parsed.Reasons))
	for _, r := range parsed.Reasons {
		reasons = append(reasons, reasoncodes.ReasonCode(r))
	}
	if !reasoncodes.ValidateList(reasons) {
		return "", nil, nil, ErrReasonsInvalid
	}
	if parsed.Verdict == contracts.AIGateModify && parsed.ModifiedDecision == nil {
		return "", nil, nil, ErrSchemaInvalid
	}
	return parsed.Verdict, reasons, parsed.ModifiedDecision, nil
}
