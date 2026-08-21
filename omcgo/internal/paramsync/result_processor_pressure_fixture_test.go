package paramsync

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/event"
)

type pressureTraceContextKey struct{}

type pressureTraceQuery struct {
	sql     string
	started time.Time
}

type paramSyncPressureStats struct {
	SQLTotal                    int
	StagingUpsertSQL            int
	OfficialDeviceParameterSQL  int
	OfficialDeviceParameterRows int64
	CoverageDeleteSQL           int
	Elapsed                     time.Duration
}

type paramSyncPressureTracer struct {
	mu    sync.Mutex
	stats paramSyncPressureStats
}

func (t *paramSyncPressureTracer) TraceQueryStart(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryStartData,
) context.Context {
	return context.WithValue(ctx, pressureTraceContextKey{}, pressureTraceQuery{
		sql:     data.SQL,
		started: time.Now(),
	})
}

func (t *paramSyncPressureTracer) TraceQueryEnd(
	ctx context.Context,
	_ *pgx.Conn,
	data pgx.TraceQueryEndData,
) {
	trace, _ := ctx.Value(pressureTraceContextKey{}).(pressureTraceQuery)
	sqlText := strings.ToLower(trace.sql)
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stats.SQLTotal++
	t.stats.Elapsed += time.Since(trace.started)
	if strings.Contains(sqlText, "insert into parameter_sync_staging_values") {
		t.stats.StagingUpsertSQL++
	}
	if strings.Contains(sqlText, "insert into device_parameters") {
		t.stats.OfficialDeviceParameterSQL++
		t.stats.OfficialDeviceParameterRows += data.CommandTag.RowsAffected()
	}
	if strings.Contains(sqlText, "delete from device_parameters") {
		t.stats.CoverageDeleteSQL++
	}
}

func (t *paramSyncPressureTracer) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stats = paramSyncPressureStats{}
}

func (t *paramSyncPressureTracer) Snapshot() paramSyncPressureStats {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.stats
}

func newParamSyncPressureTestPool(t *testing.T, tracer pgx.QueryTracer) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PG_URL")
	if dsn == "" {
		dsn = "postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable"
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Skipf("invalid PostgreSQL test configuration: %v", err)
	}
	config.ConnConfig.Tracer = tracer
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Skipf("no PostgreSQL available: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("no PostgreSQL available: %v", err)
	}
	return pool
}

func TestFullResultProcessingPressureFixtureRecordsWriteAmplification(t *testing.T) {
	ctx := context.Background()
	tracer := &paramSyncPressureTracer{}
	pool := newParamSyncPressureTestPool(t, tracer)
	request := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	unchangedPath := "Device.X_OMC.Issue365.Pressure.Unchanged"
	runID := insertFullResultProcessorRunWithCoveragePaths(t, pool, request, 3, []string{
		"Device.X_OMC.Issue365.Pressure.NewA",
		unchangedPath,
		"Device.X_OMC.Issue365.Pressure.NewB",
		"Device.X_OMC.Issue365.Pressure.Stale",
	})
	_, err := pool.Exec(ctx, `
INSERT INTO device_parameters (
  device_id, parameter_path, parameter_value, parameter_type, writable,
  last_updated_at, fap_instance, param_group
) VALUES ($1, $2, 'stable', 'string', false, now() - interval '1 hour', 0, 'other')`,
		request.DeviceID, unchangedPath,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM device_parameters WHERE device_id=$1`, request.DeviceID)
	})

	tasks := []struct {
		commandKey string
		path       string
		value      string
	}{
		{"issue365-pressure-1", "Device.X_OMC.Issue365.Pressure.NewA", "alpha"},
		{"issue365-pressure-2", unchangedPath, "stable"},
		{"issue365-pressure-3", "Device.X_OMC.Issue365.Pressure.NewB", "bravo"},
	}
	payloads := make([]event.ParamSyncTaskResultPayload, 0, len(tasks))
	for _, item := range tasks {
		taskRow := insertCompletedParamSyncTaskForProcessorTest(t, pool, request, runID,
			item.commandKey,
			item.path,
			item.value,
		)
		payloads = append(payloads, event.ParamSyncTaskResultPayload{
			RunID: runID, RequestID: request.ID, TaskID: taskRow.ID,
			DeviceID: request.DeviceID, DeviceSN: request.DeviceSN,
			EventID: uuid.NewString(), Success: true,
		})
	}

	tracer.Reset()
	started := time.Now()
	processor := NewPGResultProcessor(pool)
	for _, payload := range payloads {
		_, err := processor.Process(ctx, payload)
		require.NoError(t, err)
	}
	wallClock := time.Since(started)
	stats := tracer.Snapshot()
	t.Logf("issue365 pressure fixture: sql_total=%d staging_upsert_sql=%d official_device_parameter_sql=%d official_device_parameter_rows=%d coverage_delete_sql=%d traced_sql_duration=%s wall_clock=%s",
		stats.SQLTotal,
		stats.StagingUpsertSQL,
		stats.OfficialDeviceParameterSQL,
		stats.OfficialDeviceParameterRows,
		stats.CoverageDeleteSQL,
		stats.Elapsed,
		wallClock,
	)

	assert.Equal(t, len(tasks), stats.StagingUpsertSQL)
	assert.Equal(t, 1, stats.OfficialDeviceParameterSQL)
	assert.EqualValues(t, 2, stats.OfficialDeviceParameterRows,
		"unchanged parameter must not become a useless UPDATE row")
	assert.Equal(t, 1, stats.CoverageDeleteSQL)
	assert.Positive(t, stats.SQLTotal)
	assert.Positive(t, wallClock)
}
