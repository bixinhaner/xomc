package admin

import (
	"context"
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

	query, args, err := storage.Psql.Insert("roles").
		Columns("id", "name", "description", "is_system", "created_at", "updated_at").
		Values(role.ID, role.Name, role.Description, role.IsSystem, now, now).
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

func (r *PgRoleRepository) GetByID(ctx context.Context, id uuid.UUID) (*Role, error) {
	query, args, err := storage.Psql.Select("id", "name", "description", "is_system", "created_at", "updated_at").
		From("roles").
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get role SQL: %w", err)
	}

	var role Role
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&role.ID, &role.Name, &role.Description, &role.IsSystem,
		&role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get role: %w", err)
	}

	perms, err := r.GetPermissions(ctx, id)
	if err != nil {
		return nil, err
	}
	role.Permissions = perms
	return &role, nil
}

func (r *PgRoleRepository) GetByName(ctx context.Context, name string) (*Role, error) {
	query, args, err := storage.Psql.Select("id", "name", "description", "is_system", "created_at", "updated_at").
		From("roles").
		Where(sq.Eq{"name": name}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get role by name SQL: %w", err)
	}

	var role Role
	err = r.pool.QueryRow(ctx, query, args...).Scan(
		&role.ID, &role.Name, &role.Description, &role.IsSystem,
		&role.CreatedAt, &role.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, commonerrors.ErrNotFound
		}
		return nil, fmt.Errorf("get role by name: %w", err)
	}
	return &role, nil
}

func (r *PgRoleRepository) Update(ctx context.Context, role *Role) error {
	now := time.Now()

	query, args, err := storage.Psql.Update("roles").
		Set("name", role.Name).
		Set("description", role.Description).
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
	query, args, err := storage.Psql.Select("id", "name", "description", "is_system", "created_at", "updated_at").
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
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (r *PgRoleRepository) ListWithPagination(ctx context.Context, filter RoleFilter) (*model.ListResponse[Role], error) {
	// Count query
	countQuery, countArgs, err := storage.Psql.Select("COUNT(*)").
		From("roles").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build count roles SQL: %w", err)
	}

	var total int64
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count roles: %w", err)
	}

	// Build filter conditions
	where := sq.And{}
	if filter.Name != nil && *filter.Name != "" {
		where = append(where, sq.Expr("name ILIKE ?", ilikePattern(*filter.Name)))
	}

	offset := filter.Offset()
	limit := filter.Limit()

	// Data query
	query, args, err := storage.Psql.Select("id", "name", "description", "is_system", "created_at", "updated_at").
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
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		items = append(items, role)
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

func (r *PgRoleRepository) GetPermissions(ctx context.Context, roleID uuid.UUID) ([]Permission, error) {
	query, args, err := storage.Psql.Select("id", "role_id", "resource", "action").
		From("permissions").
		Where(sq.Eq{"role_id": roleID}).
		OrderBy("resource ASC", "action ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get permissions SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("get permissions: %w", err)
	}
	defer rows.Close()

	var perms []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.RoleID, &p.Resource, &p.Action); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		perms = append(perms, p)
	}
	return perms, nil
}

func (r *PgRoleRepository) ListAllPermissions(ctx context.Context) ([]Permission, error) {
	query, args, err := storage.Psql.Select("id", "role_id", "resource", "action").
		From("permissions").
		OrderBy("resource ASC", "action ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list all permissions SQL: %w", err)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list all permissions: %w", err)
	}
	defer rows.Close()

	var perms []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.RoleID, &p.Resource, &p.Action); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		perms = append(perms, p)
	}
	return perms, nil
}

func (r *PgRoleRepository) AddPermissions(ctx context.Context, roleID uuid.UUID, perms []Permission) error {
	if len(perms) == 0 {
		return nil
	}

	builder := storage.Psql.Insert("permissions").
		Columns("id", "role_id", "resource", "action")

	for _, p := range perms {
		id := p.ID
		if id == uuid.Nil {
			id = uuid.New()
		}
		builder = builder.Values(id, roleID, p.Resource, p.Action)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build add permissions SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("add permissions: %w", err)
	}

	if r.authorizer != nil {
		_ = r.authorizer.NotifyPolicyChange()
	}
	return nil
}

func (r *PgRoleRepository) RemoveAllPermissions(ctx context.Context, roleID uuid.UUID) error {
	query, args, err := storage.Psql.Delete("permissions").
		Where(sq.Eq{"role_id": roleID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build remove all permissions SQL: %w", err)
	}

	_, err = r.pool.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("remove all permissions: %w", err)
	}

	if r.authorizer != nil {
		_ = r.authorizer.NotifyPolicyChange()
	}
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

func (r *PgRoleRepository) CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	// Use Casbin in-memory evaluation when available
	if r.authorizer != nil {
		return r.authorizer.CheckPermission(ctx, userID, resource, action)
	}

	// Fallback: SQL query
	query, args, err := storage.Psql.Select("COUNT(*)").
		From("permissions p").
		Join("user_roles ur ON ur.role_id = p.role_id").
		Where(sq.And{
			sq.Eq{"ur.user_id": userID},
			sq.Eq{"p.resource": resource},
			sq.Eq{"p.action": action},
		}).
		ToSql()
	if err != nil {
		return false, fmt.Errorf("build check permission SQL: %w", err)
	}

	var count int64
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return false, fmt.Errorf("check permission: %w", err)
	}
	return count > 0, nil
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
