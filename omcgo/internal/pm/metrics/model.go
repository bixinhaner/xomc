package metrics

import (
	"time"

	"github.com/google/uuid"
)

// MetricType 区分 counter / kpi —— pm_metrics 单表内的子类型。
type MetricType string

const (
	MetricTypeCounter MetricType = "counter"
	MetricTypeKPI     MetricType = "kpi"
)

// StatisType 是 counter 类指标的聚合方式提示，驱动 G5 自然桶聚合（sum/avg/max/min/pct 五路）。
// KPI 类不填（由 perf 平台公式决定），存 NULL。
// 与老 OMC 系统 perf_indicators.statis_type 五个枚举一一对应；DB CHECK 约束亦同。
type StatisType string

const (
	StatisSum StatisType = "sum"
	StatisAvg StatisType = "avg"
	StatisMax StatisType = "max"
	StatisMin StatisType = "min"
	StatisPct StatisType = "pct" // 百分比 / 比率，arithmetic 上下文（G5 cron 用）
)

// Granularity 粒度。G3 阶段仅 '15min'；G5 加 hourly/daily/weekly/monthly。
type Granularity string

const (
	Granularity15Min   Granularity = "15min"
	GranularityHourly  Granularity = "hourly"
	GranularityDaily   Granularity = "daily"
	GranularityWeekly  Granularity = "weekly"
	GranularityMonthly Granularity = "monthly"
)

// PMMetric 是 pm_metrics 表行。
//
// 设备唯一标识：按 TR-069 标准用 (DeviceOUI, DeviceSN) 双键（T-0164-P3 fix 引入）。
// DeviceOUI = TR-069 DeviceId.OUI（6 位 hex 大写），DeviceSN = DeviceId.SerialNumber。
// 全系统级切换见 docs/project/plan-T-0165-system-wide-oui-sn-migration.md。
//
// 唯一性维度（自然键）：(DeviceOUI, DeviceSN, MetricPath, Granularity, EndTime, Time, ObjectLDN)
// 唯一索引 uq_pm_metrics_natural 上挂 ON CONFLICT DO UPDATE 保证补传幂等。
// （TimescaleDB 要求 UNIQUE 索引必须含分区列 Time；业务上 Time = EndTime，约束意义不变）
// ObjectLDN 在 DB 层 NOT NULL DEFAULT ”（migration 000171），Go 端 *string nil → ”
// 落盘。原因：UNIQUE 中 NULL ≠ NULL，必须强制非空才能严格唯一；BUG-6 即由此触发：
// 同 PM 文件中同 MetricPath 跨多个 cell（不同 ObjectLDN）撞自然键二次命中同行。
//
// 时间三字段（G4 在 parser 层已落）：
//   - StartTime  基站采集窗口起（基站时钟）
//   - EndTime    基站采集窗口止（基站时钟，与 Time 列同值）
//   - IngestTime OMC 入库时刻（OMC 时钟，默认 NOW()）
//
// 字段映射（旧 → 新）：
//   - PMCounter.OUI + PMCounter.DeviceSN → DeviceOUI + DeviceSN（TR-069 双键）
//   - PMCounter.CellID(string)  → ObjectLDN(*string)
//   - PMCounter.CounterGroup    → 进 Extra JSONB
//   - PMCounter.CounterName     → MetricPath + MetricType='counter'
//   - PMCounter.CounterValue    → MetricValue
//   - PMCounter.Granularity(int 分钟) → Granularity(text '15min'/...)
//   - KPIValue.KPIName          → MetricPath + MetricType='kpi'
//   - KPIValue.Carrier/Tech     → 进 Extra JSONB
type PMMetric struct {
	ID          uuid.UUID
	DeviceOUI   string // TR-069 DeviceId.OUI（6 位 hex 大写）
	DeviceSN    string // TR-069 DeviceId.SerialNumber
	MetricPath  string
	MetricType  MetricType
	MetricValue float64
	StatisType  *StatisType // counter 必填，kpi nil
	Granularity Granularity
	Time        time.Time // 与 EndTime 同值（hypertable 分区列）
	StartTime   time.Time
	EndTime     time.Time
	IngestTime  time.Time
	ObjectLDN   *string
	Extra       map[string]any
}
