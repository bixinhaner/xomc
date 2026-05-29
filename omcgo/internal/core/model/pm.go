package model

import (
	"time"

	"github.com/google/uuid"
)

// PMCounter 表示一条性能管理计数器采样。
// T-0164-P3 / G3 后 pm_counters 表已合入统一表 pm_metrics（metric_type='counter'），
// 本 struct 作为 pm.Collector / handler / KPIEngine 内部的传输类型保留；
// pm/counter.PgCounterRepository 作为 pm/metrics.Repository 的薄包装做字段双向转换。
//
// 设备唯一标识：按 TR-069 标准使用 (OUI, DeviceSN) 双键（T-0164-P3 fix 引入）。
// DeviceID (uuid) 保留作 internal PK 与 devices 表关联，业务键以 OUI+DeviceSN 为准。
// 完整系统级切换见 docs/project/plan-T-0165-system-wide-oui-sn-migration.md。
//
// 时间列 (Time) 是 TimescaleDB 分区键，Granularity 单位为分钟。
type PMCounter struct {
	Time         time.Time `json:"time" db:"time"`
	DeviceID     uuid.UUID `json:"device_id" db:"device_id"` // internal PK，与 devices.id 对应
	OUI          string    `json:"oui" db:"device_oui"`      // TR-069 DeviceId.OUI（6 位 hex）
	DeviceSN     string    `json:"device_sn" db:"device_sn"` // TR-069 DeviceId.SerialNumber
	CellID       string    `json:"cell_id" db:"cell_id"`
	CounterGroup string    `json:"counter_group" db:"counter_group"`
	CounterName  string    `json:"counter_name" db:"counter_name"`
	CounterValue float64   `json:"counter_value" db:"counter_value"`
	Granularity  int       `json:"granularity" db:"granularity"` // minutes
	// StatisType 标记该 counter 的聚合方式（sum/avg/max/pct），由 collector 在
	// filterByWhitelist 阶段从 indicator 元数据填入，counterToMetric 透传到
	// pm_metrics.statis_type 列，驱动 G5 自然桶聚合 (CASE WHEN m.statis_type)。
	// 空串表示未知（白名单未注入或 lookup fail-open 场景），下游聚合会跳过该行。
	StatisType   string    `json:"statis_type,omitempty" db:"-"`
}

// KPIValue 表示一个计算后的 KPI 指标值。
// T-0164-P3 / G3 后 kpi_values 表已合入统一表 pm_metrics（metric_type='kpi'），
// 本 struct 作为 pm.KPICalculator / pm.Handler 内部的传输类型保留；
// pm/kpi.PgKPIRepository 作为 pm/metrics.Repository 的薄包装做字段双向转换。
//
// 设备唯一标识：按 TR-069 标准使用 (OUI, DeviceSN) 双键（T-0164-P3 fix 引入），
// 与 PMCounter 一致。
type KPIValue struct {
	Time        time.Time   `json:"time" db:"time"`
	DeviceID    uuid.UUID   `json:"device_id" db:"device_id"`
	OUI         string      `json:"oui" db:"device_oui"`
	DeviceSN    string      `json:"device_sn" db:"device_sn"`
	CellID      string      `json:"cell_id" db:"cell_id"`
	IndicatorID string      `json:"indicator_id" db:"indicator_id"` // K 编号，落库 metric_path 用
	KPIName     string      `json:"kpi_name" db:"kpi_name"`         // 显示名（en_name），日志/兼容用
	KPIValue    float64     `json:"kpi_value" db:"kpi_value"`
	Carrier     CarrierCode `json:"carrier" db:"carrier"`
	Technology  Technology  `json:"technology" db:"technology"`
}

// KPIDefinition 描述 KPI 的计算公式和元数据。
// 注意：此结构体与 carrier.KPIDefinition 的区别——
// 此处包含 Carrier/Technology 字段，用于入库和查询（存入 DB）；
// carrier.KPIDefinition 由运营商适配器返回，不含 Carrier/Technology 字段。
type KPIDefinition struct {
	Name        string      `json:"name"`
	DisplayName string      `json:"display_name"`
	Formula     string      `json:"formula"`
	Unit        string      `json:"unit"`
	Carrier     CarrierCode `json:"carrier"`
	Technology  Technology  `json:"technology"`
	Counters    []string    `json:"counters"` // dependent counter names
}
