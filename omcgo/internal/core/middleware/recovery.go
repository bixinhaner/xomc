package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery 返回一个 Gin 中间件，捕获 Handler 内的 panic，
// 记录堆栈跟踪和请求信息至日志，并返回 500 Internal Server Error。
// 使用场景：全局第一个中间件，尽早注册以捕获所有 Handler 的意外 panic，
// 防止单个请求 panic 导致整个服务崩溃。
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					zap.Any("error", r),
					zap.String("stack", string(debug.Stack())),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": "internal server error",
				})
			}
		}()
		c.Next()
	}
}
