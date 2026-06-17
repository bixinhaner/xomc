package adhoc

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// ISSUE-398：持续任务窗口应被收窄到「最近一个已完成周期」那一格（各粒度）。

func TestLastCompletedBucket_PerGranularity(t *testing.T) {
	loc := time.UTC
	// 固定一个「正在进行」的时刻：2026-06-16（周二）18:44。
	now := time.Date(2026, 6, 16, 18, 44, 0, 0, loc)

	// #479 改动四：在「上一完整格」基础上再回退 pipelineLagBuckets(=1) 格补偿管线延迟，
	// 落到聚合 cron 必已跑完的那一格。
	cases := []struct {
		name string
		g    metrics.Granularity
		want time.Time
	}{
		{
			name: "hourly: 18:44 跑 → 上一完整小时 17:00 再回退 1 格 → 16:00",
			g:    metrics.GranularityHourly,
			want: time.Date(2026, 6, 16, 16, 0, 0, 0, loc),
		},
		{
			name: "15min: 18:44 跑 → 当前格 18:30 起 → 上一格 18:15 再回退 1 格 → 18:00",
			g:    metrics.Granularity15Min,
			want: time.Date(2026, 6, 16, 18, 0, 0, 0, loc),
		},
		{
			name: "daily: 今天进行中 → 上一完整天 06-15 再回退 1 天 → 06-14 00:00",
			g:    metrics.GranularityDaily,
			want: time.Date(2026, 6, 14, 0, 0, 0, 0, loc),
		},
		{
			name: "weekly: 本周一 06-15 → 上一完整周 06-08 再回退 1 周 → 06-01 00:00",
			g:    metrics.GranularityWeekly,
			want: time.Date(2026, 6, 1, 0, 0, 0, 0, loc),
		},
		{
			name: "monthly: 6月进行中 → 上一完整月 05-01 再回退 1 月 → 04-01 00:00",
			g:    metrics.GranularityMonthly,
			want: time.Date(2026, 4, 1, 0, 0, 0, 0, loc),
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := lastCompletedBucket(c.g, now, loc)
			assert.True(t, got.Equal(c.want), "got %v want %v", got, c.want)
		})
	}
}

func TestLastCompletedBucket_NilLocDefaultsUTC(t *testing.T) {
	now := time.Date(2026, 6, 16, 18, 44, 0, 0, time.UTC)
	got := lastCompletedBucket(metrics.GranularityDaily, now, nil)
	// 上一完整天 06-15 再回退 1 格（管线延迟补偿）→ 06-14。
	want := time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC)
	assert.True(t, got.Equal(want), "got %v want %v", got, want)
}

// #479 改动四：持续任务在「上一格刚结束、但其聚合 cron（结束后过 5 分钟）尚未跑完」的空窗里运行时，
// 回看口径必须跳到「确定已汇总好」的更前一格，而非扑空那个还没汇总的上一格。
//
// 红绿：旧口径（只回看上一完整格）会落在 [18:00,19:00) 这格——它的设备级/组级聚合 cron 在 19:05 才跑，
// 此刻（19:02）尚未汇总 → 扑空（复现 #479 根因 4）；新口径再回退 1 格落到 [17:00,18:00)，其聚合 18:05 早已完成。
func TestLastCompletedBucket_SkipsBucketWhoseAggregationNotYetRun(t *testing.T) {
	loc := time.UTC
	// 19:02 跑：本格 19:00-20:00 进行中；上一格 18:00-19:00 刚结束、其聚合 cron 19:05 尚未触发。
	now := time.Date(2026, 6, 16, 19, 2, 0, 0, loc)

	got := lastCompletedBucket(metrics.GranularityHourly, now, loc)

	// 绿：落到 17:00（其聚合 18:05 早已完成，确定已汇总好）。
	wantAggregated := time.Date(2026, 6, 16, 17, 0, 0, 0, loc)
	assert.True(t, got.Equal(wantAggregated), "应回看到确定已汇总好的 17:00，实际 %v", got)

	// 红：绝不落在「上一格 18:00」——它此刻还没汇总（聚合 cron 19:05 才跑）。
	notYetAggregated := time.Date(2026, 6, 16, 18, 0, 0, 0, loc)
	assert.False(t, got.Equal(notYetAggregated), "不应落在尚未汇总的上一格 18:00")
}

func TestIsContinuous(t *testing.T) {
	nonZero := time.Date(2026, 6, 16, 10, 0, 0, 0, time.UTC)

	// Mode==continuous → true（即使窗口非零也按持续）。
	assert.True(t, isContinuous(&Task{Mode: ModeContinuous, WindowStart: nonZero, WindowEnd: nonZero}))
	// 零窗口（创建清窗存 NULL）→ true。
	assert.True(t, isContinuous(&Task{Mode: ""}))
	// oneshot + 非零窗口 → false。
	assert.False(t, isContinuous(&Task{Mode: ModeOneshot, WindowStart: nonZero, WindowEnd: nonZero}))
}

// 持续任务首跑/滚动：每个粒度各自被收窄到最近一格（窗口被覆盖为 桶起点==桶起点）。
func TestQueryAndConvert_ContinuousOverridesWindowPerGranularity(t *testing.T) {
	loc := time.UTC
	fixedNow := time.Date(2026, 6, 16, 18, 44, 0, 0, loc)

	aggr := &stubAggr{}
	e := NewExecutor(aggr, &stubRepo{}, nil, nil).SetLocation(loc)
	e.now = func() time.Time { return fixedNow }

	task := &Task{
		ID:            uuid.New(),
		Mode:          ModeContinuous,
		Granularities: []string{"hourly", "daily", "monthly"},
		// 持续任务窗口存零值（创建清窗）。
	}

	// #479 改动四：持续任务回看口径在「上一完整格」上再回退 1 格补偿管线延迟。
	checks := []struct {
		g    metrics.Granularity
		want time.Time
	}{
		{metrics.GranularityHourly, time.Date(2026, 6, 16, 16, 0, 0, 0, loc)},
		{metrics.GranularityDaily, time.Date(2026, 6, 14, 0, 0, 0, 0, loc)},
		{metrics.GranularityMonthly, time.Date(2026, 4, 1, 0, 0, 0, 0, loc)},
	}
	for _, c := range checks {
		_, err := e.queryAndConvert(context.Background(), task, c.g)
		require.NoError(t, err)
		// 单格：start==end==桶起点 → time 过滤只命中这一格。
		assert.True(t, aggr.lastReq.StartTime.Equal(c.want), "%s start 应=%v，实际 %v", c.g, c.want, aggr.lastReq.StartTime)
		assert.True(t, aggr.lastReq.EndTime.Equal(c.want), "%s end 应=%v，实际 %v", c.g, c.want, aggr.lastReq.EndTime)
	}
}

// oneshot 任务的固定用户窗口保持原样，不被覆盖。
func TestQueryAndConvert_OneshotWindowUnchanged(t *testing.T) {
	loc := time.UTC
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, loc)
	end := time.Date(2026, 5, 22, 0, 0, 0, 0, loc)

	aggr := &stubAggr{}
	e := NewExecutor(aggr, &stubRepo{}, nil, nil).SetLocation(loc)
	e.now = func() time.Time { return time.Date(2026, 6, 16, 18, 44, 0, 0, loc) }

	task := &Task{
		ID:            uuid.New(),
		Mode:          ModeOneshot,
		Granularities: []string{"hourly"},
		WindowStart:   start,
		WindowEnd:     end,
	}
	_, err := e.queryAndConvert(context.Background(), task, metrics.GranularityHourly)
	require.NoError(t, err)

	assert.True(t, aggr.lastReq.StartTime.Equal(start), "oneshot start 应保持用户窗口 %v，实际 %v", start, aggr.lastReq.StartTime)
	assert.True(t, aggr.lastReq.EndTime.Equal(end), "oneshot end 应保持用户窗口 %v，实际 %v", end, aggr.lastReq.EndTime)
}
