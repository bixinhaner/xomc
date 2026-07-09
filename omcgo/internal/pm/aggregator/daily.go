package aggregator

import "github.com/omcgo/omcgo/internal/pm/metrics"

// JobTypeDaily 是 daily aggregator 的 asyncjob 类型主键。
//
// 触发：业务日最后一个 hourly bucket 成功后 chain。
// 数据流：pm_metrics_hourly → pm_metrics_daily（级联聚合，避免重扫 15min）。
const JobTypeDaily = "pm_aggregate_daily"

// NewDailyRunner 构造 daily asyncjob runner。
func NewDailyRunner(a *Aggregator) *Runner {
	return &Runner{
		aggregator:  a,
		jobType:     JobTypeDaily,
		source:      "pm_metrics_hourly",
		target:      "pm_metrics_daily",
		granularity: metrics.GranularityDaily,
	}
}
