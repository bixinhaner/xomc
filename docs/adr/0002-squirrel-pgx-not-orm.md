# 0002 — 数据库访问用 Squirrel + pgx，不用 ORM

## Status

Accepted（追认 retroactive）。决策于项目立项期落地，本目录成立时补记。

## Context

OMC 是商用级运营商网管，业务数据量大（10 万 → 100 万基站）、查询形态多样：既有简单 CRUD，也有跨设备/站点/区域的多级聚合、批量写入、复杂过滤分页。数据层选型需要在以下方案中取舍：

- **ORM（GORM 等）**：开发快、CRUD 样板少，但对复杂查询的表达力受限，常退化为字符串拼接或难以预测的 SQL；隐藏的 N+1、自动迁移、反射开销在高写入/热路径上是隐患；SQL 不透明导致排障困难。
- **裸 SQL 字符串拼接**：完全可控但易出 SQL 注入、可读性差、重复样板多。
- **SQL 构建器（Squirrel）+ 原生驱动（pgx）**：参数化构建中等复杂度查询，复杂查询直接写 SQL，驱动层零 ORM 抽象、性能可控。

底层数据库为 PostgreSQL 16（+ TimescaleDB 时序扩展），UUID 主键、JSONB 扩展字段，需要充分用到 PG 原生能力（`ON CONFLICT`、部分索引、超表 hypertable、advisory lock 等），ORM 的方言抽象反而是阻碍。

## Decision

1. **构建 SQL 用 `github.com/Masterminds/squirrel`**：简单与中等复杂度查询用 Squirrel 参数化构建（含批量 INSERT/UPDATE），杜绝字符串拼接，天然防 SQL 注入。
2. **执行用 `github.com/jackc/pgx/v5` + `pgxpool`**：原生 PostgreSQL 驱动 + 连接池，直接面对 SQL，无 ORM 中间层。
3. **复杂查询直接写 SQL**：多级聚合、复杂 JOIN 等用手写 SQL（放在模块 repository 层或 `queries/`），不强行用构建器表达。
4. **分层落地**：每个模块自带 repository 层（`pg_repository.go`），遵循 handler → service → repository 分层；写操作在事务内、正确处理回滚。
5. **明令禁止 ORM**：根 `CLAUDE.md §8.2 / §10`、`omcgo/CLAUDE.md §4.1` 将"禁止 ORM、禁止字符串拼接 SQL"列为硬规范。

## Consequences

**正向**：

- SQL 完全透明可审、可 EXPLAIN，热路径（ACS、PM 写入、聚合查询）性能可控，无反射开销。
- 充分使用 PG 原生特性：`ON CONFLICT DO NOTHING`/`DO UPDATE` 做幂等、部分唯一索引、TimescaleDB 超表、`pg_advisory_lock` 等，不受 ORM 方言抽象限制。
- 参数化构建消除 SQL 注入面；批量操作用 Squirrel 批量语句避免逐条往返。
- 数据层依赖少、可测试性好（repository 接口可 mock，SQL 可独立验证）。

**代价 / 约束**：

- 简单 CRUD 也要手写映射与 repository 样板，无 ORM 的自动生成红利，初期代码量更大。
- 没有 ORM 的自动迁移：schema 演进全靠 goose 迁移人工维护（版本号连续、up/down 配对、幂等），纪律要求高（见 `omcgo/CLAUDE.md §4.6`）。
- 团队需具备直接写 SQL 的能力，复杂查询的正确性靠测试与 review 保障，不能依赖框架兜底。

> 规范见根 `CLAUDE.md §8.2`、`omcgo/CLAUDE.md §4.1`。依赖 import 路径见 `omcgo/CLAUDE.md §2`，存储选型见根 `CLAUDE.md §12`。
