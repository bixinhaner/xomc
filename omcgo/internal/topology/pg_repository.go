package topology

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

var groupColumns = []string{
	"id", "name", "parent_id", "carrier", "description", "sort_order",
	"created_at", "updated_at",
}

// PgDeviceGroupRepository implements DeviceGroupRepository using PostgreSQL.
type PgDeviceGroupRepository struct {
	pool *pgxpool.Pool
}

// NewPgDeviceGroupRepository creates a new PgDeviceGroupRepository.
func NewPgDeviceGroupRepository(pool *pgxpool.Pool) *PgDeviceGroupRepository {
	return &PgDeviceGroupRepository{pool: pool}
}

func (r *PgDeviceGroupRepository) Create(ctx context.Context, group *DeviceGroup) error {
	if group.ID == uuid.Nil {
		group.ID = uuid.New()
	}
	now := time.Now()
	group.CreatedAt = now
	group.UpdatedAt = now

	query, args, err := psql.Insert("device_groups").
		Columns(groupColumns...).
		Values(
			group.ID, group.Name, nullableUUID(group.ParentID),
			nullableString(string(group.Carrier)), nullableString(group.Description),
			group.SortOrder, group.CreatedAt, group.UpdatedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert group SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert device group: %w", err)
	}
	return nil
}

func (r *PgDeviceGroupRepository) GetByID(ctx context.Context, id uuid.UUID) (*DeviceGroup, error) {
	query, args, err := psql.Select(groupColumns...).
		From("device_groups").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build select group SQL: %w", err)
	}

	group, err := scanGroup(r.pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (r *PgDeviceGroupRepository) Update(ctx context.Context, group *DeviceGroup) error {
	group.UpdatedAt = time.Now()

	query, args, err := psql.Update("device_groups").
		Set("name", group.Name).
		Set("parent_id", nullableUUID(group.ParentID)).
		Set("carrier", nullableString(string(group.Carrier))).
		Set("description", nullableString(group.Description)).
		Set("sort_order", group.SortOrder).
		Set("updated_at", group.UpdatedAt).
		Where(sq.Eq{"id": group.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update group SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update device group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("update group %s: %w", group.ID, commonerrors.ErrNotFound)
	}
	return nil
}

func (r *PgDeviceGroupRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query, args, err := psql.Delete("device_groups").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete group SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete device group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete group %s: %w", id, commonerrors.ErrNotFound)
	}
	return nil
}

func (r *PgDeviceGroupRepository) ListRoots(ctx context.Context) ([]DeviceGroup, error) {
	query, args, err := psql.Select(groupColumns...).
		From("device_groups").
		Where("parent_id IS NULL").
		OrderBy("sort_order ASC", "name ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list roots SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list root groups: %w", err)
	}
	defer rows.Close()

	return scanGroups(rows)
}

func (r *PgDeviceGroupRepository) ListChildren(ctx context.Context, parentID uuid.UUID) ([]DeviceGroup, error) {
	query, args, err := psql.Select(groupColumns...).
		From("device_groups").
		Where(sq.Eq{"parent_id": parentID}).
		OrderBy("sort_order ASC", "name ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list children SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list child groups: %w", err)
	}
	defer rows.Close()

	return scanGroups(rows)
}

// GetTree returns a flat list of all groups. Tree assembly is done in the service layer.
func (r *PgDeviceGroupRepository) GetTree(ctx context.Context) ([]DeviceGroup, error) {
	query, args, err := psql.Select(groupColumns...).
		From("device_groups").
		OrderBy("sort_order ASC", "name ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get tree SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get group tree: %w", err)
	}
	defer rows.Close()

	return scanGroups(rows)
}

func (r *PgDeviceGroupRepository) AddDevice(ctx context.Context, groupID, deviceID uuid.UUID) error {
	query, args, err := psql.Insert("device_group_members").
		Columns("group_id", "device_id", "added_at").
		Values(groupID, deviceID, time.Now()).
		ToSql()
	if err != nil {
		return fmt.Errorf("build add device SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("add device to group: %w", err)
	}
	return nil
}

func (r *PgDeviceGroupRepository) RemoveDevice(ctx context.Context, groupID, deviceID uuid.UUID) error {
	query, args, err := psql.Delete("device_group_members").
		Where(sq.And{
			sq.Eq{"group_id": groupID},
			sq.Eq{"device_id": deviceID},
		}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build remove device SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("remove device from group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("device %s not in group %s: %w", deviceID, groupID, commonerrors.ErrNotFound)
	}
	return nil
}

func (r *PgDeviceGroupRepository) ListDeviceIDs(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	query, args, err := psql.Select("device_id").
		From("device_group_members").
		Where(sq.Eq{"group_id": groupID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list device IDs SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list device IDs: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan device ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// --- Helpers ---

func nullableString(s string) interface{} {
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

func scanGroup(row pgx.Row) (*DeviceGroup, error) {
	var g DeviceGroup
	var (
		parentID    sql.NullString
		carrier     sql.NullString
		description sql.NullString
	)

	err := row.Scan(
		&g.ID, &g.Name, &parentID, &carrier, &description,
		&g.SortOrder, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan group row: %w", err)
	}

	if parentID.Valid {
		id, _ := uuid.Parse(parentID.String)
		g.ParentID = &id
	}
	if carrier.Valid {
		g.Carrier = model.CarrierCode(carrier.String)
	}
	if description.Valid {
		g.Description = description.String
	}

	return &g, nil
}

func scanGroups(rows pgx.Rows) ([]DeviceGroup, error) {
	var items []DeviceGroup
	for rows.Next() {
		var g DeviceGroup
		var (
			parentID    sql.NullString
			carrier     sql.NullString
			description sql.NullString
		)

		err := rows.Scan(
			&g.ID, &g.Name, &parentID, &carrier, &description,
			&g.SortOrder, &g.CreatedAt, &g.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan group row: %w", err)
		}

		if parentID.Valid {
			id, _ := uuid.Parse(parentID.String)
			g.ParentID = &id
		}
		if carrier.Valid {
			g.Carrier = model.CarrierCode(carrier.String)
		}
		if description.Valid {
			g.Description = description.String
		}

		items = append(items, g)
	}
	return items, rows.Err()
}

var _ DeviceGroupRepository = (*PgDeviceGroupRepository)(nil)
