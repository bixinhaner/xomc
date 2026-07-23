# MML Config Deduplicate GPS Review

## 结论

PASS_WITH_WARNINGS

## 范围

- 调整 CMCC TD-LTE MML 目录和 loader seed，移除 FAP_SERVICE 下重复的 RRC timer 参数。
- 调整首次部署 seed，使重复路径按既定保留规则收敛。
- 调整增量脚本，兼容已部署库的重复分组清理和 GPS 信息参数管理归属修复。

## Findings

### WARNING

- 未执行完整 `go test ./...`：该命令在提交前被用户中断。本次已执行相关包测试和首次部署临时库验证，覆盖迁移、seed 和 loader 相关改动。

### INFO

- `mml_apply_config_updates_20260721.sql` 保留 `cleanup_mml_duplicate_group_bindings()`，仅用于增量历史库修复；首次部署不依赖 schema 中的数据清理函数。
- YAML 本地环境配置改动按用户要求未纳入本次提交。

## 验证

- `go build ./...`：通过。
- `go test ./internal/config/parammodel/mmlstandardloader ./cmd/tools/gen_seed_sql ./cmd/migrate`：通过。
- `git diff --check`：通过。
- 首次部署临时库：schema + seed 迁移成功，`duplicate_path_count = 0`。
- 本地已部署库：`duplicate_path_count = 0`，`LST/MOD SO_SUB_01` 归属为 `GPS信息参数管理`。

## 风险

- 增量脚本包含历史库修复逻辑，需要保持在既有 `20260721` 运维脚本中执行；全新部署路径已由 seed 直接生成最终数据。
