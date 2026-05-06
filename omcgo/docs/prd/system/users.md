# 系统管理 — 用户管理（System / Users）PRD

> 文档目的：作为前后端开发对齐的唯一事实源（Single Source of Truth）。
> 涵盖：实体模型、字段定义、UI 组件、校验规则、操作清单、接口契约、当前前后端差异（gap）与对齐建议。

| 版本 | 日期 | 作者 | 备注 |
|------|------|------|------|
| 0.1  | 2026-05-06 | Backend/Frontend Team | 基于现网 `UserManagement/index.tsx` + `internal/admin/` 现状抽取，标注 gap |
| 0.2  | 2026-05-06 | Backend/Frontend Team | `users.source` 升为显式枚举（`builtIn`/`admin`/`LDAP`）；重写「内置用户」判定规则与操作权限矩阵；详见 §11 决策记录 |
| 0.3  | 2026-05-06 | Backend/Frontend Team | 术语统一：用户身上挂的是**角色**（Role）；废弃「用户组 / Group」语义；前端字段 `groupNames` / `targetGroupId` 与后端 `/admin/groups`、`/admin/users/move-group` 列入重命名计划；详见 §11.5 |
| 0.4  | 2026-05-06 | Backend/Frontend Team | 显式声明数据权限模型：**用户 → 角色 → 设备分组**；新增 §1.3 与 §2.5 描述 `role_device_groups` 关联表与超管旁路；详见 §11.6 |
| 0.5  | 2026-05-06 | Backend/Frontend Team | 数据权限三项决议：① 角色未绑分组时 UI 强提示；② 缓存失效**不引入 EventBus**，handler 同步调 `InvalidateUserCache`；③ 设备分组被删除时，角色侧需提示「分组已删除」；详见 §11.7 |
| 0.6  | 2026-05-06 | Backend/Frontend Team | 「角色 → 设备分组」相关描述全部迁出：详见 [roles.md §1.4 / §2.6 / §11](./roles.md)。本 PRD 仅保留用户操作侧的 cache invalidation 约束（§6.1 / §10 DoD）+ R3/R4 用户视角规则 + 决议 ② 用户操作侧落点 |
| 0.7  | 2026-05-06 | Backend/Frontend Team | 列表页"用户名称"列**不再标记**内置（取消 `<Tag color="blue">内置</Tag>`），内置/管理员/LDAP 三种来源统一由独立的"来源"列（§3.1 第 8 列）承担，避免双源信息冗余；操作菜单按钮的置灰 + Tooltip 提示规则不变。详见 §11.8 |
| 0.8  | 2026-05-06 | Backend/Frontend Team | 列表页"创建人/更新人"列：值为空（`createdBy / updatedBy IS NULL`）时显示**"内置"**而非 `-`。语义：seed 写入的内置用户与 LDAP 同步任务都没有 operator UUID，统一以"内置"兜底；非空时仍走 `useAllUsers` 做 id→username 映射。详见 §11.9 |
| 0.9  | 2026-05-06 | Backend/Frontend Team | 下拉列表必须显示**业务名字**（而非 UUID）：`<Select>` `value` 用 UUID（提交后端用），`label` 用 `roleName/displayName/...`。Options 必须从对应资源的「全量」端点（如 `GET /admin/roles/all`）拿，不能复用分页端点。修复 `adminApi.getAllRoles` 误调 `/admin/roles`（分页）→ 改为 `/admin/roles/all`，否则编辑用户时角色下拉显示 UUID。详见 §11.10 |
| 0.9  | 2026-05-06 | Backend/Frontend Team | 完整 **LDAP 集成方案** 落定：登录 bind 验证、周期+手动同步、字段映射、group→role 映射、异常处理、3 类新接口、2 张新表（`ldap_group_role_mappings` / `ldap_sync_logs`）。详见 §12；与 [system-config.md `ldap` Tab](./system-config.md) 共享配置项 |
| 1.0  | 2026-05-06 | Backend/Frontend Team | **删除 `users.carrier` 字段**：超管判定改用 `source = 'builtIn'`；JWT 新增 `IsSuperAdmin` claim；Casbin 去 domain；`PermissionService.GetUserVisibleGroupIDs` 签名改为传 `isSuperAdmin bool`；UI 多租户 carrier（设备/告警维度）不动。LDAP `sync_carrier` 配置项作废。详见 §11.11 |
| 1.1  | 2026-05-06 | Backend/Frontend Team | UI 文案与表单同步：① 新增/编辑/查看/列表四处的字段标签统一为 `username → 用户账号`、`displayName → 用户昵称`，列表第 2 列 title 同步改"用户昵称"；② **取消"导入用户"功能前端入口**（Radio 模式切换 + ImportPanel 集成全部移除），新增 Drawer 仅保留"添加用户"表单，字段顺序与编辑 Drawer 完全对齐；后端 `POST /admin/users/import` + `GET /admin/users/import/template` 端点能力**保留**（暂作内部工具，不暴露 UI）。详见 §11.12 |

**关联功能域**：F06 OMC-R 核心 / RBAC

**相关文件**：

| 层 | 路径 |
|----|------|
| 前端页面 | [omcmb/webcode/src/pages/system/UserManagement/index.tsx](../../../../omcmb/webcode/src/pages/system/UserManagement/index.tsx) |
| 前端 API | [omcmb/frontend-core/src/services/api/adminApi.ts](../../../../omcmb/frontend-core/src/services/api/adminApi.ts) |
| 前端 Hook | [omcmb/frontend-core/src/hooks/api/useSystem.ts](../../../../omcmb/frontend-core/src/hooks/api/useSystem.ts) |
| 前端 Types | [omcmb/frontend-core/src/types/system.ts](../../../../omcmb/frontend-core/src/types/system.ts) |
| 后端 Handler | [omcgo/internal/admin/user_handler.go](../../../internal/admin/user_handler.go) |
| 后端 Service | [omcgo/internal/admin/service.go](../../../internal/admin/service.go) |
| 后端 Model | [omcgo/internal/admin/model.go](../../../internal/admin/model.go) |
| 后端 Repo | [omcgo/internal/admin/pg_user_repository.go](../../../internal/admin/pg_user_repository.go) |
| 路由 | [omcgo/cmd/app/provider/router.go](../../../cmd/app/provider/router.go) |
| 数据库 | [omcgo/migrations/000002_users_roles.sql](../../../migrations/000002_users_roles.sql) |

---

## 1. 业务背景与目标

OMC 是面向运营商的商用网管系统，用户管理是 RBAC 的入口：
- **谁能登录** —— 管理本地账号、外部 LDAP 同步账号
- **能做什么** —— 通过角色（Role / Group）授予权限
- **能管哪些设备** —— 通过**角色绑定的设备分组（DeviceGroup）** 限定数据可见域；详见 §1.3
- **何时可用** —— 账号有效期、启用 / 禁用、强制下线、登录失败锁定
- **可被审计** —— 所有变更走 `audit_logs`

### 1.1 用户类型（由 `users.source` 唯一标识）

> v0.2 起：「内置」不再从角色或用户名派生，而是 `users.source` 列的显式取值。

| 类型 | `users.source` | 创建路径 | 关键约束 |
|------|----------------|---------|---------|
| **内置用户** | `builtIn` | 由 `migrations/seed/` 在系统初始化时写入（如 `admin`） | **不可删除、不可禁用、不可改用户名**；其他字段（`displayName` / `email` / `phone` / `password` / `roles` / `description`）可改 |
| **管理员添加** | `admin` | 由有权限的管理员通过本页面 `POST /admin/users` 创建 | 可执行全部用户操作 |
| **LDAP 用户** | `LDAP` | 由 LDAP 同步任务（v0.9 起由 worker 进程的 cron 触发；详见 §12.4）+ 手动同步 / 即时（JIT）首登创建 | **登录走 LDAP bind 验证**，不查本地 `password_hash`；**禁止"重置密码"**（密码归属外部域）；可禁用、可删除（仅删本地映射，不影响 LDAP 源）。完整方案见 §12 |

⚠️ **大小写约定**：枚举字面值大小写混杂（camelCase / lower / UPPER），由 DB CHECK 约束锁定。前后端 / 文档 / SQL 中必须严格保持字面值。如未来需统一，走 P3 数据迁移。

### 1.2 验收口径

- 创建用户（`source=admin`）→ 用该账号登录成功 → 在列表看到 `lastLoginTime`
- 给某用户分配"运维角色" → 登录后只看到运维菜单与受限设备域
- 禁用用户 → 该用户已签发的 token 强制失效（force-logout 联动）
- LDAP 同步任务写入用户后，前端列表「来源」列展示「LDAP」，且行内"重置密码"按钮置灰
- 内置用户（如 `admin`）在 UI 上 **删除 + 禁用** 按钮置灰；**改密、改邮箱、改角色、强制下线** 仍可用

### 1.3 数据权限模型（用户视角）

> v0.6 起本节仅承载**用户视角**的简要说明。完整模型（派生链、`role_device_groups` 表、L1/L2 树展开、network_types 过滤、决议 ① ③ 等）参见 [roles.md §1.4 / §2.6 / §11](./roles.md)。

**用户视角的两条规则**（详细规则见 roles.md §11.1 R 全集）：

- **R3 用户继承**：用户的设备数据可见域 = 其所拥有角色的 `role_device_groups` 绑定**并集**（多角色按并集放宽）。
- **R4 超管旁路**（v1.0 改）：`users.source = 'builtIn'` 表示超管，绕过角色绑定检查，看见所有设备分组、所有菜单、所有 API。`source` 是创建时即固定的不可变属性，确保超管身份不可被运行期改写。

**用户管理操作触发的缓存失效要求**：当用户被分配 / 解绑角色（影响可见角色集合）时，service 层必须同步调 `PermissionService.InvalidateUserCache(userID)`（详见 §6.1 副作用注解 + §10 DoD）。其它影响可见域的写操作（角色 → 设备分组绑定变更、设备分组层级变更）由 [roles.md §11.3](./roles.md) 与 F06 拓扑管理 PRD 承担。

---

## 2. 实体模型（Domain Model）

### 2.1 数据库表（`migrations/000002_users_roles.sql`）

#### `users`
| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | UUID | PK，`gen_random_uuid()` | 主键 |
| `username` | VARCHAR(64) | NOT NULL, UNIQUE | 登录名 |
| `password_hash` | VARCHAR(256) | NOT NULL | bcrypt 哈希，**不下发到前端**（json:"-"） |
| `display_name` | VARCHAR(128) | NULL | 展示名 |
| `email` | VARCHAR(256) | NULL | 邮箱 |
| ~~`carrier`~~ | ~~VARCHAR(4)~~ | ~~NULL~~ | ~~运营商：`cmcc` / `ctcc` / `cucc`~~ **v1.0 删除**（迁移文件 `000NNN_drop_users_carrier.sql`，超管判定改用 `source = 'builtIn'`，详见 §11.11）|
| `status` | VARCHAR(16) | NOT NULL, DEFAULT 'active' | `active` / `disabled` |
| `source` | VARCHAR(16) | NOT NULL, DEFAULT 'admin', `CHECK (source IN ('builtIn','admin','LDAP'))` | 用户来源（v0.2 新增）：`builtIn` 系统内置（不可删/不可禁用） / `admin` 管理员添加 / `LDAP` LDAP 同步 |
| `last_login_at` | TIMESTAMPTZ | NULL | 最后登录时间 |
| `failed_login_attempts` | INT | NOT NULL, DEFAULT 0 | 连续失败次数 |
| `locked_until` | TIMESTAMPTZ | NULL | 锁定到期时间（超过 N 次失败触发） |
| `last_failed_login_at` | TIMESTAMPTZ | NULL | 最近一次失败时间 |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | 创建时间 |
| `updated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() | 更新时间（trigger） |

索引：`idx_users_username(username)`、`idx_users_status(status)`、`idx_users_source(source)`（v0.2 新增，用于按来源筛选与防误删校验）。~~`idx_users_carrier`~~ v1.0 随 carrier 列一并删除。

**约束建议**（v0.2）：

- `CHECK (source IN ('builtIn','admin','LDAP'))` — 字面值大小写敏感
- 删除/禁用前在 service 层校验 `source != 'builtIn'`，违反返回 `409 Conflict`
- seed 文件创建 admin 用户时显式写 `source = 'builtIn'`
- LDAP 同步任务批量 upsert 时显式写 `source = 'LDAP'`

#### `roles`
| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| `id` | UUID | PK | |
| `name` | VARCHAR(64) | NOT NULL, UNIQUE | 角色名（前端历史命名 `groupName`，v0.3 起统一为 `roleName`） |
| `description` | TEXT | NULL | |
| `is_system` | BOOLEAN | NOT NULL, DEFAULT FALSE | 是否系统内置（前端称 `builtIn`） |
| `created_at` / `updated_at` | TIMESTAMPTZ | | |

#### `user_roles`（关联表）
| 列名 | 类型 | 约束 |
|------|------|------|
| `user_id` | UUID | FK → users(id) ON DELETE CASCADE |
| `role_id` | UUID | FK → roles(id) ON DELETE CASCADE |
| `is_default` | BOOLEAN | NOT NULL, DEFAULT FALSE，每用户至多一个默认角色（部分唯一索引） |
| `created_at` | TIMESTAMPTZ | |
| PK | (user_id, role_id) | |

#### `permissions`
| 列名 | 类型 | 约束 |
|------|------|------|
| `id` | UUID | PK |
| `role_id` | UUID | FK → roles(id) |
| `resource` | VARCHAR(64) | NOT NULL |
| `action` | VARCHAR(16) | NOT NULL |
| UNIQUE | (role_id, resource, action) | |

### 2.2 后端 Go 模型（`internal/admin/model.go`）

```go
type User struct {
    ID                  uuid.UUID          `json:"id"`
    Username            string             `json:"username"`
    PasswordHash        string             `json:"-"`
    DisplayName         string             `json:"display_name"`
    Email               string             `json:"email,omitempty"`
    // Carrier *model.CarrierCode `json:"carrier,omitempty"` — v1.0 删除，超管判定走 source='builtIn'
    Status              UserStatus         `json:"status"`         // "active" | "disabled"
    Roles               []Role             `json:"roles,omitempty"`
    FailedLoginAttempts int                `json:"failed_login_attempts"`
    LockedUntil         *time.Time         `json:"locked_until,omitempty"`
    LastFailedLoginAt   *time.Time         `json:"last_failed_login_at,omitempty"`
    LastLoginAt         *time.Time         `json:"last_login_at,omitempty"`
    CreatedAt           time.Time          `json:"created_at"`
    UpdatedAt           time.Time          `json:"updated_at"`
}
```

### 2.3 前端类型（`frontend-core/src/types/system.ts` + `adminApi.ts`）

```ts
type UserStatus = 'active' | 'inactive' | 'locked';     // 前端枚举
type UserRole   = 'admin' | 'operator' | 'viewer' | 'auditor';

interface User {
  id: string;
  username: string;
  displayName: string;
  email: string;
  role: UserRole;
  status: UserStatus;
  phone?: string;
  // carrier?: string; — v1.0 删除（前端 User 类型同步移除）
  lastLoginTime?: string;
  createTime: string;
  updateTime?: string;
  department?: string;
  description?: string;
  // 页面额外使用（业务侧扩展，未在后端实体中）：
  userName?: string;        // 兼容老 key
  groupNames?: string[];    // 角色名数组（前端 UI 直接绑定，历史命名；v0.3 起建议改 roleNames）
  source?: string;          // "本地" | "LDAP"
  onlineStatus?: 'online' | 'offline';
  builtIn?: 0 | 1;
  expireTime?: string;
  createUser?: string;
  updateUser?: string;
}
```

### 2.4 前后端字段映射 & Gap 一览（重要）

> **本节是后端补齐工作的清单**。当前前端 UI 字段比后端 DB / API 多，部分字段是"前端纯展示 / 硬编码"。

| 前端字段 | 类型 | 后端字段 | DB 列 | 状态 |
|---------|------|---------|-------|------|
| `username` | string | `Username` | `users.username` | ✅ 已对齐 |
| `password` / `confirmPassword` | string | `Password`（仅创建/重置） | `users.password_hash` | ✅ |
| `displayName` | string | `DisplayName` | `users.display_name` | ✅（前端列表暂未展示，可用作复制时的副名） |
| `email` | string | `Email` | `users.email` | ✅ |
| `status` | `'enabled' \| 'disabled'`（页面） | `'active' \| 'disabled'`（后端） | `users.status` | ⚠️ **枚举值不一致**：前端写 `enabled`，后端定义 `active`。映射层 (`adminApi.ts:268-275`) 直传 → 后端会返 400/不识别。**需对齐：前端统一改 `active`，或在 `mapFrontendUser` 内做 `enabled→active` 转换** |
| `groupNames`（历史命名） / `roleNames`（v0.3 目标） | string[]（角色名） | `RoleIDs` UUID[] | `user_roles` | ⚠️ **类型不匹配 + 命名待统一**：前端送字符串名数组，后端要 UUID 数组。v0.3 决议：字段重命名 `groupNames → roleNames`；用户语义上挂的是**角色**，不是「组」。当前 `adminApi.createUser` 已传 `roleIds`，但页面 `Select` 仍写 `groupNames`，需后续统一 |
| `phone` | string | — | — | ❌ **后端缺字段**：DB / Model / DTO 都没有。前端表单输入后被丢弃。需补 `users.phone VARCHAR(32)` |
| `expireTime` | datetime | — | — | ❌ **后端缺字段**：账号有效期。需补 `users.expire_at TIMESTAMPTZ NULL`，登录时校验 |
| `description` | string | — | — | ❌ **后端缺字段**：需补 `users.description TEXT` |
| `source` | `'builtIn' \| 'admin' \| 'LDAP'`（v0.2） | `Source` | `users.source` | ⚠️ **DB 待补字段**（见 §7 P1）。**前端不再硬编码** —— 创建走 `POST /admin/users` 时后端默认写 `'admin'`；LDAP 同步任务写 `'LDAP'`；seed 写 `'builtIn'` |
| `onlineStatus` | 'online'/'offline' | — | — | ❌ **后端缺**。建议从会话/JWT 黑名单计算，不入主表（详见 §6.4） |
| `builtIn`（v0.1 字段，v0.2 移除） | 0/1 | — | — | 🗑️ **v0.2 起删除该独立字段**：前端从 `source === 'builtIn'` 派生 `isBuiltIn` 布尔值；后端响应里**不再下发** `built_in`（避免与 `source` 双源） |
| `lastLoginTime` | datetime | `LastLoginAt` | `users.last_login_at` | ✅ |
| `createTime` / `updateTime` | datetime | `CreatedAt` / `UpdatedAt` | `users.created_at` / `updated_at` | ✅ |
| `createUser` / `updateUser` | string | — | — | ❌ **后端缺**：当前没记录"谁创建/更新了这条用户"。建议补 `created_by UUID NULL` / `updated_by UUID NULL` 引用 `users.id` |

> Gap 列表对应的后端工作：见 [§7 后端补齐 Backlog](#7-后端补齐-backlog)。

### 2.5 数据权限关联表 → 已迁出

> v0.6：`role_device_groups` 表 schema、与 `device_groups` 的关系、`PermissionService` 判定逻辑、API 入口 全部迁至 [roles.md §2.6](./roles.md)。本 PRD 不再描述。

---

## 3. 列表页（Table）字段定义

### 3.1 列定义

| 顺序 | key | 列标题 | dataIndex 类型 | UI 渲染 | 列宽 | 备注 |
|------|------|-------|----------------|--------|------|------|
| 1 | `actions` | 操作 | — | `<Button type="link">查看</Button> + <Dropdown>更多</Dropdown>` | 100，`fixed: 'right'` | 见 §4 操作清单 |
| 2 | `displayName` | 用户昵称 | string | 纯文本（v0.7 起取消"内置" Tag — 内置/admin/LDAP 三种来源统一由第 8 列"来源"承担，避免冗余）；v1.1 列 title 由"用户名"改"用户昵称"，与表单字段对齐 | 150 | dataIndex 由前端历史 `userName` 改为真实字段 `displayName` |
| 2.5 | `username` | 用户账号 | string | 纯文本，monospace 字体（与 ID/账号语义一致） | 130 | v1.1 新增独立列，与"用户昵称"分离 |
| 3 | `status` | 状态 | enum | `<Tag color="success">启用</Tag>` / `<Tag color="error">禁用</Tag>` | 90 | 后端枚举对齐后改读 `active/disabled` |
| 4 | `onlineStatus` | 在线状态 | enum | `<Tag color="green">在线</Tag>` / `<Tag>离线</Tag>` | 90 | |
| 5 | `email` | 邮箱 | string | 文本，`ellipsis: true` | flex | |
| 6 | `phone` | 手机 | string | 文本，空显示 `-` | 120 | |
| 7 | `roles`（dataIndex；历史 `groupNames` 兼容） | 角色 | string[] | 单值直显；多值显示首项 + `...`，`<Tooltip>` 显示完整列表 | 150 | v0.3：dataIndex 已统一为 `roles` |
| 8 | `source` | 来源 | enum | 文本映射（v0.2）：`builtIn` → "内置"、`admin` → "管理员添加"、`LDAP` → "LDAP" | 100 | 建议用 `<Tag>`：`builtIn`→蓝色 / `admin`→默认 / `LDAP`→紫色 |
| 9 | `expireTime` | 过期时间 | datetime | 格式化 `YYYY-MM-DD HH:mm:ss`，空显示 `永久` | 160 | |
| 10 | `lastLoginTime` | 最后登录时间 | datetime | 格式化，空 `-` | 160 | |
| 11 | `createTime` | 创建时间 | datetime | 格式化，空 `-` | 160 | |
| 12 | `updateTime` | 更新时间 | datetime | 格式化，空 `-` | 160 | |
| 13 | `createdBy`（前端实际 dataIndex；PRD 历史名 `createUser`） | 创建人 | UUID | 非空 → `useAllUsers` 做 id→username 映射；**空 → "内置"**（v0.8） | 110 | |
| 14 | `updatedBy`（前端实际 dataIndex；PRD 历史名 `updateUser`） | 更新人 | UUID | 同上（共用 `renderUserId`） | 110 | |
| 15 | `description` | 备注 | string | `ellipsis: true`，长度 > 20 加 `<Tooltip>` 显示全量 | 150 | |

### 3.2 表格能力

| 能力 | 实现 | 备注 |
|------|------|------|
| 分页 | `pageSize` 默认 `20`，支持自定义页大小 | `total` 由后端返回 |
| 多选 | `selectable=true`，行级 `checkbox` | 选中后顶部出现批量操作栏 |
| 横向滚动 | `scroll={{ x: 1200 }}` | |
| 列设置 | 由 `DataTable` 通用组件提供（隐藏/显示列、列宽拖动、记忆） | `tableId="user-management-list"` |
| 刷新 | `onRefresh = refetch` | |

### 3.3 顶部筛选栏（FilterBar）

| 字段 | 标签 | UI 组件 | 校验 | 提交后端参数 |
|------|------|--------|------|------------|
| `userName` | 用户名 | `<Input>` | — | `?search=<keyword>`（后端按 `username` 模糊匹配） |

> **建议补充**（后续迭代）：状态筛选（启用/禁用）、来源筛选（本地/LDAP）、角色筛选、创建时间范围。后端 `UserFilter` 支持 `Status`/`Search` 字段（v1.0 起 `Carrier` 已删，按角色名隐式过滤 carrier 范围）。

### 3.4 顶部右侧按钮

| 按钮 | 图标 | type | 行为 |
|------|------|------|------|
| 添加 | `PlusOutlined` | `primary` | 打开 Drawer（含"添加用户" / "导入用户" Radio 切换） |
| 导出 | `ExportOutlined` | `default` | TODO（接口待定） |

---

## 4. 操作清单（Operations）

### 4.1 单行操作（"查看" + "更多"下拉）

> v0.2：权限按 `source` 判定（替代 v0.1 的 `builtIn` 派生）。✅ = 可用 / ❌ = 按钮置灰。

| 操作 | 图标 | 触发 | builtIn | admin | LDAP | 在线状态约束 | 后端接口 |
|------|------|------|---------|-------|------|-----------|---------|
| 查看 | — | "查看"按钮 | ✅ | ✅ | ✅ | — | `GET /admin/users/{id}` |
| 编辑 | `EditOutlined` | 更多菜单 | ✅（`username` 只读） | ✅ | ✅（`email/phone/角色/备注` 可改，`username` 只读） | — | `PUT /admin/users/{id}` |
| 复制 | `CopyOutlined` | 更多菜单 | ✅（副本 `source=admin`） | ✅ | ✅（副本 `source=admin`） | — | `POST /admin/users/{id}/copy` ❌ 后端未实现 |
| 启用 / 禁用 | `CheckCircleOutlined` / `StopOutlined` | 更多菜单 | ❌（**避免锁死系统**） | ✅ | ✅ | — | `POST /admin/users/{id}/lock` / `unlock` |
| 强制下线 | `LogoutOutlined` | 更多菜单 | ✅ | ✅ | ✅ | 仅在线时可点 | `POST /admin/users/force-logout` ❌ 后端未实现 |
| 重置密码 | `KeyOutlined` | 更多菜单（弹 Modal） | ✅（**紧急通道**：admin 忘密恢复） | ✅ | ❌（密码归属外部域） | — | `POST /admin/users/{id}/reset-password` |
| 删除 | `DeleteOutlined` | 更多菜单（弹 Confirm） | ❌ | ✅ | ✅（仅删本地映射，不影响 LDAP 源） | — | `DELETE /admin/users/{id}` |

**判定函数（前端 useMemo）建议**：

```ts
const isBuiltIn = (u: User) => u.source === 'builtIn';
const isLdap    = (u: User) => u.source === 'LDAP';
const canDelete = (u: User) => u.source !== 'builtIn';
const canToggleStatus = (u: User) => u.source !== 'builtIn';
const canResetPassword = (u: User) => u.source !== 'LDAP';
```

> ⚠️ v0.1 → v0.2 行为变化：内置用户**新增允许**「编辑非 username 字段、改密、强制下线」；**继续禁止**「删除、禁用」。

### 4.2 批量操作（多选后弹出）

> v0.2：判定改为基于 `source`。"自动跳过" = 接口对部分用户拒绝时，前端汇总成功/跳过列表展示。

| 批量操作 | 图标 | 选中含 `builtIn` 时 | 选中含 `LDAP` 时 | 接口 |
|---------|------|------------------|-----------------|------|
| 强制下线 | `LogoutOutlined` | 允许（builtIn 可被强制下线） | 允许 | `POST /admin/users/force-logout`（`{ userIds: [] }`）❌ 后端未实现 |
| 禁用 | `StopOutlined` | **整批被禁用**（按钮 disabled） | 允许 | 循环调用 `POST /{id}/lock` |
| 启用 | `CheckCircleOutlined` | 自动跳过 builtIn（始终启用） | 允许 | 循环调用 `POST /{id}/unlock` |
| 重置密码 | `KeyOutlined` | 允许（紧急通道） | 自动跳过 LDAP | 待补：`POST /admin/users/batch-reset-password` |
| 批量分配角色 | — | **整批被禁用** | 允许 | `POST /admin/users/assign-roles`（`{ userIds, roleIds }`，v0.3 重命名自 `move-group`）❌ 后端未实现 |
| 批量删除 | — | 自动跳过 builtIn | 允许（仅删本地映射） | 循环调用 `DELETE /{id}` |

**前端判定**：

```ts
const hasBuiltInSelected = selectedUsers.some(u => u.source === 'builtIn');
// 「禁用」「批量分配角色」按钮根据 hasBuiltInSelected 控制 disabled
```

**后端语义**：批量循环调用单条接口时，单条返回 `409 Conflict`（如对 builtIn 删除）由前端聚合为「跳过 N 条」展示，不打断整批流程。

### 4.3 弹窗 / 抽屉一览

| 名称 | 触发 | UI 容器 | 关键内容 |
|------|------|--------|---------|
| 创建用户 | 顶部"添加" | `<Drawer width=520>` | Radio 切换"添加" / "导入"；表单字段见 §5.1 |
| 编辑用户 | 行内"编辑" | `<Drawer width=520>` | 字段见 §5.2，`username` 只读 |
| 查看用户 | 行内"查看" | `<Drawer width=520>` | 全字段只读；状态用 Tag |
| 重置密码 | 行内"重置密码" | `<Modal width=420>` | 新密码 + 确认密码 |
| 批量分配角色 | 批量"分配角色" | `<Modal width=420>` | 单选/多选目标角色 |
| 导入用户 | 创建抽屉切换"导入" | 内嵌 `<ImportPanel>` | 接收 `.xlsx/.xls`，最大 10MB，含模板下载占位 |

---

## 5. 表单字段定义

### 5.1 创建用户 Form（Drawer，仅一种模式）

> v1.1 起：① 取消"导入用户"模式切换（Radio 移除）；② 字段顺序与 §5.2 编辑表单**完全一致**（创建独有的 password / confirmPassword 紧随 username 之后），便于 UI 视觉同步；③ 标签统一为"用户账号 / 用户昵称"。

| # | 字段 name | 标签 | 类型 | UI 组件 | 必填 | 默认值 | 校验 | 占位/选项 | 备注 |
|---|----------|------|------|--------|------|-------|------|---------|------|
| 1 | `username` | 用户账号 | string | `<Input maxLength=32>` | ✅ | — | `pattern: /^[a-zA-Z0-9_-]{3,32}$/`（字母/数字/`_`/`-`，3–32 位）| `用户账号` | 创建后不可改；v1.1 标签由"用户名"改"用户账号" |
| 2 | `password` | 密码 | string | `<Input.Password maxLength=20>` | ✅ | — | `min: 8` | `密码` | 提交时明文，由后端 bcrypt |
| 3 | `confirmPassword` | 确认密码 | string | `<Input.Password maxLength=20>` | ✅ | — | 必须等于 `password` | `确认密码` | 仅前端校验，不提交 |
| 4 | `displayName` | 用户昵称 | string | `<Input maxLength=64>` | — | — | — | `留空则与用户账号相同` | v1.1 标签由"用户名称"改"用户昵称"；提交时如为空，后端使用 username 兜底 |
| 5 | `email` | 邮箱 | string | `<Input maxLength=50>` | — | — | `type: email` | `邮箱` | |
| 6 | `phone` | 手机 | string | `<Input maxLength=11>` | — | — | `pattern: /^1\d{10}$/`（中国手机号）| `手机` | |
| 7 | `roleIds` | 角色 | UUID[] | `<Select mode="multiple">` | ✅ | — | 至少选一项 | options 来自 `useAllRoles()` → `{ label: r.roleName, value: r.id }` | 字段统一为 `roleIds`，与后端 `CreateUserRequest.RoleIDs` 对齐 |
| 8 | `status` | 状态 | enum | `<Radio.Group>` 激活 / 禁用 | — | `active` | — | — | |
| 9 | `expireTime` | 过期时间 | datetime | `<DatePicker showTime format="YYYY-MM-DD HH:mm:ss">`，禁用过去日期 | — | — | — | `留空表示永久有效；过期后该用户将无法登录` | |
| 10 | `description` | 备注 | string | `<Input.TextArea rows=3 maxLength=500 showCount>` | — | — | — | `备注（可选）` | |

**前端当前提交（payload）**（`UserManagement/index.tsx:154-166`，待重构）：

```json
{
  "username": "...",
  "password": "...",
  "email": "...",
  "phone": "...",
  "groupNames": ["...","..."],  // v0.3 起前端统一改 roleIds: ["uuid",...]
  "status": "enabled",
  "expireTime": "2026-12-31 23:59:59",
  "description": "...",
  "source": "本地",     // v0.2 起前端不再传，待移除
  "onlineStatus": "offline",  // 派生字段，待移除
  "builtIn": 0          // v0.2 起删除该字段，待移除
}
```

> 注：`adminApi.createUser` 在发送前重新挑字段（仅 `username/password/email/displayName/status`），其他字段被静默丢弃。**v0.2 对齐后**前端应直接送：

```json
{
  "username": "...",
  "password": "...",
  "displayName": "...",
  "email": "...",
  "phone": "...",
  "status": "active",
  "expireAt": "2026-12-31T23:59:59Z",
  "description": "...",
  "roleIds": ["uuid-1","uuid-2"]
}
```

**`source` 不在请求体中**：后端在 handler 里根据调用上下文（认证主体的角色 / 路由）自动写入：
- `POST /admin/users` 走管理员通道 → `source = 'admin'`
- LDAP 同步任务（独立服务路径）→ `source = 'LDAP'`
- `migrations/seed/` 文件 → `source = 'builtIn'`

### 5.2 编辑用户 Form（Drawer）

> v1.1 起：字段顺序与 §5.1 创建表单同步（除了创建独有的 password/confirmPassword）；标签统一为"用户账号 / 用户昵称"。

| # | 字段 name | 标签 | UI 组件 | 是否可编辑 | 备注 |
|---|----------|------|--------|-----------|------|
| 1 | `username` | 用户账号 | `<Input readOnly>` | ❌ 只读 | 创建后不可改 |
| 2 | `displayName` | 用户昵称 | `<Input maxLength=64>` | ✅ | v1.1 标签由"用户名称"改"用户昵称" |
| 3 | `email` | 邮箱 | `<Input>` | ✅ | 同创建 |
| 4 | `phone` | 手机 | `<Input>` | ✅ | 同创建 |
| 5 | `roleIds` | 角色 | `<Select mode="multiple" allowClear>` | ✅ | 后端 `service.syncUserRoles` 做差量同步 |
| 6 | `status` | 状态 | `<Radio.Group>` | ✅ | 激活/禁用 |
| 7 | `expireTime` | 过期时间 | `<DatePicker>` | ✅ | 留空=永久 |
| 8 | `description` | 备注 | `<Input.TextArea>` | ✅ | |

> **不可改字段**（所有用户类型）：
> - `username`（创建后不可改）
> - `password`（走"重置密码"专用接口；LDAP 用户连重置密码也不可用）
> - `source`（仅由系统按创建路径写入）
> - 系统时间戳 `createTime` / `updateTime` / `lastLoginTime`

### 5.3 查看用户 Drawer（只读）

按下表分组展示，全部只读：

| 字段 | 展示组件 | 空值兜底 |
|------|---------|---------|
| `onlineStatus` | `<Tag>` 在线/离线 | — |
| `status` | `<Tag>` 启用/禁用 | — |
| `username`（用户账号） | `<Input readOnly>` | — |
| `displayName`（用户昵称） | `<span>` 文本 | `-` |
| `email` | `<Input readOnly>` | — |
| `phone` | `<span>` 文本 | `-` |
| `roles`（角色名数组） | `<span>` 用 `, ` 拼接 | `-` |
| `source` | `<Tag>` 文本映射：`builtIn`→"内置" / `admin`→"管理员添加" / `LDAP`→"LDAP" | — |
| `expireTime` | `<span>` 格式化 | `永久` |
| `lastLoginTime` | `<span>` 格式化 | `-` |
| `createTime` / `updateTime` | `<span>` 格式化 | `-` |
| `createUser` / `updateUser` | `<span>` | `-` |
| `description` | `<span>` | `-` |

### 5.4 重置密码 Modal

| 字段 name | 标签 | UI 组件 | 必填 | 校验 |
|----------|------|--------|------|------|
| `newPassword` | 新密码 | `<Input.Password maxLength=20>` | ✅ | `min: 8` |
| `confirmPassword` | 确认密码 | `<Input.Password maxLength=20>` | ✅ | 必须等于 `newPassword` |

**提交**：`POST /admin/users/{id}/reset-password` `{ "new_password": "..." }`

### 5.5 批量分配角色 Modal（v0.3 重命名自「移动到组」）

| 字段 | 标签 | UI 组件 | 必填 | 选项 |
|------|------|--------|------|------|
| `targetRoleIds` | 目标角色 | `<Select mode="multiple">` 多选 | ✅ | options 来自 `useAllRoles()` → `{ label: r.name, value: r.id }`；历史 hook `useAllGroups()` 等价但建议替换 |

**提交**：`POST /admin/users/assign-roles` `{ "userIds": ["..."], "roleIds": ["..."] }`（**后端待实现**；v0.3 重命名自 `move-group`）

> 语义：用 `roleIds` **整体替换** 选中用户的角色集（与 `UpdateUserRequest.RoleIDs` 一致的差量同步语义）。如需「追加角色」交互，另开端点 `POST /admin/users/append-roles`。

### 5.6 导入用户（v1.1 起前端入口移除）

> v1.1：取消用户管理页面的"导入用户"前端入口（Radio 模式切换 + ImportPanel 集成均移除）。新增 Drawer 仅保留单一"添加用户"表单。

**保留的后端能力**（不暴露 UI，可作内部工具/调试用）：

- `POST /admin/users/import`（multipart/form-data，xlsx）— `internal/admin/user_import_handler.go`
- `GET /admin/users/import/template`（xlsx 模板下载）

**未来若重新启用**：单独立项「批量导入入口」需求，UI 入口可选（独立按钮 / 单独菜单 / 独立页面），不再寄生在"添加用户"Drawer 里。

---

## 6. 接口契约（API Contracts）

> Base URL：`/api/v1/admin`
> 鉴权：所有接口需 JWT（`Authorization: Bearer <token>`），中间件 `RequireAuth + RequireAdminRole`
> 错误响应统一：`{ "error": "<message>", "code": <int> }`，错误码见 [`global/errors.go`](../../../global/errors.go)

### 6.1 用户管理接口（已实现）

#### `GET /admin/users` — 列表

**Query 参数**（来自 `UserFilter` + `ListRequest`）：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `page` | int | — | 默认 1 |
| `pageSize` | int | — | 默认 20 |
| `search` | string | — | 模糊匹配 `username` |
| `status` | `active` / `disabled` | — | |
| ~~`carrier`~~ | ~~`cmcc` / `ctcc` / `cucc`~~ | — | **v1.0 删除**（参 §11.11）|

**Response 200**：

```json
{
  "items": [{ /* User */ }],
  "total": 123,
  "page": 1,
  "pageSize": 20,
  "totalPages": 7
}
```

#### `GET /admin/users/{id}` — 详情

**Response 200**：单个 `User` 对象（含 `roles[]`）

#### `POST /admin/users` — 创建

**Body**（`CreateUserRequest`）：

```json
{
  "username": "alice",            // required, 3-64
  "password": "P@ssw0rd",         // required, min 6
  "display_name": "Alice",
  "email": "alice@op.cn",         // optional, email
  "role_ids": ["uuid-1","uuid-2"] // optional （v1.0 删除 carrier 字段）
}
```

> **注**：请求体中**不接受 `source` 字段**。该端点为「管理员添加」入口，后端固定写 `source = 'admin'`。LDAP 同步走独立内部 API，不暴露给前端。

**Response 201**：创建后的 `User` 对象（含 `source: 'admin'`）

#### `PUT /admin/users/{id}` — 更新

**Body**（`UpdateUserRequest`，全部可选指针；`role_ids` 非 nil 时整体替换角色集）：

```json
{
  "display_name": "Alice Liu",
  "email": "alice2@op.cn",
  "phone": "13800000000",
  "status": "disabled",
  "role_ids": ["uuid-1", "uuid-2"]
}
```

> **数据权限副作用**（v0.4 + v1.0 修订）：当 `role_ids` 字段被更新时，service 必须 `InvalidateUserCache(userID)`。v1.0 起 `carrier` 字段已删，超管身份由 `source = 'builtIn'` 决定且 source 不可修改，无升降超管路径。

**Response 200**：更新后的 `User` 对象

#### `DELETE /admin/users/{id}` — 删除

**前置校验**（v0.2）：
- `source = 'builtIn'` → 返回 `409 Conflict` `{ "error": "built-in user cannot be deleted", "code": <int> }`
- `source = 'LDAP'` → 允许（仅删本地映射，LDAP 服务端数据不动）

**Response 204** No Content

#### `POST /admin/users/{id}/reset-password` — 重置密码

**前置校验**（v0.2）：
- `source = 'LDAP'` → 返回 `409 Conflict` `{ "error": "LDAP user password is managed externally" }`（LDAP 用户密码归外部域，重置请到 LDAP/AD 端；详见 §12.5）
- `source = 'builtIn'` → 允许（紧急通道）

**Body**：`{ "new_password": "..." }`（min 6）
**Response 200**：`{ "message": "password reset" }`

#### `POST /admin/users/{id}/lock` — 禁用

**前置校验**（v0.2）：
- `source = 'builtIn'` → 返回 `409 Conflict` `{ "error": "built-in user cannot be disabled" }`

**Response 200**：`{ "message": "user locked" }`（实质：`status = disabled`）

#### `POST /admin/users/{id}/unlock` — 启用

**Response 200**：`{ "message": "user unlocked" }`（实质：`status = active`）

#### `POST /admin/users/{id}/roles` — 分配角色

**Body**：`{ "role_id": "uuid" }`
**Response 200**：`{ "message": "role assigned" }`

> **数据权限副作用**（v0.4）：分配角色后，service 层必须调用 `PermissionService.InvalidateUserCache(userID)` 失效该用户 `perm:visible_groups` 缓存（模型详见 [roles.md §1.4](./roles.md)）。否则用户在长达 5 分钟内看不到新角色对应的设备数据。

#### `DELETE /admin/users/{id}/roles/{roleId}` — 解绑角色

**Response 200**：`{ "message": "role removed" }`

> **数据权限副作用**（v0.4）：同上，必须 `InvalidateUserCache(userID)`，避免用户继续看到已解绑角色的设备数据。

### 6.2 角色查询接口

| Method | 路径 | 说明 |
|--------|------|------|
| GET | `/admin/roles` | 列出角色（分页） |
| GET | `/admin/roles/{id}` | 单角色详情 |
| GET | `/admin/roles/all` | 全量角色（用于下拉选项，无分页） |

**v0.3 弃用别名（保留兼容期，建议下版本删除）**：

| Method | 路径 | 等价于 | 备注 |
|--------|------|--------|------|
| GET | `/admin/groups` | `GET /admin/roles` | 仅为前端历史命名兼容；新代码不应再调用 |
| GET | `/admin/groups/{id}` | `GET /admin/roles/{id}` | 同上 |

### 6.3 待补接口（前端已调用，后端未实现）

> **需求来源**：前端 `adminApi.ts` 中已有调用代码，但后端 router 里未注册 → 当前调用会返回 404。

| Method | 路径 | 用途 | Body | 响应 | 优先级 |
|--------|------|------|------|------|--------|
| POST | `/admin/users/force-logout` | 批量强制下线 | `{ "userIds": ["uuid",...] }` | `{ "message": "logged out N users" }` | P1 |
| POST | `/admin/users/assign-roles` | 批量替换选中用户的角色集（v0.3 重命名自 `move-group`）。**v0.4：成功后逐个 invalidate `perm:visible_groups:{userID}` 缓存** | `{ "userIds": [...], "roleIds": ["uuid",...] }` | `{ "message": "assigned" }` | P1 |
| POST | `/admin/users/{id}/copy` | 复制用户（生成 `username_copy_N`，沿用角色，密码需重置） | — | 新 `User` 对象 | P2 |
| POST | `/admin/users/{id}/force-logout` | 单用户强制下线（也可由批量接口覆盖） | — | `{ "message": "logged out" }` | P2 |
| POST | `/admin/users/import` | Excel 批量导入 | `multipart/form-data: file=*.xlsx` | `{ "succeeded": N, "failed": [{row, reason}] }` | P2 |
| GET | `/admin/users/import-template` | 下载导入模板 | — | `*.xlsx` 二进制流 | P3 |
| POST | `/admin/users/batch-reset-password` | 批量重置密码（生成随机密码或统一密码） | `{ "userIds": [...], "strategy": "random" \| "fixed", "password"?: "..." }` | `{ [userId]: newPassword }`（仅 `random` 时返回，需安全通道） | P3 |

### 6.4 在线状态 / `onlineStatus` 设计建议

当前 DB 无该列。建议实现路径：

1. **会话源数据**：登录时把 `<userId, jti, exp>` 写入 Redis `auth:session:{userId}:{jti}`（TTL=token 过期）
2. **`onlineStatus` 派生**：列表查询时批量 `EXISTS auth:session:{userId}:*` → 标记 online
3. **强制下线**：删除该用户所有 `auth:session:*` key + 在 Redis 写入 `auth:revoked:{jti}` 黑名单（中间件每次校验）

不建议入主表，避免高频写。

---

## 7. 后端补齐 Backlog

> 按优先级排序，建议拆为独立 sprint 任务。

### P0（阻塞前端 UI 已展示功能）

1. **状态枚举对齐**：前端把 `enabled / disabled` 替换为 `active / disabled`（或后端兼容 `enabled` 别名）
2. **角色绑定打通**：`adminApi.createUser` payload 已加上 `role_ids`（已完成）。v0.3 起前端 `Select` 直接绑 `roleIds`（UUID），不再需要 `lookup-by-names` 兼容路径
3. **`source` 字段落地**（v0.2 替代 v0.1 的 `builtIn` 派生方案）：
   - DDL：`ALTER TABLE users ADD COLUMN source VARCHAR(16) NOT NULL DEFAULT 'admin' CHECK (source IN ('builtIn','admin','LDAP'))`
   - 索引：`CREATE INDEX idx_users_source ON users(source)`
   - seed：把现有内置 admin 用户更新为 `source = 'builtIn'`
   - Model/DTO：`User` 结构体加 `Source string` 字段，JSON tag `"source"`；`CreateUserRequest` **不含** `Source`（由 handler 写入）
   - Handler 校验：DELETE / lock / 重置密码 接口加前置校验（详见 §6.1）
   - 前端 `User` 类型加 `source` 字段，移除 `builtIn` 字段；列表 / 操作按钮判定全部改基于 `source`

### P1（前端已建表单，后端未承接）

4. **DDL 增列**（一个迁移文件 `000NNN_users_extend.sql`，与 P0 第 3 项可合并为同一个迁移文件）：
   - `phone VARCHAR(32) NULL`
   - `expire_at TIMESTAMPTZ NULL`
   - `description TEXT NULL`
   - `created_by UUID NULL` / `updated_by UUID NULL`（`REFERENCES users(id) ON DELETE SET NULL`）
5. **Model / DTO 跟进**：`User` 结构体、`CreateUserRequest`、`UpdateUserRequest` 增对应字段
6. **登录守卫扩展**：登录时校验 `expire_at` 已过 / `status = disabled`；`source = 'LDAP'` 改走 LDAP bind 验证流程（**v0.9 已落定完整方案，详见 §12.5**）
7. **强制下线接口**（§6.3 第 1 条）+ Redis 会话/黑名单设计（§6.4）
8. **批量分配角色接口** `POST /admin/users/assign-roles`（v0.3 重命名自 `move-group`，§6.3 第 2 条）：语义为「整体替换角色集」（与 `UpdateUserRequest.RoleIDs` 差量同步一致）。如需「追加」语义另开 `append-roles`

### P2（提升体验）

9. 复制用户接口
10. Excel 导入 / 模板下载
11. 批量重置密码（含密码策略校验）
12. 列表筛选条件扩展（前端 + 后端）：状态、来源、角色、创建时间范围

### P3（运维加固）

13. 登录失败锁定策略（DB 已有 `failed_login_attempts` / `locked_until`，需在登录中间件落地）
14. 密码策略（最小长度、复杂度、历史 N 次不可重复）
15. `audit_logs` 全覆盖：创建 / 更新 / 删除 / 锁定 / 重置密码 / 强制下线 全部写审计

---

## 8. 非目标（Out of Scope）

- 单点登录（SSO / OAuth2）—— 由独立模块处理
- ~~LDAP 同步实现~~（v0.9 起**已纳入本 PRD 范围**，详见 §12）
- ~~多租户用户隔离 —— 当前以 `carrier` 字段做软隔离~~（v1.0 已删除 `carrier` 字段；多 carrier 隔离能力由 `role_device_groups` 通过设备分组层提供，不在 user 表层做隔离）
- 用户偏好（主题、语言）—— 由 `appStore` 前端本地保存，不入用户主表
- API Keys 管理 —— 已有 `api_keys` 表（见 §2.1），但本页面不涉及

---

## 9. 度量指标（Metrics）

| 指标 | 类型 | 来源 | 用途 |
|------|------|------|------|
| `omc_user_count_total{status}` | Gauge | 周期上报 | 看启用/禁用用户数 |
| `omc_user_login_attempts_total{result}` | Counter | 登录中间件 | success/failed/locked 区分 |
| `omc_user_session_active` | Gauge | Redis SCAN `auth:session:*` | 当前在线数 |
| `omc_admin_user_op_total{op}` | Counter | handler 中间件 | create/update/delete/reset_password/lock/unlock |

---

## 10. 验收清单（DoD）

后端：

- [ ] §6.1 全部接口 `go test` 单测覆盖（含成功/失败/权限不足三类）
- [ ] §6.3 P1 接口实现并补 E2E 用例（`scripts/e2e_verify.sh`）
- [ ] §7 P0 + P1 全部完成
- [ ] `users.source` DDL 落地，CHECK 约束生效，seed 文件中 admin 用户的 source = `'builtIn'`
- [ ] DELETE / lock 接口对 `source = 'builtIn'` 用户返回 `409 Conflict`
- [ ] reset-password 接口对 `source = 'LDAP'` 用户返回 `409 Conflict`
- [ ] `POST /admin/users` 忽略请求体里的 `source` 字段，固定写入 `'admin'`
- [ ] 所有写操作（create/update/delete/reset/lock/unlock/force-logout/assign-roles）写入 `audit_logs`
- [ ] **数据权限缓存一致性**（v0.4 + v0.5 决议 ②）：以下操作成功后必须**同步**调 `PermissionService.InvalidateUserCache(userID)`，**禁止改造为 EventBus / 异步事件**：
  - `POST /admin/users/{id}/roles`（分配单角色）
  - `DELETE /admin/users/{id}/roles/{roleId}`（解绑单角色）
  - `PUT /admin/users/{id}` 当 `role_ids` 字段被更新（v1.0：`carrier` 已删除，无需关注）
  - `POST /admin/users/assign-roles`（批量分配角色，逐用户失效）
  - `DELETE /admin/users/{id}`（删除用户，顺带清空缓存避免泄漏）
- [ ] 上述每个 service 方法的单元测试断言：写库成功后 `InvalidateUserCache(userID)` 被调用恰好 1 次（mock 验证）
- [ ] `InvalidateUserCache` 失败不阻塞 handler 返回，但记 ERROR 日志 + 计数器 `omc_perm_cache_invalidate_failed_total`
- [ ] **v0.5 决议 ③ 交叉引用**：删除设备分组的 service（属 F06 拓扑管理 PRD）实现时，必须按 §11.7 决议 ③ 的审计字段约定写 `audit_logs`，并对受影响角色下的所有用户调 `InvalidateUserCache`

前端：

- [ ] 状态枚举与后端对齐（`active`/`disabled`）
- [ ] 创建用户提交 payload 与 `CreateUserRequest` 对齐（含 `role_ids`，**不再传** `source` / `onlineStatus` / `builtIn`）
- [ ] `User` TS 类型移除 `builtIn` 字段，新增 `source: 'builtIn' | 'admin' | 'LDAP'`
- [ ] 列表「内置」Tag、操作按钮 disabled 判定全部改用 `user.source === 'builtIn'`
- [ ] 内置用户的 **禁用/删除** 按钮置灰且鼠标悬停有提示；**改密/编辑/强制下线** 按钮可用
- [ ] LDAP 用户的"重置密码"按钮置灰
- [ ] 强制下线 / 批量分配角色 / 复制用户 三个按钮调用真实接口（不再 toast 假成功）
- [ ] 字段命名收口：`groupNames → roleIds`（创建 / 编辑表单），`targetGroupId → targetRoleIds`（批量分配角色 Modal），hook `useAllGroups → useAllRoles`
- [ ] **v0.5 决议 ① 落地**：「分配角色」下拉 / 批量分配角色 Modal 中，对 `role.deviceGroupIds.length === 0` 的角色 option，文本后追加 ⚠️ + 副文「(无设备权限)」
- [ ] `npm run typecheck` & `npm run lint` 通过
- [ ] Playwright E2E 覆盖：创建（admin 来源） → 登录 → 重置密码 → 禁用 → 强制下线 → 删除时 builtIn 用户被拒绝
- [ ] Playwright E2E 覆盖：分配「未绑分组」角色后，下拉 option 显示 ⚠️；用户登录后设备列表为空（验证 v0.5 决议 ①）

数据库：

- [ ] 新增迁移版本号严格连续（参考 `omcgo/CLAUDE.md §5.5`）
- [ ] Down 迁移完整（DROP COLUMN IF EXISTS source / phone / expire_at / description / created_by / updated_by）
- [ ] 索引覆盖 `source`（操作权限判定）、`phone`（如需按手机号查找）、`expire_at`（如需定期清理过期账号）
- [ ] CHECK 约束 `source IN ('builtIn','admin','LDAP')` 已校验（写入非法值返回数据库错误）

---

## 11. 决策记录（v0.1 → v0.2）

> 本节记录 v0.2 相对 v0.1 的语义变化，便于回顾与后续迭代评估。

### 11.1 字段语义变更

| 字段 | v0.1 | v0.2 | 决策依据 |
|------|------|------|---------|
| `users.source` | DB 缺，前端硬编码 `"本地"` | DB 显式列，枚举 `builtIn`/`admin`/`LDAP` | 用户类型作为系统行为分支（删除/禁用/改密权限），应作为权威字段而非派生 |
| `User.builtIn` | 0/1 字段（计划由后端派生下发） | **删除**，由前端从 `source === 'builtIn'` 派生 | 避免 `source` 与 `builtIn` 双源不一致；单事实源原则 |
| 「内置」判定 | `username == 'admin'` 或 `is_system` 角色 | `source === 'builtIn'` | 前者是规则耦合，后者是显式语义 |
| `is_system`（roles 表） | 仍保留 | 仍保留，但 **不再参与** 内置用户判定 | `is_system` 表达「内置角色不可删除」，与「内置用户」是两个正交维度 |

### 11.2 操作权限矩阵变更

| 操作 × 类型 | v0.1 | v0.2 | 理由 |
|------------|------|------|------|
| 编辑（内置） | ❌ 全字段只读 | ✅ 除 `username` 外可改 | 内置用户也需要日常维护（改邮箱/角色/备注） |
| 重置密码（内置） | ❌ disabled | ✅ 紧急通道 | admin 忘密恢复必须可达 |
| 强制下线（内置） | ❌ disabled | ✅ 可用 | 安全运维场景：强制内置账号下线后审计 |
| 启用 / 禁用（内置） | ❌ disabled | ❌ disabled（保留） | 禁用后系统失去登录入口，硬性约束不变 |
| 删除（内置） | ❌ disabled | ❌ disabled（保留） | 不可删除，硬性约束不变 |
| 删除（LDAP） | ✅ 未明确语义 | ✅ 仅删本地映射 | 不影响 LDAP 服务端，下次同步可重新拉取 |

### 11.3 字面值大小写约定

`'builtIn' | 'admin' | 'LDAP'` 三者大小写不统一（camelCase / lower / UPPER）：

- 这是用户在 v0.2 需求中给定的字面值，**保留不变**
- 由 DB CHECK 约束锁定，前后端 / 文档 / SQL 中必须严格保持
- 若未来希望统一（如全部小写 `builtin/admin/ldap`），需走数据迁移 + 前后端同步发版（列入 P3）

### 11.4 待澄清事项（后续迭代再决议）

- [ ] LDAP 用户被「批量删除」时，是否需要从 LDAP 同步排除列表中追加？避免下次同步又拉回来
- [ ] 多个 LDAP 数据源场景下，`source = 'LDAP'` 是否要细分成 `LDAP:<domain>`？当前假设单一 LDAP 域
- [ ] 内置用户密码重置后，是否强制下次登录修改密码？建议引入 `must_change_password BOOLEAN` 字段（独立任务）

### 11.5 v0.3 — 术语统一：用户组 → 用户角色

**背景**：v0.1/v0.2 中前端历史命名「用户组（Group）」与后端「角色（Role）」语义重复 —— 用户身上挂的就是角色，不存在第二层「组」概念（设备分组 `DeviceGroup` 是另一个独立概念，不在本 PRD 范围）。继续两套术语会让产品文案、API、字段三方持续错位。

**决策**：用户身上挂的统一称作 **角色（Role）**。所有 PRD 文案、UI 标签、新代码字段一律使用「角色 / Role / role / roleId / roleIds / roleNames」。

**术语对照表**：

| 旧（v0.1/v0.2，弃用） | 新（v0.3，统一） | 影响面 |
|--------------------|---------------|-------|
| 用户组 / 移动到组 | 用户角色 / 批量分配角色 | 文档、UI 标签、按钮文案 |
| `groupNames`（字段） | `roleIds`（字段，UUID 数组） | 前端创建/编辑表单 |
| `targetGroupId`（字段） | `targetRoleIds`（字段，UUID 数组） | 前端批量分配角色 Modal |
| `useAllGroups()`（hook） | `useAllRoles()`（hook） | `frontend-core/src/hooks/api/` |
| `Group` / `BackendGroup`（TS 类型） | `Role` / `BackendRole` | `frontend-core/src/types/system.ts` |
| `GET /admin/groups`（端点） | `GET /admin/roles` | 后端路由；旧路径保留兼容期 |
| `POST /admin/users/move-group`（端点） | `POST /admin/users/assign-roles` | 后端新接口（旧名从未实现，直接弃用） |
| `g.groupName`（角色名属性） | `r.name` | 与后端 `Role.name` 直接对齐 |

**保留兼容期**：

- `GET /admin/groups` / `GET /admin/groups/{id}` 保留为 `GET /admin/roles*` 的别名一个发布周期，给前端旧版本留迁移窗口；下版本删除。
- `POST /admin/users/move-group` 从未在后端落地，**直接以新名 `assign-roles` 实现，不写别名**。
- 前端 `groupNames` / `targetGroupId` 字段 **不保留兼容**，按 §10 DoD 一次切换到 `roleIds` / `targetRoleIds`。

**不在范围**：

- `internal/admin/role.go` 的 `Group` 资源（如果存在）属于「设备分组」概念，与本次术语统一无关。
- 「角色 vs 权限」语义不变：角色仍是权限的容器（继承自 `roles.permissions`）。

### 11.6 v0.4 — 数据权限模型（用户视角，v0.6 重整）

**背景**：v0.3 完成「用户身上挂的是角色」语义统一后，仍未在 PRD 中说明「角色凭什么能看到设备数据」。后端 `PermissionService` + `role_device_groups` 表 + `device_groups` 树形分组 + Redis 缓存早已实现。本节 v0.4 引入完整 R 规则；v0.6 起将"角色侧"规则（R1/R2/R5/R6）迁至 [roles.md §11.1](./roles.md)，本节仅保留**用户视角**两条规则。

**用户视角规则**（详细 R 全集见 [roles.md §11.1](./roles.md)）：

| 规则 | 内容 |
|------|------|
| **R3 用户继承** | 用户可见域 = 各角色 `role_device_groups` 的**并集**；多角色按并集放宽，不是交集收紧 |
| **R4 超管旁路**（v1.0 改）| `users.source = 'builtIn'` 表示超管，绕过 R2/R3，看见所有分组（实现：`PermissionService` + `User.IsSuperAdmin()` 派生）。source 创建时即固定，不可运行期改写 |
| **R7 缓存一致性**（用户操作侧）| 用户管理任何会改变可见域的操作（分配/解绑角色、删用户）必须**同步**调 `InvalidateUserCache(userID)`（v1.0：`carrier` 已删，无需再列 carrier 变更场景）|

**与本 PRD 的边界**（v0.6 更新）：

- 数据权限派生链与 R1/R2/R5/R6 → [roles.md §1.4 / §11.1](./roles.md)
- `role_device_groups` 表 schema → [roles.md §2.6](./roles.md)
- 角色 ↔ 设备分组绑定 CRUD → [roles.md §6.4](./roles.md)
- 决议 ① 角色未绑分组 UI 提示 → [roles.md §11.2](./roles.md)
- 决议 ③ 设备分组被删除时角色侧感知 → [roles.md §11.4](./roles.md)
- 「设备分组」本身的 CRUD → F06 拓扑管理 PRD

**待澄清事项**：→ 三项均在 v0.5 落定，详见 §11.7。

### 11.7 v0.5 — 数据权限三项决议（v0.6 重整）

延续 §11.6 v0.4 留下的三个待澄清事项，v0.5 给出最终决议；v0.6 起将决议 ① ③ 迁至 [roles.md §11.2 / §11.4](./roles.md)（属角色侧职责），本节仅保留决议 ② 全文与决议 ① ③ 在用户管理侧的落点。

#### 决议 ① → 已迁至 [roles.md §11.2](./roles.md)

**用户管理侧落点**：「分配角色」下拉 / 「批量分配角色」Modal 中，对 `role.deviceGroupIds.length === 0` 的角色 option，文本后追加 ⚠️ + 副文「(无设备权限)」。详见 §10 DoD。

#### 决议 ②：缓存失效不引入 EventBus，handler 同步直调

**规则**：用户管理任何会改变 `perm:visible_groups` 缓存的写操作，handler 调 service 完成业务后，**同步**调用 `PermissionService.InvalidateUserCache(userID)`，**不引入** EventBus / `user.role.changed` 等异步事件。

**理由**：

| 维度 | 同步直调 | EventBus 异步 |
|------|---------|--------------|
| 一致性窗口 | 0ms（请求返回前已失效） | 由消费者吞吐 + JetStream 延迟决定，可能 50ms~ 数秒 |
| 故障模式 | service 调用失败可立即报错回滚 | 事件发布成功但消费失败 → 失效漏掉，权限漂移 |
| 调用链 | handler → service → InvalidateUserCache（一层） | handler → publish → broker → consumer → service（多层） |
| 测试 | 单元测试直接 stub `permissionService.InvalidateUserCache` | 需启 NATS 测试容器，或 mock 整个事件管道 |
| 可观测性 | 失败直接出现在 handler 调用栈 | 跨进程追踪需依赖 OpenTelemetry trace 链路 |
| 运维负担 | 无新增组件 | 增加 NATS subject 监控、积压告警、消费者健康检查 |

**当前业务规模**：用户管理写操作 QPS 极低（< 1 QPS 量级），耦合一层 service 调用对延迟 / 性能无可观测影响，不需要异步解耦带来的复杂度。

**实施约束**：

- `AdminService.AssignRole` / `RemoveRole` / `UpdateUser`（含 `role_ids`）/ `DeleteUser` / `BatchAssignRoles` —— 这 5 个方法必须在写库成功后**返回前**调 `InvalidateUserCache`（v1.0：carrier 已删，触发条件简化）
- `InvalidateUserCache` 失败**不阻塞**写库结果（已落库），但记 ERROR 日志 + Prometheus 计数器 `omc_perm_cache_invalidate_failed_total`
- 单元测试覆盖：每个方法的 happy path 必须断言 cache invalidate 被调用一次（用 mock）

**未来回退**：若用户管理 QPS 突破 100 QPS（多租户 SaaS 场景）或缓存失效成为热点，可改异步，届时另开决议。

#### 决议 ③ → 已迁至 [roles.md §11.4](./roles.md)

**用户管理侧落点**：删除设备分组的 service（属 F06 拓扑管理 PRD）实现时，必须对受影响角色下的所有用户调 `InvalidateUserCache`，避免缓存中保留已失效的可见域。详见 §10 DoD「v0.5 决议 ③ 交叉引用」条目。

### 11.8 v0.7 — 列表"用户名称"列取消"内置"标记

**问题**：v0.2 引入 `users.source` 显式枚举后，列表已有独立的"来源"列（§3.1 第 8 列）通过 `<Tag>` 颜色映射 `builtIn`/`admin`/`LDAP` 三态。同时"用户名称"列在 builtIn 用户后面再追加 `<Tag color="blue">内置</Tag>`，导致：

- 同一信息（来源）在两列重复展示，视觉噪声
- 当列设置允许用户隐藏"来源"列时，行为依然正确（标记仅在第 8 列承担），不需要在用户名列做兜底
- "管理员添加"/"LDAP" 两类来源在用户名列没有对应 Tag，唯独 builtIn 有，三态不对称

**决议**：

- 列表"用户名称"列回归纯文本，**不再追加任何 Tag**
- 内置/管理员/LDAP 来源信息**统一由"来源"列**通过 `<Tag>` 颜色映射展现（`builtIn`→蓝色 / `admin`→默认 / `LDAP`→紫色）
- 操作菜单上对内置/LDAP 用户的按钮**置灰 + Tooltip 解释**规则**不变**（§4.1 + 前端 DoD）

**不在范围**：

- 「用户名称」列下方与 `username` 一起组合展示的两行布局（如某些设计稿见过）—— 当前 PRD 仍保持两个独立列（`displayName` + `username`）
- 「来源」列改用更紧凑的 icon（PRD 仍保留 `<Tag>` 文本形态）

**实施面**：

- 前端 `UserManagement/index.tsx` 列定义中 `displayName` 列 `render` 移除 `isBuiltIn(user) && <Tag>` 分支，回到纯文本

### 11.9 v0.8 — 列表"创建人/更新人"列空值显示"内置"

**问题**：`users.created_by / updated_by` 是 v0.4 / 000054 迁移加入的可空 UUID 列，目前在三种情况下会是 `NULL`：

1. **内置用户（source=builtIn）**：seed 文件写入，没有 operator
2. **LDAP 同步任务**：同步进程没有 operator UUID（属于"系统"代理）
3. **历史数据兼容**：v0.7 之前已存在的用户，迁移 000054 加列时也是 NULL

之前列里这三种都显示 `-`，让管理员看不懂"为什么这条没创建人"。

**决议**：

- "创建人" / "更新人" 列：**`createdBy / updatedBy` 为空时显示"内置"**；非空时仍走 `useAllUsers` 做 id→username 映射
- 复用同一个 `renderUserId(val)` 函数，列表两列 + 查看 Drawer 两个 `Form.Item` 共享，避免散落
- 含义说明：「内置」在本上下文等价于「无 operator UUID」，覆盖 builtIn / LDAP / 历史 三种成因

**为何不区分 LDAP / 历史 / builtIn**：

| 选项 | 取舍 |
|------|------|
| 当前选定：统一显示"内置" | 简单可读；来源信息已由"来源"列承担，不会真混淆 |
| 备选 A：用 `source` 派生（builtIn→"内置" / LDAP→"LDAP 同步" / 其它空 → "-"） | 更精准，但渲染依赖第二个字段，且历史数据无 source 时仍要兜底 |
| 备选 B：后端 join `users` 表把"系统"显示为伪用户名 | 引入伪行，污染 users 表 |

选项一被采纳：UI 信息密度低、来源已显式列出、避免后端污染。

**实施面**：

- 前端 `UserManagement/index.tsx` `renderUserId` 助手：值为 falsy 时返回 `'内置'`（替代之前的 `'-'`）
- 列表 `createdBy` / `updatedBy` 两列、查看 Drawer 的"创建人 / 更新人"两个 `Form.Item` 共享同一函数，自动一致

**未来回退**：若运营商要求严格区分"系统"与"内置"，可改回选项 A：基于 `selectedUser.source` 做三态映射，作为 v0.9 决议另开。

### 11.10 v0.9 — 下拉列表显示业务名字 + 全量端点规则

**问题**：编辑用户时，角色 `<Select mode="multiple">` 显示 UUID 而非角色名字。

根因：`adminApi.getAllRoles` 误调分页端点 `/admin/roles`（返回 `PageResponse<BackendRole>`），由 `Array.isArray(data)` 兜底返回 **空数组** → `roleOptions` 为空 → AntD `<Select>` 拿到 `value=UUID` 在 options 找不到对应 `label` → 退化显示原始 UUID。

后端实际**已实现** `GET /admin/roles/all`（`handler.go` `roles.GET("/all", h.ListAllRoles)`，不分页，专供下拉），与 PRD §6.2 标注一致。

**决议（本 PRD 范围）**：

| 维度 | 规则 |
|------|------|
| 下拉 `value` | 用资源唯一标识（UUID），后端通信契约保持稳定 |
| 下拉 `label` | 用业务名字（`roleName / displayName / username / groupName ...`），可包含 Tag 等装饰节点（如 §11.7 ① ⚠️ 标记） |
| Options 来源 | **必须**调用对应资源的「全量」端点（`getAllRoles` → `/admin/roles/all`），不得复用分页端点 |
| API 契约 | 全量端点返回**裸数组**（`BackendRole[]`），分页端点返回 `PageResponse<T>`；前端按 shape 区分 |
| 兜底 | options 异步加载阶段，若 value 不在 options 中，UI 应优先显示 `user.roles[name]` 等已有名字字段；不行才退到 UUID |

**全局推广（建议而非强制）**：本规则适用于所有"用户管理 / 角色管理 / 设备分组 / 字典 / 菜单 / API 管理 / 通知通道 ..." 的 `<Select>` `<Cascader>` `<TreeSelect>` 等组件，应抽到一份"前端 UI 通用规约"文档（`docs/project/frontend-conventions.md`，待立项）统一约束。本 PRD 仅落地用户管理范围内的修复。

**实施面**：

- 前端 `adminApi.getAllRoles`：URL 由 `/admin/roles` → `/admin/roles/all`
- 验证：`useAllRoles` 拿到全量 → `roleOptions` 完整 → 编辑用户 / 创建用户 / 批量分配角色 三处 `<Select>` 全部显示角色名字
- 注：列表页"角色"列（dataIndex `roles`）已经是名字数组（`mapBackendUser` 把 `bu.roles[].name` 取出），不受本 bug 影响

**已知边界**：

- `useAllRoles` 异步加载阶段（首次进入页面 + 缓存 miss），下拉 options 在数据到达前是空的，此时打开编辑 Drawer 会有毫秒级 UUID 闪烁 → React Query 默认 5min 缓存，二次打开不会出现
- 若运营商角色总数 > 1000 → `getAllRoles` 一次拉全集会有性能问题；当前规模可接受，未来若超阈值改 `getRoles({ page, pageSize })` + Select `<showSearch onSearch>` 异步搜索（v1.0+ 决议）

### 11.11 v1.0 — 删除 `users.carrier` + 超管走 `source = 'builtIn'`

**背景**：经 v0.10 的字段必要性分析，`users.carrier` 同时承载三重含义（业务标签 + 多租户 Casbin domain + 隐式超管标志 NULL）违反单一职责。同时 v0.4 已将"数据可见域"明确归 `role_device_groups` 决定，user 层的 carrier 在数据权限层属冗余双保险。

**决策**：

| 项 | 旧（v0.10 之前）| 新（v1.0）|
|----|----------------|----------|
| 超管判定 | `users.carrier IS NULL` 隐式 | `users.source = 'builtIn'` 显式 |
| 超管权限范围 | 仅数据可见域旁路 | **数据 + 菜单 + API 三层旁路**（等同 root）|
| 多 carrier 隔离 | Casbin domain 用 carrier | Casbin 去 domain（单域）；运营商隔离由 `role_device_groups` 通过设备分组层实现 |
| JWT Claims | `Carrier *string` | **新增 `IsSuperAdmin bool`，删除 `Carrier`** |
| `users.carrier` 列 | 存在（VARCHAR(4)）| **DROP COLUMN**（goose 迁移文件 `000NNN_drop_users_carrier.sql`）|
| `UserFilter.Carrier` | 用 carrier 过滤用户 | **删除**（按 role 隐式过滤）|
| `PermissionService.GetUserVisibleGroupIDs` 签名 | `(ctx, userID, carrier *CarrierCode)` | `(ctx, userID, isSuperAdmin bool)` |
| `LDAP.sync_carrier` 配置项 | LDAP 同步用户的 carrier 默认值 | **作废**：LDAP 用户 source='LDAP' 必非 builtIn，无升超管路径 |
| 前端 `User.carrier` | 字段存在 | **删除** |

**11 项决议（Q1-Q11）落定**：

1. **Q1 全权限范围**：builtIn = 数据可见域 + 菜单可见性 + API 鉴权 三层全旁路
2. **Q2 现存 `carrier=NULL` 非 admin 处理**：迁移前必须 audit `SELECT id, username, source FROM users WHERE carrier IS NULL`，业务逐个确认；预期 seed admin 已是 source='builtIn'，其他人保持 source='admin'（即降级为普通用户）
3. **Q3 Casbin domain**：完全去 domain，policy 统一在 'system' 单域；如未来真有多 carrier 需求另开决议
4. **Q4 JWT 签发 `is_super_admin`**：是；同时修复告警模块 latent bug（`alarm/data_permission.go:93` 早已读取该 claim 但 jwt 不签发，永远 false）
5. **Q5 PermissionService 签名**：传 `isSuperAdmin bool`；最小破坏；调用方在 ctx 解 claims 转换
6. **Q6 DROP COLUMN 时机**：分两步——本发布周期 Phase 2 仅停用代码引用（不删列），下个发布周期 Phase 5 才物理 DROP；中间观察期 1 周
7. **Q7 前端 `UserFilter.carrier`**：删除
8. **Q8 LDAP `sync_carrier`**：删除
9. **Q9 前端 `User.carrier`**：删除
10. **Q10 是否引入 `users.is_super_admin` 列**：**不加**；超管完全由 `source = 'builtIn'` 派生，避免双源不一致
11. **Q11 sqlc 实验代码**：清理（删 `internal/admin/sqlc/`、`queries/`、`sqlc.yaml`、`pg_user_repository_v2.go`、Makefile sqlc-* target）；与 CLAUDE.md「squirrel + pgx，不使用 ORM」立场对齐

**实施分 6 个 Phase**：

| Phase | 范围 | 风险 |
|-------|------|------|
| 1 | 文档先行（users.md / roles.md / system-config.md / README.md / ui-customization.md）| 🟢 |
| 2 | 后端代码改造（model / repo / service / jwt / middleware / casbin / permission_service / alarm / device + 单测）| 🟡 |
| 3 | 前端代码改造（types / adminApi / UserManagement page）| 🟢 |
| 4 | 数据 audit + 迁移文件草拟（不入主目录）| 🟡 |
| 5 | goose 迁移上线（DROP COLUMN，1 周观察）| 🔴 不可逆 |
| 6 | sqlc 实验代码清理 | 🟢 |

**关键约束**：

- DROP COLUMN 不可回滚（goose Down 仅恢复列定义，业务数据需备份恢复）
- Phase 2 完成后必须留至少 1 周观察期，验证所有功能正常，才能进 Phase 5
- LDAP 模块如已签发的 v0.9 spec 中含 `sync_carrier`，全部按本节作废
- alarm 模块的 `IsSuperAdmin` 行为变化：之前**永远 false**（latent bug），v1.0 后**仅 builtIn 用户为 true**，符合预期，但需通知运维

**风险与缓解**：

| 风险 | 缓解 |
|------|------|
| 现存"假超管"（手工建的 carrier=NULL 非 admin 用户）失去权限 | Phase 4 audit 报告 + 业务确认 |
| Casbin 多租户 policy 隔离失效 | 当前 seed 未用多 domain policy，影响小；如未来需要恢复，加 domain 字段 |
| 已签发 JWT 仍含 `Carrier` claim | go-jwt 默认忽略未识别字段；新签发不再包含；老 token 自然过期 |
| 30+ 处单测含 carrier mock | Phase 2G 系统化清理 |

**度量**：

- `omc_user_super_admin_count` Gauge：按 source 统计，验证超管数量在预期内
- `omc_perm_check_total{decision}` Counter：审计超管旁路次数（高峰可能成为攻击信号）

### 11.12 v1.1 — UI 文案统一 + 取消"导入用户"前端入口

#### 11.12.1 字段标签统一

**问题**：v1.0 之前用户管理 UI 的字段标签存在三处不一致：

| 字段 | 列表表头 | 创建表单 | 编辑表单 | 查看 Drawer |
|------|---------|---------|---------|-----------|
| `username` | 用户账号 | 用户名（i18n `user.userName`） | 用户账号 | 用户账号 |
| `displayName` | 用户名称 | 用户名称 | 用户名称 | 用户名称 |

用户调研反馈"用户名"和"用户名称"难以区分（前者像登录账号，后者更像别名 / nickname）。

**决议**：

- `username` 字段标签全局统一为 **"用户账号"**（语义：登录用、唯一、不可改）
- `displayName` 字段标签全局统一为 **"用户昵称"**（语义：展示用、可改、留空兜底为 username）
- 列表第 2 列 title 由"用户名"/"用户名称"改"用户昵称"
- 创建表单字段顺序与编辑表单同步，便于视觉迁移：`username → password → confirmPassword → displayName → email → phone → roleIds → status → expireTime → description`

**不在范围**：

- i18n key `user.userName` 等保留（其它页面可能仍引用），但 UserManagement 页面**不再使用** `t('user.userName')` 而是直接写中文常量"用户账号"
- 后端字段名（`username` / `display_name`）保持不变，仅 UI 标签改

#### 11.12.2 取消"导入用户"前端入口

**问题**：之前"添加用户" Drawer 内嵌 Radio 切换"添加 / 导入"两个模式：
- 用户体验：单一 Drawer 承载两种语义混乱
- ImportPanel 集成 ref + state（importing / createMode）增加心智负担
- 当前阶段批量导入需求弱，单条添加更高频

**决议**：

- 用户管理页面**取消**"导入用户"前端入口：
  - 移除 `<Radio.Group>` 模式切换
  - 移除 `<ImportPanel>` 集成（连带 `importPanelRef` / `importing` state / `createMode` state / 类型 `CreateMode` / `handleImport` / `handleImportSuccess` / `handleDownloadTemplate` 等辅助）
  - Drawer 标题简化为"添加用户"，Footer 按钮直接调 `handleCreate`
- 后端 `POST /admin/users/import` + `GET /admin/users/import/template` 端点能力**保留**（`internal/admin/user_import_handler.go` + 路由注册），作为**内部工具**（运维直接 curl / 内部 CLI 脚本）
- UI 与表单：与 §5.2 编辑表单字段顺序对齐，视觉一致性最大化

**未来回退**：若重新引入批量导入入口，应单独立项（独立按钮 / 独立菜单项 / 独立页面），不再寄生在"添加"Drawer 里 — 单一职责。

**实施面**：

- 前端 `UserManagement/index.tsx`：删除 `Radio.Group` 与 `<ImportPanel>` 段落；删除 `CreateMode` 类型、`createMode` / `setCreateMode` / `importing` / `setImporting` state、`importPanelRef` ref；删除 `handleImport` / `handleImportSuccess` / `handleDownloadTemplate` 三个 callback；Drawer footer 按钮简化
- 前端 `adminApi.importUsers / downloadImportTemplate` **保留**（内部可能继续调用）
- 不动后端端点

---

## 12. LDAP 集成方案（v0.9）

> 本章是「v0.9 决议 + 实施方案」合并体，独立成章便于实施时整体看完。
> 所有 LDAP 配置项**复用** [system-config.md `ldap` Tab](./system-config.md)，但本章对字段集与默认值做权威约定。

### 12.1 业务目标

| # | 目标 | 说明 |
|---|------|------|
| G1 | 统一登录 | 运营商/集团客户的 LDAP/AD 账号无需在 OMC 重新创建，直接登录 |
| G2 | 密码集中 | 密码完全归外部域管理；OMC 不存 / 不缓存 LDAP 用户密码 |
| G3 | 账号同步 | LDAP 加人 → OMC 自动可见；LDAP 删人 → OMC 自动禁用（不删，便于回滚）|
| G4 | 角色映射 | LDAP group `memberOf` → OMC role 自动绑定，避免手工维护 |
| G5 | 可观测 | 同步成功率、最近同步时间、失败明细在 UI 与 Prometheus 双通道可见 |

**与超管/管理员添加用户的关系**：

- LDAP 用户与 admin / builtIn 用户**同表共存**（`users`，按 `source` 区分）
- 一个 username 不可同时属于两个 source（unique 约束本来就保证）
- 登录入口同一个 `POST /auth/login`，service 内部按 `source` 分流（详见 §12.5）

### 12.2 配置项（`sys_configs` 表，category=`ldap`）

> 本表是配置项**权威集**。[system-config.md §3.2 LDAP](./system-config.md) 的字段示例需要按此扩展。

| key | value_type | 默认值 | 说明 |
|-----|-----------|-------|------|
| `enabled` | bool | `false` | 总开关；为 false 时所有 LDAP 行为关闭，`source=LDAP` 用户登录会拒绝 |
| `server_url` | string | — | `ldap://host:389` 或 `ldaps://host:636`（生产强制 ldaps）|
| `bind_dn` | string | — | 服务账号 DN，如 `CN=omc-bind,OU=service,DC=corp,DC=local` |
| `bind_password` | string（**加密存储**）| — | 服务账号密码 |
| `search_base` | string | — | 用户搜索基，如 `OU=users,DC=corp,DC=local` |
| `user_filter` | string | `(&(objectClass=user)(sAMAccountName=%s))` | 用户过滤器，`%s` 占位符替换为 username |
| `attr_email` | string | `mail` | LDAP 属性名 → `users.email` |
| `attr_phone` | string | `telephoneNumber` | → `users.phone` |
| `attr_displayname` | string | `displayName` | → `users.display_name` |
| `attr_groups` | string | `memberOf` | LDAP group DN 列表，用于 group→role 映射 |
| `sync_enabled` | bool | `true` | 是否启用周期同步（关闭后仅手动 / JIT）|
| `sync_interval_min` | int | `60` | 周期同步间隔（分钟）|
| ~~`sync_carrier`~~ | — | — | **v1.0 删除**：`users.carrier` 字段已不存在，且 LDAP 用户 `source='LDAP'` ≠ `'builtIn'`，永远不可能升超管 |
| `default_role_id` | uuid \| null | `null` | 新同步用户的兜底角色（无 group 映射时用）|
| `jit_provisioning_enabled` | bool | `false` | 即时入库：未在本地但 LDAP 中存在的账号首次登录时自动创建 |
| `tls_skip_verify` | bool | `false` | 仅测试环境用 |
| `connect_timeout_sec` | int | `5` | LDAP 连接超时 |
| `request_timeout_sec` | int | `10` | LDAP 操作超时 |
| `max_failed_login_in_omc` | int | `5` | OMC 端独立的密码错误锁定阈值（与 LDAP 端策略隔离，避免互锁）|

**敏感字段加密**：`bind_password` 在 service 层用 AES-256-GCM 加密入库；handler 层解密。加密 key 来自环境变量 `OMC_CONFIG_ENC_KEY`。

### 12.3 字段映射（LDAP attribute ↔ `users` 列）

| `users` 列 | LDAP attribute（默认）| 同步行为 | 备注 |
|-----------|---------------------|---------|------|
| `username` | sAMAccountName / uid | 主键，**首次创建后不可改** | 可配，由 `user_filter` 中的属性决定 |
| `email` | `mail`（可配 `attr_email`）| 同步覆盖 | LDAP 端为空时不清空本地值 |
| `phone` | `telephoneNumber`（可配 `attr_phone`）| 同步覆盖 | 同上 |
| `display_name` | `displayName`（可配 `attr_displayname`）| 同步覆盖 | LDAP 空 → 用 username 兜底 |
| ~~`carrier`~~ | — | — | **v1.0 删除**：users 表无此字段；LDAP 用户的"运营商范围"通过其绑定的角色 → role_device_groups 决定 |
| `status` | （LDAP 不再可见时）| 同步标记 `disabled`（不删除）| 见 §12.4 处理规则 |
| `source` | （固定）| `'LDAP'` | |
| `password_hash` | — | 写入哨兵值 `'__LDAP__'`（不可被 bcrypt 校验通过）| 防止任何路径误用本地密码校验 |
| `description` | （可选）| 不从 LDAP 同步，留给管理员手记 | |
| `expire_at` | （可选）| 不从 LDAP 同步 | LDAP 端 accountExpires 属性如需对接，列入 v1.0+ |
| `last_login_at` | — | 由登录流程更新 | |
| `created_by` / `updated_by` | NULL | 同步任务无 operator | 列表渲染显示"内置"（参 [v0.8 决议](#119-v08--列表创建人--更新人空值显示内置-而非)）|

**LDAP group → OMC role 映射**：单独表 `ldap_group_role_mappings`（详见 §12.6）。

### 12.4 同步流程

#### 12.4.1 周期同步（cron，运行在 worker 进程）

| 步骤 | 行为 |
|------|------|
| 1 | `cron` 触发（间隔 = `ldap.sync_interval_min`）；如 `ldap.enabled = false` 或 `ldap.sync_enabled = false` 立即返回 |
| 2 | bind LDAP server（用 `bind_dn` + 解密后的 `bind_password`）|
| 3 | search：base = `search_base`，filter = `(objectClass=user)`（去掉 `sAMAccountName=%s` 部分），attributes = 列出本次需要的字段 |
| 4 | 流式遍历返回的 entry list，与 `SELECT id, username, ... FROM users WHERE source='LDAP'` 做 diff |
| 5 | LDAP 中存在 + OMC 中不存在 → INSERT（source='LDAP', password_hash='`__LDAP__`'；**v1.0：不再写 carrier，字段已删**）|
| 6 | LDAP 中存在 + OMC 中存在 → UPDATE 字段（仅 `email/phone/display_name/updated_at`，**不动** `status` / `description` / 角色绑定）|
| 7 | LDAP 中不存在 + OMC 中存在（且 status=active） → UPDATE `status='disabled'` + 写 `audit_logs`（不删除）|
| 8 | 处理 group 变更（详见 §12.6）|
| 9 | 写 `ldap_sync_logs` 一条记录（含 created/updated/disabled/failed 计数 + 总耗时）|

**Schema：`ldap_sync_logs`**（新表，DDL 见 §12.8）：

| 列 | 类型 | 说明 |
|----|------|------|
| `id` | UUID | PK |
| `started_at` | TIMESTAMPTZ | 开始时间 |
| `finished_at` | TIMESTAMPTZ | 结束时间 |
| `triggered_by` | VARCHAR(16) | `cron` / `manual` / `jit` |
| `triggered_user_id` | UUID NULL | 手动触发时的操作人 |
| `result` | VARCHAR(16) | `success` / `failed` / `partial` |
| `created_count` | INT | 新建用户数 |
| `updated_count` | INT | 更新用户数 |
| `disabled_count` | INT | 标记禁用用户数 |
| `failed_count` | INT | 处理失败条数 |
| `error_summary` | TEXT NULL | 错误摘要（失败时填）|
| `details` | JSONB NULL | 完整明细（失败的 username + 错误，限 100 条避免膨胀）|

#### 12.4.2 手动同步

UI 入口：系统配置 → LDAP Tab → "立即同步"按钮。

| 步骤 | 行为 |
|------|------|
| 1 | 前端 `POST /admin/ldap/sync` |
| 2 | handler 立即返回 `{ "job_id": "uuid", "status": "running" }`，同步在 goroutine 后台执行 |
| 3 | 前端 polling `GET /admin/ldap/sync/status/{job_id}`（2 秒间隔）|
| 4 | 同步完成后任务状态置 `success/failed`，前端弹 toast 显示结果统计 |

**并发控制**：同时只允许一个 LDAP 同步任务运行（用 Redis 分布式锁 `lock:ldap:sync`，TTL = `connect_timeout_sec * 10`）。重复触发返回 409。

#### 12.4.3 JIT 即时同步（首次登录）

仅当 `jit_provisioning_enabled = true` 时启用：

| 步骤 | 行为 |
|------|------|
| 1 | 用户输入 username + password 登录，`GetByUsername` 返回 ErrNotFound |
| 2 | service 检查 `jit_provisioning_enabled`；为 true 则继续，否则返回 401 |
| 3 | bind LDAP，search 该 username，找到则按 §12.3 字段映射创建本地记录 |
| 4 | 创建后继续走 §12.5 LDAP bind 验证流程 |
| 5 | 在 `ldap_sync_logs` 写一条 `triggered_by='jit'` 记录（仅含该用户）|

**安全考量**：JIT 默认关闭。开启意味着任何 LDAP 中存在的账号都能直接登录，无需管理员预审；适用于完全信任 LDAP 域的场景。

### 12.5 登录流程（service 内部分流）

```
POST /auth/login {username, password}
  ↓
1. GetByUsername(username)
   ├─ ErrNotFound + jit_enabled=true → §12.4.3 JIT 路径
   ├─ ErrNotFound + jit_enabled=false → 返回 401 「user not found」
   └─ user 找到，继续
  ↓
2. 校验 user.status / user.locked_until / user.expire_at（统一逻辑，与 source 无关）
  ↓
3. switch user.source:
   ├─ 'admin' / 'builtIn' → bcrypt.CompareHashAndPassword(user.password_hash, password)
   └─ 'LDAP' →
        a. 检查 ldap.enabled = true，否则 503
        b. bind LDAP（用 bind_dn + bind_password 服务账号）
        c. search ldap.user_filter % username 拿到该用户的 DN
        d. **第二次 bind**：用拿到的 user DN + 用户输入的 password
        e. bind 成功 = 密码正确；失败 = 密码错误
        f. 失败计数走本地 `users.failed_login_attempts`（不依赖 LDAP，避免互锁）
  ↓
4. 成功 → 生成 token + UpdateLastLogin（与 admin 路径一致）
```

**关键约束**：

- LDAP 用户的 `password_hash` 入库为 `'__LDAP__'` 哨兵值，确保即便误调 `bcrypt.CompareHashAndPassword` 也必然失败（哨兵值不可能匹配任何密码）
- 第二次 bind 后**立即** unbind，不长连接（避免 LDAP 端连接资源耗尽）
- LDAP 服务器宕机：登录返回 `503 Service Unavailable` + 明确错误体 `{ "error": "LDAP server unavailable", "retry_after_sec": 30 }`
- 登录失败次数：与本地一致用 `users.failed_login_attempts`，达到 `max_failed_login_in_omc` 时本地锁定（不试图改 LDAP 端密码错误次数）

### 12.6 LDAP group → OMC role 映射

#### 12.6.1 表 `ldap_group_role_mappings`（DDL 见 §12.8）

| 列 | 类型 | 说明 |
|----|------|------|
| `id` | UUID | PK |
| `ldap_group_dn` | VARCHAR(512) | LDAP group 完整 DN，如 `CN=omc-ops,OU=groups,DC=corp,DC=local`，UNIQUE |
| `role_id` | UUID | FK → `roles(id)` ON DELETE CASCADE |
| `description` | TEXT | NULL |
| `created_by` / `updated_by` | UUID | NULL |
| `created_at` / `updated_at` | TIMESTAMPTZ | NOT NULL |

#### 12.6.2 同步时的 role 计算

每个用户同步时：

1. 从 LDAP entry 取 `attr_groups`（默认 `memberOf`）拿到 group DN 数组
2. 查 `ldap_group_role_mappings` 命中的 `role_id` 集合
3. 对该用户当前 `user_roles`（仅 source='LDAP' 来源的角色，标记需新增字段 `assigned_by_ldap BOOLEAN`）做差量同步：
   - 新命中 → INSERT user_roles，标 `assigned_by_ldap=true`
   - 已存在但本次未命中 → DELETE
   - 管理员手动加的角色（`assigned_by_ldap=false`）**不动**

**Schema 调整**：`user_roles` 表加 `assigned_by_ldap BOOLEAN NOT NULL DEFAULT false`（迁移文件示例见 §12.8）。

#### 12.6.3 兜底角色

若用户的 LDAP groups 没有任何映射命中：

- 若 `ldap.default_role_id` 非 NULL → 绑定该角色（`assigned_by_ldap=true`）
- 若也未配置 → 用户登录后**无任何角色**（无设备数据可见、无菜单可见，对齐 [roles.md §11.2 决议 ①](./roles.md) 的提示规则）

### 12.7 异常处理

| 场景 | 行为 | 用户可见性 |
|------|------|----------|
| LDAP 服务器宕机 | 登录返回 503；周期同步任务 retry 3 次后中止本轮，记 `ldap_sync_logs.result='failed'` | UI 顶部 banner 显示「LDAP 不可用」，持续到下次同步成功 |
| `bind_dn` 凭据失效 | 同上，error message 区分（`auth_failed` vs `unreachable`）| 系统配置 LDAP Tab 加红色提示 |
| 用户 password 错误 | 走本地 `failed_login_attempts` + `lockout_duration` 锁定 | 与 admin 用户一致 |
| LDAP 删用户但本地未同步 | 用户仍可登录（直到下次同步标记 disabled）| 接受该窗口；缩短 `sync_interval_min` 可降低风险 |
| LDAP 同步用户 `attr_groups` 解析失败 | 该用户 group 映射跳过，保留旧角色 + 写入 `ldap_sync_logs.details` | 管理员可在同步历史查看 |
| TLS 证书校验失败 | 同步与登录均失败；除非 `tls_skip_verify=true` | 强烈建议生产不开 skip_verify |
| 同步任务正在运行时再次触发 | 返回 409 `{ "error": "sync already in progress" }` | UI toast 提示 |
| `default_role_id` 指向已删除的 role | 同步时检测到 → 写 `ldap_sync_logs.error_summary` + 跳过该用户的兜底绑定 | 管理员需在配置页修复 |

### 12.8 接口契约

> Base URL：`/api/v1/admin`（admin 鉴权）

#### 12.8.1 LDAP 同步管理

| Method | 路径 | 说明 |
|--------|------|------|
| POST | `/ldap/sync` | 触发手动同步，返回 `{ "job_id": "uuid", "status": "running" }`，409 if 已有进行中任务 |
| GET | `/ldap/sync/status/{job_id}` | 查同步进度，返回 `ldap_sync_logs` 一条记录 |
| GET | `/ldap/sync/history?page=&pageSize=` | 同步历史分页 |
| POST | `/ldap/test-connection` | 仅 bind 测试，验证 server_url + bind_dn + bind_password；返回 `{ "ok": true }` 或 `{ "ok": false, "error": "..." }` |

#### 12.8.2 LDAP group→role 映射 CRUD

| Method | 路径 | 说明 |
|--------|------|------|
| GET | `/ldap/group-mappings?page=&pageSize=` | 列表 |
| POST | `/ldap/group-mappings` | 新增（Body：`{ "ldap_group_dn": "...", "role_id": "uuid", "description": "..." }`）|
| PUT | `/ldap/group-mappings/{id}` | 编辑 |
| DELETE | `/ldap/group-mappings/{id}` | 删除 |

#### 12.8.3 现有接口的 LDAP 适配

| 接口 | 适配点 |
|------|--------|
| `POST /auth/login` | service 内部按 `source` 分流（详见 §12.5），无需新端点 |
| `POST /admin/users/{id}/reset-password` | `source='LDAP'` 返回 409（v0.2 已实现）|
| `POST /admin/users` | 不接受 `source` 字段，固定 `'admin'`，无法通过本接口创建 LDAP 用户（必须走同步）|
| `DELETE /admin/users/{id}` | LDAP 用户允许删除（仅删本地，不影响 LDAP 源）|
| `PUT /admin/users/{id}` | `source='LDAP'` 用户的 `email/phone/display_name` 字段被覆盖时**写本地**，但下次 LDAP 同步会再次覆盖。建议前端 banner 提示「LDAP 用户的部分字段以 LDAP 端为准」|

### 12.9 数据库迁移（DDL 示例）

迁移文件版本号按发布时实际为准。下面给出参考 SQL。

```sql
-- +goose Up
-- 1. user_roles 加来源标记
ALTER TABLE user_roles
    ADD COLUMN IF NOT EXISTS assigned_by_ldap BOOLEAN NOT NULL DEFAULT FALSE;

-- 2. ldap_group_role_mappings
CREATE TABLE IF NOT EXISTS ldap_group_role_mappings (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ldap_group_dn VARCHAR(512) NOT NULL UNIQUE,
    role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    description   TEXT,
    created_by    UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    updated_by    UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_ldap_group_mappings_role_id ON ldap_group_role_mappings(role_id);

-- 3. ldap_sync_logs
CREATE TABLE IF NOT EXISTS ldap_sync_logs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    started_at        TIMESTAMPTZ NOT NULL,
    finished_at       TIMESTAMPTZ NULL,
    triggered_by      VARCHAR(16) NOT NULL CHECK (triggered_by IN ('cron','manual','jit')),
    triggered_user_id UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    result            VARCHAR(16) NOT NULL CHECK (result IN ('running','success','failed','partial')),
    created_count     INT NOT NULL DEFAULT 0,
    updated_count     INT NOT NULL DEFAULT 0,
    disabled_count    INT NOT NULL DEFAULT 0,
    failed_count      INT NOT NULL DEFAULT 0,
    error_summary     TEXT NULL,
    details           JSONB NULL
);
CREATE INDEX IF NOT EXISTS idx_ldap_sync_logs_started_at ON ldap_sync_logs(started_at DESC);

-- 4. seed sys_configs ldap.* 默认值
INSERT INTO sys_configs (category, key, value, value_type, description, is_public) VALUES
    ('ldap', 'enabled', 'false', 'bool', 'LDAP 总开关', FALSE),
    ('ldap', 'sync_enabled', 'true', 'bool', '启用周期同步', FALSE),
    ('ldap', 'sync_interval_min', '60', 'int', '周期同步间隔（分钟）', FALSE),
    ('ldap', 'user_filter', '(&(objectClass=user)(sAMAccountName=%s))', 'string', '用户过滤器，%s 替换为 username', FALSE),
    ('ldap', 'attr_email', 'mail', 'string', 'LDAP 属性名 → email', FALSE),
    ('ldap', 'attr_phone', 'telephoneNumber', 'string', 'LDAP 属性名 → phone', FALSE),
    ('ldap', 'attr_displayname', 'displayName', 'string', 'LDAP 属性名 → display_name', FALSE),
    ('ldap', 'attr_groups', 'memberOf', 'string', 'LDAP 属性名 → groups DN list', FALSE),
    ('ldap', 'jit_provisioning_enabled', 'false', 'bool', '即时入库（首登自动创建）', FALSE),
    ('ldap', 'tls_skip_verify', 'false', 'bool', '跳过 TLS 证书校验（仅测试）', FALSE),
    ('ldap', 'connect_timeout_sec', '5', 'int', 'LDAP 连接超时（秒）', FALSE),
    ('ldap', 'request_timeout_sec', '10', 'int', 'LDAP 操作超时（秒）', FALSE),
    ('ldap', 'max_failed_login_in_omc', '5', 'int', 'OMC 端独立的密码错误锁定阈值', FALSE)
ON CONFLICT (category, key) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS ldap_sync_logs;
DROP TABLE IF EXISTS ldap_group_role_mappings;
ALTER TABLE user_roles DROP COLUMN IF EXISTS assigned_by_ldap;
DELETE FROM sys_configs WHERE category = 'ldap';
```

### 12.10 后端代码组织

新模块路径：`internal/ldap/`（独立于 `internal/admin/`，避免 admin 包过大）。

```
internal/ldap/
├── client.go              # go-ldap/v3 wrapper：bind / search / unbind 封装
├── client_test.go         # 单元测试用 mock LDAP server (gldap 库)
├── config.go              # sys_configs 读取 + bind_password 解密
├── sync_service.go        # SyncOnce(ctx, triggeredBy) → *SyncResult
├── sync_service_test.go
├── login_service.go       # AuthenticateLDAP(ctx, username, password) → (*User, error)
├── group_mapping_repo.go  # ldap_group_role_mappings CRUD
├── sync_log_repo.go       # ldap_sync_logs CRUD
├── handler.go             # /ldap/sync, /ldap/test-connection 等
├── group_handler.go       # /ldap/group-mappings CRUD
└── cron.go                # worker 进程注册的周期任务入口
```

**依赖**：

| 库 | 用途 |
|----|------|
| `github.com/go-ldap/ldap/v3` | LDAP 客户端 |
| `github.com/jimlambrt/gldap`（仅 test） | mock LDAP server |
| 现有 `internal/admin` | 复用 `userRepo` / `roleRepo` |
| 现有 `internal/core/components/redis` | 分布式锁 + 缓存 |

**集成点**：

- `cmd/worker/main.go`：启动时调 `ldap.RegisterCron(scheduler, deps)`
- `cmd/app/provider/router.go`：挂 `ldapHandler.RegisterRoutes(adminGroup)`
- `internal/admin/service.go::Login`：分流到 `ldap.AuthenticateLDAP` 当 `user.Source == UserSourceLDAP`

### 12.11 后端补齐 Backlog

#### P0（与 v0.9 决议同时落地）

1. **DDL 迁移**（§12.9）落地
2. **配置项 seed**（§12.9 INSERT 部分）
3. **`internal/ldap/` 模块基线**：client.go / config.go / sync_service.go / login_service.go
4. **`AdminService.Login` 分流**：按 `user.Source` 走 LDAP 或 bcrypt
5. **worker cron 注册**：`cmd/worker/main.go` 调 `ldap.RegisterCron`
6. **`POST /admin/users` 拒绝** `source` 字段，已在 v0.2 落地（确认仍生效）
7. **bind_password 加密存储 / 解密**：service 层用 AES-256-GCM，key 从环境变量

#### P1

8. **LDAP 同步 UI**：系统配置 LDAP Tab 加 「立即同步」 按钮 + 同步历史 tab + 测试连接按钮
9. **LDAP group→role 映射页面**：新建 `pages/system/LdapGroupMappings/index.tsx` + 路由
10. **失败告警**：周期同步连续失败 3 次 → 写 `notifications` 表（消息中心）通知所有 admin 角色用户；Prometheus metric `omc_ldap_sync_failed_total`
11. **测试连接接口** `/ldap/test-connection`
12. **JIT provisioning** 实现（默认配置开关关闭）

#### P2

13. **多 LDAP 数据源**：支持配置多个 LDAP server，`users.source` 细分为 `LDAP:<server-id>`
14. **LDAP 用户密码到期提醒**：解析 LDAP `pwdLastSet` / `accountExpires` → 登录时 banner 提示
15. **LDAP 嵌套 group**：`memberOf` 不能直接拿到嵌套关系，需要递归查 `member` 反查 → 性能优化
16. **同步预演（dry-run）**：手动同步前可选择"仅预览不写库"

### 12.12 验收清单（DoD）

#### 后端

- [ ] §12.9 DDL 迁移落地，版本号连续，Down 完整
- [ ] `internal/ldap/` 模块单测覆盖：bind / search / sync diff 三类核心逻辑
- [ ] worker 进程启动后周期任务运行（日志可见）
- [ ] `POST /admin/users` 提交 `source: 'LDAP'` 时被忽略，固定 `'admin'`
- [ ] LDAP 服务器宕机时 `/auth/login` 对 source='LDAP' 用户返回 503，对 admin 用户不受影响
- [ ] `bind_password` 在 DB 中是密文，`GET /admin/configs?category=ldap` 响应中字段被脱敏（返回 `'****'`）
- [ ] `ldap_sync_logs` 一周内的记录可查
- [ ] 同步任务并发锁生效（重复触发返 409）
- [ ] 单元测试 mock LDAP server 覆盖：成功 bind / 错误密码 / 服务器超时 / 用户不存在 / TLS 失败 5 类场景
- [ ] Prometheus 指标暴露：`omc_ldap_sync_total{result}`、`omc_ldap_sync_duration_seconds`、`omc_ldap_login_total{result}`、`omc_ldap_failed_total{reason}`

#### 前端

- [ ] 系统配置 LDAP Tab 字段集与 §12.2 对齐（含 13 个配置项）
- [ ] 「立即同步」按钮 + 进度 polling + 结果 toast
- [ ] LDAP 同步历史页（最近 30 天，含 created/updated/disabled/failed 计数 + error_summary）
- [ ] LDAP group→role 映射 CRUD 页面
- [ ] 用户列表 `source` 过滤器加 `LDAP` 选项
- [ ] LDAP 用户的「编辑用户」表单顶部 banner：「该用户由 LDAP 同步管理，邮箱/手机/姓名等字段会被下次同步覆盖」
- [ ] LDAP 用户的「重置密码」按钮置灰（v0.2 已实现，本节确认）
- [ ] `npm run typecheck` & `lint` 通过

#### 文档

- [ ] [system-config.md §3.2 LDAP](./system-config.md) 字段集按 §12.2 扩展（13 个键）
- [ ] [roles.md](./roles.md) 在 §1 边界声明：`ldap_group_role_mappings` 由本 PRD 承担，roles.md 不重复描述
- [ ] [README.md 索引](./README.md) 在 P0 列表中加入 LDAP 集成相关条目

#### 部署

- [ ] `OMC_CONFIG_ENC_KEY` 环境变量在所有部署 yaml 中存在；缺失时 worker 启动失败
- [ ] 默认 `ldap.enabled = false`，新部署不影响现有功能
- [ ] 升级文档说明：开启 LDAP 前必须配置完整 §12.2 字段并 `test-connection` 通过

### 12.13 v1.0+ 候选（待澄清事项）

- [ ] 多个 LDAP 数据源场景下，`source = 'LDAP:<id>'` 细分（与 [users.md §11.4](#114-待澄清事项) 联动）
- [ ] LDAP 嵌套 group 性能优化（recursive `member` 查询）
- [ ] OMC 端的 RBAC 角色变更是否反向同步到 LDAP（默认不做，避免改动外部域）
- [x] ~~LDAP 用户能否升超管（修改 carrier=NULL）~~ → **v1.0 已闭环**：source='LDAP' ≠ 'builtIn'，超管身份无法获取，UI 也无可改字段
