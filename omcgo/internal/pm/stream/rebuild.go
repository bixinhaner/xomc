package stream

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
	"github.com/redis/go-redis/v9"
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
	hourlyPublishGrace   = 12 * time.Minute
	rebuildDeadlineLead  = 30 * time.Second
	rebuildBatchSize     = 100
)

func rebuildClaimBatchSelect(quietPeriod time.Duration, limit uint64) sq.SelectBuilder {
	if quietPeriod < 0 {
		quietPeriod = 0
	}
	if limit == 0 {
		limit = 1
	}
	if limit > rebuildBatchSize {
		limit = rebuildBatchSize
	}
	return storage.Psql.Select(
		"id", "task_id", "task_version_id", "entity_key", "granularity",
		"window_start", "window_end", "source_event_id",
		"attempts", "request_generation",
	).From("pm_aggregation_rebuilds").
		Where(sq.Or{
			sq.Expr(
				"requested_at <= now() - (? * interval '1 microsecond')",
				quietPeriod.Microseconds(),
			),
			sq.And{
				sq.Eq{"granularity": string(GranularityHourly)},
				sq.Expr(
					"now() >= window_end + (? * interval '1 microsecond')",
					(hourlyPublishGrace - rebuildDeadlineLead).Microseconds(),
				),
			},
		}).
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
		OrderByClause(`CASE
  WHEN granularity = 'hourly'
   AND now() >= window_end + (`+fmt.Sprint((hourlyPublishGrace-rebuildDeadlineLead).Microseconds())+` * interval '1 microsecond')
  THEN 0 ELSE 1 END`).
		OrderByClause(`CASE
  WHEN granularity = 'hourly'
   AND now() >= window_end + (`+fmt.Sprint((hourlyPublishGrace-rebuildDeadlineLead).Microseconds())+` * interval '1 microsecond')
  THEN window_end END DESC`).
		OrderBy("next_attempt_at", "requested_at", "id").
		Limit(limit).
		Suffix("FOR UPDATE SKIP LOCKED")
}

func rebuildLeaseBatchUpdate(
	jobs []RebuildJob,
	duration time.Duration,
) sq.UpdateBuilder {
	owners := make(sq.Or, 0, len(jobs))
	for _, job := range jobs {
		owners = append(owners, sq.And{
			sq.Eq{"id": job.ID},
			sq.Eq{"lease_owner": job.LeaseOwner},
		})
	}
	return storage.Psql.Update("pm_aggregation_rebuilds").
		Set("lease_expires_at", sq.Expr(
			"now() + (? * interval '1 microsecond')",
			duration.Microseconds(),
		)).
		Where(sq.Eq{"status": "running"}).
		Where("lease_expires_at > now()").
		Where(owners)
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

type rebuildCompletionTx interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Commit(context.Context) error
	Rollback(context.Context) error
}

type rebuildCompletionBegin func(context.Context) (rebuildCompletionTx, error)

func NewRebuildRepository(pool *pgxpool.Pool) *RebuildRepository {
	return &RebuildRepository{pool: pool}
}

func lockPublicationForRebuild(ctx context.Context, tx pgx.Tx, key WindowKey) error {
	var marker int
	err := tx.QueryRow(ctx, `
SELECT 1
FROM pm_aggregation_publications
WHERE task_version_id = $1 AND granularity = $2 AND window_start = $3
FOR UPDATE`, key.TaskVersionID, string(key.Granularity), key.Start).Scan(&marker)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("lock PM publication before rebuild: %w", err)
	}
	return nil
}

func (r *RebuildRepository) Enqueue(
	ctx context.Context,
	key WindowKey,
	sourceEventID string,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin enqueue PM aggregation rebuild: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockPublicationForRebuild(ctx, tx, key); err != nil {
		return err
	}
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
	if err := tx.QueryRow(ctx, query, args...).Scan(&generation); err != nil {
		return fmt.Errorf("enqueue PM aggregation rebuild: %w", err)
	}
	markSQL, markArgs, err := rebuildMarkRequestedUpdate(key).ToSql()
	if err != nil {
		return fmt.Errorf("build mark PM aggregation rebuild requested: %w", err)
	}
	if _, err := tx.Exec(ctx, markSQL, markArgs...); err != nil {
		return fmt.Errorf("mark PM aggregation rebuild requested: %w", err)
	}
	dirtySQL, dirtyArgs, err := storage.Psql.Update("pm_aggregation_publications").
		Set("dirty_entities", sq.Expr("dirty_entities + 1")).
		Set("updated_at", time.Now().UTC()).
		Where(sq.Eq{
			"task_version_id": key.TaskVersionID,
			"granularity":     string(key.Granularity),
			"window_start":    key.Start,
		}).Where(sq.Or{
		sq.Eq{"status": "preparing"},
		sq.Expr("preparing_revision IS NOT NULL"),
	}).ToSql()
	if err != nil {
		return fmt.Errorf("build mark PM publication dirty: %w", err)
	}
	if _, err := tx.Exec(ctx, dirtySQL, dirtyArgs...); err != nil {
		return fmt.Errorf("mark PM publication dirty: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit enqueue PM aggregation rebuild: %w", err)
	}
	if generation > 1 && r.metrics != nil {
		r.metrics.RebuildCoalescedTotal.Inc()
	}
	return nil
}

func rebuildMarkRequestedUpdate(key WindowKey) sq.UpdateBuilder {
	return storage.Psql.Update("pm_aggregation_windows").
		Set("rebuild_requested_at", time.Now().UTC()).
		Set("status", sq.Expr("CASE WHEN status = 'prepared' THEN 'rebuilding' ELSE status END")).
		Set("updated_at", time.Now().UTC()).
		Where(windowKeyPredicate(key))
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

func (r *RebuildRepository) renewLeases(
	ctx context.Context,
	jobs []RebuildJob,
) error {
	if len(jobs) == 0 || len(jobs) > rebuildBatchSize {
		return fmt.Errorf(
			"PM aggregation rebuild lease batch size %d is outside 1..%d",
			len(jobs), rebuildBatchSize,
		)
	}
	query, args, err := rebuildLeaseBatchUpdate(
		jobs, rebuildLeaseDuration,
	).ToSql()
	if err != nil {
		return fmt.Errorf("build renew PM aggregation rebuild leases: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("renew PM aggregation rebuild leases: %w", err)
	}
	if tag.RowsAffected() != int64(len(jobs)) {
		return fmt.Errorf(
			"PM aggregation rebuild lease ownership lost: renewed %d of %d",
			tag.RowsAffected(), len(jobs),
		)
	}
	return nil
}

func maintainRebuildLeases(
	ctx context.Context,
	ticks <-chan time.Time,
	renew func(context.Context) error,
	cancelWork context.CancelFunc,
) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticks:
			if err := renew(ctx); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				cancelWork()
				return err
			}
		}
	}
}

type rebuildActiveLeases struct {
	mu   sync.Mutex
	jobs map[int64]RebuildJob
}

func newRebuildActiveLeases(jobs []RebuildJob) *rebuildActiveLeases {
	active := &rebuildActiveLeases{
		jobs: make(map[int64]RebuildJob, len(jobs)),
	}
	for _, job := range jobs {
		active.jobs[job.ID] = job
	}
	return active
}

func (active *rebuildActiveLeases) renew(
	ctx context.Context,
	renew func(context.Context, []RebuildJob) error,
) error {
	active.mu.Lock()
	defer active.mu.Unlock()
	if len(active.jobs) == 0 {
		return nil
	}
	jobs := make([]RebuildJob, 0, len(active.jobs))
	for _, job := range active.jobs {
		jobs = append(jobs, job)
	}
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].ID < jobs[j].ID
	})
	return renew(ctx, jobs)
}

// beginCompletion excludes one job from future batch renewals only after its
// completion transaction has acquired the row lock. Holding this guard while
// acquiring that lock prevents a concurrent renewal from observing a
// half-completed job and treating it as lost ownership.
func (active *rebuildActiveLeases) beginCompletion(jobID int64) func() {
	active.mu.Lock()
	var once sync.Once
	return func() {
		once.Do(func() {
			delete(active.jobs, jobID)
			active.mu.Unlock()
		})
	}
}

func (r *RebuildRepository) resetWindow(ctx context.Context, key WindowKey) (RebuildWindowState, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return RebuildWindowState{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockPublicationForRebuild(ctx, tx, key); err != nil {
		return RebuildWindowState{}, err
	}
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
		Set("revision", sq.Expr(`CASE WHEN status = 'published' THEN GREATEST(
  revision,
  COALESCE((
    SELECT publication.revision
    FROM pm_aggregation_publications publication
    WHERE publication.task_version_id = pm_aggregation_windows.task_version_id
      AND publication.granularity = pm_aggregation_windows.granularity
      AND publication.window_start = pm_aggregation_windows.window_start
      AND publication.status = 'published'
  ), revision)
) + 1 ELSE revision END`)).
		Set("close_reason", nil).
		Set("last_error", nil).
		Set("updated_at", time.Now().UTC()).
		Where(windowKeyPredicate(key)).
		Where(sq.Eq{"status": []string{"open", "prepared", "published", "rebuilding", "failed"}}).
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

func (r *RebuildRepository) finishAndEnqueueParents(
	ctx context.Context,
	job RebuildJob,
	snapshot *TaskSnapshot,
	rebuildErr error,
	onLocked func(),
) (bool, error) {
	return completeRebuildAtomically(
		ctx,
		func(ctx context.Context) (rebuildCompletionTx, error) {
			return r.pool.Begin(ctx)
		},
		job,
		snapshot,
		rebuildErr,
		onLocked,
	)
}

func completeRebuildAtomically(
	ctx context.Context,
	begin rebuildCompletionBegin,
	job RebuildJob,
	snapshot *TaskSnapshot,
	rebuildErr error,
	onLocked ...func(),
) (bool, error) {
	tx, err := begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin complete PM aggregation rebuild: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	generationSQL, generationArgs, err := storage.Psql.Select("request_generation").
		From("pm_aggregation_rebuilds").
		Where(sq.Eq{
			"id": job.ID, "status": "running", "lease_owner": job.LeaseOwner,
		}).
		Where("lease_expires_at > now()").
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build lock PM aggregation rebuild completion: %w", err)
	}
	var currentGeneration int64
	if err := tx.QueryRow(
		ctx, generationSQL, generationArgs...,
	).Scan(&currentGeneration); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, fmt.Errorf("PM aggregation rebuild lease ownership lost before finish")
		}
		return false, fmt.Errorf("lock PM aggregation rebuild completion: %w", err)
	}
	if len(onLocked) > 0 && onLocked[0] != nil {
		onLocked[0]()
	}
	if currentGeneration < job.RequestGeneration {
		return false, fmt.Errorf(
			"PM aggregation rebuild generation regressed from %d to %d",
			job.RequestGeneration, currentGeneration,
		)
	}
	stable := rebuildGenerationStable(job, currentGeneration)
	builder := storage.Psql.Update("pm_aggregation_rebuilds").
		Set("lease_expires_at", nil).
		Set("lease_owner", nil).
		Where(sq.Eq{
			"id": job.ID, "status": "running", "lease_owner": job.LeaseOwner,
		}).
		Where("lease_expires_at > now()")
	if !stable {
		builder = builder.
			Set("status", "pending").
			Set("completed_at", nil).
			Set("next_attempt_at", sq.Expr("now()")).
			Set("last_error", nil)
	} else if rebuildErr != nil {
		builder = builder.
			Set("status", "failed").
			Set("completed_at", nil).
			Set("next_attempt_at", sq.Expr(
				"now() + (? * interval '1 microsecond')",
				rebuildRetryDelay(job.Attempts).Microseconds(),
			)).
			Set("last_error", rebuildErr.Error())
	} else {
		builder = builder.
			Set("status", "completed").
			Set("completed_at", sq.Expr("now()")).
			Set("last_error", nil)
	}
	updateSQL, updateArgs, err := builder.ToSql()
	if err != nil {
		return false, fmt.Errorf("build complete PM aggregation rebuild: %w", err)
	}
	tag, err := tx.Exec(ctx, updateSQL, updateArgs...)
	if err != nil {
		return false, fmt.Errorf("complete PM aggregation rebuild: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return false, fmt.Errorf("PM aggregation rebuild lease ownership lost before finish")
	}
	if stable && rebuildErr == nil {
		if err := enqueuePublishedParentsTx(ctx, tx, job, snapshot); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit PM aggregation rebuild completion: %w", err)
	}
	return stable && rebuildErr == nil, nil
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
			"status":          []string{"open", "prepared", "published", "rebuilding", "finalizing", "failed"},
			"task_id":         parentTaskID,
			"task_version_id": parentVersionID,
			"entity_key":      job.Key.EntityKey,
			"granularity":     string(target),
		}).
		Where(sq.Lt{"window_start": job.Key.End}).
		Where(sq.Gt{"window_end": job.Key.Start})
}

func enqueuePublishedParentsTx(
	ctx context.Context,
	tx rebuildCompletionTx,
	job RebuildJob,
	snapshot *TaskSnapshot,
) error {
	parentTaskID, parentVersionID := rebuildParentLineage(job.Key, snapshot)
	for _, target := range RebuildCascadeTargets(job.Key) {
		sourceID := fmt.Sprintf(
			"cascade:%d:%d:%s",
			job.ID, job.RequestGeneration, target,
		)
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
  END
WHERE pm_aggregation_rebuilds.source_event_id IS DISTINCT FROM EXCLUDED.source_event_id`).
			ToSql()
		if err != nil {
			return fmt.Errorf("build enqueue parent PM aggregation rebuild: %w", err)
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("enqueue parent PM aggregation rebuild: %w", err)
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
	activeLeases := newRebuildActiveLeases(jobs)
	batchCtx, cancelBatch := context.WithCancel(ctx)
	heartbeatDone := make(chan error, 1)
	go func() {
		ticker := time.NewTicker(rebuildLeaseDuration / 3)
		defer ticker.Stop()
		heartbeatDone <- maintainRebuildLeases(
			batchCtx,
			ticker.C,
			func(ctx context.Context) error {
				return activeLeases.renew(ctx, r.repo.renewLeases)
			},
			cancelBatch,
		)
	}()
	results := r.rebuildClaimedBatch(batchCtx, jobs)
	var batchErr error
	for _, job := range jobs {
		rebuildErr := results[job.ID]
		releaseLeaseGuard := activeLeases.beginCompletion(job.ID)
		stable, finishErr := r.repo.finishAndEnqueueParents(
			batchCtx, job, r.recovery.snapshot.Current(), rebuildErr,
			releaseLeaseGuard,
		)
		// Release also covers begin/query failures before the row-lock callback.
		releaseLeaseGuard()
		rebuildErr = errors.Join(rebuildErr, finishErr)
		if rebuildErr != nil {
			if r.metrics != nil {
				r.metrics.RebuildErrorsTotal.Inc()
			}
			batchErr = errors.Join(batchErr, rebuildErr)
			continue
		}
		if stable && r.metrics != nil {
			r.metrics.RebuildsTotal.WithLabelValues(
				string(job.Key.Granularity),
			).Inc()
		}
	}
	cancelBatch()
	batchErr = errors.Join(batchErr, <-heartbeatDone)
	return batchErr
}

func (r *Rebuilder) rebuildClaimedBatch(
	ctx context.Context,
	jobs []RebuildJob,
) map[int64]error {
	results := make(map[int64]error, len(jobs))
	snapshot := r.recovery.snapshot.Current()
	groups, ungrouped := groupRollupRebuildJobs(jobs, snapshot)
	const ungroupedConcurrency = 8
	if len(ungrouped) > 0 {
		work := make(chan RebuildJob)
		var mu sync.Mutex
		var workers sync.WaitGroup
		workerCount := min(ungroupedConcurrency, len(ungrouped))
		for range workerCount {
			workers.Add(1)
			go func() {
				defer workers.Done()
				for job := range work {
					err := r.rebuild(ctx, job)
					mu.Lock()
					results[job.ID] = err
					mu.Unlock()
				}
			}()
		}
		for _, job := range ungrouped {
			work <- job
		}
		close(work)
		workers.Wait()
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
				r.recovery.matcher.Location(),
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
		r.metrics.RebuildSnapshotPagesTotal.Add(float64(rollupSnapshotDataPages(rows)))
		r.metrics.RebuildSnapshotScanSeconds.Observe(
			time.Since(scanStarted).Seconds(),
		)
	}
	for _, item := range prepared {
		if scanErr != nil {
			item.err = errors.Join(item.err, scanErr)
		}
		if item.err == nil {
			item.err = r.completeRebuildUnderLock(
				workCtx, item.job.Key, item.previous, item.lock,
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

func rollupSnapshotDataPages(rows int) int {
	if rows <= 0 {
		return 0
	}
	return (rows + int(rollupSnapshotPageSize) - 1) / int(rollupSnapshotPageSize)
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
	lock *Lock,
) error {
	state, readErr := r.store.Read(ctx, key)
	usesRawSources := rebuildUsesRawSources(
		key, r.recovery.snapshot.Current(),
	)
	state, missing, err := rebuiltWindowState(state, readErr, usesRawSources)
	if err != nil {
		return fmt.Errorf("read rebuilt PM aggregation window: %w", err)
	}
	if missing {
		if err := r.store.InitializeEmptyWithLock(ctx, key, lock); err != nil {
			return err
		}
	}
	if state.ReceivedSlots < previous.PreviousReceived &&
		usesRawSources {
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

func rebuiltWindowState(
	state WindowState,
	err error,
	usesRawSources bool,
) (WindowState, bool, error) {
	if err == nil {
		return state, false, nil
	}
	if errors.Is(err, redis.Nil) && !usesRawSources {
		// Rollup snapshots are the authoritative materialized source for parent
		// windows. An empty scan is therefore a real zero-valued replacement,
		// not missing durable data. Finalization will delete the prior revision's
		// result and rollup rows in the same publication transaction.
		return WindowState{}, true, nil
	}
	return WindowState{}, false, err
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
	return r.completeRebuildUnderLock(ctx, job.Key, previous, lock)
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
					r.recovery.matcher.Location(),
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
			}
			return nil
		},
	)
	if err != nil {
		return err
	}
	return nil
}
