package event

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMiniredisForDedup(t *testing.T) (redis.UniversalClient, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb, mr
}

func TestDeduper_FirstTime_FreshEvent(t *testing.T) {
	rdb, _ := newMiniredisForDedup(t)
	d := NewDeduper(rdb, time.Hour, nil)

	first, err := d.FirstTime(context.Background(), "ns", "evt-1")
	require.NoError(t, err)
	assert.True(t, first, "first call must return true")
}

func TestDeduper_FirstTime_RepeatedEvent(t *testing.T) {
	rdb, _ := newMiniredisForDedup(t)
	d := NewDeduper(rdb, time.Hour, nil)

	first, err := d.FirstTime(context.Background(), "ns", "evt-1")
	require.NoError(t, err)
	require.True(t, first)

	second, err := d.FirstTime(context.Background(), "ns", "evt-1")
	require.NoError(t, err)
	assert.False(t, second, "second call must return false")
}

func TestDeduper_FirstTime_DifferentNamespaces(t *testing.T) {
	rdb, _ := newMiniredisForDedup(t)
	d := NewDeduper(rdb, time.Hour, nil)

	a, err := d.FirstTime(context.Background(), "ns-a", "evt-1")
	require.NoError(t, err)
	require.True(t, a)

	b, err := d.FirstTime(context.Background(), "ns-b", "evt-1")
	require.NoError(t, err)
	assert.True(t, b, "same evt id in different namespace must be treated as fresh")
}

func TestDeduper_FirstTime_EmptyEventID(t *testing.T) {
	rdb, _ := newMiniredisForDedup(t)
	d := NewDeduper(rdb, time.Hour, nil)

	first, err := d.FirstTime(context.Background(), "ns", "")
	require.NoError(t, err)
	assert.True(t, first, "empty event id falls open")

	// Repeated empty calls also fall open (no key written).
	second, err := d.FirstTime(context.Background(), "ns", "")
	require.NoError(t, err)
	assert.True(t, second, "subsequent empty event id also falls open")
}

func TestDeduper_FirstTime_TTLExpiry(t *testing.T) {
	rdb, mr := newMiniredisForDedup(t)
	d := NewDeduper(rdb, 100*time.Millisecond, nil)

	first, err := d.FirstTime(context.Background(), "ns", "evt-1")
	require.NoError(t, err)
	require.True(t, first)

	mr.FastForward(200 * time.Millisecond)

	again, err := d.FirstTime(context.Background(), "ns", "evt-1")
	require.NoError(t, err)
	assert.True(t, again, "after ttl expiry the event becomes fresh again")
}

func TestDeduper_FirstTime_RedisFailFallsOpen(t *testing.T) {
	rdb, mr := newMiniredisForDedup(t)
	d := NewDeduper(rdb, time.Hour, nil)

	mr.Close()

	first, err := d.FirstTime(context.Background(), "ns", "evt-1")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrDedupRedis))
	assert.True(t, first, "redis failure must fall open (return true)")
}

func TestDeduper_Wrap_FirstCallInvokesHandler(t *testing.T) {
	rdb, _ := newMiniredisForDedup(t)
	d := NewDeduper(rdb, time.Hour, nil)

	var calls int32
	wrapped := d.Wrap("ns", func(ctx context.Context, evt Event) error {
		atomic.AddInt32(&calls, 1)
		return nil
	})

	err := wrapped(context.Background(), Event{ID: "evt-1"})
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}

func TestDeduper_Wrap_RepeatCallSkipsHandler(t *testing.T) {
	rdb, _ := newMiniredisForDedup(t)
	d := NewDeduper(rdb, time.Hour, nil)

	var calls int32
	wrapped := d.Wrap("ns", func(ctx context.Context, evt Event) error {
		atomic.AddInt32(&calls, 1)
		return nil
	})

	require.NoError(t, wrapped(context.Background(), Event{ID: "evt-1"}))
	require.NoError(t, wrapped(context.Background(), Event{ID: "evt-1"}))
	require.NoError(t, wrapped(context.Background(), Event{ID: "evt-1"}))
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "handler must be invoked exactly once for repeated event")
}

func TestDeduper_Wrap_HandlerErrorPropagates(t *testing.T) {
	rdb, _ := newMiniredisForDedup(t)
	d := NewDeduper(rdb, time.Hour, nil)

	expected := fmt.Errorf("downstream failure")
	wrapped := d.Wrap("ns", func(ctx context.Context, evt Event) error {
		return expected
	})

	err := wrapped(context.Background(), Event{ID: "evt-1"})
	assert.ErrorIs(t, err, expected)
}

func TestDeduper_Wrap_RedisFailFallsOpenAndInvokes(t *testing.T) {
	rdb, mr := newMiniredisForDedup(t)
	d := NewDeduper(rdb, time.Hour, nil)
	mr.Close()

	var calls int32
	wrapped := d.Wrap("ns", func(ctx context.Context, evt Event) error {
		atomic.AddInt32(&calls, 1)
		return nil
	})

	require.NoError(t, wrapped(context.Background(), Event{ID: "evt-1"}))
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "handler must run when redis is down (fail-open)")
}

func TestDeduper_Wrap_DifferentEventsBothInvoke(t *testing.T) {
	rdb, _ := newMiniredisForDedup(t)
	d := NewDeduper(rdb, time.Hour, nil)

	var calls int32
	wrapped := d.Wrap("ns", func(ctx context.Context, evt Event) error {
		atomic.AddInt32(&calls, 1)
		return nil
	})

	require.NoError(t, wrapped(context.Background(), Event{ID: "evt-1"}))
	require.NoError(t, wrapped(context.Background(), Event{ID: "evt-2"}))
	assert.Equal(t, int32(2), atomic.LoadInt32(&calls))
}

func TestDeduper_WrapAfterSuccess_RetriesFailureThenDeduplicatesSuccess(t *testing.T) {
	rdb, _ := newMiniredisForDedup(t)
	d := NewDeduper(rdb, time.Hour, nil)

	var calls int32
	wrapped := d.WrapAfterSuccess("ns", func(context.Context, Event) error {
		attempt := atomic.AddInt32(&calls, 1)
		if attempt == 1 {
			return errors.New("temporary projection failure")
		}
		return nil
	})
	evt := Event{ID: "evt-retry"}

	require.Error(t, wrapped(context.Background(), evt))
	require.NoError(t, wrapped(context.Background(), evt))
	require.NoError(t, wrapped(context.Background(), evt))
	assert.Equal(t, int32(2), atomic.LoadInt32(&calls))
}

func TestDeduper_WrapAfterSuccess_RejectsConcurrentDeliveryUntilOwnerCompletes(t *testing.T) {
	rdb, _ := newMiniredisForDedup(t)
	d := NewDeduper(rdb, time.Hour, nil)
	started := make(chan struct{})
	release := make(chan struct{})
	var calls int32
	wrapped := d.WrapAfterSuccess("ns", func(context.Context, Event) error {
		atomic.AddInt32(&calls, 1)
		close(started)
		<-release
		return nil
	})
	evt := Event{ID: "evt-concurrent"}
	firstDone := make(chan error, 1)
	go func() { firstDone <- wrapped(context.Background(), evt) }()
	<-started

	require.ErrorIs(t, wrapped(context.Background(), evt), ErrDedupInProgress)
	close(release)
	require.NoError(t, <-firstDone)
	require.NoError(t, wrapped(context.Background(), evt))
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls))
}
