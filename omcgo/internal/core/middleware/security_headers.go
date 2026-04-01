package middleware

import "github.com/gin-gonic/gin"

// SecurityHeaders 返回一个 Gin 中间件，为应答设置 HTTP 安全响应头，
// 防御点击劫持（X-Frame-Options）、MIME 嵅听（X-Content-Type-Options）、
// XSS 注入（Content-Security-Policy）等常见 Web 攻击。
// HTTPS 连接时自动添加 HSTS 头（有效期 1 年）。
// 使用场景：App 服务的全局中间件，小固定代价覆盖安全基线。
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}
