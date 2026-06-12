# Database Migrations

迁移工具：[`pressly/goose/v3`](https://github.com/pressly/goose)。版本号记录在数据库 `goose_db_version`（DDL）和 `goose_db_version_seed`（DML）两个表中，由 `cmd/migrate` 包装执行。

## 当前状态：consolidated baseline（2026-06-12，KPI/时序库物理分离后重新合并）

KPI/时序库物理分离落地后，对**三条流各做一次干净的 consolidated baseline**（把分离引入的全部增量折叠进基线，作全新部署使用，不背历史版本号包袱）。现在**每条流各一个 baseline 文件，共 3 个**：

| 文件 | 目标库 | 说明 |
|------|--------|------|
| `000001_init_schema.sql` | 主库（PgPool）| 全量业务表结构（195 张 public 表 + pgcrypto/uuid-ossp/pg_trgm/ltree 等扩展 + 触发器/函数）。**纯 PostgreSQL 16，不含 timescaledb、无任何超表**——由全量迁移后的库 `pg_dump --schema-only` 生成（函数块已包 goose `StatementBegin/End`）。 |
| `seed/000001_init_seed.sql` | 主库（PgPool）| 全量内置参考数据（admin/角色/菜单/权限 + 字典 + sys_configs + standard_params…）。由干净 schema+seed 库 `pg_dump --data-only --inserts` 生成（**不含运行期 dictloader 加载的 perf_indicators 等**）。 |
| `tsdb/000001_tsdb_schema.sql` | 时序库（TsPool / postgres-tsdb）| 见下「双流」。 |

> 验证：三流在全新双实例 `goose up` 全绿；consolidated 主库 schema/seed 与「分离前多文件迁移」逐表行数完全一致（195 表 diff=0）。主库镜像随之从 `timescale/timescaledb` 改为 `postgres:16-alpine`。

## 双流：主库（main）+ 时序库（tsdb）物理分离

> KPI/时序库物理分离后，迁移分两条物理目标库的流，互不交叉。

OMC 现在跑**两个 PostgreSQL/TimescaleDB 实例**：

- **主库**（`postgres` / PgPool，镜像 `postgres:16-alpine`）—— 业务数据（devices、RBAC、config、products、device_groups、alarm_definitions、`alarms`[当前告警]、pm_tasks、pm_summary、mr_files…）。**纯 PostgreSQL 16，无 timescaledb 扩展、无任何超表**（分离后主库不再需要）。
- **时序库**（`postgres-tsdb` / TsPool）—— 14 张时序/PM 表（`pm_metrics`+4 rollup、`pm_group_metrics_*` 4、`pm_adhoc_aggregation_results`、`alarms_history`、`mr_records`、`trace_messages`、`pm_files`）+ 6 张「影子维度表」（`device_dim`/`device_group_member_dim`/`cell_band_dim`/`product_dim`/`device_group_dim`/`alarm_definition_dim`，由 worker `tsdbsync` 从主库同步，供本库 JOIN 替代跨库 JOIN）+ 1 个告警效率物化视图 `alarm_efficiency_metrics`。

| 流 | 目录 | 版本表 | compose 服务 | DSN |
|----|------|--------|--------------|-----|
| 主库 schema | `migrations/*.sql` | `goose_db_version` | `migrate-schema` | `postgres` |
| 主库 seed | `migrations/seed/*.sql` | `goose_db_version_seed` | `migrate-seed` | `postgres` |
| 时序库 schema | `migrations/tsdb/*.sql` | `goose_db_version_tsdb` | `migrate-tsdb-schema` | `postgres-tsdb` |

要点：

1. **主库 `000001_init_schema.sql`**（consolidated）—— 纯业务 schema，**不含上述 14 张时序表、不含 timescaledb 扩展**。由分离后的全量迁移库 `pg_dump` 生成并折叠了分离引入的全部 schema 增量，故已无 `SELECT 1;` 空操作残留。
2. **主库 `seed/000001_init_seed.sql`**（consolidated）—— 纯内置参考数据，**不含 `_timescaledb_catalog` 注册块、不含 kpi_definitions（已随表迁时序库）、不含运行期 dictloader 数据**。由干净 schema+seed 库 `--data-only --inserts` 生成。
3. **时序库 `tsdb/000001_tsdb_schema.sql`** 用**显式 DDL** 还原 14 张表的**最终形态**（= 主库 000001 原定义叠加上述增量净效果）+ 显式 `create_hypertable`（**`pm_metrics` chunk 间隔 `INTERVAL '4 hours'`，修 B0**；其余还原自原 seed catalog 的 `dimension.interval_length`）+ 压缩策略（原 5 张）+ 保留策略（原 7 张）+ 6 张影子维度表 + `alarm_efficiency_metrics` matview（含支持 `REFRESH CONCURRENTLY` 的唯一索引）。不再依赖 seed 的 pg_restore catalog 注入。

各超表的 chunk/压缩/保留参数（还原自原 seed `_timescaledb_catalog`）：

| 超表 | chunk 间隔 | compress_after | drop_after（retention）|
|------|-----------|----------------|------------------------|
| `pm_metrics` | **4 hours**（修 B0；原 seed 为 1 day）| 7 days | 30 days |
| `pm_metrics_hourly` | 7 days | 14 days | 180 days |
| `pm_group_metrics_hourly` | 7 days | 14 days | 180 days |
| `pm_adhoc_aggregation_results` | 30 days | 90 days | 365 days |
| `mr_records` | 1 day | 7 days | 90 days |
| `alarms_history` | 7 days | （无压缩）| 365 days |
| `trace_messages`（按 `captured_at` 分区）| 1 day | （无压缩）| 3 days |

## 新增迁移的版本号规则（版本号分配约定 · #24）

### 三条独立版本序列（边界）

schema、seed 与 tsdb 是**三条相互独立的 goose 版本序列**，各自记录在不同的版本表，**不共享号段**：

| 序列 | 目录 | goose 版本表 | 执行服务 | 目标库 |
|------|------|--------------|----------|--------|
| schema（DDL）| `migrations/*.sql` | `goose_db_version` | compose `migrate-schema` | 主库（PgPool）|
| seed（DML）| `migrations/seed/*.sql` | `goose_db_version_seed` | compose `migrate-seed` | 主库（PgPool）|
| tsdb（时序 DDL）| `migrations/tsdb/*.sql` | `goose_db_version_tsdb` | compose `migrate-tsdb-schema` | 时序库（TsPool / postgres-tsdb）|

因此 `000001` 在三处各出现一次是**正常的**（不是撞号）——`ls ... | sort | uniq -d` 查撞号时必须分目录各查一遍，**不要把目录的文件名合并去重**（合并会把 `000001`/`000002` 等误报成重复）。

### 分配规则

- 新增 = 该序列**现有最大号 + 1**，不在两序列间借号、不复用已删号、不回填低位号。
- 取下一号：
  ```bash
  # schema 下一号
  ls omcgo/migrations/[0-9]*.sql | xargs -n1 basename | sed 's/_.*//' | sort -n | tail -1
  # seed 下一号
  ls omcgo/migrations/seed/[0-9]*.sql | xargs -n1 basename | sed 's/_.*//' | sort -n | tail -1
  ```
- 同序列内**撞号自查**：
  ```bash
  ls omcgo/migrations/[0-9]*.sql      | xargs -n1 basename | sed 's/_.*//' | sort | uniq -d   # 应为空
  ls omcgo/migrations/seed/[0-9]*.sql | xargs -n1 basename | sed 's/_.*//' | sort | uniq -d   # 应为空
  ```

### 号段允许有空洞（gap），且空洞不可复用

goose 以 `version_id`（号）为唯一键判定「已应用」。历史上若某号曾被应用又被删档，其号已写入版本表；**复用该号会让 goose 把新内容当作「已应用」而跳过执行**。所以：

- 当前 schema 序列在 `000001..000033` 间存在空洞（如 12–15、22、28、32），seed 序列在 `000001..000032` 间存在空洞（如 21、23、26、27）——这些是被弃用/删档迁移留下的，**属正常现象**。
- 新增一律用 `max+1`，**绝不**回填这些空洞，也不重排已有文件的号。
- 因此 CLAUDE.md §4.6 里「连续递增、无跳跃」应理解为「**单调递增、不回填**」：相邻号之间可以有历史遗留空洞，新号只许在最大号之上递增。

其它规则（StatementBegin/End、TimescaleDB 压缩顺序、分区表外键、UUID 校验、Down 完整性、TRUNCATE/FK、自查清单）见 `omcgo/CLAUDE.md` §4.6 数据库迁移规范。

> CLAUDE.md 中提到的 `migrations/000057`、`000058`、`000063`、`000215`、`000217` 等具体编号在本次合并后**已不存在**，对应能力已并入 `000001_init_schema.sql`，文档中的编号只用作历史溯源参考。

## seed baseline 幂等性（#24）

`seed/000001_init_seed.sql` 的全部 public 部署数据 `INSERT`（admin/RBAC/菜单/字典/MML 字典/standard_params 等共 30 条语句）均带 `ON CONFLICT DO NOTHING`，使该 baseline 对**全新库重复前向应用**安全无错（行数稳定、不报唯一键冲突）。

- 采用**无目标** `ON CONFLICT DO NOTHING`（不写 `(col)` 推断列）：这些表多数带不止一个唯一约束（PK + 业务唯一键），重灌的是**逐字段一致的同一行**，无目标形式可对任意唯一约束冲突一并静默跳过，比指定单一仲裁索引更稳。系统/参考型种子取 `DO NOTHING`（不 `DO UPDATE`），避免覆盖运行期可能被合法修改的值。
- `_timescaledb_catalog.*`（hypertable/bgw_job/dimension/compression_settings 共 4 条 INSERT）**不加** `ON CONFLICT`：它们是 TimescaleDB 内部 catalog，靠文件头尾的 `timescaledb_pre_restore()` / `timescaledb_post_restore()` 括号 + `DISABLE/ENABLE TRIGGER` 还原机制保证一致性，是 pg_dump 还原约定的一部分，人为加 `ON CONFLICT` 反而偏离该约定。
- 边界：goose **不会重跑已应用版本**，改动已应用的 baseline 只对**全新部署**生效（这正是本次幂等加固的目标场景）；已存量库不受影响、也无需回灌。

## 连接池核定（#24）

三个部署单元连同一套 PostgreSQL/TimescaleDB（同一实例、`db` 走 PG、`tsdb` 走 TimescaleDB 扩展）。各单元 `cmd/*/etc/config.{test,prod}.yaml` 的 `max_conns` 核定如下（**dev/local 保持原值不动，保证本地可用**）：

| 单元 | db（PG）max_conns | tsdb（TS）max_conns | 理由 |
|------|------------------|---------------------|------|
| app | 50 → **60** | 30 → **40** | REST + gRPC 主负载，连接需求最高 |
| acs | 20 → **30** | （无 tsdb 块）| 高并发 CWMP 会话（`session.max_concurrent=10000`）|
| worker | 20 → **25** | 20 → **25** | PM/MR 文件处理 + KPI 聚合的后台批负载，受 WorkerPool 约束，温和上调 |

**单副本聚合预算**：`app(60+40) + acs(30) + worker(25+25) = 180` 个连接，要求 PostgreSQL `max_connections ≥ 200`（给 superuser / 复制 / 临时连接留余量）。

**横向扩展告警**：app 与 acs 可水平扩展，连接数随副本数**线性累加**（如 app×3 + acs×3 即 300+90）。在逼近 PG `max_connections` 之前，**必须前置 pgbouncer**（`transaction` pooling 模式）做连接复用，把上游成百上千的客户端连接收敛到几十个真实后端连接——这是 100 万级扩展的既定演进项，本次仅做单副本下的保守上调，不引入 pgbouncer 依赖。

## Goose 包装约定

baseline 文件全文用单个 `-- +goose StatementBegin / StatementEnd` 块包裹，避免 goose 默认按 `;` 切语句时把 PL/pgSQL 函数体（`$$...$$`）和 TimescaleDB 内部 SQL 错切。新增的常规迁移仍按 CLAUDE.md §5.5.1 的规则只包 DO/FUNCTION 块。

## 执行迁移

```bash
# 本地（dev）— 通过 docker compose
cd deployments/docker
OMCGO_ENV=test docker compose -f docker-compose.yml up -d --build
# postgres → migrate-schema(applies 000001_init_schema) → migrate-seed(applies 000001_init_seed) → app/acs/worker

# 本地（绕过 docker）
cd omcgo
make migrate-up   # = go run ./cmd/migrate up --paths migrations,migrations/seed
```

## 重生 baseline 的标准流程

未来若再次需要 consolidate，按以下步骤（基于 `OMCGO_ENV=test` 环境）：

```bash
# 1. 备份现有迁移
cp -r omcgo/migrations omcgo/migrations.backup-$(date +%Y%m%d)

# 2. 拉起干净环境，跑完所有现有迁移
cd deployments/docker
docker compose down -v
OMCGO_ENV=test docker compose -f docker-compose.yml up -d --build postgres redis nats minio migrate-schema migrate-seed

# 3. 等 migrate-schema / migrate-seed 退出码 0 后,从容器内 pg_dump
docker compose exec -T postgres pg_dump -U omcgo -d omcgo \
  --schema-only --no-owner --no-privileges --no-tablespaces \
  --no-publications --no-subscriptions \
  --exclude-table='goose_db_version*' > /tmp/raw_schema.sql

docker compose exec -T postgres pg_dump -U omcgo -d omcgo \
  --data-only --inserts --rows-per-insert=500 \
  --no-owner --no-privileges --disable-triggers \
  --exclude-table='goose_db_version*' > /tmp/raw_data.sql

# 4. 必须的 3 处后处理（否则 goose 跑不过 — 历史踩坑记录）
#    A. 剔除 psql 元命令 \restrict / \unrestrict（pgx Exec 不识别）
sed -i.bak '/^\\\(restrict\|unrestrict\) /d' /tmp/raw_schema.sql /tmp/raw_data.sql

#    B. 剔除 search_path 重置（会让 goose 后续 INSERT goose_db_version 找不到表）
sed -i.bak "/^SELECT pg_catalog.set_config('search_path', '', false);$/d" \
  /tmp/raw_schema.sql /tmp/raw_data.sql

#    C. 从 raw_data.sql 中删除 _timescaledb_catalog.metadata 的 INSERT 块
#    （install_timestamp / timescaledb_version 行会与 CREATE EXTENSION 装机指纹冲突 metadata_pkey）
#    人工 sed 删 ALTER ... DISABLE TRIGGER → INSERT INTO _timescaledb_catalog.metadata → ENABLE 三行块

# 5. 用 -- +goose Up/Down + StatementBegin/End 包裹两份 dump 写入：
#    omcgo/migrations/000001_init_schema.sql
#    omcgo/migrations/seed/000001_init_seed.sql
#    Down 段建议 DROP SCHEMA public CASCADE; CREATE SCHEMA public; + DROP EXTENSION（schema 文件）
#    Seed 段的 Down 委托 schema Down 处理，写 SELECT 1; 即可

# 6. 验证：down -v 后再 up，比对 row count
docker compose down -v
docker compose up -d postgres redis nats minio migrate-schema migrate-seed
# row count 应与备份的 golden 完全一致
```

## 跳过的内容

- `goose_db_version*` 两张表（goose 启动期自建）
- TimescaleDB 内部 catalog 的 `metadata` 表数据（CREATE EXTENSION 时自动装填）

## 相关位置

- 完整迁移规范：`omcgo/CLAUDE.md` §5.5
- 迁移命令封装：`omcgo/cmd/migrate/main.go`
- compose 中的 migrate 服务：`deployments/docker/docker-compose.yml`（`migrate-schema` + `migrate-seed`）
- 历史归档：`omcgo/migrations.backup-20260531/`（gitignore）
