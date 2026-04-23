# BridgeQueue 下线可行性分析报告

> **日期**：2026-04-23
> **背景**：`docs/消息队列业务流转详细说明-20260422.md` §5.2 / §10.1 记载 `BridgeQueue` 作为 `cmdqueue.CommandQueue` 接口的兼容适配器，底层已委托 `task.TaskService`，不再写 `acs:cmdq:*`。
> **问题**：**不保留 BridgeQueue、直接使用 taskq（`task.TaskService`）是否可行？**
> **结论**：**技术可行，但不是一次 PR 能完成的改动**——核心阻塞不在 BridgeQueue 自身（107 行适配器），而在 **`cmdqueue.Command` 类型被 ACS `rpc.Dispatcher` 的 12 个 `BuildRequest` 方法签名深度锚定**，以及 5 个业务模块 / 25 个 `.go` 文件 / 9 个 `_test.go` 的 import 面。建议分 3 步（P1→P2→P3）推进，每步可独立编译、独立验证。

---

## 1. 现状快照

| 维度 | 事实 | 来源 |
|-----|------|------|
| BridgeQueue 代码量 | 107 行 | [internal/task/bridge_queue.go](omcgo/internal/task/bridge_queue.go) |
| cmdqueue 包代码量 | `queue.go` 155 行（含 `Deprecated` 注释）+ 单测 | [internal/acs/cmdqueue/queue.go](omcgo/internal/acs/cmdqueue/queue.go) |
| 生产写路径 | 所有 Push 已走 `TaskService.CreateTask` → Redis `acs:taskq:*` + PG `device_tasks` | [redis_queue.go](omcgo/internal/task/redis_queue.go) |
| `acs:cmdq:*` Key | **已停写**。`RedisCommandQueue` 仅在 `queue_test.go` 被构造 | [cmdqueue/queue.go §43-45](omcgo/internal/acs/cmdqueue/queue.go) |
| 依赖 cmdqueue 的 `.go` 文件 | **25 个**（含测试），合计约 148 行引用 | 调研统计 |
| Dispatcher BuildRequest 签名 | **12 个 Handler 接受 `*cmdqueue.Command`** | [rpc/dispatcher.go](omcgo/internal/acs/rpc/dispatcher.go) |
| 装配点 | `cmd/app/bootstrap.go` / `cmd/worker/bootstrap.go` 用 `task.NewBridgeQueue(taskSvc)` 填充 `Container.CmdQueue` | [cmd/app/bootstrap.go:56-60](omcgo/cmd/app/bootstrap.go) |

### 1.1 现状拓扑

```
┌─────────────────────────────┐
│ 业务模块（5 个）             │
│  provision/ config/         │
│  software/ filemanager/     │   构造 *cmdqueue.Command
│  interop/                   │──────┐
└─────────────────────────────┘      │
                                     ▼
                        ┌────────────────────┐
                        │ cmdqueue.CommandQueue (接口)
                        └────────────────────┘
                                     │
                                     ▼
                        ┌────────────────────┐     写 acs:taskq:*
                        │  BridgeQueue       │────────────────▶
                        │  (107 行适配器)    │   写 device_tasks (PG)
                        └────────────────────┘
                                     │
                                     ▼ 委托
                        ┌────────────────────┐
                        │  task.TaskService  │
                        └────────────────────┘
                                     ▲
                                     │ PopTask
                                     │
┌─────────────────────────────┐      │
│ ACS handler.go              │──────┘
│  Pop 后临时造 *cmdqueue.Command
│  喂给 rpc.Dispatcher.BuildRequest
└─────────────────────────────┘
           │
           ▼
 rpc/dispatcher.go ─── 12 个 Handler.BuildRequest(*cmdqueue.Command)
```

**关键观察**：当前链路在 Pop 侧 `task.Task` 反被"降维"成 `cmdqueue.Command` 再喂给 Dispatcher，这是 Dispatcher 签名锁死造成的**反向翻译成本**，而不是真正的解耦需求。

---

## 2. 能力对齐：taskq 是 cmdqueue 的严格超集

| 字段 | `cmdqueue.Command` | `task.Task` | 状态 |
|-----|-------------------|-------------|------|
| ID | ✓ | ✓ | 一致 |
| Method | ✓ | ✓ | 一致 |
| Params (`json.RawMessage`) | ✓ | ✓ | 一致 |
| Priority | ✓ | ✓ | 一致 |
| CommandKey | ✓ | ✓ | 一致 |
| CWMPID | ✓ | ✓ | 一致 |
| CreatedAt | ✓ | ✓ | 一致 |
| ExpiresAt | ✓ | ✓ | 一致 |
| DeviceSN | — | ✓ | Task 扩展 |
| Status（状态机） | — | ✓ | Task 扩展 |
| SentAt / CompletedAt | — | ✓ | Task 扩展 |
| Result / ErrorCode / ErrorMessage | — | ✓ | Task 扩展 |
| Source / SourceID / CreatorID | — | ✓ | Task 扩展（MML 扇出依赖） |
| RetryCount / MaxRetries | — | ✓ | Task 扩展 |

**事实结论**：`task.Task` 在字段层**完全覆盖**`cmdqueue.Command`，且多出生命周期 + 审计 + 重试 + MML 扇出所需字段。"类型替换"不存在语义损失，只是 **签名改造面广**。

`TaskService` 方法 vs `CommandQueue`：

| CommandQueue 方法 | TaskService 等价 | 差异 |
|-------------------|------------------|------|
| `Push(ctx, sn, cmd)` | `CreateTask(ctx, req)` | 参数更丰富，返回 `*Task` |
| `Pop(ctx, sn)` | `PopTask(ctx, sn)` | 一致 |
| `Peek(ctx, sn)` | `GetPendingTasks(ctx, sn, limit)` | 支持 limit |
| `Len(ctx, sn)` | `GetQueueLength(ctx, sn)` | 一致 |
| `Clear(ctx, sn)` | **无** | BridgeQueue 返回 nil；仅 provision/sync.go 2 处调用且都忽略返回值 |
| — | `MarkTaskSent/Completed/Failed` | Task 新增，状态机闭环 |

**`Clear()` 的特殊性**：该方法在 `BridgeQueue` 是空实现（直接 `return nil`），下游仅 `provision/sync.go` 两个 `_ = s.cmdQueue.Clear(...)` 调用 —— **既无人真正依赖这个语义，也无对应 taskq 能力**。下线时可安全去除，无需补建等价物。

---

## 3. 可行性判定

### 3.1 技术层：**可行（无阻塞）**

- taskq 能力覆盖 cmdqueue 全部字段与所有生产方法。
- `acs:cmdq:*` Redis key 已无生产写入，物理层已无耦合。
- BridgeQueue 本身只是 107 行类型翻译，没有隐藏状态、没有独立持久化、没有并发语义差异。

### 3.2 工程层：**不是一次 PR 能收工**

三个真正的改造成本点：

**(A) `cmdqueue.Command` 是 Dispatcher 的参数锚点（最硬）**
- [rpc/dispatcher.go](omcgo/internal/acs/rpc/dispatcher.go) 中 **12 个 RPC Handler** 的 `BuildRequest(cmd *cmdqueue.Command, cwmpID string) ([]byte, error)` 签名一致。
- 每个 Handler 内部通过 `json.Unmarshal(cmd.Params, &params)` 提取参数。
- 12 个对应的 `dispatcher_test.go` / `*_handler_test.go` 全部使用 `*cmdqueue.Command` 构造输入。
- **不改这里，就没办法删掉 cmdqueue 类型**。

**(B) 5 个业务模块显式构造 `*cmdqueue.Command` 再 Push（次硬）**

| 模块 | 文件 | Push 点数 |
|-----|------|----------|
| provision | `sync.go` / `model_upload.go` / `orchestrator.go` | 多处（最大户） |
| config | `sync_handler.go` | 2 |
| software | `service.go` | 2 |
| filemanager | `service.go` | 1 |
| interop | `runner.go` | 1 |
| acs（Inform 响应路径） | `handler.go` | 3 |

合计约 **10+ 个 Push 点**需要逐一改成 `taskService.CreateTask(ctx, &task.CreateTaskRequest{...})`。

**(C) 测试 fixture 面广（较软，但量大）**

9 个 `_test.go` 直接 mock 或构造 cmdqueue：
- [internal/acs/cmdqueue/queue_test.go](omcgo/internal/acs/cmdqueue/queue_test.go)（测 `RedisCommandQueue` —— 可直接删）
- [internal/acs/rpc/dispatcher_test.go](omcgo/internal/acs/rpc/dispatcher_test.go)（12 Handler 的 BuildRequest 测试 —— 必改）
- [internal/acs/handler_test.go](omcgo/internal/acs/handler_test.go)
- [internal/provision/engine_test.go](omcgo/internal/provision/engine_test.go) / `orchestrator_test.go`
- [internal/config/sync_handler_test.go](omcgo/internal/config/sync_handler_test.go)
- [internal/software/service_test.go](omcgo/internal/software/service_test.go)
- [internal/interop/runner_test.go](omcgo/internal/interop/runner_test.go)
- [internal/backup/executor_test.go](omcgo/internal/backup/executor_test.go)

---

## 4. 实施路线建议（3 阶段 / 渐进迁移）

核心思路：**不要一次把 cmdqueue 包删光**。按"先删 BridgeQueue、再搬类型、最后删包"的节奏，每一步都保留可编译 + 可回退状态。

### 4.1 Phase A（小步，1 Sprint 内可完成）—— 业务模块直连 TaskService，保留 `cmdqueue.Command` 类型

**目标**：**删掉 `BridgeQueue` + `cmdqueue.CommandQueue` 接口**，让业务模块不再通过"接口"写命令，改为直连 `TaskService.CreateTask`。`cmdqueue.Command` 类型**暂不动**，继续作为 Dispatcher 入参。

**动作**：
1. 在 `Container` 里删 `CmdQueue cmdqueue.CommandQueue` 字段；业务模块直接持有 `*task.TaskService`。
2. 5 个业务模块的 Push 点替换：
   ```go
   // Before
   cmd := &cmdqueue.Command{Method: "Download", Params: p, ...}
   s.cmdQueue.Push(ctx, sn, cmd)

   // After
   _, err := s.taskSvc.CreateTask(ctx, &task.CreateTaskRequest{
       DeviceSN: sn, Method: "Download", Params: p, ...,
   })
   ```
3. ACS handler.go Pop 侧**保留现状**——`PopTask` 返回 `*task.Task`，再降维造一个临时 `*cmdqueue.Command` 喂给 `Dispatcher.BuildRequest`。
4. 删除 [task/bridge_queue.go](omcgo/internal/task/bridge_queue.go) 和 `cmdqueue.CommandQueue` 接口定义（保留 `Command` 结构体）。
5. 删除 [internal/acs/cmdqueue/queue_test.go](omcgo/internal/acs/cmdqueue/queue_test.go) + `RedisCommandQueue` 实现（已无生产装配）。
6. 测试迁移：业务模块的 `_test.go` 把 `mockCommandQueue` 换成 `TaskService` 的 in-memory 实现或假 repository。

**收益**：
- BridgeQueue 兼容层彻底下线（本次问题的直接目标 ✓）。
- `cmdqueue.CommandQueue` 接口消失（-155 行）。
- Redis `acs:cmdq:*` 在代码中不再出现（仅 `const` 残留）。
- 业务模块拿到 `*task.Task` 返回值，可用于后续状态追踪。

**风险**：小。
- 改动集中在装配 + Push 点，无状态迁移。
- 每个模块独立 PR 可滚动合入。
- `Clear()` 调用点删除前需确认 provision/sync.go 的 2 处语义（即"清空设备队列"是否真有业务需要；从代码看不是）。

**剩余问题**：`cmdqueue.Command` 类型仍存在于 12 个 Dispatcher Handler 签名中 —— 这是 Phase B 的任务。

---

### 4.2 Phase B（中等，2 Sprint 内）—— 把 `Command` 类型搬家到 `internal/acs/cmd/`

**目标**：解除 `internal/acs/cmdqueue` 这个"过时包名"对 Dispatcher 的约束，把 `Command` 结构体搬到一个更合适的位置（建议 `internal/acs/cmd/` 或直接定义在 `internal/acs/rpc/`），让 12 个 BuildRequest 用上新名字。

**两种选择**：

**选项 B1（推荐，风险低）**：新建 `internal/acs/rpc/command.go`，把 `Command` 类型复制过来，重命名为 `rpc.Command`。Dispatcher 的 12 个 Handler 签名改为 `BuildRequest(cmd *Command, cwmpID string)`。ACS handler.go Pop 后的临时转换改为 `rpc.Command`。**删掉整个 `internal/acs/cmdqueue/` 包**。

- 改动量：12 Handler 签名 + 12 测试 + `handler.go` Pop 侧转换函数。import 改 ~15 个文件。
- 语义：Dispatcher 专属结构体，职责清晰。

**选项 B2（更激进，但更彻底）**：直接让 `BuildRequest` 接受 `*task.Task`，取消中间层转换。ACS handler.go 的 `PopTask` 返回值直接传给 Dispatcher。

- 改动量：12 Handler 签名 + 12 测试 + handler.go + 所有用 `task.Task` 子集的地方。import 改 ~20 个文件。
- 语义最干净，但 `task.Task` 比实际 Dispatcher 需要的字段多（Status/SentAt/Result 等 Dispatcher 用不到）——违反"接口接受最小依赖"原则。

**建议采用 B1**：Dispatcher 是 TR-069 协议层，不应依赖 task 包的状态机字段。`rpc.Command` = "下发一条 RPC 所需的最小输入"，`task.Task` = "带生命周期的业务实体"，职责分层清晰。

---

### 4.3 Phase C（收尾，0.5 Sprint）—— 物理清理

1. 运维清理生产 Redis 历史 `acs:cmdq:*` key（脚本 `redis-cli --scan --pattern 'acs:cmdq:*' | xargs redis-cli del`）。
2. 更新 [omcgo/CLAUDE.md §5.2](omcgo/CLAUDE.md) Redis Key 命名段，移除 `acs:cmdq:*` 的说明。
3. 更新 [docs/消息队列业务流转详细说明-20260422.md](docs/消息队列业务流转详细说明-20260422.md)：
   - §5.2 整节删除或标记"已下线"。
   - §10.1 Phase 1 标 ✅ 并记录完成日期。
   - §2 / §附录 A 中的 `bridge_queue.go` 条目删除。
4. 更新 [docs/消息队列使用分析-20260422.md](docs/消息队列使用分析-20260422.md) 方案 C 的对应章节。
5. 搜索全仓 `cmdqueue` / `acs:cmdq` / `CommandQueue` 残留引用。

---

## 5. 风险评估

| 风险项 | 级别 | 说明 | 缓解 |
|-------|-----|------|------|
| Phase A 破坏业务 Push | 🟡 中 | 5 模块 Push 点逐一替换，漏改会编译错 | 分模块 PR，CI 全绿再合入；`go vet ./...` 可早期发现 |
| Phase B Dispatcher 签名变更 | 🟡 中 | 12 Handler + 12 测试同步改，跨模块 | 用 IDE rename refactor 一次完成；签名变动集中于 rpc/ 目录 |
| `Clear()` 方法下线 | 🟢 低 | 仅 2 处调用且都忽略返回值，无业务语义 | 直接删，CI + E2E 兜底 |
| `cmdqueue.Command` 字段后续扩展 | 🟢 低 | Phase A 后该类型只为 Dispatcher 服务，变更面小 | Phase B 搬到 rpc/ 后就纯属 ACS 内部事务 |
| Redis 历史 key 残留 | 🟢 低 | 仅占空间，不影响功能（无人读） | Phase C 运维脚本清理 |
| 测试 Mock 迁移 | 🟡 中 | 9 个测试文件，mock 实现有细节差异 | 提供统一的 `TaskService` 测试替身（in-memory repo + in-memory queue） |
| MML 扇出兼容性 | 🟢 低 | MML 走的是 `TaskService.CreateTask` 直连，不经 BridgeQueue | 无额外风险 |
| 回滚能力 | 🟢 低 | Phase A 每个 commit 可独立回退；cmdqueue 包保留期间可 revert | git revert 即可 |

---

## 6. 工作量估算

| Phase | 内容 | 代码改动面 | 估算（1 人） |
|-------|------|-----------|-------------|
| A | 删 BridgeQueue + 接口、5 模块直连 TaskService | ~15 个 .go 文件、~8 个 _test.go | **5-7 人日** |
| B | Command 类型搬家到 rpc/、12 Handler 签名改造 | ~15 个 .go 文件、12+ _test.go | **3-5 人日** |
| C | 文档更新 + Redis 清理 + CLAUDE.md 同步 | 4-6 份 .md 文件 | **0.5-1 人日** |
| 合计 | | **~30 个文件 / 50+ import** | **9-13 人日** |

---

## 7. 对"直接删 BridgeQueue"的直接回答

> **问**：不保留 BridgeQueue，直接使用 taskq，是否可行？

**可行，但拆成两层看**：

1. **物理目标（删除 `bridge_queue.go` + `cmdqueue.CommandQueue` 接口）**：
   - **立刻可做**。Phase A 就是这个事情，5-7 人日。
   - 完成后业务模块直连 `TaskService.CreateTask`，不再有"兼容层"。

2. **终极目标（删除整个 `internal/acs/cmdqueue/` 包）**：
   - **需要多一步**（Phase B），因为 `cmdqueue.Command` 类型被 Dispatcher 签名锁定。
   - 再 3-5 人日即可彻底清除。

**不推荐的做法**：
- ❌ 一把梭把"删 BridgeQueue + 删包 + 改 Dispatcher"做成一次 PR —— 改动面 ~30 文件，review 难度大，回滚困难。
- ❌ 在 Phase A 完成前尝试改 Dispatcher 签名 —— 业务模块仍在用 `cmdqueue.Command`，会形成环状依赖。

**推荐的做法**：
- ✅ Phase A 先上 —— 本次问题的核心目标"不保留 BridgeQueue"即告完成。
- ✅ Phase B 随后作为独立 Sprint 做 —— 收尾物理清理。
- ✅ Phase C 跟 B 一起合入 —— 文档 + 运维。

---

## 8. 关键决策点（需人拍板）

1. **Dispatcher 入参选型（Phase B）**：`rpc.Command`（B1，推荐）vs `task.Task`（B2）？
2. **`Clear()` 下线**：provision/sync.go 的 2 处 `_ = s.cmdQueue.Clear(...)` 是否真的无业务依赖？（从代码看是的，但建议让 provision 模块 Owner 确认一次。）
3. **Phase A 是否需要一次合入**：业务模块 5 个 Push 点要不要分 5 个小 PR？还是打包一次？—— 建议**分模块 PR** 以便精细化 review 和回滚。
4. **Phase A 与 Phase B 节奏**：是否立刻安排 B？还是 Phase A 稳定 1-2 周再做 B？—— 建议**至少观察一个 Sprint** 再做 B，避免叠加风险。

---

## 9. 对照 §10.1 Roadmap 的增量

原 §10.1 列出的剩余任务（🟡 语义已迁完，物理删除待做）：

| 原计划项 | 本报告方案映射 | 状态 |
|---------|--------------|------|
| 把 `cmdqueue.Command` 类型搬到 `internal/task/` 或 `internal/acs/cmd/` | Phase B | 建议搬 `internal/acs/rpc/command.go`（不搬到 task，避免循环依赖） |
| 替换所有 `cmdqueue.CommandQueue` 字段为 `task.TaskService` | Phase A | 可先做，收益最快 |
| 删除 `bridge_queue.go` | Phase A | 同上 |
| 删除 `internal/acs/cmdqueue/` 包 | Phase C（紧跟 B） | 拆分后 |
| 运维清理 Redis `acs:cmdq:*` | Phase C | 无代码风险 |
| 更新 CLAUDE.md §5.2 | Phase C | 文档同步 |

**差异**：原 Roadmap 把 A+B 合并成"一个 Phase 3 Sprint"；本报告建议**拆成 A / B / C 三步**，让 Phase A 的收益（删 BridgeQueue）先兑现。

---

## 附录 A：涉及文件清单

**生产代码（必改）**：
- [internal/task/bridge_queue.go](omcgo/internal/task/bridge_queue.go)（删）
- [internal/acs/cmdqueue/queue.go](omcgo/internal/acs/cmdqueue/queue.go)（Phase A 删接口，Phase B 删整包）
- [cmd/app/bootstrap.go](omcgo/cmd/app/bootstrap.go) · [cmd/worker/bootstrap.go](omcgo/cmd/worker/bootstrap.go) · [cmd/app/provider/container.go](omcgo/cmd/app/provider/container.go)
- [internal/provision/sync.go](omcgo/internal/provision/sync.go) · [provision/model_upload.go](omcgo/internal/provision/model_upload.go) · [provision/orchestrator.go](omcgo/internal/provision/orchestrator.go)
- [internal/config/sync_handler.go](omcgo/internal/config/sync_handler.go)
- [internal/software/service.go](omcgo/internal/software/service.go)
- [internal/filemanager/service.go](omcgo/internal/filemanager/service.go)
- [internal/interop/runner.go](omcgo/internal/interop/runner.go)
- [internal/acs/handler.go](omcgo/internal/acs/handler.go)
- [internal/acs/rpc/dispatcher.go](omcgo/internal/acs/rpc/dispatcher.go)（Phase B）

**测试（必改）**：
- `internal/acs/cmdqueue/queue_test.go`（Phase A 删）
- `internal/acs/rpc/dispatcher_test.go`（Phase B 改）
- `internal/acs/handler_test.go`（Phase A 改）
- `internal/provision/{engine,orchestrator}_test.go`（Phase A 改）
- `internal/config/sync_handler_test.go`（Phase A 改）
- `internal/software/service_test.go`（Phase A 改）
- `internal/interop/runner_test.go`（Phase A 改）
- `internal/backup/executor_test.go`（Phase A 改）

**文档（Phase C 改）**：
- [omcgo/CLAUDE.md §5.2](omcgo/CLAUDE.md)
- [docs/消息队列业务流转详细说明-20260422.md](docs/消息队列业务流转详细说明-20260422.md)
- [docs/消息队列使用分析-20260422.md](docs/消息队列使用分析-20260422.md)

---

## 附录 B：一句话总结

**BridgeQueue 可以立即删除（Phase A，~1 周）；`cmdqueue` 包需要多一个 Sprint（Phase B）因为 Dispatcher 签名拖累；全部清理（含文档与 Redis 历史 Key）总计 9-13 人日，建议分 3 步走。**
