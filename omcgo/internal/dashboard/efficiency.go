package dashboard

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// EfficiencyMetrics represents alarm handling efficiency metrics.
// 告警处理效率指标，包含 MTTA、MTTR、确认率、清除率等关键运维指标
type EfficiencyMetrics struct {
	Severity               string                `json:"severity"`                // 告警级别
	AcknowledgedCount      int64                 `json:"acknowledged_count"`      // 已确认告警数
	ClearedCount           int64                 `json:"cleared_count"`           // 已清除告警数
	TotalCount             int64                 `json:"total_count"`             // 总告警数
	AvgAcknowledgeMinutes  float64               `json:"avg_acknowledge_minutes"`  // 平均确认时间(MTTA)
	AvgResolveMinutes      float64               `json:"avg_resolve_minutes"`      // 平均解决时间(MTTR)
	AcknowledgeRate        float64               `json:"acknowledge_rate"`        // 确认率(%)
	ClearRate              float64               `json:"clear_rate"`              // 清除率(%)
	DailyTrend             []DailyEfficiencyTrend `json:"daily_trend"`           // 近7天趋势
}

// DailyEfficiencyTrend represents daily efficiency trend data.
// 每日效率趋势数据点
type DailyEfficiencyTrend struct {
	Date                  string  `json:"date"`                  // 日期
	AvgAcknowledgeMinutes float64 `json:"avg_acknowledge_minutes"` // 当日平均确认时间
	AvgResolveMinutes     float64 `json:"avg_resolve_minutes"`     // 当日平均解决时间
}

// GetEfficiencyMetrics retrieves alarm handling efficiency metrics.
// 获取告警处理效率指标，包括 MTTA、MTTR、确认率、清除率及近7天趋势
func (s *Service) GetEfficiencyMetrics(ctx context.Context) (*EfficiencyMetrics, error) {
	// 从物化视图获取按严重程度分组的效率指标
	query := `
		SELECT
			severity,
			acknowledged_count,
			cleared_count,
			total_count,
			avg_acknowledge_minutes,
			avg_resolve_minutes,
			acknowledge_rate,
			clear_rate
		FROM alarm_efficiency_metrics
		ORDER BY severity
	`

	rows, err := s.pgPool.Query(ctx, query)
	if err != nil {
		s.logger.Error("failed to query efficiency metrics", zap.Error(err))
		return nil, fmt.Errorf("query efficiency metrics: %w", err)
	}
	defer rows.Close()

	var metrics []EfficiencyMetrics
	var scanErrors int
	for rows.Next() {
		var m EfficiencyMetrics
		if err := rows.Scan(
			&m.Severity,
			&m.AcknowledgedCount,
			&m.ClearedCount,
			&m.TotalCount,
			&m.AvgAcknowledgeMinutes,
			&m.AvgResolveMinutes,
			&m.AcknowledgeRate,
			&m.ClearRate,
		); err != nil {
			s.logger.Warn("failed to scan efficiency metrics row", zap.Error(err))
			scanErrors++
			continue
		}

		// 获取该严重程度的7天趋势
		m.DailyTrend, err = s.getEfficiencyTrend(ctx, m.Severity)
		if err != nil {
			s.logger.Warn("failed to get efficiency trend",
				zap.String("severity", m.Severity),
				zap.Error(err))
		}

		metrics = append(metrics, m)
	}

	// 记录扫描错误统计，便于监控数据质量
	if scanErrors > 0 {
		s.logger.Warn("efficiency metrics scan completed with errors",
			zap.Int("total_rows", len(metrics)+scanErrors),
			zap.Int("scan_errors", scanErrors))
	}

	// 如果没有数据，返回空指标
	if len(metrics) == 0 {
		return &EfficiencyMetrics{
			Severity:      "all",
			DailyTrend:    []DailyEfficiencyTrend{},
			AvgAcknowledgeMinutes: 0,
			AvgResolveMinutes:     0,
			AcknowledgeRate:       0,
			ClearRate:             0,
		}, nil
	}

	// 返回第一个指标（或可以合并所有严重程度）
	return &metrics[0], nil
}

// getEfficiencyTrend retrieves daily efficiency trend for a given severity over the past 7 days.
// 获取指定严重程度的近7天效率趋势
func (s *Service) getEfficiencyTrend(ctx context.Context, severity string) ([]DailyEfficiencyTrend, error) {
	query := `
		SELECT
			TO_CHAR(DATE(raised_at), 'YYYY-MM-DD') as date,
			COALESCE(AVG(EXTRACT(EPOCH FROM (acknowledged_at - raised_at)) / 60)
				FILTER (WHERE acknowledged_at IS NOT NULL), 0) as avg_ack_minutes,
			COALESCE(AVG(EXTRACT(EPOCH FROM (cleared_at - raised_at)) / 60)
				FILTER (WHERE cleared_at IS NOT NULL), 0) as avg_resolve_minutes
		FROM alarms_history
		WHERE raised_at > NOW() - INTERVAL '7 days'
			AND severity = $1
		GROUP BY DATE(raised_at)
		ORDER BY date DESC
	`

	rows, err := s.pgPool.Query(ctx, query, severity)
	if err != nil {
		return nil, fmt.Errorf("query efficiency trend: %w", err)
	}
	defer rows.Close()

	var trend []DailyEfficiencyTrend
	for rows.Next() {
		var t DailyEfficiencyTrend
		if err := rows.Scan(&t.Date, &t.AvgAcknowledgeMinutes, &t.AvgResolveMinutes); err != nil {
			continue
		}
		trend = append(trend, t)
	}

	return trend, nil
}

// RefreshEfficiencyMetrics manually refreshes the materialized view.
// 手动刷新告警效率指标物化视图
func (s *Service) RefreshEfficiencyMetrics(ctx context.Context) error {
	_, err := s.pgPool.Exec(ctx, "REFRESH MATERIALIZED VIEW CONCURRENTLY alarm_efficiency_metrics")
	if err != nil {
		s.logger.Error("failed to refresh efficiency metrics", zap.Error(err))
		return fmt.Errorf("refresh efficiency metrics: %w", err)
	}

	s.logger.Info("refreshed alarm efficiency metrics")
	return nil
}

// GetOverallEfficiencyMetrics retrieves aggregated efficiency metrics across all severities.
// 获取所有严重程度汇总的效率指标
// 注意：这里直接从 alarms_history 原始表计算，而非从物化视图聚合，
// 因为物化视图中的 avg_acknowledge_minutes 已经是平均值，不能直接用于加权平均
func (s *Service) GetOverallEfficiencyMetrics(ctx context.Context) (*EfficiencyMetrics, error) {
	query := `
		SELECT
			'overall' as severity,
			COUNT(*) FILTER (WHERE acknowledged_at IS NOT NULL) as acknowledged_count,
			COUNT(*) FILTER (WHERE cleared_at IS NOT NULL) as cleared_count,
			COUNT(*) as total_count,
			-- MTTA: 平均确认时间（分钟），使用原始时间差计算
			COALESCE(AVG(EXTRACT(EPOCH FROM (acknowledged_at - raised_at)) / 60)
				FILTER (WHERE acknowledged_at IS NOT NULL), 0) as avg_acknowledge_minutes,
			-- MTTR: 平均解决时间（分钟），使用原始时间差计算
			COALESCE(AVG(EXTRACT(EPOCH FROM (cleared_at - raised_at)) / 60)
				FILTER (WHERE cleared_at IS NOT NULL), 0) as avg_resolve_minutes,
			-- 确认率（百分比）
			ROUND(100.0 * COUNT(*) FILTER (WHERE acknowledged_at IS NOT NULL) /
				NULLIF(COUNT(*), 0), 2) as acknowledge_rate,
			-- 清除率（百分比）
			ROUND(100.0 * COUNT(*) FILTER (WHERE cleared_at IS NOT NULL) /
				NULLIF(COUNT(*), 0), 2) as clear_rate
		FROM alarms_history
		WHERE cleared_at > NOW() - INTERVAL '30 days'
	`

	var m EfficiencyMetrics
	err := s.pgPool.QueryRow(ctx, query).Scan(
		&m.Severity,
		&m.AcknowledgedCount,
		&m.ClearedCount,
		&m.TotalCount,
		&m.AvgAcknowledgeMinutes,
		&m.AvgResolveMinutes,
		&m.AcknowledgeRate,
		&m.ClearRate,
	)
	if err != nil {
		s.logger.Error("failed to query overall efficiency metrics", zap.Error(err))
		return nil, fmt.Errorf("query overall efficiency metrics: %w", err)
	}

	// 获取近7天趋势（所有严重程度）
	m.DailyTrend, err = s.getOverallEfficiencyTrend(ctx)
	if err != nil {
		s.logger.Warn("failed to get overall efficiency trend", zap.Error(err))
	}

	return &m, nil
}

// getOverallEfficiencyTrend retrieves overall daily efficiency trend across all severities.
// 获取所有严重程度汇总的近7天效率趋势
func (s *Service) getOverallEfficiencyTrend(ctx context.Context) ([]DailyEfficiencyTrend, error) {
	query := `
		SELECT
			TO_CHAR(DATE(raised_at), 'YYYY-MM-DD') as date,
			COALESCE(AVG(EXTRACT(EPOCH FROM (acknowledged_at - raised_at)) / 60)
				FILTER (WHERE acknowledged_at IS NOT NULL), 0) as avg_ack_minutes,
			COALESCE(AVG(EXTRACT(EPOCH FROM (cleared_at - raised_at)) / 60)
				FILTER (WHERE cleared_at IS NOT NULL), 0) as avg_resolve_minutes
		FROM alarms_history
		WHERE raised_at > NOW() - INTERVAL '7 days'
		GROUP BY DATE(raised_at)
		ORDER BY date DESC
	`

	rows, err := s.pgPool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query overall efficiency trend: %w", err)
	}
	defer rows.Close()

	var trend []DailyEfficiencyTrend
	for rows.Next() {
		var t DailyEfficiencyTrend
		if err := rows.Scan(&t.Date, &t.AvgAcknowledgeMinutes, &t.AvgResolveMinutes); err != nil {
			continue
		}
		trend = append(trend, t)
	}

	return trend, nil
}
