package agent

import (
	"errors"
	"regexp"
	"strings"
)

var validNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,50}$`)

func ValidateAgentName(name string) error {
	if name == "" {
		return errors.New("agent name cannot be empty")
	}
	if len(name) > 50 {
		return errors.New("agent name exceeds maximum length of 50 characters")
	}
	if !validNameRegex.MatchString(name) {
		return errors.New("agent name contains invalid characters")
	}
	return nil
}

var forbiddenPatterns = []string{
	"os.system",
	"os.popen",
	"subprocess",
	"__import__",
	"eval",
	"exec",
	"open(",
	"shutil",
	"rm -rf",
}

func SanitizeAgentCode(code string) (string, error) {
	for _, pattern := range forbiddenPatterns {
		if strings.Contains(strings.ToLower(code), pattern) {
			return "", errors.New("code contains potentially harmful patterns")
		}
	}
	return code, nil
}
