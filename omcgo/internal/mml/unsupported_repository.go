package mml

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// UnsupportedPath 是某产品下一条不支持参数 PATH 的读 / 写支持状态。
type UnsupportedPath struct {
	Path             string `json:"path"`
	ReadUnsupported  bool   `json:"read_unsupported"`
	WriteUnsupported bool   `json:"write_unsupported"`
}

// ProductUnsupportedPathRepository 维护「按 product_id 记录的设备不支持参数 PATH」自学习表
// （product_unsupported_paths，migration 000029）。读 / 写分别标记。
//
//   - Record：MML 执行命中 path 不支持类故障时 upsert (product_id, standardPath)，按 fault 决定读/写标记。
//   - ListByProduct：前端「选择命令 / 配置参数」按所选产品 + 命令读/写类型过滤展示。
type ProductUnsupportedPathRepository interface {
	// Record upsert 一条 (product_id, standardPath)；冲突则 OR 累加读/写标记、刷新 last_seen/hit_count。
	Record(ctx context.Context, productID uuid.UUID, standardPath string, readUnsupported, writeUnsupported bool, faultCode int, deviceSN string) error
	// ListByProduct 返回该产品下记录的全部不支持 path（含读/写标记）。
	ListByProduct(ctx context.Context, productID uuid.UUID) ([]UnsupportedPath, error)
}

// PgProductUnsupportedPathRepository 是 ProductUnsupportedPathRepository 的 PostgreSQL 实现。
type PgProductUnsupportedPathRepository struct {
	pool *pgxpool.Pool
}

// NewPgProductUnsupportedPathRepository 构造仓库。
func NewPgProductUnsupportedPathRepository(pool *pgxpool.Pool) *PgProductUnsupportedPathRepository {
	return &PgProductUnsupportedPathRepository{pool: pool}
}

func (r *PgProductUnsupportedPathRepository) Record(
	ctx context.Context, productID uuid.UUID, standardPath string,
	readUnsupported, writeUnsupported bool, faultCode int, deviceSN string,
) error {
	if productID == uuid.Nil || standardPath == "" || (!readUnsupported && !writeUnsupported) {
		return nil
	}
	query, args, err := sq.Insert("product_unsupported_paths").
		Columns("product_id", "firmware_version", "standard_path", "read_unsupported", "write_unsupported",
			"last_fault_code", "last_device_sn").
		Values(productID, "", standardPath, readUnsupported, writeUnsupported, faultCode, deviceSN).
		Suffix(`ON CONFLICT (product_id, firmware_version, standard_path) DO UPDATE SET
            read_unsupported  = product_unsupported_paths.read_unsupported  OR EXCLUDED.read_unsupported,
            write_unsupported = product_unsupported_paths.write_unsupported OR EXCLUDED.write_unsupported,
            last_fault_code   = EXCLUDED.last_fault_code,
            last_device_sn    = EXCLUDED.last_device_sn,
            hit_count         = product_unsupported_paths.hit_count + 1,
            last_seen_at      = now(),
            updated_at        = now()`).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return fmt.Errorf("build record unsupported path: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("record unsupported path: %w", err)
	}
	return nil
}

func (r *PgProductUnsupportedPathRepository) ListByProduct(
	ctx context.Context, productID uuid.UUID,
) ([]UnsupportedPath, error) {
	if productID == uuid.Nil {
		return nil, nil
	}
	query, args, err := sq.Select("standard_path", "bool_or(read_unsupported)", "bool_or(write_unsupported)").
		From("product_unsupported_paths").
		Where(sq.Eq{"product_id": productID}).
		GroupBy("standard_path").OrderBy("standard_path").
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list unsupported paths: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list unsupported paths: %w", err)
	}
	defer rows.Close()

	var out []UnsupportedPath
	for rows.Next() {
		var u UnsupportedPath
		if err := rows.Scan(&u.Path, &u.ReadUnsupported, &u.WriteUnsupported); err != nil {
			return nil, fmt.Errorf("scan unsupported path: %w", err)
		}
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate unsupported paths: %w", err)
	}
	return out, nil
}
