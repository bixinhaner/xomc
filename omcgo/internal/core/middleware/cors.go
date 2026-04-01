package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORSConfig 定义 CORS 中间件的允许来源列表。
// AllowOrigins 应填入前端部署的完整域名，如 "https://omc.example.com"。
// 快速开发时可写 "http://localhost:3000"。
type CORSConfig struct {
	AllowOrigins []string
}

// CORS 返回一个 Gin 中间件，根据 AllowOrigins 加白名单处理跨域请求。
// 对地址在列表中的请求，自动添加 Access-Control-Allow-* 响应头；
// 对 OPTIONS 预检请求直接返回 204，不进入后续处理链。
// 使用场景：App 服务的全局路由组入口，在识别中间件之前注册。
func CORS(cfg CORSConfig) gin.HandlerFunc {
	allowedOrigins := make(map[string]bool, len(cfg.AllowOrigins))
	for _, o := range cfg.AllowOrigins {
		allowedOrigins[o] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Max-Age", "86400")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
