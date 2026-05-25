# MML 命令缺失 Path 排查报告

| 项 | 值 |
|---|---|
| 日期 | 2026-05-25 |
| 检查范围 | `mml_commands` 全表 1002 行（standard=524 / extension=478） |
| 检查依据 | `target_paths` (JSONB) 与 `target_object` (varchar) 两个 Path 承载字段 |
| 关联 schema | `omcgo/internal/mml/model.go:MMLCommand` |
| 关联代码 | `omcgo/internal/mml/console_executor.go:buildStatementCommandEntry` |

---

## 0. TL;DR

| 类别 | 数量 | 结论 |
|---|---|---|
| 真正完全缺失 Path 的命令（`target_paths=[]` AND `target_object` 空） | **0** | 不存在 |
| ADD/RMV schema 不一致：同时填 `target_paths` 与 `target_object`（冗余存储） | 23 | **正常但需统一**：低优先治理 |
| ADD/RMV 用 `target_paths` 而非 `target_object` 承载父对象（Pattern B） | 48+48=96 | **正常**：schema 允许双载体，但建议统一 |
| ADD/RMV 用 `target_object` 而 `target_paths=[]`（Pattern A） | 28+28=56 | **正常**：经典约定 |
| **MOD standard 命令未挂任何 `mml_command_sub_fields`（UI 不可写）** | **57** | **异常 / 高优先级修复** |
| LST 命令未挂任何 sub_field | 0 | **正常**（LST 走 param_refs 直接 SOAP，但本 DB 全部 LST 命令均挂有 sub_field，无异常） |
| ALL ADD/RMV 命令无 sub_field（共 198 行） | 198 | **正常**：ADD/RMV 不需要按字段选择，由 `target_object` 单独驱动 |
| `group_id IS NULL`（无 chapter 归属） | **224 行（22%）** | **异常 / 中优先级**：UI 树渲染不可见 |
| `logical_code` 为空字符串 + `command_code` 末尾带空格（"`MOD `" / "`LST `"） | 2 | **异常 / 高优先级**：catalog 解析器 bug 残留 |

**核心结论**：**严格意义上"缺失 Path"的命令 = 0**。用户给出的示例 "添加 自配置启动状态" 经核实**已携带 Path**（`target_object = "Device.Services.FAPService.{i}.FAPControl.SelfConfig."`），其实际问题是**3 行 redundant 重复**（见配套《MML命令去重与清洗日志.md》Case A）。

但本次扫描发现了 4 类**"虽有 Path 但形态异常"**的命令需要治理，详见 §2-§5。

---

## 1. 正常无 Path 行为的情况

### 1.1 ADD / RMV 命令的 Path 形态约定

依据 `console_executor.go:buildStatementCommandEntry` 的 ADD/RMV 分支：

```go
case "ADD":
    targetObject := strings.TrimSpace(cmd.TargetObject)
    if targetObject == "" {
        return nil, fmt.Errorf("ADD: command %s missing target_object", cmd.CommandCode)
    }
    ...
    entry["rpc_method"] = "AddObject"
    entry["parameters"] = map[string]interface{}{"object_name": targetObject}
```

**正常约定**：
- ADD/RMV 操作的对象路径是 *父级容器* 而非具体字段，因此 `target_paths` 留空是设计预期。
- 代码层使用 `target_object` 字段承载该父级路径（`Device.X.{i}.Y.`，末尾带点表示 partial path / parent collection）。

**实际数据情况**：99 条 ADD + 99 条 RMV = 共 198 条命令，**全部不依赖 `target_paths`**，符合设计预期。其中：
- 28 条 ADD + 28 条 RMV **只填 `target_object`，`target_paths=[]`** — 经典 Pattern A，完全规范。
- 详见 §3 关于 Pattern B/C 的异常说明。

### 1.2 ALL ADD/RMV 命令无 `mml_command_sub_fields` 关联

- ADD：99/99 行无 sub_field
- RMV：99/99 行无 sub_field

**正常**：ADD/RMV 不需要按字段选择。前端 UI 在 MML 控制台 ADD/RMV 命令面板中不渲染 sub_field 列表，只让用户填 instance index（RMV）或 SPV value（ADD 复合）。

代码佐证：`internal/mml/console_executor.go:343` LST 分支只对 LST 走 `buildLSTParamRefs(stmt.SelectedSubFieldIDs, subFields, cmd.Params)`，ADD/RMV 分支不读 sub_fields。

### 1.3 完全缺失 Path 的命令：0 条

```sql
SELECT COUNT(*) FROM mml_commands
WHERE jsonb_array_length(target_paths) = 0
  AND (target_object IS NULL OR target_object = '');
-- = 0
```

**结论**：**整个 catalog 不存在"既无 target_paths 也无 target_object"的真空命令**。所有命令均有路径承载。

---

## 2. 异常类型 1（高优）：MOD standard 命令未挂 sub_field（UI 不可写）

### 2.1 现象

| op_type | source | 总数 | 无 sub_field | 占比 |
|---|---|---|---|---|
| LST | standard | 182 | 0 | 0% |
| LST | extension | 272 | 0 | 0% |
| MOD | standard | 144 | **57** | **40%** |
| MOD | extension | 206 | 0 | 0% |
| ADD | standard | 99 | 99 | 100% （正常，见 §1.2） |
| RMV | standard | 99 | 99 | 100% （正常，见 §1.2） |

**57 条 MOD standard 命令的 `target_paths` 完全有效**（path_cnt 介于 1~22），但没有任何 `mml_command_sub_fields` 行关联到它们。

代码层面（`console_executor.go:354`）：MOD 操作必须经 `stmt.Values` 携带至少 1 条 user-input value；而 `stmt.Values` 的 key 来自前端勾选的 sub_field。**0 sub_field = 用户无法在 UI 上选任何字段填值 = 该 MOD 命令在控制台上完全不可用**。

### 2.2 命令清单（前 20 条）

| command_code | command_name | path_cnt |
|---|---|---|
| MOD_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_ConnMode_EUTRA_A1MeasureCtrl_i | MOD A1 测量控制 | 10 |
| MOD_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_ConnMode_EUTRA_A2MeasureCtrl_i | MOD A2 测量控制 | 10 |
| MOD_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_ConnMode_EUTRA_A3MeasureCtrl_i | MOD A3 测量控制 | 10 |
| MOD_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_ConnMode_EUTRA_A4MeasureCtrl_i | MOD A4 测量控制 | 10 |
| MOD_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_ConnMode_EUTRA_A5MeasureCtrl_i | MOD A5 测量控制 | 12 |
| MOD_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_ConnMode_IRAT_B1MeasureCtrl_i | MOD B1 测量控制 | 10 |
| MOD_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_ConnMode_IRAT_B2MeasureCtrl_i | MOD B2 测量控制 | 12 |
| MOD_Device_Services_FAPService_i_CellConfig_LTE_RAN_MAC_DrxInitialParam_i | MOD DRX 初始 | 6 |
| MOD_Device_Services_FAPService_i_CellConfig_LTE_EPC | MOD EPC | 2 |
| MOD_Device_DeviceInfo_MU_i_Slot_i_EU_i | MOD EU 扩展单元 | 2 |
| MOD_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_ConnMode_EUTRA | MOD EUTRA 测量 | 1 |
| MOD_Device_Services_FAPService_i | MOD FAPService 载波 | 22 |
| MOD_Device_Services_FAPService_i_CellConfig_LTE_RAN_Mobility_IdleMode_IRAT_GERAN_GERANFreqGroup_i | MOD GERAN 频组 | 6 |
| MOD_Device_Services_FAPService_i_CellConfig_LTE_RAN_NeighborList_InterRATCell_GSM_i | MOD GSM 邻区 | 7 |
| MOD_Device_Ethernet_IpRoute_i | MOD IP 路由 | 5 |
| MOD_Device_IPsec | MOD IPsec | 2 |
| MOD_Device_Ethernet_Interface_i_IPv4Address_i | MOD IPv4 地址 | 5 |
| MOD_Device_Ethernet_Interface_i_VlanInterface_i_IPv4Address_i | MOD IPv4 地址 | 5 |
| MOD_Device_Ethernet_Interface_i_VlanInterface_i_IPv6Address_i | MOD IPv6 地址 | 5 |
| MOD_Device_Ethernet_Interface_i_IPv6Address_i | MOD IPv6 地址 | 5 |

剩余 37 条结构相似（均为新 `LST_` 风格 logical_code + 完整 standard_path 派生形态）。

10/57 同时也是重复簇成员（其他 47 条是孤立行）。

### 2.3 根因分析

这 57 条 MOD 命令的 `command_code` 形态特征为 `MOD_<完整 path>_格式`（用下划线代替点），与传统 `MOD <语义缩写>` 形态（如 `MOD MR_MGMT_CONFIG`）不同。说明它们是由**spec parser 的不同代码路径**派生的：

- 老路径：`MOD <chapter_code>` → 同时 INSERT 命令行 + sub_field 行（migration `000111_mml_old_catalog_import.sql` 等老种子文件）
- 新路径：`MOD_<full_path_underscored>` → 由 spec md parser 派生，**未同步生成 sub_field**

关联到 T-0123 / T-0170 修复 sub_field/standard_path_id 链路时，遗漏了对这批新派生 MOD 命令的 sub_field 回填。

### 2.4 建议方案

**方案 A（推荐）**：补 sub_field

写 migration 扫描这 57 条 MOD 命令的 `target_paths`，对每条 standardPath 在 `standard_params` 查找对应 `id`，回填 `mml_command_sub_fields(command_id, standard_path_id, mml_code, ...)`。`mml_code` 取 standardPath 末段（同 T-0171 humanize 思路）。

预估：57 命令 × 平均 7 path = 约 400 条 sub_field 行需要补。

**方案 B**：声明废弃

若业务方判定这些 MOD 命令与 §3 中已有同名命令重复（参见配套《MML命令去重与清洗日志》Case A 簇），直接 DELETE 这 57 条。

**方案 C**：UI 兜底

修 console UI 当某 MOD 命令的 sub_field 为空时，回退到展示 `target_paths` 列表让用户勾选；前端 `useStructuredExecute` Hook 直接用 path 作 key（绕开 sub_field 反查）。但这违背 T-0098 ParamModel 字典化路线，**不推荐**。

---

## 3. 异常类型 2（中优）：ADD/RMV schema 双载体存储不一致

### 3.1 现象

ADD/RMV 命令的"父对象路径"在 DB 中存在 3 种存储模式：

| Pattern | `target_object` | `target_paths` | ADD 行数 | RMV 行数 | 评估 |
|---|---|---|---|---|---|
| A（target_object only） | `"Device.X.{i}.Y."` | `[]` | 28 | 28 | 经典约定（与 model 注释一致） |
| B（target_paths only） | `""` | `["Device.X.{i}.Y."]` 或 `["Device.X.{i}.Y.{i}.*"]` | 48 | 48 | 非典型但 SOAP 层能解 |
| C（双载体） | `"Device.X.{i}.Y."` | `["Device.X.{i}.Y."]` | 23 | 23 | 冗余存储 |

### 3.2 Pattern C 案例（10 条）

```
RMV LTE_MME_POOL_CONFIG_PARAM
  target_object  = Device.Services.FAPControl.LTE.MmePoolConfigParam.
  target_paths   = ["Device.Services.FAPControl.LTE.MmePoolConfigParam."]

RMV FAP_MR_MGMT_CONFIG
  target_object  = Device.FAP.MRMgmt.Config.
  target_paths   = ["Device.FAP.MRMgmt.Config."]
...
```

### 3.3 根因分析

`buildStatementCommandEntry` 的 ADD/RMV 分支**只读 `cmd.TargetObject`**，不读 `target_paths`。Pattern B 行根本不能用于 ADD/RMV 执行——执行时会触发 `ADD: command X missing target_object` 错误（见 `console_executor.go:373`）。

实际上这 48+48 个 Pattern B 命令的 `target_paths` 内容已 *暗示* 该有 target_object，但由于种子文件历史版本不同步、parser 派生策略变化等原因，target_object 字段从未填值。

### 3.4 建议方案

**方案 A（推荐）**：统一为 Pattern A（target_object only）

写 migration：
```sql
UPDATE mml_commands
   SET target_object = target_paths->>0,
       target_paths  = '[]'::jsonb
 WHERE operation_type IN ('ADD','RMV')
   AND (target_object IS NULL OR target_object = '')
   AND jsonb_array_length(target_paths) > 0;
-- 影响 48+48 = 96 行（Pattern B → A）

UPDATE mml_commands
   SET target_paths = '[]'::jsonb
 WHERE operation_type IN ('ADD','RMV')
   AND target_object IS NOT NULL AND target_object <> ''
   AND jsonb_array_length(target_paths) > 0;
-- 影响 23+23 = 46 行（Pattern C → A）
```

迁移完后所有 ADD/RMV 行统一 Pattern A。

**预期收益**：消除 schema 双义性；下游代码（fanouter / executor / TR-069 payload builder）只需读 `target_object` 一处，逻辑简化。

**风险**：现有 Pattern B 的 48 ADD + 48 RMV 命令在 console 上**当前已不能执行**（target_object 为空触发 `console_executor.go:373` 报错）。统一为 Pattern A 后立即可用 → **是 bug fix，不是破坏性变更**。

---

## 4. 异常类型 3（中优）：224 行 `group_id IS NULL`（无 chapter 归属）

### 4.1 现象

```sql
SELECT COUNT(*) FILTER (WHERE group_id IS NULL) FROM mml_commands;
-- 224 (22% of 1002)
```

无 group_id 的命令在 MML console 命令树中**无法渲染**（树按 chapter 分组），相当于在 UI 上失访。

### 4.2 关联类别

- T-0171 之前，extension catalog 完全没有 group_id（孤儿 path 补偿产物 478 行）— 经 migration `000175_mml_catalog_extension_chapter_merge_t0171.sql` 已映射到 4 个汇总 chapter。
- 当前剩余 224 NULL 主要落在 standard catalog 内（spec parser 派生的 524 行中的一部分）。

### 4.3 根因与建议

需进一步排查（超出本报告范围）：
- 写 SQL：`SELECT source, COUNT(*) FROM mml_commands WHERE group_id IS NULL GROUP BY source;` 看分布
- 若全部落在 standard，则是 spec parser 路径未与 chapter table 关联，需补 chapter 归属表
- 若 extension 仍有，则 T-0171 step 2 反查策略遗漏（应当 `target_paths->>0` 取首段）

**建议**：本期暂不动；后续 T-0173 专项治理 + 复用 T-0171 chapter 合并经验。

---

## 5. 异常类型 4（高优）：2 行 `logical_code` 为空 + `command_code` 末尾空格

### 5.1 现象

```
id                                   | command_code | command_name           | operation_type | logical_code | path_cnt
433c15c6-2420-46de-bbdb-5534914b1bd8 | LST          | 列出 FAP 载波基本配置  | LST            |              | 34
9fa45a43-ccff-4935-9627-c5f680bb4333 | MOD          | 修改 FAP 载波基本配置  | MOD            |              | 32
```

`command_code` 字段值为 `"LST "` / `"MOD "`（末尾带空格，与 UNIQUE 约束不冲突）。

### 5.2 根因

catalog parser 派生 `command_code` 时，当 chapter 的 logical part 为空字符串时拼接出 `"LST "` 或 `"MOD "`，未校验拼接结果合法性。

### 5.3 影响

- 前端命令树渲染时显示 `"LST "` / `"MOD "` —— 体验差
- 它们是 §2 §3 §6 提及"修改 FAP 载波基本配置"重复簇的成员，进 Case A 合并目标

### 5.4 建议

**方案**：在配套《MML命令去重与清洗日志》Case A 合并时，**优先丢弃这 2 行**，保留同簇内 `logical_code` 非空的 canonical 行。无需独立 migration。

---

## 6. 用户原例 "添加 自配置启动状态" 的实际状态

| id | command_code | command_name | logical_code | target_object | path_cnt |
|---|---|---|---|---|---|
| da05cc9a-... | ADD FAP_SERVICE_FAP_CONTROL_SELF_CONFIG | 增加 自配置启动状态 | FAP_SERVICE_FAP_CONTROL_SELF_CONFIG | Device.Services.FAPService.{i}.FAPControl.SelfConfig. | 1（target_paths 也含 1 条同路径，**Pattern C**） |
| 1c3334d8-... | ADD SELF_CONFIG | 增加 自配置启动状态 | SELF_CONFIG | Device.Services.FAPService.{i}.FAPControl.SelfConfig. | 1（Pattern C） |

**判定**：用户认为"添加 自配置启动状态"没有 Path，是误读。该命令**已携带 Path**（`target_object`），实际问题是同名命令存在 2 行重复（属于 Case A 冗余）。本报告 §3 Pattern C 治理 + 配套《MML命令去重与清洗日志》Case A 合并将一并解决。

另：该命令簇还存在 1 行 `command_name = "LST 自配置启动"` 的异常行（label 与同语义其他行不一致 —— 用了原始英文 logical_code 当 label），见配套《MML命令去重与清洗日志》§Label 异常清单。

---

## 7. 治理优先级建议

| 优先级 | 异常类型 | 影响 | 建议行动 |
|---|---|---|---|
| **P0** | §2 — 57 MOD standard 0 sub_field | 命令在 UI 完全不可用 | 写 migration 回填 sub_fields |
| **P0** | §5 — 2 行空 logical_code | UI 渲染异常 | Case A 合并时丢弃 |
| **P1** | §3 — ADD/RMV schema 不一致 | Pattern B 命令完全不可执行 | 写 migration 统一为 Pattern A |
| **P2** | §4 — 224 行 NULL group_id | UI 不可见 | 后续 T-0173 专项治理 |

---

## 8. 配套文档

- 《MML命令去重与清洗日志.md》(同目录)— Step 1/3/4 重复命令的去重策略
- `docs/project/prd/F06-mml-catalog-orphan-compensation.md`（T-0171 上下文）
- `omcgo/internal/mml/model.go`（schema 定义）
- `omcgo/internal/mml/console_executor.go`（ADD/RMV/MOD 执行约定）

---

**版本历史**：2026-05-25 v1.0 — 首次扫描 + 报告
