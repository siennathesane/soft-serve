package models

import (
	"testing"
)

func TestMergeRequestStateString(t *testing.T) {
	tests := []struct {
		state    MergeRequestState
		expected string
	}{
		{MergeRequestStateOpen, "open"},
		{MergeRequestStateMerged, "merged"},
		{MergeRequestStateClosed, "closed"},
		{MergeRequestState(-1), "unknown"},
		{MergeRequestState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("MergeRequestState.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}
