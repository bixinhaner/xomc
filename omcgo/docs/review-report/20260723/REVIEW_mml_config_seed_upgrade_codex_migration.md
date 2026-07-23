# Review: MML Config Seed Upgrade

Date: 2026-07-23
Author: Codex
Scope: migration
Result: PASS

## Summary

本次审查覆盖 MML 配置基线和增量脚本调整：

- 将参数映射默认值/校验规则字段并入 `000001_init_schema.sql`，删除单独的 `000003` 迁移文件。
- 在 `mml_apply_config_updates_20260721.sql` 增加 `ADD COLUMN IF NOT EXISTS`，保障老库未执行原 `000003` 时也能补齐字段。
- 整理 MML350 技术路径分组名称，将活跃命令树收敛为业务分组。
- 合并 GPS/TFC 参数展示，隐藏 MR 对象级 ADD/RMV。

## Findings

No CRITICAL findings.

No WARNING findings.

INFO:

- 当前库 `goose` schema 版本显示为 3，是本地历史验证曾应用过已删除迁移的结果；验证中已额外模拟“缺少原 003 字段”的老库，确认增量脚本可补列。
- 增量脚本会输出部分 `NOTICE`，用于说明历史命令已缺失或已废弃，属于兼容旧数据状态的预期行为。

## Validation

- `go build ./...` - passed.
- `go test ./internal/mml ./cmd/migrate` - passed.
- Clean deployment:
  - Created `omcgo_codex_clean_full_verify`.
  - Ran schema migrations.
  - Ran seed migrations.
  - Verified value-rule columns, MML350 group names, GPS merge, MR ADD/RMV hiding, interface binding merge.
- Incremental deployment:
  - Created `omcgo_codex_incremental_full_verify`.
  - Dropped value-rule columns to simulate an old database without former `000003`.
  - Ran `scripts/mml_apply_config_updates_20260721.sql`.
  - Verified columns were recreated and MML tree assertions passed.
- Current local deployment:
  - Ran schema/seed `goose up`.
  - Ran `scripts/mml_apply_config_updates_20260721.sql`.
  - Verified all SQL assertions passed.
  - `http://localhost:8081/` returned `200 OK`.

## Risk

The change is data and migration focused. Main risk is existing operational data with partially imported MML350 commands. The incremental script uses idempotent upserts, command moves, and `ADD COLUMN IF NOT EXISTS` to converge both clean and upgraded databases.
