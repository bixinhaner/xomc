package model

import (
	"time"

	"github.com/google/uuid"
)

// PMCounter 表示一条性能管理计数器采样。
// T-0164-P3 / G3 后 pm_counters 表已合入统一表 pm_metrics（metric_type='counter'），
// 本 struct 作为 pm.Collector / handler / KPIEngine 内部的传输类型保留；
// pm/counter.PgCounterRepository 作为 pm/metrics.Repository 的薄包装做字段双向转换。
// 时间列 (Time) 是 TimescaleDB 分区键，Granularity 单位为分钟。
type PMCounter struct {
	Time         time.Time `json:"time" db:"time"`
	DeviceID     uuid.UUID `json:"device_id" db:"device_id"`
	CellID       string    `json:"cell_id" db:"cell_id"`
	CounterGroup string    `json:"counter_group" db:"counter_group"`
	CounterName  string    `json:"counter_name" db:"counter_name"`
	CounterValue float64   `json:"counter_value" db:"counter_value"`
	Granularity  int       `json:"granularity" db:"granularity"` // minutes
}

// KPIValue 表示一个计算后的 KPI 指标值。
// T-0164-P3 / G3 后 kpi_values 表已合入统一表 pm_metrics（metric_type='kpi'），
// 本 struct 作为 pm.KPICalculator / pm.Handler 内部的传输类型保留；
// pm/kpi.PgKPIRepository 作为 pm/metrics.Repository 的薄包装做字段双向转换。
type KPIValue struct {
	Time       time.Time   `json:"time" db:"time"`
	DeviceID   uuid.UUID   `json:"device_id" db:"device_id"`
	CellID     string      `json:"cell_id" db:"cell_id"`
	KPIName    string      `json:"kpi_name" db:"kpi_name"`
	KPIValue   float64     `json:"kpi_value" db:"kpi_value"`
	Carrier    CarrierCode `json:"carrier" db:"carrier"`
	Technology Technology  `json:"technology" db:"technology"`
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
