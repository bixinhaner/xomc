// Package tracing provides OpenTelemetry span helpers for business operations.
// All helpers are safe to call when tracing is disabled (no-op TracerProvider);
// in that case spans are not recorded and overhead is negligible.
package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Tracer names for each subsystem — used to group spans by component.
const (
	ACSTracerName       = "github.com/omcgo/omcgo/acs"
	DeviceTracerName    = "github.com/omcgo/omcgo/device"
	ProvisionTracerName = "github.com/omcgo/omcgo/provision"
	TaskTracerName      = "github.com/omcgo/omcgo/task"
)

// StartSpan creates a new span with the given tracer name, span name, and optional attributes.
// Returns the updated context and the span. The caller must call span.End() when done.
//
// Usage:
//
//	ctx, span := tracing.StartSpan(ctx, tracing.ACSTracerName, "ACS HandleInform",
//	    attribute.String("device_sn", deviceSN))
//	defer span.End()
func StartSpan(ctx context.Context, tracerName, spanName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, spanName,
		trace.WithAttributes(attrs...),
	)
	return ctx, span
}

// RecordError records an error on the span and sets the span status to Error.
// Safe to call with a nil span (no-op).
func RecordError(span trace.Span, err error) {
	if err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

// SetOK marks the span status as OK.
func SetOK(span trace.Span) {
	span.SetStatus(codes.Ok, "")
}
