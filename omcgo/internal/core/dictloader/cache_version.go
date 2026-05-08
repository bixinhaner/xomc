package dictloader

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// CacheVersion implements the cross-instance cache invalidation protocol.
//
// Write side (after a DB mutation): Increment bumps the Redis counter.
// Read side (every instance):       Watch polls the counter; when it differs
//
//	from the last seen local value, registered
//	OnBump callbacks fire so the consumer can
//	clear its in-memory L1 cache.
//
// Polling is preferred over Pub/Sub because Pub/Sub messages are lost when an
// instance is offline; polling provides eventual consistency with bounded delay.
type CacheVersion struct {
	rdb       redis.UniversalClient
	domain    string
	key       string
	pollEvery time.Duration
	logger    *zap.Logger

	mu     sync.RWMutex
	local  int64
	onBump []func()
}

// NewCacheVersion constructs a CacheVersion helper. domain is used in the
// Redis key ("{domain}:cache_version") and in log fields. pollEvery <= 0
// defaults to 30 seconds. A nil logger uses zap.NewNop.
func NewCacheVersion(rdb redis.UniversalClient, domain string, pollEvery time.Duration, logger *zap.Logger) *CacheVersion {
	if pollEvery <= 0 {
		pollEvery = 30 * time.Second
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &CacheVersion{
		rdb:       rdb,
		domain:    domain,
		key:       domain + ":cache_version",
		pollEvery: pollEvery,
		logger:    logger,
	}
}

// Key returns the Redis key tracked by this CacheVersion.
func (cv *CacheVersion) Key() string { return cv.key }

// Current returns the most recently observed version (0 before any read).
func (cv *CacheVersion) Current() int64 {
	cv.mu.RLock()
	defer cv.mu.RUnlock()
	return cv.local
}

// Increment atomically bumps the Redis counter and returns its new value.
// Callers should invoke this after committing a DB write.
func (cv *CacheVersion) Increment(ctx context.Context) (int64, error) {
	if cv.rdb == nil {
		return 0, errors.New("dictloader.cacheversion: nil redis client")
	}
	n, err := cv.rdb.Incr(ctx, cv.key).Result()
	if err != nil {
		return 0, fmt.Errorf("dictloader.cacheversion: incr %s: %w", cv.key, err)
	}
	cv.mu.Lock()
	cv.local = n
	cv.mu.Unlock()
	return n, nil
}

// OnBump registers a callback fired when Watch observes a remote bump. Callbacks
// run synchronously in Watch's goroutine and should be cheap (e.g. clear sync.Map).
func (cv *CacheVersion) OnBump(fn func()) {
	if fn == nil {
		return
	}
	cv.mu.Lock()
	cv.onBump = append(cv.onBump, fn)
	cv.mu.Unlock()
}

// Watch polls Redis for version changes until ctx is cancelled. The first poll
// runs immediately to seed the local counter; subsequent polls fire every pollEvery.
func (cv *CacheVersion) Watch(ctx context.Context) {
	ticker := time.NewTicker(cv.pollEvery)
	defer ticker.Stop()

	cv.refresh(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cv.refresh(ctx)
		}
	}
}

func (cv *CacheVersion) refresh(ctx context.Context) {
	if cv.rdb == nil {
		return
	}
	raw, err := cv.rdb.Get(ctx, cv.key).Result()
	if errors.Is(err, redis.Nil) {
		return
	}
	if err != nil {
		cv.logger.Warn("dictloader.cacheversion: poll failed",
			zap.String("domain", cv.domain), zap.Error(err))
		return
	}
	remote, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		cv.logger.Warn("dictloader.cacheversion: parse failed",
			zap.String("domain", cv.domain), zap.String("raw", raw), zap.Error(err))
		return
	}

	cv.mu.Lock()
	bumped := remote != cv.local
	cv.local = remote
	callbacks := append([]func(){}, cv.onBump...)
	cv.mu.Unlock()

	if !bumped {
		return
	}
	cv.logger.Info("dictloader.cacheversion: bump detected",
		zap.String("domain", cv.domain), zap.Int64("version", remote))
	for _, fn := range callbacks {
		fn()
	}
}
