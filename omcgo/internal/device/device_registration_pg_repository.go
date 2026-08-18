package device

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/omcgo/omcgo/global"
	"github.com/omcgo/omcgo/internal/authz"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

var regColumns = []string{
	"id", "serial_number", "group_id", "carrier", "site_name",
	"longitude", "latitude", "status", "remark", "created_by",
	"created_at", "updated_at",
}

// PgRegistrationRepository implements RegistrationRepository using PostgreSQL.
type PgRegistrationRepository struct {
	pool *pgxpool.Pool
}

var ErrRegistrationOwnershipConflict = errors.New("registration ownership conflict")

// EnsureCandidateOwnershipTx establishes the pre-registration evidence used by
// device access control. It intentionally accepts the caller transaction so
// candidate review, allow-list mutation and reevaluation outbox remain atomic.
func (r *PgRegistrationRepository) EnsureCandidateOwnershipTx(
	ctx context.Context,
	tx pgx.Tx,
	reg *DeviceRegistration,
) error {
	if tx == nil {
		return fmt.Errorf("ensure candidate ownership: transaction is required")
	}
	if reg == nil || strings.TrimSpace(reg.SerialNumber) == "" || strings.TrimSpace(string(reg.Carrier)) == "" || reg.GroupID == nil {
		return fmt.Errorf("ensure candidate ownership: serial number, carrier and group are required")
	}

	conflictQuery, conflictArgs, err := storage.Psql.Select(`EXISTS (
		SELECT 1 FROM devices d, requested_identity requested
		WHERE d.serial_number = requested.serial_number AND d.carrier <> requested.carrier AND d.deleted_at IS NULL
	) OR EXISTS (
		SELECT 1 FROM device_registrations dr, requested_identity requested
		WHERE dr.serial_number = requested.serial_number AND dr.carrier <> requested.carrier
	)`).Prefix(
		"WITH requested_identity AS (SELECT ?::varchar AS serial_number, ?::varchar AS carrier)",
		reg.SerialNumber, string(reg.Carrier),
	).ToSql()
	if err != nil {
		return fmt.Errorf("build candidate ownership conflict query: %w", err)
	}
	var conflict bool
	if err := tx.QueryRow(ctx, conflictQuery, conflictArgs...).Scan(&conflict); err != nil {
		return fmt.Errorf("check candidate ownership conflict: %w", err)
	}
	if conflict {
		return fmt.Errorf("serial number belongs to another carrier: %w", ErrRegistrationOwnershipConflict)
	}

	existingDeviceQuery, existingDeviceArgs, err := storage.Psql.Select(`EXISTS (
		SELECT 1 FROM devices d, requested_identity requested
		WHERE d.serial_number = requested.serial_number AND d.carrier = requested.carrier AND d.deleted_at IS NULL
	)`).Prefix(
		"WITH requested_identity AS (SELECT ?::varchar AS serial_number, ?::varchar AS carrier)",
		reg.SerialNumber, string(reg.Carrier),
	).ToSql()
	if err != nil {
		return fmt.Errorf("build candidate ownership device query: %w", err)
	}
	var deviceExists bool
	if err := tx.QueryRow(ctx, existingDeviceQuery, existingDeviceArgs...).Scan(&deviceExists); err != nil {
		return fmt.Errorf("check candidate ownership device: %w", err)
	}
	if deviceExists {
		return nil
	}

	existingRegistrationQuery, existingRegistrationArgs, err := storage.Psql.Select("id", "status").
		From("device_registrations").
		Where(sq.Eq{"serial_number": reg.SerialNumber, "carrier": string(reg.Carrier)}).
		OrderBy("created_at DESC").Limit(1).Suffix("FOR UPDATE").ToSql()
	if err != nil {
		return fmt.Errorf("build candidate ownership registration query: %w", err)
	}
	var existingRegistrationID uuid.UUID
	var existingRegistrationStatus string
	err = tx.QueryRow(ctx, existingRegistrationQuery, existingRegistrationArgs...).Scan(
		&existingRegistrationID, &existingRegistrationStatus,
	)
	if err == nil {
		if existingRegistrationStatus != string(global.RegistrationExpired) {
			return nil
		}
		query, args, buildErr := storage.Psql.Update("device_registrations").
			Set("status", string(global.RegistrationPending)).
			Set("remark", nullableStr(reg.Remark)).
			Set("created_by", nullableStr(reg.CreatedBy)).
			Set("updated_at", time.Now().UTC()).
			Where(sq.Eq{"id": existingRegistrationID}).ToSql()
		if buildErr != nil {
			return fmt.Errorf("build candidate ownership registration renewal: %w", buildErr)
		}
		if _, execErr := tx.Exec(ctx, query, args...); execErr != nil {
			return fmt.Errorf("renew candidate ownership registration: %w", execErr)
		}
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("check candidate ownership registration: %w", err)
	}

	if reg.ID == uuid.Nil {
		reg.ID = uuid.New()
	}
	now := time.Now().UTC()
	reg.Status = global.RegistrationPending
	reg.CreatedAt = now
	reg.UpdatedAt = now
	query, args, err := storage.Psql.Insert("device_registrations").
		Columns(regColumns...).
		Values(
			reg.ID, reg.SerialNumber, nullableUUID(reg.GroupID), reg.Carrier,
			nullableStr(reg.SiteName), reg.Longitude, reg.Latitude,
			string(reg.Status), nullableStr(reg.Remark), nullableStr(reg.CreatedBy),
			reg.CreatedAt, reg.UpdatedAt,
		).ToSql()
	if err != nil {
		return fmt.Errorf("build candidate ownership registration: %w", err)
	}
	if _, err := tx.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("insert candidate ownership registration: %w", err)
	}
	return nil
}

var _ RegistrationRepository = (*PgRegistrationRepository)(nil)

// NewPgRegistrationRepository creates a new PgRegistrationRepository.
func NewPgRegistrationRepository(pool *pgxpool.Pool) *PgRegistrationRepository {
	return &PgRegistrationRepository{pool: pool}
}

func (r *PgRegistrationRepository) Create(ctx context.Context, reg *DeviceRegistration) error {
	if reg.ID == uuid.Nil {
		reg.ID = uuid.New()
	}
	now := time.Now()
	reg.CreatedAt = now
	reg.UpdatedAt = now
	if reg.Status == "" {
		reg.Status = global.RegistrationPending
	}

	query, args, err := storage.Psql.Insert("device_registrations").
		Columns(regColumns...).
		Values(
			reg.ID, reg.SerialNumber, nullableUUID(reg.GroupID), reg.Carrier,
			nullableStr(reg.SiteName), reg.Longitude, reg.Latitude,
			string(reg.Status), nullableStr(reg.Remark), nullableStr(reg.CreatedBy),
			reg.CreatedAt, reg.UpdatedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert registration SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert registration: %w", err)
	}
	return nil
}

func (r *PgRegistrationRepository) BatchCreate(ctx context.Context, regs []*DeviceRegistration) (int, error) {
	if len(regs) == 0 {
		return 0, nil
	}

	now := time.Now()
	created := 0

	batch := &pgx.Batch{}
	for _, reg := range regs {
		if reg.ID == uuid.Nil {
			reg.ID = uuid.New()
		}
		reg.CreatedAt = now
		reg.UpdatedAt = now
		if reg.Status == "" {
			reg.Status = global.RegistrationPending
		}

		query, args, err := storage.Psql.Insert("device_registrations").
			Columns(regColumns...).
			Values(
				reg.ID, reg.SerialNumber, nullableUUID(reg.GroupID), reg.Carrier,
				nullableStr(reg.SiteName), reg.Longitude, reg.Latitude,
				string(reg.Status), nullableStr(reg.Remark), nullableStr(reg.CreatedBy),
				reg.CreatedAt, reg.UpdatedAt,
			).
			Suffix("ON CONFLICT DO NOTHING").
			ToSql()
		if err != nil {
			return 0, fmt.Errorf("build batch insert SQL: %w", err)
		}
		batch.Queue(query, args...)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range regs {
		tag, err := br.Exec()
		if err != nil {
			return created, fmt.Errorf("batch insert registration: %w", err)
		}
		created += int(tag.RowsAffected())
	}

	return created, nil
}

func (r *PgRegistrationRepository) GetBySerialNumber(ctx context.Context, sn string) (*DeviceRegistration, error) {
	query, args, err := storage.Psql.Select(regColumns...).
		From("device_registrations").
		Where(sq.Eq{"serial_number": sn, "status": string(global.RegistrationPending)}).
		Limit(1).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get registration SQL: %w", err)
	}

	reg, err := scanRegistration(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, err
	}
	return reg, nil
}

func (r *PgRegistrationRepository) List(ctx context.Context, filter RegistrationFilter) (*model.ListResponse[DeviceRegistration], error) {
	builder := storage.Psql.Select(regColumns...).From("device_registrations")
	countBuilder := storage.Psql.Select("COUNT(*)").From("device_registrations")

	if filter.Status != nil {
		builder = builder.Where(sq.Eq{"status": string(*filter.Status)})
		countBuilder = countBuilder.Where(sq.Eq{"status": string(*filter.Status)})
	}
	if filter.SerialNumber != nil && *filter.SerialNumber != "" {
		keyword := "%" + *filter.SerialNumber + "%"
		builder = builder.Where(sq.ILike{"serial_number": keyword})
		countBuilder = countBuilder.Where(sq.ILike{"serial_number": keyword})
	}
	if filter.Carrier != nil {
		builder = builder.Where(sq.Eq{"carrier": string(*filter.Carrier)})
		countBuilder = countBuilder.Where(sq.Eq{"carrier": string(*filter.Carrier)})
	}
	// #64 设备组数据权限：预注册表自带 group_id，按可见分组三态收口（nil 超管不过滤 /
	// [] WHERE FALSE / [g...] group_id IN(...)）。group_id IS NULL 行对非超管天然不命中被排除。
	builder = authz.ApplyGroupVisibilityFilter(builder, "group_id", filter.VisibleGroups)
	countBuilder = authz.ApplyGroupVisibilityFilter(countBuilder, "group_id", filter.VisibleGroups)

	countQuery, countArgs, _ := countBuilder.ToSql()
	var total int64
	r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total)

	builder = builder.
		OrderBy("created_at DESC").
		Limit(uint64(filter.Limit())).
		Offset(uint64(filter.Offset()))

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list registrations SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list registrations: %w", err)
	}
	defer rows.Close()

	var items []DeviceRegistration
	for rows.Next() {
		reg, err := scanRegistrationFromRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *reg)
	}

	return model.NewListResponse(items, total, filter.Page, filter.Limit()), nil
}

func (r *PgRegistrationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query, args, err := storage.Psql.Update("device_registrations").
		Set("status", status).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update status SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update registration status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgRegistrationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := storage.Psql.Delete("device_registrations").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete registration SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete registration: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

// --- Scan helpers ---

func nullableStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func nullableUUID(id *uuid.UUID) interface{} {
	if id == nil {
		return nil
	}
	return *id
}

func scanRegistration(row pgx.Row) (*DeviceRegistration, error) {
	var reg DeviceRegistration
	var (
		groupID   sql.NullString
		siteName  sql.NullString
		remark    sql.NullString
		createdBy sql.NullString
		status    string
	)

	err := row.Scan(
		&reg.ID, &reg.SerialNumber, &groupID, &reg.Carrier, &siteName,
		&reg.Longitude, &reg.Latitude, &status, &remark, &createdBy,
		&reg.CreatedAt, &reg.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan registration: %w", err)
	}

	reg.Status = global.RegistrationStatus(status)
	if groupID.Valid {
		id, _ := uuid.Parse(groupID.String)
		reg.GroupID = &id
	}
	if siteName.Valid {
		reg.SiteName = siteName.String
	}
	if remark.Valid {
		reg.Remark = remark.String
	}
	if createdBy.Valid {
		reg.CreatedBy = createdBy.String
	}

	return &reg, nil
}

func scanRegistrationFromRows(rows pgx.Rows) (*DeviceRegistration, error) {
	var reg DeviceRegistration
	var (
		groupID   sql.NullString
		siteName  sql.NullString
		remark    sql.NullString
		createdBy sql.NullString
		status    string
	)

	err := rows.Scan(
		&reg.ID, &reg.SerialNumber, &groupID, &reg.Carrier, &siteName,
		&reg.Longitude, &reg.Latitude, &status, &remark, &createdBy,
		&reg.CreatedAt, &reg.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan registration row: %w", err)
	}

	reg.Status = global.RegistrationStatus(status)
	if groupID.Valid {
		id, _ := uuid.Parse(groupID.String)
		reg.GroupID = &id
	}
	if siteName.Valid {
		reg.SiteName = siteName.String
	}
	if remark.Valid {
		reg.Remark = remark.String
	}
	if createdBy.Valid {
		reg.CreatedBy = createdBy.String
	}

	return &reg, nil
}
