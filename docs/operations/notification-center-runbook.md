# OMC 通知中心运行手册

## 1. 适用范围与原则

本手册适用于小基站 OMC 的告警邮件通知试点、Shadow 验证、故障处置和回退。当前只允许
阿里企业邮箱邮件渠道；短信 Adapter、quiet hours、生产测试发送均未实现，不得通过直接改库绕过。

当前 SMTP 约定：`smtp.qiye.aliyun.com:465`、隐式 TLS、SMTP AUTH。账号密码使用阿里企业邮箱
授权码，不得把网页登录密码、授权码或完整收件地址写入代码、工单、日志或截图。Office365/
OAuth、SMS 和短信不属于本阶段范围。Zed Mobile 状态汇总代码已具备，但默认关闭；周期、
设备范围和收件人未完成老 OMC/业务确认前，不得启用。

通知链路是告警主流程的异步下游。任何通知故障都不得阻塞告警写入、确认、清除和北向
接口。SMTP 返回成功只表示服务器 `accepted`，不能表述为最终送达。

告警范围、时间策略和收件人必须通过“告警管理 → 邮件通知设置”完成；Zed 汇总配置仅由
内置超管通过“通知管理 → 状态汇总邮件”维护；“通知管理 → 通知中心”查看站内消息、
邮件通道健康和逐收件人投递历史。所有写操作必须经过 RBAC、
审计和 `If-Match` 保护的专用 API。除本手册中的只读诊断查询外，不允许直接修改通知表。

## 2. 角色与发布门禁

上线操作至少由 OMC 管理员和现场/NOC 代表双人复核。满足以下全部条件前，保持：

```yaml
alarm:
  lifecycle_mode: legacy
notification:
  smtp:
    enabled: false
```

同时保持邮件渠道和通知规则为 disabled。

生产外发的硬门禁：

1. 取得只用于验收的阿里企业邮箱 SMTP 地址、465/隐式 TLS/AUTH、出口 ACL 和测试收件人白名单；
2. 取得可回滚的小基站样本，覆盖 raised、Tolerance Duration、ack/unack、clear 和
   recovery；本阶段不验收重复提醒；
3. 完成 legacy/new Shadow 命中数、收件人数和网元权限范围对比；
4. 按现场平均/峰值事件率、最长消费者中断和磁盘预算复算 `DOMAIN_ALARM MaxBytes`，并以
   至少 2 倍峰值复测；
5. NATS、TimescaleDB、SMTP 故障注入和回退演练均有记录；
6. 地址脱敏、操作审计、人工重试和双管理员 412 冲突通过真实浏览器验收。

本机验证记录只能证明实现可继续试点，不能替代上述生产门禁。

## 3. 变更前快照

记录发布单号、操作者、配置版本、开始时间和以下信息。命令中的地址、容器名和数据库名
必须替换为现场明确值；禁止使用模糊匹配批量操作。

### 3.1 配置一致性

核对 app 与 worker 的以下配置完全一致：

- `alarm.lifecycle_mode`
- `alarm.lifecycle_start_sequence`
- `alarm.lifecycle_stream_max_bytes`
- `notification.smtp.enabled/host/port/from/tls_mode/timeout`

密码只通过部署 Secret 注入，不进入工单、日志或截图。canonical 编排还必须为 app 注入
同一稳定的 `OMC_NOTIFICATION_RECIPIENT_KEY`；缺失时系统会拒绝启用编排或投递运行时。

### 3.2 告警 Outbox

在主 PostgreSQL 执行只读查询：

```sql
SELECT status, count(*)
FROM alarm_event_outbox
GROUP BY status
ORDER BY status;

SELECT min(created_at) AS oldest_due_at
FROM alarm_event_outbox
WHERE status IN ('pending', 'failed', 'publishing')
  AND next_attempt_at <= now();
```

对应 Prometheus 指标：

- `omc_alarm_outbox_backlog`
- `omc_alarm_outbox_oldest_age_seconds`
- `omc_alarm_outbox_publish_total`
- `omc_alarm_outbox_retry_total`
- `omc_alarm_outbox_dead_total`

### 3.3 Stream 与三个 durable

```bash
nats stream info DOMAIN_ALARM
nats consumer info DOMAIN_ALARM alarm-history-projector-v1
nats consumer info DOMAIN_ALARM northbound-alarm-lifecycle-v1
nats consumer info DOMAIN_ALARM notification-lifecycle-v1
```

记录 stream 的消息数、bytes、first/last sequence、MaxAge、MaxBytes，以及每个 durable 的
pending、ack pending、redelivered。切入 canonical 前三个 durable 必须存在且追平；任一有
积压或 start sequence 未记录都停止变更。

### 3.4 通知投影与投递

```sql
SELECT processing_state, orchestration_state, count(*)
FROM notification_events
GROUP BY processing_state, orchestration_state
ORDER BY processing_state, orchestration_state;

SELECT state, count(*), min(due_at) AS oldest_due_at
FROM notification_schedules
GROUP BY state
ORDER BY state;

SELECT flow_state, delivery_result, count(*)
FROM notification_deliveries
GROUP BY flow_state, delivery_result
ORDER BY flow_state, delivery_result;

SELECT c.channel, c.name, c.enabled, h.circuit_state,
       h.consecutive_failures, h.last_verified_at, h.last_error_category
FROM notification_channel_configs c
LEFT JOIN notification_channel_health h ON h.channel_config_id = c.id
ORDER BY c.channel, c.name;
```

不得查询或导出 `address_ciphertext`、密钥或完整收件地址。界面和 API 只能展示脱敏地址。

## 4. Shadow 验证

1. 保持邮件渠道、SMTP 和所有候选规则 disabled；记录 `DOMAIN_ALARM` 当前 last sequence，
   将其作为 `lifecycle_start_sequence`。
2. 同时把 app、worker 的 `alarm.lifecycle_mode` 改为 `shadow`，按现场既定发布流程重启；
   不得只切单个实例。
3. 确认告警主流程仍走 legacy；Shadow 只写标准 Outbox、history、northbound audit 和
   notification Inbox，不执行邮件编排或外发。
4. 使用受控小基站样本产生完整生命周期，核对三个 durable 追平、Outbox 最终 published、
   `notification_events.event_id` 唯一、occurrence 版本连续。
5. 对旧 `alarm_filters.notify_email` 与新候选规则比较命中事件、FirstMatch 顺序、收件人数、
   网元/设备组范围和 recovery 语义。设备组或 tolerance duration 无法无损映射时必须保留
   legacy，不得扩大范围。
6. Shadow 对比必须包含有效样本且 legacy/new 计数一致。当前迁移 barrier 没有开放运维
   HTTP 入口；在正式受控切换工具交付前，不允许用 SQL 手工制造 barrier 或切换候选规则。

观察窗口内出现 version gap、durable 持续积压、Shadow mismatch、权限范围扩大或重复事件，
立即按第 9 节回退到 legacy。

## 5. canonical 与邮件试点开关顺序

只有第 2 节门禁全部通过且受控 barrier 工具可用时才执行：

1. 冻结通知规则编辑，确认 Shadow 三个 durable 追平；
2. 使用受控迁移服务校验并切换单个兼容范围的 legacy barrier；禁止批量全网切换；
3. app、worker 同步切到 `canonical` 并重启，确认启动期 readiness 通过；
4. 保持 SMTP disabled，先验证 Inbox、规则命中、schedule、suppressed 轨迹；
5. 注入收件人加密密钥，配置真实测试 SMTP，但仍保持数据库邮件渠道 disabled；
6. 在“渠道”页执行 verify。verify 只做 SMTP 握手，不发送测试邮件；
7. 启用唯一邮件渠道，再启用一条已发布、严格限制网元范围和测试收件人的规则；
8. 完成一次 raised → ack/unack → clear，确认逐收件人 attempt、accepted/recovery 配对；
9. 扩大范围必须逐批执行，每批重新检查积压、重复、脱敏和权限，不允许直接全网放量。

## 6. 日常巡检与告警阈值

每 5 分钟采集第 3 节快照。至少设置以下门禁：

- `omc_alarm_outbox_backlog > 0` 持续 5 分钟告警；
- `omc_alarm_outbox_oldest_age_seconds > 30` 告警；
- 任一 durable pending/ack pending 持续增长告警；
- stream bytes 达 `MaxBytes` 的 70% 预警、80% 告警，并结合 first sequence 前移判断提前淘汰；
- `notification_events` 的 failed、version gap 或 oldest pending age 超 30 秒告警；
- schedule oldest due age 或 delivery oldest available age超 30 秒告警；
- 渠道 circuit 为 open/half_open、连续失败增长、dead_letter/unknown 增长告警。

当前通知 schedule/delivery 主要通过只读数据库视图和管理页巡检；在专用 Prometheus 指标
落地前不得把生产可观测性门禁标为完成。

## 7. 重放、死信与熔断恢复

### 7.1 Outbox/Inbox

标准 Outbox 会对 pending/failed 和过期 publishing 租约自动重试。先修复 NATS 或网络，
观察自动追平；禁止通过复制 payload 产生新 event ID。已 published 事件的 Replay 是仓储级
受控能力，当前没有运维 API，未提供审批工具时不得直接改库重放。

notification Inbox 以 `event_id` 唯一并按 occurrence/version 幂等应用。durable 恢复后应自动
重投；若 `processing_state=failed` 或 version gap 未自动收敛，保存事件 ID、occurrence ID、
版本和日志后升级研发处理，不得跳版本或删除记录。

### 7.2 投递死信

人工重试只允许 `dead_letter`，每次 1–100 条且必须填写原因。通过通知历史页操作，或调用
受鉴权的 `POST /api/v1/notification-deliveries/:id/retry`；服务会再次校验网元权限、渠道已
启用和当前状态，并记录操作者、时间和原因。

`unknown` 表示外呼结果不确定，禁止直接重发，先向 SMTP 管理员核对 message/time window，
避免向 NOC 发送重复告警。

### 7.3 渠道熔断

修复 SMTP 后，在渠道页执行 verify；成功会记录 `last_verified_at` 并关闭 circuit。恢复自动
投递前复核 dead_letter 和 retry_wait，不得手工把健康表改为 closed。

## 8. 故障注入矩阵

仅在隔离测试/预生产执行，所有地址必须是白名单测试收件人。每次只注入一种故障并保留前后
快照。

| 故障 | 操作 | 必须成立 | 恢复证据 |
| --- | --- | --- | --- |
| NATS 中断 | 停止隔离 NATS，产生受控告警 | 主库告警可写/确认/清除；Outbox 进入失败/积压，不丢事实 | NATS 恢复后 Outbox published，三个 durable 追平，event ID 无重复 |
| TimescaleDB 中断 | 停止隔离 TSDB，执行受控生命周期 | 主库活动告警事实不被通知反向阻塞；实际 ack/clear 结果必须记录，不符合即停止上线 | TSDB 恢复后投影追平，无手工伪造历史 |
| SMTP 中断/451 | 断开假 SMTP 或返回 451 | 每次只产生一个 attempt；有界退避，达到阈值打开 circuit；告警主流程正常 | verify 成功关闭 circuit，重试不产生重复 accepted |
| SMTP DATA 响应丢失 | 假 SMTP 接收 DATA 后断开 | delivery 记录 unknown，不自动盲目重发 | 与 SMTP 侧核对后人工处置 |

渠道故障不能路由回同一故障渠道形成递归通知。当前未实现渠道故障 system incident 的独立
接收人契约，使用 OMC 健康监控/NOC 既有值守路径升级。

## 9. 一键停止外发与完整回退

### 9.1 紧急停止邮件，保持告警主流程

1. 在“渠道”页禁用邮件渠道。这是最快的数据库最终发送栅栏；已领取但尚未外呼的 delivery
   会在 `AuthorizeSend` 时转为 `suppressed/channel_disabled`；
2. 观察最多一个发送租约窗口（当前 1 分钟），核对没有新的 SMTP accepted；
3. 将所有 app/worker 实例的 `notification.smtp.enabled` 改为 false 并按发布流程重启；
4. 保持告警接收、活动告警、确认、清除、北向和站内通知运行；不要停主库或告警 Worker；
5. 记录禁用时间、最后 accepted delivery、在途/unknown 和 suppression 数量。

不要先只关闭 SMTP 服务：这会把已领取投递转为失败/重试并制造不必要积压。也不要删除规则、
模板、delivery 或 attempt，它们是审计和恢复依据。

### 9.2 canonical 回退到 legacy

1. 先按 9.1 停止外发并确认没有发送中的 attempt；
2. 冻结规则和迁移操作，记录三个 durable、Outbox、Inbox 和 delivery 快照；
3. app、worker 同时将 `alarm.lifecycle_mode` 改为 `legacy` 并重启；
4. 使用受控迁移工具恢复原 legacy `notify_email` 规则；若工具尚未交付，保持邮件禁用并升级
   研发处理，禁止直接 SQL 删除 barrier；
5. 用受控小基站确认活动告警、ack、clear 和 legacy stream 正常；
6. 标准 Outbox/`DOMAIN_ALARM` 不再推进是 legacy 的预期行为，既有数据不得删除；
7. 故障分析和 Shadow 复验通过后，才能重新进入第 4 节。

## 10. 容量计算与验收记录

容量至少按下式计算，并保留 20% 磁盘安全余量：

```text
RequiredBytes = peak_events_per_second × max_consumer_outage_seconds
                × measured_average_event_bytes × 1.2
```

同时计算 7 天平均保留需求；`MaxAge=7d` 不是保留承诺，若 `MaxBytes` 先达到会提前淘汰。
压测必须记录：输入速率、事件处理 P50/P95/P99、Outbox oldest age、schedule 延迟、SMTP
accepted 吞吐、数据库锁等待、stream bytes、三个 durable lag 和最大积压。通知内部实时处理
P95 必须小于 30 秒。

本机里程碑 A 实测约 1.18 KiB/生命周期事件、当前 10 GiB 在 80% 水位下只支持约 12 条/秒
的 7 天窗口，仅用于发现默认值风险，不能作为现场容量结论。部署后验收方案见
`docs/superpowers/reviews/2026-08-11-email-notification-server-acceptance.md`。

每次演练或上线附一份记录，至少包含：环境、版本、现场输入、开关前后值、样本网元、测试
收件人范围、各阶段时间戳、查询/指标快照、浏览器截图、失败项、回退结果和双人签字。
