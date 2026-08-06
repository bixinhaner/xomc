# 迁移 Baseline 运维手册

本文承接 `omcgo/migrations/README.md` 中不适合放在日常入口里的低频操作说明：既有库升级约束、连接池核定、封版本后追加迁移规则，以及未来重生 baseline 的标准流程。

## 既有库升级约束

当前三个 `000001` baseline 只面向全新发布部署或允许清库重建的环境。既有库的 `goose_db_version*` 可能仍记录已经删除的历史增量版本，不能靠简单 `goose up` 升级，也不要仅重置版本表后在非空库上重跑 baseline。

需要保留数据的环境必须先备份。对于纯新增且已放入 `MainReconcile` 标记区段的主库
schema/seed 变化，迁移程序会在既有 `migrations/seed` 流程中自动、事务化、幂等协调；
协调失败时业务容器不会启动。未进入标记区段的变化，以及删除、改名、类型收紧或复杂
数据回填，仍需另行设计和验证升级迁移；无法证明安全时，支持路径仍是备份所需数据、
重建数据库，再从三个 `000001` baseline 初始化。

## 连接池核定

三个部署单元的 `cmd/*/etc/config.{test,prod}.yaml` 中，`max_conns` 核定如下；dev/local 配置不按此处调整。

| 单元 | db（PG）max_conns | tsdb（TS）max_conns |
|------|------------------|---------------------|
| app | 60 | 40 |
| acs | 30 | 无 tsdb 块 |
| worker | 25 | 25 |

单副本聚合预算为 `app(60+40)+acs(30)+worker(25+25)=180`，要求 PostgreSQL `max_connections >= 200`。app/acs 水平扩展后连接数线性累加，逼近上限前须前置 pgbouncer（transaction pooling）；这是 100 万级扩展的既定演进项，当前仓库暂不引入。

## 封版本后追加迁移规则

三条流是相互独立的 goose 版本序列，记在不同版本表，不共享号段。因此 `000001` 在 `migrations/`、`migrations/seed/`、`migrations/tsdb/` 各出现一次是正常的。

- 新增迁移 = 该流现有最大号 + 1。当前三条流在封版本后的下一号均为 `000002`；封版本前不得使用该号段。
- DDL 放 `omcgo/migrations/`，DML 种子放 `omcgo/migrations/seed/`，时序 DDL 放 `omcgo/migrations/tsdb/`。
- `DO $$`、`CREATE [OR REPLACE] FUNCTION`、循环条件必须用 goose `StatementBegin/End` 包裹。
- 幂等写法优先使用 `CREATE TABLE IF NOT EXISTS`、`ADD COLUMN IF NOT EXISTS`、`INSERT ... ON CONFLICT DO NOTHING`。
- Down 段删除 Up 段创建的所有对象。
- 改已被 seed 引用的表时，新增列应可空或带 `DEFAULT`；收紧约束前同一迁移先 backfill；删列或改名走两阶段跨 release。

```bash
# 取某流下一号（seed 示例）
ls migrations/seed/[0-9]*.sql | xargs -n1 basename | sed 's/_.*//' | sort -n | tail -1

# 撞号自查：三流各查一遍，应都为空
ls migrations/[0-9]*.sql      | xargs -n1 basename | sed 's/_.*//' | sort | uniq -d
ls migrations/seed/[0-9]*.sql | xargs -n1 basename | sed 's/_.*//' | sort | uniq -d
ls migrations/tsdb/[0-9]*.sql | xargs -n1 basename | sed 's/_.*//' | sort | uniq -d
```

## 重生 baseline 的标准流程

未来如果再次需要 consolidate，把积累的增量折回单个 `000001`，按下面流程执行。示例以 `OMCGO_ENV=test` 为准。

```bash
# 1. 备份，backup-* 已 gitignore
cp -r omcgo/migrations omcgo/migrations.backup-$(date +%Y%m%d)

# 2. 在仓库根 xomc/ 下执行，干净库跑完全部现有迁移，先去掉任何撞号
export COMPOSE="docker compose -f deployments/docker/docker-compose.yml"
$COMPOSE down -v
$COMPOSE up -d --wait postgres postgres-tsdb redis nats minio
$COMPOSE up -d migrate-schema migrate-seed migrate-tsdb-schema

# 3. 先记录 golden 逐表行数，作为后续验证基准

# 4. 从容器内 pg_dump 主库 schema
$COMPOSE exec -T postgres pg_dump -U omcgo -d omcgo \
  --schema-only --no-owner --no-privileges --no-tablespaces \
  --no-publications --no-subscriptions \
  --exclude-table='goose_db_version*' > /tmp/raw_schema.sql

# 5. 从容器内 pg_dump 主库 seed，保留幂等 INSERT
$COMPOSE exec -T postgres pg_dump -U omcgo -d omcgo \
  --data-only --inserts --rows-per-insert=500 --on-conflict-do-nothing \
  --no-owner --no-privileges --disable-triggers \
  --exclude-table='goose_db_version' --exclude-table='goose_db_version_seed' > /tmp/raw_data.sql
```

## pg_dump 后处理细节

pg_dump 输出不能直接塞进 goose 文件，至少要做这些处理：

1. 删除 psql 元命令 `\restrict` / `\unrestrict`。pgx Exec 不识别这些命令，新版 pg_dump 可能会输出。
2. 修正 `search_path`。pg_dump 顶部的 `set_config('search_path','',...)` 会导致 goose 写版本表时报 `42P01`。schema dump 末尾通常自带 `public` 兜底；seed dump 如果没有，必须在 goose Down 前补一行：

```sql
SELECT pg_catalog.set_config('search_path','public',false);
```

3. 主库保持纯 PostgreSQL 16，没有 TimescaleDB 扩展和 `_timescaledb_catalog`，不需要相关处理。
4. 用 goose Up/Down 包裹最终 SQL。schema 的函数块必须使用 `StatementBegin/End`；schema Down 可用 `DROP SCHEMA public CASCADE;`；seed 无回滚时 Down 写 `SELECT 1;`。
5. 写回 `omcgo/migrations/000001_init_schema.sql`、`omcgo/migrations/seed/000001_init_seed.sql`，删除旧增量文件。

## 重生后的验证

重生 baseline 后必须重新清库验证：

- `$COMPOSE down -v` 后重新启动依赖服务。
- `migrate-schema`、`migrate-seed`、`migrate-tsdb-schema` 三条流都成功退出。
- 逐表行数与 golden 记录对比，差异为 0。
- 需要页面验收时，使用真实浏览器检查页面，不用命令行请求替代页面验证。
