package provision

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/common/errors"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

var taskColumns = []string{
	"id", "device_id", "template_id", "status", "current_step", "total_steps",
	"error_message", "retry_count", "max_retries",
	"started_at", "completed_at", "created_at", "updated_at",
}

// PgProvisioningTaskRepository implements ProvisioningTaskRepository using PostgreSQL.
type PgProvisioningTaskRepository struct {
	pool *pgxpool.Pool
}

// NewPgProvisioningTaskRepository creates a new PgProvisioningTaskRepository.
func NewPgProvisioningTaskRepository(pool *pgxpool.Pool) *PgProvisioningTaskRepository {
	return &PgProvisioningTaskRepository{pool: pool}
}

func (r *PgProvisioningTaskRepository) Create(ctx context.Context, task *ProvisioningTask) error {
	if task.ID == uuid.Nil {
		task.ID = uuid.New()
	}
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now

	query, args, err := psql.Insert("provisioning_tasks").
		Columns(taskColumns...).
		Values(
			task.ID, task.DeviceID, nullableUUID(task.TemplateID), task.Status,
			task.CurrentStep, task.TotalSteps, nullableString(task.ErrorMessage),
			task.RetryCount, task.MaxRetries,
			nullableTime(task.StartedAt), nullableTime(task.CompletedAt),
			task.CreatedAt, task.UpdatedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert task SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert provisioning task: %w", err)
	}
	return nil
}

func (r *PgProvisioningTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*ProvisioningTask, error) {
	query, args, err := psql.Select(taskColumns...).
		From("provisioning_tasks").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select task SQL: %w", err)
	}

	task, err := scanTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (r *PgProvisioningTaskRepository) GetByDeviceID(ctx context.Context, deviceID uuid.UUID) (*ProvisioningTask, error) {
	query, args, err := psql.Select(taskColumns...).
		From("provisioning_tasks").
		Where(sq.Eq{"device_id": deviceID}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select task by device SQL: %w", err)
	}

	task, err := scanTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (r *PgProvisioningTaskRepository) Update(ctx context.Context, task *ProvisioningTask) error {
	task.UpdatedAt = time.Now()

	query, args, err := psql.Update("provisioning_tasks").
		Set("template_id", nullableUUID(task.TemplateID)).
		Set("status", task.Status).
		Set("current_step", task.CurrentStep).
		Set("total_steps", task.TotalSteps).
		Set("error_message", nullableString(task.ErrorMessage)).
		Set("retry_count", task.RetryCount).
		Set("max_retries", task.MaxRetries).
		Set("started_at", nullableTime(task.StartedAt)).
		Set("completed_at", nullableTime(task.CompletedAt)).
		Set("updated_at", task.UpdatedAt).
		Where(sq.Eq{"id": task.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update task SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update provisioning task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update task %s: %w", task.ID, commonerrors.ErrNotFound)
	}
	return nil
}

func (r *PgProvisioningTaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status ProvisioningState, errorMsg string) error {
	builder := psql.Update("provisioning_tasks").
		Set("status", status).
		Set("error_message", nullableString(errorMsg)).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id})

	if status == StateCompleted || status == StateFailed {
		now := time.Now()
		builder = builder.Set("completed_at", now)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update task status SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update task status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update task status %s: %w", id, commonerrors.ErrNotFound)
	}
	return nil
}

func (r *PgProvisioningTaskRepository) List(ctx context.Context, filter ProvisioningTaskFilter) ([]ProvisioningTask, int64, error) {
	pred := sq.And{}
	if filter.DeviceID != nil {
		pred = append(pred, sq.Eq{"device_id": *filter.DeviceID})
	}
	if filter.Status != "" {
		pred = append(pred, sq.Eq{"status": filter.Status})
	}

	// Count.
	countBuilder := psql.Select("COUNT(*)").From("provisioning_tasks")
	if len(pred) > 0 {
		countBuilder = countBuilder.Where(pred)
	}
	countSQL, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count tasks SQL: %w", err)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	queryBuilder := psql.Select(taskColumns...).
		From("provisioning_tasks").
		Limit(uint64(pageSize)).
		Offset(uint64(offset)).
		OrderBy("created_at DESC")

	if len(pred) > 0 {
		queryBuilder = queryBuilder.Where(pred)
	}

	querySQL, queryArgs, err := queryBuilder.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build list tasks SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, querySQL, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	items, err := scanTasks(rows)
	if err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

func (r *PgProvisioningTaskRepository) CountByStatus(ctx context.Context) (map[ProvisioningState]int64, error) {
	query, _, err := psql.Select("status", "COUNT(*)").
		From("provisioning_tasks").
		GroupBy("status").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count by status SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("count tasks by status: %w", err)
	}
	defer rows.Close()

	result := make(map[ProvisioningState]int64)
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan count row: %w", err)
		}
		result[ProvisioningState(status)] = count
	}
	return result, rows.Err()
}

// --- Helper functions ---

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullableUUID(id *uuid.UUID) interface{} {
	if id == nil {
		return nil
	}
	return *id
}

func nullableTime(t *time.Time) interface{} {
	if t == nil {
		return nil
	}
	return *t
}

func scanTask(row pgx.Row) (*ProvisioningTask, error) {
	var task ProvisioningTask
	var (
		templateID   sql.NullString
		errorMessage sql.NullString
		startedAt    sql.NullTime
		completedAt  sql.NullTime
	)

	err := row.Scan(
		&task.ID, &task.DeviceID, &templateID, &task.Status,
		&task.CurrentStep, &task.TotalSteps, &errorMessage,
		&task.RetryCount, &task.MaxRetries,
		&startedAt, &completedAt, &task.CreatedAt, &task.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan task row: %w", err)
	}

	if templateID.Valid {
		id, _ := uuid.Parse(templateID.String)
		task.TemplateID = &id
	}
	if errorMessage.Valid {
		task.ErrorMessage = errorMessage.String
	}
	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}

	return &task, nil
}

func scanTasks(rows pgx.Rows) ([]ProvisioningTask, error) {
	var items []ProvisioningTask
	for rows.Next() {
		var task ProvisioningTask
		var (
			templateID   sql.NullString
			errorMessage sql.NullString
			startedAt    sql.NullTime
			completedAt  sql.NullTime
		)

		err := rows.Scan(
			&task.ID, &task.DeviceID, &templateID, &task.Status,
			&task.CurrentStep, &task.TotalSteps, &errorMessage,
			&task.RetryCount, &task.MaxRetries,
			&startedAt, &completedAt, &task.CreatedAt, &task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan task row: %w", err)
		}

		if templateID.Valid {
			id, _ := uuid.Parse(templateID.String)
			task.TemplateID = &id
		}
		if errorMessage.Valid {
			task.ErrorMessage = errorMessage.String
		}
		if startedAt.Valid {
			task.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}

		items = append(items, task)
	}
	return items, rows.Err()
}

// Compile-time interface compliance check.
var _ ProvisioningTaskRepository = (*PgProvisioningTaskRepository)(nil)
