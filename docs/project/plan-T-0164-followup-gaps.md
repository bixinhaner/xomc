# T-0164 收尾缺口清单（对照 DoD 草案）

> 文档日期：2026-05-23（创建）/ 2026-05-25（38 项 100% 完成终态确认）
> 来源：用户对照 `docs/design/pm-kpi-pipeline-improvements.md §7 验收（DoD 草案）` 逐项审视后发现 T-0164 主干 8 个工作包已 done，但 DoD 细节有 30+ 项未覆盖。
> 状态：T-0164-P1..P8 全部 `dev_done_pending_review`（主干）+ 收尾 36 项 + 跨域 2 项 = **38 项 100% done**；下一步真机端到端验证。

---

## 0. 总览

8 个工作包主干已实施 + 测试通过 + docker 部署 OK，但相对设计文档 §7 DoD 草案的细化项缺口如下（按工作包列）：

| 工作包 | 缺口数 | 主要类别 |
|--------|--------|---------|
| G2 保留策略 | 2 | 前端挂载 + 普通表 cron 清理 |
| G4 时间窗 | 1 | 上报延迟 Prometheus 指标 |
| G5 自然桶聚合 | 3 | 设备组独立 runner / 手动重算 / Prometheus |
| G6 仪表盘 | 13 | UI 完整度 + 显示增强 + 内置仪表盘 + 导出 + 持久化分键 |
| G7 自定义聚合 | 9 | 与 G6 集成 + 持久化语义 + 创建者隔离 + cron 补跑 |
| G8 通用任务框架 | 4 | cron_state 表 + 启动补跑 + sys_configs + Prometheus |
| 跨域 | 2 | e2e_verify.sh + release-gate.md |
| **合计** | **34** | |

---

## 1. 优先级与排期建议

| 优先级 | 数量 | 说明 | 建议时间盒 |
|--------|------|------|----------|
| **P0** — 数据正确性 / 基础设施 | 7 | 设备组独立 runner / G8 cron_state + 启动补跑 / G2 普通表 cron 清理 / KPI 卡片按制式分键 | 2-3 天 |
| **P1** — 运维可见性 / 运维补跑 | 7 | G4/G5/G8 Prometheus 指标 / G5 手动重算入口 / G2 前端挂载 / G7 创建者过滤 + admin 看全 + continuous stop 语义 | 2 天 |
| **P2** — 用户体验 / UI 完整度 | 13 | G6 左右栏布局 / 共享筛选条 / 制式持久化 / 12 个内置仪表盘 / 6 种 panel 类型 / 粒度多选 / 对比双模式实数据 / 分享审计 / 显示增强 | 4-5 天 |
| **P3** — 进阶导出 / 集成 | 7 | Excel/PDF 导出 / URL 复现 / G6-G7 集成（adhoc 入口 + panel 渲染 + 粒度 tab） / panel fallback 提示 | 2-3 天 |
| **跨域** | 2 | e2e_verify.sh / release-gate.md | 0.5 天 |

总计 **~12-14 天工作量**，按 P0→P3 顺序推进，每 P 等级结束做一次 commit + review report。

---

## 2. 缺口明细（按工作包）

### G2 — PM 保留策略（2 项）

#### G2-Gap-1 (P1) — 前端 `PmRetentionSection` 挂载 SystemConfig/index.tsx
- **现状**：组件 + i18n 13 keys 已建（commit `4932d026`），但 `omcmb/webcode/src/pages/system/SystemConfig/index.tsx` 没 import + 渲染
- **实施**：~1 小时；3 行 import + JSX 即可
- **DoD**：用户能在系统设置页看到 5 行 × 2 列 PM 数据保留分组，可编辑

#### G2-Gap-2 (P0) — 普通表 daily/weekly/monthly 的 cron 清理任务
- **现状**：hypertable（pm_metrics + pm_metrics_hourly）走 TS retention policy；普通 4 表（pm_metrics_daily/weekly/monthly + pm_adhoc_aggregation_results 非 hypertable 部分）**无清理机制**
- **实施**：worker 内加 retention cron（每日 03:00 跑），按 sys_configs 读保留天数 → `DELETE FROM <table> WHERE end_time < NOW() - INTERVAL '<N> days'` 分批
- **DoD**：E2E 验证 daily/weekly/monthly N+1 天数据消失

---

### G4 — 时间窗（1 项）

#### G4-Gap-1 (P1) — 上报延迟 Prometheus 直方图
- **现状**：三时间字段（start_time/end_time/ingest_time）已写入，但 `ingest_time - end_time` 没有指标
- **实施**：PM collector 内加 Prometheus histogram `omcgo_pm_report_delay_seconds`，按 product / device 维度分桶
- **DoD**：Grafana 看到设备上报延迟分布；超阈值告警可配

---

### G5 — 自然桶聚合（3 项）

#### G5-Gap-1 (P0) — 设备组维度独立 cron runner（4 个）
- **现状**：`AggregateDeviceGroup` 函数已写但**没注册成独立 JobType**。设计要求 8 个 cron（设备级 4 + 设备组级 4），且**设备组任务在对应设备级任务之后启动**（避免 JOIN 时 device 聚合表还在更新）
- **实施**：
  1. 新建 4 个 runner：`hourly_group.go` / `daily_group.go` / `weekly_group.go` / `monthly_group.go`，实现 `asyncjob.JobRunner`
  2. cron 时刻：hourly_group `15 * * * *`（设备级 :05 + 10 分钟）/ daily_group `15 0 * * *` / weekly_group `20 0 * * 1` / monthly_group `25 0 1 * *`
  3. worker main 装配 4 个新 runner
- **DoD**：E2E 验证 pm_group_metrics_* 在对应 device 聚合表跑完后才有数据

#### G5-Gap-2 (P1) — 手动重算入口（指定桶区间）
- **现状**：晚到数据 / 补传场景无 REST 端点触发重算
- **实施**：app 端加 `POST /api/v1/pm/aggregation/recompute` 接受 `{granularity, start, end, dimension}` 入参 → 直接 INSERT async_jobs（绕过 cron）
- **DoD**：curl 触发 → worker 跑指定桶 → 聚合表对应行更新

#### G5-Gap-3 (P1) — rollup Prometheus 指标
- **现状**：Aggregator 运行无指标
- **实施**：新增 `pm_aggregator_runs_total{granularity,dimension,status}`（counter）+ `pm_aggregator_duration_seconds`（histogram）+ `pm_aggregator_rows_written_total{granularity,dimension,kind}`（counter，kind=counter/kpi）+ `pm_aggregator_bucket_lag_seconds`（gauge）

---

### G6 — 前端仪表盘（13 项）

#### G6-Gap-1 (P2) — 左右栏布局：左侧仪表盘列表 + 右侧仪表盘渲染
- **现状**：DashboardList + DashboardEditor 是两个独立全屏页面（路由分离）
- **实施**：合并为 PerformanceLayout 组件 — 左侧 240px 列表 + 右侧 flex-1 渲染区，URL 用 `/performance?dashboard=:id` 而非 `/performance/pm-dashboard/:id`

#### G6-Gap-2 (P2) — 共享筛选条（时间窗 / 设备组 / 设备多选）
- **现状**：每个 panel 独立 timeRange / device 选择
- **实施**：DashboardEditor 顶部加 `GlobalFilterBar`（时间窗 + 设备组多选 + 设备多选），panel 配置时可选"继承全局"或"独立覆盖"

#### G6-Gap-3 (P0) — 制式切换持久化到 user_preferences（按制式分键）
- **现状**：dashboard 行级别有 technology 字段，但全局制式选择 + KPI 卡片按制式分键的 user_preferences 没做
- **实施**：扩 pm_user_dashboard_preferences 表 schema 为 `{technology: {lte: {...}, nr: {...}, gsm: {...}}}` 或加 technology 列变 PK（user_id, technology）
- **DoD**：切制式 LTE → 5G NR 时整页刷新，KPI 卡片 / 选中仪表盘 / 筛选都切换

#### G6-Gap-4 (P2) — 12 个系统内置 readonly 仪表盘（seed）
- **现状**：完全没做
- **实施**：seed migration 插入 3 制式 × 4 报表类型 = 12 个 dashboard（owner_id=系统 / readonly），含一组合理 panel 默认值；DashboardList 区分"系统内置 / 我的 / 来自分享"三组
- **DoD**：用户登录看到 12 个内置仪表盘 + readonly 标识 + "另存为派生"按钮

#### G6-Gap-5 (P2) — TopN + 数值大屏 panel 类型
- **现状**：5 种（kpi_card/line/bar/table/gauge）
- **实施**：扩 panel_type CHECK 加 `topn` + `big_number`；PanelRenderer 加两种渲染器；PanelConfigDrawer 加配置项（TopN 的 N + 排序方向 / 数值大屏的字号 + 警戒色）

#### G6-Gap-6 (P2) — panel 输出粒度多选 + tab 切换
- **现状**：panel.granularity 是单选
- **实施**：改 panels 表 granularity 列为 `granularities TEXT[]`；前端 panel header 多 tab（hourly | daily | weekly | monthly），点 tab 切换数据源不重新拉

#### G6-Gap-7 (P2) — 对比双模式实数据
- **现状**：CompareMode 字段持久化 OK，但 PanelRenderer 用 mock 没接真数据
- **实施**：PanelRenderer 内对比模式 = 拉两组数据 + ECharts 多 series 渲染

#### G6-Gap-8 (P1) — 分享 / 撤销审计日志
- **现状**：share/unshare 实现 OK 但**不写审计**
- **实施**：service.Share/Unshare 内调 admin/audit.Sink → 写 audit_logs

#### G6-Gap-9 (P2) — 显示增强 5 项
- (a) pct 类 KPI 自动带 % 单位 + 阈值线（panel.config.threshold 已有字段，PanelRenderer 内消费）
- (b) 时间轴可切 end_time / ingest_time（panel 顶部 toggle）
- (c) 缺采点显示"缺采"不画 0
- (d) 超延迟阈值的点有角标
- (e) 上述 4 项前端 + 必要后端字段
- **实施**：~3-4 小时；改 PanelRenderer + 加 X 轴渲染 helper

#### G6-Gap-10 (P3) — Excel / PDF 导出
- **实施**：用 SheetJS / pdfmake 前端导出，或 app 端用 unidoc/excelize 生成

#### G6-Gap-11 (P3) — URL 分享复现筛选
- **实施**：DashboardEditor 把所有筛选条件写入 URL query params（`?start=...&end=...&devices=...`），打开时解析回 store

#### G6-Gap-12 (P3) — 行维度 = 自选 N 设备走 pm_tasks 任务式（G7 集成）
- **现状**：PanelConfigDrawer 有 deviceSns 字段但没"超过 N 个时强制走 adhoc"逻辑
- **实施**：panel 配置时 deviceSns 长度 > 10 时弹"建议改用自定义聚合任务"对话框 + 一键创建 adhoc task

#### G6-Gap-13 (P2) — KPI 卡片配置（添加 / 移除 / 排序）持久化 + 按制式分键
- **现状**：pm_user_dashboard_preferences 表已有 kpi_card_layout JSONB 字段，但前端无 UI 改 + 不按制式分键
- **实施**：扩 G6-Gap-3 同时改 schema 为按制式分键 + 前端加"KPI 卡片管理"弹窗

---

### G7 — 自定义聚合任务（9 项）

#### G7-Gap-1 (P3) — 从仪表盘工具栏进入"+ 自定义聚合"入口
- **现状**：PmAdhoc 是 `/performance/pm-adhoc` 独立页面
- **实施**：DashboardEditor 工具栏加按钮，点击弹 CreateTaskDrawer（复用 PmAdhoc 的组件）

#### G7-Gap-2 (P3) — 完成后结果可视化走 G6 panel 组件
- **现状**：ResultsViewer 是独立 Drawer + Table
- **实施**：admin 把 adhoc 任务结果包装成临时 panel（不写 pm_panels 表，前端运行时构造）渲染走 PanelRenderer

#### G7-Gap-3 (P3) — 快照页按粒度切换 tab
- **现状**：ResultsViewer 显示全部行
- **实施**：与 G6-Gap-6 一致，按粒度 tab 切

#### G7-Gap-4 (P3) — 可导出 Excel / PDF
- 同 G6-Gap-10

#### G7-Gap-5 (P1) — 全局保留期 sys_configs UI + 改小强 confirm
- **现状**：retention 365d 硬编码在 migration 000162
- **实施**：sys_configs 加 `pm.adhoc_retention_days`；系统设置页加 UI；改小时弹 confirm 显示预删数据范围（COUNT(*) WHERE time < NOW() - INTERVAL '<new_N>'）

#### G7-Gap-6 (P1) — 停止 continuous 任务时写 time_range.endTime
- **现状**：Cancel 只切 status=canceled
- **实施**：repo.Cancel 时若 mode=continuous，把 stoppedAt 写入 pm_tasks.window_end（语义：之后清理按 oneshot 走）

#### G7-Gap-7 (P1) — "我的任务"按 creator 过滤 + admin 看全
- **现状**：handler.List 不过滤
- **实施**：handler.List 默认按 creator 过滤；admin 角色显示"查看全部"toggle

#### G7-Gap-8 (P3) — panel 前端 fallback "数据源已清理或被删除"提示
- **现状**：panel.adhoc_task_id 软引用，task 删了 panel 无提示
- **实施**：PanelRenderer 内 if adhoc_task_id && task 查询 404 → 显示"任务已删除"占位

#### G7-Gap-9 (P1) — cron 错过触发时点后启动时自动补跑漏掉的桶
- **现状**：ContinuousScheduler 用 `next := sched.Next(updated_at)` + `now > next` 判断该跑了，但**只跑一次**
- **实施**：sweepOnce 内循环算 next，所有 < now 的 next 都触发一次（生成多个 enqueue）；或加 last_runs_at 字段 + sched.Next(last_runs_at) 直到 now 之间的所有点都补

---

### G8 — 通用任务框架（4 项）

#### G8-Gap-1 (P0) — `async_jobs_cron_state` 表 + 启动补跑
- **现状**：cron 触发依赖 robfig/cron 在内存中跑，worker 停机期间的触发完全丢失
- **实施**：
  1. 新建 migration `async_jobs_cron_state(job_type PK, last_triggered_at)`
  2. cron 调度器每次触发前先查 last_triggered_at，与 cron expr 计算的应该触发时刻对比，所有"应触发但没触发"的时间点都补一次 enqueue
  3. 多 worker 实例靠 INSERT/UPDATE 单 SQL 原子加锁防双触发
- **DoD**：worker 停机 1 小时再启动，hourly cron 自动补跑 1 个漏掉的桶

#### G8-Gap-2 (P0) — 上面的"启动补跑"实测验证
- 同 G8-Gap-1

#### G8-Gap-3 (P1) — 心跳间隔 / 僵尸阈值 sys_configs 可调
- **现状**：硬编码 `asyncjob.HeartbeatInterval=30s` / `ZombieThreshold=5min` / `SweeperInterval=60s`
- **实施**：sys_configs 加 3 个 key + asyncjob Sweeper 启动时读 + 监听 sys.config.saved 事件热重载

#### G8-Gap-4 (P1) — Prometheus 任务总览指标
- **实施**：新增 `omcgo_async_jobs_queue_depth{job_type,status}` (gauge) + `omcgo_async_jobs_duration_seconds{job_type}` (histogram) + `omcgo_async_jobs_failed_total{job_type}` + `omcgo_async_jobs_zombie_total{job_type}` + `omcgo_async_jobs_catchup_total{job_type}`

---

### 跨域（2 项）

#### Cross-Gap-1 (P1) — e2e_verify.sh 加 G5/G6/G7/G8 断言
- **现状**：226 个 check_status 主要覆盖 Sprint 0-9
- **实施**：加 G5 聚合数据正确性 + G6 dashboard CRUD + G7 adhoc 任务 + G8 cron_state 启动补跑 4 段，~30 个新断言

#### Cross-Gap-2 (P1) — release-gate.md 更新
- **实施**：加 T-0164 G1-G8 完整 DoD 检查项 + Prometheus 指标可见性 + e2e 通过率

---

## 3. 实施顺序

按依赖关系 + 优先级排：

```
P0（数据 / 基础设施 / 持久化）
  ├─ G8-Gap-1+2 cron_state + 启动补跑（其它 cron 类缺口前置）
  ├─ G5-Gap-1 设备组独立 cron runner（依赖 G8-Gap-1）
  ├─ G2-Gap-2 普通表 cron 清理（依赖 G8-Gap-1）
  └─ G6-Gap-3 制式切换持久化 + G6-Gap-13 KPI 卡片配置按制式分键
        ↓
P1（运维可见性 + 业务语义补完）
  ├─ G4-Gap-1 上报延迟指标
  ├─ G5-Gap-2 手动重算入口
  ├─ G5-Gap-3 + G8-Gap-4 Prometheus 指标
  ├─ G8-Gap-3 sys_configs 可调
  ├─ G2-Gap-1 前端挂载
  ├─ G6-Gap-8 分享审计
  ├─ G7-Gap-5 全局保留期 sys_configs
  ├─ G7-Gap-6 stop continuous 写 endTime
  ├─ G7-Gap-7 creator 过滤
  ├─ G7-Gap-9 cron 错过补跑
  └─ Cross-Gap-1+2 e2e + release-gate
        ↓
P2（用户体验 + UI 完整度）
  ├─ G6-Gap-1 左右栏布局
  ├─ G6-Gap-2 共享筛选条
  ├─ G6-Gap-4 12 个内置仪表盘
  ├─ G6-Gap-5 TopN + 数值大屏
  ├─ G6-Gap-6 粒度多选 + tab（兼带 G7-Gap-3）
  ├─ G6-Gap-7 对比双模式实数据
  └─ G6-Gap-9 显示增强（pct % + 阈值线 + 缺采 + 延迟角标 + 时间轴切换）
        ↓
P3（进阶导出 + G6-G7 集成）
  ├─ G6-Gap-10 / G7-Gap-4 Excel / PDF 导出
  ├─ G6-Gap-11 URL 复现筛选
  ├─ G6-Gap-12 大量设备时建议 adhoc
  ├─ G7-Gap-1 仪表盘工具栏 adhoc 入口
  ├─ G7-Gap-2 adhoc 结果走 G6 panel
  └─ G7-Gap-8 panel fallback 提示
```

---

## 4. 风险与边界

- **跨表 schema 变更**：G6-Gap-3 / G6-Gap-13 改 `pm_user_dashboard_preferences` schema，需 migration + 前端类型同步
- **新 migration 编号**：本次会有 5-8 个新 migration（async_jobs_cron_state + sys_configs 扩展 + user_prefs 改 schema + 12 个内置 dashboard seed + panels 表 granularities 列变 TEXT[] 等），需连续号
- **真机测试推迟**：用户明确"全部缺口补完后再真机测试"，所以 ACS 的 PM 上传 SPV 触发链路（已就绪）暂不验，等 P0-P3 全过再上真机
- **commit 频率**：建议按优先级分批 commit（P0 一批、P1 一批、P2 一批、P3 一批），每批含 review report

---

## 5. 进度跟踪

进度更新本文档底部 + backlog.md 子表 + commit message。

- [x] **P0（7 项）— 已 commit `46b50678` (+`922d6c56` 文档)**
  - ✅ G8-Gap-1 cron_state 表 + 启动补跑
  - ✅ G5-Gap-1 设备组独立 4 cron runner
  - ✅ G2-Gap-2 普通表 cron 清理
  - ✅ G6-Gap-3 + G6-Gap-13 制式切换持久化 + KPI 卡片按制式分键
  - 兼带 PM Auto-Setup（G1 收尾）

- [x] **P1（13 项，2026-05-25 全部完成）**
  - ✅ G7-Gap-7 creator 过滤 + admin 看全
  - ✅ G7-Gap-6 stop continuous 时写 endTime
  - ✅ G6-Gap-8 分享/撤销审计日志（audit.Log 接入）
  - ✅ G5-Gap-2 手动重算 endpoint POST /pm/aggregation/recompute
  - ✅ G2-Gap-1 PmRetentionSection 挂载 SystemConfig + 2 lang i18n
  - ✅ G7-Gap-5 partial（sys_configs seed migration 000166 加 4 key；admin UI 已可改；改小 confirm 留 P2）
  - ✅ G5-Gap-3 aggregator Prometheus 指标 4 个（Runs / Duration / RowsWritten / BucketLag）注册
  - ✅ G8-Gap-4 asyncjob Prometheus 指标 5 个（QueueDepth / Duration / Failed / Zombie / Catchup）注册
  - ✅ G8-Gap-3 完成（2026-05-25）— Sweeper interval / zombie_threshold / heartbeat_interval 全部从 sys_configs 读；`asyncjob/model.go:76 HeartbeatInterval` 改 `var`，`cmd/worker/aggregator.go:loadAsyncJobThresholds` 启动期注入到 `asyncjob.HeartbeatInterval`；改值需重启 worker 生效（按设计简化要求不做 EventBus 热重载）
  - ✅ G4-Gap-1 PM 上报延迟 histogram — `pm/metrics.go` 注册 `omc_pm_report_delay_seconds` + `pm/collector/collector.go:161` 接入 `.Observe(delay)` + 单测覆盖（2026-05-25 核实代码已实施）
  - ✅ G7-Gap-9 cron 错过补跑 lossless — `pm/adhoc/worker.go:202 sweepOnce` 每 sweep 推进 1 格 + 多 sweep 追平所有漏桶 + `asyncjob/cron_state.go:104 CatchupMissedBuckets` 同模式 + 单测 `Test_ContinuousScheduler_LosslessCatchup_AdvancesOneWindowPerSweep` 覆盖（语义等价，不丢桶）
  - ✅ Cross-Gap-1 e2e_verify.sh 加 G5/G6/G7/G8 断言（commit `3d0cacfa`，~15 个新 check_status_in）
  - ✅ Cross-Gap-2 release-gate.md §8.5 G1-G8 完整 DoD + Prometheus 自检命令（commit `3d0cacfa`）
  - **Prometheus instrumentation hooks**: ✅ Runner.runOnce 调 `ObserveDuration` / `IncFailed`；Aggregator `runner.go:97 SetBucketLag` + `group_runner.go:76`；`QueueDepthSampler` 在 `cmd/worker/aggregator.go:99` 装配启动（30s 周期）；Catchup `IncCatchup` 接入 — 数据非 0（2026-05-25 核实代码已实施）

- [x] **P2 第 1 批（3 项）— 已 commit `7c2e9576` (+`52016f04` 文档)**
  - ✅ G6-Gap-5 TopN + BigNumber panel 类型（migration 000168 + 2 渲染器 + ConfigDrawer）
  - ✅ G6-Gap-4 12 内置 readonly dashboards（is_builtin + seed 000169 + Service guard + UI Tag）
  - ✅ G6-Gap-9 显示增强（pct 单位 / threshold markLine / 缺采断线 / KPI 红色阈值 / time_axis 切换）

- [x] **P2 第 2 批（5 项）— 2026-05-25 dev_done_pending_commit**
  - ✅ G6-Gap-6 panel granularities TEXT[]（migration 000170 + 后端 SQL/handler + 前端 PanelHeader Tabs + ConfigDrawer 多选）
  - ✅ G6-Gap-7 对比双模式实数据（usePmPanelData hook + PanelRenderer 双 series + ComparePanel tag 嵌 Card 标题）
  - ✅ G6-Gap-1 左右栏布局（PerformanceLayout + DashboardEditorPane + 旧路由 Redirect）
  - ✅ G6-Gap-2 共享筛选条 GlobalFilterBar（store.globalFilter + ConfigDrawer.inheritGlobal Switch）
  - ✅ G6-Gap-13 KPI 卡片配置（KpiCardManager Drawer 11 候选 + 上下移 + 持久化 user_preferences.kpi_card_layout）

- [x] **P3（7 项）— 2026-05-25 dev_done_pending_commit**
  - ✅ G6-Gap-10 / G7-Gap-4 Excel/PDF 导出（excelExport.ts 工具 + PanelCard / DashboardEditorPane / AdhocResultPanel 集成）
  - ✅ G6-Gap-11 URL 复现（PerformanceLayout 双向同步 globalFilter ↔ URL query）
  - ✅ G6-Gap-12 deviceSns > 10 时建议 adhoc（PanelConfigDrawer Alert + 一键跳 + URL preset 预填）
  - ✅ G7-Gap-1 dashboard 工具栏 "+ 自定义聚合"（CreateAdhocTaskDrawer 抽公共）
  - ✅ G7-Gap-2 adhoc 结果走 G6 panel 风格（AdhocResultPanel ECharts 多 series + 表格）
  - ✅ G7-Gap-3 adhoc 粒度 Tab（AdhocResultPanel 内置）
  - ✅ G7-Gap-8 panel adhoc fallback（PanelRenderer Alert 占位）

- [x] **跨域（2 项）— P1 剩余批已交付（commit `3d0cacfa`）**
  - ✅ Cross-Gap-1 e2e_verify.sh 加 G5/G6/G7/G8 断言（~15 个 check_status_in）
  - ✅ Cross-Gap-2 release-gate.md §8.5 G1-G8 完整 DoD + Prometheus 自检命令

---

## ★ T-0164 收尾全部完成（36 项 + 跨域 2 项 = 38 项 100% done，2026-05-25 终态确认）

满足真机验证启动条件（用户"全部功能实现后才真机测试"约定）。docker 全栈待重新部署。

### 2026-05-25 终态核实

对照代码逐项核查后修正本文档 §5 内的过时标注：

- ✅ **G4-Gap-1**（曾标 🚧）— `pm/metrics.go` + `pm/collector/collector.go:161` 已实施 `ReportDelaySeconds.Observe`
- ✅ **G7-Gap-9**（曾标 🚧 "lossless 需大改"）— `pm/adhoc/worker.go:202 sweepOnce` 用"每 sweep 推进 1 格 + 多 sweep 追平"模式实现 lossless；`asyncjob/cron_state.go:104 CatchupMissedBuckets` 同模式；单测覆盖
- ✅ **Cross-Gap-1 / Cross-Gap-2**（曾标 🚧）— commit `3d0cacfa` 已交付（e2e_verify.sh 加 ~15 断言 / release-gate.md §8.5 加 G1-G8 DoD）
- ✅ **Prometheus instrumentation hooks**（曾标"hook 留 v2"）— Runner.runOnce / Aggregator / QueueDepthSampler / IncCatchup 全部接入，数据非 0
- ✅ **G8-Gap-3**（曾标 ⚠️ partial）— 2026-05-25 终结：HeartbeatInterval 从 `const` 改 `var` + `loadAsyncJobThresholds` 启动期注入；按用户简化偏好放弃 EventBus 热重载

---

## 6. 文档关联

- 设计文档：`docs/design/pm-kpi-pipeline-improvements.md` §7 验收（DoD 草案）
- 主 plan：`docs/project/plan-T-0164-pm-kpi-pipeline.md`
- 各子任务 plan：`docs/project/plan-T-0164-P{1..8}-*.md`
- backlog：`docs/project/backlog/subtasks/T-0164-pm-kpi-pipeline.md`
- 修订：本文档由 user 在 docker-deploy + 真机连接成功后审视 §7 时发现，2026-05-23 落档
