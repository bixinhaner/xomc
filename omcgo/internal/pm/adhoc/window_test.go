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

	cases := []struct {
		name string
		g    metrics.Granularity
		want time.Time
	}{
		{
			name: "hourly: 18:44 跑 → 上一完整小时 17:00",
			g:    metrics.GranularityHourly,
			want: time.Date(2026, 6, 16, 17, 0, 0, 0, loc),
		},
		{
			name: "15min: 18:44 跑 → 当前格 18:30 起 → 上一格 18:15",
			g:    metrics.Granularity15Min,
			want: time.Date(2026, 6, 16, 18, 15, 0, 0, loc),
		},
		{
			name: "daily: 今天进行中 → 上一完整天 06-15 00:00",
			g:    metrics.GranularityDaily,
			want: time.Date(2026, 6, 15, 0, 0, 0, 0, loc),
		},
		{
			name: "weekly: 本周一 06-15 → 上一完整周 06-08 00:00",
			g:    metrics.GranularityWeekly,
			want: time.Date(2026, 6, 8, 0, 0, 0, 0, loc),
		},
		{
			name: "monthly: 6月进行中 → 上一完整月 05-01 00:00",
			g:    metrics.GranularityMonthly,
			want: time.Date(2026, 5, 1, 0, 0, 0, 0, loc),
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
	want := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	assert.True(t, got.Equal(want), "got %v want %v", got, want)
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

	checks := []struct {
		g    metrics.Granularity
		want time.Time
	}{
		{metrics.GranularityHourly, time.Date(2026, 6, 16, 17, 0, 0, 0, loc)},
		{metrics.GranularityDaily, time.Date(2026, 6, 15, 0, 0, 0, 0, loc)},
		{metrics.GranularityMonthly, time.Date(2026, 5, 1, 0, 0, 0, 0, loc)},
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
