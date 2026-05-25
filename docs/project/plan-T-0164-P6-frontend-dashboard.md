# T-0164-P6 / G6 前端"性能查看"单 tab + 仪表盘 + 面板模型 Plan

> **For agentic workers:** REQUIRED SUB-SKILL: superpowers:subagent-driven-development / executing-plans.

**Goal:** 把 "性能管理" 模块从三 tab（概览 / 趋势探查 / 报表）+ 报表模板，重构为单 tab "性能查看" + **Dashboard**（仪表盘）+ **Panel**（面板）两概念模型；制式顶层切换；对比双模式；点对点用户分享 / 协作；派生（fork）；KPI 卡片配置持久化到 user_preferences；多皮肤兼容（webcode / v2 / v3）。

**Architecture:** 业务层（types/services/hooks/store/i18n/mock）全部入 `frontend-core`，UI 壳放 `webcode/src/pages/performance/`。Dashboard 是一组 Panel 的容器（JSON 配置存 DB）；Panel 是单个图表组件（chart_type、metric_path、granularity、dimensions、compare_with 等属性）。数据加载按 panel 行维度静态分流——device 维度查 device 表，device_group 维度查 group 表。后端用 G7 自定义聚合任务（fork 时引用 task_id）+ G5 预聚合表（仪表盘默认查 hourly/daily）。

**Tech Stack:** React 19 + TypeScript 严格模式 + Ant Design 5 + ECharts 6 + Zustand 5 + React Query v5 + react-grid-layout（dashboard 拖拽布局）。

**Deps:** T-0164-P5 ✅（聚合表）+ T-0164-P7 ✅（adhoc 任务）+ 后端 dashboard CRUD 端点（本 plan 内 backend Task 一并做）。

---

## 0. 概念模型

```
Dashboard {
  id: uuid
  name: string
  description: string
  owner_id: uuid
  shared_with: uuid[]           # 点对点分享给的用户列表
  parent_dashboard_id: uuid?    # fork 派生引用
  technology: 'lte' | 'nr' | 'gsm'  # 制式顶层切换
  layout: {                      # react-grid-layout 配置
    panels: PanelLayout[]
  }
  created_at, updated_at: timestamp
}

Panel {
  id: uuid
  dashboard_id: uuid (FK)
  panel_type: 'kpi_card' | 'line_chart' | 'bar_chart' | 'table' | 'gauge'
  title: string
  metric_paths: string[]
  granularity: '15min' | 'hourly' | 'daily' | 'weekly' | 'monthly'
  dimension: 'device' | 'device_group'
  device_sns?: string[]          # device 维度
  device_group_ids?: uuid[]      # device_group 维度
  time_range: { start_offset, end_offset } | { absolute_start, absolute_end }
  compare_mode?: 'same_window_other_devices' | 'previous_window'  # 对比双模式
  adhoc_task_id?: uuid           # 关联 G7 自定义聚合任务（fork 时引用）
  config: JSONB                  # 图表特有配置（颜色、单位等）
}

UserPreferences (existing or new):
  user_id, kpi_card_layout: JSONB
```

**关键设计**：
- Panel 数据源**按 panel 行维度静态分流**：dimension='device' 查 `pm_metrics_*`；dimension='device_group' 查 `pm_group_metrics_*`。granularity 路由到对应聚合表。
- **对比双模式**：① same_window_other_devices — 同时间窗对比多设备；② previous_window — 同设备前一个时间窗对比
- **Fork**：复制 Dashboard 全部 layout + Panel；parent_dashboard_id 引用源
- **点对点分享**：Dashboard.shared_with 列表 + ACL 守卫（只读 / 编辑）

## 1. 文件结构

**后端**（新建）：
- `omcgo/migrations/000170_create_dashboard_and_panels.sql`
- `omcgo/internal/dashboard/model.go` / `service.go` / `pg_repository.go` / `handler.go`
- `omcgo/internal/dashboard/share.go` — 分享 ACL
- 路由挂 `/api/v1/dashboards` + 子资源 `/panels`

**前端 frontend-core（新建）**：
- `omcmb/frontend-core/src/types/dashboard.ts`
- `omcmb/frontend-core/src/services/api/dashboardApi.ts`
- `omcmb/frontend-core/src/services/api/adhocAggregationApi.ts`
- `omcmb/frontend-core/src/hooks/api/useDashboard.ts`
- `omcmb/frontend-core/src/hooks/api/useAdhocAggregation.ts`
- `omcmb/frontend-core/src/store/dashboardStore.ts`
- `omcmb/frontend-core/src/i18n/{zh-CN,en-US}/dashboard.ts`
- `omcmb/frontend-core/src/mock/dashboardMock.ts`

**前端 webcode（改造）**：
- `omcmb/webcode/src/pages/performance/index.tsx` — 单 tab "性能查看"
- `omcmb/webcode/src/pages/performance/DashboardList.tsx` — 仪表盘列表
- `omcmb/webcode/src/pages/performance/DashboardEditor/` — 编辑器组件群
  - `index.tsx`
  - `PanelGrid.tsx`（react-grid-layout）
  - `PanelConfigDrawer.tsx`
  - `PanelRenderer.tsx`（按 panel_type 路由到 KpiCard / LineChart / BarChart / Table / Gauge）
  - `ComparePanel.tsx`（双模式对比）
  - `ForkDialog.tsx`
  - `ShareDialog.tsx`
- `omcmb/webcode/src/pages/performance/AdhocAggregation/` — G7 自定义聚合任务管理
  - `index.tsx`
  - `CreateTaskDrawer.tsx`
  - `TaskList.tsx`
  - `ResultsViewer.tsx`

**删除**：
- 旧三 tab 入口（概览 / 趋势探查 / 报表）的 page 组件 + router 配置

## 2. Tasks（后端 4 + 前端 8 + 集成 1）

### Backend Task 1: migration + model + Repository

**Files:**
- Create: `omcgo/migrations/000170_create_dashboard_and_panels.sql`
- Create: `omcgo/internal/dashboard/{model,pg_repository}.go`

```sql
CREATE TABLE dashboards (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name TEXT NOT NULL,
  description TEXT,
  owner_id UUID NOT NULL REFERENCES users(id),
  shared_with UUID[] NOT NULL DEFAULT ARRAY[]::UUID[],
  parent_dashboard_id UUID REFERENCES dashboards(id) ON DELETE SET NULL,
  technology TEXT NOT NULL CHECK (technology IN ('lte','nr','gsm')),
  layout JSONB NOT NULL DEFAULT '{"panels":[]}'::JSONB,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE panels (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  dashboard_id UUID NOT NULL REFERENCES dashboards(id) ON DELETE CASCADE,
  panel_type TEXT NOT NULL,
  title TEXT NOT NULL,
  metric_paths TEXT[] NOT NULL,
  granularity TEXT NOT NULL,
  dimension TEXT NOT NULL CHECK (dimension IN ('device','device_group')),
  device_sns TEXT[],
  device_group_ids UUID[],
  time_range JSONB NOT NULL,
  compare_mode TEXT,
  adhoc_task_id UUID,
  config JSONB NOT NULL DEFAULT '{}'::JSONB,
  created_at, updated_at ...
);

-- KPI 卡片配置（per-user）
CREATE TABLE user_dashboard_preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id),
  kpi_card_layout JSONB NOT NULL DEFAULT '{}'::JSONB
);
```

- [ ] migration up/down/up + Repository 单测全过

### Backend Task 2: Service + ACL

**Files:**
- Create: `omcgo/internal/dashboard/service.go` + `share.go`

```go
type Service interface {
    Create(ctx, ownerID, dashboard) (*Dashboard, error)
    Get(ctx, requesterID, id) (*Dashboard, error)   // 校验 owner or shared_with
    List(ctx, ownerID) ([]Dashboard, error)         // 自己拥有的 + 被分享的
    Update(ctx, requesterID, dashboard) error       // 仅 owner 可改
    Delete(ctx, requesterID, id) error              // 仅 owner
    Fork(ctx, requesterID, srcID, newName) (*Dashboard, error)
    Share(ctx, ownerID, dashID, withUserIDs) error  // 加入 shared_with 列表
    Unshare(ctx, ownerID, dashID, userID) error
}

// share.go：CanRead / CanWrite ACL 检查
```

- [ ] Service 单测覆盖 CRUD + Fork + Share + ACL → 全过

### Backend Task 3: Handler + 路由

REST 端点：
- `GET/POST/PUT/DELETE /api/v1/dashboards` + `/:id`
- `POST /api/v1/dashboards/:id/fork`
- `POST /api/v1/dashboards/:id/share` { user_ids: [] }
- `DELETE /api/v1/dashboards/:id/share/:user_id`
- `GET/PUT /api/v1/user-preferences/dashboard` (KPI 卡片配置)

挂在 `cmd/app/router/router.go`，需 RequireAuth。

- [ ] handler 单测 + commit point

### Backend Task 4: 集成 + commit（后端独立 commit）

跑 `go build && go test ./internal/dashboard/...`。commit：
```
feat(dashboard): G6 后端 — Dashboard / Panel CRUD + Fork + Share + 用户偏好
```

- [ ] go test 全过 → commit

### Frontend Task 5: frontend-core types + mock

**Files:**
- Create: `omcmb/frontend-core/src/types/dashboard.ts`（含 Dashboard / Panel / PanelType / Granularity / Dimension / CompareMode + Backend* + mapBackend* 函数）
- Create: `omcmb/frontend-core/src/mock/dashboardMock.ts`

`types/dashboard.ts` 含完整类型定义 + mapper。约 200-300 LOC。

- [ ] typecheck 三皮肤全过

### Frontend Task 6: dashboardApi + useDashboard

**Files:**
- Create: `omcmb/frontend-core/src/services/api/dashboardApi.ts`（dashboardService 含 list/get/create/update/delete/fork/share/unshare/getUserPrefs/setUserPrefs；mockService 用 dashboardMock）
- Create: `omcmb/frontend-core/src/hooks/api/useDashboard.ts`（10 个 Hooks，含 invalidation）

- [ ] vitest 单测覆盖 mock 路径 / 真 API 路径 / mapper

### Frontend Task 7: dashboardStore (Zustand)

**Files:**
- Create: `omcmb/frontend-core/src/store/dashboardStore.ts`

```typescript
interface DashboardState {
  current: Dashboard | null;
  editMode: boolean;
  unsavedChanges: boolean;
  
  // actions
  loadDashboard: (id: string) => Promise<void>;
  updatePanelLayout: (panelId: string, layout: GridLayoutItem) => void;
  addPanel: (panel: PanelInput) => void;
  removePanel: (panelId: string) => void;
  toggleEditMode: () => void;
  save: () => Promise<void>;
}
```

- [ ] vitest unit test

### Frontend Task 8: webcode DashboardList + Editor 主框架

**Files:**
- Create: `omcmb/webcode/src/pages/performance/DashboardList.tsx`
- Create: `omcmb/webcode/src/pages/performance/DashboardEditor/index.tsx`
- Create: `omcmb/webcode/src/pages/performance/DashboardEditor/PanelGrid.tsx`

DashboardList：列出当前用户拥有 + 被分享的仪表盘，每行支持打开 / fork / share / delete 操作。

DashboardEditor：顶部 toolbar（制式切换 lte/nr/gsm + 添加 Panel + 保存 + Fork + Share）+ PanelGrid（react-grid-layout）。

- [ ] typecheck + vitest

### Frontend Task 9: PanelConfigDrawer + PanelRenderer

**Files:**
- Create: `omcmb/webcode/src/pages/performance/DashboardEditor/PanelConfigDrawer.tsx`
- Create: `omcmb/webcode/src/pages/performance/DashboardEditor/PanelRenderer.tsx`

PanelConfigDrawer：右侧抽屉，配置一个 Panel 的所有属性（type / metric_paths 选择器 / granularity / dimension / device or device_group 选择 / time_range / compare_mode 等）。

PanelRenderer：按 `panel_type` 路由到具体图表组件（KpiCard / LineChart / BarChart / Table / Gauge）。每个图表组件用 `useDashboardPanelData(panel)` Hook 拿数据。

- [ ] typecheck + vitest

### Frontend Task 10: ComparePanel + ForkDialog + ShareDialog

**Files:**
- Create: `omcmb/webcode/src/pages/performance/DashboardEditor/ComparePanel.tsx`
- Create: `omcmb/webcode/src/pages/performance/DashboardEditor/ForkDialog.tsx`
- Create: `omcmb/webcode/src/pages/performance/DashboardEditor/ShareDialog.tsx`

ComparePanel：在 Panel 配置 compare_mode='same_window_other_devices' 时，渲染多个对比序列；compare_mode='previous_window' 时，渲染当前+前一窗口双系列。

ForkDialog：弹窗 — 输入新名称 → POST /api/v1/dashboards/:id/fork → 跳转到新 dashboard。

ShareDialog：弹窗 — 用户选择器 → POST /api/v1/dashboards/:id/share。

- [ ] typecheck + vitest

### Frontend Task 11: AdhocAggregation 页面

**Files:**
- Create: `omcmb/webcode/src/pages/performance/AdhocAggregation/index.tsx` + 3 子组件
- 复用 frontend-core/src/services/api/adhocAggregationApi.ts + useAdhocAggregation.ts（G7 已建）

页面：左侧 TaskList + 右侧 ResultsViewer + 顶部 CreateTaskDrawer。

- [ ] typecheck + vitest

### Frontend Task 12: 删除旧三 tab + 路由整改

**Files:**
- Modify: `omcmb/webcode/src/pages/performance/index.tsx` — 改单 tab "性能查看"
- Delete: `omcmb/webcode/src/pages/performance/Overview/` / `TrendExplore/` / `Report/`（如存在）
- Modify: `omcmb/webcode/src/router/routes.tsx` — 改路由

playwright MCP 自测：
```
mcp__playwright__browser_navigate http://localhost:3000/performance
mcp__playwright__browser_snapshot  # 截图存 .playwright-mcp/T-0164-P6-dashboard-list.png
# 点 "新建 Dashboard"
# 加 Panel "KPI 卡片" + "曲线图"
# 切粒度 hourly → daily → monthly
# 测对比 same_window_other_devices + previous_window
# Fork
# Share to admin
mcp__playwright__browser_snapshot  # 多个截图
```

- [ ] playwright 完整链路截图 + typecheck + vitest

### Integration Task 13: 全链路 + commit

跑：
```bash
cd omcgo && go build ./... && go test ./internal/dashboard/...
cd omcmb/webcode && npm run typecheck && npm run lint && npm run test
cd omcmb/webcode-v2 && npm run typecheck 2>&1 | tail -20   # pre-existing baseline
cd omcmb/webcode-v3 && npm run typecheck 2>&1 | tail -20
```

commit message：
```
feat(dashboard,frontend): G6 前端"性能查看"单 tab + 仪表盘 + 面板模型 + 对比 + fork + 分享

What: 后端 migration 000170 创建 dashboards / panels / user_dashboard_preferences 三表 + Dashboard service（CRUD + Fork + Share + ACL）+ 8 REST 端点；frontend-core 新建 types/dashboard + dashboardApi + useDashboard + dashboardStore + i18n + mock；webcode 新建 pages/performance/{DashboardList,DashboardEditor,AdhocAggregation}/* 12+ 组件；删除旧三 tab（概览/趋势探查/报表）；KPI 卡片配置接 user_dashboard_preferences；制式顶层切换（lte/nr/gsm）；对比双模式（same_window_other_devices + previous_window）；fork 派生 + 点对点分享 ACL。
Why: G6 设计文档 §4.6；统一性能查看入口；用户拥有可定制仪表盘 + 派生 + 协作；按 panel 行维度静态分流（device → pm_metrics_* / device_group → pm_group_metrics_*）；持久化配置避免每次重选。
Impact: 后端 +3 表 +8 端点；前端业务层入 frontend-core 多皮肤共享；删除 3 个旧 page；webcode-v2/v3 typecheck 保持 pre-existing baseline。

PRD: docs/design/pm-kpi-pipeline-improvements.md
Sprint: wave-3
Risk: -
Backlog: T-0164-P6
Review: <审查报告路径>
```

- [ ] 全链路 typecheck/test 全过 → playwright 截图存 .playwright-mcp/ → /commit skill

## 3. 验收

- [ ] 后端 dashboard 单测全过
- [ ] frontend-core 三件套（API+Hook+Store）vitest 全过
- [ ] webcode typecheck + lint + vitest 全过
- [ ] webcode-v2/v3 typecheck 保持 pre-existing baseline（不退化）
- [ ] playwright：完整链路截图 ≥5 张（列表 / 编辑 / 加 panel / 对比 / fork / 分享）
- [ ] 端到端：创建 dashboard → 加 panel → 选 metric / granularity → 看到数据（用 G5 聚合表）→ fork → 分享给 admin → admin 能看到

## 4. Out of scope

- 实时数据 SSE（dashboard 自动刷新）→ 留 GA
- 模板市场（公共 dashboard 模板）→ 留 GA
- 多 KPI 联动钻取 → 留 GA
- 报表 PDF 导出 → 留 GA（旧"报表"功能完全删除，重做留二期）
- 移动端响应式适配 → 不在本期
