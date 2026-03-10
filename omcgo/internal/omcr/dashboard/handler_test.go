package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/omcr/admin"
	"github.com/stretchr/testify/assert"
)

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
	h := NewHandler(nil) // nil service — we only test validation paths
	h.RegisterRoutes(r.Group("/api/v1"))
	return r, h
}

// ---------------------------------------------------------------------------
// Tests: parameter validation (no service call needed)
// ---------------------------------------------------------------------------

func TestDashHandler_AlarmTrend_InvalidDays(t *testing.T) {
	router, _ := dashHSetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/dashboard/alarm-trend?days=abc", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDashHandler_AlarmTrend_NegativeDays(t *testing.T) {
	router, _ := dashHSetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/dashboard/alarm-trend?days=-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDashHandler_KPITrend_MissingKPIName(t *testing.T) {
	router, _ := dashHSetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/dashboard/kpi-trend", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDashHandler_KPITrend_InvalidDays(t *testing.T) {
	router, _ := dashHSetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/dashboard/kpi-trend?kpi_name=rrc&days=xyz", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDashHandler_KPITimeSeries_MissingKPINames(t *testing.T) {
	router, _ := dashHSetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/dashboard/kpi-time-series", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDashHandler_KPITimeSeries_InvalidStartTime(t *testing.T) {
	router, _ := dashHSetupRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet,
		"/api/v1/dashboard/kpi-time-series?kpi_names=rrc&start_time=bad", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDashHandler_Widgets_NoAuth(t *testing.T) {
	router, _ := dashHSetupRouter()

	// No user_id in gin context → should return 401
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/dashboard/widgets", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

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

func TestDashHandler_GetUserID_InvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/test", func(c *gin.Context) {
		c.Set(admin.CtxKeyUserID, "not-a-uuid") // wrong type
		_, err := getUserID(c)
		assert.Error(t, err)
		c.Status(http.StatusOK)
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)
}

func TestDashHandler_RegisterRoutes(t *testing.T) {
	router, _ := dashHSetupRouter()

	// Verify 404 for unknown path under dashboard group
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/dashboard/nonexistent", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
