package northbound

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/northbound/push"
	nbsync "github.com/omcgo/omcgo/internal/northbound/sync"
	"github.com/omcgo/omcgo/internal/pm/counter"
	"github.com/omcgo/omcgo/internal/pm/kpi"
)

// NorthboundService provides business logic for northbound/OSS data export.
type NorthboundService struct {
	alarmStore  alarm.AlarmStore
	counterRepo counter.CounterRepository
	kpiRepo     kpi.KPIRepository
	paramRepo   device.DeviceParameterRepository
	pushEngine  *push.Engine
	syncService *nbsync.Service
	logger      *zap.Logger
}

// NewNorthboundService creates a new NorthboundService.
func NewNorthboundService(
	alarmStore alarm.AlarmStore,
	counterRepo counter.CounterRepository,
	kpiRepo kpi.KPIRepository,
	paramRepo device.DeviceParameterRepository,
	pushEngine *push.Engine,
	syncService *nbsync.Service,
	logger *zap.Logger,
) *NorthboundService {
	return &NorthboundService{
		alarmStore:  alarmStore,
		counterRepo: counterRepo,
		kpiRepo:     kpiRepo,
		paramRepo:   paramRepo,
		pushEngine:  pushEngine,
		syncService: syncService,
		logger:      logger.Named("northbound"),
	}
}

// ExportAlarms queries alarms using the given filter for northbound export.
func (s *NorthboundService) ExportAlarms(ctx context.Context, filter alarm.AlarmFilter) (*model.ListResponse[model.Alarm], error) {
	result, err := s.alarmStore.ListActive(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("export alarms: %w", err)
	}
	return result, nil
}

// ListActiveAlarms returns active alarms with default pagination for northbound sync.
func (s *NorthboundService) ListActiveAlarms(ctx context.Context, listReq model.ListRequest) (*model.ListResponse[model.Alarm], error) {
	filter := alarm.AlarmFilter{
		ListRequest: listReq,
	}
	result, err := s.alarmStore.ListActive(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list active alarms: %w", err)
	}
	return result, nil
}

// ExportConfig returns all device parameters for a given device.
func (s *NorthboundService) ExportConfig(ctx context.Context, deviceID uuid.UUID) ([]model.DeviceParameter, error) {
	params, err := s.paramRepo.GetByDevice(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("export config for device %s: %w", deviceID, err)
	}
	return params, nil
}

// ExportPM queries PM counters using the given filter for northbound export.
func (s *NorthboundService) ExportPM(ctx context.Context, filter counter.CounterFilter) (*model.ListResponse[model.PMCounter], error) {
	result, err := s.counterRepo.Query(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("export PM counters: %w", err)
	}
	return result, nil
}

// ExportKPI queries KPI values using the given filter for northbound export.
func (s *NorthboundService) ExportKPI(ctx context.Context, filter kpi.KPIFilter) (*model.ListResponse[model.KPIValue], error) {
	result, err := s.kpiRepo.Query(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("export KPI values: %w", err)
	}
	return result, nil
}

// PushEngine returns the push engine for push target management.
func (s *NorthboundService) PushEngine() *push.Engine {
	return s.pushEngine
}

// SyncService returns the sync service for full/incremental sync operations.
func (s *NorthboundService) SyncService() *nbsync.Service {
	return s.syncService
}
