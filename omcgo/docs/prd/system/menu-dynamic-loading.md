# 系统管理 — 菜单动态加载与权限联动 技术方案

> 文档目的：把当前**前端硬编码菜单 + 写死路由**改造为**后端驱动 + 角色权限联动**。
> 与 [menus.md](./menus.md) 配套：menus.md 管菜单本身的 CRUD（已就位）；本文管菜单消费侧（Sidebar 渲染、动态路由、按钮级权限、图标可配）。
> 本文为技术方案，**等审核确认后再分阶段实施**。

| 版本 | 日期 | 作者 | 备注 |
|------|------|------|------|
| 0.1  | 2026-05-08 | Backend/Frontend Team | 初稿，待 review |
| 0.2  | 2026-05-08 | Backend/Frontend Team | 7 项开放决策定稿；补一级菜单图标尺寸实测 |
| 0.3  | 2026-05-08 | Backend/Frontend Team | 附录 A 100 图标白名单冻结；P0 启动前最终版 |
| 0.4  | 2026-05-08 | Backend Team | §4.2.4 修订：B3 端点级方案 + 双段实施（Phase1 准备 / Phase2 切换）|
| 0.5  | 2026-05-08 | Backend/Frontend Team | B3-Phase2-B 完成 (DROP permissions) + 前端 P1/P2/P3 全部落地；§4.2.4 / §5 / §7 状态收尾 |
| 0.6  | 2026-05-08 | Frontend Team | v0.5 复核收口：adminApi.getUserMenuTree 删除 + logout 清 menuStore + Forbidden §4.3.5 文案；新增 §10 待办（buildRouter 完全动态化 / e2e / 灰度切换） |

**关联功能域**：F06 OMC-R 核心 / RBAC

**关联 PRD**：[menus.md](./menus.md) v0.1（菜单 CRUD 现状基线）、[roles.md](./roles.md)（角色 ↔ 菜单绑定）

---

## 1. 业务背景与诉求

### 1.1 用户原始诉求

1. 一级菜单图标当前写死在前端代码，需要在 `/system/menus` 菜单管理页配置。
2. 动态加载菜单：根据登录用户的角色，只展示其有权限的菜单，而非现在所有菜单都写死渲染。
3. 分析当前菜单管理的数据，给出"实现动态加载"还缺什么数据/功能/接口的清单。
4. 输出技术方案，先审核再实施。

### 1.2 业务目的

- **运营商定制**：交付到不同局点时，运营商希望按需开关功能模块、调整图标/名称，不应每次重打前端包。
- **租户/角色差异**：不同角色看见的菜单数量差异极大（admin 全量、operator 仅运维相关、viewer 只读视图），目前无论谁登录都看见**完整 NAV_CONFIG**——视觉冗余、且暴露未授权入口（即使路由层 401，也是糟糕体验）。
- **菜单即权限的契约**：当前权限模型分散（前端 PERMISSION_MODULES 字符串数组 / 后端 permissions 表 / role_menus 表三套并存），落地"菜单即权限"统一模型可显著简化角色管理。

### 1.3 现状测量：一级菜单图标尺寸（决定 IconPicker 设计）

Playwright 实测当前部署的 dist（[Sidebar.module.css 第 109/150/240/246 行](../../../../omcmb/webcode/src/components/Layout/Sidebar/Sidebar.module.css)）：

| 状态 | icon font-size | 实际渲染（W × H） | 容器 item 高度 | 容器内边距 |
|------|---------------|------------------|---------------|-----------|
| 侧边栏垂直展开（默认）| 16px | 16 × 16 px | 40 px | padding-left 12px |
| 侧边栏折叠（仅图标）| 16px | 16 × 16 px | 40 px（icon 居中）| 0 |
| 顶部水平菜单（top mode）| 16px | 16 × 16 px | 48 px | padding-inline 16px |

**结论与 IconPicker 设计约束**：

1. 三种模式下图标统一 **16px × 16px**——IconPicker 预览必须按 16px 渲染（避免选择器里图标看着大、Sidebar 里看着小的视觉差）。
2. 16px 下细节有限——白名单**只取 antd 的 Outlined 风格**（Filled/TwoTone 在 16px 下视觉偏厚或对比度异常）；这与现有 [NavMenu.tsx 的 `ICON_MAP` 25 个图标](../../../../omcmb/webcode/src/components/Layout/Sidebar/NavMenu.tsx) 选型一致。
3. 现 25 个使用中的图标全部沿用，并扩展至 ~100 个常用 Outlined 图标（含 Settings、Database、ApiOutlined、TeamOutlined、SafetyCertificateOutlined 等）；bundle 增量估算 ≈ 80 KB（gzipped），可接受。
4. **不允许用户上传自定义 SVG**——通过 ui-customization（已在 commit `5dc0f80f` 落地）做品牌资产，菜单 icon 走白名单。这条约束需要在 MenuManagement 表单上明确（"图标只能从内置库选择"）。

---

## 2. 现状梳理（已实现部分）

### 2.1 数据库

| 表 | 字段 | 状态 |
|----|------|------|
| `menus` | `id, name, type, permission_key, parent_id, sort_order, route_path, component_path, icon, show_status, status, created_by/at, updated_by/at` | ✅ 字段齐全（[migrations/000009_sys_admin.sql:11-46](../../../migrations/000009_sys_admin.sql)）|
| `role_menus` | `role_id, menu_id` | ✅ 已建（同上 line 55-65）|
| `roles.permissions` 字符串数组 | 通过 `permissions` 表（`role_id + resource + action`）持久化 | ⚠️ 老式 RBAC，与 `role_menus` **共存且重复**，前端目前用这个 |

**菜单种子**：[migrations/seed/000057_refresh_menu_seed.sql](../../../migrations/seed/000057_refresh_menu_seed.sql) + [000059_menu_buttons.sql](../../../migrations/seed/000059_menu_buttons.sql) 已写入大部分目录/菜单/按钮，但 **icon 字段大量留空**（一级目录除外）、**component_path 部分缺失**（待补全清单见 §4.1）。

### 2.2 后端 API

| 端点 | Handler | 用途 | 状态 |
|------|---------|------|------|
| `POST   /admin/menus`           | `CreateMenu`        | 新建菜单 | ✅ |
| `GET    /admin/menus`           | `ListMenus`         | 列表（带过滤分页） | ✅ |
| `GET    /admin/menus/:id`       | `GetMenu`           | 获取单个 | ✅ |
| `PUT    /admin/menus/:id`       | `UpdateMenu`        | 更新（含 icon、route_path、component_path、show_status、status）| ✅ |
| `DELETE /admin/menus`           | `DeleteMenus`       | 批量删除（service 层级联删除子节点）| ✅ |
| `GET    /admin/menus/tree`      | `GetMenuTree`       | 完整菜单树（管理后台用） | ✅ |
| `GET    /admin/menus/user-tree` | `GetUserMenuTree`   | 当前用户**默认角色**的菜单树 | ✅ 但前端**未使用** |
| `GET    /auth/menus`            | `GetUserMenusByRole` | 当前用户**当前激活角色**的菜单树（含切换角色后新角色的菜单） | ✅ 但前端**未使用** |
| `PUT    /admin/roles/:id/menus` | `SetRoleMenus`      | 设置角色 ↔ 菜单 | ✅ 但前端 RolePermission **未使用** |
| `GET    /admin/roles/:id/menus` | `GetRoleMenus`      | 获取角色已绑菜单 | ✅ 但前端 RolePermission **未使用** |

**结论**：后端能力 100% 就位，前端仅消费了 menu CRUD（MenuManagement 页面），其余链路（用户菜单树、角色菜单绑定）**完全未对接**。

### 2.3 前端

| 模块 | 文件 | 现状 |
|------|------|------|
| Sidebar 数据源 | `omcmb/webcode/src/components/Layout/Sidebar/navConfig.ts` | **NAV_CONFIG 硬编码常量数组**，10 个一级目录、约 35 个二级菜单（大量已注释隐藏） |
| Sidebar 渲染 | `omcmb/webcode/src/components/Layout/Sidebar/NavMenu.tsx` | `ICON_MAP` 硬编码 25 个 antd icon component |
| 路由注册 | `omcmb/webcode/src/router/routes.tsx` | 100+ 条 `React.lazy(() => import(...))` 静态注册 |
| 菜单管理 | `omcmb/webcode/src/pages/system/MenuManagement/index.tsx` | 已能 CRUD 后端 menus 表，但**编辑表单只有"显示菜单图标" Checkbox**（boolean），**没有让用户输入具体 icon 字符串** |
| 角色权限编辑 | `omcmb/webcode/src/pages/system/RolePermission/index.tsx` | 菜单权限 Tree 数据来自前端常量 `PERMISSION_MODULES`，与 `menus` 表脱钩；保存写到 `permissions` 表（resource:action），不写 `role_menus` |
| 路由守卫 | `omcmb/webcode/src/router/PrivateRoute.tsx` | 仅判 `isAuthenticated`，无菜单权限校验（用户输 URL 可访问任何已注册路由） |

### 2.4 现状权限模型矛盾

```
角色权限源（共存）：
  permissions 表       ← 前端 RolePermission 写入（PERMISSION_MODULES 字符串数组 → resource/action）
  role_menus 表        ← 后端 SetRoleMenus 接口，前端从未调用
  role_api_permissions ← API 端点级（已落地，前端 RolePermission 在用）
  role_device_groups   ← 数据权限（已落地，前端 RolePermission 在用）
```

**症状**：管理员在菜单管理改了一个菜单的 `permission_key`，对角色权限**毫无影响**（permissions 表里仍保留旧 resource/action）。两条线各管各的。

---

## 3. Gap 清单（实现动态加载缺什么）

### 3.1 数据层

| Gap | 严重性 | 解决方案 |
|-----|-------|---------|
| 一级目录的 `icon` 字段在 seed 里部分缺失 | 中 | seed 修订脚本：用现 `navConfig.ts` 的 iconName 反向写回 `menus.icon` |
| 二级菜单的 `component_path` 大量为空 | 高 | seed 修订脚本：参 routes.tsx 列表逐条补齐（约 100 条） |
| 缺少 `menus.is_external`（外链）字段 | 低 | 当前 MenuManagement 表单已有"是否外链"选项但**字段未持久化**，可选补 ALTER COLUMN，或仅在前端处理（约定 route_path 以 http(s):// 开头视为外链） |

### 3.2 后端 API

| Gap | 严重性 | 解决方案 |
|-----|-------|---------|
| `GET /admin/menus/user-tree` vs `GET /admin/menus/user` 路径前后端不一致 | 高 | 统一为 `/auth/menus`（已注册、已实现 GetUserMenusByRole）；前端废弃 `adminApi.getUserMenus()` |
| 公开端点是否暴露当前用户的 `permission_keys` 集合（按钮级判断用） | 中 | 现 GetUserMenuTree 返回的 menus 含 button 类型节点，前端递归收集即可，**无需新接口** |
| 老 `permissions` 表是否保留 | 高（影响兼容） | **保留 1 个版本周期**，废弃 `RemoveAllPermissions/AddPermissions` 调用，新写仅写 role_menus；下一版本删表 |

### 3.3 前端

| Gap | 严重性 | 解决方案 |
|-----|-------|---------|
| MenuManagement 缺 icon 输入框 / 选择器 | 高 | 新增 `IconPicker` 组件 + `iconRegistry` 白名单常量（80-120 个 antd icons） |
| Sidebar 用硬编码 NAV_CONFIG 而非用户菜单 | 高 | 新增 `menuStore` (zustand) + `useUserMenus` query；NavMenu 改造为消费 store；保留 `VITE_DYNAMIC_MENU` 开关用于灰度 |
| 路由 100+ 条静态懒加载 | 高 | 新增 `componentRegistry`（component_path → React.lazy），动态注册 |
| `PrivateRoute` 缺菜单权限校验 | 中 | 注入 `menuStore`，用户菜单不含当前 path → 重定向 `/403` 或 `/dashboard` |
| RolePermission 用 PERMISSION_MODULES 硬编码 | 高 | 替换为 `useMenuTree`（拉 `/admin/menus/tree`），保存改用 `setRoleMenus` |
| 按钮级权限校验缺 helper | 中 | 新增 `usePermission(key: string)` hook，从 menuStore.permissionKeys 取 |
| 切换角色 / 退出 / F5 后 menuStore 状态 | 高 | persist 仅 cache（lastFetchAt），切换角色后强制 invalidate query |

---

## 4. 设计方案

### 4.1 设计原则

1. **菜单即权限（Menu-as-Permission）**：菜单是 RBAC 唯一权威源；P3 阶段一次性删除 `permissions` 表 + 前端 PERMISSION_MODULES 常量。`permission_key` 既是 RBAC key，也是按钮级前端判断 key。
2. **图标白名单（约 100 个 Outlined）**：基于 §1.3 实测 16 × 16px 渲染约束，维护约 100 个 antd Outlined 图标白名单；菜单管理用 IconPicker 选择。运营商交付场景无需无限图标，白名单可控、bundle 体积可预测（gzipped +80KB）。**不允许用户上传自定义 SVG。**
3. **组件注册表（不是字符串 require）**：前端维护一份 `componentRegistry: Record<string, ComponentType>`，菜单 `component_path` 在表内查找；表外打印警告并跳过该路由，便于发现配置错误。**不引入** `import()` 动态字符串路径（vite 不支持完全动态 import，且无类型安全）。
4. **超管定义 = `user.source = 'builtIn'`（唯一标识）**：仅 source='builtIn' 的用户登录后 `GetUserMenuTree` 返回**全部 status='active'** 菜单（service 层判定），不依赖 role_menus；与现有中间件 builtIn 旁路（`internal/admin/middleware.go:138/191`）严格对齐。`role.code='admin'` 的用户**不**视为超管——他/她的菜单仍受 role_menus 控制。
5. **无权限路由 → /403**：用户菜单不含的 path 通过 `<Navigate to="/403" replace />` 跳转，**不**默默回 `/dashboard`（默默回会让用户搞不清是没权限还是 URL 错了）。
6. **按钮级权限：禁用而非隐藏**：无权限的按钮渲染为 `disabled` 状态、显示 tooltip "无权限"，**不**隐藏按钮。理由：(a) 隐藏会让用户怀疑系统在 selectively 出错；(b) disabled 提供"为什么没看到这个功能"的可发现性，便于让用户主动联系管理员申请权限；(c) 与现有 `disabled={isBuiltIn(role)}` 等模式一致。
7. **菜单变更生效策略 = 刷新 / 重新登录**：管理员改了菜单 (CRUD) 后，**不**通过 WebSocket 实时推送给在线用户；当前用户 F5 或重新登录时拉新菜单生效。React Query staleTime = 5 min，5 分钟内复用 cache。这条决策避免了引入 NATS / SSE / WebSocket 的复杂度，对运营商场景（菜单变更频率极低）足够。
8. **渐进灰度**：通过 env feature flag `VITE_DYNAMIC_MENU` 控制启用动态加载，未启用走旧 NAV_CONFIG 路径；过渡期双轨并行。
9. **首屏阻塞最小化**：菜单数据用 React Query stale-while-revalidate；persist 缓存上次结果，新一轮拉取在后台进行；首次登录无 cache 时显示 Spin。

### 4.2 后端方案

#### 4.2.1 路由对齐（无破坏性）

```
公共路径（所有用户）：
  GET /api/v1/auth/menus                 — 当前用户当前激活角色的菜单树（已就位，前端首选）

管理路径（管理员）：
  GET /api/v1/admin/menus/tree           — 全量菜单树（菜单管理用）
  GET /api/v1/admin/menus/user-tree      — 当前用户默认角色菜单树（与 /auth/menus 区别在不感知 currentRoleID）
```

`/admin/menus/user` 这个错路径在 frontend `adminApi.getUserMenus()` 里——直接删除该方法，前端统一走 `/auth/menus`（参 §4.3.2）。

#### 4.2.2 Menu service 行为约定

`GetUserMenuTreeByRole(userID, roleID) -> []Menu`：
- **超管旁路（唯一判定）**：若 `users.source = 'builtIn'` → 返回所有 `status='active'` 的菜单。判定**仅**看 source 字段，不看 role_id / role.code / is_super_admin 等其他标识，与 `internal/admin/middleware.go:138/191` 保持一致。
- 非超管：`SELECT m.* FROM menus m JOIN role_menus rm ON rm.menu_id = m.id WHERE rm.role_id = $1 AND m.status='active' AND m.show_status='show'`，按 `parent_id, sort_order` 组织成树。
- 返回字段含 `permission_key`（前端按钮级判断用），含 button 类型节点（用于权限收集）。

> 现 `pg_menu_repository.go` 已实现该逻辑，**P3 阶段需 code review 校对超管分支**：必须用 `user.source` 判定，不得引入 `role.code = 'admin'` 这类二级判断（避免与 middleware 不对齐）。

#### 4.2.3 数据迁移

新增 1 个 DML 迁移（版本号紧接现有最大值 + 1）：

**文件**：`migrations/seed/000NNN_menu_icon_component_path_backfill.sql`

```sql
-- +goose Up
-- 一级目录：从 navConfig.ts 反向回填 icon
UPDATE menus SET icon = 'DashboardOutlined'   WHERE name = '仪表板'   AND type = 'directory';
UPDATE menus SET icon = 'ClusterOutlined'     WHERE name = '设备管理' AND type = 'directory';
UPDATE menus SET icon = 'AlertOutlined'       WHERE name = '告警管理' AND type = 'directory';
-- ... 共 10 条

-- 二级菜单：补 component_path（与 routes.tsx 的 lazy import 一一对齐）
UPDATE menus SET component_path = 'system/UserManagement'  WHERE route_path = '/system/users';
UPDATE menus SET component_path = 'system/RolePermission'  WHERE route_path = '/system/roles';
-- ... 共 ~80 条（按 routes.tsx 全量列出）

-- +goose Down
UPDATE menus SET icon = NULL WHERE icon IN ('DashboardOutlined', /* ... */);
UPDATE menus SET component_path = '' WHERE component_path LIKE '%/UserManagement' /* ... */;
```

#### 4.2.4 permissions 表退役（B3 端点级方案，分 Phase1/Phase2 实施）

**v0.4 修订（2026-05-08）**：原 v0.3 决策"P3 一次性 DROP TABLE"在调研后判定**不可行**：
- `casbin.go::pgAdapter.LoadPolicy` 把 `permissions` 表喂给 Casbin → 中间件鉴权依赖
- `router.go` 用 `permGroup(resource)` helper 注册了 30+ 路由组，全部依赖 Casbin (resource, action) 模型
- 直接 DROP = 全部非 builtIn 用户 API 鉴权失败

**v0.4 决策**：升级为 **B3 端点级方案**（对齐 GVA `casbin_rule (role, path, method)` 风格），**分双段实施**：

##### B3-Phase1（向下兼容准备段，本 PR / commit `cca8bc59`）

零破坏现网行为，仅做数据源准备 + 新中间件就位 + 停止增量写入：

1. **Casbin 双源 LoadPolicy**：
   - 保留原 `SELECT permissions p JOIN roles r` 输出 (resource, action) 策略
   - 新增 `SELECT role_api_permissions rap JOIN api_endpoints ae JOIN roles r` 输出 (path, method) 端点级策略
   - 同一 `model.conf` 装两类策略，matcher 含 `keyMatch` 兼容路径通配
2. **新中间件 `RequireAPIPermission(roleRepo)`**：自动读 `c.Request.URL.Path` / `c.Request.Method`，调 `CheckPermission`。**Phase1 暂未挂任何路由组**。
3. **`viewer` role_api_permissions seed**（[migrations/seed/000064_seed_role_api_permissions_viewer.sql](../../../migrations/seed/000064_seed_role_api_permissions_viewer.sql)）：补 198 行 GET 类 endpoints，避免 Phase2 切换后只读角色 API 全 403。`admin`/`operator` 已有 445 行（=全集）。
4. **`service.UpdateRole/CreateRole/CopyRole` 删除 permissions 写入**：3 处 `AddPermissions` + 1 处 `RemoveAllPermissions` 移除；保留 `req.Permissions` 字段兼容前端 payload；老 `permissions` 表数据"凝固"，不再增长，仍服务现 Casbin LoadPolicy。

##### B3-Phase2-A（router 切换，commit fd507e5d）✅

`router.go` 30+ 路由组的 `permGroup(resource)` helper 内部从 `RequirePermission(...)` 切到 `RequireAPIPermission(ad.roleRepo)`，含原 4 处粗粒度 `RequirePermission` 的位置。permGroup 第一参数已被忽略（保留兼容），所有受保护路由经 RequireAPIPermission 中间件触发端点级 (path, method) 鉴权。

##### B3-Phase2-B（permissions 表 DROP，commit d07a5cef）✅

破坏性改动落地：

1. ✅ **迁移 `migrations/000065_drop_permissions_table.sql`**：DROP TABLE permissions CASCADE + DROP INDEX idx_permissions_role；Down 段 CREATE TABLE 兜底闭环。
2. ✅ **`casbin.go::pgAdapter.LoadPolicy` 改单源**：删 SELECT permissions 块，仅保留 role_api_permissions JOIN api_endpoints 端点级策略加载。
3. ✅ **`PgRoleRepository.AddPermissions / RemoveAllPermissions / GetPermissions / ListAllPermissions` 全部 stub** 为返回 nil/[]Permission{}（保留接口签名，避免破坏 PermissionChecker / PermissionWriter contract）；CheckPermission 移除 SQL fallback，强依赖 Casbin。
4. ✅ **service.go 清理**：GetRole 不再调 GetPermissions；CreateRole / UpdateRole / CopyRole 注释升级为 Phase2-B；ListAllPermissions deprecated 注释。
5. ✅ **`seed/000001_seed_data.sql`**：移除 3 处 INSERT INTO permissions 与 Down 段 DELETE FROM permissions（fresh deploy 不再尝试写入已 DROP 的表）。
6. ✅ **mockMenuRepo.GetAllActive stub**：handler_test.go / service_test.go 补缺漏，admin 测试包恢复编译。

**Verify**: go build ./... ✅ / go test -timeout 60s ./internal/admin/ → ok ✅

**风险隔离**：Phase2-B 改动均为代码 stub + DDL DROP；现网 staging 环境如需回滚，goose down 000065 会 CREATE TABLE 空壳（数据无法恢复，但 Casbin 已切端点级单源，行为不依赖该表数据）。

#### 4.2.5 数据迁移文件（P0/P2/P3 各一份）

| 阶段 | 文件 | 内容 |
|------|------|------|
| P0   | `seed/000NNN_menu_icon_backfill.sql` | UPDATE 一级目录 icon（10 条），按 navConfig.ts iconName 反向回填 |
| P2   | `seed/000NNN_menu_component_path_backfill.sql` | UPDATE 二级菜单 component_path（约 80 条），与 routes.tsx 严格对齐 |
| P3   | `000NNN_drop_permissions_table.sql` | DROP TABLE permissions, DROP INDEX idx_permissions_role |

> 版本号写时取 `ls migrations/ migrations/seed/ \| sort \| tail -3` 实时确定。

---

### 4.3 前端方案

#### 4.3.1 图标白名单与 IconPicker

**新增文件**：`omcmb/frontend-core/src/components/IconPicker/icons.ts`

```ts
import {
  // 现有 NavMenu 已用 25 个（必含，避免回归）
  DashboardOutlined, ClusterOutlined, AlertOutlined, SettingOutlined,
  LineChartOutlined, CodeOutlined, GlobalOutlined, SaveOutlined,
  CloudUploadOutlined, FolderOutlined, FileTextOutlined, ToolOutlined,
  BarChartOutlined, RadarChartOutlined, SafetyOutlined, AppstoreOutlined,
  GatewayOutlined, DeploymentUnitOutlined, WifiOutlined, ThunderboltOutlined,
  ApartmentOutlined, CloudServerOutlined, AimOutlined, ExperimentOutlined,
  // 系统/管理类（约 30 个）
  TeamOutlined, UserOutlined, KeyOutlined, LockOutlined, SafetyCertificateOutlined,
  AuditOutlined, SolutionOutlined, IdcardOutlined, ApiOutlined, DatabaseOutlined,
  HddOutlined, MonitorOutlined, DesktopOutlined, LaptopOutlined, MobileOutlined,
  // 操作类（约 20 个）
  EyeOutlined, EditOutlined, CopyOutlined, ScissorOutlined, PrinterOutlined,
  CameraOutlined, VideoCameraOutlined, AudioOutlined, BellOutlined, MessageOutlined,
  // 业务类（约 25 个）
  HomeOutlined, ShopOutlined, BankOutlined, GoldOutlined, CrownOutlined,
  TrophyOutlined, MedicineBoxOutlined, FireOutlined, BulbOutlined, RocketOutlined,
  // ...完整列表见 §附录 A，目标约 100 个
} from '@ant-design/icons';
import type { ComponentType } from 'react';

// 图标白名单：name → ComponentType
// key 完全对齐 antd icon 的 export name（如 'DashboardOutlined'），
// 直接作为 menus.icon 字段值持久化。
export const iconRegistry: Record<string, ComponentType<{ style?: React.CSSProperties }>> = {
  DashboardOutlined, ClusterOutlined, /* ...展开全部 */
};

export const iconNames = Object.keys(iconRegistry);
```

**新增组件**：`omcmb/frontend-core/src/components/IconPicker/index.tsx`

- 弹出 Popover 显示 grid（8 列 × N 行），每个图标 **按 16px 渲染**（与 Sidebar 实际尺寸一致，所见即所得）；
- 每个图标 hover tooltip 显示 antd export name（如 "DashboardOutlined"）；
- 顶部搜索框按 name 模糊匹配（不区分大小写、支持去 Outlined 后缀的简写如 "dashboard"）；
- 受控组件：`value: string | undefined` + `onChange(name: string | undefined)`；
- 已选中图标在选择器输入框内回显（图标 16px + 简短 name）；
- 提供"清除"按钮设回 undefined（`menus.icon = ''`）。

**MenuManagement 改造**（[index.tsx 现行 619/882 行的 showIcon Checkbox](../../../../omcmb/webcode/src/pages/system/MenuManagement/index.tsx)）：
- **删除** `showIcon` Checkbox（boolean 字段已无意义）。
- **新增** `Form.Item name="icon"`：`<IconPicker />`，仅当 `type === 'directory'` 时显示（对齐"只有一级目录有 icon"现状）。
- **辅助说明文字**："图标只能从内置库中选择，不支持上传自定义 SVG。" 防止用户误操作。
- save payload 增加 `icon: vals.icon || ''`，覆盖原 `if (vals.showIcon === false) payload.icon = ''` 的兜底逻辑。

#### 4.3.2 menuStore + 数据获取

**新增文件**：`omcmb/frontend-core/src/store/menuStore.ts`

```ts
import { create } from 'zustand';
import { persist, createJSONStorage } from 'zustand/middleware';
import type { Menu } from '../types/system';

interface MenuState {
  menus: Menu[];                     // 树形结构
  flatMenus: Menu[];                 // 扁平，便于路径查找
  permissionKeys: Set<string>;       // 全部 button 级 permission_key 集合
  routePaths: Set<string>;           // 全部菜单/目录 route_path 集合（路由守卫用）
  loaded: boolean;
  setMenus: (menus: Menu[]) => void;
  clear: () => void;
}

export const useMenuStore = create<MenuState>()(
  persist(
    (set) => ({
      menus: [],
      flatMenus: [],
      permissionKeys: new Set(),
      routePaths: new Set(),
      loaded: false,
      setMenus: (menus) => {
        const flat = flatten(menus);
        set({
          menus,
          flatMenus: flat,
          permissionKeys: new Set(flat.filter((m) => m.type === 'button').map((m) => m.permissionKey)),
          routePaths: new Set(flat.filter((m) => m.routePath).map((m) => m.routePath!)),
          loaded: true,
        });
      },
      clear: () => set({ menus: [], flatMenus: [], permissionKeys: new Set(), routePaths: new Set(), loaded: false }),
    }),
    {
      name: 'omc-menu-store',
      storage: createJSONStorage(() => localStorage),
      // Set/Map 不能直接序列化：partialize 转 array
      partialize: (state) => ({
        menus: state.menus,
        flatMenus: state.flatMenus,
        permissionKeys: Array.from(state.permissionKeys),
        routePaths: Array.from(state.routePaths),
        loaded: state.loaded,
      }),
      merge: (persisted, current) => {
        const p = persisted as Record<string, unknown>;
        return {
          ...current,
          ...p,
          permissionKeys: new Set(Array.isArray(p?.permissionKeys) ? (p.permissionKeys as string[]) : []),
          routePaths: new Set(Array.isArray(p?.routePaths) ? (p.routePaths as string[]) : []),
        } as MenuState;
      },
    },
  ),
);
```

> ⚠️ Set/Map 不可 JSON 序列化，必须自定义 `partialize` + `merge`。**参 [previously-fixed userStore token getter bug](../../../../omcmb/frontend-core/src/store/userStore.ts)**：避免在 store 工厂里写 getter 触发 hydration 时 dispatcher null。

**新增 Hook**：`omcmb/frontend-core/src/hooks/api/useMenus.ts`

```ts
import { useQuery } from '@tanstack/react-query';
import { http } from '../../services/http';
import { useMenuStore } from '../../store/menuStore';
import type { Menu } from '../../types/system';

// 拉取当前用户当前角色菜单树。stale-while-revalidate：
// - persist 已恢复 → 立即可用，后台静默更新
// - 切换角色后调 refetch 强制刷新
export function useUserMenus() {
  const setMenus = useMenuStore((s) => s.setMenus);
  return useQuery({
    queryKey: ['userMenus'],
    queryFn: async () => {
      const { data } = await http.get<Menu[]>('/auth/menus');
      setMenus(data);
      return data;
    },
    staleTime: 5 * 60 * 1000,
    refetchOnWindowFocus: false,
  });
}

// 全量菜单树（管理后台用）
export function useMenuTree() {
  return useQuery({
    queryKey: ['menus', 'tree'],
    queryFn: async () => {
      const { data } = await http.get<{ data: Menu[] }>('/admin/menus/tree');
      return data.data;
    },
    staleTime: 60 * 1000,
  });
}
```

**App 启动流程改造**：

`omcmb/webcode/src/App.tsx` → 在 RouterProvider 之前包一层 `MenuBootstrap`：

```tsx
function MenuBootstrap({ children }: { children: ReactNode }) {
  const { isAuthenticated } = useUserStore();
  const { isLoading, error } = useUserMenus(); // useQuery 自动跑
  const loaded = useMenuStore((s) => s.loaded);

  // 未登录 → 跳过（让 PrivateRoute 处理）
  if (!isAuthenticated) return <>{children}</>;

  // 已登录但首次拉菜单且无 persist cache：阻塞 + Spin
  if (isLoading && !loaded) return <FullScreenSpin />;
  if (error && !loaded) return <ErrorPage error={error} />;

  return <>{children}</>;
}
```

#### 4.3.3 NavMenu 改造（消费 menuStore）

**`omcmb/webcode/src/components/Layout/Sidebar/NavMenu.tsx`** 改造点：

- 删除 `import { NAV_CONFIG } from './navConfig'`、删除 hardcoded `ICON_MAP`。
- 改为：`const menus = useMenuStore((s) => s.menus);`
- icon 渲染：`iconRegistry[menu.icon]` —— 找不到就回退默认占位（`<AppstoreOutlined />` 或不显示）。
- buildMenuItems 递归构建（支持任意深度，目前业务约定 3 级：directory → menu → button，但 Sidebar 只渲染 directory + menu，button 级跳过）。
- 单子节点目录的"扁平化"行为保留（NAV_CONFIG 现行处理）。

**Feature flag**：

```ts
const dynamicMenuEnabled = import.meta.env.VITE_DYNAMIC_MENU === 'true';
const menus = useMenuStore((s) => s.menus);
const menuSource = dynamicMenuEnabled && menus.length > 0 ? menus : NAV_CONFIG_FALLBACK;
```

灰度期 `.env` 默认 `VITE_DYNAMIC_MENU=false`，UAT 通过后切 `true`，再清理 fallback。

#### 4.3.4 动态路由

**新增文件**：`omcmb/webcode/src/router/componentRegistry.ts`

```ts
import { lazy } from 'react';
import type { ComponentType } from 'react';

// component_path → lazy component
// 命名约定：'<module>/<ComponentName>'，如 'system/UserManagement'。
// 与 menus.component_path 字段值严格对齐。
export const componentRegistry: Record<string, ComponentType> = {
  'dashboard/Dashboard':           lazy(() => import('@/pages/dashboard')),
  'device/DeviceList':             lazy(() => import('@/pages/device/DeviceList')),
  'system/UserManagement':         lazy(() => import('@/pages/system/UserManagement')),
  'system/RolePermission':         lazy(() => import('@/pages/system/RolePermission')),
  'system/MenuManagement':         lazy(() => import('@/pages/system/MenuManagement')),
  // ... ~100 条，与 routes.tsx 当前 lazy import 一一对齐
};
```

**改造文件**：`omcmb/webcode/src/router/routes.tsx`

```tsx
function buildProtectedRoutes(menus: Menu[]): RouteObject[] {
  const routes: RouteObject[] = [];
  for (const m of menus) {
    if (m.type === 'menu' && m.routePath && m.componentPath) {
      const Component = componentRegistry[m.componentPath];
      if (!Component) {
        console.warn(`[router] component not registered: ${m.componentPath} (menu=${m.name})`);
        continue;
      }
      routes.push({
        path: m.routePath.replace(/^\//, ''), // 去掉前导 /，相对父 path /
        element: withSuspense(Component),
      });
    }
    if (m.children?.length) {
      routes.push(...buildProtectedRoutes(m.children));
    }
  }
  return routes;
}

export function buildRouter(menus: Menu[]) {
  const protectedRoutes = buildProtectedRoutes(menus);
  return createBrowserRouter([
    { path: '/login', element: <LoginPage /> },
    {
      path: '/',
      element: <PrivateRoute><AppShell /></PrivateRoute>,
      children: [
        { index: true, element: <Navigate to="/dashboard" replace /> },
        ...protectedRoutes,
        { path: '*', element: <NotFound /> },
      ],
    },
  ]);
}
```

`router/index.tsx` 改为：

```tsx
function RouterRoot() {
  const menus = useMenuStore((s) => s.menus);
  const router = useMemo(() => buildRouter(menus), [menus]);
  return <RouterProvider router={router} />;
}
```

> ⚠️ `menus` 数组每次拉取后都是**新引用**——需要 useMemo 才不会每次 render 重建 router。但 menus 变化（角色切换）时 router 必须重建——`useMemo` 依赖项就是 menus reference，符合需求。

#### 4.3.5 路由守卫强化（无权限 → /403，不静默回退）

**`PrivateRoute.tsx`** 增加 routePath 检查：

```tsx
const { allowedPaths, loaded } = useMenuStore((s) => ({ allowedPaths: s.routePaths, loaded: s.loaded }));
const location = useLocation();

// 已认证 + 菜单已加载 + 当前 path 不在允许集合 + 不是 /dashboard（默认登录跳转）
if (loaded && !allowedPaths.has(location.pathname) && location.pathname !== '/dashboard') {
  return <Navigate to="/403" replace />;
}
```

**新增 `pages/error/Forbidden.tsx`**：

- 显示 403 图（antd `<Result status="403">`）+ 中文提示"无权访问该页面"
- 副标题："如需访问该功能，请联系系统管理员为您的角色（{role}）增加菜单权限。"
- 主操作按钮"返回首页"→ navigate('/dashboard')；次操作"重新登录"→ logout + /login
- **故意不显示**当前 path 字符串（避免暴露未授权资源命名）

**为什么不静默回退到 /dashboard**：
- 用户搞不清"是没权限"还是"URL 拼错了"——两种都会回 dashboard，无法自查
- 运营/客服反馈难度上升（"我点的链接打开是 dashboard，是不是 bug？"）
- /403 提供清晰的"原因 + 行动指引"

#### 4.3.6 按钮级权限 hook（禁用而非隐藏）

**新增**：`omcmb/frontend-core/src/hooks/usePermission.ts`

```ts
import { useMenuStore } from '../store/menuStore';

/** 判断当前用户是否拥有指定 permission_key（来自菜单 button 类型节点） */
export function usePermission(key: string): boolean {
  return useMenuStore((s) => s.permissionKeys.has(key));
}

/** 同步版本，用于事件回调内 */
export function hasPermission(key: string): boolean {
  return useMenuStore.getState().permissionKeys.has(key);
}
```

**消费约定（CRITICAL）**：无权限按钮 **设为 disabled，不要隐藏**：

```tsx
const canDelete = usePermission('system:user:delete');

// ✅ 推荐写法：disabled + Tooltip 提示
<Tooltip title={canDelete ? undefined : '无权限：请联系管理员申请「系统:用户:删除」权限'}>
  <Button danger disabled={!canDelete} onClick={handleDelete}>
    删除
  </Button>
</Tooltip>

// ❌ 不要这样写（隐藏）：
{ canDelete && <Button danger onClick={handleDelete}>删除</Button> }
```

**为什么禁用而非隐藏**：
1. **可发现性**：用户看见但不能用 → 知道这功能存在 → 主动申请权限；隐藏会让用户不知道有这功能。
2. **运营/客服效率**：用户截图描述"删除按钮灰着点不动"比"我看不到删除按钮"更准确。
3. **一致性**：与本仓既有约定一致，如内置角色编辑按钮 `disabled={isBuiltIn(role)}`（参 [RolePermission/index.tsx:860](../../../../omcmb/webcode/src/pages/system/RolePermission/index.tsx)）。
4. **无安全顾虑**：disabled 仅 UI 层提示，真正的访问控制在后端 API 层（`role_api_permissions` + 中间件）。

**例外（仍可隐藏）**：
- 整列表的列控制（如管理员才看见的"操作日志"列）—— 隐藏整列比每行 disabled 体验好；
- 顶部 tab 切换 —— 没权限的 tab 隐藏避免空白页；
- **判断依据**：用户感知该入口"可被申请到"→ disabled；用户根本不该感知存在 → 隐藏。

#### 4.3.7 RolePermission 模型迁移

**改造点**（[RolePermission/index.tsx](../../../../omcmb/webcode/src/pages/system/RolePermission/index.tsx)）：

1. **删除** `PERMISSION_MODULES` 常量（约 192 行）和派生常量 `ALL_PERMISSION_KEYS / ALL_PERMISSION_LEAF_KEYS / ALL_SECOND_LEVEL_KEYS`。
2. **替换菜单权限 Tree 数据源**：
   ```tsx
   const { data: menuTree } = useMenuTree(); // /admin/menus/tree
   const permissionTreeData = useMemo(() => buildAntdTreeFromMenus(menuTree), [menuTree]);
   ```
3. **替换权限读取/保存**：
   ```tsx
   // 读：getRoleMenus(roleId) 返回 menu_id 数组
   // 写：setRoleMenus(roleId, { menu_ids: [...] })
   const checkedMenuIds = checkedKeys.filter((k) => isUuid(k));
   await setRoleMenus.mutateAsync({ roleId: selectedRole.id, data: { menu_ids: checkedMenuIds }});
   ```
4. **保留** API 权限 Tree（`role_api_permissions`）、设备分组 Tree（`role_device_groups`）：这两条线已对齐后端，无需改动。
5. **删除** `permissions` 字段相关的 createRole/updateRole 调用（payload 不再带 permissions）；后端在该期内若收到 `permissions` 字段忽略即可。

**Tree 节点 key 设计**：直接用 `menu.id`（UUID）。父子联动行为不变。

---

### 4.4 字段映射对照表（供运营/迁移参考）

| 旧 PERMISSION_MODULES key | 等价 menu permission_key | 等价 menu route_path | component_path |
|---|---|---|---|
| `device.list` | `device:list` | `/device/list` | `device/DeviceList` |
| `device.list.query` | `device:list:query` | — (button) | — |
| `device.list.add` | `device:list:add` | — (button) | — |
| `system.users` | `system:user` | `/system/users` | `system/UserManagement` |
| `system.roles` | `system:role` | `/system/roles` | `system/RolePermission` |
| `system.menus` | `system:menu` | `/system/menus` | `system/MenuManagement` |
| ... | ... | ... | ... |

> 完整映射表在 §4.2.3 的迁移脚本里，以菜单 seed 为准。

---

## 5. 实施分阶段路线图

| 阶段 | 范围 | 入口文件 | 风险 | 状态 |
|------|------|---------|------|-----------|
| **P0：图标可配** ✅ | IconPicker（116 个 Outlined 白名单）+ iconRegistry + MenuManagement icon 字段 | `webcode/src/components/IconPicker/`, `pages/system/MenuManagement/index.tsx` | 低 | commit e319759e |
| **P1：用户菜单接口对接** ✅ | menuStore（partialize/merge for Set）+ useUserMenus / useMenuTree + MenuBootstrap 阻塞首屏 + NavMenu 灰度 + .env VITE_DYNAMIC_MENU=false | `frontend-core/src/store/menuStore.ts`, `frontend-core/src/services/api/menuApi.ts`, `frontend-core/src/types/menu.ts`, `frontend-core/src/hooks/api/useMenus.ts`, `frontend-core/src/hooks/usePermission.ts`, `webcode/src/components/MenuBootstrap/*`, `webcode/src/components/Layout/Sidebar/NavMenu.tsx` | 中 | commit 01e35c32 |
| **P2：动态路由 + 守卫** ✅ | componentRegistry（~95 entries）+ PrivateRoute routePath 守卫 + 403 页面接入 + seed 000066 回填 component_path | `webcode/src/router/componentRegistry.ts`, `webcode/src/router/PrivateRoute.tsx`, `webcode/src/router/routes.tsx`, `migrations/seed/000066_menu_component_path_backfill.sql` | 中 | commit 365abf19 |
| **P3：RolePermission 模型迁移** ✅ | RolePermission 删 PERMISSION_MODULES（250+ 行硬编码）+ 派生常量；改用 useMenuTree + fetchRoleMenuIds + setRoleMenus；createRole/updateRole 的 permissions 字段恒为 [] | `pages/system/RolePermission/index.tsx` | 中 | commit 099b0a12 |
| **B3-Phase2-A：router 切端点级鉴权** ✅ | router.go 30+ 路由组 permGroup helper 改用 RequireAPIPermission 中间件 | `cmd/app/provider/router.go` | 中 | commit fd507e5d |
| **B3-Phase2-B：permissions 表 DROP** ✅ | migrations/000065 DROP TABLE; casbin LoadPolicy 单源；pg_role_repository 4 方法 stub；seed/000001 移除 INSERT | `internal/admin/{casbin.go, pg_role_repository.go, service.go}`, `migrations/000065_drop_permissions_table.sql`, `migrations/seed/000001_seed_data.sql` | 高 | commit d07a5cef |

**总工作量**：约 8-9 工作日，P0 → P1 → P2 → P3 严格顺序。

**P3 一次到位的合并依据**（按 5/8 决策）：
- 跳过原计划的"P3 停写 → P4 DROP"两步走，直接同 PR 完成 service 层删调用、repo 层删函数、SQL 层 DROP TABLE、前端清理灰度代码。
- 风险换收益：避免长尾的"半截清理状态"——staging/prod 环境可能因为版本不同步出现一边写一边读老表的怪异行为。
- 上线前必做：staging 环境跑完整 e2e + 4 个内置角色逐一验证可用功能。

---

## 6. 风险与降级

| 风险 | 影响 | 缓解 |
|------|------|------|
| 菜单 seed 缺失 component_path → 用户首屏 404 | 阻断登录 | seed 提交前用脚本对照 routes.tsx 全量校验；componentRegistry 找不到时打印警告 + 跳过该路由（不挂 404） |
| 用户切换角色后菜单延迟刷新 | 体验差 | switchRole 成功后立即 invalidate `['userMenus']` query；buildRouter 依赖 menuStore 自动重建 |
| 前端 persist 缓存与新版本菜单不一致 | 老缓存指向已删除 component | persist version + storage 版本字段，bump 版本后 merge 时丢弃旧缓存；iconRegistry 缺失图标 → 占位图标 |
| 超管菜单返回全量过大（理论 200+ 条）| 首屏慢 | 实测：menus 表 < 500 行，single tree query < 50ms；不分页可接受 |
| 灰度期 NAV_CONFIG vs 后端菜单不一致导致开发混乱 | 中 | feature flag 默认 false；clear 文档"这是过渡期，新菜单只往后端加" |
| `usePermission` 在 menuStore 未加载时返回 false | 按钮闪烁 disabled | MenuBootstrap 阻塞首屏直到 loaded；第二次 F5 用 persist 立即可用 |
| `permissions` 表删除时机过早 | 旧角色权限丢失 | P3 阶段 service 仅"读不写"老表 1 个版本；P4 才 DROP |
| 用户在浏览器输 URL 直接访问无权限路由 | 信息泄漏 | 路由守卫 `routePaths.has(pathname)` 兜底 → /403 |

**降级策略**：
- P0 任意问题 → 不影响主线功能（仅 icon 配置）
- P1/P2 问题 → `VITE_DYNAMIC_MENU=false` 立即回退到 NAV_CONFIG + 静态路由
- P3 问题 → RolePermission 暂时保留 PERMISSION_MODULES 分支兜底（feature flag）

---

## 7. 验收标准（DoD）

### P0
- [ ] MenuManagement 编辑/新增表单含 IconPicker，能保存 icon 字段值到 menus.icon
- [ ] iconRegistry 含至少 80 个常用 antd icon
- [ ] seed 回填 10 个一级目录的 icon
- [ ] typecheck / lint / 单元测试通过

### P1
- [ ] 登录后 `/auth/menus` 被调用一次，response 写入 menuStore
- [ ] menuStore 在 F5 后从 persist 立即恢复，后台 stale-while-revalidate
- [ ] `VITE_DYNAMIC_MENU=true` 时 Sidebar 渲染来自 menus 表，icon 由 iconRegistry 解析
- [ ] `VITE_DYNAMIC_MENU=false` 时 Sidebar 回退 NAV_CONFIG，行为与今日完全一致
- [ ] admin 用户登录看到全部菜单（验证超管旁路）
- [ ] 新建测试角色 `viewer-test` 仅绑 `dashboard` + `alarm-current` 两个菜单 → 该角色登录只看见这两项

### P2
- [ ] componentRegistry 完整覆盖现有 routes.tsx 全部 lazy import
- [ ] `VITE_DYNAMIC_MENU=true` 时所有现有功能页可正常打开
- [ ] 角色无权限的 path 直接输 URL → 跳 /403
- [ ] seed 已回填全部 ~80 条 menu 的 component_path
- [ ] 切换角色后 router 自动重建，新角色菜单立即生效

### P3（一次到位）
- [ ] RolePermission 编辑面板的菜单权限 Tree 来自 `/admin/menus/tree`
- [ ] 保存调用 `PUT /admin/roles/:id/menus`
- [ ] 4 个内置角色（admin/operator/viewer/auditor）+ 3 个测试角色 e2e 通过：菜单可见性、按钮 disable、API 鉴权全部对齐
- [ ] 老 PERMISSION_MODULES 常量已删除（grep 全仓库无残留）
- [ ] 后端 `RemoveAllPermissions` / `AddPermissions` / `service.go` permissions 表写入路径已删除
- [ ] `permissions` 表 DROP 迁移已合入并在 staging 跑过 down/up 验证
- [ ] `VITE_DYNAMIC_MENU` feature flag 默认 true，无 fallback 代码

---

## 8. 决策已确认（2026-05-08）

> 第 0.1 版的 7 个开放问题，本节给出最终决策。后续实施按此执行，无需再次 review。

| # | 问题 | 决策 |
|---|------|------|
| 1 | 图标白名单大小 | **约 100 个 antd Outlined 图标**（含现 25 个 + 系统/管理/操作/业务类 75 个），完整列表见 §附录 A 待补 |
| 2 | 无权限路由处理 | **跳 `/403`**（`<Result status="403" />` + 返回首页/重新登录两个 CTA），不静默回 /dashboard |
| 3 | 超管定义 | **仅 `users.source = 'builtIn'`** 视为超管 → 全量菜单。`role.code='admin'` 的非 builtIn 用户仍受 role_menus 限制 |
| 4 | 图标 / 路线建议 | **按建议 100 个 Outlined**，三段路线（P0→P1→P2→P3）保持 |
| 5 | permissions 表退役 | **P3 一次到位**：service 删写入、repo 删函数、迁移 DROP TABLE、前端清理灰度，同 PR 同上线 |
| 6 | 菜单变更生效策略 | **刷新（F5）或重新登录后生效**，5 分钟 staleTime；不引入 WebSocket / SSE / NATS 推送 |
| 7 | 按钮级权限 | **disabled + Tooltip "无权限"**，不隐藏按钮（参 §4.3.6 详细论证） |

---

## 附录 A：图标白名单完整清单（v0.3 冻结）

**总计 116 个 antd Outlined 图标**（基础 100 个 + 兼容现网 seed 补 16 个），按业务用途分组。所有 key 严格对齐 `@ant-design/icons` 的 export name，可直接作为 `menus.icon` 字段值。

### A.1 现有 NavMenu 在用（24 个 — 必含，避免回归）

`DashboardOutlined`, `ClusterOutlined`, `AlertOutlined`, `SettingOutlined`,
`LineChartOutlined`, `CodeOutlined`, `GlobalOutlined`, `SaveOutlined`,
`CloudUploadOutlined`, `FolderOutlined`, `FileTextOutlined`, `ToolOutlined`,
`BarChartOutlined`, `RadarChartOutlined`, `SafetyOutlined`, `AppstoreOutlined`,
`GatewayOutlined`, `DeploymentUnitOutlined`, `WifiOutlined`, `ThunderboltOutlined`,
`ApartmentOutlined`, `CloudServerOutlined`, `AimOutlined`, `ExperimentOutlined`

### A.2 用户/团队类（10 个）

`UserOutlined`, `TeamOutlined`, `UsergroupAddOutlined`, `UsergroupDeleteOutlined`,
`UserAddOutlined`, `UserDeleteOutlined`, `IdcardOutlined`, `SolutionOutlined`,
`ContactsOutlined`, `CrownOutlined`

### A.3 安全/权限类（8 个）

`LockOutlined`, `UnlockOutlined`, `KeyOutlined`, `SafetyCertificateOutlined`,
`SecurityScanOutlined`, `AuditOutlined`, `EyeOutlined`, `EyeInvisibleOutlined`

### A.4 通信/网络类（10 个）

`ApiOutlined`, `LinkOutlined`, `ShareAltOutlined`, `BranchesOutlined`,
`ForkOutlined`, `NodeIndexOutlined`, `SwapOutlined`, `SyncOutlined`,
`RetweetOutlined`, `BlockOutlined`

### A.5 存储/数据类（10 个）

`DatabaseOutlined`, `HddOutlined`, `ContainerOutlined`, `InboxOutlined`,
`BookOutlined`, `ReadOutlined`, `ProfileOutlined`, `SnippetsOutlined`,
`CodepenOutlined`, `BoxPlotOutlined`

### A.6 文件/操作类（15 个）

`EditOutlined`, `DeleteOutlined`, `CopyOutlined`, `ScissorOutlined`,
`FileAddOutlined`, `FileSearchOutlined`, `FilePdfOutlined`, `FileExcelOutlined`,
`FileImageOutlined`, `DownloadOutlined`, `UploadOutlined`, `SearchOutlined`,
`ReloadOutlined`, `FilterOutlined`, `SortAscendingOutlined`

### A.7 通知/状态类（10 个）

`BellOutlined`, `NotificationOutlined`, `MessageOutlined`, `MailOutlined`,
`SoundOutlined`, `FlagOutlined`, `TagOutlined`, `StarOutlined`,
`HeartOutlined`, `ClockCircleOutlined`

### A.8 反馈/进度类（5 个）

`CheckCircleOutlined`, `CloseCircleOutlined`, `InfoCircleOutlined`,
`WarningOutlined`, `QuestionCircleOutlined`

### A.9 业务/场景类（8 个）

`HomeOutlined`, `ShopOutlined`, `BankOutlined`, `RocketOutlined`,
`BulbOutlined`, `FireOutlined`, `BugOutlined`, `MedicineBoxOutlined`

### A.10 兼容现网 seed 补充（16 个）

> 来源：[seed/000057_refresh_menu_seed.sql](../../../migrations/seed/000057_refresh_menu_seed.sql) 的二级菜单 / 三级按钮已使用的图标。这 16 个虽不属于 §4.1 #2 提及的 100 个常用，但为避免编辑现有菜单时 IconPicker 显示"图标已下线"必须收录。

`AreaChartOutlined`, `BgColorsOutlined`, `CalendarOutlined`, `ConsoleSqlOutlined`,
`ControlOutlined`, `FieldNumberOutlined`, `FileDoneOutlined`, `FileZipOutlined`,
`HistoryOutlined`, `MenuOutlined`, `PartitionOutlined`, `RollbackOutlined`,
`ScheduleOutlined`, `UndoOutlined`, `UnorderedListOutlined`, `UpCircleOutlined`

### 维护规则

- **禁止任意添加**：未来需要新图标必须先 PR 改本附录，再改 `iconRegistry`，避免清单与代码不一致。
- **禁止 Filled / TwoTone**：16px 渲染下视觉不达标（参 §1.3）。
- **总量上限 200**：超过 200 后 bundle 影响明显，需要按需 lazy load 重构。
- **删除图标的兼容**：若某图标从清单移除而历史菜单仍在用 → MenuManagement 编辑时显示"⚠️ 图标已下线"提示，IconPicker 不展示该图标，渲染时回退到无图标。

---

## 10. 待办事项（v0.6 复核遗留）

> P0/P1/P2/P3 + B3 主线全部 ✅；本节列出 PRD 字面要求中**部分实现**或**留待后续**的项，
> 不阻塞 v0.6 标记"主线收口"，但需在后续 wave 跟进。

### 10.1 部分实现（与 PRD 字面要求有差距，已在 commit message 中标注偏离原因）

| # | PRD 章节 | 实际状态 | 偏离原因 | 建议处理 |
|---|---------|---------|---------|---------|
| 1 | §4.3.4 完全动态 buildRouter | 仅做 componentRegistry + PrivateRoute 守卫；routes.tsx 保留 100+ 静态 lazy import | 一次性切完全动态 = 触碰 18 模块 100+ 路由，灰度风险大 | P2.5 wave：env=true 在 staging 跑过 1-2 周后再做 buildRouter 重构（commit 365abf19 已说明） |
| 2 | §5 P3 DoD #5「后端 PgRoleRepository 4 方法已删除」 | 4 方法 stub 为返回 nil/[]，未真删 | 删除会破坏 PermissionChecker / PermissionWriter 接口 contract，影响 mock 与 RequireAPIPermission 中间件签名 | 待 RolePermission 灰度上线 1 个版本周期后，统一接口清理（同时移除 `Role.Permissions` / `CreateRoleRequest.Permissions` / `UpdateRoleRequest.Permissions` 三个 DTO 字段） |
| 3 | §5 P3 DoD #7「VITE_DYNAMIC_MENU 默认 true，无 fallback」 | 默认 false，NAV_CONFIG fallback 仍在 | §6 降级策略矛盾：P1/P2 出问题需要立刻回退到 false；UAT 未通过前不能切 true | UAT 通过后单独发版切 default=true；同 PR 删 NAV_CONFIG / STATIC_ICON_MAP fallback |

### 10.2 上线前必做（PRD §5 P3 DoD #3 / #6）

| # | 项目 | 说明 |
|---|------|------|
| 1 | e2e 4 内置角色 + 3 测试角色端到端鉴权验证 | 在 `omcgo/scripts/e2e_verify.sh` 增加 W2.D.2 RBAC 章节：admin / operator / viewer / 自定义 viewer-test 各自访问 30+ 受保护 API 路由组，验证 200/403 符合预期 |
| 2 | staging 环境跑 permissions 表 DROP 迁移 down/up | `make migrate-down` 回到 000064 → `make migrate-up` 到 000066，验证 Casbin LoadPolicy 单源、所有非 builtIn 用户鉴权正常 |
| 3 | RolePermission UAT | 4 内置角色 + 1 自定义角色的"菜单可见性 + 按钮 disable + API 鉴权"三轨对齐 |

### 10.3 后续清理工作

| # | 项目 | 说明 |
|---|------|------|
| 1 | adminApi.createRole / updateRole 移除 permissions 字段 | RolePermission P3 已传 []，但 API client 还在 split 资源/动作；UAT 后清理 |
| 2 | adminApi.MenuItem 类型与 frontend-core/types/menu.ts Menu 类型合并 | 两套并存（MenuItem 字段名不规整、Menu 干净）；后续 RolePermission / MenuManagement 全切 Menu 后删 MenuItem |
| 3 | NAV_CONFIG / STATIC_ICON_MAP 兜底分支删除 | 跟 §10.1 #3 灰度切换同 PR 完成 |

---

## 9. 关联文档

- [menus.md](./menus.md) — 菜单 CRUD（前置基线）
- [roles.md](./roles.md) — 角色管理 + role_menus 关联
- [users.md](./users.md) — 用户管理 + 默认角色
- 后端 menu_handler：[omcgo/internal/admin/menu_handler.go](../../../internal/admin/menu_handler.go)
- 后端 menu repository：[omcgo/internal/admin/pg_menu_repository.go](../../../internal/admin/pg_menu_repository.go)
- 前端 navConfig（待废弃）：[omcmb/webcode/src/components/Layout/Sidebar/navConfig.ts](../../../../omcmb/webcode/src/components/Layout/Sidebar/navConfig.ts)
- 前端 NavMenu（待改造）：[omcmb/webcode/src/components/Layout/Sidebar/NavMenu.tsx](../../../../omcmb/webcode/src/components/Layout/Sidebar/NavMenu.tsx)
- 前端 routes.tsx（待改造）：[omcmb/webcode/src/router/routes.tsx](../../../../omcmb/webcode/src/router/routes.tsx)
- 前端 RolePermission（P3 改造）：[omcmb/webcode/src/pages/system/RolePermission/index.tsx](../../../../omcmb/webcode/src/pages/system/RolePermission/index.tsx)
