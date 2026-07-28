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
	assert.Equal(t, []string{"Cellid=1,PLMN=00101"}, req.ObjectLDNs)
	// A1：保留旧返回值给导出直查路径，同时 QueryRequest 也带同一份白名单供补骨架复用。
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
	filter, err := parseAdhocParams([]byte(`{
		"task_id":"11111111-1111-1111-1111-111111111111",
		"start_time":"2026-06-01T00:00:00Z",
		"product_ids":["22222222-2222-2222-2222-222222222222"],
		"object_ldns":["DeviceGroup=33333333-3333-3333-3333-333333333333,Tech=lte"],
		"weekdays":[1,2,3],
		"hours":[8,9]
	}`))
	require.NoError(t, err)
	assert.Equal(t, "11111111-1111-1111-1111-111111111111", filter.TaskID.String())
	assert.False(t, filter.StartTime.IsZero())
	assert.True(t, filter.EndTime.IsZero())
	require.Len(t, filter.ProductIDs, 1)
	assert.Equal(t, "22222222-2222-2222-2222-222222222222", filter.ProductIDs[0].String())
	assert.Equal(t, []string{"DeviceGroup=33333333-3333-3333-3333-333333333333,Tech=lte"}, filter.ObjectLDNs)
	assert.Equal(t, []int{1, 2, 3}, filter.Weekdays)
	assert.Equal(t, []int{8, 9}, filter.Hours)
}

func TestParseAdhocParams_MissingTaskID(t *testing.T) {
	_, err := parseAdhocParams([]byte(`{}`))
	require.Error(t, err)
}

func TestParseAdhocParams_BadTaskID(t *testing.T) {
	_, err := parseAdhocParams([]byte(`{"task_id":"nope"}`))
	require.Error(t, err)
}

func TestParseAdhocParams_BadProductID(t *testing.T) {
	_, err := parseAdhocParams([]byte(`{"task_id":"11111111-1111-1111-1111-111111111111","product_ids":["nope"]}`))
	require.Error(t, err)
}
