package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates a Redis UniversalClient that auto-detects
// Cluster mode (multiple addrs) vs Standalone mode (single addr).
func NewRedisClient(cfg appconfig.RedisConfig) (redis.UniversalClient, error) {
	opts := &redis.UniversalOptions{
		Addrs:    cfg.Addrs,
		Password: cfg.Password,
		DB:       cfg.DB,
	}

	if cfg.PoolSize > 0 {
		opts.PoolSize = cfg.PoolSize
	}

	client := redis.NewUniversalClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return client, nil
}

// RedisHealthCheck verifies the Redis connection is alive.
func RedisHealthCheck(ctx context.Context, client redis.UniversalClient) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return client.Ping(ctx).Err()
}
