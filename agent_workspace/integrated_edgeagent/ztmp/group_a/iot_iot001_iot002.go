package iot

import (
	"regexp"
)

var validDevicePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

func ValidateDeviceID(deviceID string) bool {
	if len(deviceID) == 0 || len(deviceID) > 64 {
		return false
	}
	return validDevicePattern.MatchString(deviceID)
}

func ValidateTelemetryValue(value, min, max float64) bool {
	if value < min || value > max {
		return false
	}
	return true
}
