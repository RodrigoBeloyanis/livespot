package audit

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/RodrigoBeloyanis/livespot/internal/domain/hash"
)

type RedactionPolicy struct {
	RemoveKeys            []string
	VolatileKeys          []string
	RejectKeySubstrings   []string
	RejectValueSubstrings []string
}

func DefaultRedactionPolicy() RedactionPolicy {
	return RedactionPolicy{
		RemoveKeys: []string{
			"authorization",
			"x-mbx-apikey",
			"cookie",
			"signature",
			"binance_api_key",
			"binance_api_secret",
			"openai_api_key",
		},
		VolatileKeys: []string{
			"timestamp",
			"ts",
			"nonce",
			"sequence",
			"seq",
			"temp_id",
			"temporary_id",
			"random",
		},
		RejectKeySubstrings: []string{
			"token",
			"secret",
			"apikey",
			"api_key",
			"password",
		},
		RejectValueSubstrings: []string{
			"BINANCE_API_KEY",
			"BINANCE_API_SECRET",
			"OPENAI_API_KEY",
		},
	}
}

func RedactAndTruncateJSON(raw []byte, maxBytes int, policy RedactionPolicy) (string, error) {
	if maxBytes <= 0 {
		return "", fmt.Errorf("audit_redacted_json_max_bytes must be > 0")
	}
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return "", fmt.Errorf("redact json parse: %w", err)
	}
	cleaned, err := sanitizeValue(payload, policy)
	if err != nil {
		return "", err
	}
	buf, err := hash.CanonicalJSON(cleaned)
	if err != nil {
		return "", fmt.Errorf("redact canonical json: %w", err)
	}
	if len(buf) > maxBytes {
		return fmt.Sprintf("<TRUNCATED len_bytes=%d>", len(buf)), nil
	}
	return string(buf), nil
}

func sanitizeValue(value any, policy RedactionPolicy) (any, error) {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any)
		for key, val := range v {
			if isExactMatch(key, policy.RemoveKeys) {
				continue
			}
			if isExactMatch(key, policy.VolatileKeys) {
				continue
			}
			if hasSubstring(key, policy.RejectKeySubstrings) {
				return nil, fmt.Errorf("redact rejected key: %s", key)
			}
			clean, err := sanitizeValue(val, policy)
			if err != nil {
				return nil, err
			}
			out[key] = clean
		}
		return out, nil
	case []any:
		out := make([]any, 0, len(v))
		for _, item := range v {
			clean, err := sanitizeValue(item, policy)
			if err != nil {
				return nil, err
			}
			out = append(out, clean)
		}
		return out, nil
	case string:
		if hasSubstring(v, policy.RejectValueSubstrings) {
			return nil, fmt.Errorf("redact rejected value")
		}
		return v, nil
	default:
		return value, nil
	}
}

func isExactMatch(value string, set []string) bool {
	value = strings.ToLower(value)
	for _, item := range set {
		if value == strings.ToLower(item) {
			return true
		}
	}
	return false
}

func hasSubstring(value string, needles []string) bool {
	value = strings.ToLower(value)
	for _, needle := range needles {
		if strings.Contains(value, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}
