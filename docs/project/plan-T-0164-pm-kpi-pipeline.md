# T-0164 PM/KPI 流水线改造（G1-G8）总实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 PM/KPI 运行时流水线从"按运营商+制式扫全表、硬编码 SUM、无保留策略"重构为"按设备→产品→平台公式路由、按 statis_type 元数据驱动聚合、五级粒度统一保留、自然日历桶预聚合、前端仪表盘+面板模型、自定义聚合任务、通用任务框架"。

**Architecture:** 模块化单体内重构 PMCollector + KPIEngine + 新增 4 张聚合表 + 新增 internal/core/asyncjob 通用任务框架 + 前端 frontend-core 业务层扩展 + webcode 主皮肤改造。元数据来源切换为 T-0098 落库的 `perf_indicators_*` / `rela_platform_indicator_formula_*` / `param_models` / `product_class_patterns`。事件驱动模型不变（NATS JetStream + Retry/DLQ），通过 EventBus 失效缓存保持多 worker 一致。

**Tech Stack:** Go 1.25 + pgx/v5 + Squirrel + TimescaleDB（compression/retention/hypertable）+ Redis（L2 缓存 + 分布式锁）+ NATS JetStream + Prometheus + React 19 + Ant Design 5 + Zustand 5 + React Query v5。

---

## 0. 关键文档与依赖

| 类型 | 路径 | 说明 |
|------|------|------|
| 设计文档 | `docs/design/pm-kpi-pipeline-improvements.md` | G1-G8 完整方案（main commit cd3d8683，1352 行） |
| Backlog umbrella | `docs/project/backlog.md` §3 T-0164 行 + §4.5 | 任务登记 |
| 子任务表 | `docs/project/backlog/subtasks/T-0164-pm-kpi-pipeline.md` | 8 子任务完整 schema |
| 前置依赖 | T-0098 ✅（数据字典平台化收官，本任务消费其产物：`perf_indicators_{enb,gsm,gnb}` / `rela_platform_indicator_formula_*` / `param_models` / `product_class_patterns` / `products`） | |
| 相关风险 | T-0106（parammodel RedisCache cache_version 失效协议）— G1 需顺手修 / T-0121（PM 真机不推送）— G1-G5 走 SQL 注入历史数据自测 | |

## 1. 实施顺序（依赖图）

```
            G2 (保留策略, 独立, S 1d)
                    │
                    ▼ 完成不影响后续
            G4 (时间窗字段, 独立, M 1.5d)
                    │
                    ▼ G3 复用 G4 三时间字段
            G3 (合并表+删物化视图, L 3-4d)
                    │
                    ▼ G1 写入 pm_metrics
            G1 (路由+KPIEngine 重写, L 3-4d)
                    │           │
                    ▼           ▼
              G8 (任务框架, 独立 L 3-4d, 可与 G1 并行)
                    │
        ┌───────────┴───────────┐
        ▼                       ▼
      G5 (预聚合, L 4-5d)    G7 (自定义聚合, L 4d, 可与 G5 并行)
        │
        ▼
      G6 (前端仪表盘, XL 6-7d)
```

**夜间执行顺序**（按依赖关系最佳推进）：
1. G2 → 子 plan `plan-T-0164-P2-pm-retention.md`
2. G4 → 子 plan `plan-T-0164-P4-pm-timewindow.md`
3. G3 → 子 plan `plan-T-0164-P3-merge-pm-metrics.md`
4. G1 → 子 plan `plan-T-0164-P1-kpi-routing.md`
5. G8 → 子 plan `plan-T-0164-P8-asyncjob-framework.md`
6. G5 → 子 plan `plan-T-0164-P5-natural-bucket-aggregation.md`
7. G7 → 子 plan `plan-T-0164-P7-adhoc-aggregation.md`
8. G6 → 子 plan `plan-T-0164-P6-frontend-dashboard.md`

每个 G 独立 commit + 走 `/commit` skill；单个 G 卡 60+ min 无进展 → 标 🚧 跳下一个，不死磕。

## 2. 文件结构总览

### 后端 (omcgo/)

**新建**：
- `omcgo/internal/core/asyncjob/` — G8 通用任务框架（job_runner.go / heartbeat.go / sweeper.go / lock.go / cron.go / model.go / pg_repository.go + tests）
- `omcgo/internal/pm/aggregator/` — G5 自然日历桶预聚合（hourly.go / daily.go / weekly.go / monthly.go / device_group.go / query.go + tests）
- `omcgo/internal/pm/adhoc/` — G7 自定义聚合任务（service.go / oneshot.go / continuous.go / handler.go + tests）
- `omcgo/migrations/000164_*.sql` ~ `000171_*.sql` — 8 个 migration 文件（每 G 一个，版本号严格连续递增）

**修改**：
- `omcgo/internal/pm/parser/parser.go` — G4 补读 fileHeader/measCollec/@beginTime + fileFooter/measCollec/@endTime
- `omcgo/internal/pm/collector/handler.go` — G4 写 ingest_time / G1 路由查产品 KPI 列表
- `omcgo/internal/pm/kpi/engine.go` — G1+G3 KPIEngine 重写（删硬编码 KPIDefinitions + 按 statis_type 路由）
- `omcgo/internal/pm/kpi/calculator.go` — G3 counter 聚合按 statis_type 路由（sum/avg/max/pct）
- `omcgo/internal/pm/counter/` 和 `omcgo/internal/pm/kpi/` 的 repository — G3 合并到 pm_metrics 单表 + metric_type
- `omcgo/internal/pm/query/handler.go` — G3 删物化视图 / G5 按粒度+行维度路由到对应聚合表
- `omcgo/internal/product/registry.go` — G1 暴露 LookupKPIPlatformByDevice 方法
- `omcgo/internal/core/dictloader/parammodel/cache.go` — G1 修 RedisCache cache_version 失效协议（T-0106 顺手修）
- `omcgo/internal/core/appconfig/sysconfigs/keys.go` — G2 注入 5 个保留期键

**删除**：
- `omcgo/internal/pm/counter/pg_repository.go` 中 pm_counters 写入路径 → G3 改写 pm_metrics
- `omcgo/internal/carrier/{cmcc,ctcc,cucc}/kpi_definitions.go`（如存在）— G1 删除硬编码 KPI 列表

### 前端 (omcmb/)

**frontend-core 业务层（新增）**：
- `omcmb/frontend-core/src/types/dashboard.ts` — G6 Dashboard / Panel / PanelConfig 类型
- `omcmb/frontend-core/src/services/api/dashboardApi.ts` — G6 仪表盘 CRUD API
- `omcmb/frontend-core/src/services/api/adhocAggregationApi.ts` — G7 自定义聚合任务 CRUD API
- `omcmb/frontend-core/src/hooks/api/useDashboard.ts` — G6 React Query Hooks
- `omcmb/frontend-core/src/hooks/api/useAdhocAggregation.ts` — G7 React Query Hooks
- `omcmb/frontend-core/src/store/dashboardStore.ts` — G6 当前编辑的 Dashboard 状态
- `omcmb/frontend-core/src/i18n/zh-CN/dashboard.ts` + `en-US/dashboard.ts` — G6/G7 多语言

**webcode 主皮肤（改造）**：
- `omcmb/webcode/src/pages/performance/` — G6 三 tab → 单 tab"性能查看" + DashboardEditor + PanelGrid + PanelConfigDrawer + ComparePanel + ForkDialog + ShareDialog
- `omcmb/webcode/src/pages/performance/AdhocAggregation/` — G7 自定义聚合任务管理页

## 3. 全局验收

### 后端验收（每个 G 子 plan 自含验收，此处列跨 G 联动）

- [ ] `go build ./...` 通过（所有 G 完成后）
- [ ] `go test ./internal/pm/... ./internal/core/asyncjob/... ./internal/product/... ./internal/carrier/...` 全过
- [ ] `golangci-lint run` 0 警告
- [ ] migration 版本号严格连续递增（`bash omcgo/scripts/check-migrations.sh` 通过）
- [ ] migrate up 全过 + migrate down 全过 + migrate up 全过（双向幂等）
- [ ] E2E 验证：用 `scripts/cpe_simulator.py` 或 SQL 注入历史数据，模拟一次 PM 文件完整流水线（PM 解析 → 写 pm_metrics → cron 预聚合 → 查询接口）

### 前端验收

- [ ] `cd omcmb/webcode && npm run typecheck` 通过
- [ ] `npm run lint` 0 警告
- [ ] `npm run test` Vitest 单测过
- [ ] playwright MCP 自测：创建 Dashboard / 加 Panel / 切粒度（15min/hour/day/week/month）/ 对比双模式 / fork 派生 / 点对点分享链路，截图存 `.playwright-mcp/T-0164-*.png`
- [ ] 改 `frontend-core` 后评估 webcode-v2 / webcode-v3 typecheck 仍过（pre-existing baseline）

### 跨 G 联动验收

- [ ] G3 完成后，老接口 `GET /pm/counters` / `GET /pm/kpi-values` 全部返 pm_metrics + metric_type 过滤
- [ ] G1 完成后，KPI 计算只算"设备所属产品声明支持的指标"（用 BLQ 设备 fixture 验证：FAPService.FAPProduct = BaiBLQ → 只算 BLQ 产品 indicator_platform 声明的 KPI 子集）
- [ ] G5 完成后，cron 任务在 G8 任务框架下注册、心跳、僵尸检测、重启续跑全过
- [ ] G6 完成后，前端仪表盘能查 G5 聚合表（5 级粒度）并按 panel 行维度静态分流

## 4. 风险登记

| 风险 | 缓解 |
|------|------|
| G3 DROP TABLE + DROP MATERIALIZED VIEW 不可逆 | 用户拍板"删了数据也没事"（项目未上生产）；migration up/down 配对；commit message 说明 |
| G1 KPIEngine 重写后历史 carrier+tech 硬编码列表被删，回滚需 revert | 在 draft 分支推进，main 不动；早上用户 review 后再 cherry-pick |
| 多 worker 同时跑 G5 cron 导致重复聚合 | G8 PG advisory lock 串行化；测试覆盖锁竞争 |
| G6 前端仪表盘大改导致 webcode-v2/v3 编译失败 | 业务层入 frontend-core，UI 壳仅按需引用；改 frontend-core 时跑三皮肤 typecheck |
| 真机不推送 PM 文件（T-0121），无法端到端真机验证 | 用 SQL 注入历史数据 / scripts/cpe_simulator.py 触发；G6 用 mock 数据验证；遗留真机验证项进 TODO |

## 5. 提交节奏

每个 G 至少一个 commit（结构性变更可拆多 commit）：
- migration 单独 commit（便于回滚）
- 服务/handler/repository 一起 commit（保持可工作状态）
- test 跟实现一起 commit（不另立 test commit）
- 子 plan 内 task 数 ≥ 8 时，按"task 组"分 2-3 commit

commit message footer 五元组（参考 backlog/done/2026Q2.md 范例）：
```
PRD: docs/design/pm-kpi-pipeline-improvements.md
Sprint: wave-3
Risk: -
Backlog: T-0164-P{N}
Review: docs/review-report/20260522/REVIEW_<hash>_shangyingbin_<scope>.md（或 N/A skipped per §C ...）
```

## 6. 8 子 plan 索引

| G | 子 plan 文件 | Est | 状态 |
|---|------------|-----|------|
| G2 PM 保留策略 | [`plan-T-0164-P2-pm-retention.md`](plan-T-0164-P2-pm-retention.md) | S (~1d) | 待写 |
| G4 PM 时间窗字段 | [`plan-T-0164-P4-pm-timewindow.md`](plan-T-0164-P4-pm-timewindow.md) | M (~1.5d) | 待写 |
| G3 合并表+删物化视图 | [`plan-T-0164-P3-merge-pm-metrics.md`](plan-T-0164-P3-merge-pm-metrics.md) | L (~3-4d) | 待写 |
| G1 KPI 路由+KPIEngine 重写 | [`plan-T-0164-P1-kpi-routing.md`](plan-T-0164-P1-kpi-routing.md) | L (~3-4d) | 待写 |
| G8 通用任务框架 | [`plan-T-0164-P8-asyncjob-framework.md`](plan-T-0164-P8-asyncjob-framework.md) | L (~3-4d) | 待写 |
| G5 自然日历桶预聚合 | [`plan-T-0164-P5-natural-bucket-aggregation.md`](plan-T-0164-P5-natural-bucket-aggregation.md) | L (~4-5d) | 待写 |
| G7 自定义聚合任务 | [`plan-T-0164-P7-adhoc-aggregation.md`](plan-T-0164-P7-adhoc-aggregation.md) | L (~4d) | 待写 |
| G6 前端仪表盘+面板模型 | [`plan-T-0164-P6-frontend-dashboard.md`](plan-T-0164-P6-frontend-dashboard.md) | XL (~6-7d) | 待写 |

## 7. 设计决策不重新评审

按设计文档 §1.2 非目标，以下决策已锁定，实施期不评审：
- 不替换 TimescaleDB 本身
- 不重构 dictloader
- 不动 PM 文件采集前段（ACS → MinIO → NATS）
- 不在本次涉及 MR / 告警
- 不为 100 万规模做存储分片
- **不引入 fallback / 灰度 / 兼容路径**（项目未上生产，直接切换）
- G7 走 pm_tasks per-module，不进 G8 async_jobs（设计文档 §4.7 锁定）
- G8 缩窄为只承载 G5 cron + 未来批量计算

## 8. 夜间执行特殊约束

- 全程 `draft/pm-kpi-impl` 分支，**禁止 push**
- 单个 G 卡 60+ min 无进展 → 标 🚧 跳下个 G，不死磕
- migration 严守 `omcgo/CLAUDE.md §5.5` 自查清单（版本号、up/down 配对、StatementBegin/End、INSERT 列名对齐 DDL）
- 不碰 `.mcp.json` / `run/` / `deployments/` / `docs/design/*.md`
- 不改 chenbo01 写的代码（如发现依赖不存在字段，记到 TODO 兜底任务）
- 晨间 TODO.MD 末尾追加 `## 🌅 夜间执行报告（2026-05-22）`，每个 G 状态 🛠/🚧/❌（绝不 ✅）+ commit hash + 决策 + 截图路径 + 复现命令
