# 系统管理 — 用户组管理（System / Groups）PRD

> ⚠️ **本菜单计划下线**。详见 [users.md §11.5 v0.3 决议](./users.md#115-v03--术语统一用户组--用户角色)。
> 本 PRD 主要任务是**记录现状 + 列出迁移与下线计划**，不规划新功能。

| 版本 | 日期 | 作者 | 备注 |
|------|------|------|------|
| 0.1  | 2026-05-06 | Backend/Frontend Team | 现状记录 + 下线计划 |

**关联功能域**：F06 OMC-R 核心 / RBAC 历史遗留

**相关文件**：

| 层 | 路径 |
|----|------|
| 前端页面 | [omcmb/webcode/src/pages/system/GroupManagement/index.tsx](../../../../omcmb/webcode/src/pages/system/GroupManagement/index.tsx) |
| 前端 i18n | `nav.system.groups: '用户组管理'` |
| 路由 | `/system/groups` |
| 后端"组"端点 | `/api/v1/admin/groups` ← **实际是 `/admin/roles` 的别名**（[handler.go:208-212](../../../internal/admin/handler.go#L208-L212)）|

---

## 1. 业务背景与下线决议

### 1.1 历史

v0.1/v0.2 期间前端使用「用户组（Group）」语义，与后端「角色（Role）」语义重复。后端为兼容前端，把 `/admin/groups` 实现为 `/admin/roles` 的别名（[handler.go:206-212](../../../internal/admin/handler.go#L206-L212)）：

```go
// Groups endpoint - alias for roles (for frontend compatibility)
groups := rg.Group("/groups")
{
    groups.GET("", h.ListRoles)
    groups.GET("/:id", h.GetRole)
}
```

### 1.2 v0.3 决议（[users.md §11.5](./users.md#115-v03--术语统一用户组--用户角色)）

- 用户身上挂的统一称作 **角色（Role）**
- 「用户组 / Group」语义废弃
- 前端字段 `groupNames` / `targetGroupId` → `roleIds` / `targetRoleIds`
- 后端 `/admin/groups*` 保留兼容期一个发布周期，下版本删除
- 前端 `Group` / `BackendGroup` TS 类型 → `Role` / `BackendRole`

### 1.3 本 PRD 的任务

不规划新功能；专注：

1. 记录当前 `GroupManagement/index.tsx` 实际行为（防止维护时误改）
2. 给出下线步骤和时间表
3. 替代方案 cross-reference 到 [roles.md](./roles.md)

---

## 2. 现状描述

### 2.1 前端实际表现

`GroupManagement/index.tsx` 当前展示的"用户组"列表，**实际后端返回的是角色列表**（通过 `/admin/groups` 别名）。所谓"内置组"`builtIn` = `roles.is_system`。

| 前端字段 | 实际后端来源 |
|---------|------------|
| `groupName` | `roles.name` |
| `description` | `roles.description` |
| `userCount` | 后端实际不返回（前端常显 0）|
| `roleCount` | 后端实际不返回（用户组关联角色这个二级嵌套从未实现）|
| `builtIn` | `roles.is_system` |

### 2.2 表单字段（创建/编辑）

| name | 实际语义 | 当前是否有效 |
|------|---------|------------|
| `groupName` | 角色名 | ✅ 等价于 `roles.name` |
| `description` | 角色描述 | ✅ |
| 关联角色 | （历史设计：组下挂角色）| ❌ 后端不支持，前端选了也不写库 |
| 关联用户 | （历史设计：组下挂用户）| ❌ 同上 |

### 2.3 后端

无独立 `groups` 表，无独立 service / handler。`/admin/groups*` 只挂了 GET 别名，无 POST/PUT/DELETE。前端的 `useCreateGroup` / `useUpdateGroup` / `useDeleteGroups` 实际**调用的是 `/admin/roles*` 端点**（adminApi.ts 中的映射）。

---

## 3. 下线计划

### 3.1 分阶段

| 阶段 | 时间 | 动作 | 责任方 |
|------|------|------|--------|
| **T0** | v0.3 决议日 | PRD 落定（[users.md §11.5](./users.md#115-v03--术语统一用户组--用户角色)）+ 本 PRD 发布 | PM/Backend |
| **T1**（当前 sprint）| ASAP | 前端 `GroupManagement` 页面挂 `<Alert>` banner：「本菜单将于下版本下线，请使用「角色管理」(`/system/roles`)」+ 在导航中折叠该菜单（visible=false 但路由保留）| 前端 |
| **T2** | 1 个发布周期后 | 删除前端代码：`GroupManagement/index.tsx` + 路由 + i18n key + Group/BackendGroup TS 类型 + adminApi 中 group 相关方法 | 前端 |
| **T3** | T2 同步 | 删除后端 `/admin/groups*` 别名路由（[handler.go:208-212](../../../internal/admin/handler.go#L208-L212)）| 后端 |
| **T4** | T3 后 | seed/migration 中"用户组"相关文案统一改为"角色" | DBA |

### 3.2 风险与回退

| 风险 | 缓解 |
|------|------|
| 第三方文档/培训资料仍使用"用户组"术语 | T1 banner 文案明确指向新菜单；发布说明中强调术语变更 |
| 前端某些角落仍 import `Group` 类型 | T1 加 `@deprecated` JSDoc + ESLint 规则禁止新引用 |
| 老版本前端访问已下线的 `/admin/groups*` 端点 | T3 前先观察 `/admin/groups*` 的访问量指标，连续 1 周 0 调用后再删 |

---

## 4. 替代方案

| 想做什么 | 在新菜单完成 |
|---------|------------|
| 创建角色 | [roles.md §4](./roles.md) — 角色管理「添加」按钮 |
| 给角色分配菜单/API/设备分组 | [roles.md §6.2-6.4](./roles.md) |
| 给用户分配角色 | [users.md §4-§5](./users.md) — 用户管理「编辑用户」时角色多选 |
| 批量操作用户的角色 | [users.md §4.2](./users.md) — 「批量分配角色」（v0.3 重命名自「移动到组」）|

---

## 5. 接口契约（仅记录别名，不规划新接口）

| Method | 路径 | 实际行为 | 下线时机 |
|--------|------|---------|---------|
| GET | `/admin/groups` | = `GET /admin/roles` | T3 |
| GET | `/admin/groups/{id}` | = `GET /admin/roles/{id}` | T3 |

---

## 6. 验收清单（DoD）

T1（当前 sprint）：
- [ ] `GroupManagement/index.tsx` 顶部加 `<Alert type="warning">` 提示下线
- [ ] 导航菜单中将「用户组管理」隐藏（`visible: false`），路由保留可访问
- [ ] adminApi 中 Group 相关方法加 `@deprecated` JSDoc

T3（下线时）：
- [ ] 删除前端 `GroupManagement/` 目录
- [ ] 删除路由 `system/groups`
- [ ] 删除 i18n key `nav.system.groups`
- [ ] 删除 `Group` / `BackendGroup` TS 类型
- [ ] 删除 [handler.go:206-212](../../../internal/admin/handler.go#L206-L212) 的 groups 别名
- [ ] 验证：`grep -r "groupName\|GroupManagement\|/admin/groups" omcmb/ omcgo/` 无残留（除 PRD 文档归档）

---

## 7. 非目标

- 不为本菜单新增任何功能
- 不修复现有 bug（除非阻塞下线流程）
