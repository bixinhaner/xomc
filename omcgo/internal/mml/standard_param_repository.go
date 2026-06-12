package mml

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"

	sq "github.com/Masterminds/squirrel"
)

// ============================================================
// standard_param_repository.go — MML 配置管理"路径下拉"数据源
//
// 与 internal/config/parammodel.StandardParam 的关系：
//   parammodel 那个 struct 是 dictloader 加载流程的内部表示，不含 ID/Description。
//   admin UI 选 path 时需要：
//     · id（FK 写入 mml_command_sub_fields.standard_path_id）
//     · 完整元数据（autofill 表单：data_type / access / change_applies / range / description）
//   因此本包定义独立的 StandardParamView，避免反向依赖 parammodel + 不污染原 model。
//
// 端点：GET /api/v1/mml/admin/standard-params?q=<text>&entry_type=parameter&page=&page_size=
// 仅 admin 角色访问（端点级 Casbin 自动收敛）。
// ============================================================

// StandardParamView 是 admin path 下拉返回的完整字段。
type StandardParamView struct {
	ID            uuid.UUID `json:"id"`
	StandardPath  string    `json:"standard_path"`
	EntryType     string    `json:"entry_type"`     // 'object' / 'parameter'
	Access        string    `json:"access"`         // READ_ONLY / READ_WRITE
	DataType      string    `json:"data_type"`      // string / unsignedInt / boolean ...
	ChangeApplies string    `json:"change_applies"` // Immediate / OnReboot / OnSession
	MinValue      *int64    `json:"min_value,omitempty"`
	MaxValue      *int64    `json:"max_value,omitempty"`
	Description   string    `json:"description"`
}

// StandardParamFilter 决定 List 的过滤维度。Search 走 ILIKE %q%。
// EntryType 空字符串 = 不过滤；为 'parameter' / 'object' 时仅返该类型。
type StandardParamFilter struct {
	Search    string
	EntryType string
	model.ListRequest
}

// StandardParamRepository 路径字典只读访问。
type StandardParamRepository interface {
	List(ctx context.Context, filter StandardParamFilter) (*model.ListResponse[StandardParamView], error)
	GetByID(ctx context.Context, id uuid.UUID) (*StandardParamView, error)
}

// PgStandardParamRepository PostgreSQL 实现。
type PgStandardParamRepository struct {
	pool *pgxpool.Pool
}

// NewPgStandardParamRepository 构造。
func NewPgStandardParamRepository(pool *pgxpool.Pool) *PgStandardParamRepository {
	return &PgStandardParamRepository{pool: pool}
}

var _ StandardParamRepository = (*PgStandardParamRepository)(nil)

const standardParamSelectColumns = `id, standard_path, entry_type,
    COALESCE(access, '') AS access,
    COALESCE(data_type, '') AS data_type,
    COALESCE(change_applies, '') AS change_applies,
    min_value, max_value, description`

// standardParamSearchCondition 构建路径下拉的搜索条件。
//
// Issue #115 调整2：本端点（GET /mml/admin/standard-params）只服务「选择标准 PATH」
// 下拉（StandardParamSelect / PathPicker）与按完整 path 解析友好名，用户视角是按 PATH 查找。
// 旧实现对 standard_path + description 两列联合 ILIKE，当关键字仅命中 description 时会
// 返回 path 不含该词的条目，表现为「显示了不匹配的 PATH」。故收窄为仅 standard_path。
// （需要按描述搜索的标准参数管理页走的是另一套 /param-models/standard 端点，不受影响。）
//
// search 为空 / 纯空白返回 nil，调用方据此不追加 WHERE。
func standardParamSearchCondition(search string) sq.Sqlizer {
	s := strings.TrimSpace(search)
	if s == "" {
		return nil
	}
	return sq.Expr("standard_path ILIKE ?", "%"+s+"%")
}

// List 按 search / entry_type 过滤并分页返回 standard_params。
func (r *PgStandardParamRepository) List(ctx context.Context, filter StandardParamFilter) (*model.ListResponse[StandardParamView], error) {
	base := storage.Psql.Select(splitColumns(standardParamSelectColumns)...).From("standard_params")
	countBase := storage.Psql.Select("COUNT(*)").From("standard_params")

	if cond := standardParamSearchCondition(filter.Search); cond != nil {
		base = base.Where(cond)
		countBase = countBase.Where(cond)
	}
	if et := strings.TrimSpace(filter.EntryType); et != "" {
		base = base.Where(sq.Eq{"entry_type": et})
		countBase = countBase.Where(sq.Eq{"entry_type": et})
	}

	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count standard_params SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count standard_params: %w", err)
	}

	base = base.
		OrderBy("standard_path ASC").
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list standard_params SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list standard_params: %w", err)
	}
	defer rows.Close()

	items := make([]StandardParamView, 0, filter.Limit())
	for rows.Next() {
		var v StandardParamView
		if err := rows.Scan(&v.ID, &v.StandardPath, &v.EntryType, &v.Access,
			&v.DataType, &v.ChangeApplies, &v.MinValue, &v.MaxValue, &v.Description); err != nil {
			return nil, fmt.Errorf("scan standard_param: %w", err)
		}
		items = append(items, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate standard_params rows: %w", err)
	}

	return model.NewListResponse(items, total, filter.Page, filter.Limit()), nil
}

// GetByID 单条查询（autofill 兜底：前端缓存 miss 时回查）。
func (r *PgStandardParamRepository) GetByID(ctx context.Context, id uuid.UUID) (*StandardParamView, error) {
	const sqlText = `SELECT ` + standardParamSelectColumns + ` FROM standard_params WHERE id = $1`
	var v StandardParamView
	if err := r.pool.QueryRow(ctx, sqlText, id).Scan(
		&v.ID, &v.StandardPath, &v.EntryType, &v.Access,
		&v.DataType, &v.ChangeApplies, &v.MinValue, &v.MaxValue, &v.Description,
	); err != nil {
		return nil, fmt.Errorf("get standard_param by id: %w", err)
	}
	return &v, nil
}

// splitColumns 把 standardParamSelectColumns 这种逗号分隔字符串拆为 []string，
// 喂给 squirrel.Select。换行 / 空白都吃掉。
func splitColumns(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
