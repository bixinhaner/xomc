# `POST /api/v1/mml/console/execute-statements-structured` 全链路分析

> **日期**：2026-05-23
> **范围**：单一 API 端到端 — Browser SPA → App → Redis 队列 → ACS → CPE → CompletionRouter → SSE Hub → Browser 终端
> **目的**：为排错 / 新人 onboarding / 后续优化提供 file:line 级别的执行链路索引
> **关联文档**：
> - [mml-console-architecture-overview-20260521.md](mml-console-architecture-overview-20260521.md) — 三栏 UI + 7 个 API + 14 张表的横向概览
> - [mml-task-flow-design-20260424.md](mml-task-flow-design-20260424.md) — MML 任务模型的纵向设计（添加 → 执行 → 回流）
> - [mml-tasks-vs-device-tasks-analysis.md](mml-tasks-vs-device-tasks-analysis.md) — mml_tasks vs device_tasks 边界
> - [mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md](mml-console-cmcc-tdlte-v23-adjustment-plan-20260519.md) — R-9.2 结构化通道的需求来源

本文档与上述文档**不重复**：架构 overview 关注横向，task-flow 关注模型，本文档关注**这一个 API 请求被点击后发生的所有事情**。

---

## 目录

- [0. 一句话总结](#0-一句话总结)
- [1. 高层数据流图](#1-高层数据流图)
- [2. 前端流程](#2-前端流程)
- [3. 后端流程](#3-后端流程)
- [4. 完整时序图（前+后端合并）](#4-完整时序图前后端合并)
- [5. 关键设计决策与亮点](#5-关键设计决策与亮点)
- [6. 易踩坑 / 已知短板](#6-易踩坑--已知短板)
- [附录 A：文件:行号引用表](#附录-a文件行号引用表)
- [附录 B：类型契约](#附录-b类型契约)

---

## 0. 一句话总结

**HTTP 201 ≠ 执行完成**。前端把结构化语句 POST 给 App，App 同步把它编译成 RPC commands、做 `standardPath → privatePath` 翻译、fanout 到 Redis 队列，立刻返 201 给前端；前端拿到 task_id 后切去订阅 `/events/stream` SSE。ACS 异步消费队列，渲染 SOAP，与 CPE 完成交互后写回 `device_tasks` 并通过事件总线触发 `ResultAggregator`，后者通过 `MessageHub` 把 `mml_device_frame` / `mml_task_completed` 帧推回浏览器的 `TerminalPanel`。**整链路通过 `trace_id` 串联可观测，通过 `task_id` / `device_sn` / `cmd_idx` 三元组寻址任意环节状态。**

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
  │                                         │   └─ Redis ZADD acs:taskq:{sn}
  ◀──201 MMLTask────────────────────         └─ audit log
  ├─ setCurrentTaskId  →  EventSource /events/stream
  ├─ appendLine "已派发"                                          ZRANGE ◀──────  ACS Poller
  └─ message.success                                              ↓
                                                                Dispatcher → SOAP ─────▶ CPE
                                                                                          ↓
                                                                                 ◀──SOAP response
                                                                ResultAggregator ◀── 写回 device_tasks
                                                                ├─ Sequencer 推下一步
  ◀──SSE mml_device_frame─────────────── hub.PublishSimple ←────┘
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

- 多 statement → `sequential=true`；单 statement → 并发（行 124）

#### 3.3.3 路径翻译（R-9.3）
`PathTranslator` 接口（service.go:57-62）按 `(productClass, softwareVersion)` 把 `standardPath` → `privatePath`：
- 优先 discovered_param_mappings（精确匹配 swVersion）
- 退化 param_mappings（默认）
- 未命中时 passthrough（**已知短板**：无告警，见 §6）
- 详细策略：参 [omcgo/CLAUDE.md §5.3](../../omcgo/CLAUDE.md)

#### 3.3.4 持久化 + 派发
`service.go:858-936` `CreateAndFanoutTask`：

| 步骤 | 操作 |
|---|---|
| 1 | INSERT `mml_tasks` 主表（id, device_sns[], commands JSON, status='pending', executor, ...） |
| 2 | `Fanouter.Fanout` 展开 M command × N device = M·N `device_tasks` 行 |
| 3 | sequential=true：只入队 cmd_idx=0；其余等 Sequencer 推 |
| 4 | sequential=false：全部入队，跨设备并发 |
| 5 | 写 `mml_audit_logs` 每 statement 一行 |
| 6 | Redis：`acs:taskq:{sn}` ZADD（score=priority）<br>`acs:task:{taskID}` HSET 详情<br>`acs:cwmp2task:{hash}` 映射 CWMP ID → task |

#### 3.3.5 Sequencer
`sequencer.go:1-99` 实现多 cmd 严格序列：
- 监听 `CompletionRouter` 派发的 `OnTaskCompleted` 回调
- device_task 终态时查 `mml_task.Commands[cmd_idx+1]`，构造下一条 device_task 并 enqueue
- 跨设备并行：设备 A 与 B 的命令链各自独立推进；同设备内部严格序列
- **AddObject response 的 `instance_number`** 被 Sequencer 提取，替换下一条命令的 `.{NEW}.` 占位（行 100+）

### 3.4 ACS 南向下发

ACS 进程（`omcgo/cmd/acs/main.go`）后台 Poller 周期 ZRANGE 取 task：

| 阶段 | 文件:行 | 说明 |
|---|---|---|
| 取队列 | `cmdqueue/...` | `acs:taskq:{sn}` ZRANGE 取最老 task ID |
| 会话 | `session.go:58-92` | 状态机 IDLE → INFORM_RECEIVED → PROCESSING → RPC_PENDING → RPC_RESPONSE → COMPLETE |
| 模板 | `rpc/dispatcher.go:33-80` | 按 device_task.Method 路由到 GetParameterValues / SetParameterValues / AddObject / DeleteObject |
| 下发 | `pkg/soap` | text/template 预渲染（避免反射开销）→ net/http POST SOAP |
| CWMP ID 关联 | `dispatcher.go:78` | `cmd.CWMPID = cwmpID` 用于 response 回填查找 task |

**Connection Request**（可选）：`task/service.go:92` `SetConnectionRequester`
- CreateTask 后异步 wake：DNS lookup device IP + HTTP GET CR
- 让 CPE 主动 Inform 缩短首包延迟
- 不支持 CR 的设备会等下一次 PERIODIC Inform 才被拉走（默认数十秒~数分钟）

**device_task 状态演进**：pending → dispatched → completed / failed / expired

### 3.5 结果回流 + 事件

| 阶段 | 进程 | 操作 |
|---|---|---|
| 1. SOAP response 解析 | ACS | 把 result/fault 写回 `device_tasks.result/error_message/completed_at` |
| 2. 事件发布 | ACS | `eventBus.Publish("task.completed"/"task.failed"/"task.expired")`（NATS subject） |
| 3. 路由分发 | App | `CompletionRouter` 订阅，dispatch 给两路： |
| 3a. Sequencer | App | `sequencer.go:100` 推进同设备下一个 cmd_idx |
| 3b. ResultAggregator | App | `result_aggregator.go:40-114` 聚合统计 |
| 4a. SSE per-frame | App | 每 device_task 终态 → `hub.PublishSimple(executor, "mml_device_frame", {task_id, device_sn, status, result})` |
| 4b. SSE 终态 | App | 全部 device 完成 → `finalizeIfComplete()` (`result_aggregator.go:172`) → `"mml_task_completed"` |
| 5. SSE 推流 | App | `events/handler.go:40-114` `GET /events/stream` for-select 写 SSE + 30s keepalive + Last-Event-ID replay |
| 6. 浏览器接收 | Browser | `EventSource` 事件 listener → `appendLine` → `TerminalPanel` 增量渲染 |

**MessageHub 设计**（events/hub.go）：
- 维护 per-user channel map（key = username）
- 同用户多 tab 共享 channel
- 切角色后 `hub.Unsubscribe` 立刻清流

### 3.6 横切关注

| 维度 | 实现 | 说明 |
|---|---|---|
| 链路追踪 | `Tracing` middleware → `logger.L(ctx)` 注 trace_id 进 zap（T-0157） | Grafana Tempo → Loki 一键跳 |
| 审计 | `service.go:1039-1070` `writeAuditLogs` | 每 statement 一行，符合电信合规精度要求 |
| 限流 | 全局 per-IP 100/s + 200 burst | MML 路径无额外限流，依赖 ACS 设备级 Inform 洪泛防护 |

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
T11   Redis ZADD acs:taskq:{sn}                         App             task/queue
T12   响应 201 + MMLTask                                App→Browser     console_handler.go:390
T13   信封剥壳 ret=1 → data                             Browser         http.ts:108
T14   setCurrentTaskId + appendLine "已派发"            Browser         RightPanel.tsx:183-191
       message.success                                  Browser         RightPanel.tsx:192
T15   EventSource(/events/stream?token=)                Browser→App     useMmlTaskStream.ts:193
       hub.Subscribe(username)                          App             events/handler.go:46
══════════════════════════════════════════════════════════════════════ 异步分界
T16   Poller ZRANGE / 取 device_task                    ACS             cmdqueue (background)
T17   Session IDLE→PROCESSING→RPC_PENDING               ACS             session.go:64
T18   Dispatcher → SOAP 模板渲染                        ACS             rpc/dispatcher.go:33
T19   net/http POST SOAP                                ACS→CPE
T20   CPE response                                      CPE→ACS
T21   解析 + 写回 device_tasks.result                   ACS             pg writer
T22   eventBus.Publish(task.completed)                  ACS             NATS subject
T23   CompletionRouter dispatch                         App             completion_router.go
       ├ Sequencer.OnTaskCompleted                      App             sequencer.go:100
       │  └ enqueue cmd_idx+1 / sub .{NEW}.             App
       └ ResultAggregator.OnTaskCompleted               App             result_aggregator.go:40
          └ hub.PublishSimple("mml_device_frame", …)    App             result_aggregator.go:82
T24   SSE 帧 → 浏览器                                   App→Browser     events/handler.go:90
T25   useMmlTaskStream 监听器 appendLine                Browser         useMmlTaskStream.ts:206
       TerminalPanel 增量渲染                           Browser
T26   重复 T16~T25 直到所有 device 完成
T27   finalizeIfComplete → mml_task_completed           App             result_aggregator.go:172
T28   SSE 终态帧 → 浏览器写汇总行                       App→Browser     useMmlTaskStream.ts:235
```

---

## 5. 关键设计决策与亮点

| # | 决策 | 代码位置 | 设计理由 |
|---|---|---|---|
| 1 | **HTTP 201 ≠ 执行完成**，只表示已派发 | `console_handler.go:390` | 异步任务模型；前端拿到 task_id 后再订阅 SSE 拿实时结果 |
| 2 | **standardPath / privatePath 双向翻译** | `parammodel.Translator` | 解耦上层（IETF/TR-181 标准路径）与下层（厂商专有路径），新厂商接入只配映射不改业务 |
| 3 | **Sequential vs Parallel 按 statement 数自动判定** | `console_executor.go:124` | 单 statement 跨设备并发；多 statement 严格序列保证依赖（如 ADD→SPV 必须等 NEW instance number） |
| 4 | **Fanouter + Sequencer 解耦** | `fanout.go` + `sequencer.go:100` | Fanouter 一次性扇出 M×N device_tasks 简化模型；sequencer 仅在 sequential 时按完成回调推进，避免硬编码协程依赖 |
| 5 | **SSE 而非 WebSocket** | `events/handler.go` | 单向推送 + 浏览器原生 EventSource + 30s keepalive + Last-Event-ID 回放足够，避免 WS 心跳/重连复杂度 |
| 6 | **MessageHub per-user channel** | `events/hub.go` | executor=username 作为路由 key；同用户多 tab 共享 channel；切角色后 `hub.Unsubscribe` 立刻清流 |
| 7 | **审计每 statement 一行** | `service.go:1039` | 符合电信合规：精细到操作命令而非任务级粒度 |
| 8 | **Tracing 跨进程**：trace_id 注入 zap + Tempo → Loki 跳转 | `logger.L(ctx)` + T-0157 | 排错时一条 trace 串起 Browser→App→ACS→CPE→ResultAggregator→SSE |
| 9 | **structuredStmtToBackend 显式 snake_case** | `mmlApi.ts:1142-1157` | 避开 axios 拦截器对 `instanceIndices` 等驼峰字段的误伤 |
| 10 | **"已派发" seed 行** | `RightPanel.tsx:184-191` | execute API 返回即刻 appendLine，不等 SSE 首帧，消除"派发后死寂"体感 |

---

## 6. 易踩坑 / 已知短板

| 项 | 位置 | 风险 / 建议 |
|---|---|---|
| **422 unknown_paths 没专门 UI 处理** | `RightPanel.tsx:194` | 用户只看到通用 toast；建议加 modal 列出失败 path + 引导排查 |
| **utils/toast.ts 仍用静态 message** | `utils/toast.ts:1` | 调用方在嵌套 portal/Tabs 内可能仍丢 toast（30+ 文件未迁），已知技术债 |
| **buildTaskName 默认 zh-CN，未根据用户 i18n 偏好** | `RightPanel.tsx:173` | 英文用户也得"查询 …"中文任务名；改 i18n locale 注入即可 |
| **任务名为空时后端兜底用 `MML console (X stmts × Y devices)`** | `console_executor.go:103` | 排查时不直观；前端 buildTaskName 永远非空所以理论不触发 |
| **路径翻译失败时 passthrough** | `parammodel translator` | 没有显式告警；用户排查会 confused "为啥下发的 path 跟我选的不一样" |
| **Connection Request 是可选的** | `task/service.go:92` | 若设备不支持或 IP 探测失败，task 要等下一个 PERIODIC Inform 才被拉走（默认 inform_interval 可能数十秒到数分钟） |
| **Sequencer 单点 risk** | `sequencer.go` | 若 sequencer 重启时正好有同设备的 cmd_idx=0 完成，cmd_idx=1 入队可能丢；需 reconcile 机制（未深入验证，建议补 reconciler） |
| **SSE 重连机制** | `useMmlTaskStream.ts` | EventSource 自带自动重连，但 Last-Event-ID 回放依赖 hub 的 buffer 大小，长时间断网 + 大流量任务可能漏帧 |
| **历史"查询 查询..."task_name 数据未清** | DB | commit `21c4d027` 修了源代码但 DB 历史 task_records 仍保留旧名；属历史档案，可不清 |

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

### 后端

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
| CreateAndFanoutTask | `omcgo/internal/mml/service.go` | 858-936 |
| writeAuditLogs | `omcgo/internal/mml/service.go` | 1039-1070 |
| Fanouter | `omcgo/internal/mml/fanout.go` | — |
| Sequencer | `omcgo/internal/mml/sequencer.go` | 1-99 + 100+ |
| TaskService.CreateTask + CR | `omcgo/internal/task/service.go` | 92-176 |
| ACS RPC dispatcher | `omcgo/internal/acs/rpc/dispatcher.go` | 33-80 |
| ACS session 状态机 | `omcgo/internal/acs/session.go` | 58-92 |
| ResultAggregator | `omcgo/internal/mml/result_aggregator.go` | 40-114 |
| finalizeIfComplete | `omcgo/internal/mml/result_aggregator.go` | 116-172 |
| SSE handler | `omcgo/internal/events/handler.go` | 40-114 |
| MessageHub | `omcgo/internal/events/hub.go` | — |

### 数据库表

| 表 | 用途 |
|---|---|
| `mml_tasks` | 任务主表（id, device_sns[], commands JSON, status, executor, ...） |
| `device_tasks` | 每设备每命令一行（source_id→mml_task.id, device_sn, command_index, method, result, ...） |
| `mml_audit_logs` | 每 statement 一行（user_id, statement, op, values, status） |
| `mml_commands` | 命令字典（含 logical_name_i18n） |
| `mml_sub_fields` | 子字段字典（含 tr069_path, value_type, access_type, change_applies） |

### Redis key

| Key 模板 | 类型 | TTL | 用途 |
|---|---|---|---|
| `acs:taskq:{sn}` | Sorted Set | — | per-device 任务队列，member=taskID，score=priority |
| `acs:task:{taskID}` | Hash | 24h | device_task 完整内容 |
| `acs:cwmp2task:{cwmpHash}` | String | 24h | CWMP ID → Task ID 映射 |
| `acs:session:{sn}` | Hash | 5min | TR069 会话状态 |

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

---

## 变更历史

| 日期 | 作者 | 变更 |
|---|---|---|
| 2026-05-23 | 分析整理 | 初稿；记录至 commit `21c4d027`（任务名 dedup）+ `b00f4928`（toast App.useApp）后的代码状态 |
