package aggregator

import "github.com/omcgo/omcgo/internal/pm/metrics"

// JobTypeDailyGroup 设备组维度日聚合 asyncjob 类型主键。
//
// cron 调度：每日 00:15（设备级 00:05 后 10 分钟）。
// 数据流：pm_metrics_daily → JOIN devices + device_group_members → pm_group_metrics_daily。
const JobTypeDailyGroup = "pm_aggregate_daily_group"

// NewDailyGroupRunner 构造 daily 设备组聚合 runner。
func NewDailyGroupRunner(a *Aggregator) *GroupRunner {
	return &GroupRunner{
		aggregator:   a,
		jobType:      JobTypeDailyGroup,
		deviceTarget: "pm_metrics_daily",
		groupTarget:  "pm_group_metrics_daily",
		granularity:  metrics.GranularityDaily,
	}
}
