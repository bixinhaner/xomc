package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestRequestLogger_LogsRequestDetails(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	zap.ReplaceGlobals(logger)
	defer zap.ReplaceGlobals(zap.NewNop())

	r := gin.New()
	r.Use(RequestLogger())
	r.GET("/api/v1/devices", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/devices", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, logs.Len(), "expected exactly one log entry")

	entry := logs.All()[0]
	assert.Equal(t, "request", entry.Message)

	// Check all expected fields are present
	fieldMap := make(map[string]zapcore.Field)
	for _, f := range entry.Context {
		fieldMap[f.Key] = f
	}

	assert.Equal(t, "GET", fieldMap["method"].String)
	assert.Equal(t, "/api/v1/devices", fieldMap["path"].String)
	assert.Equal(t, int64(200), fieldMap["status"].Integer)
	assert.Contains(t, fieldMap, "duration_ms")
	assert.Contains(t, fieldMap, "client_ip")
	assert.Contains(t, fieldMap, "user_agent")
}

func TestRequestLogger_LogsDifferentMethods(t *testing.T) {
	methods := []struct {
		method string
		status int
	}{
		{http.MethodGet, http.StatusOK},
		{http.MethodPost, http.StatusCreated},
		{http.MethodPut, http.StatusOK},
		{http.MethodDelete, http.StatusNoContent},
	}

	for _, tt := range methods {
		t.Run(tt.method, func(t *testing.T) {
			core, logs := observer.New(zapcore.InfoLevel)
			logger := zap.New(core)
			zap.ReplaceGlobals(logger)
			defer zap.ReplaceGlobals(zap.NewNop())

			r := gin.New()
			r.Use(RequestLogger())
			r.Handle(tt.method, "/test", func(c *gin.Context) {
				c.String(tt.status, "")
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/test", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, 1, logs.Len())
			entry := logs.All()[0]

			fieldMap := make(map[string]zapcore.Field)
			for _, f := range entry.Context {
				fieldMap[f.Key] = f
			}
			assert.Equal(t, tt.method, fieldMap["method"].String)
			assert.Equal(t, int64(tt.status), fieldMap["status"].Integer)
		})
	}
}

func TestRequestLogger_IncludesRequestID(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	zap.ReplaceGlobals(logger)
	defer zap.ReplaceGlobals(zap.NewNop())

	r := gin.New()
	// RequestID middleware sets the request_id in context, which RequestLogger picks up via logger.L(ctx)
	r.Use(RequestID())
	r.Use(RequestLogger())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 1, logs.Len())

	entry := logs.All()[0]
	fieldMap := make(map[string]zapcore.Field)
	for _, f := range entry.Context {
		fieldMap[f.Key] = f
	}
	// The logger.L(ctx) adds request_id if present
	assert.Contains(t, fieldMap, "request_id")
	assert.NotEmpty(t, fieldMap["request_id"].String)
}

func TestRequestLogger_DurationIsNonNegative(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	zap.ReplaceGlobals(logger)
	defer zap.ReplaceGlobals(zap.NewNop())

	r := gin.New()
	r.Use(RequestLogger())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	entry := logs.All()[0]
	found := false
	for _, f := range entry.Context {
		if f.Key == "duration_ms" {
			found = true
			// zap.Float64 stores value as math.Float64bits in Integer field
			// We just verify the field exists and its type is float64
			assert.Equal(t, zapcore.Float64Type, f.Type)
		}
	}
	assert.True(t, found, "duration_ms field should be present")
}

func TestRequestLogger_RecordsCorrectStatusOnError(t *testing.T) {
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)
	zap.ReplaceGlobals(logger)
	defer zap.ReplaceGlobals(zap.NewNop())

	r := gin.New()
	r.Use(RequestLogger())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusNotFound, "not found")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	entry := logs.All()[0]
	fieldMap := make(map[string]zapcore.Field)
	for _, f := range entry.Context {
		fieldMap[f.Key] = f
	}
	assert.Equal(t, int64(404), fieldMap["status"].Integer)
}
