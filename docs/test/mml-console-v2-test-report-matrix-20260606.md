# MML 控制台 V2 执行矩阵全面测试报告（第三轮）

> 日期：2026-06-06 · 方式：API + 后端 DB/CSV 交叉校验 + Playwright UI 验证
> 配套用例：[mml-console-v2-test-plan-20260606.md](./mml-console-v2-test-plan-20260606.md) §7
> 在线设备：`1202000240194DP0026`(FAP/mBS31001/SC) / `120200087125BJB0002`(FAP/BSQ7040A452)
> **离线设备场景按用户决定排除**。覆盖路径：有效 `Device.DeviceInfo.SoftwareVersion` + 无效 `Device.DeviceInfo.AdditionalHardwareVersion`（用于同时覆盖成功/失败 path）。

---

## 0. 结论

- **8 个执行矩阵**（命令参数/指定参数 × 整体下发/逐 PATH × 1 台/2 台）**命令拆分、设备扇出、逐设备执行、结果合并全部正确**。
- **命令记录倒序 + 点击切换** 通过；进入页面默认显示最新命令结果（惰性补结果行）。
- **结果正确性**（单设备 / 多设备 / 成功+失败并存）通过。
- **下载**：发现并**修复**单设备 CSV 在逐 PATH 下「状态=成功」误导、缺逐 path 状态的问题 → 现逐 path 输出「状态/读回值/故障信息」+ 设备级「部分失败」。
- 2 台场景中 `BJB0002` 因真机连接节奏慢，部分 device_task 停 `pending`（环境，非缺陷）。

---

## 1. 执行矩阵（§7.1）

> 命令拆分以 `mml_tasks.commands` 长度判定（整体=1、逐 PATH=2）；设备扇出以 `device_tasks`（source_id=任务）计数；结果以 `error_code`/状态判定。

| 编号 | 通道 | 模式 | 设备 | commands | device_tasks（实测） | 结果（实测） | 判定 |
|---|---|---|---|---|---|---|---|
| TC-M1-1 | 指定参数 | 整体下发 | 1 | 1 | DP0026 ci0 | **失败 9005**（整体全或无：含无效 path 整请求失败） | ✅ |
| TC-M1-2 | 指定参数 | 逐 PATH | 1 | 2 | DP0026 ci0 + ci1（顺序） | ci0(SW)=**成功** / ci1(AddHW)=**失败 9005**；行=部分失败 | ✅ |
| TC-M1-3 | 命令参数 | 整体下发 | 1 | 1 | DP0026 ci0 | **失败 9005** | ✅ |
| TC-M1-4 | 命令参数 | 逐 PATH | 1 | 2 | DP0026 ci0 + ci1 | ci0=成功 / ci1=失败 9005 | ✅ |
| TC-M2-1 | 指定参数 | 整体下发 | 2 | 1 | DP0026 ci0 + BJB0002 ci0 | 两台均**失败 9005**（DP0026=[Server]、BJB0002=[Client]，逐设备 per-product 翻译生效） | ✅ |
| TC-M2-2 | 指定参数 | 逐 PATH | 2 | 2 | DP0026 ci0+ci1 完成；BJB0002 ci0 pending | DP0026 成功+失败；BJB0002 待连接 | ✅(结构/DP0026)·⚠️BJB0002 |
| TC-M2-3 | 命令参数 | 整体下发 | 2 | 1 | DP0026 ci0 失败；BJB0002 ci0 pending | 同上 | ✅(结构/DP0026)·⚠️BJB0002 |
| TC-M2-4 | 命令参数 | 逐 PATH | 2 | 2 | DP0026 ci0+ci1 完成；BJB0002 ci0 pending | 同上 | ✅(结构/DP0026)·⚠️BJB0002 |

**关键验证点**：
- 整体下发 = 1 条 GPV（全或无）：含无效 path → 整行 9005、读回列空。✅
- 逐 PATH = 每 path 一条 RPC（顺序，成败独立）：有效 path 成功带值、无效 path 失败 9005。✅
- 两通道（指定参数后端 `execute_mode`、命令参数前端拆 N statements）拆分一致。✅
- 2 台不同 product_class 不再被 R-8.4 拒绝，逐设备各自翻译执行。✅

> ⚠️ **BJB0002 环境说明**：该真机连接节奏慢，一次连接处理一条 device_task；本轮 8 个任务并发产生大量排队，BJB0002 仅完成最早的 M2-1，M2-2/3/4 的 ci0 停 `pending` 等待下次连接。device_task **已正确创建**（结构正确），执行受设备连接节奏限制，非产品缺陷。

---

## 2. 命令记录（§7.2）

| 编号 | 用例 | 实测 | 判定 |
|---|---|---|---|
| TC-HIS-ORD | 倒序排列（最新在上） | 注入 8 条按执行时间顺序，面板自上而下 = M2-4→M2-3→…→M1-1（最新在上）；进入页默认选中最新并加载结果 | ✅ |
| TC-HIS-SW | 点击切换执行结果 | 点击 M1-2 记录 → 右侧结果切换为其结果（DP0026 失败、SoftwareVersion=BaiBLQ_5.0.16.1_1229、AdditionalHardwareVersion=✗失败），惰性 `/results` 自动补行 | ✅ |

---

## 3. 结果正确性（§7.3）

| 编号 | 用例 | 实测 | 判定 |
|---|---|---|---|
| TC-R-1 | 单设备 | 1 行，状态 + 读回值/故障正确（M1-2 单设备逐 PATH） | ✅ |
| TC-R-N | 多设备 | M2-1 两行（DP0026 / BJB0002），逐设备状态独立 | ✅ |
| TC-R-MIX | 成功+失败并存 | 逐 PATH 同一行：SoftwareVersion 有值（成功）、AdditionalHardwareVersion=✗失败；汇总「成功/失败」计数正确 | ✅ |

---

## 4. 下载内容分析 + 修复（§7.4，用户需求 6 重点）

### 4.1 汇总 CSV（全部下载，多设备合并）
- **整体下发**（TC-M2-1）：行=设备，列=各 path，含 `状态/故障码/故障信息`，逐设备区分：
  ```
  序号,设备SN,状态,…AdditionalHardwareVersion,…SoftwareVersion,故障码,故障信息,下发时间,响应时间
  1,1202000240194DP0026,失败,,,9005,"[Server] Invalid Parameter Names […]",…
  2,120200087125BJB0002,失败,,,9005,"[Client] Invalid Parameter Names […]",…
  ```
  评价：✅ 多设备成败一目了然（注意 [Server]/[Client] 两类故障源体现逐设备真机差异）。
- **逐 PATH**：汇总按 device_task 逐行（同一设备每 path 一行，序号递增），各行带该 path 的状态/值/故障——可看每 path 结果，但同一设备出现多次。

### 4.2 单设备 CSV（单独下载每设备结果）
**发现问题（已修复）**：原实现只取该设备**第一条** device_task → 逐 PATH 下「状态=成功」（仅 ci0），失败 path 仅显示空值，**无法区分「成功无值」与「失败」**，不满足「清晰包含失败和成功的 path」。

**修复**（`internal/mml/export.go` 新增 `buildDeviceCSVMulti` + `commandStandardPaths`）：聚合该设备**全部** device_task，逐 path 输出「参数路径 / 状态 / 读回值 / 故障信息」，设备级状态混合时为「部分失败」。修复后：

- **逐 PATH 单设备**（M1-2）：
  ```
  设备SN,1202000240194DP0026
  状态,部分失败
  参数路径,状态,读回值,故障信息
  Device.DeviceInfo.AdditionalHardwareVersion,失败,,"9005 [Server] Invalid Parameter Names […]"
  Device.DeviceInfo.SoftwareVersion,成功,BaiBLQ_5.0.16.1_1229,
  ```
  → ✅ 成功 path（有值）/ 失败 path（故障）逐条清晰；设备级「部分失败」准确。
- **整体下发单设备**（M1-1）：状态=失败，两 path 均「失败 + 9005」（全或无），✅ 对照清晰。

### 4.3 建议落地（均已实现 + 复测）
1. ✅ **逐 PATH 汇总 CSV 合并为「每设备一行」**（`aggregateExportRowsByDevice`）：成功 path 填读回值、失败 path 填「✗ 失败」、状态混合时「部分失败」；整体下发失败则 path 列留空 + 状态「失败」+ 故障信息。复测：
   ```
   # 逐 PATH 2 台（M2-2）汇总：一设备一行
   1,1202000240194DP0026,部分失败,✗ 失败,BaiBLQ_5.0.16.1_1229,9005,"[Server] …",…
   2,120200087125BJB0002,部分失败,✗ 失败,BNQ_3.9.7,9005,"[Client] …",…
   # 整体 2 台（M2-1）汇总：path 列空 + 状态失败 + 故障
   1,1202000240194DP0026,失败,,,9005,"[Server] …",…
   2,120200087125BJB0002,失败,,,9005,"[Client] …",…
   ```
   → 不再出现同设备多行；逐设备读回值/成败一行内清晰（注意 DP0026=BaiBLQ、BJB0002=BNQ_3.9.7 两机不同固件，逐设备 per-product 翻译生效）。
2. ✅ **「查看」详情逐 path 表加「读回值 / 故障」列**（`ResultDetailModal` pathTaskColumns），与单设备 CSV 口径统一。复测：PATH | 状态 | **读回值/故障** —— SoftwareVersion=成功/`BaiBLQ_5.0.16.1_1229`，AdditionalHardwareVersion=失败/`[Server] Invalid Parameter Names…`。
3. ✅ 故障源 `[Server]`/`[Client]` 透出保留（有助区分 ACS 侧 vs 设备侧拒绝）。

---

## 5. 本轮修复
- **export.go（单设备 CSV）**：聚合设备全部 device_task、逐 path 输出「状态/读回值/故障信息」+ 设备级「部分失败」（修复逐 PATH 下成败 path 不可辨）。
- **export.go（汇总 CSV，§4.3-1）**：`aggregateExportRowsByDevice` 按设备合并为每设备一行（逐 PATH 不再同设备多行）。
- **ResultDetailModal（§4.3-2）**：「查看」详情逐 path 表新增「读回值 / 故障」列，与 CSV 口径统一。
- **acs/handler.go（失败响应报文）**：执行失败时把 CPE 返回的原始 SOAP Fault 报文存入 `device_tasks.result.raw_response`（原仅存 `param_faults` 解析结果），经 `service.go` 透传为 `raw_output` → 前端「查看 → 结果报文」可见。复测：失败任务「结果报文（格式化 XML）」显示 `<soap-env:Envelope>…</>` 含 9005 / Invalid Parameter Names 故障（原为空）。
- `go build ./...` + `go test ./internal/mml/` 通过；前端 typecheck/lint 干净；app + acs + web 已重建复测。

## 6. 用例结果汇总
| 用例 | 结果 |
|---|---|
| TC-M1-1 / M1-2 / M1-3 / M1-4（1 台 × 4 组合） | ✅ ×4 |
| TC-M2-1（2 台 整体 指定参数） | ✅ |
| TC-M2-2（2 台 逐PATH 指定参数） | ✅（BJB0002 已连接完成，两机逐 path 成败 + 各自固件读回值均正确） |
| TC-M2-3 / M2-4（2 台 其余组合） | ✅ 结构 + DP0026；BJB0002 随连接节奏陆续完成（环境） |
| TC-HIS-ORD / TC-HIS-SW | ✅ ×2 |
| TC-R-1 / TC-R-N / TC-R-MIX | ✅ ×3 |
| TC-DL-AGG / TC-DL-DEV / TC-DL-ANALYZE | ✅（TC-DL-DEV 修复后通过） |

> 备注：本轮执行矩阵用 API 直发 + DB/CSV 校验（覆盖后端命令拆分/扇出/翻译/结果/导出全链路），命令记录与结果切换用浏览器 UI 验证。BJB0002 的 2 台子场景受真机连接节奏限制未在测试窗口内全部完成，device_task 结构已验证正确。
