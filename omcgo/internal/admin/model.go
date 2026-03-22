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

// User represents a system user.
type User struct {
	ID                  uuid.UUID          `json:"id"`
	Username            string             `json:"username"`
	PasswordHash        string             `json:"-"`
	DisplayName         string             `json:"display_name"`
	Email               string             `json:"email,omitempty"`
	Carrier             *model.CarrierCode `json:"carrier,omitempty"`
	Status              UserStatus         `json:"status"`
	Roles               []Role             `json:"roles,omitempty"`
	FailedLoginAttempts int                `json:"failed_login_attempts"`
	LockedUntil         *time.Time         `json:"locked_until,omitempty"`
	LastFailedLoginAt   *time.Time         `json:"last_failed_login_at,omitempty"`
	LastLoginAt         *time.Time         `json:"last_login_at,omitempty"`
	CreatedAt           time.Time          `json:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at"`
}

// Role represents a named role with associated permissions.
type Role struct {
	ID          uuid.UUID    `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	IsSystem    bool         `json:"is_system"`
	Permissions []Permission `json:"permissions,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
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
	UserID   uuid.UUID          `json:"user_id"`
	Username string             `json:"username"`
	Carrier  *model.CarrierCode `json:"carrier,omitempty"`
	Roles    []string           `json:"roles"`
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
	Carrier     *model.CarrierCode `json:"carrier"`
	RoleIDs     []uuid.UUID        `json:"role_ids"`
}

// UpdateUserRequest is the input for updating an existing user.
type UpdateUserRequest struct {
	DisplayName *string            `json:"display_name"`
	Email       *string            `json:"email" binding:"omitempty,email"`
	Carrier     *model.CarrierCode `json:"carrier"`
	Status      *UserStatus        `json:"status"`
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
