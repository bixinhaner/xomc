# Review: MML TRPath final bindings

## 结论

PASS

## 范围

- `omcgo/migrations/seed/000001_init_seed.sql`
- `omcgo/scripts/mml_apply_config_updates_20260721.sql`

## 审查要点

- 首次部署路径：baseline seed 增加同等幂等 DML，避免全新库漏补目标 MML 参数绑定。
- 增量部署路径：复用现有 `mml_apply_config_updates_20260721.sql`，未新增迁移文件，符合本次交付要求。
- 幂等性：使用 `UPDATE` 和 `INSERT ... ON CONFLICT (command_id, standard_path_id) DO UPDATE`，重复执行不会产生重复 sub-field。
- 覆盖范围：设备基本信息、软件版本控制、NTP 配置目标命令均刷新 `target_paths` / `tree_node_refs`。

## 发现

- CRITICAL: 无
- WARNING: 无
- INFO: 本次为内置 MML catalog 数据修复，未涉及 Go/前端代码路径。

## 验证

- `PGPASSWORD=omcgo123 psql -h localhost -p 5432 -U omcgo -d omcgo -v ON_ERROR_STOP=1 -v mml_trpath_regrouping=1 -f omcgo/scripts/mml_apply_config_updates_20260721.sql`：通过
- 目标 13 个 standard path 有效活跃绑定计数：`13`
- `python3 scripts/recount_mml_trpath_remaining.py`：通过，当前剩余明细 10 行
- `git diff --staged --check`：通过
