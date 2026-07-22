# Review: MML MOCN/MME grouping SQL

日期：2026-07-22

结论：PASS

## 范围

- `migrations/seed/000001_init_seed.sql`
- `scripts/mml_apply_config_updates_20260721.sql`

## 检查项

- MOCN 配置命令直接归属 `小区服务参数管理（总体）`，不新增子分组。
- MOCN 查询/修改/添加/删除命令使用唯一 `command_code`，并通过 `ON CONFLICT` 保持幂等。
- MOCN 字段绑定按 `Device.Services.FAPService.{i}.CellConfig.LTE.MocnConfigParam.%` 精确前缀匹配。
- MME 池配置补充绑定 `CellConfig.LTE.MmePoolConfigParam.{i}.%`，并隐藏重复无字段的旧添加命令。
- `target_paths` 与 `tree_node_refs` 在字段绑定后刷新，避免树节点和 PATH 列表不一致。

## 发现

无 CRITICAL / WARNING。

## 验证

- `go build ./...`：通过
- `go test ./...`：通过
- `go test ./test/integration -run TestSeedBaselineHasOnConflict -v`：通过，118 条 `INSERT INTO public.*` 均带 `ON CONFLICT`
- 本地部署 SQL 已执行：MOCN 4 条命令、32 条字段绑定、4 条命令路径刷新成功
- 本地热部署：`http://localhost:8081/` 返回 `HTTP/1.1 200 OK`

## 风险

- 本次为 seed/部署 SQL 数据配置变更，不涉及 Go 运行时代码和接口协议。
- 回滚可通过隐藏新增命令或恢复对应 seed/部署 SQL 片段处理。
