package aggregator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

// T-0182：band 维度本任务仅入枚举，聚合实现拆到 T-0183 → SelectTable 必须返 ErrBandNotImplemented。
func Test_SelectTable_BandNotImplemented(t *testing.T) {
	for _, g := range []metrics.Granularity{
		metrics.Granularity15Min, metrics.GranularityHourly, metrics.GranularityDaily,
		metrics.GranularityWeekly, metrics.GranularityMonthly,
	} {
		t.Run(string(g), func(t *testing.T) {
			got, err := SelectTable(g, DimensionBand)
			assert.ErrorIs(t, err, ErrBandNotImplemented)
			assert.Empty(t, got)
		})
	}
}
