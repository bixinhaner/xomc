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
	"status", "start_time", "end_time", "type", "progress", "result",
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

// UpdateLifecycle 只写生命周期相关列：status / start_time / end_time / progress / result。
// 不动描述性字段（script_name / content / tags），避免生命周期变更意外覆盖人工编辑。
func (r *PgScriptRepository) UpdateLifecycle(ctx context.Context, script *MMLScript) error {
	resultJSON, err := json.Marshal(script.Result)
	if err != nil {
		return fmt.Errorf("marshal result: %w", err)
	}
	q := storage.Psql.Update("mml_scripts").
		Set("status", script.Status).
		Set("progress", script.Progress).
		Set("result", resultJSON)

	// start_time / end_time 允许 null，分别显式处理。
	if script.StartTime != nil {
		q = q.Set("start_time", *script.StartTime)
	} else {
		q = q.Set("start_time", nil)
	}
	if script.EndTime != nil {
		q = q.Set("end_time", *script.EndTime)
	} else {
		q = q.Set("end_time", nil)
	}

	query, args, err := q.Where(sq.Eq{"id": script.ID}).ToSql()
	if err != nil {
		return fmt.Errorf("build update mml_script lifecycle SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update mml_script lifecycle: %w", err)
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
	var tagsJSON, resultJSON []byte

	err := row.Scan(
		&s.ID, &s.ScriptName, &s.Description, &s.Content,
		&s.DeviceType, &s.Creator, &tagsJSON,
		&s.Status, &s.StartTime, &s.EndTime, &s.Type, &s.Progress, &resultJSON,
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
	if resultJSON != nil && len(resultJSON) > 2 {
		_ = json.Unmarshal(resultJSON, &s.Result)
	}
	return &s, nil
}

func scanScriptRow(rows pgx.Rows) (*MMLScript, error) {
	var s MMLScript
	var tagsJSON, resultJSON []byte

	err := rows.Scan(
		&s.ID, &s.ScriptName, &s.Description, &s.Content,
		&s.DeviceType, &s.Creator, &tagsJSON,
		&s.Status, &s.StartTime, &s.EndTime, &s.Type, &s.Progress, &resultJSON,
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
	if resultJSON != nil && len(resultJSON) > 2 {
		_ = json.Unmarshal(resultJSON, &s.Result)
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
	if filter.TaskName != nil && *filter.TaskName != "" {
		like := "%" + *filter.TaskName + "%"
		base = base.Where(sq.ILike{"task_name": like})
		countBase = countBase.Where(sq.ILike{"task_name": like})
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
// PgCustomCommandRepository
// ======================================================================

var _ CustomCommandRepository = (*PgCustomCommandRepository)(nil)

var customCommandAllowedSortColumns = map[string]bool{
	"command_name":  true,
	"command_code":  true,
	"operation_type": true,
	"command_scope": true,
	"creator":       true,
	"created_at":    true,
	"updated_at":    true,
}

var customCommandColumns = []string{
	"id", "command_name", "command_code", "operation_type",
	"command_scope", "category_group", "parameters", "param_paths",
	"description", "product_types", "creator",
	"created_at", "updated_at",
}

// PgCustomCommandRepository is a PostgreSQL implementation of CustomCommandRepository.
type PgCustomCommandRepository struct {
	pool *pgxpool.Pool
}

// NewPgCustomCommandRepository creates a new PgCustomCommandRepository.
func NewPgCustomCommandRepository(pool *pgxpool.Pool) *PgCustomCommandRepository {
	return &PgCustomCommandRepository{pool: pool}
}

func (r *PgCustomCommandRepository) Create(ctx context.Context, cmd *MMLCustomCommand) error {
	parametersJSON, err := json.Marshal(cmd.Parameters)
	if err != nil {
		return fmt.Errorf("marshal parameters: %w", err)
	}
	paramPathsJSON, err := json.Marshal(cmd.ParamPaths)
	if err != nil {
		return fmt.Errorf("marshal param_paths: %w", err)
	}
	productTypesJSON, err := json.Marshal(cmd.ProductTypes)
	if err != nil {
		return fmt.Errorf("marshal product_types: %w", err)
	}

	query, args, err := storage.Psql.Insert("mml_custom_command").
		Columns("command_name", "command_code", "operation_type",
			"command_scope", "category_group", "parameters", "param_paths",
			"description", "product_types", "creator").
		Values(cmd.CommandName, cmd.CommandCode, cmd.OperationType,
			cmd.CommandScope, cmd.CategoryGroup, parametersJSON, paramPathsJSON,
			cmd.Description, productTypesJSON, cmd.Creator).
		Suffix("RETURNING " + joinColumns(customCommandColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert mml_custom_command SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanCustomCommand(row)
	if err != nil {
		return fmt.Errorf("create mml_custom_command: %w", err)
	}
	*cmd = *created
	return nil
}

func (r *PgCustomCommandRepository) GetByID(ctx context.Context, id uuid.UUID) (*MMLCustomCommand, error) {
	query, args, err := storage.Psql.Select(customCommandColumns...).
		From("mml_custom_command").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get mml_custom_command SQL: %w", err)
	}

	cmd, err := scanCustomCommand(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get mml_custom_command: %w", err)
	}
	return cmd, nil
}

func (r *PgCustomCommandRepository) Update(ctx context.Context, cmd *MMLCustomCommand) error {
	parametersJSON, err := json.Marshal(cmd.Parameters)
	if err != nil {
		return fmt.Errorf("marshal parameters: %w", err)
	}
	paramPathsJSON, err := json.Marshal(cmd.ParamPaths)
	if err != nil {
		return fmt.Errorf("marshal param_paths: %w", err)
	}
	productTypesJSON, err := json.Marshal(cmd.ProductTypes)
	if err != nil {
		return fmt.Errorf("marshal product_types: %w", err)
	}

	query, args, err := storage.Psql.Update("mml_custom_command").
		Set("command_name", cmd.CommandName).
		Set("command_code", cmd.CommandCode).
		Set("operation_type", cmd.OperationType).
		Set("command_scope", cmd.CommandScope).
		Set("category_group", cmd.CategoryGroup).
		Set("parameters", parametersJSON).
		Set("param_paths", paramPathsJSON).
		Set("description", cmd.Description).
		Set("product_types", productTypesJSON).
		Where(sq.Eq{"id": cmd.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update mml_custom_command SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update mml_custom_command: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgCustomCommandRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Delete("mml_custom_command").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete mml_custom_command SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete mml_custom_command: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgCustomCommandRepository) List(ctx context.Context, filter CustomCommandFilter) (*model.ListResponse[MMLCustomCommand], error) {
	base := storage.Psql.Select(customCommandColumns...).From("mml_custom_command")
	countBase := storage.Psql.Select("COUNT(*)").From("mml_custom_command")

	// Visibility rules: public commands + user's own private commands
	if filter.Creator != nil && *filter.Creator != "" {
		cond := sq.Or{
			sq.Eq{"command_scope": "public"},
			sq.And{sq.Eq{"command_scope": "private"}, sq.Eq{"creator": *filter.Creator}},
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
	if filter.CommandScope != nil {
		base = base.Where(sq.Eq{"command_scope": *filter.CommandScope})
		countBase = countBase.Where(sq.Eq{"command_scope": *filter.CommandScope})
	}
	if filter.CategoryGroup != nil {
		base = base.Where(sq.Eq{"category_group": *filter.CategoryGroup})
		countBase = countBase.Where(sq.Eq{"category_group": *filter.CategoryGroup})
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count mml_custom_command SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count mml_custom_commands: %w", err)
	}

	// Pagination
	sortBy := "created_at"
	if filter.SortBy != "" && customCommandAllowedSortColumns[filter.SortBy] {
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
		return nil, fmt.Errorf("build list mml_custom_command SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list mml_custom_commands: %w", err)
	}
	defer rows.Close()

	var items []MMLCustomCommand
	for rows.Next() {
		cmd, err := scanCustomCommandRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan mml_custom_command row: %w", err)
		}
		items = append(items, *cmd)
	}

	if items == nil {
		items = []MMLCustomCommand{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// ---- custom command scanning helpers ----

func scanCustomCommand(row pgx.Row) (*MMLCustomCommand, error) {
	var c MMLCustomCommand
	var parametersJSON, paramPathsJSON, productTypesJSON []byte

	err := row.Scan(
		&c.ID, &c.CommandName, &c.CommandCode, &c.OperationType,
		&c.CommandScope, &c.CategoryGroup, &parametersJSON, &paramPathsJSON,
		&c.Description, &productTypesJSON, &c.Creator,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if parametersJSON != nil {
		if err := json.Unmarshal(parametersJSON, &c.Parameters); err != nil {
			return nil, fmt.Errorf("unmarshal parameters: %w", err)
		}
	}
	if c.Parameters == nil {
		c.Parameters = map[string]interface{}{}
	}
	if paramPathsJSON != nil {
		if err := json.Unmarshal(paramPathsJSON, &c.ParamPaths); err != nil {
			return nil, fmt.Errorf("unmarshal param_paths: %w", err)
		}
	}
	if c.ParamPaths == nil {
		c.ParamPaths = []string{}
	}
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

func scanCustomCommandRow(rows pgx.Rows) (*MMLCustomCommand, error) {
	var c MMLCustomCommand
	var parametersJSON, paramPathsJSON, productTypesJSON []byte

	err := rows.Scan(
		&c.ID, &c.CommandName, &c.CommandCode, &c.OperationType,
		&c.CommandScope, &c.CategoryGroup, &parametersJSON, &paramPathsJSON,
		&c.Description, &productTypesJSON, &c.Creator,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if parametersJSON != nil {
		if err := json.Unmarshal(parametersJSON, &c.Parameters); err != nil {
			return nil, fmt.Errorf("unmarshal parameters: %w", err)
		}
	}
	if c.Parameters == nil {
		c.Parameters = map[string]interface{}{}
	}
	if paramPathsJSON != nil {
		if err := json.Unmarshal(paramPathsJSON, &c.ParamPaths); err != nil {
			return nil, fmt.Errorf("unmarshal param_paths: %w", err)
		}
	}
	if c.ParamPaths == nil {
		c.ParamPaths = []string{}
	}
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

// PgCommandParamRepository is a PostgreSQL implementation of CommandParamRepository.
// Commands reference mml_params directly via mml_command_params_rel.
type PgCommandParamRepository struct {
	pool *pgxpool.Pool
}

func NewPgCommandParamRepository(pool *pgxpool.Pool) *PgCommandParamRepository {
	return &PgCommandParamRepository{pool: pool}
}

func (r *PgCommandParamRepository) ListByCommandID(ctx context.Context, commandID uuid.UUID) ([]MMLParamRef, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT p.id, p.param_code, p.param_name_zh, p.tr069_path, p.value_type, p.is_writable, p.value_constraint
		 FROM mml_params p
		 JOIN mml_command_params_rel r ON r.param_id = p.id
		 WHERE r.command_id = $1
		 ORDER BY r.sort_order`, commandID)
	if err != nil {
		return nil, fmt.Errorf("list params by command: %w", err)
	}
	defer rows.Close()

	var result []MMLParamRef
	for rows.Next() {
		var pr MMLParamRef
		var constraintJSON []byte
		if err := rows.Scan(&pr.ID, &pr.ParamCode, &pr.ParamNameZh, &pr.Tr069Path, &pr.ValueType, &pr.IsWritable, &constraintJSON); err != nil {
			return nil, fmt.Errorf("scan param ref: %w", err)
		}
		if constraintJSON != nil && len(constraintJSON) > 2 {
			_ = json.Unmarshal(constraintJSON, &pr.ValueConstraint)
		}
		result = append(result, pr)
	}
	return result, nil
}

func (r *PgCommandParamRepository) ListByCommandIDs(ctx context.Context, commandIDs []uuid.UUID) (map[uuid.UUID][]MMLParamRef, error) {
	result := make(map[uuid.UUID][]MMLParamRef, len(commandIDs))
	if len(commandIDs) == 0 {
		return result, nil
	}

	query := `SELECT r.command_id, p.id, p.param_code, p.param_name_zh, p.tr069_path, p.value_type, p.is_writable, p.value_constraint
			  FROM mml_params p
			  JOIN mml_command_params_rel r ON r.param_id = p.id
			  WHERE r.command_id = ANY($1)
			  ORDER BY r.command_id, r.sort_order`

	rows, err := r.pool.Query(ctx, query, commandIDs)
	if err != nil {
		return nil, fmt.Errorf("list params by command IDs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cmdID uuid.UUID
		var pr MMLParamRef
		var constraintJSON []byte
		if err := rows.Scan(&cmdID, &pr.ID, &pr.ParamCode, &pr.ParamNameZh, &pr.Tr069Path, &pr.ValueType, &pr.IsWritable, &constraintJSON); err != nil {
			return nil, fmt.Errorf("scan param ref: %w", err)
		}
		if constraintJSON != nil && len(constraintJSON) > 2 {
			_ = json.Unmarshal(constraintJSON, &pr.ValueConstraint)
		}
		result[cmdID] = append(result[cmdID], pr)
	}
	return result, nil
}
