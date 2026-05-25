# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-05-25 11:12 |
| 提交 | c04b7577 |
| 作者 | zhanglu |
| 范围 | alarm |
| 变更文件数 | 2 |
| 新增行数 | +10 |
| 删除行数 | -2 |

## 变更概要

本次变更修复了当前告警和历史告警页面导出成功提示的国际化占位参数遗漏问题。原实现调用 `common.exportSuccess` 时未传入 `count`，导致页面直接显示 `{count}` 占位符；修复后改为按实际导出条数展示成功提示。

## 审查发现

### 🔴 CRITICAL (严重)

无

### 🟡 WARNING (警告)

无

### 🔵 INFO (建议)

- 已有 `common.exportSuccess` 文案依赖 `count` 占位，本次两个告警页面都已对齐，后续其他导出入口若复用同文案，也应统一传入 `count`。

## 详细分析

### `omcmb/webcode/src/pages/alarm/HistoricalAlarms/index.tsx`

- [HistoricalAlarms 导出成功提示](omcmb/webcode/src/pages/alarm/HistoricalAlarms/index.tsx#L442)：`t('common.exportSuccess', { count: exportItems.length })` 修复了历史告警导出成功消息中 `{count}` 未插值的问题。

### `omcmb/webcode/src/pages/alarm/CurrentAlarms/index.tsx`

- [CurrentAlarms 导出成功提示](omcmb/webcode/src/pages/alarm/CurrentAlarms/index.tsx#L521)：同样补齐 `count` 参数，保证当前告警和历史告警的导出反馈行为一致。

## 业务完整性检查

- Handler-Service-Repository 链路：未涉及
- 路由注册：未涉及
- 迁移文件配套：未涉及
- API 服务配套：未涉及
- Hook 配套：未涉及
- Mock 配套：未涉及
- E2E 测试用例：本次为前端提示修复，未新增接口

## 验证记录

- `npm run typecheck`：通过

## 审查结论

PASS
