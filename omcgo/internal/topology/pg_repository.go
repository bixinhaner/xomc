package topology

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/singleflight"

	"github.com/omcgo/omcgo/global"
	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// groupTreeCountsCacheTTL 是 GetTreeWithCounts 的短TTL读缓存有效期。
// 该查询会读取 devices 分区表和 device_group_members 成员表，高并发轮询（如分组树
// 前端页面）下会成为 postgres CPU 的主要来源之一。分组结构和设备计数没有强一致要求，
// 短TTL缓存可以把同一窗口内的并发重复调用收敛成一次真实查询。
const groupTreeCountsCacheTTL = 3 * time.Second

var groupColumns = []string{
	"id", "name", "parent_id", "carrier", "description", "sort_order",
	"level", "status", "is_default", "remark", "created_by", "updated_by",
	"created_at", "updated_at",
	"matching_mode", "name_rule_list", "lac_list", "tac_list",
	"serial_number_list", "source_group_id",
	"name_i18n", "description_i18n", "remark_i18n", // migration 000003
}

// PgDeviceGroupRepository implements DeviceGroupRepository using PostgreSQL.
type PgDeviceGroupRepository struct {
	pool *pgxpool.Pool

	treeCountsMu   sync.Mutex
	treeCountsAt   time.Time
	treeCountsData []DeviceGroup
	treeCountsGen  uint64
	treeCountsLoad singleflight.Group
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

	// 序列化 name_rule_list 为 JSONB
	var nameRuleListJSON []byte
	if len(group.NameRuleList) > 0 {
		var err error
		nameRuleListJSON, err = json.Marshal(group.NameRuleList)
		if err != nil {
			return fmt.Errorf("marshal name_rule_list: %w", err)
		}
	}
	// i18n JSONB 列 — migration 000003
	nameI18nJSON := marshalI18n(group.NameI18n)
	descI18nJSON := marshalI18n(group.DescriptionI18n)
	remarkI18nJSON := marshalI18n(group.RemarkI18n)

	query, args, err := storage.Psql.Insert("device_groups").
		Columns(groupColumns...).
		Values(
			group.ID, group.Name, nullableUUID(group.ParentID),
			nullableString(string(group.Carrier)), nullableString(group.Description),
			group.SortOrder, group.Level, group.Status, group.IsDefault,
			nullableString(group.Remark), nullableString(group.CreatedBy),
			nullableString(group.UpdatedBy),
			group.CreatedAt, group.UpdatedAt,
			nullableString(string(group.MatchingMode)),
			nullableJSONB(nameRuleListJSON),
			nullableIntArray(group.LACList),
			nullableIntArray(group.TACList),
			nullableStringArray(group.SerialNumberList),
			nullableUUID(group.SourceGroupID),
			nameI18nJSON, descI18nJSON, remarkI18nJSON,
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
	query, args, err := storage.Psql.Select(groupColumns...).
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

	// 序列化 name_rule_list 为 JSONB
	var nameRuleListJSON []byte
	if len(group.NameRuleList) > 0 {
		var err error
		nameRuleListJSON, err = json.Marshal(group.NameRuleList)
		if err != nil {
			return fmt.Errorf("marshal name_rule_list: %w", err)
		}
	}

	query, args, err := storage.Psql.Update("device_groups").
		Set("name", group.Name).
		Set("parent_id", nullableUUID(group.ParentID)).
		Set("level", group.Level).
		Set("carrier", nullableString(string(group.Carrier))).
		Set("description", nullableString(group.Description)).
		Set("sort_order", group.SortOrder).
		Set("remark", nullableString(group.Remark)).
		Set("updated_by", nullableString(group.UpdatedBy)).
		Set("updated_at", group.UpdatedAt).
		Set("matching_mode", nullableString(string(group.MatchingMode))).
		Set("name_rule_list", nullableJSONB(nameRuleListJSON)).
		Set("lac_list", nullableIntArray(group.LACList)).
		Set("tac_list", nullableIntArray(group.TACList)).
		Set("serial_number_list", nullableStringArray(group.SerialNumberList)).
		Set("source_group_id", nullableUUID(group.SourceGroupID)).
		Set("name_i18n", marshalI18n(group.NameI18n)). // migration 000003 i18n
		Set("description_i18n", marshalI18n(group.DescriptionI18n)).
		Set("remark_i18n", marshalI18n(group.RemarkI18n)).
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
	query, args, err := storage.Psql.Delete("device_groups").
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
	query, args, err := storage.Psql.Select(groupColumns...).
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
	query, args, err := storage.Psql.Select(groupColumns...).
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
	query, args, err := storage.Psql.Select(groupColumns...).
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

const getTreeWithCountsRawSQL = `
		WITH device_counts AS (
			SELECT COALESCE(dgm.group_id, $1::uuid) AS group_id,
			       COUNT(*) AS count
			FROM devices d
			LEFT JOIN device_group_members dgm ON dgm.device_id = d.id
			WHERE d.deleted_at IS NULL
			GROUP BY COALESCE(dgm.group_id, $1::uuid)
		)
		SELECT dg.id, dg.name, dg.parent_id, dg.carrier, dg.description, dg.sort_order,
		       dg.level, dg.status, dg.is_default, dg.remark, dg.created_by, dg.updated_by,
		       dg.created_at, dg.updated_at,
		       dg.matching_mode, dg.name_rule_list, dg.lac_list, dg.tac_list,
		       dg.serial_number_list, dg.source_group_id,
		       COALESCE(device_counts.count, 0) AS device_count,
		       dg.name_i18n, dg.description_i18n, dg.remark_i18n
		FROM device_groups dg
		LEFT JOIN device_counts ON device_counts.group_id = dg.id
		ORDER BY dg.sort_order ASC, dg.name ASC`

// GetTreeWithCounts returns all groups with device_count populated.
// Optimized: aggregates device membership counts once and joins the result
// back to the group tree.
//
// 短TTL读缓存（groupTreeCountsCacheTTL）：分组树接口在高并发轮询/多用户打开分组页
// 时会被大量并发重复调用，缓存把 TTL 窗口内的重复调用收敛成一次真实查询，避免 devices
// 分区表和成员表的计数开销被并发放大成 CPU 热点。分组结构和设备计数没有强一致性要求，
// 短暂（几秒）过期可接受。
func (r *PgDeviceGroupRepository) GetTreeWithCounts(ctx context.Context) ([]DeviceGroup, error) {
	return r.getTreeWithCountsCached(ctx, r.loadTreeWithCounts)
}

func (r *PgDeviceGroupRepository) loadTreeWithCounts(ctx context.Context) ([]DeviceGroup, error) {
	rows, err := r.pool.Query(ctx, getTreeWithCountsRawSQL, global.DefaultLevel2GroupID)
	if err != nil {
		return nil, fmt.Errorf("get tree with counts: %w", err)
	}
	defer rows.Close()

	return scanGroupsWithCount(rows)
}

// getTreeWithCountsCached combines concurrent cache misses into one database
// load. The second cache check inside the singleflight callback is required:
// another request may have filled the cache after this caller's fast-path miss.
func (r *PgDeviceGroupRepository) getTreeWithCountsCached(
	ctx context.Context,
	load func(context.Context) ([]DeviceGroup, error),
) ([]DeviceGroup, error) {
	cached, ok, generation := r.cachedTreeWithCounts()
	if ok {
		return cached, nil
	}

	value, err, _ := r.treeCountsLoad.Do("tree-with-counts", func() (any, error) {
		if cached, ok, currentGeneration := r.cachedTreeWithCounts(); ok {
			return cached, nil
		} else {
			generation = currentGeneration
		}

		groups, loadErr := load(ctx)
		if loadErr != nil {
			return nil, loadErr
		}
		r.storeTreeWithCountsCache(groups, generation)
		return groups, nil
	})
	if err != nil {
		return nil, err
	}
	groups := value.([]DeviceGroup)
	return cloneDeviceGroups(groups), nil
}

// cachedTreeWithCounts 返回缓存副本（未过期时）。返回副本而非共享切片，是因为
// service.buildTree 会就地改写传入切片元素（flat[i].Children = nil 等），共享同一
// 底层数组会在并发调用间互相污染。
func (r *PgDeviceGroupRepository) cachedTreeWithCounts() ([]DeviceGroup, bool, uint64) {
	r.treeCountsMu.Lock()
	defer r.treeCountsMu.Unlock()
	if r.treeCountsData == nil || time.Since(r.treeCountsAt) > groupTreeCountsCacheTTL {
		return nil, false, r.treeCountsGen
	}
	return cloneDeviceGroups(r.treeCountsData), true, r.treeCountsGen
}

func (r *PgDeviceGroupRepository) storeTreeWithCountsCache(groups []DeviceGroup, generation uint64) {
	r.treeCountsMu.Lock()
	defer r.treeCountsMu.Unlock()
	if generation != r.treeCountsGen {
		return
	}
	r.treeCountsData = groups
	r.treeCountsAt = time.Now()
}

// InvalidateDeviceGroupCounts clears the short-lived group tree count cache.
// Device lifecycle writes live in another module, so they call this narrow
// invalidation seam after a successful write instead of waiting for the TTL.
func (r *PgDeviceGroupRepository) InvalidateDeviceGroupCounts() {
	r.treeCountsMu.Lock()
	defer r.treeCountsMu.Unlock()
	r.treeCountsGen++
	r.treeCountsData = nil
	r.treeCountsAt = time.Time{}
}

// cloneDeviceGroups 返回顶层切片的独立拷贝。DeviceGroup 在这个扁平列表阶段还没有
// 填充 Children（由 service.buildTree 之后才构建），值拷贝足够安全。
func cloneDeviceGroups(src []DeviceGroup) []DeviceGroup {
	out := make([]DeviceGroup, len(src))
	copy(out, src)
	return out
}

// ExistsByParentAndName checks if a group with the given name exists under the parent.
func (r *PgDeviceGroupRepository) ExistsByParentAndName(ctx context.Context, parentID *uuid.UUID, name string, excludeID *uuid.UUID) (bool, error) {
	builder := storage.Psql.Select("1").
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
	// device 计数一律排除软删除设备（deleted_at IS NOT NULL）——回收/删除走软删
	// (device_repository.BatchDelete:UPDATE deleted_at + 删 device_group_members)。
	// 否则被删设备脱离分组后仍计入「未分组」，与 grouped_devices 的 -1 相抵，「全部」总数不变。
	const rawSQL = `
		SELECT
			(SELECT COUNT(*) FROM device_groups) AS total_groups,
			(SELECT COUNT(DISTINCT dgm.device_id)
			   FROM device_group_members dgm
			   JOIN devices d ON d.id = dgm.device_id
			  WHERE d.deleted_at IS NULL) AS grouped_devices,
			(SELECT COUNT(*) FROM devices d
			  WHERE d.deleted_at IS NULL
			    AND NOT EXISTS (
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
	query, args, err := storage.Psql.Select("COUNT(*)").
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
	query, args, err := storage.Psql.Select("id").
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
	query, args, err := storage.Psql.Select("level").
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

// AddDeviceWithSource UPSERT 设备到 group 并标注来源（T-0027 D5.B）。
// SQL 层 A4 守护：WHERE device_group_members.source_type != 'manual' 让
// PG 在冲突时跳过 UPDATE — manual 行被永久保留。
// 返回 rowsAffected：1 表示真插入/更新，0 表示因 manual override 被跳过。
// 调用方按返回值决定 metric counter（matched / skipped_manual）。
func (r *PgDeviceGroupRepository) AddDeviceWithSource(ctx context.Context, groupID, deviceID uuid.UUID, sourceType string, sourceRuleID *uuid.UUID) (int64, error) {
	const rawSQL = `
		INSERT INTO device_group_members (group_id, device_id, added_at, source_type, source_rule_id)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (device_id) DO UPDATE SET
			group_id = EXCLUDED.group_id,
			added_at = EXCLUDED.added_at,
			source_type = EXCLUDED.source_type,
			source_rule_id = EXCLUDED.source_rule_id
		WHERE device_group_members.source_type IS DISTINCT FROM 'manual'`

	tag, err := r.pool.Exec(ctx, rawSQL, groupID, deviceID, time.Now(), sourceType, sourceRuleID)
	if err != nil {
		return 0, fmt.Errorf("add device to group with source: %w", err)
	}
	return tag.RowsAffected(), nil
}

// MoveDeviceAutoMatched 按显式源组原子移动。源组条件本身就是授权边界，
// 因此源组中的 manual 归属也允许被该规则移动；其他组的 manual 行不会被触碰。
func (r *PgDeviceGroupRepository) MoveDeviceAutoMatched(ctx context.Context, sourceGroupID, targetGroupID, deviceID uuid.UUID) (int64, error) {
	var (
		tag pgconn.CommandTag
		err error
	)
	if sourceGroupID.String() == global.DefaultLevel2GroupID {
		const insertSQL = `
			INSERT INTO device_group_members (group_id, device_id, added_at, source_type)
			SELECT $1, $2, $3, 'rule'
			WHERE EXISTS (SELECT 1 FROM device_groups WHERE id = $4 AND level = 2)
			  AND EXISTS (SELECT 1 FROM device_groups WHERE id = $1 AND level = 2)
			ON CONFLICT (device_id) DO UPDATE
			SET group_id = EXCLUDED.group_id,
			    added_at = EXCLUDED.added_at,
			    source_type = 'rule'
			WHERE device_group_members.group_id = $4`
		tag, err = r.pool.Exec(ctx, insertSQL, targetGroupID, deviceID, time.Now(), sourceGroupID)
	} else {
		const updateSQL = `
			UPDATE device_group_members AS membership
			SET group_id = $1, added_at = $2, source_type = 'rule'
			FROM device_groups AS source_group, device_groups AS target_group
			WHERE membership.device_id = $3
			  AND membership.group_id = $4
			  AND source_group.id = $4 AND source_group.level = 2
			  AND target_group.id = $1 AND target_group.level = 2`
		tag, err = r.pool.Exec(ctx, updateSQL, targetGroupID, time.Now(), deviceID, sourceGroupID)
	}
	if err != nil {
		return 0, fmt.Errorf("move auto-matched device from source group: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *PgDeviceGroupRepository) RemoveDevice(ctx context.Context, groupID, deviceID uuid.UUID) error {
	query, args, err := storage.Psql.Delete("device_group_members").
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
	query, args, err := storage.Psql.Select("device_id").
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

// BatchAddDevices adds multiple devices to a group via a single multi-row UPSERT.
func (r *PgDeviceGroupRepository) BatchAddDevices(ctx context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error) {
	if len(deviceIDs) == 0 {
		return 0, nil
	}

	now := time.Now()
	// Build: INSERT INTO device_group_members (group_id, device_id, added_at)
	// VALUES ($1,$2,$3),($1,$4,$3),... ON CONFLICT ... DO UPDATE ...
	// $1=groupID, $2=now (shared), remaining args are device IDs.
	// Layout: args[0]=groupID, args[1]=now, args[2..]=deviceIDs
	args := make([]interface{}, 0, 2+len(deviceIDs))
	args = append(args, groupID, now)

	valueParts := make([]string, 0, len(deviceIDs))
	for i, did := range deviceIDs {
		placeholder := fmt.Sprintf("($1, $%d, $2)", i+3)
		valueParts = append(valueParts, placeholder)
		args = append(args, did)
	}

	rawSQL := "INSERT INTO device_group_members (group_id, device_id, added_at) VALUES " +
		joinStrings(valueParts, ", ") +
		" ON CONFLICT (device_id) DO UPDATE SET group_id = EXCLUDED.group_id, added_at = EXCLUDED.added_at"

	tag, err := r.pool.Exec(ctx, rawSQL, args...)
	if err != nil {
		return 0, fmt.Errorf("batch add devices: %w", err)
	}
	return tag.RowsAffected(), nil
}

// BatchRemoveDevices removes multiple devices from a specific group.
func (r *PgDeviceGroupRepository) BatchRemoveDevices(ctx context.Context, groupID uuid.UUID, deviceIDs []uuid.UUID) (int64, error) {
	if len(deviceIDs) == 0 {
		return 0, nil
	}

	query, args, err := storage.Psql.Delete("device_group_members").
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

// RemoveDevicesFromAllGroups 按 device_id 删除给定设备的全部归属记录（不限分组）。
// 默认组是普通真实分组；该方法仅保留给历史无归属数据修复/管理类任务。
func (r *PgDeviceGroupRepository) RemoveDevicesFromAllGroups(ctx context.Context, deviceIDs []uuid.UUID) (int64, error) {
	if len(deviceIDs) == 0 {
		return 0, nil
	}

	query, args, err := storage.Psql.Delete("device_group_members").
		Where(sq.Eq{"device_id": deviceIDs}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build remove devices from all groups SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("remove devices from all groups: %w", err)
	}
	return tag.RowsAffected(), nil
}

// MoveGroupDevicesToDefault 删除分组时处理其成员设备。
// 默认组是一个真实 L2 组：删除其他组时把成员归属更新到默认组，回收站也能继续
// 通过 device_group_members 显示设备的最终归属。
func (r *PgDeviceGroupRepository) MoveGroupDevicesToDefault(ctx context.Context, groupIDs []uuid.UUID) (int64, error) {
	return moveGroupDevicesToDefaultTx(ctx, r.pool, groupIDs)
}

// moveGroupDevicesToDefaultTx executes MoveGroupDevicesToDefault on a generic executor (pool or tx).
func moveGroupDevicesToDefaultTx(ctx context.Context, ex pgxExecutor, groupIDs []uuid.UUID) (int64, error) {
	if len(groupIDs) == 0 {
		return 0, nil
	}

	defaultGroupID := uuid.MustParse(global.DefaultLevel2GroupID)
	const rawSQL = `
		UPDATE device_group_members
		   SET group_id = $1, added_at = NOW()
		 WHERE group_id = ANY($2)
		   AND group_id <> $1`

	tag, err := ex.Exec(ctx, rawSQL, defaultGroupID, groupIDs)
	if err != nil {
		return 0, fmt.Errorf("move device memberships to default on group delete: %w", err)
	}
	return tag.RowsAffected(), nil
}

// deleteGroupTx deletes a group row on a generic executor (pool or tx).
func deleteGroupTx(ctx context.Context, ex pgxExecutor, id uuid.UUID) error {
	query, args, err := storage.Psql.Delete("device_groups").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete group SQL: %w", err)
	}

	tag, err := ex.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete device group: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("delete group %s: %w", id, commonerrors.ErrNotFound)
	}
	return nil
}

// listChildIDsTx returns direct child IDs on a generic executor (pool or tx).
func listChildIDsTx(ctx context.Context, ex pgxQuerier, parentID uuid.UUID) ([]uuid.UUID, error) {
	query, args, err := storage.Psql.Select("id").
		From("device_groups").
		Where(sq.Eq{"parent_id": parentID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list child IDs SQL: %w", err)
	}

	rows, err := ex.Query(ctx, query, args...)
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

// BatchSort updates sort_order for multiple groups in a single CASE WHEN SQL.
func (r *PgDeviceGroupRepository) BatchSort(ctx context.Context, items []BatchSortItem) (int64, error) {
	if len(items) == 0 {
		return 0, nil
	}

	rawSQL, args := buildBatchSortSQL(items)

	tag, err := r.pool.Exec(ctx, rawSQL, args...)
	if err != nil {
		return 0, fmt.Errorf("batch sort groups: %w", err)
	}
	return tag.RowsAffected(), nil
}

// buildBatchSortSQL builds:
//
//	UPDATE device_groups SET sort_order = CASE WHEN id = $1::uuid THEN $2::int ... END
//	WHERE id IN ($1::uuid, $3::uuid, ...)
//
// Placeholders carry explicit casts: without them PG infers the bare CASE
// branch ($N) as text and rejects the assignment to the integer sort_order
// column with SQLSTATE 42804.
func buildBatchSortSQL(items []BatchSortItem) (string, []interface{}) {
	args := make([]interface{}, 0, len(items)*2)
	caseParts := make([]string, 0, len(items))
	inParts := make([]string, 0, len(items))

	for i, item := range items {
		idxID := i*2 + 1
		idxVal := i*2 + 2
		caseParts = append(caseParts, fmt.Sprintf("WHEN id = $%d::uuid THEN $%d::int", idxID, idxVal))
		inParts = append(inParts, fmt.Sprintf("$%d::uuid", idxID))
		args = append(args, item.ID, item.SortOrder)
	}

	rawSQL := "UPDATE device_groups SET sort_order = CASE " +
		joinStrings(caseParts, " ") +
		" END WHERE id IN (" + joinStrings(inParts, ", ") + ")"
	return rawSQL, args
}

// pgxExecutor abstracts pgxpool.Pool and pgx.Tx for Exec/Query operations.
type pgxExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// pgxQuerier abstracts pgxpool.Pool and pgx.Tx for Query operations.
type pgxQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// joinStrings joins a slice of strings with sep (avoids importing strings package).
func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for _, s := range ss[1:] {
		result += sep + s
	}
	return result
}

// BatchSortItem holds an ID and its new sort_order for batch updates.
type BatchSortItem struct {
	ID        uuid.UUID
	SortOrder int
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

func nullableJSONB(data []byte) interface{} {
	if len(data) == 0 {
		return nil
	}
	return data
}

// marshalI18n: map[string]string → JSONB bytes for direct pgx insert.
// Empty/nil map → '{}'::jsonb (not NULL) since DB column is NOT NULL DEFAULT '{}'.
func marshalI18n(m map[string]string) []byte {
	if len(m) == 0 {
		return []byte(`{}`)
	}
	b, err := json.Marshal(m)
	if err != nil {
		// Marshal of map[string]string never fails in practice; fall back to empty JSON to keep INSERT safe.
		return []byte(`{}`)
	}
	return b
}

func nullableIntArray(arr []int) interface{} {
	if len(arr) == 0 {
		return nil
	}
	return arr
}

// nullableStringArray 处理 TEXT[] 列：空切片 → NULL，否则原样下传。
// SerialNumberList 等 PG TEXT[] 类型字段统一走这里（migration 000124）。
func nullableStringArray(arr []string) interface{} {
	if len(arr) == 0 {
		return nil
	}
	return arr
}

func scanGroup(row pgx.Row) (*DeviceGroup, error) {
	var g DeviceGroup
	var (
		parentID         sql.NullString
		carrier          sql.NullString
		description      sql.NullString
		remark           sql.NullString
		createdBy        sql.NullString
		updatedBy        sql.NullString
		matchingMode     sql.NullString
		nameRuleList     []byte
		lacList          []int
		tacList          []int
		serialNumberList []string // migration 000124
		sourceGroupID    sql.NullString
		nameI18n         []byte // migration 000003 i18n columns
		descI18n         []byte
		remarkI18n       []byte
	)

	err := row.Scan(
		&g.ID, &g.Name, &parentID, &carrier, &description,
		&g.SortOrder, &g.Level, &g.Status, &g.IsDefault,
		&remark, &createdBy, &updatedBy,
		&g.CreatedAt, &g.UpdatedAt,
		&matchingMode, &nameRuleList, &lacList, &tacList, &serialNumberList, &sourceGroupID,
		&nameI18n, &descI18n, &remarkI18n,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("scan group row: %w", err)
	}
	if len(nameI18n) > 0 {
		_ = json.Unmarshal(nameI18n, &g.NameI18n)
	}
	if len(descI18n) > 0 {
		_ = json.Unmarshal(descI18n, &g.DescriptionI18n)
	}
	if len(remarkI18n) > 0 {
		_ = json.Unmarshal(remarkI18n, &g.RemarkI18n)
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
	if matchingMode.Valid {
		g.MatchingMode = MatchingMode(matchingMode.String)
	}
	if sourceGroupID.Valid {
		id, _ := uuid.Parse(sourceGroupID.String)
		g.SourceGroupID = &id
	}
	if len(nameRuleList) > 0 {
		if err := json.Unmarshal(nameRuleList, &g.NameRuleList); err != nil {
			return nil, fmt.Errorf("unmarshal name_rule_list: %w", err)
		}
	}
	g.LACList = lacList
	g.TACList = tacList
	g.SerialNumberList = serialNumberList

	return &g, nil
}

func scanGroups(rows pgx.Rows) ([]DeviceGroup, error) {
	var items []DeviceGroup
	for rows.Next() {
		var g DeviceGroup
		var (
			parentID         sql.NullString
			carrier          sql.NullString
			description      sql.NullString
			remark           sql.NullString
			createdBy        sql.NullString
			updatedBy        sql.NullString
			matchingMode     sql.NullString
			nameRuleList     []byte
			lacList          []int
			tacList          []int
			serialNumberList []string // migration 000124
			sourceGroupID    sql.NullString
			nameI18n         []byte // migration 000003 i18n columns
			descI18n         []byte
			remarkI18n       []byte
		)

		err := rows.Scan(
			&g.ID, &g.Name, &parentID, &carrier, &description,
			&g.SortOrder, &g.Level, &g.Status, &g.IsDefault,
			&remark, &createdBy, &updatedBy,
			&g.CreatedAt, &g.UpdatedAt,
			&matchingMode, &nameRuleList, &lacList, &tacList, &serialNumberList, &sourceGroupID,
			&nameI18n, &descI18n, &remarkI18n,
		)
		if err != nil {
			return nil, fmt.Errorf("scan group row: %w", err)
		}
		if len(nameI18n) > 0 {
			_ = json.Unmarshal(nameI18n, &g.NameI18n)
		}
		if len(descI18n) > 0 {
			_ = json.Unmarshal(descI18n, &g.DescriptionI18n)
		}
		if len(remarkI18n) > 0 {
			_ = json.Unmarshal(remarkI18n, &g.RemarkI18n)
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
		if matchingMode.Valid {
			g.MatchingMode = MatchingMode(matchingMode.String)
		}
		if sourceGroupID.Valid {
			id, _ := uuid.Parse(sourceGroupID.String)
			g.SourceGroupID = &id
		}
		if len(nameRuleList) > 0 {
			if err := json.Unmarshal(nameRuleList, &g.NameRuleList); err != nil {
				return nil, fmt.Errorf("unmarshal name_rule_list: %w", err)
			}
		}
		g.LACList = lacList
		g.TACList = tacList
		g.SerialNumberList = serialNumberList

		items = append(items, g)
	}
	return items, rows.Err()
}

func scanGroupsWithCount(rows pgx.Rows) ([]DeviceGroup, error) {
	var items []DeviceGroup
	for rows.Next() {
		var g DeviceGroup
		var (
			parentID         sql.NullString
			carrier          sql.NullString
			description      sql.NullString
			remark           sql.NullString
			createdBy        sql.NullString
			updatedBy        sql.NullString
			matchingMode     sql.NullString
			nameRuleList     []byte
			lacList          []int
			tacList          []int
			serialNumberList []string // migration 000124
			sourceGroupID    sql.NullString
			nameI18n         []byte // migration 000003 i18n columns
			descI18n         []byte
			remarkI18n       []byte
		)

		err := rows.Scan(
			&g.ID, &g.Name, &parentID, &carrier, &description,
			&g.SortOrder, &g.Level, &g.Status, &g.IsDefault,
			&remark, &createdBy, &updatedBy,
			&g.CreatedAt, &g.UpdatedAt,
			&matchingMode, &nameRuleList, &lacList, &tacList, &serialNumberList, &sourceGroupID,
			&g.DeviceCount,
			&nameI18n, &descI18n, &remarkI18n,
		)
		if err != nil {
			return nil, fmt.Errorf("scan group with count: %w", err)
		}
		if len(nameI18n) > 0 {
			_ = json.Unmarshal(nameI18n, &g.NameI18n)
		}
		if len(descI18n) > 0 {
			_ = json.Unmarshal(descI18n, &g.DescriptionI18n)
		}
		if len(remarkI18n) > 0 {
			_ = json.Unmarshal(remarkI18n, &g.RemarkI18n)
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
		if matchingMode.Valid {
			g.MatchingMode = MatchingMode(matchingMode.String)
		}
		if sourceGroupID.Valid {
			id, _ := uuid.Parse(sourceGroupID.String)
			g.SourceGroupID = &id
		}
		if len(nameRuleList) > 0 {
			if err := json.Unmarshal(nameRuleList, &g.NameRuleList); err != nil {
				return nil, fmt.Errorf("unmarshal name_rule_list: %w", err)
			}
		}
		g.LACList = lacList
		g.TACList = tacList
		g.SerialNumberList = serialNumberList

		items = append(items, g)
	}
	return items, rows.Err()
}

// UpdateBoundRule 更新分组的绑定规则
func (r *PgDeviceGroupRepository) UpdateBoundRule(ctx context.Context, groupID, ruleID uuid.UUID) error {
	query, args, err := storage.Psql.Update("device_groups").
		Set("bound_rule_id", ruleID).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": groupID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update bound rule SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update bound rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("group not found: %s", groupID)
	}

	return nil
}

// ClearBoundRule 清除分组的绑定规则
func (r *PgDeviceGroupRepository) ClearBoundRule(ctx context.Context, groupID uuid.UUID) error {
	query, args, err := storage.Psql.Update("device_groups").
		Set("bound_rule_id", nil).
		Set("updated_at", time.Now()).
		Where(sq.Eq{"id": groupID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build clear bound rule SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("clear bound rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("group not found: %s", groupID)
	}

	return nil
}

var _ DeviceGroupRepository = (*PgDeviceGroupRepository)(nil)
