package license

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// licenseEnforcementColumns is the full column list including the
// enforcement-extension columns added by migration 000043. Used by
// methods that need GracePeriodDays / CapacityAlertThresholds / etc.
var licenseEnforcementColumns = []string{
	"id", "license_name", "license_code", "product_name",
	"license_type", "status", "max_devices", "used_devices",
	"features", "issue_date", "expiry_date",
	"licensor", "device_type", "region", "notes",
	"created_at", "updated_at",
	"grace_period_days", "capacity_alert_thresholds",
	"last_capacity_alert_at", "last_capacity_alert_threshold",
}

// PostgreSQL unique_violation error code.
// See: https://www.postgresql.org/docs/current/errcodes-appendix.html
const pgUniqueViolation = "23505"

var licenseColumns = []string{
	"id", "license_name", "license_code", "product_name",
	"license_type", "status", "max_devices", "used_devices",
	"features", "issue_date", "expiry_date",
	"licensor", "device_type", "region", "notes",
	"created_at", "updated_at",
}

// ======================================================================
// PgLicenseRepository
// ======================================================================

var _ LicenseRepository = (*PgLicenseRepository)(nil)

// PgLicenseRepository is a PostgreSQL implementation of LicenseRepository.
type PgLicenseRepository struct {
	pool *pgxpool.Pool
}

// NewPgLicenseRepository creates a new PgLicenseRepository.
func NewPgLicenseRepository(pool *pgxpool.Pool) *PgLicenseRepository {
	return &PgLicenseRepository{pool: pool}
}

func (r *PgLicenseRepository) Create(ctx context.Context, lic *License) error {
	featuresJSON := lic.Features
	if featuresJSON == nil {
		featuresJSON = json.RawMessage("[]")
	}

	query, args, err := storage.Psql.Insert("licenses").
		Columns(
			"license_name", "license_code", "product_name",
			"license_type", "status", "max_devices", "used_devices",
			"features", "issue_date", "expiry_date",
			"licensor", "device_type", "region", "notes",
		).
		Values(
			lic.LicenseName, lic.LicenseCode, lic.ProductName,
			lic.LicenseType, lic.Status, lic.MaxDevices, lic.UsedDevices,
			featuresJSON, lic.IssueDate, lic.ExpiryDate,
			lic.Licensor, lic.DeviceType, lic.Region, lic.Notes,
		).
		Suffix("RETURNING " + joinColumns(licenseColumns)).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert license SQL: %w", err)
	}

	row := r.pool.QueryRow(ctx, query, args...)
	created, err := scanLicense(row)
	if err != nil {
		// Map PostgreSQL unique_violation (23505) to ErrAlreadyExists so the
		// HTTP layer can surface 409 instead of 500.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return fmt.Errorf("license already exists: %w", commonerrors.ErrAlreadyExists)
		}
		return fmt.Errorf("create license: %w", err)
	}
	*lic = *created
	return nil
}

func (r *PgLicenseRepository) GetByID(ctx context.Context, id uuid.UUID) (*License, error) {
	query, args, err := storage.Psql.Select(licenseColumns...).
		From("licenses").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get license SQL: %w", err)
	}

	lic, err := scanLicense(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get license: %w", err)
	}
	return lic, nil
}

func (r *PgLicenseRepository) GetByCode(ctx context.Context, code string) (*License, error) {
	query, args, err := storage.Psql.Select(licenseColumns...).
		From("licenses").
		Where(sq.Eq{"license_code": code}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get license by code SQL: %w", err)
	}

	lic, err := scanLicense(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get license by code: %w", err)
	}
	return lic, nil
}

func (r *PgLicenseRepository) Update(ctx context.Context, lic *License) error {
	featuresJSON := lic.Features
	if featuresJSON == nil {
		featuresJSON = json.RawMessage("[]")
	}

	query, args, err := storage.Psql.Update("licenses").
		Set("license_name", lic.LicenseName).
		Set("license_code", lic.LicenseCode).
		Set("product_name", lic.ProductName).
		Set("license_type", lic.LicenseType).
		Set("status", lic.Status).
		Set("max_devices", lic.MaxDevices).
		Set("used_devices", lic.UsedDevices).
		Set("features", featuresJSON).
		Set("issue_date", lic.IssueDate).
		Set("expiry_date", lic.ExpiryDate).
		Set("licensor", lic.Licensor).
		Set("device_type", lic.DeviceType).
		Set("region", lic.Region).
		Set("notes", lic.Notes).
		Where(sq.Eq{"id": lic.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update license SQL: %w", err)
	}

	result, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update license: %w", err)
	}
	if result.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgLicenseRepository) List(ctx context.Context, filter LicenseFilter) (*model.ListResponse[License], error) {
	base := storage.Psql.Select(licenseColumns...).From("licenses")
	countBase := storage.Psql.Select("COUNT(*)").From("licenses")

	if filter.Status != nil {
		base = base.Where(sq.Eq{"status": *filter.Status})
		countBase = countBase.Where(sq.Eq{"status": *filter.Status})
	}
	if filter.LicenseType != nil {
		base = base.Where(sq.Eq{"license_type": *filter.LicenseType})
		countBase = countBase.Where(sq.Eq{"license_type": *filter.LicenseType})
	}
	if filter.DeviceType != nil {
		base = base.Where(sq.Eq{"device_type": *filter.DeviceType})
		countBase = countBase.Where(sq.Eq{"device_type": *filter.DeviceType})
	}

	// Count total
	countSQL, countArgs, err := countBase.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count license SQL: %w", err)
	}
	var total int64
	if err := r.pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count licenses: %w", err)
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
		return nil, fmt.Errorf("build list license SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list licenses: %w", err)
	}
	defer rows.Close()

	var items []License
	for rows.Next() {
		lic, err := scanLicenseRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan license row: %w", err)
		}
		items = append(items, *lic)
	}

	if items == nil {
		items = []License{}
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func (r *PgLicenseRepository) Summary(ctx context.Context) (*LicenseSummary, error) {
	query := `SELECT COUNT(*) AS total,
		COUNT(*) FILTER (WHERE status = 'active') AS active,
		COUNT(*) FILTER (WHERE status = 'expired') AS expired,
		COUNT(*) FILTER (WHERE status = 'pending') AS pending,
		COUNT(*) FILTER (WHERE status = 'active' AND expiry_date IS NOT NULL AND expiry_date < NOW() + INTERVAL '30 days') AS expiring_soon
	FROM licenses`

	var s LicenseSummary
	err := r.pool.QueryRow(ctx, query).Scan(
		&s.Total, &s.Active, &s.Expired, &s.Pending, &s.ExpiringSoon,
	)
	if err != nil {
		return nil, fmt.Errorf("summary licenses: %w", err)
	}
	return &s, nil
}

// ---- scanning helpers ----

func scanLicense(row pgx.Row) (*License, error) {
	var l License
	var featuresJSON []byte

	err := row.Scan(
		&l.ID, &l.LicenseName, &l.LicenseCode, &l.ProductName,
		&l.LicenseType, &l.Status, &l.MaxDevices, &l.UsedDevices,
		&featuresJSON, &l.IssueDate, &l.ExpiryDate,
		&l.Licensor, &l.DeviceType, &l.Region, &l.Notes,
		&l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if featuresJSON != nil {
		l.Features = json.RawMessage(featuresJSON)
	}
	if l.Features == nil {
		l.Features = json.RawMessage("[]")
	}
	return &l, nil
}

func scanLicenseRow(rows pgx.Rows) (*License, error) {
	var l License
	var featuresJSON []byte

	err := rows.Scan(
		&l.ID, &l.LicenseName, &l.LicenseCode, &l.ProductName,
		&l.LicenseType, &l.Status, &l.MaxDevices, &l.UsedDevices,
		&featuresJSON, &l.IssueDate, &l.ExpiryDate,
		&l.Licensor, &l.DeviceType, &l.Region, &l.Notes,
		&l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	if featuresJSON != nil {
		l.Features = json.RawMessage(featuresJSON)
	}
	if l.Features == nil {
		l.Features = json.RawMessage("[]")
	}
	return &l, nil
}

// ======================================================================
// Enforcement methods (T-0015 / R-103)
// ======================================================================

// scanLicenseFull scans a row containing the full licenseEnforcementColumns
// (21 columns), populating the enforcement-extension fields.
func scanLicenseFull(row pgx.Row) (*License, error) {
	var l License
	var featuresJSON []byte
	var thresholdsJSON []byte

	err := row.Scan(
		&l.ID, &l.LicenseName, &l.LicenseCode, &l.ProductName,
		&l.LicenseType, &l.Status, &l.MaxDevices, &l.UsedDevices,
		&featuresJSON, &l.IssueDate, &l.ExpiryDate,
		&l.Licensor, &l.DeviceType, &l.Region, &l.Notes,
		&l.CreatedAt, &l.UpdatedAt,
		&l.GracePeriodDays, &thresholdsJSON,
		&l.LastCapacityAlertAt, &l.LastCapacityAlertThreshold,
	)
	if err != nil {
		return nil, err
	}
	if featuresJSON != nil {
		l.Features = json.RawMessage(featuresJSON)
	}
	if l.Features == nil {
		l.Features = json.RawMessage("[]")
	}
	if thresholdsJSON != nil {
		l.CapacityAlertThresholds = json.RawMessage(thresholdsJSON)
	}
	if l.CapacityAlertThresholds == nil {
		l.CapacityAlertThresholds = json.RawMessage("[80, 90, 95]")
	}
	return &l, nil
}

// GetActiveLicenseWithMaxDevices returns the active license with the largest
// MaxDevices, the canonical "enforcement" license when multiple actives
// exist. Returns (nil, nil) if no active license exists.
func (r *PgLicenseRepository) GetActiveLicenseWithMaxDevices(ctx context.Context) (*License, error) {
	query, args, err := storage.Psql.Select(licenseEnforcementColumns...).
		From("licenses").
		Where(sq.Eq{"status": StatusActive}).
		OrderBy("max_devices DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build active license query: %w", err)
	}

	lic, err := scanLicenseFull(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get active license with max devices: %w", err)
	}
	return lic, nil
}

// ListActiveLicenses returns all licenses currently in active status.
func (r *PgLicenseRepository) ListActiveLicenses(ctx context.Context) ([]*License, error) {
	query, args, err := storage.Psql.Select(licenseEnforcementColumns...).
		From("licenses").
		Where(sq.Eq{"status": StatusActive}).
		OrderBy("max_devices DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list active licenses query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list active licenses: %w", err)
	}
	defer rows.Close()

	var out []*License
	for rows.Next() {
		lic, err := scanLicenseFull(rows)
		if err != nil {
			return nil, fmt.Errorf("scan active license: %w", err)
		}
		out = append(out, lic)
	}
	return out, nil
}

// ListActiveByDimension returns active licenses with the same (device_type,
// region) dimension. NULL is matched against NULL (业务约束：同维度最多一个
// active；NULL device_type/region 视为同一"未指定"维度桶)。
//
// T-0100-P3：Activate 同维度冲突检测，用于 force=false 时返回 409 + 冲突列表，
// 或 force=true 时定位待自动 revoke 的旧 license。
func (r *PgLicenseRepository) ListActiveByDimension(
	ctx context.Context, deviceType *string, region *string,
) ([]*License, error) {
	q := storage.Psql.Select(licenseEnforcementColumns...).
		From("licenses").
		Where(sq.Eq{"status": StatusActive})

	if deviceType == nil {
		q = q.Where("device_type IS NULL")
	} else {
		q = q.Where(sq.Eq{"device_type": *deviceType})
	}
	if region == nil {
		q = q.Where("region IS NULL")
	} else {
		q = q.Where(sq.Eq{"region": *region})
	}

	query, args, err := q.OrderBy("created_at DESC").ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list active by dimension query: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list active by dimension: %w", err)
	}
	defer rows.Close()

	var out []*License
	for rows.Next() {
		lic, err := scanLicenseFull(rows)
		if err != nil {
			return nil, fmt.Errorf("scan active license: %w", err)
		}
		out = append(out, lic)
	}
	return out, nil
}

// CountDevices returns the total number of registered devices.
// Used as the "used_devices" measurement for capacity enforcement.
func (r *PgLicenseRepository) CountDevices(ctx context.Context) (int, error) {
	var count int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM devices").Scan(&count); err != nil {
		return 0, fmt.Errorf("count devices: %w", err)
	}
	return count, nil
}

// MarkExpired transitions a license from active to expired. Idempotent.
func (r *PgLicenseRepository) MarkExpired(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Update("licenses").
		Set("status", StatusExpired).
		Where(sq.Eq{"id": id}).
		Where(sq.Eq{"status": StatusActive}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build mark expired SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("mark license expired: %w", err)
	}
	return nil
}

// UpdateCapacityAlert records the most recent capacity alert (threshold and
// timestamp) so the hourly checker can dedupe within a 6h window.
func (r *PgLicenseRepository) UpdateCapacityAlert(ctx context.Context, id uuid.UUID, threshold int, at time.Time) error {
	query, args, err := storage.Psql.Update("licenses").
		Set("last_capacity_alert_at", at).
		Set("last_capacity_alert_threshold", threshold).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update capacity alert SQL: %w", err)
	}
	if _, err := r.pool.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("update capacity alert: %w", err)
	}
	return nil
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
