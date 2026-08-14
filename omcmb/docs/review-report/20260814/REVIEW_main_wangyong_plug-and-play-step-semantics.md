# 即插即用步骤语义与参数校验修复审查报告

- 日期：2026-08-14
- 基线：`main` / `2dd1c8670`
- 分支：`fix/plug-and-play-step-semantics`
- 范围：即插即用参数编译、任务技术制式、执行详情步骤语义及终态持久化
- 结论：`PASS_WITH_WARNINGS`

## 变更概述

- STRING 类型参数的 min/max 按 Unicode 字符长度校验，避免 GSM BSC `IpaUnitId` 被误判为整数。
- 任务列表和详情接口补充设备 `technology`，前端据此区分 NR 与 LTE/GSM 编排步骤。
- NR Stage 3 展示为设备上报的开站阶段，不再宣称 OMC 执行小区激活；结果步骤调整为“确认开站结果”和“记录开站成功”。
- LTE/GSM 展示真实的重启、上线、状态稳定、状态同步和开站结果校验流程。
- LTE/GSM 成功或失败终态持久化到准确步骤，避免完成任务仍停留在 `11/12` 或失败原因归属错误。

## 审查结果

### CRITICAL

无。

### WARNING

1. 仓库 `main` 当前存在两项与本次改动无关的全量测试基线失败：
   - `internal/alarm/definition`：内置告警定义期望 442，实际 443。
   - `internal/product`：BLN 参数模型 XML 元数据期望 380，实际 329。
   两项均已在干净 `HEAD 2dd1c8670` 临时 worktree 独立复现，本次未修改对应模块或数据文件。

### INFO

- 未新增接口、数据库迁移或权限逻辑。
- `technology` 为响应扩展字段，缺失时前端保留历史任务步骤名推断，兼容旧数据。
- PostgreSQL 查询使用既有 squirrel 构造器和参数绑定，无字符串拼接 SQL。

## 重点检查

- 后端任务查询列与 `scanEnrichedTask` 扫描顺序一致。
- STRING 长度校验与 `parammodel.MappingValidator` 口径一致，使用 UTF-8 rune 数量。
- NR、LTE、GSM 的原始状态名、展示别名和数字步骤对应关系一致。
- `activation_verified` 完成状态通过完整任务更新一次性持久化步骤、状态和结束时间。
- 前端 API 类型、列表映射、详情 Props 和 i18n 键完整贯通。

## 验证

- `cd omcgo && go build ./...`：通过。
- `cd omcgo && go test ./internal/provision -count=1`：通过。
- `cd omcgo && go test ./...`：本次相关包及 E2E/集成测试通过；存在上述两项 `main` 基线失败。
- `cd omcmb && npm run typecheck`：通过。
- 前端步骤、API、i18n 相关测试：3 个文件、16 个用例通过。
- 相关前端文件 ESLint：通过。
- `git diff --check`：通过。

## 最终结论

本次变更未发现阻断提交的 CRITICAL 问题。两项全量测试失败属于已确认的 `main` 基线问题，已在本报告和 MR 风险说明中保留记录。
