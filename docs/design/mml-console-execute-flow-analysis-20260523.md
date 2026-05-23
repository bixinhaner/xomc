# `POST /api/v1/mml/console/execute-statements-structured` 全链路分析

> **版本**：v1.2（2026-05-23）
> **范围**：单一 API 端到端 — Browser SPA → App → NATS / Redis → ACS → CPE → NATS task.* → CompletionRouter → SSE Hub → Browser 终端
> **目的**：为排错 / 新人 onboarding / 后续优化提供 file:line 级别的执行链路索引
> **关联文档**：
> - [../消息队列全流程流转说明书.md](../消息队列全流程流转说明书.md) — **队列权威说明书**（本文档与之对齐）
> - [mml-console-architecture-overview-20260521.md](mml-console-architecture-overview-20260521.md) — 三栏 UI + 7 个 API + 14 张表的横向概览
> - [mml-task-flow-design-20260424.md](mml-task-flow-design-20260424.md) — MML 任务模型的纵向设计
> - [mml-tasks-vs-device-tasks-analysis.md](mml-tasks-vs-device-tasks-analysis.md) — mml_tasks vs device_tasks 边界
> - [mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md](mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md) — R-9.2 结构化通道的需求来源

本文档与上述文档**不重复**：架构 overview 关注横向，task-flow 关注模型，本文档关注**这一个 API 请求被点击后发生的所有事情**。

### v1.2 变更说明（基于 v1.1 短板表的实际代码修复）

v1.1 完成文档对齐后，对照 §6 短板逐条 grep 验证真实性，本轮（v1.2）落地了 2 处代码修复 + 2 项移到 backlog：

| 短板 | v1.1 状态 | v1.2 处理 | 验证 |
|---|---|---|---|
| **P0 #1** RecoverPendingTasks 未挂线 | 仅文档列出 | ✅ **已修**：`acs/handler.go::handleInform` 在 InformResponse 前同步调用，按 5min 阈值恢复 sent 僵死任务 | `go build ./...` + `go test ./internal/acs/...` 通过 |
| **P2 #4** PathTranslator passthrough 无日志 | 仅文档列出 | ✅ **已修**：`mml/fanout.go::translateParamRefs` 聚合到 task 级 WARN 含 device_sn / product_class / sw_version / 前 10 条 sample_missed_paths | `go test ./internal/mml/...` 通过 |
| **P1 #3** Sequencer 缺启动期 reconciler | 仅文档列出 | 🟡 **进 backlog**：复发概率年 1-2 次；实现需新 SQL + 新 repo 接口 + mock + 测试 ~140 行；deduper.Wrap 已覆盖大部分 NATS 重投。详见 §6 backlog |
| **P3 #5** deduper 24h TTL 外的重投 | 仅文档列出 | 🟡 **进 backlog**：极低概率；ResultAggregator IncrementStats 业务幂等为主防线 |

#### v1.2 真实修订点

- §3.5 — 新增 §3.5.6 "僵死任务恢复"小节，记录 RecoverPendingTasks 已挂线
- §3.7.3 — 状态从"设计文档建议；当前 ACS handler 未挂线"改为"**已挂线 commit XXX**"
- §6 — 短板表新增"状态"列，标记 P0/P2 已修复；P1/P3 拆到独立 backlog 子表
- 附录 A — 新增 ACS handleInform RecoverPendingTasks 调用行 + fanout WARN 日志行

#### 历史变更（v1.0 → v1.1）

v1.0 在 ACS 侧出现 2 处关键错误和 3 处重要缺失，经对照 `消息队列全流程流转说明书.md` 与 `internal/acs/handler.go` 实际代码修订：

| # | 类型 | v1.0 | v1.1 |
|---|---|---|---|
| 1 | ❌ 错误 | 称 ACS 有"后台 Poller 周期 ZRANGE 取 task" | 实际是 **CPE-Inform 驱动**：`PopTask` 仅在 `HandleInform` / `HandleEmpty` / `HandleSOAPFault` 三个 SOAP handler 内同步调用（handler.go:565/772/1084） |
| 2 | ❌ 错误 | ACS 直接 → CompletionRouter → ResultAggregator | 实际经 **NATS TASK stream**：`notifyCompletion` 跨进程 Publish → APP `task-completion-bridge` QueueGroup 订阅 → deduper 防重 → CompletionRouter.Dispatch |
| 3 | ➕ 补充 | 未提 NATS 重投 / 死信 | §3.7 增加：5 次重投 + 退避（1/2/4/8/16s）+ `msg.Term()` 死信 |
| 4 | ➕ 补充 | 未提兜底机制 | §3.7 增加：RestorePendingQueues / RecoverPendingTasks / task-reboot-closer 三道兜底 |
| 5 | ➕ 补充 | 未提发布过滤 | §3.5 增加：`SourceID && CreatorID 均空时跳过` 的关键过滤 |

---

## 目录

- [0. 一句话总结](#0-一句话总结)
- [1. 高层数据流图](#1-高层数据流图)
- [2. 前端流程](#2-前端流程)
- [3. 后端流程](#3-后端流程)
  - [3.1 路由 + 鉴权](#31-路由--鉴权)
  - [3.2 Handler](#32-handler)
  - [3.3 Service 核心](#33-service-核心)
  - [3.4 Task 创建 + 派发（App）](#34-task-创建--派发app)
  - [3.5 ACS 南向下发（CPE-Inform 驱动）](#35-acs-南向下发cpe-inform-驱动)
  - [3.6 结果回流（NATS TASK stream）](#36-结果回流nats-task-stream)
  - [3.7 可靠性兜底机制](#37-可靠性兜底机制)
- [4. 完整时序图（前+后端合并）](#4-完整时序图前后端合并)
- [5. 关键设计决策与亮点](#5-关键设计决策与亮点)
- [6. 易踩坑 / 已知短板](#6-易踩坑--已知短板)
- [附录 A：文件:行号引用表](#附录-a文件行号引用表)
- [附录 B：类型契约](#附录-b类型契约)
- [附录 C：消息分层对照](#附录-c消息分层对照)

---

## 0. 一句话总结

**HTTP 201 ≠ 执行完成，而且 ACS 完全被动**。前端把结构化语句 POST 给 App，App 同步把它编译成 RPC commands、做 `standardPath → privatePath` 翻译、写入 Redis ZSET 任务队列（`acs:taskq:{sn}`）并立刻返 201；前端拿到 task_id 后切去订阅 `/events/stream` SSE。**ACS 没有后台 Poller —— TR-069 是 CPE-initiated 协议，ACS 只能在 CPE 发 SOAP 进来（Inform / 空 POST / Fault response）时同步 PopTask 把下一条 RPC 塞进响应体回发**。CPE 返 SOAP 响应后 ACS 写回 device_tasks 并通过 `notifyCompletion → NATS TASK stream` 跨进程广播 `task.completed/failed`，App 侧 `task-completion-bridge` QueueGroup 订阅、经 deduper 防重后由 `CompletionRouter` 按 `task.Source` 路由到 `mml.ResultAggregator`，后者通过 `MessageHub` 把 `mml_device_frame` / `mml_task_completed` 帧推回浏览器的 `TerminalPanel`。

> **三个核心机制**：① Redis ZSET TaskQ（下行队列，CPE-Inform 触发 Pop）；② NATS JetStream TASK stream（回流广播，QueueGroup 多副本负载均衡 + 重投/死信）；③ `cwmp2task` 映射（CPE 响应回来时按 CWMP ID 反查原 task）。

---

## 1. 高层数据流图

```
[浏览器]                                   [App 进程]                              [ACS 进程]              [CPE]
  RightPanel.runExecute
  ├─ statementToStructured  ────POST──▶  console_handler.PostExecuteStatementsStructured
  ├─ buildTaskName                          ↓
  │                                       ExecuteStructured
  │                                         ├─ StructuredToStatement (path → sub_field)
  │                                         ├─ BuildStatementCommands (LST/MOD/ADD/RMV → RPC)
  │                                         ├─ standardPath → privatePath (per device)
  │                                         ├─ CreateAndFanoutTask
  │                                         │   ├─ INSERT mml_tasks + device_tasks
  │                                         │   ├─ Fanouter 展 M cmd × N device
  │                                         │   └─ Redis ZADD acs:taskq:{sn}  (TaskQ)
  ◀──201 MMLTask────────────────────         └─ audit log
  ├─ setCurrentTaskId  →  EventSource /events/stream
  ├─ appendLine "已派发"
  └─ message.success
                                                                  ▲ CPE 主动 Inform / 空 POST / Fault 响应
                                                                  │ (ACS 完全被动，无后台 Poller)
                                                                  │
                                                                  ACS HandleInform / HandleEmpty / HandleSOAPFault
                                                                    ├─ PopTask (ZRANGEBYSCORE + ZREM atomic)
                                                                    ├─ MarkTaskSent + acs:cwmp2task:{hash}=taskID
                                                                    └─ rpcDispatcher.BuildRequest → SOAP ───▶ CPE
                                                                                                              ↓
                                                                                                    ◀──SOAP response
                                                                  ACS 解析 response, 按 CWMP ID 反查 task,
                                                                  MarkTaskCompleted / MarkTaskFailed
                                                                    ↓
                                                                  notifyCompletion (service.go:687)
                                                                  if SourceID || CreatorID 非空:
                                                                    EventBus.Publish(task.completed/failed)
                                                                    ↓
                                                       ┌──── NATS JetStream "TASK" stream ────┐
                                                       │  (持久化 72h, At-Least-Once,         │
                                                       │   重投 5 次 + 退避 1/2/4/8/16s)       │
                                                       └──────────────────────────────────────┘
                                                                    ↓
                                       APP: QueueSubscribe(SubjectTaskCompleted/Failed,
                                            queue="task-completion-bridge-{completed|failed}")
                                            → deduper.Wrap("task-completion-bridge") 防 NATS 重投双发
                                            → CompletionRouter.Dispatch(task)  按 task.Source 路由：
                                                  ├─ source=mml → ResultAggregator.OnTaskCompleted
                                                  │              ├─ Sequencer 推下一步 (cmd_idx+1)
                                                  │              ├─ IncrementStats (mml_tasks)
                                                  │              └─ hub.PublishSimple("mml_device_frame")
                                                  │                  ↓
  ◀──SSE mml_device_frame──────────────────  events/handler.go for-select 推流
  ◀──SSE mml_task_completed (最终)
```

---

## 2. 前端流程

### 2.1 触发与前置校验

`ConsoleActionBar.tsx:97-105` 主按钮 `onClick={onExecute}` 冒泡到父组件 `RightPanel.handleExecute`。

`RightPanel.tsx:200-217` `handleExecute` 三层校验顺序：

| 顺序 | 校验项 | 失败行为 |
|---|---|---|
| 1 | `selectedDeviceSns.length === 0` | `message.warning(t('mml.console.execute.noDevices'))` → return |
| 2 | `statements.length === 0` | `message.warning(t('mml.console.execute.noStatements'))` → return |
| 3 | `hasOnRebootHits(statements)`（命中 `changeApplies='OnReboot'`） | `Modal.confirm` 二次确认 → onOk: runExecute |

三层都通过 → 直接调 `runExecute()`（行 216）。

### 2.2 数据转换

`runExecute` (`RightPanel.tsx:170-198`) 做两件事：

**`statementToStructured`** (`mmlConsole.ts:543`) — UI 编辑态 → wire format：

| 字段 | 来源 | 目标 |
|---|---|---|
| `paths[]` | `selectedSubFieldIds` 经 `sf.id` 反查 `tr069Path` | 后端校验 + path 翻译 |
| `values` | `values[mmlCode]` 翻译成 `values[tr069Path]` | 后端 SPV 模板查表 |
| `rmvInstanceIndices` | 优先于 `rmvInstanceIndex`（降序） | 后端展成 N 条 DeleteObject |
| `instanceSelectors` | UI 直通 | 后端按字典序 iα/iβ/iγ 替换 `.{i}.` 占位 |

**`buildTaskName`** (`RightPanel.tsx:60-84`) — `opVerb + " " + cmdName + " " + sn`：
- opVerb：LST→`查询` / MOD→`修改` / ADD→`添加` / RMV→`删除`
- cmdName：`logicalNameI18n[lang] ?? logicalNameI18n['zh'] ?? logicalCode`
- **防御 dedup**（commit `21c4d027`）：`cmdName.startsWith(opVerb)` 时跳过 prepend，防止"查询 查询 设备基本信息"
- 多设备：`{sn1} 等N台`

### 2.3 HTTP 层

```
RightPanel.runExecute
  └─ executeMutation.mutateAsync(req)        useMmlConsole.ts:197 (React Query mutation)
       └─ mmlApi.executeStatementsStructured  mmlApi.ts:1015
            └─ http.post('/mml/console/execute-statements-structured', body)
                 ├─ Request 拦截器           http.ts:69-87
                 │   ├─ 附 Bearer token
                 │   └─ params 转 snake_case（注意：用 structuredStmtToBackend
                 │       显式 snake，避开 axios 拦截器对 instanceIndices 误伤）
                 │
                 └─ Response 拦截器          http.ts:108-130
                     ├─ ret=1 → 拆 envelope.data 透出 MMLTask
                     ├─ ret=0 → reject BusinessError
                     └─ 422 / 4xx / 5xx 不拦，原样 reject
```

**特殊错误码处理**：
- **401** → http.ts:160 硬跳 `/login`
- **422 unknown_paths** → 不拦截，业务可读 `error.response?.data.unknown_paths`（后端把 `[]string` 塞 envelope.data）
- **其他 4xx/5xx** → 富化 `userMessage` 字段，业务 try/catch 自取

### 2.4 成功副作用与 SSE 订阅

`RightPanel.tsx:180-193` 成功分支：

1. `setCurrentTaskId(task.id)` — 触发 `useMmlTaskStream` (`useMmlTaskStream.ts:159+`) 用新 taskId 重新挂载 `EventSource('/api/v1/events/stream?token=<jwt>')`
2. `appendLine("已派发 taskId=… 等待 N 台设备执行")` — 立刻显示一行，**避免 SSE 建联前的死寂**
3. `message.success(t('mml.console.execute.success', { taskId }))` — 已统一为 `App.useApp().message`（commit `b00f4928`）
4. `onExecuted?.(task)` 回调（外层可选订阅）

SSE 监听三类事件（`useMmlTaskStream.ts:206-256`）：

| 事件 | 触发 | UI 行为 |
|---|---|---|
| `mml_device_frame` | 每 device_task 终态一次 | `deviceFrameToLines()` 转 header + stdout/stderr 多行 → appendLine |
| `mml_task_status` | 首次 `running` / `cancelled` | 写 "任务开始执行" / "已取消" 行；状态机推进 |
| `mml_task_completed` | 所有 device 完成 | 写汇总行（成功 N / 失败 N）；状态收敛终态 |

状态机：`idle → dispatched → running → completed / failed / cancelled`

### 2.5 失败处理

`RightPanel.tsx:194-196` 统一口径：
```typescript
catch (e) {
  const msg = e instanceof Error ? e.message : String(e);
  message.error(t('mml.console.execute.failed', { message: msg }));
}
```
无差异化处理；422 unknown_paths 也只是普通 toast（**已知短板**，见 §6）。

### 2.6 关键类型契约

见 [附录 B](#附录-b类型契约)。

---

## 3. 后端流程

### 3.1 路由 + 鉴权

| 层 | 文件:行 | 作用 |
|---|---|---|
| 注册 | `router.go:462-471` | `md.mmlConsoleHandler.RegisterRoutes(permGroup("devices"))` |
| 端点 | `console_handler.go:90` | `mml.POST("/console/execute-statements-structured", h.PostExecuteStatementsStructured)` |
| Tracing | `router.go:221` | `middleware.Tracing("omcgo-app")` 注入 trace_id 进 ctx |
| 限流 | `router.go:227-235` | 全局 per-IP 100/s + 200 burst |
| 认证 | `middleware.go:199-245` | `RequireAuthWithAPIKey` 解 JWT 或 X-API-Key，注入 user_id / username / isSuperAdmin / claims |
| 授权 | `middleware.go (RequireAPIPermission)` | Casbin 按 `path + method` 查策略 |

> SSE 端点 `/events/stream` 走相同认证，但允许 `?token=` query 参数 fallback（middleware.go:99），因为 EventSource 不支持 custom header。

### 3.2 Handler

`console_handler.go:349-391` `PostExecuteStatementsStructured`：

1. bind `StructuredExecuteRequest` JSON
2. `creator` 缺时从 ctx 取 username（行 357）；`executor` 默认 = creator（行 362）
3. 调 `h.svc.ExecuteStructured(ctx, req, h.taskCreator)`（行 365）
4. 错误映射：

| 错误 | HTTP | 响应体附加 |
|---|---|---|
| `ErrUnknownPaths` | 422 | `{ unknown_paths: [], command_id: uuid }` |
| `ErrCommandNotFound` | 404 | — |
| `ErrMixedProductClass` / `ErrProductClassUnresolved` | 422 | — |
| 其他 | 500 | — |

5. 成功 `201 Created` + `MMLTask` 对象

### 3.3 Service 核心

#### 3.3.1 结构化适配
`console_structured.go:90-137` `StructuredToStatement`：
- 加载 command + sub_fields，建 `pathToSubField` 索引（行 113）
- `paths[]` → `SelectedSubFieldIDs`；`values keys` 翻译为 `sub_field.MMLCode`
- 任一未命中 → 汇总到 `ErrUnknownPaths.Paths` 抛 422（行 134-136）

#### 3.3.2 编译
`console_executor.go:81+` `ExecuteStatements`：
- 调 `BuildStatementCommands(stmts)`（行 92）按 op 类型展开 RPC：

| op | RPC method | 备注 |
|---|---|---|
| LST | GetParameterValues | `SelectedSubFieldIDs` 空时默认全选 |
| MOD | SetParameterValues | `values` key=MMLCode 转为 ParamCode 供 SPV 模板查表 |
| ADD | AddObject (+ optional SPV) | **复合 ADD**：单 statement = 2 commands。entry[0] AddObject；entry[1] SPV 含 `.{NEW}.` 占位待 Sequencer 替换 |
| RMV | DeleteObject | parameters["object_name"] = TargetObject + Index（单实例，2026-05-20 用户决策） |

- **`sequential` 标志**（行 124）：`sequential := len(commands) > 1`
  - **不是 sequential vs parallel 的整体选择**，而是 Fanouter 的入队策略：
    - `sequential=true`：Fanouter 只入队每个设备的 cmd_idx=0，剩下的等 Sequencer 在终态回调里入下一条
    - `sequential=false`：所有 cmd 一次性入队
  - **跨设备永远并行**：设备 A 和 B 的命令链在 Redis 是各自独立的 `acs:taskq:{sn_A}` / `acs:taskq:{sn_B}`，ACS 也是每个 CPE Inform 独立处理

#### 3.3.3 路径翻译（R-9.3）
`PathTranslator` 接口（service.go:57-62）按 `(productClass, softwareVersion)` 把 `standardPath` → `privatePath`：
- 优先 discovered_param_mappings（精确匹配 swVersion）
- 退化 param_mappings（默认）
- 未命中时 passthrough（**v1.2 修复**：`fanout.go::translateParamRefs` 现在聚合 missCount，>0 时输出 WARN 含 sample_missed_paths，可在日志中追溯"为何下发 standardPath"）
- 详细策略：参 [omcgo/CLAUDE.md §5.3](../../omcgo/CLAUDE.md)

### 3.4 Task 创建 + 派发（App）

`service.go:858-936` `CreateAndFanoutTask`：

| 步骤 | 操作 |
|---|---|
| 1 | INSERT `mml_tasks` 主表（id, device_sns[], commands JSON, status='pending', executor, source=mml, source_id=…, ...） |
| 2 | `Fanouter.Fanout` 展开 M command × N device = M·N `device_tasks` 行 |
| 3 | sequential=true：每设备只入队 cmd_idx=0；其余等 Sequencer 推 |
| 4 | sequential=false：所有 cmd 一次性入队 |
| 5 | 写 `mml_audit_logs` 每 statement 一行（合规） |
| 6 | Redis 双写：<br>· `acs:taskq:{sn}` ZADD（score = `priority × 1e13 + created_at_ns`，小者先出）<br>· `acs:task:{taskID}` HSET 详情（TTL 24h）<br>· Redis 失败时**回滚 PG**保证最终一致 |

#### Sequencer（多 cmd 串联）
`sequencer.go:1-99` 实现同设备多 cmd 严格序列：
- 订阅 `CompletionRouter` 派发的 `OnTaskCompleted`
- device_task 终态时查 `mml_task.Commands[cmd_idx+1]`，构造下一条 device_task 并 enqueue
- 跨设备并行：设备 A 与 B 的命令链各自独立推进；同设备内部严格序列
- **AddObject response 的 `instance_number`** 被 Sequencer 提取，替换下一条命令的 `.{NEW}.` 占位（行 100+）

### 3.5 ACS 南向下发（CPE-Inform 驱动）

> **关键认知**：TR-069 是 CPE-initiated 协议。ACS 没有后台 Poller / Ticker / cron 去拉队列。**`PopTask` 完全是被动调用** —— 必须等 CPE 主动 SOAP HTTP POST 进来时（开了 session）才能在响应体里塞下一条 RPC。

#### 3.5.1 三个 PopTask 调用点（handler.go）

| 行 | 入口 | 触发条件 |
|---|---|---|
| **565** | `HandleEmpty` | CPE 在已有会话中发空 POST（想结束会话），ACS 看队列还有任务就再发一条 |
| **772** | `HandleInform` | CPE 上报 Inform 进入会话，ACS 拉队列下发首条 RPC |
| **1084** | `HandleSOAPFault` response 处理 | CPE 上一条 RPC 响应 fault 后，ACS 尝试 pop 下一条继续 |

三处 PopTask 代码骨架一致：
```go
taskItem, err := h.taskService.PopTask(r.Context(), deviceSN)  // ZRANGEBYSCORE + ZREM atomic
if taskItem != nil {
    cwmpID := task.GenerateCWMPID(taskItem.Method)
    h.taskService.MarkTaskSent(r.Context(), taskItem.ID, cwmpID)  // 写 acs:cwmp2task:{cwmpID} = taskID (TTL 24h)
    session.State = StateRPCPending
    session.LastTaskID = taskItem.ID
    session.LastTaskCWMPID = cwmpID
    cmd := &rpc.Command{ID: taskItem.ID, Method: taskItem.Method, Params: taskItem.Params, ...}
    respData, _ := h.rpcDispatcher.BuildRequest(cmd, cwmpID)  // 渲染 SOAP 模板
    h.sendSOAPResponse(w, respData, log)  // 直接写回当前 HTTP response
}
```

#### 3.5.2 会话状态机
`session.go:58-92`：IDLE → INFORM_RECEIVED → PROCESSING → RPC_PENDING → RPC_RESPONSE → COMPLETE → IDLE

每次 PopTask 成功后 → `session.State = StateRPCPending`，等 CPE 下一次 POST（带 RPC response）。

#### 3.5.3 SOAP 模板渲染
`rpc/dispatcher.go:33-80`：按 `device_task.Method`（GetParameterValues / SetParameterValues / AddObject / DeleteObject）路由到对应 handler，用 `text/template` 预编译模板渲染（避免反射开销）。

#### 3.5.4 Connection Request（加速主动推送）
TR-069 协议规定 ACS 可向 CPE 发 HTTP GET 触发 CPE 立刻 Inform，避免等 PERIODIC（默认数十秒到数分钟）：
- `task.TaskService.SetConnectionRequester` 注入 CR sender
- CreateTask 后异步 `wakeDevice(sn)`：从 `acs:connreq:pending:{sn}` 30s 去重 → DNS lookup device IP → HTTP GET CR
- **CR 不是另一条派发路径**，只是把"等 CPE 自然 Inform"加速；不支持 CR 的设备仍要等 PERIODIC

#### 3.5.5 device_task 状态演进
`pending → sent → completed / failed / expired`
- sent：ACS Pop 后 MarkTaskSent
- completed/failed：CPE 返响应后 ACS MarkTaskCompleted/Failed
- expired：Pop 时发现 `ExpiresAt < now` → 跳过并 MarkTaskFailed(errorCode=expired)

#### 3.5.6 僵死任务恢复（v1.2 新增 / 已挂线）

`handler.go::handleInform` 在 InformResponse 之前同步调用：
```go
if err := h.taskService.RecoverPendingTasks(r.Context(), deviceSN); err != nil {
    log.Warn("recover pending tasks failed (non-blocking)", ..., zap.Error(err))
}
```
- 调用方：每次 CPE Inform（PERIODIC / BOOTSTRAP / CR-触发）
- 实现：`task/service.go:458` 扫 `device_tasks WHERE device_sn=$1 AND status='sent' AND sent_at < now()-5min`，按 `CanRetry()` 重置 `pending`（重入队）或标记 `failed`
- 性能：单设备 indexed query (device_sn + status + sent_at)，亚毫秒级
- 失败兜底：only warn log，不阻塞 InformResponse（业务可继续）
- **覆盖语义**：所有 sent 状态僵死任务（CPE 网络波动 / RPC 丢包 / 设备重启）在 CPE 重连即被恢复
- 与 `RestorePendingQueues`（Worker 启动幂等补灌 pending 状态）形成完整 sent + pending 双覆盖

> **v1.1 → v1.2 修订**：此前 §3.7.3 标注"设计文档建议；当前未挂线"是 v1.1 验证发现的 P0 短板。本轮已实际接入，详见 commit 列表。

### 3.6 结果回流（NATS TASK stream）

> v1.0 这一段画错了：以为 ACS 直接调用 APP 进程的 CompletionRouter。实际是**跨进程经 NATS JetStream TASK stream**。

#### 3.6.1 ACS 侧：MarkTask*** → notifyCompletion → Publish

```
ACS handler 收到 CPE 响应:
  解析 SOAP response → 按 cwmp2task:{cwmpID} 反查 taskID
  ↓
TaskService.MarkTaskCompleted(taskID, result)  service.go:307
  或 MarkTaskFailed(taskID, code, msg)         service.go:340
  ↓
内部均调 s.notifyCompletion(ctx, task)  service.go:687
  ↓
过滤：if task.SourceID == "" && task.CreatorID == "" → return  (service.go:692)
  ↓ 关键过滤：匿名/临时任务不广播，避免占用 NATS 带宽
  ↓ MML 任务的 source_id = mml_tasks.id ⇒ 必然广播
  ↓
subject := SubjectForStatus(task.Status)  // task.completed / task.failed
EventBus.Publish(ctx, subject, evt)
```

> v1.0 错误：没说这条 Publish 是跨进程的。`EventBus` 在生产是 NATSEventBus → 写入 NATS JetStream `TASK` stream 持久化 72h。

#### 3.6.2 APP 侧：task-completion-bridge QueueGroup 订阅

`event_bridge.go:38-62` `CompletionEventBridge.Subscribe`：
```go
subjects := map[string]string{
    event.SubjectTaskCompleted: "task-completion-bridge-completed",
    event.SubjectTaskFailed:    "task-completion-bridge-failed",
}
for subject, queue := range subjects {
    handler := b.handle  // 内部调 router.Dispatch(ctx, task)
    if b.deduper != nil {
        handler = b.deduper.Wrap("task-completion-bridge", handler)  // 防 NATS 重投双发
    }
    bus.QueueSubscribe(subject, queue, handler)
}
```

**关键细节**：
- `QueueSubscribe` 保证多 APP 副本部署时每条事件只被一个副本消费（NATS QueueGroup 语义）
- `deduper.Wrap` 在 NATS InterestPolicy 下重投时去重，避免 ResultAggregator 双重计数（mml_tasks.success_count++ 两次）
- `task.completed` 和 `task.failed` 用**不同的 QueueGroup name**，独立游标互不干扰

#### 3.6.3 CompletionRouter 按 source 路由
`completion_router.go:70` `Dispatch(ctx, t)`：
- 按 `t.Source`（"mml" / "provision" / "backup" / ...）查注册表
- 当前只注册了 `mml.ResultAggregator`（其他 source 走 UnknownHandler.warn）
- 未来扩展只需 `router.Register(source, handler)`

#### 3.6.4 ResultAggregator 聚合 + SSE
`result_aggregator.go:40-114` `OnTaskCompleted`：
1. UPDATE `mml_tasks.success_count++` 或 `failed_count++`（IncrementStats）
2. `hub.PublishSimple(executor, "mml_device_frame", payload)` — 每 device_task 终态一帧
3. `finalizeIfComplete()` 行 116-172：done==total 时把 `mml_tasks.status` 置 completed/failed/partial → `hub.PublishSimple("mml_task_completed", payload)`

#### 3.6.5 SSE 推流
`events/handler.go:40-114` `GET /events/stream`：
- 认证后 `hub.Subscribe(username)` 得到 per-user channel
- for-select 写 SSE：`event: <name>\ndata: <json>\n\n` + 30s keepalive
- Last-Event-ID replay：客户端断线重连指定 ID，hub 从缓冲区回放
- 浏览器 `EventSource` 监听器 → `appendLine` → `TerminalPanel` 增量渲染

### 3.7 可靠性兜底机制

> v1.0 漏写了整段。这套机制覆盖了"ACS 崩溃 / NATS 短连断 / Worker 重启 / CPE 异常重启"等 4 类异常恢复。

#### 3.7.1 NATS 重投 + 死信
所有 `QueueSubscribe` 共享 wrapHandler（`nats_bus.go:105`）：

| 阶段 | 处理 |
|---|---|
| Unmarshal 失败 | `msg.Term()` — 视为永久错误，不重试 |
| Handler 返回 error & 投递次数 < 5 | `msg.NakWithDelay(2^(n-1) s)` — 1s / 2s / 4s / 8s / 16s 递增退避 |
| 投递次数 >= 5 | `msg.Term()` — 视为死信终止，记 metrics |
| Handler 成功 | `msg.Ack()` |

业务关键链路（如告警抬升）通过**多订阅者冗余**缓解死信导致的丢失。`task-completion-bridge` 上游有 `deduper.Wrap` 防重投双发。

#### 3.7.2 Worker 启动幂等补灌
`task/service.go:526` `RestorePendingQueues(ctx, limit)`：
- 调用方：`cmd/worker/main.go:79` 在 Worker 启动后立刻调用，limit=0 表示无限制
- 扫 PG `device_tasks WHERE status='pending'`
- 对每条做 `ZSCORE acs:taskq:{sn} task.id` 判存 — **不在才 Push**
- 多 Worker / 多次重启幂等，不会重复入队
- 用于：Redis 重启丢数据后从 PG 重新灌回；新 Worker 副本启动接手老积压

#### 3.7.3 RecoverPendingTasks（v1.2 已挂线）
`task/service.go:458` `RecoverPendingTasks(ctx, deviceSN)`：
- 设计意图：CPE 每次 Inform 时由 ACS 调用，扫该设备所有 `status=sent` 且 `sent_at > 5min` 的僵死任务，按 `CanRetry()` 重置 pending / failed
- **v1.2 状态**：✅ 已在 `internal/acs/handler.go::handleInform` 同步调用（InformResponse 之前），详见 §3.5.6
- 接口暴露：`internal/acs/task_service.go` 的 `TaskService` 接口已添加 `RecoverPendingTasks` 方法（含 doc 注释引用队列说明书 §4.3.2）
- 历史：v1.0 / v1.1 漏检（设计文档建议但代码未挂线），v1.2 经 grep 验证修复

#### 3.7.4 task-reboot-closer（Worker 兜底闭环）
`task/reboot_closer.go`（队列文档 §6.4）：
- Worker 订阅 `device.inform.reboot_complete`，QueueGroup=`task-reboot-closer`
- Inform 含 `M Reboot` 时扫 PG `method IN (Reboot, FactoryReset) AND status IN (pending, sent)` → 主动 `MarkTaskCompleted`
- 解决"CPE 重启后老 Reboot 任务永不应答"的悬挂问题
- 与 `device-mgr-reboot` QueueGroup 共享同一 Subject 但独立消费

#### 3.7.5 双写一致性
`CreateTask` 流程：PG INSERT 成功 → Redis ZADD/HSET 成功 → return；Redis 失败时**显式回滚 PG**（DELETE 刚 INSERT 的行）保证 PG = Redis 最终一致。Redis pipeline 内 ZADD + HSET 也是 atomic。

#### 3.7.6 优雅降级
| 条件 | 表现 |
|---|---|
| `cfg.NATS.URL == ""` | `NewEventBus` 返回 `ChannelEventBus` 进程内通配实现，订阅/发布 API 不变，牺牲跨进程能力 |
| `EventBus == nil`（单测） | `notifyCompletion` 退回同进程 callback；Casbin Watcher 退化 no-op |
| Redis 短暂失败 | `redisx.Retry` 对非关键读操作自动重试 3 次后放弃；写操作回滚 PG |

---

## 4. 完整时序图（前+后端合并）

```
时刻   过程                                              进程            关键代码
─────────────────────────────────────────────────────────────────────────────
T0    用户点「执行 LST」                                Browser         ConsoleActionBar.tsx:100
T1    handleExecute 三层校验                            Browser         RightPanel.tsx:200-215
T2    runExecute                                        Browser         RightPanel.tsx:170
       ├ statementToStructured                          Browser         mmlConsole.ts:543
       └ buildTaskName                                  Browser         RightPanel.tsx:60
T3    POST /…/execute-statements-structured             Browser→App     mmlApi.ts:1015
       (request 拦截器附 Bearer + snake_case)           Browser         http.ts:69
T4    JWT 鉴权 + Casbin 权限                            App             middleware.go:199
T5    bind StructuredExecuteRequest                     App             console_handler.go:349
T6    StructuredToStatement (path→SubField)             App             console_structured.go:90
T7    BuildStatementCommands (RPC 展开)                 App             console_executor.go:150
T8    PathTranslator standardPath→privatePath           App             parammodel translator
T9    INSERT mml_tasks + device_tasks                   App             service.go:890
T10   Fanouter 扇出（sequential? cmd_idx=0 only）       App             fanout.go
T11   Redis ZADD acs:taskq:{sn} + HSET acs:task:{id}    App             task/queue
      (双写：失败回滚 PG)
T12   (可选) wakeDevice → HTTP GET Connection Request   App→CPE         task/service.go:92
       让 CPE 立刻 Inform（30s 去重 acs:connreq:pending）
T13   响应 201 + MMLTask                                App→Browser     console_handler.go:390
T14   信封剥壳 ret=1 → data                             Browser         http.ts:108
T15   setCurrentTaskId + appendLine "已派发"            Browser         RightPanel.tsx:183-191
       message.success                                  Browser         RightPanel.tsx:192
T16   EventSource(/events/stream?token=)                Browser→App     useMmlTaskStream.ts:193
       hub.Subscribe(username)                          App             events/handler.go:46
══════════════════════════════════════════════════════════════════════ 异步分界（等 CPE）
T17   CPE 发 SOAP Inform (PERIODIC / CR 触发)           CPE→ACS         net/http POST
T18   ACS HandleInform 进入 SOAP handler                ACS             handler.go:772
T19   PopTask: ZRANGEBYSCORE + ZREM atomic              ACS             task/queue
T20   MarkTaskSent + 写 acs:cwmp2task:{cwmpID}=taskID   ACS             handler.go:573
       Session.State = RPC_PENDING                                      session.go
T21   rpcDispatcher.BuildRequest → SOAP 模板渲染        ACS             rpc/dispatcher.go:33
T22   sendSOAPResponse 把 SOAP 写入当前 HTTP response   ACS→CPE         handler.go:618
T23   CPE 解析 SOAP → 处理 RPC → 返响应 (新 POST)       CPE→ACS
T24   ACS HandleResponse 解 SOAP → 按 cwmp_id 反查 task ACS             handler.go
T25   MarkTaskCompleted(taskID, result)                 ACS             service.go:307
       UPDATE device_tasks SET status='completed', ...                  pg writer
T26   notifyCompletion: if SourceID||CreatorID 非空 →   ACS             service.go:687
       EventBus.Publish(SubjectTaskCompleted, evt)
─────────────────────────────────────── 跨进程 NATS JetStream "TASK" stream（持久化 72h）─────
T27   APP QueueSubscribe("task-completion-bridge-       App             event_bridge.go:49
       completed").handler.invoke
T28   deduper.Wrap 防 NATS 重投双发                     App             event_bridge.go:54
T29   CompletionRouter.Dispatch(task)                   App             completion_router.go:70
       按 task.Source==mml 路由
T30   ResultAggregator.OnTaskCompleted                  App             result_aggregator.go:40
       ├ Sequencer 推 cmd_idx+1（若 sequential 且未完）App             sequencer.go:100
       │  └ AddObject result.instance_number → 替换下一条 .{NEW}.
       ├ UPDATE mml_tasks.success_count++              App
       └ hub.PublishSimple(executor, "mml_device_frame", data)
T31   events/handler.go for-select 写 SSE 帧            App→Browser     events/handler.go:90
T32   useMmlTaskStream listener → appendLine            Browser         useMmlTaskStream.ts:206
       TerminalPanel 增量渲染
T33   重复 T17~T32 直到所有 device 完成
T34   finalizeIfComplete → mml_task_completed           App             result_aggregator.go:172
       UPDATE mml_tasks.status = 'completed'/'partial'/'failed'
T35   SSE 终态帧 → 浏览器写汇总行                       App→Browser     useMmlTaskStream.ts:235
```

**异常恢复点**：
- T19 PopTask 时若发现 task.ExpiresAt < now → 跳过并 MarkTaskFailed(expired)
- T26 NATS Publish 失败 → 5 次重投退避 → 死信 + metrics
- T27 多 APP 副本：QueueGroup 保证一条事件只被一个副本消费
- Worker 启动：`RestorePendingQueues` 从 PG 把 pending 行补灌回 Redis（幂等）
- CPE 重启：`task-reboot-closer` 闭环关掉 Reboot/FactoryReset 老任务

---

## 5. 关键设计决策与亮点

| # | 决策 | 代码位置 | 设计理由 |
|---|---|---|---|
| 1 | **HTTP 201 ≠ 执行完成**，只表示已派发 | `console_handler.go:390` | 异步任务模型；前端拿到 task_id 后再订阅 SSE 拿实时结果 |
| 2 | **ACS 完全被动**，没有后台 Poller | `handler.go:565/772/1084` | TR-069 是 CPE-initiated 协议；ACS 只能在 SOAP handler 内同步 Pop |
| 3 | **Redis ZSET TaskQ + PG 双写** | `task/service.go CreateTask` | Redis 低延迟出队，PG 持久化供启动恢复；Redis 失败回滚 PG 保证一致 |
| 4 | **NATS JetStream + QueueGroup** 回流 | `event_bridge.go:49` | 跨进程 At-Least-Once + 多 APP 副本负载均衡；持久化 72h 可回放 |
| 5 | **deduper 包 QueueSubscribe** | `event_bridge.go:54` | 防 NATS 重投导致 ResultAggregator 双重计数 |
| 6 | **notifyCompletion source 过滤** | `service.go:692` | 匿名 / 临时任务（SourceID 与 CreatorID 都空）跳过广播，节省 NATS 带宽 |
| 7 | **standardPath / privatePath 双向翻译** | `parammodel.Translator` | 上层 IETF/TR-181 标准路径，下层厂商专有路径解耦；新厂商接入只配映射 |
| 8 | **`sequential` 仅控同设备序列，跨设备永远并发** | `console_executor.go:124` + `sequencer.go:100` | 同设备多 cmd 严格序列（AddObject→SPV 等依赖）；跨设备独立 Redis 队列 |
| 9 | **Fanouter + Sequencer 解耦** | `fanout.go` + `sequencer.go` | Fanouter 仅入队 cmd_idx=0；sequencer 在终态回调里推下一步，避免硬编码协程依赖 |
| 10 | **SSE 而非 WebSocket** | `events/handler.go` | 单向推送 + 浏览器原生 EventSource + 30s keepalive + Last-Event-ID 回放 |
| 11 | **MessageHub per-user channel** | `events/hub.go` | executor=username 路由 key；多 tab 共享；切角色 Unsubscribe 立刻清流 |
| 12 | **审计每 statement 一行** | `service.go:1039` | 符合电信合规：精细到操作命令而非任务级粒度 |
| 13 | **NATS 重投 + 死信** | `nats_bus.go:105` | 5 次重投退避 + `msg.Term()` 死信；关键链路靠多订阅者冗余 |
| 14 | **三道兜底机制** | RestorePendingQueues / RecoverPendingTasks / task-reboot-closer | 覆盖 Worker 重启 / CPE 异常 / 任务僵死三类场景 |
| 15 | **Tracing 跨进程** | `logger.L(ctx)` + T-0157 | trace_id 注 zap + Tempo → Loki 一键跳，排错串起 Browser→App→NATS→ACS→CPE |
| 16 | **structuredStmtToBackend 显式 snake_case** | `mmlApi.ts:1142-1157` | 避开 axios 拦截器对 `instanceIndices` 等驼峰字段的误伤 |
| 17 | **"已派发" seed 行** | `RightPanel.tsx:184-191` | execute API 返回即刻 appendLine，不等 SSE 首帧，消除"派发后死寂"体感 |

---

## 6. 已知短板（v1.2 状态跟踪）

### 6.1 已修复

| 项 | 位置 | 风险 | v1.2 处理 |
|---|---|---|---|
| **P0：RecoverPendingTasks 未挂线** | `task/service.go:458` + 原 `acs/handler.go` 无调用点 | 队列文档 §4.3.2 明确要求 CPE Inform 触发僵死任务恢复，但实际 ACS handler 没调 → 所有 `status=sent` 且 sent_at>5min 的任务永久悬挂，只有 Worker 重启才被兜底 | ✅ **已修**：`acs/handler.go::handleInform` 在 InformResponse 之前同步调用 RecoverPendingTasks；`acs/task_service.go::TaskService` 接口增加该方法；handler_test.go mock 加 no-op 实现 |
| **P2：路径翻译失败时 passthrough 无日志** | `mml/fanout.go::translateParamRefs` 原仅累 metrics.missPathUnmapped | 排查"为何下发的 SOAP 用了 standardPath"无可追溯线索 | ✅ **已修**：聚合到 task 级 WARN 含 device_sn / product_class / sw_version / product_id / miss_count / total_count / sample_missed_paths（前 10 条限量防日志洪泛） |

### 6.2 进 backlog（本轮未动）

| 项 | 位置 | 风险 | 不实施原因 |
|---|---|---|---|
| **P1：Sequencer 缺启动期 reconciler** | `sequencer.go` | APP 进程在 cmd_idx=0 完成、cmd_idx=1 入队之间崩溃时，同设备 cmd_idx≥1 链丢失（user 视角：脚本跑一半永远卡住） | 复发概率年 1-2 次（agent 估算）；实现需新 SQL + 新 repo 接口 + mock + 测试 ~140 行；deduper.Wrap 已覆盖大部分 NATS 重投。建议作为独立 backlog ticket 立项实施 |
| **P3：task-completion-bridge deduper 24h 窗口外** | `event_bridge.go:54` + `event/dedup.go` | NATS 死信队列内的消息 24h 后 dedup key 过期，重新投递可能双发 ResultAggregator | 极低概率（需要消息在死信反复投递 24h+）；ResultAggregator IncrementStats 业务幂等设计是主防线；建议增强 dedup TTL 或加 PG 唯一约束作为后续优化 |

### 6.3 其他已知短板（UX / 边界）

| 项 | 位置 | 风险 / 建议 |
|---|---|---|
| **422 unknown_paths 没专门 UI 处理** | `RightPanel.tsx:194` | 用户只看到通用 toast；建议加 modal 列出失败 path + 引导排查 |
| **utils/toast.ts 仍用静态 message** | `utils/toast.ts:1` | 调用方在嵌套 portal/Tabs 内可能仍丢 toast（30+ 文件未迁），已知技术债 |
| **buildTaskName 默认 zh-CN** | `RightPanel.tsx:173` | 英文用户也得"查询 …"中文任务名；改 i18n locale 注入即可 |
| **Connection Request 是可选的** | `task/service.go:92` | 若设备不支持 CR 或 IP 探测失败，task 要等下一个 PERIODIC Inform 才被拉走（默认 inform_interval 数十秒~数分钟） |
| **SSE 重连机制依赖 hub buffer** | `useMmlTaskStream.ts` + `events/store.go` | EventSource 自带自动重连，但 Last-Event-ID 回放依赖 hub 的 buffer 大小，长时间断网 + 大流量任务可能漏帧 |
| **历史 task_records "查询 查询..." 旧数据** | DB | commit `21c4d027` 修源代码，DB 历史不动；属历史档案，可不清 |

---

## 附录 A：文件:行号引用表

### 前端

| 功能 | 文件 | 行号 |
|---|---|---|
| 主按钮 | `omcmb/webcode/src/pages/mml/Console/components/ConsoleActionBar.tsx` | 97-105 |
| handleExecute 三层校验 | `omcmb/webcode/src/pages/mml/Console/components/RightPanel.tsx` | 200-217 |
| runExecute 异步链 | `omcmb/webcode/src/pages/mml/Console/components/RightPanel.tsx` | 170-198 |
| buildTaskName（含 dedup） | `omcmb/webcode/src/pages/mml/Console/components/RightPanel.tsx` | 60-84 |
| statementToStructured | `omcmb/frontend-core/src/types/mmlConsole.ts` | 543-588 |
| useExecuteStatementsStructured | `omcmb/frontend-core/src/hooks/api/useMmlConsole.ts` | 197-203 |
| mmlApi.executeStatementsStructured | `omcmb/frontend-core/src/services/api/mmlApi.ts` | 1015-1029 |
| structuredStmtToBackend (snake_case) | `omcmb/frontend-core/src/services/api/mmlApi.ts` | 1142-1157 |
| http.ts 请求拦截器 | `omcmb/frontend-core/src/services/http.ts` | 69-87 |
| http.ts 响应拦截器（envelope） | `omcmb/frontend-core/src/services/http.ts` | 108-130 |
| http.ts 401 → /login | `omcmb/frontend-core/src/services/http.ts` | 160 |
| useMmlTaskStream（SSE） | `omcmb/frontend-core/src/hooks/api/useMmlTaskStream.ts` | 159-272 |
| mml_device_frame listener | `omcmb/frontend-core/src/hooks/api/useMmlTaskStream.ts` | 206-209 |
| mml_task_status listener | `omcmb/frontend-core/src/hooks/api/useMmlTaskStream.ts` | 211-233 |
| mml_task_completed listener | `omcmb/frontend-core/src/hooks/api/useMmlTaskStream.ts` | 235-257 |
| CommandTree.loadCommand | `omcmb/webcode/src/pages/mml/Console/components/CommandTree.tsx` | 588-630 |

### 后端 — 请求入站 + 编译

| 功能 | 文件 | 行号 |
|---|---|---|
| 路由挂载 | `omcgo/cmd/app/provider/router.go` | 462-471 |
| RegisterRoutes | `omcgo/internal/mml/console_handler.go` | 78-95 |
| Tracing middleware | `omcgo/cmd/app/provider/router.go` | 221 |
| 限流 middleware | `omcgo/cmd/app/provider/router.go` | 227-235 |
| 认证 middleware | `omcgo/internal/admin/middleware.go` | 199-245 |
| PostExecuteStatementsStructured | `omcgo/internal/mml/console_handler.go` | 349-391 |
| ExecuteStructured 入口 | `omcgo/internal/mml/console_structured.go` | 203+ |
| StructuredToStatement | `omcgo/internal/mml/console_structured.go` | 90-137 |
| ErrUnknownPaths | `omcgo/internal/mml/console_structured.go` | 61-75 |
| ExecuteStatements | `omcgo/internal/mml/console_executor.go` | 81-137 |
| BuildStatementCommands | `omcgo/internal/mml/console_executor.go` | 150+ |
| sequential 判定 | `omcgo/internal/mml/console_executor.go` | 124 |

### 后端 — 持久化 + Redis 派发

| 功能 | 文件 | 行号 |
|---|---|---|
| CreateAndFanoutTask | `omcgo/internal/mml/service.go` | 858-936 |
| writeAuditLogs | `omcgo/internal/mml/service.go` | 1039-1070 |
| Fanouter | `omcgo/internal/mml/fanout.go` | — |
| Sequencer | `omcgo/internal/mml/sequencer.go` | 1-99 + 100+ |
| TaskService.CreateTask | `omcgo/internal/task/service.go` | 127-176 |
| TaskService.notifyCompletion | `omcgo/internal/task/service.go` | 687-720 |
| 发布过滤 (SourceID && CreatorID) | `omcgo/internal/task/service.go` | 692 |

### 后端 — ACS（CPE-Inform 驱动）

| 功能 | 文件 | 行号 |
|---|---|---|
| PopTask in HandleEmpty | `omcgo/internal/acs/handler.go` | 565 |
| PopTask in HandleInform | `omcgo/internal/acs/handler.go` | 772 |
| PopTask in HandleSOAPFault | `omcgo/internal/acs/handler.go` | 1084 |
| ConnectionRequester 接口 | `omcgo/internal/acs/handler.go` | 39-45 |
| ConnectionRequestURL 缓存 | `omcgo/internal/acs/handler.go` | 89, 421-425 |
| ACS RPC dispatcher | `omcgo/internal/acs/rpc/dispatcher.go` | 33-80 |
| ACS session 状态机 | `omcgo/internal/acs/session.go` | 58-92 |
| MarkTaskCompleted | `omcgo/internal/task/service.go` | 305-310 |
| MarkTaskFailed | `omcgo/internal/task/service.go` | 338-342 |

### 后端 — 回流 NATS / Bridge / Aggregator

| 功能 | 文件 | 行号 |
|---|---|---|
| CompletionEventBridge | `omcgo/internal/task/event_bridge.go` | 22-65 |
| QueueGroup 配置 | `omcgo/internal/task/event_bridge.go` | 49-50 |
| deduper.Wrap | `omcgo/internal/task/event_bridge.go` | 53-54 |
| CompletionRouter.Dispatch | `omcgo/internal/task/completion_router.go` | 70-99 |
| CompletionRouter.Register | `omcgo/internal/task/completion_router.go` | 50-60 |
| ResultAggregator.OnTaskCompleted | `omcgo/internal/mml/result_aggregator.go` | 40-114 |
| finalizeIfComplete | `omcgo/internal/mml/result_aggregator.go` | 116-172 |
| MessageHub.PublishSimple | `omcgo/internal/events/hub.go` | — |
| SSE handler | `omcgo/internal/events/handler.go` | 40-114 |

### 后端 — 兜底机制

| 功能 | 文件 | 行号 |
|---|---|---|
| RestorePendingQueues（Worker 启动） | `omcgo/internal/task/service.go` | 526+ |
| RestorePendingQueues 调用方 | `omcgo/cmd/worker/main.go` | 79 |
| RecoverPendingTasks（未挂线） | `omcgo/internal/task/service.go` | 458-470 |
| GetStaleSentTasks (5min 阈值) | `omcgo/internal/task/service.go` | 460 |
| task-reboot-closer | `omcgo/internal/task/reboot_closer.go` | — |
| NATS 重投 wrapHandler | `omcgo/internal/core/event/nats_bus.go` | 105+ |

### 数据库表

| 表 | 用途 |
|---|---|
| `mml_tasks` | 任务主表（id, device_sns[], commands JSON, status, executor, source, source_id, ...） |
| `device_tasks` | 每设备每命令一行（source_id→mml_task.id, device_sn, command_index, method, result, status, sent_at, ...） |
| `mml_audit_logs` | 每 statement 一行（user_id, statement, op, values, status） |
| `mml_commands` | 命令字典（含 logical_name_i18n） |
| `mml_sub_fields` | 子字段字典（含 tr069_path, value_type, access_type, change_applies） |

### Redis key

| Key 模板 | 类型 | TTL | 用途 |
|---|---|---|---|
| `acs:taskq:{sn}` | Sorted Set | 无 | per-device 任务队列，member=taskID，score = `priority × 1e13 + created_at_ns` |
| `acs:task:{taskID}` | Hash | 24h | device_task 完整内容 |
| `acs:cwmp2task:{cwmpID_hash}` | String | 24h | CWMP ID → Task ID 反查（CPE 响应回来时定位 task） |
| `acs:session:{sn}` | Hash | 5min | TR069 会话状态 |
| `acs:connreq:pending:{sn}` | String | 30s | Connection Request 去重 |
| `acs:heartbeat:{sn}` | String（时间戳） | 2×inform | 心跳追踪 |

### NATS Subjects

| Subject | Stream | QueueGroup | 用途 |
|---|---|---|---|
| `task.completed` | TASK | `task-completion-bridge-completed` | 任务成功终态（App 订阅） |
| `task.failed` | TASK | `task-completion-bridge-failed` | 任务失败终态（App 订阅） |

> 完整 Subject 清单见 [消息队列全流程流转说明书 §10.A.2](../消息队列全流程流转说明书.md)

---

## 附录 B：类型契约

### 前端 wire types

```typescript
// omcmb/frontend-core/src/types/mmlConsole.ts
export interface StructuredStatement {
  commandId: string;
  operationType: 'LST' | 'MOD' | 'ADD' | 'RMV';
  commandCode?: string;
  paths: string[];                              // standardPath 列表
  values?: Record<string, string>;              // key=standardPath, value=新值
  instanceSelectors?: Record<string, string>;   // iα/iβ/iγ → 实例号
  instanceIndices?: number[];                   // RMV 多选实例
}

export interface StructuredExecuteRequest {
  deviceSns: string[];
  statements: StructuredStatement[];
  taskName?: string;
  creator?: string;
  executor?: string;
  executeType?: 'immediate' | 'scheduled' | 'periodic' | 'suspended';
}
```

### 后端响应

```typescript
// omcmb/frontend-core/src/services/api/mmlApi.ts:336-394
export interface MMLTask {
  id: string;
  taskName: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  deviceSns: string[];
  commands: Array<Record<string, unknown>>;
  results: DeviceTaskResultItem[];
  totalDevices: number;
  successCount: number;
  failedCount: number;
  creator: string;
  executor: string;
  executeType: string;
  source: string;       // = "mml"
  sourceId: string;     // = mml_tasks.id (用于 notifyCompletion 过滤)
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
  // ...
}
```

### SSE 事件 payload（关键字段）

```json
// mml_device_frame
{
  "task_id": "uuid",
  "device_sn": "string",
  "command_index": 0,
  "status": "completed | failed | expired",
  "result": { ... },
  "error_message": "string"
}

// mml_task_completed
{
  "task_id": "uuid",
  "status": "completed | failed | cancelled",
  "result": "all_success | partial | all_failed",
  "success_count": 1,
  "failed_count": 0
}
```

### NATS TASK stream payload (Go `task.Task`)

```go
type Task struct {
    ID           string
    DeviceSN     string
    Method       string         // "GetParameterValues" / "SetParameterValues" / ...
    Params       map[string]any
    CommandKey   string
    Status       TaskStatus     // pending / sent / completed / failed / expired
    Result       json.RawMessage
    ErrorCode    int
    ErrorMessage string
    Source       string         // "mml" / "provision" / "backup" / ...
    SourceID     string         // = mml_tasks.id（按业务关联到上游任务）
    CreatorID    string         // user_id（用户操作触发的任务）
    CommandIndex int            // 多 cmd 任务的第几条
    Priority     int            // ZSET score 的高位
    CreatedAt    time.Time
    SentAt       *time.Time     // RecoverPendingTasks 5min 阈值判定依据
    ExpiresAt    *time.Time     // Pop 时检查；过期直接 fail(expired)
}
```

---

## 附录 C：消息分层对照

参考 `消息队列全流程流转说明书.md` §2.3：

| 层 | 机制 | 库 | 跨进程 | 持久化 | 用在本流程的哪步 |
|---|---|---|:---:|:---:|---|
| L1 | NATS JetStream | `nats-io/nats.go` | ✅ | ✅ 文件 | T26-T29：ACS publish task.completed → App task-completion-bridge 订阅 |
| L1 降级 | Channel EventBus | 纯 Go channel | ❌ | ❌ | 单进程 / 无 NATS 部署兜底（dev/test） |
| L2 | Redis TaskQ | `redis/go-redis` | ✅ | ✅ Redis + PG 双写 | T11：CreateTask 入队；T19：ACS PopTask 出队 |
| L3 | Redis 普通 Hash | `redis/go-redis` | ✅ | ✅ | `acs:task:{id}` task 详情 / `acs:cwmp2task:{hash}` 响应反查 |
| L4 | SSE Hub | Go channel | ❌（单进程）| 可选 | T31-T35：mml_device_frame / mml_task_completed → 浏览器 |

> ❌ 已下线：原 `acs:cmdq:*` 命令队列（2026-04-23 合入 TaskQ）；原 Redis Pub/Sub `casbin:policy:reload`（2026-04-22 迁 NATS）。当前 Redis **不再承载任何广播语义**。

---

## 变更历史

| 日期 | 版本 | 变更 |
|---|---|---|
| 2026-05-23 | v1.0 | 初稿；记录至 commit `21c4d027` + `b00f4928` 后的代码状态 |
| 2026-05-23 | v1.1 | 修订：① ACS 改为 CPE-Inform 驱动（删除 Poller 误述）；② 回流路径补全 NATS TASK stream + task-completion-bridge QueueGroup + deduper + CompletionRouter 四层；③ 新增 §3.7 可靠性兜底机制；④ 新增 SourceID 过滤说明；⑤ 时序图 T17-T35 重画；⑥ 附录 A 补 12+ 处新引用 |
| 2026-05-23 | v1.2 | 落地 v1.1 列出的 2 处短板代码修复：① P0 — `acs/handler.go::handleInform` 接入 RecoverPendingTasks 同步调用（5min 阈值恢复 sent 僵死任务）；② P2 — `mml/fanout.go::translateParamRefs` 加 WARN 日志含 sample_missed_paths（路径翻译可观测性）。P1 Sequencer reconciler + P3 deduper 24h 窗口 进 backlog（§6.2）。§6 重排为「已修复 / backlog / 其他短板」三表 |
