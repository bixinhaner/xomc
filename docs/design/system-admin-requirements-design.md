# OMC 系统管理功能需求设计文档

> **文档版本**: v2.0  
> **创建日期**: 2026-04-13  
> **适用项目**: OMC（基站网络运营管理系统）  
> **技术栈**: Go + Gin + PostgreSQL + TimescaleDB + Casbin + React 18

---

## 1. 文档概述

### 1.1 文档目的

本文档基于 OMC 项目当前系统管理模块的实现现状，整理已有功能的详细规格，并提出待完善事项与实施建议，作为后续开发和测试的功能需求设计依据。

文档覆盖系统管理模块的五大功能子模块，以及贯穿各模块的数据权限设计方案。

### 1.2 适用范围

本文档适用于 OMC 系统后台管理模块，包括：

| 模块 | 说明 |
|------|------|
| 用户管理 | 用户 CRUD、登录认证、密码管理、账户锁定、角色分配 |
| 角色管理 | 角色 CRUD、权限配置（菜单/API/数据）、角色继承 |
| 菜单管理 | 动态菜单树、路由配置、按钮级权限 |
| API 管理 | API 端点注册、Casbin 权限集成、权限验证中间件 |
| 字典管理 | 系统枚举字典、前端缓存策略 |

### 1.3 项目技术栈

- **后端**: Go 1.21+ + Gin 框架 + sqlc（原生 SQL）
- **数据库**: PostgreSQL 15 + TimescaleDB
- **权限引擎**: Casbin v2（PostgreSQL 适配器 + Redis 分布式同步）
- **认证**: JWT（golang-jwt）
- **前端**: React 18 + TypeScript + Vite + Ant Design 5 (antd) + @ant-design/pro-components
- **缓存**: Redis（会话管理、策略同步）

---

## 2. 功能总览

### 2.1 五大功能模块

```
┌─────────────────────────────────────────────────────────┐
│                    OMC 系统管理模块                        │
├──────────┬──────────┬──────────┬──────────┬────────────┤
│  用户管理 │  角色管理 │  菜单管理 │  API管理  │  字典管理  │
│  (User)  │  (Role)  │  (Menu)  │  (API)   │(Dictionary)│
└──────────┴──────────┴──────────┴──────────┴────────────┘
```

### 2.2 模块间关系图

```
用户 (users)
  └── 持有多角色 (user_roles) ─────────────────────┐
                                                    ▼
                                         角色 (roles)
                                           ├── 拥有菜单权限 ──────► 菜单 (menus)
                                           │    (role_menus)
                                           ├── 拥有API权限 ───────► Casbin 规则
                                           │    (permissions)
                                           └── 拥有数据权限 ──────► 设备分组 (device_groups)
                                                (role_device_groups)

字典 (sys_dictionaries) ──► 字典详情 (sys_dictionary_details)
  [独立模块，为其他模块提供枚举值]
```

### 2.3 RBAC + 数据权限模型说明

OMC 系统采用 **RBAC + 数据权限** 的复合权限模型：

```
用户 (users)
  └── 属于 ──► 角色 (roles)
                  ├── 菜单权限：控制可访问的前端页面和按钮
                  ├── API 权限：控制可调用的后端接口（Casbin）
                  └── 数据权限：控制可查看的设备数据范围
                                  ├── 设备分组维度（device_groups树）
                                  └── 网络类型维度（eNB/gNB/GSM/CPE/eGW）
```

**运营商隔离层**：所有查询在数据权限之上自动应用运营商（carrier）过滤，确保跨运营商数据严格隔离。

### 2.4 Casbin 权限模型

**模型文件** (`configs/casbin_model.conf`):

```ini
[request_definition]
r = sub, dom, obj, act   # 主体、域(运营商)、对象(资源)、操作

[policy_definition]
p = sub, dom, obj, act   # 与请求定义对应

[role_definition]
g = _, _, _              # 角色继承（含域隔离）

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub, r.dom) && r.dom == p.dom && r.obj == p.obj && r.act == p.act || r.sub == "admin"
```

策略示例：`(p, role_id, carrier_domain, resource, action)`

---

## 3. 用户管理

### 3.1 功能需求

| 需求编号 | 功能描述 | 状态 | 优先级 |
|---------|---------|------|--------|
| USR-001 | 用户列表查询（分页、多条件搜索） | ✅ 已实现 | P0 |
| USR-002 | 新增用户（设置用户名、密码、显示名、角色） | ✅ 已实现 | P0 |
| USR-003 | 编辑用户信息 | ✅ 已实现 | P0 |
| USR-004 | 删除用户 | ✅ 已实现 | P0 |
| USR-005 | 启用/禁用用户账号 | ✅ 已实现 | P0 |
| USR-006 | 管理员重置用户密码 | ✅ 已实现 | P0 |
| USR-007 | 为用户分配角色（支持多角色） | ✅ 已实现 | P0 |
| USR-008 | 移除用户角色 | ✅ 已实现 | P0 |
| USR-009 | 锁定用户账号 | ✅ 已实现 | P1 |
| USR-010 | 解锁用户账号 | ✅ 已实现 | P1 |
| USR-011 | 用户登录（用户名+密码，支持验证码） | ✅ 已实现 | P0 |
| USR-012 | JWT Token 刷新 | ✅ 已实现 | P0 |
| USR-013 | 获取当前登录用户信息 | ✅ 已实现 | P0 |
| USR-014 | 切换用户当前角色 | ✅ 已实现 | P1 |
| USR-015 | 修改自己的密码 | ⚠️ 待完善 | P1 |

### 3.2 数据模型

#### 主表：users

```sql
CREATE TABLE users (
    id                    UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    username              VARCHAR(64)     NOT NULL UNIQUE,
    display_name          VARCHAR(128)    NOT NULL DEFAULT '',
    email                 VARCHAR(256)    UNIQUE,
    phone                 VARCHAR(32),
    password_hash         VARCHAR(256)    NOT NULL,
    carrier               VARCHAR(4),
    status                VARCHAR(16)     NOT NULL DEFAULT 'active',  -- active/disabled
    failed_login_attempts INTEGER         NOT NULL DEFAULT 0,
    locked_until          TIMESTAMPTZ,
    last_login_at         TIMESTAMPTZ,
    created_at            TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    created_by            VARCHAR(64),
    updated_by            VARCHAR(64)
);

CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_carrier  ON users(carrier);
```

#### 关联表：user_roles（用户-角色多对多）

```sql
CREATE TABLE user_roles (
    user_id     UUID    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id     UUID    NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);
```

### 3.3 API 接口设计

#### 认证接口（无需 JWT）

| 接口名称 | 方法 | 路径 | 说明 |
|---------|------|------|------|
| 用户登录 | POST | `/api/v1/auth/login` | 用户名+密码登录，返回 JWT |
| 刷新 Token | POST | `/api/v1/auth/refresh` | 使用 RefreshToken 换取新 AccessToken |
| 获取验证码 | GET | `/api/v1/auth/captcha` | 图形验证码（Base64 图像） |

**登录请求**:
```json
{
  "username": "admin",
  "password": "a@123456",
  "captcha": "A3Xk",
  "captcha_id": "xxxxxxxx"
}
```

**登录响应**:
```json
{
  "code": 0,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "...",
    "expires_at": 1713086400,
    "user": {
      "id": "uuid",
      "username": "admin",
      "display_name": "系统管理员",
      "email": "admin@example.com",
      "carrier": "ct",
      "roles": [{ "id": "uuid", "role_name": "超级管理员" }]
    }
  },
  "msg": "登录成功"
}
```

#### 受保护接口（需要 JWT）

| 接口名称 | 方法 | 路径 | 说明 |
|---------|------|------|------|
| 获取当前用户 | GET | `/api/v1/auth/me` | 获取当前登录用户详情 |
| 切换角色 | POST | `/api/v1/auth/switch-role` | 切换当前生效角色 |
| 获取用户菜单 | GET | `/api/v1/auth/menus` | 获取当前用户可见菜单树 |

#### 用户管理接口（需要 JWT + admin 权限）

| 接口名称 | 方法 | 路径 | 说明 |
|---------|------|------|------|
| 获取用户列表 | GET | `/api/v1/admin/users` | 分页查询用户 |
| 创建用户 | POST | `/api/v1/admin/users` | 管理员创建用户 |
| 获取用户详情 | GET | `/api/v1/admin/users/:id` | 获取单个用户信息 |
| 更新用户 | PUT | `/api/v1/admin/users/:id` | 修改用户信息 |
| 删除用户 | DELETE | `/api/v1/admin/users/:id` | 删除用户 |
| 分配角色 | POST | `/api/v1/admin/users/:id/roles` | 为用户分配角色 |
| 移除角色 | DELETE | `/api/v1/admin/users/:id/roles/:roleId` | 移除用户的某个角色 |
| 重置密码 | POST | `/api/v1/admin/users/:id/reset-password` | 管理员重置密码 |
| 锁定用户 | POST | `/api/v1/admin/users/:id/lock` | 锁定用户账号 |
| 解锁用户 | POST | `/api/v1/admin/users/:id/unlock` | 解锁用户账号 |

**用户列表查询参数**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| page | int | 否 | 页码，默认1 |
| page_size | int | 否 | 每页条数，默认20 |
| username | string | 否 | 用户名（模糊匹配） |
| display_name | string | 否 | 显示名（模糊匹配） |
| email | string | 否 | 邮箱（模糊匹配） |
| status | string | 否 | 状态筛选：active/disabled |
| carrier | string | 否 | 运营商筛选 |

### 3.4 页面设计

#### 用户列表页

**搜索区域字段**:

| 字段 | 控件类型 | 搜索方式 |
|------|---------|---------|
| 用户名 | Input 文本框 | 模糊匹配 |
| 显示名 | Input 文本框 | 模糊匹配 |
| 邮箱 | Input 文本框 | 模糊匹配 |
| 状态 | Select 下拉 | 精确匹配：正常/已禁用/已锁定 |

**操作按钮**: 查询、重置、新增用户

**列表字段**:

| 列名 | 字段 | 类型 | 说明 |
|------|------|------|------|
| 用户名 | username | Text | 登录用户名 |
| 显示名 | display_name | Text | 展示名称 |
| 邮箱 | email | Text | 电子邮箱 |
| 运营商 | carrier | Tag | 所属运营商 |
| 角色 | roles | Tag | 角色标签（多个时显示"+N"） |
| 状态 | status | Badge | 正常/已禁用/已锁定 |
| 最后登录 | last_login_at | DateTime | 最近登录时间 |
| 操作 | - | Buttons | 编辑、重置密码、锁定/解锁、删除 |

#### 新增/编辑用户表单

| 字段名 | 控件类型 | 必填 | 校验规则 | 说明 |
|--------|---------|------|---------|------|
| 用户名 | Input | 是 | 长度4-64，字母数字下划线 | 唯一，编辑时不可修改 |
| 密码 | Password | 是（新增）/ 否（编辑） | 最小长度由系统配置决定 | 编辑时留空则不修改 |
| 显示名 | Input | 是 | 长度1-128 | 展示名称 |
| 邮箱 | Input | 否 | 邮箱格式，唯一 | 电子邮箱 |
| 手机号 | Input | 否 | 格式验证 | 联系电话 |
| 运营商 | Select | 否 | - | 所属运营商 |
| 角色 | Select（多选） | 是 | 至少选1个 | 从角色列表中选择 |
| 状态 | Switch | - | - | 默认启用 |

### 3.5 业务规则

#### 3.5.1 JWT 认证流程

```
用户提交 [用户名 + 密码（+ 验证码）]
          │
          ▼
    验证码校验（配置项：是否启用）
          │
          ▼
    查询用户 WHERE username = $1
          │
     ┌────┴────┐
  不存在     存在
     │          │
     ▼          ▼
  返回错误   检查 locked_until > NOW()
               │
          ┌────┴────┐
       已锁定     未锁定
          │          │
          ▼          ▼
       返回错误   bcrypt 密码比对
                    │
               ┌────┴────┐
            密码错误   密码正确
               │          │
               ▼          ▼
    failed_login_attempts++   检查 status = 'active'
    达到上限则锁定账号          │
                         ┌────┴────┐
                      已禁用     正常
                         │          │
                         ▼          ▼
                      返回错误   生成 JWT (AccessToken + RefreshToken)
                                       │
                                       ▼
                               重置 failed_login_attempts = 0
                               更新 last_login_at = NOW()
                                       │
                                       ▼
                               返回 Token + 用户信息
```

#### 3.5.2 密码策略

- 使用 **bcrypt** 算法加密，cost 值建议 12
- 密码最小长度由系统配置 `password_min_length` 控制（默认6位）
- 重置密码时生成随机初始密码或使用固定初始密码（`a@123456`）
- 建议生产环境强制要求包含大写字母、小写字母和数字

#### 3.5.3 账户锁定策略

- 连续登录失败次数超过 `max_login_attempts`（默认5次）时自动锁定
- 锁定时长由 `lockout_duration`（默认30分钟）控制
- 锁定期间登录请求直接拒绝，不再验证密码
- 管理员可通过 `POST /api/v1/admin/users/:id/unlock` 手动解锁

#### 3.5.4 JWT Claims 结构

```go
type Claims struct {
    UserID    string `json:"user_id"`
    Username  string `json:"username"`
    RoleID    string `json:"role_id"`    // 当前生效角色
    Carrier   string `json:"carrier"`   // 运营商（用于数据隔离）
    jwt.RegisteredClaims
}
```

---

## 4. 角色管理

### 4.1 功能需求

| 需求编号 | 功能描述 | 状态 | 优先级 |
|---------|---------|------|--------|
| ROLE-001 | 角色列表查询（分页） | ✅ 已实现 | P0 |
| ROLE-002 | 新增角色 | ✅ 已实现 | P0 |
| ROLE-003 | 编辑角色信息 | ✅ 已实现 | P0 |
| ROLE-004 | 删除角色（不可删除系统内置角色） | ✅ 已实现 | P0 |
| ROLE-005 | 配置角色菜单权限（树形勾选） | ✅ 已实现 | P0 |
| ROLE-006 | 配置角色 API 权限（表格勾选） | ✅ 框架已建立 | P0 |
| ROLE-007 | 配置角色数据权限（设备分组树 + 网络类型） | ✅ 已实现 | P0 |
| ROLE-008 | 获取角色下的用户列表 | ⚠️ 待完善 | P1 |
| ROLE-009 | 角色继承配置 | ✅ 已实现 | P1 |
| ROLE-010 | 获取所有角色（用于下拉框） | ✅ 已实现 | P0 |

### 4.2 数据模型

#### 主表：roles

```sql
CREATE TABLE roles (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(64) NOT NULL UNIQUE,
    description TEXT,
    is_system   BOOLEAN     NOT NULL DEFAULT FALSE,  -- 系统内置角色，不可删除
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by  VARCHAR(64),
    updated_by  VARCHAR(64)
);

CREATE INDEX idx_roles_name ON roles(name);
```

#### 关联表：permissions（角色-资源权限）

```sql
CREATE TABLE permissions (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id     UUID        NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    resource    VARCHAR(64) NOT NULL,  -- 资源名：users, roles, devices, alarms...
    action      VARCHAR(32) NOT NULL,  -- 操作：create, read, update, delete, admin
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (role_id, resource, action)
);

CREATE INDEX idx_permissions_role_id ON permissions(role_id);
```

#### 关联表：role_menus（角色-菜单）

```sql
CREATE TABLE role_menus (
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    menu_id     UUID NOT NULL REFERENCES menus(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, menu_id)
);
```

#### 关联表：role_device_groups（角色-设备分组数据权限）

```sql
CREATE TABLE role_device_groups (
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    group_id    UUID NOT NULL REFERENCES device_groups(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, group_id)
);

CREATE INDEX idx_role_device_groups_role_id  ON role_device_groups(role_id);
CREATE INDEX idx_role_device_groups_group_id ON role_device_groups(group_id);
```

#### 关联表：role_inheritance（角色继承）

```sql
CREATE TABLE role_inheritance (
    parent_role_id  UUID        NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    child_role_id   UUID        NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    domain          VARCHAR(64) NOT NULL DEFAULT 'default',
    PRIMARY KEY (parent_role_id, child_role_id, domain)
);
```

### 4.3 API 接口设计

| 接口名称 | 方法 | 路径 | 说明 |
|---------|------|------|------|
| 获取角色列表 | GET | `/api/v1/admin/roles` | 分页查询角色 |
| 获取所有角色 | GET | `/api/v1/admin/roles/all` | 不分页（用于下拉框） |
| 获取角色详情 | GET | `/api/v1/admin/roles/:id` | 获取单个角色信息 |
| 创建角色 | POST | `/api/v1/admin/roles` | 新增角色 |
| 更新角色 | PUT | `/api/v1/admin/roles/:id` | 修改角色信息 |
| 删除角色 | DELETE | `/api/v1/admin/roles/:id` | 删除角色（系统内置不可删） |
| 获取角色菜单权限 | GET | `/api/v1/admin/roles/:id/menus` | 获取已分配菜单 |
| 设置角色菜单权限 | PUT | `/api/v1/admin/roles/:id/menus` | 全量覆盖角色菜单 |
| 获取角色API权限 | GET | `/api/v1/admin/roles/:id/api-permissions` | 获取已分配 API 权限 |
| 设置角色API权限 | PUT | `/api/v1/admin/roles/:id/api-permissions` | 全量覆盖角色 API 权限 |
| 获取角色设备分组 | GET | `/api/v1/admin/roles/:id/device-groups` | 获取已分配设备分组 |
| 设置角色设备分组 | PUT | `/api/v1/admin/roles/:id/device-groups` | 全量覆盖角色设备分组 |

**创建角色请求**:
```json
{
  "name": "运维工程师",
  "description": "负责设备运维操作，只读告警查看"
}
```

**设置角色菜单权限请求**:
```json
{
  "menu_ids": ["uuid1", "uuid2", "uuid3"]
}
```

**设置角色 API 权限请求**:
```json
{
  "api_permissions": [
    { "method": "GET",  "path": "/api/v1/admin/users" },
    { "method": "POST", "path": "/api/v1/admin/users" }
  ]
}
```

**设置角色设备分组（数据权限）请求**:
```json
{
  "group_ids": ["uuid1", "uuid2"],
  "network_types": ["eNB", "gNB"]
}
```

### 4.4 页面设计

#### 角色列表页

**列表字段**:

| 列名 | 字段 | 类型 | 说明 |
|------|------|------|------|
| 角色名称 | name | Text | 角色显示名称 |
| 描述 | description | Text | 角色说明 |
| 系统内置 | is_system | Badge | 是/否（内置角色不可删除） |
| 创建时间 | created_at | DateTime | 角色创建时间 |
| 操作 | - | Buttons | 操作按钮组 |

**行内操作按钮**:
- **权限配置**: 打开权限配置抽屉（三标签页）
- **编辑**: 修改角色名称和描述
- **删除**: 删除角色（系统内置角色禁用此按钮）

#### 新增/编辑角色表单

| 字段名 | 控件类型 | 必填 | 校验规则 | 说明 |
|--------|---------|------|---------|------|
| 角色名称 | Input | 是 | 长度1-64，全局唯一 | 角色唯一标识名 |
| 描述 | Textarea | 否 | 长度0-500 | 角色功能说明 |

#### 权限配置抽屉（800px 宽，三标签页）

**Tab 1 - 角色菜单**:

| 控件 | 说明 |
|------|------|
| 展开/折叠所有 | 批量展开或收起菜单树 |
| 全选/全不选 | 批量勾选或取消全部菜单 |
| 菜单树（多选复选框） | 三层树形结构（目录→菜单→按钮），勾选即授权 |
| 确定/取消按钮 | 保存或放弃菜单权限变更 |

菜单树节点结构示例：
```
设备管理（目录）
  ├── 设备列表（菜单）
  │     ├── 查询（按钮）
  │     ├── 添加（按钮）
  │     ├── 编辑（按钮）
  │     └── 删除（按钮）
  ├── 设备分组（菜单）
  └── 设备注册（菜单）
```

**Tab 2 - 角色API**:

| 控件 | 说明 |
|------|------|
| 搜索框（API名称/路径） | 快速过滤 API 端点 |
| 全选/清除 | 批量选择或清除所有 API |
| API 端点表格 | 含 Method（彩色标签）、Path、名称、模块列，每行有复选框 |
| 分页（每页10条） | 避免一次性加载过多 |

HTTP 方法颜色规范：
- GET → 绿色
- POST → 蓝色
- PUT → 橙色
- DELETE → 红色

**Tab 3 - 数据权限**:

数据权限通过两个维度控制角色可见的设备数据范围：

**维度一：设备分组（device_groups 树）**

| 控件 | 说明 |
|------|------|
| 全选 | 勾选所有设备分组 |
| 设备分组树（两级） | L1 根分组（可展开）→ L2 叶分组（实际设备所属），勾选即授权该分组数据 |
| 设备数量展示 | 每个分组旁显示设备计数，帮助管理员判断范围 |

**维度二：网络类型筛选**

| 控件 | 说明 |
|------|------|
| 网络类型多选框 | 可选项：eNB (LTE)、gNB (5G)、GSM、CPE、eGW |
| 说明文本 | "选中的网络类型的设备才在授权分组内可见" |

数据权限的组合效果：
```
角色可见的设备 = 设备分组范围 ∩ 网络类型筛选
```

例如：分组选"华东大区/上海"，网络类型选"eNB+gNB"，则该角色只能看到上海分组内的 4G/5G 基站。

### 4.5 业务规则

#### 4.5.1 Casbin RBAC 策略更新流程

```
管理员修改角色 API 权限
          │
          ▼
后端接收新的 api_permissions 列表
          │
          ▼
事务操作：
  1. DELETE FROM permissions WHERE role_id = $1
  2. 批量 INSERT INTO permissions (role_id, resource, action)
  3. 同步到 Casbin 内存策略
          │
          ▼
Redis Pub/Sub 广播策略变更通知
          │
          ▼
所有 OMC 实例接收通知后各自刷新 Casbin 策略缓存
          │
          ▼
权限立即在所有实例生效
```

#### 4.5.2 系统内置角色保护

系统预置的角色（`is_system = TRUE`）受保护：
- **超级管理员**（admin）：拥有全部权限，`r.sub == "admin"` 直接放行
- **只读观察员**（observer）：只读权限，适用于外部审计
- 以上角色不可删除、不可修改 `is_system` 标志

#### 4.5.3 数据权限隔离机制

1. **运营商隔离**：JWT Claims 中携带 `carrier` 字段，所有数据查询自动附加 `WHERE carrier = $carrier` 条件
2. **设备分组隔离**：设备列表查询时，从 `role_device_groups` 获取当前角色的授权分组列表，自动过滤
3. **网络类型筛选**：可在角色数据权限中额外限制只能看特定网络类型的设备（扩展字段，存于 JSONB 或独立表）

---

## 5. 菜单管理

### 5.1 功能需求

| 需求编号 | 功能描述 | 状态 | 优先级 |
|---------|---------|------|--------|
| MENU-001 | 菜单列表（树形展示，支持折叠） | ✅ 已实现 | P0 |
| MENU-002 | 新增菜单（指定父级） | ✅ 已实现 | P0 |
| MENU-003 | 编辑菜单配置 | ✅ 已实现 | P0 |
| MENU-004 | 删除菜单（级联删除子菜单） | ✅ 已实现 | P0 |
| MENU-005 | 获取用户动态菜单树 | ✅ 已实现 | P0 |
| MENU-006 | 菜单排序调整 | ✅ 已实现 | P1 |
| MENU-007 | 菜单启用/禁用 | ✅ 已实现 | P1 |

### 5.2 数据模型

#### 主表：menus

```sql
CREATE TABLE menus (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(64) NOT NULL,
    type            VARCHAR(16) NOT NULL,        -- 'directory'|'menu'|'button'
    permission_key  VARCHAR(128),               -- 权限标识（如 device:list:query）
    parent_id       UUID        REFERENCES menus(id) ON DELETE CASCADE,
    sort_order      INTEGER     NOT NULL DEFAULT 0,
    route_path      VARCHAR(256),               -- React Router 路由路径
    component       VARCHAR(256),               -- 前端组件相对路径
    icon            VARCHAR(64),                -- 图标类名
    status          VARCHAR(16) NOT NULL DEFAULT 'active',
    is_hidden       BOOLEAN     NOT NULL DEFAULT FALSE,
    keep_alive      BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_menus_parent_id ON menus(parent_id);
CREATE INDEX idx_menus_type      ON menus(type);
```

**菜单层级说明**：
- `directory`：目录节点（如"设备管理"），有子菜单，无路由
- `menu`：页面菜单（如"设备列表"），对应前端路由
- `button`：按钮权限（如"添加设备"），绑定到操作权限 key

### 5.3 API 接口设计

| 接口名称 | 方法 | 路径 | 说明 |
|---------|------|------|------|
| 获取菜单列表 | GET | `/api/v1/admin/menus` | 获取所有菜单（平铺列表） |
| 获取菜单树 | GET | `/api/v1/admin/menus/tree` | 获取完整菜单树（管理用） |
| 获取用户菜单树 | GET | `/api/v1/admin/menus/user-tree` | 获取当前用户可见菜单树 |
| 获取菜单详情 | GET | `/api/v1/admin/menus/:id` | 获取单个菜单信息 |
| 创建菜单 | POST | `/api/v1/admin/menus` | 新增菜单项 |
| 更新菜单 | PUT | `/api/v1/admin/menus/:id` | 修改菜单配置 |
| 删除菜单 | DELETE | `/api/v1/admin/menus` | 删除菜单（含子菜单） |

**创建菜单请求**:

| 参数名 | 类型 | 必填 | 说明 |
|--------|------|------|------|
| name | string | 是 | 菜单名称 |
| type | string | 是 | 菜单类型：directory/menu/button |
| parent_id | string | 否 | 父级菜单 UUID（根目录留空） |
| route_path | string | 否（menu 类型必填） | 路由路径 |
| component | string | 否（menu 类型必填） | 组件路径 |
| permission_key | string | 否（button 类型必填） | 权限 key |
| icon | string | 否 | 图标 |
| sort_order | int | 否 | 排序号，默认0 |
| is_hidden | bool | 否 | 是否隐藏，默认 false |
| keep_alive | bool | 否 | 是否缓存，默认 false |

### 5.4 页面设计

#### 菜单列表页

**列表字段**:

| 列名 | 字段 | 类型 | 说明 |
|------|------|------|------|
| 菜单名称 | name | Text（树形缩进） | 菜单显示名称 |
| 类型 | type | Badge | 目录/菜单/按钮，不同颜色区分 |
| 路由路径 | route_path | Text | URL 路径 |
| 组件路径 | component | Text | 前端组件路径 |
| 排序 | sort_order | Number | 排序号 |
| 状态 | status | Switch | 启用/禁用 |
| 操作 | - | Buttons | 添加子菜单、编辑、删除 |

#### 编辑菜单表单

**基础信息区**:

| 字段名 | 控件类型 | 必填 | 说明 |
|--------|---------|------|------|
| 菜单名称 | Input | 是 | 菜单显示名称 |
| 菜单类型 | Radio | 是 | 目录/菜单/按钮 |
| 父级菜单 | TreeSelect | 否 | 选择父节点（根节点不选） |
| 图标 | IconSelect | 否 | 从图标库选择 |
| 排序号 | InputNumber | 否 | 数值越小越靠前 |

**路由配置区**（仅 menu 类型显示）:

| 字段名 | 控件类型 | 必填 | 说明 |
|--------|---------|------|------|
| 路由路径 | Input | 是 | URL 路径（如 /devices/list） |
| 组件路径 | Input | 是 | 前端组件路径（如 device/List） |
| KeepAlive | Switch | 否 | 是否缓存页面组件 |
| 是否隐藏 | Switch | 否 | 控制菜单显示/隐藏 |

**权限配置区**（仅 button 类型显示）:

| 字段名 | 控件类型 | 必填 | 说明 |
|--------|---------|------|------|
| 权限 Key | Input | 是 | 如 device:list:add |

### 5.5 业务规则

#### 5.5.1 动态菜单生成流程

```
用户登录成功
      │
      ▼
前端调用 GET /api/v1/auth/menus
      │
      ▼
后端从 JWT 解析 RoleID（当前生效角色）
      │
      ▼
查询角色关联菜单：
  SELECT m.* FROM menus m
  JOIN role_menus rm ON m.id = rm.menu_id
  WHERE rm.role_id = $1
  AND m.status = 'active'
  ORDER BY m.sort_order ASC
      │
      ▼
递归构建树形结构（parent_id = NULL 为根节点）
      │
      ▼
前端接收菜单树
      │
      ▼
动态注册 React Router 路由：
  component 字段 → () => import(`@/pages/${component}`)
      │
      ▼
渲染侧边栏导航菜单
```

#### 5.5.2 预置菜单结构

系统预置菜单（通过迁移脚本 seed 数据初始化）：

```
设备管理（directory）
  ├── 设备列表（menu）
  │     ├── 查询（button, key: device:list:query）
  │     ├── 添加（button, key: device:list:add）
  │     ├── 编辑（button, key: device:list:edit）
  │     └── 删除（button, key: device:list:delete）
  ├── 设备分组（menu）
  └── 设备注册（menu）
告警管理（directory）
  ├── 当前告警（menu）
  └── 历史告警（menu）
系统管理（directory）
  ├── 用户管理（menu）
  ├── 角色管理（menu）
  ├── 菜单管理（menu）
  └── 操作日志（menu）
```

---

## 6. API 管理

### 6.1 功能需求

| 需求编号 | 功能描述 | 状态 | 优先级 |
|---------|---------|------|--------|
| API-001 | 获取所有 API 端点列表 | ✅ 已实现 | P0 |
| API-002 | 获取角色的 API 权限配置 | ✅ 已实现 | P0 |
| API-003 | 设置角色的 API 权限 | ✅ 已实现 | P0 |
| API-004 | API 端点持久化到数据库 | ⚠️ 待完善 | P1 |
| API-005 | API 权限验证中间件（Casbin集成） | ⚠️ 待完善 | P0 |
| API-006 | 刷新 Casbin 权限缓存 | ✅ 已实现 | P1 |
| API-007 | API 分组管理 | ⚠️ 待完善 | P2 |

### 6.2 数据模型

#### 待完善：api_endpoints 表（设计方案）

当前 API 端点列表由后端代码硬编码，建议迁移到数据库持久化：

```sql
CREATE TABLE api_endpoints (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    path        VARCHAR(256) NOT NULL,           -- API 路径（如 /api/v1/admin/users）
    method      VARCHAR(16)  NOT NULL,           -- HTTP 方法（GET/POST/PUT/DELETE）
    name        VARCHAR(128) NOT NULL DEFAULT '', -- API 名称（如 获取用户列表）
    module      VARCHAR(64)  NOT NULL DEFAULT '', -- 所属模块（如 用户管理）
    description TEXT,
    is_public   BOOLEAN      NOT NULL DEFAULT FALSE, -- 是否公开（无需权限）
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (path, method)
);

CREATE INDEX idx_api_endpoints_module ON api_endpoints(module);
```

#### 现有实现：通过 permissions 表 + Casbin 管理 API 权限

```sql
-- 角色拥有某 API 的访问权限
-- resource = API path, action = HTTP method
INSERT INTO permissions (role_id, resource, action)
VALUES ($role_id, '/api/v1/admin/users', 'GET');
```

### 6.3 API 接口设计

| 接口名称 | 方法 | 路径 | 说明 |
|---------|------|------|------|
| 获取所有 API 端点 | GET | `/api/v1/admin/api-endpoints` | 返回系统所有注册的 API 列表 |
| 获取角色 API 权限 | GET | `/api/v1/admin/roles/:id/api-permissions` | 获取某角色已授权的 API 列表 |
| 设置角色 API 权限 | PUT | `/api/v1/admin/roles/:id/api-permissions` | 全量覆盖角色 API 权限 |

**获取 API 端点列表响应**:
```json
{
  "code": 0,
  "data": {
    "items": [
      {
        "method": "GET",
        "path": "/api/v1/admin/users",
        "name": "获取用户列表",
        "module": "用户管理"
      }
    ]
  }
}
```

### 6.4 页面设计

API 权限配置集成在角色管理的权限配置抽屉（Tab 2 - 角色API）中，不单独提供 API 管理页面。

前端 API 端点数据模型：
```typescript
interface ApiEndpoint {
  method: string;   // GET/POST/PUT/DELETE
  path: string;     // API 路径
  name: string;     // 端点名称（中文）
  module: string;   // 所属模块
}
```

### 6.5 业务规则

#### 6.5.1 权限验证中间件流程（当前实现）

```
HTTP 请求到达 Gin 路由
      │
      ▼
RequireAuthWithAPIKey 中间件：
  - 解析 Authorization: Bearer <token>
  - 或 X-API-Key: <key> 方式
  失败 → 401 Unauthorized
      │
      ▼
RequireCarrier 中间件：
  - 从 JWT Claims 注入运营商上下文
  - 后续查询自动应用运营商过滤
      │
      ▼
RequirePermission / RequireResourcePermission 中间件：
  - resource = 路由对应的资源名
  - action = 操作类型
  - 调用 casbin.Enforce(userID, carrier, resource, action)
  false → 403 Forbidden
      │
      ▼
AuditLogger 中间件：
  - 异步记录操作审计日志
      │
      ▼
业务处理器执行
```

#### 6.5.2 待完善：API 路由级权限中间件

目前权限验证基于资源-操作（resource/action）模式，建议扩展为支持基于 API 路径+HTTP 方法的 Casbin 规则检查：

```go
// 设计方案：API路由级权限中间件
func APIPermissionMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        claims := GetClaims(c)
        path   := c.FullPath()    // 模板路径，如 /api/v1/admin/users/:id
        method := c.Request.Method
        
        ok, _ := casbinEnforcer.Enforce(
            claims.RoleID,
            claims.Carrier,
            path,
            method,
        )
        if !ok {
            c.AbortWithStatusJSON(403, gin.H{"msg": "权限不足"})
            return
        }
        c.Next()
    }
}
```

---

## 7. 字典管理

### 7.1 功能需求

| 需求编号 | 功能描述 | 状态 | 优先级 |
|---------|---------|------|--------|
| DICT-001 | 字典列表（分页查询） | ✅ 已实现 | P0 |
| DICT-002 | 新增字典 | ✅ 已实现 | P0 |
| DICT-003 | 编辑字典基本信息 | ✅ 已实现 | P0 |
| DICT-004 | 删除字典（级联删除字典详情） | ✅ 已实现 | P0 |
| DICT-005 | 启用/禁用字典 | ✅ 已实现 | P1 |
| DICT-006 | 字典详情（字典值）管理 CRUD | ✅ 已实现 | P0 |
| DICT-007 | 前端字典数据缓存 | ⚠️ 待完善 | P1 |

### 7.2 数据模型

#### 字典主表：sys_dictionaries

```sql
CREATE TABLE sys_dictionaries (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(128) NOT NULL,             -- 字典中文名（如 性别）
    type        VARCHAR(64)  NOT NULL UNIQUE,       -- 字典英文 key（如 gender）
    status      BOOLEAN      NOT NULL DEFAULT TRUE, -- 是否启用
    description TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sys_dictionaries_type ON sys_dictionaries(type);
```

#### 字典详情表：sys_dictionary_details

```sql
CREATE TABLE sys_dictionary_details (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    sys_dictionary_id   UUID        NOT NULL REFERENCES sys_dictionaries(id) ON DELETE CASCADE,
    label               VARCHAR(128) NOT NULL,  -- 展示文本（如 男）
    value               VARCHAR(128) NOT NULL,  -- 实际存储值（如 1）
    sort                INTEGER      NOT NULL DEFAULT 0,
    status              BOOLEAN      NOT NULL DEFAULT TRUE,
    extend              VARCHAR(256),           -- 扩展字段（自定义用途）
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sys_dictionary_details_dict_id ON sys_dictionary_details(sys_dictionary_id);
```

### 7.3 API 接口设计

#### 字典接口

| 接口名称 | 方法 | 路径 | 说明 |
|---------|------|------|------|
| 获取字典列表 | GET | `/api/v1/admin/dictionaries` | 分页查询字典 |
| 获取字典详情 | GET | `/api/v1/admin/dictionaries/:id` | 按 id 查询（含详情列表） |
| 创建字典 | POST | `/api/v1/admin/dictionaries` | 新增字典 |
| 更新字典 | PUT | `/api/v1/admin/dictionaries/:id` | 修改字典信息 |
| 删除字典 | DELETE | `/api/v1/admin/dictionaries/:id` | 删除字典（含所有详情） |

**按 type 查询字典（前端最常用，获取枚举选项）**:
```
GET /api/v1/admin/dictionaries?type=gender
```

响应：
```json
{
  "code": 0,
  "data": {
    "id": "uuid",
    "name": "性别",
    "type": "gender",
    "details": [
      { "label": "男", "value": "1", "sort": 1 },
      { "label": "女", "value": "2", "sort": 2 }
    ]
  }
}
```

### 7.4 页面设计

**页面布局**: 左侧字典列表 + 右侧字典详情表格（左右分栏布局）

**左侧 - 字典列表**:

| 字段 | 说明 |
|------|------|
| 字典名称 | 中文名（如 性别） |
| 字典标识 | 英文 type（如 gender） |
| 启用状态 | 启用/禁用 Badge |
| 操作 | 编辑、删除 |

**右侧 - 字典详情表格**（点击左侧字典项后显示）:

| 列名 | 字段 | 说明 |
|------|------|------|
| 展示值 | label | 显示文本 |
| 字典值 | value | 实际存储值 |
| 扩展值 | extend | 附加信息 |
| 启用状态 | status | 是/否 |
| 排序 | sort | 排序号 |
| 操作 | - | 编辑、删除 |

### 7.5 业务规则

#### 7.5.1 前端字典缓存策略

```typescript
// 全局字典缓存（基于 React Query + useRef/模块级缓存）
const dictCache = new Map<string, DictItem[]>()

export function useDictOptions(type: string) {
  return useQuery({
    queryKey: ['dict', type],
    queryFn: async () => {
      // 先从内存缓存查找
      if (dictCache.has(type)) {
        return dictCache.get(type)!
      }
      // 调用 API 获取
      const { data } = await api.getDictionaryByType(type)
      const options = data.details
      // 存入缓存（会话级别，刷新页面清除）
      dictCache.set(type, options)
      return options
    },
    staleTime: Infinity, // 字典数据不会自动重新请求
  })
}
```

#### 7.5.2 预置字典

系统预置以下字典（通过迁移 seed 数据初始化）：

| 字典名称 | 类型 key | 说明 |
|---------|---------|------|
| 性别 | gender | 男/女 |
| 用户状态 | user_status | 正常/禁用/已锁定 |
| 网络类型 | network_type | eNB/gNB/GSM/CPE/eGW |
| 运营商 | carrier | 中国移动/中国联通/中国电信 |

---

## 8. 数据权限设计（核心章节）

### 8.1 数据权限概述

OMC 系统的数据权限体系解决"不同角色的用户能看到哪些设备数据"的问题，区别于 API 权限（控制"能做什么操作"），数据权限控制"能看到哪些数据"。

**三层数据隔离**：

```
第一层：运营商隔离（carrier）
  └── 用户只能看到其所属运营商的数据
  └── 由 JWT Claims.Carrier 自动注入，无需配置

第二层：设备分组权限（role_device_groups）
  └── 角色只能看到被授权分组内的设备
  └── 在角色权限配置的"数据权限"标签页配置

第三层：网络类型筛选（network_types）
  └── 在已授权分组内，进一步按网络类型过滤
  └── 可选配置，不设置则显示分组内所有类型设备
```

### 8.2 设备分组数据模型

**设备分组树结构**（两级）：

```
L1 根分组（region level）
  ├── 华东大区
  │     ├── L2 上海（叶级分组，实际包含设备）
  │     ├── L2 江苏
  │     └── L2 浙江
  ├── 华北大区
  │     ├── L2 北京
  │     └── L2 天津
  └── 华南大区
        ├── L2 广东
        └── L2 福建
```

**device_groups 表关键字段**：

| 字段名 | 类型 | 说明 |
|--------|------|------|
| id | UUID | 分组唯一标识 |
| name | VARCHAR(128) | 分组名称 |
| parent_id | UUID | 父分组 ID（L2 级必需，L1 级为 NULL） |
| level | SMALLINT | 分组级别：1=L1根级，2=L2叶级 |
| matching_mode | VARCHAR(16) | 自动匹配规则：deviceName/lac/tac |
| name_rule_list | JSONB | 设备名称匹配规则列表 |
| lac_list | INTEGER[] | LAC 列表（用于 GSM/LTE 自动匹配） |
| tac_list | INTEGER[] | TAC 列表（用于 5G NR 自动匹配） |
| carrier | VARCHAR(4) | 所属运营商 |
| status | VARCHAR(16) | 状态：active/inactive |

### 8.3 设备网络类型定义

| 前端显示 | 前端枚举值 | 后端值 | 说明 |
|----------|-----------|--------|------|
| eNB (LTE) | `eNB` | `lte` | 4G LTE 基站 |
| gNB (5G) | `gNB` | `nr` | 5G NR 基站 |
| GSM | `GSM` | `gsm` | 2G GSM 基站 |
| CPE | `CPE` | `cpe` | 客户端设备 |
| eGW | `eGW` | `egw` | 边缘网关 |

### 8.4 role_device_groups 关联机制

**表结构**（已实现）：

```sql
CREATE TABLE role_device_groups (
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    group_id    UUID NOT NULL REFERENCES device_groups(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, group_id)
);
```

**数据权限查询应用示例**：

```sql
-- 查询指定角色可见的设备列表（自动应用数据权限过滤）
SELECT d.*
FROM devices d
JOIN device_group_members dgm ON d.id = dgm.device_id
JOIN role_device_groups rdg   ON dgm.group_id = rdg.group_id
WHERE rdg.role_id = $1        -- 当前用户角色
  AND d.carrier   = $2        -- 运营商隔离
  -- 可选：网络类型筛选
  AND ($3::text[] IS NULL OR d.tech_type = ANY($3))
ORDER BY d.created_at DESC;
```

### 8.5 网络类型维度的数据权限扩展

当前 `role_device_groups` 表只关联了角色和分组，网络类型筛选为扩展功能，可通过以下方案实现：

**方案A：扩展 role_device_groups 表（推荐）**

```sql
-- 在关联表中增加 network_types 字段
ALTER TABLE role_device_groups
ADD COLUMN network_types TEXT[] DEFAULT NULL;

-- NULL 表示不限制网络类型（可见所有类型）
-- 非 NULL 则只能看到数组内的网络类型
-- 示例：
-- ('role_uuid', 'group_uuid', ARRAY['lte', 'nr'])  -- 仅限 4G/5G
-- ('role_uuid', 'group_uuid', NULL)                 -- 不限类型
```

**方案B：独立的角色网络类型权限表**

```sql
CREATE TABLE role_network_types (
    role_id      UUID    NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    network_type VARCHAR(16) NOT NULL,  -- lte/nr/gsm/cpe/egw
    PRIMARY KEY (role_id, network_type)
);
```

### 8.6 运营商（carrier）隔离机制

运营商隔离通过以下机制自动实现：

1. **用户注册时绑定运营商**：`users.carrier` 字段记录用户所属运营商
2. **登录时注入 Claims**：JWT 生成时将 `carrier` 写入 Claims
3. **中间件自动注入**：`RequireCarrier` 中间件从 Claims 提取运营商并存入 Gin Context
4. **查询自动过滤**：所有数据查询通过 `WHERE carrier = $carrier` 实现隔离

```go
// RequireCarrier 中间件示例
func RequireCarrier() gin.HandlerFunc {
    return func(c *gin.Context) {
        claims := GetClaims(c)
        c.Set("carrier", claims.Carrier)
        c.Next()
    }
}
```

---

## 9. 核心业务流程

### 9.1 登录认证流程

```
[浏览器] POST /api/v1/auth/login {username, password}
          │
          ▼
[后端] 验证码校验（可选，系统配置控制）
  失败 → 返回 {"code": 7, "msg": "验证码错误"}
          │
          ▼
[后端] 查询用户 WHERE username = $1
  不存在 → 返回 {"code": 7, "msg": "用户名或密码错误"}
          │
          ▼
[后端] 检查 locked_until > NOW()
  已锁定 → 返回 {"code": 7, "msg": "账号已锁定，请联系管理员"}
          │
          ▼
[后端] bcrypt.CompareHashAndPassword(hash, password)
  失败 → failed_login_attempts++
         达到 max_login_attempts → 自动锁定 lockout_duration 分钟
         → 返回 {"code": 7, "msg": "用户名或密码错误"}
          │
          ▼
[后端] 检查 status = 'active'
  已禁用 → 返回 {"code": 7, "msg": "账号已禁用"}
          │
          ▼
[后端] 重置 failed_login_attempts = 0，更新 last_login_at
          │
          ▼
[后端] 生成 JWT AccessToken + RefreshToken:
  Claims: {user_id, username, role_id, carrier, exp}
          │
          ▼
[后端] 记录 sys_login_logs 登录日志
          │
          ▼
[后端] 返回 {access_token, refresh_token, expires_at, user}
          │
          ▼
[前端] 存储 token 到 localStorage
          │
          ▼
[前端] 请求 GET /api/v1/auth/menus 获取动态菜单
          │
          ▼
[前端] 动态注册 React Router 路由 + 渲染侧边栏
          │
          ▼
[前端] 跳转到默认首页（Dashboard）
```

### 9.2 权限校验流程（JWT + Casbin + 数据权限）

```
[请求] GET /api/v1/admin/users
  Headers: Authorization: Bearer eyJhbGci...
          │
          ▼
[RequireAuthWithAPIKey 中间件]
  1. 提取 Bearer Token
  2. 解析并校验 JWT 签名和有效期
  3. 将 Claims 存入 Gin Context
  失败 → 401 Unauthorized
          │
          ▼
[RequireCarrier 中间件]
  从 Claims 提取 carrier 并存入 Context
          │
          ▼
[RequirePermission 中间件]
  resource = "users", action = "read"
  调用 casbin.Enforce(roleID, carrier, "users", "read")
  false → 403 Forbidden
          │
          ▼
[AuditLogger 中间件]
  异步记录：操作人、操作类型、模块、IP、时间
          │
          ▼
[业务处理器]
  从 Context 获取 user_id、role_id、carrier
  执行查询（自动应用运营商过滤 + 数据权限过滤）
          │
          ▼
[响应] 200 OK + 过滤后的业务数据
```

### 9.3 动态菜单生成流程

```
[前端] GET /api/v1/auth/menus
          │
          ▼
[后端] 从 JWT 获取 role_id
          │
          ▼
[后端] 查询角色菜单：
  SELECT m.* FROM menus m
  JOIN role_menus rm ON m.id = rm.menu_id
  WHERE rm.role_id = $1
  AND m.status = 'active'
  ORDER BY m.sort_order ASC
          │
          ▼
[后端] 构建树形结构（parent_id = NULL 为根）
          │
          ▼
[前端] 接收菜单树 JSON
          │
          ▼
[前端] 动态注册路由（React Router v6）：
  const routes = menus.map(menu => ({
    path: menu.route_path,
    element: React.lazy(() => import(`/omcmb/webcode/src/pages/${menu.component}`)),
    handle: { title: menu.name, icon: menu.icon }
  }))
  // 通过 useRoutes 或 router.navigate 动态加载路由
          │
          ▼
[前端] 渲染侧边栏导航
```

### 9.4 角色权限配置流程

```
[管理员] 点击角色列表中的"权限配置"按钮
          │
          ▼
[前端] 打开 800px 宽权限配置抽屉，并行请求：
  ① GET /api/v1/admin/menus/tree            → 完整菜单树
  ② GET /api/v1/admin/roles/:id/menus       → 角色已授权菜单
  ③ GET /api/v1/admin/api-endpoints         → 所有 API 端点
  ④ GET /api/v1/admin/roles/:id/api-permissions → 角色已授权 API
  ⑤ GET /api/v1/admin/roles/:id/device-groups  → 角色数据权限（分组）
          │
          ▼
[抽屉展示三个标签页]
  Tab1: 菜单权限树（已授权的节点勾选）
  Tab2: API 权限表格（已授权的行勾选）
  Tab3: 数据权限（已授权的设备分组勾选 + 网络类型选择）
          │
          ▼
[管理员] 调整各权限项
          │
          ▼
[点击"保存"按钮]
          │
          ▼
[前端] 并行提交（根据有变更的标签页决定）：
  ① PUT /api/v1/admin/roles/:id/menus           → 菜单权限变更
  ② PUT /api/v1/admin/roles/:id/api-permissions → API 权限变更
  ③ PUT /api/v1/admin/roles/:id/device-groups   → 数据权限变更
          │
          ▼
[后端] 各接口事务操作：
  - 删除旧的关联记录
  - 批量插入新的关联记录
  - 同步 Casbin 策略（API权限变更时）
  - Redis Pub/Sub 广播策略刷新通知
          │
          ▼
[权限立即在所有实例生效]
```

---

## 10. 关联表设计

### 10.1 完整 ER 关系图

```
┌───────────────────┐         ┌─────────────────────┐
│      users        │  1   N  │     user_roles       │
├───────────────────┤─────────►├─────────────────────┤
│ id (UUID, PK)     │         │ user_id (FK→users)   │
│ username          │         │ role_id (FK→roles)   │
│ display_name      │         │ is_default           │
│ email             │         └──────────┬──────────┘
│ password_hash     │                    │ N
│ carrier           │                    │
│ status            │                    ▼ 1
│ failed_login_...  │        ┌───────────────────────┐
│ locked_until      │        │        roles          │
│ last_login_at     │        ├───────────────────────┤
└───────────────────┘        │ id (UUID, PK)         │
                             │ name                  │
                             │ description           │
                             │ is_system             │
                             └─────┬───────┬─────────┘
                                   │       │
              ┌────────────────────┘       └──────────────────┐
              │ N (role_menus)                   N (role_device_groups)
              ▼ N                                ▼ N
┌─────────────────────┐        ┌────────────────────────────┐
│       menus         │        │    role_device_groups      │
├─────────────────────┤        ├────────────────────────────┤
│ id (UUID, PK)       │        │ role_id (FK→roles)         │
│ name                │        │ group_id (FK→device_groups)│
│ type (dir/menu/btn) │        └────────────┬───────────────┘
│ permission_key      │                     │ N
│ parent_id (self)    │                     ▼ 1
│ sort_order          │        ┌────────────────────────────┐
│ route_path          │        │      device_groups         │
│ component           │        ├────────────────────────────┤
│ icon                │        │ id (UUID, PK)              │
│ status              │        │ name                       │
│ is_hidden           │        │ parent_id (self-ref)       │
└─────────────────────┘        │ level (1=L1, 2=L2)        │
                               │ matching_mode              │
                               │ name_rule_list (JSONB)     │
                               │ lac_list (INTEGER[])       │
                               │ tac_list (INTEGER[])       │
                               │ carrier                    │
                               └────────────────────────────┘

┌──────────────────────────┐
│     sys_dictionaries     │
├──────────────────────────┤   1:N  ┌──────────────────────────────┐
│ id (UUID, PK)            │───────►│   sys_dictionary_details     │
│ name                     │        ├──────────────────────────────┤
│ type (UNIQUE)            │        │ id (UUID, PK)                │
│ status                   │        │ sys_dictionary_id (FK)       │
│ description              │        │ label                        │
└──────────────────────────┘        │ value                        │
                                    │ sort                         │
                                    │ status                       │
                                    │ extend                       │
                                    └──────────────────────────────┘

┌──────────────────────────────────────────────────┐
│              permissions (Casbin 策略源)           │
├──────────────────────────────────────────────────┤
│ id (UUID, PK)                                    │
│ role_id (FK → roles)                            │
│ resource (VARCHAR)  -- 资源名或 API 路径          │
│ action (VARCHAR)    -- 操作或 HTTP 方法           │
└──────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────┐
│              role_inheritance                     │
├──────────────────────────────────────────────────┤
│ parent_role_id (FK → roles)                      │
│ child_role_id  (FK → roles)                      │
│ domain (VARCHAR)  -- 运营商域                     │
└──────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────┐
│              api_keys                             │
├──────────────────────────────────────────────────┤
│ id (UUID, PK)                                    │
│ user_id (FK → users)                            │
│ name                                             │
│ key_prefix                                       │
│ key_hash                                         │
│ scopes (TEXT[])                                  │
│ expires_at (TIMESTAMPTZ)                         │
└──────────────────────────────────────────────────┘
```

### 10.2 日志相关表

```sql
-- 登录日志
CREATE TABLE sys_login_logs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        REFERENCES users(id) ON DELETE SET NULL,
    username    VARCHAR(64) NOT NULL,
    ip_address  VARCHAR(64),
    user_agent  TEXT,
    status      VARCHAR(16) NOT NULL,  -- success/failed/locked
    message     TEXT,
    login_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_login_logs_user_id  ON sys_login_logs(user_id);
CREATE INDEX idx_login_logs_login_at ON sys_login_logs(login_at DESC);

-- 操作审计日志
CREATE TABLE sys_oper_logs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        REFERENCES users(id) ON DELETE SET NULL,
    username    VARCHAR(64),
    action      VARCHAR(64) NOT NULL,   -- create/update/delete/read
    module      VARCHAR(64) NOT NULL,   -- 操作模块
    target      TEXT,                   -- 操作对象（如资源ID）
    request     JSONB,                  -- 请求参数（脱敏）
    response    JSONB,                  -- 响应结果
    status      VARCHAR(16) NOT NULL,   -- success/failed
    error_msg   TEXT,
    ip_address  VARCHAR(64),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_oper_logs_user_id    ON sys_oper_logs(user_id);
CREATE INDEX idx_oper_logs_created_at ON sys_oper_logs(created_at DESC);

-- 系统配置表
CREATE TABLE sys_configs (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    category    VARCHAR(64) NOT NULL,
    key         VARCHAR(128) NOT NULL UNIQUE,
    value       TEXT        NOT NULL,
    value_type  VARCHAR(16) NOT NULL DEFAULT 'string',  -- string/int/bool/json
    description TEXT,
    is_public   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 10.3 系统配置预置值

```sql
INSERT INTO sys_configs (category, key, value, value_type, description, is_public) VALUES
  ('system',   'system_name',         'OMC 网管系统', 'string', '系统名称',           TRUE),
  ('system',   'system_version',      '1.0.0',        'string', '系统版本',           TRUE),
  ('security', 'session_timeout',     '30',           'int',    '会话超时时间（分钟）', FALSE),
  ('security', 'max_login_attempts',  '5',            'int',    '最大登录失败次数',   FALSE),
  ('security', 'lockout_duration',    '30',           'int',    '锁定时长（分钟）',   FALSE),
  ('security', 'password_min_length', '6',            'int',    '密码最短长度',       FALSE),
  ('captcha',  'captcha_enabled',     'false',        'bool',   '是否启用登录验证码', FALSE);
```

---

## 11. 待完善事项与实施建议

### 11.1 当前缺口与优先级

| 待完善项 | 当前状态 | 优先级 | 说明 |
|---------|---------|--------|------|
| API权限持久化 | 端点列表硬编码于前端 | P1 | 建议新增 api_endpoints 表，后端扫描路由后自动同步 |
| API权限验证中间件 | 基于 resource/action，未做路径级校验 | P1 | 在现有 Casbin 框架上扩展 API 路径+方法维度 |
| 数据权限网络类型维度 | role_device_groups 无网络类型字段 | P2 | 在关联表增加 network_types 字段，前端 Tab3 已支持选择 |
| 用户修改自己的密码 | 仅管理员可重置他人密码 | P2 | 增加 POST /api/v1/auth/change-password 接口 |
| 角色下用户列表 | 缺少该查询接口 | P2 | 增加 GET /api/v1/admin/roles/:id/users 接口 |
| 字典前端缓存 | 每次使用都请求后端 | P3 | 在前端使用 React Query (TanStack Query) 实现字典缓存策略 |

### 11.2 分阶段实施计划

**阶段一（已完成）：基础系统管理**
- ✅ 用户/角色/菜单/字典 CRUD
- ✅ JWT 认证 + 账户锁定
- ✅ Casbin RBAC 权限引擎（PostgreSQL 适配器 + Redis 同步）
- ✅ 数据权限（role_device_groups 表）
- ✅ 前端三标签页权限配置框架

**阶段二（建议下一阶段）：API 权限完善**

```
任务1：API 端点数据库化
  - 新增 api_endpoints 表迁移
  - 启动时自动扫描 Gin 路由并同步到表
  - 修改 GET /api/v1/admin/api-endpoints 接口从 DB 读取
  
任务2：API 路径级权限中间件
  - 实现基于 Casbin 的 API 路径+HTTP方法权限检查
  - 在需要细粒度控制的路由组挂载中间件
  - 测试验证权限规则正确性
```

**阶段三（后续优化）：数据权限增强**

```
任务3：网络类型维度支持
  - 迁移 role_device_groups 增加 network_types 字段
  - 后端数据权限查询增加网络类型过滤
  - 前端 Tab3 数据权限中网络类型选择与后端联调

任务4：用户自助密码修改
  - 新增 POST /api/v1/auth/change-password 接口
  - 前端个人中心增加密码修改入口
```

### 11.3 测试验证建议

| 测试场景 | 验证要点 |
|---------|---------|
| 角色菜单权限 | 分配菜单后用户登录只能看到授权菜单 |
| 角色 API 权限 | 未授权 API 返回 403，已授权 API 正常响应 |
| 数据权限 | 不同分组权限的角色只看到各自分组的设备 |
| 运营商隔离 | 跨运营商数据不可见 |
| 账户锁定 | 连续失败 5 次后账号锁定，30 分钟后自动解锁 |
| 系统内置角色 | admin 角色不可删除，Casbin `r.sub == "admin"` 规则直接放行 |
| JWT 过期 | Token 过期后返回 401，引导重新登录 |

---

*文档版本 v2.0，基于 OMC 项目代码库实际实现整理。*  
*下次更新：API 权限持久化功能完成后，补充第 6 章数据模型和接口规格。*
