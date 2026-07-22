# 参数同步 DB recovery 与 result worker 改造代码审查

- 审查时间：2026-07-16 16:11 CST
- 审查范围：
  - `omcgo/internal/paramsync/reconciler.go`
  - `omcgo/internal/paramsync/result_consumer.go`
  - `omcgo/internal/core/event/nats_bus.go`
  - `omcgo/cmd/app/provider/paramsync.go`
  - `omcgo/internal/core/appconfig/config.go`
  - `omcgo/internal/paramsync/handler.go`
  - 相关测试与前端类型/展示改动
- 已运行验证：
  - `go test -count=1 ./internal/paramsync ./internal/core/event ./cmd/app/provider ./internal/core/appconfig`
  - `npm run typecheck` (`omcmb/webcode`)
  - `git diff --check`

## 总体结论

本轮复审已重新覆盖当前未提交 diff，未再保留已经修复的问题。当前代码已经把 DB recovery 从“重新发布缺失 NATS result”改为优先复用 `PGResultProcessor.Process()` 直接收口，并且 result 消费链路已经具备按设备 shard、可配置 pull batch/concurrency/AckWait/MaxAckPending、既有 durable consumer 配置兼容、指标、recovery 全局 task budget 和真实 PG processor 集成测试。

复审发现的 pull loop 隐形队列与 MaxAckPending 配置问题已在本轮修复。当前未发现新的必须阻断合入的问题。

## 问题清单

暂无未关闭问题。

## 本轮确认点

- `runPullSubscription()` 已按可用并发槽限制每轮 `Fetch()` 数量，避免已拉取但未进入 handler 的消息在本地隐形排队消耗 AckWait。
- `PullTuning` 已纳入 `MaxAckPending`，`param_sync.task.result` 默认使用独立值 `512`，app 配置新增 `param_sync.result_consumer_max_ack_pending`。
- `PullSubscribe()` 已在订阅既有 durable consumer 时优先通过 `UpdateConsumer()` 覆盖 AckWait/MaxAckPending，避免升级后因本地配置变更导致 app 启动失败；若更新失败才沿用服务端配置继续启动，同时不删除已有 consumer，保留 pending 消息。
- 已补充低 concurrency/high batch 的纯函数回归测试，覆盖 Fetch 数量不会超过可用并发槽。

## 当前可接受点

- `ResultConsumer` 仍保持 handler 等待 worker 完成后才返回，ACK-after-commit 语义没有被破坏。
- 同设备优先 hash 到同一 shard，可降低同设备/run 内 result 并发写造成的额外竞态。
- `RecoverMissingResults()` 复用 `PGResultProcessor`，没有绕过 staging、失败收敛、finalize、terminal outbox 等业务语义。
- `stalled_finalizing` 已同步到前端接口类型和参数页展示，当前顶层字段契约前后端一致。
- ACS 直接发布 result 与 `TaskTerminalBridge` 转换 terminal task 两条入口可能产生同 task 重复事件；当前由 `(run_id, task_id)` 幂等写入吸收，属于可接受的 at-least-once 行为，但压测时需要把重复事件计入 NATS/result consumer 压力。

## 建议修复顺序

无需继续修复。若后续现场压测出现 redelivery 或 PG 背压，再基于 `result_consumer_pull_concurrency`、`result_consumer_max_ack_pending`、shard queue 指标和 DB 连接池指标做容量调参。
