# 代码审查报告 — T-0159 paramModel 交叉镜像约束 mirrorWith

| 字段 | 值 |
|------|------|
| 任务 | T-0159（交叉镜像约束 — TDD 上下行带宽同步） |
| 范围 | parammodel + device + frontend |
| 改动量 | 15 文件（1 migration / 7 XML / 4 Go / 3 TS） |
| 审查结论 | **PASS** |

---

## 1. 变更概览

引入声明式交叉镜像约束：一个参数的 `mirrorWith` 属性指向其镜像字段（完整 standardPath，含 `{i}`），前端在快速设置页改 A 字段时自动同步写 B 字段。典型场景：TDD-LTE 上下行带宽必须相等。

### 后端

- **Migration**: `000130_param_mappings_mirror.sql` — 给 `param_mappings` 和 `discovered_param_mappings` 加 `mirror_with VARCHAR(256)` 列
- **XML 解析**: `xmlParamEntry.MirrorWith xml:"mirrorWith,attr"`
- **领域模型**: `ParamMapping.MirrorWith *string`
- **持久层**: loader INSERT + pg_repository SELECT/Scan（默认 + discovered 双路）
- **API**: `Constraints.MirrorWith *string json:"mirror_with,omitempty"` 由 `constraintsFromMapping` 透传

### 前端

- **类型**: `ParameterConstraints.mirrorWith?: string` + `BackendConstraints.mirror_with?` + mapper 透传
- **同步逻辑**: `CellParameterForm.tsx` 加 `paramNameByPath` 反查 + `onValuesChange` 中按 `constraints.mirrorWith` 解析对端字段并 `form.setFieldValue` + 同步 draft + 重新校验对端 fieldError
- **UI 优化**: `formatConstraintHint` 对枚举字段返回 `''`（Select 下拉框本身已展示候选项，移除 label 后的 `{...}` 冗余占位）

### XML 数据

7 个 paramModel 共 28 处 `mirrorWith=` 属性：

| 文件 | 处数 | 路径 |
|------|------|------|
| BLQ.xml | 3 | `{i}.DLBandwidth` / `{i}.ULBandwidth` / `FAPService.1.DLBandwidth` |
| MLQ.xml | 3 | 同上 |
| BaiBNQ.xml | 2 | 5G NR `{i}.DLBandwidth` / `{i}.ULBandwidth` |
| BM.xml | 11 | `{i}.DL/UL` + FAPService.1-9.DLBandwidth |
| MLN.xml | 5 | `{i}.DL/UL` + FAPService.1-3.DLBandwidth |
| ENB_DEFAULT_098.xml | 2 | `{i}.DL/UL` |
| ENB_DEFAULT_181.xml | 2 | 同上 |

BSC/BTS（2G GSM）跳过 — 无 LTE 带宽概念。

## 2. 设计决策

| 决策 | 理由 |
|------|------|
| `mirrorWith` 取完整 standardPath 含 `{i}` | 跨容器字段无歧义；与 standardPath 同管道；Translator 已天然支持 |
| 声明对称（两端均填对方路径） | 解析无方向耦合；任一端 onChange 都能触发同步；无需 loader 推导反向链接 |
| 默认不做后端校验 | 符合"做减法"原则，前端 sync 保证 SPV 一致；若设备外部 API 收到不一致请求，由设备自行 reject |
| `form.setFieldValue` 不触发 `onValuesChange` | antd Form 行为，避免镜像循环递归无需额外保险 |
| 枚举 label 不再显示 `{a | b | c}` 占位文字 | Select 下拉框已展示候选项，UI 冗余 |

## 3. 审查检查项

| 项 | 结果 | 备注 |
|----|------|------|
| XML 良构 | ✅ | 7 个 paramModel `xmllint --noout` 全部通过 |
| Go 编译 | ✅ | `go build ./...` PASS |
| Go 单元测试 | ✅ | `go test ./internal/config/parammodel/...` PASS |
| 前端 typecheck | ✅ | `npm run typecheck` PASS |
| Migration 版本号连续 | ✅ | 000129 → 000130 |
| Migration Up/Down 配对 | ✅ | Down DROP COLUMN IF EXISTS 对应 Up ADD |
| 无运营商硬编码 | ✅ | mirrorWith 为通用机制，跨运营商共享 |
| 无敏感数据 | ✅ | 纯参数元属性 |
| 无 SQL 注入风险 | ✅ | squirrel 参数化（默认/discovered 两路均沿用既有模式） |
| API 响应字段命名一致 | ✅ | 后端 `mirror_with` snake_case，前端 mapper 转 camelCase `mirrorWith` |
| Frontend `any` 检查 | ✅ | 全部精确类型，无 `any` |
| 反向递归保险 | ✅ | `form.setFieldValue` 不触发 onValuesChange；显式比对 `current === value` 跳过；mirrorName === name 跳过 |

## 4. 部署验证（已完成）

| 测试 | 操作 | 结果 |
|------|------|------|
| 镜像同步 DL→UL | BLQ 设备 DLBandwidth = CELL_BW_50(10M) | UL 自动变 CELL_BW_50(10M) ✅ |
| 镜像同步 UL→DL | ULBandwidth = CELL_BW_100(20M) | DL 自动变 CELL_BW_100(20M) ✅ |
| 枚举 label 干净 | 下行/上行带宽 / 子帧配比 / 特殊子帧 / 小区类型 | label 不再含 `{...}` 占位 ✅ |
| 数值字段约束保留 | TAC / PCI / 频段标识 / 下行频点 / 根序列索引 / 功率等级 | `[min~max]` 提示正常 ✅ |
| DB 实际数据 | PG 查询 `mirror_with IS NOT NULL` | 28 行对应 7 个 XML 文件全部落库 ✅ |

## 5. 风险

- **Redis L2 缓存陷阱**：dictloader 重灌 DB 但不主动 invalidate Redis `parammodel:*`，本次部署手动清理后才生效。属于 T-0158 系列已知遗留问题，建议立 R-NNN 追 dictloader cache invalidation 钩子。
- **BaiBNQ 5G NR**：name 用 `1.CellConfig.{i}.NR` private path 但 standardPath 用 `{i}.CellConfig.LTE`，看起来是 XML 模板未完全适配 5G NR；mirrorWith 沿用 standardPath 形式（LTE 占位），需要业务方按 5G 实际带宽 list 后续覆写。
- **ENB_DEFAULT_181.xml 行 514** 有重复前缀 bad path（`Device.Services.FAPService.Device.Services.FAPService.{i}.CellConfig.LTE.RAN.RF.DLBandwidth`）— 本次未修，属于另一个数据清理问题。

## 6. Out of scope

- 后端 SetParameters handler 不校验 mirror（前端保证即可，设备做最后兜底）
- IntersectService 路径未补 mirrorWith propagate（T-0158 P2 已有相同遗留，dc08e314 commit 只修了 enum；当前设备 discovered_param_mappings 为空，影响面 0）
- BSC/BTS（2G）未覆盖（无带宽镜像需求）

---

**审查人**：Claude Opus 4.7  
**日期**：2026-05-20  
**结论**：PASS
