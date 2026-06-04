# 代码审查报告 — UFTE 文件传输任务模板厂商对齐修复

| 项 | 值 |
|----|----|
| 审查 ID | 356c84e7 |
| 日期 | 2026-06-04 |
| 作者 | renpengfei |
| Scope | migration（种子数据 DML） |
| 激活专家 | 数据与存储专家 + 电信业务专家 |
| 变更文件 | `omcgo/migrations/seed/000001_init_seed.sql`（M）<br>`omcgo/migrations/seed/000020_fix_ufte_transfer_templates.sql`（A） |
| 结论 | **PASS_WITH_WARNINGS** |

---

## 1. 变更概述

修复 baseline 种子 `seed/000001`（277 迁移合并产物）在合并时丢失的"文件传输"任务类型（`ufte_task_types`）厂商对齐，重新合入原迁移 000139 / 000142 / 000143 / 000156 / 000157 的效果：

- **运行日志收集（RUNTIME_LOG_COLLECT）**：`file_type` 由 `'6'` 改为厂商标准串 `'4 Vendor Log File 1,2,3,4'`；`transport_path` 改为字面 `fileType=LOG` + `{taskId32}` + 空 `filename`。修复"运行日志收集卡 45% → 设备收到 `<FileType>6</FileType>` 不认"。
- **故障日志收集（FAULT_LOG_COLLECT）**：由 `UPLOAD` 改为 `SET_PARAM_VALUES` 模式（写 `Device.DeviceInfo.FaultLogURL` 触发 CPE 主动上传，落 MinIO 后由 `backup.file.received` 推进完成），step_chain 末步改 `WAIT_FILE_UPLOAD`。
- **配置备份（XML/NV）**：`transport_path` 末尾 `filename` 置空（设备自行命名 / ACS 按 sn+taskId 派生）。
- **模板排序**：恢复 10 类文件传输任务类型在 UI 的 `sort_order`（baseline 里全退回默认 100）。

**双轨设计**：`000001` 修正 baseline → 全新部署直接正确；`000020` 用 `UPDATE` 修存量库（`000001` 已 applied 不会重跑）→ 全新部署上 `000020` 为幂等 no-op。两份文件的最终目标值已逐字段比对一致（RUNTIME / FAULT / CONFIG_BACKUP_XML / CONFIG_BACKUP_NV / 10 类 sort_order）。

---

## 2. 检查项

### 数据与存储专家

- [x] **版本号连续递增**：seed 本地连续到 `000019`，新增 `000020` = max+1，远端无 `000020`（远端列表中的 `000190` 系 grep 误匹配文件名 `000019_..._c000190004.sql`）。本地无重复序号。✅
- [x] **goose 标记完整**：含 `-- +goose Up` 与 `-- +goose Down` 两段。✅
- [x] **StatementBegin/End**：全文均为单语句 `UPDATE`，无 `DO $$` / 函数 / 循环，无需包裹。✅
- [x] **幂等性**：`UPDATE ... WHERE type_code = ...` 天然幂等，可重复执行。✅
- [x] **Down 完整性**：Down 段把 RUNTIME / FAULT / CONFIG_BACKUP_XML / CONFIG_BACKUP_NV 四类回退为 baseline 合并后的原值，并将 10 类 sort_order 复位 100，与 Up 一一对应。✅
- [x] **未新增 schema 对象**：纯 DML，无表 / 索引 / 约束变更，无 Down 漏删风险。✅
- [x] **不破坏既有 seed（§5.5.11）**：未改表结构，仅改数据值；目标列（file_type / file_type_label / transport_path / rpc_type / step_chain / url_template / description / sort_order）均为既有列。✅

### 电信业务专家

- [x] **厂商对齐正确性**：`file_type='4 Vendor Log File 1,2,3,4'` 与设备侧 `<FileType>` 期望串一致，修复卡 45% 根因。✅
- [x] **故障日志 SPV 链路自洽**：rpc_type / step_chain（末步 `WAIT_FILE_UPLOAD`）/ url_template（`Device.DeviceInfo.FaultLogURL`）/ description 四者描述同一条 SPV 触发上传链路，内部一致。✅
- [x] **无运营商硬编码**：种子为任务类型字典，未在 SQL 内做 `carrier == 'cmcc'` 分支。✅

---

## 3. 发现

### WARNING

- **W1 — 改动已 applied 的 baseline 种子 `000001`**（§5.5.11 铁律 4）。
  - 风险：单看"改已 applied 种子"是反模式（存量库不会重跑 000001，改动不生效）。
  - 缓解：本次已配对 `000020` 用 `UPDATE` 覆盖存量库，且 `000001` 仅作"全新部署的正确 baseline"，迁移注释已显式说明此意图。`check-schema-drift.sh` 不做文件内容 checksum（仅 orphan / 跳号未跑 / CREATE TABLE 表不存在三类检查），改 000001 内容不会触发其告警。
  - 结论：**可接受**，设计意图正确且自洽，无需整改。

### INFO

- **I1 — `FAULT_LOG_COLLECT.url_template = 'Device.DeviceInfo.FaultLogURL'` 为厂商私有路径**。按团队约定（记忆 `feedback-ufte-spv-gpv-translator`），UFTE 下发 SPV/GPV 时一律 standardPath→privatePath 经 Translator 翻译再进 `cwmp:Name`。此处种子存的是任务模板配置值，运行时翻译由 dispatcher 代码负责；本次仅恢复原迁移 000156/000157 行为，未触碰翻译链路。**建议在真机联调时确认 dispatcher 对该 url_template 走 Translator**（非本次阻塞项）。

### CRITICAL

无。

---

## 4. 结论

DML 种子修复，版本号合规、goose 标记齐全、Up/Down 对称、双轨设计自洽、与 000001 目标值逐字段一致。1 项 WARNING（改 baseline 种子）已由配对的 000020 + 显式注释缓解，1 项 INFO（SPV 翻译链路）留真机联调确认。**PASS_WITH_WARNINGS，准予提交。**
