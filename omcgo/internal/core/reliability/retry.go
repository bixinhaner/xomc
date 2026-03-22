package reliability

import (
	"context"
	"fmt"
	"math"
	"time"
)

// RetryConfig defines the configuration for retry behavior.
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

// Retry executes fn with exponential backoff according to cfg.
// It returns nil on success or the last error after all attempts are exhausted.
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
