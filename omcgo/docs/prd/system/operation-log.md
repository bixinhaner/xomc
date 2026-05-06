# 系统管理 — 操作日志（System / Operation Log）PRD

> 文档目的：以审计视角集中展示用户在系统内的写操作（who / what / when / where），支持搜索/筛选/导出。

| 版本 | 日期 | 作者 | 备注 |
|------|------|------|------|
| 0.1  | 2026-05-06 | Backend/Frontend Team | 基于现网 `OperationLog` + `internal/admin/handler.go:227 ListAuditLogs` 抽取 |

**关联功能域**：F06 OMC-R 核心 / 审计

**相关文件**：

| 层 | 路径 |
|----|------|
| 前端页面 | [omcmb/webcode/src/pages/log/OperationLog/index.tsx](../../../../omcmb/webcode/src/pages/log/OperationLog/index.tsx) |
| 系统管理路由 | `/system/operation-log` 直接 `export { default } from '@/pages/log/OperationLog'`（同一组件复用）|
| 后端 Handler | [omcgo/internal/admin/handler.go:227](../../../internal/admin/handler.go#L227) `ListAuditLogs` |
| 数据库 | `audit_logs` 表（[migrations/000002_users_roles.sql](../../../migrations/000002_users_roles.sql)）|

---

## 1. 业务背景

操作日志是 RBAC 系统的合规要求：所有写操作（创建/更新/删除/登录/授权变更）必须有审计记录，以应对安全事件追溯和合规审计。

**与系统日志（System Log）的区别**：
- 操作日志：**业务层**用户行为，强相关用户身份与业务对象
- 系统日志：**进程层**应用日志（来自 Zap），记录技术细节、错误堆栈

**写入入口（已落地）**：
- 登录 / 退出 / Token 刷新（`AdminService.Login`）
- 用户管理写操作（create/update/delete/lock/reset-password）
- 角色管理写操作
- 菜单/API 端点变更
- 配置变更

---

## 2. 实体模型

### 2.1 `audit_logs` 表（[migrations/000002_users_roles.sql](../../../migrations/000002_users_roles.sql)）

| 列 | 类型 | 约束 | 说明 |
|----|------|------|------|
| `id` | UUID | PK | |
| `user_id` | UUID | NULL, FK → `users(id)` ON DELETE SET NULL | 操作者；登录失败可能为空 |
| `username` | VARCHAR(64) | NOT NULL | 冗余记录用户名（user 删除后仍可查）|
| `action` | VARCHAR(64) | NOT NULL | 动作，如 `user.create` / `user.delete` / `role.assign` |
| `resource` | VARCHAR(64) | NULL | 资源类型，如 `user` / `role` / `device_group` |
| `resource_id` | VARCHAR(64) | NULL | 资源 ID（UUID 或其它）|
| `details` | JSONB | NULL | 任意附加上下文（before/after、参数等）|
| `ip_address` | INET / VARCHAR | NULL | 客户端 IP |
| `user_agent` | TEXT | NULL | UA 字符串 |
| `created_at` | TIMESTAMPTZ | NOT NULL DEFAULT NOW() | |

**索引**：
- `idx_audit_logs_user_id (user_id)`
- `idx_audit_logs_created_at (created_at DESC)` ← 列表默认按时间倒序
- 建议补：`idx_audit_logs_action (action)` / `idx_audit_logs_resource (resource, resource_id)`

### 2.2 后端 Go 模型（[model.go](../../../internal/admin/model.go#L65-L76)）

```go
type AuditLog struct {
    ID         uuid.UUID              `json:"id"`
    UserID     *uuid.UUID             `json:"user_id,omitempty"`
    Username   string                 `json:"username"`
    Action     string                 `json:"action"`
    Resource   string                 `json:"resource,omitempty"`
    ResourceID string                 `json:"resource_id,omitempty"`
    Details    map[string]interface{} `json:"details,omitempty"`
    IPAddress  string                 `json:"ip_address,omitempty"`
    UserAgent  string                 `json:"user_agent,omitempty"`
    CreatedAt  time.Time              `json:"created_at"`
}

type AuditLogFilter struct {
    UserID    *uuid.UUID `form:"user_id"`
    Action    *string    `form:"action"`
    Resource  *string    `form:"resource"`
    StartTime *string    `form:"start_time"`
    EndTime   *string    `form:"end_time"`
    model.ListRequest
}
```

### 2.3 前端类型（[frontend-core/src/types/system.ts](../../../../omcmb/frontend-core/src/types/system.ts) `OperationLog`）

```ts
type OperationResult = 'success' | 'failure' | 'partial';
type OperationType = 'create' | 'update' | 'delete' | 'query' | 'export' | 'import' | 'login' | 'logout' | 'execute' | 'deploy' | 'approve';

interface OperationLog {
  id: string;
  operator: string;          // ← username
  clientIp: string;          // ← ip_address
  module: string;            // ← resource
  operationType: OperationType;  // ← action 解析后的枚举
  target: string;            // ← resource_id
  content: string;           // ← details JSON 序列化
  result: OperationResult;
  message: string;
  operationTime: string;     // ← created_at
  logName?: string;
  detail?: string;
  reason?: string;
  startTime?: string;
  endTime?: string;
}
```

### 2.4 前后端字段映射 & Gap

| 前端字段 | 后端字段 | 状态 |
|---------|---------|------|
| `operator` | `username` | ✅（mapping 在 [adminApi.ts:303-321](../../../../omcmb/frontend-core/src/services/api/adminApi.ts#L303-L321)）|
| `clientIp` | `ip_address` | ✅ |
| `module` | `resource` | ✅ |
| `operationType` | `action` | ⚠️ 当前实现：`(ba.action \|\| 'query') as OperationType` —— 未做严格枚举映射，后端 `action` 是自由字符串如 `user.create`，前端强转为 `OperationType` 仅取 `query` 兜底；需补 action → operationType 的解析逻辑（如 `user.create → 'create'`）|
| `target` | `resource_id` | ✅ |
| `content` | `details` | ✅（JSON.stringify）|
| `result` | — | ❌ 后端无 result 字段；当前前端硬编码 `'success'`。建议后端补 `result VARCHAR(16)` |
| `message` | — | ❌ 后端无；建议补 `message TEXT NULL` 用于失败场景说明 |
| `logName` / `detail` / `reason` / `startTime` / `endTime` | — | ❌ 前端富审计字段，后端缺；建议保留前端可选字段，后端按需补充 |

---

## 3. 列表页字段定义

| key | 列标题 | dataIndex | UI 渲染 | 备注 |
|-----|-------|-----------|--------|------|
| `operationTime` | 操作时间 | `operationTime` | `toLocaleString('zh-CN')` | 默认按此排序倒序 |
| `operator` | 操作人 | `operator` | 文本 | |
| `module` | 模块 | `module` | `<Tag>` | 颜色按模块分类 |
| `operationType` | 操作类型 | `operationType` | `<Tag>`：create 绿 / delete 红 / update 蓝 / login 紫 | |
| `target` | 目标 | `target` | `ellipsis` | |
| `clientIp` | IP 地址 | `clientIp` | monospace | |
| `result` | 结果 | `result` | `<Tag>` success 绿 / failure 红 | |
| `actions` | 操作 | — | 查看（弹 Modal 显示完整 details JSON）| |

---

## 4. 操作清单

> 操作日志页面只读，无 add/edit/delete。

| 操作 | 触发 | 接口 |
|------|------|------|
| 查看详情 | 行内"查看" | 已有数据，前端弹 Modal 展示 `details`（JSON viewer） |
| 导出 | 顶部 | `GET /admin/audit-logs/export?<filters>`（**待补**）— 流式返回 CSV |

### 4.1 顶部筛选

| 字段 | 标签 | UI 组件 | 后端 query |
|------|------|--------|----------|
| `operator` | 操作人 | `<Select>` 远程搜索 | `?user_id=` |
| `module` | 模块 | `<Select>` | `?resource=` |
| `operationType` | 操作类型 | `<Select>` | `?action=` |
| 时间范围 | 时间范围 | `<RangePicker>` | `?start_time=&end_time=` |

---

## 5. 表单字段定义

无表单（只读页面）。

---

## 6. 接口契约

> Base URL：`/api/v1/admin`

### 6.1 已实现

| Method | 路径 | 说明 | Query |
|--------|------|------|-------|
| GET | `/audit-logs` | 分页列表 | `?page=&pageSize=&user_id=&action=&resource=&start_time=&end_time=` |

### 6.2 待补

| Method | 路径 | 说明 | 优先级 |
|--------|------|------|--------|
| GET | `/audit-logs/{id}` | 单条详情（含完整 details）| P1 |
| GET | `/audit-logs/export` | CSV 导出 | P1 |
| GET | `/audit-logs/actions` | 全量 action 枚举（下拉选项数据源）| P2 |

---

## 7. 后端补齐 Backlog

### P0
1. **写入覆盖率审计**：grep `service.go` 各业务方法，确认所有写操作（create/update/delete/lock/reset/assign）都已写入 audit_logs。当前 [users.md §10 DoD](./users.md#10) 已明确要求，但其它模块（roles/menus/api-endpoints/configs）需要逐一确认
2. **action 枚举规范化**：定义 `<resource>.<verb>` 格式（如 `user.create`、`role.assign_menu`、`config.update`），前端按 `.` 后段映射 `OperationType`

### P1
3. **`result` / `message` 字段**：DDL 增列 + handler 写入失败时记录错误信息
4. **保留期 + TimescaleDB 超表**：`audit_logs` 数据增长快（每个写操作一条），建议改为 TimescaleDB hypertable，按 30/90 天保留期自动清理
5. **导出接口**（§6.2）

### P2
6. **聚合视图**：按操作人/模块/时间维度聚合统计（合规报表）
7. **告警联动**：异常行为（如 5 分钟内 100 次失败登录）触发告警

---

## 8. 验收清单（DoD）

后端：
- [ ] 全模块写操作覆盖审计（grep 各 `service.go` 排查）
- [ ] action 字段使用规范格式 `<resource>.<verb>`
- [ ] 列表查询响应时间 < 500ms（千万级数据，依赖 TimescaleDB 改造）

前端：
- [ ] 筛选条件持久化到 URL query（刷新页面保留筛选）
- [ ] action → operationType 的映射逻辑补全
- [ ] details JSON viewer 支持折叠/搜索
- [ ] `npm run typecheck` & `lint` 通过
