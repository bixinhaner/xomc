package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newMiniRedis(t *testing.T) (*miniredis.Miniredis, redis.UniversalClient) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return mr, rdb
}

// TestFixedWindow_AllowsUpToLimitThenRejects 验证窗口内放行到 limit 次，第 limit+1 次被拒。
func TestFixedWindow_AllowsUpToLimitThenRejects(t *testing.T) {
	_, rdb := newMiniRedis(t)
	lim := NewFixedWindowLimiter(rdb, "test:device", 3, time.Minute)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		allowed, count, err := lim.Allow(ctx, "SN-1")
		require.NoError(t, err)
		require.True(t, allowed, "第 %d 次应放行", i)
		require.Equal(t, int64(i), count)
	}

	allowed, count, err := lim.Allow(ctx, "SN-1")
	require.NoError(t, err)
	require.False(t, allowed, "超过 limit 应被拒绝")
	require.Equal(t, int64(4), count)
}

// TestFixedWindow_KeysIsolated 验证不同 key 各自独立计数（per-device 隔离）。
func TestFixedWindow_KeysIsolated(t *testing.T) {
	_, rdb := newMiniRedis(t)
	lim := NewFixedWindowLimiter(rdb, "test:device", 1, time.Minute)
	ctx := context.Background()

	allowed, _, err := lim.Allow(ctx, "SN-A")
	require.NoError(t, err)
	require.True(t, allowed)

	// SN-A 已用满，但 SN-B 仍应放行。
	allowedA, _, _ := lim.Allow(ctx, "SN-A")
	require.False(t, allowedA, "SN-A 第二次应被拒")
	allowedB, _, err := lim.Allow(ctx, "SN-B")
	require.NoError(t, err)
	require.True(t, allowedB, "SN-B 不受 SN-A 计数影响")
}

// TestFixedWindow_WindowResets 验证窗口过期后计数重置。
func TestFixedWindow_WindowResets(t *testing.T) {
	mr, rdb := newMiniRedis(t)
	lim := NewFixedWindowLimiter(rdb, "test:device", 1, time.Minute)
	ctx := context.Background()

	allowed, _, _ := lim.Allow(ctx, "SN-1")
	require.True(t, allowed)
	rejected, _, _ := lim.Allow(ctx, "SN-1")
	require.False(t, rejected)

	// 推进 70s 让窗口桶滚动 + TTL 过期。
	mr.FastForward(70 * time.Second)

	allowedAgain, count, err := lim.Allow(ctx, "SN-1")
	require.NoError(t, err)
	require.True(t, allowedAgain, "新窗口应重新放行")
	require.Equal(t, int64(1), count)
}

// TestFixedWindow_FailOpenWithoutRedis 验证未注入 redis 时放行（dev/test 降级）。
func TestFixedWindow_FailOpenWithoutRedis(t *testing.T) {
	lim := NewFixedWindowLimiter(nil, "test:device", 1, time.Minute)
	allowed, _, err := lim.Allow(context.Background(), "SN-1")
	require.NoError(t, err)
	require.True(t, allowed)
}

// TestFixedWindow_ZeroLimitDisables 验证 limit<=0 时不限流。
func TestFixedWindow_ZeroLimitDisables(t *testing.T) {
	_, rdb := newMiniRedis(t)
	lim := NewFixedWindowLimiter(rdb, "test:device", 0, time.Minute)
	for i := 0; i < 10; i++ {
		allowed, _, err := lim.Allow(context.Background(), "SN-1")
		require.NoError(t, err)
		require.True(t, allowed)
	}
}
