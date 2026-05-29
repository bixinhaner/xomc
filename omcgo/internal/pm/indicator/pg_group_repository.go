package indicator

import (
	"context"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/storage"
)

var groupColumns = []string{
	"id", "en_name", "operator_code", "is_build_in",
	"description", "parent_id", "cn_name", "created_at", "updated_at",
}

var _ GroupRepository = (*PgGroupRepository)(nil)

type PgGroupRepository struct {
	db storage.DB
}

func NewPgGroupRepository(pool *pgxpool.Pool) *PgGroupRepository {
	return &PgGroupRepository{db: storage.NewPoolDB(pool)}
}

// List 列出该 device type 下全部分组(平台无关)。
func (r *PgGroupRepository) List(ctx context.Context, dt DeviceType) ([]*IndicatorGroup, error) {
	return r.listInternal(ctx, dt, "")
}

// ListByPlatform 列出**有该 platform 公式关联的指标所属**的分组。
// 用途:product/kpi-library 详情态下拉,只显示当前 platform 实际涉及的分组。
// 实现:WHERE EXISTS (SELECT 1 FROM perf_indicators_X i WHERE i.group_id = g.id
//                    AND EXISTS (SELECT 1 FROM rela_platform_indicator_formula_X f
//                                 WHERE f.indicator_id = i.id AND f.platform_name = $1))
// platform 空时等价于 List(全量)。
func (r *PgGroupRepository) ListByPlatform(ctx context.Context, dt DeviceType, platform string) ([]*IndicatorGroup, error) {
	return r.listInternal(ctx, dt, platform)
}

func (r *PgGroupRepository) listInternal(ctx context.Context, dt DeviceType, platform string) ([]*IndicatorGroup, error) {
	table := dt.GroupTable()
	prefixedCols := make([]string, len(groupColumns))
	for i, c := range groupColumns {
		prefixedCols[i] = "g." + c
	}
	builder := storage.Psql.Select(prefixedCols...).From(table + " g")
	if platform != "" {
		// EXISTS 嵌套 EXISTS 与 pg_indicator_repository.go::PlatformName 过滤一致语义
		builder = builder.Where(sq.Expr(
			fmt.Sprintf(
				"EXISTS (SELECT 1 FROM %s i WHERE i.group_id = g.id "+
					"AND EXISTS (SELECT 1 FROM %s f WHERE f.indicator_id = i.id AND f.platform_name = ?))",
				dt.IndicatorTable(), dt.FormulaTable(),
			),
			platform,
		))
	}
	query, args, err := builder.OrderBy("g.en_name").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list %s SQL: %w", table, err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", table, err)
	}
	defer rows.Close()

	var groups []*IndicatorGroup
	for rows.Next() {
		g, err := scanGroup(rows)
		if err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating group rows: %w", err)
	}
	return groups, nil
}

func (r *PgGroupRepository) GetByID(ctx context.Context, dt DeviceType, id string) (*IndicatorGroup, error) {
	table := dt.GroupTable()
	query, args, err := storage.Psql.Select(groupColumns...).
		From(table).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get %s SQL: %w", table, err)
	}

	g, err := scanGroupRow(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get %s: %w", table, err)
	}
	return g, nil
}

func (r *PgGroupRepository) Create(ctx context.Context, dt DeviceType, group *IndicatorGroup) error {
	table := dt.GroupTable()
	now := time.Now()
	group.CreatedAt = now
	group.UpdatedAt = now

	query, args, err := storage.Psql.Insert(table).
		Columns("id", "en_name", "operator_code", "is_build_in",
			"description", "parent_id", "cn_name", "created_at", "updated_at").
		Values(group.ID, group.EnName, group.OperatorCode, group.IsBuildIn,
			group.Description, group.ParentID, group.CnName, group.CreatedAt, group.UpdatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert %s SQL: %w", table, err)
	}

	_, err = r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert %s: %w", table, err)
	}
	return nil
}

func (r *PgGroupRepository) Update(ctx context.Context, dt DeviceType, id string, req *UpdateGroupRequest) error {
	table := dt.GroupTable()
	builder := storage.Psql.Update(table)

	if req.EnName != nil {
		builder = builder.Set("en_name", *req.EnName)
	}
	if req.CnName != nil {
		builder = builder.Set("cn_name", *req.CnName)
	}
	if req.Description != nil {
		builder = builder.Set("description", *req.Description)
	}

	query, args, err := builder.Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("build update %s SQL: %w", table, err)
	}

	// No SET clause means no fields to update
	if !strings.Contains(query, "SET ") {
		return nil
	}

	result, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update %s: %w", table, err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgGroupRepository) Delete(ctx context.Context, dt DeviceType, id string, tx pgx.Tx) error {
	table := dt.GroupTable()
	query, args, err := storage.Psql.Delete(table).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete %s SQL: %w", table, err)
	}

	q := querier(r.db, tx)
	result, err := q.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete %s: %w", table, err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgGroupRepository) CountIndicatorsByGroup(ctx context.Context, dt DeviceType) (map[string]int64, error) {
	indTable := dt.IndicatorTable()
	grpTable := dt.GroupTable()

	// Count indicators per group, including groups with 0 indicators.
	query, args, err := storage.Psql.Select(
		"g.id", "COALESCE(COUNT(i.id), 0)").
		From(grpTable + " g").
		LeftJoin(indTable+" i ON i.group_id = g.id").
		GroupBy("g.id").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count indicators by group SQL: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("count indicators by group: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int64)
	for rows.Next() {
		var groupID string
		var count int64
		if err := rows.Scan(&groupID, &count); err != nil {
			return nil, fmt.Errorf("scan indicator count: %w", err)
		}
		counts[groupID] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating group count rows: %w", err)
	}
	return counts, nil
}

func scanGroup(rows pgx.Rows) (*IndicatorGroup, error) {
	var g IndicatorGroup
	err := rows.Scan(
		&g.ID, &g.EnName, &g.OperatorCode, &g.IsBuildIn,
		&g.Description, &g.ParentID, &g.CnName, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan indicator_group row: %w", err)
	}
	return &g, nil
}

func scanGroupRow(row pgx.Row) (*IndicatorGroup, error) {
	var g IndicatorGroup
	err := row.Scan(
		&g.ID, &g.EnName, &g.OperatorCode, &g.IsBuildIn,
		&g.Description, &g.ParentID, &g.CnName, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// querier returns tx if provided, otherwise falls back to db.
// pgx.Tx satisfies storage.DB, so this works for transaction participation.
func querier(db storage.DB, tx pgx.Tx) storage.DB {
	if tx != nil {
		return tx
	}
	return db
}
