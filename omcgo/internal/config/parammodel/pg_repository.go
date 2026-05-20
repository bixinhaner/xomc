package parammodel

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgRepository 是 Repository 的 PostgreSQL 实现（pgx + squirrel）。
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewPgRepository 构造一个绑定到给定 pgxpool 的 PgRepository。
func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

// ListMappingsByParamModel 实现 Repository。
//
// 默认映射按 standard_path 升序，确保 Translator.Mappings() 的输出在跨实例间稳定可比。
func (r *PgRepository) ListMappingsByParamModel(ctx context.Context, paramModelID uuid.UUID) ([]ParamMapping, error) {
	sqlStr, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("id", "param_model_id", "standard_path", "private_path",
			"entry_type", "access", "data_type", "change_applies",
			"min_value", "max_value",
			"enum_values", "enum_labels", // T-0158
			"mirror_with", // T-0159
			"is_storable", "is_active", "is_supported").
		From("param_mappings").
		Where(sq.Eq{"param_model_id": paramModelID, "is_active": true}).
		OrderBy("standard_path").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build sql list default mappings: %w", err)
	}
	rows, err := r.pool.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("query default mappings %s: %w", paramModelID, err)
	}
	defer rows.Close()
	return scanDefaultMappings(rows)
}

// ListDiscoveredMappings 实现 Repository。
//
// 返回的 ParamMapping.ParamModelID 为零值——discovered_param_mappings 表无该列。
// 若调用方需要 paramModelID（罕见），从 product 反查。
func (r *PgRepository) ListDiscoveredMappings(ctx context.Context, productID uuid.UUID, swVersion string) ([]ParamMapping, error) {
	sqlStr, args, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Select("id", "product_id", "software_version", "standard_path", "private_path",
			"entry_type", "access", "data_type", "change_applies",
			"min_value", "max_value",
			"enum_values", "enum_labels", // T-0158
			"mirror_with", // T-0159
			"is_storable", "is_active", "is_supported").
		From("discovered_param_mappings").
		Where(sq.Eq{"product_id": productID, "software_version": swVersion, "is_active": true}).
		OrderBy("standard_path").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build sql list discovered mappings: %w", err)
	}
	rows, err := r.pool.Query(ctx, sqlStr, args...)
	if err != nil {
		return nil, fmt.Errorf("query discovered mappings %s/%s: %w", productID, swVersion, err)
	}
	defer rows.Close()
	return scanDiscoveredMappings(rows)
}

func scanDefaultMappings(rows pgx.Rows) ([]ParamMapping, error) {
	var out []ParamMapping
	for rows.Next() {
		var m ParamMapping
		var access, dataType, changeApplies *string
		if err := rows.Scan(
			&m.ID, &m.ParamModelID, &m.StandardPath, &m.PrivatePath,
			&m.EntryType, &access, &dataType, &changeApplies,
			&m.MinValue, &m.MaxValue,
			&m.EnumValues, &m.EnumLabels, // T-0158
			&m.MirrorWith, // T-0159
			&m.IsStorable, &m.IsActive, &m.IsSupported,
		); err != nil {
			return nil, fmt.Errorf("scan default mapping: %w", err)
		}
		m.Access = strDeref(access)
		m.DataType = strDeref(dataType)
		m.ChangeApplies = strDeref(changeApplies)
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate default mappings: %w", err)
	}
	return out, nil
}

func scanDiscoveredMappings(rows pgx.Rows) ([]ParamMapping, error) {
	var out []ParamMapping
	for rows.Next() {
		var (
			m             ParamMapping
			productID     uuid.UUID
			swVersion     string
			access        *string
			dataType      *string
			changeApplies *string
		)
		if err := rows.Scan(
			&m.ID, &productID, &swVersion, &m.StandardPath, &m.PrivatePath,
			&m.EntryType, &access, &dataType, &changeApplies,
			&m.MinValue, &m.MaxValue,
			&m.EnumValues, &m.EnumLabels, // T-0158
			&m.MirrorWith, // T-0159
			&m.IsStorable, &m.IsActive, &m.IsSupported,
		); err != nil {
			return nil, fmt.Errorf("scan discovered mapping: %w", err)
		}
		m.Access = strDeref(access)
		m.DataType = strDeref(dataType)
		m.ChangeApplies = strDeref(changeApplies)
		sw := swVersion
		m.SoftwareVersion = &sw
		_ = productID // 调用方已持有 productID；不回填 ParamModelID（无该列）
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate discovered mappings: %w", err)
	}
	return out, nil
}

func strDeref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// UpsertDiscoveredMappings 实现 IntersectRepository。
//
// 写入语义：DELETE WHERE (product_id, software_version) + INSERT 给定 mappings，
// 单事务保证。len(mappings)==0 时仅执行 DELETE（用于"立即重置"路径）。
//
// id 列由 PG `gen_random_uuid()` 默认生成；调用方传入零值即可。
func (r *PgRepository) UpsertDiscoveredMappings(ctx context.Context, productID uuid.UUID, swVersion string, mappings []ParamMapping) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx upsert discovered: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	delSQL, delArgs, err := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
		Delete("discovered_param_mappings").
		Where(sq.Eq{"product_id": productID, "software_version": swVersion}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete discovered: %w", err)
	}
	if _, err := tx.Exec(ctx, delSQL, delArgs...); err != nil {
		return fmt.Errorf("exec delete discovered %s/%s: %w", productID, swVersion, err)
	}

	if len(mappings) > 0 {
		ib := sq.StatementBuilder.PlaceholderFormat(sq.Dollar).
			Insert("discovered_param_mappings").
			Columns(
				"product_id", "software_version", "standard_path", "private_path",
				"entry_type", "access", "data_type", "change_applies",
				"min_value", "max_value",
				"enum_values", "enum_labels", // T-0158
				"mirror_with", // T-0159
				"is_storable", "is_active", "is_supported",
			)
		for _, m := range mappings {
			ib = ib.Values(
				productID, swVersion, m.StandardPath, m.PrivatePath,
				m.EntryType, nilIfEmpty(m.Access), nilIfEmpty(m.DataType), nilIfEmpty(m.ChangeApplies),
				m.MinValue, m.MaxValue,
				m.EnumValues, m.EnumLabels, // T-0158
				m.MirrorWith, // T-0159
				m.IsStorable, m.IsActive, m.IsSupported,
			)
		}
		insSQL, insArgs, err := ib.ToSql()
		if err != nil {
			return fmt.Errorf("build insert discovered: %w", err)
		}
		if _, err := tx.Exec(ctx, insSQL, insArgs...); err != nil {
			return fmt.Errorf("exec insert discovered %s/%s: %w", productID, swVersion, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit upsert discovered: %w", err)
	}
	return nil
}

// nilIfEmpty 把空字符串转 nil，匹配 access/data_type/change_applies 列的可空语义。
func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
