package paramsync

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/task"
)

type Reconciler struct {
	pool    *pgxpool.Pool
	bus     event.EventBus
	metrics *Metrics
	now     func() time.Time
}

type rowScanner interface {
	Scan(dest ...any) error
}

type missingTaskResult struct {
	taskID       string
	deviceSN     string
	status       task.TaskStatus
	errorCode    string
	errorMessage string
	runID        uuid.UUID
	requestID    uuid.UUID
}

// scanMissingTaskResult preserves the nullable device_tasks.error_code contract.
// A task can terminate without a TR-069 fault code (for example when a run is
// cancelled), so absence must remain distinct from a concrete non-zero code.
func scanMissingTaskResult(row rowScanner) (missingTaskResult, error) {
	var result missingTaskResult
	var errorCode pgtype.Int4
	if err := row.Scan(
		&result.taskID,
		&result.deviceSN,
		&result.status,
		&errorCode,
		&result.errorMessage,
		&result.runID,
		&result.requestID,
	); err != nil {
		return missingTaskResult{}, err
	}
	if errorCode.Valid {
		result.errorCode = canonicalTaskErrorCode(int(errorCode.Int32))
	}
	return result, nil
}

func NewReconciler(pool *pgxpool.Pool, bus event.EventBus, metrics *Metrics) *Reconciler {
	return &Reconciler{pool: pool, bus: bus, metrics: metrics, now: func() time.Time { return time.Now().UTC() }}
}

func (r *Reconciler) SweepExpiredRequests(ctx context.Context, limit int) (int64, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
SELECT id, COALESCE(active_run_id, run_id), deadline_at
FROM parameter_sync_requests
WHERE status IN ('accepted','queued','running') AND deadline_at IS NOT NULL AND deadline_at < $1
ORDER BY deadline_at LIMIT $2`, r.now(), limit)
	if err != nil {
		return 0, fmt.Errorf("list expired parameter sync requests: %w", err)
	}
	type candidate struct {
		requestID uuid.UUID
		runID     pgtype.UUID
		deadline  time.Time
	}
	var candidates []candidate
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.requestID, &item.runID, &item.deadline); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan expired parameter sync request: %w", err)
		}
		candidates = append(candidates, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate expired parameter sync requests: %w", err)
	}
	rows.Close()
	var swept int64
	for _, item := range candidates {
		changed, err := r.sweepExpiredRequest(ctx, item.requestID, item.runID)
		if err != nil {
			return swept, err
		}
		if changed {
			swept++
		}
	}
	if r.metrics != nil && swept > 0 {
		r.metrics.ReconcileRepairs.WithLabelValues("request_timeout").Add(float64(swept))
	}
	return swept, nil
}

func (r *Reconciler) sweepExpiredRequest(ctx context.Context, requestID uuid.UUID, candidateRunID pgtype.UUID) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin expired parameter sync request: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if candidateRunID.Valid {
		var ignored uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT id FROM parameter_sync_runs WHERE id=$1 FOR UPDATE`, candidateRunID.Bytes).Scan(&ignored); err != nil && err != pgx.ErrNoRows {
			return false, fmt.Errorf("lock expired parameter sync run: %w", err)
		}
	}
	var status RequestStatus
	var deadline *time.Time
	var actualRunID pgtype.UUID
	if err := tx.QueryRow(ctx, `SELECT status, deadline_at, COALESCE(active_run_id, run_id) FROM parameter_sync_requests WHERE id=$1 FOR UPDATE`, requestID).
		Scan(&status, &deadline, &actualRunID); err != nil {
		return false, fmt.Errorf("lock expired parameter sync request: %w", err)
	}
	now := r.now()
	if status.Terminal() || deadline == nil || !deadline.Before(now) || actualRunID != candidateRunID {
		return false, nil
	}
	if _, err := tx.Exec(ctx, `UPDATE parameter_sync_requests SET status='timed_out', result_code='DEADLINE_EXCEEDED',
error_message='parameter sync deadline exceeded', completed_at=$2, updated_at=$2, active_run_id=NULL WHERE id=$1`, requestID, now); err != nil {
		return false, fmt.Errorf("time out parameter sync request: %w", err)
	}
	if actualRunID.Valid {
		runID := uuid.UUID(actualRunID.Bytes)
		if _, err := tx.Exec(ctx, `UPDATE parameter_sync_runs SET status='cancelling', error_message='parameter sync deadline exceeded', version=version+1
WHERE id=$1 AND status NOT IN ('succeeded','failed','cancelled')`, runID); err != nil {
			return false, fmt.Errorf("cancel expired parameter sync run: %w", err)
		}
		if err := cancelUnsentRunTasks(ctx, tx, &SyncRun{ID: runID, ErrorMessage: "parameter sync deadline exceeded"}, now); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit expired parameter sync request: %w", err)
	}
	return true, nil
}

// ReconcileStalledRequests closes requests that were durably accepted but did
// not reach planning before the process stopped. Accepted is a transient state
// in the synchronous submit path, so an old row without a run cannot recover by
// itself and must not remain visible as in progress forever.
func (r *Reconciler) ReconcileStalledRequests(ctx context.Context, olderThan time.Time, limit int) (int64, error) {
	if limit <= 0 {
		limit = 100
	}
	const query = `
WITH stalled AS (
  SELECT id FROM parameter_sync_requests
  WHERE status='accepted' AND run_id IS NULL AND active_run_id IS NULL
    AND updated_at < $1
  ORDER BY updated_at
  LIMIT $2 FOR UPDATE SKIP LOCKED
)
UPDATE parameter_sync_requests req SET
  status='failed', result_code='RESULT_PROCESSING_FAILED',
  error_message='parameter sync request did not advance beyond accepted state',
  completed_at=$3, updated_at=$3
FROM stalled WHERE req.id=stalled.id`
	tag, err := r.pool.Exec(ctx, query, olderThan, limit, r.now())
	if err != nil {
		return 0, fmt.Errorf("reconcile stalled parameter sync requests: %w", err)
	}
	if r.metrics != nil && tag.RowsAffected() > 0 {
		r.metrics.ReconcileRepairs.WithLabelValues("stalled_requests").Add(float64(tag.RowsAffected()))
	}
	return tag.RowsAffected(), nil
}

// ReconcileStalledRuns releases the per-device active-run gate after a process
// stops between committing a run and durably dispatching its task plan.
// planning/enqueuing are synchronous submit states and must not survive for
// minutes under normal operation.
func (r *Reconciler) ReconcileStalledRuns(ctx context.Context, olderThan time.Time, limit int) (int64, error) {
	if limit <= 0 {
		limit = 100
	}
	const query = `
WITH stalled AS (
  SELECT id, request_id FROM parameter_sync_runs
  WHERE status IN ('planning','enqueuing') AND started_at < $1
  ORDER BY started_at
  LIMIT $2 FOR UPDATE SKIP LOCKED
), updated_runs AS (
  UPDATE parameter_sync_runs run SET
    status='failed', error_message='parameter sync task plan was not durably dispatched',
    completed_at=$3, version=version+1
  FROM stalled WHERE run.id=stalled.id
  RETURNING stalled.request_id, run.id, run.device_id, run.status, run.sync_scope, run.trigger_reason
), updated_requests AS (
  UPDATE parameter_sync_requests req SET
  status='failed', result_code='RESULT_PROCESSING_FAILED',
  error_message='parameter sync task plan was not durably dispatched',
  active_run_id=NULL, completed_at=COALESCE(completed_at,$3), updated_at=$3
  FROM updated_runs WHERE req.id=updated_runs.request_id AND req.run_id=updated_runs.id
    AND req.status IN ('accepted','queued','running')
  RETURNING req.id
), updated_devices AS (
  UPDATE devices d SET last_param_sync_failed_at=$3,
    last_param_sync_error='parameter sync task plan was not durably dispatched'
  FROM updated_runs run WHERE d.id=run.device_id AND run.sync_scope='full'
  RETURNING d.id
), automatic_state AS (
  INSERT INTO parameter_sync_device_state
    (device_id, consecutive_failures, next_auto_sync_at, last_error, updated_at)
  SELECT run.device_id, 1, $3 + interval '1 minute',
    'parameter sync task plan was not durably dispatched', $3
  FROM updated_runs run
  WHERE run.trigger_reason IN ('device_online','periodic','firmware_changed')
  ON CONFLICT (device_id) DO UPDATE SET
    consecutive_failures=parameter_sync_device_state.consecutive_failures+1,
    next_auto_sync_at=$3 + CASE
      WHEN parameter_sync_device_state.consecutive_failures+1 <= 1 THEN interval '1 minute'
      WHEN parameter_sync_device_state.consecutive_failures+1 = 2 THEN interval '5 minutes'
      WHEN parameter_sync_device_state.consecutive_failures+1 = 3 THEN interval '15 minutes'
      WHEN parameter_sync_device_state.consecutive_failures+1 = 4 THEN interval '1 hour'
      ELSE interval '6 hours' END,
    last_error=EXCLUDED.last_error, updated_at=EXCLUDED.updated_at
  RETURNING device_id
), inserted_outbox AS (
  INSERT INTO parameter_sync_outbox
    (event_type, aggregate_type, aggregate_id, dedupe_key, payload, created_at, updated_at)
  SELECT $4::text, 'run', run.id, $4::text || ':' || run.id::text,
    jsonb_build_object('request_id', run.request_id, 'run_id', run.id,
      'device_id', run.device_id, 'status', run.status), $3, $3
  FROM updated_runs run
  ON CONFLICT (dedupe_key) DO NOTHING
  RETURNING id
)
SELECT count(*) FROM updated_runs`
	var count int64
	if err := r.pool.QueryRow(ctx, query, olderThan, limit, r.now(), event.SubjectParamSyncRunFailed).Scan(&count); err != nil {
		return 0, fmt.Errorf("reconcile stalled parameter sync runs: %w", err)
	}
	if r.metrics != nil && count > 0 {
		r.metrics.ReconcileRepairs.WithLabelValues("stalled_runs").Add(float64(count))
	}
	return count, nil
}

// ReconcileCancellingRuns closes the recovery-vs-cancel race. A recovery task
// created just after the initial cancellation sweep would otherwise keep the
// run at expected=N, terminal=N-1 until its 30-minute task expiry, while the
// active-run API continues to report a permanent-looking sync in progress.
func (r *Reconciler) ReconcileCancellingRuns(ctx context.Context, limit int) (int64, error) {
	if limit <= 0 {
		limit = 100
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin cancelling parameter sync reconciliation: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	rows, err := tx.Query(ctx, `SELECT id FROM parameter_sync_runs
WHERE status='cancelling' ORDER BY started_at LIMIT $1 FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return 0, fmt.Errorf("select cancelling parameter sync runs: %w", err)
	}
	var runIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan cancelling parameter sync run: %w", err)
		}
		runIDs = append(runIDs, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate cancelling parameter sync runs: %w", err)
	}
	rows.Close()

	now := r.now()
	var finalized int64
	for _, runID := range runIDs {
		run, err := loadRunForUpdate(ctx, tx, runID)
		if err != nil {
			return 0, err
		}
		if err := cancelUnsentRunTasks(ctx, tx, run, now); err != nil {
			return 0, err
		}
		counts, err := loadAuthoritativeRunCounts(ctx, tx, run.ID)
		if err != nil {
			return 0, err
		}
		applyAuthoritativeRunCounts(run, counts)
		if run.ReadyToFinalize() {
			if err := finalizeConvergedFailedRun(ctx, tx, run, now); err != nil {
				return 0, err
			}
			finalized++
		} else if err := updateRunProgress(ctx, tx, run, RunStatusCancelling); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit cancelling parameter sync reconciliation: %w", err)
	}
	if r.metrics != nil && finalized > 0 {
		r.metrics.ReconcileRepairs.WithLabelValues("cancelling_runs").Add(float64(finalized))
	}
	return finalized, nil
}

func (r *Reconciler) ReconcileRunCounts(ctx context.Context) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin parameter sync run count reconciliation: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	// A previous implementation could commit a result in "received" before its
	// processing state was finalized. Terminal runs cannot be processed again, so
	// normalize those durable historical rows from their success bit first.
	const normalizeResults = `
UPDATE parameter_sync_task_results res SET
  status=CASE WHEN res.success THEN 'processed' ELSE 'failed' END,
  processed_at=COALESCE(res.processed_at, $1)
FROM parameter_sync_runs run, device_tasks t
WHERE res.status='received' AND run.id=res.run_id
  AND run.status IN ('succeeded','failed','cancelled')
  AND t.id=res.task_id AND t.source='param_sync' AND t.source_id=run.id
  AND t.status IN ('completed','failed','expired','cancelled')`
	normalized, err := tx.Exec(ctx, normalizeResults, r.now())
	if err != nil {
		return 0, fmt.Errorf("normalize historical parameter sync results: %w", err)
	}

	const query = `
WITH actual AS (
  SELECT run.id,
    count(t.id)::int AS expected,
    count(t.id) FILTER (WHERE t.status IN ('completed','failed','expired','cancelled'))::int AS terminal,
    count(res.task_id) FILTER (
      WHERE t.status IN ('completed','failed','expired','cancelled')
        AND res.status IN ('processed','failed')
    )::int AS processed,
    count(res.task_id) FILTER (
      WHERE t.status IN ('completed','failed','expired','cancelled')
        AND res.status IN ('processed','failed')
        AND (res.status='failed' OR NOT res.success)
    )::int AS failed
  FROM parameter_sync_runs run
  LEFT JOIN device_tasks t ON t.source='param_sync' AND t.source_id=run.id
  LEFT JOIN parameter_sync_task_results res ON res.run_id=run.id AND res.task_id=t.id
  GROUP BY run.id
)
UPDATE parameter_sync_runs run SET
  expected_task_count=actual.expected, terminal_task_count=actual.terminal,
  processed_task_count=actual.processed, failed_task_count=actual.failed, version=version+1
FROM actual WHERE run.id=actual.id AND (
  run.expected_task_count<>actual.expected OR run.terminal_task_count<>actual.terminal OR
  run.processed_task_count<>actual.processed OR run.failed_task_count<>actual.failed
)`
	tag, err := tx.Exec(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("reconcile parameter sync run counts: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit parameter sync run count reconciliation: %w", err)
	}
	if r.metrics != nil && normalized.RowsAffected() > 0 {
		r.metrics.ReconcileRepairs.WithLabelValues("historical_results").Add(float64(normalized.RowsAffected()))
	}
	if r.metrics != nil && tag.RowsAffected() > 0 {
		r.metrics.ReconcileRepairs.WithLabelValues("run_counts").Add(float64(tag.RowsAffected()))
	}
	return tag.RowsAffected(), nil
}

// ReconcileTerminalBindings closes the database feedback loop when a terminal
// outbox event exhausts its retry budget. It is intentionally independent of
// event delivery and therefore also repairs historical waiting bindings.
func (r *Reconciler) ReconcileTerminalBindings(ctx context.Context) (int64, error) {
	const query = `
WITH terminal AS (
  UPDATE parameter_sync_request_bindings b SET
    status=CASE run.status WHEN 'succeeded' THEN 'completed' WHEN 'cancelled' THEN 'cancelled' ELSE 'failed' END,
    completed_at=COALESCE(b.completed_at, $1)
  FROM parameter_sync_runs run
  WHERE b.run_id=run.id AND b.status='waiting'
    AND run.status IN ('succeeded','failed','cancelled')
  RETURNING b.provisioning_task_id, run.status, COALESCE(run.error_message, '') AS run_error
), provisioning AS (
  UPDATE provisioning_tasks p SET
    status=CASE WHEN terminal.status='succeeded' THEN 'completed' ELSE 'failed' END,
    error_message=CASE WHEN terminal.status='succeeded' THEN ''
      WHEN terminal.run_error<>'' THEN terminal.run_error ELSE 'parameter synchronization failed' END,
    completed_at=COALESCE(p.completed_at, $1), updated_at=$1
  FROM terminal WHERE p.id=terminal.provisioning_task_id
    AND p.status NOT IN ('completed','failed')
  RETURNING p.id
)
SELECT count(*) FROM terminal`
	var count int64
	if err := r.pool.QueryRow(ctx, query, r.now()).Scan(&count); err != nil {
		return 0, fmt.Errorf("reconcile terminal parameter sync bindings: %w", err)
	}
	if r.metrics != nil && count > 0 {
		r.metrics.ReconcileRepairs.WithLabelValues("terminal_bindings").Add(float64(count))
	}
	return count, nil
}

func (r *Reconciler) RepublishMissingResults(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = 100
	}
	const query = `
SELECT t.id, t.device_sn, t.status, t.error_code, COALESCE(t.error_message,''), t.source_id, t.creator_id
FROM device_tasks t
LEFT JOIN parameter_sync_task_results res ON res.run_id=t.source_id AND res.task_id=t.id
WHERE t.source='param_sync' AND t.status IN ('completed','failed','expired','cancelled')
  AND res.task_id IS NULL
ORDER BY t.completed_at ASC NULLS LAST LIMIT $1`
	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return 0, fmt.Errorf("find missing parameter sync results: %w", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		result, err := scanMissingTaskResult(rows)
		if err != nil {
			return count, fmt.Errorf("scan missing parameter sync result: %w", err)
		}
		payload := event.ParamSyncTaskResultPayload{
			EventID: result.taskID + ":reconcile:" + string(result.status), RequestID: result.requestID, RunID: result.runID,
			TaskID: result.taskID, DeviceSN: result.deviceSN, Success: result.status == task.TaskStatusCompleted,
			ResultRef: "device_tasks:" + result.taskID, ErrorCode: result.errorCode, ErrorMessage: result.errorMessage,
		}
		evt, err := event.NewEvent(event.SubjectParamSyncTaskResult, payload)
		if err != nil {
			return count, err
		}
		if err := r.bus.Publish(ctx, event.SubjectParamSyncTaskResult, evt); err != nil {
			return count, fmt.Errorf("republish missing parameter sync result: %w", err)
		}
		count++
	}
	if r.metrics != nil && count > 0 {
		r.metrics.ReconcileRepairs.WithLabelValues("missing_result").Add(float64(count))
	}
	return count, rows.Err()
}

func (r *Reconciler) CleanStaging(ctx context.Context, olderThan time.Time, limit int) (int64, error) {
	if limit <= 0 {
		limit = 10000
	}
	const query = `
DELETE FROM parameter_sync_staging_values s WHERE (s.run_id, s.parameter_path) IN (
  SELECT s2.run_id, s2.parameter_path FROM parameter_sync_staging_values s2
  JOIN parameter_sync_runs run ON run.id=s2.run_id
  WHERE (run.status IN ('failed','cancelled')
    OR (run.status='succeeded' AND COALESCE(run.completed_at, run.started_at) < $1))
  LIMIT $2
)`
	tag, err := r.pool.Exec(ctx, query, olderThan, limit)
	if err != nil {
		return 0, fmt.Errorf("clean parameter sync staging: %w", err)
	}
	return tag.RowsAffected(), nil
}

// CollectMetrics refreshes gauges from durable PostgreSQL state. Gauges are
// intentionally snapshots rather than in-memory increments so restarts,
// redelivery and reconciliation cannot make observability drift from reality.
func (r *Reconciler) CollectMetrics(ctx context.Context) error {
	if r.metrics == nil {
		return nil
	}
	for _, status := range []RunStatus{
		RunStatusPlanning, RunStatusEnqueuing, RunStatusWaitingDevice,
		RunStatusExecuting, RunStatusProcessing, RunStatusCancelling,
	} {
		r.metrics.RunsActive.WithLabelValues(string(status)).Set(0)
	}
	rows, err := r.pool.Query(ctx, `
SELECT status, count(*) FROM parameter_sync_runs
WHERE status IN ('planning','enqueuing','waiting_device','executing','processing','cancelling')
GROUP BY status`)
	if err != nil {
		return fmt.Errorf("collect active parameter sync runs: %w", err)
	}
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			rows.Close()
			return fmt.Errorf("scan active parameter sync runs: %w", err)
		}
		r.metrics.RunsActive.WithLabelValues(status).Set(float64(count))
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("iterate active parameter sync runs: %w", err)
	}
	rows.Close()

	var expected, terminal, processed, failed int64
	if err := r.pool.QueryRow(ctx, `
SELECT COALESCE(sum(expected_task_count),0), COALESCE(sum(terminal_task_count),0),
       COALESCE(sum(processed_task_count),0), COALESCE(sum(failed_task_count),0)
FROM parameter_sync_runs
WHERE status IN ('planning','enqueuing','waiting_device','executing','processing','cancelling')`).
		Scan(&expected, &terminal, &processed, &failed); err != nil {
		return fmt.Errorf("collect parameter sync task gauges: %w", err)
	}
	r.metrics.TasksExpected.Set(float64(expected))
	r.metrics.TasksTerminal.Set(float64(terminal))
	r.metrics.TasksProcessed.Set(float64(processed))
	r.metrics.TasksFailed.Set(float64(failed))

	var outbox, staging int64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM parameter_sync_outbox WHERE status IN ('pending','failed','delivering')`).Scan(&outbox); err != nil {
		return fmt.Errorf("collect parameter sync outbox backlog: %w", err)
	}
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM parameter_sync_staging_values`).Scan(&staging); err != nil {
		return fmt.Errorf("collect parameter sync staging rows: %w", err)
	}
	r.metrics.OutboxBacklog.Set(float64(outbox))
	r.metrics.StagingRows.Set(float64(staging))
	return nil
}
