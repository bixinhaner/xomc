package paramsync

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/model"
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

func TestPGRepositoryCreateRequestSilentlyDeduplicatesIdempotencyKey(t *testing.T) {
	pool := newParamSyncTestPool(t)
	repo := NewPGRepository(pool)
	now := time.Now().UTC()
	key := "integration-idempotency:" + uuid.NewString()
	first := &SyncRequest{
		ID: uuid.New(), DeviceID: uuid.New(), DeviceSN: "TEST-IDEMPOTENCY-FIRST",
		CallerType: "integration", TriggerReason: TriggerManual, SyncScope: SyncScopeFull,
		Status: RequestStatusAccepted, Priority: 10, NextAttemptAt: now,
		IdempotencyKey: &key, CreatedAt: now, UpdatedAt: now,
	}
	second := *first
	second.ID = uuid.New()
	second.DeviceID = uuid.New()
	second.DeviceSN = "TEST-IDEMPOTENCY-SECOND"
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM parameter_sync_requests WHERE caller_type=$1 AND idempotency_key=$2`,
			first.CallerType, key,
		)
	})

	require.NoError(t, repo.CreateRequest(context.Background(), first))
	err := repo.CreateRequest(context.Background(), &second)

	require.ErrorIs(t, err, ErrRequestIdempotencyConflict)
	existing, findErr := repo.FindRequestByIdempotency(context.Background(), first.CallerType, key)
	require.NoError(t, findErr)
	assert.Equal(t, first.ID, existing.ID)
}

func TestPGRepositoryAutomaticIdempotencyConflictRollsBackDeviceGate(t *testing.T) {
	pool := newParamSyncTestPool(t)
	repo := NewPGRepository(pool)
	now := time.Now().UTC()
	key := "integration-automatic-idempotency:" + uuid.NewString()
	first := &SyncRequest{
		ID: uuid.New(), DeviceID: uuid.New(), DeviceSN: "TEST-AUTOMATIC-FIRST",
		CallerType: "provision", TriggerReason: TriggerDeviceOnline, SyncScope: SyncScopeFull,
		Status: RequestStatusAccepted, Priority: 10, NextAttemptAt: now,
		IdempotencyKey: &key, CreatedAt: now, UpdatedAt: now,
	}
	second := *first
	second.ID = uuid.New()
	second.DeviceID = uuid.New()
	second.DeviceSN = "TEST-AUTOMATIC-SECOND"
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM parameter_sync_requests WHERE caller_type=$1 AND idempotency_key=$2`,
			first.CallerType, key,
		)
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM parameter_sync_device_state WHERE device_id=ANY($1)`,
			[]uuid.UUID{first.DeviceID, second.DeviceID},
		)
	})

	allowed, err := repo.CreateAutomaticRequest(context.Background(), first, now)
	require.NoError(t, err)
	assert.True(t, allowed)
	_, err = repo.CreateAutomaticRequest(context.Background(), &second, now)
	require.ErrorIs(t, err, ErrRequestIdempotencyConflict)

	var secondGateCount int
	require.NoError(t, pool.QueryRow(context.Background(),
		`SELECT count(*) FROM parameter_sync_device_state WHERE device_id=$1`,
		second.DeviceID,
	).Scan(&secondGateCount))
	assert.Zero(t, secondGateCount)
}

func TestPGRepositoryAutomaticAdmissionReservationIsReleasedAfterTerminalRequest(t *testing.T) {
	pool := newParamSyncTestPool(t)
	repo := NewPGRepository(pool)
	now := time.Now().UTC()
	request := &SyncRequest{
		ID: uuid.New(), DeviceID: uuid.New(), DeviceSN: "TEST-AUTOMATIC-ADMISSION",
		CallerType: "integration", TriggerReason: TriggerDeviceRegistered, SyncScope: SyncScopeFull,
		Status: RequestStatusAccepted, Priority: 10, NextAttemptAt: now,
		CreatedAt: now, UpdatedAt: now,
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `UPDATE parameter_sync_requests SET status='cancelled', completed_at=now() WHERE id=$1 AND status NOT IN ('succeeded','failed','timed_out','cancelled','deduplicated','rejected')`, request.ID)
		_, _ = repo.ReconcileAutomaticAdmission(context.Background(), time.Now().UTC(), 500)
		_, _ = pool.Exec(context.Background(), `DELETE FROM parameter_sync_requests WHERE id=$1`, request.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM parameter_sync_device_state WHERE device_id=$1`, request.DeviceID)
		_, _ = repo.ReconcileAutomaticAdmission(context.Background(), time.Now().UTC(), 500)
	})

	allowed, err := repo.CreateAutomaticRequest(context.Background(), request, now)
	require.NoError(t, err)
	require.True(t, allowed)
	stats, err := repo.GetAutomaticAdmissionStats(context.Background(), now.Add(time.Second))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, stats.ReservedRuns, 1)

	var reservationStatus string
	var reservedTasks int
	require.NoError(t, pool.QueryRow(context.Background(), `
SELECT status, reserved_tasks FROM parameter_sync_admission_reservations
WHERE request_id=$1 AND admission_class=$2`, request.ID, automaticAdmissionClass).Scan(&reservationStatus, &reservedTasks))
	assert.Equal(t, "reserved", reservationStatus)
	assert.Zero(t, reservedTasks)
	_, err = pool.Exec(context.Background(), `
UPDATE parameter_sync_admission_reservations SET lease_until=$2::timestamptz - interval '1 second' WHERE request_id=$1`,
		request.ID, now.Add(time.Second))
	require.NoError(t, err)
	recovered, err := repo.ReconcileAutomaticAdmission(context.Background(), now.Add(time.Second), 500)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, recovered, 1)
	stored, err := repo.GetRequest(context.Background(), request.ID)
	require.NoError(t, err)
	assert.Equal(t, RequestStatusQueued, stored.Status, "accepted requests left by a crash must recover to durable dispatch")
	request.Status = RequestStatusQueued

	adjusted, err := repo.AdjustAutomaticAdmissionTasks(context.Background(), request, 37, now.Add(time.Second))
	require.NoError(t, err)
	require.True(t, adjusted)
	require.NoError(t, pool.QueryRow(context.Background(), `
SELECT reserved_tasks FROM parameter_sync_admission_reservations
WHERE request_id=$1 AND admission_class=$2`, request.ID, automaticAdmissionClass).Scan(&reservedTasks))
	assert.Equal(t, 37, reservedTasks)

	_, err = pool.Exec(context.Background(), `
UPDATE parameter_sync_requests SET status='running', updated_at=$2 WHERE id=$1`,
		request.ID, now.Add(2*time.Second))
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `
UPDATE parameter_sync_admission_reservations SET lease_until=$2::timestamptz - interval '1 second' WHERE request_id=$1`,
		request.ID, now.Add(2*time.Second))
	require.NoError(t, err)
	renewed, err := repo.ReconcileAutomaticAdmission(context.Background(), now.Add(2*time.Second), 500)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, renewed, 1)
	var renewedLease time.Time
	require.NoError(t, pool.QueryRow(context.Background(), `
SELECT status, lease_until FROM parameter_sync_admission_reservations
WHERE request_id=$1 AND admission_class=$2`, request.ID, automaticAdmissionClass).Scan(&reservationStatus, &renewedLease))
	assert.Equal(t, "reserved", reservationStatus)
	assert.True(t, renewedLease.After(now.Add(2*time.Second)), "active requests must renew rather than release capacity")

	_, err = pool.Exec(context.Background(), `
UPDATE parameter_sync_requests
SET status='queued', result_code='AUTOMATIC_BACKPRESSURE', admission_reason='retry', updated_at=$2
WHERE id=$1`, request.ID, now.Add(500*time.Millisecond))
	require.NoError(t, err)
	request.Status = RequestStatusQueued
	request.ResultCode = ResultCodeAutomaticBackpressure
	request.AdmissionReason = "retry"
	reused, err := repo.TryReserveAutomaticAdmission(context.Background(), request, now.Add(time.Second))
	require.NoError(t, err)
	assert.True(t, reused)
	stored, err = repo.GetRequest(context.Background(), request.ID)
	require.NoError(t, err)
	assert.Empty(t, stored.ResultCode)
	assert.Empty(t, stored.AdmissionReason)

	_, err = pool.Exec(context.Background(), `
UPDATE parameter_sync_requests
SET status='succeeded', completed_at=$2, updated_at=$2
WHERE id=$1`, request.ID, now.Add(time.Second))
	require.NoError(t, err)
	released, err := repo.ReconcileAutomaticAdmission(context.Background(), now.Add(2*time.Second), 500)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, released, 1)

	require.NoError(t, pool.QueryRow(context.Background(), `
SELECT status FROM parameter_sync_admission_reservations
WHERE request_id=$1 AND admission_class=$2`, request.ID, automaticAdmissionClass).Scan(&reservationStatus))
	assert.Equal(t, "released", reservationStatus)
}

func TestPGRepositoryQueuesActiveRunConflictAndReleasesAdmissionAtomically(t *testing.T) {
	pool := newParamSyncTestPool(t)
	repo := NewPGRepository(pool)
	now := time.Now().UTC()
	request := &SyncRequest{
		ID: uuid.New(), DeviceID: uuid.New(), DeviceSN: "TEST-AUTOMATIC-ACTIVE-CONFLICT",
		CallerType: "integration", TriggerReason: TriggerPeriodic, SyncScope: SyncScopeFull,
		Status: RequestStatusAccepted, Priority: 10, NextAttemptAt: now, CreatedAt: now, UpdatedAt: now,
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM parameter_sync_requests WHERE id=$1`, request.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM parameter_sync_device_state WHERE device_id=$1`, request.DeviceID)
		_, _ = repo.RepairAutomaticAdmissionCounters(context.Background(), time.Now().UTC())
	})
	allowed, err := repo.CreateAutomaticRequest(context.Background(), request, now)
	require.NoError(t, err)
	require.True(t, allowed)
	adjusted, err := repo.AdjustAutomaticAdmissionTasks(context.Background(), request, 5, now.Add(time.Second))
	require.NoError(t, err)
	require.True(t, adjusted)

	nextAttemptAt := now.Add(time.Minute)
	require.NoError(t, repo.QueueRequestAndReleaseAutomaticAdmission(
		context.Background(), request, nextAttemptAt, now.Add(2*time.Second),
	))
	stored, err := repo.GetRequest(context.Background(), request.ID)
	require.NoError(t, err)
	assert.Equal(t, RequestStatusQueued, stored.Status)
	assert.Equal(t, ResultCodeActiveSyncExists, stored.ResultCode)
	assert.WithinDuration(t, nextAttemptAt, stored.NextAttemptAt, time.Second)
	var reservationStatus string
	require.NoError(t, pool.QueryRow(context.Background(), `
SELECT status FROM parameter_sync_admission_reservations
WHERE request_id=$1 AND admission_class=$2`, request.ID, automaticAdmissionClass).Scan(&reservationStatus))
	assert.Equal(t, "released", reservationStatus)
}

func TestPGRepositoryCounterRepairWaitsForReservationTransaction(t *testing.T) {
	pool := newParamSyncTestPool(t)
	repo := NewPGRepository(pool)
	now := time.Now().UTC()
	request := insertParamSyncRequestForTest(t, pool, RequestStatusAccepted)
	bucketID := automaticAdmissionBucket(request.DeviceID)
	_, err := pool.Exec(context.Background(), `
INSERT INTO parameter_sync_admission_state
  (admission_class, bucket_id, active_run_limit, active_task_limit, reserved_runs, reserved_tasks)
VALUES ($1, $2, 32, 864, 0, 0)
ON CONFLICT (admission_class, bucket_id) DO UPDATE SET reserved_runs=0, reserved_tasks=0`,
		automaticAdmissionClass, bucketID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = repo.RepairAutomaticAdmissionCounters(context.Background(), time.Now().UTC())
	})

	tx, err := pool.Begin(context.Background())
	require.NoError(t, err)
	_, err = tx.Exec(context.Background(), `
UPDATE parameter_sync_admission_state
SET reserved_runs=reserved_runs+1, reserved_tasks=reserved_tasks+7
WHERE admission_class=$1 AND bucket_id=$2`, automaticAdmissionClass, bucketID)
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		_, repairErr := repo.RepairAutomaticAdmissionCounters(context.Background(), now.Add(time.Second))
		done <- repairErr
	}()
	select {
	case repairErr := <-done:
		require.Failf(t, "repair returned before reservation transaction committed", "error=%v", repairErr)
	case <-time.After(100 * time.Millisecond):
	}
	_, err = tx.Exec(context.Background(), `
INSERT INTO parameter_sync_admission_reservations
  (request_id, admission_class, bucket_id, reserved_runs, reserved_tasks, status, lease_until)
VALUES ($1, $2, $3, 1, 7, 'reserved', $4)`, request.ID, automaticAdmissionClass, bucketID, now.Add(time.Hour))
	require.NoError(t, err)
	require.NoError(t, tx.Commit(context.Background()))
	require.NoError(t, <-done)

	var stateRuns, stateTasks, truthRuns, truthTasks int
	require.NoError(t, pool.QueryRow(context.Background(), `
SELECT state.reserved_runs, state.reserved_tasks,
       COALESCE(SUM(reservation.reserved_runs) FILTER (WHERE reservation.status='reserved'), 0),
       COALESCE(SUM(reservation.reserved_tasks) FILTER (WHERE reservation.status='reserved'), 0)
FROM parameter_sync_admission_state state
LEFT JOIN parameter_sync_admission_reservations reservation
  ON reservation.admission_class=state.admission_class AND reservation.bucket_id=state.bucket_id
WHERE state.admission_class=$1 AND state.bucket_id=$2
GROUP BY state.reserved_runs, state.reserved_tasks`, automaticAdmissionClass, bucketID).
		Scan(&stateRuns, &stateTasks, &truthRuns, &truthTasks))
	assert.Equal(t, truthRuns, stateRuns)
	assert.Equal(t, truthTasks, stateTasks)
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

func TestPGRepositoryCreateAutomaticDeviceOnlineRequestQueuesDuringBackoff(t *testing.T) {
	pool := newParamSyncTestPool(t)
	repo := NewPGRepository(pool)
	now := time.Now().UTC()
	nextAttemptAt := now.Add(5 * time.Minute)
	deviceID := uuid.New()
	request := &SyncRequest{
		ID:            uuid.New(),
		DeviceID:      deviceID,
		DeviceSN:      "TEST-DEVICE-ONLINE-BACKOFF-" + uuid.NewString(),
		CallerType:    "provision",
		TriggerReason: TriggerDeviceOnline,
		SyncScope:     SyncScopeFull,
		Status:        RequestStatusAccepted,
		Priority:      10,
		NextAttemptAt: now,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	_, err := pool.Exec(context.Background(), `
INSERT INTO parameter_sync_device_state
  (device_id, consecutive_failures, next_auto_sync_at, updated_at)
VALUES ($1, 1, $2, $3)`, deviceID, nextAttemptAt, now)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM parameter_sync_requests WHERE id=$1`, request.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM parameter_sync_device_state WHERE device_id=$1`, deviceID)
	})

	allowed, err := repo.CreateAutomaticRequest(context.Background(), request, now)

	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, RequestStatusQueued, request.Status)
	assert.Equal(t, ResultCodeAutomaticBackoff, request.ResultCode)
	assert.Nil(t, request.CompletedAt)
	assert.WithinDuration(t, nextAttemptAt, request.NextAttemptAt, time.Second)
	stored, err := repo.GetRequest(context.Background(), request.ID)
	require.NoError(t, err)
	assert.Equal(t, RequestStatusQueued, stored.Status)
	assert.Nil(t, stored.CompletedAt)
	assert.WithinDuration(t, nextAttemptAt, stored.NextAttemptAt, time.Second)

	service := NewService(repo, stubPlanner{plan: &Plan{
		Batches: []TaskBatch{{Paths: []string{"Device.DeviceInfo."}}},
	}})
	service.now = func() time.Time { return nextAttemptAt.Add(time.Second) }
	dispatched, err := service.DispatchQueued(context.Background(), 10)
	require.NoError(t, err)
	assert.Equal(t, 1, dispatched)

	stored, err = repo.GetRequest(context.Background(), request.ID)
	require.NoError(t, err)
	assert.Equal(t, RequestStatusRunning, stored.Status)
	assert.NotNil(t, stored.RunID)
	assert.Empty(t, stored.ResultCode)
	assert.Empty(t, stored.ErrorMessage)
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

type failFirstRecoveryProcessor struct {
	calls    int
	payloads []event.ParamSyncTaskResultPayload
}

func (p *failFirstRecoveryProcessor) Process(_ context.Context, payload event.ParamSyncTaskResultPayload) (ResultProcessOutcome, error) {
	p.calls++
	if p.calls == 1 {
		return ResultProcessOutcome{}, errors.New("permanent malformed result")
	}
	p.payloads = append(p.payloads, payload)
	return ResultProcessOutcome{}, nil
}

func TestRecoverMissingResultsContinuesAfterEarlierRunFails(t *testing.T) {
	pool := newParamSyncTestPool(t)
	type fixture struct {
		runID uuid.UUID
		task  *task.Task
	}
	createFixture := func(offset time.Duration) fixture {
		req := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
		runID := uuid.New()
		_, err := pool.Exec(context.Background(), `INSERT INTO parameter_sync_runs
(id, request_id, device_id, device_sn, trigger_reason, sync_scope, status,
 expected_task_count, terminal_task_count, processed_task_count, failed_task_count, started_at)
VALUES ($1, $2, $3, $4, 'manual', 'full', 'waiting_device', 1, 1, 0, 0, $5)`,
			runID, req.ID, req.DeviceID, req.DeviceSN, time.Now().UTC().Add(offset))
		require.NoError(t, err)
		_, err = pool.Exec(context.Background(), `UPDATE parameter_sync_requests SET run_id=$2, active_run_id=$2 WHERE id=$1`, req.ID, runID)
		require.NoError(t, err)

		completed := task.NewTask(&task.CreateTaskRequest{
			DeviceSN: req.DeviceSN, Method: "GetParameterValues",
			Params: []byte(`{"names":["Device.Good"]}`), CommandKey: "param-sync-" + runID.String() + "-0",
			Source: task.TaskSourceParamSync, SourceID: runID.String(), CreatorID: req.ID.String(),
		})
		require.NoError(t, task.NewPgTaskRepository(pool).Create(context.Background(), completed))
		t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM device_tasks WHERE id=$1`, completed.ID) })
		_, err = pool.Exec(context.Background(), `UPDATE device_tasks SET status='completed', completed_at=now() WHERE id=$1`, completed.ID)
		require.NoError(t, err)
		return fixture{runID: runID, task: completed}
	}

	first := createFixture(-time.Minute)
	second := createFixture(0)
	processor := &failFirstRecoveryProcessor{}
	recovered, err := NewReconciler(pool, nil, nil).WithResultProcessor(processor).
		RecoverMissingResults(context.Background(), 20, 200, 200)

	require.ErrorContains(t, err, "permanent malformed result")
	assert.Equal(t, 1, recovered)
	assert.Equal(t, 2, processor.calls)
	require.Len(t, processor.payloads, 1)
	assert.Equal(t, second.runID, processor.payloads[0].RunID)
	assert.Equal(t, second.task.ID, processor.payloads[0].TaskID)
	assert.NotEqual(t, first.runID, processor.payloads[0].RunID)
}

func TestRecoverMissingResultsWithPGProcessorFinalizesSucceededRun(t *testing.T) {
	pool := newParamSyncTestPool(t)
	req := insertParamSyncRequestForTest(t, pool, RequestStatusRunning)
	insertReleaseCandidateForTest(t, pool, req.DeviceID, true)
	oldSyncAt := time.Now().UTC().Add(-6 * time.Hour)
	_, err := pool.Exec(context.Background(), `
UPDATE devices
SET last_param_sync_at=$2, last_param_sync_failed_at=$3, last_param_sync_error='previous failure'
WHERE id=$1`, req.DeviceID, oldSyncAt, oldSyncAt.Add(time.Hour))
	require.NoError(t, err)
	runID := uuid.New()
	_, err = pool.Exec(context.Background(), `INSERT INTO parameter_sync_runs
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
	var lastParamSyncAt time.Time
	var failedAt *time.Time
	var syncErr *string
	require.NoError(t, pool.QueryRow(context.Background(), `
SELECT last_param_sync_at, last_param_sync_failed_at, last_param_sync_error
FROM devices WHERE id=$1`, req.DeviceID).Scan(&lastParamSyncAt, &failedAt, &syncErr))
	assert.True(t, lastParamSyncAt.After(oldSyncAt), "partial sync success must refresh devices.last_param_sync_at")
	assert.Nil(t, failedAt)
	assert.Nil(t, syncErr)
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

func insertReleaseCandidateForTest(
	t *testing.T,
	pool *pgxpool.Pool,
	deviceID uuid.UUID,
	online bool,
) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO devices (
			id, serial_number, oui, carrier, technology, lifecycle_state, is_online
		) VALUES ($1, $2, 'AABBCC', $3, $4, $5, $6)
	`, deviceID, "TEST-RELEASE-"+deviceID.String(), model.CarrierCMCC,
		model.TechLTE, model.LifecycleCommissioned, online)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM devices WHERE id=$1`, deviceID)
	})
}

func insertReleaseRequestForPGRepoTest(
	t *testing.T,
	pool *pgxpool.Pool,
	campaignID uuid.UUID,
	deviceID uuid.UUID,
	status RequestStatus,
) {
	t.Helper()
	now := time.Now().UTC()
	sourceEventID := "omc_upgrade:" + campaignID.String() + ":" + deviceID.String()
	req := &SyncRequest{
		ID:            uuid.New(),
		DeviceID:      deviceID,
		DeviceSN:      "TEST-RELEASE-" + deviceID.String(),
		CallerType:    "test",
		TriggerReason: TriggerOMCUpgrade,
		SyncScope:     SyncScopeFull,
		Status:        status,
		Priority:      10,
		NextAttemptAt: now,
		CampaignID:    &campaignID,
		SourceEventID: &sourceEventID,
		CreatedAt:     now,
		UpdatedAt:     now,
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
}

func TestPGRepository_ListReleaseCandidatesFiltersState(t *testing.T) {
	pool := newParamSyncTestPool(t)
	var preexistingEligible int
	require.NoError(t, pool.QueryRow(context.Background(), `
		SELECT count(*)
		FROM devices d
		LEFT JOIN parameter_sync_device_state state ON state.device_id = d.id
		WHERE d.deleted_at IS NULL
		  AND d.lifecycle_state = $1
		  AND d.is_online = true
		  AND (state.next_auto_sync_at IS NULL OR state.next_auto_sync_at <= now())
	`, model.LifecycleCommissioned).Scan(&preexistingEligible))
	campaignID := uuid.New()
	noRequestID := uuid.New()
	offlineID := uuid.New()
	succeededID := uuid.New()
	backedOffID := uuid.New()
	retryableID := uuid.New()

	insertReleaseCandidateForTest(t, pool, noRequestID, true)
	insertReleaseCandidateForTest(t, pool, offlineID, false)
	insertReleaseCandidateForTest(t, pool, succeededID, true)
	insertReleaseCandidateForTest(t, pool, backedOffID, true)
	insertReleaseCandidateForTest(t, pool, retryableID, true)
	insertReleaseRequestForPGRepoTest(t, pool, campaignID, succeededID, RequestStatusSucceeded)
	insertReleaseRequestForPGRepoTest(t, pool, campaignID, retryableID, RequestStatusFailed)

	now := time.Now().UTC()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO parameter_sync_device_state (
			device_id, consecutive_failures, next_auto_sync_at, updated_at
		) VALUES
			($1, 1, $2, $3),
			($4, 1, $5, $3)
	`, backedOffID, now.Add(time.Hour), now, retryableID, now.Add(-time.Minute))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(
			context.Background(),
			`DELETE FROM parameter_sync_device_state WHERE device_id = ANY($1)`,
			[]uuid.UUID{backedOffID, retryableID},
		)
	})

	devices, err := NewPGRepository(pool).ListReleaseCandidates(
		context.Background(),
		campaignID,
		preexistingEligible+5,
	)

	require.NoError(t, err)
	ids := make(map[uuid.UUID]bool, len(devices))
	for _, dev := range devices {
		ids[dev.ID] = true
	}
	assert.True(t, ids[noRequestID])
	assert.True(t, ids[retryableID])
	assert.False(t, ids[offlineID])
	assert.False(t, ids[succeededID])
	assert.False(t, ids[backedOffID])

	postBoundaryID := uuid.New()
	insertReleaseCandidateForTest(t, pool, postBoundaryID, true)
	devices, err = NewPGRepository(pool).ListReleaseCandidates(context.Background(), campaignID, preexistingEligible+6)
	require.NoError(t, err)
	for _, dev := range devices {
		assert.NotEqual(t, postBoundaryID, dev.ID, "devices registered after the persisted release boundary must be excluded")
	}
}

func TestPGRepository_ListReleaseCandidatesPrioritizesUnattemptedDevices(t *testing.T) {
	pool := newParamSyncTestPool(t)
	campaignID := uuid.New()
	attemptedLowID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	unattemptedHighID := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	insertReleaseCandidateForTest(t, pool, attemptedLowID, true)
	insertReleaseCandidateForTest(t, pool, unattemptedHighID, true)

	now := time.Now().UTC()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO parameter_sync_device_state (
			device_id, consecutive_failures, last_attempt_at,
			next_auto_sync_at, updated_at
		) VALUES ($1, 1, $2, $3, $4)
	`, attemptedLowID, now.Add(-2*time.Minute), now.Add(-time.Minute), now)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(
			context.Background(),
			`DELETE FROM parameter_sync_device_state WHERE device_id=$1`,
			attemptedLowID,
		)
	})

	devices, err := NewPGRepository(pool).ListReleaseCandidates(
		context.Background(),
		campaignID,
		1,
	)

	require.NoError(t, err)
	require.Len(t, devices, 1)
	assert.Equal(t, unattemptedHighID, devices[0].ID)
}

func TestPGRepository_ListReleaseCandidatesSupersedesOlderQueuedCampaigns(t *testing.T) {
	pool := newParamSyncTestPool(t)
	oldCampaignID := uuid.New()
	newCampaignID := uuid.New()
	legacyCampaignID := uuid.New()
	queuedDeviceID := uuid.New()
	runningDeviceID := uuid.New()
	newQueuedDeviceID := uuid.New()
	legacyQueuedDeviceID := uuid.New()
	insertReleaseCandidateForTest(t, pool, queuedDeviceID, true)
	insertReleaseCandidateForTest(t, pool, runningDeviceID, true)
	insertReleaseCandidateForTest(t, pool, newQueuedDeviceID, true)
	insertReleaseCandidateForTest(t, pool, legacyQueuedDeviceID, true)
	_, err := NewPGRepository(pool).ListReleaseCandidates(context.Background(), oldCampaignID, 1)
	require.NoError(t, err)
	insertReleaseRequestForPGRepoTest(t, pool, oldCampaignID, queuedDeviceID, RequestStatusQueued)
	insertReleaseRequestForPGRepoTest(t, pool, oldCampaignID, runningDeviceID, RequestStatusRunning)
	now := time.Now().UTC()
	legacyRequest := &SyncRequest{
		ID: uuid.New(), DeviceID: legacyQueuedDeviceID,
		DeviceSN:   "TEST-RELEASE-" + legacyQueuedDeviceID.String(),
		CallerType: "test", TriggerReason: TriggerOMCUpgrade, SyncScope: SyncScopeFull,
		Status: RequestStatusAccepted, Priority: 10, NextAttemptAt: now,
		CampaignID: &legacyCampaignID, CreatedAt: now, UpdatedAt: now,
	}
	repo := NewPGRepository(pool)
	allowed, err := repo.CreateAutomaticRequest(context.Background(), legacyRequest, now)
	require.NoError(t, err)
	require.True(t, allowed)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `UPDATE parameter_sync_requests SET status='cancelled', completed_at=now() WHERE id=$1 AND status NOT IN ('succeeded','failed','timed_out','cancelled','deduplicated','rejected')`, legacyRequest.ID)
		_, _ = repo.ReconcileAutomaticAdmission(context.Background(), time.Now().UTC(), 500)
		_, _ = pool.Exec(context.Background(), `DELETE FROM parameter_sync_requests WHERE id=$1`, legacyRequest.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM parameter_sync_device_state WHERE device_id=$1`, legacyRequest.DeviceID)
	})

	_, err = NewPGRepository(pool).ListReleaseCandidates(context.Background(), newCampaignID, 1)
	require.NoError(t, err)

	var cancelled, queued, running int
	err = pool.QueryRow(context.Background(), `
		SELECT
			count(*) FILTER (WHERE status='cancelled'),
			count(*) FILTER (WHERE status='queued'),
			count(*) FILTER (WHERE status='running')
		FROM parameter_sync_requests
		WHERE campaign_id=$1
	`, oldCampaignID).Scan(&cancelled, &queued, &running)
	require.NoError(t, err)
	assert.Equal(t, 1, cancelled)
	assert.Zero(t, queued)
	assert.Equal(t, 1, running, "already executing work must finish through the normal lifecycle")
	var legacyStatus RequestStatus
	var legacyResult ResultCode
	var legacyCompletedAt *time.Time
	require.NoError(t, pool.QueryRow(context.Background(), `
		SELECT status, result_code, completed_at
		FROM parameter_sync_requests
		WHERE id=$1
	`, legacyRequest.ID).Scan(&legacyStatus, &legacyResult, &legacyCompletedAt))
	assert.Equal(t, RequestStatusCancelled, legacyStatus)
	assert.Equal(t, ResultCodeSupersededRelease, legacyResult)
	assert.NotNil(t, legacyCompletedAt, "legacy campaigns without a boundary must still be terminalized")
	_, err = repo.ReconcileAutomaticAdmission(context.Background(), time.Now().UTC(), 500)
	require.NoError(t, err)
	var legacyReservationStatus string
	require.NoError(t, pool.QueryRow(context.Background(), `
		SELECT status FROM parameter_sync_admission_reservations
		WHERE request_id=$1 AND admission_class=$2
	`, legacyRequest.ID, automaticAdmissionClass).Scan(&legacyReservationStatus))
	assert.Equal(t, "released", legacyReservationStatus)

	insertReleaseRequestForPGRepoTest(t, pool, newCampaignID, newQueuedDeviceID, RequestStatusQueued)
	_, err = NewPGRepository(pool).ListReleaseCandidates(context.Background(), oldCampaignID, 1)
	require.NoError(t, err)
	var newQueued int
	require.NoError(t, pool.QueryRow(context.Background(), `
		SELECT count(*) FROM parameter_sync_requests
		WHERE campaign_id=$1 AND status='queued'
	`, newCampaignID).Scan(&newQueued))
	assert.Equal(t, 1, newQueued, "an older rolling instance must not cancel the newer release campaign")
}

func insertRegisteredSyncCandidateForTest(t *testing.T, pool *pgxpool.Pool, deviceID uuid.UUID, deleted bool) string {
	t.Helper()
	sourceEventID := "device_registered:" + deviceID.String()
	extensionData, err := json.Marshal(map[string]string{
		"_system_registration_source_event_id": "inform:" + uuid.NewString(),
	})
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), `
		INSERT INTO devices (
			id, serial_number, oui, carrier, technology, lifecycle_state, extension_data, deleted_at
		) VALUES ($1, $2, 'AABBCC', $3, $4, $5, $6, CASE WHEN $7 THEN now() ELSE NULL END)
	`, deviceID, "TEST-REGISTERED-"+deviceID.String(), model.CarrierCMCC,
		model.TechLTE, model.LifecycleRegistered, extensionData, deleted)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM devices WHERE id=$1`, deviceID)
	})
	return sourceEventID
}

func insertRegisteredSyncRequestForTest(
	t *testing.T,
	pool *pgxpool.Pool,
	deviceID uuid.UUID,
	sourceEventID string,
	status RequestStatus,
	activeRunStatus *RunStatus,
) {
	t.Helper()
	now := time.Now().UTC()
	req := &SyncRequest{
		ID:            uuid.New(),
		DeviceID:      deviceID,
		DeviceSN:      "TEST-REGISTERED-" + deviceID.String(),
		CallerType:    "provision",
		TriggerReason: TriggerDeviceRegistered,
		SyncScope:     SyncScopeFull,
		Status:        status,
		Priority:      10,
		NextAttemptAt: now,
		SourceEventID: &sourceEventID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if status.Terminal() {
		req.CompletedAt = &now
	}
	query, args, err := buildCreateRequest(req)
	require.NoError(t, err)
	_, err = pool.Exec(context.Background(), query, args...)
	require.NoError(t, err)
	if activeRunStatus != nil {
		runID := uuid.New()
		_, err = pool.Exec(context.Background(), `
			INSERT INTO parameter_sync_runs (
				id, request_id, device_id, device_sn, trigger_reason, sync_scope, status
			) VALUES ($1, $2, $3, $4, 'device_registered', 'full', $5)
		`, runID, req.ID, deviceID, req.DeviceSN, *activeRunStatus)
		require.NoError(t, err)
		_, err = pool.Exec(context.Background(), `UPDATE parameter_sync_requests SET active_run_id=$2 WHERE id=$1`, req.ID, runID)
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM parameter_sync_requests WHERE id=$1`, req.ID)
	})
}

func TestPGRepository_ListRegisteredSyncCandidatesRetriesOnlyUncoveredRegistrations(t *testing.T) {
	pool := newParamSyncTestPool(t)
	repo, ok := any(NewPGRepository(pool)).(interface {
		ListRegisteredSyncCandidates(context.Context, int) ([]*model.Device, error)
	})
	require.True(t, ok, "PG repository must find registrations requiring durable reconciliation")

	noRequestID := uuid.New()
	failedID := uuid.New()
	rejectedID := uuid.New()
	deduplicatedFailedID := uuid.New()
	coveredID := uuid.New()
	deduplicatedActiveID := uuid.New()
	backedOffID := uuid.New()
	deletedID := uuid.New()

	noRequestSource := insertRegisteredSyncCandidateForTest(t, pool, noRequestID, false)
	failedSource := insertRegisteredSyncCandidateForTest(t, pool, failedID, false)
	rejectedSource := insertRegisteredSyncCandidateForTest(t, pool, rejectedID, false)
	deduplicatedFailedSource := insertRegisteredSyncCandidateForTest(t, pool, deduplicatedFailedID, false)
	coveredSource := insertRegisteredSyncCandidateForTest(t, pool, coveredID, false)
	deduplicatedActiveSource := insertRegisteredSyncCandidateForTest(t, pool, deduplicatedActiveID, false)
	backedOffSource := insertRegisteredSyncCandidateForTest(t, pool, backedOffID, false)
	deletedSource := insertRegisteredSyncCandidateForTest(t, pool, deletedID, true)
	_ = noRequestSource

	insertRegisteredSyncRequestForTest(t, pool, failedID, failedSource, RequestStatusFailed, nil)
	insertRegisteredSyncRequestForTest(t, pool, rejectedID, rejectedSource, RequestStatusRejected, nil)
	failedRun := RunStatusFailed
	insertRegisteredSyncRequestForTest(t, pool, deduplicatedFailedID, deduplicatedFailedSource, RequestStatusDeduplicated, &failedRun)
	insertRegisteredSyncRequestForTest(t, pool, coveredID, coveredSource, RequestStatusSucceeded, nil)
	executingRun := RunStatusExecuting
	insertRegisteredSyncRequestForTest(t, pool, deduplicatedActiveID, deduplicatedActiveSource, RequestStatusDeduplicated, &executingRun)
	insertRegisteredSyncRequestForTest(t, pool, backedOffID, backedOffSource, RequestStatusFailed, nil)
	insertRegisteredSyncRequestForTest(t, pool, deletedID, deletedSource, RequestStatusFailed, nil)

	_, err := pool.Exec(context.Background(), `
		INSERT INTO parameter_sync_device_state (device_id, consecutive_failures, next_auto_sync_at, updated_at)
		VALUES ($1, 1, now() + interval '1 hour', now())
	`, backedOffID)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM parameter_sync_device_state WHERE device_id=$1`, backedOffID)
	})

	devices, err := repo.ListRegisteredSyncCandidates(context.Background(), 20)

	require.NoError(t, err)
	ids := make(map[uuid.UUID]bool, len(devices))
	for _, dev := range devices {
		ids[dev.ID] = true
	}
	assert.True(t, ids[noRequestID])
	assert.True(t, ids[failedID])
	assert.True(t, ids[rejectedID])
	assert.True(t, ids[deduplicatedFailedID])
	assert.False(t, ids[coveredID])
	assert.False(t, ids[deduplicatedActiveID])
	assert.False(t, ids[backedOffID])
	assert.False(t, ids[deletedID])
}
