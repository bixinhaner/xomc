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

// StatisType 是 counter 类指标的聚合方式提示，驱动 G5 自然桶聚合（sum/avg/max/pct 四路）。
// KPI 类不填（由 perf 平台公式决定），存 NULL。
type StatisType string

const (
	StatisSum StatisType = "sum"
	StatisAvg StatisType = "avg"
	StatisMax StatisType = "max"
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
// 唯一性维度（自然键）：(DeviceSN, MetricPath, Granularity, EndTime)
// 唯一索引 uq_pm_metrics_natural 上挂 ON CONFLICT DO UPDATE 保证补传幂等。
//
// 时间三字段（G4 在 parser 层已落）：
//   - StartTime  基站采集窗口起（基站时钟）
//   - EndTime    基站采集窗口止（基站时钟，与 Time 列同值）
//   - IngestTime OMC 入库时刻（OMC 时钟，默认 NOW()）
//
// 字段映射（旧 → 新）：
//   - PMCounter.DeviceID(uuid)  → DeviceSN(text)（过渡期存 UUID 字符串，后续 collector 改传真实 SN）
//   - PMCounter.CellID(string)  → ObjectLDN(*string)
//   - PMCounter.CounterGroup    → 进 Extra JSONB
//   - PMCounter.CounterName     → MetricPath + MetricType='counter'
//   - PMCounter.CounterValue    → MetricValue
//   - PMCounter.Granularity(int 分钟) → Granularity(text '15min'/...)
//   - KPIValue.KPIName          → MetricPath + MetricType='kpi'
//   - KPIValue.Carrier/Tech     → 进 Extra JSONB
type PMMetric struct {
	ID          uuid.UUID
	DeviceSN    string
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
