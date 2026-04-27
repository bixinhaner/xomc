# 测试 OMC 的 RPC 任务执行流程

## 流程说明

1. 通过 OMC 管理后台 → MML 控制台，选择设备 SN `1202000240194DP0026`，命令"设备信息 LST DEVICE_INFO"，添加一条 MML 命令（对应 RPC `GetParameterValues`）。`mml_tasks` 表能正确写入命令数据。
2. `device_tasks` 也能记录对应任务（每设备 × 每命令一行），`source='mml'`、`status='sent'`、`sent_at` 有时间戳，但 `result` 为空。
3. `mml_tasks` 永远拿不到 RPC 的执行结果（统计/状态停留在 `running`/`pending`，不会推进到 `completed`）。

---

## 整条链路（结合现有代码）

```
[1] Frontend MML 控制台 (omcmb)
        │  POST /api/v1/mml/execute  body: { device_sns, command_code, parameters: {...} }
        ▼
[2] internal/mml/handler.go      ExecuteCommand handler
        │  归一化 parameters（nil → {}）→ Service.ExecuteCommand
        ▼
[3] internal/mml/service.go:462  Service.ExecuteCommand
        │  ① 解析 command_code → 取 cmd.RPCMethod（如 GetParameterValues）
        │  ② 写 mml_tasks（Status=running 或 pending）
        │  ③ Fanouter.Fanout(ctx, mmlTask)
        ▼
[4] internal/mml/fanout.go:35    Fanouter.Fanout
        │  按 (command × device) 笛卡尔展开，BatchCreateTasks 写入 device_tasks
        │  device_tasks.source='mml'  source_id=mml_task.id
        │  device_tasks.params = json.Marshal(commands[i].parameters)  ← ⚠️ 关键
        ▼
[5] internal/task/service.go     TaskService.BatchCreateTasks
        │  Redis ZAdd acs:taskq:{deviceSN}  +  PG INSERT device_tasks(status='pending')
        ▼
[6] CPE 周期 Inform → ACS 弹队列下发  internal/acs/handler.go:553
        │  PopTask → MarkTaskSent (status='sent', sent_at=NOW(), cwmp_id=…)
        │  Dispatcher.BuildRequest → 渲染 SOAP cwmp:GetParameterValues 写入 HTTP 响应
        │  log: "ACS sending RPC request from task"
        ▼
[7] CPE 处理后回 POST 一条 SOAP 响应  internal/acs/handler.go:622 handleRPCResponse
        │  log: "RPC response received" / "ACS received RPC response" (debug 含 xml)
        │  GetTaskByCWMPID 命中 → MarkTaskCompleted / MarkTaskFailed
        │  result = {"method":"GetParameterValues","raw_response":<xml>}
        ▼
[8] internal/task/service.go:568 TaskService.notifyCompletion
        │  Publish NATS subject task.completed / task.failed (payload = Task)
        ▼
[9] internal/task/event_bridge.go  CompletionEventBridge.handle
        │  router.Dispatch(ctx, t)
        ▼
[10] internal/task/completion_router.go  按 t.Source 分发
        │  Register(TaskSourceMML, mmlAggregator)（cmd/app/provider/modules.go:272）
        ▼
[11] internal/mml/result_aggregator.go:41 OnTaskCompleted
        │  IncrementStats → finalizeIfComplete
        │  全部子任务完成 → mml_tasks.status=completed, finished_at=NOW()
        │  SSE Publish "mml_task_completed"
```

**关键事实 1**：`device_tasks.status='sent'` + `result=NULL` 严格意味着流程卡在 **[6] 之后、[7] 之前**——ACS 已经把 SOAP 投递出去了，但 `MarkTaskCompleted/MarkTaskFailed` 这两个分支谁都没执行过。

**关键事实 2**：`device_tasks.params` 实际值是
```json
{"LTE_BTSNUM":"", "LTE_GSM_IP":"", ..., "LTE_DEVICE_HARDWARE_VERSION":""}
```
这是**前端表单字段名 → 空字符串的 map**，**不是 TR-069 协议要求的形状**。

---

## 根因初判

ACS 的 GetParameterValues handler（`internal/acs/rpc/dispatcher.go:79-94`）期望的入参 schema 是：

```go
var params struct {
    Names []string `json:"names"`   // ← 必须是 names 数组
}
json.Unmarshal(cmd.Params, &params)
// 然后把每个 name 渲染进 cwmp:GetParameterValues > ParameterNames
```

而 Fanouter（`internal/mml/fanout.go:71-77`）目前只是把 `commands[i].parameters` 原样 `json.Marshal` 塞进 `device_tasks.params`，**完全没有做 TR-069 协议适配**：
- 期望形状：`{"names":["Device.X_VENDOR.LTE.BtsNum", "Device.IP.Interface.1.IPv4Address.1.IPAddress", ...]}`
- 实际写入：`{"LTE_BTSNUM":"", "LTE_GSM_IP":"", ...}`

后果：

1. `json.Unmarshal` 不报错（结构兼容，没有 `names` 字段而已），但 `params.Names` = `nil` / 空。
2. SOAP 模板渲染出来是 **`ParameterNames` 数组为空**的 `cwmp:GetParameterValues` 请求。
3. 行业上多数 CPE 的反应：
   - 少数实现宽松：返回空 `ParameterList` → ACS 走成功分支 → `result` 应该非空（不符合现象）
   - 多数实现严格：直接返回 SOAP Fault `9003 Invalid arguments`/`9005 Invalid parameter name` → ACS 应当 `MarkTaskFailed`（也不符合现象）
   - 部分实现：判定为非法报文 **TCP 直接关连接**或 **HTTP 502 后悄悄丢弃**，**不发任何 SOAP 响应** ← 这正好对应 `status=sent` 持续不动
4. 由于 [7] 永远没跑到，[8]-[11] 的 NATS 事件链一条都不会触发，`mml_tasks` 永远收不到回流，停在中间态。

> 旁证：`device_tasks.params` 的键 `LTE_BTSNUM`、`DEVICEGSM_MCC` 等一看就是 **MML 命令树里某个 command_code 的 param_code**（见 `mml_params.param_code`），不是 `mml_params.tr069_path`。前端在控制台只把用户表单 key 发上来，整个 OMC 后端**没有一处把 param_code 映射回 tr069_path 再重组成 `{"names":[...]}`**。

---

## 5 分钟自检（按现象快速定位是不是同一个根因）

> 设 `<MML>` = mml_task.id，`<DT>` = device_task.id，`<CWMPID>` = device_tasks.cwmp_id

```bash
# 1) 看 device_tasks 实际下发参数（如果 params 是 {key: "", ...} 形态，则就是本根因）
psql -c "SELECT id, source, status, sent_at, result IS NOT NULL AS has_result, params
         FROM device_tasks WHERE source='mml' ORDER BY created_at DESC LIMIT 3;"

# 2) 看 ACS 是否真的发出去了 + 看真正发出去的 SOAP 报文
#    新增字段：soap_size / soap_body / trigger（first | after_rpc_response | after_soap_fault）
grep "ACS sending RPC request from task" omcgo-acs.log | grep <DT>
#    取出 soap_body 单独审：
jq -r 'select(.task_id=="<DT>") | .soap_body' omcgo-acs.log

# 3) 看 ACS 是否收到响应（根因下应当看不到——CPE 收到非法报文不响应 / HTTP 4xx 丢弃）
grep -E "RPC response received|ACS received SOAP Fault" omcgo-acs.log | grep <CWMPID>

# 4) 抓包对照（可选，最权威）：在 ACS 主机抓包，对照日志 soap_body 与实际发出的报文是否一致
tcpdump -i any -A -s 0 'tcp port 7547 and host <CPE_IP>' -w /tmp/cpe.pcap
```

**判定结论**：

| #1 params 形态 | #2 日志命中？ | #2 soap_body ParameterNames | #3 日志命中？ | 结论 |
|---|---|---|---|---|
| `{kv 空值 map}` 无 `names` 键 | ✓ | **空**（仅 `<ParameterNames soap-enc:arrayType="xsd:string[0]"/>`）| ✗ | **本文档分析的根因**（Q4 修复） |
| 含 `names` 数组 | ✓ | 有合法 TR-069 路径 | ✗ | CPE 侧问题（设备离线/防火墙/认证）→ 转 Q3 抓包 |
| 含 `names` 数组 | ✓ | 有合法 TR-069 路径 | ✓ 但 result 仍 NULL | 事件链断裂 → 转 Q5/Q6 |
| 任意 | ✗ | — | — | task 没出 ACS（队列没入或 PopTask 异常）→ 转 Q1 决策树第 1-2 行 |

---

## 排查问题逐条回答

### Q1 — 关键节点日志/排查方法

**查日志的命名空间分两份**：
- ACS 进程：`omcgo-acs.*.log`（zap json 输出，带 `device_sn`、`session_id`、`cwmp_id`、`task_id` 字段）
- APP 进程：`omcgo-app.*.log`（含 `mml-fanout`、`mml-result-aggregator`、`completion-router` 命名 logger）

**整条链路的"开/关锁信号"**（按出现顺序）：

| # | 现象 | grep 关键字 | 关键字段 | 文件:行 |
|---|------|-----------|---------|--------|
| 1 | MML 任务建好 | `mml task fan-out completed` | `mml_task_id` `device_count` `device_tasks_created` | fanout.go:50 |
| 2 | ACS 弹到 device_task 并下发 SOAP | `ACS sending RPC request from task` | `task_id` `cwmp_id` `method` `soap_size` `soap_body`✨ `trigger`✨ | acs/handler.go:586 / 752 / 1009 |
| 2-debug | 任意方向 SOAP 出口 | `ACS sent SOAP` (Debug) | `soap_size` `soap_body` | acs/handler.go:1303 |
| 3 | Redis 标记已发送 | `task sent` | `task_id` `cwmp_id` | task/service.go:210 |
| 4 | ACS 收到响应（含完整 XML，Debug） | `ACS received RPC response` (Debug) `RPC response received` (Info) | `cwmp_id` `xml`/`method` | acs/handler.go:634 / 640 |
| 5a | 响应是 Fault | `task failed with SOAP fault` | `task_id` `fault_code` `fault_msg` | acs/handler.go:684 |
| 5b | 响应是成功 | `task completed` | `task_id` `method` | acs/handler.go:697 |
| 6 | TaskService 收尾 | `task completed` / `task failed` | `task_id` | task/service.go:240 / 272 |
| 7 | NATS 桥接订阅（启动一次）| `task completion event bridge subscribed` | `subjects` | task/event_bridge.go:49 |
| 8 | Router 找不到处理器（异常）| `no completion handler registered for task source` | `task_id` `source` `source_id` | task/completion_router.go:120 |
| 9 | MML 聚合器最终化 | `mml task finalized` | `mml_task_id` `status` `result` `success` `failed` | mml/result_aggregator.go:106 |
| 10 | 推 SSE | （SSE 客户端日志）| — | result_aggregator.go:127 |

> ✨ = 本次新增字段。`soap_body` 字段直接含完整 SOAP XML（含 `cwmp:GetParameterValues` 元素及其子节点），可以离线 `xmllint --format` 美化后人工对照 TR-069 协议。`trigger` 取值：`first`（Inform 后首条）/ `after_rpc_response` / `after_soap_fault`，区分三种下发触发场景。
>
> **生产开关建议**：`soap_body` 在 Info 级；高 QPS 下若担心日志量，可在 logger 配置里把 `acs.handler` 的 Info 字段过滤掉这一项，但**调试期一定要打开**——它是诊断 Q4（params 形状不对）唯一不需要抓包就能拿到的证据。

**故障定位决策树**（先看 #2 的 `soap_body`，能省掉一半排查）：

```
[1] mml task fan-out completed ?
  ├─ 否 → MML Fanouter 没注入 / mml_command 没 RPCMethod / 事务失败
  │      • 查 omcgo-app.log 是否有 "MML fan-out bridge enabled"（启动一次）
  │      • 查 service.go:520 resolveRPCMethods 是否返回空
  │      • 查 mml_commands.rpc_method 字段是否落库
  └─ 是 → continue

[2] ACS sending RPC request from task ?
  ├─ 否 → CPE 没 Inform 或 task 没入 Redis
  │      • redis-cli ZRANGE acs:taskq:<sn> 0 -1
  │      • redis-cli HGETALL acs:task:<task_id>
  │      • 抓包看 :7547 是否有 Inform 上来
  └─ 是 → 看 soap_body：
      ├─ 形态 A: <cwmp:GetParameterValues>
      │           <ParameterNames soap-enc:arrayType="xsd:string[0]"/>
      │   → 命中 Q4 根因（params 没翻译成 names），下文 Q4 修复
      │
      ├─ 形态 B: <ParameterNames> 有合法 Device.* / IGD.* 路径
      │   → 报文合法，问题在 CPE 侧
      │   → continue 看 [4]
      │
      └─ 形态 C: BuildRequest 阶段就报错 → 直接 grep "build RPC request from task"

[4] RPC response received ?
  ├─ 否 → CPE 没回 / 防火墙吞包 / Cookie 丢失
  │      • 抓包确认 CPE 是否真的回了 POST 到 :7547
  │      • 若包到了 → grep "no valid session cookie for RPC response"
  │      • 若包没到 → CPE 侧 syslog
  └─ 是 → continue

[5] task completed | task failed with SOAP fault ?
  ├─ 都没有 → cwmp_id 没匹配上
  │      • redis-cli GET acs:cwmp2task:<cwmp_id>
  │      • grep "get task by cwmp_id" + WARN
  └─ 有 → continue

[7-9] NATS 事件链 → MML 聚合器
  ├─ 启动期没有 "task completion event bridge subscribed"
  │   → cmd/app/provider/modules.go EventBus 为 nil 或 NATS 不通
  ├─ 有 "no completion handler registered for task source source=mml"
  │   → completionRouter.Register 漏调（modules.go:272）
  ├─ 有 "decode task event payload" WARN → 编解码不兼容
  └─ 有 "mml task finalized" → 数据库已更新，前端没刷出来 → SSE 通道
```

### Q2 — 基站是否拿到了 RPC 任务

**判定方法**（任一为真即可确认 CPE 已收到）：

1. **ACS 日志**：`grep "ACS sending RPC request from task" omcgo-acs.log | grep <task_id>` 能找到一行，说明 SOAP 已写入 HTTP 响应、ACS 一侧的 send 已完成。
2. **device_tasks 状态**：`status='sent'` 且 `sent_at` 非空。这俩是同一个动作（`MarkTaskSent`）原子完成的。
3. **CPE 侧确认**（最权威）：
   - 设备 syslog 搜索 `cwmp` / `GetParameterValues`
   - 在 ACS 主机抓包：`tcpdump -i any -A 'tcp port 7547 and host <CPE_IP>'`，看请求体里有没有 `<cwmp:GetParameterValues>`，对照 `<cwmp:ID>` 与 `device_tasks.cwmp_id` 是否一致

> 当前现象 `status=sent` 已经是"OMC 这一侧确信发出去了"。所以问题不在能不能拿到，而在**报文内容能不能被 CPE 接受**。

### Q3 — 基站是否对 RPC 进行了响应

**判定方法**（按可信度从高到低）：

1. **ACS 主机抓包**：`tcpdump -i any -A -s 0 'tcp port 7547'` 持续 30 秒，看是否有 `Server: ` 起头的 HTTP 应答从 CPE 传回。
2. **ACS 日志**：
   - 正常响应：`RPC response received` + `cwmp_id=<X>` —— [acs/handler.go:640](omcgo/internal/acs/handler.go#L640)
   - 协议级错误：`ACS received SOAP Fault` —— [acs/handler.go:929](omcgo/internal/acs/handler.go#L929)
   - 响应来了但 Cookie 失效：`no valid session cookie for RPC response` —— [acs/handler.go:648](omcgo/internal/acs/handler.go#L648)
3. **CPE 侧 syslog**：搜 `cwmp` / `parameter` / 错误码（厂商相关）。

> 当前现象 `result=NULL` 表示 [acs/handler.go:694] `MarkTaskCompleted` 与 [:681] `MarkTaskFailed` 两个分支都没走过。意思是 ACS 进程**根本没收到这次响应**。原因（按概率排序）：CPE 因报文非法直接断 TCP / HTTP 4xx；CPE 暂时离线；CPE 在等更长会话超时。

### Q4 — `device_tasks.params` 字段是否符合 TR-069 标准

**不符合**。

| 字段 | 实际值（来自现象） | TR-069/CWMP 要求 |
|------|------------------|-----------------|
| 顶层 schema | `{key: "", ...}`（map） | `{"names": [<param_path>, ...]}`（GetParameterValues）<br>或 `{"values": [{"name": "...", "value": "...", "type": "xsd:..."}]}`（SetParameterValues） |
| 单个 key | `LTE_BTSNUM`（OMC 内部参数代码 `mml_params.param_code`） | `Device.X_<OUI>.LTE.BtsNum` 或 `InternetGatewayDevice...`（`mml_params.tr069_path`） |

后果：`internal/acs/rpc/dispatcher.go:81-94` 反序列化结果是 `params.Names == nil`，渲染出空 `ParameterNames` 列表的 SOAP 报文，CPE 多半视为非法。

**修复方向**（任选一条，建议 2）：

1. **Fanouter 转换**：在 `buildDeviceTaskRequests` 之前，按 `cmd.RPCMethod` + 命令的 `mml_command_param_refs` 把前端表单 map 翻译为 TR-069 wire 格式：
   - LST/DSP（GetParameterValues）：取 `tr069_path` 列表 → `{"names":[...]}`
   - MOD/ADD（SetParameterValues）：把 `param_code → value` 翻成 `[{name=tr069_path, value, type=value_type}]` → `{"values":[...]}`
   - 其它（Reboot/FactoryReset 等）按各自 RPC 的 schema

2. **新增统一编排层**（推荐，可独立测试）：
   - `internal/mml/rpc_payload.go` 提供 `BuildTR069Params(cmd *MMLCommand, formValues map[string]any) (json.RawMessage, error)`
   - Fanouter 调它，service 单测覆盖每种 RPCMethod
   - 好处：未来 `acs/rpc/dispatcher.go` 接受的 schema 不再泄漏到 mml 模块

3. **前端预转换**（不推荐）：会把协议知识下放到前端，多个端（Web/SSE/脚本）各自维护一份。

修复后 `device_tasks.params` 的样例应当是：
```json
{"names":["Device.DeviceInfo.HardwareVersion",
          "Device.X_BAICELLS_LTE.BtsNum",
          "Device.X_BAICELLS_LTE.GsmMcc",
          ...]}
```

### Q5 — ACS 收到响应但没推到 MML 队列？

**当前现象不可能是这个原因**。`status='sent'` + `result=NULL` 说明 ACS 没走到 `MarkTaskCompleted`，`notifyCompletion` 也就没机会发布 NATS 事件——根本没"推"这一步可言。

但仍记录下推送链路的诊断点，**等修复 Q4 之后**如果 mml_tasks 还是不更新，再按这里查：

| 检查 | 命令 |
|------|------|
| ACS 进程是否注入了 EventBus | `grep "task completion event bridge subscribed" omcgo-app.log` —— APP 启动一次（注意：发布方在 ACS，订阅方在 APP，两边都要有 NATS 配置） |
| NATS 是否真的能收到消息 | `nats sub 'task.>'`，重新执行命令，看是否有 `task.completed` / `task.failed` |
| `notifyCompletion` 是不是被跳过了 | 它要求 `task.SourceID != ""`（service.go:569）。看 device_tasks 这一行的 `source_id` 是否非空——MML fanout 一定会写入 `mmlTask.ID`，正常应该都有 |
| 桥接侧解码是否失败 | `grep "decode task event payload" omcgo-app.log` |

### Q6 — MML 任务没从队列订阅？

**指 `mml.ResultAggregator` 有没有挂上 `task.completed/failed`**。

挂载点在 [cmd/app/provider/modules.go:269-282](omcgo/cmd/app/provider/modules.go#L269-L282)：

```go
aggregator := mml.NewResultAggregator(...)
if c.EventBus != nil {
    completionRouter := task.NewCompletionRouter(logger)
    completionRouter.Register(task.TaskSourceMML, aggregator)
    bridge := task.NewCompletionEventBridge(logger, completionRouter)
    if err := bridge.Subscribe(c.EventBus); err != nil {
        logger.Warn("subscribe task completion bridge", zap.Error(err))
    }
} else {
    c.miscDeps.taskSvc.AddCompletionCallback(aggregator)  // 单进程退化
}
logger.Info("MML fan-out bridge enabled")
```

**自检方法**（按顺序，任一失败说明订阅没生效）：

1. APP 启动日志有 `MML fan-out bridge enabled` —— 装配代码跑到了
2. APP 启动日志有 `task completion event bridge subscribed subjects=[task.completed task.failed]` —— NATS 订阅成功
3. 没有 `subscribe task completion bridge ... error=...` 的 warn 行
4. 没有 `no completion handler registered for task source source=mml`（[completion_router.go:120](omcgo/internal/task/completion_router.go#L120)）

> 同样：当前根因不是这个，但修完 Q4 后必须确认这四条都健康。

---

## 修复顺序建议（从低风险到根因）

1. ✅ **ACS 下发日志加 SOAP 完整报文**（已完成）：[acs/handler.go:586/752/1009](omcgo/internal/acs/handler.go#L586) 的 Info 日志补 `soap_size` + `soap_body` + `trigger` 字段；[sendSOAPResponse](omcgo/internal/acs/handler.go#L1303) 加 `ACS sent SOAP` Debug 兜底。
2. ✅ **Fanout 入参形状校验日志**（已完成）：[fanout.go:96-110](omcgo/internal/mml/fanout.go#L96-L110) 增加 `device_task params built` Info 日志，字段 `has_names / has_values / has_object_name / names_count / values_count / payload_size` 直接暴露 schema 形态。`SummarizeSchema` 公共工具：[tr069_payload.go:206](omcgo/internal/mml/tr069_payload.go#L206)。
3. ✅ **最小回归用例**（已完成）：[fanout_test.go](omcgo/internal/mml/fanout_test.go)（5 个用例覆盖 GPV/SPV/无 refs 跳过/JSON roundtrip）+ [tr069_payload_test.go](omcgo/internal/mml/tr069_payload_test.go)（18 个用例覆盖各 RPC method 与边界）。
4. ✅ **实现 Q4 的 `BuildTR069Params`** + 集成到 Fanouter（已完成）：[tr069_payload.go](omcgo/internal/mml/tr069_payload.go) + [fanout.go](omcgo/internal/mml/fanout.go) 整合；[service.go attachParamRefs](omcgo/internal/mml/service.go#L666) 在 ExecuteCommand / resolveRPCMethods 时把 `mml_command_param_refs` 挂到 entry，使 Fanouter 拿到 `tr069_path` + `value_type` 做翻译。
5. **CPE 抓包验证**（待执行，需要真实设备环境）：用真实 CPE 跑一次 LST DEVICE_INFO，`tcpdump -i any -A 'tcp port 7547'` 抓 SOAP 包，确认 `<ParameterNames>` 含 `<string>...</string>` 子项且每个都是合法 TR-069 路径。或直接 `jq -r '.soap_body' $ACS_LOG | xmllint --format -` 拉日志里的报文做对照。
6. **观察 mml_tasks 是否进入终态**：执行 `bash omcgo/scripts/diag_mml_task.sh --sn <SN> --mml-task <UUID>` 自动检查 8 个 checkpoint，配合 [docs/operations/troubleshoot-mml-rpc.md](../operations/troubleshoot-mml-rpc.md) 决策树定位。

## 配套交付物

| 路径 | 用途 |
|------|------|
| [internal/mml/tr069_payload.go](../../omcgo/internal/mml/tr069_payload.go) | TR-069 wire 格式翻译层（按 RPC method 路由） |
| [internal/mml/tr069_payload_test.go](../../omcgo/internal/mml/tr069_payload_test.go) | 18 个翻译层单测 |
| [internal/mml/fanout.go](../../omcgo/internal/mml/fanout.go) | Fanouter 接入翻译层 + schema 形态日志 |
| [internal/mml/fanout_test.go](../../omcgo/internal/mml/fanout_test.go) | 5 个 Fanouter 端到端单测（含 Q4 形态回归） |
| [internal/acs/handler.go](../../omcgo/internal/acs/handler.go) | ACS 下发日志补 `soap_body` + `trigger` 字段 |
| [docs/operations/troubleshoot-mml-rpc.md](../operations/troubleshoot-mml-rpc.md) | 8 步排查手册（落地命令、决策树） |
| [omcgo/scripts/diag_mml_task.sh](../../omcgo/scripts/diag_mml_task.sh) | 一键诊断脚本：`--sn <SN> --mml-task <UUID>` 输出 ✓/✗/? 报告 |

---

## 历史已完成项（保留参考）

> 2026-04-27 之前的 to-do：
> - MOD 提交时过滤未填参数 ✅（commit 097477dc）
> - MOD 智能 placeholder + 默认值提示 ✅（commit 097477dc）
> 详见 `git show 097477dc`。


## MML控制台需求完善
1、选择设备后，不从命令树下选择命令，“参数路径命令” 改为 “参数路径指定”
2、支持 LST、MOD、ADD、RMV 四种操作类型
3、MOD 时，有一直“参数值”的设置输入框
4、api/v1/mml/execute 调用存在问题，请结合实际情况修复，分析 API 的参数并正确传递
请求数据：{
    "task_name": "",
    "device_sns": [
        "120200024719AAB0039"
    ],
    "commands": []
}
响应报错：
{
    "code": 400,
    "message": "Bad Request",
    "details": "one of command_code, script_id, commands, param_paths is required",
    "request_id": "app-20260427175540-d92b01de"
}
