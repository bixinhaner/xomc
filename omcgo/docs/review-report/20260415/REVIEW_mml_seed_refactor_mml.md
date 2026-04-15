# Code Review: MML 种子数据重构

**Date**: 2026-04-15
**Reviewer**: Claude (automated)
**Scope**: mml/migration
**Conclusion**: PASS

## Summary

MML 种子数据重构：命令分类从中文改为数字编码，产品类型字典值对齐实际设备数据。

## Files Changed

| File | Lines Changed | Risk |
|------|--------------|------|
| `migrations/seed/900004_mml_enhance.sql` | +12/-9 | LOW |

## Findings

| # | Severity | File | Description |
|---|----------|------|-------------|
| 1 | INFO | 900004_mml_enhance.sql | product_type 字典值从 PM-B4860/QAFA 等改为 SmallCell-LTE/gNB-100 等实际 product_class 值，确保前后端筛选一致 |
| 2 | INFO | 900004_mml_enhance.sql | mml_command_category 字典 value 从中文改为数字 (1-7)，label 保持中文用于前端展示 |
| 3 | INFO | 900004_mml_enhance.sql | 字典详情使用 DELETE + INSERT 模式确保幂等更新 |

## Checklist

- [x] SQL 安全：使用参数化插入，无字符串拼接
- [x] 幂等性：ON CONFLICT DO NOTHING + DELETE/INSERT 确保可重复执行
- [x] 数据一致性：category 数字值 1-7 与字典 value 一一对应
- [x] 向后兼容：goose Down 脚本已覆盖清理逻辑
