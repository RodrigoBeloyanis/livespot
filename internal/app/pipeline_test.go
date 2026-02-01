package app

import "testing"

func TestDefaultStageSequenceValid(t *testing.T) {
	sequence := DefaultStageSequence()
	if err := sequence.Validate(); err != nil {
		t.Fatalf("expected valid sequence: %v", err)
	}
	if len(sequence.Stages) == 0 {
		t.Fatalf("expected stages")
	}
}
