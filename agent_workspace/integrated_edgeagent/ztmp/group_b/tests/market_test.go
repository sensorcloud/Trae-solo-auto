package market

import (
	"testing"
)

func TestCalculatePricePerHour(t *testing.T) {
	tests := []struct {
		name     string
		gpuCount int
		gpuModel string
		want     float64
	}{
		{"single A100", 1, "A100", 8.5},
		{"four A100", 4, "A100", 34.0},
		{"eight H100", 8, "H100", 120.0},
		{"single V100", 1, "V100", 5.0},
		{"two V100", 2, "V100", 10.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalculatePricePerHour(tt.gpuCount, tt.gpuModel); got != tt.want {
				t.Errorf("CalculatePricePerHour() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateGPUCount(t *testing.T) {
	tests := []struct {
		name  string
		count int
		want  bool
	}{
		{"1 GPU", 1, true},
		{"8 GPUs", 8, true},
		{"64 GPUs", 64, true},
		{"0 GPUs", 0, false},
		{"-1 GPU", -1, false},
		{"100 GPUs", 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateGPUCount(tt.count); got != tt.want {
				t.Errorf("ValidateGPUCount() = %v, want %v", got, tt.want)
			}
		})
	}
}
