package imsparam

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/internal/core/storage"
)

// paramFileColumns 是 SELECT 列表与 scan 顺序的单一事实源。
var paramFileColumns = []string{
	"id",
	"param_type",
	"file_name",
	"object_bucket",
	"object_path",
	"md5",
	"file_size",
	"description",
	"uploaded_by",
	"device_sn",
	"created_at",
	"updated_at",
}

// Repository 是参数文件库的持久化接口。
type Repository interface {
	// Upsert 按 (param_type, file_name, device_sn) upsert；手动上传的空 SN 归为同一组。
	// 行 ID 保持稳定（下发任务引用的历史 ID 继续有效）。
	Upsert(ctx context.Context, file *ParamFile) error
	GetByID(ctx context.Context, id uuid.UUID) (*ParamFile, error)
	List(ctx context.Context, filter ParamFileFilter) ([]ParamFile, int64, error)
	Delete(ctx context.Context, id uuid.UUID) error
	BatchDelete(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error)
}

// PgRepository 是 Repository 的 PostgreSQL 实现（squirrel + pgx，禁 ORM）。
type PgRepository struct {
	pool *pgxpool.Pool
}

func NewPgRepository(pool *pgxpool.Pool) *PgRepository {
	return &PgRepository{pool: pool}
}

var _ Repository = (*PgRepository)(nil)

func (r *PgRepository) Upsert(ctx context.Context, file *ParamFile) error {
	if file == nil {
		return fmt.Errorf("nil ParamFile")
	}
	if file.ParamType == "" || file.FileName == "" {
		return fmt.Errorf("param_type 和 file_name 必填")
	}
	if file.ObjectBucket == "" || file.ObjectPath == "" {
		return fmt.Errorf("object_bucket 和 object_path 必填")
	}
	query, args, err := storage.Psql.Insert("ims_param_files").
		Columns(
			"param_type", "file_name", "object_bucket", "object_path",
			"md5", "file_size", "description", "uploaded_by", "device_sn",
		).
		Values(
			file.ParamType, file.FileName, file.ObjectBucket, file.ObjectPath,
			file.MD5, file.FileSize, file.Description, file.UploadedBy, nullIfEmpty(file.DeviceSN),
		).
		Suffix(`ON CONFLICT (param_type, file_name, COALESCE(device_sn, '')) DO UPDATE SET
			object_bucket = EXCLUDED.object_bucket,
			object_path   = EXCLUDED.object_path,
			md5           = EXCLUDED.md5,
			file_size     = EXCLUDED.file_size,
			description   = COALESCE(EXCLUDED.description, ims_param_files.description),
			uploaded_by   = COALESCE(EXCLUDED.uploaded_by, ims_param_files.uploaded_by),
			device_sn     = COALESCE(EXCLUDED.device_sn, ims_param_files.device_sn),
			updated_at    = NOW()
			RETURNING id, created_at, updated_at`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build ims_param_files upsert SQL: %w", err)
	}
	if err := r.pool.QueryRow(ctx, query, args...).
		Scan(&file.ID, &file.CreatedAt, &file.UpdatedAt); err != nil {
		return fmt.Errorf("upsert ims_param_files: %w", err)
	}
	return nil
}

func (r *PgRepository) GetByID(ctx context.Context, id uuid.UUID) (*ParamFile, error) {
	if id == uuid.Nil {
		return nil, nil
	}
	query, args, err := storage.Psql.
		Select(paramFileColumns...).From("ims_param_files").
		Where(sq.Eq{"id": id}).Limit(1).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build ims_param_files get SQL: %w", err)
	}
	file, err := scanParamFile(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get ims_param_files id=%s: %w", id, err)
	}
	return file, nil
}

func (r *PgRepository) List(ctx context.Context, filter ParamFileFilter) ([]ParamFile, int64, error) {
	base := storage.Psql.Select(paramFileColumns...).From("ims_param_files")
	countBase := storage.Psql.Select("COUNT(*)").From("ims_param_files")
	apply := func(q sq.SelectBuilder) sq.SelectBuilder {
		if filter.ParamType != "" {
			q = q.Where(sq.Eq{"param_type": NormalizeParamType(filter.ParamType)})
		}
		if filter.FileName != "" {
			q = q.Where(sq.ILike{"file_name": "%" + filter.FileName + "%"})
		}
		if filter.UploadedBy != "" {
			q = q.Where(sq.ILike{"uploaded_by": "%" + filter.UploadedBy + "%"})
		}
		if filter.DeviceSN != "" {
			q = q.Where(sq.ILike{"device_sn": "%" + filter.DeviceSN + "%"})
		}
		return q
	}
	base = apply(base)
	countBase = apply(countBase)

	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count ims_param_files SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count ims_param_files: %w", err)
	}

	sortBy := filter.SortBy
	switch sortBy {
	case "created_at", "updated_at", "file_name", "param_type":
	default:
		sortBy = "created_at"
	}
	sortDir := filter.SortDir
	if sortDir != "asc" {
		sortDir = "desc"
	}
	base = base.OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build list ims_param_files SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list ims_param_files: %w", err)
	}
	defer rows.Close()
	items := make([]ParamFile, 0)
	for rows.Next() {
		file, scanErr := scanParamFile(rows)
		if scanErr != nil {
			return nil, 0, fmt.Errorf("scan ims_param_files row: %w", scanErr)
		}
		items = append(items, *file)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, 0, fmt.Errorf("iterate ims_param_files: %w", rowsErr)
	}
	return items, total, nil
}

func (r *PgRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Delete("ims_param_files").
		Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("build ims_param_files delete SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("delete ims_param_files id=%s: %w", id, err)
	}
	return nil
}

func (r *PgRepository) BatchDelete(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	query, args, err := storage.Psql.Delete("ims_param_files").
		Where(sq.Eq{"id": ids}).
		Suffix("RETURNING id").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build ims_param_files batch-delete SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("batch-delete ims_param_files: %w", err)
	}
	defer rows.Close()
	deleted := make([]uuid.UUID, 0, len(ids))
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, fmt.Errorf("scan deleted ims_param_files id: %w", scanErr)
		}
		deleted = append(deleted, id)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate deleted ims_param_files: %w", rowsErr)
	}
	return deleted, nil
}

func scanParamFile(row pgx.Row) (*ParamFile, error) {
	var f ParamFile
	var deviceSN *string
	if err := row.Scan(
		&f.ID, &f.ParamType, &f.FileName, &f.ObjectBucket, &f.ObjectPath,
		&f.MD5, &f.FileSize, &f.Description, &f.UploadedBy,
		&deviceSN,
		&f.CreatedAt, &f.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if deviceSN != nil {
		f.DeviceSN = *deviceSN
	}
	return &f, nil
}

func nullIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
