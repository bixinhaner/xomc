# 系统管理 — 菜单管理（System / Menus）PRD

> 文档目的：作为前后端开发对齐的唯一事实源（Single Source of Truth）。
> 与 [roles.md](./roles.md) 配套：本 PRD 管菜单本身的 CRUD，角色 ↔ 菜单绑定 (`role_menus`) 在 roles.md。

| 版本 | 日期 | 作者 | 备注 |
|------|------|------|------|
| 0.1  | 2026-05-06 | Backend/Frontend Team | 基于现网 `MenuManagement/index.tsx` + `internal/admin/menu_handler.go` 抽取 |

**关联功能域**：F06 OMC-R 核心 / RBAC

**相关文件**：

| 层 | 路径 |
|----|------|
| 前端页面 | [omcmb/webcode/src/pages/system/MenuManagement/index.tsx](../../../../omcmb/webcode/src/pages/system/MenuManagement/index.tsx) |
| 后端 Handler | [omcgo/internal/admin/menu_handler.go](../../../internal/admin/menu_handler.go)（推断路径，搜索 `func.*ListMenus`）|
| 数据库 | [omcgo/migrations/000009_sys_admin.sql](../../../migrations/000009_sys_admin.sql)（`menus` 表） |
| seed | [omcgo/migrations/000009_sys_admin.sql](../../../migrations/000009_sys_admin.sql)（行 220-254：导航菜单种子）|

---

## 1. 业务背景

菜单（Menu）定义系统左侧导航树 + 按钮级权限标识，是 RBAC 的"前端可见性"载体：

| 类型 | 说明 | 示例 |
|------|------|------|
| `directory` | 一级目录 | "系统管理"、"性能管理" |
| `menu` | 二级菜单（带路由）| "用户管理"（`/system/users`）|
| `button` | 按钮级权限点（无路由） | "用户:删除"（`system:user:delete`）|

**与角色权限的关系**：
- `role_menus` 关联表存储「角色 ↔ 可见菜单」（参 [roles.md §6.2](./roles.md)）
- 用户登录后，`GET /admin/menus/user-tree` 返回**该用户实际可见的菜单子树**，前端用于渲染导航 + 路由守卫
- 按钮级菜单决定页面内细粒度操作的 `disabled` 状态

---

## 2. 实体模型

### 2.1 `menus` 表（[migrations/000009_sys_admin.sql:11-46](../../../migrations/000009_sys_admin.sql#L11-L46)）

| 列 | 类型 | 约束 | 说明 |
|----|------|------|------|
| `id` | UUID | PK | |
| `name` | VARCHAR(64) | NOT NULL | 菜单名称（中文展示） |
| `type` | VARCHAR(16) | NOT NULL, CHECK IN (`directory`,`menu`,`button`) | |
| `permission_key` | VARCHAR(128) | NOT NULL, UNIQUE | 权限标识，形如 `system:user:add` |
| `parent_id` | UUID | NULL, FK → `menus(id)` ON DELETE CASCADE | 父菜单（NULL = 一级目录） |
| `sort_order` | INT | NOT NULL DEFAULT 0 | 同级排序 |
| `route_path` | VARCHAR(256) | NULL | 路由地址（仅 `menu` 类型有） |
| `component_path` | VARCHAR(256) | NULL | 组件路径（前端动态加载用） |
| `icon` | VARCHAR(64) | NULL | Ant Design 图标名（如 `UserOutlined`）|
| `route_params` | JSONB | NULL | 路由附带参数 |
| `is_external` | BOOLEAN | NOT NULL DEFAULT FALSE | 是否外链 |
| `show_status` | VARCHAR(16) | NOT NULL DEFAULT 'show' | `show` / `hide`（隐藏菜单仍可访问，不显示在导航）|
| `status` | VARCHAR(16) | NOT NULL DEFAULT 'normal' | `normal` / `disabled` |
| `api_permission` | VARCHAR(16) | NOT NULL DEFAULT 'none' | `required` / `none`（菜单是否需要 API 端点权限校验）|
| `created_by` / `updated_by` | UUID | NULL | 操作者审计 |
| `created_at` / `updated_at` | TIMESTAMPTZ | NOT NULL | |

唯一约束：`uniq_menu_name_per_parent (name, COALESCE(parent_id, ...))`

### 2.2 关联表 `role_menus`

| 列 | 类型 | 约束 |
|----|------|------|
| `role_id` | UUID | NOT NULL, FK → `roles(id)` ON DELETE CASCADE |
| `menu_id` | UUID | NOT NULL, FK → `menus(id)` ON DELETE CASCADE |
| PRIMARY KEY | (`role_id`, `menu_id`) | |

### 2.3 前端类型（[frontend-core/src/types/system.ts](../../../../omcmb/frontend-core/src/types/system.ts)）

```ts
type MenuType = 'menu' | 'button' | 'link';   // ⚠️ 与后端 'directory'/'menu'/'button' 不一致
type MenuStatus = 'active' | 'disabled';      // ⚠️ 与后端 'normal'/'disabled' 不一致

interface MenuItem {
  id: string;
  name: string;
  title: string;        // ⚠️ 后端无 title 字段
  icon?: string;
  path?: string;        // 对应后端 route_path
  component?: string;   // 对应后端 component_path
  type: MenuType;
  parentId?: string;
  sortOrder: number;
  status: MenuStatus;
  visible: boolean;     // ⚠️ 后端是 show_status 字符串
  children?: MenuItem[];
  createdAt?: string;
  updatedAt?: string;
}
```

### 2.4 前后端字段映射 & Gap

| 前端字段 | 后端字段 | 状态 |
|---------|---------|------|
| `id` | `id` | ✅ |
| `name` | `name` | ✅ |
| `title` | — | ⚠️ 前端独立字段，建议后端补 `title VARCHAR(128)`（用于多语言/标题与权限名分开）|
| `path` | `route_path` | ✅（snake↔camel 自动转换）|
| `component` | `component_path` | ✅ |
| `type` | `type` | ⚠️ **枚举值不一致**：前端 `menu/button/link`，后端 `directory/menu/button`。需对齐：取后端枚举为准 |
| `parentId` | `parent_id` | ✅ |
| `sortOrder` | `sort_order` | ✅ |
| `status` | `status` | ⚠️ **枚举值不一致**：前端 `active/disabled`，后端 `normal/disabled`。需对齐 |
| `visible` (bool) | `show_status` (`show`/`hide`) | ⚠️ 类型与枚举均不一致 |
| `permissionKey` | `permission_key` | ✅ |
| `apiPermission` | `api_permission` | ✅（`required`/`none`）|

---

## 3. 列表页（树形）字段定义

> 表格用**树形展示**（不分页），通过 `parent_id` 自连接重组为树。

### 3.1 列定义

| key | 列标题 | dataIndex | UI 渲染 | 备注 |
|-----|-------|-----------|--------|------|
| `name` | 菜单名称 | `name` | 缩进展示（按层级深度）| |
| `type` | 类型 | `type` | `<Tag>`：`directory` 蓝 / `menu` 绿 / `button` 默认 | |
| `sortOrder` | 排序 | `sortOrder` | 数字 + ↑↓ 按钮（点击立即 PUT 更新） | |
| `permissionKey` | 权限标识 | `permissionKey` | monospace | |
| `componentPath` | 组件路径 | `componentPath` | monospace，`ellipsis` | 仅 `menu` 类型有 |
| `status` | 状态 | `status` | `<Tag>`：`normal` 绿 / `disabled` 红 | |
| `actions` | 操作 | — | 编辑 / 更多（删除）| |

### 3.2 顶部筛选

| 字段 | 标签 | UI 组件 |
|------|------|--------|
| `name` | 菜单名称 | `<Input>` |
| `status` | 状态 | `<Select>` 全部 / normal / disabled |

### 3.3 顶部按钮

| 按钮 | 行为 |
|------|------|
| 添加菜单 | 打开创建 Drawer（默认顶级目录，可在表单选父级）|

---

## 4. 操作清单

| 操作 | 触发 | 接口 |
|------|------|------|
| 添加 | 顶部按钮 / 行内"添加子节点" | `POST /admin/menus` |
| 编辑 | 行内"编辑" | `PUT /admin/menus/{id}` |
| 删除 | 行内"更多 → 删除"（含子树确认）| `DELETE /admin/menus`（Body：`{ "ids": ["uuid"] }`，CASCADE 删子）|
| 排序调整 | `↑` / `↓` 按钮 | `PUT /admin/menus/{id}` 仅更新 `sort_order` |

> ⚠️ 当前前端 `MenuManagement` **完全用本地 mock 数据**（`MOCK_MENUS` 常量），未对接后端。这是 P0 阻塞项。

---

## 5. 表单字段定义（创建 / 编辑 Drawer）

> 字段按 `type` 动态显示。

### 5.1 公共字段（所有类型）

| name | 标签 | UI 组件 | 必填 | 校验 |
|------|------|--------|------|------|
| `type` | 菜单类型 | `<Radio.Group>` directory / menu / button | ✅ | 创建后不可改 |
| `parentId` | 上级菜单 | `<TreeSelect>` | type=`directory` 时无 | options 来自菜单树（仅 `directory` / `menu` 可作父）|
| `name` | 菜单名称 | `<Input maxLength=64>` | ✅ | |
| `permissionKey` | 权限标识 | `<Input maxLength=128>` | ✅ | unique；建议格式 `<domain>:<resource>:<action>` |
| `sortOrder` | 显示排序 | `<InputNumber min=0>` | — | 默认 0 |
| `status` | 菜单状态 | `<Radio.Group>` normal / disabled | — | 默认 normal |

### 5.2 directory / menu 类型额外字段

| name | 标签 | UI 组件 | 必填 |
|------|------|--------|------|
| `icon` | 菜单图标 | `<Input>` 或 IconPicker | — |
| `showStatus` | 显示状态 | `<Radio.Group>` show / hide | — |
| `apiPermission` | API 权限校验 | `<Radio.Group>` required / none | — |

### 5.3 menu 类型额外字段

| name | 标签 | UI 组件 | 必填 | 校验 |
|------|------|--------|------|------|
| `routePath` | 路由地址 | `<Input>` | ✅ | 以 `/` 开头 |
| `componentPath` | 组件路径 | `<Input>` | ✅ | 形如 `system/UserManagement` |
| `isExternal` | 是否外链 | `<Radio.Group>` yes / no | — | 默认 no |
| `routeParams` | 路由参数 | `<Input.TextArea>` JSON | — | |

---

## 6. 接口契约

> Base URL：`/api/v1/admin`

### 6.1 已实现（[handler.go:215-224](../../../internal/admin/handler.go#L215-L224)）

| Method | 路径 | 说明 |
|--------|------|------|
| GET | `/menus` | 列表（扁平）|
| GET | `/menus/tree` | 树形（构建好层级，前端 Tree 直接用）|
| GET | `/menus/user-tree` | **当前登录用户**可见的菜单树（按角色过滤后）|
| GET | `/menus/{id}` | 单菜单详情 |
| POST | `/menus` | 创建 |
| PUT | `/menus/{id}` | 编辑 |
| DELETE | `/menus` | 批量删除（Body：`{ "ids": [...] }`）|

### 6.2 用户菜单树（前端导航的数据源）

| Method | 路径 | 说明 |
|--------|------|------|
| GET | `/auth/menus` | 登录后获取当前用户菜单（无需 admin 权限），见 [router.go:241](../../../cmd/app/provider/router.go#L241) |

---

## 7. 后端补齐 Backlog

### P0（阻塞前端 UI）
1. **前端对接真实 API**：`MenuManagement/index.tsx` 当前完全用 `MOCK_MENUS`，需替换为 `useMenuTree()` / `useCreateMenu()` / `useUpdateMenu()` / `useDeleteMenus()` Hook，并新建 `menuApi.ts` 封装 §6.1 的端点
2. **枚举值对齐**（§2.4 已记）：
   - `type`：前端 `link` 改为 `directory`，与后端一致
   - `status`：前端 `active` 改为 `normal`
   - `visible: boolean` → `showStatus: 'show' | 'hide'`

### P1
3. **`title` 字段**：后端 `menus` 表加 `title VARCHAR(128)`，区分"权限名"与"展示标题"，并支持 i18n
4. **路径循环检测**：创建/编辑时校验 `parent_id` 不能形成环（A → B → A）
5. **删除前置检查**：删除菜单前查 `role_menus` 是否仍有引用，提示影响的角色数量
6. **菜单变更联动缓存**：`PUT /menus/{id}` / `DELETE /menus` 需失效所有关联角色下用户的 `auth.menus` 缓存（如有）

### P2
7. **批量导入/导出**：菜单结构是部署期产物，支持 JSON 导入导出，便于跨环境同步
8. **拖拽排序**：树形 Tree 支持节点拖拽重排，一次提交多条 `sort_order` 变更

---

## 8. 非目标

- 菜单的多语言（i18n）— 当前只有中文 `name`，多语言走前端 i18n 字典而非 DB
- 动态权限（运行时新增菜单类型）— 类型固定为 `directory/menu/button` 三种

---

## 9. 验收清单（DoD）

后端：
- [ ] §6.1 全部接口单测覆盖
- [ ] 删除菜单时 CASCADE 删 `role_menus` 关联（已有外键 ON DELETE CASCADE 兜底，但需确认 audit 写入完整）
- [ ] 父子环检测覆盖

前端：
- [ ] 删除 `MOCK_MENUS`，改用真实 API
- [ ] 类型/状态枚举与后端对齐
- [ ] 排序按钮调用 `PUT /menus/{id}` 实时持久化（不再仅本地 setState）
- [ ] `npm run typecheck` & `lint` 通过
- [ ] Playwright E2E：新建菜单 → 分配给角色 → 该角色用户登录后能看到该菜单
