package aggregator

import "github.com/omcgo/omcgo/internal/pm/metrics"

// JobTypeMonthly 是 monthly aggregator 的 asyncjob 类型主键。
//
// cron 调度：每月 1 日 00:15（聚合上一自然月）。
// 数据流：pm_metrics_weekly → pm_metrics_monthly。
const JobTypeMonthly = "pm_aggregate_monthly"

// NewMonthlyRunner 构造 monthly cron runner。
func NewMonthlyRunner(a *Aggregator) *Runner {
	return &Runner{
		aggregator:  a,
		jobType:     JobTypeMonthly,
		source:      "pm_metrics_weekly",
		target:      "pm_metrics_monthly",
		granularity: metrics.GranularityMonthly,
	}
}
