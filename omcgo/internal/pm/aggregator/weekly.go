package aggregator

import "github.com/omcgo/omcgo/internal/pm/metrics"

// JobTypeWeekly 是 weekly aggregator 的 asyncjob 类型主键。
//
// cron 调度：每周一 00:10（ISO 周一对齐）。
// 数据流：pm_metrics_daily → pm_metrics_weekly。
const JobTypeWeekly = "pm_aggregate_weekly"

// NewWeeklyRunner 构造 weekly cron runner。
func NewWeeklyRunner(a *Aggregator) *Runner {
	return &Runner{
		aggregator:  a,
		jobType:     JobTypeWeekly,
		source:      "pm_metrics_daily",
		target:      "pm_metrics_weekly",
		granularity: metrics.GranularityWeekly,
	}
}
