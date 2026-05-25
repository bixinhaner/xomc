// Package retention 定义 PM 时序数据 5 级粒度保留策略的元数据层。
//
// 本包仅提供"策略源 + 配置层"：
//   - 5 个 sys_configs 键名 + 默认值（金字塔保留：15min=30d / hour=180d / day=2y / week=2y / month=5y）
//   - 5 个粒度常量
//   - 供 G3/G5 的 migration 内嵌挂 compression / retention policy 时引用本包常量
//
// 实际挂表 policy 的动作 **不在本包内**：
//   - hypertable 走 TimescaleDB add_compression_policy / add_retention_policy（在 G3/G5 migration 内做）
//   - 普通表（daily/weekly/monthly）走 G5 cron 清理任务
//
// sys_configs 5 键变化时，retention.Service 通过 chenbo01 留好的 SysConfigSavedHook
// 机制（admin.SysConfigService.RegisterSavedHook，category="pm.retention"）触发 Reload，
// 重新读取常量 + 通知 G5 cron + alter_job 联动 hypertable policy。
package retention

import "time"

// Category 是 sys_configs 表中 PM 保留策略所有键的 category 值。
// chenbo01 的 SysConfigSavedHook 回调按 category 触发，必须固定此常量。
const Category = "pm.retention"

// PolicyKey 是 sys_configs 表中（category="pm.retention", key=...）的 key 值。
type PolicyKey string

const (
	KeyRaw15MinDays PolicyKey = "raw_15min_days"
	KeyHourlyDays   PolicyKey = "hourly_days"
	KeyDailyDays    PolicyKey = "daily_days"
	KeyWeeklyDays   PolicyKey = "weekly_days"
	KeyMonthlyDays  PolicyKey = "monthly_days"
)

// AllKeys 返回所有 PM 保留策略键名，按粒度从细到粗排序。
func AllKeys() []PolicyKey {
	return []PolicyKey{
		KeyRaw15MinDays,
		KeyHourlyDays,
		KeyDailyDays,
		KeyWeeklyDays,
		KeyMonthlyDays,
	}
}

// DefaultDays 是 5 级粒度的默认保留天数（金字塔保留，与设计文档 §4.2 一致）。
// 运维可在系统设置页改，sys_configs 变更后 retention.Service.Reload 联动到 cron + alter_job。
var DefaultDays = map[PolicyKey]int{
	KeyRaw15MinDays: 30,   // 15min 原始表保留 30 天
	KeyHourlyDays:   180,  // 小时聚合表保留 180 天（≈ 6 个月）
	KeyDailyDays:    730,  // 日聚合表保留 2 年
	KeyWeeklyDays:   730,  // 周聚合表保留 2 年
	KeyMonthlyDays:  1825, // 月聚合表保留 5 年
}

// CompressionAfterDays 是 hypertable 启用 compression 的延迟（仅 hypertable 适用，
// 普通表 daily/weekly/monthly 无 compression，由 G5 cron 清理）。
//
// 与 sys_configs 解耦：compression 阈值是"何时压缩"，retention 是"何时删"，
// 当前阶段 compression 阈值 hard-code 在 G3/G5 migration 内（7d / 14d），
// 未来需要 UI 可调时再扩 sys_configs。
var CompressionAfterDays = map[PolicyKey]int{
	KeyRaw15MinDays: 7,  // pm_metrics（15min）压缩阈值
	KeyHourlyDays:   14, // pm_metrics_hourly 压缩阈值
}

// Granularity 是 PM 时序数据的粒度标识，与 pm_metrics.granularity 列对齐。
type Granularity string

const (
	Granularity15Min   Granularity = "15min"
	GranularityHourly  Granularity = "hourly"
	GranularityDaily   Granularity = "daily"
	GranularityWeekly  Granularity = "weekly"
	GranularityMonthly Granularity = "monthly"
)

// KeyForGranularity 把 Granularity 映射到对应的 PolicyKey。
func KeyForGranularity(g Granularity) (PolicyKey, bool) {
	switch g {
	case Granularity15Min:
		return KeyRaw15MinDays, true
	case GranularityHourly:
		return KeyHourlyDays, true
	case GranularityDaily:
		return KeyDailyDays, true
	case GranularityWeekly:
		return KeyWeeklyDays, true
	case GranularityMonthly:
		return KeyMonthlyDays, true
	default:
		return "", false
	}
}

// GranularityForKey 把 PolicyKey 反向映射到 Granularity。
func GranularityForKey(k PolicyKey) (Granularity, bool) {
	switch k {
	case KeyRaw15MinDays:
		return Granularity15Min, true
	case KeyHourlyDays:
		return GranularityHourly, true
	case KeyDailyDays:
		return GranularityDaily, true
	case KeyWeeklyDays:
		return GranularityWeekly, true
	case KeyMonthlyDays:
		return GranularityMonthly, true
	default:
		return "", false
	}
}

// DefaultDuration 把 PolicyKey 转为 time.Duration（默认值）。
func DefaultDuration(k PolicyKey) time.Duration {
	days, ok := DefaultDays[k]
	if !ok {
		return 0
	}
	return time.Duration(days) * 24 * time.Hour
}

// 保留期合法范围：避免误操作把保留期设为 0（瞬时删除）或天文数字（占爆磁盘）。
const (
	MinRetentionDays = 1
	MaxRetentionDays = 3650 // 10 年
)

// ValidateDays 检查保留天数是否在合法范围内。
func ValidateDays(days int) error {
	if days < MinRetentionDays {
		return ErrRetentionTooShort
	}
	if days > MaxRetentionDays {
		return ErrRetentionTooLong
	}
	return nil
}
