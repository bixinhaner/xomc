package adhoc

import (
	"time"

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// pipelineLagBuckets 是持续任务回看口径相对「最近一个已完成周期」额外多回退的格数（#479 改动四）。
//
// 背景（#479 根因 4 + 阶段1/2 定稿后的真实管线延迟）：
//   - 设备级聚合 cron 在每格结束后过 5 分钟（hourly :05 / daily 00:05 / …）才触发；
//     成功后立即 chain 设备组聚合（阶段2 实测组 ingest 紧随设备级完成，约 +5s worker tick）。
//   - 也就是说，覆盖区间 [B_start, B_end) 的桶，其设备组汇总要到约 B_end+5分钟 才落库。
//
// 若只回看「最近一个已完成周期」（即上一格），当持续任务恰在本格 [B_end, B_end+5分钟] 这段
// 「上一格已结束、但其聚合 cron 尚未跑完」的空窗里运行，就会查到一格还没汇总好的桶 → 扑空，
// 而游标只进不退、再不回头（#479 根因 4 描述的"永远赶在数据之前"）。
//
// 治本：把回看口径再往前推 1 格。被回看的目标桶比 now 至少早一整格 + 上一格，
// 其聚合 cron（在目标桶结束后过 5 分钟、即仍在上一格区间内）必已跑完——
// 无论持续任务在本格内哪个时刻运行，目标桶都是「确定已汇总好」的那一格。
//
// 取 1 而非更大：阶段1/2 把管线延迟压到"结束后约 5 分钟 + 5s chain tick"，远小于任一粒度的格宽，
// 1 格的安全裕度足以覆盖（hourly 目标桶结束在 now 前 ≥1 小时，5 分钟延迟绰绰有余）；
// 多回退会无谓拉大数据新鲜度落后，YAGNI。
const pipelineLagBuckets = 1

// lastCompletedBucket 计算给定粒度下「确定已汇总好」的那一格的桶起点 time（#479 改动四）。
//
// ISSUE-398（方案 A·有界滚动窗口）：持续型任务每次运行只算「一格」，而非空窗扫全历史。
// 基线是「当前正在进行那格的前一格」= 最近一个已收完的完整周期；
// 在此基础上再按 pipelineLagBuckets（管线延迟，见上）额外回退，落到聚合 cron 必已跑完的那一格。
// 例：小时任务 18:44 跑 → 当前 18:00-19:00 进行中 → 上一完整小时 17:00-18:00 的聚合要 18:05 才落库，
// 故再回退 1 格 → 算 16:00-17:00（其聚合 17:05 早已完成），返回桶起点 16:00。
//
// 算法与 worker cron（cmd/worker/aggregator.go::pmAggregatorCronEntries）的窗口口径一致：
// 先把 now 截到本格起点，再回退 (1 + pipelineLagBuckets) 格 → 目标格起点。结果聚合表里该桶行的
// `time` 列即为桶起点，所以返回值直接用作 aggregator 的 time 过滤边界
// （StartTime==EndTime==桶起点 → 只命中这一格）。
//
// loc 为业务时区（daily/weekly/monthly 的零点对齐依赖时区；hourly 与时区无关）。
// 返回桶起点 time（UTC 表示，已折算业务时区零点）。
func lastCompletedBucket(g metrics.Granularity, now time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = time.UTC
	}
	// 回退格数 = 1（上一完整格）+ pipelineLagBuckets（管线延迟补偿）。
	back := 1 + pipelineLagBuckets
	switch g {
	case metrics.Granularity15Min:
		// 15min 整点对齐与时区无关：截到本 15 分钟格起点再回退 back 格。
		curStart := now.Truncate(15 * time.Minute)
		return curStart.Add(-time.Duration(back) * 15 * time.Minute)
	case metrics.GranularityHourly:
		// 整点对齐与时区无关（整点 UTC = 整点本地同一瞬间）。
		curStart := now.Truncate(time.Hour)
		return curStart.Add(-time.Duration(back) * time.Hour)
	case metrics.GranularityDaily:
		today := truncateAdhocDay(now.In(loc))
		return today.AddDate(0, 0, -back)
	case metrics.GranularityWeekly:
		thisMon := truncateAdhocWeekISO(now.In(loc))
		return thisMon.AddDate(0, 0, -7*back)
	case metrics.GranularityMonthly:
		n := now.In(loc)
		thisMonth := time.Date(n.Year(), n.Month(), 1, 0, 0, 0, 0, n.Location())
		return thisMonth.AddDate(0, -back, 0)
	default:
		// 未知粒度：退回 hourly 口径（保守，仍是有界一格而非全历史）。
		curStart := now.Truncate(time.Hour)
		return curStart.Add(-time.Duration(back) * time.Hour)
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
