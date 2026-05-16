# Code Review Report

| 项目 | 值 |
|------|-----|
| 日期 | 2026-05-15 09:38 |
| 提交 | efce14b7 |
| 作者 | zhanglu |
| 范围 | dashboard-product |
| 变更文件数 | 4 |
| 新增行数 | +267 |
| 删除行数 | -91 |

## 变更概要

本次变更将仪表盘最近告警区从静态展示切到当前告警 Hook 数据源，并重构了告警库页面的 unknown/fallback 展示与筛选交互。同时补充了告警规则页的 Playwright 场景覆盖，并修正了登录辅助器对中文占位符的兼容性。

## 审查发现

### 🔴 CRITICAL (严重)

无

### 🟡 WARNING (警告)

无

### 🔵 INFO (建议)

- omcmb/webcode/e2e/alarm.spec.ts:62,75,89
  告警规则页新增了新增弹窗校验、仅允许编辑禁用规则、创建规则三条端到端用例，覆盖面比之前的 smoke test 明显更完整。
- omcmb/webcode/src/pages/dashboard/index.tsx:102,129,312
  最近告警区已改为消费 useCurrentAlarms，并把列表 loading 态切到真实查询状态，方向正确，减少了页面与业务层数据源漂移。

## 详细分析

### omcmb/webcode/src/pages/product/alarm-library/index.tsx

- INFO: omcmb/webcode/src/pages/product/alarm-library/index.tsx:83,108-109,250,256
  unknownFallbackItems 与 tableItems/tableLoading 的拆分让 unknown-only 模式可以复用同一张表格；识别状态下拉现在也会把“已识别”正确映射到 isUnknown=false，筛选文案与查询参数已一致。

### omcmb/webcode/src/pages/dashboard/index.tsx

- INFO: omcmb/webcode/src/pages/dashboard/index.tsx:37,102,129-137
  最近告警列表已从硬编码 mock 数据切到 useCurrentAlarms 输出，并通过 formatAlarmTime 做了展示层格式化，属于合理的页面收敛。

### omcmb/webcode/e2e/alarm.spec.ts

- INFO: omcmb/webcode/e2e/alarm.spec.ts:62-103
  新增用例与现有 mock 行为一致，特别是创建规则后翻到第 2 页校验，规避了 mock getRules 仅分页不筛选的实现差异，断言是可信的。

### omcmb/webcode/e2e/helpers/auth.ts

- INFO: omcmb/webcode/e2e/helpers/auth.ts:24-25
  登录 helper 兼容中文占位符后，mock 模式下的 E2E 运行更稳定，避免了 UI 文案语言切换导致的选择器脆弱性。

## 业务完整性检查

- 页面层变更使用的仍是现有 frontend-core Hook，没有引入新的 API/Hook/Mock 配套缺口。
- 告警规则页 E2E 已覆盖新增、编辑权限和创建流程，验证深度满足本次前端改动范围。
- 告警库“已识别”与“未识别 fallback”两条筛选路径现在都能落到明确的数据源或查询参数，没有遗留单侧分支。

## 影响范围检查

- 仪表盘最近告警区现在依赖 current alarms 查询返回值；如果后端或 mock 的 severity/eventTime 字段契约变化，会直接影响展示。
- 告警库页 unknown-only 模式已经切到 unknown stats 数据源，筛选分支与普通 definitions 数据源形成双路径，需要后续回归两种模式切换。

## 前后端一致性检查

- Dashboard 当前告警列表消费 useCurrentAlarms 返回的 id/alarmName/alarmIdentifier/deviceName/deviceSn/severity/eventTime 字段，未发现新的契约漂移。
- Alarm Library 的识别状态下拉已把 false/true 正确映射到 AlarmDefinitionFilter.isUnknown 或 unknown-stats 视图，前端查询契约保持一致。

## 验证记录

- 通过: cd /home/zhanglu/goomc/omcmb/webcode && npm run test:e2e -- e2e/alarm.spec.ts
- 结果: 8 passed
- 通过: cd /home/zhanglu/goomc/omcmb/webcode && npm run typecheck

## 审查结论

PASS

未发现阻塞或警告级问题；本次告警页改动与规则页 E2E 补强方向正确，且已有针对性验证。