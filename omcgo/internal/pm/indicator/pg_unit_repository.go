package indicator

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PgUnitRepository 是 indicator_unit 单位字典的 PG 实现（T-0098 P3-03）。
type PgUnitRepository struct {
	pool *pgxpool.Pool
}

func NewPgUnitRepository(pool *pgxpool.Pool) *PgUnitRepository {
	return &PgUnitRepository{pool: pool}
}

// List 返回全量单位（设计 §2 27 行典型规模）。
func (r *PgUnitRepository) List(ctx context.Context) ([]*IndicatorUnit, error) {
	const q = `SELECT id, COALESCE(en_name,''), COALESCE(cn_name,''), created_at, updated_at
	          FROM indicator_unit ORDER BY id`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("query indicator_unit: %w", err)
	}
	defer rows.Close()
	var out []*IndicatorUnit
	for rows.Next() {
		u := &IndicatorUnit{}
		if err := rows.Scan(&u.ID, &u.EnName, &u.CnName, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan indicator_unit: %w", err)
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// Get 单条详情。
func (r *PgUnitRepository) Get(ctx context.Context, id string) (*IndicatorUnit, error) {
	const q = `SELECT id, COALESCE(en_name,''), COALESCE(cn_name,''), created_at, updated_at
	          FROM indicator_unit WHERE id = $1`
	u := &IndicatorUnit{}
	if err := r.pool.QueryRow(ctx, q, id).Scan(&u.ID, &u.EnName, &u.CnName, &u.CreatedAt, &u.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("indicator_unit %q not found", id)
		}
		return nil, fmt.Errorf("get indicator_unit %q: %w", id, err)
	}
	return u, nil
}

// Upsert 新建或更新；id 由调用方提供（业务唯一码）。
func (r *PgUnitRepository) Upsert(ctx context.Context, id, enName, cnName string) (*IndicatorUnit, error) {
	if id == "" {
		return nil, fmt.Errorf("id required")
	}
	const upsertSQL = `
INSERT INTO indicator_unit (id, en_name, cn_name)
VALUES ($1, NULLIF($2,''), NULLIF($3,''))
ON CONFLICT (id) DO UPDATE
SET en_name = EXCLUDED.en_name, cn_name = EXCLUDED.cn_name, updated_at = NOW()`
	if _, err := r.pool.Exec(ctx, upsertSQL, id, enName, cnName); err != nil {
		return nil, fmt.Errorf("upsert indicator_unit: %w", err)
	}
	return r.Get(ctx, id)
}

// Delete 删除前校验是否被三表 unit_id 引用，被引用则拒绝。
func (r *PgUnitRepository) Delete(ctx context.Context, id string) (bool, error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	for _, table := range []string{"perf_indicators_enb", "perf_indicators_gsm", "perf_indicators_gnb"} {
		cSQL, cArgs, err := psql.Select("COUNT(*)").From(table).Where(sq.Eq{"unit_id": id}).ToSql()
		if err != nil {
			return false, fmt.Errorf("build count sql for %s: %w", table, err)
		}
		var n int
		if err := r.pool.QueryRow(ctx, cSQL, cArgs...).Scan(&n); err != nil {
			return false, fmt.Errorf("count refs in %s: %w", table, err)
		}
		if n > 0 {
			return false, fmt.Errorf("indicator_unit %q referenced by %d rows in %s", id, n, table)
		}
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM indicator_unit WHERE id = $1`, id)
	if err != nil {
		return false, fmt.Errorf("delete indicator_unit %q: %w", id, err)
	}
	return tag.RowsAffected() > 0, nil
}
