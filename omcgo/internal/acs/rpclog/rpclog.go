package rpclog

import (
	"bytes"
	"context"
	"net/http"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// LogEntry — 可变元数据，附在请求 context 中供 handler 填充
// ─────────────────────────────────────────────────────────────────────────────

// LogEntry holds mutable metadata populated by ACS handlers during request processing.
// It is attached to the request context via WithEntry and retrieved via EntryFromContext.
type LogEntry struct {
	SessionID string
	DeviceSN  string
	Sequence  int
	Method    string
	TaskID    string
	CwmpID    string
	StartTime time.Time
}

type ctxKey struct{}

// WithEntry stores a LogEntry pointer in the context.
func WithEntry(ctx context.Context, entry *LogEntry) context.Context {
	return context.WithValue(ctx, ctxKey{}, entry)
}

// EntryFromContext retrieves the LogEntry from the context. Returns nil if not present.
func EntryFromContext(ctx context.Context) *LogEntry {
	entry, _ := ctx.Value(ctxKey{}).(*LogEntry)
	return entry
}

// ─────────────────────────────────────────────────────────────────────────────
// ResponseCapturer — http.ResponseWriter 包装器，捕获响应字节
// ─────────────────────────────────────────────────────────────────────────────

// ResponseCapturer wraps http.ResponseWriter to capture the response body and status code.
// Write() tees data to both the underlying writer and an internal buffer.
type ResponseCapturer struct {
	http.ResponseWriter
	body       bytes.Buffer
	statusCode int
}

// NewResponseCapturer creates a new ResponseCapturer wrapping the given ResponseWriter.
func NewResponseCapturer(w http.ResponseWriter) *ResponseCapturer {
	return &ResponseCapturer{ResponseWriter: w, statusCode: http.StatusOK}
}

// WriteHeader records the status code and delegates to the underlying ResponseWriter.
func (c *ResponseCapturer) WriteHeader(code int) {
	c.statusCode = code
	c.ResponseWriter.WriteHeader(code)
}

// Write captures the data and delegates to the underlying ResponseWriter.
func (c *ResponseCapturer) Write(b []byte) (int, error) {
	c.body.Write(b)
	return c.ResponseWriter.Write(b)
}

// Body returns the captured response body bytes.
func (c *ResponseCapturer) Body() []byte {
	return c.body.Bytes()
}

// StatusCode returns the captured HTTP status code.
func (c *ResponseCapturer) StatusCode() int {
	return c.statusCode
}
