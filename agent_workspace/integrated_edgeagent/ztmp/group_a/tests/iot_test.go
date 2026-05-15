package iot

import (
	"testing"
)

func TestValidateDeviceID(t *testing.T) {
	cases := []struct {
		name     string
		deviceID string
		want     bool
	}{
		{"valid hyphen", "DEV-001", true},
		{"valid underscore", "device_123", true},
		{"empty", "", false},
		{"invalid special", "too!invalid", false},
		{"too long", "averylongdevicenamethatexceeds64charactersxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidateDeviceID(tc.deviceID); got != tc.want {
				t.Errorf("ValidateDeviceID() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestValidateTelemetryValue(t *testing.T) {
	cases := []struct {
		name  string
		value float64
		min   float64
		max   float64
		want  bool
	}{
		{"in range", 25.5, -50.0, 100.0, true},
		{"above max", 101.0, -50.0, 100.0, false},
		{"below min", -60.0, -50.0, 100.0, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidateTelemetryValue(tc.value, tc.min, tc.max); got != tc.want {
				t.Errorf("ValidateTelemetryValue() = %v, want %v", got, tc.want)
			}
		})
	}
}
