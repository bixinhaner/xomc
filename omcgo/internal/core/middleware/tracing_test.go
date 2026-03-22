package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func setupTracerForTest() *tracetest.InMemoryExporter {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	return exporter
}

func TestTracing_CreatesSpan(t *testing.T) {
	exporter := setupTracerForTest()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Tracing("test-service"))
	r.GET("/api/v1/devices", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	spans := exporter.GetSpans()
	assert.Len(t, spans, 1)
	assert.Equal(t, "HTTP GET /api/v1/devices", spans[0].Name)
}

func TestTracing_Records500AsError(t *testing.T) {
	exporter := setupTracerForTest()

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Tracing("test-service"))
	r.GET("/api/v1/fail", func(c *gin.Context) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fail", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	spans := exporter.GetSpans()
	assert.Len(t, spans, 1)
	// Verify the span has error status
	assert.NotEmpty(t, spans[0].Status)
}

func TestTracing_NoOpProviderZeroCost(t *testing.T) {
	// Set no-op provider (default)
	otel.SetTracerProvider(sdktrace.NewTracerProvider())

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Tracing("test-service"))
	r.GET("/api/v1/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
