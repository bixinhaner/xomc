# 邮件短信通知中心 Task 8 本地验收记录

日期：2026-08-05

范围：Inbox 顺序投影与持久化调度，不包含规则 API、收件人解析、模板渲染及真实邮件/短信外发。

## 实施结论

Task 8 本地开发门禁通过：

- `notification-lifecycle-v1` 使用固定 durable 和 `occurrence_id` 有界分片；数据库事务仍负责多实例并发下的最终顺序收敛。
- Inbox 以 `event_id` 幂等；旧版本只保留审计，版本缺口进入 waiting，缺失版本到达后连续补洞。
- ack、unack、clear 推进 `schedule_generation` 并取消旧代际任务；unack 只恢复仍在未来的既有 repeat，不重建 initial，也不扩张原规则次数。
- schedule 通过单条 PostgreSQL 条件更新和 `FOR UPDATE SKIP LOCKED` 领取；活租约不可重复领取，过期租约可恢复，完成/取消/释放均校验 worker 所有权。
- `Scheduler.RunOnce` 只支持 `initial_gate`、`repeat`、`digest_flush`、`quiet_hours_release` 四类显式 handler；没有后台 sleep，也没有在本任务提前启动空轮询。
- canonical 激活前同时检查北向和通知 Inbox 两个 durable 已追平；shadow 只写审计投影，不创建 delivery。

## 验证证据

```text
go test ./internal/notification -count=1
PASS

go test ./cmd/app/provider ./cmd/worker -count=1
PASS

go vet ./internal/notification ./cmd/app/provider ./cmd/worker
PASS

NOTIFICATION_LIFECYCLE_TEST_DSN=postgres://.../omcgo_notification_lifecycle \
  go test ./internal/notification -run TestPgLifecycleRepository_Integration -count=1 -v
PASS
```

集成测试使用本地 shadow PostgreSQL 容器中的专用数据库
`omcgo_notification_lifecycle`，先应用当前完整 `000001_init_schema.sql`，再验证：

1. v1 创建 occurrence，重复事件幂等；
2. v3 先到进入 waiting，v2 到达后一次事务推进至 v3；
3. 旧 v2 只记 ignored；
4. ack/unack/clear 的代际为 1 → 2 → 3 → 4；
5. unack 恢复 future repeat 且不恢复 initial；
6. 两个 repository 实例不能重复领取活租约，31 秒后可回收 30 秒租约；
7. 旧 worker 完成已转移租约时返回 `ErrScheduleLeaseLost`。

## 部署边界

现有 `xomc-email-sms-shadow` 的 `omcgo` volume 创建于 Task 7 基线之前，只读检查确认
不存在 `notification_events`、`notification_occurrences`、`notification_schedules`。本任务没有
向该旧 volume 局部灌表，也没有重建其数据；避免形成“部分基线”环境。下一次应用部署应使用
干净 volume 完整应用最新 `000001` 后再启动 Task 8 消费者。

当前仍禁止 canonical 生产切换和任何真实外部发送。
