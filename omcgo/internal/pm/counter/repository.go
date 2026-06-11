package counter

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// CounterFilter defines query parameters for counter retrieval.
type CounterFilter struct {
	DeviceID     *uuid.UUID
	CellID       *string
	CounterGroup *string
	CounterName  *string
	StartTime    time.Time
	EndTime      time.Time
	// VisibleGroups 是 #64 设备组数据权限的三态可见分组（nil=超管 / []=fail-closed / [g...]=限定）。
	// handler 解析调用者身份后注入，仓库层透传到 metrics.QueryRequest 按 device_sn 收口。
	VisibleGroups []uuid.UUID
	model.ListRequest
}

// AggregatedCounter represents a time-bucketed aggregation of counter values.
type AggregatedCounter struct {
	Bucket       time.Time `json:"bucket" db:"bucket"`
	DeviceID     uuid.UUID `json:"device_id" db:"device_id"`
	CellID       string    `json:"cell_id" db:"cell_id"`
	CounterGroup string    `json:"counter_group" db:"counter_group"`
	CounterName  string    `json:"counter_name" db:"counter_name"`
	SumValue     float64   `json:"sum_value" db:"sum_value"`
	AvgValue     float64   `json:"avg_value" db:"avg_value"`
	MinValue     float64   `json:"min_value" db:"min_value"`
	MaxValue     float64   `json:"max_value" db:"max_value"`
	SampleCount  int64     `json:"sample_count" db:"sample_count"`
}

// CounterRepository defines the interface for PM counter persistence.
type CounterRepository interface {
	BatchInsert(ctx context.Context, counters []model.PMCounter) error
	Query(ctx context.Context, filter CounterFilter) (*model.ListResponse[model.PMCounter], error)
	QueryAggregated(ctx context.Context, filter CounterFilter) ([]AggregatedCounter, error)
	QueryForKPI(ctx context.Context, deviceID uuid.UUID, cellID string, counterNames []string, startTime, endTime time.Time) (map[string]float64, error)
	// QueryForKPICells 是 QueryForKPI 的批量版本：一次查询拿到设备在时间窗内的全部
	// counter 行，再按 cellID 分组返回 cellID → (counter_name → SUM)。语义与对每个
	// cellID 各调一次 QueryForKPI 完全等价（含 cellID=="" 表示"跨全部 cell 求和"的
	// 历史约定、period_seconds 注入），但把单文件 N(=cell 数) 次"全设备扫描+逐 cell
	// 内存过滤"塌缩成 1 次查询，消除 O(cell²) 的数据传输与 N 次 device 反查。
	QueryForKPICells(ctx context.Context, deviceID uuid.UUID, cellIDs []string, counterNames []string, startTime, endTime time.Time) (map[string]map[string]float64, error)
}
