package mml

import (
	"context"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

// ParamRepository provides read access to the parameter library.
type ParamRepository interface {
	ListVersions(ctx context.Context) ([]ParamVersion, error)
	ListGroupsByVersion(ctx context.Context, versionCode string) ([]CommandGroup, error)
	ListParamsByGroup(ctx context.Context, groupID uuid.UUID) ([]Param, error)
	SearchParams(ctx context.Context, filter ParamFilter) ([]Param, error)
}

// PgParamRepository implements ParamRepository with PostgreSQL.
type PgParamRepository struct {
	pool *pgxpool.Pool
}

// NewPgParamRepository creates a new PgParamRepository.
func NewPgParamRepository(pool *pgxpool.Pool) *PgParamRepository {
	return &PgParamRepository{pool: pool}
}

var _ ParamRepository = (*PgParamRepository)(nil)

var paramVersionColumns = []string{
	"version_code", "version_name", "description", "release_date",
	"product_models", "software_versions",
	"is_active", "is_deprecated", "group_count", "param_count",
	"created_at", "updated_at",
}

var paramGroupColumns = []string{
	"id", "group_code", "group_name_zh", "group_name_en",
	"parent_id", "level",
	"is_listable", "is_modifiable", "is_addable", "is_removable",
	"add_object_path", "delete_object_path",
	"param_version", "platform_support", "mobile_support", "broadband_support",
	"cell_number", "cell_index_location",
	"require_second_confirm", "confirm_message_zh", "confirm_message_en",
	"display_order",
}

var paramColumns = []string{
	"p.id", "p.param_code", "p.param_name_zh", "p.param_name_en",
	"p.tr069_path", "p.value_type", "p.value_constraint",
	"p.default_value", "p.js_regex",
	"p.is_writable", "p.is_listable", "p.is_modifiable",
	"p.is_addable", "p.is_removable", "p.is_leaf", "p.is_dynamic",
	"p.display_order", "p.param_version",
	"p.software_version", "p.platform_support",
	"p.mobile_support", "p.broadband_support",
	"p.memo", "p.explanation_zh", "p.explanation_en",
	"p.title_zh", "p.title_en",
	"p.require_second_confirm", "p.confirm_message_zh", "p.confirm_message_en",
}

func (r *PgParamRepository) ListVersions(ctx context.Context) ([]ParamVersion, error) {
	query, args, err := storage.Psql.Select(paramVersionColumns...).
		From("mml_param_versions").
		OrderBy("version_code").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list param versions SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list param versions: %w", err)
	}
	defer rows.Close()

	var result []ParamVersion
	for rows.Next() {
		var v ParamVersion
		if err := rows.Scan(
			&v.VersionCode, &v.VersionName, &v.Description, &v.ReleaseDate,
			&v.ProductModels, &v.SoftwareVersions,
			&v.IsActive, &v.IsDeprecated, &v.GroupCount, &v.ParamCount,
			&v.CreatedAt, &v.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan param version: %w", err)
		}
		result = append(result, v)
	}
	return result, rows.Err()
}

func (r *PgParamRepository) ListGroupsByVersion(ctx context.Context, versionCode string) ([]CommandGroup, error) {
	query, args, err := storage.Psql.Select(paramGroupColumns...).
		From("mml_command_groups").
		Where(sq.Eq{"param_version": versionCode, "is_active": true}).
		OrderBy("display_order", "group_code").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list param groups SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list param groups: %w", err)
	}
	defer rows.Close()

	var result []CommandGroup
	for rows.Next() {
		var g CommandGroup
		if err := scanCommandGroup(rows, &g); err != nil {
			return nil, fmt.Errorf("scan param group: %w", err)
		}
		result = append(result, g)
	}
	return result, rows.Err()
}

func (r *PgParamRepository) ListParamsByGroup(ctx context.Context, groupID uuid.UUID) ([]Param, error) {
	query, args, err := storage.Psql.Select(paramColumns...).
		From("mml_params p").
		Join("mml_group_param_rel r ON r.param_id = p.id").
		Where(sq.Eq{"r.group_id": groupID, "p.is_active": true}).
		OrderBy("r.sort_order", "p.display_order").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list params by group SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list params by group: %w", err)
	}
	defer rows.Close()

	return scanParams(rows)
}

func (r *PgParamRepository) SearchParams(ctx context.Context, filter ParamFilter) ([]Param, error) {
	base := storage.Psql.Select(paramColumns...).
		From("mml_params p").
		Where(sq.Eq{"p.param_version": filter.VersionCode, "p.is_active": true})

	if filter.Search != nil && *filter.Search != "" {
		pattern := "%" + *filter.Search + "%"
		base = base.Where(sq.Or{
			sq.ILike{"p.param_name_zh": pattern},
			sq.ILike{"p.param_name_en": pattern},
			sq.ILike{"p.param_code": pattern},
		})
	}

	if filter.Tr069Path != nil && *filter.Tr069Path != "" {
		pattern := "%" + *filter.Tr069Path + "%"
		base = base.Where(sq.ILike{"p.tr069_path": pattern})
	}

	query, args, err := base.OrderBy("p.display_order").Limit(100).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build search params SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search params: %w", err)
	}
	defer rows.Close()

	return scanParams(rows)
}

// ---- scan helpers ----

func scanCommandGroup(rows pgx.Rows, g *CommandGroup) error {
	return rows.Scan(
		&g.ID, &g.GroupCode, &g.GroupNameZh, &g.GroupNameEn,
		&g.ParentID, &g.Level,
		&g.IsListable, &g.IsModifiable, &g.IsAddable, &g.IsRemovable,
		&g.AddObjectPath, &g.DeleteObjectPath,
		&g.ParamVersion, &g.PlatformSupport, &g.MobileSupport, &g.BroadbandSupport,
		&g.CellNumber, &g.CellIndexLocation,
		&g.RequireSecondConfirm, &g.ConfirmMessageZh, &g.ConfirmMessageEn,
		&g.DisplayOrder,
	)
}

func scanParams(rows pgx.Rows) ([]Param, error) {
	var result []Param
	for rows.Next() {
		var p Param
		var constraint []byte
		if err := rows.Scan(
			&p.ID, &p.ParamCode, &p.ParamNameZh, &p.ParamNameEn,
			&p.Tr069Path, &p.ValueType, &constraint,
			&p.DefaultValue, &p.JsRegex,
			&p.IsWritable, &p.IsListable, &p.IsModifiable,
			&p.IsAddable, &p.IsRemovable, &p.IsLeaf, &p.IsDynamic,
			&p.DisplayOrder, &p.ParamVersion,
			&p.SoftwareVersion, &p.PlatformSupport,
			&p.MobileSupport, &p.BroadbandSupport,
			&p.Memo, &p.ExplanationZh, &p.ExplanationEn,
			&p.TitleZh, &p.TitleEn,
			&p.RequireSecondConfirm, &p.ConfirmMessageZh, &p.ConfirmMessageEn,
		); err != nil {
			return nil, fmt.Errorf("scan param: %w", err)
		}
		if constraint != nil {
			p.ValueConstraint = json.RawMessage(constraint)
		}
		result = append(result, p)
	}
	return result, rows.Err()
}
