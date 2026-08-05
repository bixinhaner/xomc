# OMC 邮件短信通知中心里程碑 B 本地后端验收

## 结论

2026-08-05 在隔离 PostgreSQL、回环测试 SMTP 和 `.invalid` 收件人上完成本地后端验收，
结果通过。验收覆盖告警生命周期调度、逐收件人邮件投递、失败退避、未知结果、渠道熔断和
旧邮件规则迁移 barrier；没有连接真实邮件服务、真实收件人或老 OMC 写接口。

本结论是“本地后端链路可进入下一阶段”，不是生产外发放行。真实 SMTP 的 TLS、认证、
网络策略、域名信誉和投递到达率，仍需在可控测试环境做上线前验证。

## 隔离环境

- 当前分支：`codex/email-sms-notification-design`
- PostgreSQL：本地容器 `xomc-email-sms-shadow-postgres-1`，端口 `25432`
- 专用库：`omcgo_notification_milestone_b_smtp`、
  `omcgo_notification_milestone_b_lifecycle`
- Schema：由当前分支 `migrations/000001_init_schema.sql` 从零迁移到 version 1
- SMTP：测试进程内监听 `127.0.0.1` 随机端口；地址使用 `example.invalid`

未重启或改写现有 app/worker 容器，未修改老 OMC 数据。

## 业务验收结果

| 场景 | 结果 | 关键证据 |
| --- | --- | --- |
| raised + minimum duration | 通过 | initial gate 按 `RaisedAt + MinimumActiveDuration` 计算，延迟处理不重置持续时间 |
| bounded repeat | 通过 | repeat 次数和时点受规则上限约束；普通 update 不重复首发 |
| ack / unack | 通过 | ack 取消旧代 schedule；unack 仅恢复未来 repeat，不重建 initial |
| clear / recovery | 通过 | 门槛前 clear 形成可审计 suppression；恢复仅配对 accepted/handoff 原收件人渠道 |
| Worker 最终栅栏 | 通过 | claim 后再次校验 occurrence version、schedule generation、渠道启用和熔断状态 |
| 逐收件人 SMTP | 通过 | 每条 delivery 独立 SMTP envelope，捕获报文只有一个 `To` 地址 |
| accepted 语义 | 通过 | SMTP DATA 成功后记录 `completed + accepted`，未误记为 `delivered` |
| 临时失败与退避 | 通过 | 三次真实 `451 RCPT` 分别产生 failed attempt 和 `retry_wait` |
| 熔断 | 通过 | 第三次连续临时失败打开 circuit；成功 verify 关闭；认证失败立即打开 |
| unknown | 通过 | DATA 最终响应丢失或 Worker lease 过期均记 unknown，不盲目立即重发 |
| 迁移兼容 | 通过 | preview/overlap/shadow 门禁、首命中语义和 PG barrier 切换均通过 |

SMTP 专用库最终只读汇总证据：

```text
delivery: completed/accepted=2, completed/unknown=1, retry_wait/failed=3
attempt:  accepted=3, provider_temporary/failed=3, outcome_unknown=1
health:   open, last_error_category=authentication_error
```

## 执行记录

以下检查通过：

```bash
go test ./internal/notification -run 'BuildOrchestrationDecision|EmailWorker|EmailSender' -count=1 -v

NOTIFICATION_DOMAIN_TEST_DSN=... \
  go test ./internal/notification -run '^TestPgDomainRepository_Integration$' -count=1 -v

NOTIFICATION_LIFECYCLE_TEST_DSN=... \
  go test ./internal/notification -run '^TestPgLifecycleRepository_Integration$' -count=1 -v

NOTIFICATION_ORCHESTRATION_TEST_DSN=... \
  go test ./internal/notification -run '^TestPgOrchestrationRepository_Integration$' -count=1 -v

ALARM_FILTER_MIGRATION_TEST_DSN=... \
  go test ./internal/alarm -run 'Migration|Barrier|FirstMatch' -count=1 -v

go test ./internal/notification -run Migration -count=1 -v
go vet ./...
```

`go test ./... -count=1` 中通知、告警及其依赖包通过，但全仓未全绿：

- `internal/paramsync` 默认连接 `localhost:5432/omcgo` 的旧 schema，缺少
  `admission_queued_at`；改为当前基线专用库后复跑通过。
- `internal/task` 的 `TestService_PG_GetTask_TerminalTombstoneReturnsDurableDetails`
  复跑仍失败，并伴随测试临时 Redis 端口失联。本分支未修改该模块，作为独立基线问题保留。

## 精简性 review

验收没有新增产品服务、常驻 SMTP 容器、迁移文件或业务分支。唯一持久化改动是增强既有
集成测试：复用原测试 SMTP，增加可配置的 RCPT 响应，并把原先手工写入三次临时失败的
测试替换为真实 SMTP `451` 会话。该改动提高链路证据，不增加生产代码复杂度。

## 上线前剩余门禁

1. 在运营商可控测试环境配置真实测试 SMTP，验证 TLS、认证、出口 ACL 和超时参数。
2. 使用可回滚的真实小基站告警样本执行 raised/ack/unack/clear，并核对告警详情通知轨迹。
3. 以真实 Shadow 样本确认 legacy/new 命中和收件人数一致后，才允许分范围切换 barrier。
4. Task 13 管理页面完成后执行真实浏览器权限、i18n、表单和历史详情验收。
5. SMS adapter 未实现/未启用，不把本次邮件验收外推为短信验收完成。
