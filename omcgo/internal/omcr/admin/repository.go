package admin

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/common/model"
)

// UserRepository defines the persistence interface for users.
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Update(ctx context.Context, user *User) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter UserFilter) (*model.ListResponse[User], error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

// RoleRepository defines the persistence interface for roles and permissions.
type RoleRepository interface {
	Create(ctx context.Context, role *Role) error
	GetByID(ctx context.Context, id uuid.UUID) (*Role, error)
	GetByName(ctx context.Context, name string) (*Role, error)
	Update(ctx context.Context, role *Role) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context) ([]Role, error)
	AssignRole(ctx context.Context, userID, roleID uuid.UUID) error
	RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]Role, error)
	GetPermissions(ctx context.Context, roleID uuid.UUID) ([]Permission, error)
	CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error)
	ListAllPermissions(ctx context.Context) ([]Permission, error)
	AddPermissions(ctx context.Context, roleID uuid.UUID, perms []Permission) error
	RemoveAllPermissions(ctx context.Context, roleID uuid.UUID) error
}

// AuditRepository defines the persistence interface for audit logs.
type AuditRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	List(ctx context.Context, filter AuditLogFilter) (*model.ListResponse[AuditLog], error)
}
