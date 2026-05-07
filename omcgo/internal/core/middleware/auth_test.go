package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/omcgo/omcgo/internal/core/response"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware_MissingHeader_Returns401(t *testing.T) {
	r := gin.New()
	r.Use(AuthMiddleware("test-secret"))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	msg, _ := response.DecodeFail(t, w.Body)
	assert.Equal(t, "missing authorization header", msg)
}

func TestAuthMiddleware_InvalidFormat_Returns401(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"no bearer prefix", "Token abc123"},
		{"basic auth", "Basic dXNlcjpwYXNz"},
		{"just a token", "abc123"},
		{"bearer lowercase", "bearer abc123"},
		{"empty bearer", "Bearer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(AuthMiddleware("test-secret"))
			r.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", tt.header)
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			response.DecodeFail(t, w.Body)
		})
	}
}

func TestAuthMiddleware_EmptyToken_Returns401(t *testing.T) {
	r := gin.New()
	r.Use(AuthMiddleware("test-secret"))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer ")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	msg, _ := response.DecodeFail(t, w.Body)
	assert.Equal(t, "empty token", msg)
}

func TestAuthMiddleware_ValidToken_PassesThrough(t *testing.T) {
	r := gin.New()
	r.Use(AuthMiddleware("test-secret"))

	var tokenFromCtx string
	r.GET("/test", func(c *gin.Context) {
		val, exists := c.Get("token")
		assert.True(t, exists)
		tokenFromCtx = val.(string)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer my-jwt-token-123")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "my-jwt-token-123", tokenFromCtx)
}

func TestAuthMiddleware_AnyNonEmptyToken_Accepted(t *testing.T) {
	// Phase 1 behavior: any non-empty token is accepted
	tokens := []string{
		"simple-token",
		"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abc",
		"a",
		"token-with-special-chars!@#$%",
	}

	for _, token := range tokens {
		t.Run(token, func(t *testing.T) {
			r := gin.New()
			r.Use(AuthMiddleware("test-secret"))
			r.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestAuthMiddleware_SetsTokenInContext(t *testing.T) {
	r := gin.New()
	r.Use(AuthMiddleware("my-secret"))

	var storedToken string
	r.GET("/test", func(c *gin.Context) {
		val, exists := c.Get("token")
		assert.True(t, exists)
		storedToken = val.(string)
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer expected-token-value")
	r.ServeHTTP(w, req)

	assert.Equal(t, "expected-token-value", storedToken)
}

func TestAuthMiddleware_AbortsPipeline_OnFailure(t *testing.T) {
	handlerCalled := false

	r := gin.New()
	r.Use(AuthMiddleware("test-secret"))
	r.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No Authorization header
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, handlerCalled, "handler should not be called when auth fails")
}
