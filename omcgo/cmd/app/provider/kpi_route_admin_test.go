package provider

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/admin"
	kpirouter "github.com/omcgo/omcgo/internal/pm/kpi/router"
)

type kpiRouteInvalidatorStub struct {
	result kpirouter.InvalidationResult
	err    error
	calls  int
}

func (s *kpiRouteInvalidatorStub) Invalidate(context.Context, kpirouter.InvalidationTrigger) (kpirouter.InvalidationResult, error) {
	s.calls++
	return s.result, s.err
}

func newKPIRouteAdminTestRouter(invalidator kpiRouteInvalidator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if c.GetHeader("X-Test-Super-Admin") == "true" {
			c.Set(admin.CtxKeyIsSuperAdmin, true)
		}
		c.Next()
	})
	group := r.Group("/api/v1")
	group.Use(admin.RequireSuperAdmin())
	registerKPIRouteAdminRoutes(invalidator, zap.NewNop(), group)
	return r
}

func TestKPIRouteAdminRefresh_RejectsNonSuperAdmin(t *testing.T) {
	invalidator := &kpiRouteInvalidatorStub{}
	r := newKPIRouteAdminTestRouter(invalidator)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/kpi-routes/refresh", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Zero(t, invalidator.calls)
}

func TestKPIRouteAdminRefresh_ReturnsNewVersion(t *testing.T) {
	invalidator := &kpiRouteInvalidatorStub{result: kpirouter.InvalidationResult{
		CacheVersion:     12,
		Attempts:         2,
		Scope:            kpirouter.InvalidationScopeGlobal,
		MultiProcessSync: true,
	}}
	r := newKPIRouteAdminTestRouter(invalidator)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/kpi-routes/refresh", nil)
	req.Header.Set("X-Test-Super-Admin", "true")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, invalidator.calls)
	var envelope struct {
		Data kpirouter.InvalidationResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Equal(t, int64(12), envelope.Data.CacheVersion)
	require.Equal(t, 2, envelope.Data.Attempts)
	require.Equal(t, kpirouter.InvalidationScopeGlobal, envelope.Data.Scope)
	require.True(t, envelope.Data.MultiProcessSync)
}

func TestKPIRouteAdminRefresh_ReturnsExplicitFailure(t *testing.T) {
	invalidator := &kpiRouteInvalidatorStub{
		result: kpirouter.InvalidationResult{Attempts: 3},
		err:    errors.New("incr kpi-route:cache_version: dial tcp redis.internal:6379: connection refused"),
	}
	r := newKPIRouteAdminTestRouter(invalidator)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/kpi-routes/refresh", nil)
	req.Header.Set("X-Test-Super-Admin", "true")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	var envelope struct {
		Ret int    `json:"ret"`
		Msg string `json:"msg"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	require.Zero(t, envelope.Ret)
	require.Contains(t, envelope.Msg, "failed after 3 attempts")
	require.NotContains(t, envelope.Msg, "redis.internal", "internal dependency details must stay in logs")
}
