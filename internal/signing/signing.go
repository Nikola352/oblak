package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

func Sign(method string, path string, body []byte, timestamp string, secretKey string) (string, error) {
	canonical := method + "_" + path + "_" + hashBytes(body) + "_" + timestamp
	bytes := sha256.Sum256([]byte(canonical))

	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write(bytes[:])

	return hex.EncodeToString(mac.Sum(nil)), nil
}

func hashBytes(bytes []byte) string {
	hash := sha256.Sum256(bytes)
	return hex.EncodeToString(hash[:])
}
