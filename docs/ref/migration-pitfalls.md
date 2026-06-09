# 数据库迁移踩坑案例库

> 从 `omcgo/CLAUDE.md §5.5` 抽出的详细历史教训。CLAUDE.md 只保留核心铁律 + 自查清单，每条规则的「为什么 + 翻车 commit」在此。
> 这些是参考资料（出问题时回查），不是每次编码都要读的常驻上下文。

## 1. goose StatementBegin/End 注解（CRITICAL）

goose 默认以分号分割 SQL 语句。以下场景 **必须** 用 `-- +goose StatementBegin` / `-- +goose StatementEnd` 包裹，否则 goose 会把内部语句截断导致解析失败：

| 场景 | 示例 |
|------|------|
| PL/pgSQL 匿名块 | `DO $$ ... END $$;` |
| 创建函数 | `CREATE OR REPLACE FUNCTION ... $$ ... $$ LANGUAGE plpgsql;` |
| 创建触发器函数 | 同上 |
| 循环/条件语句 | `FOR ... IN ... LOOP ... END LOOP;` |

```sql
-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        CREATE EXTENSION IF NOT EXISTS timescaledb;
    END IF;
END $$;
-- +goose StatementEnd
```

> **历史教训**：6 个迁移文件因缺少此注解导致 goose panic（commit `05e0d6f`）。

## 2. TimescaleDB 迁移：先启用压缩再建压缩策略

```sql
-- 1. 创建 hypertable
SELECT create_hypertable('table_name', 'time_column');
-- 2. 启用压缩（必须在压缩策略之前）
ALTER TABLE table_name SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'column_name',
    timescaledb.compress_orderby = 'time_column DESC'
);
-- 3. 添加压缩策略
SELECT add_compression_policy('table_name', INTERVAL '7 days');
```

> **历史教训**：2 个 hypertable 因缺步骤 2 报 `columnstore not enabled`（commit `dcd4870`）。

## 3. 分区表外键限制

PostgreSQL **不支持** 对分区表建立外键引用（分区表主键必须包含分区键，外键目标必须是唯一约束列）。分区表间关联用应用层保证一致性。

```sql
-- 错误：devices 是分区表，不支持外键
-- CONSTRAINT fk_device FOREIGN KEY (device_sn) REFERENCES devices(serial_number)
-- 正确：仅逻辑引用，应用层校验
device_sn VARCHAR(64) NOT NULL  -- 逻辑引用 devices.serial_number
```

> **历史教训**：device_tasks 因外键引用分区表失败（commit `c0adfa4`）。

## 4. INSERT 语句与表 Schema 一致性

迁移文件中的 `INSERT` **必须** 与同目录 DDL 定义的实际表结构完全匹配：先确认目标表实际列定义；所有 NOT NULL 列必须提供值；不能引用不存在的列；区分 `''` 与 `NULL`（注意 CHECK 约束）；JSON 字符串无尾随逗号。

> **历史教训**：2 次 INSERT 与 Schema 不匹配导致迁移失败（commits `c2e820c`, `985e469`）。

## 5. CREATE TABLE 内不支持部分唯一约束

PostgreSQL `CREATE TABLE` 内联 `CONSTRAINT ... UNIQUE(...) WHERE ...` 不被支持。带 WHERE 的部分唯一约束必须用独立 `CREATE UNIQUE INDEX`：

```sql
CREATE TABLE t (name VARCHAR(100), deleted_at TIMESTAMPTZ);
CREATE UNIQUE INDEX uniq_name ON t(name) WHERE deleted_at IS NULL;
```

> **历史教训**：字典表迁移因此语法错误失败（commit `fce3a65`）。

## 6. UUID 格式校验

PostgreSQL UUID 类型严格校验格式（8-4-4-4-12），最后一段固定 12 个十六进制字符。用 `gen_random_uuid()` 或经校验的硬编码 UUID。批量插入前可正则验证。

> **历史教训**：菜单表种子 UUID 最后一段仅 11 字符导致插入失败（commit `0ba37b0`）。

## 7. 种子数据与 DDL 迁移去重

DDL 迁移文件与 `seed/` 种子文件中的初始数据会重叠：两者的 INSERT 都用 `ON CONFLICT DO NOTHING` 保证幂等；**禁止** 在 DDL 迁移中放不带 `ON CONFLICT` 的 INSERT。

## 8. Down 迁移完整性

Down 必须清除 Up 创建的 **所有** 对象（表/函数/触发器/索引/类型/迁移专属扩展）。共享函数（如 `update_updated_at_column()`）在 `000001` 创建，后续迁移不重复创建；如需确保存在用 `CREATE OR REPLACE FUNCTION`。

## 9. TRUNCATE 与外键约束

**被 FK 引用的表不能单独 TRUNCATE**，即使引用方为空、即使声明了 `ON DELETE CASCADE`。两条独立 TRUNCATE 会被拒（`SQLSTATE 0A000`）。要清空被引用表必须二选一：

```sql
-- 方案 A（推荐）：同一条语句 truncate 所有相关表
TRUNCATE alarm_library_i18n, alarm_libraries RESTART IDENTITY;
-- 方案 B：CASCADE 递归
TRUNCATE alarm_libraries CASCADE;
```

更优：种子数据用 `INSERT ... ON CONFLICT (key) DO UPDATE SET ...` upsert，根本不需要 truncate。

> **历史教训**：`seed/000028_alarm_library_import.sql` 用两条独立 TRUNCATE，在有数据环境下 migrate-seed exit 1 → acs/app/worker 因 depends_on 全起不来（commit `7afc3241`）。（注：seed 编号在 2026-05-31 consolidation 后已重排，此 commit 为当时编号。）

## 10. Schema 演进不破坏既有种子数据（CRITICAL）

**根因**：`migrate-seed` 用独立版本表 `goose_db_version_seed` 且 `depends_on migrate-schema`，执行模型是「先全部 schema，再全部 seed」。因此**每个旧 seed 文件总在「被后续所有 DDL 改过的最终表结构」上执行**。改 schema 时只要破坏任一既有 seed 前提，全新库/存量库重跑就失败。

**五种破坏场景与安全做法**：

| schema 变更 | 旧 seed 为何失败 | 安全做法 |
|------------|----------------|---------|
| 删列 | INSERT 引用不存在列 | 两阶段：先废弃（保留列）→ 确认无引用 → 下个 release 再 DROP |
| 加无默认 NOT NULL 列 | 旧 seed 不给该列 | 直接带 DEFAULT；或加可空列 → UPDATE backfill → 再加 NOT NULL |
| 收紧 CHECK/UNIQUE/FK | 旧 seed 值不满足新约束 | 同一迁移内先 UPDATE 修存量数据，再加约束 |
| 改列名 | seed 引用旧列名 | 先加新列双写 → 迁移 seed/代码 → 再删旧列（跨 release） |
| 改类型 | seed 字面值无法隐式转换 | `ALTER ... TYPE ... USING <转换>`，确认 seed 值可转 |

**五条铁律**：
1. 新增列一律可空或带 `DEFAULT`，绝不裸加「无默认 NOT NULL」列。
2. 收紧约束前，同一迁移内先 backfill 修存量数据（含 seed 行），再加约束。
3. 删列/改名走两阶段，跨 release 完成。
4. 已 applied 的旧 seed 文件内容不回头改（改了触发 checksum 漂移）——兼容责任放在新 schema 迁移里。
5. seed 侧防御：显式列名 + `ON CONFLICT` + 只插稳定核心列（易变列交给 DEFAULT）。

**治本建议**：CI 应起全新 postgres 真跑 `migrate-schema-up && migrate-seed-up`，并补「存量库重跑」场景，让「旧 seed × 新结构」不兼容在 PR 就 fail。

> **历史教训**：`f9565e1a` —— `000004` pm_tasks 的 `dimension` CHECK 漏了 `'network'`，存量库重跑时既有数据/seed 不满足 CHECK 直接失败。
