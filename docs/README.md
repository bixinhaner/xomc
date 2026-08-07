# docs/ 文档总索引

> OMC 文档分流台 —— 一句话总览：`docs/` 下 530+ 篇 Markdown 按**活文档（权威，每天/每周维护）→ 当前架构设计（canonical，按主题选首选篇）→ 历史归档（只读，仅供溯源）**三层组织，目标是让你 5 分钟内定位到任一主题的权威单篇。
>
> **如何使用本索引**：
> 1. 找**当前怎么做 / 现状是什么** → 看【一、活文档】或【二、当前架构设计】，每个主题只读列出的首选 1-3 篇。
> 2. 找**当初为什么这么定 / 某个老方案** → 看【三、历史归档区】，明确这些只读、非现状权威。
> 3. 本索引**不逐个枚举 530+ 篇**，只给每主题的首选阅读 + 活/归档区分；同主题有多代文档时，**日期最新或标注 canonical 的那篇为准**，其余视为归档。
> 4. 其它入口文档分工：`README.md`（人类上手 / 仓库地图）· `CLAUDE.md`（AI 全局指导）· `CONTEXT.md`（领域术语表）· `omcgo/CLAUDE.md`（后端详细指导）· `docs/expert-personas.md`（审查专家视角）。

---

## 一、活文档（权威，持续维护）

> 这些文档反映**当前真实状态**，是日常决策的事实基准。改了系统行为应同步改这里。

### 项目治理（`docs/project/`）

| 主题 | 文档 | 说明 |
|------|------|------|
| 历史任务归档 | `project/backlog.md` | ⚠️ **已冻结** —— 任务源已迁 GitHub Issues，本文件保留旧流水线的 closing-evidence 追溯链，不再新增任务 |
| 风险登记册 | `project/risk-register.md` | 活文档，风险 + Owner + 复盘日 |
| 完成定义 / 发布门控 | `project/dod.md` · `project/release-gate.md` | DoD 清单 / Release Gate 清单 |
| 全流程编排 Skill | `.claude/commands/ship.md` | **活** —— `/ship` 一键贯穿全流程，只调度委派标准套件 + 项目自有 Skill |
| 旧开发流水线设计 | `project/dev-pipeline-design-20260420.md` | ⚠️ **归档** —— 旧 `/dev-pipeline` S0–S7 设计，已被 `/ship` 取代，仅留作硬门 rationale 参考 |
| 前端历史多皮肤方案（已归档） | `project/frontend-multi-skin-plan-20260422.md` | V2/V3 已移除；正文仅供历史追溯 |
| 里程碑 / 冲刺 / 发布 | `project/milestone/2026Q2-to-RC.md` · `project/sprint/sprint-{09,10,11}.md` · `project/release/RC-2026Q2-001.md` | 当前 RC 路线、活跃 Sprint、发布记录（更早 sprint 完成度快照见归档区）|
| 决策记录 | `project/decision-records/` | Wave 级决策留档 |

### 交付 / 可观测性 / 故障演练

| 主题 | 文档 | 说明 |
|------|------|------|
| 交付与部署手册 | `operations/OMC交付构建快速上手.md` · `OMC离线交付包构建手册（构建侧）.md` · `OMC内网离线部署手册（运维侧）.md` | 离线交付包构建与内网部署 SOP |
| 可观测性使用手册 | `operations/OMC可观测性使用手册.md` · `operations/资源、队列与存储保护监控说明-20260728.md` | OTel Collector → Tempo/Loki/Prometheus 落地；资源、队列与存储写入保护监控入口和实现说明 |
| 告警处置 / 诊断 | `operations/告警处置Runbook.md` · `diag-mml-gpn-probe.md` · `troubleshoot-mml-rpc.md` | 运维侧处置与 MML 诊断 |
| 故障演练 Runbook（8 篇）| `runbook/` | `acs-overload` · `db-backup-restore` · `disaster-recovery` · `pg-failover` · `redis-failover` · `nats-failover` · `mr-task-troubleshooting` · `parammodel-iteration-annotation-guide` |

### Agent 约定 / 方法论 / 参考

| 主题 | 文档 | 说明 |
|------|------|------|
| Agent 约定（matt-pocock 套件）| `agents/issue-tracker.md` · `triage-labels.md` · `domain.md` · `git-boundaries.md` | 任务源（GitHub Issues）/ triage 标签 / 单上下文域文档 / git 软硬约束矩阵 |
| 审查专家视角 | `expert-personas.md` | 文件路径→专家激活映射 + 各专家审查清单（CLAUDE.md §16 的展开篇）|
| 工程方法论（3 篇）| `methodology/从0到生产可发布完整方法论.md` · `AI承诺对峙清单.md` · `整改运行手册-2026Q2.md` | 方法论与整改运行手册 |
| PM / KPI 知识库 | `ref/pm-metrics-knowledge.md` | PM 统计口径、入口矩阵、原始文件取证、页面与真实数据验证清单 |
| 历史经验参考 | `ref/migration-pitfalls.md` · `ref/three-library-xml-import-history.md` · `ref/mml-validation-rules/` | 迁移踩坑库 / 三库 XML 导入演进史 / MML 参数校验规则快照 |

### 跨域 SOP / 接口契约

| 主题 | 文档 | 说明 |
|------|------|------|
| 消息队列全流程 | `消息队列全流程流转说明书.md` | **canonical SOP** —— NATS/任务/事件全链路（旧版分析见归档区）|
| EventBus 主题 | `eventbus/topics.md` | 事件主题命名与清单 |
| omcctl 手册 | `omcctl-manual.md` | 运维 CLI 手册 |
| 告警 API | `alarm-api.md` | F04 告警接口契约 |

---

## 二、当前架构设计（canonical，按主题选首选篇）

> 同一主题常有多代设计文档，**下表每主题给出当前首选 1-3 篇**（日期最新 / 标注 canonical）。同主题的更早版本一律视为归档（见第三段）。全部位于 `docs/design/` 除非另注。

| 主题分区 | 首选阅读 | 备注 |
|---------|---------|------|
| **参数 / KPI / 告警 三库平台化** | `参数-KPI-告警-整合设计方案.md` · `three-library-xml-import-redesign-20260604.md` | 三库统一平台化 + XML 导入重设计；演进史见 `ref/three-library-xml-import-history.md` |
| **产品中心** | `product-center-pages-redesign-20260528.md` | 产品装配件 + ProductRegistry 页面重设计（配图 `design/assets/product-*.png`）|
| **PM / KPI 管线** | `pm-kpi-pipeline-improvements.md` · `pm-metric-aggregation-dashboard-redesign-20260529.md` | 采集→多级聚合→仪表盘；实施计划 `project/plan-T-0164-*.md` |
| **北向 / OSS 接口** | `northbound/README.md` · `northbound/page-config-redesign-20260731.md` · `project/prd/F08-oss-protocol.md` | 北向专题入口 + 页面可配置化文件/Inventory/Socket/SNMP/API 综合设计；旧 SNMP Trap PRD 作为产品背景和子能力参考 |
| **MML 脚本任务** | `mml-script-txt-import-redesign-20260710.md` · `mml-script-task-device-bound-redesign-20260708.md` | TXT 导入式脚本库 + 按设备编排执行（本次设计以 TXT 导入重设计为首选）|
| **MML 控制台** | `mml-console-redesign-20260603.md` · `mml-console-architecture-overview-20260521.md` · `mml-empty-path-commands-spec-mapping-20260609.md` | 控制台重设计、架构总览与空 PATH 命令规范映射（MML 多代历史方案见归档区）|
| **设备管理** | `device-lifecycle-online-status-decouple-20260520.md` · `device-list-and-group-improvements-20260520.md` · `device-detail-basic-fields-from-parameters-20260525.md` | 生命周期/在线状态解耦 + 列表分组 + 详情字段 |
| **跨域基础设施** | `统一文件传输任务引擎-设计方案.md` · `notification-center-design-20260519.md` | 统一文件传输任务引擎 / 通知中心 |
| **模块设计回填** | `参数设置页-设计.md`（↔ quicksettings）· `TR069报文跟踪-设计.md`（↔ trace）| 模块代码与设计对应 |
| **可观测性迁移** | `observability-otelcol-migration-plan-20260520.md` | OTel Collector 迁移计划（落地手册见活文档 `operations/OMC可观测性使用手册.md`）|
| **告警 / 库重设计** | `alarm-library-redesign-20260529.md` · `kpi-library-redesign-20260529.md` | 告警库 / KPI 库重设计 |
| **运营商规范实现** | `cmcc-tdlte-v2.3-implementation-plan-20260521.md` | CMCC TD-LTE v2.3 实现计划（规范原件在 `omcgo/规范/`）|
| **测试方案** | `test/mml-console-v2-test-{plan,result,report-matrix}-20260606.md` | MML console-v2 测试三件套 |

> PRD 入口：`docs/project/prd/`（按功能域 F01-F10 + T-NNNN 组织，每个功能/子功能一份）。

---

## 三、历史归档区（只读，仅供溯源，非现状权威）

> ⚠️ **本段所有内容仅供追溯"当初为什么这么定 / 某个老方案长什么样"，不代表系统现状。判断现状一律以第一、二段为准。**

### 已归档到 `docs/archive/`

| 子目录 | 内容 | 归档原因 |
|--------|------|---------|
| `archive/analysis/` | Phase A-D 分析、Sprint0-9 完成度、20260323 会话诊断、容量规划、数据流向全景、开源组件分析、rpc-flow、联合调试验证、各类 fix-summary | 阶段性一次性分析，结论已并入活文档 / 代码 |
| `archive/reports/` | expert-analysis/review-20260322（3 份）、expert-personas-proposal（已落地为 `expert-personas.md`）、cpe-simulator 合规/集成报告、基站上线数据流程图、设备管理调研三件套、rbac 设计+实施计划、implementation-completeness、seed_importer 验证、alarm-test-report、menu_tables_design.sql | 早期一次性报告 / 调研 / 已实现设计 |
| `archive/legacy-handoff/` | 项目接手时的原始交付参考：`Back-end/`（2026-03~04 早期模块设计）、`Front-end/`（GIS 选型 / 页面模板）、`UED/`（从遗留 JSP 反推的页面规格）。**原 `db/`（small_cell MySQL dump 6MB）+ `Back-end/seed-data/`（KPI/PM 指标 SQL 3MB）已删除**——遗留 MySQL / 已被 `omcgo/data/` XML 字典取代，无代码引用 | 早期外部交付设计，供考古 |
| `archive/skins/` | `DESIGN-{Apple,Stripe,Vercel,Raycast,Supabase,Claude,Linear}.md` | 多皮肤设计灵感参考，无 OMC 业务内容 |

### 原地只读（不移动）

- **`review-report/**`**（240+ 篇，按日期分目录）—— 每提交评审 / 验证快照，**append-only 审计日志**，被 `backlog.md` 约 65 处「Closing Evidence」引用，故原地保留只读，不进 `archive/`。
- **`project/backlog.md` 及 `project/backlog/`** —— 已**冻结为历史任务归档**（活任务源转 GitHub Issues），保留 T-NNNN ↔ review-report 追溯链。

### 仍在原位、按历史对待（权威性已被取代，后续可继续下沉）

> 以下仍散落原目录，但现状一律以第一、二段为准，引用时视为历史快照：
> - `design/` 旧参数模型设计（`parameter-model-*` / `parameter-rpc-*` / `parameter-template-*` / `参数树自动发现*`，被三库平台化取代）、MML 多代历史方案（`mml-rebuild-plan` / `mml-restore-old-interaction` / `mml-requirements-design` / `mml-console-qa-test-report-20260523*`，被 `mml-console-redesign-20260603` 取代）
> - `project/plan-*.md`（已交付项的一次性实施计划）、`project/{开发实施计划*,整改路线图-2026Q2,全模块对齐开发计划}.md`
> - `消息队列使用分析-20260422.md`（被 canonical `消息队列全流程流转说明书.md` 取代）
> - `qa-report/**`（旧三栏 MML console 测试，被 `test/mml-console-v2-*` 取代）、`superpowers/plans/`、`security/scan-baseline-*`、`perf/slow-queries-top10`

> **保留为活文档的协议参考**：`tr069-protocol-compliance-report.md` 与 `acs-verification-plan.md` 仍在 `docs/` 根（README 协议入口引用），作长期协议合规参考，不归档。
