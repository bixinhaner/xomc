# Review Report — T-0164 收尾 P3（G6-Gap-10 / 11 / 12 + G7-Gap-1 / 2 / 3 / 4 / 8）

- **Branch**: draft/pm-kpi-impl
- **Scope**: frontend-core utils + webcode PmDashboard / PmAdhoc 集成（导出 / URL 复现 / adhoc 入口 / fallback）
- **Backlog**: T-0164 收尾 P3 全 7 项
- **Date**: 2026-05-25
- **Author**: shangyingbin (kevin)

## Conclusion

**PASS（pending hash backfill）** — 可合入。0 CRITICAL，0 WARNING，3 INFO（mock 数据 + 浏览器原生 PDF）。

- 前端 `npm run typecheck` 通过
- 无新 migration（P3 全部为前端集成 + 工具函数）
- 复用现有 xlsx 0.18.x（webcode 已声明 dep）；PDF 用 window.print 原生不引入新 dep

## Files Changed

**新文件（3）**：
- `omcmb/frontend-core/src/utils/excelExport.ts` — 通用 xlsx 导出 + window.print 包装
- `omcmb/webcode/src/pages/performance/PmAdhoc/CreateAdhocTaskDrawer.tsx` — 创建抽屉公共组件（从 PmAdhoc/index.tsx 提取，供 DashboardEditorPane 复用）
- `omcmb/webcode/src/pages/performance/PmAdhoc/AdhocResultPanel.tsx` — adhoc 结果 G6 panel 风格视图（多指标 ECharts + 粒度 Tab + 表格 + 导出按钮）

**修改（6）**：
- `omcmb/webcode/src/pages/performance/PmAdhoc/index.tsx` — 用 CreateAdhocTaskDrawer + AdhocResultPanel 替换内嵌实现；处理 ?preset=panel URL 联动
- `omcmb/webcode/src/pages/performance/PmDashboard/PerformanceLayout.tsx` — URL ↔ store.globalFilter 双向同步（G6-Gap-11）
- `omcmb/webcode/src/pages/performance/PmDashboard/DashboardEditorPane.tsx` — "+ 自定义聚合" 按钮 / 导出配置 Excel / 打印 PDF
- `omcmb/webcode/src/pages/performance/PmDashboard/PanelConfigDrawer.tsx` — deviceSns > 10 时 Alert + 一键跳 adhoc（G6-Gap-12）
- `omcmb/webcode/src/pages/performance/PmDashboard/PanelGrid.tsx` — 每个 PanelCard 加导出 Excel 按钮（按粒度分 sheet）
- `omcmb/webcode/src/pages/performance/PmDashboard/PanelRenderer.tsx` — adhoc_task_id 软引用 fallback（任务不存在/已删除 → Alert 占位，G7-Gap-8）

## 实施清单

| 缺口 ID | 实施 | 状态 |
|---------|------|------|
| G6-Gap-10 / G7-Gap-4 (P3) | excelExport.ts 工具（多 sheet xlsx + window.print 包装）；PanelCard 单 panel 导出；DashboardEditorPane 导出配置 + 打印；AdhocResultPanel 导出结果 + 打印 | ✓ |
| G6-Gap-11 (P3) | PerformanceLayout 双向同步 globalFilter ↔ URL (start / abs_start / abs_end / devices / groups query)；URL 链接可复现完整筛选 | ✓ |
| G6-Gap-12 (P3) | PanelConfigDrawer deviceSns > 10 时 Alert + "去创建自定义聚合任务" 一键跳；URL 带 preset 预填到 CreateAdhocTaskDrawer | ✓ |
| G7-Gap-1 (P3) | DashboardEditorPane 工具栏 "+ 自定义聚合"；CreateAdhocTaskDrawer 抽公共组件被双方复用 | ✓ |
| G7-Gap-2 (P3) | AdhocResultPanel 用 G6 panel 风格渲染（ECharts Line 多 series + 表格）；不写 pm_panels 表，运行时构造 | ✓ |
| G7-Gap-3 (P3) | AdhocResultPanel 内置粒度 Tab（与 G6-Gap-6 一致）；切 Tab 只切数据视图 | ✓ |
| G7-Gap-8 (P3) | PanelRenderer 主入口检测 panel.adhocTaskId；usePmAdhocDetail 报错时 Alert 占位"数据源不可用，请编辑 panel 解除关联或重新创建任务" | ✓ |

## 关键设计决策

### Excel 导出策略
- 用 xlsx（已存在 dep）+ json_to_sheet；按粒度分 sheet 输出 — adhoc 4 粒度对应 4 sheet
- 文件名 = `<scope>_<name>_<id8>.xlsx`；scope=panel / adhoc / dashboard
- 单 panel 数据导出只导当前 active 粒度（多粒度切 Tab 后再点）— 避免 hook 在循环里调用的 React 限制

### PDF 导出策略
- 故意不引 pdfmake / jspdf（重 dep + 中文字体复杂）；用 window.print 原生
- 用户在打印对话框选"保存为 PDF"；标题动态注入到 document.title 决定生成文件名
- 后续若有 server 端 PDF 需求，可考虑 chromium headless 渲染（worker 侧），不在前端做

### URL 复现设计（G6-Gap-11）
- query schema：`?dashboard=:id&start=-24h&devices=SN-1,SN-2&groups=uuid-1`
- 也支持绝对时间：`?abs_start=ISO&abs_end=ISO`（与 startOffset 互斥，URL 同步代码保证只一个生效）
- 双向 useEffect：URL → store（仅 dashboard 切换时触发，避免循环）+ store → URL（用户操作 GlobalFilterBar 时）
- 用 toString() 比较防止 setSearchParams 无变更触发 history pop

### G6-Gap-12 跳转预填
- PanelConfigDrawer 关闭抽屉 + navigate('/performance/pm-adhoc?preset=panel&...')
- PmAdhoc 页加载时检测 ?preset=panel → 自动打开 CreateAdhocTaskDrawer + 预填 deviceSns/metricPaths/granularities
- 处理后清除 URL preset 参数（防刷新重弹）

### CreateAdhocTaskDrawer 公共化
- 从 PmAdhoc/index.tsx 提取（原 240 行 → 拆分 80+100）
- preset 入参驱动初始 setFieldsValue；任务名默认"从 panel 派生 (N 设备)"
- onCreated 回调把新 taskId 抛给父；DashboardEditorPane 收到后 navigate(/performance/pm-adhoc) 让用户立即看到

### AdhocResultPanel 视觉（G7-Gap-2/3）
- Card title + status Tag + 维度/指标/粒度计数；extra 区 导出 Excel / 导出 PDF
- 内嵌 Tabs（活动键 = 当前粒度）+ 每个 Tab 渲染 GranularityView（ECharts Line + 表格）
- 多指标多 series：buildSeriesByMetric 按 metric path 分组，按 startTime 横轴对齐；缺采 '-' 断线（与 G6 PanelRenderer 行为一致）

### Panel adhoc fallback（G7-Gap-8）
- PanelRenderer 主入口 if panel.adhocTaskId 调 usePmAdhocDetail；isError / no data → Alert
- 故意把 Alert 描述写成"已被删除或结果已超出保留期，请编辑解除关联或重新创建任务" — 用户能直接知道操作路径

## 跨包影响

- 新 utils `frontend-core/src/utils/excelExport.ts`：webcode-v2/v3 后续若要导出可直接复用
- xlsx 依赖位于 omcmb root node_modules（hoisted），frontend-core/webcode 均可解析
- CreateAdhocTaskDrawer 位于 `omcmb/webcode/src/pages/performance/PmAdhoc/`（UI 壳），不在 frontend-core；若 v2/v3 也要 adhoc 创建复用需后续抽到 frontend-core/components

## DoD 对照

| 项 | 目标 | 验证 |
|----|------|------|
| 前端 typecheck | `npm run typecheck` | ✓ |
| 无新 migration | P3 全前端 | ✓ |
| 无后端改动 | 仅 frontend-core + webcode | ✓ |
| 复用现有 dep | xlsx 0.18.5 已声明 | ✓ |
| URL 复现可分享 | 改全局筛选后复制 URL 给同事打开能看到一样视图 | ✓ 实测待真机 |
| adhoc 入口闭环 | DashboardEditorPane → CreateAdhocTaskDrawer → 任务列表 → 结果 Panel | ✓ 链路通 |
| fallback 行为 | PanelRenderer.panel.adhocTaskId 不存在的 task → Alert | ✓ |

## INFO

1. **mock 数据**：导出的 panel 数据来自 usePmPanelData（deterministic mock）；接真 API（v2）后行为透明等价
2. **多粒度 panel 导出**：只导出当前 active 粒度的数据（用户切 Tab 后可再次导出其它粒度）；一次性导出所有粒度需用 hook 在循环里调，违反 React rules，留 v2
3. **PDF 中文字体**：window.print 依赖浏览器默认字体；用户系统无中文字体时打印可能乱码 — 后续如有真 PDF 需求建议 server 端 chromium headless 方案

## TODO（不阻塞合入）

- xlsx 多粒度一次性导出（需用 useQueries 或自定义 hook 链）
- CreateAdhocTaskDrawer：deviceSns / metricPaths 改为下拉多选（接 useDevices / useIndicators）
- AdhocResultPanel：增加分页 / 时间窗筛选，结果 > 10K 行时分批拉
- URL 复现：扩展支持 panel 级 Tab 状态（当前粒度 Tab）持久化到 URL

## T-0164 收尾总进度

- ✅ **P0**（7 项）— commit `46b50678` + `922d6c56`
- ✅ **P1 主体**（9 项）+ GPV object 隔离 — commit `ac77b74b` + `d4092f89`
- ✅ **P1 剩余**（4 项）+ Prometheus hooks — commit `3d0cacfa` + `a4243c44`
- ✅ **P2 第 1 批**（3 项）— commit `7c2e9576` + `52016f04`
- ✅ **P2 第 2 批**（5 项）— commit `e2866ec1` + `d41306e0`
- ✅ **P3**（7 项）— 本批次 ★ → P0+P1+P2+P3 共 36 项 **全部 done**

剩余跨域 2 项（Cross-Gap-1 e2e_verify.sh / Cross-Gap-2 release-gate.md 在 P1 剩余已实施 ✓）

**真机验证启动条件已满足**（按用户"全部功能实现后才真机测试"约定）：
- docker 全栈重新部署：`bash /Users/shangyingbin/project/omc-docker/docker-run.sh`
- 真机 BLQ 1202000240194DP0026 回到现场后即可端到端测
