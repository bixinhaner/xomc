package dashboard

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/model"
	"go.uber.org/zap"
)

// HeatmapData represents alarm heatmap data for visualization.
// 告警热度图数据，按星期几和小时统计告警数量
type HeatmapData struct {
	DaysOfWeek []DayOfWeekData `json:"days_of_week"` // 7天数据，0=周一, 6=周日
	MaxCount   int64           `json:"max_count"`    // 最大告警数（用于热力图颜色范围）
}

// DayOfWeekData represents alarm counts for each hour of a specific day.
// 一天24小时的告警数量分布
type DayOfWeekData struct {
	Day   int     `json:"day"`   // 星期几，0=周一, 6=周日
	Hours []int64 `json:"hours"` // 24小时告警数量，索引0=00:00-00:59, 23=23:00-23:59
}

// AlarmHeatmapBySeverity represents heatmap data broken down by severity.
// 按严重程度分组的告警热度图数据
type AlarmHeatmapBySeverity struct {
	Severity string       `json:"severity"` // 告警级别
	Data     *HeatmapData `json:"data"`     // 热度图数据
}

const alarmHeatmapQuery = `
		SELECT
			EXTRACT(DOW FROM raised_at)::integer as day_of_week,
			EXTRACT(HOUR FROM raised_at)::integer as hour_of_day,
			COUNT(*) as alarm_count
		FROM %s
		WHERE raised_at > NOW() - INTERVAL '1 day' * $1
		GROUP BY day_of_week, hour_of_day
		ORDER BY day_of_week, hour_of_day
	`

const alarmHeatmapBySeverityQuery = `
		SELECT
			severity,
			EXTRACT(DOW FROM raised_at)::integer as day_of_week,
			EXTRACT(HOUR FROM raised_at)::integer as hour_of_day,
			COUNT(*) as alarm_count
		FROM %s
		WHERE raised_at > NOW() - INTERVAL '1 day' * $1
			AND ($2 = 0 OR severity = $2 OR severity = $2 + 31000)
		GROUP BY severity, day_of_week, hour_of_day
		ORDER BY severity, day_of_week, hour_of_day
	`

// GetAlarmHeatmap retrieves alarm heatmap data for the specified time range.
// 获取指定时间范围的告警热度图数据
func (s *Service) GetAlarmHeatmap(ctx context.Context, days int) (*HeatmapData, error) {
	result := initializeEmptyHeatmap()
	if err := s.queryAlarmHeatmapInto(ctx, s.pgPool, "alarms_active", days, result); err != nil {
		return nil, err
	}
	if s.tsPool != nil {
		if err := s.queryAlarmHeatmapInto(ctx, s.tsPool, "alarms_history", days, result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *Service) queryAlarmHeatmapInto(ctx context.Context, pool *pgxpool.Pool, tableName string, days int, result *HeatmapData) error {
	if pool == nil {
		return nil
	}
	rows, err := pool.Query(ctx, fmt.Sprintf(alarmHeatmapQuery, tableName), days)
	if err != nil {
		logDashboardQueryError(s.logger, "failed to query alarm heatmap", err, zap.String("table", tableName))
		return fmt.Errorf("query alarm heatmap from %s: %w", tableName, err)
	}
	defer rows.Close()

	var scanErrors int
	totalRows := 0
	for rows.Next() {
		totalRows++
		var dayOfWeek, hourOfDay int
		var alarmCount int64
		if err := rows.Scan(&dayOfWeek, &hourOfDay, &alarmCount); err != nil {
			s.logger.Warn("failed to scan heatmap row", zap.Error(err))
			scanErrors++
			continue
		}

		// PostgreSQL DOW: 0=Sunday, 调整为 0=Monday
		adjustedDay := (dayOfWeek + 6) % 7
		addHeatmapBucket(result, adjustedDay, hourOfDay, alarmCount)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate alarm heatmap rows from %s: %w", tableName, err)
	}

	if scanErrors > 0 {
		s.logger.Warn("heatmap scan completed with errors",
			zap.String("table", tableName),
			zap.Int("total_rows", totalRows),
			zap.Int("scan_errors", scanErrors),
			zap.Int("successful_rows", totalRows-scanErrors))
	}

	return nil
}

// GetAlarmHeatmapBySeverity retrieves alarm heatmap data broken down by severity.
// 获取按严重程度分组的告警热度图数据
func (s *Service) GetAlarmHeatmapBySeverity(ctx context.Context, days int, severity string) (*AlarmHeatmapBySeverity, error) {
	heatmapBySeverity, err := s.queryHeatmapBySeverityMap(ctx, days, severity)
	if err != nil {
		return nil, err
	}

	return s.selectHeatmapResult(heatmapBySeverity, severity), nil
}

// queryHeatmapBySeverityMap executes the heatmap query and returns data grouped by severity.
// 执行热度图查询并按严重程度分组返回数据
func (s *Service) queryHeatmapBySeverityMap(ctx context.Context, days int, severity string) (map[string]*HeatmapData, error) {
	result := make(map[string]*HeatmapData)
	if err := s.queryHeatmapBySeverityInto(ctx, s.pgPool, "alarms_active", days, severity, result); err != nil {
		return nil, err
	}
	if s.tsPool != nil {
		if err := s.queryHeatmapBySeverityInto(ctx, s.tsPool, "alarms_history", days, severity, result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *Service) queryHeatmapBySeverityInto(ctx context.Context, pool *pgxpool.Pool, tableName string, days int, severity string, result map[string]*HeatmapData) error {
	if pool == nil {
		return nil
	}
	rows, err := pool.Query(ctx, fmt.Sprintf(alarmHeatmapBySeverityQuery, tableName), days, severityFilterValue(severity))
	if err != nil {
		logDashboardQueryError(s.logger, "failed to query alarm heatmap by severity", err, zap.String("table", tableName))
		return fmt.Errorf("query alarm heatmap by severity from %s: %w", tableName, err)
	}
	defer rows.Close()

	scanned, err := s.scanHeatmapBySeverity(rows)
	if err != nil {
		return err
	}
	mergeHeatmapBySeverity(result, scanned)
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate alarm heatmap by severity rows from %s: %w", tableName, err)
	}
	return nil
}

// scanHeatmapBySeverity scans query results and organizes data by severity.
// 扫描查询结果并按严重程度组织数据
func (s *Service) scanHeatmapBySeverity(rows interface{}) (map[string]*HeatmapData, error) {
	// Type assertion for pgx rows
	type RowScanner interface {
		Next() bool
		Scan(dest ...any) error
	}

	scanner, ok := rows.(RowScanner)
	if !ok {
		return nil, fmt.Errorf("invalid rows type")
	}

	heatmapBySeverity := make(map[string]*HeatmapData)

	for scanner.Next() {
		var sevValue int
		var dayOfWeek, hourOfDay int
		var alarmCount int64
		if err := scanner.Scan(&sevValue, &dayOfWeek, &hourOfDay, &alarmCount); err != nil {
			s.logger.Warn("failed to scan heatmap severity row", zap.Error(err))
			continue
		}

		sev := severityToLabel(model.AlarmSeverity(sevValue))

		// 初始化该严重程度的数据结构（如果尚未存在）
		if _, exists := heatmapBySeverity[sev]; !exists {
			heatmapBySeverity[sev] = initializeEmptyHeatmap()
		}

		// PostgreSQL DOW: 0=Sunday, 调整为 0=Monday
		adjustedDay := (dayOfWeek + 6) % 7
		addHeatmapBucket(heatmapBySeverity[sev], adjustedDay, hourOfDay, alarmCount)
	}

	return heatmapBySeverity, nil
}

func addHeatmapBucket(result *HeatmapData, adjustedDay, hourOfDay int, alarmCount int64) {
	if adjustedDay < 0 || adjustedDay >= 7 || hourOfDay < 0 || hourOfDay >= 24 {
		return
	}
	result.DaysOfWeek[adjustedDay].Hours[hourOfDay] += alarmCount
	if result.DaysOfWeek[adjustedDay].Hours[hourOfDay] > result.MaxCount {
		result.MaxCount = result.DaysOfWeek[adjustedDay].Hours[hourOfDay]
	}
}

func mergeHeatmapBySeverity(dst, src map[string]*HeatmapData) {
	for severity, data := range src {
		if _, exists := dst[severity]; !exists {
			dst[severity] = initializeEmptyHeatmap()
		}
		for dayIdx, day := range data.DaysOfWeek {
			for hourIdx, count := range day.Hours {
				addHeatmapBucket(dst[severity], dayIdx, hourIdx, count)
			}
		}
	}
}

// selectHeatmapResult selects the appropriate heatmap based on severity filter.
// 根据严重程度过滤器选择合适的热度图结果
func (s *Service) selectHeatmapResult(heatmapBySeverity map[string]*HeatmapData, severity string) *AlarmHeatmapBySeverity {
	// 如果指定了严重程度，返回单个
	if severity != "" {
		if data, exists := heatmapBySeverity[severity]; exists {
			return &AlarmHeatmapBySeverity{
				Severity: severity,
				Data:     data,
			}
		}
		// 没有数据时返回空结构
		return createEmptyAlarmHeatmapBySeverity(severity)
	}

	// 返回所有严重程度中告警最多的一个作为默认
	maxSev, _ := findMaxSeverity(heatmapBySeverity)
	if maxSev != "" {
		return &AlarmHeatmapBySeverity{
			Severity: maxSev,
			Data:     heatmapBySeverity[maxSev],
		}
	}

	// 没有任何数据
	return createEmptyAlarmHeatmapBySeverity("all")
}

// severityFilterValue converts the optional severity label from the query string
// to the smallint filter value used in SQL; empty means no filter (0 sentinel).
// 把请求侧的 severity 标签转为 SQL 过滤用的整型值；空串=不过滤（0 哨兵）
func severityFilterValue(severity string) int {
	if severity == "" {
		return 0
	}
	return severityFromLabel(severity)
}

// initializeEmptyHeatmap creates an empty heatmap structure with 7 days and 24 hours.
// 创建空的热度图结构（7天 x 24小时）
func initializeEmptyHeatmap() *HeatmapData {
	h := &HeatmapData{
		DaysOfWeek: make([]DayOfWeekData, 7),
		MaxCount:   0,
	}
	for i := 0; i < 7; i++ {
		h.DaysOfWeek[i] = DayOfWeekData{
			Day:   i,
			Hours: make([]int64, 24),
		}
	}
	return h
}

// findMaxSeverity finds the severity with the maximum alarm count.
// 查找告警数最多的严重程度
func findMaxSeverity(heatmapBySeverity map[string]*HeatmapData) (string, int64) {
	var maxSev string
	var maxCount int64 = 0
	for sev, data := range heatmapBySeverity {
		if data.MaxCount > maxCount {
			maxCount = data.MaxCount
			maxSev = sev
		}
	}
	return maxSev, maxCount
}

// createEmptyAlarmHeatmapBySeverity creates an empty AlarmHeatmapBySeverity structure.
// 创建空的 AlarmHeatmapBySeverity 结构
func createEmptyAlarmHeatmapBySeverity(severity string) *AlarmHeatmapBySeverity {
	return &AlarmHeatmapBySeverity{
		Severity: severity,
		Data:     initializeEmptyHeatmap(),
	}
}

// GetAlarmHeatmapAll retrieves all severity heatmap data.
// 获取所有严重程度的告警热度图数据（用于前端切换）
func (s *Service) GetAlarmHeatmapAll(ctx context.Context, days int) (map[string]*HeatmapData, error) {
	query := `
		SELECT
			severity,
			EXTRACT(DOW FROM raised_at)::integer as day_of_week,
			EXTRACT(HOUR FROM raised_at)::integer as hour_of_day,
			COUNT(*) as alarm_count
		FROM alarms_history
		WHERE raised_at > NOW() - INTERVAL '1 day' * $1
		GROUP BY severity, day_of_week, hour_of_day
		ORDER BY severity, day_of_week, hour_of_day
	`

	// alarms_history 在时序库（TsPool）。
	rows, err := s.tsPool.Query(ctx, query, days)
	if err != nil {
		logDashboardQueryError(s.logger, "failed to query all alarm heatmaps", err)
		return nil, fmt.Errorf("query all alarm heatmaps: %w", err)
	}
	defer rows.Close()

	result := make(map[string]*HeatmapData)

	for rows.Next() {
		var sev string
		var dayOfWeek, hourOfDay int
		var alarmCount int64
		if err := rows.Scan(&sev, &dayOfWeek, &hourOfDay, &alarmCount); err != nil {
			s.logger.Warn("failed to scan all heatmap row", zap.Error(err))
			continue
		}

		if _, exists := result[sev]; !exists {
			result[sev] = &HeatmapData{
				DaysOfWeek: make([]DayOfWeekData, 7),
				MaxCount:   0,
			}
			for i := 0; i < 7; i++ {
				result[sev].DaysOfWeek[i] = DayOfWeekData{
					Day:   i,
					Hours: make([]int64, 24),
				}
			}
		}

		adjustedDay := (dayOfWeek + 6) % 7
		if adjustedDay >= 0 && adjustedDay < 7 && hourOfDay >= 0 && hourOfDay < 24 {
			result[sev].DaysOfWeek[adjustedDay].Hours[hourOfDay] = alarmCount
			if alarmCount > result[sev].MaxCount {
				result[sev].MaxCount = alarmCount
			}
		}
	}

	return result, nil
}
