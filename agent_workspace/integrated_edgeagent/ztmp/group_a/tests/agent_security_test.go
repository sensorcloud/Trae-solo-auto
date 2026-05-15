package agent

import (
	"testing"
)

func TestSanitizeAgentCodeEdgeCases(t *testing.T) {
	testCases := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{"safe code", "safe_code = 1 + 2", false},
		{"exec with parens", "exec('a')", true},
		{"exec without parens", "exec 'a'", true},
		{"eval with parens", "eval('x')", true},
		{"import os", "import os; print(1)", true},
		{"os.system", "os.system('rm -rf')", true},
		{"subprocess", "import subprocess; subprocess.call(['ls'])", true},
		{"__import__", "__import__('os')", true},
		{"open func", "f = open('file.txt')", true},
		{"shutil", "shutil.rmtree('/tmp')", true},
		{"rm -rf", "import os; os.system('rm -rf /')", true},
		{"safe print", "print('hello world')", false},
		{"safe math", "result = 1 + 2 * 3", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := SanitizeAgentCode(tc.code)
			if (err != nil) != tc.wantErr {
				t.Errorf("SanitizeAgentCode() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr && out != tc.code {
				t.Error("Safe code should not be modified")
			}
		})
	}
}
