package monitor

import (
	"testing"
)

func TestIsValidMetricName(t *testing.T) {
	testCases := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid with underscore", "cpu_usage", true},
		{"valid with dot", "memory.usage", true},
		{"empty", "", false},
		{"too long", "averylongmetricnamethatexceeds100charactersxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsValidMetricName(tc.input); got != tc.want {
				t.Errorf("IsValidMetricName(%s) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}
