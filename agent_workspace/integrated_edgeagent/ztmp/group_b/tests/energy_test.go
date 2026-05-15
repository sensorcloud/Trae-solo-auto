package energy

import (
	"testing"
)

func TestCalculateCarbonIntensity(t *testing.T) {
	testCases := []struct {
		name     string
		source   string
		expected int
	}{
		{"solar", "solar", 15},
		{"wind", "wind", 12},
		{"gas", "gas", 450},
		{"storage", "storage", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if res := CalculateCarbonIntensity(tc.source); res != tc.expected {
				t.Errorf("CalculateCarbonIntensity(%s) = %d, want %d", tc.source, res, tc.expected)
			}
		})
	}
}

func TestIsValidPowerSource(t *testing.T) {
	testCases := []struct {
		name     string
		source   string
		expected bool
	}{
		{"valid solar", "solar", true},
		{"valid wind", "wind", true},
		{"valid gas", "gas", true},
		{"valid storage", "storage", true},
		{"invalid source", "invalid", false},
		{"empty", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsValidPowerSource(tc.source); got != tc.expected {
				t.Errorf("IsValidPowerSource(%s) = %v, want %v", tc.source, got, tc.expected)
			}
		})
	}
}
