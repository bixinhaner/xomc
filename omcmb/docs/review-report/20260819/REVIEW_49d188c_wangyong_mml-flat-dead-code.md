# MML flat 前端死代码清理审查报告

- 审查结论：PASS
- 审查日期：2026-08-19
- 审查基线：`origin/main` (`49d188c`)
- 审查分支：`refactor/mml-remove-flat-frontend`
- Issue：N/A（清理仓库内无业务调用的前端遗留代码）

## 变更目标

删除 MML 前端未被业务页面调用的 `format=flat` 请求链路，避免过期注释、类型和 API 封装继续造成误导；保留现有页面实际使用的层级命令树链路。

## 审查范围

- 删除未被引用的 `useGroupTreeFlat` React Query hook。
- 删除仅由该 hook 调用的 `mmlApi.buildGroupTreeFlat` 方法。
- 删除 flat 响应专用类型、名称解析函数和 `object_path` type guard。
- 确认常规 `useGroupTree`、MML 用户页面以及后端 `format=flat` 接口均未修改。

## 审查结果

未发现 CRITICAL、WARNING 或 INFO 级问题。

关键结论：

1. 前端源码中已不存在被删除符号的引用。
2. MML 用户页面继续通过 `useGroupTree` 获取层级命令树，运行行为不变。
3. 后端 `GET /mml/group-tree?format=flat` 能力及其测试、smoke、文档不在本次清理范围内。
4. 变更仅删除不可达代码，不涉及用户可见文案、页面交互、接口合同或数据迁移。

## 风险与边界

- 如果未来重新启用 flat 响应，需根据当时的业务需求重新设计前端消费模型。
- 本次不代表后端 flat 接口可以立即下线；后端下线仍需独立确认外部调用和兼容策略。

## 验证记录

- `cd omcmb && npm run typecheck`：通过。
- `cd omcmb && npx eslint frontend-core/src/hooks/api/useMmlConsole.ts frontend-core/src/services/api/mmlApi.ts frontend-core/src/types/mmlConsole.ts`：通过。
- 删除符号全仓前端引用扫描：无残留。
- `git diff --staged --check`：通过。

## 结论

变更边界清晰，删除内容均无前端业务消费者，且不影响现有 MML 页面和后端接口，可提交进入 MR 审查。
