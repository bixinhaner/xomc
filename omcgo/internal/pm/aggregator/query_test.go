package aggregator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

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
	// 设备维度表仍走设备编号子查询筛制式（保持原样）
	assert.Contains(t, sql, "(device_oui, device_sn) IN (SELECT oui, serial_number FROM devices WHERE technology = ANY(")
	// 制式作为整切片传 ANY(?) 占位参数
	assert.Contains(t, args, []string{"nr"})
}

// 设备维度（applyDeviceFilters）制式过滤同样走设备编号子查询，分流不破坏既有行为。
func Test_applyDeviceFilters_TechnologyDeviceSubquery_Unaffected(t *testing.T) {
	q := QueryRequest{
		DeviceSNs:    []string{"SN1"},
		Technologies: []string{"lte"},
	}
	qb := storage.Psql.Select("metric_path").From("pm_metrics_hourly")
	sql, _, err := applyDeviceFilters(qb, q).ToSql()
	require.NoError(t, err)
	assert.Contains(t, sql, "(device_oui, device_sn) IN (SELECT oui, serial_number FROM devices WHERE technology = ANY(")
}
