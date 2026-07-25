package stream

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/omcgo/omcgo/internal/core/storage"
)

type WindowRecord struct {
	Key           WindowKey
	Status        string
	ExpectedSlots int64
	ReceivedSlots int64
}

type WindowRepository struct {
	pool *pgxpool.Pool
}

func NewWindowRepository(pool *pgxpool.Pool) *WindowRepository {
	return &WindowRepository{pool: pool}
}

func (r *WindowRepository) EnsureOpen(ctx context.Context, contribution Contribution) error {
	query, args, err := storage.Psql.Insert("pm_aggregation_windows").
		Columns(
			"task_id", "task_version_id", "granularity",
			"window_start", "window_end", "expected_slots",
		).
		Values(
			contribution.Key.TaskID, contribution.Key.TaskVersionID,
			string(contribution.Key.Granularity), contribution.Key.Start,
			contribution.Key.End, contribution.ExpectedSlots,
		).
		Suffix("ON CONFLICT (task_version_id, granularity, window_start) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("build ensure PM aggregation window SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("ensure PM aggregation window: %w", err)
	}
	return nil
}

func (r *WindowRepository) ObserveReceived(
	ctx context.Context,
	key WindowKey,
	received int64,
) error {
	query, args, err := storage.Psql.Update("pm_aggregation_windows").
		Set("received_slots", received).
		Set("updated_at", time.Now().UTC()).
		Where(sq.Eq{
			"task_version_id": key.TaskVersionID,
			"granularity":     string(key.Granularity),
			"window_start":    key.Start,
			"status":          "open",
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build observe PM aggregation window SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("observe PM aggregation window received slots: %w", err)
	}
	return nil
}

func (r *WindowRepository) IsPublished(ctx context.Context, key WindowKey) (bool, error) {
	query, args, err := storage.Psql.Select("status").
		From("pm_aggregation_windows").
		Where(windowKeyPredicate(key)).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build PM aggregation window status SQL: %w", err)
	}
	var status string
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&status); err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("query PM aggregation window status: %w", err)
	}
	return status == "published", nil
}

func (r *WindowRepository) ListDue(
	ctx context.Context,
	now time.Time,
	grace time.Duration,
	limit uint64,
) ([]WindowRecord, error) {
	query, args, err := storage.Psql.Select(
		"task_id", "task_version_id", "granularity", "window_start", "window_end",
		"status", "expected_slots", "received_slots",
	).From("pm_aggregation_windows").
		Where(sq.Eq{"status": []string{"open", "failed"}}).
		Where(sq.LtOrEq{"window_end": now.Add(-grace)}).
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
	query, args, err := storage.Psql.Select(
		"task_id", "task_version_id", "granularity", "window_start", "window_end",
		"status", "expected_slots", "received_slots",
	).From("pm_aggregation_windows").
		Where(sq.Eq{"status": []string{"open", "failed"}}).
		Where(due).
		OrderBy("window_end").
		Limit(limit).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list due PM aggregation windows by granularity SQL: %w", err)
	}
	return r.queryWindows(ctx, query, args...)
}

func (r *WindowRepository) ListActive(ctx context.Context, limit uint64) ([]WindowRecord, error) {
	query, args, err := storage.Psql.Select(
		"task_id", "task_version_id", "granularity", "window_start", "window_end",
		"status", "expected_slots", "received_slots",
	).From("pm_aggregation_windows").
		Where(sq.Eq{"status": []string{"open", "failed", "finalizing"}}).
		OrderBy("window_start").
		Limit(limit).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list active PM aggregation windows SQL: %w", err)
	}
	return r.queryWindows(ctx, query, args...)
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
	var result []WindowRecord
	for rows.Next() {
		var record WindowRecord
		if err := rows.Scan(
			&record.Key.TaskID, &record.Key.TaskVersionID, &record.Key.Granularity,
			&record.Key.Start, &record.Key.End, &record.Status,
			&record.ExpectedSlots, &record.ReceivedSlots,
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
		"granularity":     string(key.Granularity),
		"window_start":    key.Start,
	}
}
