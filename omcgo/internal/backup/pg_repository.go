package backup

import (
	"context"
	"encoding/json"
	"fmt"

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
