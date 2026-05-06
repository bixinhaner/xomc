# 系统管理 — API 管理（System / API Management）PRD

> 文档目的：管理后端 API 端点注册表（`api_endpoints`），为「角色 ↔ API 权限」配置提供数据源。
> 与 [roles.md](./roles.md) §6.3 配套。

| 版本 | 日期 | 作者 | 备注 |
|------|------|------|------|
| 0.1  | 2026-05-06 | Backend/Frontend Team | 基于现网 `ApiManagement/index.tsx` + `internal/admin/handler.go:194-204` 抽取 |

**关联功能域**：F06 OMC-R 核心 / RBAC

**相关文件**：

| 层 | 路径 |
|----|------|
| 前端页面 | [omcmb/webcode/src/pages/system/ApiManagement/index.tsx](../../../../omcmb/webcode/src/pages/system/ApiManagement/index.tsx) |
| 前端 API | [omcmb/frontend-core/src/services/api/apiPermissionApi.ts](../../../../omcmb/frontend-core/src/services/api/apiPermissionApi.ts) |
| 后端 Handler | [omcgo/internal/admin/handler.go:194-204](../../../internal/admin/handler.go#L194-L204) |
| 数据库 | [omcgo/migrations/000012_api_endpoints_and_data_perm.sql](../../../migrations/000012_api_endpoints_and_data_perm.sql) |

---

## 1. 业务背景

API 端点（`api_endpoints`）是后端 Gin 路由的元数据登记表，承担两个角色：

1. **角色 API 权限的数据源** — 角色管理面板按 `api_group` 分组列出所有端点供勾选
2. **运行时鉴权依据** — HTTP 中间件按 `(method, path)` 查 `role_api_permissions` 决定是否放行

端点来源：
- **手工录入**：管理员通过本页面 `POST /admin/api-endpoints` 添加（`is_auto = false`）
- **自动同步**：点击"同步"按钮，后端扫描 Gin 路由树，新端点入库 `is_auto = true`

---

## 2. 实体模型

### 2.1 `api_endpoints` 表（[migrations/000012_api_endpoints_and_data_perm.sql:4-26](../../../migrations/000012_api_endpoints_and_data_perm.sql)）

| 列 | 类型 | 约束 | 说明 |
|----|------|------|------|
| `id` | UUID | PK, `gen_random_uuid()` | |
| `path` | VARCHAR(256) | NOT NULL | 路径，如 `/api/v1/admin/users/:id` |
| `method` | VARCHAR(8) | NOT NULL, CHECK IN (`GET`,`POST`,`PUT`,`DELETE`,`PATCH`) | |
| `name` | VARCHAR(128) | NOT NULL | 中文展示名，如 "更新用户" |
| `description` | TEXT | NULL | 接口说明 |
| `api_group` | VARCHAR(64) | NULL | 分组标识，如 `用户管理`、`角色管理` |
| `is_auto` | BOOLEAN | NOT NULL DEFAULT FALSE | 是否同步生成（vs 手工录入）|
| `created_at` / `updated_at` | TIMESTAMPTZ | NOT NULL | |

唯一约束：`UNIQUE(path, method)`

### 2.2 关联表 `role_api_permissions`（待补，参 [roles.md §7 P0 #1](./roles.md)）

| 列 | 类型 |
|----|------|
| `role_id` | UUID, FK → `roles(id)` ON DELETE CASCADE |
| `endpoint_id` | UUID, FK → `api_endpoints(id)` ON DELETE CASCADE |
| PRIMARY KEY | (`role_id`, `endpoint_id`) |

### 2.3 前端类型

```ts
interface ApiEndpoint {
  id: string;
  path: string;
  method: string;
  name: string;
  apiGroup?: string;
  module: string;        // ⚠️ 后端无此字段，前端可能用作 apiGroup 别名
  description: string;
  createdAt?: string;
  updatedAt?: string;
}

interface SyncApiResult {
  created: number;
  updated: number;
  total: number;
}
```

---

## 3. 列表页字段定义

| key | 列标题 | dataIndex | UI 渲染 |
|-----|-------|-----------|--------|
| `path` | API 路径 | `path` | monospace |
| `apiGroup` | API 分组 | `apiGroup` | `<Tag>` |
| `name` | API 名称 | `name` | 文本 |
| `description` | API 描述 | `description` | `ellipsis` + `<Tooltip>` |
| `method` | HTTP 方法 | `method` | `<Tag>`：GET 蓝 / POST 绿 / PUT 橙 / DELETE 红 / PATCH 紫 |
| `actions` | 操作 | — | 编辑 / 删除 |

### 3.1 顶部筛选

| 字段 | 标签 | UI 组件 |
|------|------|--------|
| `path` | 路径 | `<Input>` |
| `name` | 名称 | `<Input>` |
| `apiGroup` | 分组 | `<Select>` 选项动态来自 `GET /admin/api-endpoints/groups` |
| `method` | 方法 | `<Select>` GET/POST/PUT/DELETE/PATCH |

### 3.2 顶部按钮

| 按钮 | 图标 | 行为 |
|------|------|------|
| 添加端点 | `PlusOutlined` | 打开创建表单 |
| 批量删除 | `DeleteOutlined` | 选中后启用，循环 `DELETE /api/v1/admin/api-endpoints/{id}` 或 `DELETE /admin/api-endpoints/batch` |
| 同步 | `SyncOutlined` | `POST /admin/api-endpoints/sync` 扫描 Gin 路由 |

---

## 4. 操作清单

| 操作 | 触发 | 接口 |
|------|------|------|
| 添加 | 顶部按钮 | `POST /admin/api-endpoints` |
| 编辑 | 行内 | `PUT /admin/api-endpoints/{id}` |
| 删除 | 行内 | `DELETE /admin/api-endpoints/{id}` |
| 批量删除 | 顶部 | `DELETE /admin/api-endpoints/batch`（Body：`{ "ids": [...] }`）|
| 同步 | 顶部 | `POST /admin/api-endpoints/sync` |

---

## 5. 表单字段定义（创建 / 编辑）

| name | 标签 | UI 组件 | 必填 | 校验 |
|------|------|--------|------|------|
| `path` | API 路径 | `<Input>` | ✅ | 以 `/` 开头；编辑模式只读建议 |
| `method` | HTTP 方法 | `<Select>` | ✅ | GET/POST/PUT/DELETE/PATCH |
| `apiGroup` | API 分组 | `<Select>` 可输入 | — | options 来自现有分组 + 自由输入 |
| `name` | API 名称 | `<Input maxLength=128>` | ✅ | |
| `description` | API 描述 | `<Input.TextArea>` | — | |

> **`is_auto` 字段不暴露**：手工创建时后端固定写 `false`，同步时写 `true`。

---

## 6. 接口契约（已实现）

> Base URL：`/api/v1/admin`（[handler.go:194-204](../../../internal/admin/handler.go#L194-L204)）

| Method | 路径 | 说明 |
|--------|------|------|
| GET | `/api-endpoints` | 分页列表 (`?page=&pageSize=&path=&method=&api_group=&name=`) |
| GET | `/api-endpoints/groups` | 全部分组名（下拉用）|
| POST | `/api-endpoints` | 创建（手工）|
| PUT | `/api-endpoints/{id}` | 编辑 |
| DELETE | `/api-endpoints/{id}` | 删除 |
| DELETE | `/api-endpoints/batch` | 批量删除 |
| POST | `/api-endpoints/sync` | 扫描 Gin 路由树同步入库（返回 `SyncApiResult`）|

### 6.1 同步逻辑

后端 service：

1. 通过依赖注入获取 `*gin.Engine` 的 `Routes()` 切片
2. 过滤掉非业务路由（如 `/healthz` / `/metrics` / 静态资源）
3. 与 DB 现有 `api_endpoints` 做 diff：
   - 新端点 → INSERT，`is_auto = true`，`name` / `apiGroup` 留空（管理员后续编辑补全）
   - 已存在端点 → 更新 `updated_at`（不覆盖 `name` / `apiGroup` 等手工编辑值）
   - DB 中存在但路由已删除 → **不自动删除**（保留以避免丢失角色权限绑定），UI 上可手动清理
4. 返回 `{ created: N, updated: M, total: K }`

---

## 7. 后端补齐 Backlog

### P0
1. **`role_api_permissions` DDL**（同 [roles.md §7 P0 #1](./roles.md)）

### P1
2. **同步规则可配置**：当前过滤逻辑硬编码，建议支持 `sys_configs` 配置黑名单 path 前缀
3. **失效端点标记**：DB 中存在但 `Gin.Routes()` 找不到的端点，标记为 `is_orphan = true`，UI 上展示警告 Tag
4. **API 调用统计**：与运行时中间件联动，记录每个端点最近 7 天调用量，列表页加 `recent_calls` 列辅助判断哪些端点可清理

### P2
5. **OpenAPI 导出**：把 `api_endpoints` 导出为 `swagger.json`，对接外部工具
6. **端点变更审计**：CRUD 写入 `audit_logs`

---

## 8. 验收清单（DoD）

后端：
- [ ] §6 全部接口已注册（已确认，handler.go:194-204）
- [ ] 同步功能不覆盖手工编辑的 `name` / `apiGroup`
- [ ] DELETE 前查 `role_api_permissions` 引用，提示影响的角色数

前端：
- [ ] 列表 + 创建/编辑/删除/同步全流程联通
- [ ] 同步成功后 toast 显示 `created/updated/total` 三个数字
- [ ] `npm run typecheck` & `lint` 通过
