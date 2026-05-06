# 系统管理 — 角色管理（System / Roles）PRD

> 文档目的：作为前后端开发对齐的唯一事实源（Single Source of Truth）。
> 与 [users.md](./users.md) 配套：用户身上挂的就是**角色**，角色聚合了**菜单 / API / 设备分组 / 网络制式** 四类权限。

| 版本 | 日期 | 作者 | 备注 |
|------|------|------|------|
| 0.1  | 2026-05-06 | Backend/Frontend Team | 基于现网 `RolePermission/index.tsx` + `internal/admin/role_handler.go` 抽取 |
| 0.2  | 2026-05-06 | Backend/Frontend Team | 接管「角色 → 设备分组」数据权限的完整描述（从 [users.md](./users.md) v0.6 迁入）：新增 §1.4 数据权限模型、§2.6 `role_device_groups` 表、§11 决策记录（R 规则全集 + 决议 ① ③）|
| 0.3  | 2026-05-06 | Backend/Frontend Team | 跟随 [users.md v1.0](./users.md) 删 `users.carrier`：§1.4 / §2.6 / §11.1 R4 / §11.2 例外条件全部改述为「超管 = `source = 'builtIn'`」；移除 §3 跨租户角色"以 carrier 隔离"措辞 |
| 0.4  | 2026-05-06 | Backend/Frontend Team | §1 顶部增加超级管理员高优先级规则块：source='builtIn' 等价 Unix root，绕过菜单/API/设备分组/制式四类权限校验。把分散在 §1.4 / §11.1 / §11.2 中的"超管旁路"规则提到首屏强提示位 |
| 0.5  | 2026-05-06 | Backend/Frontend Team | 修复角色编辑面板「菜单权限」「数据权限」回显空白 + 「数据权限」实际未保存的双 bug。**前端**：编辑/查看打开时通过 `getRoleById` + `getRoleDeviceGroups` 专项端点回显（与 API 权限模式一致）；保存时通过 `setRoleDeviceGroups` 专项端点写入。**后端**：`PgRoleRepository.List/ListWithPagination/GetByID` 批量填充 `DeviceGroupIDs`，让 [users.md §11.2 决议 ① ⚠️ 标识](./users.md) 判定准确。详见 §11.6 |
| 0.6  | 2026-05-06 | Backend/Frontend Team | **§7 Backlog 全量落地**：① `role_api_permissions` DDL（P0 #1）；② `Role.UserCount/Code/CreatedBy/UpdatedBy` 字段（P0 #2 + P1 #5 + P2 #8）；③ ListWithPagination 批量派生 UserCount；④ CreateRole/UpdateRole 自动写入 created_by/updated_by；⑤ network_types 决议 **方案 B**（接受角色级简化，DDL 加注释，UI 不变）；⑥ §11.2 决议 ① 5 个 UI 触点全覆盖（角色列表 ⚠️ Tag、编辑/查看 banner、清空设备分组二次确认、用户管理分配下拉/批量 Modal 已在 v0.5 完成）；⑦ 角色克隆 `POST /admin/roles/{id}/copy`（P2 #9）。详见 §11.7 |

**关联功能域**：F06 OMC-R 核心 / RBAC

**相关文件**：

| 层 | 路径 |
|----|------|
| 前端页面 | [omcmb/webcode/src/pages/system/RolePermission/index.tsx](../../../../omcmb/webcode/src/pages/system/RolePermission/index.tsx) |
| 前端 API | [omcmb/frontend-core/src/services/api/adminApi.ts](../../../../omcmb/frontend-core/src/services/api/adminApi.ts)（`getRoles` / `createRole` / `updateRole` / ...） |
| 前端 API | [omcmb/frontend-core/src/services/api/apiPermissionApi.ts](../../../../omcmb/frontend-core/src/services/api/apiPermissionApi.ts) |
| 前端 Hook | [omcmb/frontend-core/src/hooks/api/useSystem.ts](../../../../omcmb/frontend-core/src/hooks/api/useSystem.ts) |
| 后端 Handler | [omcgo/internal/admin/role_handler.go](../../../internal/admin/role_handler.go) |
| 后端 Service | [omcgo/internal/admin/service.go](../../../internal/admin/service.go) |
| 后端 Model | [omcgo/internal/admin/model.go](../../../internal/admin/model.go#L37-L46) |
| 后端 Repo | [omcgo/internal/admin/pg_role_repository.go](../../../internal/admin/pg_role_repository.go) |
| 数据库 | [omcgo/migrations/000002_users_roles.sql](../../../migrations/000002_users_roles.sql) + [000009_sys_admin.sql](../../../migrations/000009_sys_admin.sql) + [000012_api_endpoints_and_data_perm.sql](../../../migrations/000012_api_endpoints_and_data_perm.sql) |

---

## 1. 业务背景

> **🔑 超级管理员（v1.0 起的最高优先级规则）**
>
> 当用户的 `users.source = 'builtIn'` 时，该用户为**超级管理员**，**绕过本 PRD 描述的所有角色权限校验**：
> - 看见**所有**设备分组（无视 `role_device_groups`）
> - 看见**所有**菜单（无视 `role_menus`）
> - 调用**任意** API（无视 `role_api_permissions`）
> - 看见**所有**网络制式（无视 `network_types`）
>
> 等价于 Unix root。`source` 创建时即固定，不可运行期改写，确保超管身份唯一来源是 seed。详见 [users.md §11.11](./users.md) 与本文 §11.1 R4。

角色（Role）是 RBAC 的核心聚合，对应「这个角色能做什么 + 能看哪些数据」（**对非超管用户生效**）：

| 维度 | 关联表 | 决定 |
|------|--------|------|
| **菜单权限** | `role_menus` | 用户登录后能看到哪些菜单与按钮（前端导航 + 路由守卫） |
| **API 权限** | `role_api_permissions`（待补） | 用户能调用哪些后端端点（HTTP 中间件层） |
| **设备分组** | `role_device_groups` | 用户在所有设备数据视图（设备/告警/PM/MR/拓扑）的可见域 |
| **网络制式** | `role_device_groups.network_types` | 在分组基础上**进一步**限定可见制式（LTE/NR/GSM/CPE/EGW） |

**与用户管理的边界**：
- [users.md](./users.md) 管「**用户 → 角色**」绑定关系
- 本 PRD 管「**角色 → 菜单/API/设备分组/网络制式**」绑定关系
- 数据权限派生链、`role_device_groups` 表 schema、缓存失效策略、决议 ①/③ —— v0.2 起统一在本 PRD §1.4 / §2.6 / §11 描述（[users.md](./users.md) 仅保留用户操作侧的 cache invalidation 约束）

### 1.4 数据权限模型（用户 → 角色 → 设备分组）

> v0.2 起本节作为权威描述（先前在 [users.md §1.3](./users.md) 临时承载）。

**基础规则**：

1. 设备数据（设备列表、告警、PM、MR、拓扑等）的可见范围由**设备分组**（DeviceGroup）切分。
2. **角色必须绑定一个或多个设备分组**（关联表 `role_device_groups`，详见 §2.6），才能查看那些分组下的设备数据。
3. 用户通过**继承所拥有角色的设备分组并集**得到自己的设备数据可见域。
4. 角色未绑定任何设备分组 → 该角色在设备数据视图上"看不到任何东西"（菜单/按钮权限不受影响）。

**派生链**：

```
User                              ← 用户
  └─ user_roles                   ← 多对多
       └─ Role                    ← 角色（权限的容器，本 PRD 主体）
            ├─ permissions        ← 操作权限（resource × action）
            ├─ menus              ← 菜单可见性
            └─ role_device_groups ← 数据权限：可见的设备分组（含 network_types 制式过滤）
                 └─ DeviceGroup   ← 设备分组（树形：L1 父组 / L2 子组）
                      └─ Devices  ← 该分组下的设备
```

**关键边界**：

| 场景 | 行为 |
|------|------|
| 用户 `source == 'builtIn'`（超管，v0.3 改）| 绕过 `role_device_groups` 校验，看见**所有**设备分组、所有菜单、所有 API（实现：`PermissionService.GetUserVisibleGroupIDs` + `User.IsSuperAdmin()`）|
| 用户 `source != 'builtIn'`（普通用户：admin / LDAP）| 严格走 user → roles → role_device_groups 派生 |
| 角色绑定 L1 父分组 | 自动展开为该 L1 下所有 L2 子分组（树展开由 `GroupExpander.ListChildIDs` 完成）|
| 角色 `role_device_groups.network_types` 非空 | 在分组基础上**进一步限制可见制式**（如只看 `lte`，看不到 `nr`）|
| 用户多角色 | 可见分组 = 各角色分组的**并集**（不是交集）|
| 角色未绑定任何分组 | 该角色不贡献任何设备数据可见域；用户仍可走其它角色的分组。**v0.2：UI 必须强提示，详见 §11.2 决议 ①** |
| 角色绑定的某分组被删除 | `role_device_groups` 因 `ON DELETE CASCADE` 自动失去关联。**v0.2：删除分组前 service 层必须查 `role_device_groups` 取得受影响角色集合 → 写入消息中心通知 + 写审计日志**（详见 §11.3 决议 ③）|

**性能与缓存**：

- 派生结果按用户缓存到 Redis：key `perm:visible_groups:{userID}`，TTL 5 分钟（`permission_service.go` 第 16 行）
- **失效时机**（必须主动失效，否则用户角色变更后最长 5 分钟才生效）：
  - **角色** `role_device_groups` 变更（PUT /admin/roles/{id}/device-groups）→ 失效该角色下所有用户缓存（**本 PRD 责任**）
  - 设备分组层级（L1↔L2）变更 → 失效全部缓存（影响范围大，建议谨慎操作）
  - 用户被分配 / 解绑角色 → `InvalidateUserCache(userID)`（[users.md §6.1](./users.md) 责任）

---

## 2. 实体模型

### 2.1 主表 `roles`（[migrations/000002_users_roles.sql](../../../migrations/000002_users_roles.sql)）

| 列 | 类型 | 约束 | 说明 |
|----|------|------|------|
| `id` | UUID | PK, `gen_random_uuid()` | |
| `name` | VARCHAR(64) | NOT NULL, UNIQUE | 角色名（前端历史命名 `groupName`，v0.3 起统一为 `roleName`）|
| `description` | TEXT | NULL | |
| `is_system` | BOOLEAN | NOT NULL, DEFAULT FALSE | 内置角色（如 `admin`/`operator`/`viewer`），**不可删除、不可改名** |
| `created_at` / `updated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |

### 2.2 关联表

| 关联表 | 列 | 用途 | 迁移 |
|-------|----|------|------|
| `user_roles` | `user_id`, `role_id`, `is_default` | 用户 ↔ 角色（多对多）| 000002 |
| `permissions` | `id`, `role_id`, `resource`, `action` | 角色 ↔ 资源-动作权限（保留，待规划是否迁移到 `role_api_permissions`）| 000002 |
| `role_menus` | `role_id`, `menu_id` | 角色 ↔ 菜单 | 000009 |
| `role_device_groups` | `role_id`, `group_id`, `network_types[]` | 角色 ↔ 设备分组（含制式过滤）| 000002 + 000012 增强 |
| `role_api_permissions` | `role_id`, `endpoint_id` | 角色 ↔ API 端点（**待补 DDL**） | TODO |

### 2.3 后端 Go 模型（[internal/admin/model.go:37-46](../../../internal/admin/model.go#L37-L46)）

```go
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
```

### 2.4 前端类型（[frontend-core/src/types/system.ts](../../../../omcmb/frontend-core/src/types/system.ts)）

```ts
interface Role {
  id: string;
  roleName: string;
  roleCode: string;
  batchOperation: number;
  description: string;
  permissions: string[];
  apiPermissions?: ApiPermission[];
  deviceGroupIds?: string[];
  networkTypes?: string[];
  menuIds?: string[];
  userCount: number;
  builtIn: number;        // 派生自 BackendRole.isSystem
  createUser?: string;
  updateUser?: string;
  createTime?: string;
  updateTime?: string;
}
```

### 2.5 前后端字段映射 & Gap

| 前端字段 | 后端字段 | DB 列 | 状态 |
|---------|---------|-------|------|
| `roleName` | `Name` | `roles.name` | ✅ |
| `description` | `Description` | `roles.description` | ✅ |
| `builtIn` (0/1) | `IsSystem` (bool) | `roles.is_system` | ✅（前端做 0/1 ↔ bool 映射）|
| `roleCode` | — | — | ⚠️ 前端独立字段，当前 `mapBackendRole` 用 `name` 兜底；建议后端补 `code` 列 |
| `menuIds` | — | `role_menus` | ⚠️ 通过 `GET /admin/roles/{id}/menus` 单独取，不在 Role 主响应里 |
| `deviceGroupIds` | `DeviceGroupIDs` | `role_device_groups` | ✅（`GET /admin/roles/{id}/device-groups`）|
| `networkTypes` | — | `role_device_groups.network_types` | ⚠️ 当前实现：每个分组独立一个 network_types 数组；前端却展示成"角色级"统一字段 → 语义错位 |
| `apiPermissions` | — | `role_api_permissions`（待补）| ⚠️ 通过 `GET /admin/roles/{id}/api-permissions` 取，DB 表待补 |
| `userCount` | — | derived | ⚠️ 后端 `Role` 主响应未派生该字段，前端列表展示时可能为空 |
| `createUser` / `updateUser` | — | — | ❌ 后端缺审计字段（与 users.md §2.4 同问题）|

### 2.6 数据权限关联表 `role_device_groups`

> v0.2 起本节作为权威描述（先前在 [users.md §2.5](./users.md) 临时承载）。
> 在 [`migrations/000002_users_roles.sql`](../../../migrations/000002_users_roles.sql) 创建，[`migrations/000012_api_endpoints_and_data_perm.sql`](../../../migrations/000012_api_endpoints_and_data_perm.sql) 增强了 `network_types` 字段。该表是 §1.4 数据权限模型的物理载体。

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `role_id` | UUID | NOT NULL, FK → `roles(id)` ON DELETE CASCADE | 角色 |
| `group_id` | UUID | NOT NULL, FK → `device_groups(id)` ON DELETE CASCADE | 设备分组（L1 或 L2 均可） |
| `network_types` | TEXT[] | NOT NULL DEFAULT `'{}'` | 在该分组上**进一步限定**的可见制式数组，如 `{lte}` / `{lte,nr}`；空数组 = 不限制 |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | |
| PRIMARY KEY | (`role_id`, `group_id`) | | 一个角色对一个分组只有一条记录 |

**与 `device_groups` 表的关系**（`migrations/000003_devices.sql`）：

```
device_groups            ← 树形（parent_id），L1 / L2 两层
  └─ device_group_members
       └─ devices        ← 实际设备
```

**判定逻辑**（`internal/admin/permission_service.go:50-107`）：

1. `if user.IsSuperAdmin() /* user.Source == UserSourceBuiltIn */` → 返回 nil（超管，所有分组可见）
2. 否则查 `role_device_groups` 拿到所有 `(group_id, network_types)`
3. 对每个 `group_id` 调 `groupExpander.GetGroupLevel`：
   - L1 → 调 `ListChildIDs` 展开为所有 L2 子组
   - L2 → 直接加入结果集
4. 结果去重写入 Redis（5min TTL）

**API 入口**（详见 §6.4）：

| Method | 路径 | 用途 |
|--------|------|------|
| GET | `/admin/roles/{id}/device-groups` | 查询角色当前绑定的设备分组列表 |
| PUT | `/admin/roles/{id}/device-groups` | 整体替换角色的设备分组绑定（含 `network_types`）|

---

## 3. 列表页（Table）字段定义

### 3.1 列定义

| key | 列标题 | dataIndex | 类型 | UI 渲染 | 列宽 | 备注 |
|-----|-------|-----------|------|--------|------|------|
| `actions` | 操作 | — | — | `<Button type="link">查看</Button> + <Dropdown>更多</Dropdown>` | 100, fixed: 'right' | 见 §4 |
| `roleName` | 角色名称 | `roleName` | string | `<span>` + `<Tag color="blue">内置</Tag>`（`builtIn === 1` 时） | 150 | |
| `description` | 角色描述 | `description` | string | 文本，`ellipsis` | flex | |
| `userCount` | 关联用户数 | `userCount` | number | 数字 | 100 | 前端 number 字段；空显示 0 |
| `createUser` | 创建人 | `createUser` | string | 文本 | 100 | 后端缺则显示 `-` |
| `createTime` | 创建时间 | `createTime` | datetime | `toLocaleString('zh-CN')` | 160 | |
| `updateUser` | 更新人 | `updateUser` | string | 文本 | 100 | |
| `updateTime` | 更新时间 | `updateTime` | datetime | 同上 | 160 | |

### 3.2 顶部筛选栏

| 字段 | 标签 | UI 组件 | 提交参数 |
|------|------|--------|---------|
| `roleName` | 角色名称 | `<Input>` | `?name=<keyword>` |

### 3.3 顶部右侧按钮

| 按钮 | 图标 | type | 行为 |
|------|------|------|------|
| 添加 | `PlusOutlined` | `primary` | 打开创建 Drawer |

---

## 4. 操作清单

### 4.1 单行操作（"查看" + "更多"下拉）

| 操作 | 触发 | 内置角色 | 接口 |
|------|------|---------|------|
| 查看 | "查看" | ✅ | `GET /admin/roles/{id}` + `GET /admin/roles/{id}/menus` + `GET /admin/roles/{id}/device-groups` + `GET /admin/roles/{id}/api-permissions` |
| 编辑 | 更多菜单 | ❌ disabled | `PUT /admin/roles/{id}` + `PUT /admin/roles/{id}/menus` + `PUT /admin/roles/{id}/device-groups` + `PUT /admin/roles/{id}/api-permissions` |
| 删除 | 更多菜单 | ❌ disabled | `DELETE /admin/roles/{id}` |

> 内置判定 = `builtIn === 1`（来自 `roles.is_system = true`）。

### 4.2 批量操作（多选）

| 操作 | 选中含内置 | 接口 |
|------|----------|------|
| 批量删除 | 自动跳过内置（后端单条返 409，前端聚合）| 循环 `DELETE /admin/roles/{id}` |

---

## 5. 表单字段定义（创建 / 编辑 Drawer）

> Drawer 宽度 ~720px（含菜单/分组树）。字段顺序：基本信息 → 菜单权限 → 数据权限（设备分组 + 制式）→ API 权限。

### 5.1 基本信息

| name | 标签 | 类型 | UI 组件 | 必填 | 校验 |
|------|------|------|--------|------|------|
| `roleName` | 角色名称 | string | `<Input>` | ✅ | unique；长度 2-64；编辑时只读（建议） |
| `description` | 角色描述 | string | `<Input.TextArea rows=3 maxLength=500>` | — | |

### 5.2 菜单权限

| name | 标签 | UI 组件 | 必填 | 数据源 |
|------|------|--------|------|--------|
| `menuIds` | 菜单权限 | `<Tree checkable>` | ✅ | options 来自 `GET /admin/menus/tree`；勾选父节点自动选中所有子节点（含按钮）|

### 5.3 数据权限（设备分组 + 网络制式）

| name | 标签 | UI 组件 | 必填 | 数据源 |
|------|------|--------|------|--------|
| `deviceGroupIds` | 数据权限（设备分组） | `<Tree checkable>` | ✅ | options 来自 `useAllDeviceGroups()` → `GET /devices/device-groups/tree`；至少选一个 L2 节点 |
| `networkTypes` | 网络类型 | `<Checkbox.Group>` 选项：lte/nr/gsm/cpe/egw | — | 空 = 不限制 |

> ⚠️ **当前实现错位**（§2.5 已记）：`network_types` 在 DB 是**每分组**独立的数组（`role_device_groups.network_types`），但前端 UI 是"角色级"统一字段。当前 `SetRoleDeviceGroups` 把同一个 `networkTypes` 数组写入每条 `role_device_groups` 记录。需要决议：(a) UI 改为按分组配置 (b) 接受当前简化实现并写入 DDL 注释

### 5.4 API 权限

| name | 标签 | UI 组件 | 必填 | 数据源 |
|------|------|--------|------|--------|
| `apiPermissions` | API 权限 | 按 `apiGroup` 分组的 `<Checkbox.Group>` | — | options 来自 `apiPermissionApi.listEndpoints()` → `GET /admin/api-endpoints?pageSize=999`；当前选中来自 `GET /admin/roles/{id}/api-permissions` |

---

## 6. 接口契约（API Contracts）

> Base URL：`/api/v1/admin`

### 6.1 角色 CRUD（已实现）

| Method | 路径 | 说明 | Body / Query |
|--------|------|------|---------------|
| GET | `/roles` | 分页列表 | `?page=&pageSize=&name=` |
| GET | `/roles/all` | 全量角色（下拉用，无分页） | — |
| GET | `/roles/{id}` | 单角色详情（含权限）| — |
| POST | `/roles` | 创建 | `CreateRoleRequest`（见下）|
| PUT | `/roles/{id}` | 编辑 | `UpdateRoleRequest` |
| DELETE | `/roles/{id}` | 删除 | — |

**CreateRoleRequest**（[model.go](../../../internal/admin/model.go)）：
```json
{
  "name": "运维",
  "description": "...",
  "permissions": [{"resource": "devices", "action": "read"}, ...]
}
```

**UpdateRoleRequest**：同上，全部字段可选指针。

### 6.2 角色菜单权限（已实现）

| Method | 路径 | 说明 |
|--------|------|------|
| GET | `/roles/{id}/menus` | 角色当前绑定的菜单列表 |
| PUT | `/roles/{id}/menus` | 整体替换角色的菜单绑定（Body：`{ "menu_ids": ["uuid",...] }`）|

### 6.3 角色 API 权限（已实现 handler，DDL 待确认）

| Method | 路径 | 说明 |
|--------|------|------|
| GET | `/roles/{id}/api-permissions` | 角色当前绑定的 API 端点列表 |
| PUT | `/roles/{id}/api-permissions` | 整体替换（Body：`{ "endpoint_ids": ["uuid",...] }`）|

> ⚠️ Handler 已注册（`role_handler.go`），但 `role_api_permissions` DDL 是否已迁移需验证；若未迁移，写操作会失败。

### 6.4 角色数据权限（设备分组）（已实现）

| Method | 路径 | 说明 |
|--------|------|------|
| GET | `/roles/{id}/device-groups` | 当前绑定的设备分组（含 `network_types`）|
| PUT | `/roles/{id}/device-groups` | 整体替换（Body：`{ "device_group_ids": ["uuid",...], "network_types": ["lte","nr"] }`）|

> 副作用要求（v0.4 决议）：写操作成功后必须对该角色下所有用户调 `PermissionService.InvalidateUserCache`。详见 [users.md §11.7 决议 ②](./users.md)。

### 6.5 角色下用户列表（已实现）

| Method | 路径 | 说明 |
|--------|------|------|
| GET | `/roles/{id}/users` | 该角色下的用户列表（分页） |

---

## 7. 后端补齐 Backlog

### P0
1. **`role_api_permissions` DDL 落地**：当前 handler 已注册但 DB 表不确定，需新建迁移 `000NNN_role_api_permissions.sql`：
   ```sql
   CREATE TABLE role_api_permissions (
       role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
       endpoint_id UUID NOT NULL REFERENCES api_endpoints(id) ON DELETE CASCADE,
       created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
       PRIMARY KEY (role_id, endpoint_id)
   );
   ```
2. **Role 响应派生 `userCount`**：`GET /roles` 列表每行加 `userCount` 子查询，避免前端 N+1 调用 `/roles/{id}/users` 才能拿到统计。
3. **Role 响应派生 `built_in`**：与 [users.md `source` 派生](./users.md) 类似，前端 `builtIn: 0|1` 直接来自 `is_system`。

### P1
4. **`network_types` 语义对齐**（决议方向二选一）：
   - 方案 A：UI 改为按分组配置 `network_types`（每行 group + 制式 checkbox）
   - 方案 B：接受当前"角色级统一"简化，DDL 加注释说明
5. **审计字段**：`roles` 表加 `created_by` / `updated_by` UUID（`REFERENCES users(id) ON DELETE SET NULL`）+ handler 写入操作者 ID
6. **角色未绑设备分组提示**（[users.md §11.7 决议 ①](./users.md)）：列表行 + 编辑面板 banner + 用户分配下拉 ⚠️
7. **设备分组被删除联动**（[users.md §11.7 决议 ③](./users.md)）：删除分组前先查 `role_device_groups`，写审计 + 通知 + 缓存失效

### P2
8. **`Role.code`（roleCode）字段**：当前前端独立字段无后端来源，建议补 `roles.code VARCHAR(64) UNIQUE NULL`，便于程序化引用
9. **角色克隆**：`POST /admin/roles/{id}/copy` 一键复制（包含菜单/API/分组绑定）

---

## 8. 非目标

- 角色继承 / 角色组合 — 当前不支持，每个用户独立持有平铺角色集
- 时序权限（限时角色）— 暂不支持
- 跨租户角色 — v1.0 起 `users.carrier` 已删；多 carrier 隔离能力由 `role_device_groups` 通过设备分组层提供（每个分组天然属于某 carrier），角色本身无 carrier 维度

---

## 9. 度量指标

| 指标 | 类型 | 用途 |
|------|------|------|
| `omc_role_count_total{built_in}` | Gauge | 内置/自定义角色数量 |
| `omc_role_op_total{op}` | Counter | create / update / delete / set_menus / set_device_groups / set_api_permissions |
| `omc_perm_visible_groups_cache_hit_total` | Counter | 数据权限缓存命中率（与 [users.md §1.3](./users.md) 联动）|

---

## 10. 验收清单（DoD）

后端：
- [ ] §6 全部接口 `go test` 单测覆盖（含 happy path + 内置删除返 409 + 缺权限返 403）
- [ ] `role_api_permissions` 迁移落地（P0 #1）
- [ ] `Role` 响应含 `userCount` + `built_in`（P0 #2-3）
- [ ] `SetRoleDeviceGroups` / `SetRoleMenus` / `SetRoleApiPermissions` 成功后对受影响用户**同步**调 `InvalidateUserCache`（参考 [users.md §11.7 决议 ②](./users.md)）
- [ ] 内置角色（`is_system = true`）的 PUT/DELETE 接口返回 409 Conflict

前端：
- [ ] 列表「内置」Tag、编辑/删除按钮 disabled 判定基于 `builtIn === 1`
- [ ] 数据权限 Tree 至少强制选一个 L2 节点（防"分配后用户什么都看不到"，对齐 §11.2 决议 ①）
- [ ] `roleName` 编辑模式只读（避免重命名引发的角色名引用断裂）
- [ ] §11.2 决议 ① 5 个触点全覆盖（角色列表 ⚠️、编辑面板 banner、用户管理分配下拉/批量 Modal、清空分组二次确认）
- [ ] `npm run typecheck` & `npm run lint` 通过
- [ ] Playwright E2E：创建角色 → 分配菜单/分组/API → 创建用户绑该角色 → 用户登录后菜单/数据可见域正确

---

## 11. 决策记录（数据权限规则与决议）

> v0.2 起本节作为「角色 → 设备分组」相关规则的权威记录（先前在 [users.md §11.6 / §11.7](./users.md) 临时承载，已迁回本 PRD）。
> 本节记录与 §1.4 数据权限模型相关的全部规则与决议。用户操作侧的缓存约束仍由 [users.md §10 DoD](./users.md) 承载。

### 11.1 R 规则全集（数据权限的 7 条核心规则）

| 编号 | 规则 | 内容 | 主责 PRD |
|------|------|------|---------|
| **R1** 数据权限单位 | — | 设备数据按 `device_groups` 切分；权限到分组级，不到单设备级 | 共享（本 PRD 主责）|
| **R2** 角色绑分组 | — | 角色通过 `role_device_groups` 关联表绑定一组 `(group_id, network_types)`；不绑 = 该角色无设备数据可见域 | 本 PRD |
| **R3** 用户继承 | — | 用户可见域 = 各角色 `role_device_groups` 的**并集**（多角色按并集放宽，不是交集收紧）| [users.md](./users.md) |
| **R4** 超管旁路（v0.3 改）| — | `users.source = 'builtIn'` 表示超管，绕过 R2/R3，看见**所有**分组、菜单、API（实现：`PermissionService` + `User.IsSuperAdmin()` 派生）。`source` 创建时即固定，不可运行期改写 | [users.md §11.11](./users.md#1111-v10--删除-userscarrier--超管走-source--builtin) |
| **R5** 树展开 | — | 角色绑 L1 父分组 = 自动可见所有 L2 子分组（`GroupExpander.ListChildIDs`）| 本 PRD |
| **R6** 制式过滤 | — | `role_device_groups.network_types` 在分组基础上**进一步限制**可见制式（空数组=不限）| 本 PRD |
| **R7** 缓存一致性 | — | 任何会改变可见域的操作必须**同步**调 `InvalidateUserCache(userID)`（用户管理操作详见 [users.md §10 DoD](./users.md)；角色管理 `SetRoleDeviceGroups` / `SetRoleMenus` / `SetRoleApiPermissions` 的责任在 §10 DoD）| 共享 |

### 11.2 决议 ①：角色未绑设备分组 → UI 必须强提示

**规则**：当 `role_device_groups` 中**没有任何记录**对应某角色时，认定该角色「设备数据可见域为空」，所有相关 UI 必须以警告样式显式提示用户，避免管理员误以为「分配了角色就能看到设备」。

| 触点 | 提示内容 | 提示样式 |
|------|---------|---------|
| 角色管理 — 列表行 | 在角色名旁加 ⚠️ tag「未绑定设备分组」 | `<Tag color="warning">` |
| 角色管理 — 编辑/详情 | 顶部 banner：「该角色未绑定任何设备分组，分配该角色的用户将无法查看任何设备数据」 | `<Alert type="warning" showIcon>` |
| 用户管理 — 创建/编辑表单的"角色"下拉 | 下拉 option 文本后追加 ⚠️ + 副文「(无设备权限)」 | option 自定义 render |
| 用户管理 — 批量分配角色 Modal | 同上 | option 自定义 render |
| 角色管理 — 分配设备分组（`PUT /admin/roles/{id}/device-groups`）面板 | 当 `device_group_ids = []` 提交时弹二次确认：「确定要清空设备分组绑定吗？该角色下的用户将立即失去设备数据可见权限」 | `Modal.confirm` |

**例外**：超管（用户 `source = 'builtIn'`，v0.3 改）走旁路，与角色绑定无关，不受此规则影响。

**前后端协同**：

- 后端 `Role` 响应**已含** `device_group_ids: []uuid.UUID`（`model.go` 第 55 行），前端可直接以 `role.deviceGroupIds.length === 0` 判定
- 角色列表分页接口 `GET /admin/roles` 必须确保返回 `deviceGroupIds`（即便为空数组），不能省略字段
- `useAllRoles()`（v0.3 命名）的下拉数据也必须带 `deviceGroupIds`，给前端 option render 用

**实现归属**：本规则的代码改动主要落在本 PRD 对应页面（`RolePermission/index.tsx`）；用户管理的「分配角色」相关下拉/Modal 在 [users.md DoD](./users.md) 落地 ⚠️ 提示。

### 11.3 决议 ②：缓存失效不引入 EventBus（跨 PRD 共识，用户操作侧详见 users.md §11.7）

**规则**：任何会改变 `perm:visible_groups` 缓存的写操作，handler 调 service 完成业务后，**同步**调用 `PermissionService.InvalidateUserCache(userID)`，**不引入** EventBus / `user.role.changed` 等异步事件。

**本 PRD 责任范围**（角色管理写操作）：

- `PUT /admin/roles/{id}/device-groups` → 失效该角色下所有用户缓存（需先 `roleRepo.ListUserIDsByRole(roleID)`）
- `PUT /admin/roles/{id}/menus` → 影响菜单可见性而非数据可见域，但仍需失效（菜单缓存与权限缓存合并管理）
- `PUT /admin/roles/{id}/api-permissions` → 同上
- `DELETE /admin/roles/{id}` → 失效该角色下所有用户缓存
- `POST /admin/users/{id}/roles` / `DELETE /admin/users/{id}/roles/{roleId}` / `PUT /admin/users/{id}` 含 role_ids → 由 [users.md §11.7 决议 ②](./users.md) 承载

**理由与回退条件**：完整对比见 [users.md §11.7 决议 ②](./users.md#决议-缓存失效不引入-eventbushandler-同步直调)。

### 11.4 决议 ③：设备分组被删除 → 角色侧必须感知并提示

**问题**：当前 `role_device_groups.group_id REFERENCES device_groups(id) ON DELETE CASCADE`。删除设备分组时，关联的 `role_device_groups` 行被无声移除，原绑定该分组的角色：

- 不会主动收到通知
- 角色管理 UI 看不出"曾经绑过现在没了"
- 该角色下用户的可见域被静默缩小，可能导致投诉

**规则**：

1. **保留 CASCADE 不变**（避免引入悬空外键），但删除分组的 service 层在执行 `DELETE FROM device_groups WHERE id=$1` **之前**：
   - 查 `role_device_groups` 取得 `(role_id, group_id, network_types)` 列表
   - 写 `audit_logs`：`action=role_device_group_revoked_by_group_delete`，`details={"group_id":...,"group_name":...,"affected_roles":[{role_id, role_name},...]}`
   - 给每个受影响角色的"角色管理员"推送消息（消息中心 / 邮件，按 `notification` 模块路由）
2. **角色管理 UI 必须能展示这类历史**：
   - 角色编辑页加 tab「最近变更」或「告警」，列出过去 30 天内被删除的曾绑分组：「`分组名 X` 已于 `2026-05-06 12:00:00` 被删除，由 `<删除人>` 操作」
   - 若该角色目前 `role_device_groups` 为空（且历史曾有绑定），banner 同时叠加 §11.2 决议 ① 的"未绑定"警告
3. **缓存联动**：删除分组的 service 对每个受影响角色下的所有用户调 `InvalidateUserCache(userID)`（同步直调，不走事件，与 §11.3 决议 ② 一致）

**实现归属**：

- 「删除设备分组前查 + 审计 + 通知」的代码落在**设备分组管理**（F06 拓扑管理 PRD）
- 「角色编辑页展示最近变更 tab」落在**本 PRD**（RolePermission/index.tsx 增加 tab）
- 本 PRD §10 DoD 已加交叉引用

**审计字段约定**（保证多模块查询一致）：

```json
{
  "action": "role_device_group_revoked_by_group_delete",
  "resource": "device_group",
  "resource_id": "<group_id>",
  "details": {
    "group_name": "华东运维分组",
    "affected_roles": [
      { "role_id": "uuid-1", "role_name": "运维角色" },
      { "role_id": "uuid-2", "role_name": "审计角色" }
    ],
    "deleted_at": "2026-05-06T12:00:00Z"
  }
}
```

### 11.5 待澄清事项（v0.3+ 候选）

- [ ] 多个 LDAP 数据源场景下，是否需要按 LDAP 来源进一步细分角色可见域？（与 [users.md §11.4](./users.md) 联动）
- [ ] `network_types` UI 与 DDL 语义错位（§7 P1 #4）：决议方向（A：UI 改为按分组配置 / B：接受当前简化）应在 v0.3 落定
- [ ] 角色克隆（§7 P2 #9）是否复制 `role_device_groups` 绑定？默认应复制；如不希望可加 `copy_device_groups: bool` 参数

### 11.6 v0.5 — 角色编辑面板回显与保存 bug 修复

**问题**：用户报告 `/system/roles` 编辑角色时「菜单权限」与「数据权限」勾选树空白，但「API 权限」回显正常。

**根因**（保存 + 回显**双 bug**，不止回显）：

| 维度 | 保存 | 回显 |
|------|------|------|
| **菜单权限**（`roles.permissions` 三元组） | ✅ 正常（`service.CreateRole/UpdateRole` 处理）| ❌ 列表接口 `ListWithPagination` 不填充 `Permissions`，前端 `role.permissions \|\| []` 永远空 |
| **数据权限**（`role_device_groups`） | ❌ **从未保存** — 前端送了 `deviceGroupIds`，但后端 `CreateRole/UpdateRole` 静默丢弃，DB 实际为空 | ❌ 列表接口与 GetRole 都不填充 `DeviceGroupIDs` |
| **API 权限**（`role_api_permissions`） | ✅ 正常（前端专门调 `setRoleApiPermissions`） | ✅ 正常（前端专门调 `getRoleApiPermissions`） |

→ **API 权限正常**是因为前端编辑面板**绕开 CreateRole/UpdateRole**，专门调用专项端点保存与回显；菜单/数据权限没有这层绕行。

**修复方案**（混合方案 A+B）：

#### 前端修改（[`webcode/src/pages/system/RolePermission/index.tsx`](../../../../omcmb/webcode/src/pages/system/RolePermission/index.tsx)）

| 改动 | 内容 |
|------|------|
| 新增 `loadRoleDetailToForm(role)` helper | 统一三个专项端点回显：`getRoleById` 拿 permissions、`getRoleDeviceGroups` 拿 `{deviceGroupIds, networkTypes}`、`getRoleApiPermissions` 拿 endpoint IDs |
| 编辑/查看按钮 onClick | 改为单行 `loadRoleDetailToForm(role)` 调用 |
| 创建 onSuccess | 增加 `setRoleDeviceGroups.mutate({roleId: newRole.id, deviceGroupIds, networkTypes})` |
| 编辑 onSuccess | 同上 |

#### 后端修改（[`pg_role_repository.go`](../../../internal/admin/pg_role_repository.go)）

| 改动 | 内容 |
|------|------|
| 新增私有方法 `populateDeviceGroupIDs(ctx, roles []Role)` | 一次性 `SELECT role_id, group_id FROM role_device_groups WHERE role_id = ANY($1)`，按 role_id 分组填回 `Role.DeviceGroupIDs`。仅取 group_id（不取 network_types，避免 list 响应膨胀；network_types 仍由专项端点返回）|
| `List()` / `ListWithPagination()` / `GetByID()` 末尾调用该 helper | 让列表 / 单角色详情都带 `device_group_ids`，让 [users.md §11.2 决议 ① "未绑分组" ⚠️](./users.md) 标识判定准确 |

**保留的设计约定**：

- 后端 `service.CreateRole/UpdateRole` **依然不处理** `device_group_ids` / `network_types` —— 这是**故意**的，与现有 `role_api_permissions` 的设计一致：所有「绑定关系」走专项端点（`PUT /admin/roles/{id}/{menus|api-permissions|device-groups}`），主请求只管 role 自身字段。前端送 `deviceGroupIds / networkTypes` 给主请求是兼容性传递，后端忽略不报错
- `Role.NetworkTypes` Go 字段**不新增** —— 仍由专项端点 `GetRoleDeviceGroups` 返回，避免列表响应携带过多分组级数据

**验证**：

- 后端 `go test -count=1 ./internal/admin/...` 全过
- 前端 `npm run typecheck` 通过
- 实际编辑流程（手测）：
  - [ ] 创建角色，勾选数据权限 + 菜单权限，保存 → DB 中 `role_device_groups` 写入；`permissions` 写入
  - [ ] 重新打开同角色编辑面板 → 数据权限 / 菜单权限 / API 权限三个勾选树**全部正确回显**
  - [ ] 用户管理「分配角色」下拉对没绑分组的角色显示 ⚠️ 标识不再误报

**不在本次范围**：

- `service.CreateRole/UpdateRole` 直接处理 device_group_ids（语义重复，且与 API 权限模式不一致）
- 列表接口附带 `network_types` 字段（性能 / 响应大小考虑）
- `Role.NetworkTypes` Go 字段（避免双源）

### 11.7 v0.6 — §7 Backlog 全量落地

本节记录 v0.6 一次性把 §7 P0/P1/P2 全部 9 项落地的实施细节、决议方向、与新增/修改的代码点。

#### 实施清单（按 §7 编号映射）

| §7 项 | 状态 | 落地文件 |
|-------|------|---------|
| **P0 #1** `role_api_permissions` DDL | ✅ | [migrations/000056_roles_v1_extras.sql](../../../migrations/000056_roles_v1_extras.sql) |
| **P0 #2** Role.UserCount 派生 | ✅ | `pg_role_repository.go` 新增 `populateUserCounts(ctx, roles)` 一次性聚合 `SELECT role_id, COUNT(*) FROM user_roles GROUP BY role_id`；`List` / `ListWithPagination` / `GetByID` 末尾调用 |
| **P0 #3** Role.built_in 派生 | ✅（已早实现）| 前端 `mapBackendRole` 直接 `builtIn: br.is_system ? 1 : 0` |
| **P1 #4** network_types 语义对齐 | ✅ 决议 **方案 B** | 落定**接受当前角色级简化**，DDL 加 `COMMENT ON COLUMN role_device_groups.network_types`；UI 不动；如未来需要"按分组配置"再走 UI 升级（DDL 已支持）|
| **P1 #5** 审计字段 created_by/updated_by | ✅ | `roles` 表加 `created_by` / `updated_by` UUID FK→`users(id) ON DELETE SET NULL`；`service.CreateRole/UpdateRole` 自动从 ctx 取 operator 写入 |
| **P1 #6** ⚠️ 未绑分组 5 触点 | ✅ 全覆盖 | 用户管理 2 触点（v0.5 完成）+ 角色管理 3 触点（v0.6）：列表行 ⚠️ Tag、编辑 banner、查看 banner、清空二次确认 |
| **P1 #7** 设备分组删除联动 | ⏳ 不在本 PRD | 跨模块（F06 拓扑管理 PRD）；本 PRD §11.4 仅声明边界 |
| **P2 #8** Role.code 字段 | ✅ | `roles.code VARCHAR(64) NULL`，部分唯一索引；`mapBackendRole` 优先用 `br.code`，未配时用 `name` 兜底 |
| **P2 #9** 角色克隆 | ✅ | `POST /admin/roles/{id}/copy`：副本 `name = base_copy / base_copy_2 ...`，复制 permissions / role_menus / role_device_groups（含 network_types）/ role_api_permissions；`is_system` 强制 false |

#### 后端代码点（5 个文件）

| 文件 | 改动 |
|------|------|
| [migrations/000056_roles_v1_extras.sql](../../../migrations/000056_roles_v1_extras.sql) | 新建 `role_api_permissions` 表 + 索引；`roles` 加 `code` (含部分唯一索引) / `created_by` / `updated_by`；DDL 注释统一 |
| [internal/admin/model.go](../../../internal/admin/model.go) | `Role` 加 `Code` / `UserCount` / `CreatedBy` / `UpdatedBy`；`CreateRoleRequest` / `UpdateRoleRequest` 加 `Code` 字段 |
| [internal/admin/pg_role_repository.go](../../../internal/admin/pg_role_repository.go) | 新增 `roleColumns` 常量 + `applyRoleNullables` helper；`Create/GetByID/GetByName/Update` 全部读写新字段；新增私有 `populateUserCounts(ctx, roles)`；`List/ListWithPagination` 末尾调用 |
| [internal/admin/service.go](../../../internal/admin/service.go) | `CreateRole`/`UpdateRole` 自动写入 created_by/updated_by；新增 `CopyRole(ctx, sourceID)` + `allocateCopyRoleName` |
| [internal/admin/role_handler.go](../../../internal/admin/role_handler.go) + [handler.go](../../../internal/admin/handler.go) | 新增 `CopyRole` HTTP handler；路由注册 `roles.POST("/:id/copy", h.CopyRole)` |

#### 前端代码点（2 个文件）

| 文件 | 改动 |
|------|------|
| [adminApi.ts](../../../../omcmb/frontend-core/src/services/api/adminApi.ts) | `BackendRole` 加 `code?: string`；`mapBackendRole` 优先用 `br.code`；新增 `adminApi.copyRole(id)` |
| [RolePermission/index.tsx](../../../../omcmb/webcode/src/pages/system/RolePermission/index.tsx) | 引入 `Alert` / `CopyOutlined`；roleName 列加 ⚠️ 未绑分组 Tag；操作菜单加「复制」；编辑/查看 Drawer 顶部加 `<Alert>` banner；handleEdit 清空设备分组改 `modal.confirm` 二次确认（`doEditSubmit` 抽出）；新增 `copyRoleMut` + `handleCopy` |

#### network_types 方案 B 详解

| 维度 | 内容 |
|------|------|
| DDL | 不变 — `role_device_groups.network_types TEXT[]`（每分组独立数组）|
| UI | 不变 — 角色编辑面板 `<Checkbox.Group>` 是"角色级"统一字段 |
| 写入 | `SetRoleDeviceGroups` 接收单个 `network_types[]`，写入时给该角色下**每条** `role_device_groups` 行赋同样数组 |
| 读取 | `GetRoleDeviceGroups` 返回任意一行的 `network_types`（约定所有行一致）|
| 升级路径 | 如未来要做"每分组独立配置"，UI 改造为按行配置；DDL 已支持，无需迁移 |
| DDL 注释 | `COMMENT ON COLUMN role_device_groups.network_types IS 'v0.6 决议 B：DB 是每分组独立数组；当前前端 UI 简化为角色级统一字段（同步 SetRoleDeviceGroups 时所有行写入相同值）';` |

#### v0.6 触点 5 二次确认行为变化

**之前（v0.5 及以前）**：

```ts
if (selectedSecondLevel.length === 0) {
  message.warning('至少选一个二级节点');
  return; // 硬阻止保存
}
```

→ 用户**无法显式清空**设备分组绑定。

**现在（v0.6）**：

```ts
if (selectedSecondLevel.length === 0) {
  modal.confirm({
    title: '确认清空设备分组绑定',
    content: '该角色下的用户将立即失去设备数据可见权限。是否继续？',
    okButtonProps: { danger: true },
    onOk: () => doEditSubmit(),
  });
  return;
}
doEditSubmit();
```

→ 允许显式清空，但需用户**确认**，避免误操作；同时编辑面板 banner（触点 2）持续可见警告。

#### 验证

| 检查 | 结果 |
|------|------|
| `CGO_ENABLED=0 go build ./...` | ✅ |
| `go test -count=1 ./internal/admin/...` | ✅ |
| `bash scripts/check-migrations.sh` | ✅ 编号连续，Up/Down 完整 |
| `npm run typecheck`（webcode） | ✅ |
| 手测列表 ⚠️ Tag | 仅当 `deviceGroupIds.length === 0 && !isBuiltIn` 时显示，避免内置 admin 误标 |
| 手测编辑/查看 banner | 仅当 `selectedDeviceGroupIds.length === 0` 时显示（与列表 ⚠️ 联动）|
| 手测清空二次确认 | 选 0 个二级节点 → 弹 Modal；点"继续保存"成功；点"取消"不保存 |
| 手测复制角色 | 列表行「更多 → 复制」→ Modal 确认 → 副本生成（`<原名>_copy`）；权限/菜单/分组/API 全部继承 |
| 手测 audit | 创建/编辑角色后，DB 中 `roles.created_by` / `updated_by` 写入操作者 UUID |

#### 待澄清事项（v0.7+ 候选）

- [ ] `Role.code` 在 UI 上是否显式暴露给管理员配置？当前后端字段就位，前端表单**未加输入框**（避免暴露给非技术用户）；如某场景需配置（如审计/SDK 集成），加 `<Input>` 即可
- [ ] CopyRole 是否要求源角色 `is_system = false`（即"内置角色不可被复制"）？当前**允许**复制内置角色（副本仍为非内置），实际业务可能希望禁止
- [ ] CopyRole 副本的 `code` 字段如何处理？当前 service **不复制** `Code`（避免唯一约束冲突）；如未来 `code` 加自增后缀策略再调整
