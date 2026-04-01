package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omcgo/omcgo/internal/core/components/logger"
)

const (
	// RequestIDHeader is the header key for request ID
	RequestIDHeader = "X-Request-ID"
	// RequestIDContextKey is the gin context key for request ID
	RequestIDContextKey = "request_id"
	// DefaultRequestIDPrefix is the default prefix for request IDs
	DefaultRequestIDPrefix = "req"
)

// RequestIDConfig 配置 RequestID 中间件的请求 ID 前缀。
// Prefix 用于区分不同服务的请求来源，建议按服务设置：
// app 服务用 "app"，acs 服务用 "acs"，worker 服务用 "worker"。
type RequestIDConfig struct {
	// Prefix is the prefix for generated request IDs (e.g., "app", "acs", "worker")
	Prefix string
}

// RequestID 返回一个 Gin 中间件，自动生成或传播请求 ID。
// 处理逐序：如存在 X-Request-ID 头就复用（支持分布式跟踪），
// 否则生成新 ID （格式: {prefix}-{timestamp}-{random}）并写入响应头。
// Request ID 同时存入 Go context 和 Gin context，下游日志和错误响应均可读取。
// 使用场景：全局第二个中间件（在 Recovery 之后），尽早注入用于全链路日志关联。
func RequestID() gin.HandlerFunc {
	return RequestIDWithConfig(RequestIDConfig{Prefix: DefaultRequestIDPrefix})
}

// RequestIDWithConfig creates a RequestID middleware with custom configuration.
func RequestIDWithConfig(cfg RequestIDConfig) gin.HandlerFunc {
	prefix := cfg.Prefix
	if prefix == "" {
		prefix = DefaultRequestIDPrefix
	}

	return func(c *gin.Context) {
		// 1. Try to get from request header (for distributed tracing)
		requestID := c.GetHeader(RequestIDHeader)

		// 2. Generate new ID if not present
		if requestID == "" {
			requestID = GenerateRequestIDWithPrefix(prefix)
		}

		// 3. Store in Go context for logger
		ctx := logger.WithRequestID(c.Request.Context(), requestID)
		c.Request = c.Request.WithContext(ctx)

		// 4. Set in response header
		c.Header(RequestIDHeader, requestID)

		// 5. Also store in gin context for easy access
		c.Set(RequestIDContextKey, requestID)

		c.Next()
	}
}

// GenerateRequestIDWithPrefix generates a unique request ID with a custom prefix.
// Format: {prefix}-{timestamp}-{random}
// Example: app-20260319150430-a1b2c3d4
func GenerateRequestIDWithPrefix(prefix string) string {
	timestamp := time.Now().Format("20060102150405")
	random := make([]byte, 4)
	rand.Read(random)
	return prefix + "-" + timestamp + "-" + hex.EncodeToString(random)
}

// GetRequestID retrieves the request ID from gin context.
// Useful for handlers that need direct access to the request ID.
func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get(RequestIDContextKey); exists {
		if str, ok := id.(string); ok {
			return str
		}
	}
	return ""
}
