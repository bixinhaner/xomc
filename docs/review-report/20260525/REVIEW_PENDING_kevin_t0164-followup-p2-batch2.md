# Review Report — T-0164 收尾 P2 第 2 批（G6-Gap-6 / 7 / 1 / 2 / 13）

- **Branch**: draft/pm-kpi-impl
- **Scope**: pm.dashboard 后端 (granularity → granularities[]) + frontend-core/pmDashboard 类型 / hook / store / UI（PerformanceLayout / DashboardEditorPane / GlobalFilterBar / KpiCardManager / PanelGrid Tab）
- **Backlog**: T-0164 收尾 P2 batch 2（剩余 5 项全完成 → P2 总计 8 项 all done）
- **Date**: 2026-05-25
- **Author**: shangyingbin (kevin)

## Conclusion

**PASS（pending hash backfill）** — 可合入。0 CRITICAL，0 WARNING，2 INFO。

- 后端 `go build ./...` 通过；`go test ./internal/pm/dashboard/` 通过
- 前端 `npm run typecheck` 通过
- migration 编号连续：000170（紧接 000169 seed）
- 旧路由 `/performance/pm-dashboard/:id` 重定向到 `/performance?dashboard=:id` 兼容

## Files Changed

**新文件（6）**：
- `omcgo/migrations/000170_pm_panels_granularities_array.sql` — granularity → granularities TEXT[]
- `omcmb/frontend-core/src/hooks/api/usePmPanelData.ts` — Panel 数据加载 hook（v1 mock，含对比双 series 语义）
- `omcmb/webcode/src/pages/performance/PmDashboard/PerformanceLayout.tsx` — G6-Gap-1 左右栏布局主入口
- `omcmb/webcode/src/pages/performance/PmDashboard/DashboardEditorPane.tsx` — 抽离的右侧编辑器（接 dashboardId prop）
- `omcmb/webcode/src/pages/performance/PmDashboard/GlobalFilterBar.tsx` — G6-Gap-2 共享筛选条
- `omcmb/webcode/src/pages/performance/PmDashboard/KpiCardManager.tsx` — G6-Gap-13 KPI 卡片管理 Drawer

**修改（14）**：
- `omcgo/internal/pm/dashboard/model.go` — Panel.Granularity → Granularities []string
- `omcgo/internal/pm/dashboard/repository.go` — panelCols / Insert / Update / Fork SQL 全改 granularities + 默认 ['hourly']
- `omcgo/internal/pm/dashboard/handler.go` — panelDTO + panelInputDTO 改 granularities（含 binding 校验 dive,oneof）
- `omcgo/internal/pm/dashboard/repository_test.go` / `service_test.go` — Granularity → Granularities 适配
- `omcmb/frontend-core/src/types/pmDashboard.ts` — Panel.granularities + BackendPanel + mapBackendPanel
- `omcmb/frontend-core/src/services/api/pmDashboardApi.ts` — panelBody + mock createPanel 改 granularities
- `omcmb/frontend-core/src/store/pmDashboardStore.ts` — GlobalFilter + setGlobalFilter 加入（dashboard 级、loadDashboard 重置）
- `omcmb/frontend-core/src/mock/data/pmDashboard.ts` — mock panel 多粒度示例
- `omcmb/webcode/src/pages/performance/PmDashboard/PanelGrid.tsx` — PanelCard 子组件 + Tabs 切粒度 + ComparePanel tag 嵌入标题
- `omcmb/webcode/src/pages/performance/PmDashboard/PanelRenderer.tsx` — 接 activeGranularity；全部 7 渲染器消费 usePmPanelData；对比 series 用虚线 / opacity 区分
- `omcmb/webcode/src/pages/performance/PmDashboard/PanelConfigDrawer.tsx` — granularity 单选 → granularities 多选；新增 inheritGlobal Switch
- `omcmb/webcode/src/pages/performance/PmDashboard/DashboardEditor.tsx` — 退化为 Redirect 到新路由（兼容旧链接）
- `omcmb/webcode/src/router/routes.tsx` — 新增 `/performance` 主路由 → PerformanceLayout；旧路由保留

## 实施清单

| 缺口 ID | 实施 | 状态 |
|---------|------|------|
| G6-Gap-6 (P2) | panel granularities TEXT[] + 后端 SQL/handler 改造 + 前端 PanelHeader Tabs + ConfigDrawer 多选 | ✓ |
| G6-Gap-7 (P2) | usePmPanelData hook（compareMode 时返双 series）+ Line/Bar 虚线/半透明对比 + KpiCard/BigNumber 副值显示 + ComparePanel tag 嵌 Card 标题 | ✓ |
| G6-Gap-1 (P2) | PerformanceLayout（左 240px 三分组列表 + 右 DashboardEditorPane）+ URL ?dashboard=:id + 旧路由 Redirect 兼容 | ✓ |
| G6-Gap-2 (P2) | GlobalFilterBar（时间窗 Segmented + 设备组/SN CSV）+ pmDashboardStore.globalFilter + PanelConfigDrawer.inheritGlobal Switch（默认 true）| ✓ |
| G6-Gap-13 (P2) | KpiCardManager Drawer（11 候选 KPI + 上下移 + 添加/移除 + 持久化 user_preferences.kpi_card_layout 按制式分键）| ✓ |

## 关键设计决策

### Panel.granularities 数组化（G6-Gap-6）
- migration 000170 单事务 ADD COLUMN → backfill → DROP COLUMN → NOT NULL + CHECK 元素白名单
- 后端 default `['hourly']`（兼容上层调用未指定时）
- 前端 PanelGrid 内 PanelCard 子组件持 `useState<Granularity>(grans[0])`；只有 `grans.length > 1` 才渲染 Tabs
- 切 Tab 只 setState，不重新调 panel CRUD（符合 plan 的"切粒度不重新拉"语义）
- ConfigDrawer 用 `Select mode="multiple"` + `binding:"dive,oneof"` 后端白名单校验

### usePmPanelData hook 设计（G6-Gap-7）
- hook 对外 API：`(panel, activeGranularity) → { series, compareMode, isLoading }`
- v1 内部是 deterministic mock（按 panelId + granularity + metric 哈希），保证渲染稳定
- compareMode='previous_window' / 'same_window_other_devices' 时多 series 返回 + kind='compare' 标记
- v2 替换内部为 `/pm/aggregation/query` 调用，组件零改动
- 故意把 mock 抽到 frontend-core 而非 webcode：保证 webcode-v2 / v3 接同一份逻辑

### PerformanceLayout vs 旧 DashboardEditor（G6-Gap-1）
- 新 `/performance` 作为单 tab 入口（替代旧 `/performance/pm-dashboard` 列表 + `/pm-dashboard/:id` 编辑两套独立页面）
- 老路由通过 `<Navigate>` 重定向到新路由 + query；用户的旧收藏不破
- DashboardEditorPane 接 dashboardId prop（不再 useParams），便于在 PerformanceLayout 内嵌入
- 三分组（系统内置 / 我的 / 来自分享）在左侧列表中分块显示，符合 plan §1 中 G6-Gap-4 与 G6-Gap-1 的协同语义

### GlobalFilterBar 持久化范围（G6-Gap-2）
- store.globalFilter 是 UI 临时态，不写 pm_panels（panel 自己的 timeRange / deviceSns 仍然持久化）
- panel.config.inherit_global=true（默认）时 PanelRenderer 应消费 globalFilter；inherit_global=false 时用 panel 自己的
- 实际 v2 接真 API 时由 usePmPanelData 内部根据 panel.config.inherit_global 决定从 store 还是 panel 取参数
- v1 mock 不区分（数据稳定），只持久化字段 + UI 切换可见

### KpiCardManager UX（G6-Gap-13）
- 上下两个 List：已选（含上下移 / 移除按钮）+ 候选（含添加按钮）
- 用 BUILTIN_KPI_CARDS 内置 11 项候选；后续可改为读 indicator 字典动态拉
- 制式从当前选中 dashboard 取（自然分键），用户切 dashboard 自动切对应制式的偏好
- 写入 user_preferences.kpi_card_layout = `{order: string[], hidden: string[]}`（与 mock 形态对齐）

## 跨包影响

- `frontend-core/types/pmDashboard.ts` Panel.granularity → granularities 是 breaking change：
  - webcode 已同步（PanelConfigDrawer / PanelRenderer / mock data 全改）
  - webcode-v2 / v3 未引用 Panel.granularity（grep 确认），不受影响
- `frontend-core/store/pmDashboardStore.ts` 新增字段，旧消费方不受影响
- `frontend-core/hooks/api/usePmPanelData.ts` 新文件，仅 webcode 引用

## DoD 对照

| 项 | 目标 | 验证 |
|----|------|------|
| 后端编译 | `go build ./...` | ✓ |
| 后端测试 | `go test ./internal/pm/dashboard/...` | ✓ ok |
| 前端 typecheck | `npm run typecheck` | ✓ |
| migration 自查 | 编号连续 / Up Down 配对 / NOT NULL + CHECK | ✓ 000170 |
| breaking change 兼容 | webcode-v2/v3 不引用 Panel.granularity | ✓ grep 确认 |
| 旧路由兼容 | `/performance/pm-dashboard/:id` 仍可打开 | ✓ Redirect |
| 多皮肤评估 | frontend-core 类型变更不破 webcode-v2/v3 | ✓ |

## INFO

1. **mock 数据**：usePmPanelData 是 deterministic mock，对比双模式与真实 KPI 时序无关；接真数据待 P3（G6-G7 集成或 v2 后续）
2. **GlobalFilterBar 后续可升级**：当前设备组 / 设备 SN 用 CSV 文本，可换成 useDevices hook 的下拉多选（v2）

## TODO（不阻塞合入）

- usePmPanelData v2：内部接 `/pm/aggregation/query` + panel.inherit_global 时融合 globalFilter
- GlobalFilterBar v2：设备 / 设备组多选改为 Select + 真实候选
- KpiCardManager v2：候选 KPI 从 indicator 字典动态拉
- migration 000170 真机回归：现有 panels 有 granularity 单值的环境部署后自动 backfill 为单元素数组
