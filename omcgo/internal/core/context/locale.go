package context

import (
	"context"
	"strings"
)

// Locale 标识请求语言，用于后端按语言返回本地化显示名（指标名 / 内置任务名）。
// 只有两种取值：中文（默认）与英文。未识别 / 缺省一律回退中文，保证中文路径零回归。
type Locale string

const (
	// LocaleZH 中文（默认）。
	LocaleZH Locale = "zh-CN"
	// LocaleEN 英文。
	LocaleEN Locale = "en-US"
)

// localeKey 是 locale 在 context 中的私有键，避免跨包碰撞。
const localeKey ctxKey = 1

// ParseAcceptLanguage 从 Accept-Language 头解析出 Locale。
// 只区分中英两类：头里出现以 "en" 开头的语言标签即英文，否则（含空头、zh、未知）一律中文。
// 不做完整 q-value 排序——只取首选项的语言前缀，足够覆盖前端注入的标准头。
func ParseAcceptLanguage(header string) Locale {
	if header == "" {
		return LocaleZH
	}
	// 取第一个语言标签（逗号前），形如 "en-US,en;q=0.9" → "en-US"
	first := header
	if i := strings.IndexByte(first, ','); i >= 0 {
		first = first[:i]
	}
	if i := strings.IndexByte(first, ';'); i >= 0 {
		first = first[:i]
	}
	first = strings.ToLower(strings.TrimSpace(first))
	if strings.HasPrefix(first, "en") {
		return LocaleEN
	}
	return LocaleZH
}

// WithLocale 把 locale 存入 context，由 locale 中间件注入。
func WithLocale(ctx context.Context, loc Locale) context.Context {
	return context.WithValue(ctx, localeKey, loc)
}

// GetLocale 从 context 取 locale。缺省（无中间件 / 非 HTTP 链路，如 worker）回退中文。
func GetLocale(ctx context.Context) Locale {
	if loc, ok := ctx.Value(localeKey).(Locale); ok && loc != "" {
		return loc
	}
	return LocaleZH
}
