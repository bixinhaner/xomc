# Code Review — T-0157 C6 入队失败兜底（CreateFailureNotifier 注入）

- **日期**：2026-05-19
- **范围**：`internal/task/service.go`（字段 / setter / notifyCreateFailure / 两处 err 路径）/ `internal/notification/{task_failure_notifier,task_failure_notifier_test}.go` / `cmd/app/bootstrap.go`（注入）
- **作者**：Claude
- **Reviewer**：Claude（self-review，架构 + Go 工程视角）
- **关联**：T-0157 Phase 1 sub-task **C6**

---

## 变更概要

1. `task.CreateFailureNotifier` 类型 + `TaskService.SetCreateFailureNotifier` setter（与 SetMetrics/SetEventBus 一致的 optional 注入模式）
2. `TaskService.notifyCreateFailure` 私有 helper：CreatorID 空 / notifier nil / panic 都安全跳过
3. `CreateTask` 两处 err 路径调 notifyCreateFailure：repo.Create 失败 + queue.Push 失败
4. `notification.NewCreateFailureNotifier(svc, log)` 工厂返回 closure（task 包不依赖 notification，避免循环）
5. `cmd/app/bootstrap.go` 注入：创建 notifSvc → SetCreateFailureNotifier
6. 5 个新测试覆盖核心场景

---

## 出口门验证

| 门 | 结果 |
|---|---|
| `go build ./...` | ✅ |
| `go test ./internal/notification/... ./internal/task/...` | ✅ 全绿 |
| 5 个 CreateFailureNotifier 测试 PASS | ✅ |
| 入队失败两条路径覆盖（DB / Redis） | ✅ |
| CreatorID 空跳过 | ✅ |
| nil task 不 panic | ✅ |

---

## 解决了的设计难题

| 难题 | 解决 |
|---|---|
| **避免改 20 处 CreateTask 调用点** | 收敛到 TaskService 单点，所有 handler/service 零改动 |
| **避免 task → notification 循环依赖** | task 定义 type 别名 `CreateFailureNotifier`；notification 实现 closure 工厂；bootstrap 时 wire；二者不直接 import |
| **避免 notifier 异常影响 CreateTask 错误链** | notifyCreateFailure 包 defer recover；closure 内部 repo 失败仅 warn 不 panic |
| **避免与订阅器路径重复写入** | 入队失败时 task.created 永远不会 publish（在 push 之后），订阅器无事件可消费；dedup_key=task.ID 互斥 |

---

## Checklist

| 类别 | 项 | 结果 |
|---|---|---|
| 架构 | 单向依赖：notification → task（订阅器 + closure 工厂）；task 反向引用仅 type 别名 | ✅ |
| 接口设计 | optional 注入 setter 与既有 (SetMetrics/SetEventBus/SetConnectionRequester) 一致 | ✅ |
| 并发安全 | 设置/读 createFailureNotifier 在初始化期，无 race | ✅ |
| 错误隔离 | notifier panic / nil / CreatorID 空都不影响主返回 | ✅ |
| 业务正确 | repo.Create 失败 → 通知 + 错误返回（不回滚，本来就没写入）；queue.Push 失败 → 回滚 PG + 通知 + 错误返回 | ✅ |
| 文案 | "参数设置 · 设备 SN001 · 入队失败" + content=errMsg；与 §4.5 风格一致 | ✅ |

---

## 发现

### CRITICAL
- 无

### WARNING
- 无

### INFO
- I-01：acs/worker 进程不调 CreateTask（仅 app 调），它们各自的 NewTaskService 不需要 SetCreateFailureNotifier。验证：`grep .CreateTask omcgo/cmd` 零命中（与 C1 阶段一致）。
- I-02：当 repo.Create / queue.Push 都成功后才 publish task.created；所以入队失败时**永远不会**触发 task.created 事件，订阅器与本 closure 路径互斥，不会出现两条消息。
- I-03：notifier 工厂内部用 task.ID 作 dedup_key —— 但失败的 task 实际上没存入 PG（repo.Create 失败时 task.ID 已生成但未 INSERT；queue.Push 失败时 PG 行已被 Delete 回滚）。dedup_key 在 notifications 表里指向一个不存在的 task，不影响功能（只是历史 task_id），仅对 §4.3 流程图中"link 反查 task"是潜在 follow-up，但 V1 link 直接指向设备详情而非 task，无关。
- I-04：错误信息（errMsg）原样写入 content。后端原始错误可能含 "redis: ..." 这种实现细节，对终端用户不够友好。V2 可加 translation map，登记 §11 L-04（如必要时）。当前接受，因为入队失败极其罕见。

---

## 结论

**PASS**

可合入，可继续 C7（AddObject/DeleteObject 返 task_id）。
