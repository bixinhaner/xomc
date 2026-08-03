package paramsync

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/task"
)

type Reconciler struct {
	pool      *pgxpool.Pool
	bus       event.EventBus
	processor ResultProcessor
	metrics   *Metrics
	now       func() time.Time

	runCountMu       sync.Mutex
	runCountCursor   *runConvergenceCursor
	runCountSweepEnd *runConvergenceCursor

	stagingCleanupMu     sync.Mutex
	stagingCleanupCursor *stagingCursor
}

// Each candidate is reconciled under one outer transaction so that its row
// lock and authoritative task/result counts stay consistent. Keep the page
// small enough to commit within the maintenance step deadline under release
// campaign load; a timed-out 200-row transaction rolled the entire page back
// and retried it forever.
const runCountReconcileBatchSize = 50
const runCountReconcilePrioritySize = 10

type runConvergenceCursor struct {
	StartedAt time.Time
	ID        uuid.UUID
}

type runConvergenceCandidate struct {
	id             uuid.UUID
	startedAt      time.Time
	advancesCursor bool
}

type stagingCursor struct {
	runID         uuid.UUID
	parameterPath string
}

const historicalResultNormalizationSQL = `
UPDATE parameter_sync_task_results res SET
  status=CASE WHEN res.success THEN 'processed' ELSE 'failed' END,
  processed_at=COALESCE(res.processed_at, $1)
FROM parameter_sync_runs run, device_tasks t
WHERE res.status='received' AND run.id=res.run_id AND run.id=ANY($2)
  AND run.status IN ('succeeded','failed','cancelled')
  AND t.id=res.task_id AND t.source='param_sync' AND t.source_id=run.id
  AND t.status IN ('completed','failed','expired','cancelled')`

const stagingCleanupSQL = `
WITH page AS MATERIALIZED (
  SELECT run_id, parameter_path
  FROM parameter_sync_staging_values
  WHERE (run_id, parameter_path) > ($1, $2)
  ORDER BY run_id, parameter_path
  LIMIT $3
), deleted AS (
  DELETE FROM parameter_sync_staging_values s USING page, parameter_sync_runs run
  WHERE s.run_id=page.run_id AND s.parameter_path=page.parameter_path
    AND run.id=page.run_id
    AND (run.status IN ('failed','cancelled')
      OR (run.status='succeeded' AND COALESCE(run.completed_at, run.started_at) < $4))
  RETURNING 1
), tail AS (
  SELECT run_id, parameter_path FROM page ORDER BY run_id DESC, parameter_path DESC LIMIT 1
)
SELECT tail.run_id, tail.parameter_path, (SELECT count(*) FROM deleted)
FROM tail`

const stagingCleanupInitialSQL = `
WITH page AS MATERIALIZED (
  SELECT run_id, parameter_path
  FROM parameter_sync_staging_values
  ORDER BY run_id, parameter_path
  LIMIT $1
), deleted AS (
  DELETE FROM parameter_sync_staging_values s USING page, parameter_sync_runs run
  WHERE s.run_id=page.run_id AND s.parameter_path=page.parameter_path
    AND run.id=page.run_id
    AND (run.status IN ('failed','cancelled')
      OR (run.status='succeeded' AND COALESCE(run.completed_at, run.started_at) < $2))
  RETURNING 1
), tail AS (
  SELECT run_id, parameter_path FROM page ORDER BY run_id DESC, parameter_path DESC LIMIT 1
)
SELECT tail.run_id, tail.parameter_path, (SELECT count(*) FROM deleted)
FROM tail`

const outboxBacklogMetricSQL = `
SELECT
  (SELECT count(*) FROM parameter_sync_outbox WHERE status IN ('pending','failed')) +
  (SELECT count(*) FROM parameter_sync_outbox WHERE status='delivering')`

const stagingRowsMetricSQL = `
SELECT GREATEST(reltuples, 0)::bigint
FROM pg_class WHERE oid='parameter_sync_staging_values'::regclass`

const missingResultRunCandidatesSQL = `
SELECT run.id
FROM parameter_sync_runs run
WHERE run.status IN ('waiting_device','executing','processing','cancelling')
  AND EXISTS (
    SELECT 1
    FROM device_tasks t
    LEFT JOIN parameter_sync_task_results res
      ON res.run_id=run.id AND res.task_id=t.id
    WHERE t.source='param_sync' AND t.source_id=run.id
      AND t.status IN ('completed','failed','expired','cancelled')
      AND res.task_id IS NULL
  )
ORDER BY run.started_at ASC
LIMIT $1`

const convergenceMetricsSQL = `
WITH active AS (
  SELECT id, status, started_at, expected_task_count, terminal_task_count,
         processed_task_count, failed_task_count
  FROM parameter_sync_runs
  WHERE status IN ('planning','enqueuing','waiting_device','executing','processing','cancelling')
), counts AS (
  SELECT a.id, a.status, a.started_at,
         a.expected_task_count AS stored_expected,
         a.terminal_task_count AS stored_terminal,
         a.processed_task_count AS stored_processed,
         a.failed_task_count AS stored_failed,
         count(t.id)::bigint AS actual_expected,
         count(t.id) FILTER (
           WHERE t.status IN ('completed','failed','expired','cancelled')
         )::bigint AS actual_terminal,
         count(res.task_id) FILTER (
           WHERE t.status IN ('completed','failed','expired','cancelled')
             AND res.status IN ('processed','failed')
         )::bigint AS actual_processed,
         count(res.task_id) FILTER (
           WHERE t.status IN ('completed','failed','expired','cancelled')
             AND res.status IN ('processed','failed')
             AND (res.status='failed' OR NOT res.success)
         )::bigint AS actual_failed,
         COALESCE(
           max(GREATEST(t.created_at, t.sent_at, t.completed_at,
                        res.created_at, res.processed_at)),
           a.started_at
         ) AS last_progress_at
  FROM active a
  LEFT JOIN device_tasks t
    ON t.source='param_sync' AND t.source_id=a.id
  LEFT JOIN parameter_sync_task_results res
    ON res.run_id=a.id AND res.task_id=t.id
  GROUP BY a.id, a.status, a.started_at, a.expected_task_count,
           a.terminal_task_count, a.processed_task_count, a.failed_task_count
), classified AS (
  SELECT *,
		 status<>'cancelling'
		   AND (stored_expected>actual_expected
		     OR (stored_expected=0 AND actual_expected=0)) AS plan_blocked
  FROM counts
)
SELECT
  count(*) FILTER (
    WHERE actual_expected>0
      AND actual_expected=actual_terminal
      AND actual_expected=actual_processed
  )::bigint,
  count(*) FILTER (
    WHERE stored_expected<>actual_expected
       OR stored_terminal<>actual_terminal
       OR stored_processed<>actual_processed
       OR stored_failed<>actual_failed
  )::bigint,
  COALESCE(max(GREATEST(EXTRACT(EPOCH FROM (now()-last_progress_at)),0)) FILTER (
    WHERE stored_expected<>actual_expected
       OR stored_terminal<>actual_terminal
       OR stored_processed<>actual_processed
       OR stored_failed<>actual_failed
  ),0)::double precision,
  COALESCE(max(EXTRACT(EPOCH FROM (now()-started_at))),0)::double precision,
  count(*) FILTER (WHERE plan_blocked)::bigint,
  COALESCE(max(GREATEST(EXTRACT(EPOCH FROM (now()-last_progress_at)),0)) FILTER (
    WHERE plan_blocked
  ),0)::double precision,
  count(*) FILTER (
    WHERE NOT plan_blocked AND actual_terminal>actual_processed
  )::bigint,
  COALESCE(max(GREATEST(EXTRACT(EPOCH FROM (now()-last_progress_at)),0)) FILTER (
    WHERE NOT plan_blocked AND actual_terminal>actual_processed
  ),0)::double precision,
  count(*) FILTER (
    WHERE NOT plan_blocked
      AND actual_terminal=actual_processed
      AND actual_terminal<actual_expected
  )::bigint,
  COALESCE(max(GREATEST(EXTRACT(EPOCH FROM (now()-last_progress_at)),0)) FILTER (
    WHERE NOT plan_blocked
      AND actual_terminal=actual_processed
      AND actual_terminal<actual_expected
  ),0)::double precision
FROM classified`

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

func (r *Reconciler) WithResultProcessor(processor ResultProcessor) *Reconciler {
	r.processor = processor
	return r
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
  WHERE run.trigger_reason IN (
    'device_online','periodic','firmware_changed','device_registered','omc_upgrade'
  )
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
	r.runCountMu.Lock()
	defer r.runCountMu.Unlock()
	started := time.Now()
	if r.metrics != nil {
		defer func() { r.metrics.ReconcileDuration.Observe(time.Since(started).Seconds()) }()
	}

	claimTx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin parameter sync run convergence claim: %w", err)
	}
	defer func() { _ = claimTx.Rollback(context.Background()) }()
	sweepEnd := r.runCountSweepEnd
	if sweepEnd == nil {
		sweepQuery, sweepArgs, buildErr := buildRunConvergenceSweepEndSQL()
		if buildErr != nil {
			return 0, fmt.Errorf("build parameter sync run convergence sweep end: %w", buildErr)
		}
		var loaded runConvergenceCursor
		if scanErr := claimTx.QueryRow(ctx, sweepQuery, sweepArgs...).Scan(&loaded.StartedAt, &loaded.ID); scanErr != nil {
			if errors.Is(scanErr, pgx.ErrNoRows) {
				if commitErr := claimTx.Commit(ctx); commitErr != nil {
					return 0, fmt.Errorf("commit empty parameter sync run convergence sweep: %w", commitErr)
				}
				r.runCountCursor = nil
				r.runCountSweepEnd = nil
				return 0, nil
			}
			return 0, fmt.Errorf("load parameter sync run convergence sweep end: %w", scanErr)
		}
		sweepEnd = &loaded
	}

	priorityQuery, priorityArgs, err := buildRunConvergencePrioritySelectSQL(runCountReconcilePrioritySize)
	if err != nil {
		return 0, fmt.Errorf("build priority parameter sync run convergence candidates: %w", err)
	}
	rows, err := claimTx.Query(ctx, priorityQuery, priorityArgs...)
	if err != nil {
		return 0, fmt.Errorf("list priority parameter sync run convergence candidates: %w", err)
	}
	candidates := make([]runConvergenceCandidate, 0, runCountReconcileBatchSize)
	excluded := make([]uuid.UUID, 0, runCountReconcilePrioritySize)
	for rows.Next() {
		var item runConvergenceCandidate
		if err := rows.Scan(&item.id, &item.startedAt); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan priority parameter sync run convergence candidate: %w", err)
		}
		candidates = append(candidates, item)
		excluded = append(excluded, item.id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate priority parameter sync run convergence candidates: %w", err)
	}
	rows.Close()

	forwardQuery, forwardArgs, err := buildRunConvergenceForwardSelectSQL(
		r.runCountCursor, sweepEnd, excluded,
		runCountReconcileBatchSize-runCountReconcilePrioritySize,
	)
	if err != nil {
		return 0, fmt.Errorf("build forward parameter sync run convergence candidates: %w", err)
	}
	rows, err = claimTx.Query(ctx, forwardQuery, forwardArgs...)
	if err != nil {
		return 0, fmt.Errorf("list forward parameter sync run convergence candidates: %w", err)
	}
	for rows.Next() {
		item := runConvergenceCandidate{advancesCursor: true}
		if err := rows.Scan(&item.id, &item.startedAt); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan forward parameter sync run convergence candidate: %w", err)
		}
		candidates = append(candidates, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate forward parameter sync run convergence candidates: %w", err)
	}
	rows.Close()
	if len(candidates) == 0 {
		if err := claimTx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit empty parameter sync run convergence claim: %w", err)
		}
		// The next maintenance tick starts a new bounded sweep. Avoid wrapping
		// inside this call, which would process the first page twice at the tail.
		r.runCountCursor = nil
		r.runCountSweepEnd = nil
		return 0, nil
	}

	var finalized int64
	var drifted int64
	blocked := map[convergenceBlockReason]int64{}
	var reconcileErr error
	for _, item := range candidates {
		// A nested pgx transaction is a savepoint. The outer transaction keeps
		// every candidate row locked until all convergence writes commit, while
		// a single broken run can roll back to its savepoint without aborting the
		// rest of the bounded batch.
		tx, beginErr := claimTx.Begin(ctx)
		if beginErr != nil {
			reconcileErr = errors.Join(reconcileErr,
				fmt.Errorf("begin parameter sync run convergence %s: %w", item.id, beginErr))
			continue
		}
		result, convergeErr := convergeRunTx(ctx, tx, item.id, r.now())
		if convergeErr != nil {
			_ = tx.Rollback(context.Background())
			reconcileErr = errors.Join(reconcileErr,
				fmt.Errorf("converge parameter sync run %s: %w", item.id, convergeErr))
			continue
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			reconcileErr = errors.Join(reconcileErr,
				fmt.Errorf("commit parameter sync run convergence %s: %w", item.id, commitErr))
			continue
		}
		if result.Finalized {
			finalized++
			if r.metrics != nil {
				r.metrics.ReconcileRepairs.WithLabelValues("ready_runs").Inc()
				status := "succeeded"
				if result.Failed {
					status = "failed"
				}
				r.metrics.ReconcileFinalized.WithLabelValues(status).Inc()
			}
		}
		if result.Drift {
			drifted++
			if r.metrics != nil {
				r.metrics.ReconcileRepairs.WithLabelValues("run_counts").Inc()
			}
		}
		if result.BlockedReason != "" {
			blocked[result.BlockedReason]++
		}
	}
	if err := claimTx.Commit(ctx); err != nil {
		return 0, errors.Join(reconcileErr,
			fmt.Errorf("commit parameter sync run convergence batch: %w", err))
	}
	if cursor, hasForward := nextRunConvergenceCursor(candidates); hasForward {
		r.runCountCursor = &cursor
		r.runCountSweepEnd = sweepEnd
	} else {
		r.runCountCursor = nil
		r.runCountSweepEnd = nil
	}
	return finalized, reconcileErr
}

func buildRunCountCandidateSelectSQL(cursor *uuid.UUID, limit uint64) (string, []interface{}, error) {
	if limit == 0 {
		limit = runCountReconcileBatchSize
	}
	builder := storage.Psql.Select("id").From("parameter_sync_runs").OrderBy("id").Limit(limit)
	if cursor != nil {
		builder = builder.Where(sq.Gt{"id": *cursor})
	}
	return builder.ToSql()
}

const runConvergenceActivePredicate = "run.status IN ('planning','enqueuing','waiting_device','executing','processing','cancelling')"
const runConvergenceStoredReadyPredicate = `(run.expected_task_count>0
  AND run.terminal_task_count=run.expected_task_count
  AND run.processed_task_count=run.expected_task_count)`

func buildRunConvergenceSweepEndSQL() (string, []interface{}, error) {
	return storage.Psql.Select("run.started_at", "run.id").
		From("parameter_sync_runs run").
		Where(sq.Expr(runConvergenceActivePredicate)).
		OrderBy("run.started_at DESC", "run.id DESC").
		Limit(1).
		ToSql()
}

func buildRunConvergencePrioritySelectSQL(limit uint64) (string, []interface{}, error) {
	if limit == 0 {
		limit = runCountReconcilePrioritySize
	}
	return storage.Psql.Select("run.id", "run.started_at").
		From("parameter_sync_runs run").
		Where(sq.Expr(runConvergenceActivePredicate)).
		Where(sq.Expr(runConvergenceStoredReadyPredicate)).
		OrderBy("run.started_at", "run.id").
		Limit(limit).
		Suffix("FOR UPDATE SKIP LOCKED").
		ToSql()
}

func buildRunConvergenceForwardSelectSQL(
	cursor, sweepEnd *runConvergenceCursor,
	excluded []uuid.UUID,
	limit uint64,
) (string, []interface{}, error) {
	if sweepEnd == nil {
		return "", nil, fmt.Errorf("parameter sync convergence sweep end is required")
	}
	if limit == 0 {
		limit = runCountReconcileBatchSize - runCountReconcilePrioritySize
	}
	builder := storage.Psql.Select("run.id", "run.started_at").
		From("parameter_sync_runs run").
		Where(sq.Expr(runConvergenceActivePredicate))
	if cursor != nil {
		builder = builder.Where(sq.Expr("(run.started_at, run.id) > (?, ?)", cursor.StartedAt, cursor.ID))
	}
	builder = builder.Where(sq.Expr("(run.started_at, run.id) <= (?, ?)", sweepEnd.StartedAt, sweepEnd.ID))
	if len(excluded) > 0 {
		builder = builder.Where(sq.Expr("run.id <> ALL(?::uuid[])", excluded))
	}
	return builder.OrderBy("run.started_at", "run.id").
		Limit(limit).
		Suffix("FOR UPDATE SKIP LOCKED").
		ToSql()
}

func nextRunConvergenceCursor(candidates []runConvergenceCandidate) (runConvergenceCursor, bool) {
	var cursor runConvergenceCursor
	found := false
	for _, candidate := range candidates {
		if !candidate.advancesCursor {
			continue
		}
		if !found || candidate.startedAt.After(cursor.StartedAt) ||
			(candidate.startedAt.Equal(cursor.StartedAt) && candidate.id.String() > cursor.ID.String()) {
			cursor = runConvergenceCursor{StartedAt: candidate.startedAt, ID: candidate.id}
			found = true
		}
	}
	return cursor, found
}

func buildRunCountReconciliationSQL(runIDs []uuid.UUID) (string, []interface{}, error) {
	if len(runIDs) == 0 {
		return "", nil, fmt.Errorf("parameter sync run count candidate batch is empty")
	}
	const query = `
WITH batch AS (
  SELECT unnest($1::uuid[]) AS id
), task_rows AS MATERIALIZED (
  SELECT id, source_id AS run_id, status
  FROM device_tasks
  WHERE source='param_sync' AND source_id=ANY($1)
), task_actual AS (
  SELECT run_id,
    count(*)::int AS expected,
    count(*) FILTER (WHERE status IN ('completed','failed','expired','cancelled'))::int AS terminal
  FROM task_rows
  GROUP BY run_id
), result_actual AS (
  SELECT t.run_id,
    count(res.task_id) FILTER (
      WHERE t.status IN ('completed','failed','expired','cancelled')
        AND res.status IN ('processed','failed')
    )::int AS processed,
    count(res.task_id) FILTER (
      WHERE t.status IN ('completed','failed','expired','cancelled')
        AND res.status IN ('processed','failed')
        AND (res.status='failed' OR NOT res.success)
    )::int AS failed
  FROM task_rows t
  JOIN parameter_sync_task_results res ON res.run_id=t.run_id AND res.task_id=t.id
  GROUP BY t.run_id
), actual AS (
  SELECT batch.id,
    COALESCE(t.expected, 0) AS expected,
    COALESCE(t.terminal, 0) AS terminal,
    COALESCE(res.processed, 0) AS processed,
    COALESCE(res.failed, 0) AS failed
  FROM batch
  LEFT JOIN task_actual t ON t.run_id=batch.id
  LEFT JOIN result_actual res ON res.run_id=batch.id
)
UPDATE parameter_sync_runs run SET
  expected_task_count=GREATEST(run.expected_task_count, actual.expected), terminal_task_count=actual.terminal,
  processed_task_count=actual.processed, failed_task_count=actual.failed, version=version+1
FROM actual WHERE run.id=actual.id AND (
  run.expected_task_count<>actual.expected OR run.terminal_task_count<>actual.terminal OR
  run.processed_task_count<>actual.processed OR run.failed_task_count<>actual.failed
)`
	return query, []interface{}{runIDs}, nil
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

func (r *Reconciler) RecoverMissingResults(ctx context.Context, runLimit, taskLimit, taskBudget int) (int, error) {
	if r.processor == nil {
		if r.bus == nil {
			return 0, fmt.Errorf("recover missing parameter sync results requires result processor or event bus")
		}
		return r.RepublishMissingResults(ctx, taskLimit)
	}
	if runLimit <= 0 {
		runLimit = 20
	}
	if taskLimit <= 0 {
		taskLimit = 200
	}
	if taskBudget <= 0 || taskBudget > runLimit*taskLimit {
		taskBudget = runLimit * taskLimit
	}
	// Select from durable terminal task state directly. Stored run counters are
	// repaired independently and may themselves be stale after an app restart;
	// making recovery depend on those counters creates a circular wait where the
	// missing result cannot be rebuilt until a separate count sweep reaches it.
	rows, err := r.pool.Query(ctx, missingResultRunCandidatesSQL, runLimit)
	if err != nil {
		return 0, fmt.Errorf("find stalled parameter sync runs with missing results: %w", err)
	}
	var runIDs []uuid.UUID
	for rows.Next() {
		var runID uuid.UUID
		if err := rows.Scan(&runID); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan stalled parameter sync run: %w", err)
		}
		runIDs = append(runIDs, runID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate stalled parameter sync runs: %w", err)
	}
	rows.Close()

	recovered := 0
	examined := 0
	var recoveryErrs []error
	for _, runID := range runIDs {
		if examined >= taskBudget {
			break
		}
		remaining := taskBudget - examined
		limit := taskLimit
		if remaining < limit {
			limit = remaining
		}
		n, attempted, err := r.recoverRunMissingResults(ctx, runID, limit)
		recovered += n
		examined += attempted
		if err != nil {
			recoveryErrs = append(recoveryErrs, err)
			if ctx.Err() != nil {
				break
			}
		}
	}
	if r.metrics != nil && recovered > 0 {
		r.metrics.ReconcileRepairs.WithLabelValues("missing_result_recovery").Add(float64(recovered))
	}
	return recovered, errors.Join(recoveryErrs...)
}

func (r *Reconciler) recoverRunMissingResults(ctx context.Context, runID uuid.UUID, limit int) (int, int, error) {
	const query = `
SELECT t.id, t.device_sn, t.status, t.error_code, COALESCE(t.error_message,''), t.source_id, t.creator_id
FROM device_tasks t
LEFT JOIN parameter_sync_task_results res ON res.run_id=t.source_id AND res.task_id=t.id
WHERE t.source='param_sync' AND t.source_id=$1
  AND t.status IN ('completed','failed','expired','cancelled')
  AND res.task_id IS NULL
ORDER BY t.command_index ASC, t.created_at ASC, t.id ASC
LIMIT $2`
	rows, err := r.pool.Query(ctx, query, runID, limit)
	if err != nil {
		return 0, 0, fmt.Errorf("find missing parameter sync results for run %s: %w", runID, err)
	}
	defer rows.Close()

	var results []missingTaskResult
	for rows.Next() {
		result, err := scanMissingTaskResult(rows)
		if err != nil {
			return 0, len(results), fmt.Errorf("scan missing parameter sync result for run %s: %w", runID, err)
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return 0, len(results), fmt.Errorf("iterate missing parameter sync results for run %s: %w", runID, err)
	}
	processed, err := processMissingTaskResults(ctx, r.processor, results)
	return processed, len(results), err
}

func processMissingTaskResults(ctx context.Context, processor ResultProcessor, results []missingTaskResult) (int, error) {
	processed := 0
	var processingErrs []error
	for _, result := range results {
		payload := event.ParamSyncTaskResultPayload{
			EventID:   "reconcile:" + result.runID.String() + ":" + result.taskID + ":" + string(result.status),
			RequestID: result.requestID, RunID: result.runID,
			TaskID: result.taskID, DeviceSN: result.deviceSN, Success: result.status == task.TaskStatusCompleted,
			ResultRef: "device_tasks:" + result.taskID, ErrorCode: result.errorCode, ErrorMessage: result.errorMessage,
		}
		outcome, err := processor.Process(ctx, payload)
		if err != nil {
			processingErrs = append(processingErrs, fmt.Errorf("recover missing parameter sync result %s for run %s: %w", result.taskID, result.runID, err))
			if ctx.Err() != nil {
				break
			}
			continue
		}
		if !outcome.Duplicate {
			processed++
		}
	}
	return processed, errors.Join(processingErrs...)
}

func (r *Reconciler) CleanStaging(ctx context.Context, olderThan time.Time, limit int) (int64, error) {
	r.stagingCleanupMu.Lock()
	defer r.stagingCleanupMu.Unlock()

	if limit <= 0 {
		limit = 10000
	}
	var runID uuid.UUID
	var parameterPath string
	initialPage := r.stagingCleanupCursor == nil
	if !initialPage {
		runID = r.stagingCleanupCursor.runID
		parameterPath = r.stagingCleanupCursor.parameterPath
	}
	var deleted int64
	var row pgx.Row
	if initialPage {
		row = r.pool.QueryRow(ctx, stagingCleanupInitialSQL, limit, olderThan)
	} else {
		row = r.pool.QueryRow(ctx, stagingCleanupSQL, runID, parameterPath, limit, olderThan)
	}
	err := row.Scan(&runID, &parameterPath, &deleted)
	if err == pgx.ErrNoRows {
		// Start a new bounded sweep on the next maintenance tick. Newly inserted
		// or newly eligible rows behind the cursor are picked up in that cycle.
		r.stagingCleanupCursor = nil
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("clean parameter sync staging: %w", err)
	}
	r.stagingCleanupCursor = &stagingCursor{runID: runID, parameterPath: parameterPath}
	return deleted, nil
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

	var ready, drift, planBlocked, resultMissing, deviceActive int64
	var driftOldestIdle, oldestAge, planOldestIdle, resultOldestIdle, deviceOldestIdle float64
	if err := r.pool.QueryRow(ctx, convergenceMetricsSQL).Scan(
		&ready, &drift, &driftOldestIdle, &oldestAge,
		&planBlocked, &planOldestIdle,
		&resultMissing, &resultOldestIdle,
		&deviceActive, &deviceOldestIdle,
	); err != nil {
		return fmt.Errorf("collect parameter sync convergence gauges: %w", err)
	}
	r.metrics.RunsReadyButNotFinalized.Set(float64(ready))
	r.metrics.RunCounterDrift.Set(float64(drift))
	r.metrics.RunCounterDriftOldestIdle.Set(driftOldestIdle)
	r.metrics.ActiveRunOldestAge.Set(oldestAge)
	r.metrics.RunsBlocked.WithLabelValues(string(convergenceBlockPlanNotDispatched)).Set(float64(planBlocked))
	r.metrics.RunsBlockedOldestIdle.WithLabelValues(string(convergenceBlockPlanNotDispatched)).Set(planOldestIdle)
	r.metrics.RunsBlocked.WithLabelValues(string(convergenceBlockTerminalResultMissing)).Set(float64(resultMissing))
	r.metrics.RunsBlockedOldestIdle.WithLabelValues(string(convergenceBlockTerminalResultMissing)).Set(resultOldestIdle)
	r.metrics.RunsBlocked.WithLabelValues(string(convergenceBlockDeviceTaskActive)).Set(float64(deviceActive))
	r.metrics.RunsBlockedOldestIdle.WithLabelValues(string(convergenceBlockDeviceTaskActive)).Set(deviceOldestIdle)

	var outbox, staging int64
	if err := r.pool.QueryRow(ctx, outboxBacklogMetricSQL).Scan(&outbox); err != nil {
		return fmt.Errorf("collect parameter sync outbox backlog: %w", err)
	}
	// Staging cardinality is an operational gauge, not a correctness boundary.
	// pg_class is refreshed by autovacuum/ANALYZE and avoids reading millions of
	// hot staging rows every 30 seconds solely for observability.
	if err := r.pool.QueryRow(ctx, stagingRowsMetricSQL).Scan(&staging); err != nil {
		return fmt.Errorf("collect parameter sync staging rows: %w", err)
	}
	r.metrics.OutboxBacklog.Set(float64(outbox))
	r.metrics.StagingRows.Set(float64(staging))
	return nil
}
