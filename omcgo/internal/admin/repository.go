package admin

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
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

// RoleReader provides read-only access to roles.
type RoleReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*Role, error)
	GetByName(ctx context.Context, name string) (*Role, error)
	List(ctx context.Context) ([]Role, error)
}

// RoleWriter provides write operations for roles.
type RoleWriter interface {
	Create(ctx context.Context, role *Role) error
	Update(ctx context.Context, role *Role) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// RoleAssigner manages user-role assignments.
type RoleAssigner interface {
	AssignRole(ctx context.Context, userID, roleID uuid.UUID) error
	RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]Role, error)
}

// PermissionChecker provides permission query capabilities.
type PermissionChecker interface {
	CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error)
	GetPermissions(ctx context.Context, roleID uuid.UUID) ([]Permission, error)
	ListAllPermissions(ctx context.Context) ([]Permission, error)
}

// PermissionWriter provides write operations for permissions.
type PermissionWriter interface {
	AddPermissions(ctx context.Context, roleID uuid.UUID, perms []Permission) error
	RemoveAllPermissions(ctx context.Context, roleID uuid.UUID) error
}

// RoleRepository defines the full persistence interface for roles and permissions.
// It composes smaller interfaces for backward compatibility.
type RoleRepository interface {
	RoleReader
	RoleWriter
	RoleAssigner
	PermissionChecker
	PermissionWriter
}

// AuditRepository defines the persistence interface for audit logs.
type AuditRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	List(ctx context.Context, filter AuditLogFilter) (*model.ListResponse[AuditLog], error)
}
