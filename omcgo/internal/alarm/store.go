package alarm

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/model"
)

// AlarmFilter defines query parameters for alarm retrieval.
type AlarmFilter struct {
	DeviceID  *uuid.UUID
	DeviceSN  *string
	Carrier   *model.CarrierCode
	Severity  *model.AlarmSeverity
	Status    *model.AlarmStatus
	StartTime *time.Time
	EndTime   *time.Time
	model.ListRequest
}

// AlarmStatistics contains aggregated alarm metrics.
type AlarmStatistics struct {
	TotalActive int64                         `json:"total_active"`
	BySeverity  map[model.AlarmSeverity]int64 `json:"by_severity"`
	ByType      map[string]int64              `json:"by_type"`
}

// AlarmStore defines the interface for alarm persistence.
type AlarmStore interface {
	SaveActive(ctx context.Context, alarm *model.Alarm) error
	GetActiveByID(ctx context.Context, id uuid.UUID) (*model.Alarm, error)
	GetActiveByDeviceAndCode(ctx context.Context, deviceSN string, alarmCode string) (*model.Alarm, error)
	UpdateActive(ctx context.Context, alarm *model.Alarm) error
	RemoveActive(ctx context.Context, id uuid.UUID) error
	ListActive(ctx context.Context, filter AlarmFilter) (*model.ListResponse[model.Alarm], error)
	Archive(ctx context.Context, alarm *model.Alarm) error
	ListHistory(ctx context.Context, filter AlarmFilter) (*model.ListResponse[model.Alarm], error)
	Statistics(ctx context.Context, filter AlarmFilter) (*AlarmStatistics, error)
}
