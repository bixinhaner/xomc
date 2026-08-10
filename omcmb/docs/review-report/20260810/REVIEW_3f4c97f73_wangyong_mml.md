# Issue #272 MML 多路径 MOD 回读结果修复审查报告

- 审查日期：2026-08-10
- 分支：`fix/272-mml-readback-results`
- 基线：`3f4c97f73`
- 审查范围：MML Console MOD/LST 复合结果聚合
- 结论：PASS

## 变更摘要

- 将同一设备的全部 LST 子任务纳入回读结果聚合，避免逐 PATH 执行多组 MOD/LST 时只保留第一条 LST。
- 回读明细保留各 LST 子任务自身的任务 ID、状态与起止时间，并按全部 LST 计算响应时间和耗时。
- 新增 Issue #272 回归测试，覆盖三个 ManagementServer 参数各自 MOD 后 LST 的真实任务排列。

## 审查结果

### CRITICAL

无。

### WARNING

无。

### INFO

- 改动仅影响前端 MML Console 的结果适配，不改变接口、数据库结构或设备下发流程。
- 聚合仍按设备维度执行，并兼容单条 LST 和一次 LST 返回多个参数的既有场景。
- 未引入 `any`、HTML 注入、Token 处理或运营商硬编码。

## 验证记录

- 回归测试先复现失败：三组 MOD/LST 的结果状态由预期 `success` 错误显示为 `mismatch`。
- `npm test -- --run src/pages/mml/Console/__tests__/buildMODReadbackRows.test.ts`：通过，7/7。
- `npm run typecheck`（`omcmb/webcode`）：通过。
- `npm run build`（`omcmb/webcode`）：通过。
- ESLint（本次两个 TypeScript 文件）：通过。
- `git diff --check`：通过。
- 本地 Docker web 镜像已重建并热重启，`http://localhost:8081/login` 返回 HTTP 200；浏览器刷新无控制台错误。

## 风险与回滚

- 风险集中在多 LST 的聚合顺序与展示元数据；新增测试覆盖全部回读值及每条 LST 子任务关联。
- 可直接回滚本次提交，不涉及数据迁移和外部接口兼容。
