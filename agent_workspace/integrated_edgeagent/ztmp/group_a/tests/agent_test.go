package agent

import (
	"testing"
)

func TestValidateAgentName(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "valid_agent_name", false},
		{"valid with hyphen", "valid-agent-123", false},
		{"invalid special", "bad!name", true},
		{"too long", "toolongnamebeyond50charactersxxxxxxxxxxxxxxxxxxxxxx", true},
		{"empty", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateAgentName(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateAgentName() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestSanitizeAgentCode(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantErr  bool
	}{
		{"safe code", "safe_code = 1 + 2", false},
		{"dangerous os.system", "import os; os.system('rm -rf /')", true},
		{"dangerous subprocess", "__import__('subprocess').check_output(['rm', '-rf', '/'])", true},
		{"dangerous exec", "exec('import os')", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := SanitizeAgentCode(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("SanitizeAgentCode() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr && result != tc.input {
				t.Errorf("SanitizeAgentCode() returned unexpected modification")
			}
		})
	}
}
