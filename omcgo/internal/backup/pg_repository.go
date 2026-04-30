package backup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// ---- column lists ----

var taskColumns = []string{
	"id", "task_type", "target_type", "target_ids",
	"status", "progress", "file_path", "error_message",
	"started_at", "completed_at", "created_at", "updated_at",
}

var scheduleColumns = []string{
	"id", "name", "cron_expr", "enabled", "task_type",
	"target_type", "target_ids", "created_at", "updated_at",
}

// ======================================================================
// PgTaskRepository
// ======================================================================

var _ TaskRepository = (*PgTaskRepository)(nil)

// PgTaskRepository is a PostgreSQL implementation of TaskRepository.
type PgTaskRepository struct {
	pool *pgxpool.Pool
}

// NewPgTaskRepository creates a new PgTaskRepository.
func NewPgTaskRepository(pool *pgxpool.Pool) *PgTaskRepository {
	return &PgTaskRepository{pool: pool}
}

func (r *PgTaskRepository) Create(ctx context.Context, task *BackupTask) error {
	targetIDsJSON, _ := json.Marshal(task.TargetIDs)

	query, args, err := storage.Psql.Insert("backup_tasks").
		Columns("task_type", "target_type", "target_ids", "status", "progress",
			"file_path", "error_message", "started_at", "completed_at").
		Values(task.TaskType, task.TargetType, targetIDsJSON, task.Status, task.Progress,
			task.FilePath, task.ErrorMessage, task.StartedAt, task.CompletedAt).
		Suffix("RETURNING " + joinColumns(taskColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert backup_task SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanTask(row)
	if err != nil {
		return fmt.Errorf("create backup_task: %w", err)
	}
	*task = *created
	return nil
}

func (r *PgTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*BackupTask, error) {
	query, args, err := storage.Psql.Select(taskColumns...).
		From("backup_tasks").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get backup_task SQL: %w", err)
	}

	task, err := scanTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get backup_task: %w", err)
	}
	return task, nil
}

func (r *PgTaskRepository) Update(ctx context.Context, task *BackupTask) error {
	targetIDsJSON, _ := json.Marshal(task.TargetIDs)

	query, args, err := storage.Psql.Update("backup_tasks").
		Set("task_type", task.TaskType).
		Set("target_type", task.TargetType).
		Set("target_ids", targetIDsJSON).
		Set("status", task.Status).
		Set("progress", task.Progress).
		Set("file_path", task.FilePath).
		Set("error_message", task.ErrorMessage).
		Set("started_at", task.StartedAt).
		Set("completed_at", task.CompletedAt).
		Where(sq.Eq{"id": task.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update backup_task SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update backup_task: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

// CleanupOldRows deletes terminal-status rows older than cutoff while keeping
// the most-recent `keepLastN` rows per target_type partition. Returns the
// non-empty file_paths of the deleted rows (for T-0076 physical delete) and
// the total deleted-row count. Pending/running tasks are excluded so an
// in-flight backup can never be deleted out from under itself.
//
// SQL strategy: a CTE ranks rows newest-first within target_type (ROW_NUMBER
// OVER PARTITION BY); the outer DELETE removes rows older than cutoff that
// are NOT in the kept-set. RETURNING file_path streams the to-be-physically-
// deleted paths back to the caller (T-0076) so MinIO RemoveObject can run
// best-effort outside the DB transaction. Rows with NULL or empty file_path
// (legacy rows from before T-0079 wired the linkage) are silently skipped at
// the application layer.
//
// Note: kept-set spans all terminal statuses (completed/failed/cancelled), so
// repeated failures on the same target_type can occupy keep slots. This is
// intentional — operators investigating recurring failures need failure rows
// retained alongside successes. To bias toward successes, future PR can add a
// secondary ORDER BY clause `(status='completed') DESC, completed_at DESC`.
// Performance note: backup_tasks has no `(target_type, completed_at)` index;
// at < 100k rows the seq scan is fine.
func (r *PgTaskRepository) CleanupOldRows(
	ctx context.Context, cutoff time.Time, keepLastN int,
) (deletedFilePaths []string, total int64, err error) {
	if keepLastN < 0 {
		keepLastN = 0
	}
	const q = `
WITH ranked AS (
    SELECT id,
           ROW_NUMBER() OVER (PARTITION BY target_type ORDER BY completed_at DESC NULLS LAST) AS rn
    FROM backup_tasks
    WHERE status IN ('completed', 'failed', 'cancelled')
)
DELETE FROM backup_tasks
WHERE status IN ('completed', 'failed', 'cancelled')
  AND completed_at IS NOT NULL
  AND completed_at < $1
  AND id NOT IN (SELECT id FROM ranked WHERE rn <= $2)
RETURNING file_path
`
	rows, err := r.pool.Query(ctx, q, cutoff, keepLastN)
	if err != nil {
		return nil, 0, fmt.Errorf("cleanup old backup_tasks: %w", err)
	}
	defer rows.Close()

	deletedFilePaths = make([]string, 0)
	for rows.Next() {
		var fp *string
		if scanErr := rows.Scan(&fp); scanErr != nil {
			return nil, 0, fmt.Errorf("scan deleted file_path: %w", scanErr)
		}
		total++
		if fp != nil && *fp != "" {
			deletedFilePaths = append(deletedFilePaths, *fp)
		}
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, 0, fmt.Errorf("iterate deleted rows: %w", rowsErr)
	}
	return deletedFilePaths, total, nil
}

func (r *PgTaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Delete("backup_tasks").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete backup_task SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete backup_task: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgTaskRepository) List(ctx context.Context, filter TaskFilter) (*model.ListResponse[BackupTask], error) {
	base := storage.Psql.Select(taskColumns...).From("backup_tasks")
	countBase := storage.Psql.Select("COUNT(*)").From("backup_tasks")

	if filter.Status != nil {
		base = base.Where(sq.Eq{"status": *filter.Status})
		countBase = countBase.Where(sq.Eq{"status": *filter.Status})
	}
	if filter.TaskType != nil {
		base = base.Where(sq.Eq{"task_type": *filter.TaskType})
		countBase = countBase.Where(sq.Eq{"task_type": *filter.TaskType})
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count backup_task SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count backup_tasks: %w", err)
	}

	// Pagination
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortDir := filter.SortDir
	if sortDir == "" {
		sortDir = "desc"
	}
	base = base.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list backup_task SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list backup_tasks: %w", err)
	}
	defer rows.Close()

	var items []BackupTask
	for rows.Next() {
		task, err := scanTaskRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan backup_task row: %w", err)
		}
		items = append(items, *task)
	}

	if items == nil {
		items = []BackupTask{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// UpdateFilePath sets file_path via atomic CAS — only writes when the column
// is currently NULL (first-write-wins). Returns ErrFilePathAlreadySet when a
// concurrent writer already set the value (multi-device backup tasks have N
// uploads racing for one file_path slot); returns ErrNotFound when no row
// matches the id at all. The CAS is performed at the DB layer, so the
// recorder's read-then-write surface is TOCTOU-free.
func (r *PgTaskRepository) UpdateFilePath(ctx context.Context, id uuid.UUID, filePath string) error {
	// Conditional UPDATE: only write when file_path is currently NULL.
	query, args, err := storage.Psql.Update("backup_tasks").
		Set("file_path", filePath).
		Where(sq.Eq{"id": id, "file_path": nil}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update backup_task file_path SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update backup_task file_path: %w", err)
	}
	if tag.RowsAffected() > 0 {
		return nil
	}
	// 0 rows affected — disambiguate "id not found" vs "already set" via a
	// follow-up read. This is a small extra cost on the rare lose-the-race
	// path; the win path is single-RTT.
	var existing *string
	err = r.pool.QueryRow(ctx,
		`SELECT file_path FROM backup_tasks WHERE id = $1`, id,
	).Scan(&existing)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("backup_task %s: %w", id, commonerrors.ErrNotFound)
	}
	if err != nil {
		return fmt.Errorf("read backup_task %s for CAS disambiguation: %w", id, err)
	}
	if existing != nil && *existing != "" {
		return ErrFilePathAlreadySet
	}
	// Row exists, file_path is null, but UPDATE matched 0 rows — should not
	// happen in practice; treat as transient and surface as ErrNotFound for
	// the recorder to no-op.
	return fmt.Errorf("backup_task %s: %w", id, commonerrors.ErrNotFound)
}

// FindByIDPrefix returns up to `limit` backup_tasks whose UUID (without
// dashes) starts with the given hex prefix. Used by FilePathRecorder to map
// the {taskID8} prefix in an upload filename back to the originating task
// (T-0079). Returns an empty slice if prefix is empty or no rows match.
//
// Implementation note: PG `uuid::text` includes dashes (e.g.
// "abcdef12-3456-..."), but our prefix is the dashless first 8 chars. Use
// REPLACE(id::text, '-', ”) LIKE 'prefix%' so the prefix matches the first
// 8 hex characters regardless of the dash position. Backup_tasks is small
// enough at MVP scale that the seq scan is acceptable; if scale demands an
// index, a generated column + functional index is a future optimization.
func (r *PgTaskRepository) FindByIDPrefix(ctx context.Context, prefix string, limit int) ([]*BackupTask, error) {
	if prefix == "" {
		return nil, nil
	}
	if limit < 1 {
		limit = 1
	}
	q := storage.Psql.Select(taskColumns...).
		From("backup_tasks").
		Where("REPLACE(id::text, '-', '') LIKE ?", prefix+"%").
		OrderBy("created_at DESC").
		Limit(uint64(limit))
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build find backup_task by prefix SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("query backup_task by prefix: %w", err)
	}
	defer rows.Close()
	out := make([]*BackupTask, 0, limit)
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, fmt.Errorf("scan backup_task row: %w", err)
		}
		out = append(out, t)
	}
	return out, nil
}

// ListAllTaskIDPrefixes returns the set of 8-char hex prefixes for all
// backup_tasks.id rows. Used by the orphan reaper (T-0083) to build a live
// set before bucket scan.
//
// Returns DISTINCT prefixes so the result size is bounded by 16^8 ≈ 4.3 ×
// 10^9 (in practice much smaller; collisions on 8-char prefixes are
// possible but the live set is intentionally conservative — collisions
// reduce reap aggressiveness without risking false-positive deletes).
//
// At MVP scale (≤ 1M rows) the seq scan + DISTINCT runs in a few hundred
// ms; if that becomes a bottleneck, materialize via generated column +
// index. PRD §2.6 sizing: 1M × 8 bytes ≈ 8MB heap, acceptable.
func (r *PgTaskRepository) ListAllTaskIDPrefixes(ctx context.Context) (map[string]struct{}, error) {
	const sqlStr = `SELECT DISTINCT SUBSTRING(REPLACE(id::text, '-', ''), 1, 8) FROM backup_tasks`
	rows, err := r.pool.Query(ctx, sqlStr)
	if err != nil {
		return nil, fmt.Errorf("query backup_task id prefixes: %w", err)
	}
	defer rows.Close()
	out := make(map[string]struct{})
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("scan backup_task id prefix: %w", err)
		}
		out[p] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate backup_task id prefixes: %w", err)
	}
	return out, nil
}

// ---- task scanning helpers ----

func scanTask(row pgx.Row) (*BackupTask, error) {
	var t BackupTask
	var targetIDsJSON []byte

	err := row.Scan(
		&t.ID, &t.TaskType, &t.TargetType, &targetIDsJSON,
		&t.Status, &t.Progress, &t.FilePath, &t.ErrorMessage,
		&t.StartedAt, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if targetIDsJSON != nil {
		_ = json.Unmarshal(targetIDsJSON, &t.TargetIDs)
	}
	if t.TargetIDs == nil {
		t.TargetIDs = []string{}
	}
	return &t, nil
}

func scanTaskRow(rows pgx.Rows) (*BackupTask, error) {
	var t BackupTask
	var targetIDsJSON []byte

	err := rows.Scan(
		&t.ID, &t.TaskType, &t.TargetType, &targetIDsJSON,
		&t.Status, &t.Progress, &t.FilePath, &t.ErrorMessage,
		&t.StartedAt, &t.CompletedAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if targetIDsJSON != nil {
		_ = json.Unmarshal(targetIDsJSON, &t.TargetIDs)
	}
	if t.TargetIDs == nil {
		t.TargetIDs = []string{}
	}
	return &t, nil
}

// ======================================================================
// PgScheduleRepository
// ======================================================================

var _ ScheduleRepository = (*PgScheduleRepository)(nil)

// PgScheduleRepository is a PostgreSQL implementation of ScheduleRepository.
type PgScheduleRepository struct {
	pool *pgxpool.Pool
}

// NewPgScheduleRepository creates a new PgScheduleRepository.
func NewPgScheduleRepository(pool *pgxpool.Pool) *PgScheduleRepository {
	return &PgScheduleRepository{pool: pool}
}

func (r *PgScheduleRepository) Create(ctx context.Context, schedule *BackupSchedule) error {
	targetIDsJSON, _ := json.Marshal(schedule.TargetIDs)

	query, args, err := storage.Psql.Insert("backup_schedules").
		Columns("name", "cron_expr", "enabled", "task_type",
			"target_type", "target_ids").
		Values(schedule.Name, schedule.CronExpr, schedule.Enabled, schedule.TaskType,
			schedule.TargetType, targetIDsJSON).
		Suffix("RETURNING " + joinColumns(scheduleColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert backup_schedule SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanSchedule(row)
	if err != nil {
		return fmt.Errorf("create backup_schedule: %w", err)
	}
	*schedule = *created
	return nil
}

func (r *PgScheduleRepository) GetByID(ctx context.Context, id uuid.UUID) (*BackupSchedule, error) {
	query, args, err := storage.Psql.Select(scheduleColumns...).
		From("backup_schedules").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get backup_schedule SQL: %w", err)
	}

	schedule, err := scanSchedule(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get backup_schedule: %w", err)
	}
	return schedule, nil
}

func (r *PgScheduleRepository) Update(ctx context.Context, schedule *BackupSchedule) error {
	targetIDsJSON, _ := json.Marshal(schedule.TargetIDs)

	query, args, err := storage.Psql.Update("backup_schedules").
		Set("name", schedule.Name).
		Set("cron_expr", schedule.CronExpr).
		Set("enabled", schedule.Enabled).
		Set("task_type", schedule.TaskType).
		Set("target_type", schedule.TargetType).
		Set("target_ids", targetIDsJSON).
		Where(sq.Eq{"id": schedule.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update backup_schedule SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update backup_schedule: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgScheduleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Delete("backup_schedules").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete backup_schedule SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete backup_schedule: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgScheduleRepository) List(ctx context.Context, filter ScheduleFilter) (*model.ListResponse[BackupSchedule], error) {
	base := storage.Psql.Select(scheduleColumns...).From("backup_schedules")
	countBase := storage.Psql.Select("COUNT(*)").From("backup_schedules")

	if filter.Enabled != nil {
		base = base.Where(sq.Eq{"enabled": *filter.Enabled})
		countBase = countBase.Where(sq.Eq{"enabled": *filter.Enabled})
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count backup_schedule SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count backup_schedules: %w", err)
	}

	// Pagination
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortDir := filter.SortDir
	if sortDir == "" {
		sortDir = "desc"
	}
	base = base.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list backup_schedule SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list backup_schedules: %w", err)
	}
	defer rows.Close()

	var items []BackupSchedule
	for rows.Next() {
		schedule, err := scanScheduleRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan backup_schedule row: %w", err)
		}
		items = append(items, *schedule)
	}

	if items == nil {
		items = []BackupSchedule{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// ---- schedule scanning helpers ----

func scanSchedule(row pgx.Row) (*BackupSchedule, error) {
	var s BackupSchedule
	var targetIDsJSON []byte

	err := row.Scan(
		&s.ID, &s.Name, &s.CronExpr, &s.Enabled, &s.TaskType,
		&s.TargetType, &targetIDsJSON, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if targetIDsJSON != nil {
		_ = json.Unmarshal(targetIDsJSON, &s.TargetIDs)
	}
	if s.TargetIDs == nil {
		s.TargetIDs = []string{}
	}
	return &s, nil
}

func scanScheduleRow(rows pgx.Rows) (*BackupSchedule, error) {
	var s BackupSchedule
	var targetIDsJSON []byte

	err := rows.Scan(
		&s.ID, &s.Name, &s.CronExpr, &s.Enabled, &s.TaskType,
		&s.TargetType, &targetIDsJSON, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if targetIDsJSON != nil {
		_ = json.Unmarshal(targetIDsJSON, &s.TargetIDs)
	}
	if s.TargetIDs == nil {
		s.TargetIDs = []string{}
	}
	return &s, nil
}

// ---- shared helpers ----

func joinColumns(cols []string) string {
	result := ""
	for i, c := range cols {
		if i > 0 {
			result += ", "
		}
		result += c
	}
	return result
}
