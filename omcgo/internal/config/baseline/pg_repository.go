package baseline

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

var baselineColumns = []string{
	"id", "baseline_name", "description", "device_type", "version",
	"params", "creator", "status", "created_at", "updated_at",
}

var configTaskColumns = []string{
	"id", "task_name", "task_type", "device_sns", "template_id",
	"baseline_id", "params", "status", "progress", "success_count",
	"fail_count", "total_count", "creator", "message",
	"created_at", "updated_at",
}

var neighborColumns = []string{
	"id", "source_cell_id", "source_cell_name", "target_cell_id",
	"target_cell_name", "neighbor_type", "params",
	"created_at", "updated_at",
}

// ======================================================================
// PgBaselineRepository
// ======================================================================

var _ BaselineRepository = (*PgBaselineRepository)(nil)

// PgBaselineRepository is a PostgreSQL implementation of BaselineRepository.
type PgBaselineRepository struct {
	pool *pgxpool.Pool
}

// NewPgBaselineRepository creates a new PgBaselineRepository.
func NewPgBaselineRepository(pool *pgxpool.Pool) *PgBaselineRepository {
	return &PgBaselineRepository{pool: pool}
}

func (r *PgBaselineRepository) Create(ctx context.Context, baseline *BaselineConfig) error {
	if baseline.Params == nil {
		baseline.Params = json.RawMessage("[]")
	}

	query, args, err := storage.Psql.Insert("config_baselines").
		Columns("baseline_name", "description", "device_type", "version",
			"params", "creator", "status").
		Values(baseline.BaselineName, baseline.Description, baseline.DeviceType,
			baseline.Version, baseline.Params, baseline.Creator, baseline.Status).
		Suffix("RETURNING " + joinColumns(baselineColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert config_baseline SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanBaseline(row)
	if err != nil {
		return fmt.Errorf("create config_baseline: %w", err)
	}
	*baseline = *created
	return nil
}

func (r *PgBaselineRepository) GetByID(ctx context.Context, id uuid.UUID) (*BaselineConfig, error) {
	query, args, err := storage.Psql.Select(baselineColumns...).
		From("config_baselines").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get config_baseline SQL: %w", err)
	}

	baseline, err := scanBaseline(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get config_baseline: %w", err)
	}
	return baseline, nil
}

func (r *PgBaselineRepository) Update(ctx context.Context, baseline *BaselineConfig) error {
	if baseline.Params == nil {
		baseline.Params = json.RawMessage("[]")
	}

	query, args, err := storage.Psql.Update("config_baselines").
		Set("baseline_name", baseline.BaselineName).
		Set("description", baseline.Description).
		Set("device_type", baseline.DeviceType).
		Set("version", baseline.Version).
		Set("params", baseline.Params).
		Set("creator", baseline.Creator).
		Set("status", baseline.Status).
		Where(sq.Eq{"id": baseline.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update config_baseline SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update config_baseline: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgBaselineRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Delete("config_baselines").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete config_baseline SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete config_baseline: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgBaselineRepository) List(ctx context.Context, filter BaselineFilter) (*model.ListResponse[BaselineConfig], error) {
	base := storage.Psql.Select(baselineColumns...).From("config_baselines")
	countBase := storage.Psql.Select("COUNT(*)").From("config_baselines")

	if filter.DeviceType != nil {
		base = base.Where(sq.Eq{"device_type": *filter.DeviceType})
		countBase = countBase.Where(sq.Eq{"device_type": *filter.DeviceType})
	}
	if filter.Status != nil {
		base = base.Where(sq.Eq{"status": *filter.Status})
		countBase = countBase.Where(sq.Eq{"status": *filter.Status})
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count config_baseline SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count config_baselines: %w", err)
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
		return nil, fmt.Errorf("build list config_baseline SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list config_baselines: %w", err)
	}
	defer rows.Close()

	var items []BaselineConfig
	for rows.Next() {
		b, err := scanBaselineRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan config_baseline row: %w", err)
		}
		items = append(items, *b)
	}

	if items == nil {
		items = []BaselineConfig{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// ---- baseline scanning helpers ----

func scanBaseline(row pgx.Row) (*BaselineConfig, error) {
	var b BaselineConfig
	var paramsJSON []byte

	err := row.Scan(
		&b.ID, &b.BaselineName, &b.Description, &b.DeviceType, &b.Version,
		&paramsJSON, &b.Creator, &b.Status, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if paramsJSON != nil {
		b.Params = json.RawMessage(paramsJSON)
	}
	if b.Params == nil {
		b.Params = json.RawMessage("[]")
	}
	return &b, nil
}

func scanBaselineRow(rows pgx.Rows) (*BaselineConfig, error) {
	var b BaselineConfig
	var paramsJSON []byte

	err := rows.Scan(
		&b.ID, &b.BaselineName, &b.Description, &b.DeviceType, &b.Version,
		&paramsJSON, &b.Creator, &b.Status, &b.CreatedAt, &b.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if paramsJSON != nil {
		b.Params = json.RawMessage(paramsJSON)
	}
	if b.Params == nil {
		b.Params = json.RawMessage("[]")
	}
	return &b, nil
}

// ======================================================================
// PgConfigTaskRepository
// ======================================================================

var _ ConfigTaskRepository = (*PgConfigTaskRepository)(nil)

// PgConfigTaskRepository is a PostgreSQL implementation of ConfigTaskRepository.
type PgConfigTaskRepository struct {
	pool *pgxpool.Pool
}

// NewPgConfigTaskRepository creates a new PgConfigTaskRepository.
func NewPgConfigTaskRepository(pool *pgxpool.Pool) *PgConfigTaskRepository {
	return &PgConfigTaskRepository{pool: pool}
}

func (r *PgConfigTaskRepository) Create(ctx context.Context, task *ConfigTask) error {
	if task.DeviceSns == nil {
		task.DeviceSns = json.RawMessage("[]")
	}

	query, args, err := storage.Psql.Insert("config_tasks").
		Columns("task_name", "task_type", "device_sns", "template_id",
			"baseline_id", "params", "status", "progress", "success_count",
			"fail_count", "total_count", "creator", "message").
		Values(task.TaskName, task.TaskType, task.DeviceSns, task.TemplateID,
			task.BaselineID, task.Params, task.Status, task.Progress,
			task.SuccessCount, task.FailCount, task.TotalCount,
			task.Creator, task.Message).
		Suffix("RETURNING " + joinColumns(configTaskColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert config_task SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanConfigTask(row)
	if err != nil {
		return fmt.Errorf("create config_task: %w", err)
	}
	*task = *created
	return nil
}

func (r *PgConfigTaskRepository) List(ctx context.Context, filter ConfigTaskFilter) (*model.ListResponse[ConfigTask], error) {
	base := storage.Psql.Select(configTaskColumns...).From("config_tasks")
	countBase := storage.Psql.Select("COUNT(*)").From("config_tasks")

	if filter.Status != nil {
		base = base.Where(sq.Eq{"status": *filter.Status})
		countBase = countBase.Where(sq.Eq{"status": *filter.Status})
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count config_task SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count config_tasks: %w", err)
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
		return nil, fmt.Errorf("build list config_task SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list config_tasks: %w", err)
	}
	defer rows.Close()

	var items []ConfigTask
	for rows.Next() {
		t, err := scanConfigTaskRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan config_task row: %w", err)
		}
		items = append(items, *t)
	}

	if items == nil {
		items = []ConfigTask{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// ---- config task scanning helpers ----

func scanConfigTask(row pgx.Row) (*ConfigTask, error) {
	var t ConfigTask
	var deviceSnsJSON []byte
	var paramsJSON []byte

	err := row.Scan(
		&t.ID, &t.TaskName, &t.TaskType, &deviceSnsJSON, &t.TemplateID,
		&t.BaselineID, &paramsJSON, &t.Status, &t.Progress, &t.SuccessCount,
		&t.FailCount, &t.TotalCount, &t.Creator, &t.Message,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if deviceSnsJSON != nil {
		t.DeviceSns = json.RawMessage(deviceSnsJSON)
	}
	if t.DeviceSns == nil {
		t.DeviceSns = json.RawMessage("[]")
	}
	if paramsJSON != nil {
		t.Params = json.RawMessage(paramsJSON)
	}
	return &t, nil
}

func scanConfigTaskRow(rows pgx.Rows) (*ConfigTask, error) {
	var t ConfigTask
	var deviceSnsJSON []byte
	var paramsJSON []byte

	err := rows.Scan(
		&t.ID, &t.TaskName, &t.TaskType, &deviceSnsJSON, &t.TemplateID,
		&t.BaselineID, &paramsJSON, &t.Status, &t.Progress, &t.SuccessCount,
		&t.FailCount, &t.TotalCount, &t.Creator, &t.Message,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if deviceSnsJSON != nil {
		t.DeviceSns = json.RawMessage(deviceSnsJSON)
	}
	if t.DeviceSns == nil {
		t.DeviceSns = json.RawMessage("[]")
	}
	if paramsJSON != nil {
		t.Params = json.RawMessage(paramsJSON)
	}
	return &t, nil
}

// ======================================================================
// PgNeighborRepository
// ======================================================================

var _ NeighborRepository = (*PgNeighborRepository)(nil)

// PgNeighborRepository is a PostgreSQL implementation of NeighborRepository.
type PgNeighborRepository struct {
	pool *pgxpool.Pool
}

// NewPgNeighborRepository creates a new PgNeighborRepository.
func NewPgNeighborRepository(pool *pgxpool.Pool) *PgNeighborRepository {
	return &PgNeighborRepository{pool: pool}
}

func (r *PgNeighborRepository) List(ctx context.Context, filter NeighborFilter) (*model.ListResponse[NeighborParam], error) {
	base := storage.Psql.Select(neighborColumns...).From("config_neighbors")
	countBase := storage.Psql.Select("COUNT(*)").From("config_neighbors")

	if filter.SourceCellID != nil {
		base = base.Where(sq.Eq{"source_cell_id": *filter.SourceCellID})
		countBase = countBase.Where(sq.Eq{"source_cell_id": *filter.SourceCellID})
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count config_neighbor SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count config_neighbors: %w", err)
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
		return nil, fmt.Errorf("build list config_neighbor SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list config_neighbors: %w", err)
	}
	defer rows.Close()

	var items []NeighborParam
	for rows.Next() {
		n, err := scanNeighborRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan config_neighbor row: %w", err)
		}
		items = append(items, *n)
	}

	if items == nil {
		items = []NeighborParam{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// ---- neighbor scanning helpers ----

func scanNeighborRow(rows pgx.Rows) (*NeighborParam, error) {
	var n NeighborParam
	var paramsJSON []byte

	err := rows.Scan(
		&n.ID, &n.SourceCellID, &n.SourceCellName, &n.TargetCellID,
		&n.TargetCellName, &n.NeighborType, &paramsJSON,
		&n.CreatedAt, &n.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if paramsJSON != nil {
		n.Params = json.RawMessage(paramsJSON)
	}
	if n.Params == nil {
		n.Params = json.RawMessage("{}")
	}
	return &n, nil
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
