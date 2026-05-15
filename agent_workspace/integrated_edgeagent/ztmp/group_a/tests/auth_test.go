package auth

import (
	"testing"
)

func TestGeneratePasswordHash(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantErr  bool
	}{
		{"valid password", "password123", false},
		{"secure password", "SecurePass123", false},
		{"empty password", "", true},
		{"too short", "short", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hash, err := GeneratePasswordHash(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("GeneratePasswordHash() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr && len(hash) == 0 {
				t.Error("GeneratePasswordHash() returned empty hash")
			}
		})
	}

	t.Run("different passwords produce different hashes", func(t *testing.T) {
		hash1, _ := GeneratePasswordHash("pass123")
		hash2, _ := GeneratePasswordHash("pass456")
		if hash1 == hash2 {
			t.Error("different passwords produced identical hashes")
		}
	})
}

func TestValidatePassword(t *testing.T) {
	validHash, _ := GeneratePasswordHash("password123")
	cases := []struct {
		name     string
		plain    string
		hash     string
		want     bool
	}{
		{"valid match", "password123", validHash, true},
		{"invalid password", "wrongpass", validHash, false},
		{"empty plain", "", validHash, false},
		{"empty hash", "password123", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidatePassword(tc.plain, tc.hash); got != tc.want {
				t.Errorf("ValidatePassword() = %v, want %v", got, tc.want)
			}
		})
	}
}
