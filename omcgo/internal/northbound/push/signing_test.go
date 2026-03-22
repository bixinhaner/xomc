package push

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSignPayload(t *testing.T) {
	secret := "test-signing-secret-key"
	timestamp := int64(1711100000)
	body := []byte(`{"event_id":"abc","subject":"oss.alarm.forward","payload":{}}`)

	sig := SignPayload(secret, timestamp, body)
	assert.NotEmpty(t, sig)
	assert.Len(t, sig, 64) // SHA-256 hex = 64 chars

	// Verify roundtrip
	assert.True(t, VerifyPayload(secret, sig, timestamp, body))

	// Wrong secret
	assert.False(t, VerifyPayload("wrong-secret", sig, timestamp, body))

	// Wrong timestamp (replay protection)
	assert.False(t, VerifyPayload(secret, sig, timestamp+1, body))

	// Tampered body
	assert.False(t, VerifyPayload(secret, sig, timestamp, []byte(`{"tampered":true}`)))
}

func TestSignPayload_Deterministic(t *testing.T) {
	secret := "key"
	ts := int64(1000)
	body := []byte("hello")

	sig1 := SignPayload(secret, ts, body)
	sig2 := SignPayload(secret, ts, body)
	assert.Equal(t, sig1, sig2)
}
