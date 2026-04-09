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
	GetUserRolesBatch(ctx context.Context, userIds []uuid.UUID) (map[uuid.UUID][]Role, error)
	GetDefaultRoleID(ctx context.Context, userID uuid.UUID) (*uuid.UUID, error)
	SetDefaultRole(ctx context.Context, userID, roleID uuid.UUID) error
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
	// ListWithPagination returns roles with pagination support.
	ListWithPagination(ctx context.Context, filter RoleFilter) (*model.ListResponse[Role], error)
}

// RoleDeviceGroupRepository manages role–device-group associations.
type RoleDeviceGroupRepository interface {
	GetGroupIDs(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)
	SetGroupIDs(ctx context.Context, roleID uuid.UUID, groupIDs []uuid.UUID) error
	GetUserVisibleGroupIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

// MenuRepository defines the persistence interface for menus.
type MenuRepository interface {
	Create(ctx context.Context, menu *Menu, operatorID uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*Menu, error)
	GetByPermissionKey(ctx context.Context, key string) (*Menu, error)
	List(ctx context.Context, filter MenuFilter) (*model.ListResponse[Menu], error)
	Update(ctx context.Context, id uuid.UUID, req *UpdateMenuRequest, operatorID uuid.UUID) error
	Delete(ctx context.Context, ids []uuid.UUID) error
	GetTree(ctx context.Context, status *MenuStatus) ([]Menu, error)
	GetByRole(ctx context.Context, roleID uuid.UUID) ([]Menu, error)
	GetByUser(ctx context.Context, userID uuid.UUID) ([]Menu, error)
	SetRoleMenus(ctx context.Context, roleID uuid.UUID, menuIDs []uuid.UUID, operatorID uuid.UUID) error
	GetRoleMenuIDs(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)
}

// AuditRepository defines the persistence interface for audit logs.
type AuditRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	List(ctx context.Context, filter AuditLogFilter) (*model.ListResponse[AuditLog], error)
}
