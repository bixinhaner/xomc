# Database Migrations

迁移工具是 [`pressly/goose/v3`](https://github.com/pressly/goose)，由 `omcgo/cmd/migrate` 包装执行。docker compose 的 `migrate-*` 服务会在容器栈启动期跑迁移，版本号记录在各自的 goose 版本表里。

## 当前 baseline

2026-07-25 在明确不保留旧数据、不提供就地升级兼容的前提下，主库 schema、主库 seed、TSDB schema 三条独立 goose 流已重新收敛为各自一个 `000001`。

| 流 | 当前文件 | 版本表 | compose 服务 | 目标库 |
|----|----------|--------|--------------|--------|
| 主库 schema (DDL) | `000001_init_schema.sql` | `goose_db_version` | `migrate-schema` | postgres（主库，纯 PG16） |
| 主库 seed (DML) | `seed/000001_init_seed.sql` | `goose_db_version_seed` | `migrate-seed` | postgres |
| 时序库 schema | `tsdb/000001_tsdb_schema.sql` | `goose_db_version_tsdb` | `migrate-tsdb-schema` | postgres-tsdb（TimescaleDB） |

三个 `000001` 文件包含截至 2026-07-25 的最终结构、流式 PM 聚合表、12 个内置聚合任务、内置角色权限修复和 PM 反压默认值。

## 未封版本维护规则

> ⚠️ 当前软件尚未封版本，迁移目录只允许维护上述三个基线文件，不允许新增 `000002+`。新增 DB、seed、tsdb 变化必须折回对应的 `000001` 基线文件。

- 主库 schema 变化折回 `000001_init_schema.sql`。
- 主库 seed 变化折回 `seed/000001_init_seed.sql`。
- 时序库 schema 变化折回 `tsdb/000001_tsdb_schema.sql`。
- 软件封版本后，先更新根 `AGENTS.md`、本 README 和 `seed/README.md`，明确允许追加迁移，再从各自流的 `000002` 开始新增文件。

> ⚠️ 当前 baseline 仍以全新安装为准。对已有的预发布数据库，仅允许把纯新增、可幂等
> 重放的主库结构和种子放进 `MainReconcile` 标记区段；迁移程序执行既有的
> `migrations/seed` 流程时会自动协调这些区段。删除、改名、类型收紧或数据回填等非加法
> 变化仍必须按 [Baseline 运维手册](../../docs/ref/migration-baseline-runbook.md) 单独设计并先备份。

## 三个基线文件

1. `000001_init_schema.sql`：主库全量业务表结构、分区子表、扩展、触发器和函数。主库保持纯 PostgreSQL 16，不包含 TimescaleDB 扩展或超表。
2. `seed/000001_init_seed.sql`：主库内置参考数据，包括 RBAC、菜单、权限、系统字典、系统配置、MML 命令树、标准参数等；不包含运行期 dictloader 从 `data/` XML 加载的数据。
3. `tsdb/000001_tsdb_schema.sql`：时序库表、超表、压缩/保留策略、影子维度表和 `alarm_efficiency_metrics` 物化视图。

参数同步基线中，`parameter_sync_task_results.task_id` 和 `parameter_sync_staging_values.task_id` 有意不声明到 `device_tasks` 的外键。`device_tasks` 的物理主键是 `(id, device_sn)`，不能只引用 `id`；不要补回不可成立的 `REFERENCES device_tasks(id)`。

## 目标库与版本表

| 目标库 | 内容 | 迁移流 | 版本表 |
|--------|------|--------|--------|
| postgres / PgPool | 业务数据、RBAC、配置、产品、设备、告警、PM 任务等 | schema + seed | `goose_db_version` / `goose_db_version_seed` |
| postgres-tsdb / TsPool | PM、MR、告警历史、trace、影子维度表、物化视图等 | tsdb schema | `goose_db_version_tsdb` |

三条流相互独立，`000001` 在三个目录各出现一次是正常的，不是撞号。封版本后的追加迁移规则见 [Baseline 运维手册](../../docs/ref/migration-baseline-runbook.md)。

## 常用执行命令

```bash
# docker compose 栈，在仓库根目录执行
docker compose -f deployments/docker/docker-compose.yml up -d --build \
  migrate-schema migrate-seed migrate-tsdb-schema

# 仅本地直跑主库 schema + seed，在 omcgo/ 下执行
make migrate-up
```

`migrate-schema` 使用默认版本表 `goose_db_version`；`migrate-seed` 必须使用 `goose_db_version_seed`；`migrate-tsdb-schema` 必须使用 `goose_db_version_tsdb`。全新库验证 seed 时不要用共享默认版本表，否则 seed 的 `000001` 会被误判为已应用。

### 预发布主库兼容协调

`omcgo-migrate` 在执行既有 `migrations/seed` 目录时自动协调，无需修改 Docker Compose
参数。该步骤从主 schema 和 seed 两个 `000001` 中提取成对的
`-- +omcgo MainReconcileBegin/End` 区段，在 PostgreSQL advisory transaction lock 下先补齐
schema，再执行 Goose seed，最后幂等补齐 seed；任一步失败都会返回非零状态，并通过既有
`depends_on: service_completed_successfully` 阻止 App、ACS 和 Worker 使用不完整结构。

协调区段只允许静态、可重复执行的 SQL：`ADD COLUMN IF NOT EXISTS`、
`CREATE TABLE/INDEX IF NOT EXISTS`、带 catalog 守卫的约束，以及 `INSERT ... ON CONFLICT`。
禁止在其中放 `DROP`、`TRUNCATE` 或无条件覆盖业务数据的 DML。104 等保留真机数据的环境
首次升级前仍必须执行 `pg_dump` 并审核实际 SQL。

## 相关链接

- Seed 说明：`seed/README.md`
- Baseline 运维手册：`../../docs/ref/migration-baseline-runbook.md`
- 迁移踩坑案例：`../../docs/ref/migration-pitfalls.md`
- 完整迁移规范：`../CLAUDE.md` §4.6
- 迁移命令封装：`../cmd/migrate/main.go`
- compose migrate 服务：`../../deployments/docker/docker-compose.yml`
