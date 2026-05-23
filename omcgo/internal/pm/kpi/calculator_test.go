package kpi

import (
	"testing"

	"github.com/omcgo/omcgo/internal/pm/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_AggregateByStatisType_Sum(t *testing.T) {
	got, err := AggregateByStatisType([]float64{1, 2, 3, 4}, metrics.StatisSum)
	require.NoError(t, err)
	assert.Equal(t, 10.0, got)
}

func Test_AggregateByStatisType_Avg(t *testing.T) {
	got, err := AggregateByStatisType([]float64{2, 4, 6, 8}, metrics.StatisAvg)
	require.NoError(t, err)
	assert.Equal(t, 5.0, got)
}

func Test_AggregateByStatisType_Max(t *testing.T) {
	got, err := AggregateByStatisType([]float64{3, 7, 1, 9, 4}, metrics.StatisMax)
	require.NoError(t, err)
	assert.Equal(t, 9.0, got)
}

func Test_AggregateByStatisType_Pct_ReturnsError(t *testing.T) {
	_, err := AggregateByStatisType([]float64{50, 50}, metrics.StatisPct)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pct requires arithmetic context")
}

func Test_AggregateByStatisType_Empty_ReturnsError(t *testing.T) {
	_, err := AggregateByStatisType(nil, metrics.StatisSum)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty values")
}

func Test_AggregateByStatisType_Unknown_ReturnsError(t *testing.T) {
	_, err := AggregateByStatisType([]float64{1}, metrics.StatisType("unknown"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown statis_type")
}

func Test_AggregateByStatisType_SingleValue(t *testing.T) {
	for _, stype := range []metrics.StatisType{metrics.StatisSum, metrics.StatisAvg, metrics.StatisMax} {
		got, err := AggregateByStatisType([]float64{42}, stype)
		require.NoError(t, err, "stype=%s", stype)
		assert.Equal(t, 42.0, got, "stype=%s", stype)
	}
}
