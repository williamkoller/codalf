package types

import "testing"

func TestSeverity_IsValid(t *testing.T) {
	tests := []struct {
		s   Severity
		exp bool
	}{
		{SeverityCritical, true},
		{SeverityWarning, true},
		{SeverityInfo, true},
		{Severity("invalid"), false},
		{Severity(""), false},
	}

	for _, tc := range tests {
		t.Run(string(tc.s), func(t *testing.T) {
			if got := tc.s.IsValid(); got != tc.exp {
				t.Errorf("IsValid() = %v, want %v", got, tc.exp)
			}
		})
	}
}
