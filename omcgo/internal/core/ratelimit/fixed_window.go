// Package ratelimit 提供跨实例共享的 Redis 固定窗口限流器，专供北向 / 网元直连
// 等低频外部接口防滥用。
//
// 为什么不复用 internal/core/middleware.RateLimit：
//   - 那个是单实例进程内 token-bucket（per-IP），用于管理面全局防洪泛；
//   - 北向 / 网元直连需要 per-device & per-endpoint 维度、且多 app/acs 实例聚合计数，
//     必须落 Redis。算法选固定窗口（INCR+EXPIRE）而非 token-bucket：实现简单、跨实例
//     天然一致，外部接口对突发平滑性无要求。与 internal/admin/ip_guard.go 同一套
//     Redis 计数惯用法。
package ratelimit

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
)

// FixedWindowLimiter 是基于 Redis INCR+EXPIRE 的固定窗口限流器。
// 每个 (scope, key) 在每个 window 内最多放行 limit 次请求。
//
// 降级语义：redis 客户端为 nil（dev/test 未注入）或 Redis 调用出错 → 放行（fail-open），
// 与 ip_guard 一致，避免缓存抖动拖垮外部接口。计数失败在日志可见由调用方处理。
type FixedWindowLimiter struct {
	redis  redis.UniversalClient
	scope  string
	limit  int64
	window time.Duration
}

// NewFixedWindowLimiter 构造一个固定窗口限流器。
//   - scope  : 限流维度命名空间（如 "nb:endpoint"、"ne:device"），用于隔离不同业务的计数器；
//   - limit  : 每窗口最大放行次数；<=0 视为不限流（Allow 永远放行）；
//   - window : 窗口时长；<=0 时退化为 1 分钟。
func NewFixedWindowLimiter(client redis.UniversalClient, scope string, limit int64, window time.Duration) *FixedWindowLimiter {
	if window <= 0 {
		window = time.Minute
	}
	return &FixedWindowLimiter{
		redis:  client,
		scope:  scope,
		limit:  limit,
		window: window,
	}
}

// Allow 在窗口内为 key 计数 +1，返回是否放行。超限返回 false，调用方应回 429。
//   - allowed : 是否放行；
//   - count   : 当前窗口内累计计数（含本次），便于日志/调试；
//   - err     : Redis 调用错误（已 fail-open，调用方可仅记日志，不应据此拒绝请求）。
func (l *FixedWindowLimiter) Allow(ctx context.Context, key string) (allowed bool, count int64, err error) {
	// 不限流 / 未注入 redis → 放行。
	if l == nil || l.redis == nil || l.limit <= 0 {
		return true, 0, nil
	}

	windowSecs := int64(l.window / time.Second)
	if windowSecs <= 0 {
		windowSecs = 1
	}
	bucket := time.Now().Unix() / windowSecs
	redisKey := redisx.Keys.RateLimitFixedWindow(l.scope, key, bucket)

	cnt, incrErr := l.redis.Incr(ctx, redisKey).Result()
	if incrErr != nil {
		// fail-open：Redis 抖动不应拒绝外部接口。
		return true, 0, fmt.Errorf("ratelimit incr %s: %w", l.scope, incrErr)
	}
	// 仅首次设置 TTL（count==1），避免每次请求都刷新窗口导致窗口被无限延长。
	if cnt == 1 {
		l.redis.Expire(ctx, redisKey, l.window)
	}

	return cnt <= l.limit, cnt, nil
}
