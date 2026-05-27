package devsweep

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository 抽象 devsweep 所需的写入与影响面查询。
//
// MarkUnsupportedBatch：把 (param_model_id, standard_path[]) 列表的
// is_supported 翻 false（仅当原值 true 才动 — 守护防重复 updated_at 抖动）。
// 返回实际影响的行数。
//
// CountDevicesByParamModel：返回该 paramModel 关联的活跃设备数（未软删）。
// 用于 paramModel-wide 安全门。
type Repository interface {
	MarkUnsupportedBatch(ctx context.Context, paramModelID uuid.UUID, standardPaths []string) (int64, error)
	CountDevicesByParamModel(ctx context.Context, paramModelID uuid.UUID) (int64, error)
}

// PgRepository 是 Repository 的 PostgreSQL 实现。
type PgRepository struct {
	pool *pgxpool.Pool
}

// NewPgRepository 构造一个绑定到给定 pgxpool 的 PgRepository。
func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

// MarkUnsupportedBatch 实现 Repository。
//
// SQL 与 mml.PgSubFieldRepository.MarkUnsupportedByStandardPath（PR-E
// 单条版本）语义一致；此处批量化避免 N 个 RTT，且做 is_supported=true
// 守护防止反复触动 updated_at。
//
// 空 standardPaths → 直接返回 (0, nil)。
func (r *PgRepository) MarkUnsupportedBatch(ctx context.Context, paramModelID uuid.UUID, standardPaths []string) (int64, error) {
	if len(standardPaths) == 0 {
		return 0, nil
	}
	const sqlText = `
UPDATE param_mappings
   SET is_supported = false,
       updated_at   = NOW()
 WHERE param_model_id = $1
   AND standard_path  = ANY($2::text[])
   AND is_supported   = true`
	tag, err := r.pool.Exec(ctx, sqlText, paramModelID, standardPaths)
	if err != nil {
		return 0, fmt.Errorf("mark param_mappings unsupported (pm=%s, paths=%d): %w",
			paramModelID, len(standardPaths), err)
	}
	return tag.RowsAffected(), nil
}

// CountDevicesByParamModel 实现 Repository。
//
// 通过 products.param_model_id 联表反查 devices —— devices 表本身不持
// param_model_id（T-0176-PR-D 后写回但 schema 仍以 product_id 为权威）。
// 用 LEFT JOIN 避免 product_id 为空时被排除（孤儿设备本就不应受影响，
// 但保留它们计数为 0 即可）。
//
// deleted_at IS NULL 排除软删；并对 devices 表是否分区无影响（COUNT 兼容）。
func (r *PgRepository) CountDevicesByParamModel(ctx context.Context, paramModelID uuid.UUID) (int64, error) {
	const sqlText = `
SELECT COUNT(*)
  FROM devices d
  JOIN products p ON p.id = d.product_id
 WHERE p.param_model_id = $1
   AND d.deleted_at IS NULL`
	var n int64
	if err := r.pool.QueryRow(ctx, sqlText, paramModelID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count devices by param_model (pm=%s): %w", paramModelID, err)
	}
	return n, nil
}
