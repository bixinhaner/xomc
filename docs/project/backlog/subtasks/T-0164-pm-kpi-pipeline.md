# T-0164 拆分子任务（PM/KPI 流水线改造 G1-G8）

> 从 `docs/project/backlog.md` §3 Active 中的 umbrella T-0164 拆出（2026-05-22，立项即拆 8 子任务）。
> 行 schema 与 backlog.md §3 Active 主表对齐（13 列）。
> umbrella 行仍在主 backlog §3，状态联动通过本表 sub-task 推进。
>
> **来源**：`docs/design/pm-kpi-pipeline-improvements.md`（2026-05-19 定稿，G1-G8 全 8 个工作包）
> **关联**：T-0098 ✅（数据字典平台化收官，本任务消费其产物）/ T-0121（PM 端到端链路缺口）/ T-0106（parammodel RedisCache cache_version 失效协议建议）
> **PgM 决策（2026-05-22）**：P0 / **Wave 3 期间全准入**（用户拍板，G1-G8 全部 W3 期间推进，不延 W4）
>
> **关键约束**：
> - 项目未上生产 — 删表 / DROP MATERIALIZED VIEW 无需保留 fallback / 兼容路径（用户 2026-05-22 拍板"删了数据也没事"）
> - 实施在 `draft/pm-kpi-impl` 分支，**不动 main**；main 上的设计文档 `pm-kpi-pipeline-improvements.md`（cd3d8683）作为契约
> - 任何偏离设计文档的实现决策需在子任务 commit message + DoD 报告显式记录"为什么改"
> - G7 走 `pm_tasks` per-module（不进 G8 `async_jobs`），用户 2026-05-19 设计文档 §4.7 锁定
> - G8 缩窄为只承载 G5 cron + 未来批量计算（G7 不进）

### 4.5 T-0164 拆分子任务（8 条，2026-05-22 立项即拆）

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Sprint | Risk/PRD | Updated | Notes |
|----|-------|------|--------|------|-------|-------|-----|------|--------|----------|---------|-------|
| T-0164-P1 | **🛠 dev done — G1 指标路由按设备→产品→平台公式 + KPIEngine 重写**（2026-05-23 夜间执行）— 新建 `internal/pm/kpi/router` 包（KPIRoute / Router / RedisCache + L1 LRU + cache_version 协议）；KPIEngine 重写按 router.LookupByDevice 拿设备所属产品 KPI 子集再算（删 carrier 适配器 KPIDefinitions 接口 + 三个 carrier kpi.go 文件 + 接口测试）；`/pm/kpi/definitions` 端点切换数据源到 indicator.IndicatorRepository（前端兼容）；**顺手修 T-0106**（parammodel RedisCache 加 schema_version 校验，BumpVersion 一次失效全部缓存）；拆 `pm/kpi/expr` 子包打破循环依赖；新增 `pm/indicator.PlatformFormulaRepository.ListByPlatform`；app provider + worker main wiring 接入 ProductRegistry + Router；DB 集成现状验证：BLQ 平台 1095 indicators / 1095 formula 行，真机 SN `1202000240194DP0026` (`FAP/mBS31001/SC`) 路由到 QRTB 系列 / IndicatorPlatform=BLQ → 1095 KPI 子集。8 单测覆盖 router 5 case + redis_cache 3 case + parammodel schema 漂移 2 case + engine 6 case；`go test ./internal/pm/... ./internal/core/carrier/... ./internal/config/parammodel/...` 全过；pre-existing failures（acs/rpc TestDownloadHandler / interop+nedirect+northbound/sync test 文件缺 `DeleteByPathPrefix`）与 G1 无关，已 git stash 验证 main 上同样 FAIL。 | feat | F03/pm+F02/parammodel | P0 | dev_done_pending_review | Claude | L (~3-4d) | T-0164-P3 ✅ + T-0098 ✅ | wave-3 | `docs/design/pm-kpi-pipeline-improvements.md` §4.1 | 2026-05-23 | 替换 PMCollector 选指标这一步；多 worker 一致生效；单测覆盖路由命中/未命中/缓存失效；集成测试 DB 现状已验证，端到端跑 CPE 模拟 PM upload 留早上 review 时实测 |
| T-0164-P2 | **🛠 dev done — G2 PM 时序数据保留策略**（后端 commit `245c313b` 9 文件 +723 LOC：retention 包 policies/errors/service + 16 单测 + seed migration 158 + provider wiring；前端 commit `4932d026` 3 文件 +188 LOC：PmRetentionSection 组件 + i18n 13 keys × 2 lang；**未挂载 SystemConfig/index.tsx**，留早上） | feat | F03/pm+ops | P0 | dev_done_pending_review | Claude | S (~1d) | — | wave-3 | `docs/design/pm-kpi-pipeline-improvements.md` §4.2 | 2026-05-22 | policy 挂动作下沉到 G3/G5 migration（引用 retention 包常量）；前端挂载只需 1-2 行 import + JSX；早上 review + 挂载后 state→done |
| T-0164-P3 | **🛠 dev done — G3 聚合元数据驱动 + 合并 pm_counters/kpi_values 为 pm_metrics + 删物化视图**（晨间实施：migration 000160 DROP 旧 + CREATE pm_metrics hypertable + compression 7d + retention 30d；新建 internal/pm/metrics 包 model+repository+单测；counter/kpi pg_repository 改为 metrics.Repository 薄包装；pm/aggregation 整删；calculator.go 加 AggregateByStatisType；dashboard SQL 配套改；2 seed sql 适配；UNIQUE 索引含 time 修 TS 限制；dry-run 全过；pg_dump 备份 `/tmp/pm-kpi-pre-G3-backup-20260523.sql`） | feat | F03/pm+infra | P0 | dev_done_pending_review | Claude | L (~3-4d) | T-0164-P4 ✅ | wave-3 | `docs/design/pm-kpi-pipeline-improvements.md` §4.3 | 2026-05-22 | 本地无生产数据可直接删表；migration up/down 配对；单测覆盖四种 statis_type；3 个 baseline build failure（device.DeleteByPathPrefix + software.FailStale 接口变化）与 G3 无关 |
| T-0164-P4 | **🛠 dev done — G4 PM 时间窗 + 入库时间字段**（commit `b2fe02f8` 4 文件 +373 LOC：PMFileContent 三时间字段 + parser 补读 fileHeader/fileFooter + fallback 推断 + ValidateTimeWindow + 13 单测；**范围裁剪**：未动 pm_counters/kpi_values schema，三字段以 PMFileContent 文件级形态传递，G3 合表为 pm_metrics 时统一写入 NOT NULL 列） | feat | F03/pm | P0 | dev_done_pending_review | Claude | M (~1.5d) | — | wave-3 | `docs/design/pm-kpi-pipeline-improvements.md` §4.4 | 2026-05-22 | parser 层完成；G3 合表时调 ValidateTimeWindow + 用 PMFileContent.{FileBeginTime,FileEndTime,IngestTime} 写新表 |
| T-0164-P5 | **🛠 dev done — G5 自然日历桶预聚合（hourly/daily/weekly/monthly）**（2026-05-23 夜间执行）— migration 000161 创建 8 张聚合表（设备维度 4 + 设备组维度 4，pm_metrics_hourly / pm_group_metrics_hourly hypertable + compression 14d + retention 180d，其余 4 张普通表 PK 自然键）；新建 internal/pm/aggregator 包：aggregator.go（CASE WHEN SUM/AVG/MAX + 二阶段 KPI 求值复用 expr 包 + 5 unit）+ runner.go（asyncjob.JobRunner 通用实现 + 3 unit）+ hourly/daily/weekly/monthly.go（4 个 JobType 常量 + 构造器 + 4 wiring unit）+ device_group.go（JOIN devices + device_group_members 二阶段聚合 + 3 unit）+ query.go（5 粒度×2 维度路由 9 张表 + 12 unit）；pm/handler.go 加 WithAggregator + ListAggregatedMetrics 新路由 GET /pm/metrics/aggregated（老 /counters/aggregated 保留兼容）；cmd/app/provider/pm.go 装配 aggregator；2 integration test（OMCGO_DB_DSN 驱动）验证 SUM 路径正确性 + Runner.Run pipeline 闭环；级联聚合策略（pm_metrics→hourly→daily→weekly→monthly），相比统一从 15min 算更省 CPU；`go test ./internal/pm/... ./cmd/worker/...` 全过；worker cron 接入见 T-0164-P8（commit `3a8ce4ec`） | feat | F03/pm | P0 | dev_done_pending_review | Claude | L (~4-5d) | T-0164-P1 ✅ + T-0164-P3 ✅ + T-0164-P8 ✅ | wave-3 | `docs/design/pm-kpi-pipeline-improvements.md` §4.5 + `docs/project/plan-T-0164-P5-natural-bucket-aggregation.md` | 2026-05-23 | daily/weekly/monthly 不用 hypertable（量小 + 按月业务查询为主）；级联聚合更省 CPU（相比统一从 15min 算）；保留期接 G2 sys_configs；TimescaleDB Continuous Aggregate 不用（理由见设计 §4.5）；handler 新路由 /pm/metrics/aggregated 已可用，老 /pm/counters/aggregated 兼容老前端 |
| T-0164-P6 | **🛠 dev done — G6 PM 性能查看仪表盘（后端 + 前端全 done）** — 后端 commit `ac3f1c54`：migration 000163 + 13 REST 端点；前端 commit `<待提交>`：frontend-core 业务层（types/pmDashboard + types/pmAdhoc + services/api/pmDashboardApi + pmAdhocApi + hooks/api/usePmDashboard + usePmAdhoc + store/pmDashboardStore + 6 store unit test + mock 数据），webcode UI（pages/performance/PmDashboard：DashboardList + DashboardEditor + PanelGrid + PanelRenderer 5 panel type + PanelConfigDrawer + ShareDialog + ComparePanel；pages/performance/PmAdhoc：完整 CRUD + 结果查看）；router 加 3 新路由（pm-dashboard list/editor + pm-adhoc）；**旧三 tab 路由保留兼容**（KPIStandard/KPIStation/KPIQuery/Charts/Threshold/Files/TaskConfig 不动），用户验证后可独立 commit 删除；webcode typecheck 全过；vitest store test pre-existing zustand 解析问题与本次无关。**未做**：react-grid-layout 拖拽（v1 用 antd Row/Col 静态布局）+ playwright MCP 截图（留用户手测） | feat | frontend-core + frontend + F03/pm | P0 | dev_done_pending_review | Claude | XL (~6-7d) | T-0164-P5 ✅ + T-0164-P7 ✅ | wave-3 | `docs/design/pm-kpi-pipeline-improvements.md` §4.6 + `docs/project/plan-T-0164-P6-frontend-dashboard.md` | 2026-05-23 | 业务层（types/services/hooks/store/i18n/mock）入 frontend-core；webcode 加 PmDashboard + PmAdhoc 新页面；旧三 tab 留用户验证后清理；v2 待办：react-grid-layout 集成 / users 下拉选择器（替代 ShareDialog 手输 UUID） / 真实数据接 G5 aggregator.Query / playwright 自测 |
| T-0164-P7 | **🛠 dev done — G7 自定义聚合任务（oneshot + continuous）**（2026-05-23 白天执行）— migration 000162 扩 pm_tasks 加 7 列（task_subtype/mode/cron_expr/metric_paths/granularities/window_start/window_end）+ 2 CHECK 约束（mode 枚举 / continuous 必填 cron）+ 新建 pm_adhoc_aggregation_results hypertable（chunk 30d / compress 90d / retention 365d）；新建 internal/pm/adhoc 包：model.go + repository.go（CRUD + LockNextPending CTE SKIP LOCKED + PgContinuousRepository SQL 适配 + 7 integration test 全过）+ executor.go（多粒度遍历 + 中间 progress 上报 + 3 unit test）+ worker.go（pool 抢任务 + ContinuousScheduler 单 ticker 用 robfig/cron parser 评估 next 时间 + 7 unit test）+ handler.go（6 REST 端点：POST/GET/DELETE tasks + GET /results + SSE /progress + 5 unit test）；cmd/worker/adhoc.go 接入 4 worker + ContinuousScheduler；cmd/app/provider/pm.go + router.go 装配 REST 路由；commit `c035e846` | feat | F03/pm + frontend | P0 | dev_done_pending_review | Claude | L (~4d) | T-0164-P3 ✅ + T-0164-P8 ✅ | wave-3 | `docs/design/pm-kpi-pipeline-improvements.md` §4.7 + `docs/project/plan-T-0164-P7-adhoc-aggregation.md` | 2026-05-23 | 数据源经 G5 aggregator.Query（路由到 pm_metrics_15min / hourly / daily / weekly / monthly）；多粒度多选；继续模式 cron 调度时刻不需与 G5 整点对齐（独立 ContinuousScheduler 按 cron_expr 自由切换 scheduled→pending）；前端 AdhocAggregation 页面留 G6 集成 |
| T-0164-P8 | **🛠 dev done — G8 通用任务框架 + worker wiring 收尾**（框架 commit `9475c84b` 6 文件 +933 LOC：migration 159 async_jobs + 3 索引 + 6 状态 CHECK；asyncjob 包 model/repository/runner/sweeper + 11 单测；LockNextPending 用 CTE + SELECT FOR UPDATE SKIP LOCKED 原子化；ResetZombie 单 SQL CASE WHEN attempt+1>max_attempts。**worker wiring 收尾 commit `97fa2457`**：新建 cmd/worker/aggregator.go 装 Registry + Sweeper + 4 个 worker goroutine + robfig/cron/v3 调度器（hourly @:05 / daily 00:05 / weekly Mon 00:10 / monthly 1 日 00:15），cmd/worker/main.go 单点接入。go build + go test 全过） | feat | infra+ops | P0 | dev_done_pending_review | Claude | L (~3-4d) | T-0164-P5 ✅ | wave-3 | `docs/design/pm-kpi-pipeline-improvements.md` §4.8 | 2026-05-23 | LeaderElector 复用 internal/provision/leader_elector.go（鸭子类型）；scheduler 接 cron 触发器随 G5 入 cmd/worker/aggregator.go；DB SQL 端到端验证已通过；docker 验证留早上 review |

### 实施顺序（夜间自治执行）

按依赖图最佳顺序：

```
G2 (独立, 风险最低先做, S)
→ G4 (独立, 加字段, M)
→ G3 (合表 DROP TABLE, 依赖 G4, L)
→ G1 (路由替换 KPIEngine 重写, 依赖 G3, L)
→ G8 (任务框架, 独立可并行 G1, L)
→ G5 (聚合, 依赖 G1+G3+G8, L)
→ G7 (自定义聚合, 依赖 G3+G8, 可与 G5 并行, L)
→ G6 (前端仪表盘, 依赖 G5, XL)
```

**每个 G 独立 commit + 走 `/commit` skill**；卡 60+ min 无进展 → 标 🚧 跳下一个，不死磕。

### 自测策略

| 子任务 | 自测命令 | 工具 |
|--------|---------|------|
| G2 | `go test ./internal/core/appconfig/...` + SQL 查 `timescaledb_information.policies` | go test + psql |
| G4 | `go test ./internal/pm/parser/...` + 校验 parser 三时间字段单测 | go test |
| G3 | `go test ./internal/pm/...` + SQL 验证 pm_metrics 表结构 | go test + psql |
| G1 | `go test ./internal/pm/kpi/... ./internal/product/...` + 集成测试 BLQ fixture | go test |
| G8 | `go test ./internal/core/asyncjob/...` + 心跳/僵尸/锁竞争 | go test |
| G5 | `go test ./internal/pm/aggregator/...` + SQL 注入历史数据 + 触发 cron 验证 | go test + psql |
| G7 | `go test ./internal/pm/adhoc/...` + REST API 集成测试 | go test + curl |
| G6 | `cd omcmb/webcode && npm run typecheck` + playwright MCP 浏览器实测 | npm + playwright |

**真机限制**：BLQ `1202000240194DP0026` UUID `b9dfacaf-a15b-4b88-995c-aa6a5aa37c28` 是唯一在线测试设备，但**T-0121 已记 ACS 不下发 PM-CONFIG**——真机不会自然推 PM 文件；G1-G5 走 SQL 注入历史数据 / mock PM 文件 / `scripts/cpe_simulator.py` 触发，**不依赖真机 PM 推送**。

---

## 设计文档关键决策回顾（合入 §4.6/§4.7/§4.8）

- **G6 前端**：三 tab → 单 tab"性能查看" + 仪表盘 + 面板模型；点对点用户分享 / 协作；派生（fork）
- **G7 自定义聚合**：复用 pm_tasks（per-module，不进 G8 async_jobs）；oneshot + continuous 双模式；全局保留期（默认 1 年，drop_chunks 整块清）；panel ↔ task 独立生命周期
- **G8 通用任务框架**：缩减为只承载 G5 cron + 未来批量计算
