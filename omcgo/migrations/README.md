# Database Migrations

迁移工具：[`pressly/goose/v3`](https://github.com/pressly/goose)。由 `cmd/migrate` 包装执行，docker compose 的 `migrate-*` 服务在容器栈启动期跑。版本号记录在数据库的 goose 版本表里。

## 当前状态：三个 consolidated baseline

2026-07-25 在明确不保留旧数据、不提供就地升级兼容的前提下，将主库 schema、主库 seed 和 TSDB schema 三条**相互独立**的 goose 流重新收敛为各自一个 `000001`。当前文件如下：

| 流 | 合并范围 | 当前文件 | 版本表 | compose 服务 | 目标库 |
|----|----------|----------|--------|--------------|--------|
| 主库 schema (DDL) | 最终状态 baseline | `migrations/000001_init_schema.sql` | `goose_db_version` | `migrate-schema` | postgres（主库，纯 PG16）|
| 主库 seed (DML) | 最终状态 baseline | `migrations/seed/000001_init_seed.sql` | `goose_db_version_seed` | `migrate-seed` | postgres |
| 时序库 schema | 最终状态 baseline | `migrations/tsdb/000001_tsdb_schema.sql` | `goose_db_version_tsdb` | `migrate-tsdb-schema` | postgres-tsdb（TimescaleDB）|

三个 `000001` 文件包含截至 2026-07-25 的最终结构、流式 PM 聚合表、12 个内置聚合任务、内置角色权限修复和 PM 反压默认值。后续新需求再按各自流从 `000002` 开始追加。

> ⚠️ 此基线仅兼容全新安装或允许清库重建的环境，不是既有数据库的就地升级路径。三条流在干净数据库上分别从版本 `000001` 起跑。

### 三个基线文件

1. **`000001_init_schema.sql`**（主库 DDL）—— 全量业务表结构（所有 public 业务表 + 分区子表 + 扩展 `ltree`/`pg_trgm`/`pgcrypto`/`uuid-ossp` + 触发器/函数）。**纯 PostgreSQL 16，无 timescaledb 扩展、无任何超表**（时序对象全在 tsdb 流）。由全量迁移后的库 `pg_dump --schema-only` 生成；PL/pgSQL 函数体已用 goose `StatementBegin/End` 包裹。
2. **`seed/000001_init_seed.sql`**（主库 DML）—— 全量内置参考数据（RBAC/菜单/权限/系统字典/`sys_configs`/MML 命令树/`standard_params` 等）。由干净 schema+seed 库 `pg_dump --data-only --inserts --on-conflict-do-nothing` 生成；**不含运行期 dictloader 从 `data/` XML 加载的 `param_models`/`param_mappings`/`products`/`alarm_definitions`/`perf_indicators` 等**。所有 INSERT 带**无目标 `ON CONFLICT DO NOTHING`**，对全新库重复前向应用幂等。
3. **`tsdb/000001_tsdb_schema.sql`**（时序库）—— 15 张时序表（`pm_metrics` + 4 rollup、`pm_group_metrics_*`、`pm_adhoc_aggregation_results`、`alarms_history`、`mr_records`、`trace_messages`、`pm_files`、`mr_files`）+ 显式 `create_hypertable` + 压缩/保留策略（`alarms_history` retention 固定每天 01:08 Asia/Shanghai）+ 7 张影子维度表（worker `tsdbsync` 从主库同步，供本库 JOIN 替代跨库 JOIN）+ `alarm_efficiency_metrics` 物化视图。显式 DDL，不依赖 pg_restore catalog 注入；**不再建已确认废弃且无任何查询引用的 `pm_metrics_hourly_cagg` 连续聚合视图**。

参数同步基线有一个有意保留的无外键设计：`parameter_sync_task_results.task_id` 和 `parameter_sync_staging_values.task_id` 都不声明到 `device_tasks` 的外键。`device_tasks` 按 `device_sn` 做 hash 分区，物理主键是 `(id, device_sn)`，PostgreSQL 不允许只引用其中的 `id`。应用处理链会校验 payload 的 `device_sn` 与同步 run 的逻辑关联，并按 `(id, device_sn)` 加载设备任务；两张表的 `run_id` 外键仍然保留。不要补回不可成立的 `REFERENCES device_tasks(id)`；如果未来在这两张表持久化 `device_sn`，再评估复合外键。

## 双库（main + tsdb）物理分离

OMC 跑两个 PostgreSQL/TimescaleDB 实例，迁移分两条物理目标库的流，互不交叉：

- **主库**（`postgres` / PgPool，镜像 `postgres:16-alpine`）—— 业务数据（devices、RBAC、config、products、device_groups、alarm_definitions、`alarms`、pm_tasks、`mr_customize_task`…）。**纯 PG16，无 timescaledb 扩展、无超表**。
- **时序库**（`postgres-tsdb` / TsPool，`timescale/timescaledb`）—— 时序/PM 表 + 影子维度表 + 物化视图。

各超表 chunk/压缩/保留参数：

| 超表 | chunk 间隔 | compress_after | drop_after（retention）|
|------|-----------|----------------|------------------------|
| `pm_measurement_anchors` / `pm_metric_values` | 4 hours | 7 days | `pm.retention.raw_15min_days` |
| `pm_hourly_anchors` / `pm_hourly_values` | 已由流式聚合替代 | — | — |
| `pm_group_metrics_hourly` | 7 days | 14 days | 180 days |
| `pm_aggregation_results` | 30 days | 90 days | 按粒度读取 `pm.retention` |
| `mr_records` | 1 day | 7 days | 90 days |
| `alarms_history` | 7 days | （无压缩）| 365 days |
| `trace_messages` | 1 day | （无压缩）| 3 days |

## 新增迁移的版本号规则

三条流是相互独立的 goose 版本序列，记在不同版本表、**不共享号段**——所以 `000001` 在三处各出现一次是**正常的**（不是撞号）。查撞号要**分目录各查**，别把三个目录的文件名合并去重。

- 新增 = 该流**现有最大号 + 1**。当前三条流的下一号均是 `000002`。
- 从本基线开始，不回填空号、不重排或复用同一发布基线内已经应用过的版本号。三条流使用独立版本表，必须分目录判断下一号。
- DDL → `migrations/`，DML 种子 → `migrations/seed/`，时序 DDL → `migrations/tsdb/`。
- `DO $$` / `CREATE [OR REPLACE] FUNCTION` / 循环条件 → 必须 goose `StatementBegin/End` 包裹。
- 幂等：`CREATE TABLE IF NOT EXISTS`、`ADD COLUMN IF NOT EXISTS`、`INSERT ... ON CONFLICT DO NOTHING`。
- Down 段删除 Up 段创建的所有对象。
- 改已被 seed 引用的表：新增列可空或带 `DEFAULT`；收紧约束前同一迁移先 backfill；删列/改名走两阶段跨 release。

```bash
# 取某流下一号（seed 示例）
ls migrations/seed/[0-9]*.sql | xargs -n1 basename | sed 's/_.*//' | sort -n | tail -1

# 撞号自查（三流各查一遍，应都为空）
ls migrations/[0-9]*.sql      | xargs -n1 basename | sed 's/_.*//' | sort | uniq -d
ls migrations/seed/[0-9]*.sql | xargs -n1 basename | sed 's/_.*//' | sort | uniq -d
ls migrations/tsdb/[0-9]*.sql | xargs -n1 basename | sed 's/_.*//' | sort | uniq -d
```

## 执行迁移

```bash
# 本地（docker compose 栈，在仓库根 goomc/ 下执行）
docker compose -f deployments/docker/docker-compose.yml up -d --build migrate-schema migrate-seed migrate-tsdb-schema
# 顺序：postgres → migrate-schema → migrate-seed；postgres-tsdb → migrate-tsdb-schema

# 本地（绕过 docker，仅主库 schema+seed）
cd omcgo && make migrate-up   # = go run ./cmd/migrate up --paths migrations,migrations/seed
```

> **版本表接口约束：**主库 schema 流（`migrations/`）的 Up/Down 必须使用 goose 默认版本表 `goose_db_version`，不得传自定义 `--table`；其 Down 会临时移动并恢复这个精确表名。只有 seed 和 TSDB 两条流使用已文档化的自定义接口：`--table goose_db_version_seed` 与 `--table goose_db_version_tsdb`。

> ⚠️ **全新库验证 seed 必须用独立版本表**（compose 的 `migrate-seed` 已设 `GOOSE_TABLE=goose_db_version_seed`）。别用 `make migrate-up` 的 `--paths` 共享默认表跑全新库验证：schema 跑完默认版本表已占 `version_id=1`，seed 的 `000001` 会因撞号被当「已应用」跳过、不执行。

## 既有库升级约束

本基线**只面向全新发布部署或清库重建**。既有（合并前已迁移过的）库的 `goose_db_version*` 仍记录已经删除的增量版本，不能靠简单 `goose up` 升级，也不要仅重置版本表后在非空库上重跑基线。需要保留数据的环境必须另行设计和验证升级迁移；当前支持路径是备份所需数据、重建数据库，再从三个 `000001` 基线初始化。

## 连接池核定

三个部署单元（app/acs/worker）各 `cmd/*/etc/config.{test,prod}.yaml` 的 `max_conns`（dev/local 不动）：

| 单元 | db（PG）max_conns | tsdb（TS）max_conns |
|------|------------------|---------------------|
| app | 60 | 40 |
| acs | 30 | （无 tsdb 块）|
| worker | 25 | 25 |

单副本聚合预算 `app(60+40)+acs(30)+worker(25+25)=180`，要求 PostgreSQL `max_connections ≥ 200`。app/acs 水平扩展后连接数线性累加，逼近上限前须前置 pgbouncer（transaction pooling）——100 万级扩展的既定演进项，本仓暂不引入。

## 重生 baseline 的标准流程

未来再要 consolidate（把积累的增量折回单 `000001`），按下面走（基于 `OMCGO_ENV=test`）：

```bash
# 1. 备份（backup-* 已 gitignore）
cp -r omcgo/migrations omcgo/migrations.backup-$(date +%Y%m%d)

# 2. 干净库跑完全部现有迁移（先去掉任何撞号）
cd deployments/docker
docker compose down -v
docker compose up -d --wait postgres postgres-tsdb redis nats minio
docker compose up -d migrate-schema migrate-seed migrate-tsdb-schema   # 等三者 exit 0

# 3. 先记录 golden 逐表行数（验证用），再从容器内 pg_dump 主库
#    schema：
docker compose exec -T postgres pg_dump -U omcgo -d omcgo \
  --schema-only --no-owner --no-privileges --no-tablespaces \
  --no-publications --no-subscriptions \
  --exclude-table='goose_db_version*' > /tmp/raw_schema.sql
#    seed（--on-conflict-do-nothing 保幂等）：
docker compose exec -T postgres pg_dump -U omcgo -d omcgo \
  --data-only --inserts --rows-per-insert=500 --on-conflict-do-nothing \
  --no-owner --no-privileges --disable-triggers \
  --exclude-table='goose_db_version' --exclude-table='goose_db_version_seed' > /tmp/raw_data.sql

# 4. 后处理（否则 goose 跑不过 — 踩坑记录）
#    A. 删 psql 元命令 \restrict / \unrestrict（pgx Exec 不识别；新版 pg_dump 会输出）。
#    B. search_path：pg_dump 顶部 set_config('search_path','',...) 会让 goose 写版本表
#       找不到表（42P01）。schema dump 自带末尾 set_config(...,'public',...) 兜底；
#       seed dump 若无，必须在末尾（goose Down 前）补一行
#       SELECT pg_catalog.set_config('search_path','public',false); 还原。
#    C. 主库是纯 PG16（无 timescaledb），无 _timescaledb_catalog，无需相关处理。

# 5. 用 goose Up/Down 包裹（schema 增量 append 进 Up 段末尾、函数块包 StatementBegin/End，
#    Down 用 DROP SCHEMA public CASCADE; seed 无回滚写 SELECT 1;），写入
#    000001_init_schema.sql / seed/000001_init_seed.sql，删旧增量文件。

# 6. 验证：down -v 再起，三流 goose up 全绿 + 逐表行数与 golden diff=0 + 以浏览器实际页面验收 V1 webcode UI 壳。
```

## 相关位置

- 完整迁移规范：`omcgo/CLAUDE.md` §4.6
- 踩坑案例库：`../docs/ref/migration-pitfalls.md`
- 迁移命令封装：`omcgo/cmd/migrate/main.go`
- compose migrate 服务：`deployments/docker/docker-compose.yml`
- 历史备份：`omcgo/migrations.backup-*`（gitignore）
