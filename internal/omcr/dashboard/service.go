package dashboard

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/common/model"
	"github.com/omcgo/omcgo/internal/omcr/device"
	"github.com/omcgo/omcgo/internal/omcr/topology"
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

// AlarmTrendEntry represents alarm counts for a single day, broken down by severity.
type AlarmTrendEntry struct {
	Date     string `json:"date"`     // "2026-03-07"
	Critical int64  `json:"critical"`
	Major    int64  `json:"major"`
	Minor    int64  `json:"minor"`
	Warning  int64  `json:"warning"`
}

// KPITrendEntry represents a single KPI data point in a time series.
type KPITrendEntry struct {
	Time  string  `json:"time"`  // ISO 8601 timestamp
	Value float64 `json:"value"`
}

// RegionStatEntry represents aggregated statistics for a device group/region.
type RegionStatEntry struct {
	Region      string `json:"region"`
	DeviceCount int64  `json:"device_count"`
	OnlineCount int64  `json:"online_count"`
	AlarmCount  int64  `json:"alarm_count"`
}

// Service aggregates data from multiple modules for the dashboard.
type Service struct {
	deviceService *device.DeviceService
	alarmStore    alarm.AlarmStore
	kpiRepo       kpi.KPIRepository
	pgPool        *pgxpool.Pool
	groupRepo     topology.DeviceGroupRepository
	logger        *zap.Logger
}

// NewService creates a new dashboard service.
func NewService(
	deviceService *device.DeviceService,
	alarmStore alarm.AlarmStore,
	kpiRepo kpi.KPIRepository,
	pgPool *pgxpool.Pool,
	groupRepo topology.DeviceGroupRepository,
	logger *zap.Logger,
) *Service {
	return &Service{
		deviceService: deviceService,
		alarmStore:    alarmStore,
		kpiRepo:       kpiRepo,
		pgPool:        pgPool,
		groupRepo:     groupRepo,
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

// GetAlarmTrend returns alarm counts grouped by date and severity for the last N days.
func (s *Service) GetAlarmTrend(ctx context.Context, days int) ([]AlarmTrendEntry, error) {
	if days < 1 {
		days = 7
	}
	if days > 365 {
		days = 365
	}

	query := `
		SELECT
			DATE(raised_at) AS d,
			COALESCE(SUM(CASE WHEN severity = 1 THEN 1 ELSE 0 END), 0) AS critical,
			COALESCE(SUM(CASE WHEN severity = 2 THEN 1 ELSE 0 END), 0) AS major,
			COALESCE(SUM(CASE WHEN severity = 3 THEN 1 ELSE 0 END), 0) AS minor,
			COALESCE(SUM(CASE WHEN severity = 4 THEN 1 ELSE 0 END), 0) AS warning
		FROM alarms_active
		WHERE raised_at >= NOW() - $1::interval
		GROUP BY DATE(raised_at)
		ORDER BY d ASC`

	interval := fmt.Sprintf("%d days", days)
	rows, err := s.pgPool.Query(ctx, query, interval)
	if err != nil {
		return nil, fmt.Errorf("query alarm trend: %w", err)
	}
	defer rows.Close()

	var entries []AlarmTrendEntry
	for rows.Next() {
		var e AlarmTrendEntry
		var d time.Time
		if err := rows.Scan(&d, &e.Critical, &e.Major, &e.Minor, &e.Warning); err != nil {
			return nil, fmt.Errorf("scan alarm trend row: %w", err)
		}
		e.Date = d.Format("2006-01-02")
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []AlarmTrendEntry{}
	}
	return entries, nil
}

// GetDeviceStatus returns device counts grouped by status.
func (s *Service) GetDeviceStatus(ctx context.Context) (map[model.DeviceStatus]int64, error) {
	counts, err := s.deviceService.CountByStatus(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("count devices by status: %w", err)
	}
	if counts == nil {
		counts = make(map[model.DeviceStatus]int64)
	}
	return counts, nil
}

// GetKPITrend returns time-series data points for a specific KPI name over N days.
func (s *Service) GetKPITrend(ctx context.Context, kpiName string, days int) ([]KPITrendEntry, error) {
	if days < 1 {
		days = 7
	}
	if days > 365 {
		days = 365
	}

	now := time.Now()
	filter := kpi.KPIFilter{
		KPIName:   &kpiName,
		StartTime: now.AddDate(0, 0, -days),
		EndTime:   now,
	}
	filter.Page = 1
	filter.PageSize = 100
	filter.SortBy = "time"
	filter.SortDir = "asc"

	result, err := s.kpiRepo.Query(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("query kpi trend: %w", err)
	}

	entries := make([]KPITrendEntry, 0, len(result.Items))
	for _, v := range result.Items {
		entries = append(entries, KPITrendEntry{
			Time:  v.Time.Format(time.RFC3339),
			Value: v.KPIValue,
		})
	}
	return entries, nil
}

// GetRegionStats returns device and alarm statistics per device group.
func (s *Service) GetRegionStats(ctx context.Context) ([]RegionStatEntry, error) {
	groups, err := s.groupRepo.GetTree(ctx)
	if err != nil {
		s.logger.Warn("dashboard: get group tree failed", zap.Error(err))
		return []RegionStatEntry{}, nil
	}
	if len(groups) == 0 {
		return []RegionStatEntry{}, nil
	}

	entries := make([]RegionStatEntry, 0, len(groups))
	for _, g := range groups {
		entry := RegionStatEntry{
			Region: g.Name,
		}

		deviceIDs, err := s.groupRepo.ListDeviceIDs(ctx, g.ID)
		if err != nil {
			s.logger.Warn("dashboard: list device IDs for group failed",
				zap.String("group", g.Name), zap.Error(err))
			entries = append(entries, entry)
			continue
		}
		entry.DeviceCount = int64(len(deviceIDs))

		if len(deviceIDs) > 0 {
			// Count online devices (status = 'active') in this group
			g2, gctx := errgroup.WithContext(ctx)

			g2.Go(func() error {
				onlineQuery := `SELECT COUNT(*) FROM devices WHERE id = ANY($1) AND status = 'active'`
				if err := s.pgPool.QueryRow(gctx, onlineQuery, deviceIDs).Scan(&entry.OnlineCount); err != nil {
					s.logger.Warn("dashboard: count online devices failed",
						zap.String("group", entry.Region), zap.Error(err))
				}
				return nil
			})

			g2.Go(func() error {
				alarmQuery := `SELECT COUNT(*) FROM alarms_active WHERE device_id = ANY($1)`
				if err := s.pgPool.QueryRow(gctx, alarmQuery, deviceIDs).Scan(&entry.AlarmCount); err != nil {
					s.logger.Warn("dashboard: count group alarms failed",
						zap.String("group", entry.Region), zap.Error(err))
				}
				return nil
			})

			g2.Wait()
		}

		entries = append(entries, entry)
	}

	return entries, nil
}
