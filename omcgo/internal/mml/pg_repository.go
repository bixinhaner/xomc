package mml

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	appcontext "github.com/omcgo/omcgo/internal/core/context"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/reliability"
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

// commandColumns 对齐 migration 000090 后的 mml_commands schema：
// 老列 (param_template / param_paths / supported_operations / product_types) 已 DROP；
// 新列 (target_paths / target_object / group_id / command_name_i18n / require_confirm /
// confirm_msg_i18n) 由 mmlstandardloader 从 standard-model.xml 写入。
var commandColumns = []string{
	"id", "command_name", "command_code", "category",
	"description", "rpc_method", "operation_type",
	"help_doc", "notes",
	"target_paths", "target_object", "group_id",
	"command_name_i18n", "require_confirm", "confirm_msg_i18n",
	"created_at",
}

var scriptColumns = []string{
	"id", "import_session_id", "script_name", "description", "content",
	"original_filename", "content_sha256", "validation_version", "validated_at",
	"plan_items", "validation_summary",
	"creator", "tags",
	"status", "start_time", "end_time", "type", "progress", "result",
	"last_run_status", "last_run_at",
	"created_at", "updated_at",
}

var taskColumns = []string{
	"id", "task_name", "request_id", "script_id", "script_content_sha256", "script_validation_version", "device_sns",
	"commands", "execute_mode", "plan_items", "status", "results", "creator", "executor",
	"created_at", "updated_at",
	"execute_type", "scheduled_at",
	"period_start", "period_end", "period_time",
	"offline_retry", "offline_retry_wait",
	"failed_retry", "failed_retry_count", "failed_retry_interval",
	"started_at", "finished_at",
	"total_devices", "success_count", "failed_count", "result",
	"next_trigger_at", "parent_task_id",
	// T-0168: 路径翻译审计 4 列（migration 000171）
	"product_resolved", "matched_product_id", "matched_product_class", "path_translation_source",
}

var taskListColumns = []string{
	"id", "task_name", "request_id", "script_id", "script_content_sha256", "script_validation_version", "device_sns",
	"COALESCE(jsonb_array_length(commands), 0) AS command_count",
	"execute_mode", "status", "creator", "executor",
	"COALESCE(jsonb_array_length(plan_items), 0) AS plan_item_count",
	"created_at", "updated_at",
	"execute_type", "scheduled_at",
	"period_start", "period_end", "period_time",
	"offline_retry", "offline_retry_wait",
	"failed_retry", "failed_retry_count", "failed_retry_interval",
	"started_at", "finished_at",
	"total_devices", "success_count", "failed_count", "result",
	"next_trigger_at", "parent_task_id",
	"product_resolved", "matched_product_id", "matched_product_class", "path_translation_source",
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

// ListByGroupID 按 group_id 查询命令（Sprint B Q-V3-1 group 批量执行 API 基础）。
// 排序：先按 operation_type 顺序（LST/MOD/ADD/RMV 习惯），再按 command_code 字典序。
func (r *PgCommandRepository) ListByGroupID(ctx context.Context, groupID uuid.UUID) ([]MMLCommand, error) {
	query, args, err := storage.Psql.Select(commandColumns...).
		From("mml_commands").
		Where(sq.Eq{"group_id": groupID}).
		OrderBy(`
			CASE operation_type
				WHEN 'LST' THEN 1
				WHEN 'MOD' THEN 2
				WHEN 'ADD' THEN 3
				WHEN 'RMV' THEN 4
				ELSE 99
			END`, "command_code ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list mml_commands by group SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list mml_commands by group: %w", err)
	}
	defer rows.Close()
	var out []MMLCommand
	for rows.Next() {
		cmd, err := scanCommand(rows)
		if err != nil {
			return nil, fmt.Errorf("scan mml_command: %w", err)
		}
		out = append(out, *cmd)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mml_commands: %w", err)
	}
	return out, nil
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
	// admin 视图过滤（T-Mml-Admin）：按分组浏览 + 仅看 standard / 排除 customized。
	if filter.GroupID != nil {
		base = base.Where(sq.Eq{"group_id": *filter.GroupID})
		countBase = countBase.Where(sq.Eq{"group_id": *filter.GroupID})
	}
	if filter.Source != nil {
		base = base.Where(sq.Eq{"source": *filter.Source})
		countBase = countBase.Where(sq.Eq{"source": *filter.Source})
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

// scanCommandFields 把 commandColumns 顺序的 row.Scan 结果填入 MMLCommand。
// 抽出来让 row / rows 两种调用复用，避免维护两份字段映射。
func scanCommandFields(scan func(...any) error) (*MMLCommand, error) {
	var c MMLCommand
	var targetPathsJSON, nameI18nJSON, confirmMsgI18nJSON []byte
	var targetObject *string
	var groupID *uuid.UUID

	if err := scan(
		&c.ID, &c.CommandName, &c.CommandCode, &c.Category,
		&c.Description, &c.RPCMethod, &c.OperationType,
		&c.HelpDoc, &c.Notes,
		&targetPathsJSON, &targetObject, &groupID,
		&nameI18nJSON, &c.RequireConfirm, &confirmMsgI18nJSON,
		&c.CreatedAt,
	); err != nil {
		return nil, err
	}

	c.TargetPaths = unmarshalStringArray(targetPathsJSON)
	if targetObject != nil {
		c.TargetObject = *targetObject
	}
	c.GroupID = groupID
	c.CommandNameI18n = unmarshalStringMap(nameI18nJSON)
	c.ConfirmMsgI18n = unmarshalStringMap(confirmMsgI18nJSON)
	// LogicalCode 不再持久化为 DB 列（已 DROP）；读时从 command_code 派生填充，
	// 保证 DTO / renderer / executor 拿到的 cmd.LogicalCode 始终有值。
	c.LogicalCode = deriveLogicalCodeFromCommandCode(c.CommandCode, c.OperationType)
	return &c, nil
}

// unmarshalStringArray decodes a JSONB string array column; bad / null → empty slice.
func unmarshalStringArray(raw []byte) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err != nil {
		return []string{}
	}
	return arr
}

// unmarshalStringMap decodes a JSONB string→string map column; bad / null → empty map.
func unmarshalStringMap(raw []byte) map[string]string {
	if len(raw) == 0 {
		return map[string]string{}
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return map[string]string{}
	}
	return m
}

func scanCommand(row pgx.Row) (*MMLCommand, error) {
	return scanCommandFields(row.Scan)
}

func scanCommandRow(rows pgx.Rows) (*MMLCommand, error) {
	return scanCommandFields(rows.Scan)
}

// ======================================================================
// PgScriptRepository
// ======================================================================

var _ ScriptRepository = (*PgScriptRepository)(nil)
var _ ImportedScriptRepository = (*PgScriptRepository)(nil)

// PgScriptRepository is a PostgreSQL implementation of ScriptRepository.
type PgScriptRepository struct {
	pool *pgxpool.Pool
}

// NewPgScriptRepository creates a new PgScriptRepository.
func NewPgScriptRepository(pool *pgxpool.Pool) *PgScriptRepository {
	return &PgScriptRepository{pool: pool}
}

func (r *PgScriptRepository) Create(ctx context.Context, script *MMLScript) error {
	if script.ImportSessionID == uuid.Nil {
		script.ImportSessionID = uuid.New()
	}
	if script.PlanItems == nil {
		script.PlanItems = []MMLPlanItem{}
	}
	if script.ValidationSummary == nil {
		script.ValidationSummary = JSONMap{}
	}
	tagsJSON, err := json.Marshal(script.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}
	planItemsJSON, err := json.Marshal(script.PlanItems)
	if err != nil {
		return fmt.Errorf("marshal script plan_items: %w", err)
	}
	validationSummaryJSON, err := json.Marshal(script.ValidationSummary)
	if err != nil {
		return fmt.Errorf("marshal script validation_summary: %w", err)
	}

	query, args, err := storage.Psql.Insert("mml_scripts").
		Columns("import_session_id", "script_name", "description", "content",
			"original_filename", "content_sha256", "validation_version", "validated_at",
			"plan_items", "validation_summary", "creator", "tags").
		Values(script.ImportSessionID, script.ScriptName, script.Description, script.Content,
			script.OriginalFilename, script.ContentSHA256, script.ValidationVersion, script.ValidatedAt,
			string(planItemsJSON), string(validationSummaryJSON), script.Creator, tagsJSON).
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

// CreateImported persists a server-authoritative TXT import snapshot. It is
// deliberately separate from the legacy Create entrypoint so import callers
// can depend on the narrower ImportedScriptRepository contract.
func (r *PgScriptRepository) CreateImported(ctx context.Context, script *MMLScript) error {
	return r.Create(ctx, script)
}

func (r *PgScriptRepository) GetByImportSessionID(ctx context.Context, sessionID uuid.UUID) (*MMLScript, error) {
	query, args, err := storage.Psql.Select(scriptColumns...).
		From("mml_scripts").Where(sq.Eq{"import_session_id": sessionID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get mml_script by import session SQL: %w", err)
	}
	script, err := scanScript(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get mml_script by import session: %w", err)
	}
	return script, nil
}

// ReplaceImported atomically replaces the complete TXT-backed snapshot. The
// updated_at predicate prevents a stale editor from overwriting a newer import.
func (r *PgScriptRepository) ReplaceImported(ctx context.Context, script *MMLScript, expectedUpdatedAt time.Time) error {
	if script.PlanItems == nil {
		script.PlanItems = []MMLPlanItem{}
	}
	if script.ValidationSummary == nil {
		script.ValidationSummary = JSONMap{}
	}
	tagsJSON, err := json.Marshal(script.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}
	planItemsJSON, err := json.Marshal(script.PlanItems)
	if err != nil {
		return fmt.Errorf("marshal script plan_items: %w", err)
	}
	validationSummaryJSON, err := json.Marshal(script.ValidationSummary)
	if err != nil {
		return fmt.Errorf("marshal script validation_summary: %w", err)
	}
	query, args, err := storage.Psql.Update("mml_scripts").
		Set("import_session_id", script.ImportSessionID).
		Set("script_name", script.ScriptName).
		Set("description", script.Description).
		Set("content", script.Content).
		Set("original_filename", script.OriginalFilename).
		Set("content_sha256", script.ContentSHA256).
		Set("validation_version", script.ValidationVersion).
		Set("validated_at", script.ValidatedAt).
		Set("plan_items", string(planItemsJSON)).
		Set("validation_summary", string(validationSummaryJSON)).
		Set("tags", tagsJSON).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": script.ID}).
		Where(sq.Eq{"updated_at": expectedUpdatedAt}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build replace imported mml_script SQL: %w", err)
	}
	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("replace imported mml_script: %w", err)
	}
	if result.RowsAffected() == 0 {
		if _, getErr := r.GetByID(ctx, script.ID); getErr != nil {
			if getErr == commonerrors.ErrNotFound || errors.Is(getErr, commonerrors.ErrNotFound) {
				return commonerrors.ErrNotFound
			}
			return fmt.Errorf("check replaced mml_script: %w", getErr)
		}
		return ErrScriptVersionConflict
	}
	updated, err := r.GetByID(ctx, script.ID)
	if err != nil {
		return fmt.Errorf("reload replaced mml_script: %w", err)
	}
	*script = *updated
	return nil
}

func (r *PgScriptRepository) UpdateMetadata(ctx context.Context, id uuid.UUID, name, description string, tags []string) error {
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}
	query, args, err := storage.Psql.Update("mml_scripts").
		Set("script_name", name).Set("description", description).Set("tags", tagsJSON).
		Set("updated_at", sq.Expr("NOW()")).Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("build update mml_script metadata SQL: %w", err)
	}
	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update mml_script metadata: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
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

func (r *PgScriptRepository) NameExistsForCreator(ctx context.Context, creator, name string, excludeID *uuid.UUID) (bool, error) {
	builder := storage.Psql.Select("1").
		From("mml_scripts").
		Where(sq.Eq{"creator": strings.TrimSpace(creator)}).
		Where("lower(btrim(script_name)) = lower(btrim(?))", name).
		Limit(1)
	if excludeID != nil && *excludeID != uuid.Nil {
		builder = builder.Where(sq.NotEq{"id": *excludeID})
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return false, fmt.Errorf("build script name exists SQL: %w", err)
	}
	var one int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&one); err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("check script name exists: %w", err)
	}
	return true, nil
}

func (r *PgScriptRepository) Update(ctx context.Context, script *MMLScript) error {
	if script.PlanItems == nil {
		script.PlanItems = []MMLPlanItem{}
	}
	if script.ValidationSummary == nil {
		script.ValidationSummary = JSONMap{}
	}
	tagsJSON, err := json.Marshal(script.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}
	planItemsJSON, err := json.Marshal(script.PlanItems)
	if err != nil {
		return fmt.Errorf("marshal script plan_items: %w", err)
	}
	validationSummaryJSON, err := json.Marshal(script.ValidationSummary)
	if err != nil {
		return fmt.Errorf("marshal script validation_summary: %w", err)
	}

	query, args, err := storage.Psql.Update("mml_scripts").
		Set("import_session_id", script.ImportSessionID).
		Set("script_name", script.ScriptName).
		Set("description", script.Description).
		Set("content", script.Content).
		Set("original_filename", script.OriginalFilename).
		Set("content_sha256", script.ContentSHA256).
		Set("validation_version", script.ValidationVersion).
		Set("validated_at", script.ValidatedAt).
		Set("plan_items", string(planItemsJSON)).
		Set("validation_summary", string(validationSummaryJSON)).
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

// UpdateLastRun 只写 last_run_status / last_run_at 两列。P1 新增，供
// MMLAggregator 在 mml_task 收敛为终态时回写脚本最近一次执行指针。
func (r *PgScriptRepository) UpdateLastRun(ctx context.Context, id uuid.UUID, status string, at time.Time) error {
	query, args, err := storage.Psql.Update("mml_scripts").
		Set("last_run_status", status).
		Set("last_run_at", at).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update mml_script last_run SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update mml_script last_run: %w", err)
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
	var tagsJSON, planItemsJSON, validationSummaryJSON, resultJSON []byte

	err := row.Scan(
		&s.ID, &s.ImportSessionID, &s.ScriptName, &s.Description, &s.Content,
		&s.OriginalFilename, &s.ContentSHA256, &s.ValidationVersion, &s.ValidatedAt,
		&planItemsJSON, &validationSummaryJSON,
		&s.Creator, &tagsJSON,
		&s.Status, &s.StartTime, &s.EndTime, &s.Type, &s.Progress, &resultJSON,
		&s.LastRunStatus, &s.LastRunAt,
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
	if planItemsJSON != nil {
		if err := json.Unmarshal(planItemsJSON, &s.PlanItems); err != nil {
			return nil, fmt.Errorf("unmarshal script plan_items: %w", err)
		}
	}
	if s.PlanItems == nil {
		s.PlanItems = []MMLPlanItem{}
	}
	if validationSummaryJSON != nil && len(validationSummaryJSON) > 2 {
		if err := json.Unmarshal(validationSummaryJSON, &s.ValidationSummary); err != nil {
			return nil, fmt.Errorf("unmarshal script validation_summary: %w", err)
		}
	}
	if s.ValidationSummary == nil {
		s.ValidationSummary = JSONMap{}
	}
	if resultJSON != nil && len(resultJSON) > 2 {
		_ = json.Unmarshal(resultJSON, &s.Result)
	}
	return &s, nil
}

func scanScriptRow(rows pgx.Rows) (*MMLScript, error) {
	var s MMLScript
	var tagsJSON, planItemsJSON, validationSummaryJSON, resultJSON []byte

	err := rows.Scan(
		&s.ID, &s.ImportSessionID, &s.ScriptName, &s.Description, &s.Content,
		&s.OriginalFilename, &s.ContentSHA256, &s.ValidationVersion, &s.ValidatedAt,
		&planItemsJSON, &validationSummaryJSON,
		&s.Creator, &tagsJSON,
		&s.Status, &s.StartTime, &s.EndTime, &s.Type, &s.Progress, &resultJSON,
		&s.LastRunStatus, &s.LastRunAt,
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
	if planItemsJSON != nil {
		if err := json.Unmarshal(planItemsJSON, &s.PlanItems); err != nil {
			return nil, fmt.Errorf("unmarshal script plan_items: %w", err)
		}
	}
	if s.PlanItems == nil {
		s.PlanItems = []MMLPlanItem{}
	}
	if validationSummaryJSON != nil && len(validationSummaryJSON) > 2 {
		if err := json.Unmarshal(validationSummaryJSON, &s.ValidationSummary); err != nil {
			return nil, fmt.Errorf("unmarshal script validation_summary: %w", err)
		}
	}
	if s.ValidationSummary == nil {
		s.ValidationSummary = JSONMap{}
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
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewPgTaskRepository creates a new PgTaskRepository.
//
// logger 默认为 NopLogger，保持构造签名向后兼容；生产经 WithLogger 注入真实
// logger 以观测事务回滚失败（MEDIUM-18）。
func NewPgTaskRepository(pool *pgxpool.Pool) *PgTaskRepository {
	return &PgTaskRepository{pool: pool, logger: zap.NewNop()}
}

// WithLogger 注入 logger 以记录事务回滚失败，返回自身便于链式调用。
func (r *PgTaskRepository) WithLogger(logger *zap.Logger) *PgTaskRepository {
	if logger != nil {
		r.logger = logger.Named("mml-task-repo")
	}
	return r
}

func (r *PgTaskRepository) Create(ctx context.Context, task *MMLTask) error {
	if task.ExecuteMode == "" {
		task.ExecuteMode = TaskExecuteModeCommon
	}
	if task.PlanItems == nil {
		task.PlanItems = []MMLPlanItem{}
	}
	deviceSNsJSON, err := json.Marshal(task.DeviceSNs)
	if err != nil {
		return fmt.Errorf("marshal device_sns: %w", err)
	}
	commandsJSON, err := json.Marshal(task.Commands)
	if err != nil {
		return fmt.Errorf("marshal commands: %w", err)
	}
	planItemsJSON, err := json.Marshal(task.PlanItems)
	if err != nil {
		return fmt.Errorf("marshal plan_items: %w", err)
	}
	resultsJSON, err := json.Marshal(task.Results)
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}

	// T-0168: 翻译审计列 — product_resolved 默认 true（与 migration 000171 列默认值一致）。
	// matched_product_class / path_translation_source 空串 → 入 NULL（DB CHECK 兼容）。
	var matchedProductClassPtr, pathTranslationSourcePtr *string
	if task.MatchedProductClass != "" {
		mpc := task.MatchedProductClass
		matchedProductClassPtr = &mpc
	}
	if task.PathTranslationSource != "" {
		pts := task.PathTranslationSource
		pathTranslationSourcePtr = &pts
	}

	query, args, err := storage.Psql.Insert("mml_tasks").
		Columns("task_name", "request_id", "script_id", "script_content_sha256", "script_validation_version", "device_sns",
			"commands", "execute_mode", "plan_items", "status", "results", "creator", "executor",
			"execute_type", "scheduled_at",
			"period_start", "period_end", "period_time",
			"offline_retry", "offline_retry_wait",
			"failed_retry", "failed_retry_count", "failed_retry_interval",
			"total_devices",
			"next_trigger_at", "parent_task_id",
			"product_resolved", "matched_product_id", "matched_product_class", "path_translation_source").
		// 2026-05-28 修复:JSONB 列用 string 传(详见 Update 函数注释)。
		// Create 当前能工作是 pgx prepare-cache 路径行为巧合,显式 string 防退化。
		Values(task.TaskName, nullIfEmpty(task.RequestID), task.ScriptID, task.ScriptContentSHA256, task.ScriptValidationVersion, string(deviceSNsJSON),
			string(commandsJSON), task.ExecuteMode, string(planItemsJSON), task.Status, string(resultsJSON), task.Creator, task.Executor,
			task.ExecuteType, task.ScheduledAt,
			task.PeriodStart, task.PeriodEnd, task.PeriodTime,
			task.OfflineRetry, task.OfflineRetryWait,
			task.FailedRetry, task.FailedRetryCount, task.FailedRetryInterval,
			task.TotalDevices,
			task.NextTriggerAt, task.PeriodicParentID,
			task.ProductResolved, task.MatchedProductID, matchedProductClassPtr, pathTranslationSourcePtr).
		Suffix("RETURNING " + joinColumns(taskColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert mml_task SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanTask(row)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("mml task already exists: %w", commonerrors.ErrAlreadyExists)
		}
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

func (r *PgTaskRepository) GetByRequestID(ctx context.Context, creator, requestID string) (*MMLTask, error) {
	query, args, err := storage.Psql.Select(taskColumns...).
		From("mml_tasks").
		Where(sq.Eq{"creator": strings.TrimSpace(creator)}).
		Where(sq.Eq{"request_id": strings.TrimSpace(requestID)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get mml_task by request id SQL: %w", err)
	}
	task, err := scanTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get mml_task by request id: %w", err)
	}
	return task, nil
}

func (r *PgTaskRepository) GetLatestPeriodicChild(ctx context.Context, parentID uuid.UUID) (*MMLTask, error) {
	query, args, err := storage.Psql.Select(taskColumns...).
		From("mml_tasks").
		Where(sq.Eq{"parent_task_id": parentID}).
		OrderBy("created_at DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get latest periodic child SQL: %w", err)
	}

	task, err := scanTask(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get latest periodic child: %w", err)
	}
	return task, nil
}

func (r *PgTaskRepository) GetActiveByScriptID(ctx context.Context, scriptID uuid.UUID) (*MMLTask, error) {
	query := `
		SELECT ` + joinColumns(taskColumns) + `
		  FROM mml_tasks
		 WHERE script_id = $1
		   AND status IN ('pending', 'running', 'paused')
		 ORDER BY CASE WHEN parent_task_id IS NULL THEN 0 ELSE 1 END,
		          created_at DESC
		 LIMIT 1`
	task, err := scanTask(r.pool.QueryRow(ctx, query, scriptID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get active mml_task by script id: %w", err)
	}
	return task, nil
}

func (r *PgTaskRepository) Update(ctx context.Context, task *MMLTask) error {
	if task.ExecuteMode == "" {
		task.ExecuteMode = TaskExecuteModeCommon
	}
	if task.PlanItems == nil {
		task.PlanItems = []MMLPlanItem{}
	}
	deviceSNsJSON, err := json.Marshal(task.DeviceSNs)
	if err != nil {
		return fmt.Errorf("marshal device_sns: %w", err)
	}
	commandsJSON, err := json.Marshal(task.Commands)
	if err != nil {
		return fmt.Errorf("marshal commands: %w", err)
	}
	planItemsJSON, err := json.Marshal(task.PlanItems)
	if err != nil {
		return fmt.Errorf("marshal plan_items: %w", err)
	}
	resultsJSON, err := json.Marshal(task.Results)
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}

	// 2026-05-28 修复:三个 JSONB 列必须用 string 而非 []byte 传入,否则 pgx v5 把
	// []byte 默认按 BYTEA wire format 编码,PG 把 BYTEA 字节流当 text 反解析为 JSON
	// 失败 → "invalid input syntax for type json (SQLSTATE 22P02)" → 整个 UPDATE
	// 失败 → finalizeIfComplete 无法更新 status/result/finished_at → 任务永远卡
	// status=running 虽然 success_count 已等于 total_devices(独立 UPDATE 增 counter
	// 不受影响)。string(...) 让 pgx 当 text 传,PG 自动 parse JSON。
	query, args, err := storage.Psql.Update("mml_tasks").
		Set("task_name", task.TaskName).
		Set("script_id", task.ScriptID).
		Set("script_content_sha256", task.ScriptContentSHA256).
		Set("script_validation_version", task.ScriptValidationVersion).
		Set("device_sns", string(deviceSNsJSON)).
		Set("commands", string(commandsJSON)).
		Set("execute_mode", task.ExecuteMode).
		Set("plan_items", string(planItemsJSON)).
		Set("status", task.Status).
		Set("results", string(resultsJSON)).
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
	base := storage.Psql.Select(taskListColumns...).From("mml_tasks")
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
	if filter.TaskOrigin != nil {
		switch *filter.TaskOrigin {
		case TaskOriginConsole:
			base = base.Where("script_id IS NULL")
			countBase = countBase.Where("script_id IS NULL")
		case TaskOriginScript:
			base = base.Where("script_id IS NOT NULL")
			countBase = countBase.Where("script_id IS NOT NULL")
		}
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
		task, err := scanTaskSummaryRow(rows)
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
	var deviceSNsJSON, commandsJSON, planItemsJSON, resultsJSON []byte
	// T-0168: 翻译审计 4 列；matched_product_class / path_translation_source 用 *string
	// 接 NULL（migration 000171 列允许 NULL）；product_resolved 默认 true，*bool 处理 NULL 兜底。
	var matchedProductClass, pathTranslationSource *string
	var requestID *string

	err := row.Scan(
		&t.ID, &t.TaskName, &requestID, &t.ScriptID, &t.ScriptContentSHA256, &t.ScriptValidationVersion, &deviceSNsJSON,
		&commandsJSON, &t.ExecuteMode, &planItemsJSON, &t.Status, &resultsJSON, &t.Creator, &t.Executor,
		&t.CreatedAt, &t.UpdatedAt,
		&t.ExecuteType, &t.ScheduledAt,
		&t.PeriodStart, &t.PeriodEnd, &t.PeriodTime,
		&t.OfflineRetry, &t.OfflineRetryWait,
		&t.FailedRetry, &t.FailedRetryCount, &t.FailedRetryInterval,
		&t.StartedAt, &t.FinishedAt,
		&t.TotalDevices, &t.SuccessCount, &t.FailedCount, &t.Result,
		&t.NextTriggerAt, &t.PeriodicParentID,
		&t.ProductResolved, &t.MatchedProductID, &matchedProductClass, &pathTranslationSource,
	)
	if err != nil {
		return nil, err
	}
	if matchedProductClass != nil {
		t.MatchedProductClass = *matchedProductClass
	}
	if pathTranslationSource != nil {
		t.PathTranslationSource = *pathTranslationSource
	}
	if requestID != nil {
		t.RequestID = *requestID
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
	t.CommandCount = len(t.Commands)
	if planItemsJSON != nil {
		if err := json.Unmarshal(planItemsJSON, &t.PlanItems); err != nil {
			return nil, fmt.Errorf("unmarshal plan_items: %w", err)
		}
	}
	if t.PlanItems == nil {
		t.PlanItems = []MMLPlanItem{}
	}
	t.PlanItemCount = len(t.PlanItems)
	if t.ExecuteMode == "" {
		t.ExecuteMode = TaskExecuteModeCommon
	}
	if resultsJSON != nil {
		if err := json.Unmarshal(resultsJSON, &t.Results); err != nil {
			return nil, fmt.Errorf("unmarshal results: %w", err)
		}
	}
	if t.Results == nil {
		t.Results = []map[string]interface{}{}
	}
	t.TaskOrigin = deriveTaskOrigin(t.ScriptID)
	return &t, nil
}

func scanTaskRow(rows pgx.Rows) (*MMLTask, error) {
	var t MMLTask
	var deviceSNsJSON, commandsJSON, planItemsJSON, resultsJSON []byte
	// T-0168: 翻译审计列 NULL 接收同 scanTask。
	var matchedProductClass, pathTranslationSource *string
	var requestID *string

	err := rows.Scan(
		&t.ID, &t.TaskName, &requestID, &t.ScriptID, &t.ScriptContentSHA256, &t.ScriptValidationVersion, &deviceSNsJSON,
		&commandsJSON, &t.ExecuteMode, &planItemsJSON, &t.Status, &resultsJSON, &t.Creator, &t.Executor,
		&t.CreatedAt, &t.UpdatedAt,
		&t.ExecuteType, &t.ScheduledAt,
		&t.PeriodStart, &t.PeriodEnd, &t.PeriodTime,
		&t.OfflineRetry, &t.OfflineRetryWait,
		&t.FailedRetry, &t.FailedRetryCount, &t.FailedRetryInterval,
		&t.StartedAt, &t.FinishedAt,
		&t.TotalDevices, &t.SuccessCount, &t.FailedCount, &t.Result,
		&t.NextTriggerAt, &t.PeriodicParentID,
		&t.ProductResolved, &t.MatchedProductID, &matchedProductClass, &pathTranslationSource,
	)
	if err != nil {
		return nil, err
	}
	if matchedProductClass != nil {
		t.MatchedProductClass = *matchedProductClass
	}
	if pathTranslationSource != nil {
		t.PathTranslationSource = *pathTranslationSource
	}
	if requestID != nil {
		t.RequestID = *requestID
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
	t.CommandCount = len(t.Commands)
	if planItemsJSON != nil {
		if err := json.Unmarshal(planItemsJSON, &t.PlanItems); err != nil {
			return nil, fmt.Errorf("unmarshal plan_items: %w", err)
		}
	}
	if t.PlanItems == nil {
		t.PlanItems = []MMLPlanItem{}
	}
	t.PlanItemCount = len(t.PlanItems)
	if t.ExecuteMode == "" {
		t.ExecuteMode = TaskExecuteModeCommon
	}
	if resultsJSON != nil {
		if err := json.Unmarshal(resultsJSON, &t.Results); err != nil {
			return nil, fmt.Errorf("unmarshal results: %w", err)
		}
	}
	if t.Results == nil {
		t.Results = []map[string]interface{}{}
	}
	t.TaskOrigin = deriveTaskOrigin(t.ScriptID)
	return &t, nil
}

func scanTaskSummaryRow(rows pgx.Rows) (*MMLTask, error) {
	var t MMLTask
	var deviceSNsJSON []byte
	var matchedProductClass, pathTranslationSource *string
	var requestID *string

	err := rows.Scan(
		&t.ID, &t.TaskName, &requestID, &t.ScriptID, &t.ScriptContentSHA256, &t.ScriptValidationVersion, &deviceSNsJSON,
		&t.CommandCount,
		&t.ExecuteMode, &t.Status, &t.Creator, &t.Executor,
		&t.PlanItemCount,
		&t.CreatedAt, &t.UpdatedAt,
		&t.ExecuteType, &t.ScheduledAt,
		&t.PeriodStart, &t.PeriodEnd, &t.PeriodTime,
		&t.OfflineRetry, &t.OfflineRetryWait,
		&t.FailedRetry, &t.FailedRetryCount, &t.FailedRetryInterval,
		&t.StartedAt, &t.FinishedAt,
		&t.TotalDevices, &t.SuccessCount, &t.FailedCount, &t.Result,
		&t.NextTriggerAt, &t.PeriodicParentID,
		&t.ProductResolved, &t.MatchedProductID, &matchedProductClass, &pathTranslationSource,
	)
	if err != nil {
		return nil, err
	}
	if requestID != nil {
		t.RequestID = *requestID
	}
	if matchedProductClass != nil {
		t.MatchedProductClass = *matchedProductClass
	}
	if pathTranslationSource != nil {
		t.PathTranslationSource = *pathTranslationSource
	}
	if deviceSNsJSON != nil {
		if err := json.Unmarshal(deviceSNsJSON, &t.DeviceSNs); err != nil {
			return nil, fmt.Errorf("unmarshal device_sns: %w", err)
		}
	}
	if t.DeviceSNs == nil {
		t.DeviceSNs = []string{}
	}
	if t.Commands == nil {
		t.Commands = []map[string]interface{}{}
	}
	if t.PlanItems == nil {
		t.PlanItems = []MMLPlanItem{}
	}
	if t.Results == nil {
		t.Results = []map[string]interface{}{}
	}
	if t.ExecuteMode == "" {
		t.ExecuteMode = TaskExecuteModeCommon
	}
	t.TaskOrigin = deriveTaskOrigin(t.ScriptID)
	return &t, nil
}

func (r *PgTaskRepository) GetResultStatsByID(ctx context.Context, id uuid.UUID) (*MMLTask, error) {
	query, args, err := storage.Psql.Select(
		"id",
		"execute_type",
		"parent_task_id",
		"commands",
		"execute_mode",
		"plan_items",
		"product_resolved",
		"matched_product_id",
		"matched_product_class",
		"path_translation_source",
	).From("mml_tasks").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get mml_task result stats SQL: %w", err)
	}

	var t MMLTask
	var commandsJSON []byte
	var planItemsJSON []byte
	var matchedProductClass, pathTranslationSource *string
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&t.ID,
		&t.ExecuteType,
		&t.PeriodicParentID,
		&commandsJSON,
		&t.ExecuteMode,
		&planItemsJSON,
		&t.ProductResolved,
		&t.MatchedProductID,
		&matchedProductClass,
		&pathTranslationSource,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get mml_task result stats: %w", err)
	}
	if commandsJSON != nil {
		if err := json.Unmarshal(commandsJSON, &t.Commands); err != nil {
			return nil, fmt.Errorf("unmarshal commands: %w", err)
		}
	}
	if t.Commands == nil {
		t.Commands = []map[string]interface{}{}
	}
	if planItemsJSON != nil {
		if err := json.Unmarshal(planItemsJSON, &t.PlanItems); err != nil {
			return nil, fmt.Errorf("unmarshal plan_items: %w", err)
		}
	}
	if t.PlanItems == nil {
		t.PlanItems = []MMLPlanItem{}
	}
	if t.ExecuteMode == "" {
		t.ExecuteMode = TaskExecuteModeCommon
	}
	if matchedProductClass != nil {
		t.MatchedProductClass = *matchedProductClass
	}
	if pathTranslationSource != nil {
		t.PathTranslationSource = *pathTranslationSource
	}
	return &t, nil
}

func deriveTaskOrigin(scriptID *uuid.UUID) TaskOrigin {
	if scriptID != nil {
		return TaskOriginScript
	}
	return TaskOriginConsole
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

// UpdateExportAggregate 记录全设备汇总 CSV 的 object key 与生成时间。
func (r *PgTaskRepository) UpdateExportAggregate(ctx context.Context, id uuid.UUID, objectKey string, t time.Time) error {
	query, args, err := storage.Psql.Update("mml_tasks").
		Set("export_object", objectKey).
		Set("export_generated_at", t).
		Set("updated_at", t).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update mml_task export SQL: %w", err)
	}
	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update mml_task export_object: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

// UpdateExportDevice 把单设备 CSV 的 object key 合并进 device_export_objects（JSONB map）。
func (r *PgTaskRepository) UpdateExportDevice(ctx context.Context, id uuid.UUID, deviceSN, objectKey string, t time.Time) error {
	query, args, err := storage.Psql.Update("mml_tasks").
		Set("device_export_objects", sq.Expr("device_export_objects || jsonb_build_object(?::text, ?::text)", deviceSN, objectKey)).
		Set("export_generated_at", t).
		Set("updated_at", t).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update mml_task device export SQL: %w", err)
	}
	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update mml_task device_export_objects: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgTaskRepository) IncrementStats(ctx context.Context, id uuid.UUID, successDelta, failedDelta int) error {
	// Completion events may be redelivered or handled concurrently across
	// processes. Reconcile from device_tasks instead of applying deltas so the
	// parent counters stay idempotent.
	query := `
WITH stats AS (
	SELECT
		COUNT(*) FILTER (WHERE status = 'completed')::int AS success_count,
		COUNT(*) FILTER (WHERE status IN ('failed', 'expired'))::int AS failed_count
	  FROM device_tasks
	 WHERE source = 'mml'
	   AND source_id = $1
)
UPDATE mml_tasks
   SET success_count = stats.success_count,
       failed_count = stats.failed_count,
       updated_at = $2
  FROM stats
 WHERE mml_tasks.id = $1`
	result, err := r.pool.Exec(ctx, query, id, time.Now())
	if err != nil {
		return fmt.Errorf("increment mml_task stats: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgTaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete mml_task tx: %w", err)
	}
	defer reliability.RollbackTx(ctx, tx, r.logger, "PgTaskRepository.Delete")

	deviceQuery, deviceArgs, err := storage.Psql.Delete("device_tasks").
		Where(sq.Eq{"source": "mml"}).
		Where(sq.Eq{"source_id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete mml device_tasks SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, deviceQuery, deviceArgs...); err != nil {
		return fmt.Errorf("delete mml device_tasks: %w", err)
	}

	query, args, err := storage.Psql.Delete("mml_tasks").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete mml_task SQL: %w", err)
	}

	result, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete mml_task: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete mml_task tx: %w", err)
	}
	return nil
}

// ListByScriptID 返回指定脚本关联的全部执行记录（模板 + 子实例），
// 按 created_at 倒序分页。P4 C11：脚本详情页"历史执行"tab 用。
func (r *PgTaskRepository) ListByScriptID(ctx context.Context, scriptID uuid.UUID, req model.ListRequest) (*model.ListResponse[MMLTask], error) {
	base := storage.Psql.Select(taskListColumns...).
		From("mml_tasks").
		Where(sq.Eq{"script_id": scriptID})
	countBase := storage.Psql.Select("COUNT(*)").
		From("mml_tasks").
		Where(sq.Eq{"script_id": scriptID})

	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count runs SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count runs: %w", err)
	}

	base = base.
		OrderBy("created_at DESC").
		Limit(uint64(req.Limit())).
		Offset(uint64(req.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list runs SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}
	defer rows.Close()

	var items []MMLTask
	for rows.Next() {
		t, err := scanTaskSummaryRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan run row: %w", err)
		}
		items = append(items, *t)
	}
	if items == nil {
		items = []MMLTask{}
	}
	return model.NewListResponse(items, total, req.Page, req.PageSize), nil
}

// -------- Scheduler 支持方法（P2/P3） --------
// 独立实现 ScheduledTaskRepository 接口；与 Scheduler 包对等。

var _ ScheduledTaskRepository = (*PgTaskRepository)(nil)

// ClaimDueTasks 在事务内认领到期 (scheduled / periodic) 任务。
//
// P2 范围：仅处理 execute_type='scheduled' ——
//
//	· SELECT FOR UPDATE SKIP LOCKED LIMIT N 挑出到期行
//	· UPDATE mml_tasks SET status='running', started_at=now, next_trigger_at=NULL
//	· 返回被更新的行（包括 periodic 模板行，留给 P3 阶段在 Scheduler 侧处理）
//
// 多副本部署下同一行不会被多 Scheduler 重复认领；事务提交后才对其它副本可见。
func (r *PgTaskRepository) ClaimDueTasks(ctx context.Context, now time.Time, limit int) ([]*MMLTask, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin claim tx: %w", err)
	}
	defer reliability.RollbackTx(ctx, tx, r.logger, "PgTaskRepository.ClaimDueTasks")

	// 1) 认领 scheduled（一次性任务）——立刻置 running、清 next_trigger_at。
	selectSQL := `
		SELECT id FROM mml_tasks
		WHERE status = 'pending'
		  AND execute_type = 'scheduled'
		  AND next_trigger_at IS NOT NULL
		  AND next_trigger_at <= $1
		ORDER BY next_trigger_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT $2`
	rows, err := tx.Query(ctx, selectSQL, now, limit)
	if err != nil {
		return nil, fmt.Errorf("select scheduled tasks: %w", err)
	}
	var scheduledIDs []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan scheduled id: %w", err)
		}
		scheduledIDs = append(scheduledIDs, id)
	}
	rows.Close()

	var claimed []*MMLTask
	for _, id := range scheduledIDs {
		updateSQL := `
			UPDATE mml_tasks
			SET status = 'running',
			    started_at = $1,
			    next_trigger_at = NULL,
			    updated_at = $1
			WHERE id = $2
			RETURNING ` + joinColumns(taskColumns)
		row := tx.QueryRow(ctx, updateSQL, now, id)
		t, err := scanTask(row)
		if err != nil {
			return nil, fmt.Errorf("claim scheduled task %s: %w", id, err)
		}
		claimed = append(claimed, t)
	}

	// 2) 认领 periodic 模板到期命中 —— 在同一事务内：
	//    · 克隆出子实例（execute_type='immediate'，status='running'）
	//    · 推进模板 next_trigger_at 到下一次 period_time 命中（若已过 period_end 则清空）
	//    返回子实例给 Scheduler 去 fanout。
	//
	// 仅 P3 范围启用；P2 阶段此块依然按设计运行（因为 scheduled / periodic 共用
	// next_trigger_at）——periodic 任务首次被认领即生成子实例，最大化 P2/P3 实现重用。
	periodicSQL := `
		SELECT id FROM mml_tasks
		WHERE status = 'pending'
		  AND execute_type = 'periodic'
		  AND next_trigger_at IS NOT NULL
		  AND next_trigger_at <= $1
		ORDER BY next_trigger_at ASC
		FOR UPDATE SKIP LOCKED
		LIMIT $2`
	rows2, err := tx.Query(ctx, periodicSQL, now, limit)
	if err != nil {
		return nil, fmt.Errorf("select periodic templates: %w", err)
	}
	var periodicIDs []uuid.UUID
	for rows2.Next() {
		var id uuid.UUID
		if err := rows2.Scan(&id); err != nil {
			rows2.Close()
			return nil, fmt.Errorf("scan periodic id: %w", err)
		}
		periodicIDs = append(periodicIDs, id)
	}
	rows2.Close()

	for _, id := range periodicIDs {
		// 读模板完整字段
		parent, err := scanTask(tx.QueryRow(ctx,
			`SELECT `+joinColumns(taskColumns)+` FROM mml_tasks WHERE id=$1`, id))
		if err != nil {
			return nil, fmt.Errorf("load periodic parent %s: %w", id, err)
		}

		// 克隆子实例：继承大多数字段，但 execute_type=immediate、status=running、
		// parent_task_id 指回模板、清空调度相关时间字段。
		child := cloneAsPeriodicChild(parent, now)
		childBytes, _ := json.Marshal(child.DeviceSNs)
		cmdBytes, _ := json.Marshal(child.Commands)
		planBytes, _ := json.Marshal(child.PlanItems)
		resultBytes, _ := json.Marshal(child.Results)

		insertSQL := `
		INSERT INTO mml_tasks (
				task_name, script_id, script_content_sha256, script_validation_version, device_sns, commands, execute_mode, plan_items, status, results, creator, executor,
				execute_type, scheduled_at, period_start, period_end, period_time,
				offline_retry, offline_retry_wait, failed_retry, failed_retry_count, failed_retry_interval,
				total_devices, next_trigger_at, parent_task_id, started_at
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26
			) RETURNING ` + joinColumns(taskColumns)
		row := tx.QueryRow(ctx, insertSQL,
			child.TaskName, child.ScriptID, child.ScriptContentSHA256, child.ScriptValidationVersion, childBytes,
			cmdBytes, child.ExecuteMode, planBytes, child.Status, resultBytes, child.Creator, child.Executor,
			child.ExecuteType, child.ScheduledAt,
			child.PeriodStart, child.PeriodEnd, child.PeriodTime,
			child.OfflineRetry, child.OfflineRetryWait,
			child.FailedRetry, child.FailedRetryCount, child.FailedRetryInterval,
			child.TotalDevices, child.NextTriggerAt, child.PeriodicParentID, child.StartedAt,
		)
		childRow, err := scanTask(row)
		if err != nil {
			return nil, fmt.Errorf("insert periodic child: %w", err)
		}

		// 推进模板 next_trigger_at；若已过 period_end 则清空 + 置 completed。
		next := computeNextPeriodicTrigger(parent, now.Add(1*time.Second))
		if next == nil {
			_, err = tx.Exec(ctx,
				`UPDATE mml_tasks SET status='completed', next_trigger_at=NULL, finished_at=$1, updated_at=$1 WHERE id=$2`,
				now, id)
		} else {
			_, err = tx.Exec(ctx,
				`UPDATE mml_tasks SET next_trigger_at=$1, updated_at=$2 WHERE id=$3`,
				*next, now, id)
		}
		if err != nil {
			return nil, fmt.Errorf("advance periodic parent %s: %w", id, err)
		}

		claimed = append(claimed, childRow)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit claim tx: %w", err)
	}
	return claimed, nil
}

// RecordPeriodicChild 保留签名以满足 ScheduledTaskRepository 接口。
// 由于 ClaimDueTasks 内部已在同事务里完成子实例插入+模板推进，本方法无额外工作。
func (r *PgTaskRepository) RecordPeriodicChild(ctx context.Context, parent, child *MMLTask, parentNext *time.Time) error {
	return nil
}

// FinalizePeriodicParent 如果 period_end 已过，把模板置 completed。
// 与 ClaimDueTasks 的 period_end 自动收敛逻辑等价，这里留作兜底（比如模板
// 从未被命中过就过期的极端情形）。
func (r *PgTaskRepository) FinalizePeriodicParent(ctx context.Context, id uuid.UUID, finishedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE mml_tasks
		SET status='completed', next_trigger_at=NULL, finished_at=$1, updated_at=$1
		WHERE id=$2
		  AND execute_type='periodic'
		  AND status='pending'
		  AND period_end IS NOT NULL
		  AND period_end < $1`, finishedAt, id)
	if err != nil {
		return fmt.Errorf("finalize periodic parent %s: %w", id, err)
	}
	return nil
}

// cloneAsPeriodicChild 从 periodic 模板生成一个子实例，用于立即 fanout。
func cloneAsPeriodicChild(parent *MMLTask, now time.Time) *MMLTask {
	parentID := parent.ID
	child := *parent
	child.ID = uuid.Nil // DB 自动分配
	child.ExecuteType = ExecuteImmediate
	child.Status = TaskRunning
	child.NextTriggerAt = nil
	child.PeriodicParentID = &parentID
	child.StartedAt = &now
	child.FinishedAt = nil
	child.SuccessCount = 0
	child.FailedCount = 0
	child.Results = []map[string]interface{}{}
	// 子实例保留 script_id（若模板有），以便 ResultAggregator 回写 last_run_status
	return &child
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
	"command_name":   true,
	"command_code":   true,
	"operation_type": true,
	"command_scope":  true,
	"creator":        true,
	"created_at":     true,
	"updated_at":     true,
}

// customCommandColumns 列出 SELECT / RETURNING 时返回的列。
// owner_user_id 由 migration 000134 加入；与 creator(varchar) 共存于 Phase 1。
var customCommandColumns = []string{
	"id", "command_name", "command_code", "operation_type",
	"command_scope", "category_group", "parameters", "param_paths",
	"description", "creator", "owner_user_id",
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

func normalizeCustomCommandParamPaths(paramPaths []string) []string {
	normalized := make([]string, 0, len(paramPaths))
	seen := make(map[string]struct{}, len(paramPaths))
	for _, path := range paramPaths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		normalized = append(normalized, path)
	}
	return normalized
}

// syncCustomCommandPaths 让历史 JSON param_paths 与标准 Path 关联表保持一致。
//
// AddTemplateModal 及兼容 API 仍按字符串 Path 提交 param_paths；关联表只存
// standard_path_id。这里在创建/更新自定义命令的同一事务中解析 standard_params，
// 保留仍存在关联的 default_selected，仅删除已移除 Path 并按 JSON 顺序更新 sort_order。
func syncCustomCommandPaths(
	ctx context.Context,
	tx pgx.Tx,
	commandID uuid.UUID,
	paramPaths []string,
) error {
	normalized := normalizeCustomCommandParamPaths(paramPaths)

	if len(normalized) == 0 {
		if _, err := tx.Exec(ctx,
			`DELETE FROM mml_custom_command_paths WHERE command_id = $1`,
			commandID,
		); err != nil {
			return fmt.Errorf("clear custom command paths: %w", err)
		}
		return nil
	}

	rows, err := tx.Query(ctx, `
		SELECT requested.standard_path, sp.id
		FROM unnest($1::text[]) WITH ORDINALITY AS requested(standard_path, sort_order)
		JOIN standard_params sp ON sp.standard_path = requested.standard_path
		ORDER BY requested.sort_order`,
		normalized,
	)
	if err != nil {
		return fmt.Errorf("resolve custom command standard paths: %w", err)
	}
	defer rows.Close()

	standardPathIDs := make(map[string]uuid.UUID, len(normalized))
	for rows.Next() {
		var path string
		var id uuid.UUID
		if err := rows.Scan(&path, &id); err != nil {
			return fmt.Errorf("scan custom command standard path: %w", err)
		}
		standardPathIDs[path] = id
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate custom command standard paths: %w", err)
	}

	missing := make([]string, 0)
	orderedIDs := make([]uuid.UUID, 0, len(normalized))
	for _, path := range normalized {
		id, exists := standardPathIDs[path]
		if !exists {
			missing = append(missing, path)
			continue
		}
		orderedIDs = append(orderedIDs, id)
	}
	if len(missing) > 0 {
		return fmt.Errorf(
			"custom command paths missing from standard_params: %s: %w",
			strings.Join(missing, ", "),
			commonerrors.ErrInvalidInput,
		)
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM mml_custom_command_paths
		WHERE command_id = $1
		  AND NOT (standard_path_id = ANY($2::uuid[]))`,
		commandID,
		orderedIDs,
	); err != nil {
		return fmt.Errorf("delete removed custom command paths: %w", err)
	}

	for sortOrder, standardPathID := range orderedIDs {
		if _, err := tx.Exec(ctx, `
			INSERT INTO mml_custom_command_paths (command_id, standard_path_id, sort_order)
			VALUES ($1, $2, $3)
			ON CONFLICT (command_id, standard_path_id) DO UPDATE
			SET sort_order = EXCLUDED.sort_order,
			    updated_at = now()`,
			commandID,
			standardPathID,
			sortOrder,
		); err != nil {
			return fmt.Errorf("upsert custom command path: %w", err)
		}
	}
	return nil
}

func (r *PgCustomCommandRepository) Create(ctx context.Context, cmd *MMLCustomCommand) error {
	cmd.ParamPaths = normalizeCustomCommandParamPaths(cmd.ParamPaths)
	parametersJSON, err := json.Marshal(cmd.Parameters)
	if err != nil {
		return fmt.Errorf("marshal parameters: %w", err)
	}
	paramPathsJSON, err := json.Marshal(cmd.ParamPaths)
	if err != nil {
		return fmt.Errorf("marshal param_paths: %w", err)
	}

	// migration 000134 Phase 1: dual-write owner_user_id（可 nil）。
	// 历史调用方（如 CloneCustomCommand）未设 OwnerUserID 时落 NULL，与脏数据语义一致。
	query, args, err := storage.Psql.Insert("mml_custom_command").
		Columns("command_name", "command_code", "operation_type",
			"command_scope", "category_group", "parameters", "param_paths",
			"description", "creator", "owner_user_id").
		Values(cmd.CommandName, cmd.CommandCode, cmd.OperationType,
			cmd.CommandScope, cmd.CategoryGroup, parametersJSON, paramPathsJSON,
			cmd.Description, cmd.Creator, cmd.OwnerUserID).
		Suffix("RETURNING " + joinColumns(customCommandColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert mml_custom_command SQL: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin create mml_custom_command: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, query, args...)
	created, err := scanCustomCommand(row)
	if err != nil {
		return fmt.Errorf("create mml_custom_command: %w", err)
	}
	if err := syncCustomCommandPaths(ctx, tx, created.ID, created.ParamPaths); err != nil {
		return fmt.Errorf("sync mml custom command paths: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit create mml_custom_command: %w", err)
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

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin update mml_custom_command: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	latestParamPaths, err := lockCustomCommandParamPaths(ctx, tx, cmd.ID)
	if err != nil {
		return err
	}
	if cmd.ParamPathsProvided {
		cmd.ParamPaths = normalizeCustomCommandParamPaths(cmd.ParamPaths)
	} else {
		cmd.ParamPaths = latestParamPaths
	}
	paramPathsJSON, err := json.Marshal(cmd.ParamPaths)
	if err != nil {
		return fmt.Errorf("marshal param_paths: %w", err)
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
		Where(sq.Eq{"id": cmd.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update mml_custom_command SQL: %w", err)
	}

	result, err := tx.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update mml_custom_command: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	if err := syncCustomCommandPaths(ctx, tx, cmd.ID, cmd.ParamPaths); err != nil {
		return fmt.Errorf("sync mml custom command paths: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit update mml_custom_command: %w", err)
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

	// 可见性规则（T-0090-c）：
	//   - public 始终可见
	//   - private 仅在 (a) creator 匹配 filter.Creator self-fallback，或
	//     (b) creator's RBAC group ∈ filter.VisibleGroupIDs（group-share）时可见
	//   - 两条件均空 → deny-by-default 仅返 public（防止匿名/无凭据请求看到任何 private）
	//
	// SQL 注入防护：所有用户输入走参数化绑定；EXISTS 子查询用列名比较，无字符串拼接。
	// R-NEW-2 mitigation：每个 private 行的可见性都经 OR 短路评估，无第三条隐式可见路径。
	privateClauses := sq.Or{}
	if filter.Creator != nil && *filter.Creator != "" {
		privateClauses = append(privateClauses, sq.Eq{"creator": *filter.Creator})
	}
	if len(filter.VisibleGroupIDs) > 0 {
		privateClauses = append(privateClauses, sq.Expr(
			`EXISTS (
				SELECT 1 FROM users u
				JOIN user_roles ur ON ur.user_id = u.id
				JOIN role_device_groups rdg ON rdg.role_id = ur.role_id
				WHERE u.username = mml_custom_command.creator
				  AND rdg.group_id = ANY(?)
			)`,
			filter.VisibleGroupIDs,
		))
	}

	var visibility sq.Sqlizer
	if len(privateClauses) == 0 {
		// 无任何 private 可见性凭据 → 默认仅 public
		visibility = sq.Eq{"command_scope": "public"}
	} else {
		visibility = sq.Or{
			sq.Eq{"command_scope": "public"},
			sq.And{sq.Eq{"command_scope": "private"}, privateClauses},
		}
	}
	base = base.Where(visibility)
	countBase = countBase.Where(visibility)

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
	var parametersJSON, paramPathsJSON []byte

	// owner_user_id 可为 NULL → *uuid.UUID 直接接收 pgx 的 nil
	err := row.Scan(
		&c.ID, &c.CommandName, &c.CommandCode, &c.OperationType,
		&c.CommandScope, &c.CategoryGroup, &parametersJSON, &paramPathsJSON,
		&c.Description, &c.Creator, &c.OwnerUserID,
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
	return &c, nil
}

func scanCustomCommandRow(rows pgx.Rows) (*MMLCustomCommand, error) {
	var c MMLCustomCommand
	var parametersJSON, paramPathsJSON []byte

	err := rows.Scan(
		&c.ID, &c.CommandName, &c.CommandCode, &c.OperationType,
		&c.CommandScope, &c.CategoryGroup, &parametersJSON, &paramPathsJSON,
		&c.Description, &c.Creator, &c.OwnerUserID,
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
	return &c, nil
}

// NameExistsForPrivate 实现见 CustomCommandRepository 接口注释。
// 关联 migration 000135 的 uq_mml_custom_command_private_name_per_owner 索引。
func (r *PgCustomCommandRepository) NameExistsForPrivate(
	ctx context.Context,
	ownerID uuid.UUID,
	name string,
	excludeID *uuid.UUID,
) (bool, error) {
	if ownerID == uuid.Nil || name == "" {
		return false, nil
	}
	q := storage.Psql.Select("1").
		From("mml_custom_command").
		Where(sq.Eq{"owner_user_id": ownerID}).
		Where(sq.Eq{"command_name": name}).
		Where(sq.Eq{"command_scope": "private"}).
		Limit(1)
	if excludeID != nil && *excludeID != uuid.Nil {
		q = q.Where(sq.NotEq{"id": *excludeID})
	}
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return false, fmt.Errorf("build name-exists SQL: %w", err)
	}
	var dummy int
	scanErr := r.pool.QueryRow(ctx, sqlStr, args...).Scan(&dummy)
	if scanErr != nil {
		if scanErr == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("query name-exists: %w", scanErr)
	}
	return true, nil
}

// NameExistsForPublic 实现见 CustomCommandRepository 接口注释。
// 公共命名空间全局唯一（跨所有用户），仅查询防重、不依赖 DB 唯一约束。
func (r *PgCustomCommandRepository) NameExistsForPublic(
	ctx context.Context,
	name string,
	excludeID *uuid.UUID,
) (bool, error) {
	if name == "" {
		return false, nil
	}
	q := storage.Psql.Select("1").
		From("mml_custom_command").
		Where(sq.Eq{"command_name": name}).
		Where(sq.Eq{"command_scope": "public"}).
		Limit(1)
	if excludeID != nil && *excludeID != uuid.Nil {
		q = q.Where(sq.NotEq{"id": *excludeID})
	}
	sqlStr, args, err := q.ToSql()
	if err != nil {
		return false, fmt.Errorf("build public name-exists SQL: %w", err)
	}
	var dummy int
	scanErr := r.pool.QueryRow(ctx, sqlStr, args...).Scan(&dummy)
	if scanErr != nil {
		if scanErr == pgx.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("query public name-exists: %w", scanErr)
	}
	return true, nil
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
//
// Migration 000090 DROP mml_command_params_rel；migration 000095 用
// mml_command_sub_fields 替代；migration 000113 又把 sub_fields.param_id 切换为
// standard_path_id FK 指向 standard_params（系统级标准 path 字典）。
// 本仓库的读路径据此重写为 JOIN mml_command_sub_fields + standard_params，
// 不再走老的 mml_params + mml_command_params_rel。
type PgCommandParamRepository struct {
	pool *pgxpool.Pool
}

func NewPgCommandParamRepository(pool *pgxpool.Pool) *PgCommandParamRepository {
	return &PgCommandParamRepository{pool: pool}
}

// paramRefSelectExpr 构造 ListByCommandID / ListByCommandIDs 共享的 SELECT
// 列表达式（不含 command_id 与 FROM/WHERE）；保持两个 query 字段顺序一致，
// Scan 才能复用同一序列。
//
// issue #67：param_name_zh 列原先硬编码取 label_i18n->>'zh-CN'，命令执行表单参数名
// 始终中文。改为按请求 locale 取键，并多级 COALESCE 回退（请求语言列 → zh-CN → en-US →
// standard_path），保证英文 locale 下表单参数名本地化、且字典缺该语言时不留空。
// lang 形如 "zh-CN"/"en-US"（长码，与 i18n JSONB 键统一后一致）。
//
// 字段映射（standard_params + sub_field → MMLParamRef）：
//
//	ID            ← csf.id              sub_field 行 id
//	ParamCode     ← csf.mml_code        命令上下文 code（BuildTR069Params MOD 用作 form values key）
//	ParamNameZh   ← label_i18n[lang] 多级回退 standard_path
//	Tr069Path     ← sp.standard_path    Fanouter 翻译为 privatePath 下发
//	ValueType     ← lower(sp.data_type)
//	IsWritable    ← sp.access = 'READ_WRITE'
//	DefaultValue  ← ''                  standard_params 无此字段
//	JsRegex       ← ''                  standard_params 无此字段
//	ValueConstraint ← '{}'              standard_params 仅提供 min/max，BuildTR069Params 暂不消费
func paramRefSelectExpr(lang string) string {
	// 主键名固定 param_name_zh（沿用既有 Scan 目标字段，避免改 MMLParamRef 结构与下游）。
	// requested → zh-CN → en-US → standard_path 四级回退，NULLIF 排除空串键。
	return `csf.id,
       csf.mml_code,
       COALESCE(NULLIF(csf.label_i18n->>` + quoteSQLLiteral(lang) + `, ''),
                NULLIF(csf.label_i18n->>'zh-CN', ''),
                NULLIF(csf.label_i18n->>'en-US', ''),
                sp.standard_path) AS param_name_zh,
       sp.standard_path AS tr069_path,
       lower(COALESCE(sp.data_type, 'string'))    AS value_type,
       (sp.access = 'READ_WRITE')                  AS is_writable,
	       csf.is_required,
       ''::text                                    AS default_value,
       ''::text                                    AS js_regex,
       '{}'::jsonb                                 AS value_constraint`
}

// quoteSQLLiteral 把 locale 码安全嵌入 JSONB ->> 键。locale 取值受
// appcontext.ParseAcceptLanguage 约束为 zh-CN/en-US 两枚白名单常量，不含用户输入；
// 单引号转义仅作纵深防御，杜绝拼接注入面。
func quoteSQLLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func (r *PgCommandParamRepository) ListByCommandID(ctx context.Context, commandID uuid.UUID) ([]MMLParamRef, error) {
	lang := string(appcontext.GetLocale(ctx))
	// UNIQUE(command_id, standard_path_id)（migration 000113）保证同命令下
	// 每个 standardPath 只有一行，无需 ROW_NUMBER 去重。
	query := `SELECT ` + paramRefSelectExpr(lang) + `
FROM mml_command_sub_fields csf
JOIN standard_params sp ON sp.id = csf.standard_path_id
WHERE csf.command_id = $1
ORDER BY csf.sort_order ASC, csf.mml_code ASC`
	rows, err := r.pool.Query(ctx, query, commandID)
	if err != nil {
		return nil, fmt.Errorf("list params by command: %w", err)
	}
	defer rows.Close()

	var result []MMLParamRef
	for rows.Next() {
		var pr MMLParamRef
		var constraintJSON []byte
		if err := rows.Scan(&pr.ID, &pr.ParamCode, &pr.ParamNameZh, &pr.Tr069Path,
			&pr.ValueType, &pr.IsWritable, &pr.IsRequired, &pr.DefaultValue, &pr.JsRegex, &constraintJSON); err != nil {
			return nil, fmt.Errorf("scan param ref: %w", err)
		}
		if constraintJSON != nil && len(constraintJSON) > 2 {
			_ = json.Unmarshal(constraintJSON, &pr.ValueConstraint)
		}
		result = append(result, pr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate param refs: %w", err)
	}
	return result, nil
}

func (r *PgCommandParamRepository) ListByCommandIDs(ctx context.Context, commandIDs []uuid.UUID) (map[uuid.UUID][]MMLParamRef, error) {
	result := make(map[uuid.UUID][]MMLParamRef, len(commandIDs))
	if len(commandIDs) == 0 {
		return result, nil
	}

	lang := string(appcontext.GetLocale(ctx))
	query := `SELECT csf.command_id, ` + paramRefSelectExpr(lang) + `
FROM mml_command_sub_fields csf
JOIN standard_params sp ON sp.id = csf.standard_path_id
WHERE csf.command_id = ANY($1)
ORDER BY csf.command_id, csf.sort_order ASC, csf.mml_code ASC`

	rows, err := r.pool.Query(ctx, query, commandIDs)
	if err != nil {
		return nil, fmt.Errorf("list params by command IDs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cmdID uuid.UUID
		var pr MMLParamRef
		var constraintJSON []byte
		if err := rows.Scan(&cmdID, &pr.ID, &pr.ParamCode, &pr.ParamNameZh, &pr.Tr069Path,
			&pr.ValueType, &pr.IsWritable, &pr.IsRequired, &pr.DefaultValue, &pr.JsRegex, &constraintJSON); err != nil {
			return nil, fmt.Errorf("scan param ref: %w", err)
		}
		if constraintJSON != nil && len(constraintJSON) > 2 {
			_ = json.Unmarshal(constraintJSON, &pr.ValueConstraint)
		}
		result[cmdID] = append(result[cmdID], pr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate param refs: %w", err)
	}
	return result, nil
}

// PgScriptValidationRepository is the PostgreSQL implementation used by TXT
// import validation. It deliberately performs set-based reads only: command
// codes, command parameters and device serial numbers are each queried as a
// batch rather than once per parsed line.
type PgScriptValidationRepository struct {
	pool *pgxpool.Pool
}

func NewPgScriptValidationRepository(pool *pgxpool.Pool) *PgScriptValidationRepository {
	return &PgScriptValidationRepository{pool: pool}
}

var _ ScriptValidationRepository = (*PgScriptValidationRepository)(nil)

func (r *PgScriptValidationRepository) LoadCommandsByCodes(ctx context.Context, codes []string, actor ValidationActor) (map[string]ValidationCommand, error) {
	result := make(map[string]ValidationCommand, len(codes))
	if len(codes) == 0 {
		return result, nil
	}

	standardRows, err := r.pool.Query(ctx, `
SELECT id, command_code, operation_type, rpc_method, COALESCE(target_object, ''), target_paths, require_confirm,
       (deprecated_at IS NOT NULL)
  FROM mml_commands
 WHERE command_code = ANY($1)
   AND source = 'standard'`, codes)
	if err != nil {
		return nil, fmt.Errorf("load standard validation commands: %w", err)
	}
	standardIDs := make([]uuid.UUID, 0, len(codes))
	standardCodes := make(map[uuid.UUID]string, len(codes))
	standardCodeSet := make(map[string]struct{}, len(codes))
	for standardRows.Next() {
		var id uuid.UUID
		var command ValidationCommand
		var targetPathsJSON []byte
		if err := standardRows.Scan(&id, &command.CommandCode, &command.OperationType, &command.RPCMethod, &command.TargetObject, &targetPathsJSON, &command.RequireConfirm, &command.Disabled); err != nil {
			standardRows.Close()
			return nil, fmt.Errorf("scan standard validation command: %w", err)
		}
		if err := json.Unmarshal(targetPathsJSON, &command.TargetPaths); err != nil {
			standardRows.Close()
			return nil, fmt.Errorf("decode standard validation target paths: %w", err)
		}
		result[command.CommandCode] = command
		standardIDs = append(standardIDs, id)
		standardCodes[id] = command.CommandCode
		standardCodeSet[command.CommandCode] = struct{}{}
	}
	if err := standardRows.Err(); err != nil {
		standardRows.Close()
		return nil, fmt.Errorf("iterate standard validation commands: %w", err)
	}
	standardRows.Close()

	// Standard commands take precedence over a custom command with the same
	// code. The custom predicate matches the existing public/self/group-share
	// visibility rule, so import cannot validate an otherwise hidden command.
	customRows, err := r.pool.Query(ctx, `
SELECT id, command_code, operation_type, parameters, param_paths
  FROM mml_custom_command cc
 WHERE command_code = ANY($1)
   AND (
       cc.command_scope = 'public'
       OR cc.creator = $2
       OR EXISTS (
           SELECT 1
             FROM users u
             JOIN user_roles ur ON ur.user_id = u.id
             JOIN role_device_groups rdg ON rdg.role_id = ur.role_id
            WHERE u.username = cc.creator
              AND rdg.group_id = ANY($3)
       )
   )`, codes, actor.Username, actor.VisibleGroupIDs)
	if err != nil {
		return nil, fmt.Errorf("load visible custom validation commands: %w", err)
	}
	for customRows.Next() {
		var id uuid.UUID
		var code, operation string
		var parametersJSON, pathsJSON []byte
		if err := customRows.Scan(&id, &code, &operation, &parametersJSON, &pathsJSON); err != nil {
			customRows.Close()
			return nil, fmt.Errorf("scan custom validation command: %w", err)
		}
		if _, standard := standardCodeSet[code]; standard {
			continue
		}
		if existing, duplicate := result[code]; duplicate {
			existing.Ambiguous = true
			result[code] = existing
			continue
		}
		command := ValidationCommand{CommandCode: code, OperationType: operation, RPCMethod: rpcMethodForOperation(operation)}
		var parameters map[string]interface{}
		if err := json.Unmarshal(parametersJSON, &parameters); err != nil {
			customRows.Close()
			return nil, fmt.Errorf("decode custom validation parameters: %w", err)
		}
		paramCodes := make([]string, 0, len(parameters))
		for paramCode := range parameters {
			paramCodes = append(paramCodes, paramCode)
		}
		sort.Strings(paramCodes)
		for _, paramCode := range paramCodes {
			command.ParamRefs = append(command.ParamRefs, MMLParamRef{ParamCode: paramCode, ValueType: "string", IsWritable: true})
		}
		if err := json.Unmarshal(pathsJSON, &command.TargetPaths); err != nil {
			customRows.Close()
			return nil, fmt.Errorf("decode custom validation paths: %w", err)
		}
		result[code] = command
	}
	if err := customRows.Err(); err != nil {
		customRows.Close()
		return nil, fmt.Errorf("iterate custom validation commands: %w", err)
	}
	customRows.Close()

	if len(standardIDs) == 0 {
		return result, nil
	}
	paramRows, err := r.pool.Query(ctx, `
SELECT csf.command_id, csf.mml_code, sp.standard_path,
       lower(COALESCE(sp.data_type, 'string')),
	       (sp.access = 'READ_WRITE'), csf.is_required, sp.min_value, sp.max_value
  FROM mml_command_sub_fields csf
  JOIN standard_params sp ON sp.id = csf.standard_path_id
 WHERE csf.command_id = ANY($1)
 ORDER BY csf.command_id, csf.sort_order ASC, csf.mml_code ASC`, standardIDs)
	if err != nil {
		return nil, fmt.Errorf("load validation command parameters: %w", err)
	}
	for paramRows.Next() {
		var commandID uuid.UUID
		var ref MMLParamRef
		var minValue, maxValue *int64
		if err := paramRows.Scan(&commandID, &ref.ParamCode, &ref.Tr069Path, &ref.ValueType, &ref.IsWritable, &ref.IsRequired, &minValue, &maxValue); err != nil {
			paramRows.Close()
			return nil, fmt.Errorf("scan validation command parameter: %w", err)
		}
		rules := standardValidationRules(ref.ValueType, minValue, maxValue)
		ref.JsRegex = rules.JsRegex
		ref.ValueConstraint = rules.ValueConstraint
		code := standardCodes[commandID]
		command := result[code]
		command.ParamRefs = append(command.ParamRefs, ref)
		result[code] = command
	}
	if err := paramRows.Err(); err != nil {
		paramRows.Close()
		return nil, fmt.Errorf("iterate validation command parameters: %w", err)
	}
	paramRows.Close()
	return result, nil
}

// standardValidationRules maps the constraints that the current authoritative
// standard_params schema actually persists (data type plus numeric bounds) to
// the generic validation contract. Boolean and unsignedInt semantics supply
// deterministic enum/regex rules in addition to persisted bounds.
func standardValidationRules(valueType string, minValue, maxValue *int64) MMLParamRef {
	rules := MMLParamRef{ValueConstraint: make(map[string]interface{})}
	if minValue != nil {
		rules.ValueConstraint["min"] = float64(*minValue)
	}
	if maxValue != nil {
		rules.ValueConstraint["max"] = float64(*maxValue)
	}
	switch canonicalValueType(valueType) {
	case "boolean", "bool":
		rules.JsRegex = "(?i)^(true|false|0|1)$"
		rules.ValueConstraint["regex"] = rules.JsRegex
		rules.ValueConstraint["enum"] = []interface{}{"true", "false", "0", "1"}
	case "unsignedint", "unsignedinteger", "uint", "uint32", "uint64":
		rules.JsRegex = "^[0-9]+$"
		rules.ValueConstraint["regex"] = rules.JsRegex
	}
	if len(rules.ValueConstraint) == 0 {
		rules.ValueConstraint = nil
	}
	return rules
}

func (r *PgScriptValidationRepository) LoadDevicesBySNs(ctx context.Context, sns []string) (map[string]*model.Device, error) {
	result := make(map[string]*model.Device, len(sns))
	if len(sns) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(ctx, `
SELECT serial_number, COALESCE(product_class, ''), is_online, COALESCE(firmware_version, '')
  FROM devices
 WHERE serial_number = ANY($1)
   AND deleted_at IS NULL`, sns)
	if err != nil {
		return nil, fmt.Errorf("load validation devices: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		device := &model.Device{}
		if err := rows.Scan(&device.SerialNumber, &device.ProductClass, &device.IsOnline, &device.FirmwareVersion); err != nil {
			return nil, fmt.Errorf("scan validation device: %w", err)
		}
		result[device.SerialNumber] = device
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate validation devices: %w", err)
	}
	return result, nil
}

func (r *PgScriptValidationRepository) LoadStandardPathSupport(ctx context.Context, lookups []StandardPathLookup) (map[string]bool, error) {
	result := make(map[string]bool, len(lookups))
	if len(lookups) == 0 {
		return result, nil
	}
	exactCandidates := make([]string, 0, len(lookups))
	prefixCandidates := make([]string, 0, len(lookups))
	for _, lookup := range lookups {
		result[lookup.key()] = false
		exactCandidates = append(exactCandidates, standardPathExactCandidates(lookup)...)
		prefixCandidates = append(prefixCandidates, standardPathPrefixCandidates(lookup)...)
	}
	exactCandidates = uniqueStrings(exactCandidates)
	prefixCandidates = uniqueStrings(prefixCandidates)

	exactMatches := make(map[string]struct{}, len(exactCandidates))
	if len(exactCandidates) > 0 {
		rows, err := r.pool.Query(ctx, `
SELECT standard_path
  FROM standard_params
 WHERE standard_path = ANY($1)`, exactCandidates)
		if err != nil {
			return nil, fmt.Errorf("load standard path exact matches: %w", err)
		}
		for rows.Next() {
			var path string
			if err := rows.Scan(&path); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan standard path exact match: %w", err)
			}
			exactMatches[path] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterate standard path exact matches: %w", err)
		}
		rows.Close()
	}

	prefixMatches := make(map[string]struct{}, len(prefixCandidates))
	if len(prefixCandidates) > 0 {
		rows, err := r.pool.Query(ctx, `
SELECT p.prefix
  FROM unnest($1::text[]) AS p(prefix)
 WHERE EXISTS (
       SELECT 1
         FROM standard_params sp
        WHERE left(sp.standard_path, length(p.prefix)) = p.prefix
 )`, prefixCandidates)
		if err != nil {
			return nil, fmt.Errorf("load standard path prefix matches: %w", err)
		}
		for rows.Next() {
			var prefix string
			if err := rows.Scan(&prefix); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan standard path prefix match: %w", err)
			}
			prefixMatches[prefix] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterate standard path prefix matches: %w", err)
		}
		rows.Close()
	}

	for _, lookup := range lookups {
		for _, candidate := range standardPathExactCandidates(lookup) {
			if _, ok := exactMatches[candidate]; ok {
				result[lookup.key()] = true
				break
			}
		}
		if result[lookup.key()] {
			continue
		}
		for _, prefix := range standardPathPrefixCandidates(lookup) {
			if _, ok := prefixMatches[prefix]; ok {
				result[lookup.key()] = true
				break
			}
		}
	}
	return result, nil
}

func rpcMethodForOperation(operation string) string {
	switch operation {
	case "MOD":
		return "SetParameterValues"
	case "ADD":
		return "AddObject"
	case "RMV":
		return "DeleteObject"
	default:
		return "GetParameterValues"
	}
}
