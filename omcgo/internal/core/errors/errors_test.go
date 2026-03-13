package errors

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBusinessError_Error_WithInner(t *testing.T) {
	inner := fmt.Errorf("db connection refused")
	be := &BusinessError{Code: 1001, Message: "device lookup failed", Err: inner}

	result := be.Error()

	assert.Contains(t, result, "[1001]")
	assert.Contains(t, result, "device lookup failed")
	assert.Contains(t, result, "db connection refused")
}

func TestBusinessError_Error_WithoutInner(t *testing.T) {
	be := &BusinessError{Code: 2001, Message: "invalid config"}

	result := be.Error()

	assert.Contains(t, result, "[2001]")
	assert.Contains(t, result, "invalid config")
	assert.NotContains(t, result, ":")    // no trailing ": <nil>"
	assert.Equal(t, "[2001] invalid config", result)
}

func TestBusinessError_Unwrap(t *testing.T) {
	inner := fmt.Errorf("underlying cause")
	be := &BusinessError{Code: 3001, Message: "acs error", Err: inner}

	unwrapped := be.Unwrap()

	assert.Equal(t, inner, unwrapped)
}

func TestNewBusinessError(t *testing.T) {
	inner := fmt.Errorf("timeout")
	be := NewBusinessError(4001, "pm collection failed", inner)

	require.NotNil(t, be)
	assert.Equal(t, 4001, be.Code)
	assert.Equal(t, "pm collection failed", be.Message)
	assert.Equal(t, inner, be.Err)
}

func TestHTTPStatusFromError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"ErrNotFound", ErrNotFound, http.StatusNotFound},
		{"ErrAlreadyExists", ErrAlreadyExists, http.StatusConflict},
		{"ErrInvalidInput", ErrInvalidInput, http.StatusBadRequest},
		{"ErrUnauthorized", ErrUnauthorized, http.StatusUnauthorized},
		{"ErrForbidden", ErrForbidden, http.StatusForbidden},
		{"ErrTimeout", ErrTimeout, http.StatusGatewayTimeout},
		{"ErrUnavailable", ErrUnavailable, http.StatusServiceUnavailable},
		{"nil (default)", nil, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := HTTPStatusFromError(tt.err)
			assert.Equal(t, tt.expected, status)
		})
	}
}

func TestAbortWithError_BusinessError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	be := NewBusinessError(1001, "device not found", nil)

	AbortWithError(c, http.StatusNotFound, be)

	assert.Equal(t, http.StatusNotFound, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `"code":1001`)
	assert.Contains(t, body, `"message":"device not found"`)
}

func TestAbortWithError_PlainError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	plainErr := fmt.Errorf("something went wrong")

	AbortWithError(c, http.StatusInternalServerError, plainErr)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	body := w.Body.String()
	assert.Contains(t, body, `"details":"something went wrong"`)
}
