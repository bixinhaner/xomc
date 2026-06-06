# MML 控制台 V2（/mml/console-v2）测试结果

> 执行日期：2026-06-06 · 方式：Playwright 浏览器自动化 + 后端 DB/API 交叉校验
> 配套方案：[mml-console-v2-test-plan-20260606.md](./mml-console-v2-test-plan-20260606.md)
> 真机设备：**SN=1202000240194DP0026**（cmcc/lte，FAP/mBS31001/SC，真实在线）
> 环境：docker compose 栈（web `:8081`）；登录 admin/admin123

---

## 0. 结论摘要

- **三步流程 UI、命令/参数配置、动态结果列、TR-069 路径语义、CSV 导出**：核心功能验证通过。
- **真机执行 + 读回**：后端全链路正确——设备真实响应 GPV，读回 `Device.DeviceInfo.SoftwareVersion = BaiBLQ_5.0.16.1_1229`，并正确落库 + 导出。
- **发现并修复 2 个健壮性缺陷**：
  - **BUG-1（HIGH）**：执行结果实时呈现**仅依赖 SSE**，丢帧即停「执行中」且任务从命令记录丢失。→ 实现**轮询兜底**（终态从后端 `/results` 落库，不依赖 SSE）。
  - **BUG-3**：结果端结果行未解析 SOAP `raw_response` 导致读回值列空。→ 复用 `parseMmlDeviceTaskResult` 解析 GPV 回填。
  - 二者修复后真机复测 ✅：结果自动呈现「成功 + 读回值」，不依赖 SSE（§3.1）。
- 另记录环境项 BUG-2（同用户多会话 SSE 互踢）与观察项 OBS-1（GPV 全或无）/OBS-2（历史重建未拉结果行）。

---

## 1. 用例结果汇总

| 编号 | 用例 | 结果 | 证据/说明 |
|---|---|---|---|
| TC-DEV-01 | 仅在线 + 列序 SN/状态/产品/产品类型 | ✅ PASS | 列表全部「在线」；列序正确，无设备分组；蓝色「搜索」按钮在产品类型后 |
| TC-DEV-02 | 输入 SN 不实时过滤 | ✅ PASS | 输入后列表仍「共 4008 台」，未触发查询 |
| TC-DEV-03 | 点「搜索」按 SN 过滤 | ✅ PASS | 命中「共 1 台」= 目标 SN |
| TC-DEV-08 | 勾选 + 确定 | ✅ PASS | 顶部「① 选择设备 已选 1 台」，②命令解禁 |
| TC-DEV-09 | 重复设备行（数据异常）影响 | ✅ PASS | 同 SN 在 DB 有 2 行，但列表只 1 行，执行未重复扇出 |
| TC-CMD-01 | 命令树 + 搜索「命令分组/名称」+ 标题「指定参数」 | ✅ PASS | 分组→命令两级正确 |
| TC-CMD-02 | 选 LST 命令右侧详情 | ✅ PASS | 操作类型 + 命令名 + 「参数 PATH（17 项）」具体路径 |
| TC-CFG-01 | 命令参数统一头部 + 勾选列表 | ✅ PASS | 「全选 LST·查询 …勾选要查询的参数」+ 17 勾选项 |
| TC-CFG-02 | `{i}` 实例选择器默认 1 | ✅ PASS | 选「当前告警实例」(`CurrentAlarm.{i}.*`)→「对象实例（{i}，默认 1）：CurrentAlarm [1]」 |
| TC-RAW-01 | 裸路径具体实例 LST 校验通过 | ✅ PASS | `…CurrentAlarm.1.AlarmIdentifier` 无报错、可执行 |
| TC-RAW-02 | 裸路径含 `{i}` 拒绝 | ✅ PASS | 行内红字「裸路径不支持 {i} 占位符」+ 执行按钮禁用 |
| TC-EXE 动态列 | 结果列按 path 动态生成 | ✅ PASS | LST 17 path → 17 结果列（用户友好名）；裸路径 1 path → `SoftwareVersion` 列 |
| TC-EXE 读回(后端) | 真机执行读回值正确 | ✅ PASS | device_task=completed，`SoftwareVersion=BaiBLQ_5.0.16.1_1229` |
| TC-EXE 实时呈现(UI) | 结果就地回填 | ✅ PASS(修复后) | 修复轮询兜底+读回解析后，真机执行结果自动呈现「成功 + BaiBLQ_5.0.16.1_1229」，不依赖 SSE（见 §3.1） |
| TC-HIS-01 | 执行后生成命令记录 | ✅ PASS | 轮询兜底落库；localStorage 持久化、「下载全部」解禁 |
| TC-DL-01 | 全设备汇总 CSV | ✅ PASS | 列含动态 path，值正确，UTF-8 中文正常（见 §2） |
| TC-DL-02 | 单设备 CSV | ✅ PASS | 纵向摘要 + 参数路径/读回值，清晰易读（见 §2） |
| TC-DL-03 | CSV 与真机读回一致 | ✅ PASS | CSV 值 = `BaiBLQ_5.0.16.1_1229` = 设备实际值 |
| TR-069 GPV | 全或无语义 | ✅ 符合 | 见 OBS-1 |

> 未单独在浏览器逐项执行：TC-DEV-05(批量)/06(分页 20/50)/07(全选 200)、TC-CFG-03/04(MOD/RMV 值默认)、TC-HIS(记录切换)、TC-RAW-03/04(ADD/RMV 形态)。其中批量/分页/校验逻辑已在代码与单测层覆盖（`instanceAndRaw.test.ts`），UI 复测列入后续补充（见 §4）。

---

## 2. CSV 导出实测内容

**全设备汇总**（`POST /mml/tasks/:id/export`）：
```
序号,设备SN,状态,Device.DeviceInfo.SoftwareVersion,故障码,故障信息,下发时间,响应时间
1,1202000240194DP0026,成功,BaiBLQ_5.0.16.1_1229,,,2026-06-06 12:17:09,2026-06-06 12:17:09
```

**单设备**（`POST /mml/tasks/:id/devices/:sn/export`）：
```
设备SN,1202000240194DP0026
状态,成功
故障码,
故障信息,
下发时间,2026-06-06 12:17:09
响应时间,2026-06-06 12:17:09

参数路径,读回值
Device.DeviceInfo.SoftwareVersion,BaiBLQ_5.0.16.1_1229
```
评价：列随 path 动态、读回值正确、中文表头可读、汇总=横向矩阵 / 单设备=纵向明细，符合「清晰易读」要求。

---

## 3. 缺陷与观察

### BUG-1（HIGH，已修复）执行结果仅依赖 SSE，丢帧即停「执行中」且任务丢失
- **现象**：真机执行后，后端 device_task 已 `completed` 且读回值正确，但 UI 结果行长期停「执行中」、`-`。
- **根因**：①结果实时回填只走 SSE `mml_device_frame`/`mml_task_completed`；②任务 ID 仅在**收到完成帧**时才写入 localStorage 命令记录——完成帧一旦丢失，任务从命令记录彻底丢失，无法事后重建。③SSE hub 为**每用户单连接**（`events/hub.go`「kicked existing SSE connection」），同一 `admin` 被两个浏览器（用户真实 Chrome + 测试 Chrome，UA 148/149）各自占用 → 每 3s 互踢重连 → 完成帧落到对端连接被本页丢弃。
- **修复**：在 `ConsoleV2/index.tsx` 增加**轮询兜底**——运行期每 4s `GET /mml/tasks/:id`，到达终态（completed/failed/expired/cancelled/timeout）即 `GET /:id/results` 重建结果行并落入命令记录，与 SSE 完成路径等价。结果呈现不再依赖实时帧，对丢帧/多会话/网络抖动健壮。
- **验证**：endpoint 实测可用（task 状态 completed + results 1 条带读回值）；UI 复测见 §3.1。

### BUG-2（环境）同用户多会话 SSE 互踢
- 上述 hub 单连接策略在「同一用户多浏览器/多标签」下会互踢，实时功能整体退化（不止 ConsoleV2）。**建议**（独立改动，影响所有实时消费方，需单独评估）：SSE hub 支持**每用户多连接 fan-out**（按连接分发，不再 kick），从根上消除多会话互踢。本次未改 hub（爆炸半径大）。

### OBS-1（TR-069，符合协议但影响 UX）GPV 全或无
- 「LST 设备基本信息」勾选 17 path，其中 5 个为该设备不支持（如 `Device.DeviceInfo.AdditionalHardwareVersion`），设备返回 **9005 Invalid Parameter Names**，**整条 GPV 失败**（TR-069 规定单个非法名即整请求失败）。
- **建议**：按 paramModel 的 `is_supported` 过滤命令可勾选 path，或对 9005 给出「剔除非法 path 重试」引导；勿默认全勾不受支持的 path。

### BUG-4（MEDIUM，已修复）「指定参数」流程下「③ 配置参数」按钮不可点
- **现象**：用「指定参数」(裸路径)流程执行后，顶部「③ 配置参数」按钮置灰不可用（即便已有「指定参数 · 1 PATH · LST」配置），无法回到配置页编辑裸路径。
- **根因**：`SelectionBar` 中「③ 配置参数」`disabled={!command}`——以「已选命令」为门；而裸路径模式**无命令**，导致该流程下永远不可点。
- **修复**：改为 `disabled={deviceCount === 0}`——选好设备即可打开配置弹框（命令参数 / 指定参数两模式都在弹框内，指定参数无需命令）。

### BUG-5（MEDIUM，已修复）裸路径执行的命令记录命名为「RAW LST」不规范
- **现象**：「指定参数」(裸路径)执行后，命令记录名为「RAW LST」/「裸路径 查询」，看不出执行了什么。
- **修复**：改为**用执行的 path 命名 + 取设备模型 path 字典里的友好名**：`mmlApi.resolveParamNames` 按 path 查 `standard_params.description`（如 `Device.DeviceInfo.SoftwareVersion`→「软件版本」），命名为「查询 软件版本」（多 path「…等N项」）；字典无命中/无权限时回退路径叶子名「查询 SoftwareVersion」。Live/轮询路径与跨刷新重建均覆盖。
- **注**：友好名查询复用了 admin 级 `standard-params` 字典端点；非超管操作员若无该读权限会回退叶子名。**建议**后续提供 console 级 path→name 端点以保证一致。
- **复测 ✅**：真机裸路径 LST `Device.DeviceInfo.SoftwareVersion` 执行后，命令记录/结果标题显示 **「查询 软件版本」**（不再「RAW LST」），执行/读回仍正确（成功 / BaiBLQ_5.0.16.1_1229）。

### OBS-2 命令记录跨刷新重建只重建列、不重建结果行
- `useConsoleHistory` 重建只调 `GET /mml/tasks/:id`（含命令/列，不含 per-device 结果），未调 `GET /:id/results`，故刷新后历史记录有列无行。**建议**：`select` 时惰性拉 `/results` 补行（设计 §3.12.4 P3 TODO）。

---

### 3.1 轮询兜底修复复测（真机）
> 注：首次 `up -d --build web` 命中 Docker 层缓存未部署新代码（服务 chunk `getTaskResults=0`）；改 `docker compose build --no-cache web` 后部署生效（`getTaskResults=1`）。**重建务必用 `--no-cache`**。

**第一轮复测**（轮询兜底已部署）：真机裸路径 LST `Device.DeviceInfo.SoftwareVersion` 执行后，**无需刷新、未依赖 SSE**，结果行在 ~十几秒内自动由「执行中」→ **状态=成功** + 响应时间（观察到 4 次 `GET /mml/tasks/:id` 轮询）。→ BUG-1 状态呈现已修复。

**第一轮暴露 BUG-3**：状态已「成功」但**读回值列仍为 `-`**。根因：`mapResultItemToRow` 把结果信封 `{method, raw_response}` 当成 name→value 直接取值（信封里没有 path 键），而读回值在 `raw_response` SOAP 里。SSE 路径 `applyFrameToRow` 用 `parseMmlDeviceTaskResult` 解析 GPV，结果端路径漏了同样的解析。
**修复 BUG-3**：`mapResultItemToRow` 改为 `parseMmlDeviceTaskResult(parsedData)` → GPV params → 按 path(exact→leaf) 回填，与 SSE 路径对齐。

**第二轮复测**（含 BUG-3 修复，`--no-cache` 重建后）：✅ **PASS**。真机裸路径 LST 执行后，结果行**无需刷新、不依赖 SSE**，自动呈现：
```
状态=成功   SoftwareVersion=BaiBLQ_5.0.16.1_1229   响应时间=12:35:09
```
读回值正确填入动态列；命令记录自动落库（localStorage 持久化 2 条 ID，「下载全部」解禁）。BUG-1 / BUG-3 修复均验证通过。

> 残留小项：结果行「下发时间」列为 `-`（`/results` 重建未回填 sent_at，响应时间正常）。非阻断，列入后续完善。

---

## 4. 后续补充测试清单
- TC-DEV-05/06/07：批量输入只显在线、分页 20/50、表头全选 200 的 UI 复测。
- TC-CFG-03/04：MOD 值默认 min_value、RMV 实例号 的 UI 复测。
- TC-HIS-01~03：多次执行后命令记录切换结果正确（依赖 BUG-1 修复后稳定产生记录）。
- TC-RAW-03/04：ADD 末级实例号/缺尾点、RMV 缺实例号 的 UI 报错复测。
- OBS-2 修复后：跨刷新历史结果行重建。

---

## 5. 第二轮：多设备 / 结果详情 / 命名 / 下载（2026-06-06，req 1–8）

> 真机复测（web `--no-cache` 重建后部署）。**重建教训**：后台 `docker compose build --no-cache web` **未必更新镜像**（实测镜像未刷新、旧 bundle 仍在 serve）；务必**前台**跑 build 并 `docker images docker-web` 确认 `CreatedSince` 刷新，再 `up -d --force-recreate web`，最后 grep 服务端 chunk 确认改动已部署。

| 需求 | 结果 | 证据 |
|---|---|---|
| req1 多设备批量（含目标） | ✅ PASS | 选 3 台（目标 + 2 在线）执行；顶部「已选 3 台」「确定并执行（3 台）」；目标行成功+读回值；汇总「总数 3 / 成功 1 / 执行中 2」 |
| req3 一台成功其余失败 | ⚠️ 环境受限（部分演示） | 目标=成功；其余 `is_online=true` 但 **stale** 设备 ACS 发 CR 后永不应答→device_task 长期 `pending`(执行中)，仅到期(~1h)转失败。测试窗口内表现为「成功 1 + 执行中 2」。真实「失败行」用 9005 复现（见 req5/7）。**非产品 bug，属测试数据环境**。 |
| req4 命令记录↔任务记录名称对应 | ✅ PASS（已修复） | 裸路径执行后 `mml_tasks.task_name` = **「查询 软件版本」** = 命令记录名（原为「LST Device.DeviceInfo.SoftwareVersion 1202…」不对应）。结构化也传 `task_name=命令名`。 |
| req5 「查看」命令 ID 显示 | ✅ PASS | 成功行：命令 ID `9fa2d6be…`、设备任务 ID、SN、状态、时间齐全；失败行（9005）：命令 ID `e37cd9dc…` + 故障信息「9005 / Invalid Parameter Names」。 |
| req7 下载成败明确区分 | ✅ PASS | 失败任务汇总 CSV：`1,…,失败,,9005,"[Server] Invalid Parameter Names […]",…`；单设备 CSV：`状态,失败` + `故障码,9005` + `故障信息,…`。状态/故障码/故障信息三列逐设备区分。 |
| req8 命令记录不再「打开任务详情」 | ✅ PASS（已修复） | 命令记录条目去掉深链按钮（`deepLinkIcons:0`）；理由：一条命令对应多条**分设备**任务记录，单 task 详情无法对应；逐设备详情走结果行「查看」。 |
| req2 单设备混合成功/失败 path | ✅ PASS（已补「逐 PATH」，见 §5.3） | 真机单设备逐 PATH LST `[SoftwareVersion(有效)+AdditionalHardwareVersion(无效)]` → 后端拆 2 条 command 顺序下发，结果合并为**一行**：状态=失败、SoftwareVersion=`BaiBLQ_5.0.16.1_1229`、AdditionalHardwareVersion=`✗ 失败`；「查看」逐 path 列成功/失败。 |
| req6 保留浏览器 | ✅ | 测试后浏览器保持打开在 `/mml/console-v2`，命令记录含本轮数据可查。 |

### 5.1 本轮修复
- **BUG-4**（req8）命令记录去掉「打开任务详情」深链（`CommandHistoryPanel`）。
- **BUG-6**（req4）命令记录名与后端 `task_name` 对齐：裸路径 `buildRawExecutePayload(..., taskName=命令名)`、结构化 `execute-statements-structured` 传 `task_name=command.commandName`、非结构化 legacy 同。
- 命令记录命名复用 BUG-5 的 `resolveParamNames`（standard_params.description）→ 友好名（「附加硬件版本」「软件版本」均正确解析）。

### 5.2 本轮发现（待办）
- ~~**GAP-1（req2）**「逐 PATH」未实现~~ → **已实现并验证，见 §5.3**。
- **GAP-2（req3）** 测试环境仅 1 台真机在线；其余 stale 设备 pending 不快速失败，无法实时演示「多设备含失败」。建议测试环境接入第 2 台真机或可快速失败的设备。
- **次要**：`查看` 详情「执行命令」行对裸路径显示偏冗余（如「LST 查询查询 附加硬件版本LST (裸路径)」，Tag 的操作标签与命令名内「查询」重复）；poll 重建行「设备任务 ID」为空（`/results` 未返 device_task id）。均非阻断。

---

## 5.3 逐 PATH 执行（req2 补齐，2026-06-06）

补齐「逐 PATH」端到端能力：单设备一条命令含多 path 时，每 path 一条 RPC 独立成败，结果合并为每设备一行 + 逐 path 详情。

### 实现
- **后端**（`internal/mml/service.go` `ExecuteCommand` + `handler.go`）：`ExecuteRequest`/`ExecuteHTTPRequest`/`CreateTaskHTTPRequest` 新增 `execute_mode`（`""/whole` | `single_path`）；`single_path` 时 LST/MOD 把 N 个 path 拆成 **N 条 command**（每条 1 path），Fanouter/Sequencer 据此为每设备生成 **N 个 device_task**（顺序执行），path 级成败独立。ADD/RMV 天然单 path 不受影响。
- **前端**：
  - `ConfigParamsModal` 的「整体下发 / 逐 PATH」单选已接线（原为 UI 桩）：`buildRequest` 带 `execMode`。
    - **指定参数(裸路径)**：`buildRawExecutePayload` 发 `execute_mode: single_path`，后端拆 command。
    - **命令参数(结构化)**：前端把 LST/MOD 多 checked path **按列序拆成 N 条 statement**（每条 1 path），`execute-statements-structured` 后端每 statement 编 1 条 command（`buildStatementCommandEntries`，LST/MOD 单 path 恒 1 条）→ N device_task；command 序与结果列序一致便于合并。ADD/RMV 不拆。
  - `mapBackendResult` 透传 `command_index`；新增 `buildDeviceRows(items, columns, read)`：按设备分组合并多 path 结果——成功 path 填读回值、失败 path 单元格标 `✗ 失败`、行状态 = 全成功才 `success` 否则 `failed`，并填 `pathTasks` 供「查看」逐 path 展示。
  - 结果收口统一走 `finalizeFromResults`（SSE 完成帧 + 轮询兜底共用，`finalizedRef` 去重）：**拉 `/results` → buildDeviceRows**，不再取被 SSE 逐帧覆盖的行；拉不到结果不收口、交轮询重试（避免落空行）。
  - `ResultTable` 动态列渲染修复：失败行不再一律 `-`，逐格呈现（成功 path 值 / `✗ 失败` 红字 / 整体失败回退 `-`）。
- 单测：`__tests__/buildDeviceRows.test.ts`（合并/失败标记/行状态/pathTasks/分组，3 例，全过）。

### 真机验证（SN=1202000240194DP0026）
逐 PATH LST `Device.DeviceInfo.SoftwareVersion`（有效）+ `Device.DeviceInfo.AdditionalHardwareVersion`（无效）：
- 后端：1 个 `mml_task`（`commands` 长度 2）→ 2 个 device_task：`command_index=0` completed、`command_index=1` failed(9005)，**顺序执行**（cmd1 在 cmd0 完成后才下发）。
- 前端结果行（合并）：
```
状态=失败   SoftwareVersion=BaiBLQ_5.0.16.1_1229   AdditionalHardwareVersion=✗ 失败
```
- 「查看」逐 PATH 子任务表：SoftwareVersion → 成功；AdditionalHardwareVersion → 失败。

**命令参数(结构化)通道**复测：选「LST查询 设备基本信息」→ 命令参数仅勾「软件版本(有效)+附加硬件版本(无效)」→ 逐 PATH 执行：
- 后端：1 个 `mml_task`（`commands` 长度 2）→ `command_index=0` completed、`command_index=1` failed(9005)。
- 前端结果行（合并，列用**友好名**）：
```
状态=失败   软件版本=BaiBLQ_5.0.16.1_1229   附加硬件版本=✗ 失败
```
- 命令记录名「查询 设备基本信息」。→ 两通道（指定参数 / 命令参数）逐 PATH 均验证通过。

### 备注
- 「逐 PATH」作用于 **LST/MOD**（指定参数裸路径 + 命令参数结构化两通道均支持）；ADD/RMV 天然单对象不拆。
- **UI 硬限制**（`ConfigParamsModal`）：ADD/RMV 时「逐 PATH」单选项 **disabled**（Tooltip「ADD / RMV 为单对象操作，不支持逐 PATH 拆分」），并以 `effectiveExecMode` 强制整体下发——即便此前选过逐 PATH，切到 ADD/RMV 也会回落整体下发。复测：LST/MOD=可选、ADD/RMV=禁用 ✅。
- 自动化测试注意：AntD AutoComplete 行输入须用**真实键盘** `pressSequentially`/`keyboard.type`（native value-set 不提交），且**勿按 Escape**（会关弹框）；逐 PATH 顺序执行需等待 ~30-40s 全部 path 完成。
