package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestCORS_AllowedOrigin_SetsHeaders(t *testing.T) {
	r := gin.New()
	r.Use(CORS(CORSConfig{AllowOrigins: []string{"https://example.com"}}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "Content-Type, Authorization, X-Request-ID", w.Header().Get("Access-Control-Allow-Headers"))
	assert.Equal(t, "GET, POST, PUT, PATCH, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
	assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
}

func TestCORS_DisallowedOrigin_NoHeaders(t *testing.T) {
	r := gin.New()
	r.Use(CORS(CORSConfig{AllowOrigins: []string{"https://allowed.com"}}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Credentials"))
}

func TestCORS_NoOriginHeader_NoHeaders(t *testing.T) {
	r := gin.New()
	r.Use(CORS(CORSConfig{AllowOrigins: []string{"https://example.com"}}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_PreflightReturns204(t *testing.T) {
	r := gin.New()
	r.Use(CORS(CORSConfig{AllowOrigins: []string{"https://example.com"}}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "should not reach")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	// Body should be empty for preflight
	assert.Empty(t, w.Body.String())
}

func TestCORS_PreflightDisallowedOrigin_Returns204NoHeaders(t *testing.T) {
	r := gin.New()
	r.Use(CORS(CORSConfig{AllowOrigins: []string{"https://allowed.com"}}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "should not reach")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/test", nil)
	req.Header.Set("Origin", "https://evil.com")
	r.ServeHTTP(w, req)

	// OPTIONS still aborts with 204 regardless of origin
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_MultipleAllowedOrigins(t *testing.T) {
	origins := []string{"https://a.com", "https://b.com", "https://c.com"}
	r := gin.New()
	r.Use(CORS(CORSConfig{AllowOrigins: origins}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	tests := []struct {
		name    string
		origin  string
		allowed bool
	}{
		{"first origin allowed", "https://a.com", true},
		{"second origin allowed", "https://b.com", true},
		{"third origin allowed", "https://c.com", true},
		{"unknown origin rejected", "https://d.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Origin", tt.origin)
			r.ServeHTTP(w, req)

			if tt.allowed {
				assert.Equal(t, tt.origin, w.Header().Get("Access-Control-Allow-Origin"))
			} else {
				assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
			}
		})
	}
}

func TestCORS_EmptyConfig_RejectsAll(t *testing.T) {
	r := gin.New()
	r.Use(CORS(CORSConfig{}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Origin", "https://any.com")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_NonOptionsPassesThrough(t *testing.T) {
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			handlerCalled := false
			r := gin.New()
			r.Use(CORS(CORSConfig{AllowOrigins: []string{"https://example.com"}}))
			r.Handle(method, "/test", func(c *gin.Context) {
				handlerCalled = true
				c.String(http.StatusOK, "ok")
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(method, "/test", nil)
			req.Header.Set("Origin", "https://example.com")
			r.ServeHTTP(w, req)

			assert.True(t, handlerCalled, "handler should be called for %s", method)
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}
