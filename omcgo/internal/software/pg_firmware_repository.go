package software

import (
	"context"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

var firmwareColumns = []string{
	"id", "carrier", "product_class", "version", "file_name", "file_size",
	"minio_path", "compatible_oui", "release_notes", "status",
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
	err := row.Scan(
		&fw.ID, &fw.Carrier, &fw.ProductClass, &fw.Version,
		&fw.FileName, &fw.FileSize, &fw.MinIOPath, &ouiJSON,
		&fw.ReleaseNotes, &fw.Status, &fw.CreatedAt, &fw.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if ouiJSON != nil {
		_ = json.Unmarshal(ouiJSON, &fw.CompatibleOUI)
	}
	return &fw, nil
}

func (r *PgFirmwareRepository) Create(ctx context.Context, fw *FirmwareVersion) error {
	ouiJSON, _ := json.Marshal(fw.CompatibleOUI)

	query, args, err := storage.Psql.Insert("firmware_versions").
		Columns("carrier", "product_class", "version", "file_name", "file_size",
			"minio_path", "compatible_oui", "release_notes", "status").
		Values(fw.Carrier, fw.ProductClass, fw.Version, fw.FileName, fw.FileSize,
			fw.MinIOPath, ouiJSON, fw.ReleaseNotes, fw.Status).
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

	if filter.Carrier != nil {
		base = base.Where(sq.Eq{"carrier": *filter.Carrier})
		countBase = countBase.Where(sq.Eq{"carrier": *filter.Carrier})
	}
	if filter.ProductClass != nil {
		base = base.Where(sq.Eq{"product_class": *filter.ProductClass})
		countBase = countBase.Where(sq.Eq{"product_class": *filter.ProductClass})
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
		var fw FirmwareVersion
		var ouiJSON []byte
		err := rows.Scan(
			&fw.ID, &fw.Carrier, &fw.ProductClass, &fw.Version,
			&fw.FileName, &fw.FileSize, &fw.MinIOPath, &ouiJSON,
			&fw.ReleaseNotes, &fw.Status, &fw.CreatedAt, &fw.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan firmware row: %w", err)
		}
		if ouiJSON != nil {
			_ = json.Unmarshal(ouiJSON, &fw.CompatibleOUI)
		}
		items = append(items, fw)
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
