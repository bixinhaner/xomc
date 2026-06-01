package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"github.com/omcgo/omcgo/internal/topology"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

// FrontendDeviceStats matches the frontend's expected device_stats format.
type FrontendDeviceStats struct {
	Total   int64 `json:"total"`
	Online  int64 `json:"online"`
	Offline int64 `json:"offline"`
	Alarm   int64 `json:"alarm"`
}

// FrontendAlarmStats matches the frontend's expected alarm_stats format.
type FrontendAlarmStats struct {
	Critical int64 `json:"critical"`
	Major    int64 `json:"major"`
	Minor    int64 `json:"minor"`
	Warning  int64 `json:"warning"`
	Total    int64 `json:"total"`
}

// FrontendRecentAlarm matches the frontend's expected recent_alarms format.
type FrontendRecentAlarm struct {
	DeviceSN   string `json:"device_sn"`   // 完整设备 SN
	Technology string `json:"technology"`  // 技术类型 (lte/nr/gsm)
	DeviceName string `json:"device_name"` // 完整设备 SN（与 device_sn 相同）
	AlarmCount int64  `json:"alarm_count"` // 告警数量
	Severity   string `json:"severity"`    // 严重程度
}

// KPIDelta represents the trend comparison data for a single KPI metric.
// 用于KPI卡片显示趋势数据（如设备总数、活跃告警等的变化趋势）
type KPIDelta struct {
	CurrentValue  float64  `json:"current_value"`
	PreviousValue float64  `json:"previous_value"`
	ChangePercent float64  `json:"change_percent"`  // 变化百分比，正数表示增长
	Trend         string   `json:"trend"`           // "up" | "down" | "stable"
	CompareType   string   `json:"compare_type"`    // "yesterday" | "last_week"
}

// DashboardSummary is the aggregated dashboard response.
type DashboardSummary struct {
	DeviceStats  FrontendDeviceStats   `json:"device_stats"`
	AlarmStats   FrontendAlarmStats    `json:"alarm_stats"`
	KPIOverview  map[string]float64    `json:"kpi_overview"`
	KPIDeltas    map[string]KPIDelta   `json:"kpi_deltas"`     // KPI趋势数据（新增）
	RecentAlarms []FrontendRecentAlarm `json:"recent_alarms"`
	Timestamp    time.Time             `json:"timestamp"`
}

// AlarmTrendEntry represents alarm counts for a single day, broken down by severity.
type AlarmTrendEntry struct {
	Date     string `json:"date"` // "2026-03-07"
	Critical int64  `json:"critical"`
	Major    int64  `json:"major"`
	Minor    int64  `json:"minor"`
	Warning  int64  `json:"warning"`
}

// KPITrendEntry represents a single KPI data point in a time series.
type KPITrendEntry struct {
	Time  string  `json:"time"` // ISO 8601 timestamp
	Value float64 `json:"value"`
}

// KPITrendComparison represents KPI trend data with comparison.
type KPITrendComparison struct {
	Current  []KPITrendEntry        `json:"current"`
	Compare  []KPITrendEntry        `json:"compare"`
	Metadata KPITrendComparisonMeta `json:"metadata"`
}

// KPITrendComparisonMeta represents metadata for KPI trend comparison.
type KPITrendComparisonMeta struct {
	KPIName       string  `json:"kpi_name"`
	CompareType   string  `json:"compare_type"`   // "yesterday" or "last_week"
	ChangePercent *float64 `json:"change_percent,omitempty"`
}

// RegionStatEntry represents aggregated statistics for a device group/region.
type RegionStatEntry struct {
	Region      string `json:"region"`
	DeviceCount int64  `json:"device_count"`
	OnlineCount int64  `json:"online_count"`
	AlarmCount  int64  `json:"alarm_count"`
}

// WidgetLayout represents a user's dashboard widget layout stored as JSONB.
type WidgetLayout struct {
	ID        uuid.UUID       `json:"id"`
	UserID    uuid.UUID       `json:"user_id"`
	Layout    json.RawMessage `json:"layout"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// AlarmTypePieEntry represents alarm counts for a single alarm type.
type AlarmTypePieEntry struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

// KPITimeSeriesEntry represents a single data point within a named KPI series.
type KPITimeSeriesEntry struct {
	Time  string  `json:"time"`
	Value float64 `json:"value"`
}

// KPITimeSeriesResponse maps KPI names to their time-series data.
type KPITimeSeriesResponse map[string][]KPITimeSeriesEntry

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
		Timestamp:   time.Now(),
		KPIOverview: make(map[string]float64),
	}

	var (
		rawDeviceCounts  map[model.DeviceStatus]int64
		rawAlarmStats    *alarm.AlarmStatistics
		rawKPIValues     []model.KPIValue
		rawAlarms        []model.Alarm
		alarmDeviceCount int64
	)

	g, ctx := errgroup.WithContext(ctx)

	// 1. Device counts by status
	g.Go(func() error {
		counts, err := s.deviceService.CountByStatus(ctx, nil)
		if err != nil {
			s.logger.Warn("dashboard: device count failed", zap.Error(err))
			rawDeviceCounts = make(map[model.DeviceStatus]int64)
			return nil
		}
		rawDeviceCounts = counts
		return nil
	})

	// 2. Alarm statistics
	g.Go(func() error {
		stats, err := s.alarmStore.Statistics(ctx, alarm.AlarmFilter{})
		if err != nil {
			s.logger.Warn("dashboard: alarm stats failed", zap.Error(err))
			rawAlarmStats = &alarm.AlarmStatistics{
				BySeverity: make(map[model.AlarmSeverity]int64),
				ByType:     make(map[string]int64),
			}
			return nil
		}
		rawAlarmStats = stats
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
			rawKPIValues = []model.KPIValue{}
			return nil
		}
		rawKPIValues = result.Items
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
			rawAlarms = []model.Alarm{}
			return nil
		}
		rawAlarms = result.Items
		return nil
	})

	// 5. Count devices with active alarms
	g.Go(func() error {
		query, args, err := storage.Psql.Select("COUNT(DISTINCT device_id)").
			From("alarms_active").
			Where(sq.Eq{"status": "active"}).
			ToSql()
		if err != nil {
			s.logger.Warn("dashboard: build alarm device count query failed", zap.Error(err))
			return nil
		}
		if err := s.pgPool.QueryRow(ctx, query, args...).Scan(&alarmDeviceCount); err != nil {
			s.logger.Warn("dashboard: alarm device count failed", zap.Error(err))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Map device counts to frontend format
	var total int64
	for _, cnt := range rawDeviceCounts {
		total += cnt
	}
	online := rawDeviceCounts[model.DeviceActive]
	offline := total - online
	summary.DeviceStats = FrontendDeviceStats{
		Total:   total,
		Online:  online,
		Offline: offline,
		Alarm:   alarmDeviceCount,
	}

	// Map alarm stats to frontend format
	if rawAlarmStats != nil {
		summary.AlarmStats = FrontendAlarmStats{
			Critical: rawAlarmStats.BySeverity[model.AlarmCritical],
			Major:    rawAlarmStats.BySeverity[model.AlarmMajor],
			Minor:    rawAlarmStats.BySeverity[model.AlarmMinor],
			Warning:  rawAlarmStats.BySeverity[model.AlarmWarning],
			Total:    rawAlarmStats.TotalActive,
		}
	}

	// Map KPI values to named fields (use latest value per KPI name)
	for _, v := range rawKPIValues {
		key := v.KPIName
		if _, exists := summary.KPIOverview[key]; !exists {
			summary.KPIOverview[key] = v.KPIValue
		}
	}

	// Map recent alarms to frontend format (aggregate by device)
	deviceAlarms := make(map[string]*FrontendRecentAlarm)
	for _, a := range rawAlarms {
		key := a.DeviceSN
		if entry, exists := deviceAlarms[key]; exists {
			entry.AlarmCount++
			// Keep highest severity
			if severityLabel(a.Severity) < severityLabel(model.AlarmSeverity(severityFromLabel(entry.Severity))) {
				entry.Severity = severityToLabel(a.Severity)
			}
		} else {
			deviceAlarms[key] = &FrontendRecentAlarm{
				DeviceSN:   a.DeviceSN,                         // 完整设备 SN
				Technology: derefOrEmpty(a.Technology),         // 技术类型
				DeviceName: a.DeviceSN,                         // 与 device_sn 相同，使用完整 SN
				AlarmCount: 1,
				Severity:   severityToLabel(a.Severity),
			}
		}
	}
	summary.RecentAlarms = make([]FrontendRecentAlarm, 0, len(deviceAlarms))
	for _, entry := range deviceAlarms {
		summary.RecentAlarms = append(summary.RecentAlarms, *entry)
	}

	// Calculate KPI deltas (trend data for cards)
	summary.KPIDeltas = s.calculateKPIDeltas(ctx, total, summary.AlarmStats.Total)

	return summary, nil
}

func severityToLabel(s model.AlarmSeverity) string {
	switch s {
	case model.AlarmCritical:
		return "critical"
	case model.AlarmMajor:
		return "major"
	case model.AlarmMinor:
		return "minor"
	case model.AlarmWarning:
		return "warning"
	default:
		return "unknown"
	}
}

func severityLabel(s model.AlarmSeverity) int {
	return int(s)
}

func severityFromLabel(label string) int {
	switch label {
	case "critical":
		return 1
	case "major":
		return 2
	case "minor":
		return 3
	case "warning":
		return 4
	default:
		return 5
	}
}

// derefOrEmpty safely dereferences a string pointer, returning empty string if nil.
func derefOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// calculateKPIDeltas calculates trend data for dashboard KPI cards.
// Compares current values with previous period (yesterday for real-time metrics, last week for daily metrics).
func (s *Service) calculateKPIDeltas(ctx context.Context, currentTotalDevices int64, currentTotalAlarms int64) map[string]KPIDelta {
	now := time.Now()
	yesterdayStart := now.Add(-24 * time.Hour).Truncate(24 * time.Hour)
	yesterdayEnd := yesterdayStart.Add(24 * time.Hour)
	lastWeekStart := now.Add(-7 * 24 * time.Hour).Truncate(24 * time.Hour)
	lastWeekEnd := lastWeekStart.Add(24 * time.Hour)

	deltas := make(map[string]KPIDelta)

	// 1. Total devices trend (compare with last week same time)
	prevTotalDevices, err := s.countDevicesAtTime(ctx, lastWeekEnd)
	if err == nil && prevTotalDevices > 0 {
		deltas["total_devices"] = computeKPIDelta(float64(currentTotalDevices), float64(prevTotalDevices), "last_week")
	} else {
		// Fallback: no trend data if query fails
		deltas["total_devices"] = KPIDelta{
			CurrentValue:  float64(currentTotalDevices),
			PreviousValue: 0,
			ChangePercent: 0,
			Trend:         "stable",
			CompareType:   "last_week",
		}
	}

	// 2. Active alarms trend (compare with yesterday)
	prevTotalAlarms, err := s.countAlarmsAtTime(ctx, yesterdayEnd)
	if err == nil {
		deltas["active_alarms"] = computeKPIDelta(float64(currentTotalAlarms), float64(prevTotalAlarms), "yesterday")
	} else {
		deltas["active_alarms"] = KPIDelta{
			CurrentValue:  float64(currentTotalAlarms),
			PreviousValue: 0,
			ChangePercent: 0,
			Trend:         "stable",
			CompareType:   "yesterday",
		}
	}

	return deltas
}

// countDevicesAtTime counts total devices at a specific point in time.
//
// TODO(T-0164-P4): Implement historical device count query.
// Current implementation returns current device count regardless of time parameter,
// which means trend comparison data is not accurate.
//
// Future implementation options:
//   Option 1: Add device_history table to track device status changes over time
//   Option 2: Use time-series database (TimescaleDB) to store historical device counts
//   Option 3: Query device lifecycle events to reconstruct historical counts
//
// For now, this serves as a baseline implementation for UI development.
func (s *Service) countDevicesAtTime(ctx context.Context, t time.Time) (int64, error) {
	// Log warning that this is not accurate historical data
	s.logger.Warn(
		"countDevicesAtTime: returning current count, historical data not available",
		zap.String("requested_time", t.Format(time.RFC3339)),
		zap.String("note", "TODO: implement historical device count query (see T-0164-P4)"),
	)

	counts, err := s.deviceService.CountByStatus(ctx, nil)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, cnt := range counts {
		total += cnt
	}
	return total, nil
}

// countAlarmsAtTime counts active alarms at a specific point in time.
//
// TODO(T-0164-P4): Implement historical alarm count query using alarm_history table.
// Current implementation returns current alarm count regardless of time parameter,
// which means trend comparison data is not accurate.
//
// Future implementation:
//   Query alarm_history table with time filter: raised_at <= t AND (cleared_at IS NULL OR cleared_at > t)
//
// For now, this serves as a baseline implementation for UI development.
func (s *Service) countAlarmsAtTime(ctx context.Context, t time.Time) (int64, error) {
	// Log warning that this is not accurate historical data
	s.logger.Warn(
		"countAlarmsAtTime: returning current count, historical data not available",
		zap.String("requested_time", t.Format(time.RFC3339)),
		zap.String("note", "TODO: implement historical alarm count query (see T-0164-P4)"),
	)

	stats, err := s.alarmStore.Statistics(ctx, alarm.AlarmFilter{})
	if err != nil {
		return 0, err
	}
	return stats.TotalActive, nil
}

// computeKPIDelta calculates delta values for a single KPI metric.
func computeKPIDelta(current, previous float64, compareType string) KPIDelta {
	delta := KPIDelta{
		CurrentValue:  current,
		PreviousValue: previous,
		CompareType:   compareType,
	}

	if previous == 0 {
		// Avoid division by zero
		if current > 0 {
			delta.ChangePercent = 100
			delta.Trend = "up"
		} else {
			delta.ChangePercent = 0
			delta.Trend = "stable"
		}
		return delta
	}

	delta.ChangePercent = ((current - previous) / previous) * 100

	// Determine trend direction with a small threshold for "stable"
	const threshold = 0.5 // 0.5% threshold for stable
	if delta.ChangePercent > threshold {
		delta.Trend = "up"
	} else if delta.ChangePercent < -threshold {
		delta.Trend = "down"
	} else {
		delta.Trend = "stable"
	}

	return delta
}

// GetAlarmTrend returns alarm counts grouped by date and severity for the last N days.
func (s *Service) GetAlarmTrend(ctx context.Context, days int) ([]AlarmTrendEntry, error) {
	if days < 1 {
		days = 7
	}
	if days > 365 {
		days = 365
	}

	// Complex aggregation with DATE(), CASE WHEN, COALESCE — raw SQL preferred over Squirrel for readability
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

// GetKPITrendComparison returns KPI trend data with comparison (today vs yesterday/last week).
func (s *Service) GetKPITrendComparison(ctx context.Context, kpiName string, compareWith string) (*KPITrendComparison, error) {
	now := time.Now()
	var compareStart, compareEnd time.Time

	// Determine comparison period
	switch compareWith {
	case "yesterday":
		// Current: today 00:00 to now
		// Compare: yesterday 00:00 to 23:59:59
		compareStart = now.AddDate(0, 0, -1).Truncate(24 * time.Hour)
		compareEnd = now.Truncate(24 * time.Hour).Add(-time.Second)
	case "last_week":
		// Current: this week (Monday to now)
		// Compare: last week (Monday to Sunday)
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday = 7
		}
		compareStart = now.AddDate(0, 0, -weekday-6).Truncate(24 * time.Hour) // Last Monday
		compareEnd = compareStart.AddDate(0, 0, 6).Add(24*time.Hour - time.Second) // Last Sunday
	default:
		// Default to yesterday
		compareWith = "yesterday"
		compareStart = now.AddDate(0, 0, -1).Truncate(24 * time.Hour)
		compareEnd = now.Truncate(24 * time.Hour).Add(-time.Second)
	}

	// Query current period data (today 00:00 to now)
	currentStart := now.Truncate(24 * time.Hour)
	currentFilter := kpi.KPIFilter{
		KPIName:   &kpiName,
		StartTime: currentStart,
		EndTime:   now,
	}
	currentFilter.Page = 1
	currentFilter.PageSize = 100
	currentFilter.SortBy = "time"
	currentFilter.SortDir = "asc"

	currentResult, err := s.kpiRepo.Query(ctx, currentFilter)
	if err != nil {
		return nil, fmt.Errorf("query current kpi trend: %w", err)
	}

	currentEntries := make([]KPITrendEntry, 0, len(currentResult.Items))
	for _, v := range currentResult.Items {
		currentEntries = append(currentEntries, KPITrendEntry{
			Time:  v.Time.Format(time.RFC3339),
			Value: v.KPIValue,
		})
	}

	// Query comparison period data
	compareFilter := kpi.KPIFilter{
		KPIName:   &kpiName,
		StartTime: compareStart,
		EndTime:   compareEnd,
	}
	compareFilter.Page = 1
	compareFilter.PageSize = 100
	compareFilter.SortBy = "time"
	compareFilter.SortDir = "asc"

	compareResult, err := s.kpiRepo.Query(ctx, compareFilter)
	if err != nil {
		return nil, fmt.Errorf("query compare kpi trend: %w", err)
	}

	compareEntries := make([]KPITrendEntry, 0, len(compareResult.Items))
	for _, v := range compareResult.Items {
		compareEntries = append(compareEntries, KPITrendEntry{
			Time:  v.Time.Format(time.RFC3339),
			Value: v.KPIValue,
		})
	}

	// Calculate change percent if both periods have data
	var changePercent *float64
	if len(currentEntries) > 0 && len(compareEntries) > 0 {
		// Use average values for comparison
		currentSum := 0.0
		for _, e := range currentEntries {
			currentSum += e.Value
		}
		currentAvg := currentSum / float64(len(currentEntries))

		compareSum := 0.0
		for _, e := range compareEntries {
			compareSum += e.Value
		}
		compareAvg := compareSum / float64(len(compareEntries))

		if compareAvg != 0 {
			change := ((currentAvg - compareAvg) / compareAvg) * 100
			changePercent = &change
		}
	}

	return &KPITrendComparison{
		Current:  currentEntries,
		Compare:  compareEntries,
		Metadata: KPITrendComparisonMeta{
			KPIName:       kpiName,
			CompareType:   compareWith,
			ChangePercent: changePercent,
		},
	}, nil
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
			// Count online devices (is_online = TRUE) in this group
			// T-0162: 使用 is_online 字段（migration 000137 替换了原 status 列）
			g2, gctx := errgroup.WithContext(ctx)

			g2.Go(func() error {
				onlineQuery, args, err := storage.Psql.Select("COUNT(*)").
					From("devices").
					Where("id = ANY(?)", deviceIDs).
					Where(sq.Eq{"is_online": true}).
					ToSql()
				if err != nil {
					s.logger.Warn("dashboard: build online devices query failed",
						zap.String("group", entry.Region), zap.Error(err))
					return nil
				}
				if err := s.pgPool.QueryRow(gctx, onlineQuery, args...).Scan(&entry.OnlineCount); err != nil {
					s.logger.Warn("dashboard: count online devices failed",
						zap.String("group", entry.Region), zap.Error(err))
				}
				return nil
			})

			g2.Go(func() error {
				alarmQuery, args, err := storage.Psql.Select("COUNT(*)").
					From("alarms_active").
					Where("device_id = ANY(?)", deviceIDs).
					ToSql()
				if err != nil {
					s.logger.Warn("dashboard: build group alarms query failed",
						zap.String("group", entry.Region), zap.Error(err))
					return nil
				}
				if err := s.pgPool.QueryRow(gctx, alarmQuery, args...).Scan(&entry.AlarmCount); err != nil {
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

// GetWidgetLayout retrieves the widget layout for a specific user.
func (s *Service) GetWidgetLayout(ctx context.Context, userID uuid.UUID) (*WidgetLayout, error) {
	query, args, err := storage.Psql.Select("id", "user_id", "layout", "created_at", "updated_at").
		From("dashboard_widgets").
		Where(sq.Eq{"user_id": userID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build widget layout query: %w", err)
	}

	var w WidgetLayout
	err = s.pgPool.QueryRow(ctx, query, args...).Scan(
		&w.ID, &w.UserID, &w.Layout, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		// Return empty layout if none found (pgx returns error for no rows)
		if err.Error() == "no rows in result set" {
			return &WidgetLayout{
				UserID: userID,
				Layout: json.RawMessage("[]"),
			}, nil
		}
		return nil, fmt.Errorf("query widget layout: %w", err)
	}
	return &w, nil
}

// SaveWidgetLayout upserts the widget layout for a specific user.
func (s *Service) SaveWidgetLayout(ctx context.Context, userID uuid.UUID, layout json.RawMessage) (*WidgetLayout, error) {
	query, args, err := storage.Psql.Insert("dashboard_widgets").
		Columns("user_id", "layout").
		Values(userID, layout).
		Suffix("ON CONFLICT (user_id) DO UPDATE SET layout = EXCLUDED.layout, updated_at = NOW() RETURNING id, user_id, layout, created_at, updated_at").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build upsert widget layout query: %w", err)
	}

	var w WidgetLayout
	err = s.pgPool.QueryRow(ctx, query, args...).Scan(
		&w.ID, &w.UserID, &w.Layout, &w.CreatedAt, &w.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert widget layout: %w", err)
	}
	return &w, nil
}

// GetAlarmTypePie returns alarm counts grouped by alarm_type.
func (s *Service) GetAlarmTypePie(ctx context.Context) ([]AlarmTypePieEntry, error) {
	// Complex aggregation with COALESCE/NULLIF and GROUP BY alias — raw SQL preferred over Squirrel for readability
	query := `SELECT
			COALESCE(NULLIF(alarm_type, ''), '其他告警') AS atype,
			COUNT(*) AS cnt
		FROM alarms_active
		GROUP BY atype
		ORDER BY cnt DESC`

	rows, err := s.pgPool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query alarm type pie: %w", err)
	}
	defer rows.Close()

	var entries []AlarmTypePieEntry
	for rows.Next() {
		var e AlarmTypePieEntry
		if err := rows.Scan(&e.Name, &e.Value); err != nil {
			return nil, fmt.Errorf("scan alarm type pie row: %w", err)
		}
		entries = append(entries, e)
	}
	if entries == nil {
		entries = []AlarmTypePieEntry{}
	}
	return entries, nil
}

// GetKPITimeSeries returns time-series data for multiple KPI names within a time range.
func (s *Service) GetKPITimeSeries(ctx context.Context, kpiNames []string, startTime, endTime time.Time) (KPITimeSeriesResponse, error) {
	result := make(KPITimeSeriesResponse, len(kpiNames))

	if len(kpiNames) == 0 {
		return result, nil
	}

	// Initialize empty slices for all requested names
	for _, name := range kpiNames {
		result[name] = []KPITimeSeriesEntry{}
	}

	// T-0164-P3 / G3：kpi_values 表合入 pm_metrics（metric_type='kpi'），列改名
	// kpi_name → metric_path，kpi_value → metric_value。
	query, args, err := storage.Psql.Select("metric_path", "time", "metric_value").
		From("pm_metrics").
		Where(sq.Eq{"metric_type": "kpi"}).
		Where("metric_path = ANY(?)", kpiNames).
		Where(sq.GtOrEq{"time": startTime}).
		Where(sq.LtOrEq{"time": endTime}).
		OrderBy("metric_path", "time ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build kpi time series query: %w", err)
	}

	rows, err := s.pgPool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query kpi time series: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var kpiName string
		var t time.Time
		var value float64
		if err := rows.Scan(&kpiName, &t, &value); err != nil {
			return nil, fmt.Errorf("scan kpi time series row: %w", err)
		}
		result[kpiName] = append(result[kpiName], KPITimeSeriesEntry{
			Time:  t.Format(time.RFC3339),
			Value: value,
		})
	}

	return result, nil
}

// parseKPINames splits a comma-separated string of KPI names into a slice.
func parseKPINames(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	names := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			names = append(names, trimmed)
		}
	}
	return names
}

// DeviceStatusByType is device status statistics grouped by technology.
type DeviceStatusByType map[string]DeviceStatusCounts

// DeviceStatusCounts represents device counts for a single technology.
type DeviceStatusCounts struct {
	Online  int64 `json:"online"`
	Offline int64 `json:"offline"`
	Alarm   int64 `json:"alarm"`
}

// GetDeviceStatusByType returns device status counts grouped by technology.
// T-0162: 使用 is_online 字段（migration 000137 替换了原 status 列）
func (s *Service) GetDeviceStatusByType(ctx context.Context) (DeviceStatusByType, error) {
	// PostgreSQL FILTER syntax for conditional aggregation
	query := `
		SELECT
			technology,
			COUNT(*) FILTER (WHERE is_online = TRUE) AS online,
			COUNT(*) FILTER (WHERE is_online = FALSE) AS offline,
			COUNT(*) FILTER (WHERE EXISTS (
				SELECT 1 FROM alarms_active aa WHERE aa.device_id = devices.id
			)) AS alarm
		FROM devices
		WHERE deleted_at IS NULL
		GROUP BY technology
		ORDER BY technology`

	rows, err := s.pgPool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query device status by type: %w", err)
	}
	defer rows.Close()

	result := make(DeviceStatusByType)
	for rows.Next() {
		var technology string
		var counts DeviceStatusCounts
		if err := rows.Scan(&technology, &counts.Online, &counts.Offline, &counts.Alarm); err != nil {
			return nil, fmt.Errorf("scan device status by type row: %w", err)
		}
		result[technology] = counts
	}

	// Check for errors during iteration
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate device status by type rows: %w", err)
	}

	return result, nil
}
