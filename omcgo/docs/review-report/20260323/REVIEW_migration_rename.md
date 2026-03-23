# 代码审查报告

| 字段 | 值 |
|------|-----|
| **日期** | 2026-03-23 |
| **审查者** | Claude Code |
| **Scope** | migration |
| **Type** | fix |
| **结论** | PASS |

## 变更概述

修复迁移文件版本号冲突，将 `000034_create_pm_files` 重命名为 `000036_create_pm_files`。

## 变更文件

| 文件 | 变更类型 | 说明 |
|------|----------|------|
| `migrations/000034_create_pm_files.up.sql` | 重命名 | → `000036_create_pm_files.up.sql` |
| `migrations/000034_create_pm_files.down.sql` | 重命名 | → `000036_create_pm_files.down.sql` |

## 问题分析

**原问题**:
```
duplicate migration file: 000034_create_pm_files.down.sql
```

**根本原因**: 存在两个版本号为 `000034` 的迁移：
- `000034_add_device_stun_fields` (Mar 23)
- `000034_create_pm_files` (Mar 12)

**修复方案**: 将 `000034_create_pm_files` 重命名为 `000036_create_pm_files`（填补空缺版本号）

## 检查清单

| 检查项 | 结果 |
|--------|------|
| 版本号唯一 | ✅ |
| up/down 配对 | ✅ |
| 文件内容不变 | ✅ |

## 发现汇总

| 级别 | 数量 |
|------|------|
| CRITICAL | 0 |
| WARNING | 0 |
| INFO | 0 |

## 审查结论

**PASS** - 版本号冲突已解决。
