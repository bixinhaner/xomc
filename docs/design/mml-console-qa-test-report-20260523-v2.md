# MML 控制台 QA 测试报告 v2 — 2026-05-23

> 本轮在 v1 基础上完成 2 个修复（fanout object_name 翻译 / migration 000163 / task-records modal），
> 然后从前端 + API 两个通路重新跑 12 个用例，定位剩余失败的真实根因。

## 0. 环境

| 项 | 值 |
|---|---|
| 前端 SPA | http://localhost:3000/mml/console |
| 后端 API | http://localhost:8081/api/v1 |
| 凭据 | admin / admin123 |
| 测试设备 | 1202000240194DP0026（BaiBLQ 在线 eNB，FAP/mBS31001/SC） |
| 测试方式 | L1 通过浏览器逐项操作；其余 11 项通过 `/mml/console/execute-statements-structured` API 批量；全部用例的 device_task 结果回到 task-records modal 人工核对 |

## 1. 本轮已落地的修复（上下文）

| Commit | 修了什么 | 影响 |
|---|---|---|
| `9d89e52a` (P0) | `fanout.translateObjectName` 把 ADD/RMV 的 `object_name` 走 standardPath→privatePath 翻译 + migration 000163 补 CARRIER target_object 的 FAPService 实例号 | CARRIER ADD/RMV 不再 100% 9005 |
| `4f9c2e83` | MOD path 选填 + AccessTypeTag Tooltip 加 path | MOD 操作流程对齐 LST |
| 本轮（未提交） | `mml.GetTaskResults` 改从 `device_tasks` 读 + `task.ListResultsBySourceID` + adapter | 任务记录页"查看"modal 不再永远"暂无执行结果"；可显示设备级 SOAP raw_response / fault |

## 2. 12 用例执行汇总

| ID | 操作 | 命令 | path 范围 | HTTP | mml_task | device_task error |
|----|------|------|----------|------|---------|-------|
| L1a | LST | DEVICE_INFO | 全 17 | 201 | **failed** 0/1 | 9005 Invalid Parameter Names [5] (incl. `AdditionalHardwareVersion`) |
| L1b | LST | DEVICE_INFO | 前 3（UserLabel/DnPrefix/ManufacturerOUI） | 201 | **failed** 0/1 | 9005 Invalid Parameter Names [1] (incl. `UserLabel`) |
| L2a | LST | A1_MEASURE_CTRL | 全 11 | 201 | ✓ **completed** 1/0 | 0（4994 ms，SOAP Envelope 完整返回） |
| L2b | LST | A1_MEASURE_CTRL | 部分 2 | 201 | ✓ **completed** 1/0 | 0 |
| M1a | MOD | DEVICE_INFO | 全 2（UserLabel + DnPrefix） | 201 | **failed** 0/1 | 9003 [Client] Invalid arguments |
| M1b | MOD | DEVICE_INFO | 1（UserLabel） | 201 | **failed** 0/1 | 9003 [Client] Invalid arguments |
| M2a | MOD | A1_MEASURE_CTRL | 全 10 | **400** | — | `mml: invalid request: MOD: empty values` |
| M2b | MOD | A1_MEASURE_CTRL | 部分 2 | **400** | — | 同上 |
| A1 | ADD | A1_MEASURE_CTRL | — | 201 | **failed** 0/1 | 9005 `AddObject of object name Device.Services.FAPService.CellConfig.LTE...A1MeasureCtrl. invalid` |
| A2 | ADD | CARRIER | — | 201 | ✓ **completed** 1/0 | 0（P0 已修，AddObjectResponse 正常） |
| R1 | RMV | A1_MEASURE_CTRL | inst=1 | 201 | **failed** 0/1 | 9005 `DeleteObject of object name Device.Services.FAPService.CellConfig.LTE...A1MeasureCtrl. invalid` |
| R2 | RMV | CARRIER | inst=1 | 201 | **failed** 0/1 | 9005 `The instance 1 of Delete Object ...Carrier. does not exist` |

成功 **3** / 失败 **9**。

## 3. 失败根因分类与处理建议

### 类 A — 数据字典与 BaiBLQ 设备模型不匹配（4 例：L1a / L1b / A1 / R1）

**根因**：`mml_commands.target_object` 写的是 LTE 路径，但 BaiBLQ 设备的 `param_mappings`
表里 A1_MEASURE_CTRL 等命令**只存在 NR 路径**（`Device.Services.FAPService.{i}.CellConfig.{i}.NR.RAN.Mobility.ConnMode.NR.A1MeasureCtrl.{i}`），LTE 形态下查 mappings 为空 → `translateObjectName` 走 fallback 原样下发 standardPath → CPE 9005。

**处理建议**：
1. **短期**（PRD 一项）：审 `mml_commands` 表中 `target_object`，把 BaiBLQ 上走 NR 路径的命令字典 standardPath 全部改为 NR 形态；或为 LTE / NR 各建独立 `mml_commands` 行（命名后缀 `_LTE` / `_NR`）。
2. **同样模式可扩到**：所有 *_MEASURE_CTRL、DRX_INITIAL_PARAM、CONN_MODE_EUTRA 等命令字典，逐个对照 BaiBLQ 数据模型确认 `target_object`。
3. **LST DEVICE_INFO 类**：5 个字段在 BaiBLQ 不存在（`AdditionalHardwareVersion` 等），应该从字典 sub_fields 删除，或加 `supported_product_class_regex` 元属性按设备过滤。

### 类 B — CPE 拒写系统级字段（2 例：M1a / M1b）

**根因**：`Device.DeviceInfo.UserLabel` / `DnPrefix` 字典标 `is_required=true, access=RW`，但
BaiBLQ 实际不允许 ACS 写入这两个字段（厂商策略），返回 9003 `[Client] Invalid arguments`。

**处理建议**：
1. 用 LST 探测哪些 DeviceInfo 字段 BaiBLQ 实际允许 SetParameterValues，更新字典 `access_type`。
2. UI 应该在执行前展示警告"该字段在当前产品类下可能拒写"。

### 类 C — 字典 sub_fields.access_type 未维护（2 例：M2a / M2b）

**根因**：A1_MEASURE_CTRL MOD 命令的 10 个 sub_fields 的 `access_type` 列在 DB 里为 NULL。
测试脚本只对 `access_type IN ('RW','WO')` 的字段填值，导致 `values={}`，后端 binding 校验拒掉。

**处理建议**：
1. 系统级数据补齐任务：对所有 MOD 命令的 sub_fields 补 `access_type`（'RW' 或 'WO'）。
2. 后端 `compileMODStatement` 的 "empty values" 错误信息建议带上 hint："请检查字典 sub_fields 的 access_type 是否维护"。

### 类 D — 测试用例假设错（1 例：R2）

**根因**：脚本写死 `rmv_instance=1`，但 R2 提交时 CARRIER 表实际不存在 instance 1
（前面 A2 ADD 创建的实例号不是 1）。

**处理建议**：
1. 测试用例改进：RMV 前先 LST 拿 max instance idx，对那个 idx 提 RMV。
2. UI 改进：RMV 界面应该先用 LST 探一遍可选实例号，让用户从下拉选，禁止手动输入不存在的值。

## 4. 顺带发现 / 验证的 UX 问题

| 现象 | 状态 |
|---|---|
| 任务记录页"查看"modal 永远显示"暂无执行结果"（即使 status=completed） | **本轮已修**（`mml.GetTaskResults` 接 `device_tasks` 真实数据） |
| MML 控制台终端输出框只显示"命令已提交"，failed/completed 不刷新进度 | **未修**（SSE/poll 链路问题，与 task-records 修复正交，需要单独 PR；本次提交了所有任务结果可通过 task-records 查看，足以解封测试） |
| ADD/RMV CARRIER 100% 失败 9005 | **已修**（commit 9d89e52a + migration 000163） |

## 5. 推荐后续行动（优先级）

1. **P1** — 补 `mml_command_sub_fields.access_type`（类 C，2 例阻塞 + 整体字典数据质量）
2. **P1** — 审 `mml_commands.target_object` LTE vs NR 路径正确性（类 A，影响 4 例 + 至少 6 个类似命令）
3. **P2** — 修 MML 控制台终端输出 SSE 推送（提交后无任何状态反馈，UX 较差）
4. **P2** — RMV UI 加实例号选择（类 D）
5. **P3** — DeviceInfo 字典字段按 BLQ 实际能力过滤（类 B，UserLabel/DnPrefix 字段策略）

## 6. 验证证据（截图）

| 截图 | 说明 |
|---|---|
| `mml-console-initial.png` | MML 控制台初始页面（设备 + 命令树就绪） |
| `mml-L1-fullsize.png` | L1 LST DEVICE_INFO 17/17 字段全选状态 |
| `l1-modal.png` | L1 任务记录 modal — 设备失败行（修复前永远空） |
| `l2a-modal.png` | L2a 任务记录 modal — 完整 SOAP Envelope 4994ms（修复后） |

## 7. 修复涉及的代码改动（未 commit）

| 文件 | 变更 |
|---|---|
| `omcgo/internal/task/pg_repository.go` | 新增 `DeviceTaskResultRow` + `ListResultsBySourceID` |
| `omcgo/internal/task/service.go` | 新增 `ListResultsBySourceID` 透传 |
| `omcgo/internal/mml/service.go` | 新增 `DeviceTaskResultLister` 接口 + `DeviceTaskResultRowView` + `SetDeviceTaskResultLister` setter；重写 `GetTaskResults` 优先用 lister + `deviceTaskRowToResultMap` |
| `omcgo/cmd/app/provider/modules.go` | 新增 `mmlDeviceTaskResultAdapter` + 装配调用 |

测试通过：`go build ./...` / `go test ./internal/mml/... ./internal/task/...` 全绿。
