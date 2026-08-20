package stream

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
	"go.uber.org/zap"
)

const (
	runtimeCleanupLockID       int64 = 0x504d434c45414e
	runtimeCleanupBacklogLimit       = uint64(100000)
)

type RuntimeCleanupConfig struct {
	BatchSize       int
	Interval        time.Duration
	MaxDuration     time.Duration
	OutboxRetention time.Duration
	ReplayRetention time.Duration
	VacuumEnabled   bool
}

type RuntimeCleaner struct {
	outbox  *OutboxRepository
	rollup  *RollupOutboxRepository
	windows *WindowRepository
	config  RuntimeCleanupConfig
	metrics *Metrics
	logger  *zap.Logger
	now     func() time.Time
}

type runtimeCleanupTarget struct {
	name    string
	table   string
	cleanup func(context.Context, uint64) (int64, error)
	backlog func(context.Context, uint64) (int64, error)
}

func NewRuntimeCleaner(
	outbox *OutboxRepository,
	rollup *RollupOutboxRepository,
	windows *WindowRepository,
	logger *zap.Logger,
) *RuntimeCleaner {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RuntimeCleaner{
		outbox: outbox, rollup: rollup, windows: windows, logger: logger,
		config: RuntimeCleanupConfig{
			BatchSize: 500, Interval: time.Hour, MaxDuration: 30 * time.Second,
			OutboxRetention: 24 * time.Hour, ReplayRetention: 45 * 24 * time.Hour,
		},
		now: func() time.Time { return time.Now().UTC() },
	}
}

func (c *RuntimeCleaner) SetConfig(config RuntimeCleanupConfig) *RuntimeCleaner {
	if config.BatchSize > 0 {
		c.config.BatchSize = min(config.BatchSize, maximumCleanupBatch)
	}
	if config.Interval > 0 {
		c.config.Interval = boundedDuration(config.Interval, time.Minute, 24*time.Hour)
	}
	if config.MaxDuration > 0 {
		c.config.MaxDuration = boundedDuration(config.MaxDuration, time.Second, 5*time.Minute)
	}
	if config.OutboxRetention > 0 {
		c.config.OutboxRetention = config.OutboxRetention
	}
	if config.ReplayRetention > 0 {
		c.config.ReplayRetention = config.ReplayRetention
	}
	c.config.VacuumEnabled = config.VacuumEnabled
	return c
}

func (c *RuntimeCleaner) SetMetrics(metrics *Metrics) *RuntimeCleaner {
	c.metrics = metrics
	return c
}

func (c *RuntimeCleaner) Run(ctx context.Context) {
	c.runAndLog(ctx)
	ticker := time.NewTicker(c.config.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.runAndLog(ctx)
		}
	}
}

func (c *RuntimeCleaner) runAndLog(ctx context.Context) {
	if err := c.RunOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
		c.logger.Warn("clean PM aggregation runtime data", zap.Error(err))
	}
}

func (c *RuntimeCleaner) RunOnce(ctx context.Context) error {
	if c == nil || c.outbox == nil || c.rollup == nil || c.windows == nil {
		return errors.New("PM aggregation runtime cleaner dependencies are missing")
	}
	lockConn, err := c.windows.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire PM runtime cleanup lock connection: %w", err)
	}
	defer lockConn.Release()
	var locked bool
	if err := lockConn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", runtimeCleanupLockID).Scan(&locked); err != nil {
		return fmt.Errorf("acquire PM runtime cleanup lock: %w", err)
	}
	if !locked {
		return nil
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = lockConn.Exec(unlockCtx, "SELECT pg_advisory_unlock($1)", runtimeCleanupLockID)
	}()

	runCtx, cancel := context.WithTimeout(ctx, c.config.MaxDuration)
	defer cancel()
	now := c.now().UTC()
	targets := c.targets(
		now.Add(-c.config.OutboxRetention),
		now.Add(-c.config.ReplayRetention),
	)
	totals := make(map[string]int64, len(targets))
	disabled := make(map[string]bool, len(targets))
	var runErrors []error
	cleanupBudget := c.config.MaxDuration * 2 / 3
	started := time.Now()
	for time.Since(started) < cleanupBudget {
		progress := false
		for _, target := range targets {
			if disabled[target.name] || runCtx.Err() != nil {
				continue
			}
			batchStart := time.Now()
			count, cleanupErr := target.cleanup(runCtx, uint64(c.config.BatchSize))
			if c.metrics != nil {
				c.metrics.RuntimeCleanupDuration.WithLabelValues(target.name).
					Observe(time.Since(batchStart).Seconds())
			}
			if cleanupErr != nil {
				disabled[target.name] = true
				runErrors = append(runErrors, fmt.Errorf("cleanup %s: %w", target.name, cleanupErr))
				if c.metrics != nil {
					c.metrics.RuntimeCleanupErrorsTotal.WithLabelValues(target.name).Inc()
				}
				continue
			}
			if count > 0 {
				progress = true
				totals[target.name] += count
				if c.metrics != nil {
					c.metrics.RuntimeCleanupRowsTotal.WithLabelValues(target.name).Add(float64(count))
				}
			}
		}
		if !progress || runCtx.Err() != nil {
			break
		}
	}

	for _, target := range targets {
		if totals[target.name] == 0 || target.table == "" || runCtx.Err() != nil {
			continue
		}
		if err := maintainRuntimeCleanupTable(runCtx, c.windows.pool, target.table, false); err != nil {
			runErrors = append(runErrors, err)
			continue
		}
		if c.config.VacuumEnabled {
			if err := maintainRuntimeCleanupTable(runCtx, c.windows.pool, target.table, true); err != nil {
				runErrors = append(runErrors, err)
			}
		}
	}

	for _, target := range targets {
		remaining := int64(-1)
		if runCtx.Err() == nil {
			sampled, backlogErr := target.backlog(runCtx, runtimeCleanupBacklogLimit)
			if backlogErr != nil {
				runErrors = append(runErrors, fmt.Errorf("sample %s cleanup backlog: %w", target.name, backlogErr))
			} else {
				remaining = sampled
				if c.metrics != nil {
					c.metrics.RuntimeCleanupBacklog.WithLabelValues(target.name).Set(float64(remaining))
				}
			}
		}
		c.logger.Info("PM aggregation runtime cleanup batch completed",
			zap.String("target", target.name),
			zap.Int64("deleted", totals[target.name]),
			zap.Int64("remaining_sample", remaining),
			zap.Duration("duration", time.Since(started)))
	}
	if ctx.Err() != nil {
		return errors.Join(append(runErrors, ctx.Err())...)
	}
	return errors.Join(runErrors...)
}

func (c *RuntimeCleaner) targets(outboxBefore, replayBefore time.Time) []runtimeCleanupTarget {
	return []runtimeCleanupTarget{
		{
			name: "outbox", table: "pm_aggregation_outbox",
			cleanup: func(ctx context.Context, limit uint64) (int64, error) {
				return c.outbox.DeletePublishedBeforeBatch(ctx, outboxBefore, replayBefore, limit)
			},
			backlog: func(ctx context.Context, limit uint64) (int64, error) {
				return c.outbox.CountPublishedCleanupBacklog(ctx, outboxBefore, replayBefore, limit)
			},
		},
		{
			name: "rollup_outbox", table: "pm_aggregation_rollup_outbox",
			cleanup: func(ctx context.Context, limit uint64) (int64, error) {
				return c.rollup.DeletePublishedBeforeBatch(ctx, outboxBefore, limit)
			},
			backlog: func(ctx context.Context, limit uint64) (int64, error) {
				return c.rollup.CountPublishedCleanupBacklog(ctx, outboxBefore, limit)
			},
		},
		{
			name: "replay_chunks",
			cleanup: func(ctx context.Context, _ uint64) (int64, error) {
				return c.outbox.DropReplayChunkBefore(ctx, replayBefore)
			},
			backlog: func(ctx context.Context, limit uint64) (int64, error) {
				return c.outbox.CountReplayChunkBacklog(ctx, replayBefore, limit)
			},
		},
		{
			name: "terminal_windows", table: "pm_aggregation_windows",
			cleanup: func(ctx context.Context, limit uint64) (int64, error) {
				return c.windows.DeleteTerminalBeforeBatch(ctx, replayBefore, limit)
			},
			backlog: func(ctx context.Context, limit uint64) (int64, error) {
				return c.windows.CountTerminalCleanupBacklog(ctx, replayBefore, limit)
			},
		},
	}
}

func outboxCleanupCandidates(before, legacyBefore time.Time, limit uint64) sq.SelectBuilder {
	return storage.Psql.Select("event_id").
		From("pm_aggregation_outbox").
		Where(sq.Or{
			sq.And{
				sq.Lt{"published_at": before}, sq.Expr("consumed_at IS NOT NULL"),
				sq.Eq{"barrier_eligible": true},
			},
			sq.And{sq.Lt{"published_at": legacyBefore}, sq.Eq{"barrier_eligible": false}},
		}).
		OrderBy("published_at", "event_id").
		Limit(limit)
}

func buildOutboxCleanupBatch(
	before, legacyBefore time.Time,
	limit uint64,
) (string, []interface{}, error) {
	candidates := outboxCleanupCandidates(before, legacyBefore, limit).
		Suffix("FOR UPDATE SKIP LOCKED")
	return storage.Psql.Delete("pm_aggregation_outbox").
		PrefixExpr(sq.Expr("WITH candidates AS (?)", candidates)).
		Where("event_id IN (SELECT event_id FROM candidates)").
		ToSql()
}

func rollupOutboxCleanupCandidates(before time.Time, limit uint64) sq.SelectBuilder {
	return storage.Psql.Select("event_id").
		From("pm_aggregation_rollup_outbox").
		Where(sq.Lt{"published_at": before}).
		Where("consumed_at IS NOT NULL").
		OrderBy("published_at", "event_id").
		Limit(limit)
}

func buildRollupOutboxCleanupBatch(
	before time.Time,
	limit uint64,
) (string, []interface{}, error) {
	candidates := rollupOutboxCleanupCandidates(before, limit).
		Suffix("FOR UPDATE SKIP LOCKED")
	return storage.Psql.Delete("pm_aggregation_rollup_outbox").
		PrefixExpr(sq.Expr("WITH candidates AS (?)", candidates)).
		Where("event_id IN (SELECT event_id FROM candidates)").
		ToSql()
}

func terminalWindowCleanupCandidates(before time.Time, limit uint64) sq.SelectBuilder {
	return storage.Psql.Select(
		"task_version_id", "entity_key", "granularity", "window_start",
	).From("pm_aggregation_windows").
		Where(sq.Or{
			sq.And{
				sq.Eq{"status": "published"}, sq.Lt{"published_at": before},
				sq.Expr(`NOT EXISTS (
SELECT 1 FROM pm_aggregation_results result
WHERE result.task_version_id = pm_aggregation_windows.task_version_id
  AND result.granularity = pm_aggregation_windows.granularity
  AND result.window_start = pm_aggregation_windows.window_start
)`),
			},
			sq.And{
				sq.Eq{"status": []string{"retired", "orphaned"}},
				sq.Lt{"recovery_terminal_at": before},
			},
			sq.And{sq.Eq{"status": "abandoned"}, sq.Lt{"updated_at": before}},
		}).
		OrderBy("window_start", "task_version_id", "entity_key", "granularity").
		Limit(limit)
}

func buildTerminalWindowCleanupBatch(
	before time.Time,
	limit uint64,
) (string, []interface{}, error) {
	candidates := terminalWindowCleanupCandidates(before, limit).
		Suffix("FOR UPDATE SKIP LOCKED")
	return storage.Psql.Delete("pm_aggregation_windows").
		PrefixExpr(sq.Expr("WITH candidates AS (?)", candidates)).
		Where(`(task_version_id, entity_key, granularity, window_start) IN (
SELECT task_version_id, entity_key, granularity, window_start FROM candidates
)`).
		ToSql()
}

func buildReplayChunkCandidateQuery(before time.Time) (string, []interface{}, error) {
	return storage.Psql.Select("range_start", "range_end").
		From("timescaledb_information.chunks").
		Where(sq.Eq{"hypertable_schema": "public", "hypertable_name": "pm_aggregation_replay_sources"}).
		Where(sq.LtOrEq{"range_end": before}).
		OrderBy("range_start").
		Limit(1).
		ToSql()
}

func maintainRuntimeCleanupTable(
	ctx context.Context,
	pool *pgxpool.Pool,
	table string,
	vacuum bool,
) error {
	var query string
	switch table {
	case "pm_aggregation_outbox":
		query = "ANALYZE public.pm_aggregation_outbox"
		if vacuum {
			query = "VACUUM (ANALYZE, INDEX_CLEANUP AUTO, TRUNCATE FALSE) public.pm_aggregation_outbox"
		}
	case "pm_aggregation_rollup_outbox":
		query = "ANALYZE public.pm_aggregation_rollup_outbox"
		if vacuum {
			query = "VACUUM (ANALYZE, INDEX_CLEANUP AUTO, TRUNCATE FALSE) public.pm_aggregation_rollup_outbox"
		}
	case "pm_aggregation_windows":
		query = "ANALYZE public.pm_aggregation_windows"
		if vacuum {
			query = "VACUUM (ANALYZE, INDEX_CLEANUP AUTO, TRUNCATE FALSE) public.pm_aggregation_windows"
		}
	default:
		return fmt.Errorf("unsupported PM runtime cleanup maintenance table %q", table)
	}
	if _, err := pool.Exec(ctx, query); err != nil {
		return fmt.Errorf("maintain PM runtime cleanup table %s: %w", table, err)
	}
	return nil
}

func countCleanupCandidates(
	ctx context.Context,
	pool *pgxpool.Pool,
	candidates sq.SelectBuilder,
) (int64, error) {
	query, args, err := storage.Psql.Select("COUNT(*)").
		FromSelect(candidates, "cleanup_backlog").
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build PM runtime cleanup backlog SQL: %w", err)
	}
	var count int64
	if err := pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("query PM runtime cleanup backlog: %w", err)
	}
	return count, nil
}

func (r *OutboxRepository) DeletePublishedBeforeBatch(
	ctx context.Context,
	before, legacyBefore time.Time,
	limit uint64,
) (int64, error) {
	query, args, err := buildOutboxCleanupBatch(before, legacyBefore, limit)
	if err != nil {
		return 0, fmt.Errorf("build cleanup PM aggregation outbox batch SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("cleanup PM aggregation outbox batch: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *OutboxRepository) CountPublishedCleanupBacklog(
	ctx context.Context,
	before, legacyBefore time.Time,
	limit uint64,
) (int64, error) {
	return countCleanupCandidates(ctx, r.pool, outboxCleanupCandidates(before, legacyBefore, limit))
}

func (r *RollupOutboxRepository) DeletePublishedBeforeBatch(
	ctx context.Context,
	before time.Time,
	limit uint64,
) (int64, error) {
	query, args, err := buildRollupOutboxCleanupBatch(before, limit)
	if err != nil {
		return 0, fmt.Errorf("build cleanup PM rollup outbox batch SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("cleanup PM rollup outbox batch: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *RollupOutboxRepository) CountPublishedCleanupBacklog(
	ctx context.Context,
	before time.Time,
	limit uint64,
) (int64, error) {
	return countCleanupCandidates(ctx, r.pool, rollupOutboxCleanupCandidates(before, limit))
}

func (r *WindowRepository) DeleteTerminalBeforeBatch(
	ctx context.Context,
	before time.Time,
	limit uint64,
) (int64, error) {
	query, args, err := buildTerminalWindowCleanupBatch(before, limit)
	if err != nil {
		return 0, fmt.Errorf("build cleanup terminal PM windows batch SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("cleanup terminal PM windows batch: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *WindowRepository) CountTerminalCleanupBacklog(
	ctx context.Context,
	before time.Time,
	limit uint64,
) (int64, error) {
	return countCleanupCandidates(ctx, r.pool, terminalWindowCleanupCandidates(before, limit))
}

func (r *OutboxRepository) DropReplayChunkBefore(
	ctx context.Context,
	before time.Time,
) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin PM replay chunk cleanup: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	query, args, err := buildReplayChunkCandidateQuery(before)
	if err != nil {
		return 0, fmt.Errorf("build PM replay chunk candidate SQL: %w", err)
	}
	var start, end time.Time
	if err := tx.QueryRow(ctx, query, args...).Scan(&start, &end); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, nil
		}
		return 0, fmt.Errorf("query PM replay chunk candidate: %w", err)
	}
	var dropped int64
	if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM drop_chunks(
'public.pm_aggregation_replay_sources', older_than => $1::timestamptz,
newer_than => $2::timestamptz
)`, end, start).Scan(&dropped); err != nil {
		return 0, fmt.Errorf("drop PM replay source chunk: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit PM replay chunk cleanup: %w", err)
	}
	return dropped, nil
}

func (r *OutboxRepository) CountReplayChunkBacklog(
	ctx context.Context,
	before time.Time,
	limit uint64,
) (int64, error) {
	candidates := storage.Psql.Select("1").
		From("timescaledb_information.chunks").
		Where(sq.Eq{"hypertable_schema": "public", "hypertable_name": "pm_aggregation_replay_sources"}).
		Where(sq.LtOrEq{"range_end": before}).
		OrderBy("range_start").
		Limit(limit)
	return countCleanupCandidates(ctx, r.pool, candidates)
}
