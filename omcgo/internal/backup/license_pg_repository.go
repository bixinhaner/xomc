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

// licenseColumns 是 SELECT 列表与 scan 顺序的单一事实源。
var licenseColumns = []string{
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
	"description",
	"auto_dispatch_pending",
	"update_by",
	"update_time",
	"created_at",
	"updated_at",
}

// PgLicenseRepository 是 LicenseRepository 的 PostgreSQL 实现。
type PgLicenseRepository struct {
	pool *pgxpool.Pool
}

func NewPgLicenseRepository(pool *pgxpool.Pool) *PgLicenseRepository {
	return &PgLicenseRepository{pool: pool}
}

var _ LicenseRepository = (*PgLicenseRepository)(nil)

func (r *PgLicenseRepository) Upsert(ctx context.Context, lic *DeviceLicense) error {
	if lic == nil {
		return fmt.Errorf("nil DeviceLicense")
	}
	if lic.SerialNumber == "" {
		return fmt.Errorf("serial_number 不能为空")
	}
	if lic.FileName == "" {
		return fmt.Errorf("file_name 不能为空")
	}
	if lic.FileExt != string(LicenseExtLIC) {
		return fmt.Errorf("file_ext 必须是 lic: %q", lic.FileExt)
	}
	if lic.ObjectBucket == "" || lic.ObjectPath == "" {
		return fmt.Errorf("object_bucket 和 object_path 必填")
	}
	if lic.Source == "" {
		lic.Source = LicenseSourceManualUpload
	}
	lic.AutoDispatchPending = true

	query, args, err := storage.Psql.Insert("device_licenses").
		Columns(
			"serial_number", "enb_name", "product_type",
			"file_name", "file_ext", "object_bucket", "object_path",
			"md5", "file_size", "source", "description", "update_by",
		).
		Values(
			lic.SerialNumber, lic.EnbName, lic.ProductType,
			lic.FileName, lic.FileExt, lic.ObjectBucket, lic.ObjectPath,
			lic.MD5, lic.FileSize, string(lic.Source), lic.Description, lic.UpdateBy,
		).
		Suffix(`ON CONFLICT (serial_number) DO UPDATE SET
			enb_name      = EXCLUDED.enb_name,
			product_type  = EXCLUDED.product_type,
			file_name     = EXCLUDED.file_name,
			file_ext      = EXCLUDED.file_ext,
			object_bucket = EXCLUDED.object_bucket,
			object_path   = EXCLUDED.object_path,
			md5           = EXCLUDED.md5,
			file_size     = EXCLUDED.file_size,
			source        = EXCLUDED.source,
			description   = COALESCE(EXCLUDED.description, device_licenses.description),
			auto_dispatch_pending = TRUE,
			update_by     = COALESCE(EXCLUDED.update_by, device_licenses.update_by),
			update_time   = NOW(),
			updated_at    = NOW()
			RETURNING created_at, updated_at, update_time`).
		ToSql()
	if err != nil {
		return fmt.Errorf("build device_licenses upsert SQL: %w", err)
	}
	if scanErr := r.pool.QueryRow(ctx, query, args...).
		Scan(&lic.CreatedAt, &lic.UpdatedAt, &lic.UpdateTime); scanErr != nil {
		return fmt.Errorf("upsert device_licenses: %w", scanErr)
	}
	return nil
}

// ClaimAutoDispatch atomically claims one pending preinstall. Registered and
// online events can arrive together, so a read-then-update sequence would
// enqueue the same license twice.
func (r *PgLicenseRepository) ClaimAutoDispatch(ctx context.Context, sn string) (bool, error) {
	query, args, err := storage.Psql.Update("device_licenses").
		Set("auto_dispatch_pending", false).
		Where(sq.Eq{"serial_number": sn, "auto_dispatch_pending": true}).
		Suffix("RETURNING serial_number").ToSql()
	if err != nil {
		return false, fmt.Errorf("build device_licenses dispatch claim SQL: %w", err)
	}
	var claimedSN string
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&claimedSN); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("claim device_license auto dispatch sn=%s: %w", sn, err)
	}
	return true, nil
}

func (r *PgLicenseRepository) ReleaseAutoDispatch(ctx context.Context, sn string) error {
	query, args, err := storage.Psql.Update("device_licenses").
		Set("auto_dispatch_pending", true).
		Where(sq.Eq{"serial_number": sn}).ToSql()
	if err != nil {
		return fmt.Errorf("build device_licenses dispatch release SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("release device_license auto dispatch sn=%s: %w", sn, err)
	}
	return nil
}

func (r *PgLicenseRepository) GetBySerialNumber(ctx context.Context, sn string) (*DeviceLicense, error) {
	if sn == "" {
		return nil, nil
	}
	query, args, err := storage.Psql.
		Select(licenseColumns...).From("device_licenses").
		Where(sq.Eq{"serial_number": sn}).Limit(1).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build device_licenses get SQL: %w", err)
	}
	lic, err := scanLicense(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get device_license sn=%s: %w", sn, err)
	}
	return lic, nil
}

func (r *PgLicenseRepository) BatchGetBySerialNumbers(
	ctx context.Context, sns []string,
) (map[string]*DeviceLicense, error) {
	if len(sns) == 0 {
		return map[string]*DeviceLicense{}, nil
	}
	query, args, err := storage.Psql.
		Select(licenseColumns...).From("device_licenses").
		Where(sq.Eq{"serial_number": sns}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build device_licenses batch-get SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("batch-get device_licenses: %w", err)
	}
	defer rows.Close()
	out := make(map[string]*DeviceLicense, len(sns))
	for rows.Next() {
		lic, scanErr := scanLicenseRow(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan device_license row: %w", scanErr)
		}
		out[lic.SerialNumber] = lic
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("iterate device_licenses: %w", rowsErr)
	}
	return out, nil
}

func (r *PgLicenseRepository) List(
	ctx context.Context, filter LicenseFilter,
) ([]DeviceLicense, int64, error) {
	base := storage.Psql.Select(licenseColumns...).From("device_licenses")
	countBase := storage.Psql.Select("COUNT(*)").From("device_licenses")
	apply := func(q sq.SelectBuilder) sq.SelectBuilder {
		if filter.SerialNumber != "" {
			q = q.Where(sq.ILike{"serial_number": "%" + filter.SerialNumber + "%"})
		}
		if filter.EnbName != "" {
			q = q.Where(sq.ILike{"enb_name": "%" + filter.EnbName + "%"})
		}
		if len(filter.ProductTypes) > 0 {
			// #602：ProductTypes 是产品的 product_class 正则模式集合，需用 PostgreSQL ~ ANY 正则匹配。
			q = q.Where(sq.Expr("product_type ~ ANY(?)", filter.ProductTypes))
		} else if filter.ProductType != "" {
			q = q.Where(sq.Eq{"product_type": filter.ProductType})
		}
		if filter.UpdatedAfter != nil {
			q = q.Where(sq.GtOrEq{"update_time": *filter.UpdatedAfter})
		}
		if filter.UpdatedBefore != nil {
			q = q.Where(sq.LtOrEq{"update_time": *filter.UpdatedBefore})
		}
		return q
	}
	base = apply(base)
	countBase = apply(countBase)

	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count device_licenses SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count device_licenses: %w", err)
	}

	sortBy := filter.SortBy
	if sortBy == "" {
		sortBy = "update_time"
	}
	sortDir := filter.SortDir
	if sortDir == "" {
		sortDir = "desc"
	}
	switch sortBy {
	case "update_time", "created_at", "serial_number", "enb_name", "product_type":
	default:
		sortBy = "update_time"
	}
	base = base.OrderBy(sortBy + " " + sortDir).
		Limit(uint64(filter.Limit())).Offset(uint64(filter.Offset()))

	query, args, err := base.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build list device_licenses SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list device_licenses: %w", err)
	}
	defer rows.Close()
	items := make([]DeviceLicense, 0)
	for rows.Next() {
		lic, scanErr := scanLicenseRow(rows)
		if scanErr != nil {
			return nil, 0, fmt.Errorf("scan device_license row: %w", scanErr)
		}
		items = append(items, *lic)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, 0, fmt.Errorf("iterate device_licenses: %w", rowsErr)
	}
	return items, total, nil
}

func (r *PgLicenseRepository) Delete(ctx context.Context, sn string) error {
	if sn == "" {
		return fmt.Errorf("serial_number 不能为空")
	}
	query, args, err := storage.Psql.Delete("device_licenses").
		Where(sq.Eq{"serial_number": sn}).ToSql()
	if err != nil {
		return fmt.Errorf("build device_licenses delete SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("delete device_licenses sn=%s: %w", sn, err)
	}
	return nil
}

func (r *PgLicenseRepository) BatchDelete(ctx context.Context, sns []string) ([]string, error) {
	if len(sns) == 0 {
		return nil, nil
	}
	query, args, err := storage.Psql.Delete("device_licenses").
		Where(sq.Eq{"serial_number": sns}).
		Suffix("RETURNING serial_number").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build device_licenses batch-delete SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("batch-delete device_licenses: %w", err)
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
		return nil, fmt.Errorf("iterate deleted device_licenses: %w", rowsErr)
	}
	return deleted, nil
}

func scanLicense(row pgx.Row) (*DeviceLicense, error) {
	var l DeviceLicense
	var source string
	if err := row.Scan(
		&l.SerialNumber, &l.EnbName, &l.ProductType,
		&l.FileName, &l.FileExt, &l.ObjectBucket, &l.ObjectPath,
		&l.MD5, &l.FileSize, &source, &l.Description, &l.AutoDispatchPending, &l.UpdateBy,
		&l.UpdateTime, &l.CreatedAt, &l.UpdatedAt,
	); err != nil {
		return nil, err
	}
	l.Source = LicenseSource(source)
	return &l, nil
}

func scanLicenseRow(rows pgx.Rows) (*DeviceLicense, error) {
	var l DeviceLicense
	var source string
	if err := rows.Scan(
		&l.SerialNumber, &l.EnbName, &l.ProductType,
		&l.FileName, &l.FileExt, &l.ObjectBucket, &l.ObjectPath,
		&l.MD5, &l.FileSize, &source, &l.Description, &l.AutoDispatchPending, &l.UpdateBy,
		&l.UpdateTime, &l.CreatedAt, &l.UpdatedAt,
	); err != nil {
		return nil, err
	}
	l.Source = LicenseSource(source)
	return &l, nil
}
