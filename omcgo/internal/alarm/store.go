package alarm

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// AlarmFilter defines query parameters for alarm retrieval.
type AlarmFilter struct {
	DeviceID  *uuid.UUID
	DeviceSN  *string
	Carrier   *model.CarrierCode
	Severity  *model.AlarmSeverity
	Severities []model.AlarmSeverity
	Status    *model.AlarmStatus
	StartTime *time.Time
	EndTime   *time.Time
	model.ListRequest
	// 新增过滤字段
	AlarmIdentifiers []string           `form:"alarm_identifiers"`
	AlarmSources   []string           `form:"alarm_sources"`
	AlarmType      *string            `form:"alarm_type"`
	EventType      *string            `form:"event_type"`
	IsRead         *bool              `form:"is_read"`
	IsUnknown      *bool              `form:"is_unknown"`
	DeviceName     *string            `form:"device_name"`
	Keyword        *string            `form:"keyword"`
	// 数据权限
	DeviceIDs      []uuid.UUID        `form:"-"`
	Technologies   []string           `form:"-"`
}

// AlarmStatistics contains aggregated alarm metrics.
type AlarmStatistics struct {
	TotalActive     int64                         `json:"total_active"`
	Unacknowledged int64                         `json:"unacknowledged"`
	Unread          int64                         `json:"unread"`
	BySeverity      map[model.AlarmSeverity]int64 `json:"by_severity"`
	ByType          map[string]int64              `json:"by_type"`
}

// AlarmStore defines the interface for alarm persistence.
type AlarmStore interface {
	SaveActive(ctx context.Context, alarm *model.Alarm) error
	GetActiveByID(ctx context.Context, id uuid.UUID) (*model.Alarm, error)
	GetHistoryByID(ctx context.Context, id uuid.UUID) (*model.Alarm, error)
	GetActiveByDeviceAndIdentifier(ctx context.Context, deviceSN string, alarmIdentifier string) (*model.Alarm, error)
	GetActiveByDeviceSN(ctx context.Context, deviceSN string) ([]*model.Alarm, error)
	UpdateActive(ctx context.Context, alarm *model.Alarm) error
	RemoveActive(ctx context.Context, id uuid.UUID) error
	ListActive(ctx context.Context, filter AlarmFilter) (*model.ListResponse[model.Alarm], error)
	Archive(ctx context.Context, alarm *model.Alarm) error
	ListHistory(ctx context.Context, filter AlarmFilter) (*model.ListResponse[model.Alarm], error)
	Statistics(ctx context.Context, filter AlarmFilter) (*AlarmStatistics, error)
	HistoryStatistics(ctx context.Context, filter AlarmFilter) (*AlarmStatistics, error)
	// 批量操作
	BatchAcknowledge(ctx context.Context, ids []uuid.UUID, by string, note string) error
	BatchUnacknowledge(ctx context.Context, ids []uuid.UUID) error
	BatchClear(ctx context.Context, ids []uuid.UUID, by string, note string) error
	BatchHistoryAcknowledge(ctx context.Context, ids []uuid.UUID, by string, note string) error
	BatchHistoryUnacknowledge(ctx context.Context, ids []uuid.UUID) error
	BatchHistoryDelete(ctx context.Context, ids []uuid.UUID) error
	MarkRead(ctx context.Context, id uuid.UUID) error
}
