package redisx

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// DefaultScanCount 单轮 SCAN 的 COUNT 参数。Redis 的 COUNT 是"建议值"而非硬限，
// 500 在绝大多数场景下能平衡往返次数与服务端消耗。
const DefaultScanCount = 500

// ScanCallback 在每轮扫描命中的 key 上被调用；返回 error 则中止扫描。
type ScanCallback func(ctx context.Context, key string) error

// Scan 迭代执行 SCAN 直到游标回到 0，对每个命中 key 调一次 cb。
// 相比 KEYS，SCAN 不阻塞 Redis 主线程、不会锁服务，适合扫描量较大的前缀。
//
// pattern 应通过 Keys.XXXPattern() 获取，避免散落的字符串字面量。
func Scan(ctx context.Context, client redis.UniversalClient, pattern string, cb ScanCallback) error {
	if cb == nil {
		return fmt.Errorf("redisx: Scan callback is nil")
	}
	var cursor uint64
	for {
		keys, next, err := client.Scan(ctx, cursor, pattern, DefaultScanCount).Result()
		if err != nil {
			return fmt.Errorf("redis scan %q: %w", pattern, err)
		}
		for _, k := range keys {
			if err := cb(ctx, k); err != nil {
				return err
			}
		}
		if next == 0 {
			return nil
		}
		cursor = next
	}
}

// ScanKeys 收集 SCAN 命中的全部键并返回。适合键数量有限（数千级）的场景；
// 海量键请使用 Scan + 流式处理。
func ScanKeys(ctx context.Context, client redis.UniversalClient, pattern string) ([]string, error) {
	var out []string
	err := Scan(ctx, client, pattern, func(_ context.Context, key string) error {
		out = append(out, key)
		return nil
	})
	return out, err
}
