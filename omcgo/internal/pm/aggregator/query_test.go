package aggregator

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

func TestRawAwareDeviceSelectUsesCorrectPhysicalSource(t *testing.T) {
	req := QueryRequest{
		Granularity: metrics.Granularity15Min,
		MetricPaths: []string{"C1"},
	}
	rawSQL, _, err := newRawAwareDeviceSelect(
		storage.Psql, "pm_metrics", req, deviceTableColumns...,
	).ToSql()
	require.NoError(t, err)
	assert.Contains(t, rawSQL, "FROM pm_measurement_anchors")
	assert.NotContains(t, rawSQL, "FROM pm_hourly_bucket_versions")
	assert.NotContains(t, rawSQL, "FROM pm_hourly_anchors")
	assert.NotContains(t, rawSQL, "FROM pm_hourly_values")

	rolledUp := []struct {
		granularity metrics.Granularity
		table       string
	}{
		{metrics.GranularityHourly, "pm_metrics_hourly"},
		{metrics.GranularityDaily, "pm_metrics_daily"},
		{metrics.GranularityWeekly, "pm_metrics_weekly"},
		{metrics.GranularityMonthly, "pm_metrics_monthly"},
	}
	for _, tc := range rolledUp {
		t.Run(string(tc.granularity), func(t *testing.T) {
			rolledSQL, args, err := newRawAwareDeviceSelect(
				storage.Psql,
				tc.table,
				QueryRequest{
					Granularity: tc.granularity,
					DeviceSNs:   []string{"SN-1"},
					MetricPaths: []string{"K1"},
				},
				deviceTableColumns...,
			).ToSql()
			require.NoError(t, err)
			assert.Contains(t, rolledSQL, "FROM pm_aggregation_results r")
			assert.NotContains(t, rolledSQL, "FROM "+tc.table)
			assert.Contains(t, rolledSQL, "r.dimension =")
			assert.Contains(t, rolledSQL, "r.granularity =")
			assert.Contains(t, rolledSQL, "r.dimension_key IN (SELECT id::text FROM device_dim")
			assert.Contains(t, args, string(tc.granularity))
			assert.Contains(t, args, []string{"SN-1"})
		})
	}

	for _, dimension := range []Dimension{DimensionAggregateGroup, DimensionNetwork} {
		t.Run(string(dimension)+"_keeps_shared_view", func(t *testing.T) {
			otherSQL, _, err := newRawAwareDeviceSelect(
				storage.Psql,
				"pm_metrics_hourly",
				QueryRequest{
					Granularity: metrics.GranularityHourly,
					Dimension:   dimension,
					DeviceSNs:   []string{"SN-1"},
				},
				deviceTableColumns...,
			).ToSql()
			require.NoError(t, err)
			assert.Contains(t, otherSQL, "FROM pm_metrics_hourly")
			assert.NotContains(t, otherSQL, "FROM pm_aggregation_results")
		})
	}

	t.Run("table_name_enforces_granularity", func(t *testing.T) {
		rolledSQL, args, err := newRawAwareDeviceSelect(
			storage.Psql,
			"pm_metrics_hourly",
			QueryRequest{DeviceSNs: []string{"SN-1"}},
			deviceTableColumns...,
		).ToSql()
		require.NoError(t, err)
		assert.Contains(t, rolledSQL, "r.granularity =")
		assert.Contains(t, args, string(metrics.GranularityHourly))
	})
}

func Test_buildDeviceTableSQL_RolledUpFiltersBeforeDedupAndPreservesProjection(t *testing.T) {
	metricType := metrics.MetricTypeKPI
	visibleGroup := uuid.MustParse("49adf511-d82a-4552-a040-1320e5c32ac5")
	start := time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 28, 16, 0, 0, 0, time.UTC)

	sql, args, err := buildDeviceTableSQL("pm_metrics_hourly", QueryRequest{
		Granularity: metrics.GranularityHourly,
		DeviceOUIs:  []string{"0019C0"},
		DeviceSNs:   []string{"120200024118AA01241"},
		Technologies: []string{
			"lte",
		},
		MetricPaths:   []string{"K900010002"},
		MetricType:    &metricType,
		ObjectLDNs:    []string{"Cellid=1,PLMN=46000"},
		StartTime:     start,
		EndTime:       end,
		VisibleGroups: []uuid.UUID{visibleGroup},
	})
	require.NoError(t, err)

	assert.Contains(t, sql, "FROM pm_aggregation_results r")
	assert.NotContains(t, sql, "FROM pm_metrics_hourly")
	assert.Contains(t, sql, "r.dimension_key IN (SELECT id::text FROM device_dim")
	assert.Contains(t, sql, "r.device_oui AS device_oui")
	assert.Contains(t, sql, "r.device_sn AS device_sn")
	assert.Contains(t, sql, "r.aggregation_op::text AS statis_type")
	assert.Contains(t, sql, `r.window_start AS "time"`)
	assert.Contains(t, sql, "r.created_at AS ingest_time")
	assert.Contains(t, sql, "r.object_ldn AS object_ldn")
	assert.Contains(t, sql, "jsonb_build_object")
	assert.Contains(t, sql, `DISTINCT ON (device_oui, device_sn, metric_path, granularity, "time", object_ldn)`)
	assert.Contains(t, sql, "metric_path IN")
	assert.Contains(t, sql, "time >=")
	assert.Contains(t, sql, "time <")
	assert.Contains(t, sql, "object_ldn IN")
	assert.Contains(t, sql, "device_group_members")
	assert.Contains(t, args, "0019C0")
	assert.Contains(t, args, []string{"120200024118AA01241"})
	assert.Contains(t, args, []string{"lte"})
	assert.Contains(t, args, "K900010002")
	assert.Contains(t, args, start)
	assert.Contains(t, args, end)
	assert.Contains(t, args, "Cellid=1,PLMN=46000")
	assert.Contains(t, args, visibleGroup)
}

func Test_applyScalarFilters_TimeRangeIsHalfOpen(t *testing.T) {
	start := time.Date(2026, 7, 6, 16, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 7, 16, 0, 0, 0, time.UTC)
	qb := storage.Psql.Select("metric_path").From("pm_metrics_daily")

	sql, args, err := applyScalarFilters(qb, QueryRequest{StartTime: start, EndTime: end}).ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "time >= $1")
	assert.Contains(t, sql, "time < $2")
	assert.NotContains(t, sql, "time <= $2")
	assert.Equal(t, []any{start, end}, args)
}

func Test_applyScalarFilters_CalendarFiltersUseConfiguredTimezone(t *testing.T) {
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	require.NoError(t, err)
	localMidnight := time.Date(2026, 7, 28, 0, 0, 0, 0, shanghai)
	require.Equal(t, time.Tuesday, localMidnight.Weekday())
	require.Equal(t, time.Monday, localMidnight.UTC().Weekday(), "UTC would misclassify this bucket as Monday")

	qb := storage.Psql.Select("metric_path").From("pm_metrics_hourly")
	sql, args, err := applyScalarFilters(qb, QueryRequest{
		StartTime:        localMidnight,
		EndTime:          localMidnight.Add(time.Hour),
		Weekdays:         []int{2},
		Hours:            []int{0},
		CalendarTimezone: shanghai.String(),
	}).ToSql()
	require.NoError(t, err)

	assert.Contains(t, sql, "EXTRACT(dow FROM (start_time AT TIME ZONE $3))::int = ANY($4)")
	assert.Contains(t, sql, "EXTRACT(hour FROM (start_time AT TIME ZONE $5))::int = ANY($6)")
	assert.NotContains(t, sql, "EXTRACT(dow FROM start_time)")
	assert.NotContains(t, sql, "EXTRACT(hour FROM start_time)")
	assert.Equal(t, []any{
		localMidnight,
		localMidnight.Add(time.Hour),
		"Asia/Shanghai",
		[]int{2},
		"Asia/Shanghai",
		[]int{0},
	}, args)
}

func Test_queryProductTable_TimeRangeIsHalfOpen(t *testing.T) {
	start := time.Date(2026, 7, 13, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	db := &recordingDB{results: []pgx.Rows{&fakeRows{}}}
	a := New(db, nil, nil)

	_, err := a.queryProductTable(context.Background(), "pm_metrics_hourly", QueryRequest{
		Granularity: metrics.GranularityHourly,
		StartTime:   start,
		EndTime:     end,
	})
	require.NoError(t, err)
	require.Len(t, db.sqls, 1)
	assert.Contains(t, db.sqls[0], "m.time >= $2")
	assert.Contains(t, db.sqls[0], "m.time < $3")
	assert.NotContains(t, db.sqls[0], "m.time <= $3")
	assert.Equal(t, []any{"hourly", start, end}, db.argsLog[0])
}

func Test_queryBandTable_TimeRangeIsHalfOpen(t *testing.T) {
	start := time.Date(2026, 7, 13, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 13, 10, 0, 0, 0, time.UTC)
	db := &recordingDB{results: []pgx.Rows{&fakeRows{}}}
	a := New(db, nil, nil)

	_, err := a.queryBandTable(context.Background(), "pm_metrics_hourly", QueryRequest{
		Granularity: metrics.GranularityHourly,
		StartTime:   start,
		EndTime:     end,
	})
	require.NoError(t, err)
	require.Len(t, db.sqls, 1)
	assert.Contains(t, db.sqls[0], "m.time >= $2")
	assert.Contains(t, db.sqls[0], "m.time < $3")
	assert.NotContains(t, db.sqls[0], "m.time <= $3")
	assert.Equal(t, []any{"hourly", start, end}, db.argsLog[0])
}

// SelectTable 10 case：5 粒度 × 2 维度。15min × device_group 必须返 ErrUnsupportedQuery。

func Test_SelectTable_5Granularities_x_2Dimensions(t *testing.T) {
	cases := []struct {
		gran    metrics.Granularity
		dim     Dimension
		want    string
		wantErr bool
	}{
		{metrics.Granularity15Min, DimensionDevice, "pm_metrics", false},
		{metrics.Granularity15Min, DimensionDeviceGroup, "", true}, // 不支持
		{metrics.GranularityHourly, DimensionDevice, "pm_metrics_hourly", false},
		{metrics.GranularityHourly, DimensionDeviceGroup, "pm_group_metrics_hourly", false},
		{metrics.GranularityDaily, DimensionDevice, "pm_metrics_daily", false},
		{metrics.GranularityDaily, DimensionDeviceGroup, "pm_group_metrics_daily", false},
		{metrics.GranularityWeekly, DimensionDevice, "pm_metrics_weekly", false},
		{metrics.GranularityWeekly, DimensionDeviceGroup, "pm_group_metrics_weekly", false},
		{metrics.GranularityMonthly, DimensionDevice, "pm_metrics_monthly", false},
		{metrics.GranularityMonthly, DimensionDeviceGroup, "pm_group_metrics_monthly", false},
	}
	for _, c := range cases {
		t.Run(string(c.gran)+"_"+string(c.dim), func(t *testing.T) {
			got, err := SelectTable(c.gran, c.dim)
			if c.wantErr {
				assert.ErrorIs(t, err, ErrUnsupportedQuery)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

func Test_SelectTable_EmptyDimDefaultsDevice(t *testing.T) {
	got, err := SelectTable(metrics.GranularityHourly, "")
	require.NoError(t, err)
	assert.Equal(t, "pm_metrics_hourly", got)
}

// #208 同窗口多文件去重：device 维度直读必须用 DISTINCT ON 折叠同键、保留 ingest_time 最新一条，
// 否则同设备同窗口的两个不同文件名 PM 文件各落一行被前端 SUM 成翻倍值。
func Test_buildDeviceTableSQL_DistinctOnDedupLatestIngest(t *testing.T) {
	sql, args, err := buildDeviceTableSQL("pm_metrics", QueryRequest{
		DeviceOUIs: []string{"0019C0"},
		DeviceSNs:  []string{"SN-1"},
		Limit:      100,
		Offset:     0,
	})
	require.NoError(t, err)

	// 内层去重：DISTINCT ON 键 + ingest_time DESC 选最新文件。
	assert.Contains(t, sql, `DISTINCT ON (device_oui, device_sn, metric_path, granularity, "time", object_ldn)`)
	assert.Contains(t, sql, `ORDER BY device_oui, device_sn, metric_path, granularity, "time", object_ldn, ingest_time DESC`)
	// 外层包子查询恢复 time DESC + Limit 语义。
	assert.Contains(t, sql, "FROM (SELECT DISTINCT ON")
	assert.Contains(t, sql, ") AS d ORDER BY time DESC")
	assert.Contains(t, sql, "LIMIT 100")
	// WHERE / 参数绑定不变：device 过滤参数仍在。
	assert.Contains(t, args, "0019C0")
	assert.Contains(t, args, "SN-1")
}

// 单设备单文件场景（无重复）不应被去重逻辑改变行为：SQL 仍是同一套去重查询，
// 单文件下 DISTINCT ON 对每个唯一键只有一行，结果零变化。
func Test_buildDeviceTableSQL_PreservesWhereFilters(t *testing.T) {
	mt := metrics.MetricTypeKPI
	sql, _, err := buildDeviceTableSQL("pm_metrics", QueryRequest{
		DeviceSNs:   []string{"SN-1"},
		MetricPaths: []string{"K900010015"},
		MetricType:  &mt,
		Granularity: metrics.Granularity15Min,
	})
	require.NoError(t, err)
	// 过滤条件全部落在内层子查询里。
	assert.Contains(t, sql, "device_sn IN")
	assert.Contains(t, sql, "metric_path IN")
	assert.Contains(t, sql, "metric_type =")
	assert.Contains(t, sql, "granularity =")
}

func Test_buildDeviceTableSQL_BindsMetricPathToInferredMetricTypeWhenTypeAbsent(t *testing.T) {
	sql, args, err := buildDeviceTableSQL("pm_metrics", QueryRequest{
		DeviceSNs:   []string{"SN-1"},
		MetricPaths: []string{" K900010015 ", "C000060216"},
		Granularity: metrics.Granularity15Min,
	})
	require.NoError(t, err)

	assert.Contains(t, sql, "metric_path =")
	assert.Contains(t, sql, "metric_type =")
	assert.Contains(t, sql, " OR ")
	assert.Contains(t, args, "K900010015")
	assert.Contains(t, args, "kpi")
	assert.Contains(t, args, "C000060216")
	assert.Contains(t, args, "counter")
}

func Test_buildDeviceTableSQL_PageByPivotRowPagesKeysThenReturnsAllMetrics(t *testing.T) {
	sql, args, err := buildDeviceTableSQL("pm_metrics", QueryRequest{
		Granularity:    metrics.Granularity15Min,
		DeviceSNs:      []string{"SN-1", "SN-2"},
		MetricPaths:    []string{"K1", "K2"},
		PageByPivotRow: true,
		Limit:          50,
		Offset:         100,
	})
	require.NoError(t, err)

	assert.Contains(t, sql, "WITH dedup AS")
	assert.Contains(t, sql, "page_keys AS")
	assert.Contains(t, sql, "SELECT DISTINCT device_oui, device_sn, COALESCE(object_ldn, '') AS object_ldn, granularity, \"time\"")
	assert.Contains(t, sql, "JOIN page_keys pk")
	assert.Contains(t, sql, "AND pk.\"time\" = d.\"time\"")
	assert.Contains(t, sql, "ORDER BY \"time\" DESC, device_sn ASC, object_ldn ASC")
	assert.Contains(t, sql, "ORDER BY d.\"time\" DESC, d.device_sn ASC, COALESCE(d.object_ldn, '') ASC, d.metric_path ASC")
	assert.Contains(t, sql, "LIMIT $")
	assert.Contains(t, sql, "OFFSET $")
	assert.Equal(t, 50, args[len(args)-2])
	assert.Equal(t, 100, args[len(args)-1])
}

func Test_buildDeviceTableSQL_PageByPivotRowSkeletonPagesKeysWithoutMetricFilter(t *testing.T) {
	metricType := metrics.MetricTypeKPI
	sql, args, err := buildDeviceTableSQL("pm_metrics", QueryRequest{
		Granularity:    metrics.Granularity15Min,
		DeviceSNs:      []string{"SN-1"},
		MetricPaths:    []string{"K1"},
		MetricType:     &metricType,
		ObjectLDNs:     []string{"Cellid=1,PLMN=46000"},
		StartTime:      time.Date(2026, 7, 16, 11, 0, 0, 0, time.UTC),
		EndTime:        time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC),
		PageByPivotRow: true,
		Limit:          1,
	})
	require.NoError(t, err)

	pageKeysAt := strings.Index(sql, "page_keys AS")
	require.NotEqual(t, -1, pageKeysAt)
	pageKeySQL := sql[pageKeysAt:]
	assert.NotContains(t, pageKeySQL, "metric_path IN")
	assert.NotContains(t, pageKeySQL, "metric_type =")
	assert.Contains(t, pageKeySQL, "FROM pm_measurement_anchors a")
	assert.Contains(t, pageKeySQL, "JOIN device_dim dev ON dev.id = a.device_dim_id")
	assert.NotContains(t, pageKeySQL, "FROM pm_metrics ")
	assert.NotContains(t, pageKeySQL, "pm_metric_dictionary")
	assert.NotContains(t, pageKeySQL, "pm_metric_values")
	assert.Equal(t, 1, args[len(args)-1])
}

func Test_buildDevicePivotRowKeysSQL_ExplicitObjectUsesRawAnchorPredicates(t *testing.T) {
	sql, args, err := buildDevicePivotRowKeysSQL("pm_metrics", QueryRequest{
		Granularity:    metrics.Granularity15Min,
		DeviceSNs:      []string{"SN-1"},
		MetricPaths:    []string{"K1"},
		Technologies:   []string{"lte"},
		ObjectLDNs:     []string{"Cellid=66"},
		StartTime:      time.Date(2026, 7, 23, 0, 0, 0, 0, time.UTC),
		EndTime:        time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC),
		PageByPivotRow: true,
		Limit:          5000,
	})
	require.NoError(t, err)

	assert.Contains(t, sql, "WITH target_devices AS")
	assert.Contains(t, sql, "anchor_keys AS")
	assert.Contains(t, sql, "FROM pm_measurement_anchors a")
	assert.Contains(t, sql, "JOIN target_devices dev ON dev.id = a.device_dim_id")
	assert.Contains(t, sql, "a.granularity =")
	assert.Contains(t, sql, "a.\"time\" >= ")
	assert.Contains(t, sql, "a.\"time\" < ")
	assert.Contains(t, sql, "a.object_ldn IN")
	assert.Contains(t, sql, "$7::text AS granularity")
	assert.Contains(t, sql, "ORDER BY k.\"time\" DESC")
	assert.Contains(t, sql, "LIMIT $8")
	assert.NotContains(t, sql, "object_source")
	assert.NotContains(t, sql, "pm_metric_dictionary")
	assert.NotContains(t, sql, "pm_metric_values")
	assert.Equal(t, []any{
		"SN-1",
		"lte",
		"15min",
		time.Date(2026, 7, 23, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC),
		"Cellid=66",
		"15min",
		5000,
	}, args)
}

func Test_DevicePivotRowKeys_ExplicitObjectSkeletonGeneratesBucketsWithoutAnchorScan(t *testing.T) {
	start := time.Date(2026, 7, 23, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	db := &recordingDB{results: []pgx.Rows{&fakeRows{rows: [][]any{{"0019C0", "SN-1"}}}}}
	a := New(db, nil, nil)

	keys, err := a.DevicePivotRowKeys(context.Background(), QueryRequest{
		Granularity:    metrics.Granularity15Min,
		DeviceSNs:      []string{"SN-1"},
		MetricPaths:    []string{"K1"},
		Technologies:   []string{"lte"},
		ObjectLDNs:     []string{"Cellid=2", "Cellid=1"},
		StartTime:      start,
		EndTime:        end,
		PageByPivotRow: true,
		Limit:          3,
		Offset:         1,
	})
	require.NoError(t, err)
	require.Len(t, keys, 3)

	assert.Len(t, db.sqls, 1)
	assert.Contains(t, db.sqls[0], "FROM device_dim dev")
	assert.Contains(t, db.sqls[0], "dev.serial_number IN")
	assert.Contains(t, db.sqls[0], "dev.technology IN")
	assert.NotContains(t, db.sqls[0], "pm_measurement_anchors")
	assert.Equal(t, []any{"SN-1", "lte"}, db.argsLog[0])
	assert.Equal(t, PivotRowKey{
		DeviceOUI:   "0019C0",
		DeviceSN:    "SN-1",
		ObjectLDN:   "Cellid=2",
		Granularity: metrics.Granularity15Min,
		Time:        start.Add(45 * time.Minute),
	}, keys[0])
	assert.Equal(t, "Cellid=1", keys[1].ObjectLDN)
	assert.Equal(t, start.Add(30*time.Minute), keys[1].Time)
	assert.Equal(t, "Cellid=2", keys[2].ObjectLDN)
	assert.Equal(t, start.Add(30*time.Minute), keys[2].Time)
}

func Test_DevicePivotRowKeys_ExplicitObjectSkeletonAppliesCalendarFilters(t *testing.T) {
	start := time.Date(2026, 7, 23, 0, 0, 0, 0, time.UTC)
	end := start.Add(3 * time.Hour)
	db := &recordingDB{results: []pgx.Rows{&fakeRows{rows: [][]any{{"0019C0", "SN-1"}}}}}
	a := New(db, nil, nil)

	keys, err := a.DevicePivotRowKeys(context.Background(), QueryRequest{
		Granularity:      metrics.Granularity15Min,
		DeviceSNs:        []string{"SN-1"},
		MetricPaths:      []string{"K1"},
		ObjectLDNs:       []string{"Cellid=1"},
		StartTime:        start,
		EndTime:          end,
		Weekdays:         []int{4},
		Hours:            []int{1},
		CalendarTimezone: "UTC",
		PageByPivotRow:   true,
	})
	require.NoError(t, err)
	require.Len(t, keys, 4)
	for _, key := range keys {
		assert.Equal(t, time.Thursday, key.Time.Weekday())
		assert.Equal(t, 1, key.Time.Hour())
	}
	assert.Equal(t, start.Add(105*time.Minute), keys[0].Time)
	assert.Equal(t, start.Add(60*time.Minute), keys[3].Time)
}

func Test_IsExplicitObjectSkeletonRequest_DeviceDimensionOnly(t *testing.T) {
	base := QueryRequest{
		Granularity: metrics.Granularity15Min,
		DeviceSNs:   []string{"SN-1"},
		MetricPaths: []string{"K1"},
		ObjectLDNs:  []string{"Cellid=1"},
		StartTime:   time.Date(2026, 7, 23, 0, 0, 0, 0, time.UTC),
		EndTime:     time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC),
	}

	for _, dimension := range []Dimension{"", DimensionDevice} {
		req := base
		req.Dimension = dimension
		assert.True(t, IsExplicitObjectSkeletonRequest(req), "dimension %q should use device object skeleton", dimension)
	}
	for _, dimension := range []Dimension{
		DimensionDeviceGroup,
		DimensionAggregateGroup,
		DimensionProduct,
		DimensionBand,
		DimensionNetwork,
	} {
		req := base
		req.Dimension = dimension
		assert.False(t, IsExplicitObjectSkeletonRequest(req), "dimension %q should keep its own query path", dimension)
	}
}

func Test_buildRawDevicePivotRowKeysSQL_PreservesOUIAndVisibilityFilters(t *testing.T) {
	visibleGroup := uuid.MustParse("49adf511-d82a-4552-a040-1320e5c32ac5")
	sql, args, err := buildRawDevicePivotRowKeysSQL(QueryRequest{
		Granularity:   metrics.Granularity15Min,
		DeviceOUIs:    []string{"0019C0"},
		DeviceSNs:     []string{"SN-1"},
		MetricPaths:   []string{"K1"},
		ObjectLDNs:    []string{"Cellid=1"},
		StartTime:     time.Date(2026, 7, 23, 0, 0, 0, 0, time.UTC),
		EndTime:       time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC),
		VisibleGroups: []uuid.UUID{visibleGroup},
	})
	require.NoError(t, err)

	assert.Contains(t, sql, "FROM device_dim dev")
	assert.Contains(t, sql, "(dev.oui = $1 AND dev.serial_number = $2)")
	assert.Contains(t, sql, "device_group_members")
	assert.Contains(t, sql, "JOIN target_devices dev ON dev.id = a.device_dim_id")
	assert.Contains(t, args, "0019C0")
	assert.Contains(t, args, "SN-1")
	assert.Contains(t, args, visibleGroup)
	assert.Contains(t, args, "Cellid=1")
}

func Test_buildRawDevicePivotRowKeysSQL_CalendarFiltersUseHourArgs(t *testing.T) {
	sql, args, err := buildRawDevicePivotRowKeysSQL(QueryRequest{
		Granularity:      metrics.Granularity15Min,
		DeviceSNs:        []string{"SN-1"},
		Weekdays:         []int{2},
		Hours:            []int{0},
		CalendarTimezone: "Asia/Shanghai",
	})
	require.NoError(t, err)

	assert.Contains(t, sql, "EXTRACT(dow FROM (a.start_time AT TIME ZONE $3))::int = ANY($4)")
	assert.Contains(t, sql, "EXTRACT(hour FROM (a.start_time AT TIME ZONE $5))::int = ANY($6)")
	assert.Equal(t, []any{
		"SN-1",
		"15min",
		"Asia/Shanghai",
		[]int{2},
		"Asia/Shanghai",
		[]int{0},
		"15min",
	}, args)
}

func Test_buildRawDevicePivotRowKeysSQL_DeviceOUIAndSNFiltersStayPaired(t *testing.T) {
	sql, args, err := buildRawDevicePivotRowKeysSQL(QueryRequest{
		Granularity: metrics.Granularity15Min,
		DeviceOUIs:  []string{"OUI-A", "OUI-B"},
		DeviceSNs:   []string{"SN-A", "SN-B"},
	})
	require.NoError(t, err)

	assert.Contains(t, sql, "((dev.oui = $1 AND dev.serial_number = $2) OR (dev.oui = $3 AND dev.serial_number = $4))")
	assert.NotContains(t, sql, "dev.oui IN")
	assert.NotContains(t, sql, "dev.serial_number IN")
	assert.Equal(t, []any{
		"OUI-A",
		"SN-A",
		"OUI-B",
		"SN-B",
		"15min",
		"15min",
	}, args)
}

func Test_buildDeviceTableSQL_PageByPivotRowUsesPrecomputedKeys(t *testing.T) {
	keyTime := time.Date(2026, 7, 23, 8, 30, 0, 0, time.UTC)
	sql, args, err := buildDeviceTableSQL("pm_metrics", QueryRequest{
		Granularity:    metrics.Granularity15Min,
		DeviceSNs:      []string{"SN-1"},
		MetricPaths:    []string{"K1", "K2"},
		PageByPivotRow: true,
		Limit:          5000,
		PivotRowKeys: []PivotRowKey{{
			DeviceOUI:   "0019C0",
			DeviceSN:    "SN-1",
			ObjectLDN:   "Cellid=1",
			Granularity: metrics.Granularity15Min,
			Time:        keyTime,
		}},
	})
	require.NoError(t, err)

	assert.Contains(t, sql, "WITH requested_keys(device_oui, device_sn, object_ldn, granularity, \"time\") AS")
	assert.Contains(t, sql, "VALUES ($1::text,$2::text,$3::text,$4::text,$5::timestamptz)")
	assert.Contains(t, sql, "FROM requested_keys pk")
	assert.Contains(t, sql, "JOIN pm_measurement_anchors a")
	assert.Contains(t, sql, "AND a.object_ldn = pk.object_ldn")
	assert.Contains(t, sql, "AND a.\"time\" = pk.\"time\"")
	assert.Contains(t, sql, "JOIN pm_metric_sets s ON s.metric_set_id=a.metric_set_id")
	assert.Contains(t, sql, "JOIN pm_metric_dictionary d ON d.metric_id=ANY(s.metric_ids)")
	assert.Contains(t, sql, "LEFT JOIN pm_metric_values v")
	assert.NotContains(t, sql, "requested_metrics")
	assert.NotContains(t, sql, "pm_files")
	assert.NotContains(t, sql, "pm_ingest_batches")
	assert.Contains(t, sql, "ingest_sequence DESC")
	assert.Contains(t, sql, "d.metric_path =")
	assert.Contains(t, sql, "d.metric_type =")
	assert.NotContains(t, sql, "page_keys AS")
	assert.NotContains(t, sql, "LIMIT")
	assert.NotContains(t, sql, "OFFSET")
	assert.Equal(t, "0019C0", args[0])
	assert.Equal(t, "SN-1", args[1])
	assert.Equal(t, "Cellid=1", args[2])
	assert.Equal(t, "15min", args[3])
	assert.Equal(t, keyTime, args[4])
	assert.Equal(t, "K1", args[5])
	assert.Equal(t, "kpi", args[6])
	assert.Equal(t, "K2", args[7])
	assert.Equal(t, "kpi", args[8])
	assert.Equal(t, "SN-1", args[9])
}

func Test_buildDeviceTableSQL_PageByPivotRowExplicitSkeletonUsesLatestAnchorsAndRequestedMetrics(t *testing.T) {
	keyTime := time.Date(2026, 7, 23, 8, 30, 0, 0, time.UTC)
	start := time.Date(2026, 7, 23, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)
	sql, args, err := buildDeviceTableSQL("pm_metrics", QueryRequest{
		Granularity:    metrics.Granularity15Min,
		DeviceSNs:      []string{"SN-1"},
		MetricPaths:    []string{"K1", "K2"},
		ObjectLDNs:     []string{"Cellid=1"},
		StartTime:      start,
		EndTime:        end,
		PageByPivotRow: true,
		Limit:          5000,
		PivotRowKeys: []PivotRowKey{{
			DeviceOUI:   "0019C0",
			DeviceSN:    "SN-1",
			ObjectLDN:   "Cellid=1",
			Granularity: metrics.Granularity15Min,
			Time:        keyTime,
		}},
	})
	require.NoError(t, err)

	assert.Contains(t, sql, "latest_anchors AS")
	assert.Contains(t, sql, "requested_metrics AS")
	assert.Contains(t, sql, "DISTINCT ON (dev.oui, dev.serial_number, a.granularity, a.\"time\", a.object_ldn)")
	assert.Contains(t, sql, "AND a.granularity = '15min'")
	assert.NotContains(t, sql, "a.granularity = pk.granularity")
	assert.Contains(t, sql, "JOIN requested_metrics d ON TRUE")
	assert.Contains(t, sql, "LEFT JOIN pm_metric_values v")
	assert.Contains(t, sql, `v."time" >= $`)
	assert.Contains(t, sql, `v."time" < $`)
	assert.NotContains(t, sql, "pm_metric_sets")
	assert.NotContains(t, sql, "d.metric_id=ANY")
	assert.NotContains(t, sql, "pm_files")
	assert.NotContains(t, sql, "pm_ingest_batches")
	assert.Equal(t, "0019C0", args[0])
	assert.Equal(t, "SN-1", args[1])
	assert.Equal(t, "Cellid=1", args[2])
	assert.Equal(t, "15min", args[3])
	assert.Equal(t, keyTime, args[4])
	assert.Contains(t, args, "K1")
	assert.Contains(t, args, "K2")
	assert.Contains(t, args, start)
	assert.Contains(t, args, end)
}

func Test_buildDeviceTableSQL_PageByPivotRowPrecomputedKeysPrunesMetricValueTime(t *testing.T) {
	keyTime := time.Date(2026, 7, 23, 8, 30, 0, 0, time.UTC)
	start := time.Date(2026, 7, 23, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 7, 24, 0, 0, 0, 0, time.UTC)
	sql, args, err := buildDeviceTableSQL("pm_metrics", QueryRequest{
		Granularity:    metrics.Granularity15Min,
		DeviceSNs:      []string{"SN-1"},
		MetricPaths:    []string{"K1"},
		PageByPivotRow: true,
		StartTime:      start,
		EndTime:        end,
		PivotRowKeys: []PivotRowKey{{
			DeviceOUI:   "0019C0",
			DeviceSN:    "SN-1",
			ObjectLDN:   "Cellid=1",
			Granularity: metrics.Granularity15Min,
			Time:        keyTime,
		}},
	})
	require.NoError(t, err)

	assert.Contains(t, sql, `LEFT JOIN pm_metric_values v ON v."time"=a."time" AND v.anchor_id=a.anchor_id AND v.metric_id=d.metric_id AND v."time" >= $`)
	assert.Contains(t, sql, `AND v."time" < $`)
	assert.Contains(t, args, start)
	assert.Contains(t, args, end)
}

func Test_buildDeviceTableSQL_RolledUpPivotSkeletonReadsAggregationResults(t *testing.T) {
	sql, _, err := buildDeviceTableSQL("pm_metrics_daily", QueryRequest{
		Granularity:    metrics.GranularityDaily,
		DeviceSNs:      []string{"SN-1"},
		MetricPaths:    []string{"K1"},
		ObjectLDNs:     []string{"Cellid=1,PLMN=46000"},
		StartTime:      time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		EndTime:        time.Date(2026, 7, 8, 0, 0, 0, 0, time.UTC),
		PageByPivotRow: true,
		Limit:          10,
	})
	require.NoError(t, err)

	pageKeysAt := strings.Index(sql, "page_keys AS")
	require.NotEqual(t, -1, pageKeysAt)
	pageKeySQL := sql[pageKeysAt:]
	assert.Contains(t, pageKeySQL, "FROM pm_aggregation_results r")
	assert.Contains(t, pageKeySQL, "dimension =")
	assert.NotContains(t, pageKeySQL, "FROM pm_metrics_daily")
}

func Test_SelectTable_UnknownGranularity(t *testing.T) {
	_, err := SelectTable(metrics.Granularity("xyz"), DimensionDevice)
	assert.Error(t, err)
}

// T-0182：product 维度走 device 维度表（同 pm_metrics*，查询层 JOIN devices GROUP BY product_id）。
func Test_SelectTable_ProductDimension(t *testing.T) {
	cases := []struct {
		gran metrics.Granularity
		want string
	}{
		{metrics.Granularity15Min, "pm_metrics"},
		{metrics.GranularityHourly, "pm_metrics_hourly"},
		{metrics.GranularityDaily, "pm_metrics_daily"},
		{metrics.GranularityWeekly, "pm_metrics_weekly"},
		{metrics.GranularityMonthly, "pm_metrics_monthly"},
	}
	for _, c := range cases {
		t.Run(string(c.gran), func(t *testing.T) {
			got, err := SelectTable(c.gran, DimensionProduct)
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

// T-0183：band 维度走 device 维度同源表（按小区行 JOIN device_parameters 取 band）。
// 实现后 SelectTable 不再返错，路由到 pm_metrics / pm_metrics_{hourly,daily,weekly,monthly}。
func Test_SelectTable_BandDimension(t *testing.T) {
	cases := []struct {
		gran metrics.Granularity
		want string
	}{
		{metrics.Granularity15Min, "pm_metrics"},
		{metrics.GranularityHourly, "pm_metrics_hourly"},
		{metrics.GranularityDaily, "pm_metrics_daily"},
		{metrics.GranularityWeekly, "pm_metrics_weekly"},
		{metrics.GranularityMonthly, "pm_metrics_monthly"},
	}
	for _, c := range cases {
		t.Run(string(c.gran), func(t *testing.T) {
			got, err := SelectTable(c.gran, DimensionBand)
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

// ── 制式过滤分流（设备组制式治本 B 方案）─────────────────────────────────────
//
// 设备组维度：制式过滤走「直接按 technology 列筛」（快表自带 technology 列），
// 绝不引用设备维度表才有的 device_oui/device_sn 子查询（旧路径的报错根）。
func Test_applyGroupFilters_TechnologyDirectColumnFilter(t *testing.T) {
	q := QueryRequest{
		Granularity:  metrics.GranularityHourly,
		MetricPaths:  []string{"C0001"},
		Technologies: []string{"lte"},
	}
	qb := storage.Psql.Select("device_group_id", "technology").From("pm_group_metrics_hourly")
	sql, args, err := applyGroupFilters(qb, q).ToSql()
	require.NoError(t, err)
	// 直接按 technology 列筛
	assert.Contains(t, sql, "technology IN (")
	// 绝不走设备编号子查询（设备组快表无 device_oui/device_sn 列，这正是 bug 根因）
	assert.NotContains(t, sql, "device_oui")
	assert.NotContains(t, sql, "serial_number")
	assert.Contains(t, args, "lte") // squirrel sq.Eq 把切片展开为标量占位参数
}

// 设备维度 / 全网维度：制式过滤仍走「按设备编号子查询 JOIN devices」收口，分流后不受影响。
func Test_applyCommonFilters_TechnologyDeviceSubquery_Unaffected(t *testing.T) {
	q := QueryRequest{
		Granularity:  metrics.GranularityHourly,
		Technologies: []string{"nr"},
	}
	qb := storage.Psql.Select("metric_path").From("pm_metrics_hourly")
	sql, args, err := applyCommonFilters(qb, q).ToSql()
	require.NoError(t, err)
	// 设备维度表仍走设备编号子查询筛制式（跨库分离后改读本库影子表 device_dim）
	assert.Contains(t, sql, "(device_oui, device_sn) IN (SELECT oui, serial_number FROM device_dim WHERE technology = ANY(")
	// 制式作为整切片传 ANY(?) 占位参数
	assert.Contains(t, args, []string{"nr"})
}

// 设备维度（applyDeviceFilters）带 device_sn + technology 时先收窄用户选中的设备集合。
func Test_applyDeviceFilters_TechnologyNarrowsSelectedDeviceSNsFirst(t *testing.T) {
	q := QueryRequest{
		DeviceSNs:    []string{"SN1"},
		Technologies: []string{"lte"},
	}
	qb := storage.Psql.Select("metric_path").From("pm_metrics_hourly")
	sql, args, err := applyDeviceFilters(qb, q).ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "(device_oui, device_sn) IN (SELECT oui, serial_number FROM device_dim WHERE serial_number = ANY(")
	assert.Contains(t, sql, "AND technology = ANY(")
	assert.Contains(t, args, []string{"SN1"})
	assert.Contains(t, args, []string{"lte"})
}

// #64 设备维度可见分组：applyCommonFilters 按 device_sn 两层子查询 fail-closed 收口。
func Test_applyCommonFilters_VisibleGroups_ThreeWay(t *testing.T) {
	g1, g2 := uuid.New(), uuid.New()

	t.Run("nil 超管不过滤", func(t *testing.T) {
		qb := storage.Psql.Select("metric_path").From("pm_metrics_hourly")
		sql, _, err := applyCommonFilters(qb, QueryRequest{}).ToSql()
		require.NoError(t, err)
		assert.NotContains(t, sql, "serial_number")
	})

	t.Run("空集 fail-closed", func(t *testing.T) {
		qb := storage.Psql.Select("metric_path").From("pm_metrics_hourly")
		sql, _, err := applyCommonFilters(qb, QueryRequest{VisibleGroups: []uuid.UUID{}}).ToSql()
		require.NoError(t, err)
		assert.Contains(t, sql, "FALSE")
	})

	t.Run("限定到可见分组下设备", func(t *testing.T) {
		qb := storage.Psql.Select("metric_path").From("pm_metrics_hourly")
		sql, args, err := applyCommonFilters(qb, QueryRequest{VisibleGroups: []uuid.UUID{g1, g2}}).ToSql()
		require.NoError(t, err)
		assert.Contains(t, sql, "device_sn IN (SELECT serial_number FROM devices WHERE id IN (SELECT device_id FROM device_group_members WHERE group_id IN (")
		assert.Contains(t, args, g1)
		assert.Contains(t, args, g2)
	})
}

// #64 设备组维度可见分组：applyGroupFilters 直接对 device_group_id 取交（fail-closed）。
func Test_applyGroupFilters_VisibleGroups_ThreeWay(t *testing.T) {
	g1 := uuid.New()

	t.Run("nil 超管不过滤", func(t *testing.T) {
		qb := storage.Psql.Select("device_group_id").From("pm_group_metrics_hourly")
		sql, _, err := applyGroupFilters(qb, QueryRequest{}).ToSql()
		require.NoError(t, err)
		assert.NotContains(t, sql, "device_group_id IN")
	})

	t.Run("空集 fail-closed", func(t *testing.T) {
		qb := storage.Psql.Select("device_group_id").From("pm_group_metrics_hourly")
		sql, _, err := applyGroupFilters(qb, QueryRequest{VisibleGroups: []uuid.UUID{}}).ToSql()
		require.NoError(t, err)
		assert.Contains(t, sql, "FALSE")
	})

	t.Run("限定到可见分组", func(t *testing.T) {
		qb := storage.Psql.Select("device_group_id").From("pm_group_metrics_hourly")
		sql, args, err := applyGroupFilters(qb, QueryRequest{VisibleGroups: []uuid.UUID{g1}}).ToSql()
		require.NoError(t, err)
		assert.Contains(t, sql, "device_group_id IN (")
		assert.Contains(t, args, g1)
	})
}

// #64 product/band 维度手拼 SQL：appendVisibleSNWhere 把可见分组收口条件与位置参数同步拼入。
func Test_appendVisibleSNWhere_ThreeWay(t *testing.T) {
	g1, g2 := uuid.New(), uuid.New()

	// add 闭包模拟 product/band 的位置参数注册：返回 $N 并收集 args。
	newAdd := func(args *[]any, pos *int) func(any) string {
		return func(v any) string {
			*args = append(*args, v)
			p := "$" + strconv.Itoa(*pos)
			*pos++
			return p
		}
	}

	t.Run("nil 不追加", func(t *testing.T) {
		args := []any{}
		pos := 1
		where := appendVisibleSNWhere(nil, "m.device_sn", nil, newAdd(&args, &pos))
		assert.Empty(t, where)
		assert.Empty(t, args)
	})

	t.Run("空集追加 FALSE 无参数", func(t *testing.T) {
		args := []any{}
		pos := 1
		where := appendVisibleSNWhere(nil, "m.device_sn", []uuid.UUID{}, newAdd(&args, &pos))
		require.Len(t, where, 1)
		assert.Equal(t, "FALSE", where[0])
		assert.Empty(t, args)
	})

	t.Run("限定追加子查询并注册参数", func(t *testing.T) {
		args := []any{}
		pos := 3 // 模拟前面已有 2 个参数
		where := appendVisibleSNWhere(nil, "m.device_sn", []uuid.UUID{g1, g2}, newAdd(&args, &pos))
		require.Len(t, where, 1)
		assert.Equal(t, "m.device_sn IN (SELECT serial_number FROM devices WHERE id IN (SELECT device_id FROM device_group_members WHERE group_id = ANY($3)))", where[0])
		require.Len(t, args, 1)
		assert.Equal(t, []uuid.UUID{g1, g2}, args[0], "可见分组作为单个 ANY 数组参数注册")
	})
}

// #64 设备组维度可见分组 × 请求侧 device_group_id 双 WHERE 叠加 = 交集（请求他组返空）。
func Test_applyGroupFilters_RequestedGroupIntersectsVisible(t *testing.T) {
	visible := uuid.New()
	requested := uuid.New() // 请求一个不在可见集合里的组
	qb := storage.Psql.Select("device_group_id").From("pm_group_metrics_hourly")
	_, args, err := applyGroupFilters(qb, QueryRequest{
		DeviceGroupIDs: []uuid.UUID{requested},
		VisibleGroups:  []uuid.UUID{visible},
	}).ToSql()
	require.NoError(t, err)
	// 两个独立 WHERE：device_group_id = requested AND device_group_id IN (visible)，AND 叠加即交集（空）。
	assert.Contains(t, args, requested)
	assert.Contains(t, args, visible)
}
