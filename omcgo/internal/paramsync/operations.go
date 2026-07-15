package paramsync

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DeadOutboxItem struct {
	ID           uuid.UUID `json:"id"`
	EventType    string    `json:"event_type"`
	AggregateID  uuid.UUID `json:"aggregate_id"`
	AttemptCount int       `json:"attempt_count"`
	LastError    string    `json:"last_error"`
	CreatedAt    time.Time `json:"created_at"`
}

type Operations struct {
	pool    *pgxpool.Pool
	service *Service
	now     func() time.Time
}

func NewOperations(pool *pgxpool.Pool, service *Service) *Operations {
	return &Operations{pool: pool, service: service, now: func() time.Time { return time.Now().UTC() }}
}

func requestCanBeCancelled(status RequestStatus) bool {
	switch status {
	case RequestStatusAccepted, RequestStatusQueued, RequestStatusRunning:
		return true
	default:
		return false
	}
}

func retryDeadline(req *SyncRequest, retryAt time.Time) *time.Time {
	if req == nil || req.DeadlineAt == nil || req.CreatedAt.IsZero() {
		return nil
	}
	timeout := req.DeadlineAt.Sub(req.CreatedAt)
	if timeout <= 0 {
		return nil
	}
	deadline := retryAt.Add(timeout)
	return &deadline
}

func (o *Operations) CancelRequest(ctx context.Context, requestID uuid.UUID) error {
	tx, err := o.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin cancel parameter sync request: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	// Observe the association without taking a request lock, then acquire locks
	// in the same run -> request order used by result finalization. If the
	// request has not attached a run yet, locking the request first safely fences
	// a concurrent attach, whose accepted-state CAS will then fail.
	var observedRunID *uuid.UUID
	if err := tx.QueryRow(ctx, "SELECT run_id FROM parameter_sync_requests WHERE id=$1", requestID).Scan(&observedRunID); err != nil {
		return fmt.Errorf("load parameter sync request association for cancel: %w", err)
	}
	var run *SyncRun
	if observedRunID != nil {
		run, err = loadRunForUpdate(ctx, tx, *observedRunID)
		if err != nil {
			return err
		}
	}
	var lockedRunID *uuid.UUID
	var status RequestStatus
	if err := tx.QueryRow(ctx, "SELECT run_id, status FROM parameter_sync_requests WHERE id=$1 FOR UPDATE", requestID).Scan(&lockedRunID, &status); err != nil {
		return fmt.Errorf("lock parameter sync request for cancel: %w", err)
	}
	if !sameOptionalUUID(observedRunID, lockedRunID) {
		return fmt.Errorf("cancel parameter sync request: %w", ErrRequestStateConflict)
	}
	if !requestCanBeCancelled(status) {
		return fmt.Errorf("parameter sync request in status %s cannot be cancelled", status)
	}
	now := o.now()
	tag, err := tx.Exec(ctx, `UPDATE parameter_sync_requests SET status='cancelled', result_code='CANCELLED', active_run_id=NULL, completed_at=$2, updated_at=$2 WHERE id=$1 AND status IN ('accepted','queued','running')`, requestID, now)
	if err != nil {
		return fmt.Errorf("cancel parameter sync request: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("parameter sync request changed while being cancelled")
	}
	if run != nil {
		if !run.Status.Terminal() {
			if err := beginCancellingRun(ctx, tx, run, "parameter sync request cancelled", now); err != nil {
				return err
			}
			if err := cancelUnsentRunTasks(ctx, tx, run, now); err != nil {
				return err
			}
			counts, err := loadAuthoritativeRunCounts(ctx, tx, run.ID)
			if err != nil {
				return err
			}
			applyAuthoritativeRunCounts(run, counts)
			if run.ReadyToFinalize() {
				if err := finalizeConvergedFailedRun(ctx, tx, run, now); err != nil {
					return err
				}
			} else if err := updateRunProgress(ctx, tx, run, RunStatusCancelling); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

func sameOptionalUUID(left, right *uuid.UUID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func (o *Operations) RetryRequest(ctx context.Context, requestID uuid.UUID) (*SubmitResult, error) {
	req, err := o.service.GetRequest(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if !req.Status.Terminal() {
		return nil, fmt.Errorf("parameter sync request is not terminal")
	}
	retryAt := o.now()
	return o.service.Submit(ctx, SubmitCommand{
		DeviceID: req.DeviceID, DeviceSN: req.DeviceSN, CallerType: req.CallerType,
		TriggerReason: req.TriggerReason, Scope: req.SyncScope, RequestedPaths: req.RequestedPaths,
		Priority: req.Priority, DeadlineAt: retryDeadline(req, retryAt), IdempotencyKey: "retry:" + requestID.String(),
	})
}

func (o *Operations) ListDeadOutbox(ctx context.Context, limit int) ([]DeadOutboxItem, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := o.pool.Query(ctx, `SELECT id,event_type,aggregate_id,attempt_count,COALESCE(last_error,''),created_at FROM parameter_sync_outbox WHERE status='dead' ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list dead parameter sync outbox: %w", err)
	}
	defer rows.Close()
	var items []DeadOutboxItem
	for rows.Next() {
		var item DeadOutboxItem
		if err := rows.Scan(&item.ID, &item.EventType, &item.AggregateID, &item.AttemptCount, &item.LastError, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (o *Operations) RetryOutbox(ctx context.Context, id uuid.UUID) error {
	tag, err := o.pool.Exec(ctx, `UPDATE parameter_sync_outbox SET status='pending', attempt_count=0, next_attempt_at=$2, last_error=NULL, updated_at=$2 WHERE id=$1 AND status='dead'`, id, o.now())
	if err != nil {
		return fmt.Errorf("retry dead parameter sync outbox: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("dead parameter sync outbox not found")
	}
	return nil
}
