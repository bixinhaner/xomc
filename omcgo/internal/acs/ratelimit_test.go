package acs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviceRateLimiter_Allow(t *testing.T) {
	// 60 per minute = 1/s, burst 5 => first 5 calls should succeed immediately
	rl := NewDeviceRateLimiter(60, 5)

	for i := 0; i < 5; i++ {
		assert.True(t, rl.Allow("device-001"), "request %d should be allowed", i+1)
	}
}

func TestDeviceRateLimiter_Deny(t *testing.T) {
	// burst=2 means only 2 tokens available at start
	rl := NewDeviceRateLimiter(60, 2)

	assert.True(t, rl.Allow("device-002"))
	assert.True(t, rl.Allow("device-002"))
	// Third call should exceed burst
	assert.False(t, rl.Allow("device-002"))
}

func TestDeviceRateLimiter_DifferentDevices(t *testing.T) {
	rl := NewDeviceRateLimiter(60, 2)

	// Exhaust device-A
	assert.True(t, rl.Allow("device-A"))
	assert.True(t, rl.Allow("device-A"))
	assert.False(t, rl.Allow("device-A"))

	// device-B should still have its own independent limit
	assert.True(t, rl.Allow("device-B"))
	assert.True(t, rl.Allow("device-B"))
}

func TestDeviceRateLimiter_Reset(t *testing.T) {
	rl := NewDeviceRateLimiter(60, 2)

	// Exhaust the limiter
	assert.True(t, rl.Allow("device-003"))
	assert.True(t, rl.Allow("device-003"))
	assert.False(t, rl.Allow("device-003"))

	// Reset clears the limiter; a new one is created on next Allow
	rl.Reset("device-003")

	assert.True(t, rl.Allow("device-003"))
	assert.True(t, rl.Allow("device-003"))
}

func TestNewDeviceRateLimiter_Defaults(t *testing.T) {
	// perMinute <= 0 defaults to 10, burst <= 0 defaults to 5
	rl := NewDeviceRateLimiter(0, 0)

	// With default burst=5, first 5 should pass
	for i := 0; i < 5; i++ {
		assert.True(t, rl.Allow("device-default"), "request %d should be allowed", i+1)
	}
	// 6th should be denied (burst exhausted)
	assert.False(t, rl.Allow("device-default"))

	// Also verify negative values are handled
	rl2 := NewDeviceRateLimiter(-1, -1)
	for i := 0; i < 5; i++ {
		assert.True(t, rl2.Allow("device-neg"), "request %d should be allowed", i+1)
	}
	assert.False(t, rl2.Allow("device-neg"))
}
