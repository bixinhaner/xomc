package ops

import (
	"context"
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

var templateColumns = []string{
	"id", "template_name", "description", "category",
	"target_device_types", "steps", "estimated_duration",
	"creator", "use_count", "tags", "created_at", "updated_at",
}

var taskColumns = []string{
	"id", "task_name", "template_id", "device_sns",
	"status", "current_step", "total_steps", "progress",
	"success_count", "fail_count", "total_count",
	"creator", "message", "started_at", "completed_at",
	"created_at", "updated_at",
}

var cmdRecordColumns = []string{
	"id", "command_text", "device_sn", "device_name",
	"operator", "execute_time", "duration", "success",
	"output", "error_message", "created_at",
}

// ======================================================================
// PgTemplateRepository
// ======================================================================

var _ TemplateRepository = (*PgTemplateRepository)(nil)

// PgTemplateRepository is a PostgreSQL implementation of TemplateRepository.
type PgTemplateRepository struct {
	pool *pgxpool.Pool
}

// NewPgTemplateRepository creates a new PgTemplateRepository.
func NewPgTemplateRepository(pool *pgxpool.Pool) *PgTemplateRepository {
	return &PgTemplateRepository{pool: pool}
}

func (r *PgTemplateRepository) Create(ctx context.Context, tmpl *OpsTemplate) error {
	if tmpl.TargetDeviceTypes == nil {
		tmpl.TargetDeviceTypes = []byte("[]")
	}
	if tmpl.Steps == nil {
		tmpl.Steps = []byte("[]")
	}
	if tmpl.Tags == nil {
		tmpl.Tags = []byte("[]")
	}

	query, args, err := storage.Psql.Insert("ops_templates").
		Columns("template_name", "description", "category",
			"target_device_types", "steps", "estimated_duration",
			"creator", "use_count", "tags").
		Values(tmpl.TemplateName, tmpl.Description, tmpl.Category,
			tmpl.TargetDeviceTypes, tmpl.Steps, tmpl.EstimatedDuration,
			tmpl.Creator, tmpl.UseCount, tmpl.Tags).
		Suffix("RETURNING " + joinColumns(templateColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert ops_template SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanTemplate(row)
	if err != nil {
		return fmt.Errorf("create ops_template: %w", err)
	}
	*tmpl = *created
	return nil
}

func (r *PgTemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*OpsTemplate, error) {
	query, args, err := storage.Psql.Select(templateColumns...).
		From("ops_templates").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get ops_template SQL: %w", err)
	}

	tmpl, err := scanTemplate(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get ops_template: %w", err)
	}
	return tmpl, nil
}

func (r *PgTemplateRepository) Update(ctx context.Context, tmpl *OpsTemplate) error {
	if tmpl.TargetDeviceTypes == nil {
		tmpl.TargetDeviceTypes = []byte("[]")
	}
	if tmpl.Steps == nil {
		tmpl.Steps = []byte("[]")
	}
	if tmpl.Tags == nil {
		tmpl.Tags = []byte("[]")
	}

	query, args, err := storage.Psql.Update("ops_templates").
		Set("template_name", tmpl.TemplateName).
		Set("description", tmpl.Description).
		Set("category", tmpl.Category).
		Set("target_device_types", tmpl.TargetDeviceTypes).
		Set("steps", tmpl.Steps).
		Set("estimated_duration", tmpl.EstimatedDuration).
		Set("creator", tmpl.Creator).
		Set("tags", tmpl.Tags).
		Where(sq.Eq{"id": tmpl.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update ops_template SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update ops_template: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Delete("ops_templates").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete ops_template SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete ops_template: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgTemplateRepository) List(ctx context.Context, filter TemplateFilter) (*model.ListResponse[OpsTemplate], error) {
	base := storage.Psql.Select(templateColumns...).From("ops_templates")
	countBase := storage.Psql.Select("COUNT(*)").From("ops_templates")

	if filter.Category != "" {
		base = base.Where(sq.Eq{"category": filter.Category})
		countBase = countBase.Where(sq.Eq{"category": filter.Category})
	}
	if filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		cond := sq.Or{sq.ILike{"template_name": like}, sq.ILike{"description": like}}
		base = base.Where(cond)
		countBase = countBase.Where(cond)
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count ops_template SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count ops_templates: %w", err)
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
		return nil, fmt.Errorf("build list ops_template SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list ops_templates: %w", err)
	}
	defer rows.Close()

	var items []OpsTemplate
	for rows.Next() {
		tmpl, err := scanTemplateRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan ops_template row: %w", err)
		}
		items = append(items, *tmpl)
	}

	if items == nil {
		items = []OpsTemplate{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func (r *PgTemplateRepository) IncrementUseCount(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Update("ops_templates").
		Set("use_count", sq.Expr("use_count + 1")).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build increment use_count SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("increment ops_template use_count: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

// ---- template scanning helpers ----

func scanTemplate(row pgx.Row) (*OpsTemplate, error) {
	var t OpsTemplate
	err := row.Scan(
		&t.ID, &t.TemplateName, &t.Description, &t.Category,
		&t.TargetDeviceTypes, &t.Steps, &t.EstimatedDuration,
		&t.Creator, &t.UseCount, &t.Tags, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if t.TargetDeviceTypes == nil {
		t.TargetDeviceTypes = []byte("[]")
	}
	if t.Steps == nil {
		t.Steps = []byte("[]")
	}
	if t.Tags == nil {
		t.Tags = []byte("[]")
	}
	return &t, nil
}

func scanTemplateRow(rows pgx.Rows) (*OpsTemplate, error) {
	var t OpsTemplate
	err := rows.Scan(
		&t.ID, &t.TemplateName, &t.Description, &t.Category,
		&t.TargetDeviceTypes, &t.Steps, &t.EstimatedDuration,
		&t.Creator, &t.UseCount, &t.Tags, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if t.TargetDeviceTypes == nil {
		t.TargetDeviceTypes = []byte("[]")
	}
	if t.Steps == nil {
		t.Steps = []byte("[]")
	}
	if t.Tags == nil {
		t.Tags = []byte("[]")
	}
	return &t, nil
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

func (r *PgTaskRepository) Create(ctx context.Context, task *OpsTask) error {
	if task.DeviceSNs == nil {
		task.DeviceSNs = []byte("[]")
	}

	query, args, err := storage.Psql.Insert("ops_tasks").
		Columns("task_name", "template_id", "device_sns",
			"status", "current_step", "total_steps", "progress",
			"success_count", "fail_count", "total_count",
			"creator", "message", "started_at", "completed_at").
		Values(task.TaskName, task.TemplateID, task.DeviceSNs,
			task.Status, task.CurrentStep, task.TotalSteps, task.Progress,
			task.SuccessCount, task.FailCount, task.TotalCount,
			task.Creator, task.Message, task.StartedAt, task.CompletedAt).
		Suffix("RETURNING " + joinColumns(taskColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert ops_task SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanTask(row)
	if err != nil {
		return fmt.Errorf("create ops_task: %w", err)
	}
	*task = *created
	return nil
}

func (r *PgTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*OpsTask, error) {
	query, args, err := storage.Psql.Select(taskColumns...).
		From("ops_tasks").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get ops_task SQL: %w", err)
	}

	task, err := scanTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get ops_task: %w", err)
	}
	return task, nil
}

func (r *PgTaskRepository) UpdateStatus(ctx context.Context, task *OpsTask) error {
	builder := storage.Psql.Update("ops_tasks").
		Set("status", task.Status).
		Set("current_step", task.CurrentStep).
		Set("progress", task.Progress).
		Set("success_count", task.SuccessCount).
		Set("fail_count", task.FailCount).
		Set("message", task.Message).
		Set("started_at", task.StartedAt).
		Set("completed_at", task.CompletedAt).
		Where(sq.Eq{"id": task.ID})

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update ops_task status SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update ops_task status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgTaskRepository) List(ctx context.Context, filter TaskFilter) (*model.ListResponse[OpsTask], error) {
	base := storage.Psql.Select(taskColumns...).From("ops_tasks")
	countBase := storage.Psql.Select("COUNT(*)").From("ops_tasks")

	if filter.Status != nil {
		base = base.Where(sq.Eq{"status": *filter.Status})
		countBase = countBase.Where(sq.Eq{"status": *filter.Status})
	}
	if filter.TemplateID != nil {
		base = base.Where(sq.Eq{"template_id": *filter.TemplateID})
		countBase = countBase.Where(sq.Eq{"template_id": *filter.TemplateID})
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count ops_task SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count ops_tasks: %w", err)
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
		return nil, fmt.Errorf("build list ops_task SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list ops_tasks: %w", err)
	}
	defer rows.Close()

	var items []OpsTask
	for rows.Next() {
		task, err := scanTaskRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan ops_task row: %w", err)
		}
		items = append(items, *task)
	}

	if items == nil {
		items = []OpsTask{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// ---- task scanning helpers ----

func scanTask(row pgx.Row) (*OpsTask, error) {
	var t OpsTask
	err := row.Scan(
		&t.ID, &t.TaskName, &t.TemplateID, &t.DeviceSNs,
		&t.Status, &t.CurrentStep, &t.TotalSteps, &t.Progress,
		&t.SuccessCount, &t.FailCount, &t.TotalCount,
		&t.Creator, &t.Message, &t.StartedAt, &t.CompletedAt,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if t.DeviceSNs == nil {
		t.DeviceSNs = []byte("[]")
	}
	return &t, nil
}

func scanTaskRow(rows pgx.Rows) (*OpsTask, error) {
	var t OpsTask
	err := rows.Scan(
		&t.ID, &t.TaskName, &t.TemplateID, &t.DeviceSNs,
		&t.Status, &t.CurrentStep, &t.TotalSteps, &t.Progress,
		&t.SuccessCount, &t.FailCount, &t.TotalCount,
		&t.Creator, &t.Message, &t.StartedAt, &t.CompletedAt,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if t.DeviceSNs == nil {
		t.DeviceSNs = []byte("[]")
	}
	return &t, nil
}

// ======================================================================
// PgCommandRecordRepository
// ======================================================================

var _ CommandRecordRepository = (*PgCommandRecordRepository)(nil)

// PgCommandRecordRepository is a PostgreSQL implementation of CommandRecordRepository.
type PgCommandRecordRepository struct {
	pool *pgxpool.Pool
}

// NewPgCommandRecordRepository creates a new PgCommandRecordRepository.
func NewPgCommandRecordRepository(pool *pgxpool.Pool) *PgCommandRecordRepository {
	return &PgCommandRecordRepository{pool: pool}
}

func (r *PgCommandRecordRepository) Create(ctx context.Context, record *OpsCommandRecord) error {
	query, args, err := storage.Psql.Insert("ops_command_records").
		Columns("command_text", "device_sn", "device_name",
			"operator", "execute_time", "duration", "success",
			"output", "error_message").
		Values(record.CommandText, record.DeviceSN, record.DeviceName,
			record.Operator, record.ExecuteTime, record.Duration, record.Success,
			record.Output, record.ErrorMessage).
		Suffix("RETURNING " + joinColumns(cmdRecordColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert ops_command_record SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanCmdRecord(row)
	if err != nil {
		return fmt.Errorf("create ops_command_record: %w", err)
	}
	*record = *created
	return nil
}

func (r *PgCommandRecordRepository) List(ctx context.Context, filter CommandRecordFilter) (*model.ListResponse[OpsCommandRecord], error) {
	base := storage.Psql.Select(cmdRecordColumns...).From("ops_command_records")
	countBase := storage.Psql.Select("COUNT(*)").From("ops_command_records")

	if filter.DeviceSN != "" {
		base = base.Where(sq.Eq{"device_sn": filter.DeviceSN})
		countBase = countBase.Where(sq.Eq{"device_sn": filter.DeviceSN})
	}
	if filter.Operator != "" {
		base = base.Where(sq.Eq{"operator": filter.Operator})
		countBase = countBase.Where(sq.Eq{"operator": filter.Operator})
	}
	if filter.Success != nil {
		base = base.Where(sq.Eq{"success": *filter.Success})
		countBase = countBase.Where(sq.Eq{"success": *filter.Success})
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count ops_command_record SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count ops_command_records: %w", err)
	}

	// Pagination
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "execute_time"
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
		return nil, fmt.Errorf("build list ops_command_record SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list ops_command_records: %w", err)
	}
	defer rows.Close()

	var items []OpsCommandRecord
	for rows.Next() {
		record, err := scanCmdRecordRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan ops_command_record row: %w", err)
		}
		items = append(items, *record)
	}

	if items == nil {
		items = []OpsCommandRecord{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// ---- command record scanning helpers ----

func scanCmdRecord(row pgx.Row) (*OpsCommandRecord, error) {
	var r OpsCommandRecord
	err := row.Scan(
		&r.ID, &r.CommandText, &r.DeviceSN, &r.DeviceName,
		&r.Operator, &r.ExecuteTime, &r.Duration, &r.Success,
		&r.Output, &r.ErrorMessage, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func scanCmdRecordRow(rows pgx.Rows) (*OpsCommandRecord, error) {
	var r OpsCommandRecord
	err := rows.Scan(
		&r.ID, &r.CommandText, &r.DeviceSN, &r.DeviceName,
		&r.Operator, &r.ExecuteTime, &r.Duration, &r.Success,
		&r.Output, &r.ErrorMessage, &r.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
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
