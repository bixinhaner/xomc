package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/pm/indicator"
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
	CurrentValue  float64 `json:"current_value"`
	PreviousValue float64 `json:"previous_value"`
	ChangePercent float64 `json:"change_percent"` // 变化百分比，正数表示增长
	Trend         string  `json:"trend"`          // "up" | "down" | "stable"
	CompareType   string  `json:"compare_type"`   // "yesterday" | "last_week"
}

// DashboardSummary is the aggregated dashboard response.
type DashboardSummary struct {
	DeviceStats  FrontendDeviceStats   `json:"device_stats"`
	AlarmStats   FrontendAlarmStats    `json:"alarm_stats"`
	KPIOverview  map[string]float64    `json:"kpi_overview"`
	KPIDeltas    map[string]KPIDelta   `json:"kpi_deltas"` // KPI趋势数据（新增）
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
	KPIName       string   `json:"kpi_name"`
	CompareType   string   `json:"compare_type"` // "yesterday" or "last_week"
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
//
// pgPool 指向主库（业务数据：alarms_active / devices / device_groups / dashboard_widgets）；
// tsPool 指向时序库（alarms_history / alarm_efficiency_metrics matview / pm_metrics）。
// KPI/时序库物理分离后，读时序表的查询必须走 tsPool，否则跨库查不到表。
type Service struct {
	deviceService *device.DeviceService
	alarmStore    alarm.AlarmStore
	kpiRepo       kpi.KPIRepository
	pgPool        *pgxpool.Pool
	tsPool        *pgxpool.Pool
	groupRepo     topology.DeviceGroupRepository
	// indicatorRepo 复用 PM 的 perf_indicators_{enb,gsm,gnb} 数据源（cnName / unit），
	// 给 GetKPIDefinitions（issue #213 Phase1）按别名表的 K 编号反查中文名与单位用。
	// 可能为 nil（测试 / 退化场景）：此时 GetKPIDefinitions 仅返回别名表静态元数据，不富化。
	indicatorRepo indicator.IndicatorRepository
	// layoutRepo 是 issue #213 S1 全局 KPI 首页布局（dashboard_kpi_layouts）读写仓库。
	// 可能为 nil（部分测试场景）：此时 GetKPILayout 回退内置默认，SaveKPILayout 报错。
	layoutRepo KPILayoutRepository
	logger     *zap.Logger
}

// NewService creates a new dashboard service.
func NewService(
	deviceService *device.DeviceService,
	alarmStore alarm.AlarmStore,
	kpiRepo kpi.KPIRepository,
	pgPool *pgxpool.Pool,
	tsPool *pgxpool.Pool,
	groupRepo topology.DeviceGroupRepository,
	indicatorRepo indicator.IndicatorRepository,
	logger *zap.Logger,
) *Service {
	s := &Service{
		deviceService: deviceService,
		alarmStore:    alarmStore,
		kpiRepo:       kpiRepo,
		pgPool:        pgPool,
		tsPool:        tsPool,
		groupRepo:     groupRepo,
		indicatorRepo: indicatorRepo,
		logger:        logger.Named("dashboard"),
	}
	// 全局 KPI 布局仓库走主库（dashboard_kpi_layouts 在主库）。pgPool 为 nil 时（测试）留空。
	if pgPool != nil {
		s.layoutRepo = NewKPILayoutRepository(pgPool)
	}
	return s
}

// GetSummary aggregates dashboard data from multiple sources in parallel.
func (s *Service) GetSummary(ctx context.Context) (*DashboardSummary, error) {
	summary := &DashboardSummary{
		Timestamp:   time.Now(),
		KPIOverview: make(map[string]float64),
	}

	var (
		totalDevices     int64
		onlineDevices    int64
		rawAlarmStats    *alarm.AlarmStatistics
		rawKPIValues     []model.KPIValue
		rawAlarms        []model.Alarm
		alarmDeviceCount int64
	)

	g, gctx := errgroup.WithContext(ctx)

	// 1. Device total / online counts —— 与设备状态柱图同源（T-0162：在线语义 = is_online=TRUE，
	// 与 lifecycle 解耦）。原走 deviceService.CountByStatus + DeriveStatusFromLifecycle 派生
	// DeviceActive 桶，会把 Maintenance/Discovered/... 但 is_online=TRUE 的设备从"在线"中漏掉，
	// 导致 KPI 卡与同页柱图数字漂移。
	g.Go(func() error {
		if err := s.pgPool.QueryRow(gctx, summaryDeviceCountsQuery).Scan(&totalDevices, &onlineDevices); err != nil {
			s.logger.Warn("dashboard: device counts query failed", zap.Error(err))
			totalDevices = 0
			onlineDevices = 0
		}
		return nil
	})

	// 2. Alarm statistics
	g.Go(func() error {
		stats, err := s.alarmStore.Statistics(gctx, alarm.AlarmFilter{})
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

	// 3. Latest KPI value per metric_path (last 24h)
	//
	// 历史实现走 kpiRepo.Query(PageSize=10, SortBy=time desc)，是"top-10 行"上限不是"每指标取最新"，
	// 当全网指标 >> 10 时 UE_ACTIVE 等会被截断 → 首页活跃 UE 卡常驻 0（issue HD01 根因）。
	// 改走 DISTINCT ON (metric_path) ORDER BY metric_path, time DESC，确保每个指标编号都拿到最新值。
	g.Go(func() error {
		now := time.Now()
		query, args, err := buildLatestKPIPerNameQuery(now.Add(-24*time.Hour), now)
		if err != nil {
			s.logger.Warn("dashboard: build latest kpi query failed", zap.Error(err))
			return nil
		}
		rows, err := s.tsPool.Query(gctx, query, args...)
		if err != nil {
			s.logger.Warn("dashboard: latest kpi query failed", zap.Error(err))
			return nil
		}
		defer rows.Close()
		for rows.Next() {
			var path string
			var val float64
			if err := rows.Scan(&path, &val); err != nil {
				s.logger.Warn("dashboard: scan latest kpi row failed", zap.Error(err))
				return nil
			}
			// 仅填 KPIName/IndicatorID/KPIValue —— 下游只读这三项映射进 KPIOverview。
			rawKPIValues = append(rawKPIValues, model.KPIValue{
				KPIName:     path,
				IndicatorID: path,
				KPIValue:    val,
			})
		}
		if err := rows.Err(); err != nil {
			s.logger.Warn("dashboard: iterate latest kpi rows failed", zap.Error(err))
		}
		return nil
	})

	// 4. Recent top 5 alarms
	g.Go(func() error {
		filter := alarm.AlarmFilter{}
		filter.Page = 1
		filter.PageSize = 5
		filter.SortBy = "raised_at"
		filter.SortDir = "desc"
		result, err := s.alarmStore.ListActive(gctx, filter)
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
		if err := s.pgPool.QueryRow(gctx, query, args...).Scan(&alarmDeviceCount); err != nil {
			s.logger.Warn("dashboard: alarm device count failed", zap.Error(err))
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	summary.DeviceStats = FrontendDeviceStats{
		Total:   totalDevices,
		Online:  onlineDevices,
		Offline: totalDevices - onlineDevices,
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
				DeviceSN:   a.DeviceSN,                 // 完整设备 SN
				Technology: derefOrEmpty(a.Technology), // 技术类型
				DeviceName: a.DeviceSN,                 // 与 device_sn 相同，使用完整 SN
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
	summary.KPIDeltas = s.calculateKPIDeltas(ctx, totalDevices, summary.AlarmStats.Total)

	return summary, nil
}

func severityToLabel(s model.AlarmSeverity) string {
	switch canonicalDashboardSeverity(s) {
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
	return int(canonicalDashboardSeverity(s))
}

func canonicalDashboardSeverity(s model.AlarmSeverity) model.AlarmSeverity {
	switch int(s) {
	case 31001:
		return model.AlarmCritical
	case 31002:
		return model.AlarmMajor
	case 31003:
		return model.AlarmMinor
	case 31004:
		return model.AlarmWarning
	default:
		return s
	}
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

// summaryDeviceCountsQuery 给 dashboard /summary 接口的 KPI 卡用：取设备总数 +
// 在线数（is_online=TRUE）。"在线"语义与生命周期解耦（T-0162），与 device 模块
// DeviceListStats.OnlineCount、设备状态柱图 GetDeviceStatusByType 同源。
const summaryDeviceCountsQuery = `
	SELECT
		COUNT(*) AS total,
		COUNT(*) FILTER (WHERE is_online = TRUE) AS online
	FROM devices
	WHERE deleted_at IS NULL
`

// countDevicesAtTimeQuery 重建时刻 t 的设备总数：created_at 在 t 之前，且
// 截至 t 尚未软删除。devices 是主库分区表（按 carrier 分区），父表查询即可贯穿所有分区。
const countDevicesAtTimeQuery = `
	SELECT COUNT(*)
	FROM devices
	WHERE created_at <= $1
	  AND (deleted_at IS NULL OR deleted_at > $1)
`

// countAlarmsAtTimeQuery 重建时刻 t 的活跃告警数：raised_at 在 t 之前，且
// 截至 t 未被清除。alarms_history 在时序库（tsPool）上，是 7d chunk 的超表。
const countAlarmsAtTimeQuery = `
	SELECT COUNT(*)
	FROM alarms_history
	WHERE raised_at <= $1
	  AND (cleared_at IS NULL OR cleared_at > $1)
`

// countDevicesAtTime 返回时刻 t 在网设备总数（含历史已下线但当时尚在网的）。
// 用于 dashboard KPI 卡片的同环比对比。
func (s *Service) countDevicesAtTime(ctx context.Context, t time.Time) (int64, error) {
	if s.pgPool == nil {
		return 0, fmt.Errorf("countDevicesAtTime: pgPool not configured")
	}
	var n int64
	if err := s.pgPool.QueryRow(ctx, countDevicesAtTimeQuery, t).Scan(&n); err != nil {
		return 0, fmt.Errorf("countDevicesAtTime: %w", err)
	}
	return n, nil
}

// countAlarmsAtTime 返回时刻 t 的活跃告警数（已 raise 未 clear）。
// 用于 dashboard KPI 卡片的同环比对比。
func (s *Service) countAlarmsAtTime(ctx context.Context, t time.Time) (int64, error) {
	if s.tsPool == nil {
		return 0, fmt.Errorf("countAlarmsAtTime: tsPool not configured")
	}
	var n int64
	if err := s.tsPool.QueryRow(ctx, countAlarmsAtTimeQuery, t).Scan(&n); err != nil {
		return 0, fmt.Errorf("countAlarmsAtTime: %w", err)
	}
	return n, nil
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

// alarmTrendByDateQuery 按天分级统计告警数。调用侧只用固定表名格式化，不接收用户输入。
// 库内 severity 列可能是 5 位字典码（31001~31004），也可能是历史 1~4 小编号，故每个
// 级别桶同时匹配两种值；只认 1~4 会让四条曲线全读 0（issue #219 同根残留）。
// 复杂聚合（DATE()/CASE WHEN/COALESCE），裸 SQL 比 Squirrel 更易读。
const alarmTrendByDateQuery = `
		SELECT
			DATE(raised_at) AS d,
			COALESCE(SUM(CASE WHEN severity IN (1, 31001) THEN 1 ELSE 0 END), 0) AS critical,
			COALESCE(SUM(CASE WHEN severity IN (2, 31002) THEN 1 ELSE 0 END), 0) AS major,
			COALESCE(SUM(CASE WHEN severity IN (3, 31003) THEN 1 ELSE 0 END), 0) AS minor,
			COALESCE(SUM(CASE WHEN severity IN (4, 31004) THEN 1 ELSE 0 END), 0) AS warning
		FROM %s
		WHERE raised_at >= NOW() - $1::interval
		GROUP BY DATE(raised_at)
		ORDER BY d ASC`

// GetAlarmTrend returns alarm counts grouped by date and severity for the last N days.
func (s *Service) GetAlarmTrend(ctx context.Context, days int) ([]AlarmTrendEntry, error) {
	if days < 1 {
		days = 7
	}
	if days > 365 {
		days = 365
	}

	interval := fmt.Sprintf("%d days", days)
	entriesByDate := make(map[string]*AlarmTrendEntry)

	if err := s.queryAlarmTrendInto(ctx, s.pgPool, "alarms_active", interval, entriesByDate); err != nil {
		return nil, err
	}
	if s.tsPool != nil {
		if err := s.queryAlarmTrendInto(ctx, s.tsPool, "alarms_history", interval, entriesByDate); err != nil {
			return nil, err
		}
	}

	entries := make([]AlarmTrendEntry, 0, len(entriesByDate))
	for _, entry := range entriesByDate {
		entries = append(entries, *entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Date < entries[j].Date })
	return entries, nil
}

func (s *Service) queryAlarmTrendInto(ctx context.Context, pool *pgxpool.Pool, tableName string, interval string, entriesByDate map[string]*AlarmTrendEntry) error {
	if pool == nil {
		return nil
	}
	query := fmt.Sprintf(alarmTrendByDateQuery, tableName)
	rows, err := pool.Query(ctx, query, interval)
	if err != nil {
		return fmt.Errorf("query alarm trend from %s: %w", tableName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var entry AlarmTrendEntry
		var d time.Time
		if err := rows.Scan(&d, &entry.Critical, &entry.Major, &entry.Minor, &entry.Warning); err != nil {
			return fmt.Errorf("scan alarm trend row from %s: %w", tableName, err)
		}
		entry.Date = d.Format("2006-01-02")
		merged, exists := entriesByDate[entry.Date]
		if !exists {
			entriesByDate[entry.Date] = &entry
			continue
		}
		merged.Critical += entry.Critical
		merged.Major += entry.Major
		merged.Minor += entry.Minor
		merged.Warning += entry.Warning
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate alarm trend rows from %s: %w", tableName, err)
	}
	return nil
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
		compareStart = now.AddDate(0, 0, -weekday-6).Truncate(24 * time.Hour)      // Last Monday
		compareEnd = compareStart.AddDate(0, 0, 6).Add(24*time.Hour - time.Second) // Last Sunday
	default:
		// Default to yesterday
		compareWith = "yesterday"
		compareStart = now.AddDate(0, 0, -1).Truncate(24 * time.Hour)
		compareEnd = now.Truncate(24 * time.Hour).Add(-time.Second)
	}

	// Query current period data (today 00:00 to now)
	currentStart := now.Truncate(24 * time.Hour)
	currentEntries, err := s.queryNetworkKPISeries(ctx, kpiName, currentStart, now)
	if err != nil {
		return nil, fmt.Errorf("query current kpi trend: %w", err)
	}

	// Query comparison period data
	compareEntries, err := s.queryNetworkKPISeries(ctx, kpiName, compareStart, compareEnd)
	if err != nil {
		return nil, fmt.Errorf("query compare kpi trend: %w", err)
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
		Current: currentEntries,
		Compare: compareEntries,
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
//
// 取数源（阶段2 改造）：从「读原始每设备每小区明细（pm_metrics）」改为「读全网预聚合结果表
// pm_adhoc_aggregation_results（network 维度）」——3 条内置全网任务每小时把全库指标
// （counter 求和、KPI 重算）汇成全网总线，首页直接拿口径正确的全网线。详见 kpi_network_query.go。
//
// 查询键：前端传指标编号（K/C 编号），直接查 metric_path。
func (s *Service) GetKPITimeSeries(ctx context.Context, kpiNames []string, startTime, endTime time.Time) (KPITimeSeriesResponse, error) {
	result := make(KPITimeSeriesResponse, len(kpiNames))

	if len(kpiNames) == 0 {
		return result, nil
	}

	// Initialize empty slices for all requested names（含 none 项 → 始终返回空序列而非缺键）。
	for _, name := range kpiNames {
		result[name] = []KPITimeSeriesEntry{}
	}

	// 去重：前端直接传指标编号（K/C 编号），去重后作为查询 kcodes。
	seen := make(map[string]struct{}, len(kpiNames))
	var kcodes []string
	for _, k := range kpiNames {
		if _, dup := seen[k]; !dup {
			seen[k] = struct{}{}
			kcodes = append(kcodes, k)
		}
	}

	// 取全网时序：优先读每小时预聚合表，缺数据时回退 15min 直读原始明细（见 fetchNetworkKCodeSeries）。
	points, err := s.fetchNetworkKCodeSeries(ctx, kcodes, startTime, endTime)
	if err != nil {
		return nil, err
	}

	for _, p := range points {
		entry := KPITimeSeriesEntry{
			Time:  p.time.Format(time.RFC3339),
			Value: p.value,
		}
		// 按指标编号直接回填。
		result[p.code] = append(result[p.code], entry)
	}

	return result, nil
}

// networkSeriesPoint 是全网时序的一行（指标编号 + 时间桶 + 值），供 GetKPITimeSeries 回填用。
type networkSeriesPoint struct {
	code  string
	time  time.Time
	value float64
}

// fetchNetworkKCodeSeries 读多个指标编号的全网时序，三段式策略（issue #359 + 尾部补点）：
//
//  1. 优先读每小时预聚合表 pm_adhoc_aggregation_results（口径正确的全网线，10 万级规模成本低）。
//  2. 预聚合表对该时窗完全无任何行时（刚灌数未到整点 / continuous scheduler 未起），
//     整体回退 15min 直读原始明细 pm_metrics 现场汇成全网线，与性能仪表板默认模板对齐容错。
//  3. 预聚合有数据，但存在「尾部缺口」时，用 15min 原始明细补充缺失数据点
//     （与 PM 性能仪表盘 15min 实时口径对齐）：
//     - 当前不完整小时：17:30 时 17:00 桶未聚合，补出 17:00/17:15 打点。
//     - 延迟的完整小时：聚合任务落后多个小时时，15min 补点同样覆盖（从缺口开始填）。
//     - 跨制式混合指标：LTE/NR/GSM 聚合任务各自独立运行，不同制式指标可能有不同的最新桶
//       时间；以所有 code 各自「下一桶起点」的最小值作为尾部查询起点，并在合并时按 code
//       过滤掉与现有小时数据重叠的 15min 点，避免同一 code 同一时段双重计数。
//
// 两条链路都查时序库（tsPool）。策略 2 是整体缺数据时触发；策略 3 是常态下的尾部实时补充。
func (s *Service) fetchNetworkKCodeSeries(ctx context.Context, kcodes []string, startTime, endTime time.Time) ([]networkSeriesPoint, error) {
	query, args, err := buildNetworkKPISeriesQuery(kcodes, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("build kpi time series query: %w", err)
	}
	hourlyPoints, err := s.scanNetworkSeries(ctx, query, args, "cagg_realtime")
	if err != nil {
		return nil, err
	}
	return hourlyPoints, nil
}

// scanNetworkSeries 跑一条「metric_path, time, metric_value」三列查询并扫成 networkSeriesPoint 列表。
func (s *Service) scanNetworkSeries(ctx context.Context, query string, args []any, label string) ([]networkSeriesPoint, error) {
	rows, err := s.tsPool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query kpi time series (%s): %w", label, err)
	}
	defer rows.Close()

	var points []networkSeriesPoint
	for rows.Next() {
		var p networkSeriesPoint
		if err := rows.Scan(&p.code, &p.time, &p.value); err != nil {
			return nil, fmt.Errorf("scan kpi time series row (%s): %w", label, err)
		}
		points = append(points, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate kpi time series rows (%s): %w", label, err)
	}
	return points, nil
}

// queryNetworkKPISeries 读单个指标在某时窗内的全网预聚合时序（供 GetKPITrendComparison 用）。
//
// kpiName 为指标编号（K/C 编号），读 pm_adhoc_aggregation_results（network 维度），与 GetKPITimeSeries 同源。
func (s *Service) queryNetworkKPISeries(ctx context.Context, kpiName string, startTime, endTime time.Time) ([]KPITrendEntry, error) {
	kcodes := []string{kpiName}

	// 与 GetKPITimeSeries 同源：优先读每小时预聚合表，缺数据时回退 15min 直读原始明细（issue #359）。
	points, err := s.fetchNetworkKCodeSeries(ctx, kcodes, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("query network kpi series: %w", err)
	}

	entries := make([]KPITrendEntry, 0, len(points))
	for _, p := range points {
		entries = append(entries, KPITrendEntry{
			Time:  p.time.Format(time.RFC3339),
			Value: p.value,
		})
	}
	return entries, nil
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
