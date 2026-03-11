package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"go.uber.org/zap"
)

// Result holds the result of a sync operation.
type Result struct {
	DataType  string      `json:"data_type"`
	Items     interface{} `json:"items"`
	Total     int         `json:"total"`
	SyncedAt  time.Time   `json:"synced_at"`
	Truncated bool        `json:"truncated,omitempty"`
}

// Service aggregates device/alarm/PM/config data for northbound synchronization.
type Service struct {
	deviceRepo  device.DeviceRepository
	alarmStore  alarm.AlarmStore
	counterRepo counter.CounterRepository
	kpiRepo     kpi.KPIRepository
	paramRepo   device.DeviceParameterRepository
	logger      *zap.Logger
}

// NewService creates a new sync Service.
func NewService(
	deviceRepo device.DeviceRepository,
	alarmStore alarm.AlarmStore,
	counterRepo counter.CounterRepository,
	kpiRepo kpi.KPIRepository,
	paramRepo device.DeviceParameterRepository,
	logger *zap.Logger,
) *Service {
	return &Service{
		deviceRepo:  deviceRepo,
		alarmStore:  alarmStore,
		counterRepo: counterRepo,
		kpiRepo:     kpiRepo,
		paramRepo:   paramRepo,
		logger:      logger,
	}
}

// FullSync performs a full data synchronization for the requested data type.
func (s *Service) FullSync(ctx context.Context, dataType string) (*Result, error) {
	switch dataType {
	case "device":
		return s.syncDevices(ctx)
	case "alarm":
		return s.syncAlarms(ctx)
	case "pm":
		return s.syncPM(ctx)
	default:
		return nil, fmt.Errorf("unsupported data type for full sync: %s", dataType)
	}
}

// IncrementalSync performs an incremental synchronization since the given time.
func (s *Service) IncrementalSync(ctx context.Context, dataType string, since time.Time) (*Result, error) {
	switch dataType {
	case "alarm":
		return s.syncAlarmsSince(ctx, since)
	case "pm":
		return s.syncPMSince(ctx, since)
	default:
		return nil, fmt.Errorf("unsupported data type for incremental sync: %s", dataType)
	}
}

func (s *Service) syncDevices(ctx context.Context) (*Result, error) {
	filter := device.DeviceFilter{
		ListRequest: model.ListRequest{Page: 1, PageSize: 100},
	}

	resp, err := s.deviceRepo.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("sync devices: %w", err)
	}

	return &Result{
		DataType:  "device",
		Items:     resp.Items,
		Total:     int(resp.Total),
		SyncedAt:  time.Now(),
		Truncated: resp.Total > int64(len(resp.Items)),
	}, nil
}

func (s *Service) syncAlarms(ctx context.Context) (*Result, error) {
	filter := alarm.AlarmFilter{
		ListRequest: model.ListRequest{Page: 1, PageSize: 100},
	}

	resp, err := s.alarmStore.ListActive(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("sync alarms: %w", err)
	}

	return &Result{
		DataType:  "alarm",
		Items:     resp.Items,
		Total:     int(resp.Total),
		SyncedAt:  time.Now(),
		Truncated: resp.Total > int64(len(resp.Items)),
	}, nil
}

func (s *Service) syncAlarmsSince(ctx context.Context, since time.Time) (*Result, error) {
	filter := alarm.AlarmFilter{
		ListRequest: model.ListRequest{Page: 1, PageSize: 100},
		StartTime:   &since,
	}

	resp, err := s.alarmStore.ListActive(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("incremental sync alarms: %w", err)
	}

	return &Result{
		DataType:  "alarm",
		Items:     resp.Items,
		Total:     int(resp.Total),
		SyncedAt:  time.Now(),
		Truncated: resp.Total > int64(len(resp.Items)),
	}, nil
}

func (s *Service) syncPM(ctx context.Context) (*Result, error) {
	now := time.Now()
	filter := counter.CounterFilter{
		StartTime:   now.Add(-24 * time.Hour),
		EndTime:     now,
		ListRequest: model.ListRequest{Page: 1, PageSize: 100},
	}

	resp, err := s.counterRepo.Query(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("sync PM counters: %w", err)
	}

	return &Result{
		DataType:  "pm",
		Items:     resp.Items,
		Total:     int(resp.Total),
		SyncedAt:  time.Now(),
		Truncated: resp.Total > int64(len(resp.Items)),
	}, nil
}

func (s *Service) syncPMSince(ctx context.Context, since time.Time) (*Result, error) {
	filter := counter.CounterFilter{
		StartTime:   since,
		EndTime:     time.Now(),
		ListRequest: model.ListRequest{Page: 1, PageSize: 100},
	}

	resp, err := s.counterRepo.Query(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("incremental sync PM: %w", err)
	}

	return &Result{
		DataType:  "pm",
		Items:     resp.Items,
		Total:     int(resp.Total),
		SyncedAt:  time.Now(),
		Truncated: resp.Total > int64(len(resp.Items)),
	}, nil
}
