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
}
