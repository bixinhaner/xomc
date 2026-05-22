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
| T-0164-P1 | **G1 指标路由按设备→产品→平台公式 + KPIEngine 重写** — 删运营商适配器代码硬编码 KPIDefinitions；新链路 device→productClass→ProductRegistry→indicator_platform→perf_indicators_* + rela_platform_indicator_formula_*；RedisCache cache_version 失效协议（参考 T-0106 修复建议） | feat | F03/pm+F02/parammodel | P0 | planned | Claude | L (~3-4d) | T-0164-P3 ✅ + T-0098 ✅ | wave-3 | `docs/design/pm-kpi-pipeline-improvements.md` §4.1 | 2026-05-22 | 替换 PMCollector 选指标这一步；多 worker 一致生效；单测覆盖路由命中/未命中/缓存失效；集成测试用 BLQ fixture 走全链路 |
| T-0164-P2 | **G2 PM 时序数据保留策略** — pm_metrics 挂 compression policy (7d) + retention policy；sys_configs 注入 5 个全局保留期键（15min/hourly/daily/weekly/monthly）+ 默认值（金字塔保留：15min=30d / hour=180d / day=2y / week=2y / month=5y）；UI 入口（系统设置页） | feat | F03/pm+ops | P0 | planned | Claude | S (~1d) | — | wave-3 | `docs/design/pm-kpi-pipeline-improvements.md` §4.2 | 2026-05-22 | 5 级粒度统一管理；hypertable 走 TimescaleDB policy，普通表走 G5 cron 清理任务；UI 改动联动 sys_configs 两条路径 |
| T-0164-P3 | **G3 聚合元数据驱动 + 合并 pm_counters/kpi_values 为 pm_metrics + 删物化视图** — DROP MATERIALIZED VIEW pm_counters_hourly + DROP TABLE pm_counters + kpi_values；CREATE pm_metrics（TS hypertable，含 metric_type 区分 counter/kpi + G4 三时间字段）；KPIEngine counter 聚合按 statis_type 路由（sum/avg/max/pct 四路）；perf_indicators_*.statis_type 启用消费 | feat | F03/pm+infra | P0 | planned | Claude | L (~3-4d) | T-0164-P4 ✅ | wave-3 | `docs/design/pm-kpi-pipeline-improvements.md` §4.3 | 2026-05-22 | 本地无生产数据可直接删表；migration up/down 配对（down 仅占位 CREATE 空表）；pg_dump 备份到 /tmp/pm-kpi-pre-G3-backup-20260522.sql；单测覆盖四种 statis_type |
| T-0164-P4 | **G4 PM 时间窗 + 入库时间字段** — pm_metrics 加 start_time / end_time / ingest_time 三字段 + 必要索引；parser.go 补读 `fileHeader/measCollec/@beginTime` + `fileFooter/measCollec/@endTime`；collector handler 写库时填 ingest_time；校验 start<end 且 end-start ≤ 配置上限 | feat | F03/pm | P0 | planned | Claude | M (~1.5d) | — | wave-3 | `docs/design/pm-kpi-pipeline-improvements.md` §4.4 | 2026-05-22 | 唯一性维度：（device_sn, metric_path, end_time, start_time, granularity）；measObjLdn 解析（LDN 通用结构）；单测覆盖 parser 三时间字段 + 校验分支 |
| T-0164-P5 | **G5 自然日历桶预聚合（hourly/daily/weekly/monthly）** — 4 张聚合表（pm_metrics_hourly hypertable + daily/weekly/monthly 普通表）；4 个 cron job（:05 / 00:05 / 周一 00:10 / 月 1 日 00:15）注册到 G8 任务框架；聚合按 statis_type 路由（sum/avg/max 从 15min 算；pct 从当前级 counter 取值代入 arithmetic）；QueryAggregated 按粒度+行维度路由；设备组维度聚合扩展 | feat | F03/pm | P0 | planned | Claude | L (~4-5d) | T-0164-P1 ✅ + T-0164-P3 ✅ + T-0164-P8 ✅ | wave-3 | `docs/design/pm-kpi-pipeline-improvements.md` §4.5 | 2026-05-22 | daily/weekly/monthly 不用 hypertable（量小 + 按月业务查询为主）；统一从 15min 原始表算（不走金字塔）；保留期接 G2 sys_configs；TimescaleDB Continuous Aggregate 不用（理由见设计 §4.5）；单测 + SQL 注入历史数据集成测试 |
| T-0164-P6 | **G6 前端"性能查看"单 tab + 仪表盘 + 面板模型** — 三 tab（概览/趋势探查/报表）→ 单 tab"性能查看" + Dashboard + Panel 两概念模型；制式顶层切换；对比双模式；点对点用户分享 / 协作；派生（fork）；KPI 卡片配置持久化 user_preferences；多皮肤评估（webcode/v2/v3） | feat | frontend-core + frontend | P0 | planned | Claude | XL (~6-7d) | T-0164-P5 ✅ | wave-3 | `docs/design/pm-kpi-pipeline-improvements.md` §4.6 | 2026-05-22 | 业务层（types/services/hooks/store/i18n/mock）入 frontend-core；webcode 改造 pages/performance；按 panel 行维度静态分流数据加载；playwright MCP 自测：创建仪表盘 / 加 panel / 切粒度 / 对比 / fork / 分享链路，截图存 .playwright-mcp/ |
| T-0164-P7 | **G7 自定义聚合任务（oneshot + continuous）** — 复用 pm_tasks per-module（不进 G8）；扩 pm_tasks 加 adhoc_aggregation 类型 + 模式标记 + cron expr 字段；pm_adhoc_aggregation_results 快照表（hypertable + drop_chunks 整块清，全局保留期接 G2 sys_configs，默认 1 年）；REST API CRUD + 进度上报 SSE；panel ↔ task 独立生命周期 | feat | F03/pm + frontend | P0 | planned | Claude | L (~4d) | T-0164-P3 ✅ + T-0164-P8 ✅ | wave-3 | `docs/design/pm-kpi-pipeline-improvements.md` §4.7 | 2026-05-22 | 数据源统一 pm_metrics 15min 原始表；多粒度多选；持续模式 cron 调度时刻与 G5 对齐；UI 集成点：性能模块 → 自定义聚合入口；单测 + 集成测试 |
| T-0164-P8 | **G8 通用任务框架（async_jobs + 心跳 + 僵尸检测 + cron 触发 + 分布式锁）** — async_jobs 表（id/type/status/heartbeat_at/started_at/lock_owner 等）；internal/core/asyncjob 包：JobRunner 接口 + 心跳上报 + 僵尸检测 cron + 分布式锁（PG advisory lock）+ cron 触发器；承载 G5 cron 实例 + 未来批量计算 | feat | infra+ops | P0 | planned | Claude | L (~3-4d) | — | wave-3 | `docs/design/pm-kpi-pipeline-improvements.md` §4.8 | 2026-05-22 | G7 走 pm_tasks per-module 不进 G8（用户 2026-05-19 设计文档 §4.7/§4.8 决策）；单测覆盖心跳/僵尸/重启续跑/分布式锁竞争 |

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
