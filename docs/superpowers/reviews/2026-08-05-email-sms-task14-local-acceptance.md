# Task 13-14 通知中心管理前端与本地全链路验收

> **范围说明（2026-08-06）：** 本文证明 Outbox、投递、重试、熔断和故障恢复底座，
> 不证明历史通用规则/模板前端仍属于产品范围。当前业务入口已收敛为告警邮件设置与
> KPI 查询模板定时报表。原 Task 13 前端评审内容已合并到本文的“管理前端复验”章节。

## 结论

Task 14 的本地故障注入、三 durable 容量基线和后端相关回归通过。NATS 中断恢复、
TimescaleDB 长时间中断恢复、SMTP 451/连接异常/结果未知和渠道熔断均已有可重复证据；
告警主事务没有被通知或历史投影反向阻塞，事件 ID 保持唯一。

本结论只允许继续 Shadow/预生产验证，不构成生产外发放行。真实 SMTP、小基站现场样本、
生产等价 Shadow 对比和现场两倍峰值容量数据仍未取得，SMTP 和候选规则必须保持禁用。

## 隔离环境

- Docker Compose project：`xomc-email-sms-shadow`；App/Worker 与原开发环境隔离。
- 主库使用专用数据库 `omcgo_task13`；故障注入没有删除或重置原数据库、卷或容器。
- 故障与容量阶段外发关闭；完整业务闭环阶段仅短时启用本机假 SMTP 和 `.invalid` 收件人，
  没有连接外部 SMTP，也没有产生真实邮件或短信。
- 故障复验结束后 App、Worker 均恢复 `alarm.lifecycle_mode=shadow`，健康端点返回 200。

## 本地完整业务闭环

2026-08-06 使用已有已发布模板、既有邮件通道和本地验收规则，通过正式管理 API 完成一次
受控的 `raised → cleared` 全流程。为覆盖恢复生命周期，只给该验收规则的新不可变版本补充
`cleared_template_version_id`，没有新增产品能力或旁路改库。SMTP 指向本机 `2525` 假服务，
发件人与收件人均为 `.invalid` 地址；假服务只接受 envelope 和 DATA，并记录脱敏后的计数。

结果如下：

- 通道先在 disabled 状态验证成功，再启用；规则发布并启用后注入
  `LAB-GNB-NOTIFY-001 / alarm_identifier=10001`；
- raised 后活动告警为 1，通知事件为 `applied/completed`，生成一条 `initial` 投递；
- initial 投递为 `completed/accepted`，第 1 次 attempt 即 `accepted`，本地 SMTP 收件数为 1；
- clear 后活动告警归零；raised/cleared 共用同一 occurrence，alarm version 为 1/2，两个事件
  均为 `applied/completed`；
- recovery 投递为 `completed/accepted`，第 1 次 attempt 即 `accepted`，并通过
  `origin_delivery_id` 关联 initial 投递；本地 SMTP 最终收件数为 2；
- 管理 API 返回 2 条投递和 2 条 attempt，收件人仅返回 `recipient-…` 指纹掩码，不返回邮件
  地址；通道健康为 `closed`，同时存在 verify 与 send success 时间；
- 浏览器页签在本轮数据验收时没有重新接入控制通道，因此没有把“含这两条真实数据的页面视觉
  复核”冒充为已完成；Task 13 的定向浏览器 E2E 仍已通过，该项保留为下一次人工刷新复核。

收口时先通过正式 API 关闭邮件通道和验收规则，再把 App、Worker 恢复为 Shadow、SMTP
disabled 并重建；两者环境值已复核为 `shadow/false`，Worker healthy，App `/healthz` 返回
200。假 SMTP 已停止。规则停用后仍保留 draft/published 版本，投递审计记录也保留在隔离
数据库，未删除或伪造历史。

## 故障注入证据

### NATS

使用专用 PostgreSQL、TimescaleDB 和无持久卷的临时 NATS 执行现有
`TestAlarmLifecycleEndToEnd_CommittedFactsReachIndependentDurables`。NATS 断开时告警主事务
提交，Outbox 进入失败/重试；恢复后只发布一次，history 和 northbound 两个独立 durable
均按 event ID 幂等应用。测试通过后删除临时 NATS 容器。

### TimescaleDB

首次真实停库演练暴露了一个不能接受的缺陷：history projector 与普通下游共用
`MaxDeliver=5`，TSDB 中断超过约 15 秒后消息被 `Term`，恢复时历史记录无法补写。

采用最小修复：只把固定 durable `alarm-history-projector-v1` 设为 JetStream 无限投递
`MaxDeliver=-1`；共享 keyed queue 保留 `0=项目默认 5`、正数=有界投递的原语义，并允许把
既有 durable 从 5 原地更新为 -1。无法解码或不支持 schema 的事件显式标记 permanent，
仍会立即终止，避免坏消息无限占住同一 occurrence lane。其他消费者继续使用 5 次上限。

修复后的真实复验结果：

- 测试网元 `LAB-GNB-NOTIFY-001` raised 后，主库活动告警为 1，Outbox 为 3/3 published；
- 停止隔离 TSDB 后 clear，主库活动告警变为 0，Outbox 为 4/4 published 且 4 个 event ID 唯一；
- history projector 在 TSDB 不可用期间连续重试到第 18 次，未在第 5 次终止；
- 恢复 TSDB 后首次轮询即追平，历史数从 6 增至 7；最新记录为 cleared、alarm version 2、
  alarm identifier 10001，没有人工改库或重复补写；
- 恢复 Shadow 配置后 Worker 启动日志确认 history durable 为 `max_deliver=-1`。

### SMTP

复用里程碑 B 的本地真实 SMTP 会话证据：逐收件人 envelope、DATA accepted、三次 451
退避、第三次失败打开 circuit、verify 关闭 circuit、认证失败、DATA 响应丢失及 lease 过期
转 unknown 均已通过。该证据来自回环测试 SMTP 和 `.invalid` 收件人，不替代运营商现场
TLS、AUTH、出口 ACL 和真实测试邮箱验收。

## 三 durable 本地容量基线

2026-08-06 在 App/Worker Shadow、SMTP disabled、邮件渠道和规则均 disabled 的隔离环境，
使用 100 台虚拟 gNB 按 `NewAlarm → ClearedAlarm` 产生受控生命周期。history、northbound、
notification 三个 durable 测试前均为 `pending=0、ack_pending=0`。

先执行目标 400 条/秒、30 秒，共 12,000 条输入。功能一致性通过：Outbox 12,000/12,000
published、event ID 唯一、活动告警归零；notification Inbox 12,000/12,000 applied；历史
增加 6,000 条；三个 durable 最终全部追平。但从首条 Outbox 创建到末条发布历时
100.53 秒，显著超过 30 秒输入窗口并出现主库慢查询，因此该档在本机不是可持续吞吐，
不得据此宣称两倍现场峰值门禁通过。

随后用独立网元前缀执行目标 200 条/秒、30 秒，结果如下：

| 指标 | 结果 |
| --- | ---: |
| 实际输入 | 174.07 条/秒，6,000 条，34.468 秒 |
| 主链路窗口 | 35.74 秒，输入结束后约 1.27 秒排空 |
| 注入端 publish P95/P99 | 7.854 / 20.178 ms |
| Outbox publish P95/P99 | 132.18 / 158.93 ms |
| notification received P95/P99 | 214.51 / 424.10 ms |
| notification applied P95/P99 | 222.09 / 450.86 ms |
| 最长 notification applied | 875.69 ms |

最终 Outbox 为 6,000/6,000 published，notification Inbox 为 6,000/6,000 applied，历史为
3,000 条，活动告警为 0；三个 durable 均为 `pending=0、ack_pending=0`，没有新增重投。
Shadow 不执行编排，6,000 条 Inbox 的 `orchestration_state=pending` 是预期状态；本轮没有
schedule、delivery 或 SMTP 外发。

负载排空后的资源快照约为 App 89.4 MiB、Worker 52.7 MiB、主 PostgreSQL 487.8 MiB、
TimescaleDB 139.2 MiB、NATS 76.0 MiB。`DOMAIN_ALARM` 27,721 条占约 32.8 MB，平均事件
大小仍约 1.18 KiB，与里程碑 A 一致。该快照不是峰值资源采样；日志存在约 0.6–1.1 秒的
通知、Outbox 及同库后台任务慢查询，现场压测仍需关注数据库竞争。

本轮没有根据单机数据扩大并发、增加索引或调整 `MaxBytes`。生产 Step 4 仍未完成：必须
取得现场平均/峰值、最长消费者中断和磁盘预算，在生产等价资源上重新执行至少两倍峰值，
并验证完整规则编排和真实测试 SMTP 吞吐。

## 回归与审查

### 管理前端复验（2026-08-06）

使用当前基线重新初始化独立数据库 `omcgo_email_sms_acceptance_20260806`，并仅向隔离的
App/Worker 注入临时收件人加密密钥；SMTP、规则外发和告警生命周期仍保持
`disabled/Shadow`。真实浏览器完成告警邮件设置的创建、列表回读、默认收件人弹窗和删除
归档，验收地址使用 `acceptance@example.invalid`，没有连接外部邮件服务器。

复验发现并修复一处列表缺陷：通用规则仓储的 `List` 按设计只返回规则头，告警邮件适配层
却直接用浅对象判断内置邮件通道和固定模板，导致已成功创建的设置被过滤。修复限定在告警
邮件适配层，逐条加载完整不可变版本后再做业务识别，没有扩大通用仓储行为；新增定向单测
覆盖该契约。浏览器复验确认创建返回 201、列表回读和归档返回 200，删除后页面恢复空表。

尝试把验收规则切为启用时，后端因默认邮件通道未启用返回 409 并整体回滚；规则仍为禁用、
revision 未增加。这证明本地禁外发门禁生效，不将该结果误记为真实 SMTP 外发验收。

- `go test ./internal/alarm ./internal/notification ./internal/backup ./internal/northbound/...`
  `./internal/core/event ./internal/core/components/nats -count=1`：9 个相关包通过。
- `go vet ./internal/alarm ./internal/core/event ./internal/notification`：通过。
- `git diff --check`：通过。
- 定向测试覆盖 durable 从 5 更新为 -1、无限投递不触发本地终止，以及 history projector
  固定配置；真实 Docker 故障注入覆盖超过 5 次后的恢复。

本次没有新增迁移文件、第二套消息总线、死信表、后台任务或配置开关。没有把 history 的
无限投递推广到 notification/northbound 等已有幂等/死信策略的消费者，也没有为当前本地
验收补造短信 Adapter、quiet hours 或生产监控模块。

## 未完成的生产门禁

1. 真实测试 SMTP 的 TLS、认证、ACL、超时和测试收件人白名单。
2. 可控小基站样本的 raised、minimum duration、repeat、ack/unack、clear/recovery 全周期。
3. 生产等价 Shadow 数据上的 legacy/new 命中、收件人数和网元权限范围对比。
4. 现场两倍峰值压测、最长消费者中断和磁盘预算下的 `DOMAIN_ALARM MaxBytes` 复算。
5. 双管理员 412、人工重试、死信以及有数据的逐 attempt 浏览器验收。

上述任一项未通过时，保持 legacy/Shadow、SMTP disabled 和通知规则 disabled，按
`docs/operations/notification-center-runbook.md` 执行检查与回退。
