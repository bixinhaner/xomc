package dashboard

import (
	"context"
	"fmt"

	"github.com/omcgo/omcgo/internal/core/jsonx"
	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// AggregatorCounterSeriesReader adapts the shared PM aggregator to the dashboard
// Counter series boundary. It is read-only and only queries the network dimension.
type AggregatorCounterSeriesReader struct {
	aggregator *aggregator.Aggregator
}

func NewAggregatorCounterSeriesReader(pmAggregator *aggregator.Aggregator) *AggregatorCounterSeriesReader {
	return &AggregatorCounterSeriesReader{aggregator: pmAggregator}
}

func (r *AggregatorCounterSeriesReader) ListSeries(ctx context.Context, query NetworkRollupQuery) ([]NetworkRollupPoint, error) {
	if r == nil || r.aggregator == nil {
		return nil, fmt.Errorf("dashboard counter aggregator is not configured")
	}
	metricType := metrics.MetricTypeCounter
	request := aggregator.QueryRequest{
		Granularity: query.Granularity,
		Dimension:   aggregator.DimensionNetwork,
		MetricPaths: query.MetricPaths,
		MetricType:  &metricType,
		StartTime:   query.StartTime,
		EndTime:     query.EndTime,
	}
	if query.Technology != "" {
		request.Technologies = []string{string(query.Technology)}
	}
	rows, err := r.aggregator.Query(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("query dashboard counter aggregator: %w", err)
	}
	points := make([]NetworkRollupPoint, 0, len(rows))
	for _, row := range rows {
		if row.MetricType != metrics.MetricTypeCounter {
			return nil, fmt.Errorf("dashboard counter aggregator returned metric type %q for %q", row.MetricType, row.MetricPath)
		}
		points = append(points, NetworkRollupPoint{
			Technology:  query.Technology,
			MetricPath:  row.MetricPath,
			Granularity: row.Granularity,
			WindowStart: row.Time,
			WindowEnd:   row.EndTime,
			Value:       jsonx.Float(row.MetricValue),
			Complete:    true,
			CreatedAt:   row.IngestTime,
		})
	}
	return points, nil
}

var _ CounterSeriesReader = (*AggregatorCounterSeriesReader)(nil)
