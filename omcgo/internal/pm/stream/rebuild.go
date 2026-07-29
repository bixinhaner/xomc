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
)

type RebuildRepository struct {
	pool *pgxpool.Pool
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
  END`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build enqueue PM aggregation rebuild: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("enqueue PM aggregation rebuild: %w", err)
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
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	query, args, err := storage.Psql.Select(
		"id", "task_id", "task_version_id", "entity_key", "granularity",
		"window_start", "window_end", "source_event_id",
		"attempts", "request_generation",
	).From("pm_aggregation_rebuilds").
		Where(sq.Or{
			sq.And{
				sq.Eq{"status": []string{"pending", "failed"}},
				sq.LtOrEq{"next_attempt_at": time.Now().UTC()},
			},
			sq.And{
				sq.Eq{"status": "running"},
				sq.Lt{"lease_expires_at": time.Now().UTC()},
			},
		}).
		OrderBy("next_attempt_at", "requested_at", "id").
		Limit(1).
		Suffix("FOR UPDATE SKIP LOCKED").
		ToSql()
	if err != nil {
		return nil, err
	}
	var job RebuildJob
	job.LeaseOwner = uuid.New()
	if err := tx.QueryRow(ctx, query, args...).Scan(
		&job.ID, &job.Key.TaskID, &job.Key.TaskVersionID, &job.Key.EntityKey,
		&job.Key.Granularity, &job.Key.Start, &job.Key.End, &job.SourceID,
		&job.Attempts,
		&job.RequestGeneration,
	); err != nil {
		return nil, err
	}
	updateSQL, updateArgs, err := storage.Psql.Update("pm_aggregation_rebuilds").
		Set("status", "running").
		Set("attempts", sq.Expr("attempts + 1")).
		Set("started_at", time.Now().UTC()).
		Set("lease_expires_at", time.Now().UTC().Add(rebuildLeaseDuration)).
		Set("lease_owner", job.LeaseOwner).
		Set("completed_at", nil).
		Set("last_error", nil).
		Where(sq.Eq{"id": job.ID}).
		ToSql()
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, updateSQL, updateArgs...); err != nil {
		return nil, err
	}
	job.Attempts++
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return &job, nil
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

func (r *RebuildRepository) finish(ctx context.Context, job RebuildJob, rebuildErr error) error {
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
		return err
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("PM aggregation rebuild lease ownership lost before finish")
	}
	return nil
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
	job, err := r.repo.claimNext(ctx)
	if err != nil {
		return err
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
				if err := r.repo.renewLease(heartbeatCtx, *job); err != nil {
					heartbeatDone <- err
					return
				}
			}
		}
	}()
	rebuildErr := r.rebuild(ctx, *job)
	if rebuildErr == nil {
		rebuildErr = r.repo.enqueuePublishedParents(
			ctx, *job, r.recovery.snapshot.Current(),
		)
	}
	stopHeartbeat()
	if heartbeatErr := <-heartbeatDone; heartbeatErr != nil {
		rebuildErr = errors.Join(rebuildErr, heartbeatErr)
	}
	if finishErr := r.repo.finish(ctx, *job, rebuildErr); finishErr != nil {
		return errors.Join(rebuildErr, finishErr)
	}
	if rebuildErr != nil {
		if r.metrics != nil {
			r.metrics.RebuildErrorsTotal.Inc()
		}
		return rebuildErr
	}
	if r.metrics != nil {
		r.metrics.RebuildsTotal.WithLabelValues(string(job.Key.Granularity)).Inc()
	}
	return nil
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
	state, err := r.store.Read(ctx, job.Key)
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
		return r.repo.reopenAfterReplay(ctx, job.Key)
	}
	reason := CloseTimeout
	if state.ExpectedSlots > 0 && state.ReceivedSlots >= state.ExpectedSlots {
		reason = CloseComplete
	}
	return r.finalizer.finalizeUnderLock(ctx, job.Key, reason)
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
