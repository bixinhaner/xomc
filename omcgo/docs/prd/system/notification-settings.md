# 系统管理 —— 通知设置范围说明
> 更新于 2026-08-11。本文用于清理旧的纯 mock 设计，不再作为“统一通知中心”开发依据。

## 当前可用能力

| 场景 | 配置入口 | 投递方式 |
|---|---|---|
| 告警邮件 | 告警规则中的 `notify_email` 动作和收件人 | 异步任务 → 统一 SMTP 适配器 |
| KPI Query 报表邮件 | KPI Query 模板行的邮件订阅 | 固定时间窗口导出 CSV → 异步 SMTP 附件 |
| 通知历史 | `/api/v1/notifications/history` | 仅超级管理员查询 `notification_history`；结果含收件人和渲染后的正文，不向普通 `Alarm.View` 用户开放 |
| 站内消息 | 顶栏铃铛和 `/notifications` | 与邮件投递独立 |

SMTP 只从 app/worker YAML 及环境变量读取，不在 Web 页面中保存密码。实际配置参见 `docs/runbook/email-notification.md`。

## 已移除的入口

- `pages/system/NotificationSettings` 的规则、收件人组和模板均为前端内存 mock，从未挂载路由，已删除。
- `SystemConfig/NotificationSettings` 的邮件/阿里云短信表单没有后端持久化和真实测试接口，“测试成功”只由定时器伪造，已删除。
- 通知模板/历史后端模型保留，不影响现有真实邮件链路。

## 明确不在本版本

- Office365：已停用，不保留专有配置或分支逻辑。
- 短信、Webhook 与 FTP 邮件投递：本版本不实现。
- #98018 自动邮件汇总：用户明确排除。
- #62360 `module_name/online_count/offline_count`：未在 xomc、老文档或现役部署信息中找到可验收的消费方，本次不迁移。只有补齐 Redmine 原文、样例邮件和现役程序证据后才重新评估。
