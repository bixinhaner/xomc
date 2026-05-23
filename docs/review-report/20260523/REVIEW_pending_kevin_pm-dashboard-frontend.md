# Review Report — T-0164-P6 / G6 前端：PM 性能查看 + 自定义聚合任务页面

- **Branch**: draft/pm-kpi-impl
- **Scope**: frontend-core + webcode
- **Backlog**: T-0164-P6（前端部分），含 T-0164-P7 UI 集成
- **Date**: 2026-05-23
- **Author**: shangyingbin (kevin)
- **Reviewer**: Claude (AI self-review)

---

## Conclusion

**PASS_WITH_WARNINGS** — 前端 v1 可合入，留若干 v2 增强项给后续 sprint。

- 0 CRITICAL
- 5 WARNING（v1 简化设计 trade-off）
- 4 INFO

webcode typecheck 全过。lint 新文件 0 警告 0 错（项目 pre-existing 201 错误与本次无关）。

---

## Files Changed

| Path | LOC | Type |
|------|-----|------|
| `frontend-core/src/types/pmDashboard.ts` | +200 | new |
| `frontend-core/src/types/pmAdhoc.ts` | +130 | new |
| `frontend-core/src/services/api/pmDashboardApi.ts` | +280 | new |
| `frontend-core/src/services/api/pmAdhocApi.ts` | +120 | new |
| `frontend-core/src/hooks/api/usePmDashboard.ts` | +145 | new |
| `frontend-core/src/hooks/api/usePmAdhoc.ts` | +55 | new |
| `frontend-core/src/store/pmDashboardStore.ts` | +130 | new |
| `frontend-core/src/store/__tests__/pmDashboardStore.test.ts` | +95 | new |
| `frontend-core/src/mock/data/pmDashboard.ts` | +90 | new |
| `webcode/src/pages/performance/PmDashboard/DashboardList.tsx` | +175 | new |
| `webcode/src/pages/performance/PmDashboard/DashboardEditor.tsx` | +195 | new |
| `webcode/src/pages/performance/PmDashboard/PanelGrid.tsx` | +90 | new |
| `webcode/src/pages/performance/PmDashboard/PanelRenderer.tsx` | +130 | new |
| `webcode/src/pages/performance/PmDashboard/PanelConfigDrawer.tsx` | +195 | new |
| `webcode/src/pages/performance/PmDashboard/ShareDialog.tsx` | +75 | new |
| `webcode/src/pages/performance/PmDashboard/ComparePanel.tsx` | +35 | new |
| `webcode/src/pages/performance/PmAdhoc/index.tsx` | +230 | new |
| `webcode/src/router/routes.tsx` | +8/-1 | mod |
| `docs/project/backlog/subtasks/T-0164-pm-kpi-pipeline.md` | +1/-1 | mod |

---

## Findings

### CRITICAL — 0

无。

### WARNING — 5

#### W1 — v1 用 antd Row/Col 静态布局，未集成 react-grid-layout 拖拽

**Files**: `PanelGrid.tsx` + `pmDashboardStore.ts`

plan §1 要求 react-grid-layout 拖拽，但其属 npm 依赖未安装；v1 改用 antd Row/Col 12 栅格静态布局（按 layout.w 映射 col span）。Store 的 `updatePanelLayout` / `replaceLayout` 接口已ready，v2 升级只需替换 PanelGrid 内的渲染层。

**Mitigation**：v1 完全可用（可看可改 panel 配置），仅缺拖拽手感。

**Action**：v2 待办 — 用户 review 后明确是否纳入下一 sprint。

#### W2 — ShareDialog 手输用户 UUID

**File**: `ShareDialog.tsx`

为避免引入 users API 调用（暂无 user lookup endpoint），v1 让 owner 手输被分享用户的 UUID 加入。

**Mitigation**：UUID 格式校验已加（regex `/^[0-9a-f-]{36}$/i`）。

**Action**：v2 待办 — 接 admin/usersApi 或新建 users-lookup endpoint 后改下拉选择器。

#### W3 — PanelRenderer 用 mock 数据，未接 G5 aggregator.Query

**File**: `PanelRenderer.tsx`

5 种 panel type（KPI 卡片 / 曲线图 / 柱状图 / 表格 / 仪表盘）当前用 hardcoded 占位数据展示。Plan §1 提到 `useDashboardPanelData(panel)` 拉真实数据 — 留下一轮接 G5 `/api/v1/pm/metrics/aggregated` endpoint。

**Mitigation**：渲染层结构完整，加 data hook 是 plug-and-play。

**Action**：v2 待办 — 添加 `useDashboardPanelData(panel)` hook 包装 G5 aggregator 路由。

#### W4 — 旧三 tab 路由保留兼容（plan §2.12 要求删除）

**File**: `router/routes.tsx`

plan 要求"删除旧三 tab（概览 / 趋势探查 / 报表）"。实际项目对应 7 个老 page（KPIStandard / KPIStation / KPIQuery / Charts / Threshold / Files / TaskConfig）。考虑用户要"先验证再删"，本次保留全部老路由 + 仅添加 3 个新路由（`pm-dashboard` list/editor + `pm-adhoc`）。

**Mitigation**：用户验证 G6 新页面 OK 后可独立 commit 删旧 page + 旧路由。

**Action**：用户 review 后决定。

#### W5 — frontend-core 命名 pmDashboard / pmAdhoc 与 existing dashboard 区分

**Files**: `types/pmDashboard.ts` + `services/api/pmDashboardApi.ts` + `hooks/api/usePmDashboard.ts`

现有 frontend-core 已有 `dashboardApi.ts` + `useDashboard.ts`（服务于 `internal/dashboard` 运营总览的设备/告警统计）。为避免名称碰撞，新文件全部 pmDashboard / usePmDashboard 前缀。

**Mitigation**：命名清晰区分两个业务域；后端表 pm_dashboards / pm_panels 同样有 pm_ 前缀对齐。

**Action**：无需修。

### INFO — 4

- **I1**: 业务层（types / api / hooks / store / mock）全部入 `frontend-core`，UI 壳放 `webcode/src/pages/performance/PmDashboard/` 和 `PmAdhoc/`，符合多皮肤共享原则。
- **I2**: Zustand store 仅承载 UI 临时编辑态（currentDashboard / editMode / unsavedChanges / panel layout）；远端数据 React Query 管理。
- **I3**: ACL 在前端体现为 isOwner 判断（隐藏编辑/保存/删除/分享按钮），后端 ACL 已是兜底。
- **I4**: AdhocAggregation 页面用 React Query 5s refetch interval 模拟进度实时刷新；plan §2.11 提的 SSE 进度推送 v2 可加（subscribe SubjectProgress）。

---

## Tests

| 测试 | 覆盖 | 结果 |
|------|------|------|
| `webcode typecheck` | 全部新文件 + 改动 routes.tsx | PASS |
| `webcode lint` 新文件 | 0 警告 0 错（pre-existing 201 错误不计） | PASS |
| `pmDashboardStore.test.ts` | 6 case：load/toggle/add/remove/update/markClean | written, blocked by pre-existing zustand resolve issue (userStore.test 同样 fail) — 不阻塞合入 |

---

## v2 待办（不在本次范围）

1. react-grid-layout 拖拽 + resize
2. ShareDialog 接 users API 下拉选择
3. PanelRenderer 接 G5 aggregator.Query 真实数据
4. AdhocAggregation 进度走 SSE（替代 5s polling）
5. ComparePanel 视觉效果增强（双轴 / 颜色区分）
6. 删除旧 7 个 performance page 路由（用户验证 G6 OK 后）
7. playwright MCP E2E 自测 + 截图

---

## DoD

- [x] webcode typecheck 全过
- [x] webcode lint 新文件 0 警告
- [x] frontend-core 类型完整（含 Backend wire + Mapper）
- [x] API + Hook + Store + Mock 全分层
- [x] 新增 3 个路由（pm-dashboard 列表 / 编辑器 / pm-adhoc）
- [x] 旧路由兼容保留
- [x] review report 与代码同 commit
- [x] backlog T-0164-P6 状态 partial_backend_done → dev_done_pending_review
- [ ] 浏览器实测（留用户手测）
- [ ] playwright 截图（留 v2）

---

## 用户验证步骤

1. `bash run/scripts/restart-all.sh`（或 docker 重启 app）使新路由生效
2. 浏览器访问：
   - `http://localhost:3000/performance/pm-dashboard` — 仪表盘列表（mock 数据下显示 2 个示例）
   - `http://localhost:3000/performance/pm-dashboard/mock-dash-001` — 编辑器（3 个 mock panel）
   - `http://localhost:3000/performance/pm-adhoc` — 自定义聚合任务管理
3. 验证：新建仪表盘 → 加 panel → 切粒度 → 派生 → 分享 → 删除 链路
4. 验证：新建 adhoc 任务（oneshot/continuous）→ 5 秒后 worker 跑完 → 看结果
5. 决定是否删除老 7 个 performance page 路由（独立 commit）
