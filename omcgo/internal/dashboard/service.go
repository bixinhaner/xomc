package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/alarm"
	"github.com/omcgo/omcgo/internal/authz"
	"github.com/omcgo/omcgo/internal/core/jsonx"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/pm/indicator"
	"github.com/omcgo/omcgo/internal/pm/kpi"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	pmstream "github.com/omcgo/omcgo/internal/pm/stream"
	"github.com/omcgo/omcgo/internal/topology"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	activeUEKPIAlias = "UE_ACTIVE"
	activeUEKPIID    = "KGNB0568"
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
	HasComparison bool    `json:"has_comparison"` // 是否存在有效的非零历史基线
}

// DashboardSummary is the aggregated dashboard response.
type DashboardSummary struct {
	DeviceStats  FrontendDeviceStats   `json:"device_stats"`
	AlarmStats   FrontendAlarmStats    `json:"alarm_stats"`
	KPIOverview  map[string]float64    `json:"kpi_overview"`
	KPIDeltas    map[string]KPIDelta   `json:"kpi_deltas"` // KPI趋势数据（新增）
	RecentAlarms []FrontendRecentAlarm `json:"recent_alarms"`
	PMSlotHealth []PMSlotHealthSummary `json:"pm_slot_health"`
	Timestamp    time.Time             `json:"timestamp"`
}

type PMSlotHealthSummary struct {
	SlotEnd         time.Time `json:"slot_end"`
	Technology      string    `json:"technology"`
	Carrier         string    `json:"carrier"`
	ExpectedDevices int64     `json:"expected_devices"`
	ReceivedDevices int64     `json:"received_devices"`
	CoverageRatio   float64   `json:"coverage_ratio"`
	Status          string    `json:"status"`
	EvaluatedAt     time.Time `json:"evaluated_at"`
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
	Time    time.Time   `json:"time"`
	Value   jsonx.Float `json:"value"`
	Partial bool        `json:"partial,omitempty"`
}

// KPITimeSeriesResponse maps KPI names to their time-series data.
type KPITimeSeriesResponse map[string][]KPITimeSeriesEntry

type KPITimeSeriesSnapshot struct {
	Series         KPITimeSeriesResponse     `json:"series"`
	PeriodProgress []pmstream.PeriodProgress `json:"period_progress"`
	ProgressState  string                    `json:"progress_state"`
}

type NetworkProgressReader interface {
	Query(
		context.Context,
		uuid.UUID,
		time.Time,
		time.Time,
	) (pmstream.ProgressQueryResult, error)
}

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
	// enabledIndicatorRepo 读取 enabled_pm_indicators_*，用于保存首页 KPI 布局时拒绝未启用指标。
	// 生产由 NewService 基于主库 PgPool 注入；nil 时跳过校验（测试/退化场景）。
	enabledIndicatorRepo indicator.EnabledIndicatorRepository
	// networkRollups 只读现有内置任务发布的 network 维度全网结果。
	// Dashboard 不得回退原始 PM 明细或调用在线聚合。
	networkRollups  NetworkRollupReader
	networkProgress NetworkProgressReader
	kpiQueryGuard   *KPIQueryGuard
	metrics         *Metrics
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
	networkRollups NetworkRollupReader,
	logger *zap.Logger,
) *Service {
	s := &Service{
		deviceService:  deviceService,
		alarmStore:     alarmStore,
		kpiRepo:        kpiRepo,
		pgPool:         pgPool,
		tsPool:         tsPool,
		groupRepo:      groupRepo,
		indicatorRepo:  indicatorRepo,
		networkRollups: networkRollups,
		logger:         logger.Named("dashboard"),
	}
	// 全局 KPI 布局仓库走主库（dashboard_kpi_layouts 在主库）。pgPool 为 nil 时（测试）留空。
	if pgPool != nil {
		s.layoutRepo = NewKPILayoutRepository(pgPool)
		s.enabledIndicatorRepo = indicator.NewPgEnabledRepository(pgPool)
	}
	return s
}

func (s *Service) SetKPIQueryGuard(guard *KPIQueryGuard) {
	s.kpiQueryGuard = guard
}

func (s *Service) SetMetrics(metrics *Metrics) {
	s.metrics = metrics
}

func (s *Service) SetNetworkProgressReader(reader NetworkProgressReader) {
	s.networkProgress = reader
}

func latestPMSlotHealthQuery() (string, []any, error) {
	return storage.Psql.Select(
		"slot_end",
		"technology", "carrier", "expected_devices", "received_devices",
		"coverage_ratio", "status", "evaluated_at",
	).
		From("pm_slot_health").
		Where("slot_end = (SELECT MAX(latest.slot_end) FROM pm_slot_health latest)").
		OrderBy("technology", "carrier").
		ToSql()
}

func (s *Service) listLatestPMSlotHealth(ctx context.Context) ([]PMSlotHealthSummary, error) {
	if s.tsPool == nil {
		return nil, nil
	}
	query, args, err := latestPMSlotHealthQuery()
	if err != nil {
		return nil, fmt.Errorf("build latest PM slot health query: %w", err)
	}
	rows, err := s.tsPool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query latest PM slot health: %w", err)
	}
	defer rows.Close()
	var result []PMSlotHealthSummary
	for rows.Next() {
		var row PMSlotHealthSummary
		if err := rows.Scan(
			&row.SlotEnd, &row.Technology, &row.Carrier,
			&row.ExpectedDevices, &row.ReceivedDevices, &row.CoverageRatio,
			&row.Status, &row.EvaluatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan latest PM slot health: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate latest PM slot health: %w", err)
	}
	return result, nil
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
		rawKPIValues     []NetworkRollupPoint
		rawAlarms        []model.Alarm
		alarmDeviceCount int64
		rawPMSlotHealth  []PMSlotHealthSummary
	)

	g, gctx := errgroup.WithContext(ctx)

	// 1. Device total / online counts —— 与设备状态柱图同源（T-0162：在线语义 = is_online=TRUE，
	// 与 lifecycle 解耦）。原走 deviceService.CountByStatus + DeriveStatusFromLifecycle 派生
	// DeviceActive 桶，会把 Maintenance/Discovered/... 但 is_online=TRUE 的设备从"在线"中漏掉，
	// 导致 KPI 卡与同页柱图数字漂移。
	g.Go(func() error {
		if err := s.pgPool.QueryRow(gctx, summaryDeviceCountsQuery).Scan(&totalDevices, &onlineDevices); err != nil {
			logDashboardQueryFailure(s.logger, "dashboard: device counts query failed", err)
			totalDevices = 0
			onlineDevices = 0
		}
		return nil
	})

	// 6. Latest persisted PM slot health. This reads only the bounded summary
	// table maintained by the worker; the homepage must never aggregate raw PM.
	g.Go(func() error {
		rows, err := s.listLatestPMSlotHealth(gctx)
		if err != nil {
			logDashboardQueryFailure(s.logger, "dashboard: PM slot health query failed", err)
			return nil
		}
		rawPMSlotHealth = rows
		return nil
	})

	// 2. Alarm statistics
	g.Go(func() error {
		stats, err := s.alarmStore.Statistics(gctx, alarm.AlarmFilter{})
		if err != nil {
			logDashboardQueryFailure(s.logger, "dashboard: alarm stats failed", err)
			rawAlarmStats = &alarm.AlarmStatistics{
				BySeverity: make(map[model.AlarmSeverity]int64),
				ByType:     make(map[string]int64),
			}
			return nil
		}
		rawAlarmStats = stats
		return nil
	})

	// 3. Latest published network KPI value per technology + metric (last 24h).
	// This path must never scan raw PM tables or invoke online aggregation.
	g.Go(func() error {
		if s.networkRollups == nil {
			s.logger.Warn("dashboard: network rollup reader not configured")
			return nil
		}
		now := time.Now()
		load := func(loadCtx context.Context) (any, error) {
			return s.networkRollups.ListLatestHourly(loadCtx, now.Add(-24*time.Hour), now)
		}
		var loaded any
		var err error
		if s.kpiQueryGuard != nil {
			loaded, _, err = s.kpiQueryGuard.Do(gctx, "summary:latest-hourly", load)
		} else {
			loaded, err = load(gctx)
		}
		if err != nil {
			logDashboardQueryFailure(s.logger, "dashboard: latest network rollup query failed", err)
			return nil
		}
		points, ok := loaded.([]NetworkRollupPoint)
		if !ok {
			s.logger.Warn("dashboard: latest network rollup cache type mismatch")
			return nil
		}
		rawKPIValues = append([]NetworkRollupPoint(nil), points...)
		s.observeNetworkRollups(points, nil, "", metrics.GranularityHourly, now)
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
			logDashboardQueryFailure(s.logger, "dashboard: recent alarms failed", err)
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
			logDashboardQueryFailure(s.logger, "dashboard: alarm device count failed", err)
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
	summary.PMSlotHealth = rawPMSlotHealth

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

	overview, conflicts := buildKPIOverviewFromNetworkRollups(rawKPIValues)
	summary.KPIOverview = overview
	if len(conflicts) > 0 {
		s.logger.Warn("dashboard: cross-technology KPI paths skipped",
			zap.Strings("metric_paths", conflicts))
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
	currentActiveUE, hasCurrentActiveUE := summary.KPIOverview[activeUEKPIAlias]
	summary.KPIDeltas = s.calculateKPIDeltas(
		ctx,
		totalDevices,
		summary.AlarmStats.Total,
		currentActiveUE,
		hasCurrentActiveUE,
	)

	return summary, nil
}

func buildKPIOverviewFromNetworkRollups(points []NetworkRollupPoint) (map[string]float64, []string) {
	type selectedPoint struct {
		technology  model.Technology
		windowStart time.Time
		value       float64
	}
	selected := make(map[string]selectedPoint, len(points))
	conflictSet := make(map[string]struct{})
	for _, point := range points {
		path := strings.TrimSpace(point.MetricPath)
		if path == "" {
			continue
		}
		if _, conflicted := conflictSet[path]; conflicted {
			continue
		}
		current, exists := selected[path]
		if exists && current.technology != point.Technology {
			delete(selected, path)
			conflictSet[path] = struct{}{}
			continue
		}
		if !exists || point.WindowStart.After(current.windowStart) {
			selected[path] = selectedPoint{
				technology:  point.Technology,
				windowStart: point.WindowStart,
				value:       float64(point.Value),
			}
		}
	}
	overview := make(map[string]float64, len(selected))
	for path, point := range selected {
		setDashboardKPIOverviewValue(overview, path, point.value)
	}
	conflicts := make([]string, 0, len(conflictSet))
	for path := range conflictSet {
		conflicts = append(conflicts, path)
	}
	sort.Strings(conflicts)
	return overview, conflicts
}

func setDashboardKPIOverviewValue(overview map[string]float64, key string, value float64) {
	if _, exists := overview[key]; !exists {
		overview[key] = value
	}
	// PM 规范化入库使用指标编号作为 metric_path；Dashboard 对外继续提供稳定的展示别名。
	if key == activeUEKPIID {
		if _, exists := overview[activeUEKPIAlias]; !exists {
			overview[activeUEKPIAlias] = value
		}
	}
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
func (s *Service) calculateKPIDeltas(
	ctx context.Context,
	currentTotalDevices int64,
	currentTotalAlarms int64,
	currentActiveUE float64,
	hasCurrentActiveUE bool,
) map[string]KPIDelta {
	now := time.Now()
	windows := dashboardKPIDeltaWindows(now)

	deltas := make(map[string]KPIDelta)

	// 1. Total devices trend (compare with last week same time)
	prevTotalDevices, err := s.countDevicesAtTime(ctx, windows.DeviceCompareAt)
	if err != nil {
		logDashboardQueryFailure(s.logger, "dashboard: previous device count unavailable", err)
	}
	deltas["total_devices"] = computeKPIDelta(
		float64(currentTotalDevices),
		float64(prevTotalDevices),
		err == nil,
		"last_week",
	)

	// 2. Active alarms trend (compare with yesterday)
	prevTotalAlarms, err := s.countAlarmsAtTime(ctx, windows.AlarmCompareAt)
	if err != nil {
		logDashboardQueryFailure(s.logger, "dashboard: previous alarm count unavailable", err)
	}
	deltas["active_alarms"] = computeKPIDelta(
		float64(currentTotalAlarms),
		float64(prevTotalAlarms),
		err == nil,
		"yesterday",
	)

	// 3. Active UE trend. Current value comes from the latest KPI overview sample;
	// the baseline reuses the dashboard network KPI time-series data chain.
	deltas[activeUEKPIAlias] = computeKPIDelta(currentActiveUE, 0, false, "last_week")
	if hasCurrentActiveUE {
		currentEntries, currentErr := s.queryNetworkKPISeries(
			ctx,
			activeUEKPIID,
			windows.UECurrentStart,
			windows.UECurrentEnd,
		)
		if currentErr != nil {
			logDashboardQueryFailure(s.logger, "dashboard: current active UE comparison unavailable", currentErr)
		}
		previousEntries, previousErr := s.queryNetworkKPISeries(
			ctx,
			activeUEKPIID,
			windows.UEPreviousStart,
			windows.UEPreviousEnd,
		)
		if previousErr != nil {
			logDashboardQueryFailure(s.logger, "dashboard: previous active UE comparison unavailable", previousErr)
		}
		if currentErr == nil && previousErr == nil {
			deltas[activeUEKPIAlias] = computeSeriesKPIDelta(currentEntries, previousEntries, "last_week")
		}
	}

	return deltas
}

type kpiDeltaWindows struct {
	DeviceCompareAt time.Time
	AlarmCompareAt  time.Time
	UECurrentStart  time.Time
	UECurrentEnd    time.Time
	UEPreviousStart time.Time
	UEPreviousEnd   time.Time
}

func dashboardKPIDeltaWindows(now time.Time) kpiDeltaWindows {
	currentStart := dashboardStartOfDay(now)
	return kpiDeltaWindows{
		DeviceCompareAt: now.AddDate(0, 0, -7),
		AlarmCompareAt:  now.AddDate(0, 0, -1),
		UECurrentStart:  currentStart,
		UECurrentEnd:    now,
		UEPreviousStart: currentStart.AddDate(0, 0, -7),
		UEPreviousEnd:   now.AddDate(0, 0, -7),
	}
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

// 当前仍未清除的告警只存在主库 alarms_active；只要在 t 前发生，它在 t 时刻就是活跃告警。
const listActiveAlarmIDsAtTimeQuery = `
	SELECT id
	FROM alarms_active
	WHERE raised_at <= $1
`

// 已清除告警会从 alarms_active 搬到时序库 alarms_history。只有在 t 之后才清除的记录，
// 在 t 时刻仍属于活跃告警。两表由告警清除链路迁移，稳定状态下互斥。
const countHistoricalAlarmsAtTimeQuery = `
	SELECT COUNT(DISTINCT alarm_id)
	FROM alarms_history
	WHERE raised_at <= $1
	  AND cleared_at > $1
	  AND NOT (alarm_id = ANY($2::uuid[]))
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
	if s.pgPool == nil {
		return 0, fmt.Errorf("countAlarmsAtTime: pgPool not configured")
	}
	if s.tsPool == nil {
		return 0, fmt.Errorf("countAlarmsAtTime: tsPool not configured")
	}
	rows, err := s.pgPool.Query(ctx, listActiveAlarmIDsAtTimeQuery, t)
	if err != nil {
		return 0, fmt.Errorf("countAlarmsAtTime active: %w", err)
	}
	defer rows.Close()
	activeIDs := make([]uuid.UUID, 0)
	seenActiveIDs := make(map[uuid.UUID]struct{})
	for rows.Next() {
		var alarmID uuid.UUID
		if err := rows.Scan(&alarmID); err != nil {
			return 0, fmt.Errorf("countAlarmsAtTime scan active: %w", err)
		}
		if _, exists := seenActiveIDs[alarmID]; exists {
			continue
		}
		seenActiveIDs[alarmID] = struct{}{}
		activeIDs = append(activeIDs, alarmID)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("countAlarmsAtTime iterate active: %w", err)
	}
	var historicalCount int64
	if err := s.tsPool.QueryRow(ctx, countHistoricalAlarmsAtTimeQuery, t, activeIDs).Scan(&historicalCount); err != nil {
		return 0, fmt.Errorf("countAlarmsAtTime history: %w", err)
	}
	return int64(len(activeIDs)) + historicalCount, nil
}

// computeKPIDelta calculates delta values for a single KPI metric.
func computeKPIDelta(current, previous float64, hasComparison bool, compareType string) KPIDelta {
	delta := KPIDelta{
		CurrentValue:  current,
		PreviousValue: previous,
		CompareType:   compareType,
		Trend:         "stable",
	}

	// 百分比变化必须有真实且非零的历史基线。缺样本、查询失败或历史值为 0
	// 都不能被解释成“增长 100%”。
	if !hasComparison || previous == 0 {
		return delta
	}
	delta.HasComparison = true

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

func averageKPITrendEntries(entries []KPITrendEntry) (float64, bool) {
	var sum float64
	count := 0
	for _, entry := range entries {
		if math.IsNaN(entry.Value) || math.IsInf(entry.Value, 0) {
			continue
		}
		sum += entry.Value
		count++
	}
	if count == 0 {
		return 0, false
	}
	return sum / float64(count), true
}

func computeSeriesKPIDelta(currentEntries, previousEntries []KPITrendEntry, compareType string) KPIDelta {
	current, hasCurrent := averageKPITrendEntries(currentEntries)
	previous, hasPrevious := averageKPITrendEntries(previousEntries)
	return computeKPIDelta(current, previous, hasCurrent && hasPrevious, compareType)
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

type alarmTrendSnapshot struct {
	Date string
	At   time.Time
}

type alarmTrendSnapshotCount struct {
	Ordinal  int
	Critical int64
	Major    int64
	Minor    int64
	Warning  int64
	AlarmIDs []uuid.UUID
}

func buildAlarmTrendSnapshots(now time.Time, days int) []alarmTrendSnapshot {
	if days < 1 {
		return nil
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	snapshots := make([]alarmTrendSnapshot, 0, days)
	for offset := days - 1; offset >= 0; offset-- {
		dayStart := today.AddDate(0, 0, -offset)
		snapshotAt := dayStart.AddDate(0, 0, 1).Add(-time.Nanosecond)
		if offset == 0 {
			snapshotAt = now
		}
		snapshots = append(snapshots, alarmTrendSnapshot{
			Date: dayStart.Format("2006-01-02"),
			At:   snapshotAt,
		})
	}
	return snapshots
}

func alarmTrendSnapshotSelect(idColumn string, includeAlarmIDs bool) sq.SelectBuilder {
	columns := []string{
		"snapshots.ordinality",
		fmt.Sprintf("COUNT(DISTINCT %s) FILTER (WHERE alarm.severity IN (1, 31001)) AS critical", idColumn),
		fmt.Sprintf("COUNT(DISTINCT %s) FILTER (WHERE alarm.severity IN (2, 31002)) AS major", idColumn),
		fmt.Sprintf("COUNT(DISTINCT %s) FILTER (WHERE alarm.severity IN (3, 31003)) AS minor", idColumn),
		fmt.Sprintf("COUNT(DISTINCT %s) FILTER (WHERE alarm.severity IN (4, 31004)) AS warning", idColumn),
	}
	if includeAlarmIDs {
		columns = append(columns, fmt.Sprintf(
			"COALESCE(array_agg(DISTINCT %s) FILTER (WHERE %s IS NOT NULL AND snapshots.ordinality = cardinality($1::timestamptz[])), '{}'::uuid[]) AS alarm_ids",
			idColumn,
			idColumn,
		))
	} else {
		columns = append(columns, "'{}'::uuid[] AS alarm_ids")
	}
	return sq.Select(columns...).
		From("unnest(?::timestamptz[]) WITH ORDINALITY AS snapshots(snapshot_at, ordinality)").
		GroupBy("snapshots.ordinality").
		OrderBy("snapshots.ordinality")
}

func buildActiveAlarmTrendSnapshotQuery(snapshotTimes []time.Time, visibleGroups []uuid.UUID) (string, []any, error) {
	builder := alarmTrendSnapshotSelect("alarm.id", true).
		LeftJoin("alarms_active alarm ON alarm.raised_at <= snapshots.snapshot_at")
	builder = authz.ApplyDeviceVisibilityFilter(builder, "alarm.device_id", visibleGroups)
	query, queryArgs, err := builder.PlaceholderFormat(sq.Dollar).ToSql()
	return query, append([]any{snapshotTimes}, queryArgs...), err
}

func buildHistoryAlarmTrendSnapshotQuery(
	snapshotTimes []time.Time,
	visibleGroups []uuid.UUID,
	activeIDs []uuid.UUID,
) (string, []any, error) {
	builder := alarmTrendSnapshotSelect("alarm.alarm_id", false).
		LeftJoin(`alarms_history alarm
			ON alarm.raised_at <= snapshots.snapshot_at
			AND alarm.cleared_at > snapshots.snapshot_at`)
	builder = authz.ApplyDeviceVisibilityFilter(builder, "alarm.device_id", visibleGroups)
	builder = builder.Where(sq.Or{
		sq.Expr("alarm.alarm_id IS NULL"),
		sq.Expr("NOT (alarm.alarm_id = ANY(?::uuid[]))", activeIDs),
	})
	query, queryArgs, err := builder.PlaceholderFormat(sq.Dollar).ToSql()
	return query, append([]any{snapshotTimes}, queryArgs...), err
}

func mergeAlarmTrendSnapshotCounts(entries []AlarmTrendEntry, counts []alarmTrendSnapshotCount) error {
	for _, count := range counts {
		index := count.Ordinal - 1
		if index < 0 || index >= len(entries) {
			return fmt.Errorf("alarm trend snapshot ordinal %d out of range", count.Ordinal)
		}
		entries[index].Critical += count.Critical
		entries[index].Major += count.Major
		entries[index].Minor += count.Minor
		entries[index].Warning += count.Warning
	}
	return nil
}

func collectAlarmTrendSnapshotIDs(counts []alarmTrendSnapshotCount) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{})
	ids := make([]uuid.UUID, 0)
	for _, count := range counts {
		for _, alarmID := range count.AlarmIDs {
			if _, exists := seen[alarmID]; exists {
				continue
			}
			seen[alarmID] = struct{}{}
			ids = append(ids, alarmID)
		}
	}
	return ids
}

func queryAlarmTrendSnapshotCounts(
	ctx context.Context,
	pool *pgxpool.Pool,
	source string,
	query string,
	args []any,
	expectedCount int,
) ([]alarmTrendSnapshotCount, error) {
	if pool == nil || expectedCount == 0 {
		return nil, nil
	}

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query active alarm trend from %s: %w", source, err)
	}
	defer rows.Close()

	counts := make([]alarmTrendSnapshotCount, 0, expectedCount)
	for rows.Next() {
		var count alarmTrendSnapshotCount
		if err := rows.Scan(
			&count.Ordinal,
			&count.Critical,
			&count.Major,
			&count.Minor,
			&count.Warning,
			&count.AlarmIDs,
		); err != nil {
			return nil, fmt.Errorf("scan active alarm trend row from %s: %w", source, err)
		}
		counts = append(counts, count)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active alarm trend rows from %s: %w", source, err)
	}
	return counts, nil
}

// GetActiveAlarmTrend returns end-of-day active-alarm snapshots, using now for today's point.
func (s *Service) GetActiveAlarmTrend(ctx context.Context, days int, visibleGroups []uuid.UUID) ([]AlarmTrendEntry, error) {
	if days < 1 {
		days = 7
	}
	if days > 365 {
		days = 365
	}

	now := response.TimeInCurrentLocation(ctx, time.Now())
	snapshots := buildAlarmTrendSnapshots(now, days)
	entries := make([]AlarmTrendEntry, len(snapshots))
	snapshotTimes := make([]time.Time, len(snapshots))
	for index, snapshot := range snapshots {
		entries[index].Date = snapshot.Date
		snapshotTimes[index] = snapshot.At
	}

	activeQuery, activeArgs, err := buildActiveAlarmTrendSnapshotQuery(snapshotTimes, visibleGroups)
	if err != nil {
		return nil, fmt.Errorf("build active alarm trend query for alarms_active: %w", err)
	}
	activeCounts, err := queryAlarmTrendSnapshotCounts(
		ctx,
		s.pgPool,
		"alarms_active",
		activeQuery,
		activeArgs,
		len(snapshotTimes),
	)
	if err != nil {
		return nil, err
	}
	if err := mergeAlarmTrendSnapshotCounts(entries, activeCounts); err != nil {
		return nil, fmt.Errorf("merge active alarm trend from alarms_active: %w", err)
	}

	// Today's point deliberately comes only from alarms_active so it has the
	// same inventory source as the current-alarm cards. Historical rows are
	// only needed to reconstruct alarms that were still active on prior days.
	historySnapshotTimes := snapshotTimes[:len(snapshotTimes)-1]
	activeIDs := collectAlarmTrendSnapshotIDs(activeCounts)
	historyQuery, historyArgs, err := buildHistoryAlarmTrendSnapshotQuery(
		historySnapshotTimes,
		visibleGroups,
		activeIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("build active alarm trend query for alarms_history: %w", err)
	}
	historyCounts, err := queryAlarmTrendSnapshotCounts(
		ctx,
		s.tsPool,
		"alarms_history",
		historyQuery,
		historyArgs,
		len(historySnapshotTimes),
	)
	if err != nil {
		return nil, err
	}
	if err := mergeAlarmTrendSnapshotCounts(entries, historyCounts); err != nil {
		return nil, fmt.Errorf("merge active alarm trend from alarms_history: %w", err)
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

func dashboardStartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// dashboardKPITrendCompareWindow returns the comparison period boundaries used by GetKPITrendComparison.
func dashboardKPITrendCompareWindow(now time.Time, compareWith string) (string, time.Time, time.Time) {
	todayStart := dashboardStartOfDay(now)
	// compareEnd 使用展示时间边界：hourly 桶 [23:00,24:00) 在首页显示为次日 00:00，
	// 因此完整自然日的结束点必须是次日 00:00，而不是 23:59:59。
	switch compareWith {
	case "yesterday":
		return compareWith, todayStart.AddDate(0, 0, -1), todayStart
	case "last_week":
		weekday := int(todayStart.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday = 7
		}
		start := todayStart.AddDate(0, 0, -weekday-6) // Last Monday
		return compareWith, start, start.AddDate(0, 0, 7)
	default:
		return "yesterday", todayStart.AddDate(0, 0, -1), todayStart
	}
}

// GetKPITrendComparison returns KPI trend data with comparison (today vs yesterday/last week).
func (s *Service) GetKPITrendComparison(ctx context.Context, kpiName string, compareWith string) (*KPITrendComparison, error) {
	now := time.Now()
	compareWith, compareStart, compareEnd := dashboardKPITrendCompareWindow(now, compareWith)

	// Query current period data (today 00:00 to now)
	currentStart := dashboardStartOfDay(now)
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
		logDashboardQueryFailure(s.logger, "dashboard: get group tree failed", err)
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
			logDashboardQueryFailure(s.logger, "dashboard: list device IDs for group failed", err,
				zap.String("group", g.Name))
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
					logDashboardQueryFailure(s.logger, "dashboard: count online devices failed", err,
						zap.String("group", entry.Region))
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
					logDashboardQueryFailure(s.logger, "dashboard: count group alarms failed", err,
						zap.String("group", entry.Region))
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
// 取数源：只读现有内置任务发布的 network 维度全网预聚合结果。
// 小时、天、周严格按请求粒度读取；点位时间使用已发布窗口的桶起点。
//
// 查询键：前端传指标编号（K/C 编号），直接查 metric_path。
func (s *Service) GetKPITimeSeries(ctx context.Context, kpiNames []string, technology model.Technology, granularity metrics.Granularity, startTime, endTime time.Time) (KPITimeSeriesResponse, error) {
	result, _, err := s.GetKPITimeSeriesWithMetadata(ctx, kpiNames, technology, granularity, startTime, endTime)
	return result, err
}

func (s *Service) GetKPITimeSeriesWithMetadata(ctx context.Context, kpiNames []string, technology model.Technology, granularity metrics.Granularity, startTime, endTime time.Time) (KPITimeSeriesResponse, KPIQueryMetadata, error) {
	if s.kpiQueryGuard == nil || len(kpiNames) == 0 {
		result, err := s.getKPITimeSeriesUnprotected(ctx, kpiNames, technology, granularity, startTime, endTime)
		return result, KPIQueryMetadata{}, err
	}
	key := dashboardSeriesCacheKey(kpiNames, technology, granularity, startTime, endTime)
	value, metadata, err := s.kpiQueryGuard.Do(ctx, key, func(loadCtx context.Context) (any, error) {
		return s.getKPITimeSeriesUnprotected(loadCtx, kpiNames, technology, granularity, startTime, endTime)
	})
	if err != nil {
		return nil, KPIQueryMetadata{}, err
	}
	result, ok := value.(KPITimeSeriesResponse)
	if !ok {
		return nil, KPIQueryMetadata{}, fmt.Errorf("dashboard KPI series cache type mismatch")
	}
	return cloneKPITimeSeriesResponse(result), metadata, nil
}

func (s *Service) GetKPITimeSeriesSnapshotWithMetadata(
	ctx context.Context,
	kpiNames []string,
	technology model.Technology,
	granularity metrics.Granularity,
	startTime, endTime time.Time,
) (KPITimeSeriesSnapshot, KPIQueryMetadata, error) {
	load := func(loadCtx context.Context) (KPITimeSeriesSnapshot, error) {
		// The snapshot explicitly supplies the current open day/week from the
		// progress reader. Absence of a final published row for that natural
		// period is expected, so it must not feed the final-result-missing alarm.
		series, err := s.getKPITimeSeriesUnprotectedWithObservation(
			loadCtx, kpiNames, technology, granularity, startTime, endTime,
			false,
		)
		if err != nil {
			return KPITimeSeriesSnapshot{}, err
		}
		snapshot := KPITimeSeriesSnapshot{
			Series: series, PeriodProgress: []pmstream.PeriodProgress{},
			ProgressState: "not_applicable",
		}
		if granularity != metrics.GranularityDaily &&
			granularity != metrics.GranularityWeekly {
			return snapshot, nil
		}
		snapshot.ProgressState = "available"
		if s.networkProgress == nil {
			snapshot.ProgressState = "unavailable"
			return snapshot, nil
		}
		taskIDs := networkTaskIDs(technology)
		requested := make(map[string]struct{}, len(kpiNames))
		for _, name := range normalizeMetricPaths(kpiNames) {
			requested[name] = struct{}{}
		}
		metricIntervals := make([]pmstream.MetricVersionInterval, 0)
		for _, taskID := range taskIDs {
			progress, err := s.networkProgress.Query(loadCtx, taskID, startTime, endTime)
			if err != nil {
				snapshot.ProgressState = "unavailable"
				snapshot.PeriodProgress = []pmstream.PeriodProgress{}
				return snapshot, nil
			}
			for _, row := range progress.Rows {
				if row.Dimension != pmstream.DimensionNetwork ||
					string(row.Granularity) != string(granularity) ||
					row.MetricType != "kpi" ||
					row.WindowStart.Before(startTime) ||
					!row.WindowStart.Before(endTime) {
					continue
				}
				if _, ok := requested[row.MetricPath]; !ok {
					continue
				}
				snapshot.Series[row.MetricPath] = append(
					snapshot.Series[row.MetricPath],
					KPITimeSeriesEntry{
						Time: row.WindowStart, Value: jsonx.Float(row.Value), Partial: true,
					},
				)
			}
			for _, period := range progress.Periods {
				if string(period.Granularity) != string(granularity) ||
					!strings.EqualFold(period.EntityKey, "network") ||
					period.WindowStart.Before(startTime) ||
					!period.WindowStart.Before(endTime) {
					continue
				}
				snapshot.PeriodProgress = append(snapshot.PeriodProgress, period)
			}
			metricIntervals = append(metricIntervals, progress.MetricIntervals...)
		}
		sortKPITimeSeriesSnapshot(&snapshot)
		s.observeSnapshotMissing(
			loadCtx, snapshot, metricIntervals, kpiNames, technology,
			granularity, startTime, endTime,
		)
		return snapshot, nil
	}

	if s.kpiQueryGuard == nil || len(kpiNames) == 0 {
		snapshot, err := load(ctx)
		return snapshot, KPIQueryMetadata{}, err
	}
	key := "series-progress:" + dashboardSeriesCacheKey(
		kpiNames, technology, granularity, startTime, endTime,
	)
	value, metadata, err := s.kpiQueryGuard.Do(ctx, key, func(loadCtx context.Context) (any, error) {
		return load(loadCtx)
	})
	if err != nil {
		return KPITimeSeriesSnapshot{}, KPIQueryMetadata{}, err
	}
	snapshot, ok := value.(KPITimeSeriesSnapshot)
	if !ok {
		return KPITimeSeriesSnapshot{}, KPIQueryMetadata{},
			fmt.Errorf("dashboard KPI progress cache type mismatch")
	}
	snapshot.Series = cloneKPITimeSeriesResponse(snapshot.Series)
	snapshot.PeriodProgress = append(
		[]pmstream.PeriodProgress(nil), snapshot.PeriodProgress...,
	)
	return snapshot, metadata, nil
}

func sortKPITimeSeriesSnapshot(snapshot *KPITimeSeriesSnapshot) {
	for name, entries := range snapshot.Series {
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].Time.Before(entries[j].Time)
		})
		out := entries[:0]
		for _, entry := range entries {
			if len(out) > 0 && out[len(out)-1].Time.Equal(entry.Time) {
				if entry.Partial {
					out[len(out)-1] = entry
				}
				continue
			}
			out = append(out, entry)
		}
		snapshot.Series[name] = out
	}
	sort.Slice(snapshot.PeriodProgress, func(i, j int) bool {
		if snapshot.PeriodProgress[i].Granularity != snapshot.PeriodProgress[j].Granularity {
			return snapshot.PeriodProgress[i].Granularity < snapshot.PeriodProgress[j].Granularity
		}
		return snapshot.PeriodProgress[i].WindowStart.Before(
			snapshot.PeriodProgress[j].WindowStart,
		)
	})
}

func (s *Service) getKPITimeSeriesUnprotected(ctx context.Context, kpiNames []string, technology model.Technology, granularity metrics.Granularity, startTime, endTime time.Time) (KPITimeSeriesResponse, error) {
	return s.getKPITimeSeriesUnprotectedWithObservation(
		ctx, kpiNames, technology, granularity, startTime, endTime, true,
	)
}

func (s *Service) getKPITimeSeriesUnprotectedWithObservation(
	ctx context.Context,
	kpiNames []string,
	technology model.Technology,
	granularity metrics.Granularity,
	startTime, endTime time.Time,
	observeMissing bool,
) (KPITimeSeriesResponse, error) {
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

	points, err := s.fetchNetworkKCodeSeries(
		ctx, kcodes, technology, granularity, startTime, endTime, observeMissing,
	)
	if err != nil {
		return nil, err
	}

	for _, p := range points {
		entry := KPITimeSeriesEntry{
			Time:  p.time,
			Value: p.value,
		}
		// 按指标编号直接回填。
		result[p.code] = append(result[p.code], entry)
	}

	return result, nil
}

func (s *Service) observeNetworkRollups(points []NetworkRollupPoint, requested []string, technology model.Technology, granularity metrics.Granularity, now time.Time) {
	if s.metrics == nil {
		return
	}
	latest := make(map[model.Technology]time.Time)
	incomplete := make(map[model.Technology]int)
	present := make(map[model.Technology]map[string]struct{})
	for _, point := range points {
		if technology != "" && point.Technology != technology {
			continue
		}
		if point.WindowEnd.After(latest[point.Technology]) {
			latest[point.Technology] = point.WindowEnd
		}
		if !point.Complete || point.MissingSlots > 0 {
			incomplete[point.Technology]++
		}
		if present[point.Technology] == nil {
			present[point.Technology] = make(map[string]struct{})
		}
		present[point.Technology][point.MetricPath] = struct{}{}
	}
	for tech, count := range incomplete {
		s.metrics.ObserveIncomplete(string(tech), string(granularity), float64(count))
	}
	for tech, windowStart := range latest {
		lag := now.Sub(windowStart).Seconds()
		if lag < 0 {
			lag = 0
		}
		s.metrics.SetRollupLag(string(tech), string(granularity), lag)
	}
	if technology != "" && len(requested) > 0 {
		missing := 0
		for _, path := range normalizeMetricPaths(requested) {
			if _, ok := present[technology][path]; !ok {
				missing++
			}
		}
		s.metrics.ObserveMissing(string(technology), string(granularity), float64(missing))
	}
}

func dashboardSeriesCacheKey(kpiNames []string, technology model.Technology, granularity metrics.Granularity, startTime, endTime time.Time) string {
	names := normalizeMetricPaths(kpiNames)
	return fmt.Sprintf("series:%s:%s:%s:%s:%s",
		technology,
		granularity,
		strings.Join(names, ","),
		startTime.UTC().Format(time.RFC3339Nano),
		endTime.UTC().Format(time.RFC3339Nano),
	)
}

func cloneKPITimeSeriesResponse(source KPITimeSeriesResponse) KPITimeSeriesResponse {
	clone := make(KPITimeSeriesResponse, len(source))
	for name, entries := range source {
		clonedEntries := make([]KPITimeSeriesEntry, len(entries))
		copy(clonedEntries, entries)
		clone[name] = clonedEntries
	}
	return clone
}

// networkSeriesPoint 是全网时序的一行（指标编号 + 图表点位时间 + 值），供 GetKPITimeSeries 回填用。
type networkSeriesPoint struct {
	code  string
	time  time.Time
	value jsonx.Float
}

// fetchNetworkKCodeSeries 直接读取多个指标编号的首页全网发布结果。
func (s *Service) fetchNetworkKCodeSeries(
	ctx context.Context,
	kcodes []string,
	technology model.Technology,
	granularity metrics.Granularity,
	startTime, endTime time.Time,
	observeMissing bool,
) ([]networkSeriesPoint, error) {
	if s.networkRollups == nil {
		return nil, fmt.Errorf("dashboard network rollup reader not configured")
	}

	rows, err := s.networkRollups.ListSeries(ctx, NetworkRollupQuery{
		Technology: technology, Granularity: granularity,
		MetricPaths: kcodes, StartTime: startTime, EndTime: endTime,
	})
	if err != nil {
		return nil, fmt.Errorf("query dashboard network %s kpi series: %w", granularity, err)
	}
	requested := kcodes
	if !observeMissing {
		// Snapshot queries can only decide whether a published result is
		// expected after the progress reader supplies the task-version
		// effective interval.
		requested = nil
	}
	s.observeNetworkRollups(rows, requested, technology, granularity, time.Now())

	points := make([]networkSeriesPoint, 0, len(rows))
	for _, row := range rows {
		if row.WindowStart.Before(startTime) || !row.WindowStart.Before(endTime) {
			continue
		}
		points = append(points, networkSeriesPoint{
			code: row.MetricPath, time: row.WindowStart, value: row.Value,
		})
	}
	return sortAndDedupeNetworkSeriesPoints(points), nil
}

func (s *Service) observeSnapshotMissing(
	ctx context.Context,
	snapshot KPITimeSeriesSnapshot,
	metricIntervals []pmstream.MetricVersionInterval,
	requested []string,
	technology model.Technology,
	granularity metrics.Granularity,
	queryStart, queryEnd time.Time,
) {
	if s.metrics == nil || technology == "" || len(requested) == 0 ||
		snapshot.ProgressState != "available" {
		return
	}
	businessNow := response.TimeInCurrentLocation(ctx, time.Now())
	cutoff := currentNaturalPeriodStart(granularity, businessNow)
	latestClosedStart := previousNaturalPeriodStart(
		granularity, cutoff, businessNow.Location(),
	)
	if latestClosedStart.Before(queryStart) ||
		!latestClosedStart.Before(queryEnd) {
		return
	}

	expected := make(map[string]struct{})
	requestedSet := make(map[string]struct{}, len(requested))
	for _, metricPath := range normalizeMetricPaths(requested) {
		requestedSet[metricPath] = struct{}{}
	}
	for _, interval := range metricIntervals {
		if _, ok := requestedSet[interval.MetricPath]; !ok ||
			interval.EffectiveFrom.IsZero() ||
			!interval.EffectiveFrom.Before(cutoff) ||
			(interval.EffectiveTo != nil &&
				!interval.EffectiveTo.After(latestClosedStart)) {
			continue
		}
		expected[interval.MetricPath] = struct{}{}
	}
	if len(expected) == 0 {
		return
	}

	missing := 0
	for metricPath := range expected {
		found := false
		entries := snapshot.Series[metricPath]
		for _, entry := range entries {
			if !entry.Partial && entry.Time.Equal(latestClosedStart) {
				found = true
				break
			}
		}
		if !found {
			missing++
		}
	}
	s.metrics.ObserveMissing(
		string(technology), string(granularity), float64(missing),
	)
}

func previousNaturalPeriodStart(
	granularity metrics.Granularity,
	currentStart time.Time,
	location *time.Location,
) time.Time {
	if location == nil {
		location = time.UTC
	}
	localStart := currentStart.In(location)
	days := -1
	if granularity == metrics.GranularityWeekly {
		days = -7
	}
	return localStart.AddDate(0, 0, days).UTC()
}

func currentNaturalPeriodStart(
	granularity metrics.Granularity,
	now time.Time,
) time.Time {
	dayStart := time.Date(
		now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location(),
	)
	if granularity != metrics.GranularityWeekly {
		return dayStart.UTC()
	}
	daysSinceMonday := (int(dayStart.Weekday()) + 6) % 7
	return dayStart.AddDate(0, 0, -daysSinceMonday).UTC()
}

func sortAndDedupeNetworkSeriesPoints(points []networkSeriesPoint) []networkSeriesPoint {
	sort.SliceStable(points, func(i, j int) bool {
		if points[i].code == points[j].code {
			return points[i].time.Before(points[j].time)
		}
		return points[i].code < points[j].code
	})
	out := points[:0]
	var lastCode string
	var lastTime time.Time
	for _, p := range points {
		if len(out) > 0 && p.code == lastCode && p.time.Equal(lastTime) {
			continue
		}
		out = append(out, p)
		lastCode = p.code
		lastTime = p.time
	}
	return out
}

// queryNetworkKPISeries 读单个指标在某时窗内的全网时序（供 GetKPITrendComparison 用）。
//
// kpiName 为指标编号（K/C 编号），与 GetKPITimeSeries 同源走 PM Aggregator network 维度。
func (s *Service) queryNetworkKPISeries(ctx context.Context, kpiName string, startTime, endTime time.Time) ([]KPITrendEntry, error) {
	kcodes := []string{kpiName}

	points, err := s.fetchNetworkKCodeSeries(
		ctx, kcodes, "", metrics.GranularityHourly, startTime, endTime, true,
	)
	if err != nil {
		return nil, fmt.Errorf("query network kpi series: %w", err)
	}

	entries := make([]KPITrendEntry, 0, len(points))
	for _, p := range points {
		entries = append(entries, KPITrendEntry{
			Time:  p.time.Format(time.RFC3339),
			Value: float64(p.value),
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
