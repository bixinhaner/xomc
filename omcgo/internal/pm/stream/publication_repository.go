package stream

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type PublicationRecord struct {
	TaskID        uuid.UUID
	TaskVersionID uuid.UUID
	Granularity   Granularity
	WindowStart   time.Time
	WindowEnd     time.Time
	Revision      int
}

func finalWindowStatus(granularity Granularity, revision int) string {
	return "prepared"
}

func requiresPublicationLock(revision int) bool {
	return revision > 1
}

func ensureInitialPublicationQuery(key WindowKey) sq.InsertBuilder {
	return storage.Psql.Insert("pm_aggregation_publications").
		Columns(
			"task_id", "task_version_id", "granularity", "window_start", "window_end",
			"revision", "preparing_revision", "status", "expected_entities", "prepared_entities", "updated_at",
		).
		Values(
			key.TaskID, key.TaskVersionID, string(key.Granularity), key.Start, key.End,
			0, 1, "preparing", 0, 0, time.Now().UTC(),
		).
		Suffix("ON CONFLICT (task_version_id, granularity, window_start) DO NOTHING")
}

func markPublicationPreparedQuery(key WindowKey, revision int) sq.InsertBuilder {
	return storage.Psql.Insert("pm_aggregation_publications").
		Columns(
			"task_id", "task_version_id", "granularity", "window_start", "window_end",
			"revision", "preparing_revision", "status", "expected_entities", "prepared_entities", "updated_at",
		).
		Values(
			key.TaskID, key.TaskVersionID, string(key.Granularity), key.Start, key.End,
			0, revision, "preparing", 1, 1, time.Now().UTC(),
		).
		Suffix(`ON CONFLICT (task_version_id, granularity, window_start) DO UPDATE SET
  task_id = EXCLUDED.task_id,
  window_end = EXCLUDED.window_end,
  preparing_revision = GREATEST(
    COALESCE(pm_aggregation_publications.preparing_revision, 0),
    EXCLUDED.preparing_revision
  ),
  status = CASE WHEN pm_aggregation_publications.revision = 0 THEN 'preparing'
                ELSE pm_aggregation_publications.status END,
  updated_at = EXCLUDED.updated_at`)
}

func publishReadyCandidatesQuery(
	now time.Time,
	grace time.Duration,
	limit uint64,
) sq.SelectBuilder {
	if limit == 0 {
		limit = 1
	}
	return storage.Psql.Select(
		"p.task_id", "p.task_version_id", "p.granularity", "p.window_start",
		"p.window_end", "COALESCE(p.preparing_revision, p.revision)",
	).From("pm_aggregation_publications p").
		Where(sq.Or{sq.Eq{"p.status": "preparing"}, sq.Expr("p.preparing_revision IS NOT NULL")}).
		Where(sq.Or{
			sq.And{sq.Eq{"p.granularity": string(GranularityHourly)}, sq.LtOrEq{"p.window_end": now.Add(-grace)}},
			sq.And{sq.Eq{"p.granularity": string(GranularityDaily)}, sq.LtOrEq{"p.window_end": now.Add(-15 * time.Minute)}},
			sq.And{sq.Eq{"p.granularity": string(GranularityWeekly)}, sq.LtOrEq{"p.window_end": now.Add(-30 * time.Minute)}},
			sq.And{sq.Eq{"p.granularity": string(GranularityMonthly)}, sq.LtOrEq{"p.window_end": now.Add(-30 * time.Minute)}},
		}).
		Where(sq.Expr(`NOT EXISTS (
  SELECT 1 FROM pm_aggregation_windows w
  WHERE w.task_version_id = p.task_version_id
    AND w.granularity = p.granularity
    AND w.window_start = p.window_start
    AND w.status NOT IN ('prepared', 'published')
)`)).
		Where(sq.Expr(`NOT EXISTS (
  SELECT 1 FROM pm_aggregation_outbox source_event
  WHERE source_event.consumed_at IS NULL
    AND source_event.barrier_eligible
    AND source_event.event_window_start >= p.window_start
    AND source_event.event_window_start < p.window_end
)`)).
		OrderBy("p.window_end", "p.task_version_id", "p.window_start").
		Limit(limit).
		Suffix("FOR UPDATE SKIP LOCKED")
}

// preparePublicationRevision preserves the currently visible generation while
// a late rebuild is computed. The first entity entering a new revision clones
// the complete previous generation in the same transaction; entity-specific
// replacement can then overwrite/delete only its rows. Readers continue to
// join the publication's old revision until the final atomic switch.
func preparePublicationRevision(
	ctx context.Context,
	tx pgx.Tx,
	key WindowKey,
	revision int,
) error {
	var currentRevision int
	var preparingRevision *int
	err := tx.QueryRow(ctx, `
SELECT revision, preparing_revision
FROM pm_aggregation_publications
WHERE task_version_id = $1 AND granularity = $2 AND window_start = $3
FOR UPDATE`, key.TaskVersionID, string(key.Granularity), key.Start).Scan(&currentRevision, &preparingRevision)
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("lock PM aggregation publication: %w", err)
	}
	if err == pgx.ErrNoRows {
		_, err = tx.Exec(ctx, `
INSERT INTO pm_aggregation_publications (
  task_id, task_version_id, granularity, window_start, window_end,
  revision, preparing_revision, status, updated_at
) VALUES ($1, $2, $3, $4, $5, 0, $6, 'preparing', now())
ON CONFLICT (task_version_id, granularity, window_start) DO NOTHING`,
			key.TaskID, key.TaskVersionID, string(key.Granularity), key.Start, key.End, revision)
		if err != nil {
			return fmt.Errorf("create PM aggregation publication: %w", err)
		}
		return nil
	}
	if currentRevision >= revision || (preparingRevision != nil && *preparingRevision >= revision) {
		return nil
	}
	if _, err := tx.Exec(ctx, `
INSERT INTO pm_aggregation_results (
  window_start, window_end, task_id, task_version_id,
  granularity, dimension, dimension_key, dimension_name,
  object_ldn, device_oui, device_sn, technology,
  metric_id, metric_path, metric_type, aggregation_op, metric_value,
  sample_count, complete, missing_slots, revision,
  version_effective_from, version_effective_to, received_slots,
  expected_slots, version_expected_slots, natural_expected_slots,
  version_slice_complete, period_complete
)
SELECT
  window_start, window_end, task_id, task_version_id,
  granularity, dimension, dimension_key, dimension_name,
  object_ldn, device_oui, device_sn, technology,
  metric_id, metric_path, metric_type, aggregation_op, metric_value,
  sample_count, complete, missing_slots, $4,
  version_effective_from, version_effective_to, received_slots,
  expected_slots, version_expected_slots, natural_expected_slots,
  version_slice_complete, period_complete
FROM pm_aggregation_results
WHERE task_version_id = $1 AND granularity = $2
  AND window_start = $3 AND revision = $5
ON CONFLICT (
  task_version_id, granularity, window_start, dimension_key,
  object_ldn, technology, metric_id, revision
) DO NOTHING`, key.TaskVersionID, string(key.Granularity), key.Start, revision, currentRevision); err != nil {
		return fmt.Errorf("clone PM aggregation publication revision: %w", err)
	}
	if _, err := tx.Exec(ctx, `
UPDATE pm_aggregation_publications
SET preparing_revision = $4, dirty_entities = 0, updated_at = now()
WHERE task_version_id = $1 AND granularity = $2 AND window_start = $3`,
		key.TaskVersionID, string(key.Granularity), key.Start, revision); err != nil {
		return fmt.Errorf("advance PM aggregation publication revision: %w", err)
	}
	return nil
}

func (r *WindowRepository) PublishHourlyReady(
	ctx context.Context,
	now time.Time,
	grace time.Duration,
	limit uint64,
) ([]WindowKey, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin publish prepared PM aggregation revisions: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query, args, err := publishReadyCandidatesQuery(now, grace, limit).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build prepared PM publication candidates: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query prepared PM publication candidates: %w", err)
	}
	var publications []PublicationRecord
	for rows.Next() {
		var record PublicationRecord
		if err := rows.Scan(
			&record.TaskID, &record.TaskVersionID, &record.Granularity,
			&record.WindowStart, &record.WindowEnd, &record.Revision,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan prepared PM publication: %w", err)
		}
		publications = append(publications, record)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate prepared PM publications: %w", err)
	}
	rows.Close()

	var published []WindowKey
	for _, publication := range publications {
		keys, err := publishPreparedRevision(ctx, tx, publication, now)
		if err != nil {
			return nil, err
		}
		published = append(published, keys...)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit prepared PM publications: %w", err)
	}
	return published, nil
}

func publishPreparedRevision(
	ctx context.Context,
	tx pgx.Tx,
	publication PublicationRecord,
	now time.Time,
) ([]WindowKey, error) {
	publicationWhere := sq.Eq{
		"task_version_id": publication.TaskVersionID,
		"granularity":     string(publication.Granularity),
		"window_start":    publication.WindowStart,
	}
	windowQuery, windowArgs, err := storage.Psql.Update("pm_aggregation_windows").
		Set("status", "published").Set("published_revision", publication.Revision).
		Set("published_at", now).Set("updated_at", now).
		Where(publicationWhere).
		Where(sq.Eq{"status": "prepared", "revision": publication.Revision}).
		Suffix("RETURNING task_id, task_version_id, entity_key, granularity, window_start, window_end").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build publish prepared PM windows: %w", err)
	}
	rows, err := tx.Query(ctx, windowQuery, windowArgs...)
	if err != nil {
		return nil, fmt.Errorf("publish prepared PM windows: %w", err)
	}
	var keys []WindowKey
	for rows.Next() {
		var key WindowKey
		if err := rows.Scan(
			&key.TaskID, &key.TaskVersionID, &key.EntityKey, &key.Granularity,
			&key.Start, &key.End,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan published PM window: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("iterate published PM windows: %w", err)
	}
	rows.Close()

	outboxQuery, outboxArgs, err := storage.Psql.Update("pm_aggregation_rollup_outbox").
		Set("barrier_eligible", true).
		Where(sq.Eq{
			"publication_task_version_id": publication.TaskVersionID,
			"granularity":                 string(publication.Granularity),
			"window_start":                publication.WindowStart,
			"revision":                    publication.Revision,
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build release prepared PM rollups: %w", err)
	}
	if _, err := tx.Exec(ctx, outboxQuery, outboxArgs...); err != nil {
		return nil, fmt.Errorf("release prepared PM rollups: %w", err)
	}
	snapshotQuery, snapshotArgs, err := storage.Psql.Update("pm_aggregation_counter_rollups").
		Set("publication_eligible", true).
		Where(sq.Eq{
			"publication_task_version_id": publication.TaskVersionID,
			"granularity":                 string(publication.Granularity),
			"window_start":                publication.WindowStart,
			"revision":                    publication.Revision,
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build release prepared PM rollup snapshots: %w", err)
	}
	if _, err := tx.Exec(ctx, snapshotQuery, snapshotArgs...); err != nil {
		return nil, fmt.Errorf("release prepared PM rollup snapshots: %w", err)
	}
	retireQuery, retireArgs, err := storage.Psql.Update("pm_aggregation_rollup_outbox").
		Set("barrier_eligible", false).
		Set("consumed_at", sq.Expr("COALESCE(consumed_at, ?)", now)).
		Where(sq.Eq{
			"publication_task_version_id": publication.TaskVersionID,
			"granularity":                 string(publication.Granularity),
			"window_start":                publication.WindowStart,
		}).Where(sq.Lt{"revision": publication.Revision}).
		Where(sq.Expr(`EXISTS (
  SELECT 1 FROM pm_aggregation_windows advanced_window
  WHERE advanced_window.task_version_id = pm_aggregation_rollup_outbox.publication_task_version_id
    AND advanced_window.entity_key = pm_aggregation_rollup_outbox.entity_key
    AND advanced_window.granularity = pm_aggregation_rollup_outbox.granularity
    AND advanced_window.window_start = pm_aggregation_rollup_outbox.window_start
    AND advanced_window.published_revision = ?
)`, publication.Revision)).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build retire superseded PM rollups: %w", err)
	}
	if _, err := tx.Exec(ctx, retireQuery, retireArgs...); err != nil {
		return nil, fmt.Errorf("retire superseded PM rollups: %w", err)
	}

	publicationQuery, publicationArgs, err := storage.Psql.Update("pm_aggregation_publications").
		Set("status", "published").Set("watermark_at", now).Set("published_at", now).
		Set("revision", publication.Revision).Set("preparing_revision", nil).
		Set("expected_entities", sq.Expr(`(
  SELECT COUNT(*) FROM pm_aggregation_windows counted
  WHERE counted.task_version_id = pm_aggregation_publications.task_version_id
    AND counted.granularity = pm_aggregation_publications.granularity
    AND counted.window_start = pm_aggregation_publications.window_start
    AND counted.status = 'published'
)`)).
		Set("prepared_entities", sq.Expr(`(
  SELECT COUNT(*) FROM pm_aggregation_windows counted
  WHERE counted.task_version_id = pm_aggregation_publications.task_version_id
    AND counted.granularity = pm_aggregation_publications.granularity
    AND counted.window_start = pm_aggregation_publications.window_start
    AND counted.status = 'published'
)`)).
		Set("dirty_entities", 0).Set("updated_at", now).
		Where(publicationWhere).Where(sq.Eq{"preparing_revision": publication.Revision}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build switch PM publication watermark: %w", err)
	}
	if _, err := tx.Exec(ctx, publicationQuery, publicationArgs...); err != nil {
		return nil, fmt.Errorf("switch PM publication watermark: %w", err)
	}
	return keys, nil
}
