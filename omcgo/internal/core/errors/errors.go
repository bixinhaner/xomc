// Package errors 定义业务错误类型、哨兵错误变量和 HTTP 状态码映射。
// 主要类型：
//   - BusinessError：带数字编码的领域错误，用于 API 响应
//   - ErrorResponse：返回给客户端的 JSON 错误结构
//   - AbortWithError：写入 JSON 错误响应并中止 Gin 处理链
//
// 错误码分段见 codes.go。
package errors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/omcgo/omcgo/internal/core/redact"
)

// Sentinel errors for common failure conditions.
var (
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrInternal      = errors.New("internal error")
	ErrTimeout       = errors.New("operation timed out")
	ErrUnavailable   = errors.New("service unavailable")
)

// Error code ranges by domain:
//   1000-1999: Device Management
//   2000-2999: Data Model / Configuration
//   3000-3999: ACS / TR069
//   4000-4999: Performance Management
//   5000-5999: Alarm Management
//   6000-6999: Auto Provisioning
//   7000-7999: Admin / Auth / RBAC
//   8000-8999: Software / Firmware
//   9000-9999: Northbound / OSS
//   10000-10999: Interop Testing

// BusinessError represents a domain-specific error with a numeric code.
type BusinessError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *BusinessError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *BusinessError) Unwrap() error {
	return e.Err
}

// NewBusinessError creates a new BusinessError.
func NewBusinessError(code int, message string, err error) *BusinessError {
	return &BusinessError{Code: code, Message: message, Err: err}
}

// ErrorResponse is the JSON structure returned to API clients on error.
type ErrorResponse struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Details   string `json:"details,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// AbortWithError writes a JSON error response and aborts the Gin handler chain.
//
// Before sending the body, the response is round-tripped through
// redact.RedactJSON so that any sensitive fields embedded in error
// messages or details (passwords, tokens, API keys, device credentials)
// are masked. This is a defense-in-depth layer: callers should still
// avoid putting secrets in error messages, but mistakes do not leak.
func AbortWithError(c *gin.Context, statusCode int, err error) {
	var reqIDStr string
	if rid, exists := c.Get("request_id"); exists {
		reqIDStr, _ = rid.(string)
	}

	resp := ErrorResponse{
		Code:      statusCode,
		Message:   http.StatusText(statusCode),
		RequestID: reqIDStr,
	}

	var bErr *BusinessError
	if errors.As(err, &bErr) {
		resp.Code = bErr.Code
		resp.Message = bErr.Message
	} else if err != nil {
		// If the error string is itself JSON (common when bubbling up
		// upstream API errors), redact embedded sensitive fields before
		// surfacing it. RedactJSON returns the input unchanged when it
		// is not valid JSON, so plain messages flow through untouched.
		resp.Details = string(redact.RedactJSON([]byte(err.Error())))
	}

	body, marshalErr := json.Marshal(resp)
	if marshalErr != nil {
		// Fallback: use Gin's default JSON marshalling so we still emit
		// a valid response even if redaction can't run.
		c.AbortWithStatusJSON(statusCode, resp)
		return
	}
	cleaned := redact.RedactJSON(body)

	c.Status(statusCode)
	c.Header("Content-Type", "application/json; charset=utf-8")
	_, _ = c.Writer.Write(cleaned)
	c.Abort()
}

// HTTPStatusFromError maps sentinel errors to HTTP status codes.
func HTTPStatusFromError(err error) int {
	switch {
	case errors.Is(err, ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, ErrInvalidInput):
		return http.StatusBadRequest
	case errors.Is(err, ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, ErrTimeout):
		return http.StatusGatewayTimeout
	case errors.Is(err, ErrUnavailable):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
