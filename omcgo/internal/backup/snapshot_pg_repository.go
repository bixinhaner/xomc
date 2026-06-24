package backup

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

// snapshotColumns 是 SELECT 列表与 scan 顺序的单一事实源。
var snapshotColumns = []string{
	"serial_number",
	"enb_name",
	"product_type",
	"file_name",
	"file_ext",
	"object_bucket",
	"object_path",
	"md5",
	"file_size",
	"source",
	"source_task_id",
	"source_version",
	"update_by",
	"update_time",
	"created_at",
	"updated_at",
}

// PgSnapshotRepository 是 SnapshotRepository 的 PostgreSQL 实现。
type PgSnapshotRepository struct {
	pool *pgxpool.Pool
}

// NewPgSnapshotRepository 构造 PgSnapshotRepository。
func NewPgSnapshotRepository(pool *pgxpool.Pool) *PgSnapshotRepository {
	return &PgSnapshotRepository{pool: pool}
}

var _ SnapshotRepository = (*PgSnapshotRepository)(nil)

// Upsert: ON CONFLICT (serial_number) DO UPDATE —— 一设备一行，冲突覆盖。
// created_at 在冲突分支**不**被覆盖；updated_at / update_time 一律刷新到 NOW()。
func (r *PgSnapshotRepository) Upsert(ctx context.Context, snap *ConfigSnapshot) error {
	if snap == nil {
		return fmt.Errorf("nil ConfigSnapshot")
	}
	if snap.SerialNumber == "" {
		return ErrEmptySerialNumber
	}
	if snap.FileName == "" {
		return ErrEmptyFileName
	}
	if snap.FileExt != string(SnapshotExtXML) && snap.FileExt != string(SnapshotExtNV) {
		return fmt.Errorf("%w: file_ext=%q", ErrInvalidConfigFileExt, snap.FileExt)
	}
	if snap.Source != SnapshotSourceBackup && snap.Source != SnapshotSourceManualUpload {
		return fmt.Errorf("invalid snapshot source: %q", snap.Source)
	}
	if snap.ObjectBucket == "" || snap.ObjectPath == "" {
		return fmt.Errorf("object_bucket and object_path are required")
	}

	query, args, err := storage.Psql.Insert("config_snapshots").
		Columns(
			"serial_number", "enb_name", "product_type",
			"file_name", "file_ext", "object_bucket", "object_path",
			"md5", "file_size", "source", "source_task_id", "source_version", "update_by",
		).
		Values(
			snap.SerialNumber, snap.EnbName, snap.ProductType,
			snap.FileName, snap.FileExt, snap.ObjectBucket, snap.ObjectPath,
			snap.MD5, snap.FileSize, string(snap.Source), snap.SourceTaskID, snap.SourceVersion, snap.UpdateBy,
		).
		Suffix(`ON CONFLICT (serial_number) DO UPDATE SET
			enb_name       = EXCLUDED.enb_name,
			product_type   = EXCLUDED.product_type,
			file_name      = EXCLUDED.file_name,
			file_ext       = EXCLUDED.file_ext,
			object_bucket  = EXCLUDED.object_bucket,
			object_path    = EXCLUDED.object_path,
			md5            = EXCLUDED.md5,
			file_size      = EXCLUDED.file_size,
			source         = EXCLUDED.source,
			source_task_id = EXCLUDED.source_task_id,
			source_version = COALESCE(EXCLUDED.source_version, config_snapshots.source_version),
			update_by      = COALESCE(EXCLUDED.update_by, config_snapshots.update_by),
			update_time    = NOW(),
			updated_at     = NOW()
			RETURNING created_at, updated_at, update_time`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build config_snapshots upsert SQL: %w", err)
	}

	if scanErr := r.pool.QueryRow(ctx, query, args...).
		Scan(&snap.CreatedAt, &snap.UpdatedAt, &snap.UpdateTime); scanErr != nil {
		return fmt.Errorf("upsert config_snapshots: %w", scanErr)
	}
	return nil
}

// GetBySerialNumber 不存在时返回 (nil, nil)，与 Repository 注释保持一致。
func (r *PgSnapshotRepository) GetBySerialNumber(ctx context.Context, sn string) (*ConfigSnapshot, error) {
	if sn == "" {
		return nil, nil
	}
	query, args, err := storage.Psql.
		Select(snapshotColumns...).
		From("config_snapshots").
		Where(sq.Eq{"serial_number": sn}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build config_snapshots get SQL: %w", err)
	}
	snap, err := scanSnapshot(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get config_snapshot by sn=%s: %w", sn, err)
	}
	return snap, nil
}

// BatchGetBySerialNumbers：用 ANY($1) 一次查询，返回 map[SN]→snapshot。
// 缺失的 SN 不会出现在返回 map 中，调用方据此识别整批拒绝的依据。
func (r *PgSnapshotRepository) BatchGetBySerialNumbers(
	ctx context.Context, sns []string,
) (map[string]*ConfigSnapshot, error) {
	if len(sns) == 0 {
		return map[string]*ConfigSnapshot{}, nil
	}
	query, args, err := storage.Psql.
		Select(snapshotColumns...).
		From("config_snapshots").
		Where(sq.Eq{"serial_number": sns}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build config_snapshots batch-get SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("batch-get config_snapshots: %w", err)
	}
	defer rows.Close()

	out := make(map[string]*ConfigSnapshot, len(sns))
	for rows.Next() {
		snap, scanErr := scanSnapshotRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan config_snapshot row: %w", scanErr)
		}
		out[snap.SerialNumber] = snap
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate config_snapshots: %w", rowsErr)
	}
	return out, nil
}

// List 按过滤条件分页查询，返回 items + total。
func (r *PgSnapshotRepository) List(
	ctx context.Context, filter SnapshotFilter,
) ([]ConfigSnapshot, int64, error) {
	base := storage.Psql.Select(snapshotColumns...).From("config_snapshots")
	countBase := storage.Psql.Select("COUNT(*)").From("config_snapshots")

	applyFilter := func(q sq.SelectBuilder) sq.SelectBuilder {
		if filter.SerialNumber != "" {
			q = q.Where(sq.ILike{"serial_number": "%" + filter.SerialNumber + "%"})
		}
		if filter.EnbName != "" {
			q = q.Where(sq.ILike{"enb_name": "%" + filter.EnbName + "%"})
		}
		if len(filter.ProductTypes) > 0 {
			// #602：ProductTypes 是产品的 product_class 正则模式集合（如 ^FAP/BSQ7258L254$），
			// 需用 PostgreSQL 正则匹配符 ~ 比对设备实际上报的 product_type 值。
			q = q.Where(sq.Expr("product_type ~ ANY(?)", filter.ProductTypes))
		} else if filter.ProductType != "" {
			q = q.Where(sq.Eq{"product_type": filter.ProductType})
		}
		if filter.Source != nil {
			q = q.Where(sq.Eq{"source": string(*filter.Source)})
		}
		if filter.UpdatedAfter != nil {
			q = q.Where(sq.GtOrEq{"update_time": *filter.UpdatedAfter})
		}
		if filter.UpdatedBefore != nil {
			q = q.Where(sq.LtOrEq{"update_time": *filter.UpdatedBefore})
		}
		return q
	}
	base = applyFilter(base)
	countBase = applyFilter(countBase)

	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count config_snapshots SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count config_snapshots: %w", err)
	}

	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "update_time"
	}
	sortDir := filter.SortDir
	if sortDir == "" {
		sortDir = "desc"
	}
	// 白名单 sort_by，防止注入。
	switch sortBy {
	case "update_time", "created_at", "serial_number", "enb_name", "product_type":
	default:
		sortBy = "update_time"
	}

	base = base.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build list config_snapshots SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list config_snapshots: %w", err)
	}
	defer rows.Close()

	items := make([]ConfigSnapshot, 0)
	for rows.Next() {
		snap, scanErr := scanSnapshotRow(rows)
		if scanErr != nil {
			return nil, 0, fmt.Errorf("scan config_snapshot row: %w", scanErr)
		}
		items = append(items, *snap)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, 0, fmt.Errorf("iterate config_snapshots: %w", rowsErr)
	}
	return items, total, nil
}

// Delete 删除单 SN 行。不存在视为成功，幂等。
func (r *PgSnapshotRepository) Delete(ctx context.Context, sn string) error {
	if sn == "" {
		return ErrEmptySerialNumber
	}
	query, args, err := storage.Psql.
		Delete("config_snapshots").
		Where(sq.Eq{"serial_number": sn}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build config_snapshots delete SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("delete config_snapshots sn=%s: %w", sn, err)
	}
	return nil
}

// BatchDelete 删除多 SN 行，返回实际被删除的 SN 列表。
func (r *PgSnapshotRepository) BatchDelete(ctx context.Context, sns []string) ([]string, error) {
	if len(sns) == 0 {
		return nil, nil
	}
	query, args, err := storage.Psql.
		Delete("config_snapshots").
		Where(sq.Eq{"serial_number": sns}).
		Suffix("RETURNING serial_number").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build config_snapshots batch-delete SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("batch-delete config_snapshots: %w", err)
	}
	defer rows.Close()

	deleted := make([]string, 0, len(sns))
	for rows.Next() {
		var sn string
		if scanErr := rows.Scan(&sn); scanErr != nil {
			return nil, fmt.Errorf("scan deleted sn: %w", scanErr)
		}
		deleted = append(deleted, sn)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate deleted config_snapshots: %w", rowsErr)
	}
	return deleted, nil
}

// ─────────────────────────────────────────────────────────────────────────
// scan helpers
// ─────────────────────────────────────────────────────────────────────────

func scanSnapshot(row pgx.Row) (*ConfigSnapshot, error) {
	var s ConfigSnapshot
	var source string
	if err := row.Scan(
		&s.SerialNumber, &s.EnbName, &s.ProductType,
		&s.FileName, &s.FileExt, &s.ObjectBucket, &s.ObjectPath,
		&s.MD5, &s.FileSize, &source, &s.SourceTaskID, &s.SourceVersion, &s.UpdateBy,
		&s.UpdateTime, &s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		return nil, err
	}
	s.Source = SnapshotSource(source)
	return &s, nil
}

func scanSnapshotRow(rows pgx.Rows) (*ConfigSnapshot, error) {
	var s ConfigSnapshot
	var source string
	if err := rows.Scan(
		&s.SerialNumber, &s.EnbName, &s.ProductType,
		&s.FileName, &s.FileExt, &s.ObjectBucket, &s.ObjectPath,
		&s.MD5, &s.FileSize, &source, &s.SourceTaskID, &s.SourceVersion, &s.UpdateBy,
		&s.UpdateTime, &s.CreatedAt, &s.UpdatedAt,
	); err != nil {
		return nil, err
	}
	s.Source = SnapshotSource(source)
	return &s, nil
}
