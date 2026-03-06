package dashboard

import (
	"context"
	"time"

	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/common/model"
	"github.com/omcgo/omcgo/internal/omcr/device"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// DashboardSummary is the aggregated dashboard response.
type DashboardSummary struct {
	DeviceStats  map[model.DeviceStatus]int64 `json:"device_stats"`
	AlarmStats   *alarm.AlarmStatistics       `json:"alarm_stats"`
	KPIOverview  []model.KPIValue             `json:"kpi_overview"`
	RecentAlarms []model.Alarm                `json:"recent_alarms"`
	Timestamp    time.Time                    `json:"timestamp"`
}

// Service aggregates data from multiple modules for the dashboard.
type Service struct {
	deviceService *device.DeviceService
	alarmStore    alarm.AlarmStore
	kpiRepo       kpi.KPIRepository
	logger        *zap.Logger
}

// NewService creates a new dashboard service.
func NewService(
	deviceService *device.DeviceService,
	alarmStore alarm.AlarmStore,
	kpiRepo kpi.KPIRepository,
	logger *zap.Logger,
) *Service {
	return &Service{
		deviceService: deviceService,
		alarmStore:    alarmStore,
		kpiRepo:       kpiRepo,
		logger:        logger.Named("dashboard"),
	}
}

// GetSummary aggregates dashboard data from multiple sources in parallel.
func (s *Service) GetSummary(ctx context.Context) (*DashboardSummary, error) {
	summary := &DashboardSummary{
		Timestamp: time.Now(),
	}

	g, ctx := errgroup.WithContext(ctx)

	// 1. Device counts by status
	g.Go(func() error {
		counts, err := s.deviceService.CountByStatus(ctx, nil)
		if err != nil {
			s.logger.Warn("dashboard: device count failed", zap.Error(err))
			summary.DeviceStats = make(map[model.DeviceStatus]int64)
			return nil
		}
		summary.DeviceStats = counts
		return nil
	})

	// 2. Alarm statistics
	g.Go(func() error {
		stats, err := s.alarmStore.Statistics(ctx, alarm.AlarmFilter{})
		if err != nil {
			s.logger.Warn("dashboard: alarm stats failed", zap.Error(err))
			summary.AlarmStats = &alarm.AlarmStatistics{
				BySeverity: make(map[model.AlarmSeverity]int64),
				ByType:     make(map[string]int64),
			}
			return nil
		}
		summary.AlarmStats = stats
		return nil
	})

	// 3. Recent KPI values (last 24h, top 10)
	g.Go(func() error {
		now := time.Now()
		filter := kpi.KPIFilter{
			StartTime: now.Add(-24 * time.Hour),
			EndTime:   now,
		}
		filter.Page = 1
		filter.PageSize = 10
		filter.SortBy = "time"
		filter.SortDir = "desc"
		result, err := s.kpiRepo.Query(ctx, filter)
		if err != nil {
			s.logger.Warn("dashboard: kpi query failed", zap.Error(err))
			summary.KPIOverview = []model.KPIValue{}
			return nil
		}
		summary.KPIOverview = result.Items
		return nil
	})

	// 4. Recent top 5 alarms
	g.Go(func() error {
		filter := alarm.AlarmFilter{}
		filter.Page = 1
		filter.PageSize = 5
		filter.SortBy = "raised_at"
		filter.SortDir = "desc"
		result, err := s.alarmStore.ListActive(ctx, filter)
		if err != nil {
			s.logger.Warn("dashboard: recent alarms failed", zap.Error(err))
			summary.RecentAlarms = []model.Alarm{}
			return nil
		}
		summary.RecentAlarms = result.Items
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return summary, nil
}
