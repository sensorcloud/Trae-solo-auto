package iot

import (
	"testing"
)

func TestValidateDeviceID(t *testing.T) {
	testCases := []struct {
		name     string
		deviceID string
		expected bool
	}{
		{"valid hyphenated", "DEV-001", true},
		{"valid underscore", "device_123", true},
		{"empty string", "", false},
		{"invalid chars", "too!invalid", false},
		{"too long", "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklm", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidateDeviceID(tc.deviceID); got != tc.expected {
				t.Errorf("ValidateDeviceID() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestValidateTelemetryValue(t *testing.T) {
	testCases := []struct {
		name  string
		value float64
		min   float64
		max   float64
		want  bool
	}{
		{"within range", 25.5, -50.0, 100.0, true},
		{"above max", 101.0, -50.0, 100.0, false},
		{"below min", -51.0, -50.0, 100.0, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidateTelemetryValue(tc.value, tc.min, tc.max); got != tc.want {
				t.Errorf("ValidateTelemetryValue() = %v, want %v", got, tc.want)
			}
		})
	}
}
