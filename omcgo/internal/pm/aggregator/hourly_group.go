package aggregator

import "github.com/omcgo/omcgo/internal/pm/metrics"

// JobTypeHourlyGroup 设备组维度小时聚合 asyncjob 类型主键。
//
// cron 调度：每小时 :15（设备级 :05 后 10 分钟，避免读到未完成的 device-level 数据）。
// 数据流：pm_metrics_hourly（device 维度）→ JOIN devices + device_group_members
//           → GROUP BY device_group_id → pm_group_metrics_hourly。
const JobTypeHourlyGroup = "pm_aggregate_hourly_group"

// NewHourlyGroupRunner 构造 hourly 设备组聚合 runner。
func NewHourlyGroupRunner(a *Aggregator) *GroupRunner {
	return &GroupRunner{
		aggregator:   a,
		jobType:      JobTypeHourlyGroup,
		deviceTarget: "pm_metrics_hourly",
		groupTarget:  "pm_group_metrics_hourly",
		granularity:  metrics.GranularityHourly,
	}
}
