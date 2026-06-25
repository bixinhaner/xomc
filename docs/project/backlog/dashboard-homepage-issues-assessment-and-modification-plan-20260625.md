# 首页仪表板问题评估与修改方案（2026-06-25）

> **基线**：`main` HEAD `8ede0c9e`（2026-06-25）。
> **范围**：F02 Dashboard 首页全部已发现问题，含已合入项与未启动项；KPI 卡 / 网络制式 / 设备状态柱图 / 告警分布柱图 / 折线图区 / 快捷入口 / KPI 配置页全链路。
> **方法**：综合 [`docs/analysis/dashboard-functional-business-alignment-analysis.md`](../../../../docs/analysis/dashboard-functional-business-alignment-analysis.md)、原 backlog [`dashboard-kpi-issues-assessment-and-modification-plan-20260623.md`](dashboard-kpi-issues-assessment-and-modification-plan-20260623.md) 与近期提交记录，按 **P 级合并为 5 个 Issue**，每个 Issue 内列子任务清单。
> **合并策略**：以严重度分波次（P1 / P2 / P3 / P4 / 已合入归档），单次 PR 内集中解决同优先级问题，减少切片次数；子任务保留原 H01–H25 编号，便于追踪历史分析。

---

## 1. 目标与范围

本文档是首页仪表板的**单一权威 backlog**：

- 把分析文档 22 条候选事实 + 原 backlog 6 条 Issue 合并收敛为 **5 个 Issue**（HD01–HD05）。
- 每个 Issue 内按子任务列出具体改动点（保留原 H01–H25 编号供溯源）。
- 替代分析文档承担"接 PR 用"职责；分析文档保留为根因/证据底稿。

不在范围：

- GIS 模块（见 [gis-issues-assessment-and-modification-plan-20260617.md](gis-issues-assessment-and-modification-plan-20260617.md)）。
- KPI 指标库 / PM 模块自身（K 编号体系、采集流水）。
- 系统管理 / 角色权限子系统内部实现（仅在涉及 Dashboard 时引用）。

---

## 2. 架构对齐原则

- **前端**：`webcode` 只做页面编排，`frontend-core` 持有 hook / 类型契约 / i18n。snake↔camel 转换收敛在 `dashboardApi.ts` 的 `mapXxx*` 函数。
- **后端**：`handler → service → repository`；DB CHECK 约束与字典 / TypeScript 联合类型三层"制式集合"短期容忍偏差，长期归一。
- **指标编号**：K 编号 / KGNB / KGSM 为唯一标识，symbolic 别名仅作短期兼容（alias 表）。
- **业务口径**：同一概念在 KPI 卡 / 柱图 / PM 模块需走同一 SQL 谓词；偏差先记入对齐矩阵（见 §附录 A）。
- **降级策略**：loading / error / 空数据 → 显式 `--` 或空图，**不注入硬编码假数**。

---

## 3. Issue 总览（5 波次）

| 编号 | 严重度 | 标题 | 子任务（原 H 编号） | 状态 |
|------|--------|------|-------------------|------|
| **HD01** | P1 | 首页 KPI 卡数据准确性 一揽子修复 | H02 步骤 1·H02b·H03 | 🔧 代码已落地（前端 + 后端 SQL），待用户评估 → commit + Issue + PR；v2/v3 同步另开 |
| **HD02** | P2 | 首页 KPI 卡口径对齐与历史快照基础设施 | H07·H08·H02 步骤 2/3 | 📌 后端排期 + 业务对齐 |
| **HD03** | P3 | 首页图表展现层一致性 + 折线图增强 | H09·H10·H11·H12·H13·H14·H15·H16·H17·H18·H19 | 📌 多组并行 |
| **HD04** | P4 | 首页 KPI 体系收尾 + 死代码 / 配置开关清债 | H20·H21·H22·H23·H24·H25 | 📌 待清债 |
| **HD05** | ✅ | 已合入归档（PR #601 / #613 / #623 / #645） | H01·H04·H05·H06 | ✅ 已合入 |

> **节奏**：HD01 与 HD05 已就绪可直接合 / 归档；HD02 依赖业务对齐；HD03 可拆 3–4 个并行 PR 切片但归在同一 Issue 追踪；HD04 视清债决策节奏。

---

## 4. Issue 卡片

### HD01（P1, Bug bundle）— 首页 KPI 卡数据准确性 一揽子修复

#### 标题
首页 KPI 卡：活跃 UE 永远为 0 + 在线率 `NaN%`（前端契约层一揽子修复）

#### 严重度
P1（首页可见的明显错误，新部署 / 真实环境均可复现）

#### 现状
- 第 2 张卡"在线设备"在 `totalDevices === 0` 时 trend 区显示 `NaN%`。
- 第 4 张卡"活跃 UE"长期显示 `0`，趋势区 H01 修复后只能渲染 `0.0% vs 上周`。
- 双层根因：
  - **后端**：`GetSummary` 用 `kpiRepo.Query(PageSize=10, SortBy=time desc)` 拿 KPI——这是"top-10 行"上限而非"每指标最新一行"，全网指标 ≫ 10 时 UE_ACTIVE 被截断。
  - **前端**：`mapBackendSummary` 只拍平 7 个固定字段丢 dynamic key；`Math.floor(... ?? 0)` 把无数据与真 0 混为 0。

#### 实施现状（2026-06-25）
- ✅ H02 步骤 1（前端 `mapBackendSummary` 透传 + UE 卡 `--` 兜底）已落地。
- ✅ H03（在线率除零保护）已落地。
- ✅ **H02b**（后端 `GetSummary` KPI 取数语义修正）已落地：新增 [`kpi_summary_query.go`](../../../omcgo/internal/dashboard/kpi_summary_query.go)，走 `SELECT DISTINCT ON (metric_path) ... ORDER BY metric_path, time DESC LIMIT 500`，以指标编号为去重键每指标取最新一行。不再依赖业务对齐 D2 即可让有 UE_ACTIVE 数据的环境稳定读到。
- ✅ `dashboardApi.test.ts` 2 个前端回归 case + `kpi_summary_query_test.go` 4 个后端 SQL build case 均已落地。
- ⏳ 本地 commit / GitHub Issue 创建尚未进行（等用户评估后再做）。

#### 皮肤范围与三皮肤铁律例外
> CLAUDE.md §8.3 规定 "前端 bug 修复 v1/v2/v3 同时改"。本 issue **显式申请例外**：本期只动 `webcode`（v1，AntD5，唯一标准皮肤）+ `frontend-core` 共享层，**不动 `webcode-v2` / `webcode-v3`**。
> - 理由：v1 改动属契约层（`frontend-core`）+ 编排层；契约修正后 v2 直接自动收益（v2 dashboard 同样消费 `useDashboardData`），UI 兜底仅 v1 需要立即修；v3 BridgePage 不渲染 UE 卡。
> - 跟踪：v2 UE 卡兜底（`Math.floor(... ?? 0)` → `--`）拆为独立 follow-up issue 走铁律配套 PR；不阻塞本 P1 合入。
> - 验证：`omcmb && npm run typecheck` 三皮肤通过、`skin-parity` 通过（仅检查路由/菜单对齐，不查内容）。

#### 子任务

##### H02b — 后端 `GetSummary` KPI 取数改 DISTINCT ON（已落地）
- 文件：
  - 新 [`internal/dashboard/kpi_summary_query.go`](../../../omcgo/internal/dashboard/kpi_summary_query.go) — `buildLatestKPIPerNameQuery` 纯函数
  - 改 [`internal/dashboard/service.go` GetSummary step 3](../../../omcgo/internal/dashboard/service.go) — 走 `s.tsPool.Query` + DISTINCT ON，不再调 `kpiRepo.Query`
- 关键 SQL：
  ```sql
  SELECT DISTINCT ON (metric_path) metric_path, metric_value
  FROM pm_metrics
  WHERE metric_type='kpi' AND time >= $1 AND time <= $2
  ORDER BY metric_path, time DESC
  LIMIT 500
  ```
- 语义转变：原来是"24h 内 time DESC 前 10 行再去重"，现在是"24h 内每个 metric_path 各取最新一行"；LIMIT 的是"不同指标数"，500 远大于系统总 KPI 指标数。

##### H03 — 在线率除零保护（已落地）
- 文件：[`pages/dashboard/index.tsx` L297-L312](../../../omcmb/webcode/src/pages/dashboard/index.tsx#L297-L312)（第 2 张 KPICard 的 `delta` prop）
- 落地代码：
  ```tsx
  delta={totalDevices > 0
    ? `${Math.round((onlineDevices / totalDevices) * 100)}%`
    : '--'}
  ```

##### H02 步骤 1 — `mapBackendSummary` 透传 `kpi_overview` 全部 dynamic key（已落地）
- 文件：[`dashboardApi.ts` L152-L181](../../../omcmb/frontend-core/src/services/api/dashboardApi.ts#L152-L181)
- 落地代码：
  ```ts
  const overview = b.kpi_overview ?? {};
  // kpiSummary:
  ...overview,                                       // ★ 透传所有 dynamic key（含 UE_ACTIVE）
  rrcSuccRate: overview.RRC_CONN_SETUP_SR ?? 0,      // 兼容老调用点
  erabSuccRate: overview.ERAB_SETUP_SR ?? 0,
  hoSuccRate:   overview.NR_SA_HO_SR ?? 0,
  dlThroughput: overview.NR_PDCP_RATE_DL ?? 0,
  radioDrop:    overview.CALL_DROP_RATE ?? 0,
  prbUtil:      overview.NR_PRB_UTIL_DL ?? 0,
  ulThroughput: 0,                                   // DB 暂无上行速率 KPI
  voLteSuccRate: 0,                                  // DB 暂无 VoLTE KPI
  ```
- 同步改 [`pages/dashboard/index.tsx` L178-L181](../../../omcmb/webcode/src/pages/dashboard/index.tsx#L178-L181)：
  ```tsx
  const ueRaw = kpiSummary['UE_ACTIVE'];
  const currentActiveUE = typeof ueRaw === 'number' ? Math.floor(ueRaw) : undefined;
  // <KPICard ... value={currentActiveUE ?? '--'} />
  ```
  - 关键：用 `typeof === 'number'` 而非 `!== undefined`，区分 "真 0 → 显示 0" 与 "无数据 → 显示 `--`"。

#### 单测
- ✅ 已扩 [`dashboardApi.test.ts`](../../../omcmb/frontend-core/src/services/api/__tests__/dashboardApi.test.ts) 2 个 case：
  - `{ kpi_overview: { UE_ACTIVE: 1234, RRC_CONN_SETUP_SR: 99.5, ... } }` → `s.kpiSummary['UE_ACTIVE'] === 1234` 且命名快捷字段 `rrcSuccRate === 99.5`。
  - 缺 `UE_ACTIVE` 时 `s.kpiSummary['UE_ACTIVE'] === undefined`（区分真 0 / 无数据）。
- ✅ 新建 [`kpi_summary_query_test.go`](../../../omcgo/internal/dashboard/kpi_summary_query_test.go) 4 个 case：
  - SQL 含 `FROM pm_metrics` / `DISTINCT ON (metric_path)` / `metric_type = $1` / 时间窗 / `ORDER BY metric_path, time DESC` / `LIMIT 500`。
  - 不限 granularity / device（首页卡是全网视图）。
  - DISTINCT ON 不变量：ORDER BY 首键必须 = DISTINCT ON 列（PG 硬约束）。
  - `latestKPISummaryLimit == 500`。
- 不新增 UI 渲染单测（取舍）：H03 为 3 行 ternary，`webcode/pages/dashboard/` 现有 5+ 个测试全为纯逻辑，为它引入 `@testing-library/react` + `useDashboardData` mock 脚手架属于过度工程；阶性以 ├ contract 单测（已覆盖 UE_ACTIVE 透传根因）+ ├ v1 dashboard 手工 smoke 验收。

#### 验收
- 空环境下 "在线设备" 卡 trend 显示 `--`，不含 `NaN`。
- DB 有 `UE_ACTIVE` 时活跃 UE 卡稳定显示真实值（不再随刷新闪烁）；无数据显示 `--` 而非 `0`。
- 同理，后端 DISTINCT ON 修复后，任何指标编号只要 24h 内有一行 KPI 就会出现在 `kpi_overview` 里。
- mock 切换行为一致；`omcmb && npm run typecheck` 三皮肤 0 错（skin-parity 通过）。
- v1 dashboard 手工 smoke：空环境 / mock 环境 / 有真实数据三态切换正确。

#### 风险与回滚
- 风险 1：mock 数据未透传 `kpi_overview` → 已加前端单测兜底；命名快捷字段保留，老调用点不变。
- 风险 2：v2 UE 卡本期不修，仍显示 `0`（已知缺口）→ follow-up issue 跟踪，验收文档化。
- 风险 3：DISTINCT ON 在 pm_metrics（大表）走 metric_path 索引，500 上限足够低于 OOM 阈；如需调优可后续加二级 (metric_path, time DESC) 覆盖索引。
- 回滚：单 PR revert；后端 SQL 改动未动表结构，零 migration。

#### 工作量
- 0.4 人日（含单测）

#### 实施顺序与依赖
- **第一波合入**；无依赖。
- 建议分支：`fix/dashboard-kpi-card-data-accuracy`

---

### HD02（P2, Bug bundle）— KPI 卡口径对齐与历史快照基础设施

#### 标题
KPI 卡口径对齐（软删过滤） + 历史快照接入（`T-0164-P4`） + UE_ACTIVE 后端取数与 delta

#### 严重度
P2（数据失真但显示路径正常；触发条件需软删或对比看趋势）

#### 现状
- 总设备数：KPI 卡含软删 / 柱图不含 → 同页两数对不上。
- 趋势 %：H01 修复后能显示，但 `countDevicesAtTime/countAlarmsAtTime` 是 `TODO(T-0164-P4)` 占位 → `previous_value == current_value` → 永远 `0% / stable`。
- 活跃 UE：HD01 步骤 1 后前端能取，但后端 `KPIOverview` 是否含 `UE_ACTIVE` 取决于 PM 表与业务口径。

#### 子任务

##### H08 — 总设备数软删过滤口径对齐
- 文件：[`internal/dashboard/service.go`](../../../omcgo/internal/dashboard/service.go) `GetSummary` 调用 `deviceService.CountByStatus(nil)`
- 改动：默认带 `deleted_at IS NULL`；若 caller 需含软删，另开 `IncludeDeleted` 参数。
- 单测：`device_service_test.go` 覆盖软删行不进 count。

##### H07 — 历史快照基础设施（解锁 `T-0164-P4`）
- **决策 D1**：数据源选型
  - 候选 a：新建 `device_lifecycle_events` audit 表（精确，需 schema 新增）
  - 候选 b：从现有 `audit_logs` 反推（零 schema）
  - 候选 c：**推荐** — cron 每 30/60 min 落 `device_snapshot_hourly(time, total, online, offline, alarm)`，`*AtTime` 查最近快照
- 文件：
  - 新 migration：`device_snapshot_hourly` 表
  - cron job：复用现有调度框架，每小时触发一次 INSERT
  - 改写 [`countDevicesAtTime`](../../../omcgo/internal/dashboard/service.go) 走快照表；[`countAlarmsAtTime`](../../../omcgo/internal/dashboard/service.go) 走 `alarms_history`（`created_at <= ts AND (cleared_at IS NULL OR cleared_at > ts)`）
- 单测：mock 快照与 `alarms_history`，验证 `calculateKPIDeltas` 在已知历史下 `change_percent` 正确。

##### H02 步骤 2 — 后端写入 `UE_ACTIVE`（部分已被 HD01·H02b 覆盖）
> **状态变更 2026-06-25**：HD01·H02b 已把 `GetSummary` KPI 取数改为 DISTINCT ON 逐指标取最新值。只要后端 PM 表里有 `UE_ACTIVE` 行（以 K 编号 / enName 为 metric_path 落库），首页卡就能读到。剩下的业务对齐仅余 D2（哪条 K 编号对应活跃 UE / 是否需采集配置）。

- **决策 D2**：业务口径
  - 候选 a：RRC Connected UE 数 → 复用某条 K 编号最新值
  - 候选 b：E-RAB 激活数
  - 候选 c：DRB 激活数
  - 候选 d：PM 暂无对应 → **下线该 KPI 卡 or 替换为有数据指标**
- 文件：[`service.go GetSummary`](../../../omcgo/internal/dashboard/service.go) 显式查 PM 表对应 K 编号最新值，写入 `summary.KPIOverview["UE_ACTIVE"]`

##### H02 步骤 3 — `UE_ACTIVE` delta
- 依赖 H07 + H02 步骤 2
- 文件：[`calculateKPIDeltas`](../../../omcgo/internal/dashboard/service.go#L407-L450) 新增 `deltas["UE_ACTIVE"]`

#### 验收
- 软删一条设备 → KPI 卡总数 = 柱图总和。
- mock 1 天前数据 → 趋势 % 与方向箭头与预期一致。
- DB 有 `UE_ACTIVE` 时业务环境显示真实非 0 值，趋势区显示 "↑ 3.2% vs 上周"。
- 快照表容量在数月范围可控。

#### 风险与回滚
- 风险 1：快照 cron 失败 → `previous_value` 用最近可用快照（已知偏差，文档化）。
- 风险 2：D2 选错 K 编号 → 业务巡检；保留 D2=d 作为兜底（下线该卡）。
- 回滚：保留旧 `*AtTime` 实现为 fallback；feature flag 切回。

#### 工作量
- H08：0.3 人日
- H07：1 人日（schema + cron + 单测）
- H02 步骤 2：0.3–0.5 人日（依赖 D2）
- H02 步骤 3：0.3 人日
- **合计：约 2 人日**

#### 实施顺序与依赖
- 业务对齐 D1/D2 后启动；H08 / H07 可并行，H02 步骤 2/3 排在 H07 之后。
- 建议分支：`fix/dashboard-kpi-historical-snapshot-and-soft-delete`

---

### HD03（P3, Mixed bundle）— 图表展现层一致性 + 折线图增强 + 配置体系收尾

#### 标题
首页图表 / 折线图 / 快捷入口 / KPI 配置 P3 一揽子（11 个子任务，建议拆 3–4 个 PR 切片但归在一个 Issue）

#### 严重度
P3（不影响功能可用，但跨模块口径不齐 / 体验债）

#### 现状
首页除 KPI 卡之外的全部 P3 项一次性梳理：图表口径不齐、折线图无切换、快捷入口未鉴权、KPI 配置无冲突检测、alias 表有歧义项与待复核项。

#### 子任务

##### 4-A. 图表层（同区域 `pages/dashboard/index.tsx`，串行避免冲突）

- **H09 — "在线"语义统一**
  - KPI 卡走 `is_online=TRUE`（与柱图同源），移除 `model.DeviceActive` 路径
  - 文件：[`service.go GetSummary`](../../../omcgo/internal/dashboard/service.go)
- **H10 — "活跃告警" 条数/设备数 UI 区分**（需业务确认）
  - 推荐：保留两数 + 二级文案 "X 条告警，Y 台设备"，加 tooltip
  - 文件：[`pages/dashboard/index.tsx`](../../../omcmb/webcode/src/pages/dashboard/index.tsx) KPI 卡 / 柱图 tooltip
- **H12 — 4 等级告警柱图与 total 在 `unknown` severity 下矛盾**
  - 推荐：后端补 `alarm_stats.unknown` 段，柱图加第 5 段
  - 文件：[`service.go severityToLabel`](../../../omcgo/internal/dashboard/service.go) + 前端柱图
- **H13 — 设备状态柱图 X 轴接字典**
  - 接 [`useTechnologyDictionary`](../../../omcmb/webcode/src/components/dashboard/useTechnologyDictionary.ts)；删 `TECH_DISPLAY_NAME`
- **H14 — 设备状态柱图 alarm 段叠加易误读**（需业务确认）
  - 推荐：改 stacked bar（在线无告警 / 在线有告警 / 离线无告警 / 离线有告警）

##### 4-B. 折线图层（同区域 `LayoutKPIPanel.*`）

- **H11 — Day/Week 切换入口**
  - 文件：[`LayoutKPIPanel.tsx`](../../../omcmb/webcode/src/components/dashboard/LayoutKPIPanel.tsx) Header 加 Segmented；状态绑到 [`DashboardKPIModules.tsx:41`](../../../omcmb/webcode/src/pages/dashboard/DashboardKPIModules.tsx#L41) 第二参数
  - i18n 新增 `dashboard.compareType.{yesterday,lastWeek}`
- **H15 — hour 桶时区一致性**
  - 短期：[`LayoutKPIPanel.helpers.ts:61,68`](../../../omcmb/webcode/src/components/dashboard/LayoutKPIPanel.helpers.ts#L61) `getHours()` → `getUTCHours()` 或 `dayjs.utc`
  - 长期：明确 PM 聚合落库 TZ 契约

##### 4-C. KPI 指标 / 配置体系

- **H16 — `K900010006` alias 去重**（需业务复核）
  - alias 表 `K900010006` 同时映 `LTE_CELL_AVAILABLE` 与 `WIRELESS_SETUP_SR`；裁定一种
  - 文件：[`kpi_alias.go`](../../../omcgo/internal/dashboard/kpi_alias.go) + 新增 `TestAliasKCodeUnique`
- **H17 — alias 表 4 项 `NeedsReview` 清零**（需党晓萍签字）
  - 文件：[`kpi_alias.go`](../../../omcgo/internal/dashboard/kpi_alias.go) L66/78/79/87/88
  - 加 `TestNoPendingReview` 单测
- **H19 — KPI 配置页并发编辑冲突检测**（需业务确认必要性）
  - `dashboard_kpi_layouts` 加 `version int`；PUT 时比较，409 + 冲突弹窗

##### 4-D. 快捷入口

- **H18 — 按角色权限过滤**
  - 文件：[`pages/dashboard/index.tsx QUICK_ACCESS_ITEMS`](../../../omcmb/webcode/src/pages/dashboard/index.tsx#L84-L96)
  - 用 `useUserStore` 拿角色菜单白名单过滤

#### 验收
- 图表层：KPI 卡 / 柱图 / 折线图三处"在线 / 告警 / 制式"口径自洽；切语言 / 切制式 / 切对比类型稳定。
- 折线图：vs 昨日 / vs 上周可切；UTC 部署下 hour 桶位置正确。
- KPI 体系：alias `KCode` 单射；`NeedsReview` 清零；双管理员场景冲突弹窗工作。
- 快捷入口：切换角色 → 入口数量随之变化，无 403。

#### 风险与回滚
- 多区域并改 → 拆 3–4 个 PR 切片（图表层 / 折线图层 / KPI 配置层 / 快捷入口），单 PR 大小控制在 ≤ 300 行 diff。
- 业务确认延迟 → H10 / H14 / H16 / H17 / H19 子项可单独退后，不阻塞其它。
- 回滚：按子任务粒度 revert。

#### 工作量
- 图表层（H09/H10/H12/H13/H14）：1.5 人日
- 折线图层（H11/H15）：0.7 人日
- KPI 体系（H16/H17/H19）：1.5 人日（含业务会议）
- 快捷入口（H18）：0.3 人日
- **合计：约 4 人日**

#### 实施顺序与依赖
- 业务对齐项（H10 / H14 / H16 / H17 / H19）先开会；其余可并行。
- 建议分支系列：
  - `refactor/dashboard-charts-consistency`（H09/H10/H12/H13/H14）
  - `feat/dashboard-kpi-line-chart-enhance`（H11/H15）
  - `chore/dashboard-kpi-alias-and-config`（H16/H17/H19）
  - `feat/dashboard-quick-access-rbac`（H18）

---

### HD04（P4, Tech debt bundle）— 死代码 / 配置开关 / 历史端点清债

#### 标题
首页清债套件：6 个 P4 项一揽子归档（taskSummary / DASHBOARD_CONFIG / default layout / 6 端点 / 旧版 KPI 面板 / locale 硬编码）

#### 严重度
P4（不影响功能，纯技术债 / 文档债 / 决策项）

#### 现状
分析文档列出 6 条 P4 项，多数为"长期未启用 / 双轨残留 / 端点未消费 / 硬编码"。一次性纳入清债 Issue 而非各开 PR。

#### 子任务

##### H20 — `taskSummary` 死代码删除
- 文件：[`types/dashboard.ts`](../../../omcmb/frontend-core/src/types/dashboard.ts) + [`dashboardApi.ts`](../../../omcmb/frontend-core/src/services/api/dashboardApi.ts) + [`pages/dashboard/index.tsx`](../../../omcmb/webcode/src/pages/dashboard/index.tsx)
- 改动：删 `taskSummary` type / mapping 写死的 `{running:0,pending:0,success:0,failed:0}` / UI 分支 + `showRunningTasks` 开关
- 验收：grep `taskSummary` 全栈 0 命中

##### H21 — `DASHBOARD_CONFIG` 3 开关决策
- 文件：[`pages/dashboard/index.tsx:66-70`](../../../omcmb/webcode/src/pages/dashboard/index.tsx#L66-L70)
- 决策：
  - `showRunningTasks` → H20 已删
  - `showRefreshControls` → 决策保留 / 移除
  - `showDeviceMap` → 决策保留 / 移除（关联 GIS backlog）
- 改动：决议落档；选移除则按 H20 模式清除背后 mapping

##### H22 — `defaultLayoutJSON` / seed sql 迁 K 编号
- 文件：[`kpi_layout_default.go`](../../../omcgo/internal/dashboard/kpi_layout_default.go) + [`seed/000001_init_seed.sql`](../../../omcgo/migrations/seed/000001_init_seed.sql) L966-977
- 改动：参考原 backlog Issue A 附录 A 映射表，全部 symbolic → K 编号；新增增量 migration（已部署环境升级用）
- 验收：全新 seed 与增量升级 layout 字段全 K 编号；alias 表埋点 `LegacyMetricCounter` 趋零

##### H23 — 后端 6 个 dashboard 端点盘点
- 文件：[`handler.go`](../../../omcgo/internal/dashboard/handler.go)
- 6 路由：`/alarm-trend` · `/alarm-type-pie` · `/alarm-efficiency` · `/alarm-heatmap` × 2 · `/region-stats` · `/widgets`
- 改动：每条路由产出"归档 / 规划"决议；归档项加 deprecation log，路由保留 6 个月观察期

##### H24 — 旧版 KPI 面板 4 件双轨残留
- 文件：
  - [`useKPIPanelData.ts`](../../../omcmb/webcode/src/components/dashboard/useKPIPanelData.ts)
  - [`KPIPanel.tsx`](../../../omcmb/webcode/src/components/dashboard/KPIPanel.tsx)
  - [`KPITrendChart.tsx`](../../../omcmb/webcode/src/components/dashboard/KPITrendChart.tsx)
  - [`MultiKPITrendChart.tsx`](../../../omcmb/webcode/src/components/dashboard/MultiKPITrendChart.tsx)
  - `useKPITrendComparison` V1（[`useDashboard.ts:186`](../../../omcmb/frontend-core/src/hooks/api/useDashboard.ts#L186)）
- 改动：grep 确认无外部消费者后批量删除；评估 `MultiKPITrendChart` 是否替换 `LayoutKPIPanel` 当前 `LineChart`（独立 Issue）

##### H25 — `formatLastLogin` locale 硬编码
- 文件：[`pages/dashboard/index.tsx` L74-L88](../../../omcmb/webcode/src/pages/dashboard/index.tsx#L74-L88)（function 入口 L74；硬编码 `'zh-CN'` 在 L82）
- 改动：locale 接 `useAppStore(s => s.locale)`，传给 `toLocaleTimeString`
- 单测：覆盖 `zh-CN` / `en-US` 边界

#### 验收
- grep `taskSummary` / 旧 4 件文件 / `'zh-CN'` 硬编码 0 命中。
- 6 端点 issue 评论中有归档 / 规划决议。
- `defaultLayoutJSON` 全 K 编号；增量环境跑 migration 后 DB layout 验证通过。
- `tsc --noEmit` + 全套单测 0 红。

#### 风险与回滚
- 6 端点中部分可能有第三方调用 → 删除前查访问日志 + 留 6 个月观察期。
- 旧 KPI 面板 4 件删除前 grep 全栈，必须无消费者。
- 回滚：按子任务粒度 revert。

#### 工作量
- H20：0.2 人日
- H21：0.1 人日决策 + 0.3 人日清理（若选删）
- H22：0.5 人日
- H23：0.5 人日盘点 + 后续按决议
- H24：0.3 人日
- H25：0.1 人日
- **合计：约 2 人日**

#### 实施顺序与依赖
- 可串行单 PR 提交，也可拆 2–3 个小 PR；建议节奏宽松，挤进 sprint 间隙。
- 建议分支：`chore/dashboard-tech-debt-cleanup`（单 PR）或拆 `chore/dashboard-remove-task-summary` / `chore/dashboard-default-layout-k-codes` / `chore/dashboard-archive-unused-endpoints` 等子分支。

---

### HD05（✅, Archive）— 已合入项归档

| 子任务 | PR | Commit | 简述 |
|--------|----|--------|------|
| **H01** — KPI 卡趋势 % 永远不渲染 | #645 | `77721b95` | `KPIDelta` snake→camel；mapping 在边界处转换 |
| **H04** — KPI 下拉中英文翻译缺失（原 backlog Issue A） | #601 | `2675ce30` | catalog + alias 三段回退；持久化层迁移 → HD04 / H22 |
| **H05** — KPI 折线图单选改多选（原 backlog Issue B） | #613 | `c991a8b6` | 1 项今日+昨日 / ≥2 项只画今日；PALETTE 循环色 |
| **H06** — 网络制式 Segmented 字典驱动（原 backlog Issue C） | #623 | `8b1eb5c4` | `useTechnologyDictionary` hook；柱图 X 轴 → HD03 / H13 |

> **延伸**：H04 → H22；H06 → H13；H01 → H07（趋势数据本身）。

---

## 5. 实施顺序与节奏建议

```
Week N:   HD01 合入（P1，最快）
Week N+1: HD02 启动（业务对齐 D1/D2 + 后端排期）
Week N+1~N+3: HD03 拆 3–4 个 PR 切片并行推进
Week N+2~N+4: HD04 清债批次（挤间隙）
```

每个 Issue 自带"建议分支系列"，PR 数量控制：
- **HD01**：1 个 PR
- **HD02**：2–3 个 PR（H08 / H07 / H02 后续）
- **HD03**：3–4 个 PR（按 4-A/B/C/D 分组）
- **HD04**：1–3 个 PR（按节奏拆）
- **总计**：7–11 个 PR，集中在 4 个里程碑节点

---

## 6. 提交前检查清单

- [ ] 每个 Issue 都能映射到明确 PR 范围与子任务清单。
- [ ] 每个 PR 都有回归用例或手工验证步骤。
- [ ] Dashboard 关键行为（KPI 卡 / 柱图 / 折线图 / 切语言 / 切制式）有对照截图。
- [ ] 数据库变更随 migration 走，不留手跑 SQL。
- [ ] 变更遵循现有分层；snake↔camel 转换收敛在 `dashboardApi.ts`。
- [ ] 涉及业务口径变更（HD02 D1/D2、HD03 H10/H14/H16/H17/H19）有业务方签字记录。

---

## 附录 A：模块横向口径一致性矩阵

| 维度 | 设备模块 | 告警模块 | PM 模块 | Dashboard | 关联 Issue |
|------|---------|---------|---------|-----------|-----------|
| 制式文案 | 字典 `network_type` | 字典 | 静态 LTE/NR/GSM | Segmented 字典 ✅ + 柱图硬编码 ⚠️ | HD03·H13 |
| 设备在线性 | `is_online` | — | — | 柱图 `is_online` ✅ / KPI 卡 `status=Active` ⚠️ | HD03·H09 |
| 软删过滤 | `deleted_at IS NULL` | — | — | 柱图 ✅ / KPI 卡 ❌ | HD02·H08 |
| 告警计数 | — | severity 4 等级 + cleared/ack | — | total 全量 active；柱图只画 4 等级 ⚠️ | HD03·H10/H12 |
| 指标编号 | — | — | K/C 编号 + cnName/unit | catalog + alias 三段回退 ✅；default layout 仍 symbolic ⚠️ | HD03·H16/H17, HD04·H22 |
| 时区 | — | — | hourly TZ 不明 | hour 桶按浏览器本地 ⚠️ | HD03·H15 |
| 权限过滤 | RBAC | RBAC | RBAC | 快捷入口未过滤 ⚠️ | HD03·H18 |
| 趋势历史 | — | `alarms_history` ✅ | `pm_metrics` ✅ | `*AtTime` 占位 ⚠️ | HD02·H07 |

---

## 附录 B：后端 dashboard 路由消费情况

| 路由 | 用途 | 当前是否被首页消费 | 关联 Issue |
|------|------|------|------|
| `GET /summary` | 全网汇总 | ✅ KPI 卡 + 告警柱图 | HD01·H02 / HD02·H07/H08 |
| `GET /device-status` | 全网状态计数 | ❌ | HD04·H23 |
| `GET /device-status-by-type` | 按制式分组 | ✅ 设备状态柱图 | HD03·H13/H14 |
| `GET /alarm-trend?days` | 告警趋势 | ❌（仅 mock） | HD04·H23 |
| `GET /alarm-type-pie` | 告警类型分布 | ❌ | HD04·H23 |
| `GET /alarm-efficiency` | MTTA/MTTR | ❌ | HD04·H23 |
| `GET /alarm-heatmap[?days]` | 7×24 热度图 | ❌ | HD04·H23 |
| `GET /alarm-heatmap-by-severity` | severity 热度图 | ❌ | HD04·H23 |
| `GET /kpi-trend?kpi_name` | 单 KPI 趋势（legacy） | ⚠️ legacy hook 仍引 | HD04·H24 |
| `GET /kpi-time-series?kpi_names` | 多 KPI 时序 | ✅ 折线图区核心 | HD03·H11/H15 |
| `GET /kpi/definitions` | KPI 动态定义 | ⚠️ 前端有 hook 未用 | HD04·H23 |
| `GET /kpi-layout?tech` | 全局 KPI 布局 | ✅ | HD04·H22 |
| `PUT /kpi-layout` | 存全局 KPI 布局 | ✅ 配置页 | HD03·H19 |
| `GET / PUT /widgets` | 每用户 widgets | ❌ | HD04·H23 |
| `GET /region-stats` | 区域统计 | ❌ | HD04·H23 |

---

## 附录 C：参考文件索引

**前端**
- [`pages/dashboard/index.tsx`](../../../omcmb/webcode/src/pages/dashboard/index.tsx)
- [`pages/dashboard/DashboardKPIModules.tsx`](../../../omcmb/webcode/src/pages/dashboard/DashboardKPIModules.tsx)
- [`pages/dashboard/kpi-config.ts`](../../../omcmb/webcode/src/pages/dashboard/kpi-config.ts)
- [`pages/system/KpiConfig/index.tsx`](../../../omcmb/webcode/src/pages/system/KpiConfig/index.tsx)
- [`components/dashboard/LayoutKPIPanel.tsx`](../../../omcmb/webcode/src/components/dashboard/LayoutKPIPanel.tsx)
- [`components/dashboard/LayoutKPIPanel.helpers.ts`](../../../omcmb/webcode/src/components/dashboard/LayoutKPIPanel.helpers.ts)
- [`components/dashboard/useTechnologyDictionary.ts`](../../../omcmb/webcode/src/components/dashboard/useTechnologyDictionary.ts)
- [`components/dashboard/useMetricMetadata.ts`](../../../omcmb/webcode/src/components/dashboard/useMetricMetadata.ts)

**前端 API / Hook / 类型**
- [`hooks/api/useDashboard.ts`](../../../omcmb/frontend-core/src/hooks/api/useDashboard.ts)
- [`services/api/dashboardApi.ts`](../../../omcmb/frontend-core/src/services/api/dashboardApi.ts)
- [`services/api/__tests__/dashboardApi.test.ts`](../../../omcmb/frontend-core/src/services/api/__tests__/dashboardApi.test.ts)
- [`types/dashboard.ts`](../../../omcmb/frontend-core/src/types/dashboard.ts)

**后端**
- [`internal/dashboard/handler.go`](../../../omcgo/internal/dashboard/handler.go)
- [`internal/dashboard/service.go`](../../../omcgo/internal/dashboard/service.go)
- [`internal/dashboard/kpi_layout.go`](../../../omcgo/internal/dashboard/kpi_layout.go)
- [`internal/dashboard/kpi_layout_default.go`](../../../omcgo/internal/dashboard/kpi_layout_default.go)
- [`internal/dashboard/kpi_alias.go`](../../../omcgo/internal/dashboard/kpi_alias.go)
- [`internal/dashboard/kpi_network_query.go`](../../../omcgo/internal/dashboard/kpi_network_query.go)
- [`migrations/seed/000001_init_seed.sql`](../../../omcgo/migrations/seed/000001_init_seed.sql) L966-977

**相关分析与历史 backlog**
- [`docs/analysis/dashboard-functional-business-alignment-analysis.md`](../../../../docs/analysis/dashboard-functional-business-alignment-analysis.md) — 本文件的根因 / 证据底稿
- [`backlog/dashboard-kpi-issues-assessment-and-modification-plan-20260623.md`](dashboard-kpi-issues-assessment-and-modification-plan-20260623.md) — 历史 backlog（A/B/C/D 已合入归档）
- [`docs/analysis/dashboard-modules-analysis.md`](../../../../docs/analysis/dashboard-modules-analysis.md)
- [`docs/analysis/dashboard-vs-pm-kpi-logic-deep-analysis.md`](../../../../docs/analysis/dashboard-vs-pm-kpi-logic-deep-analysis.md)
- [`docs/analysis/dashboard-kpi-full-analysis.md`](../../../../docs/analysis/dashboard-kpi-full-analysis.md)
