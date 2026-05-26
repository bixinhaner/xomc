package parammodel

import (
	"context"
	"errors"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// 本文件提供 P3-02 handler 用的 CRUD/查询方法（独立于 Registry-only Repository 接口），
// 避免破坏 Registry 抽象。所有方法挂在 PgRepository 上。

// ── ParamModel ──────────────────────────────────────────────────────

// ListParamModels 列出全部 param_models（含统计字段）。
func (r *PgRepository) ListParamModels(ctx context.Context) ([]ParamModel, error) {
	const q = `SELECT id, name, total_entries, total_objects, total_params,
	                 COALESCE(description,''), is_active, COALESCE(loaded_from,'')
	          FROM param_models ORDER BY name ASC`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query param_models: %w", err)
	}
	defer rows.Close()
	var out []ParamModel
	for rows.Next() {
		var m ParamModel
		if err := rows.Scan(&m.ID, &m.Name, &m.TotalEntries, &m.TotalObjects, &m.TotalParams,
			&m.Description, &m.IsActive, &m.LoadedFrom); err != nil {
			return nil, fmt.Errorf("scan param_model: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetParamModelByName 单条详情；不存在 → (nil, ErrNoParamModel)。
func (r *PgRepository) GetParamModelByName(ctx context.Context, name string) (*ParamModel, error) {
	const q = `SELECT id, name, total_entries, total_objects, total_params,
	                 COALESCE(description,''), is_active, COALESCE(loaded_from,'')
	          FROM param_models WHERE name = $1`
	var m ParamModel
	if err := r.pool.QueryRow(ctx, q, name).Scan(
		&m.ID, &m.Name, &m.TotalEntries, &m.TotalObjects, &m.TotalParams,
		&m.Description, &m.IsActive, &m.LoadedFrom,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoParamModel
		}
		return nil, fmt.Errorf("get param_model %q: %w", name, err)
	}
	return &m, nil
}

// GetParamModelByID 按主键查 param_model（F05 MR dispatcher 用于
// productClass → product.ParamModelID → param_models.name 链路）。
func (r *PgRepository) GetParamModelByID(ctx context.Context, id uuid.UUID) (*ParamModel, error) {
	const q = `SELECT id, name, total_entries, total_objects, total_params,
	                 COALESCE(description,''), is_active, COALESCE(loaded_from,'')
	          FROM param_models WHERE id = $1`
	var m ParamModel
	if err := r.pool.QueryRow(ctx, q, id).Scan(
		&m.ID, &m.Name, &m.TotalEntries, &m.TotalObjects, &m.TotalParams,
		&m.Description, &m.IsActive, &m.LoadedFrom,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoParamModel
		}
		return nil, fmt.Errorf("get param_model id=%s: %w", id, err)
	}
	return &m, nil
}

// UpdateParamModelMeta 仅更新元信息（description / is_active），不动统计字段。
func (r *PgRepository) UpdateParamModelMeta(ctx context.Context, name string, description *string, isActive *bool) (*ParamModel, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	ub := psql.Update("param_models").Where(sq.Eq{"name": name})
	dirty := false
	if description != nil {
		ub = ub.Set("description", *description)
		dirty = true
	}
	if isActive != nil {
		ub = ub.Set("is_active", *isActive)
		dirty = true
	}
	if !dirty {
		return r.GetParamModelByName(ctx, name)
	}
	uSQL, uArgs, err := ub.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update param_model sql: %w", err)
	}
	tag, err := r.pool.Exec(ctx, uSQL, uArgs...)
	if err != nil {
		return nil, fmt.Errorf("update param_model %q: %w", name, err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNoParamModel
	}
	return r.GetParamModelByName(ctx, name)
}

// DeleteParamModel 删除 param_model 及其全部 mappings（CASCADE）。
func (r *PgRepository) DeleteParamModel(ctx context.Context, name string) (bool, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM param_models WHERE name = $1`, name)
	if err != nil {
		return false, fmt.Errorf("delete param_model %q: %w", name, err)
	}
	return tag.RowsAffected() > 0, nil
}

// ── Mapping CRUD ────────────────────────────────────────────────────

// CreateMappingInput 是 CreateMapping 的入参。
type CreateMappingInput struct {
	StandardPath  string
	PrivatePath   string
	EntryType     string // object | parameter
	Access        string
	DataType      string
	ChangeApplies string
	MinValue      *int64
	MaxValue      *int64
	IsStorable    bool
}

// CreateMapping 在 paramModel 下新增一条默认映射。
func (r *PgRepository) CreateMapping(ctx context.Context, paramModelID uuid.UUID, in CreateMappingInput) (*ParamMapping, error) {
	if strings.TrimSpace(in.StandardPath) == "" || strings.TrimSpace(in.PrivatePath) == "" {
		return nil, fmt.Errorf("standard_path / private_path required")
	}
	if in.EntryType != "object" && in.EntryType != "parameter" {
		return nil, fmt.Errorf("entry_type must be object|parameter, got %q", in.EntryType)
	}
	const insertSQL = `
INSERT INTO param_mappings (
    param_model_id, standard_path, private_path, entry_type,
    access, data_type, change_applies, min_value, max_value, is_storable, is_active
) VALUES ($1, $2, $3, $4, NULLIF($5,''), NULLIF($6,''), NULLIF($7,''), $8, $9, $10, TRUE)
RETURNING id`
	var id uuid.UUID
	if err := r.pool.QueryRow(ctx, insertSQL,
		paramModelID, in.StandardPath, in.PrivatePath, in.EntryType,
		in.Access, in.DataType, in.ChangeApplies, in.MinValue, in.MaxValue, in.IsStorable,
	).Scan(&id); err != nil {
		return nil, fmt.Errorf("insert mapping: %w", err)
	}
	return r.getMappingByID(ctx, id)
}

// UpdateMappingInput 是 UpdateMapping 的入参；nil 字段表示不变。
type UpdateMappingInput struct {
	PrivatePath   *string
	Access        *string
	DataType      *string
	ChangeApplies *string
	MinValue      *int64
	MaxValue      *int64
	IsStorable    *bool
	IsActive      *bool
}

// UpdateMapping 局部更新映射；不允许改 standard_path / param_model_id（迁移用 CRUD）。
func (r *PgRepository) UpdateMapping(ctx context.Context, mappingID uuid.UUID, in UpdateMappingInput) (*ParamMapping, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	ub := psql.Update("param_mappings").Where(sq.Eq{"id": mappingID})
	dirty := false
	if in.PrivatePath != nil {
		ub = ub.Set("private_path", *in.PrivatePath)
		dirty = true
	}
	if in.Access != nil {
		ub = ub.Set("access", nilIfEmpty(*in.Access))
		dirty = true
	}
	if in.DataType != nil {
		ub = ub.Set("data_type", nilIfEmpty(*in.DataType))
		dirty = true
	}
	if in.ChangeApplies != nil {
		ub = ub.Set("change_applies", nilIfEmpty(*in.ChangeApplies))
		dirty = true
	}
	if in.MinValue != nil {
		ub = ub.Set("min_value", *in.MinValue)
		dirty = true
	}
	if in.MaxValue != nil {
		ub = ub.Set("max_value", *in.MaxValue)
		dirty = true
	}
	if in.IsStorable != nil {
		ub = ub.Set("is_storable", *in.IsStorable)
		dirty = true
	}
	if in.IsActive != nil {
		ub = ub.Set("is_active", *in.IsActive)
		dirty = true
	}
	if !dirty {
		return r.getMappingByID(ctx, mappingID)
	}
	uSQL, uArgs, err := ub.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update mapping sql: %w", err)
	}
	tag, err := r.pool.Exec(ctx, uSQL, uArgs...)
	if err != nil {
		return nil, fmt.Errorf("update mapping %s: %w", mappingID, err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNoMapping
	}
	return r.getMappingByID(ctx, mappingID)
}

// DeleteMapping 删除单条 mapping。
func (r *PgRepository) DeleteMapping(ctx context.Context, mappingID uuid.UUID) (bool, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM param_mappings WHERE id = $1`, mappingID)
	if err != nil {
		return false, fmt.Errorf("delete mapping %s: %w", mappingID, err)
	}
	return tag.RowsAffected() > 0, nil
}

// ListDiscoveredVersions 列出某产品已有的 swVersion 集合（去重）。
func (r *PgRepository) ListDiscoveredVersions(ctx context.Context, productID uuid.UUID) ([]string, error) {
	const q = `SELECT DISTINCT software_version FROM discovered_param_mappings
	          WHERE product_id = $1 ORDER BY software_version`
	rows, err := r.pool.Query(ctx, q, productID)
	if err != nil {
		return nil, fmt.Errorf("query discovered versions: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan version: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// DeleteDiscoveredVersion 清除某产品某 swVersion 的全部交集映射；返回影响行数。
func (r *PgRepository) DeleteDiscoveredVersion(ctx context.Context, productID uuid.UUID, swVersion string) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM discovered_param_mappings WHERE product_id = $1 AND software_version = $2`,
		productID, swVersion)
	if err != nil {
		return 0, fmt.Errorf("delete discovered %s/%s: %w", productID, swVersion, err)
	}
	return tag.RowsAffected(), nil
}

// DeleteDiscoveredAll 清空某产品全部 swVersion 的交集（设计 §4.4 reset 端点支撑）。
func (r *PgRepository) DeleteDiscoveredAll(ctx context.Context, productID uuid.UUID) (int64, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM discovered_param_mappings WHERE product_id = $1`, productID)
	if err != nil {
		return 0, fmt.Errorf("delete all discovered for %s: %w", productID, err)
	}
	return tag.RowsAffected(), nil
}

// ── Standard params CRUD ────────────────────────────────────────────

// ListStandardParams 列出全部 standard_params；keyword 在 standard_path 上做 ILIKE。
func (r *PgRepository) ListStandardParams(ctx context.Context, keyword string, entryType string) ([]StandardParam, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	qb := psql.Select("standard_path", "entry_type",
		"COALESCE(access,'')", "COALESCE(data_type,'')", "COALESCE(change_applies,'')",
		"min_value", "max_value").
		From("standard_params").
		OrderBy("standard_path ASC").
		Limit(2000)
	if strings.TrimSpace(keyword) != "" {
		qb = qb.Where(sq.ILike{"standard_path": "%" + strings.TrimSpace(keyword) + "%"})
	}
	if entryType == "object" || entryType == "parameter" {
		qb = qb.Where(sq.Eq{"entry_type": entryType})
	}
	sqlStr, args, err := qb.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list standard sql: %w", err)
	}
	rows, err := r.pool.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("query standard_params: %w", err)
	}
	defer rows.Close()
	return scanStandardParams(rows)
}

// GetStandardParam 按 standard_path 取单条。
func (r *PgRepository) GetStandardParam(ctx context.Context, standardPath string) (*StandardParam, error) {
	const q = `SELECT standard_path, entry_type,
	                 COALESCE(access,''), COALESCE(data_type,''), COALESCE(change_applies,''),
	                 min_value, max_value
	          FROM standard_params WHERE standard_path = $1`
	row := r.pool.QueryRow(ctx, q, standardPath)
	var sp StandardParam
	if err := row.Scan(&sp.StandardPath, &sp.EntryType, &sp.Access, &sp.DataType, &sp.ChangeApplies,
		&sp.MinValue, &sp.MaxValue); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("standard_path %q not found", standardPath)
		}
		return nil, fmt.Errorf("get standard_param: %w", err)
	}
	return &sp, nil
}

// UpsertStandardParamInput 同时承载 Create 与 Update（按 standard_path UPSERT）。
type UpsertStandardParamInput struct {
	StandardPath  string
	EntryType     string
	Access        string
	DataType      string
	ChangeApplies string
	MinValue      *int64
	MaxValue      *int64
}

// UpsertStandardParam 新建或更新单条 standard_param。
func (r *PgRepository) UpsertStandardParam(ctx context.Context, in UpsertStandardParamInput) (*StandardParam, error) {
	if strings.TrimSpace(in.StandardPath) == "" {
		return nil, fmt.Errorf("standard_path required")
	}
	if in.EntryType != "object" && in.EntryType != "parameter" {
		return nil, fmt.Errorf("entry_type must be object|parameter")
	}
	const upsertSQL = `
INSERT INTO standard_params (
    standard_path, entry_type, access, data_type, change_applies, min_value, max_value
) VALUES ($1, $2, NULLIF($3,''), NULLIF($4,''), NULLIF($5,''), $6, $7)
ON CONFLICT (standard_path) DO UPDATE
SET entry_type     = EXCLUDED.entry_type,
    access         = EXCLUDED.access,
    data_type      = EXCLUDED.data_type,
    change_applies = EXCLUDED.change_applies,
    min_value      = EXCLUDED.min_value,
    max_value      = EXCLUDED.max_value`
	if _, err := r.pool.Exec(ctx, upsertSQL,
		in.StandardPath, in.EntryType, in.Access, in.DataType, in.ChangeApplies,
		in.MinValue, in.MaxValue,
	); err != nil {
		return nil, fmt.Errorf("upsert standard_param: %w", err)
	}
	return r.GetStandardParam(ctx, in.StandardPath)
}

// DeleteStandardParam 删除 standard_path；被 param_mappings 引用时拒绝（应用层校验）。
func (r *PgRepository) DeleteStandardParam(ctx context.Context, standardPath string) (bool, error) {
	// 检查引用
	var refCount int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM param_mappings WHERE standard_path = $1`, standardPath,
	).Scan(&refCount); err != nil {
		return false, fmt.Errorf("count standard_param refs: %w", err)
	}
	if refCount > 0 {
		return false, fmt.Errorf("standard_path %q is referenced by %d mappings", standardPath, refCount)
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM standard_params WHERE standard_path = $1`, standardPath)
	if err != nil {
		return false, fmt.Errorf("delete standard_param: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

// ── helpers ─────────────────────────────────────────────────────────

func (r *PgRepository) getMappingByID(ctx context.Context, id uuid.UUID) (*ParamMapping, error) {
	const q = `SELECT id, param_model_id, standard_path, private_path, entry_type,
	                 access, data_type, change_applies, min_value, max_value,
	                 is_storable, is_active, is_supported
	          FROM param_mappings WHERE id = $1`
	row := r.pool.QueryRow(ctx, q, id)
	var (
		m                               ParamMapping
		access, dataType, changeApplies *string
	)
	if err := row.Scan(&m.ID, &m.ParamModelID, &m.StandardPath, &m.PrivatePath, &m.EntryType,
		&access, &dataType, &changeApplies, &m.MinValue, &m.MaxValue,
		&m.IsStorable, &m.IsActive, &m.IsSupported); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNoMapping
		}
		return nil, fmt.Errorf("get mapping %s: %w", id, err)
	}
	m.Access = strDeref(access)
	m.DataType = strDeref(dataType)
	m.ChangeApplies = strDeref(changeApplies)
	return &m, nil
}

func scanStandardParams(rows pgx.Rows) ([]StandardParam, error) {
	var out []StandardParam
	for rows.Next() {
		var sp StandardParam
		if err := rows.Scan(&sp.StandardPath, &sp.EntryType, &sp.Access, &sp.DataType, &sp.ChangeApplies,
			&sp.MinValue, &sp.MaxValue); err != nil {
			return nil, fmt.Errorf("scan standard_param: %w", err)
		}
		out = append(out, sp)
	}
	return out, rows.Err()
}

// 确保导入的包都被使用（context/errors 在 helpers 中）
var _ = context.Background
