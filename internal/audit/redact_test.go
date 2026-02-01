package audit

import (
	"strings"
	"testing"
)

func TestRedactAndTruncateJSONRemovesKeys(t *testing.T) {
	policy := DefaultRedactionPolicy()
	raw := []byte(`{"Authorization":"Bearer token","nonce":123,"keep":true}`)
	out, err := RedactAndTruncateJSON(raw, 1024, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "Authorization") || strings.Contains(out, "nonce") {
		t.Fatalf("expected removed keys, got %s", out)
	}
	if !strings.Contains(out, "keep") {
		t.Fatalf("expected keep field")
	}
}

func TestRedactAndTruncateJSONRejectsKey(t *testing.T) {
	policy := DefaultRedactionPolicy()
	raw := []byte(`{"api_token":"secret"}`)
	_, err := RedactAndTruncateJSON(raw, 1024, policy)
	if err == nil {
		t.Fatalf("expected rejection for key")
	}
}

func TestRedactAndTruncateJSONTruncates(t *testing.T) {
	policy := DefaultRedactionPolicy()
	raw := []byte(`{"keep":"` + strings.Repeat("a", 50) + `"}`)
	out, err := RedactAndTruncateJSON(raw, 10, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(out, "<TRUNCATED len_bytes=") {
		t.Fatalf("expected truncation, got %s", out)
	}
}
