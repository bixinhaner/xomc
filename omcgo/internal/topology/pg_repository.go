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
	"github.com/omcgo/omcgo/global"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

var groupColumns = []string{
	"id", "name", "parent_id", "carrier", "description", "sort_order",
	"level", "status", "is_default", "remark", "created_by", "updated_by",
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

	// Derive level from parent_id.
	if group.ParentID == nil {
		group.Level = 1
	} else {
		group.Level = 2
	}
	if group.Status == "" {
		group.Status = string(global.GroupStatusActive)
	}

	query, args, err := psql.Insert("device_groups").
		Columns(groupColumns...).
		Values(
			group.ID, group.Name, nullableUUID(group.ParentID),
			nullableString(string(group.Carrier)), nullableString(group.Description),
			group.SortOrder, group.Level, group.Status, group.IsDefault,
			nullableString(group.Remark), nullableString(group.CreatedBy),
			nullableString(group.UpdatedBy),
			group.CreatedAt, group.UpdatedAt,
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
		Set("remark", nullableString(group.Remark)).
		Set("updated_by", nullableString(group.UpdatedBy)).
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

// GetTreeWithCounts returns all groups with device_count populated via LEFT JOIN.
func (r *PgDeviceGroupRepository) GetTreeWithCounts(ctx context.Context) ([]DeviceGroup, error) {
	const rawSQL = `
		SELECT dg.id, dg.name, dg.parent_id, dg.carrier, dg.description, dg.sort_order,
		       dg.level, dg.status, dg.is_default, dg.remark, dg.created_by, dg.updated_by,
		       dg.created_at, dg.updated_at,
		       COUNT(dgm.device_id) AS device_count
		FROM device_groups dg
		LEFT JOIN device_group_members dgm ON dgm.group_id = dg.id
		GROUP BY dg.id
		ORDER BY dg.sort_order ASC, dg.name ASC`

	rows, err := r.pool.Query(ctx, rawSQL)
	if err != nil {
		return nil, fmt.Errorf("get tree with counts: %w", err)
	}
	defer rows.Close()

	return scanGroupsWithCount(rows)
}

// ExistsByParentAndName checks if a group with the given name exists under the parent.
func (r *PgDeviceGroupRepository) ExistsByParentAndName(ctx context.Context, parentID *uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
	builder := psql.Select("1").
		From("device_groups").
		Where(sq.Eq{"name": name})

	if parentID == nil {
		builder = builder.Where("parent_id IS NULL")
	} else {
		builder = builder.Where(sq.Eq{"parent_id": *parentID})
	}
	if excludeID != nil {
		builder = builder.Where(sq.NotEq{"id": *excludeID})
	}

	query, args, err := builder.Limit(1).ToSql()
	if err != nil {
		return false, fmt.Errorf("build exists by name SQL: %w", err)
	}

	var dummy int
	err = r.pool.QueryRow(ctx, query, args...).Scan(&dummy)
	if err == pgx.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check name existence: %w", err)
	}
	return true, nil
}

// GetStats returns aggregate statistics about groups and device membership.
func (r *PgDeviceGroupRepository) GetStats(ctx context.Context) (*GroupStats, error) {
	const rawSQL = `
		SELECT
			(SELECT COUNT(*) FROM device_groups) AS total_groups,
			(SELECT COUNT(DISTINCT device_id) FROM device_group_members) AS grouped_devices,
			(SELECT COUNT(*) FROM devices d WHERE NOT EXISTS (
				SELECT 1 FROM device_group_members dgm WHERE dgm.device_id = d.id
			)) AS ungrouped_devices`

	var stats GroupStats
	err := r.pool.QueryRow(ctx, rawSQL).Scan(
		&stats.TotalGroups, &stats.GroupedDevices, &stats.UngroupedDevices,
	)
	if err != nil {
		return nil, fmt.Errorf("get group stats: %w", err)
	}
	return &stats, nil
}

// CountDevicesByGroup counts devices in a specific group.
func (r *PgDeviceGroupRepository) CountDevicesByGroup(ctx context.Context, groupID uuid.UUID) (int, error) {
	query, args, err := psql.Select("COUNT(*)").
		From("device_group_members").
		Where(sq.Eq{"group_id": groupID}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count devices SQL: %w", err)
	}

	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count devices by group: %w", err)
	}
	return count, nil
}

// ListChildIDs returns the IDs of all direct children of the given parent.
func (r *PgDeviceGroupRepository) ListChildIDs(ctx context.Context, parentID uuid.UUID) ([]uuid.UUID, error) {
	query, args, err := psql.Select("id").
		From("device_groups").
		Where(sq.Eq{"parent_id": parentID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list child IDs SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list child IDs: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan child ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// GetGroupLevel returns the level of a group by ID. Used by PermissionService.
func (r *PgDeviceGroupRepository) GetGroupLevel(ctx context.Context, id uuid.UUID) (int, error) {
	query, args, err := psql.Select("level").
		From("device_groups").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build get group level SQL: %w", err)
	}

	var level int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&level); err != nil {
		if err == pgx.ErrNoRows {
			return 0, commonerrors.ErrNotFound
		}
		return 0, fmt.Errorf("get group level: %w", err)
	}
	return level, nil
}

// --- GroupMembership methods ---

func (r *PgDeviceGroupRepository) AddDevice(ctx context.Context, groupID, deviceID uuid.UUID) error {
	// UPSERT: moves device if already in another group.
	const rawSQL = `
		INSERT INTO device_group_members (group_id, device_id, added_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (device_id) DO UPDATE SET group_id = EXCLUDED.group_id, added_at = EXCLUDED.added_at`

	_, err := r.pool.Exec(ctx, rawSQL, groupID, deviceID, time.Now())
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

// BatchAddDevices adds multiple devices to a group via UPSERT.
func (r *PgDeviceGroupRepository) BatchAddDevices(ctx context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error) {
	if len(deviceIDs) == 0 {
		return 0, nil
	}

	now := time.Now()
	var totalAffected int64
	batch := &pgx.Batch{}
	const rawSQL = `
		INSERT INTO device_group_members (group_id, device_id, added_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (device_id) DO UPDATE SET group_id = EXCLUDED.group_id, added_at = EXCLUDED.added_at`

	for _, did := range deviceIDs {
		batch.Queue(rawSQL, groupID, did, now)
	}

	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range deviceIDs {
		tag, err := br.Exec()
		if err != nil {
			return totalAffected, fmt.Errorf("batch add device: %w", err)
		}
		totalAffected += tag.RowsAffected()
	}
	return totalAffected, nil
}

// BatchRemoveDevices removes multiple devices from a specific group.
func (r *PgDeviceGroupRepository) BatchRemoveDevices(ctx context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error) {
	if len(deviceIDs) == 0 {
		return 0, nil
	}

	query, args, err := psql.Delete("device_group_members").
		Where(sq.And{
			sq.Eq{"group_id": groupID},
			sq.Eq{"device_id": deviceIDs},
		}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build batch remove SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("batch remove devices: %w", err)
	}
	return tag.RowsAffected(), nil
}

// MoveDevices moves devices to a target group (UPSERT).
func (r *PgDeviceGroupRepository) MoveDevices(ctx context.Context, deviceIDs []uuid.UUID, targetGroupID uuid.UUID) (int64, error) {
	return r.BatchAddDevices(ctx, targetGroupID, deviceIDs)
}

// MoveGroupDevicesToDefault moves all devices from the given groups to the default L2 group.
func (r *PgDeviceGroupRepository) MoveGroupDevicesToDefault(ctx context.Context, groupIDs []uuid.UUID) (int64, error) {
	if len(groupIDs) == 0 {
		return 0, nil
	}

	defaultGroupID, _ := uuid.Parse(global.DefaultLevel2GroupID)
	const rawSQL = `
		UPDATE device_group_members
		SET group_id = $1, added_at = NOW()
		WHERE group_id = ANY($2)`

	tag, err := r.pool.Exec(ctx, rawSQL, defaultGroupID, groupIDs)
	if err != nil {
		return 0, fmt.Errorf("move devices to default group: %w", err)
	}
	return tag.RowsAffected(), nil
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
		remark      sql.NullString
		createdBy   sql.NullString
		updatedBy   sql.NullString
	)

	err := row.Scan(
		&g.ID, &g.Name, &parentID, &carrier, &description,
		&g.SortOrder, &g.Level, &g.Status, &g.IsDefault,
		&remark, &createdBy, &updatedBy,
		&g.CreatedAt, &g.UpdatedAt,
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
	if remark.Valid {
		g.Remark = remark.String
	}
	if createdBy.Valid {
		g.CreatedBy = createdBy.String
	}
	if updatedBy.Valid {
		g.UpdatedBy = updatedBy.String
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
			remark      sql.NullString
			createdBy   sql.NullString
			updatedBy   sql.NullString
		)

		err := rows.Scan(
			&g.ID, &g.Name, &parentID, &carrier, &description,
			&g.SortOrder, &g.Level, &g.Status, &g.IsDefault,
			&remark, &createdBy, &updatedBy,
			&g.CreatedAt, &g.UpdatedAt,
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
		if remark.Valid {
			g.Remark = remark.String
		}
		if createdBy.Valid {
			g.CreatedBy = createdBy.String
		}
		if updatedBy.Valid {
			g.UpdatedBy = updatedBy.String
		}

		items = append(items, g)
	}
	return items, rows.Err()
}

func scanGroupsWithCount(rows pgx.Rows) ([]DeviceGroup, error) {
	var items []DeviceGroup
	for rows.Next() {
		var g DeviceGroup
		var (
			parentID    sql.NullString
			carrier     sql.NullString
			description sql.NullString
			remark      sql.NullString
			createdBy   sql.NullString
			updatedBy   sql.NullString
		)

		err := rows.Scan(
			&g.ID, &g.Name, &parentID, &carrier, &description,
			&g.SortOrder, &g.Level, &g.Status, &g.IsDefault,
			&remark, &createdBy, &updatedBy,
			&g.CreatedAt, &g.UpdatedAt,
			&g.DeviceCount,
		)
		if err != nil {
			return nil, fmt.Errorf("scan group with count: %w", err)
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
		if remark.Valid {
			g.Remark = remark.String
		}
		if createdBy.Valid {
			g.CreatedBy = createdBy.String
		}
		if updatedBy.Valid {
			g.UpdatedBy = updatedBy.String
		}

		items = append(items, g)
	}
	return items, rows.Err()
}

var _ DeviceGroupRepository = (*PgDeviceGroupRepository)(nil)
