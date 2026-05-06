package admin

import (
	"time"

	"github.com/google/uuid"
	"github.com/omcgo/omcgo/internal/core/model"
)

// UserStatus represents the lifecycle status of a user account.
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

// UserSource represents the origin of a user account.
// 字面值大小写敏感，由 DB CHECK 约束锁定，详见 docs/prd/system/users.md §11.3。
type UserSource string

const (
	UserSourceBuiltIn UserSource = "builtIn" // 系统初始化写入，不可删除/不可禁用
	UserSourceAdmin   UserSource = "admin"   // 管理员通过 API 创建
	UserSourceLDAP    UserSource = "LDAP"    // LDAP 同步任务写入，密码归属外部域
)

// User represents a system user.
type User struct {
	ID                  uuid.UUID          `json:"id"`
	Username            string             `json:"username"`
	PasswordHash        string             `json:"-"`
	DisplayName         string             `json:"display_name"`
	Email               string             `json:"email,omitempty"`
	Phone               string             `json:"phone,omitempty"`
	Description         string             `json:"description,omitempty"`
	Carrier             *model.CarrierCode `json:"carrier,omitempty"`
	Status              UserStatus         `json:"status"`
	Source              UserSource         `json:"source"`
	Roles               []Role             `json:"roles,omitempty"`
	FailedLoginAttempts int                `json:"failed_login_attempts"`
	LockedUntil         *time.Time         `json:"locked_until,omitempty"`
	LastFailedLoginAt   *time.Time         `json:"last_failed_login_at,omitempty"`
	LastLoginAt         *time.Time         `json:"last_login_at,omitempty"`
	ExpireAt            *time.Time         `json:"expire_at,omitempty"`
	CreatedBy           *uuid.UUID         `json:"created_by,omitempty"`
	UpdatedBy           *uuid.UUID         `json:"updated_by,omitempty"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
}

// Role represents a named role with associated permissions.
type Role struct {
	ID             uuid.UUID    `json:"id"`
	Name           string       `json:"name"`
	Description    string       `json:"description"`
	IsSystem       bool         `json:"is_system"`
	Permissions    []Permission `json:"permissions,omitempty"`
	DeviceGroupIDs []uuid.UUID  `json:"device_group_ids,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}

// Permission represents a resource-action pair bound to a role.
type Permission struct {
	ID       uuid.UUID `json:"id"`
	RoleID   uuid.UUID `json:"role_id"`
	Resource string    `json:"resource"`
	Action   string    `json:"action"`
}

// TokenPair contains JWT access and refresh tokens.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// AuditLog records a user action for auditing purposes.
type AuditLog struct {
	ID         uuid.UUID              `json:"id"`
	UserID     *uuid.UUID             `json:"user_id,omitempty"`
	Username   string                 `json:"username"`
	Action     string                 `json:"action"`
	Resource   string                 `json:"resource,omitempty"`
	ResourceID string                 `json:"resource_id,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	UserAgent  string                 `json:"user_agent,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
}

// Claims contains the JWT token claims for authenticated users.
type Claims struct {
	UserID        uuid.UUID          `json:"user_id"`
	Username      string             `json:"username"`
	Carrier       *model.CarrierCode `json:"carrier,omitempty"`
	Roles         []string           `json:"roles"`
	CurrentRoleID *uuid.UUID         `json:"current_role_id,omitempty"`
	// IssuedAt 是 JWT iat（Unix 秒），用于配合 TokenRevoker 判定 token 是否被强制下线。
	IssuedAt int64 `json:"iat,omitempty"`
}

// UserFilter provides filtering options for listing users.
type UserFilter struct {
	Carrier *model.CarrierCode `form:"carrier"`
	Status  *UserStatus        `form:"status"`
	Search  *string            `form:"search"`
	model.ListRequest
}

// AuditLogFilter provides filtering options for listing audit logs.
type AuditLogFilter struct {
	UserID    *uuid.UUID `form:"user_id"`
	Action    *string    `form:"action"`
	Resource  *string    `form:"resource"`
	StartTime *string    `form:"start_time"`
	EndTime   *string    `form:"end_time"`
	model.ListRequest
}

// CreateUserRequest is the input for creating a new user.
type CreateUserRequest struct {
	Username    string             `json:"username" binding:"required,min=3,max=64"`
	Password    string             `json:"password" binding:"required,min=6"`
	DisplayName string             `json:"display_name"`
	Email       string             `json:"email" binding:"omitempty,email"`
	Phone       string             `json:"phone"`
	Description string             `json:"description"`
	ExpireAt    *time.Time         `json:"expire_at"`
	Carrier     *model.CarrierCode `json:"carrier"`
	RoleIDs     []uuid.UUID        `json:"role_ids"`
}

// UpdateUserRequest is the input for updating an existing user.
//
// RoleIDs 语义：
//   - nil      → 不变更角色（保持当前关联）
//   - 非 nil（含空切片） → 用 RoleIDs 整体替换当前角色列表（差量执行 Assign/Remove）
type UpdateUserRequest struct {
	DisplayName *string            `json:"display_name"`
	Email       *string            `json:"email" binding:"omitempty,email"`
	Phone       *string            `json:"phone"`
	Description *string            `json:"description"`
	ExpireAt    *time.Time         `json:"expire_at"`
	Carrier     *model.CarrierCode `json:"carrier"`
	Status      *UserStatus        `json:"status"`
	RoleIDs     *[]uuid.UUID       `json:"role_ids"`
}

// LoginRequest is the input for user authentication.
type LoginRequest struct {
	Username      string `json:"username" binding:"required"`
	Password      string `json:"password" binding:"required"`
	CaptchaID     string `json:"captcha_id"`
	CaptchaAnswer string `json:"captcha_answer"`
}

// RefreshRequest is the input for refreshing a JWT token.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// AssignRoleRequest is the input for assigning a role to a user.
type AssignRoleRequest struct {
	RoleID uuid.UUID `json:"role_id" binding:"required"`
}

// ResetPasswordRequest is the input for resetting a user's password.
type ResetPasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// CreateRoleRequest is the input for creating a new role.
type CreateRoleRequest struct {
	Name        string            `json:"name" binding:"required"`
	Description string            `json:"description"`
	Permissions []PermissionInput `json:"permissions"`
}

// PermissionInput represents a resource-action pair for role permission assignment.
type PermissionInput struct {
	Resource string `json:"resource" binding:"required"`
	Action   string `json:"action" binding:"required"`
}

// UpdateRoleRequest is the input for updating an existing role.
type UpdateRoleRequest struct {
	Name        *string           `json:"name"`
	Description *string           `json:"description"`
	Permissions []PermissionInput `json:"permissions"`
}

// RoleFilter provides filtering options for listing roles.
type RoleFilter struct {
	Search *string `form:"search"`
	Name   *string `form:"name"`
	model.ListRequest
}

// MenuStatus represents the lifecycle status of a menu.
type MenuStatus string

const (
	MenuStatusActive   MenuStatus = "active"
	MenuStatusInactive MenuStatus = "inactive"
	// MenuStatusNormal is an alias for active status (backward compatibility)
	MenuStatusNormal MenuStatus = "active"
)

// MenuType constants for menu item types.
const (
	MenuTypeDirectory string = "directory"
	MenuTypeMenu      string = "menu"
	MenuTypeButton    string = "button"
)

// Menu represents a navigation menu item.
type Menu struct {
	ID             uuid.UUID   `json:"id"`
	Name           string      `json:"name"`
	Type           string      `json:"type"`
	PermissionKey  string      `json:"permission_key"`
	ParentID       *uuid.UUID  `json:"parent_id,omitempty"`
	SortOrder      int         `json:"sort_order"`
	RoutePath      string      `json:"route_path,omitempty"`
	ComponentPath  string      `json:"component_path,omitempty"`
	Icon           string      `json:"icon,omitempty"`
	ShowStatus     bool        `json:"show_status"`
	Status         MenuStatus  `json:"status"`
	CreatedBy      *uuid.UUID  `json:"created_by,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedBy      *uuid.UUID  `json:"updated_by,omitempty"`
	UpdatedAt      time.Time   `json:"updated_at"`
	Children       []Menu      `json:"children,omitempty"`
}

// MenuFilter provides filtering options for listing menus.
type MenuFilter struct {
	Type     *string     `form:"type"`
	Status   *MenuStatus `form:"status"`
	ParentID *uuid.UUID  `form:"parent_id"`
	model.ListRequest
}

// UpdateMenuRequest is the input for updating an existing menu.
type UpdateMenuRequest struct {
	Name          *string     `json:"name"`
	Type          *string     `json:"type"`
	PermissionKey *string     `json:"permission_key"`
	ParentID      *uuid.UUID  `json:"parent_id"`
	SortOrder     *int        `json:"sort_order"`
	RoutePath     *string     `json:"route_path"`
	ComponentPath *string     `json:"component_path"`
	Icon          *string     `json:"icon"`
	ShowStatus    *bool       `json:"show_status"`
	Status        *MenuStatus `json:"status"`
}

// CreateMenuRequest is the input for creating a new menu.
type CreateMenuRequest struct {
	Name          string     `json:"name" binding:"required"`
	Type          string     `json:"type" binding:"required"`
	PermissionKey string     `json:"permission_key" binding:"required"`
	ParentID      *uuid.UUID `json:"parent_id"`
	SortOrder     int        `json:"sort_order"`
	RoutePath     string     `json:"route_path"`
	ComponentPath string     `json:"component_path"`
	Icon          string     `json:"icon"`
	ShowStatus    bool       `json:"show_status"`
}

// SetRoleMenusRequest is the input for setting role menu permissions.
type SetRoleMenusRequest struct {
	MenuIDs []uuid.UUID `json:"menu_ids" binding:"required"`
}
