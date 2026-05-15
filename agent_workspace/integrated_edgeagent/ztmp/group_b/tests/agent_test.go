package agent

import (
	"testing"
)

func TestValidateAgentName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		wantErr  bool
	}{
		{"valid simple", "myagent", false},
		{"valid with underscore", "my_agent", false},
		{"valid with hyphen", "my-agent-123", false},
		{"invalid exclamation", "my!agent", true},
		{"too long", "thisnameiswaytoolongtobeacceptedbythefunctionxxxxxx", true},
		{"empty", "", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateAgentName(tc.input); (err != nil) != tc.wantErr {
				t.Errorf("ValidateAgentName() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestSanitizeAgentCode(t *testing.T) {
	testCases := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{"safe code", "x = 1 + 2", false},
		{"os.system", "import os; os.system('rm /')", true},
		{"subprocess", "from subprocess import call; call(['rm', '-rf', '/'])", true},
		{"eval", "eval('__import__(\"os\")')", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := SanitizeAgentCode(tc.code)
			if (err != nil) != tc.wantErr {
				t.Errorf("SanitizeAgentCode() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && out != tc.code {
				t.Error("SanitizeAgentCode() should not modify safe code")
			}
		})
	}
}
