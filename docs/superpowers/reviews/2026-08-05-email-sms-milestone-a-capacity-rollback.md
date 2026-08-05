# OMC 邮件短信通知中心里程碑 A 本机容量与回退记录

## 结论

里程碑 A 的**本地开发门禁通过**：分档负载、事件唯一性、durable 追平和
`shadow → legacy → shadow` 回退均通过，可以继续 Task 7 的本地开发。该结论不等于生产
canonical 放行；生产仍保持 `legacy`，待取得现场峰值、最长中断时间和磁盘预算后重新计算
容量，并在测试/预生产环境复演。

## 方法与隔离边界

- Compose project：`xomc-email-sms-shadow`，独立网络、卷和端口。
- 使用 100 台 `LAB-LOAD-*` 虚拟 gNB 和既有受控告警定义 `10001`。
- 每个 occurrence 严格发送 `NewAlarm → ClearedAlarm`，在 100 台设备间轮转。
- 三档输入分别为目标 20、100、200 条/秒，各持续 30 秒；总计 9600 条生命周期事件。
- 注入器和 Compose 覆盖位于 `/private/tmp`，没有进入产品代码，也没有访问老 OMC。
- 当前只有历史、北向两个 durable；通知 Shadow durable 在通知 Inbox 建立后补齐。

## 容量结果

| 目标输入 | 实际输入 | 事件数 | 客户端发布 P95/P99 | Outbox 发布 P95/P99 | 结果 |
| --- | ---: | ---: | ---: | ---: | --- |
| 20/s | 20.00/s | 600 | 7.65/26.01ms | 99.15/102.53ms | 零积压 |
| 100/s | 97.48/s | 3000 | 2.78/9.11ms | 107.79/117.87ms | 零积压 |
| 200/s | 188.46/s | 6000 | 4.10/10.48ms | 98.89/112.60ms | 零积压 |

最高档 raised 事件从注入时间到主库 Outbox 的 P95/P99 为 623.42/872.94ms，从注入到
JetStream 发布的 P95/P99 为 629.54/890.23ms，最大约 1.02s。该口径覆盖本地 NATS 入站、
AlarmEngine 事务和 Outbox Relay，不包含尚未实现的通知 Inbox、规则、调度和渠道耗时。

最终一致性：

- `alarm_event_outbox`：9600 条，9600 个不同 `event_id`，全部 `published`。
- occurrence：4800 个，每个版本范围严格为 1–2。
- `alarms_active`：0 条容量测试残留。
- `DOMAIN_ALARM`：两个 durable 均 `pending=0`、`ack_pending=0`、`redelivered=0`。
- 负载后容器快照：App 约 61.9MiB、Worker 约 33.7MiB、PostgreSQL 约 251MiB、NATS
  约 70.1MiB；快照发生在负载排空后，只用于发现资源泄漏，不代表峰值资源使用。

日志中没有 panic、fatal 或 Shadow history mismatch。观察到一次 Outbox 拉取慢查询约
762ms，发生在高负载后进程重启阶段；没有造成积压或错误，后续现场容量测试需继续跟踪。
OTLP 地址缺失、MinIO ILM 和 `northbound.server.changed` 无匹配 Stream 是隔离环境既有告警，
与本次生命周期负载结果无关。

## 7 天保留窗口计算

本次 `DOMAIN_ALARM` 9617 条消息占用约 10.85MiB，实测平均约 1.18KiB/条。按当前
`MaxBytes=10GiB` 静态计算，约可容纳 900 万条同尺寸事件：

| 平均生命周期速率 | 10GiB 理论窗口 |
| ---: | ---: |
| 20/s | 约 5.2 天 |
| 100/s | 约 1.0 天 |
| 188/s | 约 0.56 天 |

不计安全余量时，10GiB 覆盖 7 天所允许的平均速率约为 15 条/秒；按 80% 磁盘安全水位，
建议上限约 12 条/秒。该结论说明 `MaxAge=7d` 不是保留承诺，`MaxBytes` 必须根据现场平均/
峰值、最长消费者中断和磁盘预算确定，不能现在盲目放大默认值。

## 回退演练

基线：`DOMAIN_ALARM last_seq=9615`、标准 Outbox 总数 9608。

1. 将隔离 App/Worker 从 `shadow` 重建为 `legacy`。
2. 在 legacy 下产生并清除受控告警：活动告警正常 `1 → 0`；标准 Outbox 保持 9608；
   `DOMAIN_ALARM last_seq` 保持 9615；legacy `ALARM` 正常推进。
3. 重建恢复 `shadow`，再次产生并清除：标准 Outbox `9608 → 9610`，
   `DOMAIN_ALARM last_seq 9615 → 9617`。
4. 最终两个标准 durable 零积压、零重投，所有受控活动告警为 0，运行模式保持 `shadow`。

## 门禁解释

- **允许继续：** Task 7–13 的隔离本地开发与测试。
- **仍然禁止：** 生产 Relay/canonical、真实邮件外发、SMS Worker 和替换 legacy
  `notify_email`。
- **生产前必须补齐：** 现场事件率与中断窗口、测试/预生产 2 倍峰值、通知 Shadow durable、
  三消费者对比、完整通知 P95、渠道故障注入、真实浏览器和生产级回退演练。
