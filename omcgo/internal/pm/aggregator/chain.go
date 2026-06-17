package aggregator

// GroupJobTypeFor 返回设备级聚合 job_type 对应的设备组聚合 job_type（#479 改动三）。
//
// 设备级该桶聚合成功后，按本映射 chain 出对应粒度的设备组聚合任务，
// 实现"设备级算完→紧接着算设备组"的确定性串联（取代旧的固定 10 分钟错峰）。
// 非设备级聚合 job_type（含设备组自身、导出等）返回空串，表示无后继链。
func GroupJobTypeFor(deviceJobType string) string {
	switch deviceJobType {
	case JobTypeHourly:
		return JobTypeHourlyGroup
	case JobTypeDaily:
		return JobTypeDailyGroup
	case JobTypeWeekly:
		return JobTypeWeeklyGroup
	case JobTypeMonthly:
		return JobTypeMonthlyGroup
	default:
		return ""
	}
}
