package redisx_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/components/redisx"
)

func TestRetry_SucceedsFirstTry(t *testing.T) {
	calls := 0
	err := redisx.Retry(context.Background(), redisx.DefaultRetry, func(_ context.Context) error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestRetry_EventuallySucceeds(t *testing.T) {
	calls := 0
	cfg := redisx.RetryConfig{MaxAttempts: 3, InitialWait: 1 * time.Millisecond, MaxWait: 2 * time.Millisecond, Multiplier: 2}
	err := redisx.Retry(context.Background(), cfg, func(_ context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
}

func TestRetry_GivesUpAfterMax(t *testing.T) {
	calls := 0
	cfg := redisx.RetryConfig{MaxAttempts: 2, InitialWait: 1 * time.Millisecond, Multiplier: 2}
	sentinel := errors.New("fail")
	err := redisx.Retry(context.Background(), cfg, func(_ context.Context) error {
		calls++
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want %v", err, sentinel)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2", calls)
	}
}

func TestRetry_PermanentErrorShortCircuits(t *testing.T) {
	calls := 0
	cfg := redisx.RetryConfig{MaxAttempts: 5, InitialWait: 1 * time.Millisecond}
	err := redisx.Retry(context.Background(), cfg, func(_ context.Context) error {
		calls++
		return fmt.Errorf("bad input: %w", redisx.ErrPermanent)
	})
	if !errors.Is(err, redisx.ErrPermanent) {
		t.Fatalf("err = %v, want ErrPermanent", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1 (permanent short-circuits)", calls)
	}
}

func TestRetry_ContextCancelStops(t *testing.T) {
	cfg := redisx.RetryConfig{MaxAttempts: 5, InitialWait: 50 * time.Millisecond, Multiplier: 2}
	ctx, cancel := context.WithCancel(context.Background())
	// 在第一次失败后的退避期间取消。
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()
	calls := 0
	err := redisx.Retry(ctx, cfg, func(_ context.Context) error {
		calls++
		return errors.New("transient")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}
