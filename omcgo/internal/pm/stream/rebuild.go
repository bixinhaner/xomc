package stream

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
	"go.uber.org/zap"
)

type RebuildJob struct {
	ID                int64
	Key               WindowKey
	SourceID          string
	LeaseOwner        uuid.UUID
	Attempts          int
	RequestGeneration int64
}

type RebuildWindowState struct {
	PreviousReceived int64
	WasPublished     bool
}

const (
	rebuildLeaseDuration = 2 * time.Minute
	rebuildLockDuration  = 15 * time.Minute
	rebuildQuietPeriod   = 2 * time.Minute
	rebuildBatchSize     = 100
)

func rebuildClaimBatchSelect(quietPeriod time.Duration, limit uint64) sq.SelectBuilder {
	if quietPeriod < 0 {
		quietPeriod = 0
	}
	if limit == 0 {
		limit = 1
	}
	return storage.Psql.Select(
		"id", "task_id", "task_version_id", "entity_key", "granularity",
		"window_start", "window_end", "source_event_id",
		"attempts", "request_generation",
	).From("pm_aggregation_rebuilds").
		Where(sq.Expr(
			"requested_at <= now() - (? * interval '1 microsecond')",
			quietPeriod.Microseconds(),
		)).
		Where(sq.Or{
			sq.And{
				sq.Eq{"status": []string{"pending", "failed"}},
				sq.Expr("next_attempt_at <= now()"),
			},
			sq.And{
				sq.Eq{"status": "running"},
				sq.Expr("lease_expires_at < now()"),
			},
		}).
		OrderBy("next_attempt_at", "requested_at", "id").
		Limit(limit).
		Suffix("FOR UPDATE SKIP LOCKED")
}

func rebuildCompletionStatus(claimedGeneration, currentGeneration int64) string {
	if currentGeneration > claimedGeneration {
		return "pending"
	}
	return "completed"
}

func rebuildGenerationStable(job RebuildJob, currentGeneration int64) bool {
	return rebuildCompletionStatus(job.RequestGeneration, currentGeneration) == "completed"
}

type RebuildRepository struct {
	pool    *pgxpool.Pool
	metrics *Metrics
}

func NewRebuildRepository(pool *pgxpool.Pool) *RebuildRepository {
	return &RebuildRepository{pool: pool}
}

func (r *RebuildRepository) Enqueue(
	ctx context.Context,
	key WindowKey,
	sourceEventID string,
) error {
	query, args, err := storage.Psql.Insert("pm_aggregation_rebuilds").
		Columns(
			"task_id", "task_version_id", "entity_key", "granularity",
			"window_start", "window_end", "source_event_id",
		).
		Values(
			key.TaskID, key.TaskVersionID, key.EntityKey, string(key.Granularity),
			key.Start, key.End, sourceEventID,
		).
		Suffix(`
ON CONFLICT (task_version_id, entity_key, granularity, window_start) DO UPDATE SET
  source_event_id = EXCLUDED.source_event_id,
  window_end = EXCLUDED.window_end,
  requested_at = now(),
  request_generation = pm_aggregation_rebuilds.request_generation + 1,
  next_attempt_at = now(),
  completed_at = NULL,
  status = CASE
    WHEN pm_aggregation_rebuilds.status = 'completed' THEN 'pending'
    ELSE pm_aggregation_rebuilds.status
  END
RETURNING request_generation`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build enqueue PM aggregation rebuild: %w", err)
	}
	var generation int64
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&generation); err != nil {
		return fmt.Errorf("enqueue PM aggregation rebuild: %w", err)
	}
	if generation > 1 && r.metrics != nil {
		r.metrics.RebuildCoalescedTotal.Inc()
	}
	markSQL, markArgs, err := storage.Psql.Update("pm_aggregation_windows").
		Set("rebuild_requested_at", time.Now().UTC()).
		Where(windowKeyPredicate(key)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build mark PM aggregation rebuild requested: %w", err)
	}
	if _, err := r.pool.Exec(ctx, markSQL, markArgs...); err != nil {
		return fmt.Errorf("mark PM aggregation rebuild requested: %w", err)
	}
	return nil
}

func (r *RebuildRepository) claimNext(ctx context.Context) (*RebuildJob, error) {
	jobs, err := r.ClaimRebuildBatch(ctx, 0, 1)
	if err != nil {
		return nil, err
	}
	return &jobs[0], nil
}

// ClaimRebuildBatch leases jobs that have received no new request during the
// supplied quiet period. Eligibility and lease timestamps use the database
// clock so workers cannot disagree because of host clock skew.
func (r *RebuildRepository) ClaimRebuildBatch(
	ctx context.Context,
	quietBefore time.Duration,
	limit uint64,
) ([]RebuildJob, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin claim PM aggregation rebuild batch: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	query, args, err := rebuildClaimBatchSelect(quietBefore, limit).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build claim PM aggregation rebuild batch: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query claim PM aggregation rebuild batch: %w", err)
	}
	jobs := make([]RebuildJob, 0, limit)
	for rows.Next() {
		var job RebuildJob
		job.LeaseOwner = uuid.New()
		if err := rows.Scan(
			&job.ID, &job.Key.TaskID, &job.Key.TaskVersionID, &job.Key.EntityKey,
			&job.Key.Granularity, &job.Key.Start, &job.Key.End, &job.SourceID,
			&job.Attempts, &job.RequestGeneration,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan claim PM aggregation rebuild batch: %w", err)
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate claim PM aggregation rebuild batch: %w", err)
	}
	rows.Close()
	if len(jobs) == 0 {
		return nil, pgx.ErrNoRows
	}
	for index := range jobs {
		job := &jobs[index]
		updateSQL, updateArgs, err := storage.Psql.Update("pm_aggregation_rebuilds").
			Set("status", "running").
			Set("attempts", sq.Expr("attempts + 1")).
			Set("started_at", sq.Expr("now()")).
			Set("lease_expires_at", sq.Expr(
				"now() + (? * interval '1 microsecond')",
				rebuildLeaseDuration.Microseconds(),
			)).
			Set("lease_owner", job.LeaseOwner).
			Set("completed_at", nil).
			Set("last_error", nil).
			Where(sq.Eq{"id": job.ID}).
			ToSql()
		if err != nil {
			return nil, fmt.Errorf("build lease PM aggregation rebuild batch: %w", err)
		}
		if _, err := tx.Exec(ctx, updateSQL, updateArgs...); err != nil {
			return nil, fmt.Errorf("lease PM aggregation rebuild batch: %w", err)
		}
		job.Attempts++
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit claim PM aggregation rebuild batch: %w", err)
	}
	return jobs, nil
}

func (r *RebuildRepository) renewLease(ctx context.Context, job RebuildJob) error {
	query, args, err := storage.Psql.Update("pm_aggregation_rebuilds").
		Set("lease_expires_at", time.Now().UTC().Add(rebuildLeaseDuration)).
		Where(sq.Eq{
			"id": job.ID, "status": "running", "lease_owner": job.LeaseOwner,
		}).
		ToSql()
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("PM aggregation rebuild lease ownership lost")
	}
	return nil
}

func (r *RebuildRepository) resetWindow(ctx context.Context, key WindowKey) (RebuildWindowState, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return RebuildWindowState{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	selectSQL, selectArgs, err := storage.Psql.Select("received_slots", "published_at IS NOT NULL").
		From("pm_aggregation_windows").
		Where(windowKeyPredicate(key)).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return RebuildWindowState{}, err
	}
	var previous RebuildWindowState
	if err := tx.QueryRow(ctx, selectSQL, selectArgs...).Scan(
		&previous.PreviousReceived, &previous.WasPublished,
	); err != nil {
		return RebuildWindowState{}, err
	}
	updateSQL, updateArgs, err := storage.Psql.Update("pm_aggregation_windows").
		Set("status", "rebuilding").
		Set("revision", sq.Expr("CASE WHEN status = 'published' THEN revision + 1 ELSE revision END")).
		Set("close_reason", nil).
		Set("last_error", nil).
		Set("updated_at", time.Now().UTC()).
		Where(windowKeyPredicate(key)).
		Where(sq.Eq{"status": []string{"open", "published", "rebuilding", "failed"}}).
		ToSql()
	if err != nil {
		return RebuildWindowState{}, err
	}
	tag, err := tx.Exec(ctx, updateSQL, updateArgs...)
	if err != nil {
		return RebuildWindowState{}, err
	}
	if tag.RowsAffected() == 0 {
		return RebuildWindowState{}, fmt.Errorf("PM aggregation window not found for rebuild")
	}
	if err := tx.Commit(ctx); err != nil {
		return RebuildWindowState{}, err
	}
	return previous, nil
}

func (r *RebuildRepository) reopenAfterReplay(ctx context.Context, key WindowKey) error {
	query, args, err := storage.Psql.Update("pm_aggregation_windows").
		Set("status", "open").
		Set("last_error", nil).
		Set("updated_at", time.Now().UTC()).
		Where(windowKeyPredicate(key)).
		Where(sq.Eq{"status": "rebuilding"}).
		ToSql()
	if err != nil {
		return err
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("PM aggregation rebuild window could not return to open")
	}
	return nil
}

func (r *RebuildRepository) finish(
	ctx context.Context,
	job RebuildJob,
	rebuildErr error,
) (bool, error) {
	builder := storage.Psql.Update("pm_aggregation_rebuilds").
		Where(sq.Eq{"id": job.ID, "lease_owner": job.LeaseOwner})
	if rebuildErr == nil {
		completedAt := time.Now().UTC()
		builder = builder.
			Set("status", sq.Expr(
				"CASE WHEN request_generation > ? THEN 'pending' ELSE 'completed' END",
				job.RequestGeneration,
			)).
			Set("completed_at", rebuildCompletedAtExpr(
				job.RequestGeneration, completedAt,
			)).
			Set("lease_expires_at", nil).
			Set("lease_owner", nil).
			Set("next_attempt_at", sq.Expr(
				"CASE WHEN request_generation > ? THEN ? ELSE next_attempt_at END",
				job.RequestGeneration, time.Now().UTC(),
			)).
			Set("last_error", nil)
	} else {
		builder = builder.Set("status", "failed").
			Set("completed_at", nil).
			Set("lease_expires_at", nil).
			Set("lease_owner", nil).
			Set("next_attempt_at", time.Now().UTC().Add(rebuildRetryDelay(job.Attempts))).
			Set("last_error", rebuildErr.Error())
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return false, err
	}
	var currentGeneration int64
	if err := r.pool.QueryRow(
		ctx, query+" RETURNING request_generation", args...,
	).Scan(&currentGeneration); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, fmt.Errorf("PM aggregation rebuild lease ownership lost before finish")
		}
		return false, fmt.Errorf("finish PM aggregation rebuild: %w", err)
	}
	if rebuildErr != nil {
		return false, nil
	}
	return rebuildGenerationStable(job, currentGeneration), nil
}

func rebuildCompletedAtExpr(
	requestGeneration int64,
	completedAt time.Time,
) sq.Sqlizer {
	return sq.Expr(
		"CASE WHEN request_generation > ? "+
			"THEN NULL::timestamptz ELSE ?::timestamptz END",
		requestGeneration, completedAt,
	)
}

func RebuildCascadeTargets(key WindowKey) []Granularity {
	switch key.Granularity {
	case GranularityHourly:
		return []Granularity{GranularityDaily}
	case GranularityDaily:
		return []Granularity{GranularityWeekly, GranularityMonthly}
	default:
		return nil
	}
}

func rebuildParentLineage(
	key WindowKey,
	snapshot *TaskSnapshot,
) (uuid.UUID, uuid.UUID) {
	taskID, versionID := key.TaskID, key.TaskVersionID
	if snapshot == nil {
		return taskID, versionID
	}
	version := snapshot.ByVersion[key.TaskVersionID]
	if version == nil || !version.DevicePipeline || version.RollupVersionID == uuid.Nil {
		return taskID, versionID
	}
	if version.TaskID != uuid.Nil {
		taskID = version.TaskID
	}
	return taskID, version.RollupVersionID
}

func publishedParentsSelect(
	job RebuildJob,
	target Granularity,
	parentTaskID uuid.UUID,
	parentVersionID uuid.UUID,
	sourceID string,
) sq.SelectBuilder {
	return storage.Psql.Select(
		"task_id", "task_version_id", "entity_key", "granularity",
		"window_start", "window_end",
	).From("pm_aggregation_windows").
		Column("?", sourceID).
		Where(sq.Eq{
			"status":          []string{"open", "published", "rebuilding", "finalizing", "failed"},
			"task_id":         parentTaskID,
			"task_version_id": parentVersionID,
			"entity_key":      job.Key.EntityKey,
			"granularity":     string(target),
		}).
		Where(sq.Lt{"window_start": job.Key.End}).
		Where(sq.Gt{"window_end": job.Key.Start})
}

func (r *RebuildRepository) enqueuePublishedParents(
	ctx context.Context,
	job RebuildJob,
	snapshot *TaskSnapshot,
) error {
	parentTaskID, parentVersionID := rebuildParentLineage(job.Key, snapshot)
	for _, target := range RebuildCascadeTargets(job.Key) {
		sourceID := fmt.Sprintf("cascade:%d:%s", job.ID, target)
		selectBuilder := publishedParentsSelect(
			job, target, parentTaskID, parentVersionID, sourceID,
		)
		query, args, err := storage.Psql.Insert("pm_aggregation_rebuilds").
			Columns(
				"task_id", "task_version_id", "entity_key", "granularity",
				"window_start", "window_end", "source_event_id",
			).
			Select(selectBuilder).
			Suffix(`
ON CONFLICT (task_version_id, entity_key, granularity, window_start) DO UPDATE SET
  source_event_id = EXCLUDED.source_event_id,
  window_end = EXCLUDED.window_end,
  requested_at = now(),
  request_generation = pm_aggregation_rebuilds.request_generation + 1,
  next_attempt_at = now(),
  completed_at = NULL,
  status = CASE
    WHEN pm_aggregation_rebuilds.status = 'completed' THEN 'pending'
    ELSE pm_aggregation_rebuilds.status
  END`).
			ToSql()
		if err != nil {
			return err
		}
		if _, err := r.pool.Exec(ctx, query, args...); err != nil {
			return err
		}
	}
	return nil
}

func rebuildRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := time.Second << min(attempt, 9)
	if delay > 5*time.Minute {
		return 5 * time.Minute
	}
	return delay
}

type Rebuilder struct {
	repo      *RebuildRepository
	recovery  *Recovery
	finalizer *Finalizer
	store     *RedisWindowStore
	rollups   *RollupOutboxRepository
	logger    *zap.Logger
	metrics   *Metrics
}

func NewRebuilder(
	repo *RebuildRepository,
	recovery *Recovery,
	finalizer *Finalizer,
	store *RedisWindowStore,
	rollups *RollupOutboxRepository,
	logger *zap.Logger,
) *Rebuilder {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Rebuilder{
		repo: repo, recovery: recovery, finalizer: finalizer,
		store: store, rollups: rollups, logger: logger,
	}
}

func (r *Rebuilder) SetMetrics(metrics *Metrics) *Rebuilder {
	r.metrics = metrics
	r.repo.metrics = metrics
	return r
}

func (r *Rebuilder) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		if err := r.runOnce(ctx); err != nil && !errors.Is(err, pgx.ErrNoRows) && ctx.Err() == nil {
			r.logger.Warn("rebuild late PM aggregation window", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (r *Rebuilder) runOnce(ctx context.Context) error {
	jobs, err := r.repo.ClaimRebuildBatch(
		ctx, rebuildQuietPeriod, rebuildBatchSize,
	)
	if err != nil {
		return err
	}
	if r.metrics != nil {
		r.metrics.RebuildBatchesTotal.Inc()
		r.metrics.RebuildJobsPerBatch.Observe(float64(len(jobs)))
	}
	heartbeatCtx, stopHeartbeat := context.WithCancel(ctx)
	heartbeatDone := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(rebuildLeaseDuration / 3)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatCtx.Done():
				heartbeatDone <- nil
				return
			case <-ticker.C:
				for _, job := range jobs {
					if err := r.repo.renewLease(heartbeatCtx, job); err != nil {
						heartbeatDone <- err
						return
					}
				}
			}
		}
	}()
	results := r.rebuildClaimedBatch(ctx, jobs)
	stopHeartbeat()
	heartbeatErr := <-heartbeatDone
	var batchErr error
	for _, job := range jobs {
		rebuildErr := results[job.ID]
		if heartbeatErr != nil {
			rebuildErr = errors.Join(rebuildErr, heartbeatErr)
		}
		stable, finishErr := r.repo.finish(ctx, job, rebuildErr)
		rebuildErr = errors.Join(rebuildErr, finishErr)
		if rebuildErr == nil && stable {
			if err := r.repo.enqueuePublishedParents(
				ctx, job, r.recovery.snapshot.Current(),
			); err != nil {
				// The child is already durable. Requeue it so a transient
				// parent-cascade failure cannot permanently lose propagation.
				rebuildErr = err
				if enqueueErr := r.repo.Enqueue(
					ctx, job.Key, "cascade-retry:"+job.SourceID,
				); enqueueErr != nil {
					rebuildErr = errors.Join(rebuildErr, enqueueErr)
				}
			}
		}
		if rebuildErr != nil {
			if r.metrics != nil {
				r.metrics.RebuildErrorsTotal.Inc()
			}
			batchErr = errors.Join(batchErr, rebuildErr)
			continue
		}
		if r.metrics != nil {
			r.metrics.RebuildsTotal.WithLabelValues(
				string(job.Key.Granularity),
			).Inc()
		}
	}
	return batchErr
}

func (r *Rebuilder) rebuildClaimedBatch(
	ctx context.Context,
	jobs []RebuildJob,
) map[int64]error {
	results := make(map[int64]error, len(jobs))
	snapshot := r.recovery.snapshot.Current()
	groups, ungrouped := groupRollupRebuildJobs(jobs, snapshot)
	for _, job := range ungrouped {
		results[job.ID] = r.rebuild(ctx, job)
	}
	for _, group := range groups {
		for id, err := range r.rebuildRollupGroup(ctx, group, snapshot) {
			results[id] = err
		}
	}
	return results
}

type preparedRebuild struct {
	job      RebuildJob
	lock     *Lock
	previous RebuildWindowState
	matched  bool
	err      error
}

func (r *Rebuilder) rebuildRollupGroup(
	ctx context.Context,
	group rollupRebuildGroup,
	snapshot *TaskSnapshot,
) map[int64]error {
	results := make(map[int64]error, len(group.Jobs))
	prepared := make([]*preparedRebuild, 0, len(group.Jobs))
	byWindow := make(map[string]*preparedRebuild, len(group.Jobs))
	for _, job := range group.Jobs {
		item := &preparedRebuild{job: job}
		lock, err := r.store.TryFinalizeLock(ctx, job.Key, rebuildLockDuration)
		if err != nil {
			results[job.ID] = err
			continue
		}
		if lock == nil {
			results[job.ID] = fmt.Errorf("PM aggregation rebuild window is busy")
			continue
		}
		item.lock = lock
		previous, err := r.repo.resetWindow(ctx, job.Key)
		if err == nil {
			err = r.store.DeleteState(ctx, job.Key)
		}
		if err != nil {
			results[job.ID] = err
			if releaseErr := lock.Release(context.Background()); releaseErr != nil {
				results[job.ID] = errors.Join(results[job.ID], releaseErr)
			}
			continue
		}
		item.previous = previous
		prepared = append(prepared, item)
		byWindow[rebuildWindowIdentity(job.Key)] = item
	}
	defer func() {
		for _, item := range prepared {
			if err := item.lock.Release(context.Background()); err != nil {
				r.logger.Warn("release batched PM aggregation rebuild lock",
					zap.Error(err))
			}
		}
	}()
	if len(prepared) == 0 {
		return results
	}
	workCtx, cancelWork := context.WithCancel(ctx)
	lockHeartbeatDone := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(rebuildLockDuration / 3)
		defer ticker.Stop()
		for {
			select {
			case <-workCtx.Done():
				lockHeartbeatDone <- nil
				return
			case <-ticker.C:
				for _, item := range prepared {
					if err := item.lock.Extend(workCtx, rebuildLockDuration); err != nil {
						lockHeartbeatDone <- err
						cancelWork()
						return
					}
				}
			}
		}
	}()
	scanStarted := time.Now()
	rows, scanErr := visitSnapshotsForRebuildBatch(
		workCtx,
		r.rollups,
		group.SourceVersionIDs,
		group.SourceGranularity,
		group.Start,
		group.End,
		func(payload RollupPayload) error {
			contributions, err := rebuildBatchContributions(
				payload, snapshot, group.TargetGranularity,
				r.recovery.matcher.location,
			)
			if err != nil {
				return err
			}
			for _, contribution := range contributions {
				item := byWindow[rebuildWindowIdentity(contribution.Key)]
				if item == nil || item.err != nil {
					continue
				}
				if _, err := r.store.accumulateWithLock(
					workCtx, contribution, item.lock,
				); err != nil {
					item.err = err
					continue
				}
				item.matched = true
			}
			return nil
		},
	)
	if r.metrics != nil {
		r.metrics.RebuildSnapshotRowsTotal.Add(float64(rows))
		r.metrics.RebuildSnapshotScanSeconds.Observe(
			time.Since(scanStarted).Seconds(),
		)
	}
	for _, item := range prepared {
		if scanErr != nil {
			item.err = errors.Join(item.err, scanErr)
		}
		if item.err == nil && !item.matched {
			item.err = fmt.Errorf("no durable rollup source matched PM rebuild window")
		}
		if item.err == nil {
			item.err = r.completeRebuildUnderLock(
				workCtx, item.job.Key, item.previous,
			)
		}
		results[item.job.ID] = item.err
	}
	cancelWork()
	if heartbeatErr := <-lockHeartbeatDone; heartbeatErr != nil {
		for _, item := range prepared {
			results[item.job.ID] = errors.Join(
				results[item.job.ID], heartbeatErr,
			)
		}
	}
	return results
}

func rebuildWindowIdentity(key WindowKey) string {
	return fmt.Sprintf(
		"%s|%s|%s|%s|%d|%d",
		key.TaskID,
		key.TaskVersionID,
		key.EntityKey,
		key.Granularity,
		key.Start.UTC().UnixNano(),
		key.End.UTC().UnixNano(),
	)
}

func rebuildBatchContributions(
	payload RollupPayload,
	snapshot *TaskSnapshot,
	target Granularity,
	location *time.Location,
) ([]Contribution, error) {
	if target == GranularityHourly {
		if !isDeviceHourPayload(payload, snapshot) {
			return nil, nil
		}
		return matchDeviceHourRules(payload, snapshot, location)
	}
	return rollupContributions(
		payload, snapshot.ByVersion[payload.TaskVersionID], location,
	)
}

func (r *Rebuilder) completeRebuildUnderLock(
	ctx context.Context,
	key WindowKey,
	previous RebuildWindowState,
) error {
	state, err := r.store.Read(ctx, key)
	if err != nil {
		return fmt.Errorf("read rebuilt PM aggregation window: %w", err)
	}
	if state.ReceivedSlots < previous.PreviousReceived {
		return fmt.Errorf(
			"durable PM rebuild source incomplete: recovered %d slots, previously published %d",
			state.ReceivedSlots, previous.PreviousReceived,
		)
	}
	if !previous.WasPublished {
		return r.repo.reopenAfterReplay(ctx, key)
	}
	reason := CloseTimeout
	if state.ExpectedSlots > 0 && state.ReceivedSlots >= state.ExpectedSlots {
		reason = CloseComplete
	}
	return r.finalizer.finalizeUnderLock(ctx, key, reason)
}

func (r *Rebuilder) rebuild(ctx context.Context, job RebuildJob) (rebuildErr error) {
	lock, err := r.store.TryFinalizeLock(ctx, job.Key, rebuildLockDuration)
	if err != nil {
		return err
	}
	if lock == nil {
		return fmt.Errorf("PM aggregation rebuild window is busy")
	}
	lockHeartbeatCtx, stopLockHeartbeat := context.WithCancel(ctx)
	lockHeartbeatDone := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(rebuildLockDuration / 3)
		defer ticker.Stop()
		for {
			select {
			case <-lockHeartbeatCtx.Done():
				lockHeartbeatDone <- nil
				return
			case <-ticker.C:
				if err := lock.Extend(lockHeartbeatCtx, rebuildLockDuration); err != nil {
					lockHeartbeatDone <- err
					return
				}
			}
		}
	}()
	defer func() {
		stopLockHeartbeat()
		rebuildErr = errors.Join(rebuildErr, <-lockHeartbeatDone)
		if err := lock.Release(context.Background()); err != nil {
			r.logger.Warn("release PM aggregation rebuild lock", zap.Error(err))
		}
	}()
	previous, err := r.repo.resetWindow(ctx, job.Key)
	if err != nil {
		return err
	}
	if err := r.store.DeleteState(ctx, job.Key); err != nil {
		return err
	}
	if err := r.replaySources(ctx, job.Key, lock); err != nil {
		return err
	}
	return r.completeRebuildUnderLock(ctx, job.Key, previous)
}

func rebuildUsesRawSources(key WindowKey, snapshot *TaskSnapshot) bool {
	if key.Granularity != GranularityHourly || snapshot == nil {
		return false
	}
	version := snapshot.ByVersion[key.TaskVersionID]
	return version != nil && version.DevicePipeline
}

func deviceRollupVersionsForRule(
	snapshot *TaskSnapshot,
	target *TaskVersionSnapshot,
) []uuid.UUID {
	if snapshot == nil || target == nil {
		return nil
	}
	versionIDs := make([]uuid.UUID, 0, 1)
	for versionID, version := range snapshot.ByVersion {
		if version == nil || !version.DeviceRollup ||
			(target.Technology != "" &&
				!strings.EqualFold(target.Technology, version.Technology)) {
			continue
		}
		versionIDs = append(versionIDs, versionID)
	}
	sort.Slice(versionIDs, func(i, j int) bool {
		return versionIDs[i].String() < versionIDs[j].String()
	})
	return versionIDs
}

type rollupRebuildGroup struct {
	SourceVersionIDs  []uuid.UUID
	SourceGranularity Granularity
	TargetGranularity Granularity
	Start             time.Time
	End               time.Time
	Jobs              []RebuildJob
}

func groupRollupRebuildJobs(
	jobs []RebuildJob,
	snapshot *TaskSnapshot,
) ([]rollupRebuildGroup, []RebuildJob) {
	if snapshot == nil {
		return nil, append([]RebuildJob(nil), jobs...)
	}
	groups := make([]rollupRebuildGroup, 0)
	groupIndexes := make(map[string]int)
	var ungrouped []RebuildJob
	for _, job := range jobs {
		if rebuildUsesRawSources(job.Key, snapshot) {
			ungrouped = append(ungrouped, job)
			continue
		}
		sourceGranularity := GranularityHourly
		if job.Key.Granularity == GranularityWeekly ||
			job.Key.Granularity == GranularityMonthly {
			sourceGranularity = GranularityDaily
		}
		sourceVersionIDs := []uuid.UUID{job.Key.TaskVersionID}
		if job.Key.Granularity == GranularityHourly {
			sourceVersionIDs = deviceRollupVersionsForRule(
				snapshot, snapshot.ByVersion[job.Key.TaskVersionID],
			)
		}
		if len(sourceVersionIDs) == 0 {
			ungrouped = append(ungrouped, job)
			continue
		}
		parts := make([]string, 0, len(sourceVersionIDs))
		for _, versionID := range sourceVersionIDs {
			parts = append(parts, versionID.String())
		}
		groupKey := fmt.Sprintf(
			"%s|%s|%d|%d|%s",
			sourceGranularity,
			job.Key.Granularity,
			job.Key.Start.UTC().UnixNano(),
			job.Key.End.UTC().UnixNano(),
			strings.Join(parts, ","),
		)
		index, exists := groupIndexes[groupKey]
		if !exists {
			index = len(groups)
			groupIndexes[groupKey] = index
			groups = append(groups, rollupRebuildGroup{
				SourceVersionIDs:  sourceVersionIDs,
				SourceGranularity: sourceGranularity,
				TargetGranularity: job.Key.Granularity,
				Start:             job.Key.Start,
				End:               job.Key.End,
			})
		}
		groups[index].Jobs = append(groups[index].Jobs, job)
	}
	return groups, ungrouped
}

func (r *Rebuilder) replaySources(ctx context.Context, key WindowKey, lock *Lock) error {
	snapshot := r.recovery.snapshot.Current()
	if snapshot == nil {
		return fmt.Errorf("PM aggregation task snapshot missing during rebuild")
	}
	// Synthetic device-pipeline hours originate directly from normalized raw
	// PM events. Ordinary rule hours originate from stable device-hour rollups.
	if rebuildUsesRawSources(key, snapshot) {
		payloads, err := r.recovery.outbox.ListPayloadsForPeriod(ctx, key.Start, key.End)
		if err != nil {
			return err
		}
		matched := false
		for _, payload := range payloads {
			contributions, err := r.recovery.matcher.MatchGranularity(
				payload, snapshot, GranularityHourly,
			)
			if err != nil {
				return err
			}
			for _, contribution := range contributions {
				if !sameWindowKey(contribution.Key, key) {
					continue
				}
				if _, err := r.store.accumulateWithLock(ctx, contribution, lock); err != nil {
					return err
				}
				matched = true
			}
		}
		if !matched {
			return fmt.Errorf("no durable raw PM source matched rebuild window")
		}
		return nil
	}
	sourceGranularity := GranularityHourly
	if key.Granularity == GranularityWeekly || key.Granularity == GranularityMonthly {
		sourceGranularity = GranularityDaily
	}
	sourceVersionIDs := []uuid.UUID{key.TaskVersionID}
	if key.Granularity == GranularityHourly {
		sourceVersionIDs = deviceRollupVersionsForRule(
			snapshot, snapshot.ByVersion[key.TaskVersionID],
		)
	}
	matched := false
	err := r.rollups.VisitSnapshotsForPeriod(
		ctx, sourceVersionIDs, sourceGranularity, key.Start, key.End,
		func(payload RollupPayload) error {
			var contributions []Contribution
			var matchErr error
			if key.Granularity == GranularityHourly {
				// SQL already restricts this stream to stable device-rollup
				// versions. Keep the classification guard as a contract check.
				if !isDeviceHourPayload(payload, snapshot) {
					return nil
				}
				contributions, matchErr = matchDeviceHourRuleWindow(
					payload, snapshot, key,
				)
			} else {
				contributions, matchErr = rollupContributions(
					payload,
					snapshot.ByVersion[payload.TaskVersionID],
					r.recovery.matcher.location,
				)
			}
			if matchErr != nil {
				return matchErr
			}
			for _, contribution := range contributions {
				if !sameWindowKey(contribution.Key, key) {
					continue
				}
				if _, err := r.store.accumulateWithLock(ctx, contribution, lock); err != nil {
					return err
				}
				matched = true
			}
			return nil
		},
	)
	if err != nil {
		return err
	}
	if !matched {
		return fmt.Errorf("no durable rollup source matched PM rebuild window")
	}
	return nil
}
