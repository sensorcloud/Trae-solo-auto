package monitor

import (
	"testing"
)

func TestIsValidMetricName(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"valid underscore", "cpu_usage", true},
		{"valid dot", "memory.usage", true},
		{"empty", "", false},
		{"too long", "averylongmetricnamethatexceeds100charactersxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsValidMetricName(tc.input); got != tc.want {
				t.Errorf("IsValidMetricName() = %v, want %v", got, tc.want)
			}
		})
	}
}
