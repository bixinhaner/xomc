# OMC 告警邮件与 Zed Mobile 状态汇总邮件实施计划（开发中）

> 当前有效版本：2026-08-06。历史设计和验收证据保留在同目录的 spec/review 文档中；本文件只记录当前范围、状态和剩余门禁。

## 当前范围

本阶段维护两条邮件流程：

1. 告警邮件：告警域负责范围、级别、间隔、Tolerance Duration、收件人和默认收件人；通知底座负责异步投递、重试、熔断和审计，标题固定为 `Alarm Notification/告警通知`。
2. Zed Mobile 状态汇总邮件：按 2G/4G/5G 统计总数、在线数、激活数，排除 CPE，按本地时区生成固定正文并复用邮件投递底座。

KPI 定时报表暂不作为本阶段需求，待 #42492/#62360 单独评审后再决定是否启用；不在本阶段建设短信适配器、通用外发模板管理、跨业务规则编辑器、Office365 兼容或值班排班。

## 开发前评审门槛

评审通过不等于立即放开生产外发。开始开发前必须完成以下业务冻结：

- 已读取并确认 RR #31002、需求单 #31315/#31316、#98018 原文；#31315/#31316 要求标题固定为 `Alarm Notification/告警通知`，不区分活动/清除且都带产生/清除时间。
- 确认告警描述、处理建议、空值和中英文字段映射。
- #98018 已确认 2G/4G/5G、总数/在线/激活、排除 CPE 和本地时区；周期、收件人和范围仍需从老 OMC 配置页面或业务确认。
- 确认 `Interval` 与 `Tolerance Duration` 的老系统语义；不确定的迁移配置保持 disabled。
- 确认启用门禁：Worker、调度、Outbox、逐收件人审计、权限校验和邮件渠道健康全部就绪
	后才允许 API 接受 `enabled=true`。

评审阶段只修改设计、测试和需求追踪，不实现 SMS、KPI 定时报表、通用模板平台或 Office365 兼容。

## 已完成

- 已实现并完成本地验证的告警生命周期 Outbox、JetStream、历史投影、通知编排、逐收件人投递、重试、熔断和审计底座。
- 告警邮件专用配置、权限范围、默认收件人隔离、固定正文和处理建议快照。
- SMTP TLS 模式、连接验证、单收件人 MIME 附件发送。
- #98018 Zed 汇总统计查询：基于 `devices.technology`、`devices.is_online`、`device_info.op_state`，统一排除 CPE，并锁定 SQL 形状测试。
- #98018 固定正文、本地时区渲染、每日 `run_key` 幂等、运行快照、逐收件人加密、租约领取和失败重试 Worker。
- #98018 配置管理 API：`GET/PATCH /api/v1/notification/status-summary-settings`，仅内置超管可操作，启用要求 SMTP 已开启并使用 `If-Match`。
- 已删除无调用方的迁移候选创建 API、旧通用模板管理页面和未接入的调度半成品。

## 验证状态

已通过：

- `go test ./internal/pm/querytemplate ./cmd/worker ./internal/notification ./internal/pm/export ./internal/report ./cmd/app/provider -count=1`
- `cd omcmb && npm run typecheck`

仍待完成：

- 告警邮件 #31002/#31315/#31316 的固定主题、描述、处理建议、产生/清除时间浏览器验收。
- 老 OMC 页面目前已确认制式、Total、Online 和 CPE 独立 tab；周期、收件人和范围仍未出现，当前配置 API 默认关闭，暂不允许普通用户配置范围。
- Zed 汇总 Worker 的真实 SMTP、重启恢复、多实例领取和权限范围验收。
- 隔离数据库 + 假 SMTP + `.invalid` 地址：配置启停、重启不重复、导出失败、SMTP 失败和重试。
- 告警 raised/clear/recovery 全流程浏览器验收。
- 真实阿里企业邮箱账号、白名单收件人、小基站告警样本、Shadow 对比和现场容量数据。
- 预生产 TLS、认证、超时、出口 ACL、附件大小、时区和新旧系统命中范围验收。

## 设计约束

- SQL 使用 Squirrel + pgx；数据库变更折回三个 `000001` 基线，不新增 `000002+` 迁移。
- 用户可见文案必须走 i18n；前端只维护 V1。
- 事件、执行记录和投递时间以 UTC 保存；窗口计算使用系统 IANA 时区。
- SMTP accepted 只表示服务端受理，不代表最终送达。
- KPI 定时报表当前不在本阶段，后续单独评审统计口径和权限。
- 真实外发保持禁用，直到上述验收门禁全部通过。

## 评审通过后的开发顺序

1. 读取老 OMC 配置页面，冻结 #98018 的周期、收件人和范围。
2. 实现状态汇总配置 API/页面和业务范围权限；默认关闭，未确认配置不允许启用。
3. 完成隔离数据库、假 SMTP、重启、多实例和浏览器全流程验收。
4. 取得真实 SMTP、白名单收件人和小基站/设备状态样本后执行邮件试点和分批放量。

## 参考文档

- 当前设计：[2026-07-31-email-sms-notification-center-design.md](../specs/2026-07-31-email-sms-notification-center-design.md)
- 范围收敛：[2026-08-06-email-notification-scope-convergence.md](../reviews/2026-08-06-email-notification-scope-convergence.md)
- 里程碑 A 验收：[2026-08-06-email-sms-milestone-a-acceptance.md](../reviews/2026-08-06-email-sms-milestone-a-acceptance.md)
- 里程碑 B 验收：[2026-08-06-email-sms-milestone-b-acceptance.md](../reviews/2026-08-06-email-sms-milestone-b-acceptance.md)
- 管理前端与本地验收：[2026-08-05-email-sms-task14-local-acceptance.md](../reviews/2026-08-05-email-sms-task14-local-acceptance.md)
