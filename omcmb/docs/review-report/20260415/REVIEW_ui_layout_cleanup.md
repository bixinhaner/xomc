# Code Review: UI 布局优化与精简

| 字段 | 值 |
|------|-----|
| 日期 | 2026-04-15 |
| 审查人 | AI Code Reviewer |
| 变更文件数 | 15 |
| 审查结论 | **PASS_WITH_WARNINGS** |

---

## 变更概述

移除页面中多余的 Card 包裹层，简化布局结构；微调表格高度和字体大小；补充告警库 Mock 数据；新增 i18n key。

---

## 详细审查

### WARNING（3 项）

#### W1: AlarmSupportLibrary scroll.y 值硬编码
- **文件**: `src/pages/alarm/AlarmSupportLibrary/index.tsx:550`
- **问题**: `scroll.y: 'calc(100vh - 350px)'` 硬编码像素偏移，不同屏幕/字体下可能不准
- **建议**: 可接受，项目内统一使用此模式，后续可考虑提取为常量

#### W2: 告警 Mock 数据字段拼写 `serverityType`
- **文件**: `src/pages/alarm/AlarmSupportLibrary/index.tsx`（新增的 id 11-20 条目）
- **问题**: 字段名 `serverityType` 应为 `severityType`，但这与已有数据（id 1-10）保持一致，属于历史遗留
- **建议**: 不在本次修复范围内，后续可统一修正

#### W3: fresh-effects.css 移除 Card hover 阴影
- **文件**: `src/styles/fresh-effects.css`
- **问题**: 移除了 `.ant-card:hover` 阴影效果，影响所有使用 fresh 主题的 Card
- **建议**: 符合"简约化"设计方向，可接受

### INFO（2 项）

#### I1: `device.serialNumber` i18n 值从"小站编码"改为"基站编码"
- **文件**: `src/i18n/zh-CN/index.ts:490`
- **说明**: 术语统一，符合业务规范

#### I2: DeviceListPanel 移除 `copyable: true`
- **文件**: `src/pages/device/DeviceGrouping/DeviceListPanel.tsx:271`
- **说明**: 移除 SN 列的复制功能，简化交互

---

## 变更范围

| 模块 | 文件 | 变更类型 |
|------|------|---------|
| components | DataTable.module.css | 字号 12→13px |
| alarm | AlarmSupportLibrary | 新增 Mock 数据 + scroll 调整 |
| alarm | CurrentAlarms | Card flex 布局 + scroll 调整 |
| alarm | CustomAlarmStats | FilterBar 包裹层 + margin 调整 |
| alarm | HistoricalAlarms | Card flex 布局 + scroll 调整 |
| backup | BackupTasks | 移除多余 Card 包裹 |
| backup | RestoreData | 移除多余 Card 包裹 |
| device | DeviceListPanel | 移除 copyable + 字号样式 |
| performance | KPIQuery | 移除分割线 + hideAdd + scroll 调整 |
| software | FirmwareUpload | 移除多余 Card 包裹 |
| software | UpgradePlan | 移除多余 Card 包裹 |
| software | VersionRollback | 移除多余 Card 包裹 |
| styles | fresh-effects.css | 移除 Card hover 阴影 |
| i18n | zh-CN / en-US | 新增 key + 术语修正 |

---

## 审查门禁

- [x] 无 CRITICAL 级别问题
- [x] 无安全风险（XSS/注入/敏感信息泄露）
- [x] 无类型安全问题（无 `any` 引入）
- [x] 变更范围合理，符合"简约化"设计方向
