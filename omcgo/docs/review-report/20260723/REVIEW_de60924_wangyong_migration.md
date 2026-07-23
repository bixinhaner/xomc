# Review: MML seed NRCell active name conflict

## 结论

PASS

## 范围

- `migrations/seed/000001_init_seed.sql`

## 背景

干净环境执行 seed 时，`ADD NR_CELL` / `RMV NR_CELL` 插入复用了 `ADD SI_SUB_05` / `RMV SI_SUB_05` 的活跃 `command_name`，触发 `uniq_mml_commands_command_name_active` 唯一索引冲突。

## 检查项

- 迁移顺序：新增处理位于 `ADD NR_CELL` / `RMV NR_CELL` 插入之前，能在唯一约束检查前解除同名活跃行。
- 幂等性：`WHERE deprecated_at IS NULL` 确保重复执行不会反复改写已废弃行。
- 数据边界：仅影响 `ADD SI_SUB_05` / `RMV SI_SUB_05` 两个即将被对象级命令替代的零字段壳命令，不触碰 LST/MOD 命令和参数绑定。
- 回归风险：后续原有隐藏 ADD/RMV SI_SUB_* 的逻辑仍保留，本变更只是提前处理 NRCell 两条同名壳命令。

## Findings

- CRITICAL: 无
- WARNING: 无
- INFO: 无

## 验证

- `go test ./test/integration -run TestSeedBaselineHasOnConflict -count=1` — PASS
- `docker compose -p omcgo-seed-repro -f deployments/docker/docker-compose.test.yml up --build --abort-on-container-exit --exit-code-from migrate-seed-test migrate-seed-test` — PASS，干净库 schema + seed 迁移成功到 version 2

