# Issue #69 Agent 对话体验与版本化 API 手册审查

结论: PASS_WITH_WARNINGS

## 范围

- 基线：`origin/main` (`bb450bd3`)
- 关联 Issue：#69「完善 Agent 对话体验与版本化 API 手册交付」
- 后端新增确定性构建的全量 API 手册包、manifest/chunk 读取接口及运行时版本校验。
- 前端新增共享自动滚动 Hook，并接入唯一保留的 V1 Agent 面板；支持主动滚离、回到最新、多行输入和宽屏回复布局。
- 归档 Agent 对话面板交互设计和效果图。
- 主线已删除 V2/V3 皮肤，本分支未恢复对应历史组件。

## 发现

- CRITICAL: 无。
- WARNING: 仓库级 `npm run lint` 仍有 4 个既有错误和 1650 个警告，错误位于本次未改动的 `runtimeClient.test.ts`、`mmlConsoleStore.ts`、`DeviceDetail/index.tsx` 和 `DeviceDetail/kpiSeries.ts`。本次改动文件的定向 ESLint 通过。
- INFO: 手册接口只提供内嵌、只读、大小受限并经过路径与摘要校验的归档内容，不扩展 Agent 对业务 API 的授权范围。
- INFO: 本地长期运行数据库来自旧迁移历史，不适合作为最新 consolidated baseline 的测试库；验证使用一次性全新数据库，完成后已删除。

## 验证

- `cd omcgo && go build ./...` 通过。
- 从空数据库执行 `go run ./cmd/migrate ... up`，19 个 schema 迁移全部通过。
- `TEST_PG_URL=<一次性测试库> go test ./...` 通过，包括 E2E 与 integration 测试。
- `cd omcmb && npm run typecheck` 通过。
- `npm run test --workspace webcode -- ../frontend-core/src/hooks/useAgentAutoScroll.test.ts` 通过，6 项测试全部成功。
- 本次前端改动文件定向 ESLint 通过。
- `git diff --check` 通过。

## 风险与影响

- 无数据库迁移，无既有 WebUI API 调用链变更。
- 新增两个只读 Agent 手册接口；接口受现有登录与 Agent 模块路由边界约束。
- 手册归档最大 8 MiB、解压最大 64 MiB、最多 5000 个文件，拒绝路径穿越、重复路径及非普通文件。
- 自动滚动只在用户位于底部时跟随；用户主动向上阅读时保持锚点，避免流式内容抢夺阅读位置。
