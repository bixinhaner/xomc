# PRD: 前端 Topology / Report 补完（T-0022）

> **关联**: Backlog T-0022 / Sprint-05 / Domain=frontend
> **作者**: Claude（代 Owner=前端）
> **创建**: 2026-04-28
> **状态**: 草案 → 实施（A 方案：主会话全程深度协作）

---

## 1. 业务背景

OMC 前端 webcode 的 Topology / Report 模块共 10 个页面（topology 6 + report 4），存在不同程度的 mock 占位：
- **mockData fallback**（生产风险）：SiteManagement `(data ?? mockData)` — 后端返空数组时显示假数据
- **完全本地 mock**：StationReport / HistoricalKPI / LTEStandardReport（部分接入）
- **占位标记**（次要）：DomainManagement / TopologyCanvas / GISMapView / PollStatistics

调研结果：
- ✅ 后端 topology 27 endpoint + report 9 endpoint 全部就绪
- ✅ frontend-core API 层 `topologyApi.ts` (17 方法) + `reportsApi.ts` (9 方法) 完整
- ✅ frontend-core Hook 层 `useTopology.ts` (17 hook) + `useReports.ts` (9 hook) 完整
- ❌ Page 层 4 个核心页面 mock 占位影响生产体验

本 PRD 聚焦 **page 层去 mock + 接已有 hook**，hook/API 层不动（已对齐）。

---

## 2. 用户故事

| 角色 | 故事 |
|------|------|
| 运维 | 我希望 SiteManagement 显示真实站点数据（即使为空也显示 Empty 状态），不显示 7 条假数据 |
| 数据分析师 | 我希望 StationReport / HistoricalKPI / LTEStandardReport 走真实 API，能基于真实 KPI 决策 |
| 前端开发者 | 我希望 mock 数据集中在 `frontend-core/mock/`，不散落在 page 层（dev 调试一致性）|

---

## 3. 验收标准（Given-When-Then）

### V1 — SiteManagement 真实数据
- **Given** 后端 `/api/v1/sites` 返回 0 条
- **When** 用户访问 SiteManagement 页面
- **Then** 显示 Antd Empty 空状态，**不**显示 7 条假数据

### V2 — StationReport 接 API
- **Given** 用户访问 StationReport 页面
- **When** `useReportSampleData` 触发查询
- **Then** 列表数据来自 hook 返回（dev 模式下 useMock=true 走 mock，prod 走真实 API），**不**直接 import 本地 `mockStationKPIs`

### V3 — HistoricalKPI 接 API
- **Given** 用户访问 HistoricalKPI 页面，选 KPI 类型 + 时间范围
- **When** `useReportRecords` 触发查询
- **Then** 显示真实历史 KPI 数据 + DownloadOutlined 触发 `useDownloadReport`

### V4 — LTEStandardReport 接 API
- **Given** 用户访问 LTEStandardReport 页面
- **When** `useReportDefinitions` 加载报表分类树 + `useGenerateReport` 触发生成
- **Then** 显示真实定义列表 + 生成按钮工作

### V5 — typecheck/lint/build 全过
- `cd omcmb/webcode && npm run typecheck` 0 error
- `cd omcmb/webcode && npm run lint` 0 error
- `cd omcmb/webcode && npm run build` 通过

### V6 — mock 关键字消除
- `grep -rE "mockData|mockStationKPIs" omcmb/webcode/src/pages/topology/SiteManagement omcmb/webcode/src/pages/report` 仅剩注释或无（不是数据 import）

### V7 — vitest 不退化
- `cd omcmb/webcode && npm run test` 维持 T-0055 基线（lines 66.66% / statements 54.7%）

---

## 4. 运营商差异矩阵

| 维度 | CMCC | CTCC | CUCC |
|------|------|------|------|
| KPI 字段 | 一致（PRB/access_rate/drop_rate/...）| 一致 | 一致 |
| 报表分类 | 一致（kpi/availability/capacity/quality）| 一致 | 一致 |
| 实际差异 | **无** | **无** | **无** |

**结论**：T-0022 对运营商透明（前端通用展示 layer），不需要 Carrier 适配点。

---

## 5. 非目标

- ❌ 不实现 site CRUD（后端仅 GET/POST，PUT/DELETE 待后端补）
- ❌ 不接 PollStatistics（轮询统计，后端 endpoint 未确认）
- ❌ 不改 DomainManagement / TopologyCanvas / GISMapView（占位标记不阻碍生产，后续 PR）
- ❌ 不动 webcode-v2 / webcode-v3（多皮肤 PR 单独处理；本 PR 仅 webcode）
- ❌ 不补 vitest 测试（T-0055 基线不退化即可，新页测试后续 PR）
- ❌ 不动后端 + frontend-core API/Hook 层（已就绪）

---

## 6. 依赖

| 依赖 | 用途 |
|------|------|
| `@core/hooks/api/useTopology` | useSites / useDomainTree 等（已就绪）|
| `@core/hooks/api/useReports` | 9 个 hook 全部已就绪 |
| `frontend-core/mock/data/reports.ts` | 集中 mock 数据（部分已存，缺则补）|

---

## 7. 设计备忘（S2）

### 7.1 4 页面修改清单

| 页面 | 改动 |
|------|------|
| **SiteManagement** | 移除 `mockData` 数组 + `tableSource = (data ?? mockData)` fallback；接 useSites 返回 PageResponse 数据；空数组时 DataTable 自然渲染 Antd Empty |
| **StationReport** | 移除 `mockStationKPIs`；接 `useReportSampleData()` hook（hook 已存在，调 `/reports/sample-data` 端点；该端点返回站点级聚合 KPI sample）；mock 备份移到 `frontend-core/mock/data/reports.ts`（如缺）|
| **HistoricalKPI** | 移除 `generateKPIData/generateTimePoints` 本地数据生成器；接 `useReportRecords()` + `useDownloadReport()` hooks |
| **LTEStandardReport** | 已接 `useDownloadReport`；补接 `useReportDefinitions()` + `useGenerateReport()` hooks |

### 7.2 mock 集中策略（D2）

保留 mock 但移到 `frontend-core/mock/data/`，page 不再 import 本地 mock 数据。`useMock=true` 时通过 hook → mock service → mock data；`useMock=false` 时走真实 API。

### 7.3 类型转换（D3）

reports/topology 已有 `BackendXxx → mapBackendXxx → camelCase` 模式（reportsApi.ts mapBackendDef），page 层直接消费 camelCase。

---

## 8. 实施清单

**修改（4 文件）**:
- `omcmb/webcode/src/pages/topology/SiteManagement/index.tsx`
- `omcmb/webcode/src/pages/report/StationReport/index.tsx`
- `omcmb/webcode/src/pages/report/HistoricalKPI/index.tsx`
- `omcmb/webcode/src/pages/report/LTEStandardReport/index.tsx`

**新建（1-2 文件）**:
- `omcmb/frontend-core/src/mock/data/reports-stations.ts`（如不存在）— 集中站点级 KPI mock 数据
- `docs/review-report/20260428/verify-T-0022.md`

**Pass 标准**:
- typecheck 0 error / lint 0 error / build 通过 / vitest 不退化
- grep mock 关键字仅剩注释 or 无

---

## 9. 后续 PR

- DomainManagement / TopologyCanvas / GISMapView 完整接入（topology 高级页）
- PollStatistics 接入（后端 endpoint 确认）
- site CRUD 后端补 PUT/DELETE → 前端补 useUpdateSite/useDeleteSite hooks
- webcode-v2 / webcode-v3 同步改造（多皮肤一致性）
- 新页 vitest 测试覆盖

---

*本 PRD 由 dev-pipeline /pick T-0022 ULTRATHINK A 方案生成，主会话全程深度协作。*
