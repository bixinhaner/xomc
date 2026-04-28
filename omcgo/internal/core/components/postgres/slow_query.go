// Package postgres provides PostgreSQL connection pool and tracer utilities.
//
// slow_query.go implements a dedicated slow-query auditing tracer that
// exposes Prometheus counters and structured zap logs for any SQL whose
// observed execution duration exceeds a configurable threshold.
//
// The tracer is intentionally separate from the existing SQLTracer / OTELSQLTracer
// so it can be composed with them (or used standalone) without disturbing the
// log levels of regular SQL traffic. Slow queries are emitted at WARN level
// regardless of the global SQL log level.
//
// Design goals
//   - Zero impact for fast queries (constant-time check on duration).
//   - Cardinality-safe metrics (we hash the SQL fingerprint and label by
//     table + truncated hash, never by raw SQL text).
//   - Composable: callers can stack the slow query tracer alongside the
//     existing SQL/OTEL tracers via ChainTracer.
package postgres

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/internal/core/components/logger"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// DefaultSlowQueryThreshold is the threshold used when callers do not specify one.
// 100ms is a pragmatic default for OLTP workloads on PostgreSQL.
const DefaultSlowQueryThreshold = 100 * time.Millisecond

// slowQueryStartKey stores the per-query start time in the request context.
type slowQueryStartKey struct{}

// slowQuerySQLKey stores the per-query SQL string in the request context.
type slowQuerySQLKey struct{}

// SlowQueryMetrics holds the Prometheus collectors emitted by SlowQueryTracer.
//
// The collectors are exported so that callers can register them with the same
// registry that exposes the rest of the application metrics.
type SlowQueryMetrics struct {
	// Total is the cumulative number of queries that exceeded the configured
	// slow-query threshold. Labelled by table (best-effort extraction) and
	// query_hash (truncated SHA-1 of the SQL fingerprint to keep cardinality
	// bounded).
	Total *prometheus.CounterVec
}

// NewSlowQueryMetrics constructs and registers the slow-query metrics with
// the provided registerer. Passing a nil registerer is supported and yields
// a tracer that still logs but skips Prometheus instrumentation.
func NewSlowQueryMetrics(reg prometheus.Registerer) *SlowQueryMetrics {
	m := &SlowQueryMetrics{
		Total: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "pgx_slow_query_total",
				Help: "Total number of PostgreSQL queries exceeding the configured slow-query threshold.",
			},
			[]string{"table", "query_hash"},
		),
	}
	if reg != nil {
		reg.MustRegister(m.Total)
	}
	return m
}

// SlowQueryTracer implements pgx.QueryTracer and emits a structured log line
// plus a Prometheus counter increment whenever a query duration exceeds the
// configured threshold. All other queries pass through silently.
type SlowQueryTracer struct {
	threshold time.Duration
	logger    *zap.Logger
	metrics   *SlowQueryMetrics
}

// NewSlowQueryTracer constructs a SlowQueryTracer.
//
// If threshold <= 0, DefaultSlowQueryThreshold is used. A nil logger falls
// back to a no-op logger (the tracer will still emit metrics if available).
// A nil metrics pointer disables Prometheus instrumentation but logs are
// still emitted when the threshold is exceeded.
func NewSlowQueryTracer(threshold time.Duration, log *zap.Logger, metrics *SlowQueryMetrics) *SlowQueryTracer {
	if threshold <= 0 {
		threshold = DefaultSlowQueryThreshold
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &SlowQueryTracer{
		threshold: threshold,
		logger:    log.Named("slow-query"),
		metrics:   metrics,
	}
}

// WithSlowQueryTracer is a helper that constructs a SlowQueryTracer with the
// metrics registered against the given registerer. It returns the tracer ready
// to be attached to a pgx connection config (or chained alongside other tracers).
//
// Example:
//
//	tracer := postgres.WithSlowQueryTracer(150*time.Millisecond, log, reg)
//	poolCfg.ConnConfig.Tracer = postgres.ChainTracer(existingTracer, tracer)
func WithSlowQueryTracer(threshold time.Duration, log *zap.Logger, reg prometheus.Registerer) *SlowQueryTracer {
	return NewSlowQueryTracer(threshold, log, NewSlowQueryMetrics(reg))
}

// Threshold returns the configured slow-query threshold (read-only accessor for tests).
func (t *SlowQueryTracer) Threshold() time.Duration { return t.threshold }

// TraceQueryStart records the start timestamp for the query.
func (t *SlowQueryTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx = context.WithValue(ctx, slowQueryStartKey{}, time.Now())
	ctx = context.WithValue(ctx, slowQuerySQLKey{}, data.SQL)
	return ctx
}

// TraceQueryEnd inspects the elapsed duration and emits a structured warning
// plus a Prometheus counter increment when the threshold is exceeded.
func (t *SlowQueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	startTime, ok := ctx.Value(slowQueryStartKey{}).(time.Time)
	if !ok {
		return
	}
	duration := time.Since(startTime)
	if duration < t.threshold {
		return
	}

	sqlStr, _ := ctx.Value(slowQuerySQLKey{}).(string)
	table := extractTable(sqlStr)
	hash := fingerprintSQL(sqlStr)

	if t.metrics != nil {
		t.metrics.Total.WithLabelValues(table, hash).Inc()
	}

	fields := []zap.Field{
		zap.Duration("duration", duration),
		zap.Duration("threshold", t.threshold),
		zap.String("table", table),
		zap.String("query_hash", hash),
		zap.String("sql", truncateForLog(sqlStr, 512)),
		zap.String("operation", sqlOperation(sqlStr)),
	}
	if requestID := logger.GetRequestID(ctx); requestID != "" {
		fields = append(fields, zap.String("request_id", requestID))
	}
	if data.CommandTag.RowsAffected() > 0 {
		fields = append(fields, zap.Int64("rows", data.CommandTag.RowsAffected()))
	}
	if data.Err != nil {
		fields = append(fields, zap.Error(data.Err))
	}
	t.logger.Warn("slow query", fields...)
}

// ChainTracer composes two pgx.QueryTracers so both observe each query.
// Either argument may be nil; nil legs are skipped silently.
func ChainTracer(a, b pgx.QueryTracer) pgx.QueryTracer {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return &chainedTracer{a: a, b: b}
}

type chainedTracer struct {
	a, b pgx.QueryTracer
}

// chainKey stores the secondary tracer's modified context so we can retrieve
// it during TraceQueryEnd.
type chainKey struct{}

func (c *chainedTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctxA := c.a.TraceQueryStart(ctx, conn, data)
	ctxB := c.b.TraceQueryStart(ctxA, conn, data)
	return context.WithValue(ctxB, chainKey{}, ctxA)
}

func (c *chainedTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	ctxA, ok := ctx.Value(chainKey{}).(context.Context)
	if !ok {
		ctxA = ctx
	}
	c.b.TraceQueryEnd(ctx, conn, data)
	c.a.TraceQueryEnd(ctxA, conn, data)
}

// fingerprintSQL returns a stable, low-cardinality identifier for the SQL
// statement: the first 12 hex characters of the SHA-1 of the normalised SQL.
// Whitespace and trailing semicolons are collapsed so logically identical
// queries hash to the same value.
func fingerprintSQL(sql string) string {
	if sql == "" {
		return "empty"
	}
	normalised := strings.TrimSpace(strings.Join(strings.Fields(sql), " "))
	normalised = strings.TrimRight(normalised, ";")
	sum := sha1.Sum([]byte(normalised))
	hexed := hex.EncodeToString(sum[:])
	return hexed[:12]
}

// extractTable performs a best-effort, allocation-light extraction of the
// primary table name from a SQL statement. It recognises SELECT, INSERT INTO,
// UPDATE, and DELETE FROM forms and falls back to "unknown" so the metric
// label remains bounded.
func extractTable(sql string) string {
	if sql == "" {
		return "unknown"
	}
	tokens := strings.Fields(sql)
	if len(tokens) == 0 {
		return "unknown"
	}

	// Walk the tokens looking for a keyword followed by a table name.
	for i := 0; i < len(tokens)-1; i++ {
		switch strings.ToUpper(tokens[i]) {
		case "FROM", "JOIN", "INTO", "UPDATE":
			candidate := stripIdentifier(tokens[i+1])
			if candidate != "" {
				return candidate
			}
		}
	}
	return "unknown"
}

// stripIdentifier removes common SQL identifier decorations (quotes, schema
// prefixes, parentheses) so two queries against the same table produce the
// same metric label.
func stripIdentifier(token string) string {
	token = strings.TrimSpace(token)
	token = strings.Trim(token, "\"`(),;")
	if token == "" {
		return ""
	}
	// Drop leading schema if present: "public.devices" -> "devices".
	if idx := strings.IndexByte(token, '.'); idx >= 0 && idx < len(token)-1 {
		token = token[idx+1:]
	}
	return strings.ToLower(token)
}

// truncateForLog truncates SQL text for log emission so we do not spam the log
// pipeline with very long statements.
func truncateForLog(sql string, max int) string {
	if len(sql) <= max {
		return sql
	}
	return sql[:max] + "...(truncated)"
}
