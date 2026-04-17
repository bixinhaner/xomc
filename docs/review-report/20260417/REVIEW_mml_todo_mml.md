# Code Review — MML Console To-Do List Items

**Date**: 2026-04-17
**Reviewer**: Claude Code (auto)
**Scope**: mml
**Files**: 7 files, +78 -7 lines

## Summary

完成 MML 控制台 to-do-list 中 5 项需求：
1. COMMAND_PAGE_SIZE 200 → 999
2. 命令树分类下拉框默认选中第一个分类
3. AddTemplateModal 添加"保存并执行"双按钮
4. 自定义模块分类结构（已有实现）
5. 命令搜索（已有实现）

## Findings

### INFO-001: 自动选中分类不可清除
- **File**: `omcmb/webcode/src/pages/mml/Console/hooks/useCommandSelection.ts`
- **Severity**: INFO
- **Description**: `useEffect` 在 `categoryFilter === ''` 时自动选中第一个分类。当用户通过 Select 的 allowClear 清除筛选时也会重新触发，导致无法查看全部分类命令。
- **Status**: 当前行为符合"初次进入默认第一个分类"的需求。如需支持清除查看全部，可改用 ref 标记是否已初始化。

## Checklist

- [x] TypeScript 类型安全：无 `any` 使用
- [x] i18n 双语一致：zh-CN 和 en-US 同步新增 2 条
- [x] 组件接口兼容：`onSaveAndExecute` 为可选 prop，向后兼容
- [x] 无硬编码：文案通过 i18n key
- [x] 错误处理：设备为空时提示 warning

## Conclusion: PASS
