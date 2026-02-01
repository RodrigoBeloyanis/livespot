package observability

import (
	"crypto/rand"
	"fmt"
	"time"
)

func NewRunID(now time.Time) (string, error) {
	suffix, err := randomHex(3)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("run_%s_%s", now.UTC().Format("20060102_150405"), suffix), nil
}

func NewCycleID(now time.Time) (string, error) {
	suffix, err := randomHex(3)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("cyc_%s_%s", now.UTC().Format("20060102_150405"), suffix), nil
}

func randomHex(bytesLen int) (string, error) {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("rand read: %w", err)
	}
	return fmt.Sprintf("%x", buf), nil
}
