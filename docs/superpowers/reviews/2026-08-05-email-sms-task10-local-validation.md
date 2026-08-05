# OMC 邮件短信通知中心 Task 10 本地验收

## 结论

Task 10 的后端编排核心可以进入提交：标准告警生命周期事件经过有序 Inbox 投影后，能够
以不可变规则版本和逐收件人维度生成持久化 schedule、摘要 bucket、suppressed 审计或
delivery；确认、取消确认、清除、维护窗口、严重度升级和事件重放均有明确状态语义。

本任务没有调用真实 SMTP、短信 Kafka 或供应商接口，也没有把排队记录伪装成发送成功。
生产装配仅在 canonical 模式且配置 `OMC_NOTIFICATION_RECIPIENT_KEY` 时启用编排；缺少密钥
时保持 Inbox/occurrence 审计投影，非法密钥则启动失败。Shadow 模式仍不产生外部投递。

## OMC 适配论证

- 事实入口仍是 `domain.alarm.lifecycle.*`，不消费 legacy `alarm.*`，符合当前项目的
  JetStream 多 durable 架构。
- raised 使用最小存活门槛和有界 repeat，普通 updated 不重发；严重度升级即时编排，但仍
  受已审批维护窗和收件人速率上限约束。
- acknowledge/clear 通过 schedule generation 围栏取消旧的 initial/repeat；unacknowledge
  仅恢复原策略中尚未到期的未来 repeat，不重新制造首次通知。
- recovery 只配对原先 `completed+accepted` 或 `completed+handoff_only` 的原收件人、原渠道，
  并使用历史规则版本绑定的 cleared 模板；门槛前即清除只写可审计 suppressed 事实。
- Minor/Warning 可进入 UTC 摘要窗口；Critical 只绕过普通摘要偏好，不绕过明确收件人上限。
  告警在摘要窗口内清除时，digest schedule 不被生命周期代际围栏静默删除。
- 维护期不删除告警事实，只为已审批、生效且范围命中的窗口记录 suppressed delivery，并
  关联 `ops_maintenance_windows`。
- 投递历史按设备组与制式 grant 求交；API 仅返回不可逆指纹尾部，不返回地址密文或明文。

这些语义延续了老 OMC 中“存活门槛、重复、恢复、维护抑制”的业务意图，但实现依托当前
项目的标准生命周期事件、PostgreSQL 事务、不可变配置版本和权限体系，没有复制老项目的
页面耦合、进程内定时器或直接外呼方式。

## 可靠性与安全边界

- 规则决策为纯函数；一个事件的解释、schedule、bucket 和 delivery 在同一主库事务提交。
- `notification_events.orchestration_state` 是事件级幂等围栏，JetStream 重投不会重复增加
  bucket 或重复创建 delivery。
- 同一摘要桶的首个事件在事务中创建唯一 flush schedule，后续并发事件只原子增加 bucket
  计数，避免基站告警风暴转化为 scheduler 风暴。
- `AuthorizeSend` 是未来渠道调用前的最后内部栅栏，原子校验 sending lease、occurrence
  状态/版本/代际、渠道启用状态和熔断状态；拒绝原因落为 cancelled 或 suppressed。
- 收件人地址采用 AES-256-GCM 随机 nonce，加密与 HMAC 指纹使用域分离派生密钥，channel
  参与 AAD；没有明文 fallback。
- 人工重试最多 100 条且必须填写原因，只接受当前用户可见、配置型 dead-letter、渠道已
  重新验证且熔断关闭的记录，并保存操作人、原因和时间。

## 本地验证证据

- `go test ./internal/notification -count=1`：通过。
- `go test ./cmd/app/provider -count=1`：通过。
- 从正式 Goose schema + seed 重建独立数据库 `omcgo_notification_task10_fresh`：schema、seed、
  domain/orchestration/rule/contact-group/template/channel 集成测试通过。
- 独立数据库 `omcgo_notification_task10_lifecycle_fresh`：lifecycle 与 orchestration 集成测试
  通过；真实规则解析、固定联系人密文、Minor digest 和 SaveDecision 重放幂等通过。
- 独立数据库 `omcgo_notification_task10_seed_fresh`：正式 schema 与 seed 从零执行通过；27 个
  通知管理端点全部登记，admin/operator 各获得 27 个端点，viewer 仅获得 11 个 GET 端点；
  两个 occurrence 对同一摘要桶累计为 2，且只创建 1 条 digest flush schedule。
- `go test ./... -count=1` 在允许本机临时端口后，通知模块及应用装配通过；全仓最终未全绿，
  原因是现有共享测试库的 `paramsync.parameter_sync_requests` 缺少既有
  `admission_queued_at` 列，以及一个既有 `internal/task` Redis 时序用例失败。两类失败均不在
  本次变更文件和调用链中，因此未修改无关模块来掩盖环境问题。

## 有意保留到下一任务

- Task 11 实现 initial/repeat/digest/quiet-hours 的持久化 handler、SMTP Worker、attempt 状态机、
  退避与熔断，不在 Orchestrator 内直接外呼。
- quiet hours 的 IANA 时区 release 和夏令时唯一执行需与真实 schedule handler 一起验证；
  规则预览在 Task 13 页面闭环前不提前暴露不可兑现的配置能力。
- SMS 仅保留渠道契约和 disabled 配置；供应商/Kafka/回执契约冻结前不实现假 Adapter。

这三个边界是为了避免过度设计，同时确保任何已暴露能力都具备可验证、可审计的真实语义。
