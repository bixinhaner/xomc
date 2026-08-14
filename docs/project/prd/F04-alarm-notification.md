# F04 告警邮件需求范围说明

> 状态：已收敛，本文不再作为开发方案。
>
> 唯一实施方案：[`docs/superpowers/plans/2026-08-11-omc-email-notification-development-plan.md`](../../superpowers/plans/2026-08-11-omc-email-notification-development-plan.md)

原草案把邮件、短信、Webhook、通用规则引擎、运营商差异、死信管理和第三方工单一次性纳入 F04，且计划新增 `000024` 迁移；这些内容既没有本轮需求单依据，也不符合当前未封版阶段只能折回 `000001` 基线迁移的仓库约束，因此不再沿用。

本轮 F04 只保留以下内容：

- 告警规则命中 `notify_email` 后，通过统一 SMTP 传输发送邮件；
- 邮件正文增加告警字典中的中英文处理建议；
- SMTP 目标为阿里企业邮箱，由部署配置提供主机、端口、TLS 模式、账号和密钥；
- 邮件失败不阻塞告警入库，并保留可查询的发送记录。

明确不属于本轮 F04：短信、Office365/OAuth、Webhook 改造、通用通知编排平台、值班排班、静默期、告警升级、联系人组、可编辑 HTML 模板、FTP、设备状态汇总和 Redmine #98018。

`docs/design/notification-center-design-20260519.md` 描述的是站内消息中心与任务反馈，不是本轮外发邮件方案，两者继续独立维护。
