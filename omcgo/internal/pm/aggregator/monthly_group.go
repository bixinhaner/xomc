package aggregator

import "github.com/omcgo/omcgo/internal/pm/metrics"

// JobTypeMonthlyGroup 设备组维度月聚合 asyncjob 类型主键。
//
// cron 调度：每月 1 日 00:25（设备级 00:15 后 10 分钟）。
// 数据流：pm_metrics_monthly → JOIN devices + device_group_members → pm_group_metrics_monthly。
const JobTypeMonthlyGroup = "pm_aggregate_monthly_group"

// NewMonthlyGroupRunner 构造 monthly 设备组聚合 runner。
func NewMonthlyGroupRunner(a *Aggregator) *GroupRunner {
	return &GroupRunner{
		aggregator:   a,
		jobType:      JobTypeMonthlyGroup,
		deviceTarget: "pm_metrics_monthly",
		groupTarget:  "pm_group_metrics_monthly",
		granularity:  metrics.GranularityMonthly,
	}
}
