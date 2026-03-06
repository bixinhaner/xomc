package admin

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	commonerrors "github.com/omcgo/omcgo/internal/common/errors"
)

// PgRoleRepository implements RoleRepository using PostgreSQL.
type PgRoleRepository struct {
	pool *pgxpool.Pool
}

var _ RoleRepository = (*PgRoleRepository)(nil)

// NewPgRoleRepository creates a new PgRoleRepository.
func NewPgRoleRepository(pool *pgxpool.Pool) *PgRoleRepository {
	return &PgRoleRepository{pool: pool}
}

func (r *PgRoleRepository) Create(ctx context.Context, role *Role) error {
	if role.ID == uuid.Nil {
		role.ID = uuid.New()
	}
	now := time.Now()
	role.CreatedAt = now
	role.UpdatedAt = now

	query, args, err := psql.Insert("roles").
		Columns("id", "name", "description", "is_system", "created_at", "updated_at").
		Values(role.ID, role.Name, role.Description, role.IsSystem, role.CreatedAt, role.UpdatedAt).
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
	query, args, err := psql.Select("id", "name", "description", "is_system", "created_at", "updated_at").
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
	query, args, err := psql.Select("id", "name", "description", "is_system", "created_at", "updated_at").
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
	role.UpdatedAt = time.Now()

	query, args, err := psql.Update("roles").
		Set("name", role.Name).
		Set("description", role.Description).
		Set("updated_at", role.UpdatedAt).
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
	checkQuery, checkArgs, _ := psql.Select("is_system").From("roles").Where(sq.Eq{"id": id}).ToSql()
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

	query, args, err := psql.Delete("roles").Where(sq.Eq{"id": id}).ToSql()
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
	query, args, err := psql.Select("id", "name", "description", "is_system", "created_at", "updated_at").
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

func (r *PgRoleRepository) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error {
	query, args, err := psql.Insert("user_roles").
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
	return nil
}

func (r *PgRoleRepository) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error {
	query, args, err := psql.Delete("user_roles").
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
	return nil
}

func (r *PgRoleRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]Role, error) {
	query, args, err := psql.Select("r.id", "r.name", "r.description", "r.is_system", "r.created_at", "r.updated_at").
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
	query, args, err := psql.Select("id", "role_id", "resource", "action").
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

func (r *PgRoleRepository) CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error) {
	query, args, err := psql.Select("COUNT(*)").
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
