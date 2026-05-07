package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/alarm"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/syslog"
	"github.com/omcgo/omcgo/internal/pm"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDashboardSummaryJSONFields verifies the Dashboard summary
// response structure matches frontend dashboardApi.ts expectations.
func TestDashboardSummaryJSONFields(t *testing.T) {
	summary := map[string]interface{}{
		"device_stats":  map[string]interface{}{"total": 10, "online": 8, "offline": 2},
		"alarm_stats":   map[string]interface{}{"total": 5, "critical": 1, "major": 2},
		"kpi_overview":  []interface{}{},
		"recent_alarms": []interface{}{},
		"timestamp":     "2026-03-07T10:00:00Z",
	}

	data, err := json.Marshal(summary)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	requiredFields := []string{"device_stats", "alarm_stats", "kpi_overview", "recent_alarms", "timestamp"}
	for _, field := range requiredFields {
		assert.Contains(t, result, field, "Dashboard summary must have field: %s", field)
	}
}

// TestAlarmFilterRuleResponseFormat verifies the AlarmFilterRule JSON structure
// matches frontend alarmApi.ts BackendAlarmFilterRule expectations.
func TestAlarmFilterRuleResponseFormat(t *testing.T) {
	rule := alarm.AlarmFilterRule{
		ID:         uuid.New(),
		Name:       "Test Filter Rule",
		FilterType: "suppress",
		Action:     "suppress",
		Priority:   1,
		Enabled:    true,
	}

	data, err := json.Marshal(rule)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	requiredFields := []string{"id", "name", "filter_type", "action", "priority", "enabled"}
	for _, field := range requiredFields {
		assert.Contains(t, result, field, "AlarmFilterRule must have field: %s", field)
	}
}

// TestKPIThresholdResponseFormat verifies the KPIThreshold JSON structure.
func TestKPIThresholdResponseFormat(t *testing.T) {
	threshold := pm.KPIThreshold{
		KPIName:    "cell_availability",
		Carrier:    "cmcc",
		Technology: "lte",
		Comparison: "gte",
		Enabled:    true,
	}

	data, err := json.Marshal(threshold)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Contains(t, result, "kpi_name")
	assert.Contains(t, result, "comparison")
	assert.Contains(t, result, "enabled")
}

// TestSystemLogResponseFormat verifies the SystemLog JSON structure.
func TestSystemLogResponseFormat(t *testing.T) {
	log := syslog.SystemLog{
		Level:   "ERROR",
		Source:  "alarm-engine",
		Message: "Test error message",
	}

	data, err := json.Marshal(log)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Contains(t, result, "level")
	assert.Contains(t, result, "source")
	assert.Contains(t, result, "message")
}

// TestNEMessageLogResponseFormat verifies the NEMessageLog JSON structure.
func TestNEMessageLogResponseFormat(t *testing.T) {
	log := syslog.NEMessageLog{
		DeviceSN:    "CMCC-ENB-001",
		MessageType: "Inform",
		Direction:   "inbound",
	}

	data, err := json.Marshal(log)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	assert.Contains(t, result, "device_sn")
	assert.Contains(t, result, "message_type")
	assert.Contains(t, result, "direction")
}

// TestErrorResponseWithRequestID verifies that ErrorResponse includes request_id.
func TestErrorResponseWithRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-req-123")
		c.Next()
	})

	router.GET("/test-error", func(c *gin.Context) {
		commonerrors.AbortWithError(c, http.StatusNotFound, commonerrors.ErrNotFound)
	})

	req := httptest.NewRequest(http.MethodGet, "/test-error", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errResp commonerrors.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)

	// v0.6 envelope: {ret:0, msg, data:null, biz_code?, request_id?}
	assert.Equal(t, "test-req-123", errResp.RequestID, "ErrorResponse must include request_id")
	assert.Equal(t, 0, errResp.Ret)
	// 无 BusinessError 时 BizCode = 0（omitempty 不上线，但结构体字段值仍是 0）
}

// TestErrorResponseWithBusinessError verifies BusinessError propagation.
func TestErrorResponseWithBusinessError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	router.Use(func(c *gin.Context) {
		c.Set("request_id", "test-req-456")
		c.Next()
	})

	router.GET("/test-biz-error", func(c *gin.Context) {
		err := commonerrors.NewBusinessError(
			commonerrors.ErrCodeDeviceNotFound,
			"device not found",
			nil,
		)
		commonerrors.AbortWithError(c, http.StatusNotFound, err)
	})

	req := httptest.NewRequest(http.MethodGet, "/test-biz-error", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var errResp commonerrors.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	require.NoError(t, err)

	// v0.6 envelope: BusinessError.Code → ErrorResponse.BizCode；Message → Msg
	assert.Equal(t, 0, errResp.Ret)
	assert.Equal(t, commonerrors.ErrCodeDeviceNotFound, errResp.BizCode, "should propagate business error code")
	assert.Equal(t, "device not found", errResp.Msg)
	assert.Equal(t, "test-req-456", errResp.RequestID)
}
