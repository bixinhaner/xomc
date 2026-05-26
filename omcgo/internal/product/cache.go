package product

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

// productCacheTTL 是 product 详情在 L2 Redis 的存活时间。
//
// 24h 取自数据模型既有缓存策略（datamodel.modelTTL）；TTL 内若管理 API 改动，
// BumpVersion 会触发其他实例 drop L1 + 走 DB 取新版。
const productCacheTTL = 24 * time.Hour

// ProductClassHitTTL 是 productClass → product 命中条目的 L2 Redis 存活时间（T-0173）。
//
// 1h 平衡缓存收益与 stale 风险；CacheVersion 字段在条目内做主动失效，TTL 仅作上限。
const ProductClassHitTTL = time.Hour

// ProductClassOrphanTTL 是 productClass orphan 结果的 negative cache TTL（T-0173）。
//
// 5min 短于 hit TTL —— 新增 pattern 后最坏 5 分钟即可让原孤儿设备被正确路由，
// 不靠主动失效。
const ProductClassOrphanTTL = 5 * time.Minute

// Cache 抽象 ProductRegistry L2 缓存。可注入 nil-safe 实现（NopCache）退化为纯 L1 + DB。
type Cache interface {
	// GetProduct 返回缓存的 Product；不存在 → (nil, nil)。
	GetProduct(ctx context.Context, id uuid.UUID) (*Product, error)
	// SetProduct 写入缓存（24h TTL）。
	SetProduct(ctx context.Context, p *Product) error
	// InvalidateProduct 删除单 product 缓存条目。
	InvalidateProduct(ctx context.Context, id uuid.UUID) error
	// GetVersion 返回当前缓存版本号；首次未设置 → 0。
	GetVersion(ctx context.Context) (int64, error)
	// BumpVersion 原子递增缓存版本号并返回新值；用于跨实例失效信号。
	BumpVersion(ctx context.Context) (int64, error)

	// GetProductClass 返回 productClass 路由结果的缓存条目；不存在 → (nil, nil)。
	// 调用方需自行比对 entry.CacheVersion 识别 stale（T-0173）。
	GetProductClass(ctx context.Context, productClass string) (*ProductClassCacheEntry, error)
	// SetProductClass 写入 productClass 路由结果（hit 1h / orphan 5min；TTL 由调用方传入）。
	SetProductClass(ctx context.Context, productClass string, entry *ProductClassCacheEntry, ttl time.Duration) error
}

// RedisCache 是 Cache 的 Redis 实现。
type RedisCache struct {
	client redis.UniversalClient
}

type cachedProduct struct {
	Version int64   `json:"version"`
	Product Product `json:"product"`
}

// NewRedisCache 构造一个绑定到给定 Redis 客户端的 Cache。
func NewRedisCache(client redis.UniversalClient) *RedisCache {
	return &RedisCache{client: client}
}

// GetProduct 实现 Cache。
func (c *RedisCache) GetProduct(ctx context.Context, id uuid.UUID) (*Product, error) {
	key := redisx.Keys.ProductByID(id.String())
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("get product cache %s: %w", key, err)
	}
	var entry cachedProduct
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("unmarshal product cache %s: %w", key, err)
	}
	version, err := c.GetVersion(ctx)
	if err != nil {
		return nil, err
	}
	if entry.Version != version {
		return nil, nil
	}
	return &entry.Product, nil
}

// SetProduct 实现 Cache。
func (c *RedisCache) SetProduct(ctx context.Context, p *Product) error {
	if p == nil {
		return errors.New("nil product")
	}
	key := redisx.Keys.ProductByID(p.ID.String())
	version, err := c.GetVersion(ctx)
	if err != nil {
		return err
	}
	data, err := json.Marshal(cachedProduct{Version: version, Product: *p})
	if err != nil {
		return fmt.Errorf("marshal product %s: %w", p.ID, err)
	}
	if err := c.client.Set(ctx, key, data, productCacheTTL).Err(); err != nil {
		return fmt.Errorf("set product cache %s: %w", key, err)
	}
	return nil
}

// InvalidateProduct 实现 Cache。
func (c *RedisCache) InvalidateProduct(ctx context.Context, id uuid.UUID) error {
	key := redisx.Keys.ProductByID(id.String())
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("del product cache %s: %w", key, err)
	}
	return nil
}

// GetVersion 实现 Cache。
func (c *RedisCache) GetVersion(ctx context.Context) (int64, error) {
	val, err := c.client.Get(ctx, redisx.Keys.ProductCacheVersion()).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, fmt.Errorf("get product cache version: %w", err)
	}
	return val, nil
}

// BumpVersion 实现 Cache。
func (c *RedisCache) BumpVersion(ctx context.Context) (int64, error) {
	val, err := c.client.Incr(ctx, redisx.Keys.ProductCacheVersion()).Result()
	if err != nil {
		return 0, fmt.Errorf("incr product cache version: %w", err)
	}
	return val, nil
}

// GetProductClass 实现 Cache。读出后调用方比对 CacheVersion 即可识别 stale，
// 因此本方法不在内部做版本检查 —— 让 Registry 的热路径只查一次 Cache.GetVersion()。
func (c *RedisCache) GetProductClass(ctx context.Context, productClass string) (*ProductClassCacheEntry, error) {
	key := redisx.Keys.ProductByProductClass(productClass)
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, fmt.Errorf("get productClass cache %s: %w", key, err)
	}
	var entry ProductClassCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("unmarshal productClass cache %s: %w", key, err)
	}
	return &entry, nil
}

// SetProductClass 实现 Cache。TTL 由调用方按 hit/orphan 区分传入。
func (c *RedisCache) SetProductClass(ctx context.Context, productClass string, entry *ProductClassCacheEntry, ttl time.Duration) error {
	if entry == nil {
		return errors.New("nil productClass cache entry")
	}
	key := redisx.Keys.ProductByProductClass(productClass)
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal productClass cache %s: %w", key, err)
	}
	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("set productClass cache %s: %w", key, err)
	}
	return nil
}

// NopCache 是 Cache 的 no-op 实现，供未启用 Redis 的进程注入（开发环境 / acs 进程）。
//
// GetProduct 永远 miss；写操作均无副作用；版本号始终为 0。Registry 此时退化为
// 纯 L1 sync.Map + DB read-through。
type NopCache struct{}

func (NopCache) GetProduct(context.Context, uuid.UUID) (*Product, error) { return nil, nil }
func (NopCache) SetProduct(context.Context, *Product) error              { return nil }
func (NopCache) InvalidateProduct(context.Context, uuid.UUID) error      { return nil }
func (NopCache) GetVersion(context.Context) (int64, error)               { return 0, nil }
func (NopCache) BumpVersion(context.Context) (int64, error)              { return 0, nil }
func (NopCache) GetProductClass(context.Context, string) (*ProductClassCacheEntry, error) {
	return nil, nil
}
func (NopCache) SetProductClass(context.Context, string, *ProductClassCacheEntry, time.Duration) error {
	return nil
}
