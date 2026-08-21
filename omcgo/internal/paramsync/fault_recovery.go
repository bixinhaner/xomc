package paramsync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/task"
)

// RecoveryTaskReleaser admits a committed replacement directly to the current
// device execution queue. The durable outbox remains pending as the crash-safe
// fallback and may replay the same task later; admission must therefore be
// idempotent.
type RecoveryTaskReleaser interface {
	ReleasePlannedTaskWithoutWake(ctx context.Context, planned *task.Task) (bool, error)
}

// GPVFaultRecoverer atomically extends a durable run before ACS completes the
// faulting task. After commit it also releases the replacement synchronously so
// the ACS fault handler can continue it in the current CWMP session instead of
// waiting for the next periodic Inform.
type GPVFaultRecoverer struct {
	pool                        *pgxpool.Pool
	releaser                    RecoveryTaskReleaser
	outboxNextAttemptAtOverride *time.Time
}

func NewGPVFaultRecoverer(pool *pgxpool.Pool, releasers ...RecoveryTaskReleaser) *GPVFaultRecoverer {
	var releaser RecoveryTaskReleaser
	if len(releasers) > 0 {
		releaser = releasers[0]
	}
	return &GPVFaultRecoverer{pool: pool, releaser: releaser}
}

// Recover returns a non-nil replacement once the durable transaction commits.
// A simultaneous non-nil error means only the fast-path queue release failed;
// callers must keep the original task recovered and let the outbox retry.
func (r *GPVFaultRecoverer) Recover(ctx context.Context, original *task.Task, remaining []string) (*task.Task, error) {
	if original == nil || original.Source != task.TaskSourceParamSync {
		return nil, fmt.Errorf("durable parameter sync task is required")
	}
	if len(remaining) == 0 {
		return nil, nil
	}
	runID, err := uuid.Parse(original.SourceID)
	if err != nil {
		return nil, fmt.Errorf("parse parameter sync recovery run: %w", err)
	}
	params, err := json.Marshal(map[string]any{"names": remaining})
	if err != nil {
		return nil, fmt.Errorf("marshal parameter sync recovery paths: %w", err)
	}
	maxRetries := task.RetryBudgetCoveringExpiry(
		parameterSyncTaskExpiresIn,
		parameterSyncTaskRetryIntervalSeconds,
	)
	replacement := task.NewTask(&task.CreateTaskRequest{
		DeviceSN: original.DeviceSN, Method: "GetParameterValues", Params: params,
		Priority: original.Priority, ExpiresIn: parameterSyncTaskExpiresIn,
		MaxRetries: &maxRetries, RetryIntervalSeconds: parameterSyncTaskRetryIntervalSeconds,
		CommandKey: original.CommandKey + "-r", Source: task.TaskSourceParamSync,
		SourceID: original.SourceID, CreatorID: original.CreatorID,
		CommandIndex: original.CommandIndex, Description: original.Description,
	})

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin parameter sync 9005 recovery: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var status RunStatus
	var requestID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT status, request_id FROM parameter_sync_runs WHERE id=$1 FOR UPDATE`, runID).
		Scan(&status, &requestID); err != nil {
		return nil, fmt.Errorf("lock parameter sync 9005 recovery run: %w", err)
	}
	if !status.AcceptsRecovery() {
		return nil, fmt.Errorf("parameter sync run %s cannot accept 9005 recovery in status %s", runID, status)
	}
	if original.CreatorID != requestID.String() {
		return nil, fmt.Errorf("parameter sync recovery creator/request mismatch")
	}
	if err := insertPlannedTask(ctx, tx, replacement); err != nil {
		return nil, err
	}
	payload, err := json.Marshal(replacement)
	if err != nil {
		return nil, fmt.Errorf("marshal parameter sync recovery task: %w", err)
	}
	outboxInsert := storage.Psql.Insert("parameter_sync_outbox").
		Columns("event_type", "aggregate_type", "aggregate_id", "dedupe_key", "payload").
		Values("param_sync.task.enqueue", "task", uuid.MustParse(replacement.ID), "task:"+replacement.ID, payload)
	if r.outboxNextAttemptAtOverride != nil {
		outboxInsert = storage.Psql.Insert("parameter_sync_outbox").
			Columns("event_type", "aggregate_type", "aggregate_id", "dedupe_key", "payload", "next_attempt_at").
			Values("param_sync.task.enqueue", "task", uuid.MustParse(replacement.ID), "task:"+replacement.ID, payload, *r.outboxNextAttemptAtOverride)
	}
	query, args, err := outboxInsert.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build parameter sync recovery outbox: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return nil, fmt.Errorf("insert parameter sync recovery outbox: %w", err)
	}
	update, updateArgs, err := storage.Psql.Update("parameter_sync_runs").
		Set("expected_task_count", sq.Expr("expected_task_count + 1")).
		Set("version", sq.Expr("version + 1")).Where(sq.Eq{"id": runID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build extend parameter sync recovery count: %w", err)
	}
	if _, err := tx.Exec(ctx, update, updateArgs...); err != nil {
		return nil, fmt.Errorf("extend parameter sync recovery count: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit parameter sync 9005 recovery: %w", err)
	}
	if r.releaser != nil {
		released, releaseErr := r.releaser.ReleasePlannedTaskWithoutWake(ctx, replacement)
		if releaseErr != nil {
			return replacement, fmt.Errorf("release parameter sync recovery task: %w", releaseErr)
		}
		if !released {
			return replacement, fmt.Errorf("release parameter sync recovery task: task was not admitted")
		}
	}
	return replacement, nil
}
