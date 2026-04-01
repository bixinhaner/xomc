// Package reliability 已在 circuit_breaker.go 中声明包注释。
package reliability

import (
	"context"
	"fmt"
	"math"
	"time"
)

// RetryConfig 定义指数退退重试的参数。
// MaxAttempts：最大尝试次数（包括第一次）。
// BaseDelay：第一次重试前的基础延迟，每次指数翻倍：1s, 2s, 4s, 8s...
// MaxDelay：重试间隔上限，防止回退时间过长。
type RetryConfig struct {
	MaxAttempts int           // Maximum number of attempts (including the first).
	BaseDelay   time.Duration // Base delay before the first retry.
	MaxDelay    time.Duration // Maximum delay between retries.
}

// DefaultRetryConfig returns a sensible default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   1 * time.Second,
		MaxDelay:    30 * time.Second,
	}
}

// RetryableFunc is a function that can be retried.
type RetryableFunc func(ctx context.Context) error

// Retry 以指数退退策略执行 fn，成功则返回 nil，
// 所有尝试耗尽后返回最后一次错误。
// 使用场景：调用不稳定的外部服务（如 NATS Publish、MinIO、CPE Connection Request），
// 配合 context.WithTimeout 设置整体重试最大时间窗口。
// 注意：重试不适用于业务性错误（如参数校验失败），fn 应自行判断进行错误分类。
func Retry(ctx context.Context, cfg RetryConfig, fn RetryableFunc) error {
	if cfg.MaxAttempts < 1 {
		cfg.MaxAttempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if attempt > 0 {
			delay := backoffDelay(attempt, cfg.BaseDelay, cfg.MaxDelay)
			select {
			case <-ctx.Done():
				return fmt.Errorf("retry cancelled after %d attempts: %w", attempt, ctx.Err())
			case <-time.After(delay):
			}
		}

		if err := fn(ctx); err != nil {
			lastErr = err
			continue
		}
		return nil
	}

	return fmt.Errorf("exhausted %d retry attempts: %w", cfg.MaxAttempts, lastErr)
}

// backoffDelay calculates exponential backoff delay capped at maxDelay.
func backoffDelay(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
	delay := time.Duration(math.Pow(2, float64(attempt-1))) * baseDelay
	if delay > maxDelay {
		delay = maxDelay
	}
	return delay
}
