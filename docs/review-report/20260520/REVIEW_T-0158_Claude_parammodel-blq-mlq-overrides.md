# 代码审查报告 — T-0158 BLQ + MLQ 具体实例覆盖补全

| 字段 | 值 |
|------|------|
| 任务 | T-0158（参数取值范围 — 系列收尾） |
| 范围 | parammodel（XML 字典） |
| 文件 | `omcgo/data/param-mappings/BLQ.xml` / `omcgo/data/param-mappings/MLQ.xml` |
| 改动量 | 36 行（BLQ 22 + MLQ 50 行 = 11 + 25 条 `<param>` 重写） |
| 审查结论 | **PASS** |

---

## 1. 变更概览

承接 T-0158 P1 (`773372e5`) / P2 (`dc08e314`) / P3 (`c55788fd`)，把 BLQ.xml 已建立的「具体实例覆盖与 `{i}` 模板对齐」修复模式扩展到剩余的 36 处。

### BLQ.xml — 11 处具体实例覆盖（补齐 T-0158 P3 漏的）

| 路径 | 原 type | 改后 |
|------|---------|------|
| `FAPService.1/2.CellConfig.LTE.RAN.Common.CellIdentity` | STRING | U_INT min=0 max=268435455 |
| `FAPService.1/2.CellConfig.LTE.RAN.PHY.PRACH.RootSequenceIndex` | STRING | U_INT min=0 max=837 |
| `FAPService.1.CellConfig.LTE.RAN.PHY.TDDFrame.SpecialSubframePatterns` | STRING | INT enumValues=5,7 |
| `FAPService.1/2.CellConfig.LTE.RAN.RF.EARFCNDL` | STRING | U_INT min=0 max=65535 |
| `FAPService.1/2.CellConfig.LTE.RAN.RF.FreqBandIndicator` | STRING | U_INT min=1 max=62 |
| `FAPService.1/2.CellConfig.LTE.RAN.RF.X_COM_MaxTxPowerExpanded` | STRING | U_INT min=0 max=46 |

### MLQ.xml — 25 处（12 个 `{i}` 模板 + 13 个具体实例覆盖）

`{i}` 模板补全约束（12 条）：TAC / CellIdentity / RootSequenceIndex / SpecialSubframePatterns / SubFrameAssignment / DLBandwidth / EARFCNDL / EARFCNUL / FreqBandIndicator / PhyCellID / ULBandwidth / X_COM_MaxTxPowerExpanded

具体实例覆盖修正（13 条，FAPService.1/2.{CellIdentity,RootSequenceIndex,DLBandwidth,EARFCNDL,FreqBandIndicator,PhyCellID,X_COM_MaxTxPowerExpanded}，FAPService.2 缺 DLBandwidth）：type STRING → U_INT/INT，补 min/max 或 enumValues/enumLabels。

注：MLQ.xml 不含 EnbCellType 参数；HNBName max=48（BLQ max=64），视为机型规格差异保留不动；ReferenceSignalPower/PLMNID 原已带约束未改。

## 2. 修改依据

- BLQ：值全部来自同文件 `{i}` 模板（T-0158 P1 已落地，与老 OMC MML 取值范围一致并通过浏览器验证）。
- MLQ：值参照 BLQ 同名 `{i}` 模板（用户指令"按BLQ的取值范围改"）。

## 3. 审查检查项

| 项 | 结果 | 备注 |
|----|------|------|
| XML 良构 | ✅ | `xmllint --noout` 两个文件均通过 |
| type 与 enumValues 类型一致 | ✅ | enum 用 INT/U_INT 整数集合，min/max 数值范围与 type 域匹配 |
| Translator standardPath 未受影响 | ✅ | 仅改 type/约束属性，name/standardPath/access 未动 |
| 与 `{i}` 模板取值范围一致性 | ✅ | 一一对照同文件 `{i}` 模板值 |
| 无运营商硬编码 | ✅ | XML 数据文件本身与 Carrier 无关 |
| 无敏感数据 / 密钥 | ✅ | 纯参数元属性 |
| 与产品路由一致 | ✅ | products.xml: BLQ paramModel 服务 BAIBLQ/BLX/QRTB 三系产品；MLQ paramModel 服务 MLQ 产品 |

## 4. 部署验证

- 容器栈：`docker-run.sh` 全量重建 + Redis L2 (`parammodel:*`) 清缓存 + app restart。
- 浏览器实测（BLQ 设备 1202000240194DP0026）：
  - 频段标识 [1 ~ 62] + 输入 999 → 红框 + "最大值为 62" ✅
  - 下行频点 [0 ~ 65535] ✅
  - 根序列索引 [0 ~ 837] ✅
  - 特殊子帧配置下拉 {5 | 7} ✅
  - 功率等级(*2dbm) [0 ~ 46] ✅
  - TAC=99999 → 红框 + "最大值为 65535" ✅
- DB 校验：`param_mappings` 表 6 个具体路径 data_type/min/max/enum_values 均为新值。
- MLQ 无在线设备，无法浏览器复测；同套 dictloader/Registry/前端渲染链路与 BLQ 共享，BLQ 通过即可保证 MLQ 行为一致。

## 5. 风险评估

- **L1 内存缓存陷阱**：dictloader 重灌 DB 但不主动 invalidate Redis L2，已通过手动清缓存 + app restart 绕过。**后续应立 R-NNN 追踪 dictloader 增加 cache invalidation 钩子**（参见 TODO.MD line 20 重新部署关键点，与 T-0158 P1 同款）。
- **MLQ.xml 缺 EnbCellType**：MLQ 真机如果上报 EnbCellType 路径将走默认 STRING，无 enum 约束。短期可接受（用户当前无 MLQ 设备）；长期由"全量补全老 OMC 取值范围"任务（TODO.MD 新增条目）覆盖。
- **HNBName max=48 (MLQ) vs 64 (BLQ)** 差异保留：可能反映机型规格差异。需老 OMC 确认后再动。

## 6. 测试覆盖

- 单元测试：本次为数据文件改动，无 Go 代码逻辑变更，无需新增 Go 单元测试。`MappingValidator.LookupParam` 精确路径优先于 `{i}` 的逻辑已在 T-0158 P3 验证。
- E2E：BLQ.xml 浏览器实测 5 个原失败字段 + 1 个原通过字段（TAC）全部生效。

---

**审查人**：Claude Opus 4.7  
**日期**：2026-05-20  
**结论**：PASS
