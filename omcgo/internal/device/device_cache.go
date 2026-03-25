package device

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	deviceCacheKeyPrefix = "device:sn:"
	deviceCacheTTL       = 10 * time.Minute
)

// DeviceCache provides Redis-backed caching for device lookups by serial number.
// Cache-aside pattern: check Redis → miss → query PostgreSQL → set Redis.
type DeviceCache struct {
	redis  redis.UniversalClient
	logger *zap.Logger
}

// NewDeviceCache creates a new DeviceCache.
func NewDeviceCache(redis redis.UniversalClient, logger *zap.Logger) *DeviceCache {
	return &DeviceCache{redis: redis, logger: logger}
}

func deviceCacheKey(sn string) string {
	return deviceCacheKeyPrefix + sn
}

// Get retrieves a device from the cache by serial number.
// Returns (device, true) on cache hit, (nil, false) on miss or error.
func (c *DeviceCache) Get(ctx context.Context, sn string) (*model.Device, bool) {
	data, err := c.redis.Get(ctx, deviceCacheKey(sn)).Bytes()
	if err != nil {
		if err != redis.Nil {
			c.logger.Warn("device cache get", zap.String("serial_number", sn), zap.Error(err))
		}
		return nil, false
	}

	var device model.Device
	if err := json.Unmarshal(data, &device); err != nil {
		c.logger.Warn("device cache unmarshal", zap.String("serial_number", sn), zap.Error(err))
		return nil, false
	}
	return &device, true
}

// Set stores a device in the cache.
func (c *DeviceCache) Set(ctx context.Context, device *model.Device) {
	data, err := json.Marshal(device)
	if err != nil {
		c.logger.Warn("device cache marshal", zap.String("serial_number", device.SerialNumber), zap.Error(err))
		return
	}
	if err := c.redis.Set(ctx, deviceCacheKey(device.SerialNumber), data, deviceCacheTTL).Err(); err != nil {
		c.logger.Warn("device cache set", zap.String("serial_number", device.SerialNumber), zap.Error(err))
	}
}

// Delete removes a device from the cache.
func (c *DeviceCache) Delete(ctx context.Context, sn string) {
	if err := c.redis.Del(ctx, deviceCacheKey(sn)).Err(); err != nil {
		c.logger.Warn("device cache delete", zap.String("serial_number", sn), zap.Error(err))
	}
}

// GetOrLoad checks the cache first, on miss calls loader to fetch from DB, then caches the result.
func (c *DeviceCache) GetOrLoad(ctx context.Context, sn string, loader func(ctx context.Context, sn string) (*model.Device, error)) (*model.Device, error) {
	// L1: Redis cache
	if device, ok := c.Get(ctx, sn); ok {
		c.logger.Debug("device cache hit", zap.String("serial_number", sn))
		return device, nil
	}

	// L2: PostgreSQL (via loader)
	c.logger.Debug("device cache miss, loading from DB", zap.String("serial_number", sn))
	device, err := loader(ctx, sn)
	if err != nil {
		return nil, fmt.Errorf("load device: %w", err)
	}

	// Cache the result (only if found)
	if device != nil {
		c.Set(ctx, device)
		c.logger.Debug("device cache populated", zap.String("serial_number", sn))
	} else {
		c.logger.Debug("device not found in DB, skip cache", zap.String("serial_number", sn))
	}

	return device, nil
}
