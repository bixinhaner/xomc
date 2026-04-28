package tracing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/attribute"
)

func Test_StartSpan_NoTracerProvider(t *testing.T) {
	// With no tracer provider configured (default no-op), StartSpan should still
	// return a non-nil context and span without panicking.
	ctx, span := StartSpan(context.Background(), ACSTracerName, "TestOp",
		attribute.String("device_sn", "DEV-001"))

	assert.NotNil(t, ctx)
	assert.NotNil(t, span)

	// span.End() must not panic
	span.End()
}

func Test_StartSpan_WithoutAttributes(t *testing.T) {
	ctx, span := StartSpan(context.Background(), DeviceTracerName, "NoAttrs")

	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
	span.End()
}

func Test_StartSpan_WithMultipleAttributes(t *testing.T) {
	ctx, span := StartSpan(context.Background(), TaskTracerName, "Multi",
		attribute.String("a", "1"),
		attribute.Int("b", 2),
		attribute.Bool("c", true),
	)
	defer span.End()

	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
}

func Test_RecordError_Nil(t *testing.T) {
	// RecordError with nil should be a safe no-op.
	_, span := StartSpan(context.Background(), ProvisionTracerName, "TestNil")
	defer span.End()

	// Should not panic.
	RecordError(span, nil)
}

func Test_RecordError_NonNil(t *testing.T) {
	_, span := StartSpan(context.Background(), ACSTracerName, "TestErr")
	defer span.End()

	err := errors.New("simulated failure")
	RecordError(span, err)
	// We can't easily assert on the no-op span state, but at least
	// confirm no panic.
}

func Test_SetOK(t *testing.T) {
	_, span := StartSpan(context.Background(), DeviceTracerName, "TestOK")
	defer span.End()

	SetOK(span)
	// No panic expected.
}

func Test_TracerNames(t *testing.T) {
	// Smoke-test tracer name constants are non-empty and unique.
	names := []string{
		ACSTracerName,
		DeviceTracerName,
		ProvisionTracerName,
		TaskTracerName,
	}
	seen := make(map[string]bool)
	for _, n := range names {
		assert.NotEmpty(t, n, "tracer name must not be empty")
		assert.False(t, seen[n], "tracer name %q must be unique", n)
		seen[n] = true
	}
}
