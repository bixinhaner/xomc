# Code Review — T-0157 C5 task → notification 订阅器

- **日期**：2026-05-19
- **范围**：`internal/core/event/subjects.go`（+1 主题常量）/ `internal/task/service.go`（CreateTask publish）/ `internal/notification/{task_subscriber,task_subscriber_test}.go` / `internal/device/{device_service,device_param_handler}.go`（SetParameters 加 creatorID）/ `cmd/worker/main.go`（注册）
- **作者**：Claude
- **Reviewer**：Claude（self-review，架构 + Go 工程 + 电信业务视角）
- **关联**：T-0157 Phase 1 sub-task **C5**（最大块 / 关键路径 2 收官）

---

## 变更概要

1. **新主题** `event.SubjectTaskCreated = "task.created"`（CompletionRouter / event_bridge 不订阅，零侵入）
2. **task.CreateTask 末尾** publish `task.created` 事件（仅 SourceID 非空时；与 notifyCompletion 一致）—— 等等 ⚠️ 见 W-01
3. **TaskSubscriber 新组件**（`task_subscriber.go` ~210 行 + `task_subscriber_test.go` 6 测试 + 3 sub-tests）
4. **device.SetParameters 加 creatorID 参数** + handler 传 `admin.UserIDStringFromCtx(c)` —— 让快速设置路径写出 user_id
5. **worker/main.go 注册** 订阅器（在 sweeper 之前；二者顺序不影响功能）
6. **§11 登记 3 个遗留**：L-01 其他 CreateTask 调用点未填 CreatorID / L-02 created_at 取 DB NOW() / L-03 文案只显示新值

---

## 出口门验证

| 门 | 结果 |
|---|---|
| `go build ./...` | ✅ |
| `go test ./internal/notification/... ./internal/task/...` | ✅ 全绿 |
| 6 个 subscriber 单测 + 3 sub-tests 全 PASS | ✅ |
| 6 个状态事件分流（created/completed + failed×3） | ✅ |
| dedup_key 升级（queued→completed 不重复插入） | ✅ |
| user 隔离（alice 2 条 / bob 1 条） | ✅ |

---

## Checklist

| 类别 | 项 | 结果 |
|---|---|---|
| 架构 | 订阅器在 worker 进程，不与 app/acs 资源竞争 | ✅ |
| 主题路由 | task.created 仅订阅器消费，CompletionRouter/event_bridge 不订阅 → 零侵入 | ✅ |
| 状态机映射 | task.failed 主题按 task.Status 分流到 failed/expired/cancelled | ✅ |
| 幂等 | dedup_key=task.ID 走 UpsertByDedup，多次状态变更 upsert 同行 | ✅ |
| 系统任务隔离 | CreatorID 空时跳过，PeriodicSyncer / F09 等不打扰用户 | ✅ |
| 错误隔离 | Subscribe 单个主题失败立即返回；handler 单 task 失败仅 warn 不阻塞 | ✅ |
| 资源 | 共享 worker.PgPool / EventBus，无额外连接 | ✅ |
| 测试 | mock repo 配合 ChannelEventBus 已可测全链；NATSBus 集成留 docker E2E | ✅ |
| 文案 | translateMethod 覆盖 8 个 RPC 方法 + statusVerb 覆盖 6 态 | ✅ |
| 链接 | linkTo 统一指向 `/device/detail/{sn}?tab=quickSettings` —— C10 前端验证 | ✅ |

---

## 发现

### CRITICAL
- 无

### WARNING（已修）
- ~~W-01~~：`task.CreateTask` publish `task.created` 原条件 `SourceID != ""` 与快速设置路径（不填 SourceID）冲突。**已在本 commit 内修正**：publish 条件改为 `CreatorID != ""`，与订阅器 upsertFromTask 跳过逻辑统一。重跑测试全 PASS。

### INFO
- I-01：worker/main.go 新建 `notificationSvc` 时传 `nil` 给 hub（SSE 推送暂不接 worker 进程；SSE channel 在 app 进程的 events.MessageHub）。如需 worker → SSE 推送，需把 hub 通过 NATS pub-sub 跨进程联通（Phase 3）。
- I-02：旧 `TaskService.notifyCompletion` 把 expired 映射到 task.failed 主题（C2 已落地），订阅器在 handleFailed 内按 task.Status 区分 expired/cancelled/failed —— 与 §4.6 注释一致。
- I-03：3 个遗留 L-01/L-02/L-03 已登记 §11；L-01 是用户实测最可能踩的（其他模块没消息），需优先在 Phase 2 推进。
- I-04：device_service.SetParameters 签名 break：仅一个调用方（device_param_handler）已同步改完。E2E 验证 typecheck 全过。

---

## 结论

**PASS**

可合入，可继续 C6（入队失败兜底）。

