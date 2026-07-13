package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/core/systimezone"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// loadShanghai 加载业务时区；测试环境（macOS/CI Linux）系统 tzdata 或内嵌 time/tzdata 兜底均可。
func loadShanghai(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err, "Asia/Shanghai 应可加载（系统 tzdata 或内嵌 time/tzdata 兜底）")
	return loc
}

// windowFor 从 pmAggregatorCronEntries(loc) 中取出指定 jobType 的窗口函数。
func windowFor(t *testing.T, loc *time.Location, jobType string) func(now time.Time) (time.Time, time.Time) {
	t.Helper()
	for _, e := range pmAggregatorCronEntries(loc) {
		if e.jobType == jobType {
			return e.window
		}
	}
	t.Fatalf("未找到 jobType=%s 的 cron entry", jobType)
	return nil
}

func TestPMAggregatorCronEntries_OnlyHourlyCron(t *testing.T) {
	loc := loadShanghai(t)
	entries := pmAggregatorCronEntries(loc)
	require.Len(t, entries, 1)
	require.Equal(t, aggregator.JobTypeHourly, entries[0].jobType)
	require.Equal(t, "5 * * * *", entries[0].spec)
}

// 验收 1：daily 桶 start/end 落业务时区本地零点（北京 00:00 = UTC 前一日 16:00）。
// 注入跨 UTC/本地零点的 now：UTC 2026-05-30 20:00 = 北京 2026-05-31 04:00。
func TestDailyWindow_LocalMidnight(t *testing.T) {
	loc := loadShanghai(t)

	now := time.Date(2026, 5, 30, 20, 0, 0, 0, time.UTC) // = 北京 5-31 04:00
	start, end := dailyAggregationWindow(now, loc)

	// 桶 end = 本地今天零点（北京 5-31 00:00 = UTC 5-30 16:00）
	wantEnd := time.Date(2026, 5, 31, 0, 0, 0, 0, loc)
	require.True(t, end.Equal(wantEnd), "daily end 应落本地零点；got=%s want=%s", end, wantEnd)
	require.True(t, end.Equal(time.Date(2026, 5, 30, 16, 0, 0, 0, time.UTC)),
		"daily end 等价 UTC 5-30 16:00；got=%s", end.UTC())

	// 桶 start = 本地前一天零点（北京 5-30 00:00 = UTC 5-29 16:00）
	wantStart := time.Date(2026, 5, 30, 0, 0, 0, 0, loc)
	require.True(t, start.Equal(wantStart), "daily start 应落本地次前一日零点；got=%s want=%s", start, wantStart)
	require.True(t, start.Equal(time.Date(2026, 5, 29, 16, 0, 0, 0, time.UTC)),
		"daily start 等价 UTC 5-29 16:00；got=%s", start.UTC())
}

// 验收 2a：weekly 桶 start 落本地周一零点（带 +08 偏移即 UTC 前一日某时刻）。
// 2026-06-03 是周三，本周一为 2026-06-01。
func TestWeeklyWindow_LocalMonday(t *testing.T) {
	loc := loadShanghai(t)

	now := time.Date(2026, 6, 2, 20, 0, 0, 0, time.UTC) // = 北京 6-03（周三）04:00
	start, end := weeklyAggregationWindow(now, loc)

	// end = 本周一本地零点（北京 6-01 00:00 = UTC 5-31 16:00）
	wantThisMon := time.Date(2026, 6, 1, 0, 0, 0, 0, loc)
	require.True(t, end.Equal(wantThisMon), "weekly end 应落本地本周一零点；got=%s want=%s", end, wantThisMon)
	require.True(t, end.Equal(time.Date(2026, 5, 31, 16, 0, 0, 0, time.UTC)),
		"weekly end 带 +08 偏移即 UTC 前一日 16:00；got=%s", end.UTC())

	// start = 上周一本地零点（北京 5-25 00:00）
	wantPrevMon := time.Date(2026, 5, 25, 0, 0, 0, 0, loc)
	require.True(t, start.Equal(wantPrevMon), "weekly start 应落本地上周一零点；got=%s want=%s", start, wantPrevMon)
	require.Equal(t, time.Monday, start.In(loc).Weekday(), "weekly start 必须是周一")
}

// 验收 2b：monthly 桶 start 落本地月初零点（带 +08 偏移即 UTC 上月某时刻）。
func TestMonthlyWindow_LocalFirstOfMonth(t *testing.T) {
	loc := loadShanghai(t)

	now := time.Date(2026, 5, 31, 20, 0, 0, 0, time.UTC) // = 北京 6-01 04:00
	start, end := monthlyAggregationWindow(now, loc)

	// end = 本月初本地零点（北京 6-01 00:00 = UTC 5-31 16:00）
	wantThisMonth := time.Date(2026, 6, 1, 0, 0, 0, 0, loc)
	require.True(t, end.Equal(wantThisMonth), "monthly end 应落本地月初零点；got=%s want=%s", end, wantThisMonth)
	require.True(t, end.Equal(time.Date(2026, 5, 31, 16, 0, 0, 0, time.UTC)),
		"monthly end 带 +08 偏移即 UTC 上月最后一日 16:00；got=%s", end.UTC())

	// start = 上月初本地零点（北京 5-01 00:00）
	wantPrevMonth := time.Date(2026, 5, 1, 0, 0, 0, 0, loc)
	require.True(t, start.Equal(wantPrevMonth), "monthly start 应落本地上月初零点；got=%s want=%s", start, wantPrevMonth)
}

// 验收 3：hourly 整点对齐不受时区影响（成功 + 跨日/跨月/跨年边界）。
func TestHourlyWindow_UnaffectedByTimezone(t *testing.T) {
	loc := loadShanghai(t)
	hourlyWindow := windowFor(t, loc, aggregator.JobTypeHourly)

	cases := []struct {
		name      string
		now       time.Time
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name:      "普通整点内",
			now:       time.Date(2026, 5, 30, 13, 37, 0, 0, time.UTC),
			wantStart: time.Date(2026, 5, 30, 12, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 5, 30, 13, 0, 0, 0, time.UTC),
		},
		{
			name:      "跨日边界（UTC 00:00 整点）",
			now:       time.Date(2026, 5, 31, 0, 5, 0, 0, time.UTC),
			wantStart: time.Date(2026, 5, 30, 23, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "跨月边界",
			now:       time.Date(2026, 6, 1, 0, 30, 0, 0, time.UTC),
			wantStart: time.Date(2026, 5, 31, 23, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "跨年边界",
			now:       time.Date(2027, 1, 1, 0, 15, 0, 0, time.UTC),
			wantStart: time.Date(2026, 12, 31, 23, 0, 0, 0, time.UTC),
			wantEnd:   time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			start, end := hourlyWindow(c.now)
			require.True(t, start.Equal(c.wantStart), "hourly start；got=%s want=%s", start.UTC(), c.wantStart)
			require.True(t, end.Equal(c.wantEnd), "hourly end；got=%s want=%s", end.UTC(), c.wantEnd)
		})
	}
}

// stubFetcher 是 systimezone.Fetcher 的可变桩：原子换值模拟 sys_configs 改时区。
type stubFetcher struct {
	val atomic.Pointer[string]
}

func (s *stubFetcher) set(v string) { s.val.Store(&v) }
func (s *stubFetcher) fetch(_ context.Context, category, key string) (string, bool) {
	if category != systimezone.Category || key != systimezone.Key {
		return "", false
	}
	if p := s.val.Load(); p != nil {
		return *p, true
	}
	return "", false
}

// newTzManagerWithProvider 构造一个用桩 Provider 的 tzManager（不走 DB）。
func newTzManagerWithProvider(p *systimezone.Provider) *tzManager {
	m := &tzManager{provider: p, logger: zap.NewNop()}
	m.cur.Store(m.resolve(context.Background()))
	return m
}

func TestExportTimezoneProvider_UsesTzManagerProvider(t *testing.T) {
	sf := &stubFetcher{}
	sf.set("Asia/Shanghai")
	m := newTzManagerWithProvider(systimezone.New(sf.fetch, nil))

	got := exportTimezoneProvider(m)
	require.NotNil(t, got)
	require.Equal(t, "Asia/Shanghai", got.Location(context.Background()).String())
}

func TestExportTimezoneProvider_NilSafe(t *testing.T) {
	require.Nil(t, exportTimezoneProvider(nil))
	require.Nil(t, exportTimezoneProvider(&tzManager{}))
}

// 验收 5（改）：业务时区改读 sys_configs 统一源；空/非法回落 UTC 不 panic。
func TestTzManager_ResolveFromSysConfig_FallbackUTC(t *testing.T) {
	ctx := context.Background()

	// 合法时区
	sf := &stubFetcher{}
	sf.set("Asia/Tokyo")
	m := newTzManagerWithProvider(systimezone.New(sf.fetch, nil))
	require.Equal(t, "Asia/Tokyo", m.Current().String())

	// 空值 → 回落 UTC（Provider DefaultTimezone=UTC）
	sf2 := &stubFetcher{}
	sf2.set("")
	m2 := newTzManagerWithProvider(systimezone.New(sf2.fetch, nil))
	require.Equal(t, time.UTC, m2.Current())

	// 非法时区 → 回落 UTC，不 panic
	require.NotPanics(t, func() {
		sf3 := &stubFetcher{}
		sf3.set("Not/AZone")
		m3 := newTzManagerWithProvider(systimezone.New(sf3.fetch, nil))
		require.Equal(t, time.UTC, m3.Current())
	})

	// 无 Provider（无 DB）→ UTC
	m4 := &tzManager{logger: zap.NewNop()}
	m4.cur.Store(m4.resolve(ctx))
	require.Equal(t, time.UTC, m4.Current())
}

// 验收（新）：动态切换后按新 loc 切桶——改 sys_configs 时区 → reload → 同一 now 的
// daily/weekly/monthly 窗口边界按新时区零点对齐（不重启 worker，核心验收）。
func TestTzManager_DynamicSwitch_BucketsCutByNewLoc(t *testing.T) {
	ctx := context.Background()
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	require.NoError(t, err)
	shanghai := loadShanghai(t)

	// 初始时区 = Asia/Tokyo（+09:00）
	sf := &stubFetcher{}
	sf.set("Asia/Tokyo")
	// 用 0 TTL 让每次 reload 后 Location() 必重读桩值（避免 5min 缓存遮蔽）。
	m := newTzManagerWithProvider(systimezone.New(sf.fetch, nil, systimezone.WithTTL(0)))
	require.Equal(t, "Asia/Tokyo", m.Current().String())

	// daily 不再独立挂 cron；链式触发时通过 tz.Current() 计算业务时区窗口。
	dailyWindow := func(now time.Time) (time.Time, time.Time) {
		return dailyAggregationWindow(now, m.Current())
	}

	// 注入跨本地零点的 now：UTC 2026-05-30 18:00 = 东京 5-31 03:00 = 北京 5-31 02:00。
	now := time.Date(2026, 5, 30, 18, 0, 0, 0, time.UTC)

	// 切换前（东京）：daily end = 东京 5-31 00:00 = UTC 5-30 15:00。
	_, endTokyo := dailyWindow(now)
	require.True(t, endTokyo.Equal(time.Date(2026, 5, 31, 0, 0, 0, 0, tokyo)),
		"切换前 daily end 应落东京本地零点；got=%s", endTokyo)
	require.True(t, endTokyo.Equal(time.Date(2026, 5, 30, 15, 0, 0, 0, time.UTC)),
		"东京零点等价 UTC 5-30 15:00；got=%s", endTokyo.UTC())

	// === 管理员改系统时区为 Asia/Shanghai（模拟改 sys_configs + 触发 reload）===
	sf.set("Asia/Shanghai")
	changed := m.reload(ctx)
	require.True(t, changed, "时区从东京改上海应被识别为变化")
	require.Equal(t, "Asia/Shanghai", m.Current().String())

	// 切换后（上海）：同一 now，window 用新时区——daily end = 上海 5-31 00:00 = UTC 5-30 16:00。
	// 这是不重启即生效的核心断言：边界从东京零点(UTC15:00)移到上海零点(UTC16:00)。
	_, endShanghai := dailyWindow(now)
	require.True(t, endShanghai.Equal(time.Date(2026, 5, 31, 0, 0, 0, 0, shanghai)),
		"切换后 daily end 应落上海本地零点；got=%s", endShanghai)
	require.True(t, endShanghai.Equal(time.Date(2026, 5, 30, 16, 0, 0, 0, time.UTC)),
		"上海零点等价 UTC 5-30 16:00；got=%s", endShanghai.UTC())

	require.False(t, endTokyo.Equal(endShanghai),
		"切换前后 daily 桶边界必须不同（东京零点 != 上海零点）")

	// 再切回 UTC：daily end = UTC 5-31 00:00（now 当天 UTC 零点之后，端落本日零点）。
	sf.set("UTC")
	require.True(t, m.reload(ctx))
	require.Equal(t, time.UTC, m.Current())
	_, endUTC := dailyWindow(now)
	require.True(t, endUTC.Equal(time.Date(2026, 5, 30, 0, 0, 0, 0, time.UTC)),
		"切回 UTC 后 daily end 应落 UTC 当日零点(now=5-30 18:00→桶端 5-30 00:00)；got=%s", endUTC.UTC())
}

// 验收（新）：reload 时区未变则不触发重建（changed=false）。
func TestTzManager_Reload_NoChange(t *testing.T) {
	sf := &stubFetcher{}
	sf.set("Asia/Tokyo")
	m := newTzManagerWithProvider(systimezone.New(sf.fetch, nil, systimezone.WithTTL(0)))
	require.False(t, m.reload(context.Background()), "时区未变 reload 应返回 false")
	require.Equal(t, "Asia/Tokyo", m.Current().String())
}

// 验收（新·替换事件驱动）：后台轮询感知改时区——startReloadPoller 启动后周期重读
// sys_configs，管理员改桩值后无需重启即被感知、Current() 切到新时区。
//
// 此用例直接驱动生产路径（startReloadPoller→reload→Provider 重读），不再用
// ChannelEventBus 掩盖真实 NATS WorkQueue 下事件订阅会失败的问题（回合1 检查阻塞缺陷）。
func TestTzManager_ReloadPoller_DynamicNoRestart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sf := &stubFetcher{}
	sf.set("Asia/Tokyo")
	// TTL=0 让每轮 reload 后 Location() 必重读桩值（生产靠 reload 内 Invalidate 同效）。
	m := newTzManagerWithProvider(systimezone.New(sf.fetch, nil, systimezone.WithTTL(0)))
	require.Equal(t, "Asia/Tokyo", m.Current().String())

	// 用很短的轮询周期（10ms）跑真实 poller。
	m.startReloadPoller(ctx, 10*time.Millisecond)

	// 管理员改 sys_configs 时区为上海（不重启）→ 轮询应在一两拍内感知。
	sf.set("Asia/Shanghai")
	require.Eventually(t, func() bool {
		return m.Current().String() == "Asia/Shanghai"
	}, 2*time.Second, 10*time.Millisecond, "轮询应感知改时区为上海（不重启即生效）")

	// 再改回 UTC → 同样被轮询感知。
	sf.set("UTC")
	require.Eventually(t, func() bool {
		return m.Current() == time.UTC
	}, 2*time.Second, 10*time.Millisecond, "轮询应感知改时区回 UTC")
}

// 验收（新）：无 Provider（无 DB）时 startReloadPoller 不启动轮询，时区恒为启动值，
// 不 panic、不泄漏 goroutine。
func TestTzManager_ReloadPoller_NoProviderNoOp(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := &tzManager{logger: zap.NewNop()}
	m.cur.Store(m.resolve(ctx))
	require.Equal(t, time.UTC, m.Current())
	require.NotPanics(t, func() {
		m.startReloadPoller(ctx, 10*time.Millisecond)
	})
	// 即便有桩 fetcher（这里没接 Provider），无 Provider 也不重读。
	require.Equal(t, time.UTC, m.Current())
}
