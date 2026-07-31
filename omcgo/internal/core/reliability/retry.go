// Package reliability 已在 circuit_breaker.go 中声明包注释。
package reliability

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"
)

// ErrPermanent 标记"不应重试"的错误。业务侧应把明确的永久性失败（如引用的实体
// 根本不存在、参数校验失败等业务性错误，而非网络抖动/DB 连接失败等瞬时问题）用
// fmt.Errorf("...: %w", reliability.ErrPermanent) 包装后返回。Retry 命中后立即
// 停止重试并原样返回该错误，不等 MaxAttempts 耗尽，调用方可继续用 errors.Is 判断。
var ErrPermanent = errors.New("reliability: permanent error, do not retry")

// ErrDeferred 标记“当前不可处理，但应由持久化事件总线延迟重投”的错误。
// Retry 遇到它时不做进程内快速重试，避免在外部状态尚未就绪时放大数据库压力；
// 上层 runner 也不会写 DLQ，由 EventBus 保留原消息并按投递退避重试。
var ErrDeferred = errors.New("reliability: deferred error, retry via durable delivery")

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
// 注意：重试不适用于业务性错误（如参数校验失败），fn 应自行判断进行错误分类——
// 判定为永久失败时返回包装了 ErrPermanent 的错误，Retry 会立即短路返回，不再重试。
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
			if errors.Is(err, ErrPermanent) || errors.Is(err, ErrDeferred) {
				// 永久性或需交给持久队列延迟重投的错误，都不应在进程内快速重试。
				// 不再等待剩余 attempts，也不用 "exhausted N attempts" 包一层
				// 掩盖原始错误链，保证上层 errors.Is 仍然成立。
				return lastErr
			}
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
