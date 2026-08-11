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
	ID                  uuid.UUID  `json:"id"`
	Username            string     `json:"username"`
	PasswordHash        string     `json:"-"`
	DisplayName         string     `json:"display_name"`
	Email               string     `json:"email,omitempty"`
	Phone               string     `json:"phone,omitempty"`
	Description         string     `json:"description,omitempty"`
	Status              UserStatus `json:"status"`
	Source              UserSource `json:"source"`
	Roles               []Role     `json:"roles,omitempty"`
	FailedLoginAttempts int        `json:"failed_login_attempts"`
	LockedUntil         *time.Time `json:"locked_until,omitempty"`
	LastFailedLoginAt   *time.Time `json:"last_failed_login_at,omitempty"`
	LastLoginAt         *time.Time `json:"last_login_at,omitempty"`
	ExpireAt            *time.Time `json:"expire_at,omitempty"`
	// P1-① 首次登录强制改密：CreateUser/ResetPassword 时按
	// sys_configs.security.modifyPWD 置为 true；ChangePassword 后清零。
	MustChangePassword bool `json:"must_change_password"`
	// P1-④ 密码最近修改时间；UpdatePassword 同步刷新。Login 时与
	// sys_configs.security.validPeriod 比较判断密码过期。
	PasswordChangedAt *time.Time `json:"password_changed_at,omitempty"`
	CreatedBy         *uuid.UUID `json:"created_by,omitempty"`
	UpdatedBy         *uuid.UUID `json:"updated_by,omitempty"`
	// CreatorUsername / UpdaterUsername 是 ListUsers / GetUser 派生字段：
	// 用 created_by / updated_by 反查 users.username，省去前端二次拉全量用户表。
	// 当对应 ID 为 NULL（seed 写入 / 内置）时为空字符串；前端按空值渲染"内置"。
	CreatorUsername string    `json:"creator_username,omitempty"`
	UpdaterUsername string    `json:"updater_username,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// IsSuperAdmin reports whether the user is a system built-in user (UserSourceBuiltIn).
// 决议：v1.0 起超管由 source='builtIn' 唯一标识（不再用 carrier IS NULL）。
// 详见 docs/prd/system/users.md §11.11。
func (u *User) IsSuperAdmin() bool {
	return u != nil && u.Source == UserSourceBuiltIn
}

// Role represents a named role with associated permissions.
//
// v0.6（roles.md §7 落地）新增字段：
//   - Code        : 程序化引用用的角色编码，可选；非空时全局唯一
//   - UserCount   : 列表派生字段（COUNT(*) FROM user_roles WHERE role_id = ...）
//   - CreatedBy   : 创建者 user_id；NULL 表示 seed 写入（内置角色）
//   - UpdatedBy   : 最近一次修改者 user_id
type Role struct {
	ID              uuid.UUID    `json:"id"`
	Name            string       `json:"name"`
	Code            string       `json:"code,omitempty"`
	Description     string       `json:"description"`
	IsSystem        bool         `json:"is_system"`
	Permissions     []Permission `json:"permissions,omitempty"`
	DeviceGroupIDs  []uuid.UUID  `json:"device_group_ids,omitempty"`
	UserCount       int          `json:"user_count"`
	CreatedBy       *uuid.UUID   `json:"created_by,omitempty"`
	UpdatedBy       *uuid.UUID   `json:"updated_by,omitempty"`
	CreatorUsername string       `json:"creator_username,omitempty"`
	UpdaterUsername string       `json:"updater_username,omitempty"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
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

	// P1 密码策略派生字段（仅 Login 响应填充；Refresh / 其他端点不带）：
	//   - MustChangePassword: 用户必须立刻改密（首次登录 / 管理员重置 / 密码已过期）
	//   - PasswordExpiresInDays: 距密码过期还剩多少天 (>=0)；nil = 未启用过期 / 已过期已在 MustChange
	//   FE 拿到 MustChangePassword=true 后强制跳改密页；
	//   拿到 PasswordExpiresInDays<=promptBeforeDays 时弹"即将过期"toast。
	MustChangePassword    bool `json:"must_change_password,omitempty"`
	PasswordExpiresInDays *int `json:"password_expires_in_days,omitempty"`

	// P2-⑪ 登录提示：sys_configs.security.enabledFlag=true 时附管理员配置的提示文案。
	// FE 拿到非空 msg 时弹 Modal/Notification（用户首次见到后可关闭）。
	LoginNotifyMsg string `json:"login_notify_msg,omitempty"`
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
//
// v1.0 起：
//   - 移除 `Carrier` 字段（users.carrier 列已删；UI/告警维度的 carrier 不再走 user 层）
//   - 新增 `IsSuperAdmin` 字段，由 service 在签发时按 user.IsSuperAdmin() 写入
//
// 兼容性：旧版 token 反序列化时 IsSuperAdmin 缺省为 false，Carrier 缺省为 nil 已忽略。
type Claims struct {
	UserID        uuid.UUID  `json:"user_id"`
	Username      string     `json:"username"`
	IsSuperAdmin  bool       `json:"is_super_admin,omitempty"`
	Roles         []string   `json:"roles"`
	CurrentRoleID *uuid.UUID `json:"current_role_id,omitempty"`
	// IssuedAt 是 JWT iat（Unix 秒），用于配合 TokenRevoker 判定 token 是否被强制下线。
	IssuedAt int64 `json:"iat,omitempty"`
	// IssuedAtMicros 是自定义微秒级签发时间，用于消除 force-logout 的同秒边界歧义。
	IssuedAtMicros int64 `json:"iat_us,omitempty"`
	// Scopes 是专用短期 token 的能力边界。普通 access/refresh token 为空。
	Scopes []string `json:"scopes,omitempty"`
}

// UserFilter provides filtering options for listing users.
// v1.0：移除 Carrier 过滤项（按 role 名间接过滤运营商范围）。
type UserFilter struct {
	Status *UserStatus `form:"status"`
	Search *string     `form:"search"`
	RoleID *string     `form:"role_id"`
	model.ListRequest
}

// AuditLogFilter provides filtering options for listing audit logs.
type AuditLogFilter struct {
	UserID    *uuid.UUID `form:"user_id"`
	Username  *string    `form:"username"`
	Action    *string    `form:"action"`
	Resource  *string    `form:"resource"`
	IPAddress *string    `form:"ip_address"`
	Result    *string    `form:"result"`
	Reason    *string    `form:"reason"`
	Keyword   *string    `form:"keyword"`
	StartTime *string    `form:"start_time"`
	EndTime   *string    `form:"end_time"`
	model.ListRequest
}

// CreateUserRequest 是 service 层创建用户的明文输入。
//
// 注意：HTTP /admin/users 端点不再直接绑定本结构（密码必须 RSA-OAEP 加密传输，
// 见 CreateUserHTTPRequest）。本结构仍保留为：
//   - service.CreateUser 的入参（明文）
//   - xlsx 批量导入路径 ImportUsers 的内部入参（导入文件本身在管理员可信范围内）
//
// v1.0：移除 Carrier 字段（users.carrier 已删）。
// issue #649：Password 改为可选（UseDefaultPassword=true 时可留空，service 从
// sys_configs.security.defaultPasswd 取值）；service 层会根据 UseDefaultPassword 严格分
// 支，不依赖字段是否为空推断（避免旧客户端意外走默认密码路径）。
type CreateUserRequest struct {
	Username           string      `json:"username" binding:"required,min=3,max=32"`
	Password           string      `json:"password" binding:"omitempty,min=6"`
	UseDefaultPassword bool        `json:"use_default_password"`
	DisplayName        string      `json:"display_name"`
	Email              string      `json:"email" binding:"required,email"`
	Phone              string      `json:"phone"`
	Description        string      `json:"description"`
	ExpireAt           *time.Time  `json:"expire_at"`
	RoleIDs            []uuid.UUID `json:"role_ids"`
}

// CreateUserHTTPRequest 是 POST /api/v1/admin/users 的请求体（密码加密传输）。
//
// 前端先调 GET /auth/public-key 拉公钥，把 {password, ts, nonce} JSON 用
// RSA-OAEP/SHA-256 加密 → base64 → 填入 encrypted_password；同时回传 key_id。
// 后端 handler 解密后构造 CreateUserRequest 调 service。
// T-0120 同 LoginRequest：去 required，handler 内做二选一校验。
// issue #649：新增 use_default_password 开关，为 true 时 handler 跳过密码解密，service
// 从默认密码取值。
type CreateUserHTTPRequest struct {
	Username           string      `json:"username" binding:"required,min=3,max=32"`
	EncryptedPassword  string      `json:"encrypted_password"`
	KeyID              string      `json:"key_id"`
	Password           string      `json:"password"` // T-0120 plaintext fallback
	UseDefaultPassword bool        `json:"use_default_password"`
	DisplayName        string      `json:"display_name"`
	Email              string      `json:"email" binding:"required,email"`
	Phone              string      `json:"phone"`
	Description        string      `json:"description"`
	ExpireAt           *time.Time  `json:"expire_at"`
	RoleIDs            []uuid.UUID `json:"role_ids"`
}

// UpdateUserRequest is the input for updating an existing user.
//
// RoleIDs 语义：
//   - nil      → 不变更角色（保持当前关联）
//   - 非 nil（含空切片） → 用 RoleIDs 整体替换当前角色列表（差量执行 Assign/Remove）
//
// v1.0：移除 Carrier 字段（users.carrier 已删）。
type UpdateUserRequest struct {
	DisplayName *string      `json:"display_name"`
	Email       *string      `json:"email" binding:"required,email"`
	Phone       *string      `json:"phone"`
	Description *string      `json:"description"`
	ExpireAt    *time.Time   `json:"expire_at"`
	Status      *UserStatus  `json:"status"`
	RoleIDs     *[]uuid.UUID `json:"role_ids"`
}

// LoginRequest 是 POST /api/v1/auth/login 的请求体。
//
// 密码必须 RSA-OAEP 加密：前端调 GET /auth/public-key 拉公钥，把
// {password, ts, nonce} JSON 用 RSA-OAEP/SHA-256 加密 → base64 → 填入
// encrypted_password；同时回传 key_id。后端 handler 解密后调 service.Login。
// T-0120：EncryptedPassword + KeyID 去掉 `required` tag，handler 内做二选一校验:
//   - 加密路径（secure context）：EncryptedPassword + KeyID 必须同时非空
//   - 明文路径（仅 LoginCrypto.AllowPlaintext=true 时启用）：Password 非空
//
// 两路径都不满足 → handler 返 400 missing password。
type LoginRequest struct {
	Username          string `json:"username"`
	EncryptedPassword string `json:"encrypted_password"`
	KeyID             string `json:"key_id"`
	Password          string `json:"password"` // T-0120 plaintext fallback (config gated)
	CaptchaID         string `json:"captcha_id"`
	CaptchaAnswer     string `json:"captcha_answer"`
}

// RefreshRequest is the input for refreshing a JWT token.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// AssignRoleRequest is the input for assigning a role to a user.
type AssignRoleRequest struct {
	RoleID uuid.UUID `json:"role_id" binding:"required"`
}

// ResetPasswordRequest 是 POST /api/v1/admin/users/:id/reset-password 的请求体。
// T-0120 双路径：加密 vs 明文 fallback。
// issue #649：新增 use_default_password 开关，为 true 时 handler 跳过密码解密，service
// 从 sys_configs.security.defaultPasswd 取值。
type ResetPasswordRequest struct {
	EncryptedNewPassword string `json:"encrypted_new_password"`
	KeyID                string `json:"key_id"`
	NewPassword          string `json:"new_password"` // T-0120 plaintext fallback
	UseDefaultPassword   bool   `json:"use_default_password"`
}

// CreateRoleRequest is the input for creating a new role.
// v0.6：新增可选 Code 字段（程序化引用），非空时全局唯一。
type CreateRoleRequest struct {
	Name        string            `json:"name" binding:"required"`
	Code        string            `json:"code"`
	Description string            `json:"description"`
	Permissions []PermissionInput `json:"permissions"`
}

// PermissionInput represents a resource-action pair for role permission assignment.
type PermissionInput struct {
	Resource string `json:"resource" binding:"required"`
	Action   string `json:"action" binding:"required"`
}

// UpdateRoleRequest is the input for updating an existing role.
// v0.6：新增可选 Code 字段（指针：nil 表示不变更）。
type UpdateRoleRequest struct {
	Name        *string           `json:"name"`
	Code        *string           `json:"code"`
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
//
// 取值与 DDL CHECK 约束严格一致：migrations/000009_sys_admin.sql 第 38 行
// `chk_status CHECK (status IN ('normal','disabled'))`。任何写库点必须使用
// 本常量，避免历史上把 "active" 当别名注入引发 SQLSTATE 23514。
type MenuStatus string

const (
	MenuStatusNormal   MenuStatus = "normal"
	MenuStatusDisabled MenuStatus = "disabled"
)

// MenuType constants for menu item types.
const (
	MenuTypeDirectory string = "directory"
	MenuTypeMenu      string = "menu"
	MenuTypeButton    string = "button"
)

// MenuShowStatus represents the visibility of a menu in the sidebar.
// DDL menus.show_status 是 VARCHAR(16) NOT NULL CHECK IN ('show','hide')；
// 此处用 string 而非 bool，与库列类型 + 前端 'show' | 'hide' 一致。
type MenuShowStatus string

const (
	MenuShow MenuShowStatus = "show"
	MenuHide MenuShowStatus = "hide"
)

// Menu represents a navigation menu item.
//
// 多语言字段（migration 000083）：
//   - NameI18n：多语言译文字典，键为 locale code（zh-CN/en-US/...），值为译文。
//     菜单管理 UI 编辑后写入；nil 表示未配置多语言，前端回退到 Name。
//
// 前端渲染优先级（参 frontend-core/src/types/menu.ts 与 NavMenu.tsx）：
//
//	NameI18n[locale] > NameI18n["zh-CN"] > Name
type Menu struct {
	ID            uuid.UUID         `json:"id"`
	Name          string            `json:"name"`
	NameI18n      map[string]string `json:"name_i18n,omitempty"`
	Type          string            `json:"type"`
	PermissionKey string            `json:"permission_key"`
	ParentID      *uuid.UUID        `json:"parent_id,omitempty"`
	SortOrder     int               `json:"sort_order"`
	RoutePath     string            `json:"route_path,omitempty"`
	ComponentPath string            `json:"component_path,omitempty"`
	Icon          string            `json:"icon,omitempty"`
	ShowStatus    MenuShowStatus    `json:"show_status"`
	Status        MenuStatus        `json:"status"`
	CreatedBy     *uuid.UUID        `json:"created_by,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedBy     *uuid.UUID        `json:"updated_by,omitempty"`
	UpdatedAt     time.Time         `json:"updated_at"`
	FeatureCodes  []string          `json:"feature_code,omitempty"`
	Children      []Menu            `json:"children,omitempty"`
}

// MenuFilter provides filtering options for listing menus.
type MenuFilter struct {
	Type     *string     `form:"type"`
	Status   *MenuStatus `form:"status"`
	ParentID *uuid.UUID  `form:"parent_id"`
	model.ListRequest
}

// UpdateMenuRequest is the input for updating an existing menu.
//
// NameI18n 为指针：nil 表示「不变更该字段」，非 nil（含空值）表示显式写入。
// 前端传 {"name_i18n": null} 时按「不变更」处理；传 {"name_i18n": {}} 表示
// 清空所有译文（合法），repo 归一为 NULL。
type UpdateMenuRequest struct {
	Name          *string            `json:"name"`
	NameI18n      *map[string]string `json:"name_i18n"`
	Type          *string            `json:"type"`
	PermissionKey *string            `json:"permission_key"`
	ParentID      *uuid.UUID         `json:"parent_id"`
	SortOrder     *int               `json:"sort_order"`
	RoutePath     *string            `json:"route_path"`
	ComponentPath *string            `json:"component_path"`
	Icon          *string            `json:"icon"`
	ShowStatus    *MenuShowStatus    `json:"show_status"`
	Status        *MenuStatus        `json:"status"`
}

// CreateMenuRequest is the input for creating a new menu.
type CreateMenuRequest struct {
	Name          string            `json:"name" binding:"required"`
	NameI18n      map[string]string `json:"name_i18n"`
	Type          string            `json:"type" binding:"required"`
	PermissionKey string            `json:"permission_key" binding:"required"`
	ParentID      *uuid.UUID        `json:"parent_id"`
	SortOrder     int               `json:"sort_order"`
	RoutePath     string            `json:"route_path"`
	ComponentPath string            `json:"component_path"`
	Icon          string            `json:"icon"`
	ShowStatus    MenuShowStatus    `json:"show_status"`
}

// SetRoleMenusRequest is the input for setting role menu permissions.
type SetRoleMenusRequest struct {
	MenuIDs []uuid.UUID `json:"menu_ids" binding:"required"`
}
