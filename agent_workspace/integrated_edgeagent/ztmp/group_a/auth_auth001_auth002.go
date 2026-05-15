package auth

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
)

func GeneratePasswordHash(password string) (string, error) {
	if len(password) < 8 {
		return "", errors.New("password too short")
	}
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashBytes), nil
}

func ValidatePassword(plaintext, hash string) bool {
	if plaintext == "" || hash == "" {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plaintext))
	return err == nil
}
