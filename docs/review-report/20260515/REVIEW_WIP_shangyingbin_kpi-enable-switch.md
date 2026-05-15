---
date: 2026-05-15
author: shangyingbin
scope: frontend (kpi-library)
type: feat
verdict: PASS
backlog: HOTFIX (源自外部 TODO.MD)
---

# KPI 指标库启用操作合入主列表 + 删除"启用指标" tab

## 变更范围

| 文件 | 操作 |
|------|------|
| `omcmb/webcode/src/pages/product/kpi-library/IndicatorTab.tsx` | 加 `OPERATOR_CODE='default'` 常量；引入 `useEnabledIndicators` / `useSetEnabledIndicators` hooks + `pendingId` state；"启用"列从 `<Tag>是/否</Tag>` 改为 `<Switch>` 即时切换，`onSettled` 清 pendingId、`onError` 弹错 |
| `omcmb/webcode/src/pages/product/kpi-library/index.tsx` | 删 `EnabledIndicatorsTab` import + tab 项（保留 4 tab：ENB / GSM / GNB / 单位定义） |
| `omcmb/webcode/src/pages/product/kpi-library/EnabledIndicatorsTab.tsx` | 文件删除（118 行清理） |

后端零改动 — `enabled_pm_indicators_*` schema、handler、repo 全部不动；per-operator 覆盖能力保留（UI 仅隐藏运营商选择器，固定用 'default' 行——与设计文档 §2.6 一致）。

## 审查发现

### CRITICAL — 0 项
### WARNING — 0 项

### INFO

1. 每行 Switch 切换是独立 API call，`pendingId` state 仅保护单条 loading 显示（同一时刻多个 Switch 看不到 loading 旋转）；连续点多行会发多个 mutation 请求，没有排队/合并机制——属预期行为（即时切换 UX 优先），但批量场景退化为多次单点。
2. 用户决策按方案 1：保留后端 per-operator 能力，UI 仅暴露 default 行。运营商覆盖能力（cmcc/ctcc/cucc 行）后端仍存在，未来需要按运营商管理时可恢复 UI 选择器。

## 关键校验

- `OPERATOR_CODE` 常量化便于未来恢复选择器（改成 state 即可）
- mutation `onError` 弹 `message.error`，符合现有产品页 UI 反馈一致性
- 删除文件无残留引用（已 grep 全仓库确认 EnabledIndicatorsTab 无其他 import）

## 验证

- `npm run typecheck` ✅
- Docker 重建 ✅（rebase 后 node-forge 缺包，npm install 补回）
- 浏览器联调（Playwright）：
  - KPI 库 tab：只剩 ENB (LTE) / GSM / GNB (5G NR) / 单位定义（"启用指标" tab 消失）
  - ENB Tab 启用列变 Switch；初始 263 行 indicators 默认 checked=true
  - 切换 `C000060073`：UI Switch true→false → DB `SELECT FROM enabled_pm_indicators_enb WHERE operator_code='default' AND indicator_id='C000060073'` 返 0 rows
  - 再切回开启：UI false→true → DB 同条件返 1 row（指标 ID 回来）— 符合设计 §2.6 "禁用=不存在该行"
  - GSM / GNB Tab 启用列同样为 Switch + 工作正常

## 结论

PASS — 无阻塞项，可合入。
