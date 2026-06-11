package acs

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLocalAdmissionController(t *testing.T) {
	ctx := context.Background()
	ac := NewAdmissionController(2)

	assert.True(t, ac.Acquire(ctx, "s1"))
	assert.Equal(t, int64(1), ac.Current(ctx))

	assert.True(t, ac.Acquire(ctx, "s2"))
	assert.Equal(t, int64(2), ac.Current(ctx))

	// Should be denied — at cap.
	assert.False(t, ac.Acquire(ctx, "s3"))
	assert.Equal(t, int64(2), ac.Current(ctx))

	ac.Release(ctx, "s2")
	assert.Equal(t, int64(1), ac.Current(ctx))

	// Now should be admitted.
	assert.True(t, ac.Acquire(ctx, "s3"))
}

// TestLocalAdmission_ReleaseNoUnderflow 验证本地控制器 Release 不会下溢为负。
func TestLocalAdmission_ReleaseNoUnderflow(t *testing.T) {
	ctx := context.Background()
	ac := NewAdmissionController(5)
	ac.Release(ctx, "never-acquired")
	ac.Release(ctx, "never-acquired")
	assert.Equal(t, int64(0), ac.Current(ctx))
}

// newTestRedisAdmission 创建一个共享 miniredis 上的 Redis 准入控制器。
func newTestRedisAdmission(t *testing.T, mr *miniredis.Miniredis, max int64, ttl time.Duration) AdmissionController {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })
	ac := NewRedisAdmissionController(rdb, max, zap.NewNop()).(*redisAdmissionController)
	if ttl > 0 {
		ac.SetSlotTTL(ttl)
	}
	return ac
}

// TestRedisAdmission_GlobalCapAcrossInstances 验证两个共享同一 Redis 的准入控制器实例
// 强制执行「全局」上限（而非 N×max），且跨实例 Acquire/Release 严格配对。
func TestRedisAdmission_GlobalCapAcrossInstances(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)

	const maxGlobal = 3
	// 模拟无亲和部署：两个独立 ACS 实例各持有自己的控制器，但共享同一 Redis。
	a := newTestRedisAdmission(t, mr, maxGlobal, 0)
	b := newTestRedisAdmission(t, mr, maxGlobal, 0)

	// 跨实例累计申请 3 个槽位 —— 应全部成功（全局共 3）。
	assert.True(t, a.Acquire(ctx, "sess-1"))
	assert.True(t, b.Acquire(ctx, "sess-2"))
	assert.True(t, a.Acquire(ctx, "sess-3"))

	// 第 4 个无论落在哪个实例都必须被拒（全局上限，不是每实例 3）。
	assert.False(t, b.Acquire(ctx, "sess-4"), "global cap must reject the 4th, not allow N×max")
	assert.False(t, a.Acquire(ctx, "sess-5"))

	// 两个实例看到的全局活跃数一致 = 3。
	assert.Equal(t, int64(maxGlobal), a.Current(ctx))
	assert.Equal(t, int64(maxGlobal), b.Current(ctx))

	// 在 B 上释放 A 申请的会话（跨实例配对）→ 腾出一个槽位。
	b.Release(ctx, "sess-1")
	assert.Equal(t, int64(2), a.Current(ctx))

	// 现在第 4 个可以进来。
	assert.True(t, a.Acquire(ctx, "sess-4"))
	assert.Equal(t, int64(maxGlobal), a.Current(ctx))
}

// TestRedisAdmission_TTLReclaimsLostRelease 验证丢失的 Release（实例崩溃）由 TTL 过期分
// 自愈回收：槽位过期后全局计数自动回落，不会永久占满，且无 false 503。
//
// 准入槽位用 sorted set 分数（非 Redis key TTL）记过期，故用可注入时钟确定性推进，
// 而非 miniredis.FastForward（后者只推进 key TTL，不影响 ZSET 分数）。
func TestRedisAdmission_TTLReclaimsLostRelease(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)

	const maxGlobal = 2
	slotTTL := 30 * time.Second
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })
	ac := NewRedisAdmissionController(rdb, maxGlobal, zap.NewNop()).(*redisAdmissionController)
	ac.SetSlotTTL(slotTTL)
	clock := time.Now()
	ac.now = func() time.Time { return clock }

	require.True(t, ac.Acquire(ctx, "lost-1"))
	require.True(t, ac.Acquire(ctx, "lost-2"))
	// 满了：第 3 个被拒。
	assert.False(t, ac.Acquire(ctx, "blocked-3"))
	assert.Equal(t, int64(2), ac.Current(ctx))

	// 模拟两个实例都崩溃、Release 永远没发出 —— 时间推进超过槽位 TTL。
	clock = clock.Add(slotTTL + time.Second)

	// 过期槽位被 admitScript 的 ZREMRANGEBYSCORE 自愈回收：计数回到 0，无 false 503。
	assert.Equal(t, int64(0), ac.Current(ctx))
	assert.True(t, ac.Acquire(ctx, "after-ttl"), "TTL must reclaim lost slots, no false 503")
}

// TestRedisAdmission_AcquireReleaseSumReturnsToZero 验证大量配对 Acquire/Release 后
// 全局计数精确回到 0（无下溢、无残留）。
func TestRedisAdmission_AcquireReleaseSumReturnsToZero(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	a := newTestRedisAdmission(t, mr, 1000, 0)

	const n = 50
	for i := 0; i < n; i++ {
		require.True(t, a.Acquire(ctx, fmt.Sprintf("s-%d", i)))
	}
	assert.Equal(t, int64(n), a.Current(ctx))
	for i := 0; i < n; i++ {
		a.Release(ctx, fmt.Sprintf("s-%d", i))
	}
	assert.Equal(t, int64(0), a.Current(ctx))

	// 重复 Release 同一 sessionID 幂等，不会把计数压到负。
	a.Release(ctx, "s-0")
	a.Release(ctx, "s-0")
	assert.Equal(t, int64(0), a.Current(ctx))
}

// TestRedisAdmission_DuplicateAcquireSameSessionID 验证同一 sessionID 重复 Acquire 不会
// 重复占用两个槽位（ZADD 同成员只更新分数）。
func TestRedisAdmission_DuplicateAcquireSameSessionID(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)
	a := newTestRedisAdmission(t, mr, 5, 0)

	require.True(t, a.Acquire(ctx, "dup"))
	require.True(t, a.Acquire(ctx, "dup")) // 同成员，ZADD 仅更新分数
	assert.Equal(t, int64(1), a.Current(ctx), "same sessionID must occupy at most one slot")
}
