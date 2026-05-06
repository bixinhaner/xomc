# 系统管理 — 通知设置（System / Notification Settings）PRD

> 文档目的：管理事件触发的通知规则、收件人分组、邮件模板。
> ⚠️ 当前前端纯 mock 数据，后端**尚未实现规则/收件人/模板模块**（`internal/notification/` 是消息中心，与本 PRD 不同）。

| 版本 | 日期 | 作者 | 备注 |
|------|------|------|------|
| 0.1  | 2026-05-06 | Backend/Frontend Team | 现状记录 + 完整后端落地方案 |

**关联功能域**：F04 告警管理 / F06 OMC-R 核心 / 通知中枢

**相关文件**：

| 层 | 路径 |
|----|------|
| 前端页面 | [omcmb/webcode/src/pages/system/NotificationSettings/index.tsx](../../../../omcmb/webcode/src/pages/system/NotificationSettings/index.tsx) |
| 前端路由 | （未挂载到 routes.tsx）|
| 后端（待实现）| `internal/notification_rules/` 或扩展 `internal/notification/` |
| 后端（已存在）| [internal/notification/handler.go](../../../internal/notification/handler.go) — 这是**消息中心**（用户收件箱），不是通知规则配置 |

---

## 1. 业务背景

### 1.1 概念辨析

| 模块 | 用途 | 当前实现 |
|------|------|---------|
| **本 PRD：通知设置** | 配置「什么事件 → 通过什么渠道 → 通知谁 → 用什么模板」| ❌ 未实现 |
| `internal/notification/`（消息中心）| 用户登录后查看自己收到的站内消息 | ✅ 已实现 `/notifications`、`/notifications/unread-count` |
| 全局 SMTP/SMS 凭证 | 通道连接参数 | ✅ 在 `sys_configs` 中（[system-config.md](./system-config.md) `notify` Tab）|

**串联**：

```
事件发生（告警/设备离线/用户登录失败 ...）
  ↓
本 PRD：匹配通知规则
  ↓
查找规则对应的「收件人分组」
  ↓
按规则配置的「渠道」（email/sms/站内）+ 「模板」格式化
  ↓
分发：
  - email/sms → 调用全局凭证发送
  - 站内 → 写入 `internal/notification/` 的 notifications 表（用户登录后看到）
```

### 1.2 用户

- **管理员**：配置规则（如"严重告警 → 邮件给运维组"）
- **运维收件人**：被配置在收件人分组里，被动接收通知

---

## 2. 实体模型（待实现）

### 2.1 `notification_rules` 表

| 列 | 类型 | 约束 | 说明 |
|----|------|------|------|
| `id` | UUID | PK | |
| `name` | VARCHAR(128) | NOT NULL | 规则名 |
| `trigger_event` | VARCHAR(64) | NOT NULL | 触发事件（枚举：见 §2.4）|
| `event_filter` | JSONB | NULL | 事件过滤条件（如告警严重级别 = 严重）|
| `channels` | TEXT[] | NOT NULL | 渠道：`email` / `sms` / `inbox`（站内）/ `webhook` |
| `recipient_group_ids` | UUID[] | NOT NULL | 关联收件人分组 |
| `template_id` | UUID | NULL | 邮件/SMS 模板 |
| `enabled` | BOOLEAN | NOT NULL DEFAULT TRUE | 启用状态 |
| `cooldown_sec` | INT | NOT NULL DEFAULT 0 | 同事件冷却时间（避免告警风暴）|
| `created_by` / `updated_by` | UUID | NULL | 审计 |
| `created_at` / `updated_at` | TIMESTAMPTZ | NOT NULL | |

### 2.2 `notification_recipient_groups` 表

| 列 | 类型 | 约束 | 说明 |
|----|------|------|------|
| `id` | UUID | PK | |
| `name` | VARCHAR(128) | NOT NULL UNIQUE | 分组名 |
| `description` | TEXT | NULL | |
| `emails` | TEXT[] | NOT NULL DEFAULT `'{}'` | 邮箱列表 |
| `phones` | TEXT[] | NOT NULL DEFAULT `'{}'` | 手机号列表 |
| `user_ids` | UUID[] | NULL | 关联系统用户 ID（自动取其邮箱/手机）|
| `created_at` / `updated_at` | TIMESTAMPTZ | NOT NULL | |

### 2.3 `notification_templates` 表

| 列 | 类型 | 约束 | 说明 |
|----|------|------|------|
| `id` | UUID | PK | |
| `name` | VARCHAR(128) | NOT NULL UNIQUE | 模板名 |
| `channel` | VARCHAR(16) | NOT NULL | `email` / `sms` |
| `subject` | VARCHAR(512) | NULL | 邮件主题（仅 email）|
| `body` | TEXT | NOT NULL | 模板正文，支持变量占位符 `{{ .device_sn }}`、`{{ .severity }}` 等 |
| `variables` | JSONB | NULL | 变量定义（用于编辑器提示）|
| `created_at` / `updated_at` | TIMESTAMPTZ | NOT NULL | |

### 2.4 触发事件枚举（建议）

| 事件 key | 说明 |
|----------|------|
| `alarm.raised` | 新告警 |
| `alarm.cleared` | 告警清除 |
| `alarm.severity_critical` | 严重告警（与上面是细分粒度）|
| `device.offline` | 设备离线 |
| `device.online` | 设备上线 |
| `device.firmware.upgrade_failed` | 固件升级失败 |
| `user.login_failed_threshold` | 用户登录失败超阈值 |
| `system.disk_full_warning` | 磁盘空间告警 |
| `pm.collection_failed` | PM 文件采集失败 |

---

## 3. 页面布局（3 Tab）

### 3.1 Tab 1：规则（Rules）

| key | 列标题 | UI |
|-----|-------|------|
| `name` | 规则名称 | 文本 |
| `triggerEvent` | 触发事件 | `<Tag>` |
| `channels` | 通知渠道 | `<Tag>` 多个：email/sms/inbox |
| `recipientGroups` | 收件人分组 | 多个 `<Tag>` |
| `enabled` | 启用 | `<Switch>` |
| `actions` | 操作 | 编辑 / 删除 |

顶部按钮：`添加规则`

### 3.2 Tab 2：收件人分组（Groups）

| key | 列标题 | UI |
|-----|-------|------|
| `name` | 分组名称 | 文本 |
| `emails` | 邮箱列表 | 多行（折叠展示前 N 个）|
| `phones` | 电话列表 | 同上 |
| `memberCount` | 成员数 | 数字 |
| `actions` | 操作 | 编辑 / 删除 |

顶部按钮：`添加分组`

### 3.3 Tab 3：模板（Templates）

| key | 列标题 | UI |
|-----|-------|------|
| `name` | 模板名 | 文本 |
| `channel` | 渠道 | `<Tag>` email / sms |
| `subject` | 主题预览 | `ellipsis` |
| `actions` | 操作 | 编辑 / 删除 |

顶部按钮：`添加模板`

---

## 4. 操作清单

### 4.1 规则

| 操作 | 触发 | 接口（待实现）|
|------|------|------------|
| 添加 | 顶部 | `POST /admin/notification-rules` |
| 编辑 | 行内 | `PUT /admin/notification-rules/{id}` |
| 删除 | 行内 | `DELETE /admin/notification-rules/{id}` |
| 启用切换 | 行 `<Switch>` | `PUT /admin/notification-rules/{id}` 仅改 `enabled` |
| 试运行 | 行内"测试" | `POST /admin/notification-rules/{id}/test` 触发一条测试通知 |

### 4.2 收件人分组

| 操作 | 触发 | 接口 |
|------|------|------|
| 添加 | 顶部 | `POST /admin/notification-recipient-groups` |
| 编辑 | 行内 | `PUT /admin/notification-recipient-groups/{id}` |
| 删除 | 行内 | `DELETE /admin/notification-recipient-groups/{id}` |

### 4.3 模板

| 操作 | 触发 | 接口 |
|------|------|------|
| 添加 | 顶部 | `POST /admin/notification-templates` |
| 编辑 | 行内 | `PUT /admin/notification-templates/{id}` |
| 删除 | 行内 | `DELETE /admin/notification-templates/{id}` |
| 预览 | 编辑面板按钮 | `POST /admin/notification-templates/{id}/render`（带样本变量返回渲染后内容）|

---

## 5. 表单字段定义

### 5.1 规则 Form

| name | 标签 | UI 组件 | 必填 |
|------|------|--------|------|
| `name` | 规则名称 | `<Input maxLength=128>` | ✅ |
| `triggerEvent` | 触发事件 | `<Select>` 选项来自 §2.4 | ✅ |
| `eventFilter` | 事件过滤 | 动态表单（按 event 类型展示，如告警事件展示 severity 多选）| — |
| `channels` | 通知渠道 | `<Checkbox.Group>` email/sms/inbox/webhook | ✅ |
| `recipientGroupIds` | 收件人分组 | `<Select mode="multiple">` 选项来自 `GET /admin/notification-recipient-groups` | ✅ |
| `templateId` | 模板 | `<Select>` 按 channel 过滤的模板 | — |
| `cooldownSec` | 冷却时间（秒）| `<InputNumber min=0>` | — |
| `enabled` | 启用 | `<Switch>` | — 默认 ON |

### 5.2 收件人分组 Form

| name | 标签 | UI | 必填 |
|------|------|----|------|
| `name` | 分组名称 | `<Input>` | ✅ |
| `description` | 描述 | `<Input.TextArea>` | — |
| `emails` | 邮箱列表 | `<Select mode="tags">` 输入回车成 tag | — |
| `phones` | 手机列表 | `<Select mode="tags">` 校验 `/^1\d{10}$/` | — |
| `userIds` | 关联系统用户 | `<Select mode="multiple">` 选项远程搜索 `/admin/users` | — |

### 5.3 模板 Form

| name | 标签 | UI | 必填 |
|------|------|----|------|
| `name` | 模板名 | `<Input>` | ✅ |
| `channel` | 渠道 | `<Radio.Group>` email / sms | ✅，编辑后不可改 |
| `subject` | 主题 | `<Input>` | email 必填 |
| `body` | 正文 | `<Input.TextArea rows=10>` 或 Markdown 编辑器 | ✅ |

> 模板使用 Go template 语法，前端编辑器可加变量插入按钮。

---

## 6. 接口契约（全部待实现）

> Base URL：`/api/v1/admin`（建议挂在 admin 分组下）

### 6.1 通知规则

| Method | 路径 |
|--------|------|
| GET | `/notification-rules` |
| GET | `/notification-rules/{id}` |
| POST | `/notification-rules` |
| PUT | `/notification-rules/{id}` |
| DELETE | `/notification-rules/{id}` |
| POST | `/notification-rules/{id}/test` |

### 6.2 收件人分组

| Method | 路径 |
|--------|------|
| GET | `/notification-recipient-groups` |
| GET | `/notification-recipient-groups/{id}` |
| POST | `/notification-recipient-groups` |
| PUT | `/notification-recipient-groups/{id}` |
| DELETE | `/notification-recipient-groups/{id}` |

### 6.3 模板

| Method | 路径 |
|--------|------|
| GET | `/notification-templates` |
| GET | `/notification-templates/{id}` |
| POST | `/notification-templates` |
| PUT | `/notification-templates/{id}` |
| DELETE | `/notification-templates/{id}` |
| POST | `/notification-templates/{id}/render` |

---

## 7. 后端补齐 Backlog

### P0（本菜单全部都是 P0，因为完全未实现）

1. **3 张 DDL 落地**（§2.1-§2.3）
2. **handler / service / repository 三层** 实现 §6 全部接口
3. **路由挂载**：`cmd/app/provider/router.go` 注册 `notificationRulesHandler.RegisterRoutes(adminGroup)`
4. **前端路由**：`/system/notification-settings` 挂到 `routes.tsx`，i18n 加 `nav.system.notificationSettings`

### P1

5. **事件发布对接**：在 EventBus 上订阅 §2.4 所列事件，匹配规则后触发通知派发
6. **派发器**：实现 email/sms/inbox/webhook 4 个渠道的 sender，复用 `sys_configs` 中 SMTP/SMS 凭证
7. **Go template 渲染**：解析 `{{ .var }}` 占位符，支持事件 payload 注入
8. **冷却逻辑**：Redis 计数器实现 `cooldown_sec`，避免告警风暴
9. **失败重试 + 死信队列**：派发失败入 `notification_dispatch_failures` 表，UI 上有"通知历史"页查看（与 `internal/notification/history_*` 联动）

### P2

10. **规则优先级**：多规则匹配同一事件时按 priority 排序
11. **静默时段**：分组级"周末不发"、"夜间不发"等
12. **模板版本/AB 测试**

---

## 8. 验收清单（DoD）

后端：
- [ ] 3 张 DDL 迁移落地 + Down 完整
- [ ] §6 全部接口单测覆盖
- [ ] 触发一条告警 → 邮件能真实发送（依赖 `sys_configs` SMTP 凭证）
- [ ] 冷却时间生效（同事件 5 秒内只发一次）

前端：
- [ ] 三个 Tab 都接入真实 API（不再 mock）
- [ ] 规则编辑面板按 `triggerEvent` 动态渲染 `eventFilter` 子表单
- [ ] 模板编辑器支持变量插入
- [ ] 路由 `/system/notification-settings` 挂载到 routes.tsx
- [ ] `npm run typecheck` & `lint` 通过

---

## 9. 非目标

- 通知历史 / 派发记录 — 由扩展的 `internal/notification/history_*` 承担（已部分实现）
- 用户站内消息收件箱 — 由 `internal/notification/handler.go` `/notifications` 承担（已实现）
- SMS 实际发送通道（阿里云/腾讯云）— 由 sys_configs 配置 + 通道适配器承担，本 PRD 假设有 `SmsService` 接口可调
