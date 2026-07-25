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
	assert.Equal(t, 1, args[len(args)-1])
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
