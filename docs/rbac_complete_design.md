# OMC 用户-角色-菜单权限系统设计

> 简洁版：通过菜单统一管理权限，删除冗余的 permissions 和 menu_operation_templates 表

---

## 一、核心概念

### 1.1 权限模型

```
用户 ───多对多──> 角色 ───多对多──> 菜单
                      │
                      └──多对多──> 设备组 (数据权限)
```

### 1.2 权限通过菜单实现

- **菜单 = 权限**：每个菜单项有唯一的 `permission_key`
- **角色 = 菜单集合**：角色拥有哪些菜单，就拥有哪些权限
- **权限检查**：检查用户的菜单列表中是否包含当前请求的 `permission_key`

**示例**：
| 菜单 | permission_key | 说明 |
|------|----------------|------|
| 设备列表 | `device:list` | 页面访问权限 |
| 查询按钮 | `device:list:query` | 查询操作权限 |
| 添加按钮 | `device:list:add` | 添加操作权限 |
| 删除按钮 | `device:list:delete` | 删除操作权限 |

---

## 二、数据库表结构

### 2.1 ER 图

```
┌─────────────────┐         ┌─────────────────┐
│     users       │         │     roles       │
│ ─────────────── │         │ ─────────────── │
│ id (PK)         │         │ id (PK)         │
│ username        │         │ name            │
│ password_hash   │         │ description     │
│ email           │         │ is_system       │
│ carrier         │         │ status          │
│ status          │         │ created_by      │
│ last_login_at   │         │ created_at      │
│ created_by ◄────┼─────────┤ updated_by ◄────┼────┐
│ created_at      │         │ updated_at      │    │  │
│ updated_by ◄────┼─────────┘                 │    │  │
│ updated_at      │                                │  │  │
└─────────────────┘                                │  │  │
       │                                          │  │  │
       │ user_roles                                │  │  │
       │                                          │  │  │
       ▼                                          │  │  │
┌─────────────────┐                                │  │  │
│   user_roles    │                                │  │  │
│ ─────────────── │                                │  │  │
│ user_id (FK)    │                                │  │  │
│ role_id (FK)    │                                │  │  │
│ created_at      │                                │  │  │
└─────────────────┘                                │  │  │
       │                                          │  │  │
       │                                          │  │  │
       ▼                                          │  │  │
┌─────────────────────────────────────────────────────────────────┐
│                         role_menus                          │
│ ────────────────────────────────────────────────────────────  │
│ role_id (FK) ────────────────────────────> roles(id)          │
│ menu_id (FK) ────────────────────────────> menus(id)          │
│ created_by (FK) ────────────────────────> users(id)           │
│ created_at                                          │           │
└─────────────────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────────┐
│                         menus                               │
│ ────────────────────────────────────────────────────────────  │
│ id (PK)                                            │           │
│ name                                              │           │
│ type (directory/menu/button)                              │           │
│ permission_key (device:list:query)                       │           │
│ parent_id (FK) ─────────> menus(id) (自关联)                 │           │
│ sort_order                                         │           │
│ route_path                                        │           │
│ icon                                              │           │
│ show_status (show/hide)                                        │           │
│ status (normal/disabled)                                     │           │
│ created_by (FK) ─────────> users(id)                          │           │
│ created_at                                                      │           │
│ updated_by (FK) ─────────> users(id)                          │           │
│ updated_at                                                      │           │
└─────────────────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────────┐
│                    role_device_groups                         │
│ ────────────────────────────────────────────────────────────  │
│ role_id (FK) ─────────> roles(id)                             │           │
│ group_id (FK) ─────────> device_groups(id)                     │           │
│ created_by (FK) ─────────> users(id)                           │           │
│ created_at                                                      │           │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 表清单

| 表名 | 说明 | 核心字段 |
|------|------|----------|
| `users` | 用户表 | id, username, email, carrier, status, created_by, updated_by |
| `roles` | 角色表 | id, name, is_system, status, created_by, updated_by |
| `user_roles` | 用户-角色关联 | user_id, role_id |
| `menus` | 菜单表（=权限） | id, name, type, **permission_key**, parent_id |
| `role_menus` | 角色-菜单关联 | role_id, menu_id |
| `role_device_groups` | 角色数据权限 | role_id, group_id |

### 2.3 表结构定义

#### 2.3.1 users（用户表）

```sql
CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username        VARCHAR(64) NOT NULL UNIQUE,
    password_hash   VARCHAR(256) NOT NULL,
    display_name    VARCHAR(128),
    email           VARCHAR(256),
    carrier         VARCHAR(4),                   -- cmcc/ctcc/cucc，NULL=全部
    status          VARCHAR(16) NOT NULL DEFAULT 'active',
    last_login_at   TIMESTAMPTZ,
    
    -- 审计字段（应用层维护，不用触发器）
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_user_status CHECK (status IN ('active', 'disabled'))
);
```

#### 2.3.2 roles（角色表）

```sql
CREATE TABLE roles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(64) NOT NULL UNIQUE,
    description     TEXT,
    is_system       BOOLEAN NOT NULL DEFAULT FALSE,  -- 内置角色不可删除
    status          VARCHAR(16) NOT NULL DEFAULT 'active',
    
    -- 审计字段
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_role_status CHECK (status IN ('active', 'disabled'))
);
```

#### 2.3.3 user_roles（用户-角色关联）

```sql
CREATE TABLE user_roles (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);
```

#### 2.3.4 menus（菜单表 = 权限表）

```sql
CREATE TABLE menus (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(64) NOT NULL,
    type            VARCHAR(16) NOT NULL,           -- directory/menu/button
    permission_key  VARCHAR(128) NOT NULL,          -- device:list:query
    parent_id       UUID REFERENCES menus(id) ON DELETE CASCADE,
    sort_order      INT NOT NULL DEFAULT 0,
    
    -- 路由显示
    route_path      VARCHAR(256),
    icon            VARCHAR(64),
    show_status     VARCHAR(16) NOT NULL DEFAULT 'show',
    component_path VARCHAR(256),
    
    -- 状态
    status          VARCHAR(16) NOT NULL DEFAULT 'normal',
    
    -- 审计字段
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_menu_type CHECK (type IN ('directory', 'menu', 'button')),
    CONSTRAINT chk_menu_show CHECK (show_status IN ('show', 'hide')),
    CONSTRAINT chk_menu_status CHECK (status IN ('normal', 'disabled'))
);

-- 菜单层级结构
-- 目录 ─── 菜单 ─── 按钮
```

#### 2.3.5 role_menus（角色-菜单关联）

```sql
CREATE TABLE role_menus (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    menu_id     UUID NOT NULL REFERENCES menus(id) ON DELETE CASCADE,
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uniq_role_menu UNIQUE(role_id, menu_id)
);
```

#### 2.3.6 role_device_groups（角色数据权限）

```sql
CREATE TABLE role_device_groups (
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    group_id    UUID NOT NULL REFERENCES device_groups(id) ON DELETE CASCADE,
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, group_id)
);
```

---

## 三、权限设计

### 3.1 permission_key 格式

```
格式：{module}:{page}[:{action}]

示例：
  device               # 设备管理模块
  device:list         # 设备列表页面
  device:list:query   # 设备列表查询按钮
  device:list:add     # 设备列表添加按钮
  alarm:current       # 告警当前页面
  system:user         # 用户管理
```

### 3.2 权限检查逻辑

```go
// 检查用户是否有某个权限（permission_key）
func HasPermission(userID uuid.UUID, permissionKey string) bool {
    // 1. 获取用户的所有角色
    roles := getUserRoles(userID)
    
    // 2. 获取角色的所有菜单
    menus := getRoleMenus(roles)
    
    // 3. 检查 permission_key 是否在菜单列表中
    for _, menu := range menus {
        if menu.PermissionKey == permissionKey {
            return true
        }
        // 如果请求的是 device:list，检查是否有 device:list:*
        if strings.HasPrefix(permissionKey, menu.PermissionKey + ":") {
            return true
        }
    }
    
    return false
}
```

### 3.3 中间件使用

```go
// 菜单权限中间件
func MenuMiddleware(permissionKey string) gin.HandlerFunc {
    return func(c *gin.Context) {
        claims := GetClaims(c)
        
        if !HasPermission(claims.UserID, permissionKey) {
            c.JSON(403, gin.H{"error": "无权限"})
            c.Abort()
            return
        }
        
        c.Next()
    }
}

// 路由使用
router.GET("/api/v1/devices", 
    MenuMiddleware("device:list"),      // 页面访问
    deviceHandler.List,
)

router.POST("/api/v1/devices", 
    MenuMiddleware("device:list:add"),   // 添加操作
    deviceHandler.Create,
)
```

---

## 四、初始化菜单数据

### 4.1 三级菜单结构

```
一级：目录
   └── 二级：菜单
       └── 三级：按钮
```

### 4.2 预置菜单示例

```sql
-- 一级：设备管理
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, icon, status, created_at) VALUES
('11111111-1111-1111-1111-111111111101', '设备管理', 'directory', 'device', NULL, 1, 'DeviceOutlined', 'normal', NOW());

-- 二级：设备管理下的菜单
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, route_path, icon, status, created_at) VALUES
('11111111-1111-1111-1111-111111111102', '设备列表', 'menu', 'device:list', '11111111-1111-1111-1111-111111111101', 1, '/device/list', 'UnorderedListOutlined', 'normal', NOW()),
('11111111-1111-1111-1111-111111111103', '设备分组', 'menu', 'device:group', '11111111-1111-1111-1111-111111111101', 2, '/device/group', 'ApartmentOutlined', 'normal', NOW()),
('11111111-1111-1111-1111-111111111104', '设备注册', 'menu', 'device:register', '11111111-1111-1111-1111-111111111101', 3, '/device/register', 'PlusOutlined', 'normal', NOW());

-- 三级：设备列表的操作按钮
INSERT INTO menus (id, name, type, permission_key, parent_id, sort_order, status, created_at) VALUES
('11111111-1111-1111-1111-111111111201', '查询', 'button', 'device:list:query', '11111111-1111-1111-1111-111111111102', 1, 'normal', NOW()),
('11111111-1111-1111-1111-111111111202', '添加', 'button', 'device:list:add', '11111111-1111-1111-1111-111111111102', 2, 'normal', NOW()),
('11111111-1111-1111-1111-111111111203', '修改', 'button', 'device:list:edit', '11111111-1111-1111-1111-111111111102', 3, 'normal', NOW()),
('11111111-1111-1111-1111-111111111204', '删除', 'button', 'device:list:delete', '11111111-1111-1111-1111-111111111102', 4, 'normal', NOW()),
('11111111-1111-1111-1111-111111111205', '导出', 'button', 'device:list:export', '11111111-1111-1111-1111-111111111102', 5, 'normal', NOW());
```

---

## 五、API 设计

### 5.1 用户管理

```
POST   /api/v1/users                    创建用户
GET    /api/v1/users                    用户列表
GET    /api/v1/users/:id                 用户详情
PUT    /api/v1/users/:id                 更新用户
DELETE /api/v1/users/:id                 删除用户
POST   /api/v1/users/:id/roles           分配角色
DELETE /api/v1/users/:id/roles           移除角色
```

### 5.2 角色管理

```
POST   /api/v1/roles                    创建角色
GET    /api/v1/roles                    角色列表
GET    /api/v1/roles/:id                 角色详情
PUT    /api/v1/roles/:id                 更新角色
DELETE /api/v1/roles/:id                 删除角色
POST   /api/v1/roles/:id/menus           设置菜单权限
GET    /api/v1/roles/:id/menus           获取角色菜单
POST   /api/v1/roles/:id/device-groups   设置数据权限
```

### 5.3 菜单管理

```
GET    /api/v1/menus                    菜单列表
GET    /api/v1/menus/tree               菜单树
GET    /api/v1/menus/:id                 菜单详情
POST   /api/v1/menus                    创建菜单
PUT    /api/v1/menus/:id                 更新菜单
DELETE /api/v1/menus/:id                 删除菜单
```

### 5.4 权限查询

```
GET    /api/v1/user/menus                当前用户的菜单树
GET    /api/v1/user/permissions          当前用户的权限列表
```

---

## 六、前端类型定义

```typescript
// ==================== 用户 ====================

export interface User {
  id: string;
  username: string;
  displayName: string;
  email?: string;
  carrier?: 'cmcc' | 'ctcc' | 'cucc';
  status: 'active' | 'disabled';
  lastLoginAt?: string;
  
  // 审计
  createdBy?: string;
  createdAt: string;
  updatedBy?: string;
  updatedAt?: string;
  
  // 关联
  roles?: Role[];
  roleIds?: string[];
  
  // 扩展
  builtIn: number;  // 从 roles 推断
}

// ==================== 角色 ====================

export interface Role {
  id: string;
  name: string;
  description?: string;
  isSystem: boolean;
  status: 'active' | 'disabled';
  
  // 权限（通过菜单体现）
  menuIds?: string[];
  menus?: MenuItem[];
  
  // 数据权限
  deviceGroupIds?: string[];
  
  // 统计
  userCount: number;
  
  // 审计
  createdBy?: string;
  createdAt: string;
  updatedBy?: string;
  updatedAt?: string;
}

// ==================== 菜单 ====================

export type MenuType = 'directory' | 'menu' | 'button';

export interface MenuItem {
  id: string;
  name: string;
  type: MenuType;
  permissionKey: string;  // 权限标识
  parentId?: string | null;
  sortOrder: number;
  
  // 路由显示
  routePath?: string;
  icon?: string;
  showStatus: 'show' | 'hide';
  componentPath?: string;
  
  // 状态
  status: 'normal' | 'disabled';
  
  // 审计
  createdBy?: string;
  createdAt: string;
  updatedBy?: string;
  updatedAt?: string;
  
  // 树形
  children?: MenuItem[];
}

// ==================== 权限检查 ====================

// 检查用户是否有某个权限
export function hasPermission(user: User, permissionKey: string): boolean {
  // 精确匹配
  if (user.menus?.some(m => m.permissionKey === permissionKey)) {
    return true;
  }
  
  // 前缀匹配（如 device:list 匹配 device:list:query）
  const prefix = permissionKey + ':';
  if (user.menus?.some(m => m.permissionKey.startsWith(prefix))) {
    return true;
  }
  
  return false;
}
```

---

## 七、设计优势

| 优势 | 说明 |
|------|------|
| **简洁** | 只用菜单表统一管理权限，删除冗余的 permissions 表 |
| **直观** | 菜单即权限，管理员一眼就能看懂 |
| **易维护** | 新增权限 = 新增菜单，不需要操作两张表 |
| **前后端统一** | 前端菜单渲染和后端权限检查用同一份数据 |
| **层级清晰** | 目录 → 菜单 → 按钮，自然对应权限层级 |

---

## 八、与原设计对比

| 项目 | 原设计 | 新设计 | 说明 |
|------|--------|--------|------|
| **权限存储** | permissions 表 | menus 表 | 统一用菜单 |
| **操作模板** | menu_operation_templates | 删除 | 没必要 |
| **权限检查** | permissions.resource + action | menus.permission_key | 更直观 |
| **表数量** | 7 张表 | 6 张表 | 精简 1 张 |

**删除的表**：
- ~~`permissions`~~
- ~~`menu_operation_templates`~~

**保留的表**：
- `users` - 用户
- `roles` - 角色
- `user_roles` - 用户角色关联
- `menus` - 菜单（= 权限）
- `role_menus` - 角色菜单关联
- `role_device_groups` - 数据权限
