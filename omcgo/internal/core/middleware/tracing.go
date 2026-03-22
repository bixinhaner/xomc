package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/omcgo/omcgo/middleware"

// Tracing returns a Gin middleware that creates OpenTelemetry spans for HTTP requests.
// When tracing is disabled (no-op TracerProvider), this middleware has zero overhead.
//
// Span naming: "HTTP {method} {route}" (e.g., "HTTP GET /api/v1/devices")
// Attributes: http.method, http.url, http.route, http.status_code, http.scheme
func Tracing(serviceName string) gin.HandlerFunc {
	tracer := otel.Tracer(tracerName)
	propagator := otel.GetTextMapPropagator()

	return func(c *gin.Context) {
		// Extract parent span context from incoming request headers
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// Determine span name from route pattern (falls back to path if no route matched yet)
		route := c.FullPath()
		if route == "" {
			route = c.Request.URL.Path
		}
		spanName := fmt.Sprintf("HTTP %s %s", c.Request.Method, route)

		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethodKey.String(c.Request.Method),
				semconv.HTTPSchemeKey.String(schemeFromRequest(c)),
				semconv.HTTPTargetKey.String(c.Request.URL.Path),
				attribute.String("http.route", route),
				attribute.String("net.host.name", c.Request.Host),
			),
		)
		defer span.End()

		// Replace request context so downstream handlers see the span
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		// Record response status
		status := c.Writer.Status()
		span.SetAttributes(semconv.HTTPStatusCodeKey.Int(status))

		if status >= 500 {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", status))
		} else {
			span.SetStatus(codes.Ok, "")
		}
	}
}

func schemeFromRequest(c *gin.Context) string {
	if c.Request.TLS != nil {
		return "https"
	}
	return "http"
}
