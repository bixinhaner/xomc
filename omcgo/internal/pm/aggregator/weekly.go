package aggregator

import "github.com/omcgo/omcgo/internal/pm/metrics"

// JobTypeWeekly 是 weekly aggregator 的 asyncjob 类型主键。
//
// 触发：业务周最后一个 daily bucket 成功后 chain。
// 数据流：pm_metrics_daily → pm_metrics_weekly。
const JobTypeWeekly = "pm_aggregate_weekly"

// NewWeeklyRunner 构造 weekly asyncjob runner。
func NewWeeklyRunner(a *Aggregator) *Runner {
	return &Runner{
		aggregator:  a,
		jobType:     JobTypeWeekly,
		source:      "pm_metrics_daily",
		target:      "pm_metrics_weekly",
		granularity: metrics.GranularityWeekly,
	}
}
