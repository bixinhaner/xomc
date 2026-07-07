package dashboard

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// kpi_network_query.go —— 首页 KPI 折线图取数改读「全网连续聚合视图（Cagg）」。
//
// 背景：通过 TimescaleDB 原生 Continuous Aggregates 建立 pm_metrics_hourly_cagg 视图，
// 取代旧版读 pm_adhoc_aggregation_results 及人工缝合 15 分钟缺口的复杂逻辑。
// TimescaleDB 在读取开启连续聚合的视图时 (materialized_only=false)，引擎会自动将
// 历史物化的聚合数据与最新进入基础表但尚未物化的明细数据合并（Real-Time Aggregation），
// 原本在应用层完成的操作全部下推至数据库。

// buildNetworkKPISeriesQuery 构建「按指标编号读全网连续聚合视图」的 SQL（纯函数，便于单测）。
// 过滤：metric_path ∈ codes + bucket_time ∈ [startTime,endTime]。
// 按 metric_path, bucket_time 升序排列。
func buildNetworkKPISeriesQuery(codes []string, startTimeArg, endTimeArg any) (string, []any, error) {
	// 针对 Continuous Aggregates 中已经归类算好的各种 statis_type 分项，
	// 我们用 CASE 根据该条指标最初设定的 statis_type 直接拾取所需数值列。
	const aggValueExpr = `
		CASE MIN(statis_type)
			WHEN 'sum' THEN SUM(sum_val)
			WHEN 'avg' THEN SUM(sum_val) / NULLIF(SUM(sample_count), 0)
			WHEN 'max' THEN MAX(max_val)
			WHEN 'min' THEN MIN(min_val)
			WHEN 'pct' THEN AVG(avg_val)
			ELSE AVG(avg_val)
		END`

	return storage.Psql.Select("metric_path", "bucket_time AS time", aggValueExpr+" AS metric_value").
		From("pm_metrics_hourly_cagg").
		Where("metric_path = ANY(?)", codes).
		Where(sq.GtOrEq{"bucket_time": startTimeArg}).
		Where(sq.LtOrEq{"bucket_time": endTimeArg}).
		GroupBy("metric_path", "bucket_time").
		OrderBy("metric_path", "bucket_time ASC").
		ToSql()
}
