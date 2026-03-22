package middleware

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestID_GeneratesNewID(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())

	var capturedID string
	r.GET("/test", func(c *gin.Context) {
		capturedID = GetRequestID(c)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, capturedID)
	// Response header should also have the ID
	assert.Equal(t, capturedID, w.Header().Get(RequestIDHeader))
}

func TestRequestID_DefaultPrefix_Format(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())

	var capturedID string
	r.GET("/test", func(c *gin.Context) {
		capturedID = GetRequestID(c)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	// Format: req-YYYYMMDDHHmmss-hexhexhex
	pattern := `^req-\d{14}-[0-9a-f]{8}$`
	assert.Regexp(t, regexp.MustCompile(pattern), capturedID)
}

func TestRequestID_CustomPrefix(t *testing.T) {
	r := gin.New()
	r.Use(RequestIDWithConfig(RequestIDConfig{Prefix: "acs"}))

	var capturedID string
	r.GET("/test", func(c *gin.Context) {
		capturedID = GetRequestID(c)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.True(t, len(capturedID) > 0)
	pattern := `^acs-\d{14}-[0-9a-f]{8}$`
	assert.Regexp(t, regexp.MustCompile(pattern), capturedID)
}

func TestRequestID_EmptyPrefix_UsesDefault(t *testing.T) {
	r := gin.New()
	r.Use(RequestIDWithConfig(RequestIDConfig{Prefix: ""}))

	var capturedID string
	r.GET("/test", func(c *gin.Context) {
		capturedID = GetRequestID(c)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	// Should fall back to "req" prefix
	pattern := `^req-\d{14}-[0-9a-f]{8}$`
	assert.Regexp(t, regexp.MustCompile(pattern), capturedID)
}

func TestRequestID_PreservesExistingHeader(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())

	var capturedID string
	r.GET("/test", func(c *gin.Context) {
		capturedID = GetRequestID(c)
		c.String(http.StatusOK, "ok")
	})

	existingID := "trace-abc-123-xyz"
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set(RequestIDHeader, existingID)
	r.ServeHTTP(w, req)

	assert.Equal(t, existingID, capturedID)
	assert.Equal(t, existingID, w.Header().Get(RequestIDHeader))
}

func TestRequestID_SetsInResponseHeader(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	responseID := w.Header().Get(RequestIDHeader)
	assert.NotEmpty(t, responseID)
}

func TestRequestID_UniquePerRequest(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())

	ids := make([]string, 0, 10)
	r.GET("/test", func(c *gin.Context) {
		ids = append(ids, GetRequestID(c))
		c.String(http.StatusOK, "ok")
	})

	for i := 0; i < 10; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(w, req)
	}

	require.Len(t, ids, 10)
	// All IDs should be unique
	seen := make(map[string]bool)
	for _, id := range ids {
		assert.False(t, seen[id], "duplicate request ID: %s", id)
		seen[id] = true
	}
}

func TestRequestID_SetsInGinContext(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())

	var contextValue string
	r.GET("/test", func(c *gin.Context) {
		val, exists := c.Get(RequestIDContextKey)
		assert.True(t, exists)
		contextValue = val.(string)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.NotEmpty(t, contextValue)
}

func TestGetRequestID_NoMiddleware_ReturnsEmpty(t *testing.T) {
	r := gin.New()

	var result string
	r.GET("/test", func(c *gin.Context) {
		result = GetRequestID(c)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Empty(t, result)
}

func TestGenerateRequestIDWithPrefix_Format(t *testing.T) {
	tests := []struct {
		prefix  string
		pattern string
	}{
		{"req", `^req-\d{14}-[0-9a-f]{8}$`},
		{"app", `^app-\d{14}-[0-9a-f]{8}$`},
		{"acs", `^acs-\d{14}-[0-9a-f]{8}$`},
		{"worker", `^worker-\d{14}-[0-9a-f]{8}$`},
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			id := generateRequestIDWithPrefix(tt.prefix)
			assert.Regexp(t, regexp.MustCompile(tt.pattern), id)
		})
	}
}
