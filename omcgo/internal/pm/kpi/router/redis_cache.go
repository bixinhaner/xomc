package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
)

// defaultKPIRouteTTL 与 ProductRegistry / ParamRegistry 对齐 — Bump 协议负责精准失效，
// TTL 只作最后兜底。
const defaultKPIRouteTTL = 24 * time.Hour

// RedisCache 是 KPIRoute 的 L2 缓存实现。
//
// cache_version 失效协议（与 ProductRegistry / ParamRegistry 一致）：
//   - 写入时把缓存值连同 SchemaVersion 一起序列化（写入瞬间 INCR 取到的整数）。
//   - 读取时先 GET kpi-route:cache_version；若与 entry.SchemaVersion 不一致，
//     视为 stale → 返回 (nil, nil) 走 DB 重建。
//   - 显式失效全集 → BumpVersion()：INCR 让所有进程下一次读全部 stale。
//
// 这套协议保证多进程下"indicator 表 / formula 表 / product 表"任一字典变更后，
// 只需调用方 BumpVersion 一次，所有 Router 实例下次读时自动重建 — 不需要
// pub/sub，避免消息丢失。
type RedisCache struct {
	client redis.UniversalClient
	keys   redisx.KeyBuilder
	ttl    time.Duration
}

// NewRedisCache 构造默认 TTL 的 RedisCache。
func NewRedisCache(client redis.UniversalClient) *RedisCache {
	return NewRedisCacheWithTTL(client, defaultKPIRouteTTL)
}

// NewRedisCacheWithTTL 允许覆写 TTL；<= 0 退化为默认值。
func NewRedisCacheWithTTL(client redis.UniversalClient, ttl time.Duration) *RedisCache {
	if ttl <= 0 {
		ttl = defaultKPIRouteTTL
	}
	return &RedisCache{client: client, ttl: ttl}
}

// cacheEntry 是 Redis value 的 wire 结构。SchemaVersion 是写入瞬间的 cache_version 整数；
// Payload 是 KPIRoute 本体。
//
// 与 parammodel/cache.go 当前 entry 形态不同：parammodel 直接存裸 payload，
// 因此 cache_version 失效在 parammodel 端实际不生效（即 T-0106 的根因）。
// Task 3 会把同样的 entry 形态迁回 parammodel 端。
type cacheEntry struct {
	SchemaVersion int64    `json:"v"`
	Payload       KPIRoute `json:"p"`
}

// Get 命中且 schema_version 匹配 → 返回 (*KPIRoute, nil)；
// miss / unmarshal 失败 / version mismatch → (nil, nil)；
// Redis 通信失败 → (nil, err)。
func (c *RedisCache) Get(ctx context.Context, productID uuid.UUID) (*KPIRoute, error) {
	key := c.keys.KPIRouteByProduct(productID.String())
	raw, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("get %s: %w", key, err)
	}
	var entry cacheEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		// 反序列化失败：当作 miss，不返回错误（旧格式遗留 / 损坏值不应阻塞业务）。
		return nil, nil
	}
	current, err := c.GetVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("get cache_version for %s: %w", key, err)
	}
	if entry.SchemaVersion != current {
		// schema_version 漂移 → stale，强制重建。
		return nil, nil
	}
	r := entry.Payload
	if !routeHasNormalizationMetadata(&r) {
		// #866：旧缓存条目没有 Unit 字段，不能用于入库结果规范化；当作 miss 走 DB 重建。
		return nil, nil
	}
	return &r, nil
}

func routeHasNormalizationMetadata(route *KPIRoute) bool {
	if route == nil {
		return false
	}
	for _, c := range route.Counters {
		if c.Unit == "" || c.StatisType == "" {
			return false
		}
	}
	for _, k := range route.KPIs {
		if k.Unit == "" || k.StatisType == "" {
			return false
		}
	}
	return true
}

// Put 写入缓存条目，SchemaVersion 取自当前 cache_version 整数（首次未 INCR → 0）。
func (c *RedisCache) Put(ctx context.Context, route *KPIRoute) error {
	if route == nil {
		return errors.New("router.RedisCache.Put: nil route")
	}
	version, err := c.GetVersion(ctx)
	if err != nil {
		return fmt.Errorf("get cache_version: %w", err)
	}
	entry := cacheEntry{SchemaVersion: version, Payload: *route}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal kpi-route entry: %w", err)
	}
	key := c.keys.KPIRouteByProduct(route.ProductID.String())
	if err := c.client.Set(ctx, key, data, c.ttl).Err(); err != nil {
		return fmt.Errorf("set %s: %w", key, err)
	}
	return nil
}

// GetVersion 读 kpi-route:cache_version；未设置 → 0。
func (c *RedisCache) GetVersion(ctx context.Context) (int64, error) {
	val, err := c.client.Get(ctx, c.keys.KPIRouteCacheVersion()).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, fmt.Errorf("get kpi-route cache_version: %w", err)
	}
	return val, nil
}

// BumpVersion INCR kpi-route:cache_version，使所有进程现存缓存条目立即视为 stale。
// 由管理 API / dictloader Refresh 后调用。
func (c *RedisCache) BumpVersion(ctx context.Context) (int64, error) {
	val, err := c.client.Incr(ctx, c.keys.KPIRouteCacheVersion()).Result()
	if err != nil {
		return 0, fmt.Errorf("incr kpi-route cache_version: %w", err)
	}
	return val, nil
}
