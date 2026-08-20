package adhoc

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/storage"
	"go.uber.org/zap"
)

type StreamingLifecycleReconcileResult struct {
	Candidates int
	Reconciled int
	Retired    int
	Failed     int
}

type StreamingLifecycleReconciler struct {
	repo     *PgRepository
	interval time.Duration
	limit    int
	logger   *zap.Logger
}

func NewStreamingLifecycleReconciler(
	repo *PgRepository,
	interval time.Duration,
	logger *zap.Logger,
) *StreamingLifecycleReconciler {
	if interval <= 0 {
		interval = time.Minute
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &StreamingLifecycleReconciler{
		repo: repo, interval: interval, limit: 200,
		logger: logger.Named("pm.streaming-lifecycle-reconciler"),
	}
}

func (r *StreamingLifecycleReconciler) Run(ctx context.Context) {
	r.runAndLog(ctx)
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.runAndLog(ctx)
		}
	}
}

func (r *StreamingLifecycleReconciler) runAndLog(ctx context.Context) {
	result, err := r.Reconcile(ctx)
	if err != nil && !errors.Is(err, context.Canceled) {
		r.logger.Warn("reconcile PM streaming task lifecycle",
			zap.Int("candidates", result.Candidates),
			zap.Int("reconciled", result.Reconciled),
			zap.Int("retired", result.Retired),
			zap.Int("failed", result.Failed),
			zap.Error(err))
		return
	}
	if result.Reconciled > 0 || result.Retired > 0 {
		r.logger.Info("PM streaming task lifecycle reconciled",
			zap.Int("candidates", result.Candidates),
			zap.Int("reconciled", result.Reconciled),
			zap.Int("retired", result.Retired))
	}
}

func (r *StreamingLifecycleReconciler) Reconcile(
	ctx context.Context,
) (StreamingLifecycleReconcileResult, error) {
	var result StreamingLifecycleReconcileResult
	if r == nil || r.repo == nil || r.repo.streamRepo == nil {
		return result, errors.New("PM streaming lifecycle reconciler dependencies are missing")
	}
	now := time.Now().UTC()
	candidateIDs, err := r.repo.listStreamingLifecycleCandidateIDs(ctx, now, r.limit)
	if err != nil {
		return result, err
	}
	result.Candidates = len(candidateIDs)
	var reconcileErrors []error
	for _, id := range candidateIDs {
		task, loadErr := r.repo.Get(ctx, id)
		if loadErr != nil {
			result.Failed++
			reconcileErrors = append(reconcileErrors, loadErr)
			continue
		}
		if syncErr := r.repo.syncStreamingTask(
			ctx, task, streamingTaskShouldBeEnabled(task, now),
		); syncErr != nil {
			result.Failed++
			reconcileErrors = append(reconcileErrors, fmt.Errorf(
				"reconcile PM streaming source task %s: %w", id, syncErr,
			))
			continue
		}
		result.Reconciled++
	}
	retired, retireErr := r.repo.streamRepo.RetireMissingSourceTasks(
		ctx, TaskSubtype, string(ModeContinuous), uint64(r.limit),
	)
	result.Retired = retired
	if retireErr != nil {
		result.Failed++
		reconcileErrors = append(reconcileErrors, retireErr)
	}
	return result, errors.Join(reconcileErrors...)
}

func streamingTaskShouldBeEnabled(task *Task, now time.Time) bool {
	if task == nil || task.Status == StatusCanceled {
		return false
	}
	return task.PlannedEndAt == nil || now.Before(*task.PlannedEndAt)
}

func (r *PgRepository) listStreamingLifecycleCandidateIDs(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]uuid.UUID, error) {
	if limit <= 0 {
		limit = 200
	}
	query, args, err := buildStreamingLifecycleCandidateIDsSQL(now, limit)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query PM streaming lifecycle candidates: %w", err)
	}
	defer rows.Close()
	ids := make([]uuid.UUID, 0, limit)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan PM streaming lifecycle candidate: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PM streaming lifecycle candidates: %w", err)
	}
	return ids, nil
}

func buildStreamingLifecycleCandidateIDsSQL(
	now time.Time,
	limit int,
) (string, []interface{}, error) {
	desiredEnabled := `(t.status <> 'canceled'
AND (t.planned_end_at IS NULL OR t.planned_end_at > ?))`
	query, args, err := storage.Psql.Select("t.id").
		From("pm_tasks t").
		LeftJoin("pm_aggregation_tasks streaming ON streaming.id = t.id").
		Where(sq.Eq{
			"t.task_subtype": TaskSubtype,
			"t.mode":         string(ModeContinuous),
			"t.is_builtin":   false,
		}).
		Where(
			"(streaming.id IS NULL OR streaming.source_updated_at IS NULL OR "+
				"streaming.enabled IS DISTINCT FROM "+desiredEnabled+" OR "+
				"(streaming.deleted_at IS NOT NULL AND "+desiredEnabled+"))",
			now, now,
		).
		OrderBy("t.updated_at", "t.id").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("build PM streaming lifecycle candidates SQL: %w", err)
	}
	return query, args, nil
}
