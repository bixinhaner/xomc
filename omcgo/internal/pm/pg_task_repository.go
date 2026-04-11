package pm

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

var pmTaskColumns = []string{
	"id", "task_name", "task_type", "device_sns", "kpi_codes",
	"granularity", "time_range", "status", "progress",
	"creator", "created_at", "updated_at",
}

var _ TaskRepository = (*PgTaskRepository)(nil)

// PgTaskRepository implements TaskRepository using PostgreSQL.
type PgTaskRepository struct {
	pool *pgxpool.Pool
}

// NewPgTaskRepository creates a new PostgreSQL-backed PM task repository.
func NewPgTaskRepository(pool *pgxpool.Pool) *PgTaskRepository {
	return &PgTaskRepository{pool: pool}
}

func (r *PgTaskRepository) Create(ctx context.Context, task *PerformanceTask) error {
	if task.ID == uuid.Nil {
		task.ID = uuid.New()
	}
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now
	if task.Status == "" {
		task.Status = PMTaskPending
	}
	if task.TaskType == "" {
		task.TaskType = PMTaskExtraction
	}
	if task.Granularity == "" {
		task.Granularity = "15min"
	}
	if task.DeviceSNs == nil {
		task.DeviceSNs = []byte("[]")
	}
	if task.KPICodes == nil {
		task.KPICodes = []byte("[]")
	}

	query, args, err := storage.Psql.Insert("pm_tasks").
		Columns("id", "task_name", "task_type", "device_sns", "kpi_codes",
			"granularity", "time_range", "status", "progress",
			"creator", "created_at", "updated_at").
		Values(task.ID, task.TaskName, task.TaskType, task.DeviceSNs, task.KPICodes,
			task.Granularity, task.TimeRange, task.Status, task.Progress,
			task.Creator, task.CreatedAt, task.UpdatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert pm_tasks SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert pm_tasks: %w", err)
	}
	return nil
}

func (r *PgTaskRepository) List(ctx context.Context, filter TaskFilter) (*model.ListResponse[PerformanceTask], error) {
	builder := storage.Psql.Select(pmTaskColumns...).From("pm_tasks")
	countBuilder := storage.Psql.Select("COUNT(*)").From("pm_tasks")

	builder = applyPMTaskFilters(builder, filter)
	countBuilder = applyPMTaskFilters(countBuilder, filter)

	// Count total
	countSQL, countArgs, err := countBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count pm_tasks SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count pm_tasks: %w", err)
	}

	// Apply sorting and pagination
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortDir := filter.SortDir
	if sortDir == "" {
		sortDir = "desc"
	}
	builder = builder.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list pm_tasks SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list pm_tasks: %w", err)
	}
	defer rows.Close()

	var items []PerformanceTask
	for rows.Next() {
		t, err := scanPMTaskRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *t)
	}

	if items == nil {
		items = []PerformanceTask{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func applyPMTaskFilters(qb sq.SelectBuilder, f TaskFilter) sq.SelectBuilder {
	if f.Status != nil {
		qb = qb.Where(sq.Eq{"status": string(*f.Status)})
	}
	if f.TaskType != nil {
		qb = qb.Where(sq.Eq{"task_type": string(*f.TaskType)})
	}
	return qb
}

func scanPMTaskRow(rows pgx.Rows) (*PerformanceTask, error) {
	var t PerformanceTask
	err := rows.Scan(
		&t.ID, &t.TaskName, &t.TaskType, &t.DeviceSNs, &t.KPICodes,
		&t.Granularity, &t.TimeRange, &t.Status, &t.Progress,
		&t.Creator, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan pm_tasks row: %w", err)
	}
	if t.DeviceSNs == nil {
		t.DeviceSNs = []byte("[]")
	}
	if t.KPICodes == nil {
		t.KPICodes = []byte("[]")
	}
	return &t, nil
}
