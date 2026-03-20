# 代码审查报告

| 字段 | 值 |
|------|-----|
| 审查时间 | 2026-03-20 16:00 |
| 审查范围 | migration |
| 变更文件 | 1 个文件 |
| 审查结论 | ✅ PASS |

---

## 变更概述

修复数据库迁移文件中 INSERT 语句与表 Schema 不匹配的问题。

### 变更文件

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `migrations/000046_add_default_data_models.up.sql` | 修复 | 修正 INSERT 语句列名与表定义匹配 |

---

## 问题分析

**原始错误**:
```
pq: column "name" of relation "data_model_definitions" does not exist
```

**根本原因**:
迁移文件使用了 `name` 列，但表 `data_model_definitions` (定义于 000003) 中不存在此列。

---

## 修复内容

| 修复项 | 修复前 | 修复后 |
|--------|--------|--------|
| 移除不存在的列 | `name` | 已移除 |
| 添加必需列 `version` | 缺失 | `'1.0'` |
| 添加必需列 `is_active` | 缺失 | `true` |
| NULL 值处理 | `''` (空字符串) | `NULL` |
| parameter_tree | `'{"version": "1.0"}'::jsonb` | `'{}'::jsonb` |

---

## 约束验证

表 `data_model_definitions` 的 `chk_scope_fields` 约束要求：

| Scope | oui | product_class | 验证 |
|-------|-----|---------------|------|
| carrier_default | NULL | NULL | ✅ |
| oui | NOT NULL | NULL | ✅ |
| product | NOT NULL | NOT NULL | ✅ |

---

## 审查结论

**✅ PASS**

变更正确修复了迁移文件与表 Schema 不匹配的问题，可以提交。

---

*审查人: Claude AI*
*审查工具版本: Smart Commit v1.0*
