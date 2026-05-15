package agent

import (
	"errors"
	"regexp"
	"strings"
)

var validNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,50}$`)

func ValidateAgentName(name string) error {
	if len(name) > 50 {
		return errors.New("agent name too long")
	}
	if !validNamePattern.MatchString(name) {
		return errors.New("invalid agent name format")
	}
	return nil
}

var dangerousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bimport\s+os\b`),
	regexp.MustCompile(`(?i)\bos\.system\b`),
	regexp.MustCompile(`(?i)\bos\.popen\b`),
	regexp.MustCompile(`(?i)\bsubprocess\b`),
	regexp.MustCompile(`(?i)\b__import__\b`),
	regexp.MustCompile(`(?i)\beval\s*\(`),
	regexp.MustCompile(`(?i)\bexec\s*\(`),
	regexp.MustCompile(`(?i)\bopen\s*\(`),
	regexp.MustCompile(`(?i)rm\s+-rf`),
	regexp.MustCompile(`(?i)\bshutil\b`),
	regexp.MustCompile(`(?i)\bos\.remove\b`),
	regexp.MustCompile(`(?i)\bos\.unlink\b`),
}

func SanitizeAgentCode(code string) (string, error) {
	for _, pattern := range dangerousPatterns {
		if pattern.MatchString(code) {
			return "", errors.New("code contains dangerous patterns")
		}
	}
	return code, nil
}
