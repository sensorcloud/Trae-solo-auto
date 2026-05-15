package monitor

import (
	"regexp"
)

var validMetricPattern = regexp.MustCompile(`^[a-zA-Z0-9_.]{1,100}$`)

func IsValidMetricName(name string) bool {
	if len(name) == 0 || len(name) > 100 {
		return false
	}
	return validMetricPattern.MatchString(name)
}
