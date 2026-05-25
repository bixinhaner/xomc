package aggregator

import "github.com/omcgo/omcgo/internal/pm/metrics"

// JobTypeDaily 是 daily aggregator 的 asyncjob 类型主键。
//
// cron 调度：每日 00:05（如 5/23 00:05 跑 [5/22 00:00, 5/23 00:00) 桶）。
// 数据流：pm_metrics_hourly → pm_metrics_daily（级联聚合，避免重扫 15min）。
const JobTypeDaily = "pm_aggregate_daily"

// NewDailyRunner 构造 daily cron runner。
func NewDailyRunner(a *Aggregator) *Runner {
	return &Runner{
		aggregator:  a,
		jobType:     JobTypeDaily,
		source:      "pm_metrics_hourly",
		target:      "pm_metrics_daily",
		granularity: metrics.GranularityDaily,
	}
}
