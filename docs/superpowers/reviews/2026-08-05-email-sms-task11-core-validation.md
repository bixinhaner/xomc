# 邮件短信通知中心 Task 11 核心链路本地验证

## 本次边界

本次只实现当前项目已有数据契约能够完整支撑的邮件可靠投递链路：

- `initial_gate`、`repeat`、`digest_flush` 到逐收件人 delivery 的事务化转换；
- 邮件 delivery 的数据库领取、外呼前最终授权、attempt、严格渲染、收件人解密和 SMTP；
- 临时错误退避、永久错误死信、外呼结果不确定时禁止盲目重发；
- SMTP 配置/鉴权/TLS 和连续临时故障的渠道熔断；
- 真实 SMTP 握手验证及渠道健康更新；
- App、Worker 和 legacy `notify_email` 统一复用 `notification.smtp` transport。

未增加短信 Adapter、消息中间件、前端、网元告警类型或新迁移文件。

## 状态与故障语义

| 场景 | delivery | attempt | 是否自动重试 |
|---|---|---|---|
| SMTP DATA 最终响应成功 | `completed + accepted` | `accepted` | 否 |
| DATA 最终响应丢失/超时 | `completed + unknown` | `unknown` | 否 |
| 网络或 SMTP 4xx | `retry_wait + failed` | `failed` | 有界指数退避，最多 5 次 |
| 收件地址/消息永久拒绝 | `dead_letter + failed` | `failed` | 否 |
| SMTP 配置/鉴权/TLS 错误 | `dead_letter + failed` | `failed` | 否，并打开渠道熔断 |
| Worker 在外呼 attempt 后崩溃 | `completed + unknown` | `unknown` | 否 |

邮件成功只表示 SMTP 服务器受理，未伪造成 `delivered`。每个 SMTP 信封只包含一个
`RCPT TO`，投递日志和数据库状态摘要不保存完整收件地址、密码或服务端敏感响应。

## PostgreSQL 集成验证

隔离数据库：`omcgo_notification_task11_fresh`（本机 shadow PostgreSQL `25432`）。

已验证：

1. schedule 租约过期后被另一实例重领，只生成一条 delivery，且
   `notification_schedules.delivery_id` 保持同一引用；
2. claim → `AuthorizeSend` → attempt → accepted 事务闭环；
3. 未完成 attempt 的发送租约过期后转 `unknown`，不会再次被领取外呼；
4. 连续三次临时渠道失败后 `notification_channel_health.circuit_state=open`；
5. verify 成功关闭熔断并写 `last_verified_at`，失败重新打开；
6. digest schedule 在同一事务中关闭并关联聚合桶，渲染得到正确的事件数和窗口时间。

验证命令：

```bash
cd omcgo
NOTIFICATION_DOMAIN_TEST_DSN='postgres://omcgo:***@127.0.0.1:25432/omcgo_notification_task11_fresh?sslmode=disable' \
  go test ./internal/notification -run '^TestPgDomainRepository_Integration$' -count=1

go test ./internal/notification ./internal/alarm ./internal/core/appconfig ./cmd/app/provider ./cmd/worker -count=1
go vet ./internal/notification ./internal/alarm ./internal/core/appconfig ./cmd/app/provider ./cmd/worker
```

以上命令均通过。

## Review 结论

Review 中发现并修正：

- App 之外的 Worker 仍装配旧 `OMC_SMTP_*` 客户端：已统一为 `notification.smtp`；
- alarm 包保留第二套 SMTP 实现和重复网络测试：已删除，只保留迁移适配接口和指标；
- 多收件人共用一个 SMTP 信封：已改为逐收件人独立信封；
- SMTP QUIT 失败会覆盖已经受理的事实：改为 DATA 成功即 accepted，忽略其后 QUIT 失败；
- 外呼后进程退出可能盲目重发：未完成 attempt 的过期租约改记 unknown；
- 主题可被告警字段注入 CR/LF：已拒绝包含换行的最终主题；
- STARTTLS 未明确最低版本：设置 TLS 1.2 下限；
- 未注册 schedule 类型会被反复领取释放：领取 SQL 只选择当前已注册 handler 类型。

当前实现没有新增 ORM、字符串拼接 SQL或 `000002+` 迁移；所有持久化 SQL 使用 Squirrel +
pgx，错误均带上下文包装。

## 明确保留项

- `quiet_hours_release`：当前 Orchestrator/规则策略尚不产生该 schedule。本次没有臆造 IANA
  时区、夏令时或释放规则；Scheduler 也不会领取未注册类型。
- 渠道故障 system incident/Prometheus 专用指标：数据库健康和熔断已具备，但独立系统事件
  的接收人和递归路由契约尚未冻结，不能伪装成网元告警。
- 测试发送 API：现有 verify 只握手、不发送。测试发送仍需独立权限、明确测试地址和审计
  契约后实现。
- SMTP 仅沿用当前项目已有的明文/STARTTLS 配置，不额外引入 implicit TLS 465。
- SMS 继续等待外部 Kafka/供应商协议冻结，不因老 OMC 曾使用某种方式而提前引入依赖。
