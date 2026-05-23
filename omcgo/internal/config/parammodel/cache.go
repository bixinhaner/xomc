package parammodel

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

// 默认 TTL（24h）；真实部署可由 appconfig.ParamRegistryConfig 覆写。
//
// 24h 与 ProductRegistry 一致，对齐 BumpVersion 失效协议——多实例下若有
// XML 重导入或 Intersect 写入，由 InvalidateProduct/ParamModel 负责精准失效，
// TTL 仅作为最后兜底。
const (
	defaultMappingTTL    = 24 * time.Hour
	discoveredMappingTTL = 24 * time.Hour
)

// Cache 抽象 ParamRegistry L2 缓存。可注入 NopCache 退化为纯 L1 + DB。
//
// 缓存值统一为 JSON 序列化的 []ParamMapping；空切片是合法值（缓存"无映射"事实，
// 避免反复打库）。所有方法在条目缺失时返回 (nil, nil)，Set 接受空切片。
type Cache interface {
	// GetDefault 返回某 paramModelId 的默认映射缓存；不存在 → (nil, nil)。
	GetDefault(ctx context.Context, paramModelID uuid.UUID) ([]ParamMapping, error)
	// SetDefault 写入默认映射缓存（默认 TTL 见 defaultMappingTTL）。
	SetDefault(ctx context.Context, paramModelID uuid.UUID, mappings []ParamMapping) error
	// InvalidateDefault 删除某 paramModelId 的默认映射缓存条目。
	InvalidateDefault(ctx context.Context, paramModelID uuid.UUID) error

	// GetDiscovered 返回某 (productId, swVersion) 的发现映射缓存；不存在 → (nil, nil)。
	GetDiscovered(ctx context.Context, productID uuid.UUID, swVersion string) ([]ParamMapping, error)
	// SetDiscovered 写入发现映射缓存（默认 TTL 见 discoveredMappingTTL）。
	SetDiscovered(ctx context.Context, productID uuid.UUID, swVersion string, mappings []ParamMapping) error
	// InvalidateDiscovered 删除单条发现映射缓存条目。
	InvalidateDiscovered(ctx context.Context, productID uuid.UUID, swVersion string) error

	// GetVersion 返回当前缓存版本号；首次未设置 → 0。
	GetVersion(ctx context.Context) (int64, error)
	// BumpVersion 原子递增缓存版本号；用于 Refresh 跨实例失效信号。
	BumpVersion(ctx context.Context) (int64, error)
}

// RedisCache 是 Cache 的 Redis 实现。
type RedisCache struct {
	client          redis.UniversalClient
	defaultTTL      time.Duration
	discoveredTTL   time.Duration
}

// NewRedisCache 构造 RedisCache，使用默认 TTL（24h / 24h）。
func NewRedisCache(client redis.UniversalClient) *RedisCache {
	return &RedisCache{
		client:        client,
		defaultTTL:    defaultMappingTTL,
		discoveredTTL: discoveredMappingTTL,
	}
}

// NewRedisCacheWithTTL 构造 RedisCache 并允许覆写 TTL。
// 任一参数 ≤ 0 时退化为默认值，避免 yaml 缺省字段导致缓存不写入。
func NewRedisCacheWithTTL(client redis.UniversalClient, defaultTTL, discoveredTTL time.Duration) *RedisCache {
	if defaultTTL <= 0 {
		defaultTTL = defaultMappingTTL
	}
	if discoveredTTL <= 0 {
		discoveredTTL = discoveredMappingTTL
	}
	return &RedisCache{client: client, defaultTTL: defaultTTL, discoveredTTL: discoveredTTL}
}

// GetDefault 实现 Cache。
func (c *RedisCache) GetDefault(ctx context.Context, paramModelID uuid.UUID) ([]ParamMapping, error) {
	return c.getMappings(ctx, redisx.Keys.ParamModelDefault(paramModelID.String()))
}

// SetDefault 实现 Cache。
func (c *RedisCache) SetDefault(ctx context.Context, paramModelID uuid.UUID, mappings []ParamMapping) error {
	return c.setMappings(ctx, redisx.Keys.ParamModelDefault(paramModelID.String()), mappings, c.defaultTTL)
}

// InvalidateDefault 实现 Cache。
func (c *RedisCache) InvalidateDefault(ctx context.Context, paramModelID uuid.UUID) error {
	key := redisx.Keys.ParamModelDefault(paramModelID.String())
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("del parammodel default cache %s: %w", key, err)
	}
	return nil
}

// GetDiscovered 实现 Cache。
func (c *RedisCache) GetDiscovered(ctx context.Context, productID uuid.UUID, swVersion string) ([]ParamMapping, error) {
	return c.getMappings(ctx, redisx.Keys.ParamModelDiscovered(productID.String(), swVersion))
}

// SetDiscovered 实现 Cache。
func (c *RedisCache) SetDiscovered(ctx context.Context, productID uuid.UUID, swVersion string, mappings []ParamMapping) error {
	return c.setMappings(ctx, redisx.Keys.ParamModelDiscovered(productID.String(), swVersion), mappings, c.discoveredTTL)
}

// InvalidateDiscovered 实现 Cache。
func (c *RedisCache) InvalidateDiscovered(ctx context.Context, productID uuid.UUID, swVersion string) error {
	key := redisx.Keys.ParamModelDiscovered(productID.String(), swVersion)
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("del parammodel discovered cache %s: %w", key, err)
	}
	return nil
}

// GetVersion 实现 Cache。
func (c *RedisCache) GetVersion(ctx context.Context) (int64, error) {
	val, err := c.client.Get(ctx, redisx.Keys.ParamModelCacheVersion()).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, fmt.Errorf("get parammodel cache version: %w", err)
	}
	return val, nil
}

// BumpVersion 实现 Cache。
func (c *RedisCache) BumpVersion(ctx context.Context) (int64, error) {
	val, err := c.client.Incr(ctx, redisx.Keys.ParamModelCacheVersion()).Result()
	if err != nil {
		return 0, fmt.Errorf("incr parammodel cache version: %w", err)
	}
	return val, nil
}

// cacheEntry 是 parammodel L2 缓存的 wire 结构（T-0106 修复）。
//
// 修复前：直接存裸 []ParamMapping JSON，cache_version 字段虽然存在但读侧不校验，
// 导致 schema 演进后旧条目无法被自动失效（必须等 TTL 24h 自然过期）。
//
// 修复后：写入时把 SchemaVersion = 当前 cache_version 一起序列化；
// 读取时如果 entry.SchemaVersion ≠ 当前 cache_version，视为 stale → miss 击穿到 DB。
// 这样 BumpVersion 一次能让全部进程的全部 parammodel 缓存条目立即失效，与
// ProductRegistry / KPIRouter 的 cache_version 协议一致。
type cacheEntry struct {
	SchemaVersion int64           `json:"v"`
	Payload       []ParamMapping  `json:"p"`
}

func (c *RedisCache) getMappings(ctx context.Context, key string) ([]ParamMapping, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("get parammodel cache %s: %w", key, err)
	}
	// 空字节兼容（不应发生但稳健处理）
	if len(data) == 0 {
		return []ParamMapping{}, nil
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// 旧格式 / 损坏值：视为 miss，不返回错误（让 DB 重建覆盖即可）。
		return nil, nil
	}
	current, err := c.GetVersion(ctx)
	if err != nil {
		return nil, fmt.Errorf("get cache_version for %s: %w", key, err)
	}
	if entry.SchemaVersion != current {
		// schema_version 漂移（BumpVersion 已被调用） → 强制重建。
		return nil, nil
	}
	out := entry.Payload
	if out == nil {
		out = []ParamMapping{}
	}
	return out, nil
}

func (c *RedisCache) setMappings(ctx context.Context, key string, mappings []ParamMapping, ttl time.Duration) error {
	if mappings == nil {
		mappings = []ParamMapping{}
	}
	version, err := c.GetVersion(ctx)
	if err != nil {
		return fmt.Errorf("get cache_version for %s: %w", key, err)
	}
	data, err := json.Marshal(cacheEntry{SchemaVersion: version, Payload: mappings})
	if err != nil {
		return fmt.Errorf("marshal parammodel cache %s: %w", key, err)
	}
	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("set parammodel cache %s: %w", key, err)
	}
	return nil
}

// NopCache 是 Cache 的 no-op 实现，供未启用 Redis 的进程注入（开发环境 / acs 进程）。
//
// Get* 永远 miss；Set* / Invalidate* 均无副作用；版本号始终为 0。
// Registry 此时退化为纯 L1 sync.Map + DB read-through。
type NopCache struct{}

// GetDefault implements Cache.
func (NopCache) GetDefault(context.Context, uuid.UUID) ([]ParamMapping, error) { return nil, nil }

// SetDefault implements Cache.
func (NopCache) SetDefault(context.Context, uuid.UUID, []ParamMapping) error { return nil }

// InvalidateDefault implements Cache.
func (NopCache) InvalidateDefault(context.Context, uuid.UUID) error { return nil }

// GetDiscovered implements Cache.
func (NopCache) GetDiscovered(context.Context, uuid.UUID, string) ([]ParamMapping, error) {
	return nil, nil
}

// SetDiscovered implements Cache.
func (NopCache) SetDiscovered(context.Context, uuid.UUID, string, []ParamMapping) error {
	return nil
}

// InvalidateDiscovered implements Cache.
func (NopCache) InvalidateDiscovered(context.Context, uuid.UUID, string) error { return nil }

// GetVersion implements Cache.
func (NopCache) GetVersion(context.Context) (int64, error) { return 0, nil }

// BumpVersion implements Cache.
func (NopCache) BumpVersion(context.Context) (int64, error) { return 0, nil }
