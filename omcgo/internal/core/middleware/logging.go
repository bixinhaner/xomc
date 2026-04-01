package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omcgo/omcgo/internal/core/components/logger"
	"go.uber.org/zap"
)

// RequestLogger 返回一个 Gin 中间件，记录每个 HTTP 请求的方法、路径、状态码、耗时和客户端 IP。
// 日志中自动包含 RequestID（由 RequestID 中间件提前注入），导入 context-aware logger。
// 使用场景：为 App/ACS 服务的所有 HTTP 请求提供访问日志。
// 建议在 RequestID 中间件之后、业务 Handler 之前注册。
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
