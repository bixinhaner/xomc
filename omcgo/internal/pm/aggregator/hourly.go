package aggregator

import "github.com/omcgo/omcgo/internal/pm/metrics"

// JobTypeHourly 是 hourly aggregator 的 asyncjob 类型主键。
//
// cron 调度：每小时 :05（如 11:05 跑 [10:00, 11:00) 桶）。
// 数据流：pm_metrics (granularity='15min') → pm_metrics_hourly。
const JobTypeHourly = "pm_aggregate_hourly"

// NewHourlyRunner 构造 hourly cron runner。
//
// 由 worker/main.go 在初始化期调用，注册到 asyncjob.Registry。
func NewHourlyRunner(a *Aggregator) *Runner {
	return &Runner{
		aggregator:  a,
		jobType:     JobTypeHourly,
		source:      "pm_metrics",
		target:      "pm_metrics_hourly",
		granularity: metrics.GranularityHourly,
	}
}
