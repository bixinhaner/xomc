# Seed Migrations

主库种子数据（DML）迁移目录，独立 goose 版本表 `goose_db_version_seed`，由 docker compose 的 `migrate-seed` 服务在 `migrate-schema` 完成后执行。当前为单个 consolidated baseline `000001_init_seed.sql`（见 `../README.md`）。

> ⚠️ **必须用独立版本表执行**（生产做法）：
> ```bash
> go run ./cmd/migrate up --path migrations                                          # schema（默认表 goose_db_version）
> GOOSE_TABLE=goose_db_version_seed go run ./cmd/migrate up --path migrations/seed    # seed（独立表）
> ```
> 别用 `make migrate-up`（`--paths migrations,migrations/seed` 共享默认表）做**全新库验证**——
> schema 跑完默认版本表已占 `version_id=1`，seed 的 `000001` 会因撞号被当「已应用」而**跳过**，
> 导致 `000001_init_seed.sql` 不执行。compose 的 `migrate-seed` 已设 `GOOSE_TABLE=goose_db_version_seed`，无此问题。

## 部署数据 vs 测试数据（2026-06-08 拆分，见 `../../../docs/design/seed-data-split-analysis-20260608.md`）

`000001_init_seed.sql` 是**纯部署 baseline**——只内置全新系统跑起来必需的基础数据，不含测试样例、也不含运行期由 dictloader 从 XML 加载的字典。

**内置（baseline 自带）**：

- **RBAC**：`users`（admin 超管 + system 内部账号，均 builtIn）、`roles`（admin / operator / viewer）、`user_roles`、`role_inheritance`、`role_menus`、`role_api_permissions`
- **菜单 / API 权限**：`menus`、`api_endpoints`
- **系统字典 / 配置**：`sys_dictionaries`、`sys_dictionary_details`、`sys_configs`
- **基础参考**：`alarm_severity_levels`、`mr_indicators`、`ufte_task_types`、`ops_templates`、`pm_tasks`（全网内置任务）、`dashboard_kpi_layouts`
- **设备分组**：仅两个系统内置组 `默认设备组`（一级）+ `默认设备组`（二级未分组视图）
- **MML 命令树 + 标准参数**：`mml_param_versions`、`mml_command_groups`、`mml_commands`、`mml_command_sub_fields`、`standard_params`（MML 命令的元属性引用；由 cmcc_tdlte 标准派生）

**不内置**的两类数据：

1. **数据模型 / 部分字典** —— `param_models` / `param_mappings` / `products` / `product_class_patterns` / `alarm_definitions` / `indicator_*` / `perf_indicators_*` / `rela_*` / `enabled_pm_indicators_*` 等
   → 改由 **dictloader 在 app 启动期从 `data/` XML 加载**（`dict_loader.auto_load_on_startup`，dev/test/prod 均 `true`），SQL seed 不保留，避免与 XML 双源漂移。
   （`kpi_definitions` 已随时序表迁到时序库，不在主库 seed。）
2. **测试样例数据** —— 大批量设备 `devices_*`、`device_group_members`、`device_parameters_*`、`device_tasks_*`、告警实例等
   → 开发 / 联调按需手动加载 `omcgo/scripts/seed_dev_data.sql`（拆分前 init_seed 中的运行时数据，含约 3.3 万测试设备 + 19 个示例分组）：
   ```bash
   docker compose -f deployments/docker/docker-compose.yml exec -T postgres \
     psql -U omcgo -d omcgo < omcgo/scripts/seed_dev_data.sql
   ```
   **不随 goose 自动执行**，按需手动加载。
   > ⚠️ `scripts/seed_test_data.sql` 与 `scripts/seed_e2e_testdata.sql` 仍在仓库但**对当前 schema 已失效**（引用了已不存在的表/列），勿直接使用。

> 历史上曾把测试环境整库（含 3.3 万测试设备 + 全部 XML 字典）pg_dump 进 `000001`，导致全新部署被灌入大量测试数据；2026-06-08 拆分后部署得到干净最小基础数据。

## 新增 seed 迁移

从该流**现有最大版本号 + 1** 起递增（当前在 `000001`，下一号 `000002`）。版本号规则、重生 baseline 标准流程、连接池核定见 `../README.md`；完整迁移规范见 `../../CLAUDE.md` §4.6。
