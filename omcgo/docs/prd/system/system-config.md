# 系统管理 — 系统配置（System / Config）PRD

> 文档目的：管理系统级运行参数（密码策略、会话超时、SMTP、LDAP 等），按 9 个分类标签页组织。

| 版本 | 日期 | 作者 | 备注 |
|------|------|------|------|
| 0.1  | 2026-05-06 | Backend/Frontend Team | 基于现网 `SystemConfig/index.tsx` + `internal/admin/sys_config*.go` 抽取 |

**关联功能域**：F06 OMC-R 核心 / 全局基础设施

**相关文件**：

| 层 | 路径 |
|----|------|
| 前端页面 | [omcmb/webcode/src/pages/system/SystemConfig/index.tsx](../../../../omcmb/webcode/src/pages/system/SystemConfig/index.tsx) |
| 前端子组件（9 个）| `omcmb/webcode/src/pages/system/SystemConfig/{Basic,Security,Device,Notification,Storage,Omc,Northbound,Sas,Ldap}Settings.tsx` |
| 后端 Handler | [omcgo/internal/admin/sys_config_handler.go](../../../internal/admin/sys_config_handler.go)（推断；service 在 sys_config.go）|
| 后端 Repo | [omcgo/internal/admin/sys_config.go](../../../internal/admin/sys_config.go) |
| 数据库 | [omcgo/migrations/000009_sys_admin.sql:113-126](../../../migrations/000009_sys_admin.sql#L113-L126) |
| 路由注册 | [cmd/app/provider/router.go:402](../../../cmd/app/provider/router.go#L402) `ad.sysConfigHandler.RegisterRoutes(adminGroup)` |

---

## 1. 业务背景

系统配置（`sys_configs`）是 KV 配置存储，按 `category` 分类。9 个分类（标签页）：

| 标签 key | 中文名 | 用途 |
|---------|--------|------|
| `basic` | 基础配置 | 系统名称、版本、Logo、主题色、时区 |
| `security` | 安全 | 密码最小长度、最大登录尝试、锁定时长、会话超时 |
| `device` | 设备 | 设备超时、重连间隔、心跳周期默认值 |
| `notify` | 通知 | SMTP 服务器、SMS 网关、Webhook URL |
| `storage` | 存储 | MinIO endpoint、bucket、保留期 |
| `omc` | OMC | OMC 系统级参数（与 ACS 协调）|
| `northbound` | 北向 | OSS 对接参数 |
| `sas` | SAS | SAS（Spectrum Access System）参数 |
| `ldap` | LDAP | LDAP server URL、bind DN、search base |

**关键差异 vs 字典**：
- 字典 = 业务枚举（每个枚举值是一行）
- 系统配置 = 系统参数（每个参数是一行）

---

## 2. 实体模型

### 2.1 `sys_configs` 表（[migrations/000009_sys_admin.sql:113-126](../../../migrations/000009_sys_admin.sql#L113-L126)）

| 列 | 类型 | 约束 | 说明 |
|----|------|------|------|
| `id` | BIGSERIAL | PK | |
| `category` | VARCHAR(32) | NOT NULL | 分组（basic/security/device/...）|
| `key` | VARCHAR(64) | NOT NULL | 参数名 |
| `value` | TEXT | NULL | 参数值（按 `value_type` 解析）|
| `value_type` | VARCHAR(16) | NOT NULL | `string` / `int` / `float` / `bool` / `json` |
| `description` | TEXT | NULL | 中文说明 |
| `is_public` | BOOLEAN | NOT NULL DEFAULT FALSE | 是否对前端登录前 API 可见（如系统名称需要）|
| `created_at` / `updated_at` | TIMESTAMPTZ | NOT NULL | |

唯一约束：`UNIQUE (category, key)`

**已 seed 的初始配置**：
- `('system', 'system_name', 'OMC 网管系统', 'string')`
- `('system', 'system_version', '1.0.0', 'string')`
- `('system', 'session_timeout', '30', 'int')`
- `('system', 'max_login_attempts', '5', 'int')`
- `('system', 'lockout_duration', '30', 'int')`
- `('system', 'password_min_length', '6', 'int')`

> ⚠️ seed 中 category 都是 `'system'`，与前端 9 分类不完全匹配。需统一：要么 seed 拆开（`security:max_login_attempts`），要么前端把 `system` 也作为合法 category。

---

## 3. 页面布局（Tab 切换）

### 3.1 顶部

- 标题：「系统配置」
- 右侧按钮：`保存`（验证当前 Tab 表单 + 调 `PUT /admin/configs`）

### 3.2 Tab 列表（9 个）

每个 Tab 对应一个子组件 Form。**字段定义在各子组件内**（本 PRD 不展开各子组件细节，列出预期字段集）。

#### `basic` BasicSettings — 基础配置

| key | 标签 | UI | value_type |
|-----|------|----|-----------|
| `system_name` | 系统名称 | Input | string |
| `system_version` | 系统版本 | Input(readonly) | string |
| `system_logo` | Logo URL | Upload | string |
| `theme_color` | 主题色 | ColorPicker | string |
| `timezone` | 时区 | Select | string |

#### `security` SecuritySettings — 安全

| key | 标签 | UI | value_type | 备注 |
|-----|------|----|-----------|------|
| `password_min_length` | 密码最小长度 | InputNumber min=6 max=32 | int | |
| `password_complexity` | 密码复杂度 | Checkbox.Group（大小写/数字/特殊字符）| json | |
| `max_login_attempts` | 最大登录尝试次数 | InputNumber min=3 | int | |
| `lockout_duration` | 锁定时长（分钟）| InputNumber | int | |
| `session_timeout` | 会话超时（分钟）| InputNumber | int | |
| `password_history_size` | 密码历史不可重复次数 | InputNumber | int | |
| `force_password_change_days` | 强制改密周期（天）| InputNumber | int | 0 = 不强制 |

#### `device` DeviceSettings — 设备

| key | 标签 | UI | value_type |
|-----|------|----|-----------|
| `device_offline_threshold_sec` | 离线判定阈值（秒）| InputNumber | int |
| `inform_interval_default` | 默认 Inform 周期 | InputNumber | int |
| `connreq_timeout_sec` | Connection Request 超时 | InputNumber | int |

#### `notify` NotificationSettings — 通知

| key | 标签 | UI | value_type |
|-----|------|----|-----------|
| `smtp_host` | SMTP 服务器 | Input | string |
| `smtp_port` | SMTP 端口 | InputNumber | int |
| `smtp_username` | SMTP 用户名 | Input | string |
| `smtp_password` | SMTP 密码 | Input.Password | string |
| `smtp_from` | 发件人 | Input | string |
| `smtp_use_tls` | 启用 TLS | Switch | bool |
| `sms_gateway_url` | SMS 网关 URL | Input | string |
| `webhook_url` | Webhook URL | Input | string |

> ⚠️ 与 [notification-settings.md](./notification-settings.md) 的「通知规则/收件人/模板」是不同概念：本 Tab 管"通道凭证"，notification-settings.md 管"业务规则"。

#### `storage` StorageSettings

| key | 标签 | UI | value_type |
|-----|------|----|-----------|
| `minio_endpoint` | MinIO 地址 | Input | string |
| `minio_bucket_pm` | PM 桶 | Input | string |
| `minio_bucket_mr` | MR 桶 | Input | string |
| `minio_bucket_firmware` | 固件桶 | Input | string |
| `pm_retention_days` | PM 文件保留天数 | InputNumber | int |
| `mr_retention_days` | MR 文件保留天数 | InputNumber | int |

#### `omc` / `northbound` / `sas` / `ldap`

依模块需求，字段集由对应子组件定义。LDAP 配置示例：

| key | 标签 | UI | value_type |
|-----|------|----|-----------|
| `ldap_enabled` | 启用 LDAP | Switch | bool |
| `ldap_server_url` | LDAP URL | Input | string |
| `ldap_bind_dn` | Bind DN | Input | string |
| `ldap_bind_password` | Bind 密码 | Input.Password | string |
| `ldap_search_base` | 搜索基 | Input | string |
| `ldap_user_filter` | 用户过滤器 | Input | string |
| `ldap_sync_interval_min` | 同步周期（分钟）| InputNumber | int |

---

## 4. 操作清单

| 操作 | 触发 | 接口 |
|------|------|------|
| 切换 Tab | Tab 点击 | 拉取该分类的配置：`GET /admin/configs?category=<tab>` |
| 保存当前 Tab | 顶部"保存" | 校验表单 → `PUT /admin/configs/batch`（一次写入该分类下所有变更项）|

---

## 5. 表单字段定义

> 见 §3.2 各子组件字段表。所有字段共用 `<Form>` 校验规则（required / min / max / pattern）。

---

## 6. 接口契约

> Base URL：`/api/v1/admin`（已挂载，[router.go:402](../../../cmd/app/provider/router.go#L402)）

### 6.1 已实现（[sys_config.go](../../../internal/admin/sys_config.go)）

推断的 handler 接口（实际以 `sys_config_handler.go` 注册的为准）：

| Method | 路径 | 说明 |
|--------|------|------|
| GET | `/configs` | 全量列表（admin 视图） |
| GET | `/configs?category={cat}` | 按分类查 |
| GET | `/configs/{category}/{key}` | 单参数查 |
| POST | `/configs` | 创建（一般只在初始化时用，运行期通过 PUT）|
| PUT | `/configs/{id}` | 单参数更新 |
| DELETE | `/configs/{id}` | 删除（一般禁用，避免运行时丢配置）|

### 6.2 待补：批量更新

| Method | 路径 | 说明 | 优先级 |
|--------|------|------|--------|
| PUT | `/configs/batch` | 一次写入多个 `(category, key, value)`，事务性 | P0 |

### 6.3 公开配置（登录前可读）

| Method | 路径 | 说明 |
|--------|------|------|
| GET | `/public/configs` | 仅返回 `is_public = true` 的配置（如系统名称、Logo），无需 auth |

---

## 7. 后端补齐 Backlog

### P0
1. **批量更新接口** `PUT /configs/batch` —— 当前只能逐个 PUT，前端"保存"按钮一次提交多个字段时若中途失败会留下脏数据
2. **value_type 强制校验** —— 后端按声明类型解析 value，类型不匹配返回 400（如 `bool` 字段写入 `"yes"` 应拒绝）
3. **seed 数据 category 调整**（§2.1 已记）

### P1
4. **配置变更审计** —— 写入 `audit_logs`，含 `before` / `after` 值（密码类字段脱敏）
5. **配置变更广播** —— 关键配置（如 `session_timeout`、`max_login_attempts`）变更后通过 EventBus 通知所有运行实例热加载（避免重启）
6. **敏感字段加密存储** —— `smtp_password` / `ldap_bind_password` 等在 DB 加密，handler 层解密

### P2
7. **配置版本快照** —— 每次保存生成快照，支持回滚
8. **配置导出/导入** —— 跨环境同步（开发 → 生产）

---

## 8. 验收清单（DoD）

后端：
- [ ] `PUT /configs/batch` 实现 + 事务性
- [ ] value_type 校验在 service 层落地（拒绝非法类型）
- [ ] 敏感配置（password 类）API 响应脱敏（返回 `****` 而非明文）

前端：
- [ ] 9 个 Tab 子组件全部实现（当前可能部分占位）
- [ ] 切换 Tab 时若当前 Tab 有未保存修改，弹确认对话框
- [ ] 保存成功后 toast + 重新拉取该分类配置（确保展示最新）
- [ ] `npm run typecheck` & `lint` 通过
