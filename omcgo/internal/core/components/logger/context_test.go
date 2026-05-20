package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// TestL_InjectsTraceIDFromContext verifies that L(ctx) reads OTel span context
// from the provided context and adds trace_id / span_id zap fields.
// 这是 T-0155 / T-0157 trace-to-logs 关联的核心契约——任何修改 L() 后必须
// 保证此测试通过，否则 Grafana Tempo → Loki 的精确跳转会回退到模糊匹配。
func TestL_InjectsTraceIDFromContext(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	restore := zap.ReplaceGlobals(zap.New(core))
	defer restore()

	traceID, _ := trace.TraceIDFromHex("0123456789abcdef0123456789abcdef")
	spanID, _ := trace.SpanIDFromHex("0123456789abcdef")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)

	L(ctx).Info("hello")
	require.Equal(t, 1, logs.Len())

	got := logs.All()[0].ContextMap()
	require.Equal(t, "0123456789abcdef0123456789abcdef", got["trace_id"])
	require.Equal(t, "0123456789abcdef", got["span_id"])
}

// TestL_NoTraceIDWhenContextEmpty verifies that L(ctx) does NOT add empty
// trace_id / span_id when no span is in context.
func TestL_NoTraceIDWhenContextEmpty(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	restore := zap.ReplaceGlobals(zap.New(core))
	defer restore()

	L(context.Background()).Info("hello")
	require.Equal(t, 1, logs.Len())

	got := logs.All()[0].ContextMap()
	_, hasTraceID := got["trace_id"]
	_, hasSpanID := got["span_id"]
	require.False(t, hasTraceID, "trace_id must not be set when no span in context")
	require.False(t, hasSpanID, "span_id must not be set when no span in context")
}

// TestL_RequestIDStillWorks ensures the existing request_id injection path
// is preserved alongside new trace_id injection.
func TestL_RequestIDStillWorks(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	restore := zap.ReplaceGlobals(zap.New(core))
	defer restore()

	ctx := WithRequestID(context.Background(), "req-abc-123")
	L(ctx).Info("hello")
	require.Equal(t, 1, logs.Len())

	got := logs.All()[0].ContextMap()
	require.Equal(t, "req-abc-123", got["request_id"])
}

// TestL_RequestIDAndTraceIDBothPresent verifies all three fields appear when
// both request_id and OTel span are in context.
func TestL_RequestIDAndTraceIDBothPresent(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	restore := zap.ReplaceGlobals(zap.New(core))
	defer restore()

	traceID, _ := trace.TraceIDFromHex("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	spanID, _ := trace.SpanIDFromHex("bbbbbbbbbbbbbbbb")
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: trace.FlagsSampled,
	})
	ctx := trace.ContextWithSpanContext(
		WithRequestID(context.Background(), "req-xyz-789"),
		sc,
	)
	L(ctx).Info("hello")
	require.Equal(t, 1, logs.Len())

	got := logs.All()[0].ContextMap()
	require.Equal(t, "req-xyz-789", got["request_id"])
	require.Equal(t, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", got["trace_id"])
	require.Equal(t, "bbbbbbbbbbbbbbbb", got["span_id"])
}
