package paramsync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type BindingCoordinator struct {
	pool *pgxpool.Pool
	bus  event.EventBus
	subs []event.Subscription
	now  func() time.Time
}

type bindingTarget struct {
	bindingStatus      string
	provisioningStatus string
	terminal           bool
}

func bindingTargetForRunStatus(status RunStatus) bindingTarget {
	switch status {
	case RunStatusSucceeded:
		return bindingTarget{bindingStatus: "completed", provisioningStatus: "completed", terminal: true}
	case RunStatusFailed:
		return bindingTarget{bindingStatus: "failed", provisioningStatus: "failed", terminal: true}
	case RunStatusCancelled:
		return bindingTarget{bindingStatus: "cancelled", provisioningStatus: "failed", terminal: true}
	default:
		return bindingTarget{bindingStatus: "waiting", provisioningStatus: "syncing"}
	}
}

func NewBindingCoordinator(pool *pgxpool.Pool, bus event.EventBus) *BindingCoordinator {
	return &BindingCoordinator{pool: pool, bus: bus, now: func() time.Time { return time.Now().UTC() }}
}

func (c *BindingCoordinator) Bind(ctx context.Context, requestID, runID, provisioningTaskID uuid.UUID) error {
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin parameter sync provisioning binding: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	// Lock the run before inserting the binding. Finalization takes an exclusive
	// row lock, so either the binding commits first and is observed by the
	// terminal event, or an already-terminal run is completed inline below.
	var runStatus RunStatus
	var runError string
	if err := tx.QueryRow(ctx, `SELECT status, COALESCE(error_message, '') FROM parameter_sync_runs WHERE id=$1 FOR SHARE`, runID).
		Scan(&runStatus, &runError); err != nil {
		return fmt.Errorf("lock parameter sync run for provisioning binding: %w", err)
	}
	target := bindingTargetForRunStatus(runStatus)
	var completedAt *time.Time
	if target.terminal {
		now := c.now()
		completedAt = &now
	}
	insert := storage.Psql.Insert("parameter_sync_request_bindings").
		Columns("request_id", "run_id", "provisioning_task_id", "status", "completed_at").
		Values(requestID, runID, provisioningTaskID, target.bindingStatus, completedAt)
	if target.terminal {
		insert = insert.Suffix(`ON CONFLICT (request_id, provisioning_task_id) DO UPDATE SET
status=EXCLUDED.status, completed_at=EXCLUDED.completed_at`)
	} else {
		insert = insert.Suffix("ON CONFLICT (request_id, provisioning_task_id) DO NOTHING")
	}
	query, args, err := insert.ToSql()
	if err != nil {
		return fmt.Errorf("build parameter sync provisioning binding: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("create parameter sync provisioning binding: %w", err)
	}
	errorMessage := ""
	if target.provisioningStatus == "failed" {
		errorMessage = runError
		if errorMessage == "" {
			errorMessage = "parameter synchronization failed"
		}
	}
	update, updateArgs, err := storage.Psql.Update("provisioning_tasks").Set("status", target.provisioningStatus).
		Set("error_message", errorMessage).Set("completed_at", completedAt).Set("updated_at", c.now()).
		Where(sq.Eq{"id": provisioningTaskID}).
		Where(sq.NotEq{"status": []string{"completed", "failed"}}).ToSql()
	if err != nil {
		return fmt.Errorf("build mark provisioning syncing: %w", err)
	}
	updateTag, err := tx.Exec(ctx, update, updateArgs...)
	if err != nil {
		return fmt.Errorf("mark provisioning syncing: %w", err)
	}
	if updateTag.RowsAffected() != 1 {
		return fmt.Errorf("mark provisioning syncing: provisioning task is missing or terminal")
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit parameter sync provisioning binding: %w", err)
	}
	return nil
}

func (c *BindingCoordinator) CompleteWithoutRun(ctx context.Context, provisioningTaskID uuid.UUID) error {
	now := c.now()
	query, args, err := storage.Psql.Update("provisioning_tasks").Set("status", "completed").
		Set("error_message", "").Set("completed_at", now).Set("updated_at", now).
		Where(sq.Eq{"id": provisioningTaskID}).Where(sq.NotEq{"status": []string{"completed", "failed"}}).ToSql()
	if err != nil {
		return fmt.Errorf("build complete provisioning without parameter sync run: %w", err)
	}
	if _, err := c.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("complete provisioning without parameter sync run: %w", err)
	}
	return nil
}

func (c *BindingCoordinator) Start() error {
	queues := map[string]string{
		event.SubjectParamSyncRunCompleted: "param-sync-provisioning-completed",
		event.SubjectParamSyncRunFailed:    "param-sync-provisioning-failed",
	}
	for _, subject := range []string{event.SubjectParamSyncRunCompleted, event.SubjectParamSyncRunFailed} {
		subject := subject
		sub, err := c.bus.QueueSubscribe(subject, queues[subject], func(ctx context.Context, evt event.Event) error {
			return c.handleTerminal(ctx, subject, evt)
		})
		if err != nil {
			return fmt.Errorf("subscribe parameter sync provisioning terminal %s: %w", subject, err)
		}
		c.subs = append(c.subs, sub)
	}
	return nil
}

func (c *BindingCoordinator) handleTerminal(ctx context.Context, subject string, evt event.Event) error {
	var payload struct {
		RunID uuid.UUID `json:"run_id"`
	}
	if err := json.Unmarshal(evt.Payload, &payload); err != nil || payload.RunID == uuid.Nil {
		return fmt.Errorf("decode parameter sync run terminal binding: %w", err)
	}
	targetBinding, targetProvisioning := "completed", "completed"
	errorMessage := ""
	if subject == event.SubjectParamSyncRunFailed {
		targetBinding, targetProvisioning = "failed", "failed"
		if err := c.pool.QueryRow(ctx, `SELECT COALESCE(error_message, '') FROM parameter_sync_runs WHERE id=$1`, payload.RunID).Scan(&errorMessage); err != nil {
			return fmt.Errorf("load failed parameter sync run error: %w", err)
		}
		if errorMessage == "" {
			errorMessage = "parameter synchronization failed"
		}
	}
	tx, err := c.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin complete parameter sync bindings: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	bindings, bindingArgs, err := storage.Psql.Update("parameter_sync_request_bindings").
		Set("status", targetBinding).Set("completed_at", c.now()).
		Where(sq.Eq{"run_id": payload.RunID, "status": "waiting"}).
		Suffix("RETURNING provisioning_task_id").ToSql()
	if err != nil {
		return fmt.Errorf("build complete parameter sync bindings: %w", err)
	}
	rows, err := tx.Query(ctx, bindings, bindingArgs...)
	if err != nil {
		return fmt.Errorf("complete parameter sync bindings: %w", err)
	}
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan parameter sync binding: %w", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) > 0 {
		now := c.now()
		update, updateArgs, err := storage.Psql.Update("provisioning_tasks").Set("status", targetProvisioning).
			Set("error_message", errorMessage).Set("completed_at", now).Set("updated_at", now).Where(sq.Eq{"id": ids}).
			Where(sq.NotEq{"status": []string{"completed", "failed"}}).ToSql()
		if err != nil {
			return fmt.Errorf("build complete bound provisioning tasks: %w", err)
		}
		if _, err := tx.Exec(ctx, update, updateArgs...); err != nil {
			return fmt.Errorf("complete bound provisioning tasks: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit complete parameter sync bindings: %w", err)
	}
	return nil
}

func (c *BindingCoordinator) Stop() error {
	var first error
	for _, sub := range c.subs {
		if err := sub.Unsubscribe(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
