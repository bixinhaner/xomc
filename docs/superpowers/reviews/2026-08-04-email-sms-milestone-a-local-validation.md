# OMC 邮件短信通知中心里程碑 A 本地验证记录

## 验证结论

可靠告警生命周期主链路已在完全隔离的本地环境通过 Shadow、canonical 和 NATS 进程级
故障恢复验证，可以确认 Task 1–6 的功能基础成立。后续本机分档容量与回退也已通过，详见
[容量与回退记录](2026-08-05-email-sms-milestone-a-capacity-rollback.md)，因此里程碑 A 的本地
开发门禁通过，可以进入 Task 7。生产配置必须继续保持 `legacy`：现场峰值至少 2 倍容量和
测试/预生产回退尚未完成，也不得启用邮件或短信发送。通知 Shadow durable 属于后续
canonical 生产切换门禁，不反向阻塞建立通知 Inbox 的 Task 7。

该结论符合通信行业 OMC 的渐进切换原则：告警事务不依赖外部通知，事实事件可重放，历史、
北向和通知使用独立消费位点；故障时允许通知延迟，但不能阻塞网元告警入库，也不能重复放大
同一告警事实。

## 业务与设计依据

本轮沿用《OMC 邮件短信通知中心复评记录》对老 OMC 文档、老系统只读业务和当前代码的
对齐结果：老系统的确认、取消确认、清除、重复告警计数及更新时间被映射为稳定生命周期
事实；老 SMTP 重试缺陷由持久化投递和明确重试归属解决；老 SMS 的 Redis/Kafka 链路仅作为
Adapter 语义参考，外部 Topic、Schema、鉴权和回执未冻结前不实现 SMS Worker。

同时保留领域边界：备份、磁盘和通知渠道故障不伪造 `device_id` 进入网元告警域；运营商只
参与业务匹配，不替代设备组与制式权限；TimescaleDB 是可重放历史投影，不宣称跨库原子。

## 隔离环境

- 工作树：`.worktrees/email-sms-notification-design`，分支
  `codex/email-sms-notification-design`。
- Compose project：`xomc-email-sms-shadow`，使用独立网络、命名卷和测试数据。
- 映射端口：PostgreSQL `25432`、TimescaleDB `25433`、NATS `24222/28222`、
  Redis `26379`、MinIO `29000/29001`、App `28081/29091`、Worker `29092`。
- 测试网元：`LAB-GNB-NOTIFY-001`，仅存在于隔离数据库。
- 未修改、重启或写入正常开发栈和老 OMC；临时 Compose 覆盖及注入器均位于
  `/private/tmp`，不进入仓库。

## 验证结果

| 项目 | 结果 | 证据 |
| --- | --- | --- |
| Shadow 产生/清除 | 通过 | 活动告警按产生/清除增删；标准 Outbox 发布；Shadow 不产生北向正式投递 |
| canonical 产生/清除 | 通过 | TSDB 每个 occurrence 仅一条清除历史；北向 Outbox 正常投递 |
| durable 隔离 | 通过（当前启用 2 个） | `alarm-history-projector-v1`、`northbound-alarm-lifecycle-v1` 均零积压、零重投 |
| NATS 停机不阻塞主事务 | 通过 | 停机期间 Outbox 进入 `failed/publish_failed`，NATS 恢复后自动变为 `published` |
| 恢复后事件唯一性 | 修复后通过 | 同一事件重放时 `DOMAIN_ALARM` 消息数 `13 -> 14`，只新增 1 条；北向 Outbox 仍为 1 条 |
| 历史幂等 | 通过 | 受控告警清除后 `alarms_history` 计数为 1，`alarm_version=2` |
| 北向报文 | 修复后通过 | 外层 `event_id/subject/timestamp` 正确，`payload` 直接为告警快照，不再二次嵌套内部信封 |
| 受影响 Go 测试 | 通过 | alarm、northbound、event、app provider、worker 全部通过 |
| 真实浏览器 | 未形成证据 | 本地端口被应用内浏览器 `ERR_BLOCKED_BY_CLIENT` 拦截；本轮无前端改动，以容器健康和真实 HTTP sink 验证后端 |

## 本轮发现并修复的问题

### 1. NATS 断线重连造成重复放大

Outbox 已经拥有持久化重试状态机，但断线的 `nats.Conn` 仍可能缓存发布请求并在重连后
补发；与此同时 Outbox 也会重新发布，导致一个 `event_id` 在 Stream 中出现多条消息。
修复后 `NATSEventBus.Publish` 在连接不可用时立即失败，不进入客户端重连缓冲，重试所有权
只保留在数据库 Outbox。

### 2. 北向 Outbox 报文二次包装

北向 Outbox 保存的是完整事件信封，Worker 原先又把整段 JSON 当成业务 payload 交给投递
引擎，造成双层信封且外层时间戳为零。修复后 Worker 识别并解开仓储信封，保留原业务
时间戳；对旧数据或外部写入的纯 payload 仍保留兼容回退。

## 容量与放行边界

本地 `DOMAIN_ALARM` 当前配置为 `MaxAge=7d`、`MaxBytes=10GiB`。本次 14 条受控消息占用
约 16.5 KiB，观测平均约 1.18 KiB/条；仅按静态字节除法，10 GiB 大约可容纳 890 万条
同尺寸事件。该数字不包含现场告警风暴、不同 payload 分布、磁盘水位、索引/文件系统开销
及最长消费者中断，因此不能作为 7 天容量承诺。

生产放行仍需：以现场估算峰值至少 2 倍压测，记录 P50/P95/P99、Outbox oldest age、
durable lag、最老消息、磁盘水位和提前淘汰风险，并在测试/预生产环境完成 Shadow 切换与
回退演练。上述门禁通过前，Relay/canonical 只能用于隔离验证。后续创建通知 Inbox 后，
还要验证第三条通知 Shadow 消费链；三条 durable 全部通过前，canonical 仍不得进入生产。

## 下一步

1. 保持默认 `legacy`，不进入生产 canonical。
2. 开始 Task 7 的隔离本地开发；所有外部渠道继续关闭。
3. 获取现场告警基线或可代表现场的回放样本，在测试/预生产环境复验容量和回退。
4. 通知 Inbox 建立后创建只入 Inbox、不发送渠道消息的通知 Shadow durable，补齐第三消费
   链门禁；其创建 sequence 使用里程碑 A 记录的切换基准。
5. 三条 durable 全部通过后才允许评估生产 canonical；邮件试点仍需单独通过里程碑 B。
