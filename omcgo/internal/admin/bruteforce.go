package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
)

const (
	failedKeyTTL     = 30 * time.Minute
	captchaThreshold = 3                // require CAPTCHA after this many failures
	lockThreshold    = 10               // lock account after this many failures
	lockDuration     = 30 * time.Minute
)

// LoginGuard tracks failed login attempts and enforces brute-force protections.
type LoginGuard struct {
	redis redis.UniversalClient
}

// NewLoginGuard creates a new LoginGuard.
func NewLoginGuard(client redis.UniversalClient) *LoginGuard {
	return &LoginGuard{redis: client}
}

// RecordFailure increments the failed attempt counter for a username.
func (g *LoginGuard) RecordFailure(ctx context.Context, username string) (int64, error) {
	key := redisx.Keys.AuthFailed(username)
	count, err := g.redis.Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("increment failed count: %w", err)
	}
	// Set/refresh TTL on every failure
	g.redis.Expire(ctx, key, failedKeyTTL)
	return count, nil
}

// GetFailedCount returns the current failed attempt count for a username.
func (g *LoginGuard) GetFailedCount(ctx context.Context, username string) int64 {
	count, err := g.redis.Get(ctx, redisx.Keys.AuthFailed(username)).Int64()
	if err != nil {
		return 0
	}
	return count
}

// Reset clears the failed attempt counter on successful login.
func (g *LoginGuard) Reset(ctx context.Context, username string) {
	g.redis.Del(ctx, redisx.Keys.AuthFailed(username))
}

// RequiresCaptcha returns true if the user has exceeded the CAPTCHA threshold.
func (g *LoginGuard) RequiresCaptcha(ctx context.Context, username string) bool {
	return g.GetFailedCount(ctx, username) >= captchaThreshold
}

// ShouldLock returns true if the failed count has reached the lock threshold.
func (g *LoginGuard) ShouldLock(count int64) bool {
	return count >= lockThreshold
}

// LockDuration returns the account lock duration.
func (g *LoginGuard) LockDuration() time.Duration {
	return lockDuration
}
