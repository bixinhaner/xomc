# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-03-18 |
| 作者 | chenbo01 |
| 基准 | 4304d1f |
| Scope | migration |
| Type | fix |
| 文件数 | 2 |

## 变更概述

修复 TimescaleDB 迁移报错 `columnstore not enabled on hypertable`。在 `add_compression_policy` 前添加 `ALTER TABLE SET (timescaledb.compress, ...)` 启用压缩功能。

## 审查结果

**结论: PASS**

### 检查项

| # | 检查项 | 结果 | 说明 |
|---|--------|------|------|
| 1 | SQL 正确性 | ✅ | `ALTER TABLE SET (timescaledb.compress)` 是新版 TimescaleDB 启用压缩的标准语法 |
| 2 | segmentby 列选择 | ✅ | pm_counters: device_id+cell_id+counter_group+counter_name; mr_records: device_id+file_id+mr_type |
| 3 | 执行顺序 | ✅ | create_hypertable → enable compress → add_compression_policy → add_retention_policy |
| 4 | 条件保护 | ✅ | 在 `IF EXISTS timescaledb` 块内，无扩展时安全跳过 |

### 零 CRITICAL / 零 WARNING
