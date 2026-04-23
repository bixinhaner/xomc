// Package redisx 是项目统一的 Redis 访问层，负责把过去散落在 task/ / admin/ /
// config/datamodel/ / acs/ 等各包的 Redis 横切关注收口为单一基础包。
//
// 该包对外提供 4 类能力：
//  1. NewClient(cfg)         —— 连接工厂（替代 components/redis.NewRedisClient）
//  2. Keys.*                  —— 类型化 KeyBuilder，禁止字符串拼接
//  3. Codec / Retry / Scanner / Expiry —— 编解码、重试、批量扫描、TTL 清理
//  4. 可选 Prometheus Hook    —— 注入后为每次 Redis 操作自动打点
//
// 验收口径：`grep '"acs:.*:"' internal/` 只应在 redisx/keys.go 命中。
package redisx

import (
	"context"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/core/appconfig"
	"github.com/redis/go-redis/v9"
)

// DefaultPingTimeout 是初始化时健康检查的超时。
const DefaultPingTimeout = 5 * time.Second

// NewClient 依据配置创建一个 Redis UniversalClient。
// - 多地址 → Cluster 模式；单地址 → Standalone 模式；
// - 创建后立即 Ping 一次，失败则关闭并返回错误；
// - 可选 hooks（例如 PrometheusHook）通过 WithHook 装配。
func NewClient(cfg appconfig.RedisConfig, hooks ...redis.Hook) (redis.UniversalClient, error) {
	opts := &redis.UniversalOptions{
		Addrs:    cfg.Addrs,
		Password: cfg.Password,
		DB:       cfg.DB,
	}
	if cfg.PoolSize > 0 {
		opts.PoolSize = cfg.PoolSize
	}

	client := redis.NewUniversalClient(opts)
	for _, h := range hooks {
		if h != nil {
			client.AddHook(h)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), DefaultPingTimeout)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return client, nil
}

// HealthCheck 对外暴露的健康探针，供 HealthChecker 注册使用。
func HealthCheck(ctx context.Context, client redis.UniversalClient) error {
	ctx, cancel := context.WithTimeout(ctx, DefaultPingTimeout)
	defer cancel()
	return client.Ping(ctx).Err()
}
