package redisx

import (
	"context"
	"errors"
	"time"
)

// RetryConfig 描述一次重试的退避策略。零值即"不重试"。
type RetryConfig struct {
	MaxAttempts int           // 含首次在内的最大尝试次数；<=1 表示不重试
	InitialWait time.Duration // 第一次失败后等待时长
	MaxWait     time.Duration // 单次最大等待时长（退避不会超过此值）
	Multiplier  float64       // 指数退避倍数；<=1 回退为线性
}

// DefaultRetry 对 Redis 瞬时错误使用的默认策略：4 次尝试，100ms → 200ms →
// 400ms → 800ms，然后放弃；适合非关键读（如缓存未命中）。
var DefaultRetry = RetryConfig{
	MaxAttempts: 4,
	InitialWait: 100 * time.Millisecond,
	MaxWait:     2 * time.Second,
	Multiplier:  2,
}

// RetryableFunc 是被 Retry 包装的 Redis 操作。若返回 ErrPermanent（或其 wrap）
// 将立即放弃重试。
type RetryableFunc func(ctx context.Context) error

// ErrPermanent 标记"不应重试"的错误。业务侧可 fmt.Errorf("...%w", redisx.ErrPermanent)。
var ErrPermanent = errors.New("redisx: permanent error")

// Retry 按 RetryConfig 重试 fn。ctx 取消或 ErrPermanent 立即终止。
// 返回最后一次尝试的错误。
func Retry(ctx context.Context, cfg RetryConfig, fn RetryableFunc) error {
	attempts := cfg.MaxAttempts
	if attempts <= 1 {
		return fn(ctx)
	}
	wait := cfg.InitialWait
	mult := cfg.Multiplier
	if mult <= 1 {
		mult = 1 // 退化为线性
	}

	var lastErr error
	for i := 0; i < attempts; i++ {
		lastErr = fn(ctx)
		if lastErr == nil {
			return nil
		}
		if errors.Is(lastErr, ErrPermanent) {
			return lastErr
		}
		if i == attempts-1 {
			break
		}
		// 本轮退避
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
		// 下一轮等待时长（封顶）
		if cfg.MaxWait > 0 && time.Duration(float64(wait)*mult) > cfg.MaxWait {
			wait = cfg.MaxWait
		} else {
			wait = time.Duration(float64(wait) * mult)
		}
	}
	return lastErr
}
