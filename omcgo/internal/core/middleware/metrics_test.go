package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRegistry(t *testing.T) *prometheus.Registry {
	t.Helper()
	return prometheus.NewRegistry()
}

func getMetricFamily(t *testing.T, reg *prometheus.Registry, name string) *dto.MetricFamily {
	t.Helper()
	families, err := reg.Gather()
	require.NoError(t, err)
	for _, fam := range families {
		if fam.GetName() == name {
			return fam
		}
	}
	return nil
}

func TestPrometheusMetrics_CounterIncremented(t *testing.T) {
	reg := newTestRegistry(t)

	r := gin.New()
	r.Use(PrometheusMetrics(reg))
	r.GET("/api/v1/devices", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	fam := getMetricFamily(t, reg, "http_requests_total")
	require.NotNil(t, fam, "http_requests_total metric should exist")
	require.Len(t, fam.GetMetric(), 1)

	m := fam.GetMetric()[0]
	assert.Equal(t, float64(1), m.GetCounter().GetValue())

	// Verify labels
	labelMap := make(map[string]string)
	for _, lp := range m.GetLabel() {
		labelMap[lp.GetName()] = lp.GetValue()
	}
	assert.Equal(t, "GET", labelMap["method"])
	assert.Equal(t, "/api/v1/devices", labelMap["path"])
	assert.Equal(t, "200", labelMap["status"])
}

func TestPrometheusMetrics_HistogramRecordsDuration(t *testing.T) {
	reg := newTestRegistry(t)

	r := gin.New()
	r.Use(PrometheusMetrics(reg))
	r.GET("/api/v1/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	r.ServeHTTP(w, req)

	fam := getMetricFamily(t, reg, "http_request_duration_seconds")
	require.NotNil(t, fam, "http_request_duration_seconds metric should exist")
	require.Len(t, fam.GetMetric(), 1)

	m := fam.GetMetric()[0]
	assert.Equal(t, uint64(1), m.GetHistogram().GetSampleCount())
	assert.Greater(t, m.GetHistogram().GetSampleSum(), float64(0))

	// Verify labels
	labelMap := make(map[string]string)
	for _, lp := range m.GetLabel() {
		labelMap[lp.GetName()] = lp.GetValue()
	}
	assert.Equal(t, "GET", labelMap["method"])
	assert.Equal(t, "/api/v1/test", labelMap["path"])
}

func TestPrometheusMetrics_MultipleRequests_CounterIncrements(t *testing.T) {
	reg := newTestRegistry(t)

	r := gin.New()
	r.Use(PrometheusMetrics(reg))
	r.GET("/api/v1/items", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/items", nil)
		r.ServeHTTP(w, req)
	}

	fam := getMetricFamily(t, reg, "http_requests_total")
	require.NotNil(t, fam)
	require.Len(t, fam.GetMetric(), 1)
	assert.Equal(t, float64(5), fam.GetMetric()[0].GetCounter().GetValue())
}

func TestPrometheusMetrics_DifferentStatusCodes_SeparateLabels(t *testing.T) {
	reg := newTestRegistry(t)

	r := gin.New()
	r.Use(PrometheusMetrics(reg))
	r.GET("/api/v1/resource", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.POST("/api/v1/resource", func(c *gin.Context) {
		c.String(http.StatusCreated, "created")
	})

	// GET => 200
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/resource", nil)
	r.ServeHTTP(w1, req1)

	// POST => 201
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/resource", nil)
	r.ServeHTTP(w2, req2)

	fam := getMetricFamily(t, reg, "http_requests_total")
	require.NotNil(t, fam)
	// Should have 2 separate metric entries (different method+status labels)
	assert.Equal(t, 2, len(fam.GetMetric()))

	// Collect all label combos
	seen := make(map[string]bool)
	for _, m := range fam.GetMetric() {
		labelMap := make(map[string]string)
		for _, lp := range m.GetLabel() {
			labelMap[lp.GetName()] = lp.GetValue()
		}
		key := labelMap["method"] + ":" + labelMap["status"]
		seen[key] = true
	}
	assert.True(t, seen["GET:200"])
	assert.True(t, seen["POST:201"])
}

func TestPrometheusMetrics_DefaultRegistry_NoPanic(t *testing.T) {
	// When no registerer is provided, should use default registry.
	// We can't easily inspect the default registry without side effects,
	// so we just verify no panic occurs.
	assert.NotPanics(t, func() {
		r := gin.New()
		r.Use(PrometheusMetrics())
		r.GET("/ping", func(c *gin.Context) {
			c.String(http.StatusOK, "pong")
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	})
}
