package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omcgo/omcgo/internal/core/components/logger"
	"go.uber.org/zap"
)

// RequestLogger returns a Gin middleware that logs request details.
// It uses the request_id from context (set by RequestID middleware) for tracing.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		// Use context-aware logger which automatically includes request_id
		log := logger.L(c.Request.Context())
		log.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Float64("duration_ms", float64(duration.Nanoseconds())/1e6),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)
	}
}
