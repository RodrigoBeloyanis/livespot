package hash

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
)

func CanonicalJSON(v any) ([]byte, error) {
	buf, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return jsoncanonicalizer.Transform(buf)
}

func HashSHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func CanonicalHash(v any) (string, error) {
	buf, err := CanonicalJSON(v)
	if err != nil {
		return "", err
	}
	return HashSHA256Hex(buf), nil
}
