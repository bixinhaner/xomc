package paramsync

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/event"
	devicepkg "github.com/omcgo/omcgo/internal/device"
	"github.com/omcgo/omcgo/internal/task"
	"github.com/omcgo/omcgo/pkg/tr069"
)

func TestDeviceParameterWriteLockSerializesConcurrentWriters(t *testing.T) {
	ctx := context.Background()
	pool := newParamSyncTestPool(t)
	deviceID := uuid.New()

	first, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = first.Rollback(context.Background()) }()
	require.NoError(t, devicepkg.AcquireParameterWriteLocks(ctx, first, deviceID))

	second, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = second.Rollback(context.Background()) }()
	waitCtx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
	defer cancel()
	err = devicepkg.AcquireParameterWriteLocks(waitCtx, second, deviceID)
	require.Error(t, err, "a concurrent writer for the same device must wait")
	require.ErrorIs(t, err, context.DeadlineExceeded)

	require.NoError(t, first.Rollback(ctx))
	third, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = third.Rollback(context.Background()) }()
	require.NoError(t, devicepkg.AcquireParameterWriteLocks(ctx, third, deviceID))
}

func TestFullResultProcessingMergesAndProjectsOnlyAtRunCompletion(t *testing.T) {
	ctx := context.Background()
	pool := newParamSyncTestPool(t)
	request := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	runID := insertFullResultProcessorRun(t, pool, request, 2)
	processor := NewPGResultProcessor(pool)

	firstTask := insertCompletedParamSyncTaskForProcessorTest(t, pool, request, runID,
		"issue365-full-batch-1",
		"Device.X_OMC.Issue365.ParamA",
		"alpha",
	)
	firstPayload := event.ParamSyncTaskResultPayload{
		RunID: runID, RequestID: request.ID, TaskID: firstTask.ID,
		DeviceID: request.DeviceID, DeviceSN: request.DeviceSN,
		EventID: uuid.NewString(), Success: true,
	}
	firstOutcome, err := processor.Process(ctx, firstPayload)
	require.NoError(t, err)
	require.False(t, firstOutcome.Finalized)

	assertDeviceParameterCount(t, pool, request.DeviceID, 0)
	assertStagingValueCount(t, pool, runID, 1)
	projection := &recordingFullRunProjection{}
	projector := NewCompletionProjector(pool, nil, projection)
	projected, err := projector.ReconcilePending(ctx, 100)
	require.NoError(t, err)
	assert.Zero(t, projected)
	assert.Empty(t, projection.seen)

	secondTask := insertCompletedParamSyncTaskForProcessorTest(t, pool, request, runID,
		"issue365-full-batch-2",
		"Device.X_OMC.Issue365.ParamB",
		"bravo",
	)
	secondPayload := event.ParamSyncTaskResultPayload{
		RunID: runID, RequestID: request.ID, TaskID: secondTask.ID,
		DeviceID: request.DeviceID, DeviceSN: request.DeviceSN,
		EventID: uuid.NewString(), Success: true,
	}
	secondOutcome, err := processor.Process(ctx, secondPayload)
	require.NoError(t, err)
	require.True(t, secondOutcome.Finalized)

	assertDeviceParameterCount(t, pool, request.DeviceID, 2)
	assertStagingValueCount(t, pool, runID, 0)
	assertTerminalOutboxCount(t, pool, runID, 1)

	projected, err = projector.ReconcilePending(ctx, 100)
	require.NoError(t, err)
	assert.Equal(t, 1, projected)
	assert.Equal(t, []uuid.UUID{request.DeviceID}, projection.seen)

	duplicateOutcome, err := processor.Process(ctx, firstPayload)
	require.NoError(t, err)
	require.True(t, duplicateOutcome.Duplicate)
	assertDeviceParameterCount(t, pool, request.DeviceID, 2)
	assertTerminalOutboxCount(t, pool, runID, 1)

	projected, err = projector.ReconcilePending(ctx, 100)
	require.NoError(t, err)
	assert.Zero(t, projected)
	assert.Equal(t, []uuid.UUID{request.DeviceID}, projection.seen)
}

func TestFullResultProcessingDoesNotTouchUnchangedDeviceParameter(t *testing.T) {
	ctx := context.Background()
	pool := newParamSyncTestPool(t)
	request := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	runID := insertFullResultProcessorRun(t, pool, request, 1)
	parameterPath := "Device.X_OMC.Issue365.Unchanged"
	originalUpdatedAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
	_, err := pool.Exec(ctx, `
INSERT INTO device_parameters (
  device_id, parameter_path, parameter_value, parameter_type, writable,
  last_updated_at, fap_instance, param_group
) VALUES ($1, $2, 'stable', 'string', false, $3, 0, 'other')`,
		request.DeviceID, parameterPath, originalUpdatedAt,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM device_parameters WHERE device_id=$1`, request.DeviceID)
	})

	taskRow := insertCompletedParamSyncTaskForProcessorTest(t, pool, request, runID,
		"issue365-full-unchanged",
		parameterPath,
		"stable",
	)
	_, err = NewPGResultProcessor(pool).Process(ctx, event.ParamSyncTaskResultPayload{
		RunID: runID, RequestID: request.ID, TaskID: taskRow.ID,
		DeviceID: request.DeviceID, DeviceSN: request.DeviceSN,
		EventID: uuid.NewString(), Success: true,
	})
	require.NoError(t, err)

	var gotUpdatedAt time.Time
	require.NoError(t, pool.QueryRow(ctx, `
SELECT last_updated_at
FROM device_parameters
WHERE device_id=$1 AND parameter_path=$2`,
		request.DeviceID, parameterPath,
	).Scan(&gotUpdatedAt))
	assert.True(t, gotUpdatedAt.Equal(originalUpdatedAt),
		"unchanged full merge must not advance last_updated_at: got %s want %s",
		gotUpdatedAt, originalUpdatedAt)
}

func insertFullResultProcessorRun(
	t *testing.T,
	pool *pgxpool.Pool,
	request *SyncRequest,
	expectedTasks int,
) uuid.UUID {
	t.Helper()
	return insertFullResultProcessorRunWithCoveragePaths(t, pool, request, expectedTasks, nil)
}

func insertFullResultProcessorRunWithCoveragePaths(
	t *testing.T,
	pool *pgxpool.Pool,
	request *SyncRequest,
	expectedTasks int,
	coveragePaths []string,
) uuid.UUID {
	t.Helper()
	runID := uuid.New()
	mappings := make([]FrozenMapping, 0, len(coveragePaths))
	for _, path := range coveragePaths {
		mappings = append(mappings, FrozenMapping{
			StandardPath: path,
			PrivatePath:  path,
			Access:       "readOnly",
			DataType:     "string",
			IsStorable:   true,
		})
	}
	coverage, err := json.Marshal([]CoverageScope{{
		Path:     "Device.X_OMC.Issue365.",
		Complete: true,
		Subtree:  true,
		Mappings: mappings,
	}})
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `
INSERT INTO parameter_sync_runs (
  id, request_id, device_id, device_sn, trigger_reason, sync_scope,
  status, coverage, expected_task_count, terminal_task_count,
  processed_task_count, failed_task_count
) VALUES ($1,$2,$3,$4,'manual','full','waiting_device',$5,$6,0,0,0)`,
		runID, request.ID, request.DeviceID, request.DeviceSN, coverage, expectedTasks,
	)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `
UPDATE parameter_sync_requests
SET run_id=$2, active_run_id=$2
WHERE id=$1`,
		request.ID, runID,
	)
	require.NoError(t, err)
	return runID
}

func insertCompletedParamSyncTaskForProcessorTest(
	t *testing.T,
	pool *pgxpool.Pool,
	request *SyncRequest,
	runID uuid.UUID,
	commandKey, parameterPath, value string,
) *task.Task {
	t.Helper()
	taskRow := task.NewTask(&task.CreateTaskRequest{
		DeviceSN:   request.DeviceSN,
		Method:     "GetParameterValues",
		Params:     []byte(`{"names":["` + parameterPath + `"]}`),
		CommandKey: commandKey + "-" + uuid.NewString(),
		Source:     task.TaskSourceParamSync,
		SourceID:   runID.String(),
		CreatorID:  request.ID.String(),
	})
	require.NoError(t, task.NewPgTaskRepository(pool).Create(context.Background(), taskRow))
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM device_tasks WHERE id=$1`, taskRow.ID)
	})
	result, err := json.Marshal(storedTaskResult{
		PrivateParameterValues: []tr069.ParameterValueStruct{{
			Name:  parameterPath,
			Value: value,
			Type:  "xsd:string",
		}},
	})
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `
UPDATE device_tasks
SET status='completed', completed_at=now(), result=$2
WHERE id=$1`,
		taskRow.ID, result,
	)
	require.NoError(t, err)
	return taskRow
}

func assertDeviceParameterCount(t *testing.T, pool *pgxpool.Pool, deviceID uuid.UUID, want int) {
	t.Helper()
	var got int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT count(*) FROM device_parameters WHERE device_id=$1`,
		deviceID,
	).Scan(&got))
	assert.Equal(t, want, got)
}

func assertStagingValueCount(t *testing.T, pool *pgxpool.Pool, runID uuid.UUID, want int) {
	t.Helper()
	var got int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT count(*) FROM parameter_sync_staging_values WHERE run_id=$1`,
		runID,
	).Scan(&got))
	assert.Equal(t, want, got)
}

func assertTerminalOutboxCount(t *testing.T, pool *pgxpool.Pool, runID uuid.UUID, want int) {
	t.Helper()
	var got int
	require.NoError(t, pool.QueryRow(context.Background(), `
SELECT count(*) FROM parameter_sync_outbox
WHERE aggregate_id=$1 AND event_type=$2`,
		runID, event.SubjectParamSyncRunCompleted,
	).Scan(&got))
	assert.Equal(t, want, got)
}

func TestConvergeRunTxRepairsAuthoritativeCountsAndFinalizes(t *testing.T) {
	ctx := context.Background()
	pool := newParamSyncTestPool(t)
	request := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	runID := uuid.New()
	_, err := pool.Exec(ctx, `
INSERT INTO parameter_sync_runs (
  id, request_id, device_id, device_sn, trigger_reason, sync_scope,
  status, expected_task_count, terminal_task_count, processed_task_count,
  failed_task_count, started_at
) VALUES ($1,$2,$3,$4,'manual','readback','executing',0,0,0,0,now()-interval '2 minutes')`,
		runID, request.ID, request.DeviceID, request.DeviceSN,
	)
	require.NoError(t, err)

	taskRow := task.NewTask(&task.CreateTaskRequest{
		DeviceSN: request.DeviceSN, Method: "GetParameterValues",
		Params: []byte(`{"names":[]}`), CommandKey: "converge-authoritative-counts",
		Source: task.TaskSourceParamSync, SourceID: runID.String(), CreatorID: request.ID.String(),
	})
	require.NoError(t, task.NewPgTaskRepository(pool).Create(ctx, taskRow))
	_, err = pool.Exec(ctx, `
UPDATE device_tasks SET status='completed', completed_at=now(), result='{}'::jsonb
WHERE id=$1`, taskRow.ID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
INSERT INTO parameter_sync_task_results (
  run_id, task_id, event_id, success, result_ref, status, processed_at, created_at
) VALUES ($1,$2,$3,true,$4,'processed',now(),now())`,
		runID, taskRow.ID, uuid.NewString(), "device_tasks:"+taskRow.ID,
	)
	require.NoError(t, err)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	result, err := convergeRunTx(ctx, tx, runID, time.Now().UTC())
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))
	require.True(t, result.Finalized)
	require.True(t, result.Drift)
	require.Equal(t, RunStatusSucceeded, result.Status)

	var status RunStatus
	var expected, terminal, processed int
	require.NoError(t, pool.QueryRow(ctx, `
SELECT status, expected_task_count, terminal_task_count, processed_task_count
FROM parameter_sync_runs WHERE id=$1`, runID).Scan(
		&status, &expected, &terminal, &processed,
	))
	require.Equal(t, RunStatusSucceeded, status)
	require.Equal(t, []int{1, 1, 1}, []int{expected, terminal, processed})
}

func TestConvergeRunTxDoesNotFinalizePartiallyDispatchedPlan(t *testing.T) {
	ctx := context.Background()
	pool := newParamSyncTestPool(t)
	request := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	runID := uuid.New()
	_, err := pool.Exec(ctx, `
INSERT INTO parameter_sync_runs (
  id, request_id, device_id, device_sn, trigger_reason, sync_scope,
  status, expected_task_count, terminal_task_count, processed_task_count,
  failed_task_count, started_at
) VALUES ($1,$2,$3,$4,'manual','readback','executing',2,0,0,0,now()-interval '2 minutes')`,
		runID, request.ID, request.DeviceID, request.DeviceSN,
	)
	require.NoError(t, err)

	taskRow := task.NewTask(&task.CreateTaskRequest{
		DeviceSN: request.DeviceSN, Method: "GetParameterValues",
		Params: []byte(`{"names":[]}`), CommandKey: "partial-dispatch-must-not-succeed",
		Source: task.TaskSourceParamSync, SourceID: runID.String(), CreatorID: request.ID.String(),
	})
	require.NoError(t, task.NewPgTaskRepository(pool).Create(ctx, taskRow))
	_, err = pool.Exec(ctx, `
UPDATE device_tasks SET status='completed', completed_at=now(), result='{}'::jsonb
WHERE id=$1`, taskRow.ID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
INSERT INTO parameter_sync_task_results (
  run_id, task_id, event_id, success, result_ref, status, processed_at, created_at
) VALUES ($1,$2,$3,true,$4,'processed',now(),now())`,
		runID, taskRow.ID, uuid.NewString(), "device_tasks:"+taskRow.ID,
	)
	require.NoError(t, err)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	result, err := convergeRunTx(ctx, tx, runID, time.Now().UTC())
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))
	require.True(t, result.Finalized)
	require.True(t, result.Failed)
	require.Equal(t, convergenceBlockPlanNotDispatched, result.BlockedReason)

	var status RunStatus
	var expected, terminal, processed int
	require.NoError(t, pool.QueryRow(ctx, `
SELECT status, expected_task_count, terminal_task_count, processed_task_count
FROM parameter_sync_runs WHERE id=$1`, runID).Scan(
		&status, &expected, &terminal, &processed,
	))
	require.Equal(t, RunStatusFailed, status)
	require.Equal(t, []int{2, 1, 1}, []int{expected, terminal, processed})
}

func TestConvergeRunTxFailsExecutingZeroTaskPlan(t *testing.T) {
	ctx := context.Background()
	pool := newParamSyncTestPool(t)
	request := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	runID := uuid.New()
	_, err := pool.Exec(ctx, `
INSERT INTO parameter_sync_runs (
  id, request_id, device_id, device_sn, trigger_reason, sync_scope,
  status, expected_task_count, terminal_task_count, processed_task_count,
  failed_task_count, started_at
) VALUES ($1,$2,$3,$4,'manual','readback','executing',0,0,0,0,now()-interval '2 minutes')`,
		runID, request.ID, request.DeviceID, request.DeviceSN,
	)
	require.NoError(t, err)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	result, err := convergeRunTx(ctx, tx, runID, time.Now().UTC())
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))
	require.True(t, result.Finalized)
	require.True(t, result.Failed)
	require.Equal(t, convergenceBlockPlanNotDispatched, result.BlockedReason)

	var status RunStatus
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT status FROM parameter_sync_runs WHERE id=$1`, runID,
	).Scan(&status))
	require.Equal(t, RunStatusFailed, status)
}

func TestConvergeRunTxFinalizesFailedAuthoritativeResult(t *testing.T) {
	ctx := context.Background()
	pool := newParamSyncTestPool(t)
	request := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	runID := uuid.New()
	_, err := pool.Exec(ctx, `
INSERT INTO parameter_sync_runs (
  id, request_id, device_id, device_sn, trigger_reason, sync_scope,
  status, expected_task_count, terminal_task_count, processed_task_count,
  failed_task_count, error_message
) VALUES ($1,$2,$3,$4,'manual','readback','executing',0,0,0,0,'device task failed')`,
		runID, request.ID, request.DeviceID, request.DeviceSN,
	)
	require.NoError(t, err)

	taskRow := task.NewTask(&task.CreateTaskRequest{
		DeviceSN: request.DeviceSN, Method: "GetParameterValues",
		Params: []byte(`{"names":[]}`), CommandKey: "converge-failed-counts",
		Source: task.TaskSourceParamSync, SourceID: runID.String(), CreatorID: request.ID.String(),
	})
	require.NoError(t, task.NewPgTaskRepository(pool).Create(ctx, taskRow))
	_, err = pool.Exec(ctx, `
UPDATE device_tasks
SET status='failed', completed_at=now(), error_message='device task failed'
WHERE id=$1`, taskRow.ID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
INSERT INTO parameter_sync_task_results (
  run_id, task_id, event_id, success, result_ref, status, error_message,
  processed_at, created_at
) VALUES ($1,$2,$3,false,$4,'failed','device task failed',now(),now())`,
		runID, taskRow.ID, uuid.NewString(), "device_tasks:"+taskRow.ID,
	)
	require.NoError(t, err)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	result, err := convergeRunTx(ctx, tx, runID, time.Now().UTC())
	require.NoError(t, err)
	require.NoError(t, tx.Commit(ctx))
	require.True(t, result.Finalized)
	require.True(t, result.Failed)
	require.True(t, result.Drift)
	require.Equal(t, RunStatusFailed, result.Status)

	var runStatus RunStatus
	var requestStatus RequestStatus
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT status FROM parameter_sync_runs WHERE id=$1`, runID,
	).Scan(&runStatus))
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT status FROM parameter_sync_requests WHERE id=$1`, request.ID,
	).Scan(&requestStatus))
	require.Equal(t, RunStatusFailed, runStatus)
	require.Equal(t, RequestStatusFailed, requestStatus)
}

func TestReconcileRunCountsFinalizesReadyActiveRun(t *testing.T) {
	ctx := context.Background()
	pool := newParamSyncTestPool(t)
	request := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	runID := uuid.New()
	_, err := pool.Exec(ctx, `
INSERT INTO parameter_sync_runs (
  id, request_id, device_id, device_sn, trigger_reason, sync_scope,
  status, expected_task_count, terminal_task_count, processed_task_count,
  failed_task_count, started_at
) VALUES ($1,$2,$3,$4,'manual','readback','executing',0,0,0,0,now()-interval '2 minutes')`,
		runID, request.ID, request.DeviceID, request.DeviceSN,
	)
	require.NoError(t, err)

	taskRow := task.NewTask(&task.CreateTaskRequest{
		DeviceSN: request.DeviceSN, Method: "GetParameterValues",
		Params: []byte(`{"names":[]}`), CommandKey: "maintenance-converges-ready-run",
		Source: task.TaskSourceParamSync, SourceID: runID.String(), CreatorID: request.ID.String(),
	})
	require.NoError(t, task.NewPgTaskRepository(pool).Create(ctx, taskRow))
	_, err = pool.Exec(ctx, `
UPDATE device_tasks SET status='completed', completed_at=now(), result='{}'::jsonb
WHERE id=$1`, taskRow.ID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
INSERT INTO parameter_sync_task_results (
  run_id, task_id, event_id, success, result_ref, status, processed_at, created_at
) VALUES ($1,$2,$3,true,$4,'processed',now(),now())`,
		runID, taskRow.ID, uuid.NewString(), "device_tasks:"+taskRow.ID,
	)
	require.NoError(t, err)

	metrics := NewMetrics(nil)
	reconciler := NewReconciler(pool, nil, metrics)
	require.NoError(t, reconciler.CollectMetrics(ctx))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.RunsReadyButNotFinalized))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.RunCounterDrift))
	require.Less(t, testutil.ToFloat64(metrics.RunCounterDriftOldestIdle), float64(10))
	require.GreaterOrEqual(t, testutil.ToFloat64(metrics.ActiveRunOldestAge), float64(100))

	finalized, err := reconciler.ReconcileRunCounts(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, finalized)
	require.Equal(t, float64(1), testutil.ToFloat64(
		metrics.ReconcileFinalized.WithLabelValues("succeeded"),
	))
	require.Equal(t, float64(1), testutil.ToFloat64(metrics.RunCounterDrift))

	var status RunStatus
	require.NoError(t, pool.QueryRow(ctx,
		`SELECT status FROM parameter_sync_runs WHERE id=$1`, runID,
	).Scan(&status))
	require.Equal(t, RunStatusSucceeded, status)
}

func TestCollectMetricsReportsPerRunBlockedIdleAge(t *testing.T) {
	ctx := context.Background()
	pool := newParamSyncTestPool(t)
	request := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	runID := uuid.New()
	_, err := pool.Exec(ctx, `
INSERT INTO parameter_sync_runs (
  id, request_id, device_id, device_sn, trigger_reason, sync_scope,
  status, expected_task_count, terminal_task_count, processed_task_count,
  failed_task_count, started_at
) VALUES ($1,$2,$3,$4,'manual','readback','executing',1,0,0,0,now()-interval '10 minutes')`,
		runID, request.ID, request.DeviceID, request.DeviceSN,
	)
	require.NoError(t, err)

	taskRow := task.NewTask(&task.CreateTaskRequest{
		DeviceSN: request.DeviceSN, Method: "GetParameterValues",
		Params: []byte(`{"names":[]}`), CommandKey: "aged-terminal-result-missing",
		Source: task.TaskSourceParamSync, SourceID: runID.String(), CreatorID: request.ID.String(),
	})
	require.NoError(t, task.NewPgTaskRepository(pool).Create(ctx, taskRow))
	_, err = pool.Exec(ctx, `
UPDATE device_tasks
SET status='completed', created_at=now()-interval '8 minutes',
    sent_at=now()-interval '8 minutes', completed_at=now()-interval '8 minutes', result='{}'::jsonb
WHERE id=$1`, taskRow.ID)
	require.NoError(t, err)

	metrics := NewMetrics(nil)
	reconciler := NewReconciler(pool, nil, metrics)
	require.NoError(t, reconciler.CollectMetrics(ctx))
	require.Equal(t, float64(1), testutil.ToFloat64(
		metrics.RunsBlocked.WithLabelValues(string(convergenceBlockTerminalResultMissing)),
	))
	require.GreaterOrEqual(t, testutil.ToFloat64(
		metrics.RunsBlockedOldestIdle.WithLabelValues(string(convergenceBlockTerminalResultMissing)),
	), float64(470))
}

func TestDuplicateResultDoesNotWaitForRunWriteLock(t *testing.T) {
	ctx := context.Background()
	pool := newParamSyncTestPool(t)
	request := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	runID := uuid.New()
	taskID := uuid.NewString()
	_, err := pool.Exec(ctx, `
INSERT INTO parameter_sync_runs (
  id, request_id, device_id, device_sn, trigger_reason, sync_scope,
  status, expected_task_count, terminal_task_count, processed_task_count,
  failed_task_count
) VALUES ($1,$2,$3,$4,'manual','full','executing',1,1,1,0)`,
		runID, request.ID, request.DeviceID, request.DeviceSN,
	)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
INSERT INTO parameter_sync_task_results (
  run_id, task_id, event_id, success, result_ref, status, created_at
) VALUES ($1,$2,$3,true,$4,'processed',now())`,
		runID, taskID, uuid.NewString(), "device_tasks:"+taskID,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM parameter_sync_runs WHERE id=$1`, runID)
	})

	locker, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = locker.Rollback(context.Background()) }()
	_, err = locker.Exec(ctx, `SELECT 1 FROM parameter_sync_runs WHERE id=$1 FOR UPDATE`, runID)
	require.NoError(t, err)
	processCtx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
	defer cancel()
	outcome, err := NewPGResultProcessor(pool).Process(processCtx, event.ParamSyncTaskResultPayload{
		RunID: runID, RequestID: request.ID, TaskID: taskID, DeviceID: request.DeviceID,
		DeviceSN: request.DeviceSN, EventID: uuid.NewString(), Success: true,
	})

	require.NoError(t, err)
	assert.True(t, outcome.Duplicate)
}

func TestResultProcessingDoesNotBackfillLegacyTaskOutbox(t *testing.T) {
	ctx := context.Background()
	pool := newParamSyncTestPool(t)
	request := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	runID := uuid.New()
	_, err := pool.Exec(ctx, `
INSERT INTO parameter_sync_runs (
  id, request_id, device_id, device_sn, trigger_reason, sync_scope,
  status, expected_task_count, terminal_task_count, processed_task_count,
  failed_task_count
) VALUES ($1,$2,$3,$4,'manual','readback','waiting_device',2,0,0,0)`,
		runID, request.ID, request.DeviceID, request.DeviceSN,
	)
	require.NoError(t, err)

	first := task.NewTask(&task.CreateTaskRequest{
		DeviceSN: request.DeviceSN, Method: "GetParameterValues",
		Params: []byte(`{"names":[]}`), CommandKey: "result-hot-path-first",
		Source: task.TaskSourceParamSync, SourceID: runID.String(), CreatorID: request.ID.String(),
		CommandIndex: 0,
	})
	second := task.NewTask(&task.CreateTaskRequest{
		DeviceSN: request.DeviceSN, Method: "GetParameterValues",
		Params: []byte(`{"names":[]}`), CommandKey: "result-hot-path-second",
		Source: task.TaskSourceParamSync, SourceID: runID.String(), CreatorID: request.ID.String(),
		CommandIndex: 1,
	})
	repo := task.NewPgTaskRepository(pool)
	require.NoError(t, repo.Create(ctx, first))
	require.NoError(t, repo.Create(ctx, second))
	_, err = pool.Exec(ctx, `UPDATE device_tasks SET status='pending', command_index=1 WHERE id=$1`, second.ID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
UPDATE device_tasks
SET status='completed', completed_at=now(), result='{}'::jsonb
WHERE id=$1`, first.ID)
	require.NoError(t, err)
	var candidates int
	require.NoError(t, pool.QueryRow(ctx, `
SELECT count(*) FROM device_tasks t
JOIN device_tasks completed ON completed.id=$2
WHERE t.source='param_sync' AND t.source_id=$1 AND t.status='pending'
  AND t.command_index > completed.command_index`, runID, first.ID).Scan(&candidates))
	require.Equal(t, 1, candidates)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM parameter_sync_runs WHERE id=$1`, runID)
	})

	locker, err := pool.Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = locker.Rollback(context.Background()) }()
	_, err = locker.Exec(ctx, `LOCK TABLE parameter_sync_outbox IN ACCESS EXCLUSIVE MODE`)
	require.NoError(t, err)
	processCtx, cancel := context.WithTimeout(ctx, 150*time.Millisecond)
	defer cancel()

	_, err = NewPGResultProcessor(pool).Process(processCtx, event.ParamSyncTaskResultPayload{
		RunID: runID, RequestID: request.ID, TaskID: first.ID,
		DeviceID: request.DeviceID, DeviceSN: request.DeviceSN,
		EventID: uuid.NewString(), Success: true,
	})
	require.NoError(t, err)
	require.NoError(t, locker.Rollback(ctx))

	var count int
	require.NoError(t, pool.QueryRow(ctx, `
SELECT count(*) FROM parameter_sync_outbox
WHERE event_type='param_sync.task.enqueue' AND aggregate_id=$1`, second.ID).Scan(&count))
	assert.Zero(t, count, "result processing must not run legacy outbox discovery on every task")
}

func TestMappedMLNRFStateCannotBeOverwrittenByLaterRawShadow(t *testing.T) {
	ctx := context.Background()
	pool := newParamSyncTestPool(t)
	deviceID := uuid.New()
	insertReleaseCandidateForTest(t, pool, deviceID, true)
	_, err := pool.Exec(ctx,
		`UPDATE devices SET product_class=$2 WHERE id=$1`,
		deviceID,
		mlnDCProductClass,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM device_parameters WHERE device_id=$1`, deviceID)
	})

	const standardPath = "Device.Services.FAPService.1.FAPControl.LTE.RFTxStatus"
	coverage := []CoverageScope{{
		Mappings: []FrozenMapping{{
			StandardPath: mlnDCRFStatusStandardPath,
			PrivatePath:  mlnDCRFStatusPrivatePath,
			Access:       "readWrite",
			DataType:     "boolean",
			IsStorable:   true,
		}},
	}}
	mapped := projectTaskValues([]tr069.ParameterValueStruct{{
		Name:  "Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.AdminCellState",
		Value: "1",
		Type:  "xsd:unsignedInt",
	}}, coverage)
	rawShadow := projectTaskValues([]tr069.ParameterValueStruct{{
		Name:  standardPath,
		Value: "0",
		Type:  "xsd:string",
	}}, coverage)

	tx, err := pool.Begin(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	rawShadow, err = filterMLNDCRawRFShadowValues(ctx, tx, deviceID, rawShadow, coverage)
	require.NoError(t, err)
	require.Empty(t, rawShadow)
	now := time.Now().UTC()
	require.NoError(t, upsertOfficialValues(ctx, tx, deviceID, mapped, now))
	require.NoError(t, upsertOfficialValues(ctx, tx, deviceID, rawShadow, now))
	require.NoError(t, tx.Commit(ctx))

	var value string
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT parameter_value
		FROM device_parameters
		WHERE device_id=$1 AND parameter_path=$2
	`, deviceID, standardPath).Scan(&value))
	assert.Equal(t, "1", value)
}

func TestRecoveredMLNDCRF9005ReplacesStaleOffWithUnknown(t *testing.T) {
	tests := []struct {
		name         string
		productClass string
		scope        SyncScope
		wantValue    string
	}{
		{name: "MLN_DC_partial", productClass: "FAP/MLN/DC", scope: SyncScopePartial, wantValue: "unknown"},
		{name: "MLN_DC_full", productClass: "FAP/MLN/DC", scope: SyncScopeFull, wantValue: "unknown"},
		{name: "MLN_SC_partial", productClass: "FAP/MLN/SC", scope: SyncScopePartial, wantValue: "0"},
		{name: "MLN_SC_full", productClass: "FAP/MLN/SC", scope: SyncScopeFull, wantValue: "0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRecoveredMLNDCRF9005(t, tt.productClass, tt.scope, tt.wantValue)
		})
	}
}

func testRecoveredMLNDCRF9005(
	t *testing.T,
	productClass string,
	scope SyncScope,
	wantValue string,
) {
	t.Helper()
	ctx := context.Background()
	pool := newParamSyncTestPool(t)
	request := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	insertReleaseCandidateForTest(t, pool, request.DeviceID, true)
	_, err := pool.Exec(ctx,
		`UPDATE devices SET product_class=$2 WHERE id=$1`,
		request.DeviceID,
		productClass,
	)
	require.NoError(t, err)

	const (
		privatePath  = "Device.Services.FAPService.2.CellConfig.LTE.RAN.RF.AdminCellState"
		standardPath = "Device.Services.FAPService.2.FAPControl.LTE.RFTxStatus"
	)
	_, err = pool.Exec(ctx, `
		INSERT INTO device_parameters (
			device_id, parameter_path, parameter_value, parameter_type,
			writable, last_updated_at, fap_instance, param_group
		) VALUES ($1, $2, '0', 'unsignedInt', true, now(), 2, 'fap_control')
	`, request.DeviceID, standardPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM device_parameters WHERE device_id=$1`, request.DeviceID)
	})

	coverage, err := json.Marshal([]CoverageScope{{
		Path:     mlnDCRFStatusStandardPath,
		Complete: true,
		Mappings: []FrozenMapping{{
			StandardPath: mlnDCRFStatusStandardPath,
			PrivatePath:  mlnDCRFStatusPrivatePath,
			Access:       "readWrite",
			DataType:     "boolean",
			IsStorable:   true,
		}},
	}})
	require.NoError(t, err)
	runID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO parameter_sync_runs (
			id, request_id, device_id, device_sn, trigger_reason, sync_scope,
			status, coverage, expected_task_count, terminal_task_count,
			processed_task_count, failed_task_count
		) VALUES ($1, $2, $3, $4, 'manual', $5, 'waiting_device',
		          $6, 1, 1, 0, 0)
	`, runID, request.ID, request.DeviceID, request.DeviceSN, scope, coverage)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		UPDATE parameter_sync_requests
		SET run_id=$2, active_run_id=$2
		WHERE id=$1
	`, request.ID, runID)
	require.NoError(t, err)

	completed := task.NewTask(&task.CreateTaskRequest{
		DeviceSN:   request.DeviceSN,
		Method:     "GetParameterValues",
		Params:     []byte(`{"names":["` + privatePath + `"]}`),
		CommandKey: "param-sync-" + runID.String() + "-1",
		Source:     task.TaskSourceParamSync,
		SourceID:   runID.String(),
		CreatorID:  request.ID.String(),
	})
	require.NoError(t, task.NewPgTaskRepository(pool).Create(ctx, completed))
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM device_tasks WHERE id=$1`, completed.ID)
	})
	recoveredResult, err := json.Marshal(map[string]any{
		"recovered":  true,
		"bad_path":   privatePath,
		"fault_code": 9005,
	})
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		UPDATE device_tasks
		SET status='completed', completed_at=now(), result=$2
		WHERE id=$1
	`, completed.ID, recoveredResult)
	require.NoError(t, err)

	recovered, err := NewReconciler(pool, nil, nil).
		WithResultProcessor(NewPGResultProcessor(pool)).
		RecoverMissingResults(ctx, 20, 200, 200)

	require.NoError(t, err)
	assert.Equal(t, 1, recovered)
	var value string
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT parameter_value
		FROM device_parameters
		WHERE device_id=$1 AND parameter_path=$2
	`, request.DeviceID, standardPath).Scan(&value))
	assert.Equal(t, wantValue, value)
}
