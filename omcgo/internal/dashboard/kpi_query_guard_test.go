package dashboard

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/stretchr/testify/require"
)

func TestKPIQueryGuardCachesCoalescesAndExpires(t *testing.T) {
	now := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	guard := NewKPIQueryGuard(KPIQueryGuardConfig{
		QueryTimeout: time.Second, MaxConcurrent: 1, QueueTimeout: 20 * time.Millisecond,
		FreshTTL: 4*time.Minute + 30*time.Second, StaleTTL: 15 * time.Minute,
	}, nil)
	guard.now = func() time.Time { return now }

	var calls atomic.Int32
	loader := func(context.Context) (any, error) {
		calls.Add(1)
		return "value", nil
	}

	first, firstMeta, err := guard.Do(context.Background(), "series:key", loader)
	require.NoError(t, err)
	require.Equal(t, "value", first)
	require.False(t, firstMeta.Stale)

	second, _, err := guard.Do(context.Background(), "series:key", loader)
	require.NoError(t, err)
	require.Equal(t, "value", second)
	require.Equal(t, int32(1), calls.Load())

	now = now.Add(4*time.Minute + 31*time.Second)
	_, _, err = guard.Do(context.Background(), "series:key", loader)
	require.NoError(t, err)
	require.Equal(t, int32(2), calls.Load())
}

func TestKPIQueryGuardReturnsStaleOnlyWithinLimit(t *testing.T) {
	now := time.Date(2026, 7, 29, 0, 0, 0, 0, time.UTC)
	guard := NewKPIQueryGuard(KPIQueryGuardConfig{
		QueryTimeout: time.Second, MaxConcurrent: 1, QueueTimeout: 20 * time.Millisecond,
		FreshTTL: time.Minute, StaleTTL: 15 * time.Minute,
	}, nil)
	guard.now = func() time.Time { return now }

	_, _, err := guard.Do(context.Background(), "summary", func(context.Context) (any, error) {
		return "last-success", nil
	})
	require.NoError(t, err)

	now = now.Add(2 * time.Minute)
	value, metadata, err := guard.Do(context.Background(), "summary", func(context.Context) (any, error) {
		return nil, errors.New("tsdb unavailable")
	})
	require.NoError(t, err)
	require.Equal(t, "last-success", value)
	require.True(t, metadata.Stale)

	now = now.Add(14 * time.Minute)
	_, _, err = guard.Do(context.Background(), "summary", func(context.Context) (any, error) {
		return nil, errors.New("tsdb unavailable")
	})
	require.Error(t, err)
}

func TestKPIQueryGuardRejectsDifferentKeyWhenConcurrencyIsFull(t *testing.T) {
	guard := NewKPIQueryGuard(KPIQueryGuardConfig{
		QueryTimeout: time.Second, MaxConcurrent: 1, QueueTimeout: 20 * time.Millisecond,
		FreshTTL: time.Minute, StaleTTL: 15 * time.Minute,
	}, nil)
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _, _ = guard.Do(context.Background(), "first", func(context.Context) (any, error) {
			close(started)
			<-release
			return "first", nil
		})
	}()
	<-started

	_, _, err := guard.Do(context.Background(), "second", func(context.Context) (any, error) {
		return "second", nil
	})
	require.ErrorIs(t, err, commonerrors.ErrUnavailable)
	close(release)
	<-done
}

func TestKPIQueryGuardMapsDeadlineToTimeout(t *testing.T) {
	guard := NewKPIQueryGuard(KPIQueryGuardConfig{
		QueryTimeout: 20 * time.Millisecond, MaxConcurrent: 1, QueueTimeout: time.Second,
		FreshTTL: time.Minute, StaleTTL: 15 * time.Minute,
	}, nil)
	_, _, err := guard.Do(context.Background(), "slow", func(ctx context.Context) (any, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	require.ErrorIs(t, err, commonerrors.ErrTimeout)
}
