package mml

import (
	"context"
	"encoding/json"
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

// ---- sort whitelists (prevent SQL injection in ORDER BY) ----

var commandAllowedSortColumns = map[string]bool{
	"command_name": true,
	"command_code": true,
	"category":     true,
	"created_at":   true,
}

var scriptAllowedSortColumns = map[string]bool{
	"script_name": true,
	"device_type": true,
	"creator":     true,
	"created_at":  true,
	"updated_at":  true,
}

var taskAllowedSortColumns = map[string]bool{
	"task_name":    true,
	"status":       true,
	"execute_type": true,
	"creator":      true,
	"result":       true,
	"created_at":   true,
	"updated_at":   true,
	"started_at":   true,
	"finished_at":  true,
}

// ---- column lists ----

var commandColumns = []string{
	"id", "command_name", "command_code", "category",
	"description", "rpc_method", "operation_type", "param_template", "param_paths",
	"supported_operations", "help_doc", "notes", "product_types",
	"created_at",
}

var scriptColumns = []string{
	"id", "script_name", "description", "content",
	"device_type", "creator", "tags",
	"created_at", "updated_at",
}

var taskColumns = []string{
	"id", "task_name", "script_id", "device_sns",
	"commands", "status", "results", "creator", "executor",
	"created_at", "updated_at",
	"execute_type", "scheduled_at",
	"period_start", "period_end", "period_time",
	"offline_retry", "offline_retry_wait",
	"failed_retry", "failed_retry_count", "failed_retry_interval",
	"started_at", "finished_at",
	"total_devices", "success_count", "failed_count", "result",
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
	query, args, err := storage.Psql.Select(commandColumns...).
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
	query, args, err := storage.Psql.Select(commandColumns...).
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
	base := storage.Psql.Select(commandColumns...).From("mml_commands")
	countBase := storage.Psql.Select("COUNT(*)").From("mml_commands")

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
	sortBy := "created_at"
	if filter.SortBy != "" && commandAllowedSortColumns[filter.SortBy] {
		sortBy = filter.SortBy
	}
	sortDir := "DESC"
	if filter.SortDir == "asc" {
		sortDir = "ASC"
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

// parseStringArray parses a []byte that may be either a PostgreSQL TEXT[] ({a,b})
// or a JSONB array (["a","b"]). Returns an empty slice on nil/empty input.
func parseStringArray(data []byte) []string {
	if len(data) == 0 {
		return []string{}
	}
	// Try JSON format first (starts with '[')
	if data[0] == '[' {
		var result []string
		if err := json.Unmarshal(data, &result); err == nil {
			return result
		}
	}
	// Try PostgreSQL array format (starts with '{')
	if data[0] == '{' {
		var result []string
		if err := json.Unmarshal([]byte("["+string(data[1:len(data)-1])+"]"), &result); err == nil {
			return result
		}
	}
	// Fallback: treat as single string
	return []string{string(data)}
}

func scanCommand(row pgx.Row) (*MMLCommand, error) {
	var c MMLCommand
	var paramTemplateJSON, paramPathsJSON, productTypesJSON []byte
	var supportedOpsJSON []byte

	err := row.Scan(
		&c.ID, &c.CommandName, &c.CommandCode, &c.Category,
		&c.Description, &c.RPCMethod, &c.OperationType, &paramTemplateJSON, &paramPathsJSON,
		&supportedOpsJSON, &c.HelpDoc, &c.Notes, &productTypesJSON,
		&c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if paramTemplateJSON != nil {
		if err := json.Unmarshal(paramTemplateJSON, &c.ParamTemplate); err != nil {
			return nil, fmt.Errorf("unmarshal param_template: %w", err)
		}
	}
	if c.ParamTemplate == nil {
		c.ParamTemplate = map[string]interface{}{}
	}
	if paramPathsJSON != nil {
		c.ParamPaths = append(c.ParamPaths[:0], paramPathsJSON...)
	}
	if c.ParamPaths == nil {
		c.ParamPaths = json.RawMessage("[]")
	}
	c.SupportedOperations = parseStringArray(supportedOpsJSON)
	if productTypesJSON != nil {
		if err := json.Unmarshal(productTypesJSON, &c.ProductTypes); err != nil {
			return nil, fmt.Errorf("unmarshal product_types: %w", err)
		}
	}
	if c.ProductTypes == nil {
		c.ProductTypes = []string{}
	}
	return &c, nil
}

func scanCommandRow(rows pgx.Rows) (*MMLCommand, error) {
	var c MMLCommand
	var paramTemplateJSON, paramPathsJSON, productTypesJSON []byte
	var supportedOpsJSON []byte

	err := rows.Scan(
		&c.ID, &c.CommandName, &c.CommandCode, &c.Category,
		&c.Description, &c.RPCMethod, &c.OperationType, &paramTemplateJSON, &paramPathsJSON,
		&supportedOpsJSON, &c.HelpDoc, &c.Notes, &productTypesJSON,
		&c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	if paramTemplateJSON != nil {
		if err := json.Unmarshal(paramTemplateJSON, &c.ParamTemplate); err != nil {
			return nil, fmt.Errorf("unmarshal param_template: %w", err)
		}
	}
	if c.ParamTemplate == nil {
		c.ParamTemplate = map[string]interface{}{}
	}
	if paramPathsJSON != nil {
		c.ParamPaths = append(c.ParamPaths[:0], paramPathsJSON...)
	}
	if c.ParamPaths == nil {
		c.ParamPaths = json.RawMessage("[]")
	}
	c.SupportedOperations = parseStringArray(supportedOpsJSON)
	if productTypesJSON != nil {
		if err := json.Unmarshal(productTypesJSON, &c.ProductTypes); err != nil {
			return nil, fmt.Errorf("unmarshal product_types: %w", err)
		}
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
	tagsJSON, err := json.Marshal(script.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	query, args, err := storage.Psql.Insert("mml_scripts").
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
	query, args, err := storage.Psql.Select(scriptColumns...).
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
	tagsJSON, err := json.Marshal(script.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	query, args, err := storage.Psql.Update("mml_scripts").
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
	query, args, err := storage.Psql.Delete("mml_scripts").
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
	base := storage.Psql.Select(scriptColumns...).From("mml_scripts")
	countBase := storage.Psql.Select("COUNT(*)").From("mml_scripts")

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
	sortBy := "created_at"
	if filter.SortBy != "" && scriptAllowedSortColumns[filter.SortBy] {
		sortBy = filter.SortBy
	}
	sortDir := "DESC"
	if filter.SortDir == "asc" {
		sortDir = "ASC"
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
		if err := json.Unmarshal(tagsJSON, &s.Tags); err != nil {
			return nil, fmt.Errorf("unmarshal tags: %w", err)
		}
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
		if err := json.Unmarshal(tagsJSON, &s.Tags); err != nil {
			return nil, fmt.Errorf("unmarshal tags: %w", err)
		}
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
	deviceSNsJSON, err := json.Marshal(task.DeviceSNs)
	if err != nil {
		return fmt.Errorf("marshal device_sns: %w", err)
	}
	commandsJSON, err := json.Marshal(task.Commands)
	if err != nil {
		return fmt.Errorf("marshal commands: %w", err)
	}
	resultsJSON, err := json.Marshal(task.Results)
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}

	query, args, err := storage.Psql.Insert("mml_tasks").
		Columns("task_name", "script_id", "device_sns",
			"commands", "status", "results", "creator", "executor",
			"execute_type", "scheduled_at",
			"period_start", "period_end", "period_time",
			"offline_retry", "offline_retry_wait",
			"failed_retry", "failed_retry_count", "failed_retry_interval",
			"total_devices").
		Values(task.TaskName, task.ScriptID, deviceSNsJSON,
			commandsJSON, task.Status, resultsJSON, task.Creator, task.Executor,
			task.ExecuteType, task.ScheduledAt,
			task.PeriodStart, task.PeriodEnd, task.PeriodTime,
			task.OfflineRetry, task.OfflineRetryWait,
			task.FailedRetry, task.FailedRetryCount, task.FailedRetryInterval,
			task.TotalDevices).
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
	query, args, err := storage.Psql.Select(taskColumns...).
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
	deviceSNsJSON, err := json.Marshal(task.DeviceSNs)
	if err != nil {
		return fmt.Errorf("marshal device_sns: %w", err)
	}
	commandsJSON, err := json.Marshal(task.Commands)
	if err != nil {
		return fmt.Errorf("marshal commands: %w", err)
	}
	resultsJSON, err := json.Marshal(task.Results)
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}

	query, args, err := storage.Psql.Update("mml_tasks").
		Set("task_name", task.TaskName).
		Set("script_id", task.ScriptID).
		Set("device_sns", deviceSNsJSON).
		Set("commands", commandsJSON).
		Set("status", task.Status).
		Set("results", resultsJSON).
		Set("creator", task.Creator).
		Set("execute_type", task.ExecuteType).
		Set("scheduled_at", task.ScheduledAt).
		Set("period_start", task.PeriodStart).
		Set("period_end", task.PeriodEnd).
		Set("period_time", task.PeriodTime).
		Set("offline_retry", task.OfflineRetry).
		Set("offline_retry_wait", task.OfflineRetryWait).
		Set("failed_retry", task.FailedRetry).
		Set("failed_retry_count", task.FailedRetryCount).
		Set("failed_retry_interval", task.FailedRetryInterval).
		Set("started_at", task.StartedAt).
		Set("finished_at", task.FinishedAt).
		Set("total_devices", task.TotalDevices).
		Set("success_count", task.SuccessCount).
		Set("failed_count", task.FailedCount).
		Set("result", task.Result).
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
	base := storage.Psql.Select(taskColumns...).From("mml_tasks")
	countBase := storage.Psql.Select("COUNT(*)").From("mml_tasks")

	if filter.Status != nil {
		base = base.Where(sq.Eq{"status": *filter.Status})
		countBase = countBase.Where(sq.Eq{"status": *filter.Status})
	}
	if filter.ExecuteType != nil {
		base = base.Where(sq.Eq{"execute_type": *filter.ExecuteType})
		countBase = countBase.Where(sq.Eq{"execute_type": *filter.ExecuteType})
	}
	if filter.Result != nil {
		base = base.Where(sq.Eq{"result": *filter.Result})
		countBase = countBase.Where(sq.Eq{"result": *filter.Result})
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
	sortBy := "created_at"
	if filter.SortBy != "" && taskAllowedSortColumns[filter.SortBy] {
		sortBy = filter.SortBy
	}
	sortDir := "DESC"
	if filter.SortDir == "asc" {
		sortDir = "ASC"
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
		&commandsJSON, &t.Status, &resultsJSON, &t.Creator, &t.Executor,
		&t.CreatedAt, &t.UpdatedAt,
		&t.ExecuteType, &t.ScheduledAt,
		&t.PeriodStart, &t.PeriodEnd, &t.PeriodTime,
		&t.OfflineRetry, &t.OfflineRetryWait,
		&t.FailedRetry, &t.FailedRetryCount, &t.FailedRetryInterval,
		&t.StartedAt, &t.FinishedAt,
		&t.TotalDevices, &t.SuccessCount, &t.FailedCount, &t.Result,
	)
	if err != nil {
		return nil, err
	}
	if deviceSNsJSON != nil {
		if err := json.Unmarshal(deviceSNsJSON, &t.DeviceSNs); err != nil {
			return nil, fmt.Errorf("unmarshal device_sns: %w", err)
		}
	}
	if t.DeviceSNs == nil {
		t.DeviceSNs = []string{}
	}
	if commandsJSON != nil {
		if err := json.Unmarshal(commandsJSON, &t.Commands); err != nil {
			return nil, fmt.Errorf("unmarshal commands: %w", err)
		}
	}
	if t.Commands == nil {
		t.Commands = []map[string]interface{}{}
	}
	if resultsJSON != nil {
		if err := json.Unmarshal(resultsJSON, &t.Results); err != nil {
			return nil, fmt.Errorf("unmarshal results: %w", err)
		}
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
		&commandsJSON, &t.Status, &resultsJSON, &t.Creator, &t.Executor,
		&t.CreatedAt, &t.UpdatedAt,
		&t.ExecuteType, &t.ScheduledAt,
		&t.PeriodStart, &t.PeriodEnd, &t.PeriodTime,
		&t.OfflineRetry, &t.OfflineRetryWait,
		&t.FailedRetry, &t.FailedRetryCount, &t.FailedRetryInterval,
		&t.StartedAt, &t.FinishedAt,
		&t.TotalDevices, &t.SuccessCount, &t.FailedCount, &t.Result,
	)
	if err != nil {
		return nil, err
	}
	if deviceSNsJSON != nil {
		if err := json.Unmarshal(deviceSNsJSON, &t.DeviceSNs); err != nil {
			return nil, fmt.Errorf("unmarshal device_sns: %w", err)
		}
	}
	if t.DeviceSNs == nil {
		t.DeviceSNs = []string{}
	}
	if commandsJSON != nil {
		if err := json.Unmarshal(commandsJSON, &t.Commands); err != nil {
			return nil, fmt.Errorf("unmarshal commands: %w", err)
		}
	}
	if t.Commands == nil {
		t.Commands = []map[string]interface{}{}
	}
	if resultsJSON != nil {
		if err := json.Unmarshal(resultsJSON, &t.Results); err != nil {
			return nil, fmt.Errorf("unmarshal results: %w", err)
		}
	}
	if t.Results == nil {
		t.Results = []map[string]interface{}{}
	}
	return &t, nil
}

func (r *PgTaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status TaskStatus) error {
	now := time.Now()

	// Build dynamic SET clause based on target status
	builder := storage.Psql.Update("mml_tasks").
		Set("status", status).
		Set("updated_at", now)

	// Set started_at when entering running state
	if status == TaskRunning {
		builder = builder.Set("started_at", now)
	}
	// Set finished_at for terminal states
	if status == TaskCompleted || status == TaskFailed || status == TaskCancelled {
		builder = builder.Set("finished_at", now)
	}

	query, args, err := builder.Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("build update mml_task status SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update mml_task status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgTaskRepository) IncrementStats(ctx context.Context, id uuid.UUID, successDelta, failedDelta int) error {
	builder := storage.Psql.Update("mml_tasks").
		Set("success_count", sq.Expr("success_count + ?", successDelta)).
		Set("failed_count", sq.Expr("failed_count + ?", failedDelta)).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id})

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build increment mml_task stats SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("increment mml_task stats: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgTaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Delete("mml_tasks").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete mml_task SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete mml_task: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
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

// ======================================================================
// PgTemplateRepository
// ======================================================================

var _ TemplateRepository = (*PgTemplateRepository)(nil)

var templateAllowedSortColumns = map[string]bool{
	"template_name": true,
	"command_code":  true,
	"operation_type": true,
	"template_scope": true,
	"creator":       true,
	"created_at":    true,
	"updated_at":    true,
}

var templateColumns = []string{
	"id", "template_name", "command_code", "operation_type",
	"template_scope", "parameters", "param_paths",
	"description", "product_types", "creator",
	"created_at", "updated_at",
}

// PgTemplateRepository is a PostgreSQL implementation of TemplateRepository.
type PgTemplateRepository struct {
	pool *pgxpool.Pool
}

// NewPgTemplateRepository creates a new PgTemplateRepository.
func NewPgTemplateRepository(pool *pgxpool.Pool) *PgTemplateRepository {
	return &PgTemplateRepository{pool: pool}
}

func (r *PgTemplateRepository) Create(ctx context.Context, tmpl *MMLTemplate) error {
	parametersJSON, err := json.Marshal(tmpl.Parameters)
	if err != nil {
		return fmt.Errorf("marshal parameters: %w", err)
	}
	paramPathsJSON, err := json.Marshal(tmpl.ParamPaths)
	if err != nil {
		return fmt.Errorf("marshal param_paths: %w", err)
	}
	productTypesJSON, err := json.Marshal(tmpl.ProductTypes)
	if err != nil {
		return fmt.Errorf("marshal product_types: %w", err)
	}

	query, args, err := storage.Psql.Insert("mml_templates").
		Columns("template_name", "command_code", "operation_type",
			"template_scope", "category_group", "parameters", "param_paths",
			"description", "product_types", "creator").
		Values(tmpl.TemplateName, tmpl.CommandCode, tmpl.OperationType,
			tmpl.TemplateScope, tmpl.CategoryGroup, parametersJSON, paramPathsJSON,
			tmpl.Description, productTypesJSON, tmpl.Creator).
		Suffix("RETURNING " + joinColumns(templateColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert mml_template SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanTemplate(row)
	if err != nil {
		return fmt.Errorf("create mml_template: %w", err)
	}
	*tmpl = *created
	return nil
}

func (r *PgTemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*MMLTemplate, error) {
	query, args, err := storage.Psql.Select(templateColumns...).
		From("mml_templates").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get mml_template SQL: %w", err)
	}

	tmpl, err := scanTemplate(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get mml_template: %w", err)
	}
	return tmpl, nil
}

func (r *PgTemplateRepository) Update(ctx context.Context, tmpl *MMLTemplate) error {
	parametersJSON, err := json.Marshal(tmpl.Parameters)
	if err != nil {
		return fmt.Errorf("marshal parameters: %w", err)
	}
	paramPathsJSON, err := json.Marshal(tmpl.ParamPaths)
	if err != nil {
		return fmt.Errorf("marshal param_paths: %w", err)
	}
	productTypesJSON, err := json.Marshal(tmpl.ProductTypes)
	if err != nil {
		return fmt.Errorf("marshal product_types: %w", err)
	}

	query, args, err := storage.Psql.Update("mml_templates").
		Set("template_name", tmpl.TemplateName).
		Set("command_code", tmpl.CommandCode).
		Set("operation_type", tmpl.OperationType).
		Set("template_scope", tmpl.TemplateScope).
		Set("category_group", tmpl.CategoryGroup).
		Set("parameters", parametersJSON).
		Set("param_paths", paramPathsJSON).
		Set("description", tmpl.Description).
		Set("product_types", productTypesJSON).
		Where(sq.Eq{"id": tmpl.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update mml_template SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update mml_template: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgTemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Delete("mml_templates").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete mml_template SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete mml_template: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgTemplateRepository) List(ctx context.Context, filter TemplateFilter) (*model.ListResponse[MMLTemplate], error) {
	base := storage.Psql.Select(templateColumns...).From("mml_templates")
	countBase := storage.Psql.Select("COUNT(*)").From("mml_templates")

	// Visibility rules: public templates + user's own private templates
	if filter.Creator != nil && *filter.Creator != "" {
		cond := sq.Or{
			sq.Eq{"template_scope": "public"},
			sq.And{sq.Eq{"template_scope": "private"}, sq.Eq{"creator": *filter.Creator}},
		}
		base = base.Where(cond)
		countBase = countBase.Where(cond)
	}

	if filter.CommandCode != nil {
		base = base.Where(sq.Eq{"command_code": *filter.CommandCode})
		countBase = countBase.Where(sq.Eq{"command_code": *filter.CommandCode})
	}
	if filter.OperationType != nil {
		base = base.Where(sq.Eq{"operation_type": *filter.OperationType})
		countBase = countBase.Where(sq.Eq{"operation_type": *filter.OperationType})
	}
	if filter.TemplateScope != nil {
		base = base.Where(sq.Eq{"template_scope": *filter.TemplateScope})
		countBase = countBase.Where(sq.Eq{"template_scope": *filter.TemplateScope})
	}
	if filter.CategoryGroup != nil {
		base = base.Where(sq.Eq{"category_group": *filter.CategoryGroup})
		countBase = countBase.Where(sq.Eq{"category_group": *filter.CategoryGroup})
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count mml_template SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count mml_templates: %w", err)
	}

	// Pagination
	sortBy := "created_at"
	if filter.SortBy != "" && templateAllowedSortColumns[filter.SortBy] {
		sortBy = filter.SortBy
	}
	sortDir := "DESC"
	if filter.SortDir == "asc" {
		sortDir = "ASC"
	}
	base = base.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list mml_template SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list mml_templates: %w", err)
	}
	defer rows.Close()

	var items []MMLTemplate
	for rows.Next() {
		tmpl, err := scanTemplateRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan mml_template row: %w", err)
		}
		items = append(items, *tmpl)
	}

	if items == nil {
		items = []MMLTemplate{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// ---- template scanning helpers ----

func scanTemplate(row pgx.Row) (*MMLTemplate, error) {
	var t MMLTemplate
	var parametersJSON, paramPathsJSON, productTypesJSON []byte

	err := row.Scan(
		&t.ID, &t.TemplateName, &t.CommandCode, &t.OperationType,
		&t.TemplateScope, &t.CategoryGroup, &parametersJSON, &paramPathsJSON,
		&t.Description, &productTypesJSON, &t.Creator,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if parametersJSON != nil {
		if err := json.Unmarshal(parametersJSON, &t.Parameters); err != nil {
			return nil, fmt.Errorf("unmarshal parameters: %w", err)
		}
	}
	if t.Parameters == nil {
		t.Parameters = map[string]interface{}{}
	}
	if paramPathsJSON != nil {
		if err := json.Unmarshal(paramPathsJSON, &t.ParamPaths); err != nil {
			return nil, fmt.Errorf("unmarshal param_paths: %w", err)
		}
	}
	if t.ParamPaths == nil {
		t.ParamPaths = []string{}
	}
	if productTypesJSON != nil {
		if err := json.Unmarshal(productTypesJSON, &t.ProductTypes); err != nil {
			return nil, fmt.Errorf("unmarshal product_types: %w", err)
		}
	}
	if t.ProductTypes == nil {
		t.ProductTypes = []string{}
	}
	return &t, nil
}

func scanTemplateRow(rows pgx.Rows) (*MMLTemplate, error) {
	var t MMLTemplate
	var parametersJSON, paramPathsJSON, productTypesJSON []byte

	err := rows.Scan(
		&t.ID, &t.TemplateName, &t.CommandCode, &t.OperationType,
		&t.TemplateScope, &t.CategoryGroup, &parametersJSON, &paramPathsJSON,
		&t.Description, &productTypesJSON, &t.Creator,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if parametersJSON != nil {
		if err := json.Unmarshal(parametersJSON, &t.Parameters); err != nil {
			return nil, fmt.Errorf("unmarshal parameters: %w", err)
		}
	}
	if t.Parameters == nil {
		t.Parameters = map[string]interface{}{}
	}
	if paramPathsJSON != nil {
		if err := json.Unmarshal(paramPathsJSON, &t.ParamPaths); err != nil {
			return nil, fmt.Errorf("unmarshal param_paths: %w", err)
		}
	}
	if t.ParamPaths == nil {
		t.ParamPaths = []string{}
	}
	if productTypesJSON != nil {
		if err := json.Unmarshal(productTypesJSON, &t.ProductTypes); err != nil {
			return nil, fmt.Errorf("unmarshal product_types: %w", err)
		}
	}
	if t.ProductTypes == nil {
		t.ProductTypes = []string{}
	}
	return &t, nil
}


// ---- Audit Repository ----

// PgAuditRepository implements AuditRepository with PostgreSQL.
type PgAuditRepository struct {
	pool *pgxpool.Pool
}

// NewPgAuditRepository creates a new PgAuditRepository.
func NewPgAuditRepository(pool *pgxpool.Pool) *PgAuditRepository {
	return &PgAuditRepository{pool: pool}
}

// Create inserts a single audit log entry.
func (r *PgAuditRepository) Create(ctx context.Context, entry *MMLAuditLog) error {
	paramsJSON, _ := json.Marshal(entry.Parameters)
	paramPathsJSON, _ := json.Marshal(entry.ParamPaths)

	_, err := r.pool.Exec(ctx,
		`INSERT INTO mml_audit_log (task_id, command_code, operation_type, device_sn, parameters, param_paths, result_status, result_message, creator, duration_ms)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		entry.TaskID, entry.CommandCode, entry.OperationType, entry.DeviceSN,
		paramsJSON, paramPathsJSON, entry.ResultStatus, entry.ResultMessage,
		entry.Creator, entry.DurationMs,
	)
	if err != nil {
		return fmt.Errorf("insert mml_audit_log: %w", err)
	}
	return nil
}

// CreateBatch inserts multiple audit log entries in a single transaction.
func (r *PgAuditRepository) CreateBatch(ctx context.Context, entries []*MMLAuditLog) error {
	if len(entries) == 0 {
		return nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin audit batch tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, entry := range entries {
		paramsJSON, _ := json.Marshal(entry.Parameters)
		paramPathsJSON, _ := json.Marshal(entry.ParamPaths)

		_, err := tx.Exec(ctx,
			`INSERT INTO mml_audit_log (task_id, command_code, operation_type, device_sn, parameters, param_paths, result_status, result_message, creator, duration_ms)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
			entry.TaskID, entry.CommandCode, entry.OperationType, entry.DeviceSN,
			paramsJSON, paramPathsJSON, entry.ResultStatus, entry.ResultMessage,
			entry.Creator, entry.DurationMs,
		)
		if err != nil {
			return fmt.Errorf("insert mml_audit_log batch: %w", err)
		}
	}

	return tx.Commit(ctx)
}
