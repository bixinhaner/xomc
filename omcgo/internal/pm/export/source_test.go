package export

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverMetricColumns_UsesRequestedMetricPaths(t *testing.T) {
	keys, err := discoverMetricColumns(context.Background(), nil, "pm_metrics", []string{"K002", "C001", "K002", ""}, zeroTime(), zeroTime())
	require.NoError(t, err)

	assert.Equal(t, []colKey{
		{code: "K002", mtype: "kpi"},
		{code: "C001", mtype: "counter"},
	}, keys)
}

func zeroTime() (t time.Time) {
	return t
}

func TestFillEmptySource_ExplicitObjectLDNsExportMissingObjectAsPlaceholder(t *testing.T) {
	bucket := time.Date(2026, 7, 14, 7, 45, 0, 0, time.UTC)
	objectWithData := "Cellid=1,PLMN=46000"
	objectWithoutData := "Cellid=2,PLMN=46000"
	src := &sliceSource{batches: [][]ExportRow{{
		{
			Device:      "1202000240194DP0015",
			CellPLMN:    objectWithData,
			MetricCode:  "K900010052",
			MetricType:  "kpi",
			Granularity: "15min",
			Time:        bucket,
			StartTime:   bucket,
			EndTime:     bucket.Add(15 * time.Minute),
			Value:       12.3,
		},
	}}}

	filled := newFillEmptySource(src, aggregator.QueryRequest{
		Dimension:   aggregator.DimensionDevice,
		Granularity: metrics.Granularity15Min,
		DeviceSNs:   []string{"1202000240194DP0015"},
		MetricPaths: []string{"K900010052"},
		ObjectLDNs:  []string{objectWithData, objectWithoutData},
		StartTime:   bucket,
		EndTime:     bucket.Add(15 * time.Minute),
	})

	rows, done, err := filled.Next(context.Background())
	require.NoError(t, err)
	assert.False(t, done)
	require.Len(t, rows, 2)
	assert.Equal(t, objectWithData, rows[0].CellPLMN)
	assert.Equal(t, 12.3, rows[0].Value)
	assert.Equal(t, objectWithoutData, rows[1].CellPLMN)
	assert.True(t, math.IsNaN(rows[1].Value), "filled=true 骨架行应交给 CSV 写成 '-'，不能导出 0")

	rows, done, err = filled.Next(context.Background())
	require.NoError(t, err)
	assert.True(t, done)
	assert.Empty(t, rows)
}
