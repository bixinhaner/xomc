package dashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/admin"
	"github.com/omcgo/omcgo/internal/pm/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dashVisibleGroupsResolver struct{}

func (dashVisibleGroupsResolver) GetUserVisibleGroupIDs(context.Context, uuid.UUID, bool) ([]uuid.UUID, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Helpers
//
// Dashboard Service depends on *pgxpool.Pool, *device.DeviceService, etc.
// (all concrete types). To test handler-level logic (parameter validation,
// auth checks) without a real Postgres, we create a handler with nil service
// and only call endpoints that fail before touching the service.
// ---------------------------------------------------------------------------

func dashHSetupRouter() (*gin.Engine, *Handler) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(nil, nil) // nil service — we only test validation paths
	h.RegisterRoutes(r.Group("/api/v1"))
	return r, h
}

func dashHDoRequest(router *gin.Engine, method, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, nil)
	router.ServeHTTP(w, req)
	return w
}

// ---------------------------------------------------------------------------
// Tests: Route registration
// ---------------------------------------------------------------------------

func TestDashHandler_RegisterRoutes_UnknownPath(t *testing.T) {
	router, _ := dashHSetupRouter()

	w := dashHDoRequest(router, http.MethodGet, "/api/v1/dashboard/nonexistent")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDashHandler_RegisterRoutes_MethodNotAllowed(t *testing.T) {
	router, _ := dashHSetupRouter()

	// POST on a GET-only route should return 405 (Method Not Allowed).
	w := dashHDoRequest(router, http.MethodPost, "/api/v1/dashboard/alarm-trend")
	// Gin by default returns 404 for unmatched method+path combos unless
	// HandleMethodNotAllowed is enabled, so we just verify it's not 200.
	assert.NotEqual(t, http.StatusOK, w.Code)
}

func TestDashHandler_TopAlarmDevicesRequiresVisibilityContext(t *testing.T) {
	router, handler := dashHSetupRouter()
	handler.SetPermissionService(dashVisibleGroupsResolver{})

	w := dashHDoRequest(router, http.MethodGet, "/api/v1/dashboard/top-alarm-devices")

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ---------------------------------------------------------------------------
// Tests: AlarmTrend parameter validation
// ---------------------------------------------------------------------------

func TestDashHandler_AlarmTrend_InvalidDays(t *testing.T) {
	router, _ := dashHSetupRouter()
	w := dashHDoRequest(router, http.MethodGet, "/api/v1/dashboard/alarm-trend?days=abc")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDashHandler_AlarmTrend_NegativeDays(t *testing.T) {
	router, _ := dashHSetupRouter()
	w := dashHDoRequest(router, http.MethodGet, "/api/v1/dashboard/alarm-trend?days=-1")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDashHandler_AlarmTrend_ZeroDays(t *testing.T) {
	router, _ := dashHSetupRouter()
	w := dashHDoRequest(router, http.MethodGet, "/api/v1/dashboard/alarm-trend?days=0")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDashHandler_AlarmTrend_InvalidMetric(t *testing.T) {
	router, _ := dashHSetupRouter()
	w := dashHDoRequest(router, http.MethodGet, "/api/v1/dashboard/alarm-trend?metric=cleared")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDashHandler_ActiveAlarmTrendRequiresVisibilityContext(t *testing.T) {
	router, handler := dashHSetupRouter()
	handler.SetPermissionService(dashVisibleGroupsResolver{})

	w := dashHDoRequest(router, http.MethodGet, "/api/v1/dashboard/alarm-trend?metric=active")

	assert.Equal(t, http.StatusForbidden, w.Code)
}

// ---------------------------------------------------------------------------
// Tests: KPITrend parameter validation
// ---------------------------------------------------------------------------

func TestDashHandler_KPITrend_MissingKPIName(t *testing.T) {
	router, _ := dashHSetupRouter()
	w := dashHDoRequest(router, http.MethodGet, "/api/v1/dashboard/kpi-trend")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDashHandler_KPITrend_EmptyKPIName(t *testing.T) {
	router, _ := dashHSetupRouter()
	w := dashHDoRequest(router, http.MethodGet, "/api/v1/dashboard/kpi-trend?kpi_name=")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDashHandler_KPITrend_InvalidDays(t *testing.T) {
	router, _ := dashHSetupRouter()
	w := dashHDoRequest(router, http.MethodGet, "/api/v1/dashboard/kpi-trend?kpi_name=rrc&days=xyz")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDashHandler_KPITrend_ZeroDays(t *testing.T) {
	router, _ := dashHSetupRouter()
	w := dashHDoRequest(router, http.MethodGet, "/api/v1/dashboard/kpi-trend?kpi_name=rrc&days=0")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Tests: KPITimeSeries parameter validation
// ---------------------------------------------------------------------------

func TestDashHandler_KPITimeSeries_MissingKPINames(t *testing.T) {
	router, _ := dashHSetupRouter()
	w := dashHDoRequest(router, http.MethodGet, "/api/v1/dashboard/kpi-time-series")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDashHandler_KPITimeSeries_EmptyKPINames(t *testing.T) {
	router, _ := dashHSetupRouter()
	w := dashHDoRequest(router, http.MethodGet, "/api/v1/dashboard/kpi-time-series?kpi_names=")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDashHandler_KPITimeSeries_InvalidStartTime(t *testing.T) {
	router, _ := dashHSetupRouter()
	w := dashHDoRequest(router, http.MethodGet,
		"/api/v1/dashboard/kpi-time-series?kpi_names=rrc&start_time=bad")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDashHandler_KPITimeSeries_InvalidEndTime(t *testing.T) {
	router, _ := dashHSetupRouter()
	w := dashHDoRequest(router, http.MethodGet,
		"/api/v1/dashboard/kpi-time-series?kpi_names=rrc&end_time=not-a-date")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDashHandler_KPITimeSeries_InvalidBothTimes(t *testing.T) {
	router, _ := dashHSetupRouter()
	w := dashHDoRequest(router, http.MethodGet,
		"/api/v1/dashboard/kpi-time-series?kpi_names=rrc&start_time=bad&end_time=also-bad")
	// Should fail on start_time first.
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDashHandler_KPITimeSeries_InvalidGranularity(t *testing.T) {
	router, _ := dashHSetupRouter()
	w := dashHDoRequest(router, http.MethodGet,
		"/api/v1/dashboard/kpi-time-series?kpi_names=rrc&granularity=15min")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestParseDashboardKPITechnology(t *testing.T) {
	tests := []struct {
		raw     string
		wantErr bool
	}{
		{raw: ""},
		{raw: "gsm"},
		{raw: "GSM", wantErr: true},
		{raw: "2g", wantErr: true},
	}
	for _, tt := range tests {
		_, err := parseDashboardKPITechnology(tt.raw)
		if tt.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}
}

func TestDashHandler_KPITimeSeries_RejectsNonIncreasingWindow(t *testing.T) {
	router, _ := dashHSetupRouter()
	w := dashHDoRequest(router, http.MethodGet,
		"/api/v1/dashboard/kpi-time-series?kpi_names=rrc&start_time=2026-07-13T00:00:00Z&end_time=2026-07-13T00:00:00Z")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestParseDashboardKPIGranularity(t *testing.T) {
	tests := []struct {
		input   string
		want    metrics.Granularity
		wantErr bool
	}{
		{"", metrics.GranularityHourly, false},
		{"hourly", metrics.GranularityHourly, false},
		{"daily", metrics.GranularityDaily, false},
		{"weekly", metrics.GranularityWeekly, false},
		{"15min", "", true},
	}
	for _, tt := range tests {
		got, err := parseDashboardKPIGranularity(tt.input)
		assert.Equal(t, tt.want, got)
		assert.Equal(t, tt.wantErr, err != nil)
	}
}

// ---------------------------------------------------------------------------
// Tests: Widgets auth
// ---------------------------------------------------------------------------

func TestDashHandler_GetWidgets_NoAuth(t *testing.T) {
	router, _ := dashHSetupRouter()
	w := dashHDoRequest(router, http.MethodGet, "/api/v1/dashboard/widgets")
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDashHandler_SaveWidgets_NoAuth(t *testing.T) {
	router, _ := dashHSetupRouter()

	w := httptest.NewRecorder()
	body := `{"layout": [{"id":"a","x":0,"y":0}]}`
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/dashboard/widgets",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestDashHandler_SaveWidgets_BadBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(nil, nil)
	h.RegisterRoutes(r.Group("/api/v1"))

	// Set user_id in context to pass auth check, then send bad body.
	userID := uuid.New()
	r.Use() // routes already registered
	// Create a custom route with middleware that injects user ID.
	gin.SetMode(gin.TestMode)
	r2 := gin.New()
	r2.PUT("/api/v1/dashboard/widgets", func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, userID)
		h.SaveWidgets(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/api/v1/dashboard/widgets",
		strings.NewReader("not-json"))
	req.Header.Set("Content-Type", "application/json")
	r2.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDashHandler_SaveWidgets_MissingLayout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(nil, nil)
	userID := uuid.New()
	r.PUT("/test-widgets", func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, userID)
		h.SaveWidgets(c)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPut, "/test-widgets",
		strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ---------------------------------------------------------------------------
// Tests: getUserID helper
// ---------------------------------------------------------------------------

func TestDashHandler_GetUserID_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	expectedID := uuid.New()

	r.GET("/test", func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, expectedID)
		id, err := getUserID(c)
		assert.NoError(t, err)
		assert.Equal(t, expectedID, id)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDashHandler_GetUserID_Missing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/test", func(c *gin.Context) {
		_, err := getUserID(c)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not authenticated")
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)
}

func TestDashHandler_GetUserID_InvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/test", func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, "not-a-uuid") // wrong type
		_, err := getUserID(c)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid user ID")
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)
}
