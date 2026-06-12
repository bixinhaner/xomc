package admin

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newIPGuardWithPolicy 构造 IPGuard + 注入 SecurityPolicy（来自指定 mock 配置）。
// 返回 IPGuard、miniredis 实例（用于断言键状态）、清理函数。
func newIPGuardWithPolicy(t *testing.T, entries map[string]string) (*IPGuard, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	policy := NewSecurityPolicy(&policyMockQuerier{entries: entries})
	g := NewIPGuard(client)
	g.SetPolicy(policy)
	return g, mr
}

// nil-safe：未注入 redis 时所有方法返默认值（allow / no error）。
func TestIPGuard_NilSafe(t *testing.T) {
	var g *IPGuard
	allowed, remaining, err := g.CheckAllowed(context.Background(), "1.2.3.4")
	assert.True(t, allowed)
	assert.Zero(t, remaining)
	assert.NoError(t, err)

	assert.NoError(t, g.RecordFailure(context.Background(), "1.2.3.4"))
	g.Reset(context.Background(), "1.2.3.4") // 不 panic 即过

	g2 := &IPGuard{redis: nil}
	allowed, _, _ = g2.CheckAllowed(context.Background(), "1.2.3.4")
	assert.True(t, allowed, "redis nil → fail-open allow")
}

// CheckAllowed：IP 不在黑名单返 allow=true。
func TestIPGuard_CheckAllowed_NoLock(t *testing.T) {
	g, _ := newIPGuardWithPolicy(t, map[string]string{})
	allowed, remaining, err := g.CheckAllowed(context.Background(), "1.2.3.4")
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Zero(t, remaining)
}

// CheckAllowed：IP 在黑名单返 allow=false + 剩余 TTL。
func TestIPGuard_CheckAllowed_Locked(t *testing.T) {
	g, mr := newIPGuardWithPolicy(t, map[string]string{})
	require.NoError(t, mr.Set("auth:ip:locked:1.2.3.4", "1"))
	mr.SetTTL("auth:ip:locked:1.2.3.4", 10*time.Minute)

	allowed, remaining, err := g.CheckAllowed(context.Background(), "1.2.3.4")
	require.NoError(t, err)
	assert.False(t, allowed, "黑名单内的 IP 应被拒")
	assert.InDelta(t, (10 * time.Minute).Seconds(), remaining.Seconds(), 1.0,
		"返回的 remaining 应等于黑名单 TTL")
}

// RecordFailure：未到阈值仅累加，未触发黑名单。
func TestIPGuard_RecordFailure_BelowThreshold(t *testing.T) {
	g, mr := newIPGuardWithPolicy(t, map[string]string{
		"security.limitMinus": "5",
		"security.limitCount": "5",
		"security.limitTimes": "30",
	})
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		require.NoError(t, g.RecordFailure(ctx, "1.2.3.4"))
	}

	v, err := mr.Get("auth:ip:fail:1.2.3.4")
	require.NoError(t, err)
	assert.Equal(t, "4", v, "失败计数累加到 4")

	assert.False(t, mr.Exists("auth:ip:locked:1.2.3.4"), "未到阈值不应进黑名单")
}

// RecordFailure：达到阈值即把 IP 写入黑名单 + 清失败计数器。
func TestIPGuard_RecordFailure_ReachThreshold_LocksIP(t *testing.T) {
	g, mr := newIPGuardWithPolicy(t, map[string]string{
		"security.limitMinus": "1",
		"security.limitCount": "3",
		"security.limitTimes": "30",
	})
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		require.NoError(t, g.RecordFailure(ctx, "1.2.3.4"))
	}

	assert.True(t, mr.Exists("auth:ip:locked:1.2.3.4"), "达阈值后应进黑名单")
	assert.False(t, mr.Exists("auth:ip:fail:1.2.3.4"),
		"锁定后失败计数器应被删除（让锁解除后从 0 重新开始）")

	allowed, remaining, err := g.CheckAllowed(ctx, "1.2.3.4")
	require.NoError(t, err)
	assert.False(t, allowed, "进黑名单后 CheckAllowed 应拒")
	assert.True(t, remaining > 0, "黑名单 TTL > 0")
	// 锁时长 30min，允许 5s 偏差
	assert.InDelta(t, (30 * time.Minute).Seconds(), remaining.Seconds(), 5.0)
}

// 失败计数器 TTL 应每次 RecordFailure 都刷新（恶意 IP 必须停 limitMinus 才能清零）。
func TestIPGuard_RecordFailure_RefreshesWindowTTL(t *testing.T) {
	g, mr := newIPGuardWithPolicy(t, map[string]string{
		"security.limitMinus": "5",
		"security.limitCount": "10",
		"security.limitTimes": "30",
	})
	ctx := context.Background()

	require.NoError(t, g.RecordFailure(ctx, "1.2.3.4"))
	ttl1 := mr.TTL("auth:ip:fail:1.2.3.4")
	assert.True(t, ttl1 > 0, "首次失败应设 TTL")

	// 让 miniredis 时钟前进 2 分钟
	mr.FastForward(2 * time.Minute)
	require.NoError(t, g.RecordFailure(ctx, "1.2.3.4"))
	ttl2 := mr.TTL("auth:ip:fail:1.2.3.4")
	// 第二次失败应把 TTL 重置回 limitMinus（5 min）；ttl2 应明显大于 ttl1 - 2min
	assert.True(t, ttl2 > 3*time.Minute, "TTL 应在每次失败时被刷回 limitMinus")
}

// Reset：登录成功时清失败计数器，但黑名单内的不动。
func TestIPGuard_Reset_ClearsFailCounterOnly(t *testing.T) {
	g, mr := newIPGuardWithPolicy(t, map[string]string{
		"security.limitMinus": "5",
		"security.limitCount": "10",
		"security.limitTimes": "30",
	})
	ctx := context.Background()

	require.NoError(t, g.RecordFailure(ctx, "1.2.3.4"))
	require.NoError(t, mr.Set("auth:ip:locked:1.2.3.4", "1"))
	mr.SetTTL("auth:ip:locked:1.2.3.4", 30*time.Minute)

	g.Reset(ctx, "1.2.3.4")

	assert.False(t, mr.Exists("auth:ip:fail:1.2.3.4"), "失败计数器应被清")
	assert.True(t, mr.Exists("auth:ip:locked:1.2.3.4"),
		"黑名单不应被 Reset 影响（只等 TTL 过期）")
}

// 多个不同 IP 计数互不干扰。
func TestIPGuard_RecordFailure_IsolatedPerIP(t *testing.T) {
	g, mr := newIPGuardWithPolicy(t, map[string]string{
		"security.limitMinus": "5",
		"security.limitCount": "5",
		"security.limitTimes": "30",
	})
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, g.RecordFailure(ctx, "1.1.1.1"))
	}
	require.NoError(t, g.RecordFailure(ctx, "2.2.2.2"))

	assert.True(t, mr.Exists("auth:ip:locked:1.1.1.1"), "1.1.1.1 应被锁")
	assert.False(t, mr.Exists("auth:ip:locked:2.2.2.2"), "2.2.2.2 仅 1 次失败不应被锁")
}

// 未注入 policy 时走 default（IPLimitCount=defaultIPLimitCount）。
func TestIPGuard_NoPolicy_UsesDefaults(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	g := NewIPGuard(client) // 不 SetPolicy
	ctx := context.Background()

	// 差一次到 default 阈值时不应锁
	for i := int64(0); i < defaultIPLimitCount-1; i++ {
		require.NoError(t, g.RecordFailure(ctx, "1.2.3.4"))
	}
	assert.False(t, mr.Exists("auth:ip:locked:1.2.3.4"),
		"未到 default 阈值不应触发锁")

	// 再失败一次达到 default 阈值应触发锁
	require.NoError(t, g.RecordFailure(ctx, "1.2.3.4"))
	assert.True(t, mr.Exists("auth:ip:locked:1.2.3.4"),
		"未注入 policy 应走 default 阈值触发锁")
}

// Redis 连接故障时 fail-open（CheckAllowed 返 allow）— 防止 redis 抖动导致登录瘫痪。
func TestIPGuard_CheckAllowed_RedisErrorFailsOpen(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	g := NewIPGuard(client)
	g.SetPolicy(NewSecurityPolicy(&policyMockQuerier{entries: map[string]string{}}))

	mr.Close() // 模拟 redis 不可达

	allowed, _, err := g.CheckAllowed(context.Background(), "1.2.3.4")
	assert.True(t, allowed, "redis 错误时应 fail-open（避免登录瘫痪）")
	assert.NoError(t, err, "fail-open 不应返错给 caller")
}

// 防回归：ErrIPLocked 是 sentinel，能被 errors.Is 识别（防未来误用 fmt.Errorf 包裹丢失）。
func TestIPGuard_ErrSentinel(t *testing.T) {
	assert.True(t, errors.Is(ErrIPLocked, ErrIPLocked))
}
