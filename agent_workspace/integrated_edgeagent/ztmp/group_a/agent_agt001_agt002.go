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

var dangerousPatterns = []string{
	"import os",
	"os.system",
	"os.popen",
	"subprocess",
	"__import__",
	"eval(",
	"exec(",
	"open(",
	"rm -rf",
	"shutil.rmtree",
}

func SanitizeAgentCode(code string) (string, error) {
	codeLower := strings.ToLower(code)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(codeLower, pattern) {
			return "", errors.New("code contains dangerous patterns")
		}
	}
	return code, nil
}
