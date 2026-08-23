package aggregator

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/metrics"
)

// T-0194 C2：Aggregator.Count 返回与 Query 同过滤下命中真实总数（忽略 Limit/Offset）。

// device 维度：按 Query 的自然键去重后计数，WHERE 含过滤项、不带分页。
func Test_Count_Device(t *testing.T) {
	var gotSQL string
	db := &stubDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		gotSQL = sql
		return countRow{n: 1234}
	}
	a := New(db, nil, nil)
	n, err := a.Count(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionDevice,
		DeviceSNs:   []string{"SN-1"},
		Limit:       100,
		Offset:      50,
	})
	require.NoError(t, err)
	assert.Equal(t, 1234, n)
	assert.True(t, strings.HasPrefix(strings.TrimSpace(gotSQL), "SELECT COUNT(*)"), gotSQL)
	assert.Contains(t, gotSQL, "SELECT DISTINCT ON (device_oui, device_sn, metric_path, granularity, \"time\", object_ldn) 1")
	assert.Contains(t, gotSQL, "FROM pm_aggregation_results r")
	assert.Contains(t, gotSQL, "r.dimension_key IN (SELECT id::text FROM device_dim")
	assert.NotContains(t, gotSQL, "FROM pm_metrics_hourly")
	assert.NotContains(t, gotSQL, "LIMIT")
	assert.NotContains(t, gotSQL, "OFFSET")
	assert.Contains(t, gotSQL, "ORDER BY device_oui, device_sn, metric_path, granularity, \"time\", object_ldn, ingest_time DESC")
}

func Test_Count_Device_PageByPivotRowCountsDistinctPivotKeys(t *testing.T) {
	var gotSQL string
	db := &stubDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		gotSQL = sql
		return countRow{n: 88}
	}
	a := New(db, nil, nil)
	n, err := a.Count(context.Background(), QueryRequest{
		Granularity:    metrics.Granularity15Min,
		Dimension:      DimensionDevice,
		DeviceSNs:      []string{"SN-1"},
		MetricPaths:    []string{"C1", "C2"},
		PageByPivotRow: true,
		Limit:          50,
		Offset:         100,
	})
	require.NoError(t, err)
	assert.Equal(t, 88, n)
	assert.Contains(t, gotSQL, "SELECT COUNT(*) FROM (")
	assert.Contains(t, gotSQL, "SELECT DISTINCT device_oui, device_sn, COALESCE(object_ldn, '') AS object_ldn, granularity, \"time\"")
	assert.Contains(t, gotSQL, "FROM pm_measurement_anchors")
	assert.Contains(t, gotSQL, "d.metric_id=ANY(s.metric_ids)")
	assert.NotContains(t, gotSQL, "unnest(s.metric_ids)")
	assert.NotContains(t, gotSQL, "LIMIT")
	assert.NotContains(t, gotSQL, "OFFSET")
	assert.NotContains(t, gotSQL, "metric_path, granularity, time")
}

func Test_Count_Device_ExplicitObjectSkeletonReadsAnchorsWithoutMetricExpansion(t *testing.T) {
	var gotSQL string
	db := &stubDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		gotSQL = sql
		return countRow{n: 18}
	}
	a := New(db, nil, nil)
	start := time.Date(2026, 7, 28, 11, 0, 0, 0, time.UTC)

	n, err := a.Count(context.Background(), QueryRequest{
		Granularity:    metrics.Granularity15Min,
		Dimension:      DimensionDevice,
		DeviceSNs:      []string{"SN-1"},
		ObjectLDNs:     []string{"Cellid=1"},
		StartTime:      start,
		EndTime:        start.Add(4 * time.Hour),
		PageByPivotRow: true,
	})

	require.NoError(t, err)
	assert.Equal(t, 18, n)
	assert.Contains(t, gotSQL, "FROM pm_measurement_anchors a")
	assert.Contains(t, gotSQL, "JOIN device_dim dev ON dev.id = a.device_dim_id")
	assert.NotContains(t, gotSQL, "FROM pm_metrics ")
	assert.NotContains(t, gotSQL, "pm_metric_dictionary")
	assert.NotContains(t, gotSQL, "pm_metric_values")
}

func Test_Count_Device_HourlyMetricPathsUsesAggregationResults(t *testing.T) {
	var gotSQL string
	db := &stubDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		gotSQL = sql
		return countRow{n: 1}
	}
	a := New(db, nil, nil)

	n, err := a.Count(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionDevice,
		DeviceSNs:   []string{"SN-1"},
		MetricPaths: []string{"C1"},
	})

	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.Contains(t, gotSQL, "FROM pm_aggregation_results r")
	assert.Contains(t, gotSQL, "r.dimension_key IN (SELECT id::text FROM device_dim")
	assert.NotContains(t, gotSQL, "FROM pm_metrics_hourly")
	assert.Contains(t, gotSQL, "metric_path =")
	assert.NotContains(t, gotSQL, "pm_hourly_bucket_versions")
	assert.NotContains(t, gotSQL, "pm_hourly_anchors")
	assert.NotContains(t, gotSQL, "pm_hourly_values")
}

// device_group 维度：GROUP BY 子查询包成 COUNT(*) FROM (...) sub，数的是分组数。
func Test_Count_DeviceGroup_SubqueryCount(t *testing.T) {
	var gotSQL string
	db := &stubDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		gotSQL = sql
		return countRow{n: 7}
	}
	a := New(db, nil, nil)
	n, err := a.Count(context.Background(), QueryRequest{
		Granularity:    metrics.GranularityDaily,
		Dimension:      DimensionDeviceGroup,
		DeviceGroupIDs: []uuid.UUID{uuid.New()},
		Limit:          100,
	})
	require.NoError(t, err)
	assert.Equal(t, 7, n)
	assert.Contains(t, gotSQL, "SELECT COUNT(*) FROM (")
	assert.Contains(t, gotSQL, "GROUP BY device_group_id")
	assert.Contains(t, gotSQL, ") sub")
}

// network 维度：GROUP BY metric_path/granularity/time 子查询计数。
func Test_Count_Network_SubqueryCount(t *testing.T) {
	var gotSQL string
	db := &stubDB{}
	db.queryRowFn = func(ctx context.Context, sql string, args ...any) pgx.Row {
		gotSQL = sql
		return countRow{n: 42}
	}
	a := New(db, nil, nil)
	n, err := a.Count(context.Background(), QueryRequest{
		Granularity: metrics.GranularityHourly,
		Dimension:   DimensionNetwork,
		Limit:       5000,
	})
	require.NoError(t, err)
	assert.Equal(t, 42, n)
	assert.Contains(t, gotSQL, "SELECT COUNT(*) FROM (")
	assert.Contains(t, gotSQL, "GROUP BY metric_path, granularity, time")
}

func Test_DiscoverObjectLDNs_IgnoresRequestedMetricPath(t *testing.T) {
	var gotSQL string
	var gotArgs []any
	db := &stubDB{}
	db.queryFn = func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
		gotSQL = sql
		gotArgs = args
		return &fakeRows{rows: [][]any{{"Cellid=1"}, {"Cellid=2"}}}, nil
	}
	a := New(db, nil, nil)
	start := time.Date(2026, 7, 14, 14, 45, 0, 0, time.UTC)
	visibleGroup := uuid.New()

	ldns, err := a.DiscoverObjectLDNs(context.Background(), QueryRequest{
		Granularity:   metrics.Granularity15Min,
		Dimension:     DimensionDevice,
		DeviceSNs:     []string{"SN-1"},
		Technologies:  []string{"lte"},
		MetricPaths:   []string{"K-MISSING"},
		StartTime:     start,
		EndTime:       start.Add(3 * time.Hour),
		VisibleGroups: []uuid.UUID{visibleGroup},
		Limit:         5000,
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"Cellid=1", "Cellid=2"}, ldns)
	assert.Contains(t, gotSQL, "SELECT DISTINCT object_ldn")
	assert.Contains(t, gotSQL, "FROM pm_measurement_anchors a")
	assert.Contains(t, gotSQL, "JOIN device_dim dev ON dev.id = a.device_dim_id")
	assert.Contains(t, gotSQL, "a.device_dim_id IN (SELECT id FROM device_dim")
	assert.Contains(t, gotSQL, "object_ldn <> ''")
	assert.Contains(t, gotSQL, "device_sn")
	assert.Contains(t, gotSQL, "technology")
	assert.Contains(t, gotSQL, "device_group_members")
	assert.Contains(t, gotSQL, "time >=")
	assert.Contains(t, gotSQL, "time <")
	assert.Contains(t, gotSQL, "ORDER BY object_ldn")
	assert.NotContains(t, gotSQL, "FROM pm_metrics ")
	assert.NotContains(t, gotSQL, "pm_files")
	assert.NotContains(t, gotSQL, "pm_metric_values")
	assert.NotContains(t, gotSQL, "metric_path")
	assert.NotContains(t, gotSQL, "LIMIT")
	assert.Contains(t, gotArgs, []string{"SN-1"})
	assert.Contains(t, gotArgs, []string{"lte"})
	assert.Contains(t, gotArgs, visibleGroup)
	assert.Contains(t, gotArgs, start)
	assert.Contains(t, gotArgs, start.Add(3*time.Hour))
}

func Test_DiscoverObjectLDNs_RolledUpReadsAggregationResults(t *testing.T) {
	var gotSQL string
	var gotArgs []any
	db := &stubDB{}
	db.queryFn = func(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
		gotSQL = sql
		gotArgs = args
		return &fakeRows{rows: [][]any{{"Cellid=1"}}}, nil
	}
	a := New(db, nil, nil)
	start := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)

	ldns, err := a.DiscoverObjectLDNs(context.Background(), QueryRequest{
		Granularity:  metrics.GranularityDaily,
		Dimension:    DimensionDevice,
		DeviceSNs:    []string{"SN-1"},
		Technologies: []string{"lte"},
		MetricPaths:  []string{"K001"},
		StartTime:    start,
		EndTime:      start.Add(7 * 24 * time.Hour),
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"Cellid=1"}, ldns)
	assert.Contains(t, gotSQL, "FROM pm_aggregation_results r")
	assert.Contains(t, gotSQL, "dimension =")
	assert.Contains(t, gotSQL, "r.dimension_key IN (SELECT id::text FROM device_dim")
	assert.Contains(t, gotSQL, "granularity =")
	assert.Contains(t, gotSQL, "window_start")
	assert.Contains(t, gotSQL, "device_sn")
	assert.Contains(t, gotSQL, "technology")
	assert.NotContains(t, gotSQL, "pm_metrics_daily")
	assert.NotContains(t, gotSQL, "metric_path")
	assert.Contains(t, gotArgs, "device")
	assert.Contains(t, gotArgs, []string{"SN-1"})
	assert.Contains(t, gotArgs, []string{"lte"})
	assert.Contains(t, gotArgs, string(metrics.GranularityDaily))
	assert.Contains(t, gotArgs, start)
	assert.Contains(t, gotArgs, start.Add(7*24*time.Hour))
}
