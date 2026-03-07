package mml

import (
	"context"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/common/errors"
	"github.com/omcgo/omcgo/internal/common/model"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// ---- column lists ----

var commandColumns = []string{
	"id", "command_name", "command_code", "category",
	"description", "rpc_method", "param_template", "product_types",
	"created_at",
}

var scriptColumns = []string{
	"id", "script_name", "description", "content",
	"device_type", "creator", "tags",
	"created_at", "updated_at",
}

var taskColumns = []string{
	"id", "task_name", "script_id", "device_sns",
	"commands", "status", "results", "creator",
	"created_at", "updated_at",
}

// ======================================================================
// PgCommandRepository (read-only)
// ======================================================================

var _ CommandRepository = (*PgCommandRepository)(nil)

// PgCommandRepository is a PostgreSQL implementation of CommandRepository.
type PgCommandRepository struct {
	pool *pgxpool.Pool
}

// NewPgCommandRepository creates a new PgCommandRepository.
func NewPgCommandRepository(pool *pgxpool.Pool) *PgCommandRepository {
	return &PgCommandRepository{pool: pool}
}

func (r *PgCommandRepository) GetByID(ctx context.Context, id uuid.UUID) (*MMLCommand, error) {
	query, args, err := psql.Select(commandColumns...).
		From("mml_commands").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get mml_command SQL: %w", err)
	}

	cmd, err := scanCommand(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get mml_command: %w", err)
	}
	return cmd, nil
}

func (r *PgCommandRepository) GetByCode(ctx context.Context, code string) (*MMLCommand, error) {
	query, args, err := psql.Select(commandColumns...).
		From("mml_commands").
		Where(sq.Eq{"command_code": code}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get mml_command by code SQL: %w", err)
	}

	cmd, err := scanCommand(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get mml_command by code: %w", err)
	}
	return cmd, nil
}

func (r *PgCommandRepository) List(ctx context.Context, filter CommandFilter) (*model.ListResponse[MMLCommand], error) {
	base := psql.Select(commandColumns...).From("mml_commands")
	countBase := psql.Select("COUNT(*)").From("mml_commands")

	if filter.Category != nil {
		base = base.Where(sq.Eq{"category": *filter.Category})
		countBase = countBase.Where(sq.Eq{"category": *filter.Category})
	}
	if filter.Search != nil {
		like := "%" + *filter.Search + "%"
		cond := sq.Or{
			sq.Like{"command_name": like},
			sq.Like{"command_code": like},
			sq.Like{"description": like},
		}
		base = base.Where(cond)
		countBase = countBase.Where(cond)
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count mml_command SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count mml_commands: %w", err)
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
		return nil, fmt.Errorf("build list mml_command SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list mml_commands: %w", err)
	}
	defer rows.Close()

	var items []MMLCommand
	for rows.Next() {
		cmd, err := scanCommandRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan mml_command row: %w", err)
		}
		items = append(items, *cmd)
	}

	if items == nil {
		items = []MMLCommand{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// ---- command scanning helpers ----

func scanCommand(row pgx.Row) (*MMLCommand, error) {
	var c MMLCommand
	var paramTemplateJSON, productTypesJSON []byte

	err := row.Scan(
		&c.ID, &c.CommandName, &c.CommandCode, &c.Category,
		&c.Description, &c.RPCMethod, &paramTemplateJSON, &productTypesJSON,
		&c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if paramTemplateJSON != nil {
		_ = json.Unmarshal(paramTemplateJSON, &c.ParamTemplate)
	}
	if c.ParamTemplate == nil {
		c.ParamTemplate = map[string]interface{}{}
	}
	if productTypesJSON != nil {
		_ = json.Unmarshal(productTypesJSON, &c.ProductTypes)
	}
	if c.ProductTypes == nil {
		c.ProductTypes = []string{}
	}
	return &c, nil
}

func scanCommandRow(rows pgx.Rows) (*MMLCommand, error) {
	var c MMLCommand
	var paramTemplateJSON, productTypesJSON []byte

	err := rows.Scan(
		&c.ID, &c.CommandName, &c.CommandCode, &c.Category,
		&c.Description, &c.RPCMethod, &paramTemplateJSON, &productTypesJSON,
		&c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if paramTemplateJSON != nil {
		_ = json.Unmarshal(paramTemplateJSON, &c.ParamTemplate)
	}
	if c.ParamTemplate == nil {
		c.ParamTemplate = map[string]interface{}{}
	}
	if productTypesJSON != nil {
		_ = json.Unmarshal(productTypesJSON, &c.ProductTypes)
	}
	if c.ProductTypes == nil {
		c.ProductTypes = []string{}
	}
	return &c, nil
}

// ======================================================================
// PgScriptRepository
// ======================================================================

var _ ScriptRepository = (*PgScriptRepository)(nil)

// PgScriptRepository is a PostgreSQL implementation of ScriptRepository.
type PgScriptRepository struct {
	pool *pgxpool.Pool
}

// NewPgScriptRepository creates a new PgScriptRepository.
func NewPgScriptRepository(pool *pgxpool.Pool) *PgScriptRepository {
	return &PgScriptRepository{pool: pool}
}

func (r *PgScriptRepository) Create(ctx context.Context, script *MMLScript) error {
	tagsJSON, _ := json.Marshal(script.Tags)

	query, args, err := psql.Insert("mml_scripts").
		Columns("script_name", "description", "content",
			"device_type", "creator", "tags").
		Values(script.ScriptName, script.Description, script.Content,
			script.DeviceType, script.Creator, tagsJSON).
		Suffix("RETURNING " + joinColumns(scriptColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert mml_script SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanScript(row)
	if err != nil {
		return fmt.Errorf("create mml_script: %w", err)
	}
	*script = *created
	return nil
}

func (r *PgScriptRepository) GetByID(ctx context.Context, id uuid.UUID) (*MMLScript, error) {
	query, args, err := psql.Select(scriptColumns...).
		From("mml_scripts").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get mml_script SQL: %w", err)
	}

	script, err := scanScript(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get mml_script: %w", err)
	}
	return script, nil
}

func (r *PgScriptRepository) Update(ctx context.Context, script *MMLScript) error {
	tagsJSON, _ := json.Marshal(script.Tags)

	query, args, err := psql.Update("mml_scripts").
		Set("script_name", script.ScriptName).
		Set("description", script.Description).
		Set("content", script.Content).
		Set("device_type", script.DeviceType).
		Set("creator", script.Creator).
		Set("tags", tagsJSON).
		Where(sq.Eq{"id": script.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update mml_script SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update mml_script: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgScriptRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := psql.Delete("mml_scripts").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete mml_script SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete mml_script: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgScriptRepository) List(ctx context.Context, filter ScriptFilter) (*model.ListResponse[MMLScript], error) {
	base := psql.Select(scriptColumns...).From("mml_scripts")
	countBase := psql.Select("COUNT(*)").From("mml_scripts")

	if filter.DeviceType != nil {
		base = base.Where(sq.Eq{"device_type": *filter.DeviceType})
		countBase = countBase.Where(sq.Eq{"device_type": *filter.DeviceType})
	}
	if filter.Creator != nil {
		base = base.Where(sq.Eq{"creator": *filter.Creator})
		countBase = countBase.Where(sq.Eq{"creator": *filter.Creator})
	}
	if filter.Search != nil {
		like := "%" + *filter.Search + "%"
		cond := sq.Or{
			sq.Like{"script_name": like},
			sq.Like{"description": like},
		}
		base = base.Where(cond)
		countBase = countBase.Where(cond)
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count mml_script SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count mml_scripts: %w", err)
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
		return nil, fmt.Errorf("build list mml_script SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list mml_scripts: %w", err)
	}
	defer rows.Close()

	var items []MMLScript
	for rows.Next() {
		script, err := scanScriptRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan mml_script row: %w", err)
		}
		items = append(items, *script)
	}

	if items == nil {
		items = []MMLScript{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// ---- script scanning helpers ----

func scanScript(row pgx.Row) (*MMLScript, error) {
	var s MMLScript
	var tagsJSON []byte

	err := row.Scan(
		&s.ID, &s.ScriptName, &s.Description, &s.Content,
		&s.DeviceType, &s.Creator, &tagsJSON,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if tagsJSON != nil {
		_ = json.Unmarshal(tagsJSON, &s.Tags)
	}
	if s.Tags == nil {
		s.Tags = []string{}
	}
	return &s, nil
}

func scanScriptRow(rows pgx.Rows) (*MMLScript, error) {
	var s MMLScript
	var tagsJSON []byte

	err := rows.Scan(
		&s.ID, &s.ScriptName, &s.Description, &s.Content,
		&s.DeviceType, &s.Creator, &tagsJSON,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if tagsJSON != nil {
		_ = json.Unmarshal(tagsJSON, &s.Tags)
	}
	if s.Tags == nil {
		s.Tags = []string{}
	}
	return &s, nil
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

func (r *PgTaskRepository) Create(ctx context.Context, task *MMLTask) error {
	deviceSNsJSON, _ := json.Marshal(task.DeviceSNs)
	commandsJSON, _ := json.Marshal(task.Commands)
	resultsJSON, _ := json.Marshal(task.Results)

	query, args, err := psql.Insert("mml_tasks").
		Columns("task_name", "script_id", "device_sns",
			"commands", "status", "results", "creator").
		Values(task.TaskName, task.ScriptID, deviceSNsJSON,
			commandsJSON, task.Status, resultsJSON, task.Creator).
		Suffix("RETURNING " + joinColumns(taskColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert mml_task SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanTask(row)
	if err != nil {
		return fmt.Errorf("create mml_task: %w", err)
	}
	*task = *created
	return nil
}

func (r *PgTaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*MMLTask, error) {
	query, args, err := psql.Select(taskColumns...).
		From("mml_tasks").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get mml_task SQL: %w", err)
	}

	task, err := scanTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get mml_task: %w", err)
	}
	return task, nil
}

func (r *PgTaskRepository) Update(ctx context.Context, task *MMLTask) error {
	deviceSNsJSON, _ := json.Marshal(task.DeviceSNs)
	commandsJSON, _ := json.Marshal(task.Commands)
	resultsJSON, _ := json.Marshal(task.Results)

	query, args, err := psql.Update("mml_tasks").
		Set("task_name", task.TaskName).
		Set("script_id", task.ScriptID).
		Set("device_sns", deviceSNsJSON).
		Set("commands", commandsJSON).
		Set("status", task.Status).
		Set("results", resultsJSON).
		Set("creator", task.Creator).
		Where(sq.Eq{"id": task.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update mml_task SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update mml_task: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgTaskRepository) List(ctx context.Context, filter TaskFilter) (*model.ListResponse[MMLTask], error) {
	base := psql.Select(taskColumns...).From("mml_tasks")
	countBase := psql.Select("COUNT(*)").From("mml_tasks")

	if filter.Status != nil {
		base = base.Where(sq.Eq{"status": *filter.Status})
		countBase = countBase.Where(sq.Eq{"status": *filter.Status})
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count mml_task SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count mml_tasks: %w", err)
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
		return nil, fmt.Errorf("build list mml_task SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list mml_tasks: %w", err)
	}
	defer rows.Close()

	var items []MMLTask
	for rows.Next() {
		task, err := scanTaskRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan mml_task row: %w", err)
		}
		items = append(items, *task)
	}

	if items == nil {
		items = []MMLTask{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// ---- task scanning helpers ----

func scanTask(row pgx.Row) (*MMLTask, error) {
	var t MMLTask
	var deviceSNsJSON, commandsJSON, resultsJSON []byte

	err := row.Scan(
		&t.ID, &t.TaskName, &t.ScriptID, &deviceSNsJSON,
		&commandsJSON, &t.Status, &resultsJSON, &t.Creator,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if deviceSNsJSON != nil {
		_ = json.Unmarshal(deviceSNsJSON, &t.DeviceSNs)
	}
	if t.DeviceSNs == nil {
		t.DeviceSNs = []string{}
	}
	if commandsJSON != nil {
		_ = json.Unmarshal(commandsJSON, &t.Commands)
	}
	if t.Commands == nil {
		t.Commands = []map[string]interface{}{}
	}
	if resultsJSON != nil {
		_ = json.Unmarshal(resultsJSON, &t.Results)
	}
	if t.Results == nil {
		t.Results = []map[string]interface{}{}
	}
	return &t, nil
}

func scanTaskRow(rows pgx.Rows) (*MMLTask, error) {
	var t MMLTask
	var deviceSNsJSON, commandsJSON, resultsJSON []byte

	err := rows.Scan(
		&t.ID, &t.TaskName, &t.ScriptID, &deviceSNsJSON,
		&commandsJSON, &t.Status, &resultsJSON, &t.Creator,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if deviceSNsJSON != nil {
		_ = json.Unmarshal(deviceSNsJSON, &t.DeviceSNs)
	}
	if t.DeviceSNs == nil {
		t.DeviceSNs = []string{}
	}
	if commandsJSON != nil {
		_ = json.Unmarshal(commandsJSON, &t.Commands)
	}
	if t.Commands == nil {
		t.Commands = []map[string]interface{}{}
	}
	if resultsJSON != nil {
		_ = json.Unmarshal(resultsJSON, &t.Results)
	}
	if t.Results == nil {
		t.Results = []map[string]interface{}{}
	}
	return &t, nil
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
