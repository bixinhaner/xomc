package acs

import (
	"sync"

	"golang.org/x/time/rate"
)

// DeviceRateLimiter enforces per-device rate limits using token buckets.
type DeviceRateLimiter struct {
	limiters sync.Map // map[string]*rate.Limiter
	limit    rate.Limit
	burst    int
}

// NewDeviceRateLimiter creates a rate limiter with the specified rate per second and burst.
func NewDeviceRateLimiter(perMinute int, burst int) *DeviceRateLimiter {
	if perMinute <= 0 {
		perMinute = 10
	}
	if burst <= 0 {
		burst = 5
	}
	return &DeviceRateLimiter{
		limit: rate.Limit(float64(perMinute) / 60.0),
		burst: burst,
	}
}

// Allow checks if a request from the given device is allowed.
func (rl *DeviceRateLimiter) Allow(deviceSN string) bool {
	limiter, _ := rl.limiters.LoadOrStore(deviceSN, rate.NewLimiter(rl.limit, rl.burst))
	return limiter.(*rate.Limiter).Allow()
}

// Reset removes the rate limiter for a specific device.
func (rl *DeviceRateLimiter) Reset(deviceSN string) {
	rl.limiters.Delete(deviceSN)
}
