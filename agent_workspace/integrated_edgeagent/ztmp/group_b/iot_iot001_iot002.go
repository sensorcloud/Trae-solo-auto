package iot

import (
	"regexp"
)

var validIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

func ValidateDeviceID(deviceID string) bool {
	if deviceID == "" || len(deviceID) > 64 {
		return false
	}
	return validIDRegex.MatchString(deviceID)
}

func ValidateTelemetryValue(value, min, max float64) bool {
	if value < min || value > max {
		return false
	}
	return true
}
