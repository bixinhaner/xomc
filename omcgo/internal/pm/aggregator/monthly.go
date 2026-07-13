package aggregator

import "github.com/omcgo/omcgo/internal/pm/metrics"

// JobTypeMonthly 是 monthly aggregator 的 asyncjob 类型主键。
//
// 触发：业务月最后一个 daily bucket 成功后 chain。
// 数据流：pm_metrics_daily → pm_metrics_monthly。
const JobTypeMonthly = "pm_aggregate_monthly"

// NewMonthlyRunner 构造 monthly asyncjob runner。
func NewMonthlyRunner(a *Aggregator) *Runner {
	return &Runner{
		aggregator:  a,
		jobType:     JobTypeMonthly,
		source:      "pm_metrics_daily",
		target:      "pm_metrics_monthly",
		granularity: metrics.GranularityMonthly,
	}
}
