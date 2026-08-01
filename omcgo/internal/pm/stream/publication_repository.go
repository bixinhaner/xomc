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
	if granularity == GranularityHourly && revision == 1 {
		return "prepared"
	}
	return "published"
}

func markPublicationPreparedQuery(key WindowKey, revision int) sq.InsertBuilder {
	return storage.Psql.Insert("pm_aggregation_publications").
		Columns(
			"task_id", "task_version_id", "granularity", "window_start", "window_end",
			"revision", "status", "expected_entities", "prepared_entities", "updated_at",
		).
		Values(
			key.TaskID, key.TaskVersionID, string(key.Granularity), key.Start, key.End,
			revision, "preparing", 1, 1, time.Now().UTC(),
		).
		Suffix(`ON CONFLICT (task_version_id, granularity, window_start) DO UPDATE SET
  task_id = EXCLUDED.task_id,
  window_end = EXCLUDED.window_end,
  revision = GREATEST(pm_aggregation_publications.revision, EXCLUDED.revision),
  status = 'preparing',
  expected_entities = pm_aggregation_publications.expected_entities + 1,
  prepared_entities = pm_aggregation_publications.prepared_entities + 1,
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
		"p.window_end", "p.revision",
	).From("pm_aggregation_publications p").
		Where(sq.Eq{"p.status": "preparing", "p.granularity": string(GranularityHourly)}).
		Where(sq.LtOrEq{"p.window_end": now.Add(-grace)}).
		Where(sq.Expr(`NOT EXISTS (
  SELECT 1 FROM pm_aggregation_windows w
  WHERE w.task_version_id = p.task_version_id
    AND w.granularity = p.granularity
    AND w.window_start = p.window_start
    AND w.status <> 'prepared'
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
		Set("status", "published").Set("published_at", now).Set("updated_at", now).
		Where(publicationWhere).Where(sq.Eq{"status": "prepared", "revision": publication.Revision}).
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
		Where(publicationWhere).Where(sq.Eq{"revision": publication.Revision}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build release prepared PM rollups: %w", err)
	}
	if _, err := tx.Exec(ctx, outboxQuery, outboxArgs...); err != nil {
		return nil, fmt.Errorf("release prepared PM rollups: %w", err)
	}
	snapshotQuery, snapshotArgs, err := storage.Psql.Update("pm_aggregation_counter_rollups").
		Set("publication_eligible", true).
		Where(publicationWhere).Where(sq.Eq{"revision": publication.Revision}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build release prepared PM rollup snapshots: %w", err)
	}
	if _, err := tx.Exec(ctx, snapshotQuery, snapshotArgs...); err != nil {
		return nil, fmt.Errorf("release prepared PM rollup snapshots: %w", err)
	}

	publicationQuery, publicationArgs, err := storage.Psql.Update("pm_aggregation_publications").
		Set("status", "published").Set("watermark_at", now).Set("published_at", now).
		Set("expected_entities", len(keys)).Set("prepared_entities", len(keys)).
		Set("dirty_entities", 0).Set("updated_at", now).
		Where(publicationWhere).Where(sq.Eq{"status": "preparing", "revision": publication.Revision}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build switch PM publication watermark: %w", err)
	}
	if _, err := tx.Exec(ctx, publicationQuery, publicationArgs...); err != nil {
		return nil, fmt.Errorf("switch PM publication watermark: %w", err)
	}
	return keys, nil
}
