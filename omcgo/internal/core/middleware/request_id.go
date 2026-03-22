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

// RequestIDConfig holds configuration for the RequestID middleware.
type RequestIDConfig struct {
	// Prefix is the prefix for generated request IDs (e.g., "app", "acs", "worker")
	Prefix string
}

// RequestID is a Gin middleware that generates or propagates a request ID.
// It follows these rules:
// 1. If X-Request-ID header exists, use it (supports distributed tracing)
// 2. Otherwise, generate a new unique request ID
// 3. Store request ID in context for logger access
// 4. Set request ID in response header for client correlation
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
