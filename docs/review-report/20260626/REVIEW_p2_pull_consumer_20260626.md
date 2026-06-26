# Code Review Report — P2 Pull Consumer 迁移

| 项目 | 值 |
|------|-----|
| 日期 | 2026-06-26 |
| Commit | `3623b40b` |
| Branch | `feat/provision-gpv-pull-consumer` |
| 范围 | `provision-gpv` 订阅从 push consumer 迁移至 pull consumer |
| 审查文件 | `internal/core/event/bus.go` · `internal/core/event/nats_bus.go` · `internal/core/event/channel_bus.go` · `internal/provision/engine.go` · `internal/provision/engine_test.go` · 16 个测试 mock 文件 |
| 审查方式 | 静态审查 + 本地 docker compose 部署验证（NATS consumer 注册 + 日志观察） |
| 结论 | **R1（DeliverNew 缺失）需修复后方可合并；其余接受或后续处理** |

---

## Background

P0（PR #688）通过异步 GPV 处理 + NATS buffer 扩容缓解快速设置刷新卡住问题，但根因是 push consumer 的 `MaxAckPending` 无上限，突发 GPV response 洪峰会导致消费者积压无法背压。  
P2 通过引入 pull consumer，由消费方主动 Fetch，天然实现速率控制（`MaxAckPending=2048`），从架构上封堵该问题。

本次变更新增的关键代码路径：
- `EventBus.PullSubscribe` 接口方法
- `NATSEventBus.PullSubscribe` + `runPullSubscription` goroutine
- `pullSubscription.Unsubscribe`（带 drain 等待）
- `ProvisioningEngine.Subscribe` 中 GPV 订阅改为 `PullSubscribe`

---

## Findings

### 🔴 R1 — 首次部署会重播历史 GPV response（需修复）

**位置**：[internal/core/event/nats_bus.go](internal/core/event/nats_bus.go#L122)

**问题**：`js.PullSubscribe` 未指定 `DeliverPolicy`，NATS 默认 `DeliverAll`。  
首次创建 `provision-gpv-pull` durable consumer 时，NATS 会从 COMMAND stream 的第一条可用消息开始投递，导致所有历史 `command.get_parameters.response` 消息被重新消费。

```go
// 当前代码（存在问题）
sub, err := b.js.PullSubscribe(subject, durable,
    nats.AckExplicit(),
    // ← 未指定 DeliverPolicy，默认 DeliverAll
    nats.AckWait(gpvPullAckWait),
    nats.MaxAckPending(gpvPullMaxAckPending),
    nats.MaxDeliver(maxDeliveries),
)
```

**影响评估**：
- `handleGPVResponse` 是参数 upsert 语义，重播不会损坏数据
- 但启动时会出现 CPU / DB 写入峰值（取决于 COMMAND stream 存量）
- 后续重启不受影响（durable consumer 记录已投递位置）

**修复方案**：

```go
sub, err := b.js.PullSubscribe(subject, durable,
    nats.AckExplicit(),
    nats.DeliverNew(),   // 新增：首次创建从当前 stream 尾部开始，不重播历史
    nats.AckWait(gpvPullAckWait),
    nats.MaxAckPending(gpvPullMaxAckPending),
    nats.MaxDeliver(maxDeliveries),
)
```

---

### 🟡 R2 — Pull subscription 在两条路径下可 double-unsubscribe（低风险，接受）

**位置**：[internal/core/event/nats_bus.go](internal/core/event/nats_bus.go#L143)

**问题**：`b.subs` 存储 pull consumer 的原始 `*nats.Subscription`。  
当 `pullSubscription.Unsubscribe()` 被外部调用后，`Close()` 仍会对同一底层 sub 再次调用 `Unsubscribe()`，产生 NATS 错误并触发 `b.logger.Warn("unsubscribe error")`。

**影响**：无数据问题，仅 shutdown 时出现一条无意义 warn 日志。

**处置决策**：接受，不阻断合并。当前调用点（`engine.go`）均丢弃返回的 `Subscription`，外部 `Unsubscribe()` 实际不会被触发。

---

### 🟡 R3 — `queue` 参数语义在 `PullSubscribe` 与 `QueueSubscribe` 中不一致（文档问题）

**位置**：[internal/core/event/bus.go](internal/core/event/bus.go#L24)

**问题**：  
- `QueueSubscribe` 的 `queue`：既是 NATS queue-group 名，也是 durable consumer 名  
- `PullSubscribe` 的 `queue`：仅作为 durable 名前缀（实际 durable = `queue + "-pull"`）

doc comment 未区分，对接口实现者可能造成歧义。

**处置决策**：补充注释，不阻断合并。

---

### 🟡 R4 — Pull consumer 配置全局硬编码，无扩展点（设计注意）

**位置**：[internal/core/event/nats_bus.go](internal/core/event/nats_bus.go#L24)

**问题**：`gpvPullBatchSize=64`、`gpvPullMaxAckPending=2048`、`gpvPullAckWait=30s` 是模块级常量，`PullSubscribe` 接口不携带配置参数。若未来新增第二个 pull consumer，将自动继承 GPV 优化参数（可能不适合该场景）。

**现状**：全代码库目前仅 `provision-gpv` 一处 `PullSubscribe` 调用，风险受控。

**处置决策**：接受现状，加注释标记，不阻断合并。

---

### ℹ️ R5 — `runPullSubscription` fetch 循环无单元测试（覆盖缺口）

**位置**：`internal/core/event/nats_bus.go`

**缺失覆盖**：
- `ErrBadSubscription` 触发退出
- `context.DeadlineExceeded` 静默 continue
- 其他未知错误 → warn + 100ms 退避
- 优雅 shutdown：`ctx` cancel → goroutine 退出 → `done` channel 关闭

**原因**：需要 embedded NATS server，成本较高。

**处置决策**：登记后续补充，不阻断合并。

---

### ℹ️ R6 — `pullDurableName` 无独立单元测试（低优先级）

**位置**：`internal/core/event/nats_bus.go`

**缺失用例**：空串 → `"pull"`，已含 `-pull` 后缀 → 原样返回，普通串 → 追加 `-pull`。

**处置决策**：成本极低，建议随 R5 一并补充。

---

## 本次 Review 前已修复的问题（本次 commit 已包含）

| # | 问题 | 触发时机 | 修复方式 |
|---|------|---------|---------|
| B1 | `nats: option Durable set more than once` | 进程启动 | 删除 `nats.Durable(durable)` option（`js.PullSubscribe` 第 2 参数已设置） |
| B2 | `nats: context and timeout can not both be set` | Fetch 循环每次迭代 | 用 `context.WithTimeout(ctx, 500ms)` 替换同时传 `nats.Context(ctx)` + `nats.MaxWait()` |
| B3 | `context deadline exceeded` 刷屏 warn | 无消息时空轮询 | 将 `context.DeadlineExceeded` 纳入静默 continue 分支，与 `ErrTimeout` 同等处理 |
| B4 | import 顺序：`"encoding/json"` 在 `"errors"` 后 | lint | 修正为标准字母序 |
| B5 | `Close()` 不等 pull goroutine 退出 | shutdown 竞争 | 新增 `pullDones []chan struct{}` 字段，`Close()` 先 drain 所有 goroutine 再 unsubscribe |

> B1–B3 通过本地 docker compose 部署验证发现，单元测试未覆盖真实 NATS 连接场景。

---

## 参数吞吐量估算

```
单 replica 峰值理论值：
  Fetch batch = 64 条
  Fetch 窗口  = 500 ms（无消息时超时）
  handler 耗时 ≈ <1 ms（enqueueGPVResponseEvent 仅入 channel）

  吞吐上限 ≈ 64 / 0.001 ≈ 64,000 msg/s（handler 决速）
  背压触发  ≈ MaxAckPending = 2048 条未 Ack

适合 10 万基站规模的 GPV response 处理（正常场景 <1000 msg/s）。
```

---

## 测试验证状态

| 测试类型 | 结果 |
|---------|------|
| `go test ./internal/core/event ./internal/provision -count=1` | ✅ PASS |
| `go test ./... -run TestDoesNotExist -count=0`（全量编译）| ✅ PASS |
| docker compose no-cache rebuild + 启动 | ✅ PASS |
| NATS consumer 注册验证（jsz HTTP 监控接口）| ✅ `provision-gpv-pull` 出现在 COMMAND stream |
| GPV response 消息处理日志 | ✅ `received GPV response` / `path-b parameter values saved` 出现 |
| 空轮询噪音 | ✅ 无 `context deadline exceeded` warn 输出 |

---

## Action Items

| 优先级 | ID | 动作 | 负责人 |
|--------|-----|------|--------|
| 🔴 Merge-blocker | R1 | `PullSubscribe` 追加 `nats.DeliverNew()` | agent |
| 🟡 Follow-up | R2 | 评估是否从 `b.subs` 排除 pull sub，消除双重 unsubscribe | 下次迭代 |
| 🟡 Follow-up | R3 | `bus.go` `PullSubscribe` doc comment 补充 `queue` 语义说明 | 下次迭代 |
| 🟡 Follow-up | R4 | `nats_bus.go` 常量处添加注释：GPV 专用，新增 pull consumer 需评估参数 | 下次迭代 |
| ℹ️ Tech-debt | R5 | 为 `runPullSubscription` 补充 embedded NATS 单元测试 | 下次迭代 |
| ℹ️ Tech-debt | R6 | 为 `pullDurableName` 补充单元测试 | 随 R5 |
