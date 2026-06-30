package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/internal/core/components/logger"
	"go.uber.org/zap"
)

// Context keys for storing query data
type queryStartTimeKey struct{}
type querySQLKey struct{}
type queryArgsKey struct{}
type batchStartTimeKey struct{}

// sanitizeParams converts []byte to string for better log readability.
// JSON/JSONB arguments are often []byte and would otherwise be base64-encoded.
func sanitizeParams(args []any) []any {
	result := make([]any, len(args))
	for i, arg := range args {
		switch v := arg.(type) {
		case []byte:
			// Convert byte slice to string for readable JSON/JSONB output
			result[i] = string(v)
		default:
			result[i] = arg
		}
	}
	return result
}

// SQLTracer implements pgx.Tracer for SQL logging with request_id support.
type SQLTracer struct {
	logger        *zap.Logger
	logParams     bool
	slowThreshold time.Duration
}

// NewSQLTracer creates a new SQL tracer.
func NewSQLTracer(log *zap.Logger, logParams bool, slowThresholdMs int) *SQLTracer {
	var threshold time.Duration
	if slowThresholdMs > 0 {
		threshold = time.Duration(slowThresholdMs) * time.Millisecond
	}
	return &SQLTracer{
		logger:        log.Named("sql"),
		logParams:     logParams,
		slowThreshold: threshold,
	}
}

// TraceQueryStart is called at the beginning of a query.
func (t *SQLTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx = context.WithValue(ctx, queryStartTimeKey{}, time.Now())
	ctx = context.WithValue(ctx, querySQLKey{}, data.SQL)
	if t.logParams && len(data.Args) > 0 {
		ctx = context.WithValue(ctx, queryArgsKey{}, data.Args)
	}
	return ctx
}

// TraceQueryEnd is called at the end of a query.
func (t *SQLTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	startTime, ok := ctx.Value(queryStartTimeKey{}).(time.Time)
	if !ok {
		return
	}

	duration := time.Since(startTime)

	// Get SQL from context
	sqlStr, _ := ctx.Value(querySQLKey{}).(string)

	fields := []zap.Field{
		zap.String("sql", sqlStr),
		zap.Duration("duration", duration),
	}

	// Add request_id if present in context
	if requestID := logger.GetRequestID(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}

	// Log parameters if enabled
	if t.logParams {
		if args, ok := ctx.Value(queryArgsKey{}).([]any); ok && len(args) > 0 {
			fields = append(fields, zap.Any("params", sanitizeParams(args)))
		}
	}

	// Log affected rows
	if data.CommandTag.RowsAffected() > 0 {
		fields = append(fields, zap.Int64("rows", data.CommandTag.RowsAffected()))
	}

	// Log error if present
	if data.Err != nil {
		fields = append(fields, zap.Error(data.Err))
		t.logger.Error("SQL ERROR", fields...)
		return
	}

	// Log based on duration
	if t.slowThreshold > 0 && duration > t.slowThreshold {
		t.logger.Warn("SLOW SQL", fields...)
	} else {
		t.logger.Debug("SQL", fields...)
	}
}

// TraceBatchStart is called at the beginning of a batch.
func (t *SQLTracer) TraceBatchStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceBatchStartData) context.Context {
	return context.WithValue(ctx, batchStartTimeKey{}, time.Now())
}

// TraceBatchQuery is called for each query in a batch.
func (t *SQLTracer) TraceBatchQuery(ctx context.Context, _ *pgx.Conn, data pgx.TraceBatchQueryData) {
	// Individual batch query logging - log at debug level
	if t.logger.Core().Enabled(zap.DebugLevel) {
		fields := []zap.Field{
			zap.String("sql", data.SQL),
		}
		if requestID := logger.GetRequestID(ctx); requestID != "" {
			fields = append(fields, zap.String("request_id", requestID))
		}
		t.logger.Debug("BATCH QUERY", fields...)
	}
}

// TraceBatchEnd is called at the end of a batch.
func (t *SQLTracer) TraceBatchEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceBatchEndData) {
	startTime, ok := ctx.Value(batchStartTimeKey{}).(time.Time)
	if !ok {
		return
	}

	duration := time.Since(startTime)
	fields := []zap.Field{
		zap.String("sql", "BATCH"),
		zap.Duration("duration", duration),
	}

	// Add request_id if present in context
	if requestID := logger.GetRequestID(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}

	// Log error if present
	if data.Err != nil {
		fields = append(fields, zap.Error(data.Err))
		t.logger.Error("BATCH SQL ERROR", fields...)
		return
	}

	t.logger.Debug("BATCH SQL", fields...)
}

// TraceCopyFromStart is called at the start of a COPY operation.
func (t *SQLTracer) TraceCopyFromStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromStartData) context.Context {
	return context.WithValue(ctx, queryStartTimeKey{}, time.Now())
}

// TraceCopyFromEnd is called at the end of a COPY operation.
func (t *SQLTracer) TraceCopyFromEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromEndData) {
	startTime, ok := ctx.Value(queryStartTimeKey{}).(time.Time)
	if !ok {
		return
	}

	duration := time.Since(startTime)
	fields := []zap.Field{
		zap.String("sql", "COPY"),
		zap.Duration("duration", duration),
		zap.Int64("rows", data.CommandTag.RowsAffected()),
	}

	if requestID := logger.GetRequestID(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}

	if data.Err != nil {
		fields = append(fields, zap.Error(data.Err))
		t.logger.Error("COPY SQL ERROR", fields...)
		return
	}

	t.logger.Debug("COPY SQL", fields...)
}

// TracePrepareStart is called when preparing a statement.
func (t *SQLTracer) TracePrepareStart(ctx context.Context, _ *pgx.Conn, data pgx.TracePrepareStartData) context.Context {
	return ctx
}

// TracePrepareEnd is called after preparing a statement.
func (t *SQLTracer) TracePrepareEnd(ctx context.Context, _ *pgx.Conn, data pgx.TracePrepareEndData) {
	// Statement preparation is usually fast and one-time, log at debug level
	if data.Err != nil {
		t.logger.Debug("PREPARE SQL ERROR", zap.Error(data.Err))
	}
}

// TraceConnectStart is called at the start of connection establishment.
func (t *SQLTracer) TraceConnectStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceConnectStartData) context.Context {
	return ctx
}

// TraceConnectEnd is called after connection establishment.
func (t *SQLTracer) TraceConnectEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceConnectEndData) {
	if data.Err != nil {
		t.logger.Warn("DATABASE CONNECT ERROR", zap.Error(data.Err))
	}
}
