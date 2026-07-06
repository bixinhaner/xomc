package admin

import (
	"context"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/core/errors"
	"github.com/omcgo/omcgo/internal/core/model"
	"github.com/omcgo/omcgo/internal/core/storage"
)

// roleColumns: v0.6 起加入 code / created_by / updated_by。
// 顺序须与 scanRole / scanRoleFromRows 中 Scan 参数一致。
var roleColumns = []string{
	"id", "name", "code", "description", "is_system",
	"created_by", "updated_by", "created_at", "updated_at",
}

// PgRoleRepository implements RoleRepository using PostgreSQL.
type PgRoleRepository struct {
	pool       *pgxpool.Pool
	authorizer *CasbinAuthorizer // optional: set via SetAuthorizer
}

var (
	_ RoleRepository            = (*PgRoleRepository)(nil)
	_ RoleDeviceGroupRepository = (*PgRoleRepository)(nil)
)

// NewPgRoleRepository creates a new PgRoleRepository.
func NewPgRoleRepository(pool *pgxpool.Pool) *PgRoleRepository {
	return &PgRoleRepository{pool: pool}
}

// SetAuthorizer sets the Casbin authorizer for in-memory permission checks.
func (r *PgRoleRepository) SetAuthorizer(auth *CasbinAuthorizer) {
	r.authorizer = auth
}

func (r *PgRoleRepository) Create(ctx context.Context, role *Role) error {
	if role.ID == uuid.Nil {
		role.ID = uuid.New()
	}
	now := time.Now()
	role.CreatedAt = now
	role.UpdatedAt = now

	query, args, err := storage.Psql.Insert("roles").
		Columns(roleColumns...).
		Values(
			role.ID, role.Name, nullableString(role.Code), role.Description, role.IsSystem,
			nullableUUID(role.CreatedBy), nullableUUID(role.UpdatedBy),
			role.CreatedAt, role.UpdatedAt,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build insert role SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert role: %w", err)
	}
	return nil
}

// scanRoleFromRow scan a single row from QueryRow result into Role, handling NULL columns.
func scanRoleFromRow(row pgx.Row, role *Role) error {
	var code *string
	return row.Scan(
		&role.ID, &role.Name, &code, &role.Description, &role.IsSystem,
		&role.CreatedBy, &role.UpdatedBy, &role.CreatedAt, &role.UpdatedAt,
	)
}

func applyRoleNullables(role *Role, code *string) {
	if code != nil {
		role.Code = *code
	}
}

func (r *PgRoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	query, args, err := storage.Psql.Select(roleColumns...).
		From("roles").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get role SQL: %w", err)
	}

	var role Role
	var code *string
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&role.ID, &role.Name, &code, &role.Description, &role.IsSystem,
		&role.CreatedBy, &role.UpdatedBy, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get role: %w", err)
	}
	applyRoleNullables(&role, code)

	// B3-Phase2-B：permissions 表已 DROP，不再回填 role.Permissions（保持空切片）。
	// 前端 P3 完成后将彻底移除该字段。

	// 与列表接口对齐，让单角色详情也带 DeviceGroupIDs / UserCount（前端编辑面板可一次拿全）
	roles := []Role{role}
	if err := r.populateDeviceGroupIDs(ctx, roles); err != nil {
		return nil, fmt.Errorf("populate role device groups: %w", err)
	}
	if err := r.populateUserCounts(ctx, roles); err != nil {
		return nil, fmt.Errorf("populate role user counts: %w", err)
	}
	role = roles[0]

	return &role, nil
}

func (r *PgRoleRepository) GetByName(ctx context.Context, name string) (*Role, error) {
	query, args, err := storage.Psql.Select(roleColumns...).
		From("roles").
		Where(sq.Eq{"name": name}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get role by name SQL: %w", err)
	}

	var role Role
	var code *string
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&role.ID, &role.Name, &code, &role.Description, &role.IsSystem,
		&role.CreatedBy, &role.UpdatedBy, &role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get role by name: %w", err)
	}
	applyRoleNullables(&role, code)
	return &role, nil
}

func (r *PgRoleRepository) Update(ctx context.Context, role *Role) error {
	now := time.Now()
	role.UpdatedAt = now

	query, args, err := storage.Psql.Update("roles").
		Set("name", role.Name).
		Set("code", nullableString(role.Code)).
		Set("description", role.Description).
		Set("updated_by", nullableUUID(role.UpdatedBy)).
		Set("updated_at", now).
		Where(sq.Eq{"id": role.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build update role SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}
	return nil
}

func (r *PgRoleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Prevent deleting system roles
	var isSystem bool
	checkQuery, checkArgs, _ := storage.Psql.Select("is_system").From("roles").Where(sq.Eq{"id": id}).ToSql()
	err := r.pool.QueryRow(ctx, checkQuery, checkArgs...).Scan(&isSystem)
	if err != nil {
		if err == pgx.ErrNoRows {
			return commonerrors.ErrNotFound
		}
		return fmt.Errorf("check system role: %w", err)
	}
	if isSystem {
		return commonerrors.ErrForbidden
	}

	query, args, err := storage.Psql.Delete("roles").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("build delete role SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	return nil
}

func (r *PgRoleRepository) List(ctx context.Context) ([]Role, error) {
	query, args, err := storage.Psql.Select(roleColumns...).
		From("roles").
		OrderBy("name ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list roles SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		var code *string
		if err := rows.Scan(
			&role.ID, &role.Name, &code, &role.Description, &role.IsSystem,
			&role.CreatedBy, &role.UpdatedBy, &role.CreatedAt, &role.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		applyRoleNullables(&role, code)
		roles = append(roles, role)
	}
	if err := r.populateDeviceGroupIDs(ctx, roles); err != nil {
		return nil, fmt.Errorf("populate role device groups: %w", err)
	}
	if err := r.populateUserCounts(ctx, roles); err != nil {
		return nil, fmt.Errorf("populate role user counts: %w", err)
	}
	return roles, nil
}

// populateUserCounts 一次性批量补全 roles 列表的 UserCount 字段（§7 P0 #2）。
// 单 SQL 聚合查询：SELECT role_id, COUNT(*) GROUP BY role_id WHERE role_id = ANY($1)。
func (r *PgRoleRepository) populateUserCounts(ctx context.Context, roles []Role) error {
	if len(roles) == 0 {
		return nil
	}
	roleIDs := make([]uuid.UUID, len(roles))
	for i, role := range roles {
		roleIDs[i] = role.ID
	}

	rows, err := r.pool.Query(ctx,
		`SELECT role_id, COUNT(*) FROM user_roles WHERE role_id = ANY($1) GROUP BY role_id`, roleIDs)
	if err != nil {
		return fmt.Errorf("query user_roles count: %w", err)
	}
	defer rows.Close()

	countByRole := make(map[uuid.UUID]int, len(roles))
	for rows.Next() {
		var roleID uuid.UUID
		var cnt int
		if err := rows.Scan(&roleID, &cnt); err != nil {
			return fmt.Errorf("scan user_roles count: %w", err)
		}
		countByRole[roleID] = cnt
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for i := range roles {
		roles[i].UserCount = countByRole[roles[i].ID]
	}
	return nil
}

// populateDeviceGroupIDs 一次性批量补全 roles 列表的 DeviceGroupIDs 字段，
// 让前端 RolePermission 列表 / users 管理「未绑分组角色 ⚠️」判定（PRD §11.2 决议 ①）准确。
//
// 仅取 group_id（不取 network_types，network_types 仍由专项端点
// `GET /admin/roles/{id}/device-groups` 单独返回，避免 list 响应膨胀）。
func (r *PgRoleRepository) populateDeviceGroupIDs(ctx context.Context, roles []Role) error {
	if len(roles) == 0 {
		return nil
	}
	roleIDs := make([]uuid.UUID, len(roles))
	for i, role := range roles {
		roleIDs[i] = role.ID
	}

	rows, err := r.pool.Query(ctx,
		`SELECT role_id, group_id FROM role_device_groups WHERE role_id = ANY($1)`, roleIDs)
	if err != nil {
		return fmt.Errorf("query role_device_groups: %w", err)
	}
	defer rows.Close()

	groupsByRole := make(map[uuid.UUID][]uuid.UUID, len(roles))
	for rows.Next() {
		var roleID, groupID uuid.UUID
		if err := rows.Scan(&roleID, &groupID); err != nil {
			return fmt.Errorf("scan role_device_groups: %w", err)
		}
		groupsByRole[roleID] = append(groupsByRole[roleID], groupID)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for i := range roles {
		if gids := groupsByRole[roles[i].ID]; gids != nil {
			roles[i].DeviceGroupIDs = gids
		}
	}
	return nil
}

func (r *PgRoleRepository) ListWithPagination(ctx context.Context, filter RoleFilter) (*model.ListResponse[Role], error) {
	// Build filter conditions first so COUNT 与数据查询共用同一套 WHERE，
	// 避免 total 落到无过滤全量（issue #135）。
	where := sq.And{}
	if filter.Name != nil && *filter.Name != "" {
		where = append(where, sq.Expr("name ILIKE ?", ilikePattern(*filter.Name)))
	}
	// Search 模糊匹配 name 或 description（form:"search"，此前被完全忽略，issue #135）。
	if filter.Search != nil && *filter.Search != "" {
		pat := ilikePattern(*filter.Search)
		where = append(where, sq.Or{
			sq.Expr("name ILIKE ?", pat),
			sq.Expr("description ILIKE ?", pat),
		})
	}

	// Count query — 带与数据查询一致的过滤条件
	countQuery, countArgs, err := storage.Psql.Select("COUNT(*)").
		From("roles").
		Where(where).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count roles SQL: %w", err)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count roles: %w", err)
	}

	offset := filter.Offset()
	limit := filter.Limit()

	// Data query
	query, args, err := storage.Psql.Select(roleColumns...).
		From("roles").
		Where(where).
		OrderBy("name ASC").
		Limit(uint64(limit)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list roles SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()

	var items []Role
	for rows.Next() {
		var role Role
		var code *string
		if err := rows.Scan(
			&role.ID, &role.Name, &code, &role.Description, &role.IsSystem,
			&role.CreatedBy, &role.UpdatedBy, &role.CreatedAt, &role.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		applyRoleNullables(&role, code)
		items = append(items, role)
	}
	if err := r.populateDeviceGroupIDs(ctx, items); err != nil {
		return nil, fmt.Errorf("populate role device groups: %w", err)
	}
	if err := r.populateUserCounts(ctx, items); err != nil {
		return nil, fmt.Errorf("populate role user counts: %w", err)
	}

	return model.NewListResponse(items, total, filter.Page, filter.PageSize), nil
}

func (r *PgRoleRepository) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	query, args, err := storage.Psql.Insert("user_roles").
		Columns("user_id", "role_id").
		Values(userID, roleID).
		Suffix("ON CONFLICT DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("build assign role SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("assign role: %w", err)
	}

	if r.authorizer != nil {
		_ = r.authorizer.NotifyPolicyChange()
	}
	return nil
}

func (r *PgRoleRepository) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error {
	query, args, err := storage.Psql.Delete("user_roles").
		Where(sq.And{sq.Eq{"user_id": userID}, sq.Eq{"role_id": roleID}}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build remove role SQL: %w", err)
	}

	tag, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("remove role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.ErrNotFound
	}

	if r.authorizer != nil {
		_ = r.authorizer.NotifyPolicyChange()
	}
	return nil
}

func (r *PgRoleRepository) GetDefaultRoleID(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error) {
	query, args, err := storage.Psql.Select("role_id").
		From("user_roles").
		Where(sq.And{sq.Eq{"user_id": userID}, sq.Eq{"is_default": true}}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get default role SQL: %w", err)
	}

	var roleID uuid.UUID
	err = r.pool.QueryRow(ctx, query, args...).Scan(&roleID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get default role: %w", err)
	}
	return &roleID, nil
}

func (r *PgRoleRepository) SetDefaultRole(ctx context.Context, userID, roleID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Clear existing default
	_, err = tx.Exec(ctx, `UPDATE user_roles SET is_default = false WHERE user_id = $1 AND is_default = true`, userID)
	if err != nil {
		return fmt.Errorf("clear default role: %w", err)
	}

	// Set new default
	tag, err := tx.Exec(ctx, `UPDATE user_roles SET is_default = true WHERE user_id = $1 AND role_id = $2`, userID, roleID)
	if err != nil {
		return fmt.Errorf("set default role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return commonerrors.NewBusinessError(7003, "role not assigned to user", commonerrors.ErrNotFound)
	}

	return tx.Commit(ctx)
}

// ListUserIDsByRole 返回当前持有该角色的所有用户 ID（无分页）。
// PRD docs/prd/system/roles.md §10 DoD：角色侧写操作后用此结果遍历调
// PermissionInvalidator.InvalidateUserCache 失效缓存。
func (r *PgRoleRepository) ListUserIDsByRole(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	query, args, err := storage.Psql.Select("user_id").
		From("user_roles").
		Where(sq.Eq{"role_id": roleID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list users by role SQL: %w", err)
	}
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query users by role: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan user id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users by role: %w", err)
	}
	return ids, nil
}

func (r *PgRoleRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]Role, error) {
	query, args, err := storage.Psql.Select("r.id", "r.name", "r.description", "r.is_system", "r.created_at", "r.updated_at").
		From("roles r").
		Join("user_roles ur ON ur.role_id = r.id").
		Where(sq.Eq{"ur.user_id": userID}).
		OrderBy("r.name ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get user roles SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get user roles: %w", err)
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, nil
}

// GetPermissions 已 deprecated（B3-Phase2-B 起 permissions 表 DROP）。
// 保留方法签名以兼容 PermissionChecker 接口，永远返回空切片。
// 待前端 P3 完成 + 接口最终清理后整段移除。
func (r *PgRoleRepository) GetPermissions(_ context.Context, _ uuid.UUID) ([]Permission, error) {
	return []Permission{}, nil
}

// ListAllPermissions 已 deprecated（B3-Phase2-B 起）。永远返回空切片。
func (r *PgRoleRepository) ListAllPermissions(_ context.Context) ([]Permission, error) {
	return []Permission{}, nil
}

// AddPermissions 已 deprecated（B3-Phase2-B 起）。no-op，不报错避免影响 service 流程。
func (r *PgRoleRepository) AddPermissions(_ context.Context, _ uuid.UUID, _ []Permission) error {
	return nil
}

// RemoveAllPermissions 已 deprecated（B3-Phase2-B 起）。no-op。
func (r *PgRoleRepository) RemoveAllPermissions(_ context.Context, _ uuid.UUID) error {
	return nil
}

// --- RoleDeviceGroupRepository methods ---

func (r *PgRoleRepository) GetGroupIDs(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	query, args, err := storage.Psql.Select("group_id").
		From("role_device_groups").
		Where(sq.Eq{"role_id": roleID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get group IDs SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get role group IDs: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan group ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *PgRoleRepository) SetGroupIDs(ctx context.Context, roleID uuid.UUID, groupIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete existing associations.
	delQuery, delArgs, err := storage.Psql.Delete("role_device_groups").
		Where(sq.Eq{"role_id": roleID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete role groups SQL: %w", err)
	}
	if _, err := tx.Exec(ctx, delQuery, delArgs...); err != nil {
		return fmt.Errorf("delete role groups: %w", err)
	}

	// Insert new associations.
	if len(groupIDs) > 0 {
		builder := storage.Psql.Insert("role_device_groups").
			Columns("role_id", "group_id")
		for _, gid := range groupIDs {
			builder = builder.Values(roleID, gid)
		}
		insQuery, insArgs, err := builder.ToSql()
		if err != nil {
			return fmt.Errorf("build insert role groups SQL: %w", err)
		}
		if _, err := tx.Exec(ctx, insQuery, insArgs...); err != nil {
			return fmt.Errorf("insert role groups: %w", err)
		}
	}

	return tx.Commit(ctx)
}

// ListRolesByGroupIDs 查询绑定了任一指定 group ID 的角色（去重）。
// PRD users.md §11.7 决议③ / roles.md §11.4：删除设备分组前调用 → 写审计 +
// 失效角色下用户的可见域缓存。返回的 Role 仅含 ID + Name（其它字段未填）。
func (r *PgRoleRepository) ListRolesByGroupIDs(ctx context.Context, groupIDs []uuid.UUID) ([]Role, error) {
	if len(groupIDs) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT DISTINCT r.id, r.name
		 FROM roles r
		 JOIN role_device_groups rdg ON rdg.role_id = r.id
		 WHERE rdg.group_id = ANY($1)`, groupIDs)
	if err != nil {
		return nil, fmt.Errorf("list roles by group ids: %w", err)
	}
	defer rows.Close()
	roles := make([]Role, 0)
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.Name); err != nil {
			return nil, fmt.Errorf("scan role row: %w", err)
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate roles by group ids: %w", err)
	}
	return roles, nil
}

// GetDeviceGroupData returns group IDs and network_types for a role.
func (r *PgRoleRepository) GetDeviceGroupData(ctx context.Context, roleID uuid.UUID) (*RoleDeviceGroupData, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT group_id, COALESCE(network_types, '{}') FROM role_device_groups WHERE role_id = $1`,
		roleID)
	if err != nil {
		return nil, fmt.Errorf("get role device group data: %w", err)
	}
	defer rows.Close()

	data := &RoleDeviceGroupData{
		GroupIDs:     []uuid.UUID{},
		NetworkTypes: []string{},
	}
	networkTypesSet := make(map[string]struct{})

	for rows.Next() {
		var groupID uuid.UUID
		var networkTypes []string
		if err := rows.Scan(&groupID, &networkTypes); err != nil {
			return nil, fmt.Errorf("scan device group data: %w", err)
		}
		data.GroupIDs = append(data.GroupIDs, groupID)
		for _, nt := range networkTypes {
			networkTypesSet[nt] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for nt := range networkTypesSet {
		data.NetworkTypes = append(data.NetworkTypes, nt)
	}
	return data, nil
}

// SetDeviceGroupData replaces group IDs and network_types for a role.
func (r *PgRoleRepository) SetDeviceGroupData(ctx context.Context, roleID uuid.UUID, data RoleDeviceGroupData) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM role_device_groups WHERE role_id = $1`, roleID)
	if err != nil {
		return fmt.Errorf("delete role device groups: %w", err)
	}

	if len(data.GroupIDs) > 0 {
		for _, gid := range data.GroupIDs {
			_, err = tx.Exec(ctx,
				`INSERT INTO role_device_groups (role_id, group_id, network_types) VALUES ($1, $2, $3)`,
				roleID, gid, data.NetworkTypes,
			)
			if err != nil {
				return fmt.Errorf("insert role device group: %w", err)
			}
		}
	}

	return tx.Commit(ctx)
}

// GetRoleApiEndpoints returns (path, method) pairs for the given role names.
func (r *PgRoleRepository) GetRoleApiEndpoints(ctx context.Context, roleNames []string) ([]RoleApiEndpoint, error) {
	if len(roleNames) == 0 {
		return nil, nil
	}

	const rawSQL = `
		SELECT ae.path, ae.method
		FROM role_api_permissions rap
		JOIN api_endpoints ae ON ae.id = rap.endpoint_id
		JOIN roles r ON r.id = rap.role_id
		WHERE r.name = ANY($1)`

	rows, err := r.pool.Query(ctx, rawSQL, roleNames)
	if err != nil {
		return nil, fmt.Errorf("get role api endpoints: %w", err)
	}
	defer rows.Close()

	var endpoints []RoleApiEndpoint
	for rows.Next() {
		var ep RoleApiEndpoint
		if err := rows.Scan(&ep.Path, &ep.Method); err != nil {
			return nil, fmt.Errorf("scan role api endpoint: %w", err)
		}
		endpoints = append(endpoints, ep)
	}
	return endpoints, rows.Err()
}

// GetRoleApiEndpointIDs returns endpoint IDs granted to a role.
func (r *PgRoleRepository) GetRoleApiEndpointIDs(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT endpoint_id FROM role_api_permissions WHERE role_id = $1 ORDER BY endpoint_id`,
		roleID)
	if err != nil {
		return nil, fmt.Errorf("get role api endpoint IDs: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan endpoint ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// SetRoleApiEndpoints replaces the full set of API endpoint grants for a role.
func (r *PgRoleRepository) SetRoleApiEndpoints(ctx context.Context, roleID uuid.UUID, endpointIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM role_api_permissions WHERE role_id = $1`, roleID)
	if err != nil {
		return fmt.Errorf("delete role api permissions: %w", err)
	}

	for _, epID := range endpointIDs {
		_, err = tx.Exec(ctx,
			`INSERT INTO role_api_permissions (role_id, endpoint_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			roleID, epID,
		)
		if err != nil {
			return fmt.Errorf("insert role api permission: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PgRoleRepository) GetUserVisibleGroupIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	const rawSQL = `
		SELECT DISTINCT rdg.group_id
		FROM user_roles ur
		JOIN role_device_groups rdg ON rdg.role_id = ur.role_id
		WHERE ur.user_id = $1`

	rows, err := r.pool.Query(ctx, rawSQL, userID)
	if err != nil {
		return nil, fmt.Errorf("get user visible group IDs: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan visible group ID: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *PgRoleRepository) GetUserVisibleDeviceGrants(ctx context.Context, userID uuid.UUID) ([]model.DeviceVisibilityGrant, error) {
	const rawSQL = `
		SELECT rdg.group_id, COALESCE(rdg.network_types, '{}')
		FROM user_roles ur
		JOIN role_device_groups rdg ON rdg.role_id = ur.role_id
		WHERE ur.user_id = $1`

	rows, err := r.pool.Query(ctx, rawSQL, userID)
	if err != nil {
		return nil, fmt.Errorf("get user visible device grants: %w", err)
	}
	defer rows.Close()

	grants := make([]model.DeviceVisibilityGrant, 0)
	for rows.Next() {
		var groupID uuid.UUID
		var networkTypes []string
		if err := rows.Scan(&groupID, &networkTypes); err != nil {
			return nil, fmt.Errorf("scan visible device grant: %w", err)
		}
		grants = append(grants, model.DeviceVisibilityGrant{
			GroupIDs:     []uuid.UUID{groupID},
			Technologies: normalizeNetworkTypesForDeviceVisibility(networkTypes),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate visible device grants: %w", err)
	}
	return grants, nil
}

func normalizeNetworkTypesForDeviceVisibility(networkTypes []string) []model.Technology {
	if len(networkTypes) == 0 {
		return nil
	}
	techSet := make(map[model.Technology]struct{})
	for _, rawType := range networkTypes {
		switch strings.ToLower(strings.TrimSpace(rawType)) {
		case "lte", "enb":
			techSet[model.TechLTE] = struct{}{}
		case "nr", "gnb":
			techSet[model.TechNR] = struct{}{}
		case "gsm":
			techSet[model.TechGSM] = struct{}{}
		}
	}
	if len(techSet) == 0 {
		return nil
	}
	techs := make([]model.Technology, 0, len(techSet))
	for tech := range techSet {
		techs = append(techs, tech)
	}
	return techs
}

// CheckPermission 通过 Casbin 检查 (path, method) 端点级权限。
// B3-Phase2-B 起 permissions 表已 DROP，无 SQL fallback；若 authorizer 未注入直接 false。
func (r *PgRoleRepository) CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	if r.authorizer == nil {
		return false, fmt.Errorf("casbin authorizer not configured")
	}
	return r.authorizer.CheckPermission(ctx, userID, resource, action)
}

// ListRoleUsers returns a paginated list of users assigned to a role.
func (r *PgRoleRepository) ListRoleUsers(ctx context.Context, roleID uuid.UUID, limit, offset int) ([]RoleUserItem, int64, error) {
	if limit <= 0 {
		limit = 20
	}

	var total int64
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM user_roles WHERE role_id = $1`, roleID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count role users: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT u.id, u.username, u.display_name, COALESCE(u.email, ''), u.status
		FROM user_roles ur
		JOIN users u ON u.id = ur.user_id
		WHERE ur.role_id = $1
		ORDER BY u.username ASC
		LIMIT $2 OFFSET $3`,
		roleID, limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list role users: %w", err)
	}
	defer rows.Close()

	var users []RoleUserItem
	for rows.Next() {
		var u RoleUserItem
		if err := rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Email, &u.Status); err != nil {
			return nil, 0, fmt.Errorf("scan role user: %w", err)
		}
		users = append(users, u)
	}
	if users == nil {
		users = []RoleUserItem{}
	}
	return users, total, rows.Err()
}

// GetUserRolesBatch 批量获取多个用户的角色
func (r *PgRoleRepository) GetUserRolesBatch(ctx context.Context, userIds []uuid.UUID) (map[uuid.UUID][]Role, error) {
	if len(userIds) == 0 {
		return make(map[uuid.UUID][]Role), nil
	}

	query, args, err := storage.Psql.Select("ur.user_id", "r.id", "r.name", "r.description", "r.is_system", "r.created_at", "r.updated_at").
		From("user_roles ur").
		Join("roles r ON ur.role_id = r.id").
		Where(sq.Eq{"ur.user_id": userIds}).
		OrderBy("r.name ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get user roles batch SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get user roles batch: %w", err)
	}
	defer rows.Close()

	// 初始化 map，确保没有角色的用户也会返回空数组
	result := make(map[uuid.UUID][]Role, len(userIds))
	for _, uid := range userIds {
		result[uid] = []Role{}
	}

	for rows.Next() {
		var userID uuid.UUID
		var role Role
		if err := rows.Scan(&userID, &role.ID, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user role: %w", err)
		}
		result[userID] = append(result[userID], role)
	}

	return result, rows.Err()
}
