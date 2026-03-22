package push

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// SignPayload computes an HMAC-SHA256 signature for a webhook payload.
// The signature is computed over: "<timestamp>.<body>" to bind the timestamp to the payload.
func SignPayload(secret string, timestamp int64, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	message := fmt.Sprintf("%d.%s", timestamp, body)
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyPayload checks that a webhook signature is valid.
func VerifyPayload(secret, signature string, timestamp int64, body []byte) bool {
	expected := SignPayload(secret, timestamp, body)
	return hmac.Equal([]byte(expected), []byte(signature))
}
