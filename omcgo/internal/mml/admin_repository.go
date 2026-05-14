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
// 5 interfaces (按资源类型分组):
//   - SubFieldRepository       — mml_command_sub_fields CRUD
//   - AdminGroupRepository     — mml_param_groups admin CRUD
//   - AdminCommandRepository   — mml_commands admin CRUD（GetByID 复用 PgCommandRepository）
//   - AdminParamRepository     — mml_params admin CRUD + List
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

// ---- 共享列表 ----

var paramAdminColumns = []string{
	"id", "param_code", "param_name_zh", "param_name_en",
	"tr069_path", "value_type", "value_constraint",
	"default_value", "js_regex", "is_writable", "is_leaf",
	"display_order", "param_version", "explanation_zh", "explanation_en",
	"title_zh", "title_en",
	// T-0123-P0 catalog 元数据
	"access_type", "is_object", "supports_add", "supports_delete",
	"change_applies", "constraint_text_i18n", "catalog_protected", "source",
	"name_i18n", "explanation_i18n",
}

var groupAdminColumns = []string{
	"id", "group_code", "group_name_zh", "group_name_en", "param_version",
	"path", "display_order",
	"name_i18n",
	// T-0123-P0 catalog 元数据
	"source", "catalog_protected",
	"created_at", "updated_at",
}

// AdminParamFilter 列查询参数字典时的过滤器。
type AdminParamFilter struct {
	Search       *string
	AccessType   *string
	IsObject     *bool
	Source       *string
	ParamVersion *string
	PageNum      int
	PageSize     int
}

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
	query, args, err := storage.Psql.Insert("mml_command_sub_fields").
		Columns("id", "command_id", "param_id", "mml_code", "label_i18n",
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
		"id", "command_id", "param_id", "mml_code", "label_i18n",
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
		"id", "command_id", "param_id", "mml_code", "label_i18n",
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

// ListEnrichedByCommand 返回 sub_fields JOIN mml_params 富集后的视图，
// 供 GET /mml/commands/:id/sub-fields API 直接 marshal 给前端 SubFieldChecklist /
// SubFieldInputList 渲染使用，无需前端再发查 param 元数据。
func (r *PgSubFieldRepository) ListEnrichedByCommand(ctx context.Context, commandID uuid.UUID) ([]MMLCommandSubFieldEnriched, error) {
	const sqlText = `
SELECT
    csf.id, csf.command_id, csf.param_id, csf.mml_code, csf.label_i18n,
    csf.default_selected, csf.is_required, csf.sort_order, csf.created_at, csf.updated_at,
    p.tr069_path, p.value_type,
    p.access_type, p.is_object, p.supports_add, p.supports_delete,
    p.change_applies, p.constraint_text_i18n,
    p.default_value, p.js_regex, p.name_i18n
FROM mml_command_sub_fields csf
JOIN mml_params p ON p.id = csf.param_id
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

func (r *PgSubFieldRepository) CountByParam(ctx context.Context, paramID uuid.UUID) (int64, error) {
	const sqlText = `SELECT COUNT(*) FROM mml_command_sub_fields WHERE param_id = $1`
	var n int64
	if err := r.pool.QueryRow(ctx, sqlText, paramID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count sub_fields by param: %w", err)
	}
	return n, nil
}

// ============================================================
// AdminGroupRepository (mml_param_groups)
// ============================================================

// AdminGroupRepository 提供 mml_param_groups 的 admin CRUD。
// Console 端读复用 ParamRepository.ListGroups；本接口承担写路径。
type AdminGroupRepository interface {
	Create(ctx context.Context, g *ParamGroup) error
	Update(ctx context.Context, g *ParamGroup) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*ParamGroup, error)
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

func (r *PgAdminGroupRepository) Create(ctx context.Context, g *ParamGroup) error {
	if g.ID == uuid.Nil {
		g.ID = uuid.New()
	}
	if g.Source == "" {
		g.Source = SourceAdmin
	}
	const sqlText = `
INSERT INTO mml_param_groups (
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

func (r *PgAdminGroupRepository) Update(ctx context.Context, g *ParamGroup) error {
	const sqlText = `
UPDATE mml_param_groups SET
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
	const sqlText = `DELETE FROM mml_param_groups WHERE id = $1`
	tag, err := r.pool.Exec(ctx, sqlText, id)
	if err != nil {
		return fmt.Errorf("delete group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrGroupNotFound
	}
	return nil
}

func (r *PgAdminGroupRepository) GetByID(ctx context.Context, id uuid.UUID) (*ParamGroup, error) {
	const sqlText = `
SELECT id, group_code, group_name_zh, group_name_en, param_version,
       display_order, source, catalog_protected
FROM mml_param_groups WHERE id = $1`
	g := &ParamGroup{}
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
// AdminParamRepository (mml_params)
// ============================================================

// AdminParamRepository 提供 mml_params 的 admin CRUD + List + sub_field 引用计数。
//
// is_writable 是 GENERATED STORED 派生列（migration 000095），INSERT/UPDATE 时
// **不能** 在列列表中包含 is_writable —— 由 access_type 派生。
type AdminParamRepository interface {
	Create(ctx context.Context, p *Param) error
	Update(ctx context.Context, p *Param) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*Param, error)
	List(ctx context.Context, f AdminParamFilter) ([]Param, int64, error)
}

// PgAdminParamRepository PostgreSQL 实现。
type PgAdminParamRepository struct {
	pool *pgxpool.Pool
}

// NewPgAdminParamRepository 构造 PgAdminParamRepository。
func NewPgAdminParamRepository(pool *pgxpool.Pool) *PgAdminParamRepository {
	return &PgAdminParamRepository{pool: pool}
}

var _ AdminParamRepository = (*PgAdminParamRepository)(nil)

func (r *PgAdminParamRepository) Create(ctx context.Context, p *Param) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	if p.Source == "" {
		p.Source = SourceAdmin
	}
	if p.AccessType == "" {
		p.AccessType = AccessTypeReadOnly
	}
	if p.ChangeApplies == "" {
		p.ChangeApplies = ChangeAppliesImmediate
	}
	constraint, err := marshalI18n(p.ConstraintTextI18n)
	if err != nil {
		return fmt.Errorf("marshal constraint_text_i18n: %w", err)
	}
	// NOTE: 不包含 is_writable（GENERATED 派生列，自动从 access_type 派生）。
	const sqlText = `
INSERT INTO mml_params (
    id, param_code, tr069_path, value_type, value_constraint,
    default_value, js_regex, param_version,
    access_type, is_object, supports_add, supports_delete,
    change_applies, constraint_text_i18n, catalog_protected, source
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`
	if _, err := r.pool.Exec(ctx, sqlText,
		p.ID, p.ParamCode, p.Tr069Path, p.ValueType, jsonOrEmpty(p.ValueConstraint),
		p.DefaultValue, p.JsRegex, p.ParamVersion,
		p.AccessType, p.IsObject, p.SupportsAdd, p.SupportsDelete,
		p.ChangeApplies, constraint, p.CatalogProtected, p.Source,
	); err != nil {
		return fmt.Errorf("insert param: %w", err)
	}
	return nil
}

func (r *PgAdminParamRepository) Update(ctx context.Context, p *Param) error {
	constraint, err := marshalI18n(p.ConstraintTextI18n)
	if err != nil {
		return fmt.Errorf("marshal constraint_text_i18n: %w", err)
	}
	// NOTE: 不包含 is_writable（GENERATED 派生列）。
	const sqlText = `
UPDATE mml_params SET
    value_constraint     = $2,
    default_value        = $3,
    js_regex             = $4,
    access_type          = $5,
    is_object            = $6,
    supports_add         = $7,
    supports_delete      = $8,
    change_applies       = $9,
    constraint_text_i18n = $10
WHERE id = $1`
	tag, err := r.pool.Exec(ctx, sqlText,
		p.ID, jsonOrEmpty(p.ValueConstraint),
		p.DefaultValue, p.JsRegex,
		p.AccessType, p.IsObject, p.SupportsAdd, p.SupportsDelete,
		p.ChangeApplies, constraint,
	)
	if err != nil {
		return fmt.Errorf("update param: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrParamNotFound
	}
	return nil
}

func (r *PgAdminParamRepository) Delete(ctx context.Context, id uuid.UUID) error {
	const sqlText = `DELETE FROM mml_params WHERE id = $1`
	tag, err := r.pool.Exec(ctx, sqlText, id)
	if err != nil {
		return fmt.Errorf("delete param: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrParamNotFound
	}
	return nil
}

func (r *PgAdminParamRepository) GetByID(ctx context.Context, id uuid.UUID) (*Param, error) {
	const sqlText = `
SELECT id, param_code, tr069_path, value_type, value_constraint,
       default_value, js_regex, is_writable, param_version,
       access_type, is_object, supports_add, supports_delete,
       change_applies, constraint_text_i18n, catalog_protected, source
FROM mml_params WHERE id = $1`
	p := &Param{}
	var constraint []byte
	if err := r.pool.QueryRow(ctx, sqlText, id).Scan(
		&p.ID, &p.ParamCode, &p.Tr069Path, &p.ValueType, &p.ValueConstraint,
		&p.DefaultValue, &p.JsRegex, &p.IsWritable, &p.ParamVersion,
		&p.AccessType, &p.IsObject, &p.SupportsAdd, &p.SupportsDelete,
		&p.ChangeApplies, &constraint, &p.CatalogProtected, &p.Source,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrParamNotFound
		}
		return nil, fmt.Errorf("scan param: %w", err)
	}
	if err := unmarshalI18n(constraint, &p.ConstraintTextI18n); err != nil {
		return nil, fmt.Errorf("unmarshal constraint_text_i18n: %w", err)
	}
	return p, nil
}

func (r *PgAdminParamRepository) List(ctx context.Context, f AdminParamFilter) ([]Param, int64, error) {
	base := storage.Psql.Select(
		"id", "param_code", "tr069_path", "value_type", "value_constraint",
		"default_value", "js_regex", "is_writable", "param_version",
		"access_type", "is_object", "supports_add", "supports_delete",
		"change_applies", "constraint_text_i18n", "catalog_protected", "source",
	).From("mml_params")

	countBase := storage.Psql.Select("COUNT(*)").From("mml_params")

	conds := sq.And{}
	if f.Search != nil && *f.Search != "" {
		pattern := "%" + *f.Search + "%"
		conds = append(conds, sq.Or{
			sq.ILike{"tr069_path": pattern},
			sq.ILike{"param_code": pattern},
		})
	}
	if f.AccessType != nil {
		conds = append(conds, sq.Eq{"access_type": *f.AccessType})
	}
	if f.IsObject != nil {
		conds = append(conds, sq.Eq{"is_object": *f.IsObject})
	}
	if f.Source != nil {
		conds = append(conds, sq.Eq{"source": *f.Source})
	}
	if f.ParamVersion != nil {
		conds = append(conds, sq.Eq{"param_version": *f.ParamVersion})
	}
	if len(conds) > 0 {
		base = base.Where(conds)
		countBase = countBase.Where(conds)
	}

	pageSize := f.PageSize
	if pageSize <= 0 {
		pageSize = 50
	}
	if pageSize > 500 {
		pageSize = 500
	}
	pageNum := f.PageNum
	if pageNum <= 0 {
		pageNum = 1
	}
	offset := uint64((pageNum - 1) * pageSize)

	listQuery, listArgs, err := base.
		OrderBy("tr069_path ASC").
		Offset(offset).
		Limit(uint64(pageSize)).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build list params SQL: %w", err)
	}
	countQuery, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count params SQL: %w", err)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count params: %w", err)
	}

	rows, err := r.pool.Query(ctx, listQuery, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query params: %w", err)
	}
	defer rows.Close()

	var out []Param
	for rows.Next() {
		p := Param{}
		var constraint []byte
		if err := rows.Scan(
			&p.ID, &p.ParamCode, &p.Tr069Path, &p.ValueType, &p.ValueConstraint,
			&p.DefaultValue, &p.JsRegex, &p.IsWritable, &p.ParamVersion,
			&p.AccessType, &p.IsObject, &p.SupportsAdd, &p.SupportsDelete,
			&p.ChangeApplies, &constraint, &p.CatalogProtected, &p.Source,
		); err != nil {
			return nil, 0, fmt.Errorf("scan param row: %w", err)
		}
		if err := unmarshalI18n(constraint, &p.ConstraintTextI18n); err != nil {
			return nil, 0, fmt.Errorf("unmarshal constraint_text_i18n: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate params rows: %w", err)
	}
	return out, total, nil
}

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
	// ErrParamNotFound 当 param 不存在或被并发删除时返回。
	ErrParamNotFound = errors.New("mml: param not found")

	// ErrCatalogProtected 当尝试修改/删除 catalog_protected=true 的 row 关键字段时返回。
	// service 层 catalog_protected 守护逻辑使用；handler 翻 HTTP 403。
	ErrCatalogProtected = errors.New("mml: entry is catalog_protected; modification denied")
	// ErrParamInUse 当尝试删除被 sub_fields 引用的 param 时返回；handler 翻 HTTP 409。
	ErrParamInUse = errors.New("mml: param is referenced by sub_fields")
	// ErrGroupNotEmpty 当尝试删除还含 commands 的 group 时返回；handler 翻 HTTP 409。
	ErrGroupNotEmpty = errors.New("mml: group has commands; cannot delete")
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
