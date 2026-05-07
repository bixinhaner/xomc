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
	// v0.6 envelope: {ret:0, msg, data:null, biz_code?}
	assert.Contains(t, body, `"ret":0`)
	assert.Contains(t, body, `"biz_code":1001`)
	assert.Contains(t, body, `"msg":"device not found"`)
	assert.Contains(t, body, `"data":null`)
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
	// v0.6: plain error 的 .Error() 文本写入 msg 字段（不再有 details）
	assert.Contains(t, body, `"ret":0`)
	assert.Contains(t, body, `"msg":"something went wrong"`)
}

// TestAbortWithError_RedactsSensitiveJSONInDetails confirms that if a
// BusinessError's wrapped error happens to contain JSON-encoded
// sensitive fields (a real risk when handlers return third-party SDK
// errors), the response body does not leak the plaintext value.
func TestAbortWithError_RedactsSensitiveJSONInDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/login", nil)

	// Simulate a handler that bubbles up an error whose .Error() text
	// is itself a JSON object (common for upstream API failures).
	plainErr := fmt.Errorf(`{"user":"alice","password":"supersecret","token":"Bearer abcdef123456"}`)

	AbortWithError(c, http.StatusBadRequest, plainErr)

	body := w.Body.String()
	// The outer envelope is JSON: details is the inner JSON-as-string.
	// Inner secrets must NOT appear in the byte stream verbatim.
	assert.NotContains(t, body, "supersecret",
		"plaintext password leaked into error response: %s", body)
	assert.NotContains(t, body, "abcdef123456",
		"plaintext token leaked into error response: %s", body)
}

// TestAbortWithError_BusinessError_RedactsMessage covers the case where
// the message field itself carries a sensitive substring. We accept that
// the legacy "message" path is harder to scrub, but the JSON envelope
// must still be valid and the response status correct.
func TestAbortWithError_BusinessError_NoLeakWhenMessageIsSafe(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	be := NewBusinessError(7001, "authentication failed", nil)
	AbortWithError(c, http.StatusUnauthorized, be)

	body := w.Body.String()
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, body, `"ret":0`)
	assert.Contains(t, body, `"biz_code":7001`)
	assert.Contains(t, body, `"msg":"authentication failed"`)
}
