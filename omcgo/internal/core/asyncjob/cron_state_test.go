package asyncjob

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// hourlyAdvance / dailyAdvance / weeklyAdvance / monthlyAdvance — 与 cmd/worker/aggregator.go
// 的实际配置对齐；测试在此包内固定下来防回归。
var (
	hourlyAdvance  = func(prev time.Time) time.Time { return prev.Add(time.Hour) }
	dailyAdvance   = func(prev time.Time) time.Time { return prev.AddDate(0, 0, 1) }
	weeklyAdvance  = func(prev time.Time) time.Time { return prev.AddDate(0, 0, 7) }
	monthlyAdvance = func(prev time.Time) time.Time { return prev.AddDate(0, 1, 0) }
)

func Test_CatchupMissedBuckets_Hourly_4Missed(t *testing.T) {
	last := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 23, 14, 30, 0, 0, time.UTC)
	got := CatchupMissedBuckets(last, now, hourlyAdvance)

	assert.Len(t, got, 4)
	assert.Equal(t, time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC), got[0].Start)
	assert.Equal(t, time.Date(2026, 5, 23, 11, 0, 0, 0, time.UTC), got[0].End)
	assert.Equal(t, time.Date(2026, 5, 23, 13, 0, 0, 0, time.UTC), got[3].Start)
	assert.Equal(t, time.Date(2026, 5, 23, 14, 0, 0, 0, time.UTC), got[3].End)
}

func Test_CatchupMissedBuckets_Daily_2Missed(t *testing.T) {
	last := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC) // 已过 5/22 00:00 但未到 5/23 00:00
	got := CatchupMissedBuckets(last, now, dailyAdvance)

	assert.Len(t, got, 2)
	assert.Equal(t, time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC), got[0].Start)
	assert.Equal(t, time.Date(2026, 5, 21, 0, 0, 0, 0, time.UTC), got[0].End)
	assert.Equal(t, time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC), got[1].End)
}

func Test_CatchupMissedBuckets_Weekly_1Missed(t *testing.T) {
	last := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC) // 周一
	now := time.Date(2026, 5, 27, 0, 0, 0, 0, time.UTC)
	got := CatchupMissedBuckets(last, now, weeklyAdvance)

	assert.Len(t, got, 1)
	assert.Equal(t, time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC), got[0].Start)
	assert.Equal(t, time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC), got[0].End)
}

func Test_CatchupMissedBuckets_Monthly_VariableLength(t *testing.T) {
	// 2 月 28 天，3 月 31 天 — monthly advance 必须用 AddDate(0,1,0) 处理变长
	last := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	got := CatchupMissedBuckets(last, now, monthlyAdvance)

	// 2 月 1 日 → 3 月 1 日（end），3 月 1 日 → 4 月 1 日（end）= 2 个 bucket
	assert.Len(t, got, 2)
	assert.Equal(t, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), got[0].Start)
	assert.Equal(t, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), got[0].End)
	assert.Equal(t, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), got[1].End)
}

func Test_CatchupMissedBuckets_NoBacklog(t *testing.T) {
	// last == now — 没有积压
	t0 := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)
	got := CatchupMissedBuckets(t0, t0, hourlyAdvance)
	assert.Empty(t, got)
}

func Test_CatchupMissedBuckets_FutureLastBucketEnd(t *testing.T) {
	// last > now（时钟回退 / 配置错误）— 不报错，返空
	last := time.Date(2026, 5, 23, 20, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 23, 14, 0, 0, 0, time.UTC)
	got := CatchupMissedBuckets(last, now, hourlyAdvance)
	assert.Empty(t, got)
}

func Test_CatchupMissedBuckets_NonAdvancingFunc_NoInfiniteLoop(t *testing.T) {
	// advance 不前进 — 不死循环
	noop := func(prev time.Time) time.Time { return prev }
	last := time.Date(2026, 5, 23, 10, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 23, 20, 0, 0, 0, time.UTC)
	got := CatchupMissedBuckets(last, now, noop)
	assert.Empty(t, got)
}

func Test_CatchupMissedBuckets_BoundedAt10K(t *testing.T) {
	// 极端：停机 20 年（用 hourly 算就是 ~175200 buckets）— 应被截到 10000
	last := time.Date(2006, 5, 23, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	got := CatchupMissedBuckets(last, now, hourlyAdvance)
	assert.LessOrEqual(t, len(got), 10001) // 上限保护
	assert.GreaterOrEqual(t, len(got), 10000)
}
