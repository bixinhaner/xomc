// Package context provides context utilities for request-scoped values.
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
