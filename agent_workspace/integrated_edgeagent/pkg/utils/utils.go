package utils

import (
	"math/rand"
	"time"

	"github.com/google/uuid"
)

func GenerateUUID() string {
	return uuid.New().String()
}

func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func GetCurrentTimestamp() int64 {
	return time.Now().Unix()
}

func GetCurrentTimestampMS() int64 {
	return time.Now().UnixMilli()
}

func FormatTime(timestamp int64) string {
	return time.Unix(timestamp, 0).Format(time.RFC3339)
}