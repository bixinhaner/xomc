// Package redis 保留作为向后兼容的 shim。实际实现已收口到
// internal/core/components/redisx，新代码请直接使用 redisx.NewClient /
// redisx.HealthCheck。
package redis

import (
	"context"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/omcgo/omcgo/internal/core/components/redisx"
	"github.com/redis/go-redis/v9"
)

// NewRedisClient 历史入口，保持签名不变；内部直接委托给 redisx.NewClient。
//
// Deprecated: 请使用 redisx.NewClient。
func NewRedisClient(cfg appconfig.RedisConfig) (redis.UniversalClient, error) {
	return redisx.NewClient(cfg)
}

// RedisHealthCheck 历史入口，保持签名不变；内部委托给 redisx.HealthCheck。
//
// Deprecated: 请使用 redisx.HealthCheck。
func RedisHealthCheck(ctx context.Context, client redis.UniversalClient) error {
	return redisx.HealthCheck(ctx, client)
}
