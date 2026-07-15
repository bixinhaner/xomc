package paramsync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/omcgo/omcgo/internal/task"
)

const parameterSyncTaskExpiresIn = 30 * 60

type PlannedTaskDispatcher interface {
	Dispatch(ctx context.Context, run *SyncRun, plan *Plan) ([]*task.Task, error)
}

type PGTaskDispatcher struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func NewPGTaskDispatcher(pool *pgxpool.Pool) *PGTaskDispatcher {
	return &PGTaskDispatcher{pool: pool, now: func() time.Time { return time.Now().UTC() }}
}

func (d *PGTaskDispatcher) Dispatch(ctx context.Context, run *SyncRun, plan *Plan) ([]*task.Task, error) {
	if run == nil || plan == nil {
		return nil, fmt.Errorf("dispatch parameter sync plan: run and plan are required")
	}
	if err := ValidateRunTransition(run.Status, RunStatusEnqueuing); err != nil {
		return nil, err
	}
	tasks, err := buildPlannedTasks(run, plan)
	if err != nil {
		return nil, err
	}
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin parameter sync task plan: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	coverageJSON, err := json.Marshal(plan.Coverage)
	if err != nil {
		return nil, fmt.Errorf("marshal parameter sync coverage: %w", err)
	}
	startQuery, startArgs, err := storage.Psql.Update("parameter_sync_runs").
		Set("status", RunStatusEnqueuing).Set("mapping_source", plan.MappingSource).
		Set("mapping_version", plan.MappingVersion).Set("coverage", coverageJSON).Set("version", sq.Expr("version + 1")).
		Where(sq.Eq{"id": run.ID, "status": RunStatusPlanning}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build begin parameter sync enqueue: %w", err)
	}
	tag, err := tx.Exec(ctx, startQuery, startArgs...)
	if err != nil {
		return nil, fmt.Errorf("begin parameter sync enqueue: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return nil, fmt.Errorf("begin parameter sync enqueue: run is not planning")
	}

	for _, planned := range tasks {
		if err := insertPlannedTask(ctx, tx, planned); err != nil {
			return nil, err
		}
	}

	// Persist an enqueue outbox item for every batch up front. Redis preserves
	// task priority/order and ACS still sends only one RPC at a time, so the CPE
	// can consume the complete run in one CWMP session without a Connection
	// Request between batches. Cancellation outboxes evict any remaining queued
	// tasks if a non-tolerated failure ends the run.
	for _, planned := range initialReleasePlan(tasks) {
		payload, err := json.Marshal(planned)
		if err != nil {
			return nil, fmt.Errorf("marshal parameter sync task outbox: %w", err)
		}
		outboxQuery, outboxArgs, err := storage.Psql.Insert("parameter_sync_outbox").
			Columns("event_type", "aggregate_type", "aggregate_id", "dedupe_key", "payload").
			Values("param_sync.task.enqueue", "task", uuid.MustParse(planned.ID), "task:"+planned.ID, payload).
			ToSql()
		if err != nil {
			return nil, fmt.Errorf("build parameter sync task outbox: %w", err)
		}
		if _, err := tx.Exec(ctx, outboxQuery, outboxArgs...); err != nil {
			return nil, fmt.Errorf("insert parameter sync task outbox: %w", err)
		}
	}

	finalStatus := RunStatusWaitingDevice
	if len(tasks) == 0 {
		finalStatus = RunStatusProcessing
	}
	if err := ValidateRunTransition(RunStatusEnqueuing, finalStatus); err != nil {
		return nil, err
	}
	finishQuery, finishArgs, err := storage.Psql.Update("parameter_sync_runs").
		Set("expected_task_count", len(tasks)).Set("status", finalStatus).
		Set("version", sq.Expr("version + 1")).Where(sq.Eq{"id": run.ID, "status": RunStatusEnqueuing}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build finish parameter sync enqueue: %w", err)
	}
	if _, err := tx.Exec(ctx, finishQuery, finishArgs...); err != nil {
		return nil, fmt.Errorf("finish parameter sync enqueue: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit parameter sync task plan: %w", err)
	}
	run.MappingSource, run.MappingVersion = plan.MappingSource, plan.MappingVersion
	run.Coverage = append([]CoverageScope(nil), plan.Coverage...)
	run.ExpectedTaskCount, run.Status = len(tasks), finalStatus
	return tasks, nil
}

func buildPlannedTasks(run *SyncRun, plan *Plan) ([]*task.Task, error) {
	tasks := make([]*task.Task, 0, len(plan.Batches))
	for i, batch := range plan.Batches {
		params, err := json.Marshal(map[string]any{"names": batch.Paths})
		if err != nil {
			return nil, fmt.Errorf("marshal GPV batch %d: %w", i, err)
		}
		priority := plan.Priority
		if priority <= 0 {
			priority = 10
		}
		planned := task.NewTask(&task.CreateTaskRequest{
			DeviceSN: run.DeviceSN, Method: "GetParameterValues", Params: params, Priority: priority + i,
			ExpiresIn: parameterSyncTaskExpiresIn, CommandKey: fmt.Sprintf("param-sync-%s-%d", run.ID, i),
			Source: task.TaskSourceParamSync, SourceID: run.ID.String(), CreatorID: run.RequestID.String(),
			CommandIndex: i, Description: plannedTaskDescription(plan, i),
		})
		tasks = append(tasks, planned)
	}
	return tasks, nil
}

func plannedTaskDescription(plan *Plan, _ int) string {
	if plan != nil && plan.TaskDescription != "" {
		return plan.TaskDescription
	}
	return "parameter synchronization"
}

func initialReleasePlan(tasks []*task.Task) []*task.Task {
	return tasks
}

func insertPlannedTask(ctx context.Context, tx interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, planned *task.Task) error {
	query, args, err := storage.Psql.Insert("device_tasks").Columns(
		"id", "device_sn", "method", "params", "priority", "command_key", "status",
		"retry_count", "max_retries", "retry_interval_seconds", "created_at", "expires_at",
		"source", "creator_id", "description", "source_id", "command_index", "device_index",
		"has_path_translation_miss", "path_translation_miss_count",
	).Values(
		planned.ID, planned.DeviceSN, planned.Method, planned.Params, planned.Priority, planned.CommandKey, planned.Status,
		planned.RetryCount, planned.MaxRetries, planned.RetryIntervalSeconds, planned.CreatedAt, planned.ExpiresAt,
		planned.Source, planned.CreatorID, planned.Description, planned.SourceID, planned.CommandIndex, planned.DeviceIndex,
		planned.HasPathTranslationMiss, planned.PathTranslationMissCount,
	).ToSql()
	if err != nil {
		return fmt.Errorf("build planned parameter sync task: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert planned parameter sync task: %w", err)
	}
	return nil
}
