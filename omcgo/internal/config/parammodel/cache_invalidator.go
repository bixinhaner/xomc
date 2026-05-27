package parammodel

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
)

// CacheInvalidationResult 报告 InvalidateCache 的产出。
type CacheInvalidationResult struct {
	KeysCleared       int   // 实际 UNLINK 的键数(default + discovered)
	CacheVersionAfter int64 // INCR 后的 parammodel:cache_version
}

// InvalidateCache 清 Redis L2(parammodel:default:* + parammodel:discovered:*) +
// 自增 parammodel:cache_version。跨实例 ParamRegistry 会通过 version 变化感知失效。
//
// 调用场景:
//   - omcctl device sweep-paths --apply 写完 XML/DB 后
//   - admin POST /api/v1/admin/dictload/reload?name=param-model 完成后
//
// SCAN + UNLINK 避免 KEYS / DEL 的阻塞。中等规模(几百-几千键)足够。
func InvalidateCache(ctx context.Context, rdb redis.UniversalClient, logger *zap.Logger) (*CacheInvalidationResult, error) {
	if rdb == nil {
		return nil, fmt.Errorf("redis client required")
	}
	log := logger
	if log == nil {
		log = zap.NewNop()
	}

	totalCleared := 0
	for _, prefix := range []string{
		redisx.Keys.ParamModelDefaultPrefix(),
		redisx.Keys.ParamModelDiscoveredPrefix(),
	} {
		n, err := scanAndUnlink(ctx, rdb, prefix+"*")
		if err != nil {
			return nil, fmt.Errorf("scan/unlink %s: %w", prefix, err)
		}
		totalCleared += n
	}

	version, err := rdb.Incr(ctx, redisx.Keys.ParamModelCacheVersion()).Result()
	if err != nil {
		return nil, fmt.Errorf("incr cache_version: %w", err)
	}

	log.Info("parammodel cache invalidated",
		zap.Int("keys_cleared", totalCleared),
		zap.Int64("cache_version", version),
	)

	return &CacheInvalidationResult{
		KeysCleared:       totalCleared,
		CacheVersionAfter: version,
	}, nil
}

// scanAndUnlink 用 SCAN 游标遍历键空间,批量 UNLINK。
func scanAndUnlink(ctx context.Context, rdb redis.UniversalClient, match string) (int, error) {
	const batchSize = 500
	var cursor uint64
	total := 0
	for {
		keys, next, err := rdb.Scan(ctx, cursor, match, batchSize).Result()
		if err != nil {
			return total, err
		}
		if len(keys) > 0 {
			n, err := rdb.Unlink(ctx, keys...).Result()
			if err != nil {
				return total, err
			}
			total += int(n)
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return total, nil
}
