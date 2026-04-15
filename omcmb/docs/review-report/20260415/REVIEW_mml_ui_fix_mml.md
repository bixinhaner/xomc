# Code Review: MML 控制台 UI 修复与优化

**Date**: 2026-04-15
**Reviewer**: Claude (automated)
**Scope**: mml
**Conclusion**: PASS

## Summary

MML 控制台 UI 修复：修复 pageSize 超限 400 错误、设备类型 Tag 间距、命令分类展示映射、产品类型筛选空结果等问题。

## Files Changed

| File | Lines | Risk | Description |
|------|-------|------|-------------|
| `src/pages/mml/CommandTree/index.tsx` | +22/-16 | LOW | category 数字映射为中文 label 显示 |
| `src/pages/mml/Console/components/CommandTree.tsx` | +5/-1 | LOW | 树节点 title 通过 catLabelMap 映射 |
| `src/pages/mml/Console/components/DeviceTree.tsx` | +8/-5 | LOW | Tag 间距 + 已选设备区域优化 |
| `src/pages/mml/Console/hooks/useDeviceSelection.ts` | +2/-1 | LOW | networkType 映射 lte→eNB/nr→gNB + 筛选翻页重置 |
| `src/pages/alarm/AlarmRules/AlarmRuleDrawer.tsx` | +1/-1 | LOW | pageSize 1000→100 |
| `src/pages/topology/GISMapView/index.tsx` | +1/-1 | LOW | pageSize 10000→100 |
| `src/services/api/apiPermissionApi.ts` | +11/-3 | LOW | 改为分页循环拉取 |
| `src/services/api/mmlApi.ts` | +14/-3 | LOW | getAllCommands 改为分页循环 |
| `src/services/api/pmApi.ts` | +1/-1 | LOW | pageSize 1000→100 |
| `src/services/api/softwareApi.ts` | +2/-2 | LOW | pageSize 200→100 |

## Findings

| # | Severity | File | Description |
|---|----------|------|-------------|
| 1 | INFO | 多文件 | 所有 API 调用 pageSize 已修复为 ≤100，符合后端 max=100 校验 |
| 2 | INFO | DeviceTree.tsx | device.type Tag 添加 marginLeft:auto + flexShrink:0 防止右侧溢出 |
| 3 | INFO | useDeviceSelection.ts | networkType 映射改为显式判断 lte→eNB/nr→gNB，不再依赖 fallback |
| 4 | INFO | CommandTree (两个) | 树节点通过字典 value→label 映射显示中文分类名 |

## Checklist

- [x] 类型安全：无 any 使用
- [x] API 模式：分页循环符合 pageSize ≤100 约束
- [x] Hook 模式：useEffect 依赖正确
- [x] XSS：无 dangerouslySetInnerHTML
- [x] 无硬编码：分类显示通过字典映射
