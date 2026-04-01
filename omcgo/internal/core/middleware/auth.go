// Package middleware 提供 Gin HTTP 中间件合集。
// 包含：认证、CORS、请求日志、Prometheus 指标、Panic 恢复、
// Request ID 传播、安全响应头和 OpenTelemetry 干源。
// 各中间件均实现 gin.HandlerFunc，由服务路由层按需组合使用。
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 返回一个 Gin 中间件，验证 HTTP 请求中的 JWT Bearer Token。
// 要求 Authorization 头格式为 "Bearer <token>"，验证失败时返回 401。
// 使用场景：App 服务的需要登录的 REST API 路由组，登录/健康检查接口跳过此中间件。
// 注意：Phase 1 仅验证 Token 非空，Phase 4 完整实现 JWT 签名验证。
func AuthMiddleware(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "missing authorization header",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid authorization header format",
			})
			return
		}

		// Phase 1: accept any non-empty token for development.
		// Phase 4 will implement proper JWT validation.
		token := parts[1]
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "empty token",
			})
			return
		}

		c.Set("token", token)
		c.Next()
	}
}
