package aigate

import "testing"

func TestParseModelResponseValid(t *testing.T) {
	raw := `{"verdict":"ALLOW","reasons":["STRAT_OK"]}`
	_, _, _, err := ParseModelResponse(raw)
	if err != nil {
		t.Fatalf("expected valid response, got %v", err)
	}
}

func TestParseModelResponseInvalidReason(t *testing.T) {
	raw := `{"verdict":"ALLOW","reasons":["UNKNOWN_CODE"]}`
	_, _, _, err := ParseModelResponse(raw)
	if err != ErrReasonsInvalid {
		t.Fatalf("expected ErrReasonsInvalid, got %v", err)
	}
}
