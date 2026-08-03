package admin

import (
	"context"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// UserRepository defines the persistence interface for users.
//
//go:generate go run go.uber.org/mock/mockgen -destination=mock_user_repository_test.go -package=admin . UserRepository
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	Update(ctx context.Context, user *User) error
	UpdatePassword(ctx context.Context, id uuid.UUID, passwordHash string) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter UserFilter) (*model.ListResponse[User], error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
	// GetUsernamesByIDs 按 ID 批量返回 username，缺失/未找到的 ID 不出现在结果里。
	// ListUsers 用此把 created_by / updated_by 反查成 username 注入响应，
	// 取代前端二次拉全量 /admin/users。
	GetUsernamesByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error)
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
	// ListUserIDsByRole 返回当前持有该角色的全部用户 ID（无分页）。
	// 用于 PRD docs/prd/system/roles.md §10 DoD：角色侧写操作（菜单/分组/API/删除）后
	// 同步失效该角色下所有用户的可见域缓存。
	ListUserIDsByRole(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)
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
//
//go:generate go run go.uber.org/mock/mockgen -destination=mock_role_repository_test.go -package=admin . RoleRepository
type RoleRepository interface {
	RoleReader
	RoleWriter
	RoleAssigner
	PermissionChecker
	PermissionWriter
	// ListWithPagination returns roles with pagination support.
	ListWithPagination(ctx context.Context, filter RoleFilter) (*model.ListResponse[Role], error)
	// ListRoleUsers returns a paginated list of users assigned to a role.
	ListRoleUsers(ctx context.Context, roleID uuid.UUID, limit, offset int) ([]RoleUserItem, int64, error)
}

// RoleDeviceGroupData holds a role's device group IDs and optional network type filters.
//
// v0.7（roles.md §11.8）：JSON tag `group_ids` 改为 `device_group_ids`，与前端
// `setRoleDeviceGroups` PUT body 与 `getRoleDeviceGroups` GET 响应解析一致。
// 之前 PUT 请求体中 `device_group_ids` 字段被静默忽略，导致 `role_device_groups` 表被空数组覆盖。
type RoleDeviceGroupData struct {
	GroupIDs     []uuid.UUID `json:"device_group_ids"`
	NetworkTypes []string    `json:"network_types"`
}

// RoleDeviceGroupRepository manages role–device-group associations.
type RoleDeviceGroupRepository interface {
	GetGroupIDs(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)
	SetGroupIDs(ctx context.Context, roleID uuid.UUID, groupIDs []uuid.UUID) error
	GetUserVisibleGroupIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	// GetUserVisibleDeviceGrants returns raw role_device_groups rows for a user.
	// PermissionService expands each grant to the role's device-group subtree and
	// applies network_types when resolving device visibility.
	GetUserVisibleDeviceGrants(ctx context.Context, userID uuid.UUID) ([]model.DeviceVisibilityGrant, error)
	// GetDeviceGroupData returns group IDs plus network_types for a role.
	GetDeviceGroupData(ctx context.Context, roleID uuid.UUID) (*RoleDeviceGroupData, error)
	// SetDeviceGroupData replaces group IDs and network_types for a role.
	SetDeviceGroupData(ctx context.Context, roleID uuid.UUID, data RoleDeviceGroupData) error
	// ListRolesByGroupIDs 返回绑定了任一指定 group ID 的角色（去重）。
	// PRD users.md §11.7 决议③ / roles.md §11.4：删除设备分组前查受影响角色。
	ListRolesByGroupIDs(ctx context.Context, groupIDs []uuid.UUID) ([]Role, error)
}

// RoleApiPermissionRepository manages role–API-endpoint associations.
type RoleApiPermissionRepository interface {
	// GetRoleApiEndpointIDs returns endpoint IDs granted to a role.
	GetRoleApiEndpointIDs(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)
	// SetRoleApiEndpoints replaces the full set of API endpoint grants for a role.
	SetRoleApiEndpoints(ctx context.Context, roleID uuid.UUID, endpointIDs []uuid.UUID) error
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
	// GetAllActive 返回全部 status='normal' 的菜单（含 directory/menu/button），
	// 给超管旁路用（user.source='builtIn'）。参 docs/prd/system/menu-dynamic-loading.md §4.2.2
	GetAllActive(ctx context.Context) ([]Menu, error)
	SetRoleMenus(ctx context.Context, roleID uuid.UUID, menuIDs []uuid.UUID, operatorID uuid.UUID) error
	GetRoleMenuIDs(ctx context.Context, roleID uuid.UUID) ([]uuid.UUID, error)
}

// ApiEndpointRepository defines the persistence interface for API endpoints.
//
//go:generate go run go.uber.org/mock/mockgen -destination=mock_api_endpoint_repository_test.go -package=admin . ApiEndpointRepository
type ApiEndpointUpsertInput struct {
	Path        string
	Method      string
	Name        string
	Description string
	ApiGroup    string
}

type ApiEndpointRepository interface {
	Create(ctx context.Context, ep *ApiEndpointDB) error
	GetByID(ctx context.Context, id uuid.UUID) (*ApiEndpointDB, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateApiEndpointRequest) (*ApiEndpointDB, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByIDs(ctx context.Context, ids []uuid.UUID) error
	List(ctx context.Context, filter ApiEndpointFilter) (*model.ListResponse[ApiEndpointDB], error)
	Upsert(ctx context.Context, path, method, name, description, apiGroup string) (created bool, err error)
	GetGroups(ctx context.Context) ([]string, error)
}

// AuditRepository defines the persistence interface for audit logs.
//
//go:generate go run go.uber.org/mock/mockgen -destination=mock_audit_repository_test.go -package=admin . AuditRepository
type AuditRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	List(ctx context.Context, filter AuditLogFilter) (*model.ListResponse[AuditLog], error)
}
