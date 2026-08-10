# MML 设备命令能力与多实例参数修复审查

## 结论

PASS_WITH_WARNINGS。未发现 CRITICAL 问题。

## 审查范围

- 按设备参数模型的显式对象映射过滤 ADD/RMV 命令。
- 补充 BM、BaiBNQ 的 WAN 对象和参数映射。
- 为多实例 ADD 继承可写 MOD 字段，并修正 X2、MME 等对象族。
- 全局停用 MR 参数管理 ADD/RMV，仅保留 LST/MOD。
- 去除对象路径展开结果中的结构性重复列。

## 发现

### WARNING

- `go test ./...` 的 `internal/task` 用例 `TestService_PG_GetTask_TerminalTombstoneReturnsDurableDetails` 失败；已在独立 `origin/main` worktree 上复现相同失败，判定为既有基线问题，与本次 MML 改动无关。

### INFO

- 历史 seed 为单文件合并迁移，已部署环境不会因文件内容变化自动重跑；本地现有环境已执行定向 SQL，全新空卷部署验证则完整覆盖了 schema/seed 初始化路径。
- MR ADD/RMV 使用 `deprecated_at` 停用，保留历史引用关系，避免直接删除命令记录破坏关联数据。

## 验证

- `go build ./...`：通过。
- `go test ./internal/config/parammodel ./internal/mml -count=1`：通过。
- `npm run typecheck`：通过。
- `npm test --workspace webcode -- --run src/pages/mml/Console/__tests__/objectPathResults.test.ts`：8/8 通过。
- 独立 Compose 项目空卷部署：schema、seed、TSDB 迁移均 exit 0；MR 有效命令仅 LST/MOD；UI HTTP 200。
- `git diff --check`：通过。

## 风险与回滚

- 设备是否显示 ADD/RMV 现在取决于显式 `{i}.` 对象映射；新增设备模型必须声明可增删对象。
- 回滚代码可恢复旧过滤逻辑，但已停用 MR 命令需按 command_code 清空 `deprecated_at` 才会重新显示。
