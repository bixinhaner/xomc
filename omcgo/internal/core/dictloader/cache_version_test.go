package dictloader

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMiniredis(t *testing.T) (redis.UniversalClient, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb, mr
}

func TestCacheVersion_Increment(t *testing.T) {
	rdb, _ := newMiniredis(t)
	cv := NewCacheVersion(rdb, "test", time.Second, nil)

	n, err := cv.Increment(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(1), n)
	assert.Equal(t, int64(1), cv.Current())

	n, err = cv.Increment(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)
	assert.Equal(t, int64(2), cv.Current())
}

func TestCacheVersion_Watch_FiresOnBump(t *testing.T) {
	// Simulates a "remote" bump: another instance INCRs the counter; this
	// instance's Watch goroutine must observe the change and fire OnBump.
	// Self-increments via cv.Increment update cv.local synchronously and do
	// not re-trigger OnBump on the same instance — the writer is expected
	// to invalidate its own L1 directly.
	rdb, mr := newMiniredis(t)
	cv := NewCacheVersion(rdb, "test", 10*time.Millisecond, nil)

	var hits int32
	cv.OnBump(func() { atomic.AddInt32(&hits, 1) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cv.Watch(ctx)

	// Let Watch seed local=0 (key still absent → redis.Nil → no bump).
	time.Sleep(40 * time.Millisecond)

	// Simulate a foreign write: bump directly via miniredis without touching cv.local.
	require.NoError(t, mr.Set(cv.Key(), "1"))

	deadline := time.After(time.Second)
	for atomic.LoadInt32(&hits) == 0 {
		select {
		case <-deadline:
			t.Fatal("OnBump callback never fired")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	assert.GreaterOrEqual(t, int(atomic.LoadInt32(&hits)), 1)
	assert.Equal(t, int64(1), cv.Current())
}

func TestCacheVersion_SelfIncrement_DoesNotFireOwnBump(t *testing.T) {
	// Writer-side semantics: cv.Increment updates cv.local in lock-step with
	// the Redis write, so the writer's own Watch loop must NOT re-fire OnBump
	// for its own increments (avoiding self-triggered cache invalidation).
	rdb, _ := newMiniredis(t)
	cv := NewCacheVersion(rdb, "test", 10*time.Millisecond, nil)

	var hits int32
	cv.OnBump(func() { atomic.AddInt32(&hits, 1) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cv.Watch(ctx)

	time.Sleep(40 * time.Millisecond)

	for i := 0; i < 5; i++ {
		_, err := cv.Increment(ctx)
		require.NoError(t, err)
	}

	time.Sleep(80 * time.Millisecond)
	assert.Zero(t, atomic.LoadInt32(&hits), "self-increments must not fire OnBump on the same instance")
}

func TestCacheVersion_Watch_RespectsContext(t *testing.T) {
	rdb, _ := newMiniredis(t)
	cv := NewCacheVersion(rdb, "test", 10*time.Millisecond, nil)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		cv.Watch(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Watch did not return after ctx cancel")
	}
}

func TestCacheVersion_NilRedis(t *testing.T) {
	cv := NewCacheVersion(nil, "test", time.Second, nil)
	_, err := cv.Increment(context.Background())
	require.Error(t, err)
}

func TestCacheVersion_Key(t *testing.T) {
	rdb, _ := newMiniredis(t)
	cv := NewCacheVersion(rdb, "param-model", time.Second, nil)
	assert.Equal(t, "param-model:cache_version", cv.Key())
}

func TestCacheVersion_OnBump_Nil_NoOp(t *testing.T) {
	rdb, _ := newMiniredis(t)
	cv := NewCacheVersion(rdb, "test", 10*time.Millisecond, nil)
	cv.OnBump(nil)
	// Just ensure no panic; subsequent bumps still work without callbacks.

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go cv.Watch(ctx)
	_, err := cv.Increment(ctx)
	require.NoError(t, err)
}

func TestCacheVersion_PollEvery_Default(t *testing.T) {
	rdb, _ := newMiniredis(t)
	cv := NewCacheVersion(rdb, "test", 0, nil)
	assert.NotNil(t, cv)
	// Implementation detail: the constructor substitutes a default; we cannot
	// observe the duration directly but we can confirm it is non-blocking.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	cv.Watch(ctx) // should return when ctx expires
}

func TestCacheVersion_GarbledRemoteValue(t *testing.T) {
	// If a foreign process writes a non-numeric value to the key, refresh
	// must log a warning and not panic; the local counter stays put.
	rdb, mr := newMiniredis(t)
	cv := NewCacheVersion(rdb, "test", 10*time.Millisecond, nil)

	require.NoError(t, mr.Set(cv.Key(), "not-a-number"))

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()
	cv.Watch(ctx)

	assert.Equal(t, int64(0), cv.Current())
}
