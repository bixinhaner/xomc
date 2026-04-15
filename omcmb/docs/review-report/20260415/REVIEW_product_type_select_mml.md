# Code Review: DeviceTree 产品类型 Select 适配数字编码字典

**Date**: 2026-04-15
**Reviewer**: Claude (automated)
**Scope**: mml
**Conclusion**: PASS

## Summary

DeviceTree 产品类型 Select 适配字典 value 为数字编码后的 value↔label 双向映射。

## Findings

| # | Severity | File | Description |
|---|----------|------|-------------|
| 1 | INFO | DeviceTree.tsx | Select value 通过 label→value 反查匹配选中态，onChange 通过 value→label 正查将实际 product_class 传给筛选器 |

## Checklist

- [x] 类型安全：无 any
- [x] 字典 value 与 label 转换正确
- [x] 筛选发送的是 label（实际 product_class），后端可正确匹配
