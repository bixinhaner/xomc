package mml

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/omcgo/omcgo/internal/core/reliability"
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
	// ListEnrichedByCommand 返回命令的 sub_fields（JOIN standard_params 元数据）。
	//
	// T-0170: paramModelID 非 nil 时叠加 param_mappings 过滤 — 只返该 paramModel
	// 真实支持的 standardPath（缺映射的 sub_field 被排除）。admin 视图传 nil 返全集。
	ListEnrichedByCommand(ctx context.Context, commandID uuid.UUID, paramModelID *uuid.UUID) ([]MMLCommandSubFieldEnriched, error)
	CountByParam(ctx context.Context, paramID uuid.UUID) (int64, error)

	// MarkUnsupportedByStandardPath 把 (paramModelID, standardPath) 在 param_mappings
	// 中标 is_supported=false (auto-learn 入口)。T-0176-PR-E 起从写
	// mml_command_sub_fields 迁到写 param_mappings，paramModelID 必填——这是 MML
	// 控制台 "is_supported 单一真值源 = param_mappings" 重构的最后一块写路径。
	//
	// 调用方：ACS 收到 SetParameterValues 9005 (Invalid parameter name) 时
	// ResultAggregator 转发到此方法，让 UI 后续 enriched 视图按 paramModel 过滤掉
	// 该 path。0 行受影响是合法情况（path 不在 param_mappings 中，或已经标过 false）。
	//
	// 注意：方法名 / 接口归属保留——MarkUnsupportedByStandardPath 仍挂在
	// SubFieldRepository 接口上虽然语义已飘到 param_mappings，T-0176 计划"第一版就
	// 地修改，类型迁回 parammodel.Registry 是后续清理项"。
	MarkUnsupportedByStandardPath(ctx context.Context, paramModelID uuid.UUID, standardPath string) (int64, error)

	// ListAdminByCommand 是 admin 视角的 sub_field 列表 —— 与 ListEnrichedByCommand 区别：
	//   · admin 端要看见 is_supported=false 的 path（catalog 维护人员需要审视哪条 path 被 auto-learn 关掉）
	//   · 不做 paramModel 过滤（admin 视角与具体设备解耦）
	//   · JOIN standard_params 返完整元数据，便于 admin UI 一次性渲染
	ListAdminByCommand(ctx context.Context, commandID uuid.UUID) ([]MMLCommandSubFieldEnriched, error)

	// BatchCreate 在单次 SQL 中批量插入 N 条 sub_field。T-Mml-Admin"按 path 列表
	// 自动建命令字段"链路：前端选完多条 standard_params 后传 standard_path_id 数组，
	// 后端逐 path 派生 mml_code / label / sort_order 等默认值。
	//
	// 触发器复 cost：mml_command_sub_fields 上有 AFTER INSERT trigger 重算
	// mml_commands.target_paths（每行一次）。N 条 = N 次重算，对 N <= 50 量级
	// 可接受；如未来需要超大批量，单独 disable trigger + 单次 UPDATE。
	BatchCreate(ctx context.Context, items []*MMLCommandSubField) error
}

// PgSubFieldRepository PostgreSQL 实现。
type PgSubFieldRepository struct {
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewPgSubFieldRepository 构造 PgSubFieldRepository。
//
// logger 默认为 NopLogger，保持构造签名向后兼容（既有测试不传 logger）；
// 生产经 WithLogger 注入真实 logger 以观测事务回滚失败（MEDIUM-18）。
func NewPgSubFieldRepository(pool *pgxpool.Pool) *PgSubFieldRepository {
	return &PgSubFieldRepository{pool: pool, logger: zap.NewNop()}
}

// WithLogger 注入 logger 以记录事务回滚失败，返回自身便于链式调用。
func (r *PgSubFieldRepository) WithLogger(logger *zap.Logger) *PgSubFieldRepository {
	if logger != nil {
		r.logger = logger.Named("mml-subfield-repo")
	}
	return r
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
	// T-0176-PR-C: 不再读 mml_command_sub_fields.is_supported 决定可见性。
	// "该 path 是否被 paramModel 支持"的真值源已收敛到 param_mappings.is_supported；
	// sub_field 行始终返完整 catalog（PR-F 之后 DROP 该列）。
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

func buildEnumOptions(enumValues, enumLabels *string) []MMLParamEnumOption {
	if enumValues == nil {
		return nil
	}
	values := splitEnumCSV(*enumValues)
	if len(values) == 0 {
		return nil
	}
	var labels []string
	if enumLabels != nil {
		labels = splitEnumCSV(*enumLabels)
	}
	options := make([]MMLParamEnumOption, 0, len(values))
	for i, value := range values {
		label := value
		if i < len(labels) && labels[i] != "" {
			label = labels[i]
		}
		options = append(options, MMLParamEnumOption{Value: value, Label: label})
	}
	return options
}

func splitEnumCSV(value string) []string {
	parts := strings.Split(strings.ReplaceAll(value, "，", ","), ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// ListEnrichedByCommand 返回 sub_fields JOIN standard_params 富集后的视图，
// 供 GET /mml/commands/:id/sub-fields API 直接 marshal 给前端 SubFieldChecklist /
// SubFieldInputList 渲染使用，无需前端再发查 param 元数据。
//
// migration 000113 之后 sub_fields.standard_path_id 直接挂 standard_params
// （系统级标准 path 字典）；MML 模块不再持有独立 path 字典。standard_params
// 字段更少（无 is_writable / supports_add / js_regex / constraint_text_i18n），
// 这里补默认值兜底，前端 SubFieldChecklist / SubFieldInputList 接口不变。
//
// constraint_text_i18n 由 standard_params.min_value / max_value 派生（2026-05-23 改）：
//   - 双侧有范围 → "[min, max]"
//   - 仅有 min   → "≥ min"
//   - 仅有 max   → "≤ max"
//   - 都为 NULL → 空（前端 AccessTypeTag Tooltip 走"无明确取值范围"兜底文案）
//
// 数字内容 zh-CN / en-US 同形，i18n 两 key 同值即可。
func (r *PgSubFieldRepository) ListEnrichedByCommand(ctx context.Context, commandID uuid.UUID, paramModelID *uuid.UUID) ([]MMLCommandSubFieldEnriched, error) {
	// 2026-05-27 用户决策(撤销 T-0183): EXISTS 重新加上 `AND pm.is_supported = true`,
	// 让 is_supported=false 的 path 物理过滤掉(不再返回给前端)。
	// 与命令树命令名 (N) 计数口径对齐 — N = 该 paramModel 下 is_supported=true 的
	// path 数 = 右栏 path 列表行数。
	//
	// 历史路径:
	//   - T-0170/T-0176-PR-C: EXISTS 含 is_supported=true → 隐藏 unsupported(老行为)
	//   - T-0183: 取消 is_supported=true 过滤,前端"显示但默认不勾"
	//   - 本改: 回到 T-0170 行为,前端不再处理 unsupported 行
	//
	// paramModelID 为 nil (admin 视角)时:不叠加 param_mappings 过滤,is_supported
	// 列走 BOOL_OR 聚合,与原行为一致(admin 看的是字典全貌)。带 paramModelID 时，
	// 同一查询按模型读取 default/pattern/enum，并让模型 min/max 覆盖标准树范围。
	paramModelFilter := ""
	modelMappingJoin := ""
	defaultValueExpr := "NULL::text"
	validationPatternExpr := "NULL::text"
	minValueExpr := "sp.min_value"
	maxValueExpr := "sp.max_value"
	valueTypeExpr := "COALESCE(sp.data_type, 'string')"
	enumValuesExpr := "NULL::text"
	enumLabelsExpr := "NULL::text"
	isSupportedExpr := `(
        SELECT BOOL_OR(pm.is_supported)
          FROM param_mappings pm
         WHERE pm.standard_path = sp.standard_path
           AND pm.is_active = true
    )`
	args := []any{commandID}
	if paramModelID != nil {
		modelMappingJoin = `
LEFT JOIN LATERAL (
	    SELECT pm.default_value, pm.validation_pattern, pm.data_type,
           pm.min_value, pm.max_value, pm.enum_values, pm.enum_labels
      FROM param_mappings pm
     WHERE pm.standard_path = sp.standard_path
       AND pm.param_model_id = $2
       AND pm.is_active = true
       AND pm.is_supported = true
     ORDER BY (pm.source = 'custom') DESC, pm.private_path ASC
     LIMIT 1
) model_pm ON true`
		defaultValueExpr = "model_pm.default_value"
		validationPatternExpr = "model_pm.validation_pattern"
		minValueExpr = "COALESCE(model_pm.min_value, sp.min_value)"
		maxValueExpr = "COALESCE(model_pm.max_value, sp.max_value)"
		valueTypeExpr = "COALESCE(model_pm.data_type, sp.data_type, 'string')"
		enumValuesExpr = "model_pm.enum_values"
		enumLabelsExpr = "model_pm.enum_labels"
		paramModelFilter = `
  AND EXISTS (
      SELECT 1 FROM param_mappings pm
       WHERE pm.standard_path  = sp.standard_path
         AND pm.param_model_id = $2
         AND pm.is_active      = true
         AND pm.is_supported   = true
  )`
		// console 端绑 paramModel — EXISTS 已物理过滤 is_supported=true 行,
		// 返回行的 is_supported 列对返回行恒为 true。
		isSupportedExpr = `COALESCE((
        SELECT pm.is_supported FROM param_mappings pm
         WHERE pm.standard_path = sp.standard_path
           AND pm.param_model_id = $2
           AND pm.is_active = true
           AND pm.is_supported = true
         LIMIT 1
    ), false)`
		args = append(args, *paramModelID)
	}

	sqlText := `
SELECT
    csf.id, csf.command_id, csf.standard_path_id AS param_id, csf.mml_code, csf.label_i18n,
    csf.default_selected, csf.is_required, csf.sort_order, csf.created_at, csf.updated_at,
    sp.standard_path                      AS tr069_path,
    ` + valueTypeExpr + `                AS value_type,
    COALESCE(sp.access, 'READ_ONLY')      AS access_type,
    (sp.entry_type = 'object')            AS is_object,
    false                                 AS supports_add,
    false                                 AS supports_delete,
    COALESCE(sp.change_applies, 'Immediate') AS change_applies,
    CASE
        WHEN ` + minValueExpr + ` IS NOT NULL AND ` + maxValueExpr + ` IS NOT NULL THEN
            jsonb_build_object(
                'zh-CN', '[' || ` + minValueExpr + `::text || ', ' || ` + maxValueExpr + `::text || ']',
                'en-US', '[' || ` + minValueExpr + `::text || ', ' || ` + maxValueExpr + `::text || ']'
            )
        WHEN ` + minValueExpr + ` IS NOT NULL THEN
            jsonb_build_object(
                'zh-CN', '≥ ' || ` + minValueExpr + `::text,
                'en-US', '≥ ' || ` + minValueExpr + `::text
            )
        WHEN ` + maxValueExpr + ` IS NOT NULL THEN
            jsonb_build_object(
                'zh-CN', '≤ ' || ` + maxValueExpr + `::text,
                'en-US', '≤ ' || ` + maxValueExpr + `::text
            )
        ELSE '{}'::jsonb
    END                                   AS constraint_text_i18n,
    ` + defaultValueExpr + `                  AS default_value,
    ` + validationPatternExpr + `             AS js_regex,
    ` + validationPatternExpr + `             AS validation_pattern,
    ` + minValueExpr + `                      AS min_value,
    ` + maxValueExpr + `                      AS max_value,
    ` + enumValuesExpr + `                    AS enum_values,
    ` + enumLabelsExpr + `                    AS enum_labels,
    jsonb_build_object(
        'zh-CN', sp.standard_path,
        'en-US', sp.standard_path
    )                                     AS name_i18n,
    COALESCE(sp.description, '')          AS description,
    COALESCE(` + isSupportedExpr + `, true)             AS is_supported
FROM mml_command_sub_fields csf
JOIN standard_params sp ON sp.id = csf.standard_path_id
` + modelMappingJoin + `
WHERE csf.command_id = $1` + paramModelFilter + `
ORDER BY csf.sort_order ASC, csf.mml_code ASC`

	rows, err := r.pool.Query(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("query enriched sub_fields: %w", err)
	}
	defer rows.Close()

	var out []MMLCommandSubFieldEnriched
	for rows.Next() {
		e := MMLCommandSubFieldEnriched{}
		var labelI18n, constraintI18n, paramNameI18n []byte
		var enumValues, enumLabels *string
		if err := rows.Scan(
			&e.ID, &e.CommandID, &e.ParamID, &e.MMLCode, &labelI18n,
			&e.DefaultSelected, &e.IsRequired, &e.SortOrder, &e.CreatedAt, &e.UpdatedAt,
			&e.Tr069Path, &e.ValueType,
			&e.AccessType, &e.IsObject, &e.SupportsAdd, &e.SupportsDelete,
			&e.ChangeApplies, &constraintI18n,
			&e.DefaultValue, &e.JsRegex, &e.ValidationPattern,
			&e.MinValue, &e.MaxValue, &enumValues, &enumLabels, &paramNameI18n,
			&e.Description,
			&e.IsSupported,
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
		e.EnumOptions = buildEnumOptions(enumValues, enumLabels)
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

// ListAdminByCommand 返回 admin 视角的 sub_field 全集；is_supported 已从
// mml_command_sub_fields.is_supported 改为按 param_mappings.is_supported 聚合
// （T-0176-PR-A 之后 catalog 真值源就在 param_mappings 上，原 sub_field 列
// 不再权威；2026-05-27 用户决策同步前端展示）。
//
// 聚合规则：
//   - 每条 sub_field 关联 N 条 param_mappings 行（按 standard_path 维度，N = 注册
//     该 path 的 paramModel 数；is_active=false 不计）
//   - is_supported = BOOL_OR(pm.is_supported)，无行兜底 true
//   - supported_model_count / total_model_count 给前端做细粒度提示用
//
// 不做 paramModel 过滤（与 console 端 ListEnrichedByCommand 区分），admin 看的是
// "字典全貌 + 跨 paramModel 总体支持状况"。
func (r *PgSubFieldRepository) ListAdminByCommand(ctx context.Context, commandID uuid.UUID) ([]MMLCommandSubFieldEnriched, error) {
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
    CASE
        WHEN sp.min_value IS NOT NULL AND sp.max_value IS NOT NULL THEN
            jsonb_build_object(
                'zh-CN', '[' || sp.min_value::text || ', ' || sp.max_value::text || ']',
                'en-US', '[' || sp.min_value::text || ', ' || sp.max_value::text || ']'
            )
        WHEN sp.min_value IS NOT NULL THEN
            jsonb_build_object('zh-CN', '≥ ' || sp.min_value::text, 'en-US', '≥ ' || sp.min_value::text)
        WHEN sp.max_value IS NOT NULL THEN
            jsonb_build_object('zh-CN', '≤ ' || sp.max_value::text, 'en-US', '≤ ' || sp.max_value::text)
        ELSE '{}'::jsonb
    END                                   AS constraint_text_i18n,
    NULL::text                            AS default_value,
    NULL::text                            AS js_regex,
    jsonb_build_object('zh-CN', sp.standard_path, 'en-US', sp.standard_path) AS name_i18n,
    COALESCE(sp.description, '')          AS description,
    -- is_supported 真值源：active param_mappings 上的 OR 聚合；无任何映射 → 兜底 true
    COALESCE(
        BOOL_OR(pm.is_supported) FILTER (WHERE pm.is_active),
        true
    )                                     AS is_supported,
    COUNT(*) FILTER (WHERE pm.is_active AND pm.is_supported)::int AS supported_model_count,
    COUNT(*) FILTER (WHERE pm.is_active)::int                     AS total_model_count
FROM mml_command_sub_fields csf
JOIN standard_params sp ON sp.id = csf.standard_path_id
LEFT JOIN param_mappings pm ON pm.standard_path = sp.standard_path
WHERE csf.command_id = $1
GROUP BY csf.id, sp.standard_path, sp.data_type, sp.access, sp.entry_type,
         sp.change_applies, sp.min_value, sp.max_value, sp.description
ORDER BY csf.sort_order ASC, csf.mml_code ASC`
	rows, err := r.pool.Query(ctx, sqlText, commandID)
	if err != nil {
		return nil, fmt.Errorf("query admin sub_fields: %w", err)
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
			&e.Description, &e.IsSupported,
			&e.SupportedModelCount, &e.TotalModelCount,
		); err != nil {
			return nil, fmt.Errorf("scan admin sub_field row: %w", err)
		}
		if err := unmarshalI18n(labelI18n, &e.LabelI18n); err != nil {
			return nil, fmt.Errorf("unmarshal label_i18n: %w", err)
		}
		if err := unmarshalI18n(constraintI18n, &e.ConstraintTextI18n); err != nil {
			return nil, fmt.Errorf("unmarshal constraint_text_i18n: %w", err)
		}
		if err := unmarshalI18n(paramNameI18n, &e.ParamNameI18n); err != nil {
			return nil, fmt.Errorf("unmarshal name_i18n: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate admin sub_fields rows: %w", err)
	}
	return out, nil
}

// BatchCreate 批量插入 sub_field。逐条 INSERT（squirrel 在同一事务里串行执行）；
// 触发器 trg_mml_sub_fields_target_paths AFTER INSERT 会被触发 N 次重算
// target_paths —— 接受这点性能成本（前端单次 batch 通常 <= 30 条），换换实现简洁。
//
// 任何一条失败 → 事务整体回滚（pgx Begin/Rollback/Commit 语义）。
func (r *PgSubFieldRepository) BatchCreate(ctx context.Context, items []*MMLCommandSubField) error {
	if len(items) == 0 {
		return nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin batch tx: %w", err)
	}
	defer reliability.RollbackTx(ctx, tx, r.logger, "PgSubFieldRepository.BatchCreate")

	for _, sf := range items {
		if sf.ID == uuid.Nil {
			sf.ID = uuid.New()
		}
		labelI18n, err := marshalI18n(sf.LabelI18n)
		if err != nil {
			return fmt.Errorf("marshal label_i18n for %s: %w", sf.MMLCode, err)
		}
		_, err = tx.Exec(ctx, `
INSERT INTO mml_command_sub_fields
    (id, command_id, standard_path_id, mml_code, label_i18n,
     default_selected, is_required, sort_order)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			sf.ID, sf.CommandID, sf.ParamID, sf.MMLCode, labelI18n,
			sf.DefaultSelected, sf.IsRequired, sf.SortOrder)
		if err != nil {
			return fmt.Errorf("insert sub_field %s: %w", sf.MMLCode, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit batch tx: %w", err)
	}
	return nil
}

// MarkUnsupportedByStandardPath 把 (paramModelID, standardPath) 在 param_mappings
// 表中标 is_supported=false，仅更新当前 is_supported=true 的行（已 false 的不重复
// 触动 updated_at，避免抖动）。返回受影响行数。
//
// T-0176-PR-E 起从写 mml_command_sub_fields 迁到写 param_mappings 单一真值源；
// paramModelID 必填以隔离同 standardPath 在不同 paramModel 间的真值差异。
//
// 调用方：ACS 收到 SetParameterValues 9005 (Invalid parameter name) 时
// ResultAggregator 转发到此方法。standardPath 为空 → 短路返 0 不报错（防御）。
// 0 行受影响是合法情况：path 不在该 paramModel 的 mappings 里，或已经标过 false。
func (r *PgSubFieldRepository) MarkUnsupportedByStandardPath(
	ctx context.Context, paramModelID uuid.UUID, standardPath string,
) (int64, error) {
	if standardPath == "" {
		return 0, nil
	}
	const sqlText = `
UPDATE param_mappings
   SET is_supported = false,
       updated_at   = NOW()
 WHERE param_model_id = $1
   AND standard_path  = $2
   AND is_supported   = true`
	tag, err := r.pool.Exec(ctx, sqlText, paramModelID, standardPath)
	if err != nil {
		return 0, fmt.Errorf("mark param_mapping unsupported (pm=%s, path=%q): %w",
			paramModelID, standardPath, err)
	}
	return tag.RowsAffected(), nil
}

// ============================================================
// AdminGroupRepository (mml_command_groups)
// ============================================================

// AdminGroupRepository 提供 mml_command_groups 的 admin CRUD。
// Console 端读复用 ParamRepository.ListGroups；本接口承担写路径 + admin 全集 List。
type AdminGroupRepository interface {
	Create(ctx context.Context, g *CommandGroup) error
	Update(ctx context.Context, g *CommandGroup) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*CommandGroup, error)
	CountCommandsByGroup(ctx context.Context, groupID uuid.UUID) (int64, error)
	// List 返回 admin 视角全集 group（不过滤 chapter:%）。
	// console 端的 group_tree 是按 chapter 过滤的子集；admin 创建非 chapter group 后必须能在管理页看到。
	List(ctx context.Context, filter GroupFilter) ([]CommandGroup, error)
}

// GroupFilter 描述 admin 端列表的过滤维度。空字段 = 不过滤。
type GroupFilter struct {
	Source       string // 'standard' / 'admin' / 'extension'；空 = 全部
	ParamVersion string
	Search       string // group_code / name 任一匹配
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
	// path 是 ltree NOT NULL,顶层分组从 group_code 派生(冒号 / 空白 / 短横线
	// 等非 ltree 合法字符替换为 '_',对外仍以 group_code 展示)。
	// 2026-05-27 修复:此前 admin Create 漏填 path,导致 NULL 约束 500。
	pathLabel := ltreeLabelFromCode(g.GroupCode)
	const sqlText = `
INSERT INTO mml_command_groups (
    id, group_code, group_name_zh, group_name_en, param_version,
    display_order, name_i18n, source, catalog_protected, path
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::ltree)`
	nameI18n := jsonOrEmpty(map[string]string{"zh-CN": g.GroupNameZh, "en-US": g.GroupNameEn})
	if _, err := r.pool.Exec(ctx, sqlText,
		g.ID, g.GroupCode, g.GroupNameZh, g.GroupNameEn, g.ParamVersion,
		g.DisplayOrder, nameI18n, g.Source, g.CatalogProtected, pathLabel,
	); err != nil {
		return fmt.Errorf("insert group: %w", err)
	}
	return nil
}

// ltreeLabelFromCode 把 group_code 规范化为 ltree label。
// ltree 仅允许 [A-Za-z0-9_],其他字符替换为 '_';开头是数字时前缀 'g_'。
func ltreeLabelFromCode(code string) string {
	if code == "" {
		return "g_" + uuid.New().String()[:8]
	}
	out := make([]byte, 0, len(code))
	for i := 0; i < len(code); i++ {
		c := code[i]
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '_':
			out = append(out, c)
		default:
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "g"
	}
	if out[0] >= '0' && out[0] <= '9' {
		return "g_" + string(out)
	}
	return string(out)
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

// List 返回 admin 视角全集 group（不过滤 chapter:%），按 (param_version, display_order, group_code) 排序。
// 与 console 端 group_tree 区分：那个只取 group_code LIKE 'chapter:%'，admin 需要看到全部。
func (r *PgAdminGroupRepository) List(ctx context.Context, filter GroupFilter) ([]CommandGroup, error) {
	base := storage.Psql.Select(
		"id", "group_code", "group_name_zh", "group_name_en", "param_version",
		"display_order", "source", "catalog_protected",
	).From("mml_command_groups")
	if s := strings.TrimSpace(filter.Source); s != "" {
		base = base.Where(sq.Eq{"source": s})
	}
	if pv := strings.TrimSpace(filter.ParamVersion); pv != "" {
		base = base.Where(sq.Eq{"param_version": pv})
	}
	if sr := strings.TrimSpace(filter.Search); sr != "" {
		like := "%" + sr + "%"
		base = base.Where(sq.Or{
			sq.Expr("group_code ILIKE ?", like),
			sq.Expr("group_name_zh ILIKE ?", like),
			sq.Expr("group_name_en ILIKE ?", like),
		})
	}
	base = base.OrderBy("param_version ASC", "display_order ASC", "group_code ASC")

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list groups SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list groups: %w", err)
	}
	defer rows.Close()
	out := make([]CommandGroup, 0, 32)
	for rows.Next() {
		var g CommandGroup
		if err := rows.Scan(&g.ID, &g.GroupCode, &g.GroupNameZh, &g.GroupNameEn,
			&g.ParamVersion, &g.DisplayOrder, &g.Source, &g.CatalogProtected); err != nil {
			return nil, fmt.Errorf("scan group: %w", err)
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate group rows: %w", err)
	}
	return out, nil
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
	// logical_code 不再持久化为 DB 列（已 DROP）；读时从 command_code 派生。
	const sqlText = `
INSERT INTO mml_commands (
    id, command_name, command_code, category, description, rpc_method,
    operation_type, help_doc, notes,
    target_paths, target_object, group_id,
    command_name_i18n, require_confirm, confirm_msg_i18n,
    logical_name_i18n, source, catalog_protected
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`
	if _, err := r.pool.Exec(ctx, sqlText,
		c.ID, c.CommandName, c.CommandCode, c.Category, c.Description, c.RPCMethod,
		c.OperationType, c.HelpDoc, c.Notes,
		targetPaths, nullIfEmpty(c.TargetObject), c.GroupID,
		commandNameI18n, c.RequireConfirm, confirmMsgI18n,
		logicalNameI18n, c.Source, c.CatalogProtected,
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
    logical_name_i18n = $12
WHERE id = $1`
	tag, err := r.pool.Exec(ctx, sqlText,
		c.ID, c.CommandName, c.Category, c.Description, c.HelpDoc, c.Notes,
		nullIfEmpty(c.TargetObject), c.GroupID,
		commandNameI18n, c.RequireConfirm, confirmMsgI18n,
		logicalNameI18n,
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
	// ErrCommandNameDuplicated 当目录命令 command_name/command_code 撞 DB 唯一约束时返回；
	// handler 翻 HTTP 409，避免裸 500 + 泄露约束名。
	ErrCommandNameDuplicated = errors.New("mml: command name or code already exists")
	// ErrGroupParamVersionNotFound 当创建/更新 group 引用了不存在的 param_version
	// （外键 mml_param_groups_param_version_fkey, SQLSTATE 23503）时返回；
	// handler 翻 HTTP 422，避免裸 500 + 泄露 SQL 约束名 / SQLSTATE。
	ErrGroupParamVersionNotFound = errors.New("mml: param_version does not exist")

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
