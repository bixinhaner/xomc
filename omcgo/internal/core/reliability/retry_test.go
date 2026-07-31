package reliability

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetry_SuccessOnFirstAttempt(t *testing.T) {
	var calls int32
	err := Retry(context.Background(), DefaultRetryConfig(), func(_ context.Context) error {
		atomic.AddInt32(&calls, 1)
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestRetry_SuccessAfterFailures(t *testing.T) {
	var calls int32
	cfg := RetryConfig{MaxAttempts: 3, BaseDelay: 10 * time.Millisecond, MaxDelay: 100 * time.Millisecond}

	err := Retry(context.Background(), cfg, func(_ context.Context) error {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			return errors.New("transient error")
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&calls))
}

func TestRetry_ExhaustedAttempts(t *testing.T) {
	cfg := RetryConfig{MaxAttempts: 2, BaseDelay: 10 * time.Millisecond, MaxDelay: 100 * time.Millisecond}

	err := Retry(context.Background(), cfg, func(_ context.Context) error {
		return errors.New("permanent error")
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "exhausted 2 retry attempts")
	assert.Contains(t, err.Error(), "permanent error")
}

func TestRetry_PermanentErrorShortCircuits(t *testing.T) {
	var calls int32
	cfg := RetryConfig{MaxAttempts: 5, BaseDelay: 10 * time.Millisecond, MaxDelay: 100 * time.Millisecond}

	err := Retry(context.Background(), cfg, func(_ context.Context) error {
		atomic.AddInt32(&calls, 1)
		return fmt.Errorf("device not found: %w", ErrPermanent)
	})

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrPermanent), "returned error should still unwrap to ErrPermanent")
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "fn must be called exactly once, no retry")
	assert.NotContains(t, err.Error(), "exhausted", "short-circuit path must not use the exhausted-attempts wrapping")
}

func TestRetry_DeferredErrorShortCircuitsInnerRetry(t *testing.T) {
	attempts := 0
	err := Retry(context.Background(), RetryConfig{
		MaxAttempts: 5,
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Millisecond,
	}, func(context.Context) error {
		attempts++
		return fmt.Errorf("waiting for device registration: %w", ErrDeferred)
	})

	require.ErrorIs(t, err, ErrDeferred)
	assert.Equal(t, 1, attempts, "durable event redelivery owns deferred retries")
}

func TestRetry_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := RetryConfig{MaxAttempts: 5, BaseDelay: 1 * time.Second, MaxDelay: 10 * time.Second}

	var calls int32
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := Retry(ctx, cfg, func(_ context.Context) error {
		atomic.AddInt32(&calls, 1)
		return errors.New("fail")
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "retry cancelled")
}

func TestRetry_ZeroMaxAttempts(t *testing.T) {
	cfg := RetryConfig{MaxAttempts: 0, BaseDelay: 10 * time.Millisecond, MaxDelay: 100 * time.Millisecond}

	var calls int32
	err := Retry(context.Background(), cfg, func(_ context.Context) error {
		atomic.AddInt32(&calls, 1)
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestBackoffDelay(t *testing.T) {
	base := 1 * time.Second
	max := 30 * time.Second

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 1 * time.Second},  // 2^0 * 1s = 1s
		{2, 2 * time.Second},  // 2^1 * 1s = 2s
		{3, 4 * time.Second},  // 2^2 * 1s = 4s
		{4, 8 * time.Second},  // 2^3 * 1s = 8s
		{5, 16 * time.Second}, // 2^4 * 1s = 16s
		{6, 30 * time.Second}, // 2^5 * 1s = 32s → capped at 30s
	}

	for _, tt := range tests {
		got := backoffDelay(tt.attempt, base, max)
		assert.Equal(t, tt.expected, got, "attempt %d", tt.attempt)
	}
}
