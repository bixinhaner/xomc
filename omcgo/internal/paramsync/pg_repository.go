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
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type PGRepository struct {
	pool *pgxpool.Pool
}

var (
	_ automaticAdmissionRepository      = (*PGRepository)(nil)
	_ automaticAdmissionTaskRepository  = (*PGRepository)(nil)
	_ automaticAdmissionQueueRepository = (*PGRepository)(nil)
	_ runValueReader                    = (*PGRepository)(nil)
)

type AutomaticAdmissionStats struct {
	ReservedRuns         int
	QueuedRequests       int
	OldestQueueAgeSecond float64
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

const (
	automaticAdmissionClass                 = "global"
	automaticAdmissionBucketCount           = 64
	automaticAdmissionRunsPerBucket         = 32
	automaticAdmissionTasksPerBucket        = 864
	automaticAdmissionLease                 = 2 * time.Hour
	automaticAdmissionRetry                 = 30 * time.Second
	automaticAdmissionMaintenanceLock int64 = 0x504152414d53594e
	releaseCampaignCategoryPrefix           = "param_sync_release:"
)

func automaticAdmissionBucket(deviceID uuid.UUID) int16 {
	value := uint16(deviceID[0])<<8 | uint16(deviceID[1])
	return int16(value % automaticAdmissionBucketCount)
}

func releaseCampaignCategory(campaignID uuid.UUID) string {
	return releaseCampaignCategoryPrefix + campaignID.String()
}

func listReleaseCandidatesQuery(campaignID uuid.UUID, limit int) (string, []any, error) {
	if limit <= 0 {
		limit = 200
	}
	category := releaseCampaignCategory(campaignID)
	// Nested Sqlizers must keep question-mark placeholders until the outer
	// Dollar-format builder performs one global numbering pass.
	completedOrActive := sq.StatementBuilder.PlaceholderFormat(sq.Question).
		Select("1").
		From("parameter_sync_requests req").
		Where("req.device_id = d.id").
		Where(sq.Eq{
			"req.campaign_id":    campaignID,
			"req.trigger_reason": TriggerOMCUpgrade,
			"req.status": []RequestStatus{
				RequestStatusAccepted,
				RequestStatusQueued,
				RequestStatusRunning,
				RequestStatusSucceeded,
			},
		})
	return storage.Psql.
		Select("d.id", "d.serial_number").
		Prefix(`WITH inserted_campaign AS (
  INSERT INTO config_apply_versions (category, config_version, updated_at)
  VALUES (?, 0, now())
  ON CONFLICT (category) DO NOTHING
  RETURNING updated_at
), campaign AS (
  SELECT updated_at FROM inserted_campaign
  UNION ALL
  SELECT updated_at FROM config_apply_versions WHERE category = ?
  LIMIT 1
), superseded_requests AS (
  UPDATE parameter_sync_requests req
  SET status = 'cancelled',
      result_code = ?,
      completed_at = COALESCE(req.completed_at, now()),
      updated_at = now()
  FROM campaign current_campaign
  WHERE req.trigger_reason = ?
    AND req.campaign_id IS NOT NULL
    AND req.campaign_id <> ?
    AND NOT EXISTS (
      SELECT 1
      FROM config_apply_versions other_campaign
      WHERE other_campaign.category = ? || req.campaign_id::text
        AND other_campaign.updated_at >= current_campaign.updated_at
    )
    AND req.status IN (?, ?)
  RETURNING req.id
)`, category, category, ResultCodeSupersededRelease, TriggerOMCUpgrade,
			campaignID, releaseCampaignCategoryPrefix, RequestStatusAccepted, RequestStatusQueued).
		From("devices d").
		LeftJoin("parameter_sync_device_state state ON state.device_id = d.id").
		Where(sq.Eq{
			"d.deleted_at":      nil,
			"d.lifecycle_state": model.LifecycleCommissioned,
			"d.is_online":       true,
		}).
		Where("d.created_at < (SELECT updated_at FROM campaign)").
		Where("(state.next_auto_sync_at IS NULL OR state.next_auto_sync_at <= now())").
		Where(sq.Expr("NOT EXISTS (?)", completedOrActive)).
		OrderBy("state.last_attempt_at ASC NULLS FIRST", "d.id").
		Limit(uint64(limit)).
		ToSql()
}

func admissionReleaseCandidatesQuery(now time.Time, limit int) (string, []any, error) {
	if limit <= 0 {
		limit = 500
	}
	return storage.Psql.
		Select("reservation.id", "reservation.request_id", "reservation.admission_class", "reservation.bucket_id",
			"reservation.reserved_runs", "reservation.reserved_tasks",
			"req.status").
		From("parameter_sync_admission_reservations reservation").
		Join("parameter_sync_requests req ON req.id = reservation.request_id").
		Where(sq.Eq{"reservation.status": "reserved"}).
		Where(sq.Or{
			sq.Lt{"reservation.lease_until": now},
			sq.Eq{"req.status": []RequestStatus{
				RequestStatusSucceeded, RequestStatusFailed, RequestStatusTimedOut,
				RequestStatusCancelled, RequestStatusDeduplicated, RequestStatusRejected,
			}},
		}).
		OrderBy("reservation.updated_at", "reservation.id").
		Limit(uint64(limit)).
		Suffix("FOR UPDATE OF reservation SKIP LOCKED").
		ToSql()
}

func (r *PGRepository) ListReleaseCandidates(
	ctx context.Context,
	campaignID uuid.UUID,
	limit int,
) ([]*model.Device, error) {
	query, args, err := listReleaseCandidatesQuery(campaignID, limit)
	if err != nil {
		return nil, fmt.Errorf("build list OMC release candidates: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list OMC release candidates: %w", err)
	}
	defer rows.Close()

	devices := make([]*model.Device, 0)
	for rows.Next() {
		dev := &model.Device{}
		if err := rows.Scan(&dev.ID, &dev.SerialNumber); err != nil {
			return nil, fmt.Errorf("scan OMC release candidate: %w", err)
		}
		devices = append(devices, dev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate OMC release candidates: %w", err)
	}
	return devices, nil
}

// ListRegisteredSyncCandidates returns registrations whose durable full sync
// was never completed. The registration marker is written with the device, so
// it survives the at-least-once event consumer after its NATS delivery budget
// has been exhausted.
func (r *PGRepository) ListRegisteredSyncCandidates(ctx context.Context, limit int) ([]*model.Device, error) {
	if limit <= 0 {
		limit = 100
	}
	covered := sq.
		Select("1").
		From("parameter_sync_requests req").
		Where("req.device_id = d.id").
		Where(sq.Eq{"req.trigger_reason": TriggerDeviceRegistered}).
		Where("req.source_event_id = 'device_registered:' || d.id::text").
		Where(sq.Or{
			sq.Eq{"req.status": []RequestStatus{
				RequestStatusAccepted,
				RequestStatusQueued,
				RequestStatusRunning,
				RequestStatusSucceeded,
			}},
			sq.And{
				sq.Eq{"req.status": RequestStatusDeduplicated},
				sq.Expr(`EXISTS (
					SELECT 1 FROM parameter_sync_runs run
					WHERE run.id = req.active_run_id
					  AND run.sync_scope = ?
					  AND run.status IN (?, ?, ?, ?, ?, ?, ?)
				)`,
					SyncScopeFull,
					RunStatusPlanning,
					RunStatusEnqueuing,
					RunStatusWaitingDevice,
					RunStatusExecuting,
					RunStatusProcessing,
					RunStatusCancelling,
					RunStatusSucceeded,
				),
			},
		})
	query, args, err := storage.Psql.
		Select("d.id", "d.serial_number").
		From("devices d").
		LeftJoin("parameter_sync_device_state state ON state.device_id = d.id").
		Where(sq.Eq{"d.deleted_at": nil}).
		Where("d.extension_data ->> '_system_registration_source_event_id' IS NOT NULL").
		Where("(state.next_auto_sync_at IS NULL OR state.next_auto_sync_at <= now())").
		Where(sq.Expr("NOT EXISTS (?)", covered)).
		OrderBy("state.last_attempt_at ASC NULLS FIRST", "d.id").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list registered parameter sync candidates: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list registered parameter sync candidates: %w", err)
	}
	defer rows.Close()

	devices := make([]*model.Device, 0)
	for rows.Next() {
		dev := &model.Device{}
		if err := rows.Scan(&dev.ID, &dev.SerialNumber); err != nil {
			return nil, fmt.Errorf("scan registered parameter sync candidate: %w", err)
		}
		devices = append(devices, dev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate registered parameter sync candidates: %w", err)
	}
	return devices, nil
}

func (r *PGRepository) CreateRequest(ctx context.Context, req *SyncRequest) error {
	query, args, err := buildCreateRequest(req)
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("create parameter sync request: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrRequestIdempotencyConflict
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
			"result_code", "error_message", "campaign_id", "source_event_id", "origin_event_type",
			"model_upload_intent_id", "model_upload_status", "admission_class", "admission_reason",
			"admission_snapshot", "admission_queued_at", "deduplicated_to_request_id", "created_at", "completed_at", "updated_at").
		Values(req.ID, req.DeviceID, req.DeviceSN, req.CallerType, req.TriggerReason, req.SyncScope,
			paths, req.Status, req.Priority, req.NextAttemptAt, req.DeadlineAt, req.IdempotencyKey,
			req.ResultCode, req.ErrorMessage, req.CampaignID, req.SourceEventID, req.OriginEventType,
			req.ModelUploadIntentID, req.ModelUploadStatus, req.AdmissionClass, req.AdmissionReason,
			req.AdmissionSnapshot, req.AdmissionQueuedAt, req.DeduplicatedToRequestID, req.CreatedAt, req.CompletedAt, req.UpdatedAt).
		Suffix(`ON CONFLICT (caller_type, idempotency_key)
			WHERE idempotency_key IS NOT NULL DO NOTHING`).
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
		"campaign_id", "source_event_id", "origin_event_type", "model_upload_intent_id", "model_upload_status",
		"admission_class", "COALESCE(admission_reason, '')", "COALESCE(admission_snapshot, 'null'::jsonb)",
		"admission_queued_at", "deduplicated_to_request_id", "created_at", "started_at", "completed_at", "updated_at",
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
		&req.CampaignID, &req.SourceEventID, &req.OriginEventType, &req.ModelUploadIntentID, &req.ModelUploadStatus,
		&req.AdmissionClass, &req.AdmissionReason, &req.AdmissionSnapshot, &req.AdmissionQueuedAt, &req.DeduplicatedToRequestID,
		&req.CreatedAt, &req.StartedAt, &req.CompletedAt, &req.UpdatedAt,
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
		Set("result_code", nil).Set("error_message", nil).Set("completed_at", nil).
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

func (r *PGRepository) ListRunValues(ctx context.Context, runID uuid.UUID) ([]RunValue, error) {
	query, args, err := storage.Psql.Select(
		"run_id",
		"parameter_path",
		"private_path",
		"value #>> '{}' AS value",
		"COALESCE(value_type, '')",
		"writable",
		"fap_instance",
		"param_group",
		"created_at",
		"updated_at",
	).From("parameter_sync_staging_values").
		Where(sq.Eq{"run_id": runID}).
		OrderBy("parameter_path ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list parameter sync run values: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list parameter sync run values: %w", err)
	}
	defer rows.Close()
	values := make([]RunValue, 0)
	for rows.Next() {
		var value RunValue
		if err := rows.Scan(
			&value.RunID,
			&value.ParameterPath,
			&value.PrivatePath,
			&value.Value,
			&value.ValueType,
			&value.Writable,
			&value.FAPInstance,
			&value.ParamGroup,
			&value.CreatedAt,
			&value.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan parameter sync run value: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate parameter sync run values: %w", err)
	}
	return values, nil
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
	class := automaticAdmissionClass
	req.AdmissionClass = &class
	if !allowed {
		message := fmt.Sprintf("automatic parameter sync backed off until %s", next.Format(time.RFC3339))
		req.ResultCode = ResultCodeAutomaticBackoff
		req.ErrorMessage = message
		if req.TriggerReason == TriggerDeviceOnline {
			// DeviceOnline/BOOT may be the device's only reliable recovery signal.
			// Persist it for delayed dispatch instead of requiring another Inform
			// after the automatic gate opens.
			req.Status = RequestStatusQueued
			req.NextAttemptAt = next
			req.CompletedAt = nil
		} else {
			req.Status = RequestStatusRejected
			req.CompletedAt = &now
		}
	}
	query, args, err := buildCreateRequest(req)
	if err != nil {
		return false, err
	}
	tag, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("create automatic parameter sync request: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return false, ErrRequestIdempotencyConflict
	}
	if allowed {
		allowed, err = reserveAutomaticAdmission(ctx, tx, req, now)
		if err != nil {
			return false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit automatic parameter sync request: %w", err)
	}
	return allowed, nil
}

func reserveAutomaticAdmission(ctx context.Context, tx pgx.Tx, req *SyncRequest, now time.Time) (bool, error) {
	bucketID := automaticAdmissionBucket(req.DeviceID)
	var existingStatus string
	var existingLease time.Time
	existingQuery, existingArgs, err := storage.Psql.
		Select("status", "lease_until").
		From("parameter_sync_admission_reservations").
		Where(sq.Eq{"request_id": req.ID, "admission_class": automaticAdmissionClass}).
		Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return false, fmt.Errorf("build load automatic admission reservation: %w", err)
	}
	err = tx.QueryRow(ctx, existingQuery, existingArgs...).Scan(&existingStatus, &existingLease)
	if err == nil && existingStatus == "reserved" {
		req.ResultCode = ""
		req.AdmissionReason = ""
		req.UpdatedAt = now
		query, args, buildErr := storage.Psql.Update("parameter_sync_requests").
			Set("result_code", nil).
			Set("admission_reason", nil).
			Set("admission_queued_at", nil).
			Set("updated_at", now).
			Where(sq.Eq{"id": req.ID}).ToSql()
		if buildErr != nil {
			return false, fmt.Errorf("build reuse automatic admission reservation: %w", buildErr)
		}
		if _, updateErr := tx.Exec(ctx, query, args...); updateErr != nil {
			return false, fmt.Errorf("reuse automatic admission reservation: %w", updateErr)
		}
		if !existingLease.After(now) {
			leaseQuery, leaseArgs, buildErr := storage.Psql.Update("parameter_sync_admission_reservations").
				Set("lease_until", now.Add(automaticAdmissionLease)).Set("updated_at", now).
				Where(sq.Eq{"request_id": req.ID, "admission_class": automaticAdmissionClass, "status": "reserved"}).ToSql()
			if buildErr != nil {
				return false, fmt.Errorf("build renew automatic admission reservation: %w", buildErr)
			}
			if _, updateErr := tx.Exec(ctx, leaseQuery, leaseArgs...); updateErr != nil {
				return false, fmt.Errorf("renew automatic admission reservation: %w", updateErr)
			}
		}
		return true, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return false, fmt.Errorf("load automatic admission reservation: %w", err)
	}

	stateQuery, stateArgs, err := storage.Psql.Insert("parameter_sync_admission_state").
		Columns("admission_class", "bucket_id", "active_run_limit", "active_task_limit", "reserved_runs", "reserved_tasks", "version", "updated_at").
		Values(automaticAdmissionClass, bucketID, automaticAdmissionRunsPerBucket, automaticAdmissionTasksPerBucket,
			1, 0, 1, now).
		Suffix(`ON CONFLICT (admission_class, bucket_id) DO UPDATE SET
  active_run_limit = CASE WHEN parameter_sync_admission_state.active_run_limit <= 0 THEN EXCLUDED.active_run_limit ELSE parameter_sync_admission_state.active_run_limit END,
  active_task_limit = CASE WHEN parameter_sync_admission_state.active_task_limit <= 0 THEN EXCLUDED.active_task_limit ELSE parameter_sync_admission_state.active_task_limit END,
  reserved_runs = parameter_sync_admission_state.reserved_runs + EXCLUDED.reserved_runs,
  reserved_tasks = parameter_sync_admission_state.reserved_tasks + EXCLUDED.reserved_tasks,
  version = parameter_sync_admission_state.version + 1,
  updated_at = EXCLUDED.updated_at
WHERE parameter_sync_admission_state.reserved_runs + EXCLUDED.reserved_runs <=
      CASE WHEN parameter_sync_admission_state.active_run_limit <= 0 THEN EXCLUDED.active_run_limit ELSE parameter_sync_admission_state.active_run_limit END
  AND parameter_sync_admission_state.reserved_tasks + EXCLUDED.reserved_tasks <=
      CASE WHEN parameter_sync_admission_state.active_task_limit <= 0 THEN EXCLUDED.active_task_limit ELSE parameter_sync_admission_state.active_task_limit END
RETURNING reserved_runs, reserved_tasks`).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build reserve automatic admission state: %w", err)
	}
	var reservedRuns, reservedTasks int
	err = tx.QueryRow(ctx, stateQuery, stateArgs...).Scan(&reservedRuns, &reservedTasks)
	if errors.Is(err, pgx.ErrNoRows) {
		snapshot, marshalErr := json.Marshal(map[string]any{
			"bucket_id":         bucketID,
			"active_run_limit":  automaticAdmissionRunsPerBucket,
			"active_task_limit": automaticAdmissionTasksPerBucket,
		})
		if marshalErr != nil {
			return false, fmt.Errorf("marshal automatic admission backpressure snapshot: %w", marshalErr)
		}
		nextAttempt := now.Add(automaticAdmissionRetry)
		req.Status = RequestStatusQueued
		req.ResultCode = ResultCodeAutomaticBackpressure
		req.AdmissionReason = "global automatic parameter sync capacity is full"
		req.AdmissionSnapshot = snapshot
		if req.AdmissionQueuedAt == nil {
			queuedAt := now
			req.AdmissionQueuedAt = &queuedAt
		}
		req.NextAttemptAt = nextAttempt
		req.UpdatedAt = now
		query, args, buildErr := storage.Psql.Update("parameter_sync_requests").
			Set("status", req.Status).
			Set("result_code", req.ResultCode).
			Set("admission_class", automaticAdmissionClass).
			Set("admission_reason", req.AdmissionReason).
			Set("admission_snapshot", snapshot).
			Set("admission_queued_at", sq.Expr("COALESCE(admission_queued_at, ?)", now)).
			Set("next_attempt_at", nextAttempt).
			Set("updated_at", now).
			Where(sq.Eq{"id": req.ID}).ToSql()
		if buildErr != nil {
			return false, fmt.Errorf("build queue backpressured automatic request: %w", buildErr)
		}
		if _, updateErr := tx.Exec(ctx, query, args...); updateErr != nil {
			return false, fmt.Errorf("queue backpressured automatic request: %w", updateErr)
		}
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("reserve automatic admission state: %w", err)
	}

	leaseUntil := now.Add(automaticAdmissionLease)
	reservationQuery, reservationArgs, err := storage.Psql.Insert("parameter_sync_admission_reservations").
		Columns("request_id", "admission_class", "bucket_id", "reserved_runs", "reserved_tasks", "status", "lease_until", "created_at", "updated_at").
		Values(req.ID, automaticAdmissionClass, bucketID, 1, 0, "reserved", leaseUntil, now, now).
		Suffix(`ON CONFLICT (request_id, admission_class) DO UPDATE SET
  bucket_id = EXCLUDED.bucket_id,
  reserved_runs = EXCLUDED.reserved_runs,
  reserved_tasks = EXCLUDED.reserved_tasks,
  status = 'reserved',
  lease_until = EXCLUDED.lease_until,
  released_at = NULL,
  updated_at = EXCLUDED.updated_at
WHERE parameter_sync_admission_reservations.status IN ('released', 'expired')
RETURNING id`).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build automatic admission reservation: %w", err)
	}
	var reservationID uuid.UUID
	if err := tx.QueryRow(ctx, reservationQuery, reservationArgs...).Scan(&reservationID); err != nil {
		return false, fmt.Errorf("create automatic admission reservation: %w", err)
	}
	snapshot, err := json.Marshal(map[string]any{
		"bucket_id":         bucketID,
		"reserved_runs":     reservedRuns,
		"reserved_tasks":    reservedTasks,
		"active_run_limit":  automaticAdmissionRunsPerBucket,
		"active_task_limit": automaticAdmissionTasksPerBucket,
		"lease_until":       leaseUntil,
	})
	if err != nil {
		return false, fmt.Errorf("marshal automatic admission snapshot: %w", err)
	}
	req.ResultCode = ""
	req.AdmissionReason = ""
	req.AdmissionSnapshot = snapshot
	req.AdmissionQueuedAt = nil
	req.UpdatedAt = now
	query, args, err := storage.Psql.Update("parameter_sync_requests").
		Set("result_code", nil).
		Set("admission_class", automaticAdmissionClass).
		Set("admission_reason", nil).
		Set("admission_snapshot", snapshot).
		Set("admission_queued_at", nil).
		Set("updated_at", now).
		Where(sq.Eq{"id": req.ID}).ToSql()
	if err != nil {
		return false, fmt.Errorf("build record automatic admission: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return false, fmt.Errorf("record automatic admission: %w", err)
	}
	return true, nil
}

func (r *PGRepository) TryReserveAutomaticAdmission(ctx context.Context, req *SyncRequest, now time.Time) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin queued automatic admission: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	var status RequestStatus
	requestQuery, requestArgs, err := storage.Psql.Select("status").From("parameter_sync_requests").
		Where(sq.Eq{"id": req.ID}).Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return false, fmt.Errorf("build lock queued automatic request: %w", err)
	}
	if err := tx.QueryRow(ctx, requestQuery, requestArgs...).Scan(&status); err != nil {
		return false, fmt.Errorf("lock queued automatic request: %w", err)
	}
	if status != RequestStatusQueued {
		return false, nil
	}
	allowed, err := reserveAutomaticAdmission(ctx, tx, req, now)
	if err != nil {
		return false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit queued automatic admission: %w", err)
	}
	return allowed, nil
}

// AdjustAutomaticAdmissionTasks replaces the planning estimate with the exact
// batch count before any run or device task is created. If the bucket has no
// task capacity, the run slot is released and the durable request stays queued.
func (r *PGRepository) AdjustAutomaticAdmissionTasks(
	ctx context.Context,
	req *SyncRequest,
	plannedTasks int,
	now time.Time,
) (bool, error) {
	if plannedTasks < 0 {
		return false, fmt.Errorf("adjust automatic admission tasks: planned task count must not be negative")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin automatic admission task adjustment: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	requestLockQuery, requestLockArgs, err := storage.Psql.Select("status").From("parameter_sync_requests").
		Where(sq.Eq{"id": req.ID}).Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return false, fmt.Errorf("build lock automatic request for task adjustment: %w", err)
	}
	var requestStatus RequestStatus
	if err := tx.QueryRow(ctx, requestLockQuery, requestLockArgs...).Scan(&requestStatus); err != nil {
		return false, fmt.Errorf("lock automatic request for task adjustment: %w", err)
	}
	if requestStatus != RequestStatusAccepted && requestStatus != RequestStatusQueued {
		return false, fmt.Errorf("adjust automatic admission tasks: %w", ErrRequestStateConflict)
	}
	query, args, err := storage.Psql.Select("bucket_id", "reserved_tasks", "status").
		From("parameter_sync_admission_reservations").
		Where(sq.Eq{"request_id": req.ID, "admission_class": automaticAdmissionClass}).
		Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return false, fmt.Errorf("build lock automatic admission task reservation: %w", err)
	}
	var bucketID int16
	var currentTasks int
	var reservationStatus string
	if err := tx.QueryRow(ctx, query, args...).Scan(&bucketID, &currentTasks, &reservationStatus); err != nil {
		return false, fmt.Errorf("lock automatic admission task reservation: %w", err)
	}
	if reservationStatus != "reserved" {
		return false, fmt.Errorf("adjust automatic admission tasks: reservation status is %s", reservationStatus)
	}
	delta := plannedTasks - currentTasks
	if delta > 0 {
		stateQuery, stateArgs, buildErr := storage.Psql.Update("parameter_sync_admission_state").
			Set("reserved_tasks", sq.Expr("reserved_tasks + ?", delta)).
			Set("version", sq.Expr("version + 1")).Set("updated_at", now).
			Where(sq.Eq{"admission_class": automaticAdmissionClass, "bucket_id": bucketID}).
			Where("reserved_tasks + ? <= active_task_limit", delta).
			Suffix("RETURNING reserved_tasks").ToSql()
		if buildErr != nil {
			return false, fmt.Errorf("build adjust automatic admission task state: %w", buildErr)
		}
		var totalTasks int
		adjustErr := tx.QueryRow(ctx, stateQuery, stateArgs...).Scan(&totalTasks)
		if errors.Is(adjustErr, pgx.ErrNoRows) {
			nextAttempt := now.Add(automaticAdmissionRetry)
			reservationQuery, reservationArgs, buildErr := storage.Psql.Update("parameter_sync_admission_reservations").
				Set("status", "released").Set("released_at", now).Set("updated_at", now).
				Where(sq.Eq{"request_id": req.ID, "admission_class": automaticAdmissionClass, "status": "reserved"}).ToSql()
			if buildErr != nil {
				return false, fmt.Errorf("build release task-backpressured reservation: %w", buildErr)
			}
			if _, updateErr := tx.Exec(ctx, reservationQuery, reservationArgs...); updateErr != nil {
				return false, fmt.Errorf("release task-backpressured reservation: %w", updateErr)
			}
			stateReleaseQuery, stateReleaseArgs, buildErr := storage.Psql.Update("parameter_sync_admission_state").
				Set("reserved_runs", sq.Expr("GREATEST(reserved_runs - 1, 0)")).
				Set("reserved_tasks", sq.Expr("GREATEST(reserved_tasks - ?, 0)", currentTasks)).
				Set("version", sq.Expr("version + 1")).Set("updated_at", now).
				Where(sq.Eq{"admission_class": automaticAdmissionClass, "bucket_id": bucketID}).ToSql()
			if buildErr != nil {
				return false, fmt.Errorf("build release task-backpressured admission state: %w", buildErr)
			}
			if _, updateErr := tx.Exec(ctx, stateReleaseQuery, stateReleaseArgs...); updateErr != nil {
				return false, fmt.Errorf("release task-backpressured admission state: %w", updateErr)
			}
			requestQuery, requestArgs, buildErr := storage.Psql.Update("parameter_sync_requests").
				Set("status", RequestStatusQueued).Set("result_code", ResultCodeAutomaticBackpressure).
				Set("admission_reason", "global automatic parameter sync task capacity is full").
				Set("admission_queued_at", sq.Expr("COALESCE(admission_queued_at, ?)", now)).
				Set("next_attempt_at", nextAttempt).Set("updated_at", now).
				Where(sq.Eq{"id": req.ID, "status": []RequestStatus{RequestStatusAccepted, RequestStatusQueued}, "run_id": nil}).ToSql()
			if buildErr != nil {
				return false, fmt.Errorf("build queue task-backpressured request: %w", buildErr)
			}
			requestTag, updateErr := tx.Exec(ctx, requestQuery, requestArgs...)
			if updateErr != nil {
				return false, fmt.Errorf("queue task-backpressured request: %w", updateErr)
			}
			if requestTag.RowsAffected() != 1 {
				return false, fmt.Errorf("queue task-backpressured request: %w", ErrRequestStateConflict)
			}
			if err := tx.Commit(ctx); err != nil {
				return false, fmt.Errorf("commit task-backpressured automatic request: %w", err)
			}
			queuedAt := now
			req.Status = RequestStatusQueued
			req.ResultCode = ResultCodeAutomaticBackpressure
			req.AdmissionReason = "global automatic parameter sync task capacity is full"
			req.AdmissionQueuedAt = &queuedAt
			req.NextAttemptAt = nextAttempt
			return false, nil
		}
		if adjustErr != nil {
			return false, fmt.Errorf("adjust automatic admission task state: %w", adjustErr)
		}
	} else if delta < 0 {
		stateQuery, stateArgs, buildErr := storage.Psql.Update("parameter_sync_admission_state").
			Set("reserved_tasks", sq.Expr("GREATEST(reserved_tasks + ?, 0)", delta)).
			Set("version", sq.Expr("version + 1")).Set("updated_at", now).
			Where(sq.Eq{"admission_class": automaticAdmissionClass, "bucket_id": bucketID}).ToSql()
		if buildErr != nil {
			return false, fmt.Errorf("build reduce automatic admission task state: %w", buildErr)
		}
		if _, updateErr := tx.Exec(ctx, stateQuery, stateArgs...); updateErr != nil {
			return false, fmt.Errorf("reduce automatic admission task state: %w", updateErr)
		}
	}
	reservationQuery, reservationArgs, err := storage.Psql.Update("parameter_sync_admission_reservations").
		Set("reserved_tasks", plannedTasks).Set("updated_at", now).
		Where(sq.Eq{"request_id": req.ID, "admission_class": automaticAdmissionClass, "status": "reserved"}).ToSql()
	if err != nil {
		return false, fmt.Errorf("build record exact automatic admission tasks: %w", err)
	}
	if _, err := tx.Exec(ctx, reservationQuery, reservationArgs...); err != nil {
		return false, fmt.Errorf("record exact automatic admission tasks: %w", err)
	}
	requestQuery, requestArgs, err := storage.Psql.Update("parameter_sync_requests").
		Set("result_code", nil).Set("admission_reason", nil).Set("admission_queued_at", nil).Set("updated_at", now).
		Where(sq.Eq{"id": req.ID}).ToSql()
	if err != nil {
		return false, fmt.Errorf("build clear automatic admission task backpressure: %w", err)
	}
	if _, err := tx.Exec(ctx, requestQuery, requestArgs...); err != nil {
		return false, fmt.Errorf("clear automatic admission task backpressure: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit automatic admission task adjustment: %w", err)
	}
	req.ResultCode = ""
	req.AdmissionReason = ""
	req.AdmissionQueuedAt = nil
	return true, nil
}

func (r *PGRepository) QueueRequestAndReleaseAutomaticAdmission(
	ctx context.Context,
	req *SyncRequest,
	nextAttemptAt time.Time,
	now time.Time,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin queue and release automatic admission: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	requestLockQuery, requestLockArgs, err := storage.Psql.Select("status").From("parameter_sync_requests").
		Where(sq.Eq{"id": req.ID}).Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return fmt.Errorf("build lock automatic request for queue: %w", err)
	}
	var requestStatus RequestStatus
	if err := tx.QueryRow(ctx, requestLockQuery, requestLockArgs...).Scan(&requestStatus); err != nil {
		return fmt.Errorf("lock automatic request for queue: %w", err)
	}
	query, args, err := storage.Psql.Select("bucket_id", "reserved_runs", "reserved_tasks", "status").
		From("parameter_sync_admission_reservations").
		Where(sq.Eq{"request_id": req.ID, "admission_class": automaticAdmissionClass}).
		Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return fmt.Errorf("build lock automatic admission release: %w", err)
	}
	var bucketID int16
	var reservedRuns, reservedTasks int
	var status string
	err = tx.QueryRow(ctx, query, args...).Scan(&bucketID, &reservedRuns, &reservedTasks, &status)
	reservationExists := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("lock automatic admission release: %w", err)
	}
	if reservationExists && status == "reserved" {
		reservationQuery, reservationArgs, buildErr := storage.Psql.Update("parameter_sync_admission_reservations").
			Set("status", "released").Set("released_at", now).Set("updated_at", now).
			Where(sq.Eq{"request_id": req.ID, "admission_class": automaticAdmissionClass, "status": "reserved"}).ToSql()
		if buildErr != nil {
			return fmt.Errorf("build release automatic admission reservation: %w", buildErr)
		}
		if _, updateErr := tx.Exec(ctx, reservationQuery, reservationArgs...); updateErr != nil {
			return fmt.Errorf("release automatic admission reservation: %w", updateErr)
		}
		stateQuery, stateArgs, buildErr := storage.Psql.Update("parameter_sync_admission_state").
			Set("reserved_runs", sq.Expr("GREATEST(reserved_runs - ?, 0)", reservedRuns)).
			Set("reserved_tasks", sq.Expr("GREATEST(reserved_tasks - ?, 0)", reservedTasks)).
			Set("version", sq.Expr("version + 1")).Set("updated_at", now).
			Where(sq.Eq{"admission_class": automaticAdmissionClass, "bucket_id": bucketID}).ToSql()
		if buildErr != nil {
			return fmt.Errorf("build release automatic admission state: %w", buildErr)
		}
		if _, updateErr := tx.Exec(ctx, stateQuery, stateArgs...); updateErr != nil {
			return fmt.Errorf("release automatic admission state: %w", updateErr)
		}
	}
	requestQuery, requestArgs, err := storage.Psql.Update("parameter_sync_requests").
		Set("status", RequestStatusQueued).Set("result_code", ResultCodeActiveSyncExists).
		Set("active_run_id", nil).Set("next_attempt_at", nextAttemptAt).
		Set("admission_queued_at", nil).Set("updated_at", now).
		Where(sq.Eq{"id": req.ID, "status": []RequestStatus{RequestStatusAccepted, RequestStatusQueued}, "run_id": nil}).ToSql()
	if err != nil {
		return fmt.Errorf("build queue released automatic request: %w", err)
	}
	requestTag, err := tx.Exec(ctx, requestQuery, requestArgs...)
	if err != nil {
		return fmt.Errorf("queue released automatic request: %w", err)
	}
	if requestTag.RowsAffected() != 1 {
		return fmt.Errorf("queue released automatic request: %w", ErrRequestStateConflict)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit queue and release automatic admission: %w", err)
	}
	return nil
}

// ReconcileAutomaticAdmission releases capacity for terminal requests and
// expires abandoned reservations whose owner did not finish before the lease.
func (r *PGRepository) ReconcileAutomaticAdmission(ctx context.Context, now time.Time, limit int) (int, error) {
	if limit <= 0 {
		limit = 500
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin automatic admission reconciliation: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	locked, err := tryAutomaticAdmissionMaintenanceLock(ctx, tx)
	if err != nil {
		return 0, err
	}
	if !locked {
		return 0, nil
	}
	recovered, err := recoverAcceptedAutomaticAdmissions(ctx, tx, now, limit)
	if err != nil {
		return 0, err
	}
	query, args, err := admissionReleaseCandidatesQuery(now, limit)
	if err != nil {
		return 0, fmt.Errorf("build automatic admission reconciliation: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("list automatic admission releases: %w", err)
	}
	type release struct {
		id             uuid.UUID
		requestID      uuid.UUID
		admissionClass string
		bucketID       int16
		reservedRuns   int
		reservedTasks  int
		requestStatus  RequestStatus
	}
	var releases []release
	for rows.Next() {
		var item release
		if err := rows.Scan(&item.id, &item.requestID, &item.admissionClass, &item.bucketID,
			&item.reservedRuns, &item.reservedTasks, &item.requestStatus); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan automatic admission release: %w", err)
		}
		releases = append(releases, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate automatic admission releases: %w", err)
	}
	rows.Close()
	for _, item := range releases {
		if !item.requestStatus.Terminal() {
			leaseQuery, leaseArgs, buildErr := storage.Psql.Update("parameter_sync_admission_reservations").
				Set("lease_until", now.Add(automaticAdmissionLease)).Set("updated_at", now).
				Where(sq.Eq{"id": item.id, "status": "reserved"}).ToSql()
			if buildErr != nil {
				return 0, fmt.Errorf("build renew automatic admission lease: %w", buildErr)
			}
			if _, updateErr := tx.Exec(ctx, leaseQuery, leaseArgs...); updateErr != nil {
				return 0, fmt.Errorf("renew automatic admission lease: %w", updateErr)
			}
			continue
		}
		reservationQuery, reservationArgs, buildErr := storage.Psql.Update("parameter_sync_admission_reservations").
			Set("status", "released").Set("released_at", now).Set("updated_at", now).
			Where(sq.Eq{"id": item.id, "status": "reserved"}).ToSql()
		if buildErr != nil {
			return 0, fmt.Errorf("build release automatic admission reservation: %w", buildErr)
		}
		tag, updateErr := tx.Exec(ctx, reservationQuery, reservationArgs...)
		if updateErr != nil {
			return 0, fmt.Errorf("release automatic admission reservation: %w", updateErr)
		}
		if tag.RowsAffected() == 0 {
			continue
		}
		stateQuery, stateArgs, buildErr := storage.Psql.Update("parameter_sync_admission_state").
			Set("reserved_runs", sq.Expr("GREATEST(reserved_runs - ?, 0)", item.reservedRuns)).
			Set("reserved_tasks", sq.Expr("GREATEST(reserved_tasks - ?, 0)", item.reservedTasks)).
			Set("version", sq.Expr("version + 1")).Set("updated_at", now).
			Where(sq.Eq{"admission_class": item.admissionClass, "bucket_id": item.bucketID}).ToSql()
		if buildErr != nil {
			return 0, fmt.Errorf("build release automatic admission state: %w", buildErr)
		}
		if _, updateErr := tx.Exec(ctx, stateQuery, stateArgs...); updateErr != nil {
			return 0, fmt.Errorf("release automatic admission state: %w", updateErr)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit automatic admission reconciliation: %w", err)
	}
	return recovered + len(releases), nil
}

func recoverAcceptedAutomaticAdmissions(ctx context.Context, tx pgx.Tx, now time.Time, limit int) (int, error) {
	query, args, err := storage.Psql.Select("req.id").From("parameter_sync_requests req").
		Join("parameter_sync_admission_reservations reservation ON reservation.request_id = req.id").
		Where(sq.Eq{"req.status": RequestStatusAccepted, "req.run_id": nil, "reservation.status": "reserved"}).
		Where(sq.Lt{"reservation.lease_until": now}).
		OrderBy("req.created_at", "req.id").Limit(uint64(limit)).
		Suffix("FOR UPDATE OF req SKIP LOCKED").ToSql()
	if err != nil {
		return 0, fmt.Errorf("build recover accepted automatic admissions: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("list recoverable accepted automatic admissions: %w", err)
	}
	var requestIDs []uuid.UUID
	for rows.Next() {
		var requestID uuid.UUID
		if err := rows.Scan(&requestID); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan recoverable accepted automatic admission: %w", err)
		}
		requestIDs = append(requestIDs, requestID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate recoverable accepted automatic admissions: %w", err)
	}
	rows.Close()
	for _, requestID := range requestIDs {
		reservationQuery, reservationArgs, buildErr := storage.Psql.Update("parameter_sync_admission_reservations").
			Set("lease_until", now.Add(automaticAdmissionLease)).Set("updated_at", now).
			Where(sq.Eq{"request_id": requestID, "admission_class": automaticAdmissionClass, "status": "reserved"}).ToSql()
		if buildErr != nil {
			return 0, fmt.Errorf("build renew recovered automatic admission: %w", buildErr)
		}
		if _, updateErr := tx.Exec(ctx, reservationQuery, reservationArgs...); updateErr != nil {
			return 0, fmt.Errorf("renew recovered automatic admission: %w", updateErr)
		}
		requestQuery, requestArgs, buildErr := storage.Psql.Update("parameter_sync_requests").
			Set("status", RequestStatusQueued).Set("next_attempt_at", now).Set("updated_at", now).
			Where(sq.Eq{"id": requestID, "status": RequestStatusAccepted, "run_id": nil}).ToSql()
		if buildErr != nil {
			return 0, fmt.Errorf("build recover admitted automatic request: %w", buildErr)
		}
		if _, updateErr := tx.Exec(ctx, requestQuery, requestArgs...); updateErr != nil {
			return 0, fmt.Errorf("recover admitted automatic request: %w", updateErr)
		}
	}
	return len(requestIDs), nil
}

// RepairAutomaticAdmissionCounters rebuilds the small sharded state table from
// durable reservations. It writes only drifted buckets, so periodic retention
// repair cannot create a steady stream of PostgreSQL WAL and disk I/O.
func (r *PGRepository) RepairAutomaticAdmissionCounters(ctx context.Context, now time.Time) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin automatic admission counter repair: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	locked, err := tryAutomaticAdmissionMaintenanceLock(ctx, tx)
	if err != nil {
		return 0, err
	}
	if !locked {
		return 0, nil
	}
	lockQuery, lockArgs, err := storage.Psql.Select("bucket_id").From("parameter_sync_admission_state").
		Where(sq.Eq{"admission_class": automaticAdmissionClass}).OrderBy("bucket_id").Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return 0, fmt.Errorf("build lock automatic admission counter state: %w", err)
	}
	rows, err := tx.Query(ctx, lockQuery, lockArgs...)
	if err != nil {
		return 0, fmt.Errorf("lock automatic admission counter state: %w", err)
	}
	for rows.Next() {
		var bucketID int16
		if err := rows.Scan(&bucketID); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan automatic admission counter state: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("iterate automatic admission counter state: %w", err)
	}
	rows.Close()
	repairQuery, repairArgs, err := storage.Psql.Update("parameter_sync_admission_state state").
		Set("reserved_runs", sq.Expr(`COALESCE((
  SELECT SUM(reservation.reserved_runs)
  FROM parameter_sync_admission_reservations reservation
  WHERE reservation.admission_class = state.admission_class
    AND reservation.bucket_id = state.bucket_id
    AND reservation.status = 'reserved'
), 0)`)).
		Set("reserved_tasks", sq.Expr(`COALESCE((
  SELECT SUM(reservation.reserved_tasks)
  FROM parameter_sync_admission_reservations reservation
  WHERE reservation.admission_class = state.admission_class
    AND reservation.bucket_id = state.bucket_id
    AND reservation.status = 'reserved'
), 0)`)).
		Set("version", sq.Expr("version + 1")).Set("updated_at", now).
		Where(sq.Eq{"state.admission_class": automaticAdmissionClass}).
		Where(`(state.reserved_runs, state.reserved_tasks) IS DISTINCT FROM (
  COALESCE((SELECT SUM(reservation.reserved_runs) FROM parameter_sync_admission_reservations reservation WHERE reservation.admission_class = state.admission_class AND reservation.bucket_id = state.bucket_id AND reservation.status = 'reserved'), 0),
  COALESCE((SELECT SUM(reservation.reserved_tasks) FROM parameter_sync_admission_reservations reservation WHERE reservation.admission_class = state.admission_class AND reservation.bucket_id = state.bucket_id AND reservation.status = 'reserved'), 0)
)`).ToSql()
	if err != nil {
		return 0, fmt.Errorf("build automatic admission counter repair: %w", err)
	}
	tag, err := tx.Exec(ctx, repairQuery, repairArgs...)
	if err != nil {
		return 0, fmt.Errorf("repair automatic admission counters: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit automatic admission counter repair: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func tryAutomaticAdmissionMaintenanceLock(ctx context.Context, tx pgx.Tx) (bool, error) {
	query, args, err := storage.Psql.Select().Column(
		sq.Expr("pg_try_advisory_xact_lock(?)", automaticAdmissionMaintenanceLock),
	).
		From("(SELECT 1) admission_maintenance_lock").ToSql()
	if err != nil {
		return false, fmt.Errorf("build automatic admission maintenance lock: %w", err)
	}
	var locked bool
	if err := tx.QueryRow(ctx, query, args...).Scan(&locked); err != nil {
		return false, fmt.Errorf("acquire automatic admission maintenance lock: %w", err)
	}
	return locked, nil
}

func (r *PGRepository) GetAutomaticAdmissionStats(ctx context.Context, now time.Time) (AutomaticAdmissionStats, error) {
	query, args, err := storage.Psql.Select(
		`COALESCE((SELECT SUM(reserved_runs) FROM parameter_sync_admission_state WHERE admission_class = 'global'), 0)`,
		`COALESCE((SELECT COUNT(*) FROM parameter_sync_requests WHERE status = 'queued' AND result_code = 'AUTOMATIC_BACKPRESSURE'), 0)`,
	).Column(sq.Expr(
		`COALESCE((SELECT EXTRACT(EPOCH FROM (?::timestamptz - MIN(admission_queued_at))) FROM parameter_sync_requests WHERE status = 'queued' AND result_code = 'AUTOMATIC_BACKPRESSURE'), 0)`,
		now,
	)).From("(SELECT 1) admission_stats").ToSql()
	if err != nil {
		return AutomaticAdmissionStats{}, fmt.Errorf("build automatic admission stats: %w", err)
	}
	var stats AutomaticAdmissionStats
	if err := r.pool.QueryRow(ctx, query, args...).Scan(
		&stats.ReservedRuns, &stats.QueuedRequests, &stats.OldestQueueAgeSecond,
	); err != nil {
		return AutomaticAdmissionStats{}, fmt.Errorf("load automatic admission stats: %w", err)
	}
	return stats, nil
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
