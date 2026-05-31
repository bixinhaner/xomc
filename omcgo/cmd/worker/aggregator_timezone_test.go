package main

import (
	"testing"
	"time"

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

// 验收 1：daily 桶 start/end 落业务时区本地零点（北京 00:00 = UTC 前一日 16:00）。
// 注入跨 UTC/本地零点的 now：UTC 2026-05-30 20:00 = 北京 2026-05-31 04:00。
func TestDailyWindow_LocalMidnight(t *testing.T) {
	loc := loadShanghai(t)
	dailyWindow := windowFor(t, loc, aggregator.JobTypeDaily)

	now := time.Date(2026, 5, 30, 20, 0, 0, 0, time.UTC) // = 北京 5-31 04:00
	start, end := dailyWindow(now)

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
	weeklyWindow := windowFor(t, loc, aggregator.JobTypeWeekly)

	now := time.Date(2026, 6, 2, 20, 0, 0, 0, time.UTC) // = 北京 6-03（周三）04:00
	start, end := weeklyWindow(now)

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
	monthlyWindow := windowFor(t, loc, aggregator.JobTypeMonthly)

	now := time.Date(2026, 5, 31, 20, 0, 0, 0, time.UTC) // = 北京 6-01 04:00
	start, end := monthlyWindow(now)

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

// 验收 5：loc 解析失败回落 UTC 不 panic；空值默认 Asia/Shanghai。
func TestResolvePMTimezone_FallbackAndDefault(t *testing.T) {
	logger := zap.NewNop()

	// 空值 → Asia/Shanghai
	loc := resolvePMTimezone("", logger)
	require.Equal(t, "Asia/Shanghai", loc.String())

	// 非法时区 → 回落 UTC，不 panic
	require.NotPanics(t, func() {
		bad := resolvePMTimezone("Not/AZone", logger)
		require.Equal(t, time.UTC, bad)
	})

	// 合法时区原样返回
	good := resolvePMTimezone("Asia/Shanghai", logger)
	require.Equal(t, "Asia/Shanghai", good.String())
}
