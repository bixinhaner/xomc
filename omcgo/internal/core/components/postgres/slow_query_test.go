package postgres

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// newObservableTracer builds a SlowQueryTracer wired to an in-memory zap
// observer plus a fresh prometheus registry. Tests can assert on either.
func newObservableTracer(t *testing.T, threshold time.Duration) (*SlowQueryTracer, *observer.ObservedLogs, *prometheus.Registry) {
	t.Helper()
	core, recorded := observer.New(zap.WarnLevel)
	log := zap.New(core)
	reg := prometheus.NewRegistry()
	metrics := NewSlowQueryMetrics(reg)
	tr := NewSlowQueryTracer(threshold, log, metrics)
	return tr, recorded, reg
}

// fakeStartAndElapse simulates a query executed `elapsed` ago by injecting a
// past start time directly into the context (so we don't need to actually
// sleep, which keeps the tests deterministic).
func fakeStartAndElapse(ctx context.Context, sql string, elapsed time.Duration) context.Context {
	ctx = context.WithValue(ctx, slowQueryStartKey{}, time.Now().Add(-elapsed))
	ctx = context.WithValue(ctx, slowQuerySQLKey{}, sql)
	return ctx
}

// runQuery exercises both halves of the tracer protocol with a deterministic
// elapsed time, returning the resulting CommandTag (unused here).
func runQuery(t *testing.T, tr *SlowQueryTracer, sql string, elapsed time.Duration, queryErr error) {
	t.Helper()
	ctx := fakeStartAndElapse(context.Background(), sql, elapsed)
	tr.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{
		CommandTag: pgconn.CommandTag{},
		Err:        queryErr,
	})
}

func TestSlowQueryTracer_BelowThreshold_NoEmission(t *testing.T) {
	tr, recorded, _ := newObservableTracer(t, 100*time.Millisecond)

	runQuery(t, tr, "SELECT id FROM devices WHERE id=$1", 5*time.Millisecond, nil)

	assert.Equal(t, 0, recorded.Len(), "should not log when below threshold")
	// CounterVec with no observations reports 0 for any label combination.
	assert.Equal(t, 0.0, testutil.ToFloat64(tr.metrics.Total.WithLabelValues("devices", fingerprintSQL("SELECT id FROM devices WHERE id=$1"))))
}

func TestSlowQueryTracer_AboveThreshold_LogsAndCounts(t *testing.T) {
	tr, recorded, _ := newObservableTracer(t, 50*time.Millisecond)

	const sql = "SELECT * FROM alarms_active WHERE device_sn=$1 ORDER BY raised_at DESC"
	runQuery(t, tr, sql, 250*time.Millisecond, nil)

	require.Equal(t, 1, recorded.Len(), "should emit exactly one warn log")
	entry := recorded.All()[0]
	assert.Equal(t, "slow query", entry.Message)

	fields := entry.ContextMap()
	assert.Equal(t, "alarms_active", fields["table"])
	assert.NotEmpty(t, fields["query_hash"])
	// Duration should be encoded as time.Duration via zap.
	dur, ok := fields["duration"].(time.Duration)
	require.True(t, ok, "duration should be a time.Duration field")
	assert.GreaterOrEqual(t, dur, 250*time.Millisecond)

	// Counter: lookup by labels and assert == 1.
	got := testutil.ToFloat64(tr.metrics.Total.WithLabelValues("alarms_active", fields["query_hash"].(string)))
	assert.Equal(t, 1.0, got, "counter should increment once")
}

func TestSlowQueryTracer_CounterAccumulates(t *testing.T) {
	tr, _, _ := newObservableTracer(t, 10*time.Millisecond)

	const sql = "UPDATE device_tasks SET status=$1 WHERE id=$2"
	runQuery(t, tr, sql, 100*time.Millisecond, nil)
	runQuery(t, tr, sql, 200*time.Millisecond, nil)
	runQuery(t, tr, sql, 50*time.Millisecond, nil)

	hash := fingerprintSQL(sql)
	got := testutil.ToFloat64(tr.metrics.Total.WithLabelValues("device_tasks", hash))
	assert.Equal(t, 3.0, got, "counter should accumulate identical slow queries")
}

func TestSlowQueryTracer_DifferentTables_DifferentLabels(t *testing.T) {
	tr, _, _ := newObservableTracer(t, 10*time.Millisecond)

	runQuery(t, tr, "SELECT * FROM devices WHERE serial=$1", 50*time.Millisecond, nil)
	runQuery(t, tr, "SELECT * FROM alarms_active WHERE id=$1", 50*time.Millisecond, nil)

	devicesHash := fingerprintSQL("SELECT * FROM devices WHERE serial=$1")
	alarmsHash := fingerprintSQL("SELECT * FROM alarms_active WHERE id=$1")

	devicesCount := testutil.ToFloat64(tr.metrics.Total.WithLabelValues("devices", devicesHash))
	alarmsCount := testutil.ToFloat64(tr.metrics.Total.WithLabelValues("alarms_active", alarmsHash))

	assert.Equal(t, 1.0, devicesCount)
	assert.Equal(t, 1.0, alarmsCount)
}

func TestSlowQueryTracer_QueryErrorAttached(t *testing.T) {
	tr, recorded, _ := newObservableTracer(t, 10*time.Millisecond)

	queryErr := errors.New("connection reset")
	runQuery(t, tr, "DELETE FROM mml_tasks WHERE id=$1", 50*time.Millisecond, queryErr)

	require.Equal(t, 1, recorded.Len())
	entry := recorded.All()[0]
	fields := entry.ContextMap()
	// zap.Error encodes to a string "error" field via the default error encoder.
	gotErr, ok := fields["error"].(string)
	require.True(t, ok, "error field should be present, got: %#v", fields)
	assert.Equal(t, queryErr.Error(), gotErr)
}

func TestSlowQueryTracer_RequestCancellationDoesNotCountOrWarn(t *testing.T) {
	for _, queryErr := range []error{
		context.Canceled,
		context.DeadlineExceeded,
		fmt.Errorf("dashboard query: %w", context.Canceled),
		fmt.Errorf("monitoring query: %w", context.DeadlineExceeded),
	} {
		tr, recorded, _ := newObservableTracer(t, 10*time.Millisecond)
		const sql = "SELECT * FROM pm_metrics WHERE time >= $1"

		runQuery(t, tr, sql, 100*time.Millisecond, queryErr)

		assert.Zero(t, recorded.Len(), "canceled query must not emit slow/failure warning: %v", queryErr)
		count := testutil.ToFloat64(tr.metrics.Total.WithLabelValues("pm_metrics", fingerprintSQL(sql)))
		assert.Zero(t, count, "canceled query must not inflate query counter: %v", queryErr)
	}
}

func TestSlowQueryTracer_DefaultsApplied(t *testing.T) {
	// Threshold <= 0 should fall back to DefaultSlowQueryThreshold.
	tr := NewSlowQueryTracer(0, nil, nil)
	assert.Equal(t, DefaultSlowQueryThreshold, tr.Threshold(), "zero threshold should default")

	tr2 := NewSlowQueryTracer(-1*time.Second, nil, nil)
	assert.Equal(t, DefaultSlowQueryThreshold, tr2.Threshold(), "negative threshold should default")

	// Nil logger should not panic; a query that exceeds the threshold should
	// still execute cleanly even without a logger.
	require.NotPanics(t, func() {
		runQuery(t, tr, "SELECT 1", 2*DefaultSlowQueryThreshold, nil)
	})
}

func TestSlowQueryTracer_NilMetrics_LogOnly(t *testing.T) {
	core, recorded := observer.New(zap.WarnLevel)
	tr := NewSlowQueryTracer(10*time.Millisecond, zap.New(core), nil)

	require.NotPanics(t, func() {
		runQuery(t, tr, "SELECT * FROM users", 50*time.Millisecond, nil)
	})
	assert.Equal(t, 1, recorded.Len(), "should still log when metrics are nil")
}

func TestSlowQueryTracer_TraceQueryStart_PopulatesContext(t *testing.T) {
	tr, _, _ := newObservableTracer(t, 10*time.Millisecond)

	ctx := tr.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{
		SQL: "SELECT NOW()",
	})
	startTime, ok := ctx.Value(slowQueryStartKey{}).(time.Time)
	require.True(t, ok, "start time should be in context")
	assert.WithinDuration(t, time.Now(), startTime, time.Second)

	sql, ok := ctx.Value(slowQuerySQLKey{}).(string)
	require.True(t, ok, "SQL should be in context")
	assert.Equal(t, "SELECT NOW()", sql)
}

func TestSlowQueryTracer_TraceQueryEnd_MissingStart_NoOp(t *testing.T) {
	tr, recorded, _ := newObservableTracer(t, 10*time.Millisecond)

	// No start time in context — TraceQueryEnd should bail out silently.
	require.NotPanics(t, func() {
		tr.TraceQueryEnd(context.Background(), nil, pgx.TraceQueryEndData{})
	})
	assert.Equal(t, 0, recorded.Len())
}

func TestExtractTable(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want string
	}{
		{"select from", "SELECT * FROM devices WHERE id=$1", "devices"},
		{"select with schema", "SELECT * FROM public.alarms_active", "alarms_active"},
		{"insert into", "INSERT INTO mml_tasks (id, status) VALUES ($1, $2)", "mml_tasks"},
		{"update", "UPDATE device_tasks SET status=$1 WHERE id=$2", "device_tasks"},
		{"delete from", "DELETE FROM notifications WHERE id=$1", "notifications"},
		{"join", "SELECT * FROM alarms_active a JOIN devices d ON d.serial = a.device_sn", "alarms_active"},
		{"empty", "", "unknown"},
		{"weird", "EXPLAIN ANALYZE", "unknown"},
		{"quoted", `SELECT * FROM "mml_tasks" WHERE id=$1`, "mml_tasks"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, extractTable(tc.sql))
		})
	}
}

func TestFingerprintSQL_Stability(t *testing.T) {
	a := fingerprintSQL("SELECT * FROM   devices WHERE id=$1;")
	b := fingerprintSQL("SELECT * FROM devices WHERE id=$1")
	assert.Equal(t, a, b, "whitespace and trailing semicolon should not change fingerprint")
	assert.Len(t, a, 12, "fingerprint should be 12 hex chars")

	c := fingerprintSQL("SELECT * FROM alarms_active WHERE id=$1")
	assert.NotEqual(t, a, c, "different SQL should produce different fingerprint")

	empty := fingerprintSQL("")
	assert.Equal(t, "empty", empty)
}

func TestTruncateForLog(t *testing.T) {
	long := "SELECT a, b, c, d FROM " + string(make([]byte, 600)) + " WHERE id=$1"
	got := truncateForLog(long, 50)
	assert.LessOrEqual(t, len(got), 50+len("...(truncated)"))
	assert.Contains(t, got, "...(truncated)")

	short := "SELECT 1"
	assert.Equal(t, short, truncateForLog(short, 100))
}

// chainCounterTracer records how many times each callback fires for tests.
type chainCounterTracer struct {
	starts int
	ends   int
}

func (c *chainCounterTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	c.starts++
	return ctx
}

func (c *chainCounterTracer) TraceQueryEnd(_ context.Context, _ *pgx.Conn, _ pgx.TraceQueryEndData) {
	c.ends++
}

func TestChainTracer_BothInvoked(t *testing.T) {
	a := &chainCounterTracer{}
	b := &chainCounterTracer{}

	chain := ChainTracer(a, b)
	ctx := chain.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{SQL: "SELECT 1"})
	chain.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})

	assert.Equal(t, 1, a.starts)
	assert.Equal(t, 1, a.ends)
	assert.Equal(t, 1, b.starts)
	assert.Equal(t, 1, b.ends)
}

func TestChainTracer_NilLegs(t *testing.T) {
	a := &chainCounterTracer{}

	chain := ChainTracer(a, nil)
	require.Same(t, pgx.QueryTracer(a), chain, "nil right leg should return left tracer unchanged")

	chain = ChainTracer(nil, a)
	require.Same(t, pgx.QueryTracer(a), chain, "nil left leg should return right tracer unchanged")

	assert.Nil(t, ChainTracer(nil, nil))
}

func TestWithSlowQueryTracer_RegistersMetrics(t *testing.T) {
	reg := prometheus.NewRegistry()
	tr := WithSlowQueryTracer(150*time.Millisecond, zap.NewNop(), reg)
	require.NotNil(t, tr)
	assert.Equal(t, 150*time.Millisecond, tr.Threshold())

	// The counter is registered but has no observations yet; testutil should
	// not panic when querying it via a label combination.
	got := testutil.ToFloat64(tr.metrics.Total.WithLabelValues("foo", "abc123"))
	assert.Equal(t, 0.0, got)
}
