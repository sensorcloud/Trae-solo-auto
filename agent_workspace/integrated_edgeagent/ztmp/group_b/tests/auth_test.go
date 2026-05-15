package auth

import (
	"testing"
)

func TestGeneratePasswordHash(t *testing.T) {
	tests := []struct {
		name    string
		pwd     string
		wantErr bool
	}{
		{"valid password 1", "testpassword123", false},
		{"valid password 2", "SecurePassw0rd!", false},
		{"empty password", "", true},
		{"too short", "1234567", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := GeneratePasswordHash(tt.pwd)
			if (err != nil) != tt.wantErr {
				t.Errorf("GeneratePasswordHash() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(hash) == 0 {
				t.Error("GeneratePasswordHash() returned empty hash for valid input")
			}
		})
	}

	t.Run("distinct passwords get distinct hashes", func(t *testing.T) {
		h1, _ := GeneratePasswordHash("pass123")
		h2, _ := GeneratePasswordHash("pass456")
		if h1 == h2 {
			t.Error("different passwords should have different hashes")
		}
	})
}

func TestValidatePassword(t *testing.T) {
	validHash, _ := GeneratePasswordHash("correctpass")
	tests := []struct {
		name      string
		plaintext string
		hash      string
		want      bool
	}{
		{"correct password", "correctpass", validHash, true},
		{"wrong password", "wrongpass", validHash, false},
		{"empty plaintext", "", validHash, false},
		{"empty hash", "correctpass", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidatePassword(tt.plaintext, tt.hash); got != tt.want {
				t.Errorf("ValidatePassword() = %v, want %v", got, tt.want)
			}
		})
	}
}
