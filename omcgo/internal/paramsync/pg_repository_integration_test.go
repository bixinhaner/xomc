package paramsync

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/task"
)

func newParamSyncTestPool(t *testing.T) *pgxpool.Pool {
	return newParamSyncTestPoolWithApp(t, "")
}

func newParamSyncTestPoolWithApp(t *testing.T, applicationName string) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PG_URL")
	if dsn == "" {
		dsn = "postgres://omcgo:omcgo123@localhost:5432/omcgo?sslmode=disable"
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Skipf("invalid PostgreSQL test configuration: %v", err)
	}
	if applicationName != "" {
		config.ConnConfig.RuntimeParams["application_name"] = applicationName
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		t.Skipf("no PostgreSQL available: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("no PostgreSQL available: %v", err)
	}
	t.Cleanup(pool.Close)
	var exists bool
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT to_regclass('public.parameter_sync_requests') IS NOT NULL`).Scan(&exists))
	if !exists {
		t.Skip("parameter-sync migrations are not applied")
	}
	return pool
}

func insertParamSyncRequestForTest(t *testing.T, pool *pgxpool.Pool, status RequestStatus) *SyncRequest {
	t.Helper()
	now := time.Now().UTC()
	req := &SyncRequest{
		ID: uuid.New(), DeviceID: uuid.New(), DeviceSN: "TEST-PARAM-SYNC-" + uuid.NewString(),
		CallerType: "test", TriggerReason: TriggerManual, SyncScope: SyncScopeFull,
		Status: status, Priority: 10, NextAttemptAt: now, CreatedAt: now, UpdatedAt: now,
	}
	if status.Terminal() {
		req.CompletedAt = &now
	}
	query, args, err := buildCreateRequest(req)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), query, args...)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM parameter_sync_requests WHERE id=$1`, req.ID)
	})
	return req
}

func TestPGRepositoryCompleteRequestRejectsTerminalState(t *testing.T) {
	pool := newParamSyncTestPool(t)
	req := insertParamSyncRequestForTest(t, pool, RequestStatusCancelled)
	repo := NewPGRepository(pool)

	err := repo.CompleteRequest(context.Background(), req.ID, RequestStatusSucceeded, ResultCodeOK, nil, "")

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrRequestStateConflict))
	stored, getErr := repo.GetRequest(context.Background(), req.ID)
	require.NoError(t, getErr)
	assert.Equal(t, RequestStatusCancelled, stored.Status)
}

func TestPGRepositoryCreateRunDoesNotResurrectCancelledRequest(t *testing.T) {
	pool := newParamSyncTestPool(t)
	req := insertParamSyncRequestForTest(t, pool, RequestStatusCancelled)
	repo := NewPGRepository(pool)

	_, err := repo.CreateOrDeduplicateRun(context.Background(), req)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrRequestStateConflict))
	var runCount int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT count(*) FROM parameter_sync_runs WHERE request_id=$1`, req.ID).Scan(&runCount))
	assert.Zero(t, runCount)
}

func TestGPVFaultRecovererRejectsCancellingRun(t *testing.T) {
	pool := newParamSyncTestPool(t)
	req := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	runID := uuid.New()
	_, err := pool.Exec(context.Background(), `INSERT INTO parameter_sync_runs
(id, request_id, device_id, device_sn, trigger_reason, sync_scope, status, error_message)
VALUES ($1, $2, $3, $4, 'manual', 'full', 'cancelling', 'another task failed')`,
		runID, req.ID, req.DeviceID, req.DeviceSN)
	require.NoError(t, err)

	original := task.NewTask(&task.CreateTaskRequest{
		DeviceSN: req.DeviceSN, Method: "GetParameterValues",
		Params:     []byte(`{"names":["Device.Bad","Device.Good"]}`),
		CommandKey: "param-sync-" + runID.String() + "-0",
		Source:     task.TaskSourceParamSync, SourceID: runID.String(), CreatorID: req.ID.String(),
	})
	_, err = NewGPVFaultRecoverer(pool).Recover(context.Background(), original, []string{"Device.Good"})

	require.ErrorContains(t, err, "cannot accept 9005 recovery")
	var taskCount, expected int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT count(*) FROM device_tasks WHERE source='param_sync' AND source_id=$1`, runID).Scan(&taskCount))
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT expected_task_count FROM parameter_sync_runs WHERE id=$1`, runID).Scan(&expected))
	assert.Zero(t, taskCount)
	assert.Zero(t, expected)
}

type recordingRecoveryTaskReleaser struct {
	planned *task.Task
	err     error
}

func (r *recordingRecoveryTaskReleaser) ReleasePlannedTaskWithoutWake(_ context.Context, planned *task.Task) (bool, error) {
	r.planned = planned
	if r.err != nil {
		return false, r.err
	}
	return true, nil
}

func TestGPVFaultRecovererReleasesReplacementBeforeReturning(t *testing.T) {
	pool := newParamSyncTestPool(t)
	req := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	runID := uuid.New()
	_, err := pool.Exec(context.Background(), `INSERT INTO parameter_sync_runs
(id, request_id, device_id, device_sn, trigger_reason, sync_scope, status,
 expected_task_count, terminal_task_count, processed_task_count, failed_task_count)
VALUES ($1, $2, $3, $4, 'manual', 'full', 'executing', 1, 0, 0, 0)`,
		runID, req.ID, req.DeviceID, req.DeviceSN)
	require.NoError(t, err)

	original := task.NewTask(&task.CreateTaskRequest{
		DeviceSN: req.DeviceSN, Method: "GetParameterValues",
		Params:     []byte(`{"names":["Device.Bad","Device.Good"]}`),
		CommandKey: "param-sync-" + runID.String() + "-0",
		Source:     task.TaskSourceParamSync, SourceID: runID.String(), CreatorID: req.ID.String(),
	})
	releaser := &recordingRecoveryTaskReleaser{}

	replacement, err := NewGPVFaultRecoverer(pool, releaser).Recover(
		context.Background(), original, []string{"Device.Good"},
	)

	require.NoError(t, err)
	require.NotNil(t, replacement)
	require.Same(t, replacement, releaser.planned)
	assert.Equal(t, []byte(`{"names":["Device.Good"]}`), []byte(replacement.Params))
	assert.Equal(t, original.CommandKey+"-r", replacement.CommandKey)

	var taskCount, expected, outboxCount int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT count(*) FROM device_tasks WHERE id=$1`, replacement.ID).Scan(&taskCount))
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT expected_task_count FROM parameter_sync_runs WHERE id=$1`, runID).Scan(&expected))
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT count(*) FROM parameter_sync_outbox WHERE aggregate_id=$1 AND status='pending'`, replacement.ID).Scan(&outboxCount))
	assert.Equal(t, 1, taskCount)
	assert.Equal(t, 2, expected)
	assert.Equal(t, 1, outboxCount, "outbox remains the crash-safe delivery fallback")
}

func TestGPVFaultRecovererKeepsDurableFallbackWhenImmediateReleaseFails(t *testing.T) {
	pool := newParamSyncTestPool(t)
	req := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	runID := uuid.New()
	_, err := pool.Exec(context.Background(), `INSERT INTO parameter_sync_runs
(id, request_id, device_id, device_sn, trigger_reason, sync_scope, status,
 expected_task_count, terminal_task_count, processed_task_count, failed_task_count)
VALUES ($1, $2, $3, $4, 'manual', 'full', 'executing', 1, 0, 0, 0)`,
		runID, req.ID, req.DeviceID, req.DeviceSN)
	require.NoError(t, err)

	original := task.NewTask(&task.CreateTaskRequest{
		DeviceSN: req.DeviceSN, Method: "GetParameterValues",
		Params:     []byte(`{"names":["Device.Bad","Device.Good"]}`),
		CommandKey: "param-sync-" + runID.String() + "-0",
		Source:     task.TaskSourceParamSync, SourceID: runID.String(), CreatorID: req.ID.String(),
	})
	releaser := &recordingRecoveryTaskReleaser{err: errors.New("redis unavailable")}

	replacement, err := NewGPVFaultRecoverer(pool, releaser).Recover(
		context.Background(), original, []string{"Device.Good"},
	)

	require.ErrorContains(t, err, "release parameter sync recovery task")
	require.NotNil(t, replacement, "a post-commit release failure must be distinguishable from a transaction failure")
	var taskCount, outboxCount int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT count(*) FROM device_tasks WHERE id=$1`, replacement.ID).Scan(&taskCount))
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT count(*) FROM parameter_sync_outbox WHERE aggregate_id=$1 AND status='pending'`, replacement.ID).Scan(&outboxCount))
	assert.Equal(t, 1, taskCount)
	assert.Equal(t, 1, outboxCount)
}

func TestReconcileCancellingRunsCancelsLateRecoveryAndFinalizes(t *testing.T) {
	pool := newParamSyncTestPool(t)
	req := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	runID := uuid.New()
	_, err := pool.Exec(context.Background(), `INSERT INTO parameter_sync_runs
(id, request_id, device_id, device_sn, trigger_reason, sync_scope, status,
 expected_task_count, terminal_task_count, processed_task_count, failed_task_count, error_message)
VALUES ($1, $2, $3, $4, 'manual', 'full', 'cancelling', 1, 0, 0, 0, 'another task failed')`,
		runID, req.ID, req.DeviceID, req.DeviceSN)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `UPDATE parameter_sync_requests SET run_id=$2, active_run_id=$2 WHERE id=$1`, req.ID, runID)
	require.NoError(t, err)

	late := task.NewTask(&task.CreateTaskRequest{
		DeviceSN: req.DeviceSN, Method: "GetParameterValues",
		Params:     []byte(`{"names":["Device.Good"]}`),
		CommandKey: "param-sync-" + runID.String() + "-0-r",
		Source:     task.TaskSourceParamSync, SourceID: runID.String(), CreatorID: req.ID.String(),
	})
	require.NoError(t, task.NewPgTaskRepository(pool).Create(context.Background(), late))
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM device_tasks WHERE id=$1`, late.ID)
	})

	finalized, err := NewReconciler(pool, nil, nil).ReconcileCancellingRuns(context.Background(), 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, finalized, int64(1))

	var taskStatus task.TaskStatus
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT status FROM device_tasks WHERE id=$1`, late.ID).Scan(&taskStatus))
	assert.Equal(t, task.TaskStatusCancelled, taskStatus)
	var runStatus RunStatus
	var expected, terminal, processed int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT status, expected_task_count, terminal_task_count, processed_task_count
FROM parameter_sync_runs WHERE id=$1`, runID).Scan(&runStatus, &expected, &terminal, &processed))
	assert.Equal(t, RunStatusFailed, runStatus)
	assert.Equal(t, 1, expected)
	assert.Equal(t, 1, terminal)
	assert.Equal(t, 1, processed)
	stored, err := NewPGRepository(pool).GetRequest(context.Background(), req.ID)
	require.NoError(t, err)
	assert.Equal(t, RequestStatusFailed, stored.Status)
	assert.Nil(t, stored.ActiveRunID)
}

func TestRecoverMissingResultsCallsProcessorForTerminalTasks(t *testing.T) {
	pool := newParamSyncTestPool(t)
	req := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	runID := uuid.New()
	_, err := pool.Exec(context.Background(), `INSERT INTO parameter_sync_runs
(id, request_id, device_id, device_sn, trigger_reason, sync_scope, status,
 expected_task_count, terminal_task_count, processed_task_count, failed_task_count)
VALUES ($1, $2, $3, $4, 'manual', 'full', 'waiting_device', 1, 1, 0, 0)`,
		runID, req.ID, req.DeviceID, req.DeviceSN)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `UPDATE parameter_sync_requests SET run_id=$2, active_run_id=$2 WHERE id=$1`, req.ID, runID)
	require.NoError(t, err)

	completed := task.NewTask(&task.CreateTaskRequest{
		DeviceSN: req.DeviceSN, Method: "GetParameterValues",
		Params:     []byte(`{"names":["Device.Good"]}`),
		CommandKey: "param-sync-" + runID.String() + "-0",
		Source:     task.TaskSourceParamSync, SourceID: runID.String(), CreatorID: req.ID.String(),
	})
	require.NoError(t, task.NewPgTaskRepository(pool).Create(context.Background(), completed))
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM device_tasks WHERE id=$1`, completed.ID)
	})
	_, err = pool.Exec(context.Background(), `UPDATE device_tasks SET status='completed', completed_at=now() WHERE id=$1`, completed.ID)
	require.NoError(t, err)

	processor := &recordingResultProcessor{}
	recovered, err := NewReconciler(pool, nil, nil).WithResultProcessor(processor).RecoverMissingResults(context.Background(), 20, 200, 200)

	require.NoError(t, err)
	assert.Equal(t, 1, recovered)
	require.Len(t, processor.payloads, 1)
	payload := processor.payloads[0]
	assert.Equal(t, runID, payload.RunID)
	assert.Equal(t, req.ID, payload.RequestID)
	assert.Equal(t, completed.ID, payload.TaskID)
	assert.Equal(t, req.DeviceSN, payload.DeviceSN)
	assert.True(t, payload.Success)
	assert.Equal(t, "device_tasks:"+completed.ID, payload.ResultRef)
	assert.Contains(t, payload.EventID, "reconcile:"+runID.String()+":"+completed.ID)
}

func TestRecoverMissingResultsWithPGProcessorFinalizesSucceededRun(t *testing.T) {
	pool := newParamSyncTestPool(t)
	req := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	runID := uuid.New()
	_, err := pool.Exec(context.Background(), `INSERT INTO parameter_sync_runs
(id, request_id, device_id, device_sn, trigger_reason, sync_scope, status,
 expected_task_count, terminal_task_count, processed_task_count, failed_task_count)
VALUES ($1, $2, $3, $4, 'manual', 'partial', 'waiting_device', 1, 1, 0, 0)`,
		runID, req.ID, req.DeviceID, req.DeviceSN)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `UPDATE parameter_sync_requests SET run_id=$2, active_run_id=$2 WHERE id=$1`, req.ID, runID)
	require.NoError(t, err)

	completed := task.NewTask(&task.CreateTaskRequest{
		DeviceSN: req.DeviceSN, Method: "GetParameterValues",
		Params:     []byte(`{"names":["Device.Good"]}`),
		CommandKey: "param-sync-" + runID.String() + "-0",
		Source:     task.TaskSourceParamSync, SourceID: runID.String(), CreatorID: req.ID.String(),
	})
	require.NoError(t, task.NewPgTaskRepository(pool).Create(context.Background(), completed))
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM device_tasks WHERE id=$1`, completed.ID)
	})
	_, err = pool.Exec(context.Background(), `UPDATE device_tasks SET status='completed', completed_at=now(),
result='{"standard_parameter_values":[]}'::jsonb WHERE id=$1`, completed.ID)
	require.NoError(t, err)

	recovered, err := NewReconciler(pool, nil, nil).
		WithResultProcessor(NewPGResultProcessor(pool)).
		RecoverMissingResults(context.Background(), 20, 200, 200)

	require.NoError(t, err)
	assert.Equal(t, 1, recovered)
	var resultStatus string
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT status FROM parameter_sync_task_results WHERE run_id=$1 AND task_id=$2`, runID, completed.ID).Scan(&resultStatus))
	assert.Equal(t, "processed", resultStatus)
	var runStatus RunStatus
	var expected, terminal, processed int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT status, expected_task_count, terminal_task_count, processed_task_count
FROM parameter_sync_runs WHERE id=$1`, runID).Scan(&runStatus, &expected, &terminal, &processed))
	assert.Equal(t, RunStatusSucceeded, runStatus)
	assert.Equal(t, 1, expected)
	assert.Equal(t, 1, terminal)
	assert.Equal(t, 1, processed)
	stored, err := NewPGRepository(pool).GetRequest(context.Background(), req.ID)
	require.NoError(t, err)
	assert.Equal(t, RequestStatusSucceeded, stored.Status)
	assert.Nil(t, stored.ActiveRunID)
}

func TestRecoverMissingResultsWithPGProcessorFinalizesFailedRun(t *testing.T) {
	pool := newParamSyncTestPool(t)
	req := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	runID := uuid.New()
	_, err := pool.Exec(context.Background(), `INSERT INTO parameter_sync_runs
(id, request_id, device_id, device_sn, trigger_reason, sync_scope, status,
 expected_task_count, terminal_task_count, processed_task_count, failed_task_count)
VALUES ($1, $2, $3, $4, 'manual', 'partial', 'waiting_device', 1, 1, 0, 0)`,
		runID, req.ID, req.DeviceID, req.DeviceSN)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `UPDATE parameter_sync_requests SET run_id=$2, active_run_id=$2 WHERE id=$1`, req.ID, runID)
	require.NoError(t, err)

	failed := task.NewTask(&task.CreateTaskRequest{
		DeviceSN: req.DeviceSN, Method: "GetParameterValues",
		Params:     []byte(`{"names":["Device.Bad"]}`),
		CommandKey: "param-sync-" + runID.String() + "-0",
		Source:     task.TaskSourceParamSync, SourceID: runID.String(), CreatorID: req.ID.String(),
	})
	require.NoError(t, task.NewPgTaskRepository(pool).Create(context.Background(), failed))
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM device_tasks WHERE id=$1`, failed.ID)
	})
	_, err = pool.Exec(context.Background(), `UPDATE device_tasks SET status='failed', completed_at=now(),
error_code=9005, error_message='Invalid parameter name' WHERE id=$1`, failed.ID)
	require.NoError(t, err)

	recovered, err := NewReconciler(pool, nil, nil).
		WithResultProcessor(NewPGResultProcessor(pool)).
		RecoverMissingResults(context.Background(), 20, 200, 200)

	require.NoError(t, err)
	assert.Equal(t, 1, recovered)
	var resultStatus string
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT status FROM parameter_sync_task_results WHERE run_id=$1 AND task_id=$2`, runID, failed.ID).Scan(&resultStatus))
	assert.Equal(t, "failed", resultStatus)
	var runStatus RunStatus
	var processed, failedCount int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT status, processed_task_count, failed_task_count
FROM parameter_sync_runs WHERE id=$1`, runID).Scan(&runStatus, &processed, &failedCount))
	assert.Equal(t, RunStatusFailed, runStatus)
	assert.Equal(t, 1, processed)
	assert.Equal(t, 1, failedCount)
	stored, err := NewPGRepository(pool).GetRequest(context.Background(), req.ID)
	require.NoError(t, err)
	assert.Equal(t, RequestStatusFailed, stored.Status)
	assert.Nil(t, stored.ActiveRunID)
}

func TestCancelAndRunFinalizationUseCompatibleLockOrder(t *testing.T) {
	applicationName := "paramsync-lock-test-" + uuid.NewString()
	pool := newParamSyncTestPoolWithApp(t, applicationName)
	req := insertParamSyncRequestForTest(t, pool, RequestStatusAccepted)
	repo := NewPGRepository(pool)
	start, err := repo.CreateOrDeduplicateRun(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, start.Run)

	finalizeTx, err := pool.Begin(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() { _ = finalizeTx.Rollback(context.Background()) })
	_, err = finalizeTx.Exec(context.Background(), `UPDATE parameter_sync_runs SET status='succeeded', completed_at=now() WHERE id=$1`, start.Run.ID)
	require.NoError(t, err)

	cancelDone := make(chan error, 1)
	go func() {
		cancelDone <- NewOperations(pool, nil).CancelRequest(context.Background(), req.ID)
	}()

	// Wait until cancellation is blocked on this transaction's run lock. With
	// the old request -> run ordering it also holds the request at this point,
	// forming a deterministic cycle when finalization updates that request.
	deadline := time.Now().Add(2 * time.Second)
	for {
		var waiting bool
		require.NoError(t, pool.QueryRow(context.Background(), `
SELECT EXISTS (
  SELECT 1 FROM pg_stat_activity
  WHERE application_name=$1 AND state='active' AND wait_event_type='Lock'
    AND query LIKE '%parameter_sync_runs%'
)`, applicationName).Scan(&waiting))
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("cancellation did not reach the run lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	_, finalizeErr := finalizeTx.Exec(context.Background(), `UPDATE parameter_sync_requests SET status='succeeded', result_code='OK', active_run_id=NULL, completed_at=now(), updated_at=now() WHERE id=$1`, req.ID)
	if finalizeErr == nil {
		finalizeErr = finalizeTx.Commit(context.Background())
	}
	cancelErr := <-cancelDone

	assertNotDeadlock := func(label string, err error) {
		t.Helper()
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			assert.NotEqual(t, "40P01", pgErr.Code, "%s must not deadlock", label)
		}
	}
	assertNotDeadlock("finalization", finalizeErr)
	assertNotDeadlock("cancellation", cancelErr)
	require.NoError(t, finalizeErr)
	require.Error(t, cancelErr, "the terminal request must win over cancellation")
}
