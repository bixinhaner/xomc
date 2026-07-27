package rawcleanup

import (
	"context"
	"fmt"
	"sort"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const advisoryLockKey int64 = 0x524157434c45414e // "RAWCLEAN"

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

type PGRepository struct{ pool *pgxpool.Pool }
type pgLockLease struct{ conn *pgxpool.Conn }
type candidateQuery struct {
	sql  string
	args []any
}

func NewPGRepository(pool *pgxpool.Pool) *PGRepository { return &PGRepository{pool: pool} }

func (r *PGRepository) TryLock(ctx context.Context) (LockLease, bool, error) {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("acquire raw cleanup lock connection: %w", err)
	}
	var acquired bool
	if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", advisoryLockKey).Scan(&acquired); err != nil {
		conn.Release()
		return nil, false, fmt.Errorf("try raw cleanup advisory lock: %w", err)
	}
	if !acquired {
		conn.Release()
		return nil, false, nil
	}
	return &pgLockLease{conn: conn}, true, nil
}

func (l *pgLockLease) Valid(ctx context.Context) error {
	if l == nil || l.conn == nil || l.conn.Conn().IsClosed() {
		return fmt.Errorf("raw cleanup advisory-lock session is closed")
	}
	if err := l.conn.Ping(ctx); err != nil {
		return fmt.Errorf("ping raw cleanup advisory-lock session: %w", err)
	}
	return nil
}

func (l *pgLockLease) Release(ctx context.Context) {
	if l == nil || l.conn == nil {
		return
	}
	_, _ = l.conn.Exec(ctx, "SELECT pg_advisory_unlock($1)", advisoryLockKey)
	l.conn.Release()
	l.conn = nil
}

func (r *PGRepository) ListCandidates(ctx context.Context, cutoff time.Time, limit int) ([]Candidate, error) {
	if limit <= 0 {
		return nil, nil
	}
	perKind := limit
	all := make([]Candidate, 0, limit*4)
	for _, item := range []struct {
		table string
		kind  Kind
	}{{"pm_files", KindPM}, {"mr_files", KindMR}} {
		queries, err := buildCandidateQueries(item.table, cutoff, perKind)
		if err != nil {
			return nil, fmt.Errorf("build %s raw cleanup candidates: %w", item.kind, err)
		}
		for _, query := range queries {
			rows, err := r.pool.Query(ctx, query.sql, query.args...)
			if err != nil {
				return nil, fmt.Errorf("query %s raw cleanup candidates: %w", item.kind, err)
			}
			for rows.Next() {
				var candidate Candidate
				candidate.Kind = item.kind
				if err := rows.Scan(&candidate.ID, &candidate.ObjectPath, &candidate.CollectTime, &candidate.Attempts); err != nil {
					rows.Close()
					return nil, fmt.Errorf("scan %s raw cleanup candidate: %w", item.kind, err)
				}
				all = append(all, candidate)
			}
			if err := rows.Err(); err != nil {
				rows.Close()
				return nil, fmt.Errorf("iterate %s raw cleanup candidates: %w", item.kind, err)
			}
			rows.Close()
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].CollectTime.Equal(all[j].CollectTime) {
			return all[i].ID.String() < all[j].ID.String()
		}
		return all[i].CollectTime.Before(all[j].CollectTime)
	})
	if len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

func buildCandidateQueries(table string, cutoff time.Time, limit int) ([]candidateQuery, error) {
	populations := []struct {
		readyWhere string
		orderBy    []string
	}{
		{"raw_delete_next_attempt_at IS NULL", []string{"collect_time ASC", "id ASC"}},
		{"raw_delete_next_attempt_at <= NOW()", []string{"raw_delete_next_attempt_at ASC", "collect_time ASC", "id ASC"}},
	}
	queries := make([]candidateQuery, 0, len(populations))
	for _, population := range populations {
		query, args, err := psql.Select("id", "minio_path", "collect_time", "raw_delete_attempts").
			From(table).
			Where(sq.Lt{"collect_time": cutoff}).
			Where("raw_deleted_at IS NULL").
			Where(population.readyWhere).
			OrderBy(population.orderBy...).
			Limit(uint64(limit)).ToSql()
		if err != nil {
			return nil, err
		}
		queries = append(queries, candidateQuery{sql: query, args: args})
	}
	return queries, nil
}

func (r *PGRepository) MarkResults(ctx context.Context, results []DeleteResult, now time.Time) error {
	success := map[Kind][]uuid.UUID{KindPM: {}, KindMR: {}}
	for _, result := range results {
		if result.Err == nil {
			success[result.Candidate.Kind] = append(success[result.Candidate.Kind], result.Candidate.ID)
			continue
		}
		table := tableFor(result.Candidate.Kind)
		next := now.Add(RetryDelay(result.Candidate.Attempts))
		query, args, err := psql.Update(table).
			Set("raw_delete_attempts", sq.Expr("raw_delete_attempts + 1")).
			Set("raw_delete_next_attempt_at", next).
			Set("raw_delete_last_error", TruncateError(result.Err.Error())).
			Where(sq.Eq{"id": result.Candidate.ID, "raw_deleted_at": nil}).ToSql()
		if err != nil {
			return fmt.Errorf("build raw cleanup failure update: %w", err)
		}
		if _, err := r.pool.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("mark %s raw cleanup failure: %w", result.Candidate.Kind, err)
		}
	}
	for kind, ids := range success {
		if len(ids) == 0 {
			continue
		}
		query, args, err := psql.Update(tableFor(kind)).
			Set("raw_deleted_at", now).
			Set("raw_delete_next_attempt_at", nil).
			Set("raw_delete_last_error", nil).
			Where(sq.Eq{"id": ids, "raw_deleted_at": nil}).ToSql()
		if err != nil {
			return fmt.Errorf("build raw cleanup success update: %w", err)
		}
		if _, err := r.pool.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("mark %s raw cleanup success: %w", kind, err)
		}
	}
	return nil
}

func (r *PGRepository) RecentCreatedCount(ctx context.Context, since time.Time) (int64, error) {
	var total int64
	for _, table := range []string{"pm_files", "mr_files"} {
		query, args, err := psql.Select("COUNT(*)").From(table).Where(sq.GtOrEq{"created_at": since}).ToSql()
		if err != nil {
			return 0, fmt.Errorf("build recent %s count: %w", table, err)
		}
		var count int64
		if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
			return 0, fmt.Errorf("count recent %s: %w", table, err)
		}
		total += count
	}
	return total, nil
}

func (r *PGRepository) OldestExpired(ctx context.Context, cutoff time.Time, kind Kind) (time.Time, error) {
	query, args, err := psql.Select().Column(sq.Expr("COALESCE(MIN(collect_time), ?)", cutoff)).
		From(tableFor(kind)).Where(sq.Lt{"collect_time": cutoff}).
		Where("raw_deleted_at IS NULL").ToSql()
	if err != nil {
		return time.Time{}, fmt.Errorf("build oldest %s candidate: %w", kind, err)
	}
	var oldest time.Time
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&oldest); err != nil {
		return time.Time{}, fmt.Errorf("oldest %s candidate: %w", kind, err)
	}
	return oldest, nil
}

// CleanupMetadata removes only rows whose raw object and all retained
// time-series/idempotency dependencies are already gone. Work is bounded and
// transactional; a dependency appearing concurrently makes the final DELETE
// predicate skip that row.
func (r *PGRepository) CleanupMetadata(ctx context.Context, cutoff time.Time, limit int) (int64, error) {
	if limit <= 0 {
		return 0, nil
	}
	perKind := max(1, limit/2)
	pmIDs, err := r.metadataIDs(ctx, KindPM, cutoff, perKind)
	if err != nil {
		return 0, err
	}
	mrIDs, err := r.metadataIDs(ctx, KindMR, cutoff, perKind)
	if err != nil {
		return 0, err
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin raw metadata cleanup: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var deleted int64
	if len(pmIDs) > 0 {
		deleteOutbox, args, buildErr := psql.Delete("pm_aggregation_outbox").
			Where(sq.Eq{"source_file_id": pmIDs}).
			Where("published_at IS NOT NULL").
			Where("NOT EXISTS (SELECT 1 FROM pm_aggregation_counter_rollups r WHERE r.event_id = pm_aggregation_outbox.event_id)").ToSql()
		if buildErr != nil {
			return 0, fmt.Errorf("build PM outbox metadata cleanup: %w", buildErr)
		}
		if _, err := tx.Exec(ctx, deleteOutbox, args...); err != nil {
			return 0, fmt.Errorf("delete safe PM outbox rows: %w", err)
		}

		deleteBatches, args, buildErr := psql.Delete("pm_ingest_batches").
			Where(sq.Eq{"source_file_id": pmIDs}).
			Where("NOT EXISTS (SELECT 1 FROM pm_measurement_anchors a WHERE a.ingest_batch_id = pm_ingest_batches.ingest_batch_id)").
			Where("NOT EXISTS (SELECT 1 FROM pm_aggregation_outbox o WHERE o.ingest_batch_id = pm_ingest_batches.ingest_batch_id)").ToSql()
		if buildErr != nil {
			return 0, fmt.Errorf("build PM batch metadata cleanup: %w", buildErr)
		}
		if _, err := tx.Exec(ctx, deleteBatches, args...); err != nil {
			return 0, fmt.Errorf("delete safe PM ingest batches: %w", err)
		}

		deleteFiles, args, buildErr := psql.Delete("pm_files").
			Where(sq.Eq{"id": pmIDs}).
			Where("raw_deleted_at IS NOT NULL").
			Where("NOT EXISTS (SELECT 1 FROM pm_measurement_anchors a WHERE a.source_file_id = pm_files.id)").
			Where("NOT EXISTS (SELECT 1 FROM pm_aggregation_outbox o WHERE o.source_file_id = pm_files.id)").
			Where("NOT EXISTS (SELECT 1 FROM pm_ingest_batches b WHERE b.source_file_id = pm_files.id)").ToSql()
		if buildErr != nil {
			return 0, fmt.Errorf("build PM file metadata cleanup: %w", buildErr)
		}
		tag, execErr := tx.Exec(ctx, deleteFiles, args...)
		if execErr != nil {
			return 0, fmt.Errorf("delete safe PM file metadata: %w", execErr)
		}
		deleted += tag.RowsAffected()
	}
	if len(mrIDs) > 0 {
		deleteFiles, args, buildErr := psql.Delete("mr_files").
			Where(sq.Eq{"id": mrIDs}).
			Where("raw_deleted_at IS NOT NULL").
			Where("NOT EXISTS (SELECT 1 FROM mr_records r WHERE r.file_id = mr_files.id)").ToSql()
		if buildErr != nil {
			return 0, fmt.Errorf("build MR file metadata cleanup: %w", buildErr)
		}
		tag, execErr := tx.Exec(ctx, deleteFiles, args...)
		if execErr != nil {
			return 0, fmt.Errorf("delete safe MR file metadata: %w", execErr)
		}
		deleted += tag.RowsAffected()
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit raw metadata cleanup: %w", err)
	}
	return deleted, nil
}

func (r *PGRepository) metadataIDs(ctx context.Context, kind Kind, cutoff time.Time, limit int) ([]uuid.UUID, error) {
	table := tableFor(kind)
	builder := psql.Select("id").From(table).
		Where(sq.Lt{"collect_time": cutoff}).
		Where("raw_deleted_at IS NOT NULL").
		OrderBy("collect_time ASC", "id ASC").Limit(uint64(limit))
	if kind == KindPM {
		builder = builder.
			Where("NOT EXISTS (SELECT 1 FROM pm_measurement_anchors a WHERE a.source_file_id = pm_files.id)").
			Where(`NOT EXISTS (
				SELECT 1 FROM pm_aggregation_outbox o
				LEFT JOIN pm_aggregation_counter_rollups r ON r.event_id = o.event_id
				WHERE o.source_file_id = pm_files.id
				  AND (o.published_at IS NULL OR r.event_id IS NOT NULL)
			)`)
	} else {
		builder = builder.Where("NOT EXISTS (SELECT 1 FROM mr_records r WHERE r.file_id = mr_files.id)")
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build %s metadata candidates: %w", kind, err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query %s metadata candidates: %w", kind, err)
	}
	defer rows.Close()
	ids := make([]uuid.UUID, 0, limit)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan %s metadata candidate: %w", kind, err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s metadata candidates: %w", kind, err)
	}
	return ids, nil
}

func tableFor(kind Kind) string {
	if kind == KindMR {
		return "mr_files"
	}
	return "pm_files"
}

func RetryDelay(attempts int) time.Duration {
	if attempts < 0 {
		attempts = 0
	}
	delay := time.Minute << min(attempts, 6)
	if delay > time.Hour {
		return time.Hour
	}
	return delay
}

func TruncateError(message string) string {
	runes := []rune(message)
	if len(runes) > 512 {
		runes = runes[:512]
	}
	return string(runes)
}
