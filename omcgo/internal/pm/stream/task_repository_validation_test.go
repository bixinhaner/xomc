package stream

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateSaveTaskAllowsEmptyMembershipSnapshot(t *testing.T) {
	req := SaveTaskRequest{
		Name: "内置-全网-LTE", Enabled: true, Visibility: "public", Creator: "system",
		Technology: "lte", Dimension: DimensionNetwork,
		Granularities: []Granularity{
			GranularityHourly, GranularityDaily, GranularityWeekly, GranularityMonthly,
		},
		Metrics: []MetricRule{{
			MetricID: "K1", MetricPath: "K1", MetricType: "kpi",
			Aggregation: AggregationFormula, Formula: "C1", Dependencies: []string{"C1"},
		}},
		Counters: []CounterRule{{MetricPath: "C1", Aggregation: AggregationSum}},
	}

	require.NoError(t, validateSaveTask(req))
}
