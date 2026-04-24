package software

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

var firmwareColumns = []string{
	"id", "product_class", "version", "file_name", "file_size",
	"file_type", "minio_path", "compatible_oui", "md5_val", "recommend",
	"uploader", "manufacturer", "release_notes", "description", "status",
	"created_at", "updated_at",
}

var _ FirmwareRepository = (*PgFirmwareRepository)(nil)

// PgFirmwareRepository is a PostgreSQL implementation of FirmwareRepository.
type PgFirmwareRepository struct {
	pool *pgxpool.Pool
}

// NewPgFirmwareRepository creates a new PgFirmwareRepository.
func NewPgFirmwareRepository(pool *pgxpool.Pool) *PgFirmwareRepository {
	return &PgFirmwareRepository{pool: pool}
}

func scanFirmware(row pgx.Row) (*FirmwareVersion, error) {
	var fw FirmwareVersion
	var ouiJSON []byte
	var md5Val, uploader, manufacturer, description sqlNilString
	var recommend sqlNilBool
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&fw.ID, &fw.ProductClass, &fw.Version,
		&fw.FileName, &fw.FileSize, &fw.FileType, &fw.MinIOPath,
		&ouiJSON, &md5Val, &recommend, &uploader,
		&manufacturer, &fw.ReleaseNotes, &description, &fw.Status,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	if ouiJSON != nil {
		_ = json.Unmarshal(ouiJSON, &fw.CompatibleOUI)
	}
	fw.MD5Val = md5Val.string
	fw.Recommend = recommend.bool
	fw.Uploader = uploader.string
	fw.Manufacturer = manufacturer.string
	fw.Description = description.string
	fw.CreatedAt = JSONTime(createdAt)
	fw.UpdatedAt = JSONTime(updatedAt)
	return &fw, nil
}

func (r *PgFirmwareRepository) Create(ctx context.Context, fw *FirmwareVersion) error {
	ouiJSON, _ := json.Marshal(fw.CompatibleOUI)

	query, args, err := storage.Psql.Insert("firmware_versions").
		Columns("product_class", "version", "file_name", "file_size",
			"file_type", "minio_path", "compatible_oui", "md5_val", "recommend",
			"uploader", "manufacturer", "release_notes", "description", "status").
		Values(fw.ProductClass, fw.Version, fw.FileName, fw.FileSize,
			fw.FileType, fw.MinIOPath, ouiJSON, fw.MD5Val, fw.Recommend,
			fw.Uploader, fw.Manufacturer, fw.ReleaseNotes, fw.Description, fw.Status).
		Suffix("RETURNING " + joinColumns(firmwareColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert firmware SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanFirmware(row)
	if err != nil {
		return fmt.Errorf("create firmware: %w", err)
	}
	*fw = *created
	return nil
}

func (r *PgFirmwareRepository) GetByID(ctx context.Context, id uuid.UUID) (*FirmwareVersion, error) {
	query, args, err := storage.Psql.Select(firmwareColumns...).
		From("firmware_versions").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get firmware SQL: %w", err)
	}

	fw, err := scanFirmware(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get firmware: %w", err)
	}
	return fw, nil
}

func (r *PgFirmwareRepository) List(ctx context.Context, filter FirmwareFilter) (*model.ListResponse[FirmwareVersion], error) {
	base := storage.Psql.Select(firmwareColumns...).From("firmware_versions")
	countBase := storage.Psql.Select("COUNT(*)").From("firmware_versions")

	if filter.ProductClass != nil {
		base = base.Where(sq.Eq{"product_class": *filter.ProductClass})
		countBase = countBase.Where(sq.Eq{"product_class": *filter.ProductClass})
	}
	if filter.FileType != nil {
		base = base.Where(sq.Eq{"file_type": *filter.FileType})
		countBase = countBase.Where(sq.Eq{"file_type": *filter.FileType})
	}

	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count firmware SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count firmware: %w", err)
	}

	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	page := filter.Page
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * pageSize

	query, args, err := base.
		OrderBy("created_at DESC").
		Limit(uint64(pageSize)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list firmware SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list firmware: %w", err)
	}
	defer rows.Close()

	var items []FirmwareVersion
	for rows.Next() {
		fw, err := scanFirmwareRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan firmware row: %w", err)
		}
		items = append(items, *fw)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &model.ListResponse[FirmwareVersion]{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

func (r *PgFirmwareRepository) Update(ctx context.Context, fw *FirmwareVersion) error {
	builder := storage.Psql.Update("firmware_versions").
		Set("product_class", fw.ProductClass).
		Set("version", fw.Version).
		Set("recommend", fw.Recommend).
		Set("description", fw.Description).
		Set("release_notes", fw.ReleaseNotes).
		Set("manufacturer", fw.Manufacturer).
		Set("status", fw.Status).
		Where(sq.Eq{"id": fw.ID})

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build update firmware SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update firmware: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgFirmwareRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Delete("firmware_versions").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete firmware SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete firmware: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func scanFirmwareRow(rows pgx.Rows) (*FirmwareVersion, error) {
	var fw FirmwareVersion
	var ouiJSON []byte
	var md5Val, uploader, manufacturer, description sqlNilString
	var recommend sqlNilBool
	var createdAt, updatedAt time.Time

	err := rows.Scan(
		&fw.ID, &fw.ProductClass, &fw.Version,
		&fw.FileName, &fw.FileSize, &fw.FileType, &fw.MinIOPath,
		&ouiJSON, &md5Val, &recommend, &uploader,
		&manufacturer, &fw.ReleaseNotes, &description, &fw.Status,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	if ouiJSON != nil {
		_ = json.Unmarshal(ouiJSON, &fw.CompatibleOUI)
	}
	fw.MD5Val = md5Val.string
	fw.Recommend = recommend.bool
	fw.Uploader = uploader.string
	fw.Manufacturer = manufacturer.string
	fw.Description = description.string
	fw.CreatedAt = JSONTime(createdAt)
	fw.UpdatedAt = JSONTime(updatedAt)
	return &fw, nil
}

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

// sqlNilString wraps nullable string scanning for firmware fields.
type sqlNilString struct {
	string
}

func (s *sqlNilString) Scan(value interface{}) error {
	if value == nil {
		s.string = ""
		return nil
	}
	switch v := value.(type) {
	case string:
		s.string = v
	case []byte:
		s.string = string(v)
	}
	return nil
}

// sqlNilBool wraps nullable bool scanning.
type sqlNilBool struct {
	bool
}

func (b *sqlNilBool) Scan(value interface{}) error {
	if value == nil {
		b.bool = false
		return nil
	}
	switch v := value.(type) {
	case bool:
		b.bool = v
	}
	return nil
}
