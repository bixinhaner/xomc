package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/event"
	"github.com/omcgo/omcgo/internal/core/storage"
	"go.uber.org/zap"
)

type RollupOutboxRecord struct {
	EventID    uuid.UUID
	Subject    string
	Payload    RollupPayload
	ClaimToken uuid.UUID
}

var maxRollupEventBytes = DefaultConfig().MaxEventBytes

func SetMaxRollupEventBytes(limit int) {
	if limit > 0 {
		maxRollupEventBytes = limit
	}
}

type RollupOutboxRepository struct {
	pool *pgxpool.Pool
}

type periodRebuildIndexDB interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

const periodRebuildIndexSQL = `
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_pm_counter_rollups_period_rebuild_page
ON public.pm_aggregation_counter_rollups (
  task_version_id, granularity, window_start, entity_key, chunk_index,
  publication_task_version_id, revision, event_id
)`

const periodRebuildIndexDropSQL = `DROP INDEX CONCURRENTLY IF EXISTS public.idx_pm_counter_rollups_period_rebuild_page`
const periodRebuildIndexLockSQL = `SELECT pg_advisory_lock(hashtextextended('omc.pm.rollup.period_rebuild_index', 0))`
const periodRebuildIndexUnlockSQL = `SELECT pg_advisory_unlock(hashtextextended('omc.pm.rollup.period_rebuild_index', 0))`

const revisionCleanupCounterRollupsIndexSQL = `
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_pm_counter_rollups_revision_cleanup
ON public.pm_aggregation_counter_rollups (
  publication_task_version_id, entity_key, granularity, window_start, revision, event_id
)`

const revisionCleanupOutboxIndexSQL = `
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_pm_rollup_outbox_revision_cleanup
ON public.pm_aggregation_rollup_outbox (
  publication_task_version_id, entity_key, granularity, window_start, revision, event_id
)`

const revisionCleanupIndexLockSQL = `SELECT pg_advisory_lock(hashtextextended('omc.pm.rollup.revision_cleanup_indexes', 0))`
const revisionCleanupIndexUnlockSQL = `SELECT pg_advisory_unlock(hashtextextended('omc.pm.rollup.revision_cleanup_indexes', 0))`

const rollupSnapshotPageSize uint64 = 256

type rollupSnapshotCursor struct {
	TaskVersionID            uuid.UUID
	WindowStart              time.Time
	EntityKey                string
	ChunkIndex               int
	PublicationTaskVersionID uuid.UUID
	Revision                 int
	EventID                  uuid.UUID
}

type rollupSnapshotPageRow struct {
	cursor  rollupSnapshotCursor
	payload RollupPayload
}

func rollupEventIDForRevision(eventID uuid.UUID, revision int) uuid.UUID {
	if revision <= 1 {
		return eventID
	}
	return uuid.NewSHA1(eventID, []byte(fmt.Sprintf("revision:%d", revision)))
}

func NewRollupOutboxRepository(pool *pgxpool.Pool) *RollupOutboxRepository {
	return &RollupOutboxRepository{pool: pool}
}

// EnsurePeriodRebuildIndex upgrades older databases without blocking hot rollup
// writers. The session lock serializes rolling worker replicas.
func (r *RollupOutboxRepository) EnsurePeriodRebuildIndex(ctx context.Context) (returnErr error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire PM rollup period rebuild index connection: %w", err)
	}
	locked := false
	lockStateUncertain := false
	defer func() {
		if !locked {
			if lockStateUncertain {
				rawConn := conn.Hijack()
				closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = rawConn.Close(closeCtx)
				return
			}
			conn.Release()
			return
		}
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var unlocked bool
		if err := conn.QueryRow(unlockCtx, periodRebuildIndexUnlockSQL).Scan(&unlocked); err == nil && unlocked {
			conn.Release()
			return
		} else if returnErr == nil {
			if err != nil {
				returnErr = fmt.Errorf("release PM rollup period rebuild index advisory lock: %w", err)
			} else {
				returnErr = errors.New("release PM rollup period rebuild index advisory lock: lock was not held")
			}
		}
		rawConn := conn.Hijack()
		_ = rawConn.Close(unlockCtx)
	}()
	lockStateUncertain = true
	if _, err := conn.Exec(ctx, periodRebuildIndexLockSQL); err != nil {
		return fmt.Errorf("acquire PM rollup period rebuild index advisory lock: %w", err)
	}
	lockStateUncertain = false
	locked = true
	return ensurePeriodRebuildIndex(ctx, conn)
}

func ensurePeriodRebuildIndex(ctx context.Context, db periodRebuildIndexDB) error {
	const qualifiedName = "public.idx_pm_counter_rollups_period_rebuild_page"
	var exists, valid bool
	if err := db.QueryRow(ctx, `
SELECT to_regclass($1) IS NOT NULL,
       COALESCE((SELECT indisvalid AND indpred IS NULL
                 FROM pg_index WHERE indexrelid = to_regclass($1)), false)`,
		qualifiedName,
	).Scan(&exists, &valid); err != nil {
		return fmt.Errorf("inspect PM rollup period rebuild index: %w", err)
	}
	if valid {
		return nil
	}
	if exists {
		if _, err := db.Exec(ctx, periodRebuildIndexDropSQL); err != nil {
			return fmt.Errorf("drop invalid PM rollup period rebuild index: %w", err)
		}
	}
	if _, err := db.Exec(ctx, periodRebuildIndexSQL); err != nil {
		return fmt.Errorf("ensure PM rollup period rebuild index: %w", err)
	}
	return nil
}

func (r *RollupOutboxRepository) EnsureRevisionCleanupIndexes(ctx context.Context) (returnErr error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire PM rollup revision cleanup index connection: %w", err)
	}
	locked := false
	lockStateUncertain := false
	defer func() {
		if !locked {
			if lockStateUncertain {
				rawConn := conn.Hijack()
				closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = rawConn.Close(closeCtx)
				return
			}
			conn.Release()
			return
		}
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var unlocked bool
		if err := conn.QueryRow(unlockCtx, revisionCleanupIndexUnlockSQL).Scan(&unlocked); err == nil && unlocked {
			conn.Release()
			return
		} else if returnErr == nil {
			if err != nil {
				returnErr = fmt.Errorf("release PM rollup revision cleanup index advisory lock: %w", err)
			} else {
				returnErr = errors.New("release PM rollup revision cleanup index advisory lock: lock was not held")
			}
		}
		rawConn := conn.Hijack()
		_ = rawConn.Close(unlockCtx)
	}()
	lockStateUncertain = true
	if _, err := conn.Exec(ctx, revisionCleanupIndexLockSQL); err != nil {
		return fmt.Errorf("acquire PM rollup revision cleanup index advisory lock: %w", err)
	}
	lockStateUncertain = false
	locked = true
	return ensureRevisionCleanupIndexes(ctx, conn)
}

func ensureRevisionCleanupIndexes(ctx context.Context, db periodRebuildIndexDB) error {
	indexes := []struct {
		qualifiedName string
		createSQL     string
	}{
		{
			qualifiedName: "public.idx_pm_counter_rollups_revision_cleanup",
			createSQL:     revisionCleanupCounterRollupsIndexSQL,
		},
		{
			qualifiedName: "public.idx_pm_rollup_outbox_revision_cleanup",
			createSQL:     revisionCleanupOutboxIndexSQL,
		},
	}
	for _, index := range indexes {
		var exists, valid bool
		if err := db.QueryRow(ctx, `
SELECT to_regclass($1) IS NOT NULL,
       COALESCE((SELECT indisvalid AND indpred IS NULL
                 FROM pg_index WHERE indexrelid = to_regclass($1)), false)`,
			index.qualifiedName,
		).Scan(&exists, &valid); err != nil {
			return fmt.Errorf("inspect PM rollup revision cleanup index %s: %w", index.qualifiedName, err)
		}
		if valid {
			continue
		}
		if exists {
			if _, err := db.Exec(ctx, "DROP INDEX CONCURRENTLY IF EXISTS "+index.qualifiedName); err != nil {
				return fmt.Errorf("drop invalid PM rollup revision cleanup index %s: %w", index.qualifiedName, err)
			}
		}
		if _, err := db.Exec(ctx, index.createSQL); err != nil {
			return fmt.Errorf("ensure PM rollup revision cleanup index %s: %w", index.qualifiedName, err)
		}
	}
	return nil
}

func insertRollupTx(ctx context.Context, tx pgx.Tx, payload RollupPayload) error {
	return insertRollupTxWithEligibility(ctx, tx, payload, payload.TaskVersionID, true, 1)
}

func insertRollupTxWithEligibility(
	ctx context.Context,
	tx pgx.Tx,
	payload RollupPayload,
	publicationTaskVersionID uuid.UUID,
	eligible bool,
	revision int,
) error {
	if err := payload.Validate(); err != nil {
		return err
	}
	payload.EventID = rollupEventIDForRevision(payload.EventID, revision)
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal PM compact rollup: %w", err)
	}
	if len(data) > maxRollupEventBytes {
		return fmt.Errorf(
			"PM compact rollup is %d bytes, exceeds limit %d",
			len(data), maxRollupEventBytes,
		)
	}
	subject := event.SubjectPMAggregationHourlyRollup
	if payload.SourceGranularity == GranularityDaily {
		subject = event.SubjectPMAggregationDailyRollup
	}
	snapshotSQL, snapshotArgs, err := storage.Psql.Insert("pm_aggregation_counter_rollups").
		Columns(
			"event_id", "task_id", "task_version_id", "publication_task_version_id", "entity_key", "granularity",
			"window_start", "window_end", "chunk_index", "chunk_count",
			"complete", "revision", "publication_eligible", "payload",
		).
		Values(
			payload.EventID, payload.TaskID, payload.TaskVersionID, publicationTaskVersionID, payload.EntityKey,
			string(payload.SourceGranularity), payload.WindowStart, payload.WindowEnd,
			payload.ChunkIndex, payload.ChunkCount, payload.Complete, revision, eligible, json.RawMessage(data),
		).
		Suffix(`ON CONFLICT (event_id) DO UPDATE SET
			payload = EXCLUDED.payload,
			task_id = EXCLUDED.task_id,
			task_version_id = EXCLUDED.task_version_id,
			publication_task_version_id = EXCLUDED.publication_task_version_id,
			entity_key = EXCLUDED.entity_key,
			granularity = EXCLUDED.granularity,
			window_start = EXCLUDED.window_start,
			window_end = EXCLUDED.window_end,
			chunk_index = EXCLUDED.chunk_index,
			chunk_count = EXCLUDED.chunk_count,
			complete = EXCLUDED.complete,
			revision = EXCLUDED.revision,
			publication_eligible = EXCLUDED.publication_eligible,
			created_at = now()`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build PM Counter rollup snapshot SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, snapshotSQL, snapshotArgs...); err != nil {
		return fmt.Errorf("insert PM Counter rollup snapshot: %w", err)
	}
	outboxSQL, outboxArgs, err := storage.Psql.Insert("pm_aggregation_rollup_outbox").
		Columns(
			"event_id", "subject", "task_version_id", "publication_task_version_id", "entity_key", "granularity", "window_start",
			"revision", "payload", "barrier_eligible",
		).
		Values(
			payload.EventID, subject, payload.TaskVersionID, publicationTaskVersionID, payload.EntityKey, string(payload.SourceGranularity),
			payload.WindowStart, revision, json.RawMessage(data), eligible,
		).
		Suffix(`ON CONFLICT (event_id) DO UPDATE SET
			payload = EXCLUDED.payload,
			subject = EXCLUDED.subject,
			task_version_id = EXCLUDED.task_version_id,
			publication_task_version_id = EXCLUDED.publication_task_version_id,
			entity_key = EXCLUDED.entity_key,
			granularity = EXCLUDED.granularity,
			window_start = EXCLUDED.window_start,
			published_at = NULL,
			consumed_at = NULL,
			claim_token = NULL,
			claim_expires_at = NULL,
			next_attempt_at = '-infinity'::timestamptz,
			barrier_eligible = EXCLUDED.barrier_eligible,
			revision = EXCLUDED.revision,
			publish_attempts = 0,
			last_error = NULL,
			created_at = now()`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build PM rollup outbox SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, outboxSQL, outboxArgs...); err != nil {
		return fmt.Errorf("insert PM rollup outbox: %w", err)
	}
	return nil
}

func deleteStaleRollupRevisionTx(
	ctx context.Context,
	tx pgx.Tx,
	key WindowKey,
	revision int,
	keep []uuid.UUID,
) error {
	if revision <= 1 {
		return nil
	}
	deleteRows := func(table string) error {
		builder := storage.Psql.Delete(table).Where(sq.Eq{
			"publication_task_version_id": key.TaskVersionID,
			"entity_key":                  key.EntityKey,
			"granularity":                 string(key.Granularity),
			"window_start":                key.Start,
			"revision":                    revision,
		})
		if len(keep) > 0 {
			builder = builder.Where(sq.NotEq{"event_id": keep})
		}
		query, args, err := builder.ToSql()
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return err
		}
		return nil
	}
	if err := deleteRows("pm_aggregation_counter_rollups"); err != nil {
		return fmt.Errorf("delete stale PM Counter rollup snapshots: %w", err)
	}
	if err := deleteRows("pm_aggregation_rollup_outbox"); err != nil {
		return fmt.Errorf("delete stale PM Counter rollup outbox rows: %w", err)
	}
	return nil
}

func (r *RollupOutboxRepository) MarkConsumed(ctx context.Context, eventID uuid.UUID) error {
	query, args, err := storage.Psql.Update("pm_aggregation_rollup_outbox").
		Set("consumed_at", time.Now().UTC()).
		Where(sq.Eq{"event_id": eventID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build mark PM rollup outbox consumed: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("mark PM rollup outbox consumed: %w", err)
	}
	return nil
}

func (r *RollupOutboxRepository) RequeueStaleUnconsumed(
	ctx context.Context,
	before time.Time,
) (int64, error) {
	query, args, err := staleBarrierRedeliveryUpdate(
		"pm_aggregation_rollup_outbox", before,
	).ToSql()
	if err != nil {
		return 0, fmt.Errorf("build requeue stale PM rollup outbox SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("requeue stale PM rollup outbox: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *RollupOutboxRepository) ClaimBatch(
	ctx context.Context,
	limit uint64,
	now time.Time,
	lease time.Duration,
) ([]RollupOutboxRecord, error) {
	if limit == 0 {
		limit = 1
	}
	if now.IsZero() {
		return nil, fmt.Errorf("claim PM rollup outbox batch: current time is required")
	}
	if lease <= 0 {
		return nil, fmt.Errorf("claim PM rollup outbox batch: lease must be positive")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin PM rollup outbox claim tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	query, args, err := pendingRollupOutboxSelect("event_id", "subject", "payload").
		Where(outboxClaimDue(now, "rollup")).
		OrderBy("rollup.created_at", "rollup.event_id").
		Limit(limit).
		Suffix("FOR UPDATE SKIP LOCKED").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build claim PM rollup outbox SQL: %w", err)
	}
	rows, err := tx.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("claim PM rollup outbox: %w", err)
	}
	defer rows.Close()
	out := make([]RollupOutboxRecord, 0, limit)
	for rows.Next() {
		var record RollupOutboxRecord
		var raw []byte
		if err := rows.Scan(&record.EventID, &record.Subject, &raw); err != nil {
			return nil, fmt.Errorf("scan PM rollup outbox claim: %w", err)
		}
		if err := json.Unmarshal(raw, &record.Payload); err != nil {
			return nil, fmt.Errorf("decode PM rollup outbox claim: %w", err)
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit empty PM rollup outbox claim: %w", err)
		}
		return nil, nil
	}
	ids := make([]uuid.UUID, 0, len(out))
	claimToken := uuid.New()
	claimExpiresAt := now.Add(lease)
	for i := range out {
		ids = append(ids, out[i].EventID)
		out[i].ClaimToken = claimToken
	}
	update, updateArgs, err := storage.Psql.Update("pm_aggregation_rollup_outbox").
		Set("claim_token", claimToken).
		Set("claim_expires_at", claimExpiresAt).
		Where(sq.Eq{"event_id": ids}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build mark PM rollup outbox claimed SQL: %w", err)
	}
	tag, err := tx.Exec(ctx, update, updateArgs...)
	if err != nil {
		return nil, fmt.Errorf("mark PM rollup outbox claimed: %w", err)
	}
	if tag.RowsAffected() != int64(len(out)) {
		return nil, fmt.Errorf("claim PM rollup outbox batch: claimed %d rows, want %d", tag.RowsAffected(), len(out))
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit PM rollup outbox claim: %w", err)
	}
	return out, nil
}

func pendingRollupOutboxSelect(columns ...string) sq.SelectBuilder {
	prefixed := make([]string, 0, len(columns))
	for _, column := range columns {
		prefixed = append(prefixed, "rollup."+column)
	}
	return storage.Psql.Select(prefixed...).
		From("pm_aggregation_rollup_outbox rollup").
		Join(`pm_aggregation_windows published_window
  ON published_window.task_version_id = rollup.publication_task_version_id
 AND published_window.entity_key = rollup.entity_key
 AND published_window.granularity = rollup.granularity
 AND published_window.window_start = rollup.window_start
 AND published_window.published_revision = rollup.revision`).
		Where("rollup.published_at IS NULL").
		Where("rollup.consumed_at IS NULL").
		Where(sq.Eq{"rollup.barrier_eligible": true})
}

func (r *RollupOutboxRepository) MarkPublished(
	ctx context.Context,
	eventID uuid.UUID,
	claimToken uuid.UUID,
	now time.Time,
) (bool, error) {
	query, args, err := markOutboxPublishedUpdate(
		"pm_aggregation_rollup_outbox", eventID, claimToken, now,
	).ToSql()
	if err != nil {
		return false, err
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *RollupOutboxRepository) MarkFailed(
	ctx context.Context,
	eventID uuid.UUID,
	claimToken uuid.UUID,
	publishErr error,
	now time.Time,
	retryAfter time.Duration,
) (bool, error) {
	query, args, err := markOutboxFailedUpdate(
		"pm_aggregation_rollup_outbox", eventID, claimToken, publishErr, now, retryAfter,
	).ToSql()
	if err != nil {
		return false, err
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *RollupOutboxRepository) ListSnapshots(
	ctx context.Context,
	taskVersionID uuid.UUID,
	entityKey string,
	granularity Granularity,
	start, end time.Time,
) ([]RollupPayload, error) {
	query, args, err := storage.Psql.Select("payload").
		From("pm_aggregation_counter_rollups rollup").
		Join(`pm_aggregation_windows published_window
  ON published_window.task_version_id = rollup.publication_task_version_id
 AND published_window.entity_key = rollup.entity_key
 AND published_window.granularity = rollup.granularity
 AND published_window.window_start = rollup.window_start
 AND published_window.published_revision = rollup.revision`).
		Where(sq.Eq{
			"rollup.task_version_id": taskVersionID,
			"rollup.entity_key":      entityKey,
			"rollup.granularity":     string(granularity),
		}).
		Where(sq.GtOrEq{"rollup.window_start": start}).
		Where(sq.Lt{"rollup.window_start": end}).
		OrderBy("rollup.window_start", "rollup.chunk_index").
		ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query PM Counter rollup snapshots: %w", err)
	}
	defer rows.Close()
	var out []RollupPayload
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var payload RollupPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			return nil, fmt.Errorf("decode PM Counter rollup snapshot: %w", err)
		}
		out = append(out, payload)
	}
	return out, rows.Err()
}

func counterRollupPeriodPageSelect(
	taskVersionIDs []uuid.UUID,
	granularity Granularity,
	start, end time.Time,
	cursor *rollupSnapshotCursor,
	limit uint64,
) sq.SelectBuilder {
	if limit == 0 {
		limit = rollupSnapshotPageSize
	}
	builder := storage.Psql.Select(
		"rollup.task_version_id", "rollup.window_start", "rollup.entity_key", "rollup.chunk_index",
		"rollup.publication_task_version_id", "rollup.revision", "rollup.event_id", "rollup.payload",
	).
		From("pm_aggregation_counter_rollups rollup").
		Join(`pm_aggregation_windows published_window
  ON published_window.task_version_id = rollup.publication_task_version_id
 AND published_window.entity_key = rollup.entity_key
 AND published_window.granularity = rollup.granularity
 AND published_window.window_start = rollup.window_start
 AND published_window.published_revision = rollup.revision`).
		Where("rollup.task_version_id = ANY(?)", taskVersionIDs).
		Where(sq.Eq{"rollup.granularity": string(granularity)}).
		Where(sq.GtOrEq{"rollup.window_start": start}).
		Where(sq.Lt{"rollup.window_start": end}).
		Where("published_window.published_revision IS NOT NULL")
	if cursor != nil {
		builder = builder.Where(`
(rollup.task_version_id, rollup.window_start, rollup.entity_key, rollup.chunk_index,
 rollup.publication_task_version_id, rollup.revision, rollup.event_id) > (?, ?, ?, ?, ?, ?, ?)`,
			cursor.TaskVersionID, cursor.WindowStart, cursor.EntityKey, cursor.ChunkIndex,
			cursor.PublicationTaskVersionID, cursor.Revision, cursor.EventID,
		)
	}
	return builder.OrderBy(
		"rollup.task_version_id", "rollup.window_start", "rollup.entity_key", "rollup.chunk_index",
		"rollup.publication_task_version_id", "rollup.revision", "rollup.event_id",
	).Limit(limit)
}

func (r *RollupOutboxRepository) VisitSnapshotsForPeriod(
	ctx context.Context,
	taskVersionIDs []uuid.UUID,
	granularity Granularity,
	start, end time.Time,
	visit func(RollupPayload) error,
) error {
	if len(taskVersionIDs) == 0 {
		return fmt.Errorf("PM Counter rollup replay task versions are empty")
	}
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{
		IsoLevel:   pgx.RepeatableRead,
		AccessMode: pgx.ReadOnly,
	})
	if err != nil {
		return fmt.Errorf("begin PM Counter rollup repeatable-read snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	if err := visitRollupSnapshotPages(
		func(cursor *rollupSnapshotCursor) ([]rollupSnapshotPageRow, error) {
			query, args, err := counterRollupPeriodPageSelect(
				taskVersionIDs, granularity, start, end, cursor, rollupSnapshotPageSize,
			).ToSql()
			if err != nil {
				return nil, err
			}
			rows, err := tx.Query(ctx, query, args...)
			if err != nil {
				return nil, fmt.Errorf("query PM Counter rollup period snapshot page: %w", err)
			}
			defer rows.Close()
			page := make([]rollupSnapshotPageRow, 0, rollupSnapshotPageSize)
			for rows.Next() {
				var row rollupSnapshotPageRow
				var raw []byte
				if err := rows.Scan(
					&row.cursor.TaskVersionID, &row.cursor.WindowStart, &row.cursor.EntityKey, &row.cursor.ChunkIndex,
					&row.cursor.PublicationTaskVersionID, &row.cursor.Revision, &row.cursor.EventID, &raw,
				); err != nil {
					return nil, err
				}
				if err := json.Unmarshal(raw, &row.payload); err != nil {
					return nil, fmt.Errorf("decode PM Counter rollup period snapshot: %w", err)
				}
				page = append(page, row)
			}
			if err := rows.Err(); err != nil {
				return nil, err
			}
			return page, nil
		},
		visit,
	); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit PM Counter rollup repeatable-read snapshot: %w", err)
	}
	return nil
}

func visitRollupSnapshotPages(
	fetch func(*rollupSnapshotCursor) ([]rollupSnapshotPageRow, error),
	visit func(RollupPayload) error,
) error {
	var cursor *rollupSnapshotCursor
	for {
		page, err := fetch(cursor)
		if err != nil {
			return err
		}
		if len(page) == 0 {
			return nil
		}
		for _, row := range page {
			if err := visit(row.payload); err != nil {
				return err
			}
		}
		next := page[len(page)-1].cursor
		cursor = &next
	}
}

type rebuildSnapshotSource interface {
	VisitSnapshotsForPeriod(
		context.Context,
		[]uuid.UUID,
		Granularity,
		time.Time,
		time.Time,
		func(RollupPayload) error,
	) error
}

func visitSnapshotsForRebuildBatch(
	ctx context.Context,
	source rebuildSnapshotSource,
	taskVersionIDs []uuid.UUID,
	granularity Granularity,
	start, end time.Time,
	dispatch func(RollupPayload) error,
) (int, error) {
	rows := 0
	err := source.VisitSnapshotsForPeriod(
		ctx, taskVersionIDs, granularity, start, end,
		func(payload RollupPayload) error {
			rows++
			return dispatch(payload)
		},
	)
	return rows, err
}

type RollupOutboxRelay struct {
	repo            rollupOutboxRelayRepository
	bus             event.EventBus
	logger          *zap.Logger
	metrics         *Metrics
	batch           int
	interval        time.Duration
	claimLease      time.Duration
	retryAfter      time.Duration
	redeliveryAfter time.Duration
	redeliveryEvery time.Duration
}

type rollupOutboxRelayRepository interface {
	ClaimBatch(context.Context, uint64, time.Time, time.Duration) ([]RollupOutboxRecord, error)
	MarkPublished(context.Context, uuid.UUID, uuid.UUID, time.Time) (bool, error)
	MarkFailed(context.Context, uuid.UUID, uuid.UUID, error, time.Time, time.Duration) (bool, error)
	RequeueStaleUnconsumed(context.Context, time.Time) (int64, error)
}

func NewRollupOutboxRelay(
	repo *RollupOutboxRepository,
	bus event.EventBus,
	logger *zap.Logger,
) *RollupOutboxRelay {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RollupOutboxRelay{
		repo: repo, bus: bus, logger: logger, batch: 100, interval: 200 * time.Millisecond,
		claimLease:      30 * time.Second,
		retryAfter:      time.Second,
		redeliveryAfter: 5 * time.Minute, redeliveryEvery: time.Minute,
	}
}

func (r *RollupOutboxRelay) SetBatch(batch int) *RollupOutboxRelay {
	if batch > 0 {
		r.batch = batch
	}
	return r
}

func (r *RollupOutboxRelay) SetMetrics(metrics *Metrics) *RollupOutboxRelay {
	r.metrics = metrics
	return r
}

func (r *RollupOutboxRelay) Run(ctx context.Context) error {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()
	go runRedeliveryLoop(ctx, r.redeliveryEvery, func() {
		r.requeueStale(ctx)
	})
	for {
		published, err := r.publishBatch(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			if r.metrics != nil {
				r.metrics.RollupOutboxErrorsTotal.Inc()
			}
			r.logger.Warn("publish PM rollup outbox", zap.Error(err))
		}
		if published {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (r *RollupOutboxRelay) requeueStale(ctx context.Context) {
	count, err := r.repo.RequeueStaleUnconsumed(
		ctx, time.Now().UTC().Add(-r.redeliveryAfter),
	)
	if err != nil {
		if r.metrics != nil {
			r.metrics.RollupOutboxErrorsTotal.Inc()
		}
		r.logger.Warn("requeue stale PM rollup outbox", zap.Error(err))
		return
	}
	if count > 0 {
		r.logger.Warn("requeued stale PM rollup events after consumer acknowledgement timeout",
			zap.Int64("events", count),
			zap.Duration("ack_timeout", r.redeliveryAfter))
	}
}

func (r *RollupOutboxRelay) publishBatch(ctx context.Context) (bool, error) {
	rows, err := r.repo.ClaimBatch(ctx, uint64(r.batch), time.Now().UTC(), r.claimLease)
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	var publishErrors []error
	publishedCount := 0
	for _, row := range rows {
		envelope, eventErr := event.NewEvent(row.Subject, row.Payload)
		if eventErr != nil {
			return false, eventErr
		}
		envelope.ID = row.EventID.String()
		if publishErr := r.bus.Publish(ctx, row.Subject, envelope); publishErr != nil {
			ok, markErr := r.repo.MarkFailed(
				ctx, row.EventID, row.ClaimToken, publishErr, time.Now().UTC(), r.retryAfter,
			)
			if markErr != nil {
				return false, errors.Join(publishErr, markErr)
			}
			if !ok {
				return false, errors.Join(publishErr, fmt.Errorf("PM rollup outbox claim ownership lost for %s", row.EventID))
			}
			publishErrors = append(publishErrors, publishErr)
			continue
		}
		ok, err := r.repo.MarkPublished(ctx, row.EventID, row.ClaimToken, time.Now().UTC())
		if err != nil {
			return false, err
		}
		if !ok {
			return false, fmt.Errorf("PM rollup outbox claim ownership lost for %s", row.EventID)
		}
		publishedCount++
	}
	if r.metrics != nil {
		r.metrics.RollupOutboxPublishedTotal.Add(float64(publishedCount))
	}
	if len(publishErrors) > 0 {
		return true, errors.Join(publishErrors...)
	}
	return true, nil
}
