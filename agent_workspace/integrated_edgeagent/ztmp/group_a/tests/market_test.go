package market

import (
	"testing"
)

func TestCalculatePricePerHour(t *testing.T) {
	cases := []struct {
		name     string
		gpuCount int
		gpuModel string
		want     float64
	}{
		{"1xA100", 1, "A100", 8.5},
		{"4xA100", 4, "A100", 34.0},
		{"8xH100", 8, "H100", 120.0},
		{"2xV100", 2, "V100", 10.0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CalculatePricePerHour(tc.gpuCount, tc.gpuModel); got != tc.want {
				t.Errorf("CalculatePricePerHour() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestValidateGPUCount(t *testing.T) {
	cases := []struct {
		name  string
		count int
		want  bool
	}{
		{"1 GPU", 1, true},
		{"8 GPUs", 8, true},
		{"0 GPUs", 0, false},
		{"negative GPUs", -1, false},
		{"too many GPUs", 100, false},
		{"max 64", 64, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidateGPUCount(tc.count); got != tc.want {
				t.Errorf("ValidateGPUCount() = %v, want %v", got, tc.want)
			}
		})
	}
}
