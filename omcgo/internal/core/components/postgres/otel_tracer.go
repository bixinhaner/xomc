package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

const dbTracerName = "github.com/omcgo/omcgo/postgres"

// Context key to store the OTEL span for query tracing.
type otelQuerySpanKey struct{}
type otelBatchSpanKey struct{}

// OTELSQLTracer wraps the existing SQLTracer and adds OpenTelemetry span creation.
// When tracing is disabled (no-op TracerProvider), the OTEL overhead is negligible.
type OTELSQLTracer struct {
	*SQLTracer
	tracer trace.Tracer
}

// NewOTELSQLTracer creates a composite tracer that logs SQL and creates OTEL spans.
func NewOTELSQLTracer(log *zap.Logger, logParams bool, slowThresholdMs int) *OTELSQLTracer {
	return &OTELSQLTracer{
		SQLTracer: NewSQLTracer(log, logParams, slowThresholdMs),
		tracer:    otel.Tracer(dbTracerName),
	}
}

// TraceQueryStart starts both logging and an OTEL span for the query.
func (t *OTELSQLTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	// Start OTEL span
	spanName := sqlSpanName(data.SQL)
	ctx, span := t.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.DBSystemPostgreSQL,
			semconv.DBStatementKey.String(truncateSQL(data.SQL, 256)),
			attribute.String("db.operation", sqlOperation(data.SQL)),
		),
	)
	ctx = context.WithValue(ctx, otelQuerySpanKey{}, span)

	// Delegate to the base tracer for logging
	return t.SQLTracer.TraceQueryStart(ctx, conn, data)
}

// TraceQueryEnd ends both the OTEL span and the base logging.
func (t *OTELSQLTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	// End OTEL span
	if span, ok := ctx.Value(otelQuerySpanKey{}).(trace.Span); ok {
		if data.Err != nil {
			span.RecordError(data.Err)
			span.SetStatus(codes.Error, data.Err.Error())
		}
		if data.CommandTag.RowsAffected() > 0 {
			span.SetAttributes(attribute.Int64("db.rows_affected", data.CommandTag.RowsAffected()))
		}
		span.End()
	}

	// Delegate to base tracer for logging
	t.SQLTracer.TraceQueryEnd(ctx, conn, data)
}

// TraceBatchStart starts both logging and an OTEL span for the batch.
func (t *OTELSQLTracer) TraceBatchStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchStartData) context.Context {
	ctx, span := t.tracer.Start(ctx, "DB BATCH",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.DBSystemPostgreSQL,
			attribute.String("db.operation", "BATCH"),
		),
	)
	ctx = context.WithValue(ctx, otelBatchSpanKey{}, span)
	return t.SQLTracer.TraceBatchStart(ctx, conn, data)
}

// TraceBatchEnd ends both the OTEL span and the base logging.
func (t *OTELSQLTracer) TraceBatchEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchEndData) {
	if span, ok := ctx.Value(otelBatchSpanKey{}).(trace.Span); ok {
		if data.Err != nil {
			span.RecordError(data.Err)
			span.SetStatus(codes.Error, data.Err.Error())
		}
		span.End()
	}
	t.SQLTracer.TraceBatchEnd(ctx, conn, data)
}

// TraceCopyFromStart starts both logging and an OTEL span for the COPY operation.
func (t *OTELSQLTracer) TraceCopyFromStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceCopyFromStartData) context.Context {
	ctx, span := t.tracer.Start(ctx, "DB COPY",
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.DBSystemPostgreSQL,
			attribute.String("db.operation", "COPY"),
			attribute.String("db.sql.table", data.TableName.Sanitize()),
		),
	)
	ctx = context.WithValue(ctx, otelQuerySpanKey{}, span)
	return t.SQLTracer.TraceCopyFromStart(ctx, conn, data)
}

// TraceCopyFromEnd ends both the OTEL span and the base logging.
func (t *OTELSQLTracer) TraceCopyFromEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceCopyFromEndData) {
	if span, ok := ctx.Value(otelQuerySpanKey{}).(trace.Span); ok {
		if data.Err != nil {
			span.RecordError(data.Err)
			span.SetStatus(codes.Error, data.Err.Error())
		}
		if data.CommandTag.RowsAffected() > 0 {
			span.SetAttributes(attribute.Int64("db.rows_affected", data.CommandTag.RowsAffected()))
		}
		span.End()
	}
	t.SQLTracer.TraceCopyFromEnd(ctx, conn, data)
}

// sqlSpanName extracts a readable span name from the SQL statement.
// Example: "SELECT * FROM devices WHERE ..." -> "DB SELECT"
func sqlSpanName(sql string) string {
	op := sqlOperation(sql)
	if op == "" {
		return "DB QUERY"
	}
	return "DB " + op
}

// sqlOperation extracts the SQL operation (SELECT, INSERT, UPDATE, DELETE, etc.) from a statement.
func sqlOperation(sql string) string {
	trimmed := strings.TrimSpace(sql)
	if trimmed == "" {
		return ""
	}
	// Find first word
	idx := strings.IndexByte(trimmed, ' ')
	if idx < 0 {
		return strings.ToUpper(trimmed)
	}
	return strings.ToUpper(trimmed[:idx])
}

// truncateSQL truncates a SQL statement to the given max length for span attributes.
func truncateSQL(sql string, maxLen int) string {
	if len(sql) <= maxLen {
		return sql
	}
	return sql[:maxLen] + "..."
}
