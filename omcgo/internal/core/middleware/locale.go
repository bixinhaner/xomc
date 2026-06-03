package middleware

import (
	"github.com/gin-gonic/gin"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
)

// LocaleContextKey 是 locale 在 gin context 中的键（供 handler 直接 c.Get 读取，可选）。
const LocaleContextKey = "locale"

// Locale 返回一个 Gin 中间件，从 Accept-Language 请求头解析语言并注入 Go context。
// 领域层（如 PM 指标名 / 内置任务名本地化）通过 core/context.GetLocale(ctx) 读取。
// 缺省 / 未识别一律回退中文，保证中文路径零回归。
// 使用场景：全局中间件，紧随 RequestID / Tracing 之后注册。
func Locale() gin.HandlerFunc {
	return func(c *gin.Context) {
		loc := appcontext.ParseAcceptLanguage(c.GetHeader("Accept-Language"))
		ctx := appcontext.WithLocale(c.Request.Context(), loc)
		c.Request = c.Request.WithContext(ctx)
		c.Set(LocaleContextKey, string(loc))
		c.Next()
	}
}
