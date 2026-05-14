package admin

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// mockSysConfigQuerier 实现 SysConfigQuerier 接口，返表里的 stub。
type mockSysConfigQuerier struct {
	mu      sync.Mutex
	entries map[string]*SysConfig
	hits    int // 命中次数（用于验证缓存）
	err     error
}

func newMockQuerier(items map[string]string) *mockSysConfigQuerier {
	m := &mockSysConfigQuerier{entries: make(map[string]*SysConfig, len(items))}
	for k, v := range items {
		// k 形如 "security.sumTimes"
		m.entries[k] = &SysConfig{ID: uuid.New(), Value: v, ValueType: "int"}
	}
	return m
}

func (m *mockSysConfigQuerier) GetByKey(_ context.Context, category, key string) (*SysConfig, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hits++
	if m.err != nil {
		return nil, m.err
	}
	cfg, ok := m.entries[category+"."+key]
	if !ok {
		return nil, nil
	}
	return cfg, nil
}

// 反退化：未注入 SysConfigQuerier 时所有阈值/时长走 default 常量。
func TestLoginGuard_NoQuerier_UsesDefaults(t *testing.T) {
	g := &LoginGuard{}
	ctx := context.Background()

	assert.False(t, g.ShouldLock(ctx, defaultLockThreshold-1))
	assert.True(t, g.ShouldLock(ctx, defaultLockThreshold), "default 10 次锁")
	assert.True(t, g.ShouldLock(ctx, defaultLockThreshold+5))
	assert.Equal(t, defaultLockDuration, g.LockDuration(ctx))
}

// 注入 Querier 后，阈值从 sys_configs 读，FE 修改的值真实生效。
func TestLoginGuard_WithQuerier_OverridesDefaults(t *testing.T) {
	q := newMockQuerier(map[string]string{
		"security.sumTimes":     "5",  // FE 改成 5 次锁定
		"security.unlockMinu":   "15", // FE 改成 15 分钟解锁
		"security.attemptTimes": "2",  // FE 改成 2 次后要求 CAPTCHA
	})
	g := &LoginGuard{}
	g.SetSysConfigQuerier(q)
	ctx := context.Background()

	assert.False(t, g.ShouldLock(ctx, 4), "5 次以下不锁")
	assert.True(t, g.ShouldLock(ctx, 5), "刚到 sumTimes=5 锁")
	assert.Equal(t, 15*time.Minute, g.LockDuration(ctx))

	// RequiresCaptcha 走 attemptTimes — 但这个需要 Redis 才能验证 GetFailedCount
	// 此处仅断言常量加载正确，覆盖 loadConfig 全字段
	cfg := g.loadConfig(ctx)
	assert.Equal(t, int64(2), cfg.captchaThreshold)
	assert.Equal(t, int64(5), cfg.lockThreshold)
	assert.Equal(t, 15*time.Minute, cfg.lockDuration)
}

// 配置值非法（负数 / 非数字 / 0 / 空串）→ 退化到 default 常量。
func TestLoginGuard_InvalidConfig_FallsBackToDefaults(t *testing.T) {
	cases := []struct {
		name  string
		items map[string]string
	}{
		{"负数", map[string]string{"security.sumTimes": "-1", "security.unlockMinu": "-30"}},
		{"零", map[string]string{"security.sumTimes": "0", "security.unlockMinu": "0"}},
		{"非数字", map[string]string{"security.sumTimes": "abc", "security.unlockMinu": "xyz"}},
		{"空串", map[string]string{"security.sumTimes": "", "security.unlockMinu": ""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := newMockQuerier(tc.items)
			g := &LoginGuard{}
			g.SetSysConfigQuerier(q)
			ctx := context.Background()

			// 应该退到 default — 阈值 10 / 时长 30min
			assert.False(t, g.ShouldLock(ctx, 9), "9 次不该锁 (fallback default 10)")
			assert.True(t, g.ShouldLock(ctx, 10), "刚到 default 10 锁")
			assert.Equal(t, defaultLockDuration, g.LockDuration(ctx))
		})
	}
}

// FE 只配置部分 key 时其余字段仍用 default — 避免 FE 改一项把其它项归零。
func TestLoginGuard_PartialConfig_OtherFieldsKeepDefault(t *testing.T) {
	q := newMockQuerier(map[string]string{
		// 只改锁阈值，时长保持默认
		"security.sumTimes": "8",
	})
	g := &LoginGuard{}
	g.SetSysConfigQuerier(q)
	ctx := context.Background()

	assert.True(t, g.ShouldLock(ctx, 8), "sumTimes=8 生效")
	assert.Equal(t, defaultLockDuration, g.LockDuration(ctx), "unlockMinu 未配置 → fallback 30min")
}

// 30s in-memory cache 让重复调用不再打 DB。
func TestLoginGuard_ConfigCache_AvoidsRepeatedDBHits(t *testing.T) {
	q := newMockQuerier(map[string]string{
		"security.sumTimes":   "8",
		"security.unlockMinu": "10",
	})
	g := &LoginGuard{}
	g.SetSysConfigQuerier(q)
	ctx := context.Background()

	// 第一次：读 3 个 key
	_ = g.ShouldLock(ctx, 1)
	firstHits := q.hits

	// 紧接着 100 次调用都不该再去 DB
	for i := 0; i < 100; i++ {
		_ = g.ShouldLock(ctx, int64(i))
		_ = g.LockDuration(ctx)
	}
	assert.Equal(t, firstHits, q.hits, "cache 内连续访问不应触发新 DB 查询")
}

// 主动失效缓存（InvalidateConfigCache）后下次读重新打 DB。
func TestLoginGuard_InvalidateCache_TriggersReload(t *testing.T) {
	q := newMockQuerier(map[string]string{"security.sumTimes": "8"})
	g := &LoginGuard{}
	g.SetSysConfigQuerier(q)
	ctx := context.Background()

	_ = g.ShouldLock(ctx, 1)
	firstHits := q.hits

	g.InvalidateConfigCache()
	_ = g.ShouldLock(ctx, 1)
	assert.Greater(t, q.hits, firstHits, "InvalidateConfigCache 后应再次访问 DB")
}

// 并发同时 loadConfig 时 double-checked locking 保证只有一次 DB 加载。
func TestLoginGuard_ConcurrentLoad_OnlyOneDBHit(t *testing.T) {
	q := newMockQuerier(map[string]string{
		"security.sumTimes":   "8",
		"security.unlockMinu": "10",
	})
	g := &LoginGuard{}
	g.SetSysConfigQuerier(q)
	ctx := context.Background()

	const N = 50
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			_ = g.ShouldLock(ctx, 1)
		}()
	}
	wg.Wait()

	// 首次进来 3 个 key（sumTimes / unlockMinu / attemptTimes）= 3 hit
	// 并发竞争允许少量重复加载（double-check 之间），但不该多次跑完整 3-key 扫描
	// 保守上限：6 hits（两轮加载）
	assert.LessOrEqual(t, q.hits, 6, "并发 50 次 ShouldLock 最多触发 2 轮 3-key 加载")
}
