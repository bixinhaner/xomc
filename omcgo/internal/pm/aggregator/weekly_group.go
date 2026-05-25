package aggregator

import "github.com/omcgo/omcgo/internal/pm/metrics"

// JobTypeWeeklyGroup 设备组维度周聚合 asyncjob 类型主键。
//
// cron 调度：每周一 00:20（设备级 00:10 后 10 分钟）。
// 数据流：pm_metrics_weekly → JOIN devices + device_group_members → pm_group_metrics_weekly。
const JobTypeWeeklyGroup = "pm_aggregate_weekly_group"

// NewWeeklyGroupRunner 构造 weekly 设备组聚合 runner。
func NewWeeklyGroupRunner(a *Aggregator) *GroupRunner {
	return &GroupRunner{
		aggregator:   a,
		jobType:      JobTypeWeeklyGroup,
		deviceTarget: "pm_metrics_weekly",
		groupTarget:  "pm_group_metrics_weekly",
		granularity:  metrics.GranularityWeekly,
	}
}
