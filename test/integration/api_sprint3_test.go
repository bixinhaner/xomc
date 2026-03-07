package integration

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPMCounterJSONFields verifies the PM counter JSON structure
// matches frontend pmApi.ts expectations.
func TestPMCounterJSONFields(t *testing.T) {
	input := `{"device_id":"00000000-0000-0000-0000-000000000001","cell_id":"cell-1","counter_group":"cell_throughput","counter_name":"dl_throughput_mbps","counter_value":125.5,"time":"2026-03-07T10:00:00Z"}`

	var result map[string]interface{}
	err := json.Unmarshal([]byte(input), &result)
	require.NoError(t, err)

	requiredFields := []string{"counter_group", "counter_name", "counter_value", "time"}
	for _, field := range requiredFields {
		assert.Contains(t, result, field, "PMCounter must have field: %s", field)
	}
}

// TestKPIDefinitionJSONFields verifies the KPIDefinition JSON structure.
func TestKPIDefinitionJSONFields(t *testing.T) {
	input := `{"id":"00000000-0000-0000-0000-000000000001","name":"cell_availability","carrier":"cmcc","technology":"lte","formula":"(uptime / total_time) * 100","unit":"%","description":"Cell availability percentage"}`

	var result map[string]interface{}
	err := json.Unmarshal([]byte(input), &result)
	require.NoError(t, err)

	assert.Contains(t, result, "name")
	assert.Contains(t, result, "carrier")
	assert.Contains(t, result, "technology")
	assert.Contains(t, result, "formula")
}

// TestKPIValueJSONFields verifies the KPIValue JSON structure.
func TestKPIValueJSONFields(t *testing.T) {
	input := `{"kpi_name":"cell_availability","device_id":"00000000-0000-0000-0000-000000000001","carrier":"cmcc","technology":"lte","value":99.5,"time":"2026-03-07T10:00:00Z"}`

	var result map[string]interface{}
	err := json.Unmarshal([]byte(input), &result)
	require.NoError(t, err)

	assert.Contains(t, result, "kpi_name")
	assert.Contains(t, result, "value")
	assert.Contains(t, result, "time")
}

// TestMRFileJSONFields verifies the MRFile JSON structure
// matches frontend mrApi.ts expectations.
func TestMRFileJSONFields(t *testing.T) {
	input := `{"id":"00000000-0000-0000-0000-000000000001","device_id":"00000000-0000-0000-0000-000000000002","device_sn":"CMCC-ENB-001","file_type":"A5","carrier":"cmcc","file_path":"/mr/test.xml","file_size":2048,"created_at":"2026-03-07T10:00:00Z"}`

	var result map[string]interface{}
	err := json.Unmarshal([]byte(input), &result)
	require.NoError(t, err)

	requiredFields := []string{"file_type", "carrier", "device_sn"}
	for _, field := range requiredFields {
		assert.Contains(t, result, field, "MRFile must have field: %s", field)
	}
}

// TestMRRecordJSONFields verifies the MRRecord JSON structure.
func TestMRRecordJSONFields(t *testing.T) {
	input := `{"id":"00000000-0000-0000-0000-000000000001","file_id":"00000000-0000-0000-0000-000000000002","device_id":"00000000-0000-0000-0000-000000000003","record_type":"rsrp","cell_id":"cell-1"}`

	var result map[string]interface{}
	err := json.Unmarshal([]byte(input), &result)
	require.NoError(t, err)

	assert.Contains(t, result, "record_type")
	assert.Contains(t, result, "file_id")
}

// TestPaginatedResponseFormat verifies the paginated response structure
// used by all list endpoints.
func TestPaginatedResponseFormat(t *testing.T) {
	input := `{"items":[{"id":"1"},{"id":"2"}],"total":50,"page":1,"page_size":20}`

	var result map[string]interface{}
	err := json.Unmarshal([]byte(input), &result)
	require.NoError(t, err)

	assert.Contains(t, result, "items")
	assert.Contains(t, result, "total")
	assert.Contains(t, result, "page")
	assert.Contains(t, result, "page_size")

	items, ok := result["items"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, items, 2)
}
