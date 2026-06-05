package export

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/pm/aggregator"
	"github.com/omcgo/omcgo/internal/pm/metrics"
)

func TestParseDashboardParams_Full(t *testing.T) {
	raw := []byte(`{
		"granularity": "hourly",
		"dimension": "device",
		"device_ouis": ["ABCDEF"],
		"device_sns": ["SN1"],
		"metric_paths": ["K001","K002"],
		"metric_type": "kpi",
		"technologies": ["lte"],
		"start_time": "2026-06-01T00:00:00Z",
		"end_time": "2026-06-02T00:00:00Z",
		"object_ldns": ["Cellid=1,PLMN=00101"]
	}`)
	req, objectLDNs, err := parseDashboardParams(raw)
	require.NoError(t, err)
	assert.Equal(t, metrics.GranularityHourly, req.Granularity)
	assert.Equal(t, aggregator.DimensionDevice, req.Dimension)
	assert.Equal(t, []string{"ABCDEF"}, req.DeviceOUIs)
	assert.Equal(t, []string{"SN1"}, req.DeviceSNs)
	assert.Equal(t, []string{"K001", "K002"}, req.MetricPaths)
	require.NotNil(t, req.MetricType)
	assert.Equal(t, metrics.MetricTypeKPI, *req.MetricType)
	assert.Equal(t, []string{"lte"}, req.Technologies)
	assert.False(t, req.StartTime.IsZero())
	assert.False(t, req.EndTime.IsZero())
	assert.Equal(t, 0, req.Limit) // 去 limit 全量
	// A1：object_ldns 白名单单独返回（QueryRequest 无此字段）。
	assert.Equal(t, []string{"Cellid=1,PLMN=00101"}, objectLDNs)
}

// A3：不传 metric_type 时 MetricType 为 nil（不过滤），导出含 counter 与 kpi 两类指标。
func TestParseDashboardParams_NoMetricType(t *testing.T) {
	req, objectLDNs, err := parseDashboardParams([]byte(`{"granularity":"15min","dimension":"device","metric_paths":["K001"]}`))
	require.NoError(t, err)
	assert.Nil(t, req.MetricType, "未传 metric_type → 不过滤类型")
	assert.Empty(t, objectLDNs, "未传 object_ldns → 不过滤小区")
}

func TestParseDashboardParams_MissingGranularity(t *testing.T) {
	_, _, err := parseDashboardParams([]byte(`{"dimension":"device"}`))
	require.Error(t, err)
}

func TestParseDashboardParams_BadJSON(t *testing.T) {
	_, _, err := parseDashboardParams([]byte(`not-json`))
	require.Error(t, err)
}

func TestParseDashboardParams_BadGroupID(t *testing.T) {
	_, _, err := parseDashboardParams([]byte(`{"granularity":"hourly","dimension":"device_group","device_group_ids":["nope"]}`))
	require.Error(t, err)
}

func TestParseAdhocParams_OK(t *testing.T) {
	taskID, st, et, err := parseAdhocParams([]byte(`{"task_id":"11111111-1111-1111-1111-111111111111","start_time":"2026-06-01T00:00:00Z"}`))
	require.NoError(t, err)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", taskID.String())
	assert.False(t, st.IsZero())
	assert.True(t, et.IsZero())
}

func TestParseAdhocParams_MissingTaskID(t *testing.T) {
	_, _, _, err := parseAdhocParams([]byte(`{}`))
	require.Error(t, err)
}

func TestParseAdhocParams_BadTaskID(t *testing.T) {
	_, _, _, err := parseAdhocParams([]byte(`{"task_id":"nope"}`))
	require.Error(t, err)
}
