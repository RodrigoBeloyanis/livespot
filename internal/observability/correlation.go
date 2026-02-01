package observability

import (
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var idRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

func NewRunID(now time.Time) string {
	return "run_" + now.UTC().Format("20060102_150405")
}

func NewCycleID(now time.Time) string {
	return "cyc_" + now.UTC().Format("20060102_150405")
}

func SnapshotIDFromHash(hash string) (string, error) {
	if !isLowerHex64(hash) {
		return "", fmt.Errorf("snapshot hash invalid")
	}
	return "snap_" + hash, nil
}

func DecisionIDFromHash(hash string) (string, error) {
	if !isLowerHex64(hash) {
		return "", fmt.Errorf("decision hash invalid")
	}
	return "dec_" + hash, nil
}

func OrderIntentIDFromHash(hash string) (string, error) {
	if !isLowerHex64(hash) {
		return "", fmt.Errorf("order_intent hash invalid")
	}
	return "oi_" + hash, nil
}

func ClientOrderID(orderIntentID string) (string, error) {
	if orderIntentID == "" {
		return "", fmt.Errorf("order_intent_id missing")
	}
	if !idRe.MatchString(orderIntentID) {
		return "", fmt.Errorf("order_intent_id invalid")
	}
	sum := sha256.Sum256([]byte(orderIntentID))
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(sum[:])
	enc = strings.ToUpper(enc)
	if len(enc) < 34 {
		return "", fmt.Errorf("client order id encoding too short")
	}
	return "X_" + enc[:34], nil
}

func ValidateID(id string) error {
	if id == "" {
		return fmt.Errorf("id missing")
	}
	if !idRe.MatchString(id) {
		return fmt.Errorf("id invalid")
	}
	return nil
}

func isLowerHex64(s string) bool {
	if len(s) != 64 {
		return false
	}
	_, err := hex.DecodeString(s)
	if err != nil {
		return false
	}
	return strings.ToLower(s) == s
}
