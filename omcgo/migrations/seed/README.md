# Seed Migrations

种子数据迁移目录，独立 goose 版本表 `goose_db_version_seed`，由 compose 中的 `migrate-seed-sql` 服务在 `migrate-schema` 完成后执行。

> ⚠️ **必须用独立版本表执行**（生产做法）：
> ```bash
> go run ./cmd/migrate up --path migrations                                   # schema（默认表 goose_db_version）
> GOOSE_TABLE=goose_db_version_seed go run ./cmd/migrate up --path migrations/seed   # seed
> ```
> 不要用 `make migrate-up`（`--paths migrations,migrations/seed` 共享默认表）做全新库验证 ——
> schema 跑完后版本表已占用 version_id 1..N，seed 的 000001 等会因 version_id 撞号被当作"已应用"而**跳过**，
> 导致 `000001_init_seed.sql` 不执行、后续增量 seed 的 FK 引用落空。

## 部署数据 vs 测试数据（2026-06-08 拆分，见 `docs/design/seed-data-split-analysis-20260608.md`）

`000001_init_seed.sql` 现在是**纯部署 baseline**（~2.2 千行），只内置全新系统跑起来必需的基础数据：

- RBAC：`users`（admin 超管 + system 内部账号，均 builtIn）、`roles`（admin 全权 / operator / viewer）、`user_roles`、`role_inheritance`、`role_menus`、`role_api_permissions`
- 菜单 / API 权限：`menus`、`api_endpoints`
- 系统字典：`sys_dictionaries`、`sys_dictionary_details`、`sys_configs`
- 其它基础：`alarm_severity_levels`、`indicator_unit`、`kpi_definitions`、`mr_indicators`、`ufte_task_types`、`ops_templates`
- 设备分组：仅两个系统内置组 `默认设备组`（一级）+ `未分组设备`（二级）

**不再内置**的两类数据：

1. **数据模型 / 字典**（`param_models` / `param_mappings` / `standard_params` / `products` / `product_class_patterns` / `indicator_*` / `perf_indicators_*` / `rela_*` / `enabled_pm_indicators_*` / `alarm_definitions` / `mml_commands` / `mml_command_groups` / `mml_command_sub_fields` / `mml_param_versions` 等）
   → 改由 **dictloader 在 app 启动期从 `datamodels/` XML 加载**（`dict_loader.auto_load_on_startup`，dev/test/prod 均为 `true`）。SQL seed 不再保留，避免与 XML 双源漂移。
2. **测试样例数据**（大批量设备 `devices_*`、`device_group_members`、`device_parameters_*`、`device_tasks_*`、告警实例等）
   → 开发 / 联调如需，手动加载 `omcgo/scripts/seed_dev_data.sql`（拆分前 init_seed 中的设备等运行时数据，含 3.3 万测试设备 + 19 个示例分组，schema 与当前一致）：
   ```bash
   # docker compose 栈
   docker compose -f deployments/docker/docker-compose.yml exec -T postgres \
     psql -U omcgo -d omcgo < omcgo/scripts/seed_dev_data.sql
   ```
   **不随 goose 自动执行**，按需手动加载。
   > ⚠️ 旧的 `scripts/seed_test_data.sql` 与 `scripts/seed_e2e_testdata.sql` 已对当前 schema **失效**（引用了已不存在的表 `licenses` / 列 `upgrade_tasks.device_id`），勿直接使用，待单独修复。

> 历史上曾把测试环境整库（含 3.3 万测试设备 + 全部 XML 字典）pg_dump 进 `000001`，导致全新部署被灌入大量测试数据。本次拆分后部署得到干净最小基础数据。原混合 baseline 可在 git 历史 `7d4ae90f^` 取回。

新增 seed 迁移从现有最大版本号 +1 起递增。完整规范、合并流程、回滚说明见 `omcgo/migrations/README.md` 和 `omcgo/CLAUDE.md` §5.5。
