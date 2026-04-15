# Code Review: product_type 字典值改数字编码 + 清理旧命令

**Date**: 2026-04-15
**Reviewer**: Claude (automated)
**Scope**: mml/migration
**Conclusion**: PASS

## Summary

product_type 字典 value 改为数字编码（1-6），label 保持实际产品型号名；清理 000007 遗留的 config/maintenance/query 旧命令。

## Findings

| # | Severity | File | Description |
|---|----------|------|-------------|
| 1 | INFO | 900004 | product_type 字典 value 从 product_class 字符串改为数字编码，与 mml_command_category 模式一致 |
| 2 | INFO | 900004 | 新增 DELETE 清理 000007 遗留的 3 条旧命令（config/maintenance/query category），通过 command_code 定位 |

## Checklist

- [x] SQL 安全：无字符串拼接，参数化插入
- [x] 幂等性：DELETE + INSERT 确保可重复执行
- [x] 清理完整：command_code IN ('LST_DEVPARAM','SET_DEVPARAM','RST_DEV') 覆盖全部旧命令
