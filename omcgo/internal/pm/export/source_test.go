package export

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

func TestDiscoverMetricColumns_UsesRequestedMetricPaths(t *testing.T) {
	keys, err := discoverMetricColumns(context.Background(), nil, "pm_metrics", aggregator.QueryRequest{
		MetricPaths: []string{"K002", "C001", "K002", ""},
	})
	require.NoError(t, err)

	assert.Equal(t, []colKey{
		{code: "K002", mtype: "kpi"},
		{code: "C001", mtype: "counter"},
	}, keys)
}

func TestDiscoverAdhocColumns_EmptyConfiguredMetricPathsFallsBackToDistinct(t *testing.T) {
	for _, metricPaths := range [][]string{nil, {}, {"", "  "}} {
		db := &recordingExportQuerier{results: []pgx.Rows{
			&adhocFakeRows{rows: [][]any{{"K_STORED", "kpi"}}},
		}}

		keys, err := discoverAdhocColumns(
			context.Background(),
			db,
			uuid.New(),
			metricPaths,
			adhocExportFilter{},
		)

		require.NoError(t, err)
		assert.Equal(t, []colKey{{code: "K_STORED", mtype: "kpi"}}, keys)
		require.Len(t, db.queries, 1)
		assert.Contains(t, db.queries[0].sql, "SELECT DISTINCT r.metric_path")
	}
}

func TestNormalizeStoredResultExportRequest_ClearsMetricTypeForMixedMetricPaths(t *testing.T) {
	mt := metrics.MetricTypeKPI
	req := normalizeStoredResultExportRequest(aggregator.QueryRequest{
		MetricType:  &mt,
		MetricPaths: []string{"KGSM0101", "CGSM0010001"},
	})

	assert.Nil(t, req.MetricType, "混选 KPI/counter 时不能用单一 metric_type 过滤")
}

func TestNormalizeStoredResultExportRequest_KeepsMetricTypeForMatchingMetricPaths(t *testing.T) {
	mt := metrics.MetricTypeKPI
	req := normalizeStoredResultExportRequest(aggregator.QueryRequest{
		MetricType:  &mt,
		MetricPaths: []string{"KGSM0101", "KGSM0102"},
	})

	require.NotNil(t, req.MetricType)
	assert.Equal(t, metrics.MetricTypeKPI, *req.MetricType)
}

func TestDashboardDeviceSource_KeysetAdvancesWhenBatchSizeReached(t *testing.T) {
	start := time.Date(2026, 7, 20, 7, 0, 0, 0, time.UTC)
	firstBatch := make([][]any, 0, batchSize)
	var lastFirstID uuid.UUID
	for i := 0; i < batchSize; i++ {
		id := uuid.New()
		lastFirstID = id
		firstBatch = append(firstBatch, deviceMetricRow(id, "SN1", "KGSM0101", "kpi", 10, start.Add(time.Duration(i)*time.Second)))
	}
	secondID := uuid.New()
	metricDB := &recordingExportQuerier{results: []pgx.Rows{
		&adhocFakeRows{rows: firstBatch},
		&adhocFakeRows{rows: [][]any{deviceMetricRow(secondID, "SN1", "CGSM0010001", "counter", 20, start.Add(time.Hour))}},
	}}
	src := newDashboardDeviceSource(metricDB, "pm_metrics_hourly", aggregator.QueryRequest{
		Dimension:   aggregator.DimensionDevice,
		Granularity: metrics.GranularityHourly,
		DeviceSNs:   []string{"SN1"},
		MetricPaths: []string{"KGSM0101", "CGSM0010001"},
	}, nil)

	rows, done, err := src.Next(context.Background())
	require.NoError(t, err)
	assert.False(t, done)
	require.Len(t, rows, batchSize)

	rows, done, err = src.Next(context.Background())
	require.NoError(t, err)
	assert.False(t, done)
	require.Len(t, rows, 1)
	assert.Equal(t, "CGSM0010001", rows[0].MetricCode)

	rows, done, err = src.Next(context.Background())
	require.NoError(t, err)
	assert.True(t, done)
	assert.Empty(t, rows)
	require.Len(t, metricDB.queries, 2)
	assert.NotContains(t, metricDB.queries[0].sql, `("time", id) >`)
	assert.Contains(t, metricDB.queries[1].sql, `("time", id) >`)
	assert.Contains(t, metricDB.queries[1].args, lastFirstID)
}

func deviceMetricRow(id uuid.UUID, sn, metricPath, metricType string, value float64, tm time.Time) []any {
	return []any{
		id, "OUI1", sn, metricPath, metricType, value, "pct", "hourly",
		tm, tm, tm.Add(time.Hour), "Cellid=1,PLMN=46000",
	}
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
