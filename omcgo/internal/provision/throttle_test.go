package provision

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// newThrottleEngine 构造一个仅含 redisClient + metrics + logger 的最小引擎，
// 用于直接测试 throttleSetNX（MEDIUM-19）。
func newThrottleEngine(rdb redis.UniversalClient, m *Metrics, logger *zap.Logger) *ProvisioningEngine {
	return &ProvisioningEngine{redisClient: rdb, metrics: m, logger: logger}
}

// TestThrottleSetNX_AcquireThenThrottle 验证首次拿到 token（acquired=true），
// TTL 内重复同 key 被节流（acquired=false），throttle 始终可用。
func TestThrottleSetNX_AcquireThenThrottle(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	metrics := NewMetrics(nil)
	eng := newThrottleEngine(rdb, metrics, zap.NewNop())

	acquired, avail := eng.throttleSetNX(context.Background(), "provision:online_sync:dev-1", "device_online", 60*time.Second)
	require.True(t, acquired, "首次应拿到 token")
	require.True(t, avail, "Redis 健康时 throttle 应可用")

	acquired2, avail2 := eng.throttleSetNX(context.Background(), "provision:online_sync:dev-1", "device_online", 60*time.Second)
	require.False(t, acquired2, "TTL 内重复 key 应被节流")
	require.True(t, avail2)

	if got := testutil.ToFloat64(metrics.RedisThrottleFailuresTotal.WithLabelValues("device_online")); got != 0 {
		t.Fatalf("健康路径不应有失败计数，实际 %v", got)
	}
}

// TestThrottleSetNX_FailOpenOnRedisDown 验证 Redis 不可达时重试耗尽后 fail-open：
// 返回 (acquired=true, available=false)，打点 + Error 日志（运维可见节流失效）。
func TestThrottleSetNX_FailOpenOnRedisDown(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	mr.Close() // 立即关闭，模拟 Redis 故障

	core, logs := observer.New(zap.ErrorLevel)
	metrics := NewMetrics(nil)
	eng := newThrottleEngine(rdb, metrics, zap.New(core))

	// 用带超时的 ctx 限制本测试在 Redis 全程不可达时也能快速返回。
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	acquired, avail := eng.throttleSetNX(ctx, "provision:online_sync:dev-2", "device_online", 60*time.Second)

	require.True(t, acquired, "fail-open：节流失效时应放行")
	require.False(t, avail, "Redis 故障时 throttle 应标记不可用")

	if got := testutil.ToFloat64(metrics.RedisThrottleFailuresTotal.WithLabelValues("device_online")); got != 1 {
		t.Fatalf("失败路径应打点 1 次，实际 %v", got)
	}
	if got := logs.FilterMessage("redis throttling disabled, duplicates possible").Len(); got != 1 {
		t.Fatalf("失败路径应记 1 条 Error 日志，实际 %d", got)
	}
}

// TestThrottleSetNX_NilClient 验证未注入 Redis 时直接放行且不打点。
func TestThrottleSetNX_NilClient(t *testing.T) {
	metrics := NewMetrics(nil)
	eng := newThrottleEngine(nil, metrics, zap.NewNop())

	acquired, avail := eng.throttleSetNX(context.Background(), "k", "device_online", time.Minute)
	require.True(t, acquired)
	require.False(t, avail)
	if got := testutil.ToFloat64(metrics.RedisThrottleFailuresTotal.WithLabelValues("device_online")); got != 0 {
		t.Fatalf("nil client 不应打失败点，实际 %v", got)
	}
}
