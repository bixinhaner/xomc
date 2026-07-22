package adhoc

import (
	"time"

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// truncateBucketStart 把任意时刻 t 截到其所属粒度格的桶起点（#528 P2）。
//
// #528：下游持续任务不再用「now − 固定 N 格」猜测目标桶，改为直接消费上游「完成水位」。
// 水位记录的就是上游已确定卷完的那一格的桶起点，本身已对齐格边界；此函数仅做防御性对齐，
// 保证无论传入的是水位桶起点还是任意时刻，落到的都是确定的格起点，作为 aggregator time 半开过滤边界
// 的起点。
//
// loc 为业务时区（daily/weekly/monthly 的零点对齐依赖时区；hourly/15min 与时区无关）。
func truncateBucketStart(g metrics.Granularity, t time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	switch g {
	case metrics.Granularity15Min:
		// 15min 整点对齐与时区无关。
		return t.Truncate(15 * time.Minute)
	case metrics.GranularityHourly:
		// 整点对齐与时区无关（整点 UTC = 整点本地同一瞬间）。
		return t.Truncate(time.Hour)
	case metrics.GranularityDaily:
		return truncateAdhocDay(t.In(loc))
	case metrics.GranularityWeekly:
		return truncateAdhocWeekISO(t.In(loc))
	case metrics.GranularityMonthly:
		n := t.In(loc)
		return time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, n.Location())
	default:
		// 未知粒度：退回 hourly 口径（保守）。
		return t.Truncate(time.Hour)
	}
}

// nextBucketStart 返回给定桶起点之后的下一桶起点，用于构造 aggregator.Query 的半开窗口。
func nextBucketStart(g metrics.Granularity, bucket time.Time) time.Time {
	switch g {
	case metrics.Granularity15Min:
		return bucket.Add(15 * time.Minute)
	case metrics.GranularityHourly:
		return bucket.Add(time.Hour)
	case metrics.GranularityDaily:
		return bucket.AddDate(0, 0, 1)
	case metrics.GranularityWeekly:
		return bucket.AddDate(0, 0, 7)
	case metrics.GranularityMonthly:
		return bucket.AddDate(0, 1, 0)
	default:
		return bucket.Add(time.Hour)
	}
}

// previousBucketStart 返回给定桶起点之前的上一桶起点。
func previousBucketStart(g metrics.Granularity, bucket time.Time) time.Time {
	switch g {
	case metrics.Granularity15Min:
		return bucket.Add(-15 * time.Minute)
	case metrics.GranularityHourly:
		return bucket.Add(-time.Hour)
	case metrics.GranularityDaily:
		return bucket.AddDate(0, 0, -1)
	case metrics.GranularityWeekly:
		return bucket.AddDate(0, 0, -7)
	case metrics.GranularityMonthly:
		return bucket.AddDate(0, -1, 0)
	default:
		return bucket.Add(-time.Hour)
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
