package stream

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type WindowRecord struct {
	Key              WindowKey
	Status           string
	ExpectedSlots    int64
	ReceivedSlots    int64
	FinalizeAttempts int
}

type WindowRepository struct {
	pool *pgxpool.Pool
}

type claimOrder int

const (
	claimOldestFirst claimOrder = iota
	claimNewestFirst
)

type claimVersionFilter struct {
	versionIDs []uuid.UUID
	exclude    bool
}

// versionMetadataBackfillLockID serializes the one-time historical metadata
// backfill across app and worker startup. The value is the ASCII bytes for
// "PMVMETA", kept stable because PostgreSQL advisory locks are process-wide.
const versionMetadataBackfillLockID int64 = 0x504d564d455441

func NewWindowRepository(pool *pgxpool.Pool) *WindowRepository {
	return &WindowRepository{pool: pool}
}

func (r *WindowRepository) EnsureOpen(ctx context.Context, contribution Contribution) error {
	query, args, err := storage.Psql.Insert("pm_aggregation_windows").
		Columns(
			"task_id", "task_version_id", "granularity",
			"entity_key", "window_start", "window_end", "expected_slots",
			"version_effective_from", "version_effective_to",
		).
		Values(
			contribution.Key.TaskID, contribution.Key.TaskVersionID,
			string(contribution.Key.Granularity), contribution.Key.EntityKey, contribution.Key.Start,
			contribution.Key.End, contribution.ExpectedSlots,
			contribution.VersionEffectiveFrom, contribution.VersionEffectiveTo,
		).
		Suffix(`
ON CONFLICT (task_version_id, entity_key, granularity, window_start) DO UPDATE SET
  version_effective_from = COALESCE(
    pm_aggregation_windows.version_effective_from,
    EXCLUDED.version_effective_from
  ),
  version_effective_to = COALESCE(
    pm_aggregation_windows.version_effective_to,
    EXCLUDED.version_effective_to
  )`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build ensure PM aggregation window SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("ensure PM aggregation window: %w", err)
	}
	return nil
}

func (r *WindowRepository) BackfillVersionMetadata(
	ctx context.Context,
	snapshot *TaskSnapshot,
) error {
	_, err := r.backfillVersionMetadata(ctx, snapshot, false)
	return err
}

// TryBackfillVersionMetadata performs the backfill only when no other process
// currently owns it. The app uses this non-blocking form so a worker doing the
// same one-time upgrade work cannot delay or fail HTTP startup.
func (r *WindowRepository) TryBackfillVersionMetadata(
	ctx context.Context,
	snapshot *TaskSnapshot,
) (bool, error) {
	return r.backfillVersionMetadata(ctx, snapshot, true)
}

func (r *WindowRepository) backfillVersionMetadata(
	ctx context.Context,
	snapshot *TaskSnapshot,
	tryLock bool,
) (bool, error) {
	if snapshot == nil {
		return true, nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin PM aggregation version metadata backfill: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if tryLock {
		var acquired bool
		if err := tx.QueryRow(
			ctx,
			versionMetadataBackfillLockSQL(true),
			versionMetadataBackfillLockID,
		).Scan(&acquired); err != nil {
			return false, fmt.Errorf("try lock PM aggregation version metadata backfill: %w", err)
		}
		if !acquired {
			return false, nil
		}
	} else if _, err := tx.Exec(
		ctx,
		versionMetadataBackfillLockSQL(false),
		versionMetadataBackfillLockID,
	); err != nil {
		return false, fmt.Errorf("lock PM aggregation version metadata backfill: %w", err)
	}
	for _, versionID := range sortedSnapshotVersionIDs(snapshot) {
		version := snapshot.ByVersion[versionID]
		if version == nil || version.EffectiveFrom.IsZero() {
			continue
		}
		query, args, buildErr := storage.Psql.Update("pm_aggregation_windows").
			Set("version_effective_from", version.EffectiveFrom).
			Set("version_effective_to", version.EffectiveTo).
			Where(sq.Eq{"task_version_id": versionID, "version_effective_from": nil}).
			ToSql()
		if buildErr != nil {
			return false, fmt.Errorf("build PM aggregation version metadata backfill: %w", buildErr)
		}
		if _, execErr := tx.Exec(ctx, query, args...); execErr != nil {
			return false, fmt.Errorf("backfill PM aggregation version metadata: %w", execErr)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit PM aggregation version metadata backfill: %w", err)
	}
	return true, nil
}

func versionMetadataBackfillLockSQL(tryLock bool) string {
	if tryLock {
		return "SELECT pg_try_advisory_xact_lock($1)"
	}
	return "SELECT pg_advisory_xact_lock($1)"
}

func sortedSnapshotVersionIDs(snapshot *TaskSnapshot) []uuid.UUID {
	if snapshot == nil {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(snapshot.ByVersion))
	for versionID := range snapshot.ByVersion {
		ids = append(ids, versionID)
	}
	sort.Slice(ids, func(left, right int) bool {
		return ids[left].String() < ids[right].String()
	})
	return ids
}

func (r *WindowRepository) ObserveReceived(
	ctx context.Context,
	key WindowKey,
	received int64,
) error {
	query, args, err := observeReceivedUpdate(key, received).
		ToSql()
	if err != nil {
		return fmt.Errorf("build observe PM aggregation window SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("observe PM aggregation window received slots: %w", err)
	}
	return nil
}

func observeReceivedUpdate(key WindowKey, received int64) sq.UpdateBuilder {
	return storage.Psql.Update("pm_aggregation_windows").
		Set("received_slots", received).
		Set("status", "open").
		Set("last_error", nil).
		Set("finalize_attempts", 0).
		Set("finalize_next_attempt_at", sq.Expr("CURRENT_TIMESTAMP")).
		Set("updated_at", time.Now().UTC()).
		Where(sq.Eq{
			"task_version_id": key.TaskVersionID,
			"entity_key":      key.EntityKey,
			"granularity":     string(key.Granularity),
			"window_start":    key.Start,
			"status":          []string{"open", "failed"},
		})
}

func finalizationClaimUpdate(
	key WindowKey,
	reason CloseReason,
	state WindowState,
	coverage finalizationCoverage,
	version *TaskVersionSnapshot,
) sq.UpdateBuilder {
	builder := storage.Psql.Update("pm_aggregation_windows").
		Set("status", "finalizing").
		Set("close_reason", string(reason)).
		Set("expected_slots", state.ExpectedSlots).
		Set("received_slots", state.ReceivedSlots).
		Set("source_expected_slots", coverage.SourceExpectedSlots).
		Set("source_received_slots", coverage.SourceReceivedSlots).
		Set("missing_slots", coverage.MissingSlots).
		Set("children_complete", coverage.ChildrenComplete).
		Set("source_incomplete_slots", state.SourceIncompleteSlots).
		Set("data_complete", coverage.DataComplete).
		Set("updated_at", time.Now().UTC()).
		Where(windowKeyPredicate(key)).
		Where(sq.Eq{"status": []string{"open", "failed", "finalizing", "prepared", "rebuilding"}})
	if version != nil {
		builder = builder.
			Set("version_effective_from", version.EffectiveFrom).
			Set("version_effective_to", versionEffectiveTo(version))
	}
	return builder
}

func finalizationClaimUpdateForOwner(
	key WindowKey,
	reason CloseReason,
	state WindowState,
	coverage finalizationCoverage,
	version *TaskVersionSnapshot,
	leaseOwner *uuid.UUID,
) sq.UpdateBuilder {
	builder := finalizationClaimUpdate(key, reason, state, coverage, version)
	if leaseOwner != nil {
		builder = builder.
			Where(sq.Eq{"finalize_lease_owner": *leaseOwner}).
			Where(sq.Expr("finalize_lease_until > CURRENT_TIMESTAMP"))
	}
	return builder
}

func (r *WindowRepository) Status(ctx context.Context, key WindowKey) (string, error) {
	status, _, err := r.StatusRevision(ctx, key)
	return status, err
}

func (r *WindowRepository) StatusRevision(ctx context.Context, key WindowKey) (string, int, error) {
	query, args, err := storage.Psql.Select("status", "revision").
		From("pm_aggregation_windows").
		Where(windowKeyPredicate(key)).
		ToSql()
	if err != nil {
		return "", 0, fmt.Errorf("build PM aggregation window status SQL: %w", err)
	}
	var status string
	var revision int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&status, &revision); err != nil {
		if err == pgx.ErrNoRows {
			return "", 0, nil
		}
		return "", 0, fmt.Errorf("query PM aggregation window status: %w", err)
	}
	return status, revision, nil
}

func (r *WindowRepository) IsPublished(ctx context.Context, key WindowKey) (bool, error) {
	status, err := r.Status(ctx, key)
	return status == "published", err
}

func (r *WindowRepository) ClaimDue(
	ctx context.Context,
	granularity Granularity,
	dueBefore time.Time,
	limit uint64,
	leaseOwner uuid.UUID,
	leaseUntil time.Time,
) ([]WindowRecord, error) {
	return r.claimDue(
		ctx, granularity, dueBefore, limit, leaseOwner, leaseUntil,
		claimVersionFilter{}, claimOldestFirst,
	)
}

func (r *WindowRepository) claimDue(
	ctx context.Context,
	granularity Granularity,
	dueBefore time.Time,
	limit uint64,
	leaseOwner uuid.UUID,
	leaseUntil time.Time,
	filter claimVersionFilter,
	order claimOrder,
) ([]WindowRecord, error) {
	if limit == 0 || (len(filter.versionIDs) == 0 && !filter.exclude &&
		filter.versionIDs != nil) {
		return nil, nil
	}
	if leaseOwner == uuid.Nil {
		return nil, fmt.Errorf("claim PM aggregation windows: lease owner is required")
	}
	leaseFor := time.Until(leaseUntil)
	if leaseFor <= 0 {
		return nil, fmt.Errorf("claim PM aggregation windows: lease must expire in the future")
	}
	query, args, err := claimDueUpdate(
		granularity, dueBefore, limit, leaseOwner, leaseFor, filter, order,
	).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build claim due PM aggregation windows SQL: %w", err)
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin claim due PM aggregation windows: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("claim due PM aggregation windows: %w", err)
	}
	result, err := scanWindowRows(rows)
	rows.Close()
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit claimed PM aggregation windows: %w", err)
	}
	return result, nil
}

func claimDueUpdate(
	granularity Granularity,
	dueBefore time.Time,
	limit uint64,
	leaseOwner uuid.UUID,
	leaseFor time.Duration,
	filter claimVersionFilter,
	order claimOrder,
) sq.UpdateBuilder {
	candidates := storage.Psql.Select(
		"w.task_version_id", "w.entity_key", "w.granularity", "w.window_start",
	).From("pm_aggregation_windows w").
		Where(sq.Eq{
			"w.status":      []string{"open", "failed"},
			"w.granularity": string(granularity),
		}).
		Where(sq.LtOrEq{"w.window_end": dueBefore}).
		Where(sq.Or{
			sq.Eq{"w.finalize_lease_until": nil},
			sq.Expr("w.finalize_lease_until <= CURRENT_TIMESTAMP"),
		}).
		Where(sq.Expr("w.finalize_next_attempt_at <= CURRENT_TIMESTAMP")).
		Where(queueBarrierConsumedPredicate()).
		Limit(limit)
	candidates = applyHourlyHierarchyBarrier(candidates, granularity, filter)
	candidates = applyClaimVersionFilter(candidates, filter)
	if order == claimNewestFirst {
		candidates = candidates.OrderBy(
			"w.window_end DESC", "w.task_version_id DESC", "w.entity_key DESC", "w.window_start DESC",
		)
	} else {
		candidates = candidates.OrderBy(
			"w.window_end", "w.task_version_id", "w.entity_key", "w.window_start",
		)
	}
	candidates = candidates.
		Suffix("FOR UPDATE SKIP LOCKED").
		PlaceholderFormat(sq.Question)
	return storage.Psql.Update("pm_aggregation_windows w").
		PrefixExpr(sq.Expr("WITH candidates AS (?)", candidates)).
		Set("finalize_lease_owner", leaseOwner).
		Set("finalize_lease_until", sq.Expr(
			"CURRENT_TIMESTAMP + (? * INTERVAL '1 second')", leaseFor.Seconds(),
		)).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		From("candidates c").
		Where(`
w.task_version_id = c.task_version_id
AND w.entity_key = c.entity_key
AND w.granularity = c.granularity
AND w.window_start = c.window_start`).
		Suffix(`
RETURNING w.task_id, w.task_version_id, w.entity_key, w.granularity,
          w.window_start, w.window_end, w.status, w.expected_slots, w.received_slots,
          w.finalize_attempts`)
}

func applyClaimVersionFilter(
	builder sq.SelectBuilder,
	filter claimVersionFilter,
) sq.SelectBuilder {
	if filter.versionIDs == nil {
		return builder
	}
	if filter.exclude {
		return builder.Where(sq.Expr(
			"w.task_version_id <> ALL(?)", filter.versionIDs,
		))
	}
	return builder.Where(sq.Expr(
		"w.task_version_id = ANY(?)", filter.versionIDs,
	))
}

func applyHourlyHierarchyBarrier(
	builder sq.SelectBuilder,
	granularity Granularity,
	filter claimVersionFilter,
) sq.SelectBuilder {
	if granularity != GranularityHourly || !filter.exclude || len(filter.versionIDs) == 0 {
		return builder
	}
	return builder.Where(sq.Expr(`
NOT EXISTS (
    SELECT 1
    FROM pm_aggregation_windows source_window
    WHERE source_window.granularity = 'hourly'
      AND source_window.window_start = w.window_start
      AND source_window.task_version_id = ANY(?)
      AND source_window.status IN ('open', 'failed', 'finalizing', 'rebuilding')
)
AND NOT EXISTS (
    SELECT 1
    FROM pm_aggregation_rollup_outbox source_rollup
    WHERE source_rollup.consumed_at IS NULL
      AND source_rollup.barrier_eligible
      AND source_rollup.subject = ?
      AND source_rollup.window_start = w.window_start
      AND EXISTS (
          SELECT 1 FROM pm_aggregation_windows rollup_window
          WHERE rollup_window.task_version_id = source_rollup.publication_task_version_id
            AND rollup_window.entity_key = source_rollup.entity_key
            AND rollup_window.granularity = source_rollup.granularity
            AND rollup_window.window_start = source_rollup.window_start
            AND rollup_window.published_revision = source_rollup.revision
      )
)`,
		filter.versionIDs,
		"pmaggregation.hourly.rollup",
	))
}

func (r *WindowRepository) hasClaimConflict(
	ctx context.Context,
	granularity Granularity,
	dueBefore time.Time,
	filter claimVersionFilter,
	leaseOwner uuid.UUID,
) (bool, error) {
	query, args, err := claimConflictSelect(
		granularity, dueBefore, filter, leaseOwner,
	).ToSql()
	if err != nil {
		return false, fmt.Errorf("build PM aggregation finalize claim conflict SQL: %w", err)
	}
	var marker int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&marker); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("query PM aggregation finalize claim conflict: %w", err)
	}
	return true, nil
}

func claimConflictSelect(
	granularity Granularity,
	dueBefore time.Time,
	filter claimVersionFilter,
	leaseOwner uuid.UUID,
) sq.SelectBuilder {
	builder := storage.Psql.Select("1").
		From("pm_aggregation_windows w").
		Where(sq.Eq{
			"w.status":      []string{"open", "failed"},
			"w.granularity": string(granularity),
		}).
		Where(sq.LtOrEq{"w.window_end": dueBefore}).
		Where(sq.Expr("w.finalize_lease_until > CURRENT_TIMESTAMP")).
		Where(sq.NotEq{"w.finalize_lease_owner": leaseOwner}).
		Where(queueBarrierConsumedPredicate()).
		Limit(1)
	builder = applyHourlyHierarchyBarrier(builder, granularity, filter)
	return applyClaimVersionFilter(builder, filter)
}

func (r *WindowRepository) CompleteClaim(
	ctx context.Context,
	key WindowKey,
	leaseOwner uuid.UUID,
) error {
	return r.clearFinalizeClaim(ctx, key, leaseOwner, "complete")
}

func (r *WindowRepository) RenewClaim(
	ctx context.Context,
	key WindowKey,
	leaseOwner uuid.UUID,
	leaseUntil time.Time,
) error {
	leaseFor := time.Until(leaseUntil)
	if leaseFor <= 0 {
		return fmt.Errorf("renew PM aggregation finalize claim: lease must expire in the future")
	}
	query, args, err := renewFinalizeClaimUpdate(key, leaseOwner, leaseFor).ToSql()
	if err != nil {
		return fmt.Errorf("build renew PM aggregation finalize claim SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("renew PM aggregation finalize claim: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf(
			"%w: renew affected %d rows",
			ErrFinalizeClaimLost,
			tag.RowsAffected(),
		)
	}
	return nil
}

func renewFinalizeClaimUpdate(
	key WindowKey,
	leaseOwner uuid.UUID,
	leaseFor time.Duration,
) sq.UpdateBuilder {
	return storage.Psql.Update("pm_aggregation_windows").
		Set("finalize_lease_until", sq.Expr(
			"CURRENT_TIMESTAMP + (? * INTERVAL '1 second')", leaseFor.Seconds(),
		)).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(windowKeyPredicate(key)).
		Where(sq.Eq{"finalize_lease_owner": leaseOwner}).
		Where(sq.Expr("finalize_lease_until > CURRENT_TIMESTAMP"))
}

func (r *WindowRepository) ReleaseClaim(
	ctx context.Context,
	key WindowKey,
	leaseOwner uuid.UUID,
) error {
	return r.clearFinalizeClaim(ctx, key, leaseOwner, "release")
}

func (r *WindowRepository) FailClaim(
	ctx context.Context,
	key WindowKey,
	leaseOwner uuid.UUID,
	cause error,
	retryAfter time.Duration,
) error {
	query, args, err := failFinalizeClaimUpdate(
		key, leaseOwner, cause, retryAfter,
	).ToSql()
	if err != nil {
		return fmt.Errorf("build fail PM aggregation finalize claim SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("fail PM aggregation finalize claim: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf(
			"%w: fail affected %d rows",
			ErrFinalizeClaimLost,
			tag.RowsAffected(),
		)
	}
	return nil
}

func failFinalizeClaimUpdate(
	key WindowKey,
	leaseOwner uuid.UUID,
	cause error,
	retryAfter time.Duration,
) sq.UpdateBuilder {
	if retryAfter < 0 {
		retryAfter = 0
	}
	lastError := ""
	if cause != nil {
		lastError = cause.Error()
	}
	return storage.Psql.Update("pm_aggregation_windows").
		Set("status", "failed").
		Set("last_error", lastError).
		Set("finalize_attempts", sq.Expr("finalize_attempts + 1")).
		Set("finalize_next_attempt_at", sq.Expr(
			"CURRENT_TIMESTAMP + (? * INTERVAL '1 microsecond')",
			retryAfter.Microseconds(),
		)).
		Set("finalize_lease_owner", nil).
		Set("finalize_lease_until", nil).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(windowKeyPredicate(key)).
		Where(sq.Eq{"finalize_lease_owner": leaseOwner})
}

func (r *WindowRepository) clearFinalizeClaim(
	ctx context.Context,
	key WindowKey,
	leaseOwner uuid.UUID,
	action string,
) error {
	query, args, err := clearFinalizeClaimUpdate(key, leaseOwner).ToSql()
	if err != nil {
		return fmt.Errorf("build %s PM aggregation finalize claim SQL: %w", action, err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s PM aggregation finalize claim: %w", action, err)
	}
	if tag.RowsAffected() != 1 {
		return fmt.Errorf("%w: %s affected %d rows", ErrFinalizeClaimLost, action, tag.RowsAffected())
	}
	return nil
}

func clearFinalizeClaimUpdate(key WindowKey, leaseOwner uuid.UUID) sq.UpdateBuilder {
	return storage.Psql.Update("pm_aggregation_windows").
		Set("finalize_lease_owner", nil).
		Set("finalize_lease_until", nil).
		Set("updated_at", sq.Expr("CURRENT_TIMESTAMP")).
		Where(windowKeyPredicate(key)).
		Where(sq.Eq{"finalize_lease_owner": leaseOwner})
}

func (r *WindowRepository) ListDue(
	ctx context.Context,
	now time.Time,
	grace time.Duration,
	limit uint64,
) ([]WindowRecord, error) {
	query, args, err := storage.Psql.Select(
		"task_id", "task_version_id", "entity_key", "granularity", "window_start", "window_end",
		"status", "expected_slots", "received_slots", "finalize_attempts",
	).From("pm_aggregation_windows w").
		Where(sq.Eq{"status": []string{"open", "failed"}}).
		Where(sq.LtOrEq{"window_end": now.Add(-grace)}).
		Where(queueBarrierConsumedPredicate()).
		OrderBy("window_end").
		Limit(limit).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list due PM aggregation windows SQL: %w", err)
	}
	return r.queryWindows(ctx, query, args...)
}

func (r *WindowRepository) ListDueByGranularity(
	ctx context.Context,
	now time.Time,
	graceByGranularity map[Granularity]time.Duration,
	fallbackGrace time.Duration,
	limit uint64,
) ([]WindowRecord, error) {
	return r.ListDueByGranularityAfter(
		ctx, now, graceByGranularity, fallbackGrace, nil, limit,
	)
}

func (r *WindowRepository) ListDueByGranularityAfter(
	ctx context.Context,
	now time.Time,
	graceByGranularity map[Granularity]time.Duration,
	fallbackGrace time.Duration,
	after *WindowKey,
	limit uint64,
) ([]WindowRecord, error) {
	due := sq.Or{}
	for _, granularity := range []Granularity{
		GranularityHourly, GranularityDaily, GranularityWeekly, GranularityMonthly,
	} {
		grace := graceByGranularity[granularity]
		if grace <= 0 {
			grace = fallbackGrace
		}
		due = append(due, sq.And{
			sq.Eq{"granularity": string(granularity)},
			sq.LtOrEq{"window_end": now.Add(-grace)},
		})
	}
	builder := storage.Psql.Select(
		"task_id", "task_version_id", "entity_key", "granularity", "window_start", "window_end",
		"status", "expected_slots", "received_slots", "finalize_attempts",
	).From("pm_aggregation_windows w").
		Where(sq.Eq{"status": []string{"open", "failed"}}).
		Where(due).
		Where(queueBarrierConsumedPredicate()).
		OrderBy("window_end", "task_version_id", "entity_key", "granularity", "window_start").
		Limit(limit).
		PlaceholderFormat(sq.Dollar)
	if after != nil {
		builder = builder.Where(sq.Expr(
			"(window_end, task_version_id, entity_key, granularity, window_start) > (?, ?, ?, ?, ?)",
			after.End, after.TaskVersionID, after.EntityKey, string(after.Granularity), after.Start,
		))
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list due PM aggregation windows by granularity SQL: %w", err)
	}
	return r.queryWindows(ctx, query, args...)
}

func queueBarrierConsumedPredicate() sq.Sqlizer {
	return sq.Expr(`
NOT EXISTS (
    SELECT 1
    FROM pm_aggregation_outbox source_event
    WHERE w.granularity = 'hourly'
      AND source_event.consumed_at IS NULL
      AND source_event.barrier_eligible
      AND source_event.event_window_start >= w.window_start
      AND source_event.event_window_start < w.window_end
)
AND NOT EXISTS (
    SELECT 1
    FROM pm_aggregation_rollup_outbox source_rollup
    WHERE source_rollup.consumed_at IS NULL
      AND source_rollup.barrier_eligible
      AND source_rollup.window_start >= w.window_start
      AND source_rollup.window_start < w.window_end
      AND EXISTS (
          SELECT 1 FROM pm_aggregation_windows rollup_window
          WHERE rollup_window.task_version_id = source_rollup.publication_task_version_id
            AND rollup_window.entity_key = source_rollup.entity_key
            AND rollup_window.granularity = source_rollup.granularity
            AND rollup_window.window_start = source_rollup.window_start
            AND rollup_window.published_revision = source_rollup.revision
      )
      AND (
          (w.granularity = 'daily' AND source_rollup.subject = ?)
          OR
          (w.granularity IN ('weekly', 'monthly') AND source_rollup.subject = ?)
      )
)`,
		"pmaggregation.hourly.rollup",
		"pmaggregation.daily.rollup",
	)
}

const finalizeDueAtSQL = `w.window_end + CASE w.granularity
    WHEN 'hourly' THEN (? * INTERVAL '1 microsecond')
    WHEN 'daily' THEN (? * INTERVAL '1 microsecond')
    WHEN 'weekly' THEN (? * INTERVAL '1 microsecond')
    WHEN 'monthly' THEN (? * INTERVAL '1 microsecond')
    ELSE (? * INTERVAL '1 microsecond')
END AS due_at`

func oldestDueSelect(
	graceByGranularity map[Granularity]time.Duration,
	fallbackGrace time.Duration,
) sq.SelectBuilder {
	graceFor := func(granularity Granularity) time.Duration {
		grace := graceByGranularity[granularity]
		if grace <= 0 {
			grace = fallbackGrace
		}
		return grace
	}
	due := sq.Or{}
	for _, granularity := range []Granularity{
		GranularityHourly,
		GranularityDaily,
		GranularityWeekly,
		GranularityMonthly,
	} {
		due = append(due, sq.And{
			sq.Eq{"w.granularity": string(granularity)},
			sq.Expr(
				"w.window_end <= CURRENT_TIMESTAMP - (? * INTERVAL '1 microsecond')",
				graceFor(granularity).Microseconds(),
			),
		})
	}
	eligible := storage.Psql.Select().
		Column(sq.Expr(
			finalizeDueAtSQL,
			graceFor(GranularityHourly).Microseconds(),
			graceFor(GranularityDaily).Microseconds(),
			graceFor(GranularityWeekly).Microseconds(),
			graceFor(GranularityMonthly).Microseconds(),
			fallbackGrace.Microseconds(),
		)).
		From("pm_aggregation_windows w").
		Where(sq.Eq{"w.status": []string{"open", "failed"}}).
		Where(due).
		Where(queueBarrierConsumedPredicate())
	return storage.Psql.Select(`
COALESCE(
    GREATEST(0, EXTRACT(EPOCH FROM CURRENT_TIMESTAMP - MIN(eligible.due_at))),
    0
)`).
		FromSelect(eligible, "eligible")
}

func (r *WindowRepository) OldestDue(
	ctx context.Context,
	graceByGranularity map[Granularity]time.Duration,
	fallbackGrace time.Duration,
) (time.Duration, error) {
	query, args, err := oldestDueSelect(graceByGranularity, fallbackGrace).ToSql()
	if err != nil {
		return 0, fmt.Errorf("build oldest due PM aggregation window SQL: %w", err)
	}
	var seconds float64
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&seconds); err != nil {
		return 0, fmt.Errorf("query oldest due PM aggregation window: %w", err)
	}
	return time.Duration(seconds * float64(time.Second)), nil
}

func (r *WindowRepository) CountWatermarkBlocked(
	ctx context.Context,
	now time.Time,
	graceByGranularity map[Granularity]time.Duration,
	fallbackGrace time.Duration,
) (int64, error) {
	query, args, err := countWatermarkBlockedSelect(
		now, graceByGranularity, fallbackGrace,
	).ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count PM windows blocked by queue watermark: %w", err)
	}
	var count int64
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count PM windows blocked by queue watermark: %w", err)
	}
	return count, nil
}

func countWatermarkBlockedSelect(
	now time.Time,
	graceByGranularity map[Granularity]time.Duration,
	fallbackGrace time.Duration,
) sq.SelectBuilder {
	due := sq.Or{}
	for _, granularity := range []Granularity{
		GranularityHourly, GranularityDaily, GranularityWeekly, GranularityMonthly,
	} {
		grace := graceByGranularity[granularity]
		if grace <= 0 {
			grace = fallbackGrace
		}
		due = append(due, sq.And{
			sq.Eq{"w.granularity": string(granularity)},
			sq.LtOrEq{"w.window_end": now.Add(-grace)},
		})
	}
	dueWindows := storage.Psql.
		Select(
			"w.granularity",
			"w.window_start",
			"w.window_end",
			"COUNT(*) AS window_count",
		).
		From("pm_aggregation_windows w").
		Where(sq.Eq{"w.status": []string{"open", "failed"}}).
		Where(due).
		GroupBy("w.granularity", "w.window_start", "w.window_end")

	return storage.Psql.
		Select("COALESCE(SUM(due_windows.window_count), 0)::bigint").
		FromSelect(dueWindows, "due_windows").
		Where(queueBarrierPendingForDueWindowsPredicate())
}

func queueBarrierPendingForDueWindowsPredicate() sq.Sqlizer {
	return sq.Expr(`
(
EXISTS (
    SELECT 1
    FROM pm_aggregation_outbox source_event
    WHERE due_windows.granularity = 'hourly'
      AND source_event.consumed_at IS NULL
      AND source_event.barrier_eligible
      AND source_event.event_window_start >= due_windows.window_start
      AND source_event.event_window_start < due_windows.window_end
)
OR EXISTS (
    SELECT 1
    FROM pm_aggregation_rollup_outbox source_rollup
    WHERE source_rollup.consumed_at IS NULL
      AND source_rollup.barrier_eligible
      AND source_rollup.window_start >= due_windows.window_start
      AND source_rollup.window_start < due_windows.window_end
      AND EXISTS (
          SELECT 1 FROM pm_aggregation_windows rollup_window
          WHERE rollup_window.task_version_id = source_rollup.publication_task_version_id
            AND rollup_window.entity_key = source_rollup.entity_key
            AND rollup_window.granularity = source_rollup.granularity
            AND rollup_window.window_start = source_rollup.window_start
            AND rollup_window.published_revision = source_rollup.revision
      )
      AND (
          (due_windows.granularity = 'daily' AND source_rollup.subject = ?)
          OR
          (due_windows.granularity IN ('weekly', 'monthly') AND source_rollup.subject = ?)
      )
)
)`,
		"pmaggregation.hourly.rollup",
		"pmaggregation.daily.rollup",
	)
}

func (r *WindowRepository) ListActive(ctx context.Context, limit uint64) ([]WindowRecord, error) {
	return r.ListActiveAfter(ctx, nil, limit)
}

func (r *WindowRepository) ListActiveAfter(
	ctx context.Context,
	after *WindowKey,
	limit uint64,
) ([]WindowRecord, error) {
	builder := storage.Psql.Select(
		"task_id", "task_version_id", "entity_key", "granularity", "window_start", "window_end",
		"status", "expected_slots", "received_slots", "finalize_attempts",
	).From("pm_aggregation_windows").
		Where(sq.Eq{"status": []string{"open", "failed", "finalizing"}}).
		OrderBy("window_start", "task_version_id", "entity_key", "granularity").
		Limit(limit).
		PlaceholderFormat(sq.Dollar)
	if after != nil {
		builder = builder.Where(sq.Expr(
			"(window_start, task_version_id, entity_key, granularity) > (?, ?, ?, ?)",
			after.Start, after.TaskVersionID, after.EntityKey, string(after.Granularity),
		))
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list active PM aggregation windows SQL: %w", err)
	}
	return r.queryWindows(ctx, query, args...)
}

func (r *WindowRepository) MarkRecoveryTerminalBatch(
	ctx context.Context,
	records []WindowRecord,
	status string,
	reason string,
) (int64, error) {
	if len(records) == 0 {
		return 0, nil
	}
	if status != "orphaned" && status != "retired" {
		return 0, fmt.Errorf("unsupported PM aggregation recovery terminal status %q", status)
	}
	predicates := make(sq.Or, 0, len(records))
	for _, record := range records {
		predicates = append(predicates, windowKeyPredicate(record.Key))
	}
	now := time.Now().UTC()
	query, args, err := storage.Psql.Update("pm_aggregation_windows").
		Set("recovery_original_status", sq.Expr("status")).
		Set("status", status).
		Set("recovery_terminal_at", now).
		Set("recovery_terminal_reason", reason).
		Set("runtime_cleaned_at", nil).
		Set("last_error", reason).
		Set("finalize_lease_owner", nil).
		Set("finalize_lease_until", nil).
		Set("updated_at", now).
		Where(sq.Eq{"status": []string{"open", "failed", "finalizing"}}).
		Where(predicates).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build mark PM aggregation recovery terminal SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("mark PM aggregation recovery terminal: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *WindowRepository) ListRecoveryRuntimeCleanupPending(
	ctx context.Context,
	limit uint64,
) ([]WindowRecord, error) {
	query, args, err := storage.Psql.Select(
		"task_id", "task_version_id", "entity_key", "granularity", "window_start", "window_end",
		"status", "expected_slots", "received_slots", "finalize_attempts",
	).From("pm_aggregation_windows").
		Where(sq.Eq{"status": []string{"orphaned", "retired"}, "runtime_cleaned_at": nil}).
		OrderBy("recovery_terminal_at", "task_version_id", "entity_key", "granularity", "window_start").
		Limit(limit).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list PM aggregation recovery cleanup SQL: %w", err)
	}
	return r.queryWindows(ctx, query, args...)
}

func (r *WindowRepository) MarkRecoveryRuntimeCleaned(
	ctx context.Context,
	keys []WindowKey,
) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	predicates := make(sq.Or, 0, len(keys))
	for _, key := range keys {
		predicates = append(predicates, windowKeyPredicate(key))
	}
	query, args, err := storage.Psql.Update("pm_aggregation_windows").
		Set("runtime_cleaned_at", time.Now().UTC()).
		Set("updated_at", time.Now().UTC()).
		Where(sq.Eq{"status": []string{"orphaned", "retired"}, "runtime_cleaned_at": nil}).
		Where(predicates).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build mark PM aggregation runtime cleaned SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("mark PM aggregation runtime cleaned: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *WindowRepository) queryWindows(
	ctx context.Context,
	query string,
	args ...any,
) ([]WindowRecord, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query PM aggregation windows: %w", err)
	}
	defer rows.Close()
	return scanWindowRows(rows)
}

func scanWindowRows(rows pgx.Rows) ([]WindowRecord, error) {
	var result []WindowRecord
	for rows.Next() {
		var record WindowRecord
		if err := rows.Scan(
			&record.Key.TaskID, &record.Key.TaskVersionID, &record.Key.EntityKey, &record.Key.Granularity,
			&record.Key.Start, &record.Key.End, &record.Status,
			&record.ExpectedSlots, &record.ReceivedSlots, &record.FinalizeAttempts,
		); err != nil {
			return nil, fmt.Errorf("scan PM aggregation window: %w", err)
		}
		result = append(result, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate PM aggregation windows: %w", err)
	}
	return result, nil
}

func (r *WindowRepository) MarkFailed(ctx context.Context, key WindowKey, cause error) error {
	query, args, err := storage.Psql.Update("pm_aggregation_windows").
		Set("status", "failed").
		Set("last_error", cause.Error()).
		Set("updated_at", time.Now().UTC()).
		Where(windowKeyPredicate(key)).
		Where(sq.NotEq{"status": "published"}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build mark PM aggregation window failed SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("mark PM aggregation window failed: %w", err)
	}
	return nil
}

func windowKeyPredicate(key WindowKey) sq.Eq {
	return sq.Eq{
		"task_version_id": key.TaskVersionID,
		"entity_key":      key.EntityKey,
		"granularity":     string(key.Granularity),
		"window_start":    key.Start,
	}
}
