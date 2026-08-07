# PRD: F04 告警通知链路

> 历史 Draft：本文保留用于 backlog 追溯，不代表当前交付范围。当前告警邮件、KPI 定时报表、短信排除项和验收门禁以 `docs/superpowers/reviews/2026-08-06-email-notification-scope-convergence.md` 及对应 milestone acceptance 为准。

**PRD ID**：F04-alarm-notification  
**功能域**：F04 告警管理  
**作者**：Claude AI（PM 角色）+ 待人工审批  
**创建日期**：2026-04-20  
**最后更新**：2026-04-20  
**状态**：Draft（待审批）  
**关联 Milestone**：`docs/project/milestone/2026Q2-to-RC.md`  
**关联 Sprint**：Sprint-01（邮件）+ Sprint-02（Webhook）+ Sprint-03（短信）  
**关联 Risk**：`docs/project/risk-register.md#R-001`

---

## 1. 业务背景

当前 F04 告警模块已实现接收、去重、规则引擎、生命周期管理，规则支持 `action="notify"`，但**下游通知渠道完全未实现**——邮件/短信/Webhook/移动推送全无。这意味着：

- CPE 上报严重告警后，系统能正确识别、去重、按规则分类，但**运维人员不会被通知**
- 运营商运维场景下，告警需要 1 分钟内送达值班人员，否则故障恢复 SLA 不达标
- 这是从 Beta 进入 RC 必须关闭的短板（`risk-register.md#R-001`，P0）

不做就等于系统告警能力"摆设" — 完整接收链路却无最后一公里触达。

---

## 2. 用户故事

> **运维工程师**：As a 运维工程师，I want 严重告警发生时在 **1 分钟内**通过邮件（和/或短信）收到通知，So that 我能及时介入处理，不超时 SLA。

> **运维主管**：As a 运维主管，I want 按告警**严重等级、设备组、时段**订阅通知，So that 我不会被低优先级告警淹没，值班夜不被无关告警吵醒。

> **系统集成人员**：As a 集成人员，I want 把告警通过 **Webhook** 推到第三方工单系统（如 Jira / ServiceNow / 运营商工单），So that 工单自动化闭环。

> **运营商网管**：As a 运营商网管（CMCC/CTCC/CUCC），I want 告警内容、格式、字段符合**各自运营商的网管规范**，So that 与上游 OSS 能顺畅对接。

---

## 3. 验收标准

### AC-1：邮件通知基本可用（Sprint-01）
```
Given: 已配置一个 SMTP 渠道 + 一条规则 action="notify" 指向该渠道
When:  触发匹配该规则的告警（严重级别 critical）
Then:  - 告警入库 activeAlarm 表
       - 邮件在 30 秒内发送成功（日志可查 MessageID）
       - 邮件主题含设备 SN + 告警码
       - 邮件正文含：时间戳、设备、告警码、级别、附加详情
       - 邮件模板支持 i18n（zh-CN / en-US 两份模板）
```

### AC-2：Webhook 通知（Sprint-02）
```
Given: 已配置一个 Webhook URL + 可选签名密钥
When:  触发匹配规则的告警
Then:  - POST 到配置的 URL，body 为标准 JSON（schema 见附录 A）
       - 含 HMAC-SHA256 签名头 X-OMC-Signature（如配置密钥）
       - 失败自动重试（指数退避，最多 5 次）
       - 5 次失败后进 dead-letter 队列（用户可在管理端手动重推）
```

### AC-3：短信通知（Sprint-03）
```
Given: 已配置短信服务商凭据（先支持阿里云短信或中移短信 OneLink，二选一）
When:  触发匹配规则的高优先级告警
Then:  - 短信在 60 秒内送达（遵守短信模板备案约束）
       - 短信内容含设备、告警码、时间（限 70 字）
       - 频率限制：同一手机号 1 分钟内最多收到 3 条（合并相同告警）
```

### AC-4：订阅过滤（Sprint-02）
```
Given: 用户订阅了"设备组 G-001 的 critical/major 告警，9:00-18:00 时段"
When:  - 设备组 G-001 在 10:00 触发 critical → 通知
       - 设备组 G-002 触发 critical → 不通知（组不匹配）
       - 设备组 G-001 在 23:00 触发 warning → 不通知（级别不匹配 + 时段不匹配）
Then:  与上述预期一致
```

### AC-5：配置管理（Sprint-01/02）
```
Given: 以管理员身份登录
When:  访问 `/system/notification-channels` 页面
Then:  - 可创建/编辑/启用/禁用 SMTP / Webhook / SMS 通道
       - 可测试通道连通性（测试按钮 → 发送测试消息）
       - 可查看通道最近 100 条发送记录（成功/失败 + 时间）
```

### AC-6：失败可恢复
```
Given: 外部渠道（SMTP 服务器、Webhook 端点）不可达
When:  告警触发后通知发送失败
Then:  - 不影响告警本身入库与 SSE 推送
       - 通知记录标记为 failed 并进入重试队列
       - 5 次失败后进 dead-letter，管理员可见并可手动重推
       - Prometheus 指标 `omc_notification_fail_total` 上升并触发告警
```

---

## 4. 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC | 备注 |
|------|------|------|------|------|
| 邮件模板规范 | 按 CMC-NM-GN-001 格式 | 无严格要求 | 无严格要求 | CMCC 要求字段顺序固定 |
| 告警码在通知中的表示 | CMCC 自定义码（6 位） | 标准告警码 | 标准告警码 | 通过 alarm_code_mapping 表转换 |
| 短信服务商 | 中移 OneLink（首选） | 电信 SMS | 联通 SMS | 未来选配 |
| Webhook 目标 | 一般对接集团网管 | 对接翼网管 | 对接沃云 | 字段映射按各自 OSS 规范 |
| 时段习惯 | 7×24 | 7×24 | 7×24 | 无差异 |

**实施约束**：差异通过 `internal/core/carrier/*.go` 中的 `Carrier.AlarmNotificationFormat()` 方法适配，不在 notification 模块内硬编码。

---

## 5. 非目标

- ✂️ **不做移动端 APP 推送**（原因：APP 未立项）
- ✂️ **不做 Teams/钉钉/企业微信/Slack 等 IM 通道**（原因：Webhook 已可覆盖，下个版本按需扩展）
- ✂️ **不做语音通知**（原因：合规复杂，需运营商备案）
- ✂️ **不做告警升级策略**（原因：已在 F04 过滤引擎中实现 action="escalate"，本 PRD 只负责触达）
- ✂️ **不做多语言告警描述的 AI 翻译**（原因：i18n 仅支持预置模板）

---

## 6. 依赖

### 阻塞项
- [x] `internal/alarm/filter_engine.go` 支持 action="notify"（已实现）
- [ ] Notification 基础设施（`internal/notification/`）扩展为渠道抽象 + 插件化（**本 PRD 实现**）
- [ ] SMTP 服务器选型（自建 or 云服务如 SendGrid/AWS SES） — **待决定**
- [ ] 短信服务商凭据申请 — Sprint-03 前需完成

### 被阻塞项
- `F08 北向推送`（R-003）可复用本 PRD 的 Webhook 基础设施
- `告警升级 escalation` 依赖通知链路就绪

### 外部依赖
- SMTP 服务器
- 短信服务商凭据（阿里云 / 中移 OneLink）
- Webhook 目标端（客户侧）

---

## 7. 度量

| 指标 | 基线 | 目标 | 度量方式 |
|------|------|------|---------|
| 告警产生 → 邮件送达的 P50 延迟 | N/A（未实现） | < 30 秒 | Prometheus histogram |
| 告警产生 → 短信送达的 P95 延迟 | N/A | < 60 秒 | Prometheus histogram |
| 通知成功率 | N/A | > 99.5% | `omc_notification_sent_total` / (`sent_total` + `fail_total`) |
| Dead-letter 队列积压 | N/A | < 100 条常驻 | `omc_notification_dlq_size` |
| 订阅配置正确率（用户反馈） | N/A | 用户抽样 ≥ 90% 满意 | 上线后问卷 |

**反例监控**：
- 不应因通知模块故障阻塞告警本身入库（通知走异步）
- 不应同一告警同一渠道在 5 分钟内重复发送 > 1 次（去重键：`notif:{channel}:{alarm_id}`）
- 不应泄露敏感字段（密钥、设备认证信息）到邮件/Webhook

---

## 8. 实施要点（非规范性）

**涉及模块**：
- `internal/notification/` — 扩展渠道接口、增加 smtp/webhook/sms 实现、dlq、订阅匹配
- `internal/alarm/filter_engine.go` — action="notify" 分发到新渠道
- `internal/core/carrier/*.go` — `AlarmNotificationFormat(carrier, alarm) string` 方法
- `omcmb/webcode/src/pages/system/notification/` — 管理 UI（列表/表单/测试）

**预计新增端点**：
- `POST /api/v1/notification-channels`（创建）
- `GET /api/v1/notification-channels`（列表）
- `PUT /api/v1/notification-channels/:id`（更新）
- `DELETE /api/v1/notification-channels/:id`
- `POST /api/v1/notification-channels/:id/test`（测试连通）
- `GET /api/v1/notification-channels/:id/records`（发送记录）
- `POST /api/v1/notification-dlq/:id/retry`（手动重推）

**预计新增迁移**：`000024_notification_channels_and_dlq.sql`

**预计工作量**：
- Sprint-01（2 周）：SMTP 渠道 + 基础管理 UI + DLQ — **L**
- Sprint-02（2 周）：Webhook + 订阅过滤 + Carrier 格式适配 — **M**
- Sprint-03（2 周）：短信渠道 + E2E 用例完善 — **M**

---

## 9. 附录 A：Webhook JSON Schema

```json
{
  "version": "1.0",
  "event": "alarm.notify",
  "timestamp": "2026-04-20T10:30:00Z",
  "alarm": {
    "id": "uuid",
    "device_sn": "CPE-00001",
    "carrier": "cmcc",
    "code": "POWER_FAIL",
    "severity": "critical",
    "status": "active",
    "first_occurred_at": "2026-04-20T10:29:58Z",
    "description": "电源模块故障",
    "extra": { }
  },
  "rule": {
    "id": "uuid",
    "name": "critical-to-oncall"
  }
}
```

Webhook 请求头：
- `Content-Type: application/json`
- `X-OMC-Signature: sha256=<hex>`（HMAC-SHA256 签名，密钥为渠道配置）
- `X-OMC-Event-Id: <uuid>`（幂等键，5 分钟内去重）

---

## 10. 审批

| 角色 | 姓名/占位 | 日期 | 备注 |
|------|---------|------|------|
| 产品经理 | AI 初稿 | 2026-04-20 | 待人工审核 |
| 架构师 | | | 关注通知与告警的解耦 |
| 电信业务专家 | | | 关注运营商差异 |
| QA/发布经理 | | | 关注 E2E 覆盖 |

---

## 11. 变更记录

| 日期 | 版本 | 变更摘要 | 作者 |
|------|------|---------|------|
| 2026-04-20 | v1.0 | 初稿（AI 生成） | Claude |
