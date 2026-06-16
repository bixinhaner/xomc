package adhoc

import (
	"time"

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// lastCompletedBucket 计算给定粒度下「最近一个已收完的完整周期」那一格的桶起点 time。
//
// ISSUE-398（方案 A·有界滚动窗口）：持续型任务每次运行只算「最近一格」，而非空窗扫全历史。
// 取「当前正在进行那格的前一格」= 最近一个已收完的完整周期。
// 例：小时任务 18:44 跑 → 当前 18:00-19:00 未收完 → 算上一格 17:00-18:00，返回桶起点 17:00。
//
// 算法与 worker cron（cmd/worker/aggregator.go::pmAggregatorCronEntries）的窗口口径一致：
// 先把 now 截到本格起点，再回退一格 → 上一格起点。结果聚合表里该桶行的 `time` 列即为桶起点，
// 所以返回值直接用作 aggregator 的 time 过滤边界（StartTime==EndTime==桶起点 → 只命中这一格）。
//
// loc 为业务时区（daily/weekly/monthly 的零点对齐依赖时区；hourly 与时区无关）。
// 返回桶起点 time（UTC 表示，已折算业务时区零点）。
func lastCompletedBucket(g metrics.Granularity, now time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	switch g {
	case metrics.Granularity15Min:
		// 15min 整点对齐与时区无关：截到本 15 分钟格起点再回退一格。
		curStart := now.Truncate(15 * time.Minute)
		return curStart.Add(-15 * time.Minute)
	case metrics.GranularityHourly:
		// 整点对齐与时区无关（整点 UTC = 整点本地同一瞬间）。
		curStart := now.Truncate(time.Hour)
		return curStart.Add(-time.Hour)
	case metrics.GranularityDaily:
		today := truncateAdhocDay(now.In(loc))
		return today.AddDate(0, 0, -1)
	case metrics.GranularityWeekly:
		thisMon := truncateAdhocWeekISO(now.In(loc))
		return thisMon.AddDate(0, 0, -7)
	case metrics.GranularityMonthly:
		n := now.In(loc)
		thisMonth := time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, n.Location())
		return thisMonth.AddDate(0, -1, 0)
	default:
		// 未知粒度：退回 hourly 口径（保守，仍是有界一格而非全历史）。
		curStart := now.Truncate(time.Hour)
		return curStart.Add(-time.Hour)
	}
}

// truncateAdhocDay 把 t 截到当天 00:00（保留 t 的时区）。
func truncateAdhocDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// truncateAdhocWeekISO 把 t 截到本周周一 00:00（ISO 周历，周一为一周开始）。
//
// time.Weekday(): Sunday=0..Saturday=6（非 ISO）；(wd+6)%7 得「距本周一的天数」。
func truncateAdhocWeekISO(t time.Time) time.Time {
	wd := int(t.Weekday())
	daysSinceMonday := (wd + 6) % 7 // Sun=6, Mon=0, Tue=1, ...
	return time.Date(t.Year(), t.Month(), t.Day()-daysSinceMonday, 0, 0, 0, 0, t.Location())
}
