package dashboard

import (
	"time"

	pmaggregator "github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// kpi_network_query.go -- 首页 KPI 折线图构造全网聚合查询请求。
//
// 首页仪表板消费 network 维度时序。KPI 派生指标不在 dashboard 层直接
// 对已经算好的 KPI 行再做 AVG/SUM，而是复用 PM Aggregator：先按 counter 明细聚合，
// 再按 KPI 公式在同一时间桶内重算，保持与 PM 性能查询页一致的口径。

// buildNetworkKPISeriesRequest 构造首页 KPI 折线图的全网查询请求。
// KPI/counter 的口径细节交给 PM Aggregator 统一处理。
func buildNetworkKPISeriesRequest(codes []string, granularity metrics.Granularity, startTime, endTime time.Time) pmaggregator.QueryRequest {
	return pmaggregator.QueryRequest{
		Granularity: granularity,
		Dimension:   pmaggregator.DimensionNetwork,
		MetricPaths: append([]string(nil), codes...),
		StartTime:   startTime,
		EndTime:     endTime,
	}
}

func buildNetworkKPIHourlySeriesRequest(codes []string, startTime, endTime time.Time) pmaggregator.QueryRequest {
	return buildNetworkKPISeriesRequest(codes, metrics.GranularityHourly, startTime, endTime)
}

func buildNetworkKPI15MinSeriesRequest(codes []string, startTime, endTime time.Time) pmaggregator.QueryRequest {
	return buildNetworkKPISeriesRequest(codes, metrics.Granularity15Min, startTime, endTime)
}
