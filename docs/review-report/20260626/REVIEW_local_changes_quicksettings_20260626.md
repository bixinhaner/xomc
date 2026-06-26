# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-06-26 |
| 范围 | 本地未提交改动（quicksettings 刷新卡住相关） |
| 审查方式 | 静态审查 + 问题视图检查（未执行全量集成测试） |
| 结论 | 本轮不阻断：接受队列极端过载下丢弃风险，优先从 DB 根因治理 |

## Findings

### 🟡 Accepted Risk（暂缓）

1. 过载时仍可能触发“最终丢消息”（本轮暂不处理）
- 证据：在 [omcgo/internal/provision/engine.go](omcgo/internal/provision/engine.go#L1079) 使用非阻塞入队，队列满直接返回错误（[omcgo/internal/provision/engine.go](omcgo/internal/provision/engine.go#L1080)）。
- 证据：错误会进入 NATS 重试决策；达到 maxDeliveries 后会 Term 永久丢弃（[omcgo/internal/core/event/nats_bus.go](omcgo/internal/core/event/nats_bus.go#L15), [omcgo/internal/core/event/nats_bus.go](omcgo/internal/core/event/nats_bus.go#L157), [omcgo/internal/core/event/nats_bus.go](omcgo/internal/core/event/nats_bus.go#L162), [omcgo/internal/core/event/nats_bus.go](omcgo/internal/core/event/nats_bus.go#L211)）。
- 风险：当 DB 持续慢于消费能力时，内部队列会长时间满，消息在多次 NAK 后仍可能被 Term，结果是 last_param_sync_at 仍可能缺失，前端症状复发。
- 处理决策：该问题本轮风险接受，不作为当前 PR 阻断项；后续由 DB 治理优先收敛负载，再评估是否需要调整事件总线终止策略。

## DB Priorities（本轮主线）

1. 优先治理高频慢 SQL 与扫表路径，降低 GPV 下游写库时延。
2. 收敛批量任务并发，避免与 quicksettings 同时竞争 DB 资源。
3. 建立 DB 观测基线（p95/p99 延迟、rows returned、update rows/s、dead tuple 比例）。
4. 以业务指标验收：刷新成功率、last_param_sync_at 写入及时性、5 分钟内告警/投诉量。

## Testing Gaps

1. 未看到针对以下关键路径的自动化测试：
- 队列满 -> NAK -> 最终 Term 的行为验证。
- 同设备分片保序（batch-0/1/2/3）与跨设备并发公平性。
- 关闭流程中 worker 的退出与未完成任务处理语义。

2. 本次仅做静态审查与问题检查，未执行：
- go test ./...
- omcmb 三皮肤 typecheck 与行为回归

## Change Summary

本地改动方向总体正确，已经覆盖了此前 RCA 中最关键的止血点：
- NATS 订阅 pending limits 增强。
- GPV 回调从“同步重处理”改为“轻回调 + worker 消费”。
- finalizePathBSync 中 deviceInfoRefresher 失败改为 non-fatal。
- quicksettings watcher 增加拥塞提示与 60s 超时收敛；v2/v3 挂载 core watcher。

当前实现仍存在“队列满后最终 Term 丢消息”风险，但本轮按团队决策暂不处理该项，先以 DB 根因治理作为主线推进。
