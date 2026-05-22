package mml

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

// ============================================================
// admin_repository.go — T-0123-P0 catalog 管理 repo 层
//
// 3 interfaces (按资源类型分组):
//   - SubFieldRepository       — mml_command_sub_fields CRUD
//   - AdminGroupRepository     — mml_command_groups admin CRUD
//   - AdminCommandRepository   — mml_commands admin CRUD（GetByID 复用 PgCommandRepository）
//
// 设计依据：
//   - PRD `docs/design/mml-restore-old-interaction-plan-20260514.md` §M.5.2
//   - 接口 1-5 method（"Small Interfaces" 原则）
//   - 接受接口、返回结构体
//   - 错误用 fmt.Errorf("...: %w", err) 包装
//
// catalog_protected 守护逻辑在 service 层（admin_service.go）实施；
// 本层仅 raw CRUD，无业务校验。
// ============================================================

// ============================================================
// SubFieldRepository (mml_command_sub_fields)
// ============================================================

// SubFieldRepository 提供 mml_command_sub_fields 的 CRUD 操作。
// 写操作触发 trg_mml_sub_fields_target_paths（migration 000095），
// 自动回填所属 mml_commands.target_paths JSONB 数组。
type SubFieldRepository interface {
	Create(ctx context.Context, sf *MMLCommandSubField) error
	Update(ctx context.Context, sf *MMLCommandSubField) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*MMLCommandSubField, error)
	ListByCommand(ctx context.Context, commandID uuid.UUID) ([]MMLCommandSubField, error)
	ListEnrichedByCommand(ctx context.Context, commandID uuid.UUID) ([]MMLCommandSubFieldEnriched, error)
	CountByParam(ctx context.Context, paramID uuid.UUID) (int64, error)
}

// PgSubFieldRepository PostgreSQL 实现。
type PgSubFieldRepository struct {
	pool *pgxpool.Pool
}

// NewPgSubFieldRepository 构造 PgSubFieldRepository。
func NewPgSubFieldRepository(pool *pgxpool.Pool) *PgSubFieldRepository {
	return &PgSubFieldRepository{pool: pool}
}

var _ SubFieldRepository = (*PgSubFieldRepository)(nil)

func (r *PgSubFieldRepository) Create(ctx context.Context, sf *MMLCommandSubField) error {
	if sf.ID == uuid.Nil {
		sf.ID = uuid.New()
	}
	labelI18n, err := marshalI18n(sf.LabelI18n)
	if err != nil {
		return fmt.Errorf("marshal label_i18n: %w", err)
	}
	// migration 000113 后 sub_fields.param_id 改名 standard_path_id；
	// 保留结构体字段名 ParamID 做 soft alias，与 admin_repository:243 SELECT
	// AS 别名策略保持一致。
	query, args, err := storage.Psql.Insert("mml_command_sub_fields").
		Columns("id", "command_id", "standard_path_id", "mml_code", "label_i18n",
			"default_selected", "is_required", "sort_order").
		Values(sf.ID, sf.CommandID, sf.ParamID, sf.MMLCode, labelI18n,
			sf.DefaultSelected, sf.IsRequired, sf.SortOrder).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert sub_field SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert sub_field: %w", err)
	}
	return nil
}

func (r *PgSubFieldRepository) Update(ctx context.Context, sf *MMLCommandSubField) error {
	labelI18n, err := marshalI18n(sf.LabelI18n)
	if err != nil {
		return fmt.Errorf("marshal label_i18n: %w", err)
	}
	query, args, err := storage.Psql.Update("mml_command_sub_fields").
		Set("mml_code", sf.MMLCode).
		Set("label_i18n", labelI18n).
		Set("default_selected", sf.DefaultSelected).
		Set("is_required", sf.IsRequired).
		Set("sort_order", sf.SortOrder).
		Where(sq.Eq{"id": sf.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update sub_field SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update sub_field: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSubFieldNotFound
	}
	return nil
}

func (r *PgSubFieldRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Delete("mml_command_sub_fields").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete sub_field SQL: %w", err)
	}
	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete sub_field: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrSubFieldNotFound
	}
	return nil
}

func (r *PgSubFieldRepository) GetByID(ctx context.Context, id uuid.UUID) (*MMLCommandSubField, error) {
	query, args, err := storage.Psql.Select(
		"id", "command_id", "standard_path_id AS param_id", "mml_code", "label_i18n",
		"default_selected", "is_required", "sort_order", "created_at", "updated_at",
	).From("mml_command_sub_fields").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get sub_field SQL: %w", err)
	}
	row := r.pool.QueryRow(ctx, query, args...)
	sf := &MMLCommandSubField{}
	var labelI18n []byte
	if err := row.Scan(
		&sf.ID, &sf.CommandID, &sf.ParamID, &sf.MMLCode, &labelI18n,
		&sf.DefaultSelected, &sf.IsRequired, &sf.SortOrder, &sf.CreatedAt, &sf.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSubFieldNotFound
		}
		return nil, fmt.Errorf("scan sub_field: %w", err)
	}
	if err := unmarshalI18n(labelI18n, &sf.LabelI18n); err != nil {
		return nil, fmt.Errorf("unmarshal label_i18n: %w", err)
	}
	return sf, nil
}

func (r *PgSubFieldRepository) ListByCommand(ctx context.Context, commandID uuid.UUID) ([]MMLCommandSubField, error) {
	query, args, err := storage.Psql.Select(
		"id", "command_id", "standard_path_id AS param_id", "mml_code", "label_i18n",
		"default_selected", "is_required", "sort_order", "created_at", "updated_at",
	).From("mml_command_sub_fields").
		Where(sq.Eq{"command_id": commandID}).
		OrderBy("sort_order ASC", "mml_code ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list sub_fields SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query sub_fields: %w", err)
	}
	defer rows.Close()

	var out []MMLCommandSubField
	for rows.Next() {
		sf := MMLCommandSubField{}
		var labelI18n []byte
		if err := rows.Scan(
			&sf.ID, &sf.CommandID, &sf.ParamID, &sf.MMLCode, &labelI18n,
			&sf.DefaultSelected, &sf.IsRequired, &sf.SortOrder, &sf.CreatedAt, &sf.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan sub_field row: %w", err)
		}
		if err := unmarshalI18n(labelI18n, &sf.LabelI18n); err != nil {
			return nil, fmt.Errorf("unmarshal label_i18n: %w", err)
		}
		out = append(out, sf)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sub_fields rows: %w", err)
	}
	return out, nil
}

// ListEnrichedByCommand 返回 sub_fields JOIN standard_params 富集后的视图，
// 供 GET /mml/commands/:id/sub-fields API 直接 marshal 给前端 SubFieldChecklist /
// SubFieldInputList 渲染使用，无需前端再发查 param 元数据。
//
// migration 000113 之后 sub_fields.standard_path_id 直接挂 standard_params
// （系统级标准 path 字典）；MML 模块不再持有独立 path 字典。standard_params
// 字段更少（无 is_writable / supports_add / js_regex / constraint_text_i18n），
// 这里补默认值兜底，前端 SubFieldChecklist / SubFieldInputList 接口不变。
func (r *PgSubFieldRepository) ListEnrichedByCommand(ctx context.Context, commandID uuid.UUID) ([]MMLCommandSubFieldEnriched, error) {
	const sqlText = `
SELECT
    csf.id, csf.command_id, csf.standard_path_id AS param_id, csf.mml_code, csf.label_i18n,
    csf.default_selected, csf.is_required, csf.sort_order, csf.created_at, csf.updated_at,
    sp.standard_path                      AS tr069_path,
    COALESCE(sp.data_type, 'string')      AS value_type,
    COALESCE(sp.access, 'READ_ONLY')      AS access_type,
    (sp.entry_type = 'object')            AS is_object,
    false                                 AS supports_add,
    false                                 AS supports_delete,
    COALESCE(sp.change_applies, 'Immediate') AS change_applies,
    '{}'::jsonb                           AS constraint_text_i18n,
    NULL::text                            AS default_value,
    NULL::text                            AS js_regex,
    jsonb_build_object(
        'zh-CN', sp.standard_path,
        'en-US', sp.standard_path
    )                                     AS name_i18n
FROM mml_command_sub_fields csf
JOIN standard_params sp ON sp.id = csf.standard_path_id
WHERE csf.command_id = $1
ORDER BY csf.sort_order ASC, csf.mml_code ASC`

	rows, err := r.pool.Query(ctx, sqlText, commandID)
	if err != nil {
		return nil, fmt.Errorf("query enriched sub_fields: %w", err)
	}
	defer rows.Close()

	var out []MMLCommandSubFieldEnriched
	for rows.Next() {
		e := MMLCommandSubFieldEnriched{}
		var labelI18n, constraintI18n, paramNameI18n []byte
		if err := rows.Scan(
			&e.ID, &e.CommandID, &e.ParamID, &e.MMLCode, &labelI18n,
			&e.DefaultSelected, &e.IsRequired, &e.SortOrder, &e.CreatedAt, &e.UpdatedAt,
			&e.Tr069Path, &e.ValueType,
			&e.AccessType, &e.IsObject, &e.SupportsAdd, &e.SupportsDelete,
			&e.ChangeApplies, &constraintI18n,
			&e.DefaultValue, &e.JsRegex, &paramNameI18n,
		); err != nil {
			return nil, fmt.Errorf("scan enriched sub_field row: %w", err)
		}
		if err := unmarshalI18n(labelI18n, &e.LabelI18n); err != nil {
			return nil, fmt.Errorf("unmarshal label_i18n: %w", err)
		}
		if err := unmarshalI18n(constraintI18n, &e.ConstraintTextI18n); err != nil {
			return nil, fmt.Errorf("unmarshal constraint_text_i18n: %w", err)
		}
		if err := unmarshalI18n(paramNameI18n, &e.ParamNameI18n); err != nil {
			return nil, fmt.Errorf("unmarshal param name_i18n: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate enriched sub_fields rows: %w", err)
	}
	return out, nil
}

// CountByParam returns the number of sub_fields referencing a given standard_path.
// 入参 paramID 在 migration 000113 后语义切换为 standard_params.id（保留参数名做 soft alias）。
func (r *PgSubFieldRepository) CountByParam(ctx context.Context, paramID uuid.UUID) (int64, error) {
	const sqlText = `SELECT COUNT(*) FROM mml_command_sub_fields WHERE standard_path_id = $1`
	var n int64
	if err := r.pool.QueryRow(ctx, sqlText, paramID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count sub_fields by standard_path: %w", err)
	}
	return n, nil
}

// ============================================================
// AdminGroupRepository (mml_command_groups)
// ============================================================

// AdminGroupRepository 提供 mml_command_groups 的 admin CRUD。
// Console 端读复用 ParamRepository.ListGroups；本接口承担写路径。
type AdminGroupRepository interface {
	Create(ctx context.Context, g *CommandGroup) error
	Update(ctx context.Context, g *CommandGroup) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*CommandGroup, error)
	CountCommandsByGroup(ctx context.Context, groupID uuid.UUID) (int64, error)
}

// PgAdminGroupRepository PostgreSQL 实现。
type PgAdminGroupRepository struct {
	pool *pgxpool.Pool
}

// NewPgAdminGroupRepository 构造 PgAdminGroupRepository。
func NewPgAdminGroupRepository(pool *pgxpool.Pool) *PgAdminGroupRepository {
	return &PgAdminGroupRepository{pool: pool}
}

var _ AdminGroupRepository = (*PgAdminGroupRepository)(nil)

func (r *PgAdminGroupRepository) Create(ctx context.Context, g *CommandGroup) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	if g.Source == "" {
		g.Source = SourceAdmin
	}
	const sqlText = `
INSERT INTO mml_command_groups (
    id, group_code, group_name_zh, group_name_en, param_version,
    display_order, name_i18n, source, catalog_protected
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`
	nameI18n := jsonOrEmpty(map[string]string{"zh-CN": g.GroupNameZh, "en-US": g.GroupNameEn})
	if _, err := r.pool.Exec(ctx, sqlText,
		g.ID, g.GroupCode, g.GroupNameZh, g.GroupNameEn, g.ParamVersion,
		g.DisplayOrder, nameI18n, g.Source, g.CatalogProtected,
	); err != nil {
		return fmt.Errorf("insert group: %w", err)
	}
	return nil
}

func (r *PgAdminGroupRepository) Update(ctx context.Context, g *CommandGroup) error {
	const sqlText = `
UPDATE mml_command_groups SET
    group_name_zh = $2,
    group_name_en = $3,
    display_order = $4,
    name_i18n     = $5,
    updated_at    = NOW()
WHERE id = $1`
	nameI18n := jsonOrEmpty(map[string]string{"zh-CN": g.GroupNameZh, "en-US": g.GroupNameEn})
	tag, err := r.pool.Exec(ctx, sqlText,
		g.ID, g.GroupNameZh, g.GroupNameEn, g.DisplayOrder, nameI18n,
	)
	if err != nil {
		return fmt.Errorf("update group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrGroupNotFound
	}
	return nil
}

func (r *PgAdminGroupRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const sqlText = `DELETE FROM mml_command_groups WHERE id = $1`
	tag, err := r.pool.Exec(ctx, sqlText, id)
	if err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrGroupNotFound
	}
	return nil
}

func (r *PgAdminGroupRepository) GetByID(ctx context.Context, id uuid.UUID) (*CommandGroup, error) {
	const sqlText = `
SELECT id, group_code, group_name_zh, group_name_en, param_version,
       display_order, source, catalog_protected
FROM mml_command_groups WHERE id = $1`
	g := &CommandGroup{}
	if err := r.pool.QueryRow(ctx, sqlText, id).Scan(
		&g.ID, &g.GroupCode, &g.GroupNameZh, &g.GroupNameEn, &g.ParamVersion,
		&g.DisplayOrder, &g.Source, &g.CatalogProtected,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("scan group: %w", err)
	}
	return g, nil
}

func (r *PgAdminGroupRepository) CountCommandsByGroup(ctx context.Context, groupID uuid.UUID) (int64, error) {
	const sqlText = `SELECT COUNT(*) FROM mml_commands WHERE group_id = $1`
	var n int64
	if err := r.pool.QueryRow(ctx, sqlText, groupID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count commands by group: %w", err)
	}
	return n, nil
}

// ============================================================
// AdminCommandRepository (mml_commands write paths)
// ============================================================

// AdminCommandRepository 提供 mml_commands 的 admin 写操作。
// 读操作复用既有 PgCommandRepository.GetByID / ListByGroupID / Search。
type AdminCommandRepository interface {
	Create(ctx context.Context, c *MMLCommand) error
	Update(ctx context.Context, c *MMLCommand) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// PgAdminCommandRepository PostgreSQL 实现。
type PgAdminCommandRepository struct {
	pool *pgxpool.Pool
}

// NewPgAdminCommandRepository 构造 PgAdminCommandRepository。
func NewPgAdminCommandRepository(pool *pgxpool.Pool) *PgAdminCommandRepository {
	return &PgAdminCommandRepository{pool: pool}
}

var _ AdminCommandRepository = (*PgAdminCommandRepository)(nil)

func (r *PgAdminCommandRepository) Create(ctx context.Context, c *MMLCommand) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.Source == "" {
		c.Source = SourceAdmin
	}
	commandNameI18n, err := marshalI18n(c.CommandNameI18n)
	if err != nil {
		return fmt.Errorf("marshal command_name_i18n: %w", err)
	}
	confirmMsgI18n, err := marshalI18n(c.ConfirmMsgI18n)
	if err != nil {
		return fmt.Errorf("marshal confirm_msg_i18n: %w", err)
	}
	logicalNameI18n, err := marshalI18n(c.LogicalNameI18n)
	if err != nil {
		return fmt.Errorf("marshal logical_name_i18n: %w", err)
	}
	targetPaths, err := json.Marshal(c.TargetPaths)
	if err != nil {
		return fmt.Errorf("marshal target_paths: %w", err)
	}
	const sqlText = `
INSERT INTO mml_commands (
    id, command_name, command_code, category, description, rpc_method,
    operation_type, help_doc, notes,
    target_paths, target_object, group_id,
    command_name_i18n, require_confirm, confirm_msg_i18n,
    logical_code, logical_name_i18n, source, catalog_protected
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`
	if _, err := r.pool.Exec(ctx, sqlText,
		c.ID, c.CommandName, c.CommandCode, c.Category, c.Description, c.RPCMethod,
		c.OperationType, c.HelpDoc, c.Notes,
		targetPaths, nullIfEmpty(c.TargetObject), c.GroupID,
		commandNameI18n, c.RequireConfirm, confirmMsgI18n,
		nullIfEmpty(c.LogicalCode), logicalNameI18n, c.Source, c.CatalogProtected,
	); err != nil {
		return fmt.Errorf("insert command: %w", err)
	}
	return nil
}

func (r *PgAdminCommandRepository) Update(ctx context.Context, c *MMLCommand) error {
	commandNameI18n, err := marshalI18n(c.CommandNameI18n)
	if err != nil {
		return fmt.Errorf("marshal command_name_i18n: %w", err)
	}
	confirmMsgI18n, err := marshalI18n(c.ConfirmMsgI18n)
	if err != nil {
		return fmt.Errorf("marshal confirm_msg_i18n: %w", err)
	}
	logicalNameI18n, err := marshalI18n(c.LogicalNameI18n)
	if err != nil {
		return fmt.Errorf("marshal logical_name_i18n: %w", err)
	}
	const sqlText = `
UPDATE mml_commands SET
    command_name      = $2,
    category          = $3,
    description       = $4,
    help_doc          = $5,
    notes             = $6,
    target_object     = $7,
    group_id          = $8,
    command_name_i18n = $9,
    require_confirm   = $10,
    confirm_msg_i18n  = $11,
    logical_code      = $12,
    logical_name_i18n = $13
WHERE id = $1`
	tag, err := r.pool.Exec(ctx, sqlText,
		c.ID, c.CommandName, c.Category, c.Description, c.HelpDoc, c.Notes,
		nullIfEmpty(c.TargetObject), c.GroupID,
		commandNameI18n, c.RequireConfirm, confirmMsgI18n,
		nullIfEmpty(c.LogicalCode), logicalNameI18n,
	)
	if err != nil {
		return fmt.Errorf("update command: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCommandNotFound
	}
	return nil
}

func (r *PgAdminCommandRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const sqlText = `DELETE FROM mml_commands WHERE id = $1`
	tag, err := r.pool.Exec(ctx, sqlText, id)
	if err != nil {
		return fmt.Errorf("delete command: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCommandNotFound
	}
	return nil
}

// ============================================================
// AdminParamRepository (mml_params) — DELETED
//
// mml_params 表已下线（v2.3 catalog 单源化），所有 admin Param CRUD / XML 导入
// 路径已移除。sub_fields 通过 standard_path_id 引用 standard_params。
// ============================================================

// ============================================================
// 共享 helper（i18n / null / sentinel）
// ============================================================

func marshalI18n(m map[string]string) ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

func unmarshalI18n(b []byte, dst *map[string]string) error {
	if len(b) == 0 || string(b) == "null" {
		*dst = map[string]string{}
		return nil
	}
	return json.Unmarshal(b, dst)
}

func jsonOrEmpty(v any) []byte {
	if v == nil {
		return []byte("{}")
	}
	if s, ok := v.([]byte); ok && len(s) > 0 {
		return s
	}
	if m, ok := v.(map[string]string); ok && len(m) == 0 {
		return []byte("{}")
	}
	if j, ok := v.(json.RawMessage); ok {
		if len(j) == 0 {
			return []byte("{}")
		}
		return j
	}
	b, err := json.Marshal(v)
	if err != nil || len(b) == 0 {
		return []byte("{}")
	}
	return b
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ============================================================
// Sentinel errors (admin layer)
// ============================================================

var (
	// ErrSubFieldNotFound 当 sub_field 不存在或被并发删除时返回。
	ErrSubFieldNotFound = errors.New("mml: sub_field not found")
	// ErrGroupNotFound 当 group 不存在或被并发删除时返回。
	ErrGroupNotFound = errors.New("mml: group not found")
	// ErrCommandNotFound 当 command 不存在或被并发删除时返回。
	ErrCommandNotFound = errors.New("mml: command not found")

	// ErrCatalogProtected 当尝试修改/删除 catalog_protected=true 的 row 关键字段时返回。
	// service 层 catalog_protected 守护逻辑使用；handler 翻 HTTP 403。
	ErrCatalogProtected = errors.New("mml: entry is catalog_protected; modification denied")
	// ErrGroupNotEmpty 当尝试删除还含 commands 的 group 时返回；handler 翻 HTTP 409。
	ErrGroupNotEmpty = errors.New("mml: group has commands; cannot delete")

	// ErrFlatTreeNotConfigured 当 ConsoleService.flatTreeRepo 未装配但调用
	// BuildFlatGroupTree 时返回；handler 翻 HTTP 503。Task #4 扁平命令树
	// 依赖 PgFlatGroupTreeRepository，provider 未调 SetFlatTreeRepo 时触发。
	ErrFlatTreeNotConfigured = errors.New("mml: flat group tree repository not configured")
)

// ============================================================
// 字段值常量（与 migration 000095 CHECK 约束一致）
// ============================================================

const (
	// access_type 取值
	AccessTypeReadOnly  = "READ_ONLY"
	AccessTypeReadWrite = "READ_WRITE"
	AccessTypeWriteOnly = "WRITE_ONLY"

	// change_applies 取值（Go 层校验，DB 无 CHECK 约束 Q3=B 决议）
	ChangeAppliesImmediate = "Immediate"
	ChangeAppliesOnReboot  = "OnReboot"

	// source 取值
	SourceStandard = "standard"
	SourceAdmin    = "admin"
)
