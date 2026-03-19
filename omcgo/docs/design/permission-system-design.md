# OMC 权限系统设计方案

> 基于 `files/Back-end/系统管理模块开发设计文档.md` 的权限系统完整设计。

---

## 1. 概述

### 1.1 设计目标

权限系统是 OMC 平台的安全核心，通过**用户-部门-角色-菜单**的分层配置，实现精细化的权限管控体系：

| 目标 | 说明 |
|------|------|
| 功能权限 | 控制用户可访问的菜单和可执行的操作（按钮级） |
| 数据权限 | 控制用户可访问的设备/站点数据范围 |
| 操作权限 | 控制用户操作方式（单条/批量） |
| 审计追溯 | 完整记录用户操作日志 |

### 1.2 权限模型

采用 **RBAC（Role-Based Access Control）** 模型，扩展支持数据权限：

```
用户(User) ←→ 角色(Role) ←→ 菜单/权限(Menu/Permission)
    │              │
    ↓              ↓
 部门(Department)  数据权限(DataScope)
```

**权限聚合规则**：用户最终权限为其所有角色权限的**并集**。

---

## 2. 当前实现分析

### 2.1 已有表结构

```sql
-- 用户表
CREATE TABLE users (
    id            UUID PRIMARY KEY,
    username      VARCHAR(64) NOT NULL UNIQUE,
    password_hash VARCHAR(256) NOT NULL,
    display_name  VARCHAR(128),
    email         VARCHAR(256),
    carrier       VARCHAR(4),
    status        VARCHAR(16) NOT NULL DEFAULT 'active',
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ,
    updated_at    TIMESTAMPTZ
);

-- 角色表
CREATE TABLE roles (
    id          UUID PRIMARY KEY,
    name        VARCHAR(64) NOT NULL UNIQUE,
    description TEXT,
    is_system   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ,
    updated_at  TIMESTAMPTZ
);

-- 用户-角色关联表
CREATE TABLE user_roles (
    user_id    UUID NOT NULL REFERENCES users(id),
    role_id    UUID NOT NULL REFERENCES roles(id),
    created_at TIMESTAMPTZ,
    PRIMARY KEY (user_id, role_id)
);

-- 权限表（资源-操作对）
CREATE TABLE permissions (
    id       UUID PRIMARY KEY,
    role_id  UUID NOT NULL REFERENCES roles(id),
    resource VARCHAR(64) NOT NULL,
    action   VARCHAR(16) NOT NULL,
    UNIQUE(role_id, resource, action)
);
```

### 2.2 缺失功能

| 模块 | 缺失内容 |
|------|---------|
| 用户管理 | 手机号、部门、登录来源、过期时间、创建人/更新人、备注 |
| 部门管理 | 整个模块缺失（部门表、层级关系） |
| 角色管理 | 角色标识、数据权限、批量操作权限 |
| 菜单管理 | 整个模块缺失（菜单表、权限配置） |
| 数据权限 | 设备组关联缺失 |
| 登录日志 | 单独的登录日志表缺失 |

---

## 3. 数据库设计

### 3.1 用户表扩展

```sql
-- Migration: 000039_extend_users_table.up.sql

-- 新增字段
ALTER TABLE users
    ADD COLUMN phone           VARCHAR(32),
    ADD COLUMN department_id   UUID,
    ADD COLUMN expired_at      TIMESTAMPTZ,
    ADD COLUMN remark          TEXT,
    ADD COLUMN created_by      UUID REFERENCES users(id),
    ADD COLUMN updated_by      UUID REFERENCES users(id);

-- 添加部门外键索引
CREATE INDEX idx_users_department ON users (department_id) WHERE department_id IS NOT NULL;

COMMENT ON COLUMN users.phone IS '用户手机号';
COMMENT ON COLUMN users.department_id IS '归属部门ID';
COMMENT ON COLUMN users.expired_at IS '账号过期时间，NULL表示永不过期';
COMMENT ON COLUMN users.remark IS '备注信息';
COMMENT ON COLUMN users.created_by IS '创建人ID';
COMMENT ON COLUMN users.updated_by IS '更新人ID';
```

### 3.2 部门表（新增）

```sql
-- Migration: 000040_create_departments.up.sql

CREATE TABLE departments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(128) NOT NULL,
    parent_id       UUID REFERENCES departments(id) ON DELETE SET NULL,
    sort_order      INTEGER NOT NULL DEFAULT 0,
    path            VARCHAR(512),           -- 物化路径，如 '/root/branch1/leaf'
    level           INTEGER NOT NULL DEFAULT 1,
    status          VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by      UUID REFERENCES users(id),
    updated_by      UUID REFERENCES users(id)
);

-- 唯一约束：同一父部门下名称唯一
CREATE UNIQUE INDEX idx_departments_unique_name ON departments (parent_id, name) WHERE parent_id IS NOT NULL;
CREATE UNIQUE INDEX idx_departments_root_unique_name ON departments (name) WHERE parent_id IS NULL;

-- 查询索引
CREATE INDEX idx_departments_parent ON departments (parent_id);
CREATE INDEX idx_departments_path ON departments (path);
CREATE INDEX idx_departments_status ON departments (status);

COMMENT ON COLUMN departments.path IS '物化路径，用于快速查询所有祖先/后代';
COMMENT ON COLUMN departments.level IS '层级深度，从1开始';

-- 触发器
CREATE TRIGGER trigger_departments_updated_at
    BEFORE UPDATE ON departments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

### 3.3 角色表扩展

```sql
-- Migration: 000041_extend_roles_table.up.sql

-- 新增字段
ALTER TABLE roles
    ADD COLUMN code             VARCHAR(64) UNIQUE,
    ADD COLUMN data_scope_type  VARCHAR(16) NOT NULL DEFAULT 'self',
    ADD COLUMN batch_permission VARCHAR(16) NOT NULL DEFAULT 'single',
    ADD COLUMN created_by       UUID REFERENCES users(id),
    ADD COLUMN updated_by       UUID REFERENCES users(id);

COMMENT ON COLUMN roles.code IS '角色唯一标识码，用于代码中引用';
COMMENT ON COLUMN roles.data_scope_type IS '数据权限类型：all(全部)/department(本部门及下级)/self(仅本人)/custom(自定义设备组)';
COMMENT ON COLUMN roles.batch_permission IS '批量操作权限：single(仅单条)/batch(允许批量)';

-- 预置系统角色
INSERT INTO roles (id, name, code, description, is_system, data_scope_type, batch_permission) VALUES
    (gen_random_uuid(), '超级管理员', 'super_admin', '系统超级管理员，拥有所有权限', TRUE, 'all', 'batch'),
    (gen_random_uuid(), '系统管理员', 'admin', '系统管理员，可管理用户和配置', TRUE, 'all', 'batch'),
    (gen_random_uuid(), '普通用户', 'user', '普通用户，仅查看权限', TRUE, 'self', 'single')
ON CONFLICT (name) DO NOTHING;
```

### 3.4 菜单表（新增）

```sql
-- Migration: 000042_create_menus.up.sql

CREATE TABLE menus (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(64) NOT NULL,
    type         VARCHAR(16) NOT NULL,         -- directory/menu/button
    parent_id    UUID REFERENCES menus(id) ON DELETE CASCADE,
    path         VARCHAR(256),                 -- 前端路由地址（菜单类型必填）
    component    VARCHAR(256),                 -- 前端组件路径
    permission   VARCHAR(128),                 -- 权限标识（如 system:user:list）
    icon         VARCHAR(64),                  -- 图标名称
    sort_order   INTEGER NOT NULL DEFAULT 0,
    visible      BOOLEAN NOT NULL DEFAULT TRUE,
    status       VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 索引
CREATE UNIQUE INDEX idx_menus_unique_name ON menus (parent_id, name);
CREATE INDEX idx_menus_parent ON menus (parent_id);
CREATE INDEX idx_menus_type ON menus (type);
CREATE INDEX idx_menus_status ON menus (status);

COMMENT ON COLUMN menus.type IS '类型：directory(目录)/menu(菜单)/button(按钮)';
COMMENT ON COLUMN menus.path IS '前端路由地址，菜单类型必填';
COMMENT ON COLUMN menus.permission IS '权限标识，用于按钮级权限控制';

-- 触发器
CREATE TRIGGER trigger_menus_updated_at
    BEFORE UPDATE ON menus
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
```

### 3.5 角色-菜单关联表（新增）

```sql
-- Migration: 000043_create_role_menus.up.sql

CREATE TABLE role_menus (
    role_id    UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    menu_id    UUID NOT NULL REFERENCES menus(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, menu_id)
);

CREATE INDEX idx_role_menus_role ON role_menus (role_id);
CREATE INDEX idx_role_menus_menu ON role_menus (menu_id);
```

### 3.6 角色-设备组关联表（数据权限）

```sql
-- Migration: 000044_create_role_device_groups.up.sql

-- 注意：依赖 device_groups 表（需确认是否已存在）
-- 如果不存在，需要先创建 device_groups 表

CREATE TABLE role_device_groups (
    role_id        UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    device_group_id UUID NOT NULL,  -- REFERENCES device_groups(id)
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, device_group_id)
);

CREATE INDEX idx_role_device_groups_role ON role_device_groups (role_id);
```

### 3.7 登录日志表（新增）

```sql
-- Migration: 000045_create_login_logs.up.sql

CREATE TABLE login_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(64) NOT NULL,
    login_ip      INET NOT NULL,
    login_source  VARCHAR(64),              -- 登录来源（web/api/mobile）
    user_agent    TEXT,
    login_status  VARCHAR(16) NOT NULL,     -- success/failed
    fail_reason   TEXT,
    login_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 索引
CREATE INDEX idx_login_logs_username ON login_logs (username);
CREATE INDEX idx_login_logs_login_at ON login_logs (login_at);
CREATE INDEX idx_login_logs_status ON login_logs (login_status);

-- 分区（可选，大数据量时）
-- CREATE TABLE login_logs_2026_q1 PARTITION OF login_logs
--     FOR VALUES FROM ('2026-01-01') TO ('2026-04-01');

COMMENT ON COLUMN login_logs.login_status IS '登录状态：success/failed';
COMMENT ON COLUMN login_logs.fail_reason IS '登录失败原因，仅失败时填充';
```

---

## 4. 模型设计

### 4.1 用户模型

```go
// omcgo/internal/admin/model.go

type UserStatus string

const (
    UserStatusActive   UserStatus = "active"
    UserStatusDisabled UserStatus = "disabled"
)

type User struct {
    ID           uuid.UUID   `json:"id" db:"id"`
    Username     string      `json:"username" db:"username"`
    PasswordHash string      `json:"-" db:"password_hash"`
    DisplayName  string      `json:"display_name" db:"display_name"`
    Email        string      `json:"email,omitempty" db:"email"`
    Phone        string      `json:"phone,omitempty" db:"phone"`
    DepartmentID *uuid.UUID  `json:"department_id,omitempty" db:"department_id"`
    Department   *Department `json:"department,omitempty" db:"department"`
    Carrier      *string     `json:"carrier,omitempty" db:"carrier"`
    Status       UserStatus  `json:"status" db:"status"`
    ExpiredAt    *time.Time  `json:"expired_at,omitempty" db:"expired_at"`
    LastLoginAt  *time.Time  `json:"last_login_at,omitempty" db:"last_login_at"`
    Remark       string      `json:"remark,omitempty" db:"remark"`
    Roles        []Role      `json:"roles,omitempty" db:"roles"`
    CreatedAt    time.Time   `json:"created_at" db:"created_at"`
    UpdatedAt    time.Time   `json:"updated_at" db:"updated_at"`
    CreatedBy    *uuid.UUID  `json:"created_by,omitempty" db:"created_by"`
    UpdatedBy    *uuid.UUID  `json:"updated_by,omitempty" db:"updated_by"`
}
```

### 4.2 部门模型

```go
// omcgo/internal/admin/model.go

type DepartmentStatus string

const (
    DeptStatusActive   DepartmentStatus = "active"
    DeptStatusDisabled DepartmentStatus = "disabled"
)

type Department struct {
    ID         uuid.UUID        `json:"id" db:"id"`
    Name       string           `json:"name" db:"name"`
    ParentID   *uuid.UUID       `json:"parent_id,omitempty" db:"parent_id"`
    Parent     *Department      `json:"parent,omitempty"`
    Children   []Department     `json:"children,omitempty"`
    SortOrder  int              `json:"sort_order" db:"sort_order"`
    Path       string           `json:"path" db:"path"`
    Level      int              `json:"level" db:"level"`
    Status     DepartmentStatus `json:"status" db:"status"`
    UserCount  int              `json:"user_count,omitempty" db:"user_count"`
    CreatedAt  time.Time        `json:"created_at" db:"created_at"`
    UpdatedAt  time.Time        `json:"updated_at" db:"updated_at"`
    CreatedBy  *uuid.UUID       `json:"created_by,omitempty" db:"created_by"`
    UpdatedBy  *uuid.UUID       `json:"updated_by,omitempty" db:"updated_by"`
}

// DepartmentTree 用于前端树形展示
type DepartmentTree struct {
    Department
    Children []DepartmentTree `json:"children,omitempty"`
}
```

### 4.3 角色模型

```go
// omcgo/internal/admin/model.go

type DataScopeType string

const (
    DataScopeAll        DataScopeType = "all"         // 全部数据
    DataScopeDepartment DataScopeType = "department"  // 本部门及下级
    DataScopeSelf       DataScopeType = "self"        // 仅本人
    DataScopeCustom     DataScopeType = "custom"      // 自定义设备组
)

type BatchPermission string

const (
    BatchSingle BatchPermission = "single"  // 仅单条操作
    BatchAllow  BatchPermission = "batch"   // 允许批量操作
)

type Role struct {
    ID              uuid.UUID      `json:"id" db:"id"`
    Name            string         `json:"name" db:"name"`
    Code            string         `json:"code" db:"code"`
    Description     string         `json:"description" db:"description"`
    IsSystem        bool           `json:"is_system" db:"is_system"`
    DataScopeType   DataScopeType  `json:"data_scope_type" db:"data_scope_type"`
    BatchPermission BatchPermission `json:"batch_permission" db:"batch_permission"`
    Menus           []Menu         `json:"menus,omitempty"`
    Permissions     []Permission   `json:"permissions,omitempty"`
    DeviceGroupIDs  []uuid.UUID    `json:"device_group_ids,omitempty"`
    CreatedAt       time.Time      `json:"created_at" db:"created_at"`
    UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
    CreatedBy       *uuid.UUID     `json:"created_by,omitempty" db:"created_by"`
    UpdatedBy       *uuid.UUID     `json:"updated_by,omitempty" db:"updated_by"`
}
```

### 4.4 菜单模型

```go
// omcgo/internal/admin/model.go

type MenuType string

const (
    MenuTypeDirectory MenuType = "directory"  // 目录
    MenuTypeMenu      MenuType = "menu"       // 菜单
    MenuTypeButton    MenuType = "button"     // 按钮
)

type MenuStatus string

const (
    MenuStatusActive   MenuStatus = "active"
    MenuStatusDisabled MenuStatus = "disabled"
)

type Menu struct {
    ID         uuid.UUID  `json:"id" db:"id"`
    Name       string     `json:"name" db:"name"`
    Type       MenuType   `json:"type" db:"type"`
    ParentID   *uuid.UUID `json:"parent_id,omitempty" db:"parent_id"`
    Path       string     `json:"path,omitempty" db:"path"`
    Component  string     `json:"component,omitempty" db:"component"`
    Permission string     `json:"permission,omitempty" db:"permission"`
    Icon       string     `json:"icon,omitempty" db:"icon"`
    SortOrder  int        `json:"sort_order" db:"sort_order"`
    Visible    bool       `json:"visible" db:"visible"`
    Status     MenuStatus `json:"status" db:"status"`
    Children   []Menu     `json:"children,omitempty"`
    CreatedAt  time.Time  `json:"created_at" db:"created_at"`
    UpdatedAt  time.Time  `json:"updated_at" db:"updated_at"`
}

// MenuTree 用于前端路由生成
type MenuTree struct {
    Menu
    Children []MenuTree `json:"children,omitempty"`
}

// UserMenuInfo 用户菜单信息（返回给前端）
type UserMenuInfo struct {
    Menus     []MenuTree `json:"menus"`      // 用户可访问的菜单树
    Perms     []string   `json:"permissions"` // 用户所有权限标识
    Roles     []string   `json:"roles"`       // 用户角色列表
}
```

### 4.5 登录日志模型

```go
// omcgo/internal/admin/model.go

type LoginStatus string

const (
    LoginStatusSuccess LoginStatus = "success"
    LoginStatusFailed  LoginStatus = "failed"
)

type LoginLog struct {
    ID          uuid.UUID   `json:"id" db:"id"`
    Username    string      `json:"username" db:"username"`
    LoginIP     string      `json:"login_ip" db:"login_ip"`
    LoginSource string      `json:"login_source,omitempty" db:"login_source"`
    UserAgent   string      `json:"user_agent,omitempty" db:"user_agent"`
    LoginStatus LoginStatus `json:"login_status" db:"login_status"`
    FailReason  string      `json:"fail_reason,omitempty" db:"fail_reason"`
    LoginAt     time.Time   `json:"login_at" db:"login_at"`
}
```

---

## 5. 权限检查机制

### 5.1 功能权限检查

#### 5.1.1 菜单权限

```go
// omcgo/internal/admin/middleware.go

// PermissionMiddleware 检查用户是否有访问特定资源的权限
func PermissionMiddleware(requiredPerm string) gin.HandlerFunc {
    return func(c *gin.Context) {
        claims, exists := c.Get("claims")
        if !exists {
            c.AbortWithStatusJSON(401, gin.H{"error": "unauthorized"})
            return
        }

        userClaims := claims.(*admin.Claims)

        // 超级管理员跳过检查
        if containsRole(userClaims.Roles, "super_admin") {
            c.Next()
            return
        }

        // 检查权限
        if !hasPermission(c.Request.Context(), userClaims.UserID, requiredPerm) {
            c.AbortWithStatusJSON(403, gin.H{"error": "permission denied"})
            return
        }

        c.Next()
    }
}

// hasPermission 检查用户是否拥有指定权限
func hasPermission(ctx context.Context, userID uuid.UUID, permission string) bool {
    // 1. 获取用户所有角色的菜单权限
    // 2. 检查 permission 是否在其中
    // 可通过缓存优化
    return true
}
```

#### 5.1.2 API 权限注解

```go
// 使用注解方式标记 API 所需权限

// @Summary 获取用户列表
// @Security BearerAuth
// @Param permission header string false "权限标识" default(system:user:list)
// @Router /api/v1/users [get]
func (h *Handler) ListUsers(c *gin.Context) {
    // 自动检查 system:user:list 权限
}
```

### 5.2 数据权限检查

#### 5.2.1 数据权限类型

| 类型 | 说明 | SQL 过滤 |
|------|------|---------|
| `all` | 全部数据 | 无过滤 |
| `department` | 本部门及下级 | `WHERE department_id IN (子部门IDs)` |
| `self` | 仅本人创建 | `WHERE created_by = current_user_id` |
| `custom` | 自定义设备组 | `WHERE device_group_id IN (角色关联的设备组)` |

#### 5.2.2 数据权限过滤器

```go
// omcgo/internal/admin/data_scope.go

type DataScopeFilter struct {
    ScopeType       DataScopeType
    DepartmentIDs   []uuid.UUID  // 部门数据权限
    DeviceGroupIDs  []uuid.UUID  // 设备组数据权限
    CurrentUserID   uuid.UUID
}

// ApplyDataScope 将数据权限应用到查询
func ApplyDataScope(builder sq.SelectBuilder, filter DataScopeFilter, tableAlias string) sq.SelectBuilder {
    switch filter.ScopeType {
    case DataScopeAll:
        return builder
    case DataScopeDepartment:
        if len(filter.DepartmentIDs) > 0 {
            return builder.Where(sq.Eq{tableAlias + ".department_id": filter.DepartmentIDs})
        }
    case DataScopeSelf:
        return builder.Where(sq.Eq{tableAlias + ".created_by": filter.CurrentUserID})
    case DataScopeCustom:
        if len(filter.DeviceGroupIDs) > 0 {
            // 需要 JOIN device_group_members 表
            return builder.Where(sq.Eq{tableAlias + ".device_group_id": filter.DeviceGroupIDs})
        }
    }
    return builder
}
```

### 5.3 批量操作权限

```go
// omcgo/internal/admin/batch_check.go

// CheckBatchPermission 检查是否允许批量操作
func CheckBatchPermission(ctx context.Context, userID uuid.UUID, operationCount int) error {
    if operationCount <= 1 {
        return nil // 单条操作始终允许
    }

    // 获取用户角色的批量权限
    roles, err := getUserRoles(ctx, userID)
    if err != nil {
        return err
    }

    for _, role := range roles {
        if role.BatchPermission == BatchAllow {
            return nil
        }
    }

    return errors.New("batch operation not permitted")
}
```

---

## 6. API 设计

### 6.1 用户管理 API

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | `/api/v1/users` | 用户列表 | `system:user:list` |
| GET | `/api/v1/users/:id` | 用户详情 | `system:user:read` |
| POST | `/api/v1/users` | 创建用户 | `system:user:create` |
| PUT | `/api/v1/users/:id` | 更新用户 | `system:user:update` |
| DELETE | `/api/v1/users/:id` | 删除用户 | `system:user:delete` |
| POST | `/api/v1/users/:id/reset-password` | 重置密码 | `system:user:reset-pwd` |
| PUT | `/api/v1/users/:id/status` | 启用/禁用 | `system:user:update` |
| POST | `/api/v1/users/import` | 导入用户 | `system:user:import` |
| GET | `/api/v1/users/export` | 导出用户 | `system:user:export` |

### 6.2 部门管理 API

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | `/api/v1/departments` | 部门列表 | `system:dept:list` |
| GET | `/api/v1/departments/tree` | 部门树 | `system:dept:list` |
| GET | `/api/v1/departments/:id` | 部门详情 | `system:dept:read` |
| POST | `/api/v1/departments` | 创建部门 | `system:dept:create` |
| PUT | `/api/v1/departments/:id` | 更新部门 | `system:dept:update` |
| DELETE | `/api/v1/departments/:id` | 删除部门 | `system:dept:delete` |

### 6.3 角色管理 API

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | `/api/v1/roles` | 角色列表 | `system:role:list` |
| GET | `/api/v1/roles/:id` | 角色详情（含权限） | `system:role:read` |
| POST | `/api/v1/roles` | 创建角色 | `system:role:create` |
| PUT | `/api/v1/roles/:id` | 更新角色 | `system:role:update` |
| DELETE | `/api/v1/roles/:id` | 删除角色 | `system:role:delete` |
| PUT | `/api/v1/roles/:id/menus` | 配置菜单权限 | `system:role:config` |
| PUT | `/api/v1/roles/:id/device-groups` | 配置数据权限 | `system:role:config` |

### 6.4 菜单管理 API

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | `/api/v1/menus` | 菜单列表 | `system:menu:list` |
| GET | `/api/v1/menus/tree` | 菜单树 | `system:menu:list` |
| GET | `/api/v1/menus/user` | 当前用户菜单 | 无需权限 |
| POST | `/api/v1/menus` | 创建菜单 | `system:menu:create` |
| PUT | `/api/v1/menus/:id` | 更新菜单 | `system:menu:update` |
| DELETE | `/api/v1/menus/:id` | 删除菜单 | `system:menu:delete` |

### 6.5 登录日志 API

| 方法 | 路径 | 说明 | 权限 |
|------|------|------|------|
| GET | `/api/v1/login-logs` | 登录日志列表 | `system:log:list` |
| DELETE | `/api/v1/login-logs/:id` | 删除日志 | `system:log:delete` |

---

## 7. 菜单数据初始化

### 7.1 系统菜单结构

```
系统管理 (system)
├── 用户管理 (system:user)
│   ├── 用户列表 (system:user:list)
│   ├── 新增用户 (system:user:create)
│   ├── 修改用户 (system:user:update)
│   ├── 删除用户 (system:user:delete)
│   ├── 重置密码 (system:user:reset-pwd)
│   └── 导入导出 (system:user:import)
├── 部门管理 (system:dept)
│   ├── 部门列表 (system:dept:list)
│   ├── 新增部门 (system:dept:create)
│   ├── 修改部门 (system:dept:update)
│   └── 删除部门 (system:dept:delete)
├── 角色管理 (system:role)
│   ├── 角色列表 (system:role:list)
│   ├── 新增角色 (system:role:create)
│   ├── 修改角色 (system:role:update)
│   └── 删除角色 (system:role:delete)
├── 菜单管理 (system:menu)
│   ├── 菜单列表 (system:menu:list)
│   ├── 新增菜单 (system:menu:create)
│   ├── 修改菜单 (system:menu:update)
│   └── 删除菜单 (system:menu:delete)
├── 字典管理 (system:dict)
├── 参数管理 (system:config)
├── 日志管理 (system:log)
│   ├── 登录日志 (system:log:login)
│   ├── 操作日志 (system:log:audit)
│   └── 任务日志 (system:log:task)
└── 国际化管理 (system:i18n)

设备管理 (device)
├── 设备列表 (device:list)
├── 设备详情 (device:detail)
├── 设备配置 (device:config)
└── ...

性能管理 (pm)
告警管理 (alarm)
...
```

### 7.2 菜单初始化 SQL

```sql
-- Migration: 000046_seed_menus.up.sql

-- 系统管理目录
INSERT INTO menus (id, name, type, parent_id, path, icon, sort_order, permission) VALUES
    (gen_random_uuid(), '系统管理', 'directory', NULL, '/system', 'setting', 1, NULL);

-- 用户管理
INSERT INTO menus (name, type, parent_id, path, component, sort_order, permission) VALUES
    ('用户管理', 'menu', (SELECT id FROM menus WHERE path = '/system'), '/system/user', 'system/user/index', 1, 'system:user:list'),
    ('用户查询', 'button', (SELECT id FROM menus WHERE path = '/system/user'), NULL, NULL, 1, 'system:user:list'),
    ('用户新增', 'button', (SELECT id FROM menus WHERE path = '/system/user'), NULL, NULL, 2, 'system:user:create'),
    ('用户修改', 'button', (SELECT id FROM menus WHERE path = '/system/user'), NULL, NULL, 3, 'system:user:update'),
    ('用户删除', 'button', (SELECT id FROM menus WHERE path = '/system/user'), NULL, NULL, 4, 'system:user:delete'),
    ('重置密码', 'button', (SELECT id FROM menus WHERE path = '/system/user'), NULL, NULL, 5, 'system:user:reset-pwd'),
    ('导入导出', 'button', (SELECT id FROM menus WHERE path = '/system/user'), NULL, NULL, 6, 'system:user:import');

-- ... 其他菜单类似 ...
```

---

## 8. 实施计划

### 8.1 阶段一：数据库扩展

| 序号 | 任务 | 产出物 |
|------|------|--------|
| 1 | 扩展用户表 | `000039_extend_users_table.up.sql` |
| 2 | 创建部门表 | `000040_create_departments.up.sql` |
| 3 | 扩展角色表 | `000041_extend_roles_table.up.sql` |
| 4 | 创建菜单表 | `000042_create_menus.up.sql` |
| 5 | 创建角色菜单关联表 | `000043_create_role_menus.up.sql` |
| 6 | 创建角色设备组关联表 | `000044_create_role_device_groups.up.sql` |
| 7 | 创建登录日志表 | `000045_create_login_logs.up.sql` |
| 8 | 初始化菜单数据 | `000046_seed_menus.up.sql` |

### 8.2 阶段二：模型与仓储层

| 序号 | 任务 | 产出物 |
|------|------|--------|
| 1 | 扩展 User 模型 | `internal/admin/model.go` |
| 2 | 新增 Department 模型 | `internal/admin/model.go` |
| 3 | 扩展 Role 模型 | `internal/admin/model.go` |
| 4 | 新增 Menu 模型 | `internal/admin/model.go` |
| 5 | 新增 LoginLog 模型 | `internal/admin/model.go` |
| 6 | 部门仓储实现 | `internal/admin/pg_department_repository.go` |
| 7 | 菜单仓储实现 | `internal/admin/pg_menu_repository.go` |
| 8 | 登录日志仓储实现 | `internal/admin/pg_login_log_repository.go` |

### 8.3 阶段三：服务层

| 序号 | 任务 | 产出物 |
|------|------|--------|
| 1 | 部门服务 | `internal/admin/department_service.go` |
| 2 | 菜单服务 | `internal/admin/menu_service.go` |
| 3 | 权限服务 | `internal/admin/permission_service.go` |
| 4 | 数据权限过滤器 | `internal/admin/data_scope.go` |
| 5 | 批量操作检查 | `internal/admin/batch_check.go` |

### 8.4 阶段四：Handler 层

| 序号 | 任务 | 产出物 |
|------|------|--------|
| 1 | 部门 Handler | `internal/admin/department_handler.go` |
| 2 | 菜单 Handler | `internal/admin/menu_handler.go` |
| 3 | 登录日志 Handler | `internal/admin/login_log_handler.go` |
| 4 | 权限中间件增强 | `internal/admin/middleware.go` |

### 8.5 阶段五：测试

| 序号 | 任务 | 产出物 |
|------|------|--------|
| 1 | 部门服务测试 | `internal/admin/department_service_test.go` |
| 2 | 菜单服务测试 | `internal/admin/menu_service_test.go` |
| 3 | 权限检查测试 | `internal/admin/permission_test.go` |
| 4 | 数据权限测试 | `internal/admin/data_scope_test.go` |

---

## 9. 权限标识规范

### 9.1 命名规范

```
{模块}:{资源}:{操作}

模块: system, device, pm, alarm, mr, config, software, ...
资源: user, dept, role, menu, log, ...
操作: list, read, create, update, delete, import, export, ...
```

### 9.2 常用权限标识

| 权限标识 | 说明 |
|---------|------|
| `system:user:list` | 用户列表 |
| `system:user:read` | 用户详情 |
| `system:user:create` | 创建用户 |
| `system:user:update` | 更新用户 |
| `system:user:delete` | 删除用户 |
| `system:user:reset-pwd` | 重置密码 |
| `system:user:import` | 导入导出 |
| `system:dept:list` | 部门列表 |
| `system:role:list` | 角色列表 |
| `system:menu:list` | 菜单列表 |
| `device:list` | 设备列表 |
| `device:config` | 设备配置 |
| `device:reboot` | 设备重启 |
| `device:batch` | 批量操作 |

---

## 10. 前端集成

### 10.1 用户菜单获取

```typescript
// GET /api/v1/menus/user
interface UserMenuInfo {
  menus: MenuTree[];
  permissions: string[];
  roles: string[];
}

// 前端根据 menus 动态生成路由
// 根据 permissions 控制按钮显示
```

### 10.2 权限指令

```typescript
// Vue 指令
v-permission="'system:user:create'"

// React Hook
const hasPermission = usePermission('system:user:create');
```

---

## 11. 附录

### 11.1 文件变更清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `migrations/000039_extend_users_table.up.sql` | 新增 | 用户表扩展 |
| `migrations/000040_create_departments.up.sql` | 新增 | 部门表 |
| `migrations/000041_extend_roles_table.up.sql` | 新增 | 角色表扩展 |
| `migrations/000042_create_menus.up.sql` | 新增 | 菜单表 |
| `migrations/000043_create_role_menus.up.sql` | 新增 | 角色菜单关联 |
| `migrations/000044_create_role_device_groups.up.sql` | 新增 | 角色设备组关联 |
| `migrations/000045_create_login_logs.up.sql` | 新增 | 登录日志表 |
| `migrations/000046_seed_menus.up.sql` | 新增 | 菜单初始数据 |
| `internal/admin/model.go` | 修改 | 扩展模型定义 |
| `internal/admin/department_service.go` | 新增 | 部门服务 |
| `internal/admin/menu_service.go` | 新增 | 菜单服务 |
| `internal/admin/permission_service.go` | 新增 | 权限服务 |
| `internal/admin/data_scope.go` | 新增 | 数据权限过滤 |
| `internal/admin/batch_check.go` | 新增 | 批量操作检查 |
| `internal/admin/pg_department_repository.go` | 新增 | 部门仓储 |
| `internal/admin/pg_menu_repository.go` | 新增 | 菜单仓储 |
| `internal/admin/pg_login_log_repository.go` | 新增 | 登录日志仓储 |
| `internal/admin/department_handler.go` | 新增 | 部门 Handler |
| `internal/admin/menu_handler.go` | 新增 | 菜单 Handler |
| `internal/admin/login_log_handler.go` | 新增 | 登录日志 Handler |
| `internal/admin/middleware.go` | 修改 | 权限中间件增强 |

### 11.2 预置角色

| 角色名 | Code | 数据权限 | 批量操作 | 说明 |
|--------|------|---------|---------|------|
| 超级管理员 | `super_admin` | all | batch | 系统内置，不可删除 |
| 系统管理员 | `admin` | all | batch | 系统内置 |
| 普通用户 | `user` | self | single | 系统内置 |
| 部门管理员 | `dept_admin` | department | batch | 可自定义 |
| 运维人员 | `operator` | custom | single | 可自定义 |
