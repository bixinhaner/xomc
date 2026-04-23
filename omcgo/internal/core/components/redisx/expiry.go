package redisx

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// TouchTTL 把一组 key 的 TTL 统一续期到 ttl。底层使用 EXPIRE per-key；若只是
// 续命脚本会话等写入时间密集的场景，调用方应在写入本身通过 Set(..., ttl)
// 一次性落 TTL，避免额外往返。
//
// 对不存在的 key EXPIRE 返回 0（go-redis 不报错），本函数不会把"缺失键"当错误。
func TouchTTL(ctx context.Context, client redis.UniversalClient, ttl time.Duration, keys ...string) error {
	if len(keys) == 0 || ttl <= 0 {
		return nil
	}
	pipe := client.Pipeline()
	for _, k := range keys {
		pipe.Expire(ctx, k, ttl)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("redisx touch ttl (%d keys): %w", len(keys), err)
	}
	return nil
}

// PurgePattern 扫描并批量删除匹配 pattern 的全部键。按 BatchSize 分批 DEL，
// 避免单个 pipeline 过大。返回实际删除的键数。
//
// 用法场景：
//   - 运维收尾时清理已下线的 key 前缀（如 Phase 1 的 acs:cmdq:*）
//   - 单测后清理 fixtures
//
// pattern 必须来自 Keys.XXXPattern()，禁止内联字面量。
func PurgePattern(ctx context.Context, client redis.UniversalClient, pattern string, batchSize int) (int64, error) {
	if batchSize <= 0 {
		batchSize = 200
	}
	var total int64
	batch := make([]string, 0, batchSize)

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		n, err := client.Del(ctx, batch...).Result()
		if err != nil {
			return fmt.Errorf("redisx purge del: %w", err)
		}
		total += n
		batch = batch[:0]
		return nil
	}

	err := Scan(ctx, client, pattern, func(_ context.Context, key string) error {
		batch = append(batch, key)
		if len(batch) >= batchSize {
			return flush()
		}
		return nil
	})
	if err != nil {
		return total, err
	}
	if err := flush(); err != nil {
		return total, err
	}
	return total, nil
}
