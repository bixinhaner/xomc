# DD-17: 用户管理与 RBAC（F06 子模块）

> 关联功能域：F06（OMC-R 核心功能 — 安全管理）
> 关联 backend-design.md 章节：第二章（模块 6 — omcr/admin）
> 实施阶段：Phase 4（北向与规模化）
> 依赖文档：DD-02, DD-03

---

## 1. 概述

### 1.1 模块定位

用户管理与 RBAC（`internal/omcr/admin/`）实现系统的认证授权和操作审计，保障系统安全性。

### 1.2 核心职责

- 用户管理（CRUD、密码策略）
- 角色与权限管理（RBAC 模型）
- JWT 认证（token 签发/验证/刷新）
- 运营商数据隔离（用户只能访问所属运营商数据）
- 操作审计日志

---

## 2. 接口设计

### 2.1 AdminService — `internal/omcr/admin/service.go`

```go
type AdminService struct {
    userRepo    UserRepository
    roleRepo    RoleRepository
    auditRepo   AuditRepository
    jwtSecret   string
    logger      *zap.Logger
}

// Authentication
func (s *AdminService) Login(ctx context.Context, username, password string) (*TokenPair, error)
func (s *AdminService) RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)

// User CRUD
func (s *AdminService) CreateUser(ctx context.Context, user *User) error
func (s *AdminService) UpdateUser(ctx context.Context, user *User) error
func (s *AdminService) DeleteUser(ctx context.Context, id uuid.UUID) error
func (s *AdminService) ListUsers(ctx context.Context, filter UserFilter) (*model.ListResponse[User], error)

// RBAC
func (s *AdminService) AssignRole(ctx context.Context, userID, roleID uuid.UUID) error
func (s *AdminService) CheckPermission(ctx context.Context, userID uuid.UUID, resource, action string) (bool, error)
```

### 2.2 数据模型

```go
type User struct {
    ID           uuid.UUID
    Username     string
    PasswordHash string
    DisplayName  string
    Email        string
    Carrier      *model.CarrierCode // nil=全运营商, 非nil=仅限该运营商
    Status       string // active, disabled
    Roles        []Role
    CreatedAt    time.Time
}

type Role struct {
    ID          uuid.UUID
    Name        string // admin, operator, viewer
    Description string
    Permissions []Permission
}

type Permission struct {
    Resource string // devices, alarms, pm, config, datamodels, users
    Action   string // read, write, delete, admin
}

type TokenPair struct {
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
}

type AuditLog struct {
    ID         uuid.UUID
    UserID     uuid.UUID
    Action     string
    Resource   string
    ResourceID string
    Details    map[string]interface{}
    IP         string
    CreatedAt  time.Time
}
```

### 2.3 RBAC 中间件 — `internal/omcr/admin/rbac.go`

```go
// RequirePermission Gin 中间件，检查用户权限
func RequirePermission(adminService *AdminService, resource, action string) gin.HandlerFunc

// RequireCarrier Gin 中间件，检查用户运营商访问权限
func RequireCarrier(adminService *AdminService) gin.HandlerFunc
```

---

## 3. 数据库 Schema

```sql
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(64) NOT NULL UNIQUE,
    password_hash VARCHAR(256) NOT NULL,
    display_name  VARCHAR(128),
    email         VARCHAR(256),
    carrier       VARCHAR(4),
    status        VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(64) NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id),
    role_id UUID NOT NULL REFERENCES roles(id),
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE permissions (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id  UUID NOT NULL REFERENCES roles(id),
    resource VARCHAR(64) NOT NULL,
    action   VARCHAR(16) NOT NULL,
    UNIQUE(role_id, resource, action)
);

CREATE TABLE audit_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID REFERENCES users(id),
    action      VARCHAR(64) NOT NULL,
    resource    VARCHAR(64),
    resource_id VARCHAR(128),
    details     JSONB,
    ip_address  INET,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user ON audit_logs (user_id, created_at DESC);
CREATE INDEX idx_audit_logs_time ON audit_logs (created_at DESC);
```

### REST API

```
POST /api/v1/auth/login             登录
POST /api/v1/auth/refresh           刷新 token
GET  /api/v1/admin/users            用户列表
POST /api/v1/admin/users            创建用户
GET  /api/v1/admin/roles            角色列表
POST /api/v1/admin/roles            创建角色
GET  /api/v1/admin/audit-logs       审计日志查询
```

---

## 4. 实施子阶段

### 阶段 17a：用户 + 角色基础（Phase 4）
### 阶段 17b：RBAC 中间件 + 审计日志（Phase 4）

---

## 5. 文件清单

```
internal/omcr/admin/service.go
internal/omcr/admin/rbac.go
internal/omcr/admin/repository.go
internal/omcr/admin/audit.go
```

---

## 6. 参考

- doc/features/06-omc-core-functions.md：F06.01 安全管理
