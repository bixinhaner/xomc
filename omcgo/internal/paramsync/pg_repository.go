package paramsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type PGRepository struct {
	pool *pgxpool.Pool
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) CreateRequest(ctx context.Context, req *SyncRequest) error {
	query, args, err := buildCreateRequest(req)
	if err != nil {
		return err
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("create parameter sync request: %w", err)
	}
	return nil
}

func buildCreateRequest(req *SyncRequest) (string, []any, error) {
	if req.ID == uuid.Nil {
		req.ID = uuid.New()
	}
	now := time.Now().UTC()
	if req.CreatedAt.IsZero() {
		req.CreatedAt = now
	}
	if req.UpdatedAt.IsZero() {
		req.UpdatedAt = req.CreatedAt
	}
	if req.NextAttemptAt.IsZero() {
		req.NextAttemptAt = req.CreatedAt
	}
	if req.Status == "" {
		req.Status = RequestStatusAccepted
	}
	paths, err := json.Marshal(req.RequestedPaths)
	if err != nil {
		return "", nil, fmt.Errorf("marshal requested paths: %w", err)
	}
	query, args, err := storage.Psql.Insert("parameter_sync_requests").
		Columns("id", "device_id", "device_sn", "caller_type", "trigger_reason", "sync_scope",
			"requested_paths", "status", "priority", "next_attempt_at", "deadline_at", "idempotency_key",
			"result_code", "error_message", "campaign_id", "created_at", "completed_at", "updated_at").
		Values(req.ID, req.DeviceID, req.DeviceSN, req.CallerType, req.TriggerReason, req.SyncScope,
			paths, req.Status, req.Priority, req.NextAttemptAt, req.DeadlineAt, req.IdempotencyKey,
			req.ResultCode, req.ErrorMessage, req.CampaignID, req.CreatedAt, req.CompletedAt, req.UpdatedAt).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build create parameter sync request: %w", err)
	}
	return query, args, nil
}

func (r *PGRepository) GetRequest(ctx context.Context, id uuid.UUID) (*SyncRequest, error) {
	query, args, err := storage.Psql.Select(
		"id", "device_id", "device_sn", "caller_type", "trigger_reason", "sync_scope",
		"requested_paths", "status", "run_id", "active_run_id", "priority", "next_attempt_at",
		"deadline_at", "idempotency_key", "COALESCE(result_code, '')", "COALESCE(result_summary, 'null'::jsonb)", "COALESCE(error_message, '')",
		"campaign_id", "created_at", "started_at", "completed_at", "updated_at",
	).From("parameter_sync_requests").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get parameter sync request: %w", err)
	}
	var req SyncRequest
	var paths []byte
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&req.ID, &req.DeviceID, &req.DeviceSN, &req.CallerType, &req.TriggerReason, &req.SyncScope,
		&paths, &req.Status, &req.RunID, &req.ActiveRunID, &req.Priority, &req.NextAttemptAt,
		&req.DeadlineAt, &req.IdempotencyKey, &req.ResultCode, &req.ResultSummary, &req.ErrorMessage,
		&req.CampaignID, &req.CreatedAt, &req.StartedAt, &req.CompletedAt, &req.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get parameter sync request: %w", err)
	}
	if err := json.Unmarshal(paths, &req.RequestedPaths); err != nil {
		return nil, fmt.Errorf("decode requested paths: %w", err)
	}
	return &req, nil
}

func (r *PGRepository) FindRequestByIdempotency(ctx context.Context, callerType, key string) (*SyncRequest, error) {
	query, args, err := storage.Psql.Select("id").From("parameter_sync_requests").
		Where(sq.Eq{"caller_type": callerType, "idempotency_key": key}).Limit(1).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find parameter sync idempotency key: %w", err)
	}
	var id uuid.UUID
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&id); err != nil {
		return nil, fmt.Errorf("find parameter sync idempotency key: %w", err)
	}
	return r.GetRequest(ctx, id)
}

func (r *PGRepository) CompleteRequest(ctx context.Context, id uuid.UUID, status RequestStatus, code ResultCode, activeRunID *uuid.UUID, message string) error {
	now := time.Now().UTC()
	query, args, err := storage.Psql.Update("parameter_sync_requests").
		Set("status", status).Set("result_code", code).Set("active_run_id", activeRunID).
		Set("error_message", message).Set("completed_at", now).Set("updated_at", now).
		Where(sq.Eq{"id": id}).Where(sq.Eq{"status": []RequestStatus{RequestStatusAccepted, RequestStatusQueued, RequestStatusRunning}}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build complete parameter sync request: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("complete parameter sync request: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("complete parameter sync request: %w", ErrRequestStateConflict)
	}
	return nil
}

func (r *PGRepository) QueueRequest(ctx context.Context, id uuid.UUID, nextAttemptAt time.Time) error {
	query, args, err := storage.Psql.Update("parameter_sync_requests").
		Set("status", RequestStatusQueued).Set("result_code", ResultCodeActiveSyncExists).
		Set("active_run_id", nil).Set("next_attempt_at", nextAttemptAt).
		Set("error_message", "waiting for active parameter sync to finish").Set("updated_at", time.Now().UTC()).
		Where(sq.Eq{"id": id, "status": []RequestStatus{RequestStatusAccepted, RequestStatusQueued}}).
		Where(sq.Eq{"run_id": nil}).ToSql()
	if err != nil {
		return fmt.Errorf("build queue parameter sync request: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("queue parameter sync request: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("queue parameter sync request: %w", ErrRequestStateConflict)
	}
	return nil
}

func (r *PGRepository) ClaimQueuedRequests(ctx context.Context, now time.Time, limit int) ([]*SyncRequest, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, `
WITH candidates AS (
  SELECT id FROM parameter_sync_requests
  WHERE status='queued' AND run_id IS NULL AND next_attempt_at <= $1
  ORDER BY priority DESC, created_at
  LIMIT $2 FOR UPDATE SKIP LOCKED
), claimed AS (
  UPDATE parameter_sync_requests req SET next_attempt_at=$1 + interval '30 seconds', updated_at=$1
  FROM candidates WHERE req.id=candidates.id
  RETURNING req.id
)
SELECT id FROM claimed`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("claim queued parameter sync requests: %w", err)
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan queued parameter sync request: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate queued parameter sync requests: %w", err)
	}
	rows.Close()
	requests := make([]*SyncRequest, 0, len(ids))
	for _, id := range ids {
		req, err := r.GetRequest(ctx, id)
		if err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, nil
}

// FailRunAndRequest compensates a failure that occurs after the run was
// committed but before its task plan was durably dispatched. Keeping both
// updates in one transaction guarantees the active device-run gate is released
// together with the caller-visible request state.
func (r *PGRepository) FailRunAndRequest(ctx context.Context, runID, requestID uuid.UUID, code ResultCode, message string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin fail parameter sync run: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var currentRunStatus RunStatus
	if err := tx.QueryRow(ctx, `SELECT status FROM parameter_sync_runs WHERE id=$1 AND request_id=$2 FOR UPDATE`, runID, requestID).Scan(&currentRunStatus); err != nil {
		return fmt.Errorf("lock failed parameter sync run: %w", err)
	}
	if err := ValidateRunTransition(currentRunStatus, RunStatusFailed); err != nil {
		return err
	}

	now := time.Now().UTC()
	runQuery, runArgs, err := storage.Psql.Update("parameter_sync_runs").
		Set("status", RunStatusFailed).Set("error_message", message).Set("completed_at", now).
		Set("version", sq.Expr("version + 1")).
		Where(sq.Eq{"id": runID, "request_id": requestID}).
		Where(sq.Eq{"status": []RunStatus{RunStatusPlanning, RunStatusEnqueuing}}).ToSql()
	if err != nil {
		return fmt.Errorf("build fail parameter sync run: %w", err)
	}
	runTag, err := tx.Exec(ctx, runQuery, runArgs...)
	if err != nil {
		return fmt.Errorf("fail parameter sync run: %w", err)
	}
	if runTag.RowsAffected() != 1 {
		return fmt.Errorf("fail parameter sync run: run is no longer dispatchable")
	}

	reqQuery, reqArgs, err := storage.Psql.Update("parameter_sync_requests").
		Set("status", RequestStatusFailed).Set("result_code", code).Set("error_message", message).
		Set("active_run_id", nil).Set("completed_at", now).Set("updated_at", now).
		Where(sq.Eq{"id": requestID, "run_id": runID}).ToSql()
	if err != nil {
		return fmt.Errorf("build fail parameter sync request after dispatch: %w", err)
	}
	requestTag, err := tx.Exec(ctx, reqQuery, reqArgs...)
	if err != nil {
		return fmt.Errorf("fail parameter sync request after dispatch: %w", err)
	}
	if requestTag.RowsAffected() != 1 {
		return fmt.Errorf("fail parameter sync request after dispatch: request not found")
	}
	runSelect, runSelectArgs, err := storage.Psql.Select(runColumns()...).
		From("parameter_sync_runs").Where(sq.Eq{"id": runID}).ToSql()
	if err != nil {
		return fmt.Errorf("build load failed parameter sync run: %w", err)
	}
	failedRun, err := scanRun(tx.QueryRow(ctx, runSelect, runSelectArgs...))
	if err != nil {
		return fmt.Errorf("load failed parameter sync run: %w", err)
	}
	if err := insertRunTerminalOutbox(ctx, tx, failedRun, event.SubjectParamSyncRunFailed, now); err != nil {
		return err
	}
	if err := recordAutomaticSyncOutcome(ctx, tx, failedRun, false, message, now); err != nil {
		return err
	}
	if failedRun.SyncScope.IsFull() {
		if _, err := tx.Exec(ctx, `UPDATE devices SET last_param_sync_failed_at=$2, last_param_sync_error=$3 WHERE id=$1`, failedRun.DeviceID, now, message); err != nil {
			return fmt.Errorf("mark planning parameter sync failed on device: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit failed parameter sync dispatch: %w", err)
	}
	return nil
}

func (r *PGRepository) CreateOrDeduplicateRun(ctx context.Context, req *SyncRequest) (*StartResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create parameter sync run: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	run := &SyncRun{
		ID: reqRunID(req), RequestID: req.ID, DeviceID: req.DeviceID, DeviceSN: req.DeviceSN,
		TriggerReason: req.TriggerReason, SyncScope: req.SyncScope, Status: RunStatusPlanning,
		StartedAt: time.Now().UTC(),
	}
	query, args, err := storage.Psql.Insert("parameter_sync_runs").
		Columns("id", "request_id", "device_id", "device_sn", "trigger_reason", "sync_scope", "status", "started_at").
		Values(run.ID, run.RequestID, run.DeviceID, run.DeviceSN, run.TriggerReason, run.SyncScope, run.Status, run.StartedAt).
		Suffix("ON CONFLICT DO NOTHING").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create parameter sync run: %w", err)
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("create parameter sync run: %w", err)
	}
	if tag.RowsAffected() == 0 {
		active, findErr := findActiveRun(ctx, tx, req.DeviceID)
		if findErr != nil {
			return nil, findErr
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit deduplicated parameter sync run: %w", err)
		}
		return &StartResult{Request: req, Deduplicated: true, ActiveRunID: &active.ID, ActiveRun: active}, nil
	}
	update, updateArgs, err := storage.Psql.Update("parameter_sync_requests").
		Set("run_id", run.ID).Set("active_run_id", run.ID).Set("status", RequestStatusRunning).
		Set("started_at", run.StartedAt).Set("updated_at", run.StartedAt).
		Where(sq.Eq{"id": req.ID, "status": []RequestStatus{RequestStatusAccepted, RequestStatusQueued}, "run_id": nil}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build attach parameter sync run: %w", err)
	}
	requestTag, err := tx.Exec(ctx, update, updateArgs...)
	if err != nil {
		return nil, fmt.Errorf("attach parameter sync run: %w", err)
	}
	if requestTag.RowsAffected() != 1 {
		return nil, fmt.Errorf("attach parameter sync run: %w", ErrRequestStateConflict)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit parameter sync run: %w", err)
	}
	req.RunID, req.ActiveRunID, req.Status = &run.ID, &run.ID, RequestStatusRunning
	return &StartResult{Request: req, Run: run}, nil
}

func reqRunID(req *SyncRequest) uuid.UUID {
	if req.RunID != nil && *req.RunID != uuid.Nil {
		return *req.RunID
	}
	return uuid.New()
}

func findActiveRun(ctx context.Context, tx pgx.Tx, deviceID uuid.UUID) (*SyncRun, error) {
	query, args, err := storage.Psql.Select(runColumns()...).From("parameter_sync_runs").
		Where(sq.Eq{"device_id": deviceID, "status": []RunStatus{
			RunStatusPlanning, RunStatusEnqueuing, RunStatusWaitingDevice, RunStatusExecuting, RunStatusProcessing, RunStatusCancelling,
		}}).OrderBy("started_at DESC").Limit(1).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find active parameter sync run: %w", err)
	}
	return scanRun(tx.QueryRow(ctx, query, args...))
}

func (r *PGRepository) GetRun(ctx context.Context, id uuid.UUID) (*SyncRun, error) {
	query, args, err := storage.Psql.Select(runColumns()...).From("parameter_sync_runs").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get parameter sync run: %w", err)
	}
	return scanRun(r.pool.QueryRow(ctx, query, args...))
}

// CreateAutomaticRequest atomically reserves the per-device execution gate and
// persists the corresponding request. If either write fails, neither survives,
// so callers cannot leak accepted requests or consume a gate without a request.
func (r *PGRepository) CreateAutomaticRequest(ctx context.Context, req *SyncRequest, now time.Time) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin automatic parameter sync request: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	const reserveQuery = `
INSERT INTO parameter_sync_device_state (device_id, last_attempt_at, next_auto_sync_at, updated_at)
VALUES ($1, $2::timestamptz, $2::timestamptz + interval '1 minute', $2::timestamptz)
ON CONFLICT (device_id) DO UPDATE SET
  last_attempt_at=EXCLUDED.last_attempt_at,
  next_auto_sync_at=EXCLUDED.next_auto_sync_at,
  updated_at=EXCLUDED.updated_at
WHERE parameter_sync_device_state.next_auto_sync_at IS NULL
   OR parameter_sync_device_state.next_auto_sync_at <= EXCLUDED.last_attempt_at
RETURNING next_auto_sync_at`
	var next time.Time
	allowed := true
	err = tx.QueryRow(ctx, reserveQuery, req.DeviceID, now).Scan(&next)
	if errors.Is(err, pgx.ErrNoRows) {
		allowed = false
		if err := tx.QueryRow(ctx, "SELECT next_auto_sync_at FROM parameter_sync_device_state WHERE device_id=$1 FOR SHARE", req.DeviceID).Scan(&next); err != nil {
			return false, fmt.Errorf("load automatic parameter sync backoff: %w", err)
		}
	} else if err != nil {
		return false, fmt.Errorf("reserve automatic parameter sync: %w", err)
	}
	if !allowed {
		message := fmt.Sprintf("automatic parameter sync backed off until %s", next.Format(time.RFC3339))
		req.Status = RequestStatusRejected
		req.ResultCode = ResultCodeAutomaticBackoff
		req.ErrorMessage = message
		req.CompletedAt = &now
	}
	query, args, err := buildCreateRequest(req)
	if err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return false, fmt.Errorf("create automatic parameter sync request: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit automatic parameter sync request: %w", err)
	}
	return allowed, nil
}

func (r *PGRepository) GetActiveRunByDevice(ctx context.Context, deviceID uuid.UUID) (*SyncRun, error) {
	query, args, err := storage.Psql.Select(runColumns()...).From("parameter_sync_runs").
		Where(sq.Eq{"device_id": deviceID, "status": []RunStatus{
			RunStatusPlanning, RunStatusEnqueuing, RunStatusWaitingDevice, RunStatusExecuting, RunStatusProcessing, RunStatusCancelling,
		}}).OrderBy("started_at DESC").Limit(1).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get active parameter sync run: %w", err)
	}
	run, err := scanRun(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, fmt.Errorf("get active parameter sync run: %w", err)
	}
	return run, nil
}

func (r *PGRepository) ListRequestsByDevice(ctx context.Context, deviceID uuid.UUID, limit int) ([]*SyncRequest, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	query, args, err := storage.Psql.Select("id").From("parameter_sync_requests").
		Where(sq.Eq{"device_id": deviceID}).OrderBy("created_at DESC").Limit(uint64(limit)).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list parameter sync request history: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list parameter sync request history: %w", err)
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan parameter sync request history id: %w", err)
		}
		ids = append(ids, id)
	}
	requests := make([]*SyncRequest, 0, len(ids))
	for _, id := range ids {
		req, err := r.GetRequest(ctx, id)
		if err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, nil
}

func runColumns() []string {
	return []string{"id", "request_id", "device_id", "device_sn", "trigger_reason", "sync_scope",
		"COALESCE(mapping_source, '')", "COALESCE(mapping_version, '')", "coverage", "status", "expected_task_count", "terminal_task_count",
		"processed_task_count", "failed_task_count", "COALESCE(error_message, '')", "started_at", "completed_at", "version"}
}

func scanRun(row pgx.Row) (*SyncRun, error) {
	var run SyncRun
	var coverage []byte
	if err := row.Scan(&run.ID, &run.RequestID, &run.DeviceID, &run.DeviceSN, &run.TriggerReason, &run.SyncScope,
		&run.MappingSource, &run.MappingVersion, &coverage, &run.Status, &run.ExpectedTaskCount, &run.TerminalTaskCount,
		&run.ProcessedTaskCount, &run.FailedTaskCount, &run.ErrorMessage, &run.StartedAt, &run.CompletedAt, &run.Version); err != nil {
		return nil, fmt.Errorf("scan parameter sync run: %w", err)
	}
	if err := json.Unmarshal(coverage, &run.Coverage); err != nil {
		return nil, fmt.Errorf("decode parameter sync run coverage: %w", err)
	}
	return &run, nil
}

func (r *PGRepository) InsertTaskResultIfAbsent(ctx context.Context, result *TaskResult) (bool, error) {
	if result.CreatedAt.IsZero() {
		result.CreatedAt = time.Now().UTC()
	}
	if result.Status == "" {
		result.Status = "received"
	}
	query, args, err := storage.Psql.Insert("parameter_sync_task_results").
		Columns("run_id", "task_id", "event_id", "success", "result_ref", "status", "error_code", "error_message", "created_at").
		Values(result.RunID, result.TaskID, result.EventID, result.Success, result.ResultRef, result.Status,
			result.ErrorCode, result.ErrorMessage, result.CreatedAt).
		Suffix("ON CONFLICT DO NOTHING").ToSql()
	if err != nil {
		return false, fmt.Errorf("build insert parameter sync task result: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("insert parameter sync task result: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *PGRepository) ClaimRunForFinalize(ctx context.Context, runID uuid.UUID) (bool, error) {
	query, args, err := storage.Psql.Update("parameter_sync_runs").
		Set("status", RunStatusProcessing).Set("version", sq.Expr("version + 1")).
		Where(sq.Eq{"id": runID, "status": []RunStatus{RunStatusWaitingDevice, RunStatusExecuting}}).
		Where("terminal_task_count = expected_task_count").
		Where("processed_task_count = expected_task_count").
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build claim parameter sync run: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("claim parameter sync run: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func IsNotFound(err error) bool { return errors.Is(err, pgx.ErrNoRows) }
