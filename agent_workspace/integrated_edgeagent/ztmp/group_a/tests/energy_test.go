package energy

import (
	"testing"
)

func TestCalculateCarbonIntensity(t *testing.T) {
	cases := []struct {
		name        string
		powerSource string
		want        int
	}{
		{"solar", "solar", 15},
		{"wind", "wind", 12},
		{"gas", "gas", 450},
		{"storage", "storage", 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CalculateCarbonIntensity(tc.powerSource); got != tc.want {
				t.Errorf("CalculateCarbonIntensity() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsValidPowerSource(t *testing.T) {
	cases := []struct {
		name   string
		source string
		want   bool
	}{
		{"valid solar", "solar", true},
		{"valid wind", "wind", true},
		{"valid gas", "gas", true},
		{"valid storage", "storage", true},
		{"invalid source", "invalid", false},
		{"empty source", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsValidPowerSource(tc.source); got != tc.want {
				t.Errorf("IsValidPowerSource() = %v, want %v", got, tc.want)
			}
		})
	}
}
