package filemanager

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

var fileColumns = []string{
	"id", "file_name", "file_type", "file_size", "minio_path",
	"content_type", "uploader", "device_sn", "status", "description",
	"created_at", "updated_at",
}

// Compile-time interface check.
var _ FileRepository = (*PgFileRepository)(nil)

// PgFileRepository is a PostgreSQL implementation of FileRepository.
type PgFileRepository struct {
	pool *pgxpool.Pool
}

// NewPgFileRepository creates a new PgFileRepository.
func NewPgFileRepository(pool *pgxpool.Pool) *PgFileRepository {
	return &PgFileRepository{pool: pool}
}

func (r *PgFileRepository) Create(ctx context.Context, file *ManagedFile) error {
	query, args, err := psql.Insert("managed_files").
		Columns("file_name", "file_type", "file_size", "minio_path",
			"content_type", "uploader", "device_sn", "status", "description").
		Values(file.FileName, file.FileType, file.FileSize, file.MinIOPath,
			file.ContentType, file.Uploader, file.DeviceSN, file.Status, file.Description).
		Suffix("RETURNING " + joinColumns(fileColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert managed_file SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanFile(row)
	if err != nil {
		return fmt.Errorf("create managed_file: %w", err)
	}
	*file = *created
	return nil
}

func (r *PgFileRepository) GetByID(ctx context.Context, id uuid.UUID) (*ManagedFile, error) {
	query, args, err := psql.Select(fileColumns...).
		From("managed_files").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get managed_file SQL: %w", err)
	}

	file, err := scanFile(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get managed_file: %w", err)
	}
	return file, nil
}

func (r *PgFileRepository) Update(ctx context.Context, file *ManagedFile) error {
	query, args, err := psql.Update("managed_files").
		Set("file_name", file.FileName).
		Set("file_type", file.FileType).
		Set("file_size", file.FileSize).
		Set("minio_path", file.MinIOPath).
		Set("content_type", file.ContentType).
		Set("uploader", file.Uploader).
		Set("device_sn", file.DeviceSN).
		Set("status", file.Status).
		Set("description", file.Description).
		Where(sq.Eq{"id": file.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update managed_file SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update managed_file: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgFileRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := psql.Delete("managed_files").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete managed_file SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete managed_file: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgFileRepository) List(ctx context.Context, filter FileFilter) (*model.ListResponse[ManagedFile], error) {
	base := psql.Select(fileColumns...).From("managed_files")
	countBase := psql.Select("COUNT(*)").From("managed_files")

	if filter.FileType != nil {
		base = base.Where(sq.Eq{"file_type": *filter.FileType})
		countBase = countBase.Where(sq.Eq{"file_type": *filter.FileType})
	}
	if filter.DeviceSN != nil {
		base = base.Where(sq.Eq{"device_sn": *filter.DeviceSN})
		countBase = countBase.Where(sq.Eq{"device_sn": *filter.DeviceSN})
	}
	if filter.Status != nil {
		base = base.Where(sq.Eq{"status": *filter.Status})
		countBase = countBase.Where(sq.Eq{"status": *filter.Status})
	}
	if filter.Search != nil && *filter.Search != "" {
		like := "%" + *filter.Search + "%"
		base = base.Where(sq.ILike{"file_name": like})
		countBase = countBase.Where(sq.ILike{"file_name": like})
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count managed_file SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count managed_files: %w", err)
	}

	// Pagination
	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortDir := filter.SortDir
	if sortDir == "" {
		sortDir = "desc"
	}
	base = base.
		OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list managed_file SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list managed_files: %w", err)
	}
	defer rows.Close()

	var items []ManagedFile
	for rows.Next() {
		file, err := scanFileRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan managed_file row: %w", err)
		}
		items = append(items, *file)
	}

	if items == nil {
		items = []ManagedFile{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

// ---- scanning helpers ----

func scanFile(row pgx.Row) (*ManagedFile, error) {
	var f ManagedFile

	err := row.Scan(
		&f.ID, &f.FileName, &f.FileType, &f.FileSize, &f.MinIOPath,
		&f.ContentType, &f.Uploader, &f.DeviceSN, &f.Status, &f.Description,
		&f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func scanFileRow(rows pgx.Rows) (*ManagedFile, error) {
	var f ManagedFile

	err := rows.Scan(
		&f.ID, &f.FileName, &f.FileType, &f.FileSize, &f.MinIOPath,
		&f.ContentType, &f.Uploader, &f.DeviceSN, &f.Status, &f.Description,
		&f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// ---- shared helpers ----

func joinColumns(cols []string) string {
	result := ""
	for i, c := range cols {
		if i > 0 {
			result += ", "
		}
		result += c
	}
	return result
}
