package monitor

import (
	"regexp"
)

var validMetricRegex = regexp.MustCompile(`^[a-zA-Z0-9_.]{1,100}$`)

func IsValidMetricName(name string) bool {
	if len(name) < 1 || len(name) > 100 {
		return false
	}
	return validMetricRegex.MatchString(name)
}
