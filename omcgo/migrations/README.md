# Database Migrations

迁移工具：[`pressly/goose/v3`](https://github.com/pressly/goose)。版本号记录在数据库 `goose_db_version`（DDL）和 `goose_db_version_seed`（DML）两个表中，由 `cmd/migrate` 包装执行。

## 当前状态：consolidated baseline（2026-05-31）

历史上累积了 **185 个 schema 迁移 + 92 个 seed 迁移**（版本号从 000001 一路跳到 000219），版本号断层、命名风格不一、累积 SQL 体量大。本次重置后只有两个 baseline 文件：

| 文件 | 大小 | 说明 |
|------|------|------|
| `000001_init_schema.sql` | ~580 KB | 全量表结构（206 张 public 表、6 个扩展、73 个触发器、若干函数、7 个 TimescaleDB hypertable + 5 个压缩策略） |
| `seed/000001_init_seed.sql` | ~17 MB | 全量种子数据（admin/角色/菜单/权限 + MML 字典 + standard_params + seed devices + zambia 2000 devices + 学习到的 param mappings + indicator definitions + license/northbound 默认） |

旧的 277 个迁移文件归档在 `omcgo/migrations.backup-20260531/`（已 gitignore，仅本地保留，可随时回滚）。

## 新增迁移的版本号规则

- **当前 goose schema 版本 = 1**，下一个新增迁移用 `000002_xxx.sql`
- **当前 goose seed 版本 = 1**，下一个新增 seed 用 `seed/000002_xxx.sql`
- 其它规则（连续递增、StatementBegin/End、TimescaleDB 压缩顺序、分区表外键、UUID 校验、Down 完整性、TRUNCATE/FK、自查清单）见 `omcgo/CLAUDE.md` §5.5 数据库迁移规范

> CLAUDE.md §5.3 和 §5.3.1 中提到的 `migrations/000057`、`000058`、`000063`、`000215`、`000217` 等具体编号在本次合并后**已不存在**，对应能力已并入 `000001_init_schema.sql`，文档中的编号只用作历史溯源参考。

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
