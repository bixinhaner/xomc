// Package context 提供请求级 context 注入工具函数。
// 目前只包含 Request ID 的存取，由 middleware/request_id.go 注入，
// 领域层通过 GetRequestID(ctx) 读取用于日志和错误响应。
package context

import "context"

type ctxKey int

const (
	requestIDKey ctxKey = iota
)

// WithRequestID stores the Request ID in the context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID retrieves the Request ID from the context.
// Returns an empty string if not found.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}
