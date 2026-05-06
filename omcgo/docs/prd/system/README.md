# 系统管理（System）— PRD 索引

> OMC 系统管理模块下 12 个菜单的需求文档集。每份 PRD 独立、自洽，覆盖前后端实现现状 + Gap + 后端补齐 Backlog。

| # | 菜单中文名 | PRD 文件 | 路由 | 后端实现度 | 前端实现度 | 备注 |
|---|----------|---------|------|-----------|-----------|------|
| 1 | 系统仪表板 | [dashboard.md](./dashboard.md) | `/system/dashboard` | 🟡 部分（`/system/info` 已注册）| 🟢 完整 | 数据来源含 `gopsutil` + 业务聚合查询 |
| 2 | 用户管理 | [users.md](./users.md) | `/system/users` | 🟢 完整（v0.1-v0.5 已落地）| 🟢 完整 | 含 `source` 字段 + 数据权限模型说明 |
| 3 | 用户组管理 | [groups.md](./groups.md) | `/system/groups` | ⚠️ 计划下线 | ⚠️ 计划下线 | v0.3 决议：术语统一为「角色」，下版本删除 |
| 4 | 角色管理 | [roles.md](./roles.md) | `/system/roles` | 🟢 完整（CRUD + 菜单/API/分组）| 🟢 完整 | API 权限 DDL `role_api_permissions` 待补 |
| 5 | 菜单管理 | [menus.md](./menus.md) | `/system/menus` | 🟢 完整（CRUD + tree）| 🔴 前端 mock | P0：前端对接真实 API |
| 6 | 操作日志 | [operation-log.md](./operation-log.md) | `/system/operation-log` | 🟢 完整 | 🟢 完整 | 路由复用 `/pages/log/OperationLog` |
| 7 | 系统配置 | [system-config.md](./system-config.md) | `/system/config` | 🟡 部分（缺批量更新）| 🟡 9 Tab 框架 | P0：批量更新接口 + value_type 校验 |
| 8 | UI 定制化 | [ui-customization.md](./ui-customization.md) | `/system/ui-custom` | 🟡 复用 sys_configs | 🟡 框架 | 依赖 system-config 的 P0 |
| 9 | API 管理 | [api-management.md](./api-management.md) | `/system/api-management` | 🟢 完整（含 sync）| 🟢 完整 | 与角色 API 权限联动 |
| 10 | 字典管理 | [data-dictionary.md](./data-dictionary.md) | `/system/data-dictionary` | 🟢 完整（双面板 CRUD）| 🟢 完整 | 待确认 dictionary handler 路由挂载 |
| 11 | 设备分类 | [device-classification.md](./device-classification.md) | `/system/device-class` | 🔴 缺专项 API | 🔴 前端 mock | P0：派生分类树聚合接口 |
| 12 | 通知设置 | [notification-settings.md](./notification-settings.md) | （路由未挂）| 🔴 完全缺失 | 🔴 前端 mock | P0：3 张 DDL + 全套 CRUD + EventBus 对接 |

---

## 跨 PRD 关键关系

### 数据权限派生链（用户 → 角色 → 设备分组）

详见 [users.md §1.3 / §11.6 / §11.7](./users.md#13-数据权限模型用户--角色--设备分组)：

```
User (users.md)
  └─ user_roles
       └─ Role (roles.md)
            ├─ role_menus → menus (menus.md)
            ├─ role_api_permissions → api_endpoints (api-management.md)
            └─ role_device_groups → device_groups (F06 拓扑管理 PRD，本系列外)
```

### 配置数据共享链

```
sys_configs（system-config.md 主管）
  ├─ category=basic/security/device/notify/storage/...   ← system-config.md
  └─ category=ui_custom                                  ← ui-customization.md
```

### 通知派发链

```
事件发生（告警/设备离线/...）
  ↓
notification_rules (notification-settings.md, 待实现)
  ↓
notification_recipient_groups + notification_templates
  ↓
派发到 SMTP/SMS（凭证来自 system-config.md `notify` Tab）
  或 inbox（写入 internal/notification/, 用户登录后查看）
```

---

## P0 / P1 / P2 后端补齐汇总

> 按 PRD 来源整理，未跨 PRD 去重。

### P0（阻塞前端 UI 已展示功能）

| 任务 | 来源 PRD |
|------|---------|
| 前端 `MenuManagement` 对接真实 API（替换 mock）| [menus.md §7](./menus.md) |
| 系统配置批量更新接口 `PUT /admin/configs/batch` | [system-config.md §7](./system-config.md) |
| `role_api_permissions` DDL 落地 | [roles.md §7](./roles.md) / [api-management.md §7](./api-management.md) |
| 设备分类树聚合接口 `/device-classifications/tree` | [device-classification.md §7](./device-classification.md) |
| 通知规则 / 收件人 / 模板 三张 DDL + CRUD | [notification-settings.md §7](./notification-settings.md) |
| 角色 `userCount` / `built_in` 派生字段 | [roles.md §7](./roles.md) |
| 状态/类型枚举对齐（前端 active ↔ 后端 normal）| [menus.md §7](./menus.md) |
| dictionary handler 路由挂载确认 | [data-dictionary.md §7](./data-dictionary.md) |

### P1

| 任务 | 来源 PRD |
|------|---------|
| 内置用户/角色 source / built_in 字段 + 权限矩阵实现（已大部分完成）| [users.md §7](./users.md) |
| 数据权限缓存失效（PermissionService.InvalidateUserCache）覆盖所有写操作 | [users.md §10 / §11.7](./users.md) |
| 角色未绑设备分组的 UI 警告 | [roles.md §7](./roles.md) |
| 设备分组删除时联动审计 + 角色侧通知 | [users.md §11.7 决议 ③](./users.md) |
| 网络制式（network_types）UI 与 DDL 语义对齐 | [roles.md §7](./roles.md) |
| 操作日志增加 result / message 字段 + TimescaleDB 改造 | [operation-log.md §7](./operation-log.md) |
| API 端点变更审计 + 失效端点标记 | [api-management.md §7](./api-management.md) |
| 通知派发实现（email/sms/inbox 4 渠道）+ 冷却逻辑 | [notification-settings.md §7](./notification-settings.md) |

### P2

| 任务 | 来源 PRD |
|------|---------|
| 字典批量导入 / 多语言 | [data-dictionary.md §7](./data-dictionary.md) |
| 系统配置版本快照 / 跨环境导入导出 | [system-config.md §7](./system-config.md) |
| UI 定制按 carrier 多租户 | [ui-customization.md §7](./ui-customization.md) |
| API 端点 OpenAPI 导出 | [api-management.md §7](./api-management.md) |
| 通知规则优先级 / 静默时段 | [notification-settings.md §7](./notification-settings.md) |
| 角色克隆 | [roles.md §7](./roles.md) |
| 自定义设备分类 | [device-classification.md §7](./device-classification.md) |

---

## 通用约定

### 文件路径引用

PRD 内引用代码用 markdown 链接，相对路径从 PRD 文件出发：
- 后端：`../../../internal/admin/...`
- 前端：`../../../../omcmb/webcode/...` / `../../../../omcmb/frontend-core/...`
- 迁移：`../../../migrations/...`

### Gap 标记图例

- ✅ 已实现 / 已对齐
- ⚠️ 部分实现 / 命名错位 / 类型不匹配
- ❌ 未实现 / 缺字段
- 🟢 / 🟡 / 🔴 完成度评级（绿 = 80%+，黄 = 30%-80%，红 = < 30%）

### 接口命名约定

- Base URL：`/api/v1/admin`（admin 鉴权）/ `/api/v1`（普通鉴权）/ `/api/v1/public`（无鉴权）
- 路径风格：kebab-case（`api-endpoints`、`notification-rules`）
- Body 字段：snake_case（后端）↔ camelCase（前端，由 axios 拦截器自动转换）
- ID：UUID（统一 `gen_random_uuid()`）

### 数据权限副作用约束

任何会改变用户可见域的写操作（角色绑定/解绑、菜单/API/分组绑定变更、用户角色变更）：

> **必须**同步调 `PermissionService.InvalidateUserCache(userID)`，**不引入** EventBus（[users.md §11.7 决议 ②](./users.md)）。

---

## 维护规范

1. **新建 PRD**：复制 [users.md](./users.md) 结构（§1-§11），按本系列体例补全
2. **更新现有 PRD**：在文档头版本表加新行，保留历史决议（§11 决策记录章节）
3. **下线菜单**：写明下线计划与时间表，参考 [groups.md](./groups.md)
4. **跨 PRD 引用**：用相对路径 + 锚点，如 `[users.md §1.3](./users.md#13-数据权限模型用户--角色--设备分组)`

---

**生成时间**：2026-05-06
**维护方**：Backend / Frontend Team
**关联功能域**：F06 OMC-R 核心
