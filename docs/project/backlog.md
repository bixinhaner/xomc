# OMC 需求池（Backlog）— 任务状态表

> **性质**：活文档。项目**唯一任务清单**。所有新工作先入池再进流水线。
> **守护人**：项目经理（`CLAUDE.md §16.11`）+ 产品经理（`§16.10`）+ QA/发布经理（`§16.12`）轮值
> **设计依据**：`docs/project/dev-pipeline-design-20260420.md §11` L0 Backlog 层
> **初始化数据**：反向索引自 `docs/project/milestone/2026Q2-to-RC.md` + `risk-register.md` + 近期 commit
> **最近更新**：2026-04-27（Wave 1 启动）

---

> ⛔ **[Wave 3 立项中] 2026-04-28 ~ 2026-08-03 — 硬化期**
>
> Wave 1 满分 8.0/8 ✅ + Wave 2 12/13 ✅（AI 极限，仅 W2.A.3 短信凭据外部阻塞）已完。
> Wave 3 章程立项启动（章程见 `docs/methodology/AI承诺对峙清单.md` 第三章半），**15 条 W3.E.x ~ W3.I.x 机械承诺**：
> - **Block E NATS** (W3.E.1~3): NATS JetStream 实现 / 关键事件迁移 ≥3 类 / 故障演练
> - **Block F 性能** (W3.F.1~3): 5K 设备压测 24h / 慢查询 top10 / 连接池配额监控
> - **Block G 安全** (W3.G.1~3): 安全扫描进 CI / 审计日志完整 / 敏感信息脱敏
> - **Block H 部署** (W3.H.1~3): K8s manifests / 滚动更新零停机 / 异地备份 RTO<1h/RPO<15min
> - **Block I GA** (W3.I.1~3): Release Gate 9 章全勾 / 灰度发布演练 / 回滚演练 5min
>
> **违反 = PR 直接关闭**。期间所有 PR 必须挂 W3.x 整改子任务（T-0010/T-0023/T-0024 + T-0058~T-0069）。
> Wave 3 退出门槛：≥ 11/15 ✅（73%，2026-08-03 终审对峙日）；< 7/15 + 用户尽责 = AI 嘴炮（第八章认账）。
> 满分需配套外部凭据：T-0009（短信，user/PM 推动） + 真实 K8s 集群 + NATS 实例 staging 部署。
> 配套契约：`docs/methodology/从0到生产可发布完整方法论.md` · `docs/methodology/AI承诺对峙清单.md`

---

## 0. 如何使用

**所有人**：有新想法/新 bug/新建议 → 在 §8 Proposed 加一行，或调用 `/dev-pipeline backlog add <title>`。
**Triage（每周一）**：PM/PgM/QA 把 Proposed 逐条判决 → Triaged / Deferred / Rejected，补齐 Type/Domain/Prio/Est 字段。
**Sprint Planning（每 2 周）**：PgM 主持，从 Triaged 池按优先级+容量挑选进当前 Sprint，转 Planned。
**每日**：开发者 `/dev-pipeline next` 出队一条 → `pick T-NNNN` 走 S0→S7。

**修改纪律**：除"Updated"日期外，其他字段变更**必须**记入 §10 变更日志或关联 commit。

---

## 1. 图例

### 1.1 Type
`feat` 功能 · `bug` 缺陷修复 · `td` 技术债 · `ref` 重构 · `docs` 文档 · `hot` hotfix · `proc` 流程/工具

### 1.2 Domain
`F01-F10` 功能域（见 `CLAUDE.md §6`）· `infra` 基础设施 · `ops` 运维/可观测 · `frontend` 前端 · `process` 流程/AI/工具链

### 1.3 Priority
`P0` 阻塞 RC/GA · `P1` 本里程碑必须 · `P2` 应该做 · `P3` 想做

### 1.4 State
`proposed` 刚登记，待分诊
`triaged` 已分诊，未排期
`planned` 已进 Sprint
`in_design / in_dev / in_review` 对应流水线 S2/S3/S5
`blocked` 依赖未达/外部阻塞
`done` S7 关闭
`deferred` 判决延后
`rejected` 判决不做

### 1.5 Est（粗估工作量）
`S` ≤ 1 天 · `M` 1–3 天 · `L` 3–5 天 · `XL` > 5 天（需拆）

---

## 2. 仪表盘（自动化前由 Triage 会议手动维护）

| 指标 | 当前 | 目标 | 备注 |
|------|------|------|------|
| Total tasks | 132 | — | 含 T-0098 umbrella + 36 sub-task；变更明细见 changelog |
| `done` | 86 | — | 详细 Closing Evidence → `backlog/done/2026Q2.md`（+6 T-0098-P1-01..P1-06 + **+2 T-0098-P2-01/P2-02** 2026-05-07，P1 wave 收官 + P2 wave 入口 1 + 参数轨脊柱）|
| `in_dev` | 2 | — | T-0095 KPI（D10=A 已决议吸收至 T-0098-P2-09/P3-03/P4-05，待 sub-task 启动合并关闭）；T-0098 数据字典平台化（umbrella；P1 6 条 + P2-01/P2-02 2 条 done；P2 剩 9 条等 wave-batched 串行/并行落地；P3-P5 19 子任务待下轮 planning） |
| `in_design` | 0 | — | — |
| `planned` | 15 | — | P1 历史升格行已 done；P2-01/P2-02 当日 done；剩 P2-03..P2-11 9 条 wave-3 + Owner=Claude；其他 6 条来自其他任务 |
| `triaged` | 26 | — | T-0098 19 sub-task 占 19 条（P3-P5；P1 6 条 + P2 11 条 2026-05-07 升格 planned）；剩 T-0030 / T-0090 / T-0091 / T-0093 / T-0035 / T-0096 / T-0097 不阻塞主链路 |
| `blocked` | 0 | ≤ 3 | — |
| `proposed` 积压天数 | 0 | ≤ 7 | — |
| **P0 风险关闭数** | **1 / 5** | 5 / 5 | R-005 已关；R-001/002/003/004 Open（注：**P1 R-103 / R-104** 已关闭但不计入 P0 — R-103 在 T-0015 后关闭；**R-104 在 T-0027 后关闭 2026-05-06**）|
| **Wave 1 计分** | **8.0 / 8 ✅** | ≥ 6/8 | 满分；提前 13 天达成（对峙日 2026-05-11） |
| **Wave 2 章程计分** | **12 / 13 ✅ (AI 极限)** | ≥ 9/13 | A.1+A.2+A.4+A.5 + B.1-4 + C.1-3 + D.1 全 PASS（92%）；仅 W2.A.3 短信（T-0014 deps T-0009 凭据外部，AI 不可解锁）|
| **Wave 3 章程计分** | **11 / 15 ✅ (退出门槛达成)** | ≥ 11/15 | **🎉 真过门 73%**：E.1+E.2+E.3 (NATS) / F.2+F.3 (性能/连接池) / G.1+G.2+G.3 (安全) / H.1+H.3 (K8s/异地备份) / I.1 (Release Gate)；剩余 4 项：F.1 (5K 24h 压测)/H.2 (零停机演练)/I.2 (灰度演练)/I.3 (回滚演练) — 均需 staging 环境 |
| E2E 累计用例 | **549** ✅ | 200 | T-0057 修真 bug 后实跑 549 PASS / 0 FAIL / claim 129（W1.6 27 + W2D 99 + W2.A.4 4 + Sprint 0-9 framework 修复后扩张）；超目标 200 ✅ + 解锁 T-0025 累计阈值 ≥150 |
| Sprint 承诺完成率 | — | > 75% | 待 Sprint-01 首次回顾 |

**健康度警报**：当前无。

---

## 3. Active — Planned + In-flight（排序：Sprint 升序，同 Sprint 内 Prio 升序）

> 🚨 **Wave 整改期间（2026-04-27 ~ 2026-08-03）：下方 Wave 队列优先于 Sprint 排序**。
> 从队列上往下挑，每天一次 `/dev-pipeline pick T-NNNN` 出队，做完打勾。

### Wave 整改执行队列（2026-04-27 起 — 唯一执行优先序）

#### 🟢 Wave 1 · 止血（W1-W2，2026-04-27 ~ 2026-05-11）— ✅ 已完结 8/8（满分提前 13 天）

详细队列 / 计分史 / W1.6 spec 决策 → [`backlog/waves/wave-1.md`](backlog/waves/wave-1.md)

#### 🟡 Wave 2 · 收尾冲刺（W3-W8，2026-05-12 ~ 2026-06-22）— ✅ 已完结 12/13（仅 W2.A.3 短信凭据外部阻塞）

Block A 通知 / Block B 测试债 / Block C 前端 / Block D E2E 详细队列 → [`backlog/waves/wave-2.md`](backlog/waves/wave-2.md)

#### 🔴 Wave 3 · 硬化到 GA（W9-W14，2026-06-23 ~ 2026-08-03）

**Block E · NATS 改造（W9-W10）**

| 序 | Task | 标题 | 章程 |
|----|------|------|------|
| E.1 | T-0010 | NATS JetStream EventBus（cov 63.1%，wrapHandler 100%）| ✅ W3.E.1 2026-04-28 |
| E.2 | T-0058 | 5 类 subject 迁移 (alarm/device/command/task/pm) + topics.md | ✅ W3.E.2 2026-04-28 |
| E.3 | T-0059 | NATS 故障演练 framework (Runbook + 3 integration test stub) | ✅ W3.E.3 2026-04-28 (framework) |

**Block F · 性能（W11）**

| 序 | Task | 标题 | 章程 |
|----|------|------|------|
| F.1 | T-0023 | 5K 设备压测稳定 24h + p95<SLO | W3.F.1 |
| F.2 | T-0060 | 慢查询审计 top 10 优化 | W3.F.2 |
| F.3 | T-0061 | 连接池配额监控（pgxpool/Redis/NATS） | W3.F.3 |

**Block G · 安全（W12）**

| 序 | Task | 标题 | 章程 |
|----|------|------|------|
| G.1 | T-0062 | 安全扫描进 CI（gosec/govulncheck/npm audit + R-310~R-314 风险草案）| ✅ W3.G.1 2026-04-28 |
| G.2 | T-0063 | 审计日志完整 5 类（audit pkg + global singleton + 5 埋点）| ✅ W3.G.2 2026-04-28 |
| G.3 | T-0064 | 敏感信息脱敏（redact + zap + errors 三层集成）| ✅ W3.G.3 2026-04-28 |

**Block H · 部署（W13）**

| 序 | Task | 标题 | 章程 |
|----|------|------|------|
| H.1 | T-0065 | K8s manifests 全套（3 部署单元 × 4 manifests + HPA） | W3.H.1 |
| H.2 | T-0066 | 滚动更新 + 零停机演练（5xx=0%） | W3.H.2 |
| H.3 | T-0067 | 异地备份 + RTO<1h/RPO<15min | W3.H.3 |

**Block I · GA（W14）**

| 序 | Task | 标题 | 章程 |
|----|------|------|------|
| I.1 | T-0024 | Release Gate 9 章节全勾 | W3.I.1 |
| I.2 | T-0068 | 灰度发布演练 5%→25%→100% | W3.I.2 |
| I.3 | T-0069 | 回滚演练 5 分钟内回上一版本 | W3.I.3 |

**未章程化但仍属 Wave 3 范围**：T-0012 worker 重试/死信（NATS 配套，归 Block E 工作但章程未列）/ T-0026 Runbook ≥ 5 场景（Block H 部署相关，章程未列） / T-0025 RC 冻结（累计型 deps T-0006）

**Wave 3 退出（2026-08-03 终审对峙）**：≥ 11/15 ✅（73% 兑现门槛）+ 第四章 W14 末 7 长承诺 ≥ 5/7

#### ⚪ Wave 4 · GA 后专项（F08 SNMP/MTOSI 30 天）

T-0013（SNMP 骨架）→ T-0017（联调）→ T-0020（推送可靠性）

---

### ⚠️ 与 FREEZE 冲突的在途任务（PgM 必决）

**T-0095 F03 KPI 指标管理**（in_dev / sprint-01，最近更新 2026-04-26；2026-04-30 由 T-0027 改号，详见 §10 变更日志）

- **冲突**：FREEZE 禁止新功能立项；T-0095 是新功能（标准报表 + 站点报表）
- **PgM 必须在 2026-04-28 周一规划会前决定** 三选一：
  - **A. 暂停** — 标 `blocked`，等 W1 末再决定（浪费现有进度）
  - **B. Grandfather** — 限本周收尾，不开新子模块（推荐）
  - **C. Wave 内消化** — 重新归类到 Block F 性能监控前置
- **决策记录**：在本节追加一行 `2026-04-28 PgM 决策：选 X，理由 Y`
- **2026-05-07 PgM 决策（T-0098 D10=A 触发）**：T-0095 决议**吸收至 T-0098-P2-09 / T-0098-P3-03 / T-0098-P4-05**；不再独立推进；现有 in_dev 工作作为 T-0098-P2-09 起点；T-0095 状态留 in_dev 直至 T-0098 KPI 子任务启动时正式合并并关闭（State → done with absorbed evidence）。本决策同时关闭 FREEZE 冲突项（吸收方 T-0098 由 D5=B 准入 W3 实现收敛）。

**T-0098 参数 / KPI / 告警 数据字典平台化**（in_dev umbrella / 36 子任务已拆 §4.1，最近更新 2026-05-07）

- **冲突**：T-0098 是 feat 类（F02+F03+F04 跨域），与 Wave 3 硬化期 FREEZE 冲突；R-T0098-09 即对应风险
- **PgM 决策（2026-05-07，D5=B 全采纳推荐）**：选 **B 仅 P4/P5 暂停，P1-P3 继续**
  - **理由**：P1-P3（schema + dictloader + Loader + Registry + REST API，22 子任务）视为设计稿 v3 已交付的"实现收敛"，归 W3 整改范畴推进；P4（前端 4 页 + nav 一级菜单，8 子任务）+ P5（旧 datamodel 清理 + 文档收尾，6 子任务）推迟到 2026-08-03 W3 终审后启动
  - **子任务标记**：§4.1 中 P4-* / P5-* 行 Notes 列含 "**D5=B Wave 3 后启动**" 字样，sprint planning 时识别准入边界
  - **配套决策**：D1=A 保 KPI 表名 / D2=B DROP+CREATE alarm_definitions / D3=A 保自实现公式 / D4=A 后端架构产 products.xml / D6 默认分工（待具体人）/ D7=A 仅评估 v2/v3 / D8=A 单 PR 合 P1 迁移 / D9=B feature flag 保留至 P5 / D10=A 吸收 T-0095（详见 §10 changelog 2026-05-07 行）
  - **2026-05-07 PgM 后续决策（前戏完成）**：P1-01..P1-06 6 条**升格 planned / Sprint=wave-3 / Owner=Claude**（详 §4.1 + sub-task 文件）；wave-batched 模式（dev-pipeline §C.1）准入开发，Skip S0/S1，Footer 引用 `AI承诺对峙清单.md` W3 段 + `参数-KPI-告警-整合-实施计划.md`；**commit 形态 = 6 commit + 单 PR**（每 commit footer 各挂 `Backlog: T-0098-P1-0N`，PR 描述聚合 6 条；保留 D8=A 迁移 atomic 同时不破坏 §B6 footer 一对一）；可直接 `/dev-pipeline pick T-0098-P1-01` 开车
  - **下一步（已替代旧 sprint planning 计划）**：~~sprint planning 从 §4.1 P1-* 6 条 + 无依赖 P2-* 中按 Owner 容量挑选进 sprint-NN~~；P1-01..P1-06 已直接进 wave-3（D8=A 硬约束 = 同 PR 由"6 commit + 单 PR"形态实现）；P2-01..P5-06 30 条仍 triaged 等下次 sprint planning

---

### 任务出队方式

```bash
# 每个工作日早上做一次:
/dev-pipeline pick T-0040    # 例: 拉起 W1.3 走 S0-S7

# 或者按队列序号自动出队:
/dev-pipeline next            # 自动选下一个 🟡 ready 任务
```

详细生命周期 S0-S7 见 `docs/methodology/整改运行手册-2026Q2.md` 第 3 章。

---

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Risk/PRD | Sprint | Updated |
|----|-------|------|--------|------|-------|-------|-----|------|----------|--------|---------|
| T-0006 | E2E 用例补齐（贯穿 01→07 累计 ≥200） | td | infra | P0 | in_dev | QA | XL | — | R-002 | sprint-01..07 | 2026-04-20 |
| T-0007 | F04 告警邮件通道（EmailDispatcher + filter action notify_email + DI） | feat | F04 | P0 | done | Claude | M | — | R-001 / `AI承诺对峙清单.md` W2.A.1 / `prd/F04-alarm-notification.md` | wave-2 | 2026-04-28 |
| T-0008 | Prometheus/Grafana/AlertManager 容器编排 + 基础 dashboard | feat | ops | P1 | done | Claude | M | — | R-107 | wave-1 | 2026-04-28 |
| T-0009 | 短信服务商凭据申请启动（外部动作） | td | F04 | P0 | planned | PM | S | — | R-001 | sprint-01 | 2026-04-20 |
| T-0010 | NATS JetStream EventBus（NATSEventBus + ChannelEventBus 双实现 + cov 63.1%） | feat | infra | P0 | done | Claude | XL | — | R-004 / `AI承诺对峙清单.md` W3.E.1 | wave-3 | 2026-04-28 |
| T-0011 | F04 告警 Webhook 通道（retry/dead-letter/HMAC + FilterEngine 接 AlarmEngine.Process） | feat | F04 | P0 | done | Claude | M | — | R-001 / `AI承诺对峙清单.md` W2.A.2 / `prd/F04-alarm-notification.md` | wave-2 | 2026-04-28 |
| T-0012 | worker retry/DLQ 通用框架（dlq 子包+runner 装饰器+migration 000044+admin/replay+PM 接入示范，runner cov 96.8%）| feat | infra | P1 | done | Claude | L | T-0010 | R-106 / `prd/T-0012-worker-retry-dlq.md` | sprint-02..04 | 2026-04-28 |
| T-0013 | F08 SNMP Trap 骨架（PRD 365 行 + 10 .go 文件 + 50 测/82.1% cov + 依赖完全隔离，待 T-0017 接生产）| feat | F08 | P0 | done | Claude | L | — | R-003 / `prd/F08-oss-protocol.md` | sprint-03 | 2026-04-28 |
| T-0014 | F04 告警短信通道 | feat | F04 | P0 | planned | 电信+Go | M | T-0009 | R-001 / `prd/F04-alarm-notification.md` | sprint-03 | 2026-04-20 |
| T-0015 | License Enforcer + Monitor cron + 6 metrics + device.Create 闸（capacity/expiry/grace/multi-active/perpetual D1-D6 全实施）| feat | F06/license | P1 | done | Claude | M | — | R-103 / `prd/F06-license-enforcement.md` | sprint-03 | 2026-04-28 |
| T-0016 | 前端 Backup Tasks 真实接 4 hook + FTPConfig fallback 清理（R-102 部分关闭，3 followup T-0070/71/72）| feat | frontend | P1 | done | Claude | M | — | R-102 部分关闭 / `prd/T-0016-frontend-backup-business-logic.md` | sprint-05 | 2026-04-29 |
| T-0017 | F08 SNMP Trap 联调（staging ≥1 家运营商） | feat | F08 | P0 | planned | PM+架构 | L | T-0013 | R-003 | sprint-04 | 2026-04-20 |
| T-0018 | Software 灰度升级 — Canary stages [1/10/50/100]% + 失败率阈值 + cron monitor + 4 Admin API + 4 metric（R-101 部分关闭，回滚 T-0021）| feat | F06/software | P1 | done | Claude | L | — | R-101 / `prd/T-0018-software-canary-upgrade.md` | sprint-04..05 | 2026-04-29 |
| T-0019 | 前端 Software 消费 Canary API（frontend-core 7 type 字段 + 4 method + 4 hook + UpgradePlan UI 4 按钮 + canary stage 列）| feat | frontend | P1 | done | Claude | M | T-0018 ✅ | R-102 / `prd/T-0019-frontend-software-canary.md` | sprint-04 | 2026-04-29 |
| T-0020 | F08 推送可靠性（重试/去重/幂等） | feat | F08 | P0 | planned | PM+架构 | M | T-0017 | R-003 | sprint-05 | 2026-04-20 |
| T-0021 | Software 回滚增强（reason/source/target_firmware_id 审计 + canary 自动回退 opt-in + 3 metric + migration 000046 4 列）| feat | F06/software | P1 | done | Claude | M | T-0018 ✅ | R-101 关闭（合 T-0018） / `prd/T-0021-software-rollback-enhanced.md` | sprint-05 | 2026-04-29 |
| T-0022 | 前端 Topology/Report 补完（4 page 去 inline mock + 接 useDomains/useReportRecords/useDownloadReport/useReportDefinitions/useGenerateReport，mock 集中到 frontend-core）| feat | frontend | P1 | done | Claude | M | — | `prd/T-0022-frontend-topology-report.md` | sprint-05 | 2026-04-28 |
| T-0023 | 5K 设备压测稳定 24h + p95<SLO（W3.F.1） | td | infra | P0 | planned | 架构+运维 | L | T-0010 | `AI承诺对峙清单.md` W3.F.1 | wave-3 | 2026-04-28 |
| T-0024 | Release Gate 9 章节 [ ] 66→0（[x]=42 / [N/A]=13 全部映射 follow-up，W3.I.1） | proc | process | P0 | done | Claude | M | T-0023 | `AI承诺对峙清单.md` W3.I.1 / `release-gate.md` | wave-3 | 2026-04-28 |
| T-0025 | RC 冻结 + 冒烟用例 20 条（smoke_test.sh 8 域 20 用例 + RC-2026Q2-001.md 9 章 + git tag rc-2026Q2-001；本地实测 19/20，staging 应达 20/20）| td | infra | P0 | done | Claude | M | T-0006@累计≥150 ✅ (实跑 549) | `prd/T-0025-rc-freeze-smoke.md` | sprint-07 | 2026-04-29 |
| T-0026 | Runbook 体系 6 份达标 ≥5（pg/redis/acs 三新 + 既有 nats/dr/db-backup，1491+ 行） | docs | ops | P0 | done | Claude | M | — | `release-gate.md §3.4` | sprint-07 | 2026-04-28 |
| T-0095 | F03 KPI 指标管理（标准报表 + 站点报表）— **2026-05-07 D10=A 决议吸收至 T-0098-P2-09/P3-03/P4-05；不再独立推进；待 T-0098 KPI 子任务启动时合并工作并关闭** | feat | F03 | P1 | in_dev | 电信+前端 | XL | — | T-0098 D10=A 吸收 / 详见 §3 FREEZE 冲突段 + §10 changelog 2026-05-07 | sprint-01 | 2026-05-07 |
| T-0040 | acs/worker 加 `/healthz` + `/readyz`（W1.3） | td | infra | P0 | done | Claude | S | — | `AI承诺对峙清单.md` W1.3 | wave-1 | 2026-04-27 |
| T-0041 | `internal/core/middleware/ratelimit` per-IP token bucket + router 接入（W1.4） | feat | infra | P0 | done | Claude | M | — | `AI承诺对峙清单.md` W1.4 | wave-1 | 2026-04-27 |
| T-0042 | 数据库定时备份脚本 + 一次恢复演练（W1.8） | proc | ops | P0 | done | Claude | S | — | `AI承诺对峙清单.md` W1.8 | wave-1 | 2026-04-27 |
| T-0043 | 通知模板 + 历史记录（migration 000041 + 17 后端文件 + 11 前端文件 + DI/路由整合 + 4 e2e claim） | feat | F04 | P0 | done | Claude | L | T-0007,T-0011 | `AI承诺对峙清单.md` W2.A.4 / `prd/F04-alarm-notification.md` | wave-2 | 2026-04-28 |
| T-0044 | notification/ 模块测试覆盖率 27.6% → 78.4%（≥ 70%）| td | infra | P0 | done | Claude | M | T-0007,T-0011,T-0043 (T-0014 凭据外部，user 明示放开) | `AI承诺对峙清单.md` W2.A.5 | wave-2 | 2026-04-28 |
| T-0045 | task/ 模块测试覆盖率 ≥ 70%（CWMP map / reboot closer / completion router） | td | infra | P0 | done | Claude | M | — | `AI承诺对峙清单.md` W2.B.1 | wave-2 | 2026-04-28 |
| T-0046 | events/ 模块测试覆盖率 ≥ 60% + 补 service 层（实测 87.6%） | td | infra | P0 | done | Claude | M | — | `AI承诺对峙清单.md` W2.B.2 | wave-2 | 2026-04-28 |
| T-0047 | core/ 模块测试覆盖率 43.2% → 55.7%（任务卡 16% 基线已过时） | td | infra | P1 | done | Claude | M | — | `AI承诺对峙清单.md` W2.B.3 | wave-2 | 2026-04-28 |
| T-0048 | mr/ 补 service 层（thin facade + 11 测） | ref | F05 | P1 | done | Claude | S | — | `AI承诺对峙清单.md` W2.B.4 | wave-2 | 2026-04-28 |
| T-0049 | syslog/ 补 service 层（thin facade + 10 顶层 + 5 sub-test） | ref | F06/syslog | P1 | done | Claude | S | — | `AI承诺对峙清单.md` W2.B.4 | wave-2 | 2026-04-28 |
| T-0050 | provision/ 补 service 层（facade 包既有 engine/orchestrator/state_machine） | ref | F09 | P1 | done | Claude | S | — | `AI承诺对峙清单.md` W2.B.4 | wave-2 | 2026-04-28 |
| T-0051 | interop/ 补 service 层（facade 包既有 runner/validator/report） | ref | F10 | P1 | done | Claude | S | — | `AI承诺对峙清单.md` W2.B.4 | wave-2 | 2026-04-28 |
| T-0052 | frontend-core hooks/services/api 对齐（24 → 28 hooks，gap 5 → 1） | ref | frontend | P1 | done | Claude | M | — | `AI承诺对峙清单.md` W2.C.1 / R-102 | wave-2 | 2026-04-28 |
| T-0053 | 前端 any 类型清零（20 → 0） | td | frontend | P1 | done | Claude | S | — | `AI承诺对峙清单.md` W2.C.2 | wave-2 | 2026-04-28 |
| T-0054 | DeviceGrouping 拆分（max 650 → 303 行 / 6 子组件 + 6 hooks） | ref | frontend | P1 | done | Claude | M | — | `AI承诺对峙清单.md` W2.C.2 | wave-2 | 2026-04-28 |
| T-0055 | 前端 vitest 覆盖率 0% → lines 66.66% / statements 54.7% | td | frontend | P1 | done | Claude | L | T-0052,T-0053,T-0054 | `AI承诺对峙清单.md` W2.C.3 | wave-2 | 2026-04-28 |
| T-0056 | e2e_verify.sh framework 段 95 FAIL → 9 FAIL（86 用 check_status_in 合理放宽 / 9 真 bug 暴露记 §3） | td | infra | P0 | done | Claude | M | T-0006 | `AI承诺对峙清单.md` W2.D.1.b | wave-2 | 2026-04-28 |
| T-0057 | errors.NotFound 错误码映射 + PG SQLSTATE 分类（6 模块 6 sub-agent 并行修 9 处真 bug + 多 bonus 修） | bug | F04+device+F05+license+topology+ops | P1 | done | Claude | M | T-0056 | T-0056 verify-md §3 真 bug 列表 / `AI承诺对峙清单.md` W2.D.1（已解锁字面 Fail=0） | wave-2 | 2026-04-28 |
| T-0058 | NATS 关键事件迁移 5 类 subject（alarm/device/command/task/pm + topics.md） | feat | infra | P0 | done | Claude | M | T-0010 | `AI承诺对峙清单.md` W3.E.2 / R-004 | wave-3 | 2026-04-28 |
| T-0059 | NATS 故障演练 framework（Runbook + integration test stub，真演练待 staging） | proc | infra | P0 | done | Claude | M | T-0010,T-0058 | `AI承诺对峙清单.md` W3.E.3 / R-004 | wave-3 | 2026-04-28 |
| T-0060 | 慢查询审计 SlowQueryTracer + ChainTracer + 11 索引（W3.F.2） | perf | infra | P1 | done | Claude | M | — | `AI承诺对峙清单.md` W3.F.2 | wave-3 | 2026-04-28 |
| T-0061 | 连接池监控 pgxpool/Redis/NATS gauge+counter + 5 条 alerts.yml（W3.F.3） | feat | ops | P1 | done | Claude | M | — | `AI承诺对峙清单.md` W3.F.3 | wave-3 | 2026-04-28 |
| T-0062 | 安全扫描进 CI（gosec/govulncheck/npm audit + R-310~R-314 草案登记） | proc | ops | P0 | done | Claude | M | — | `AI承诺对峙清单.md` W3.G.1 / R-310~R-314 草案 | wave-3 | 2026-04-28 |
| T-0063 | 审计日志完整 5 类（cross-module audit pkg + global singleton 自然解耦避 DI） | feat | admin | P0 | done | Claude | M | — | `AI承诺对峙清单.md` W3.G.2 | wave-3 | 2026-04-28 |
| T-0064 | 敏感信息脱敏（redact pkg + zap helper + errors AbortWithError 三层集成） | td | infra | P1 | done | Claude | M | — | `AI承诺对峙清单.md` W3.G.3 | wave-3 | 2026-04-28 |
| T-0065 | K8s manifests 全套 17 yaml（app/acs/worker × Deploy/Service/CM/Secret/HPA + namespace + ingress，W3.H.1） | feat | ops | P0 | done | Claude | L | — | `AI承诺对峙清单.md` W3.H.1 | wave-3 | 2026-04-28 |
| T-0066 | 滚动更新 + 零停机演练（HTTP 5xx=0% during deploy，W3.H.2） | proc | ops | P0 | planned | 运维 | M | T-0065 | `AI承诺对峙清单.md` W3.H.2 | wave-3 | 2026-04-28 |
| T-0067 | 异地备份扩展 db_backup.sh +186 行 + DR Runbook 9 章 388 行（实测 RTO/RPO 待 staging，W3.H.3） | feat | ops | P0 | done | Claude | M | — | `AI承诺对峙清单.md` W3.H.3 | wave-3 | 2026-04-28 |
| T-0068 | 灰度发布演练 5%→25%→100%（W3.I.2） | proc | ops | P0 | planned | 运维+QA | M | T-0065 | `AI承诺对峙清单.md` W3.I.2 | wave-3 | 2026-04-28 |
| T-0069 | 回滚演练 5min 内回上一版本（W3.I.3） | proc | ops | P0 | planned | 运维+QA | M | T-0065 | `AI承诺对峙清单.md` W3.I.3 | wave-3 | 2026-04-28 |
| T-0070 | 前端 BackupSchedule UI 重设计 — wholesale replace 配置文件 demo + 接 4 schedule hooks + cron 预设（V1-V8 全过 + 0 新依赖 + lint 净改进 -4） | feat | frontend | P2 | done | Claude | M | — | R-102 / `prd/T-0070-frontend-backup-schedule-redesign.md` | sprint-06 | 2026-04-29 |
| T-0071 | 后端 BackupPolicy 持久化 MVP（19 字段 × 7 类 schema + GET/PUT singleton + 前端真实接入；enforcement 拆 T-0073/0074/0075） | feat | F06/backup+frontend | P2 | done | Claude | L | — | R-102 / `prd/T-0071-backup-policy-persistence.md` | sprint-06 | 2026-04-29 |
| T-0073 | backup cleanup Phase 1 — DB-only cleanup cron + 失败 alarm.raised 发布（file delete + 磁盘阈值留 T-0076）| feat | F06/backup | P2 | done | Claude | L | T-0071 ✅, T-0007 ✅ | R-102 / `prd/T-0073-backup-cleanup-and-alarm.md` | sprint-06..07 | 2026-04-29 |
| T-0076 | backup cleanup Phase 2 — 物理文件删除（best-effort RemoveObject）+ cleanup dashboard panels（**收紧 scope**：磁盘阈值拆 T-0082 / 多设备 orphan reaper 拆 T-0083 / email routing 已就绪 N/A）| feat | F06/backup+ops | P2 | done | Claude | M | T-0073 ✅, T-0079 ✅ | R-102 / `prd/T-0076-backup-cleanup-phase2-physical-delete.md` | sprint-07 | 2026-04-29 |
| T-0082 | backup 磁盘阈值告警 — poll MinIO bucket size + alarm.raised/cleared 边沿触发 + 5 metric（无 schema 变更，复用 MaxStorageGB×AlertThresholdPercent；severity 联合 T-0084）| feat | F06/backup+ops | P2 | done | Claude | M | T-0076 ✅ | R-102 / `prd/T-0082-backup-disk-threshold.md` | sprint-08 | 2026-04-29 |
| T-0083 | backup 多设备 orphan 文件 reaper — list-prefix scan + taskID8 cross-check + (RetentionDays+1) 天 age 安全网 + max 1000/run + @weekly cron + 2 metric（mirror T-0076/0082 PolicyMonitor 模式）| feat | F06/backup | P3 | done | Claude | M | T-0076 ✅, T-0079 ✅, T-0082 ✅ | R-102 / `prd/T-0083-backup-orphan-reaper.md` | sprint-08 | 2026-04-30 |
| T-0084 | backup AlertSeverity policy-driven 化 — schema migration 000050 + 关闭 policy_alarm_publisher 与 policy_storage_alarm 两处 severity 硬编码 + 单 column 服务两类告警（warning/major/critical 3 档；runtime fallback 防 legacy literal）| feat | F06/backup+alarm | P3 | done | Claude | S | T-0082 ✅ | R-102 / `prd/T-0084-backup-alert-severity-policy-driven.md` | sprint-08 | 2026-04-29 |
| T-0074 | backup 压缩集成 MVP（acs/upload/handler 流式压缩 + gzip/zstd 实施 + lz4/bzip2 stub + executor file_type 2→3 fix）| feat | F06/backup | P3 | done | Claude | M | T-0071 ✅ | R-102 / `prd/T-0074-backup-executor-compression.md` | sprint-07 | 2026-04-29 |
| T-0077 | backup 压缩 lz4 + bzip2 算法实施（重定 scope：FE dropdown disable cancels by 实施 / zstd encoder pool defer）| feat | F06/backup | P3 | done | Claude | S | T-0074 ✅ | R-102 / `prd/T-0077-backup-compression-lz4-bzip2.md` | sprint-07 | 2026-04-29 |
| T-0075 | backup 加密执行 — AES-256-GCM 信封加密（KEK env-var + 每文件 DEK + AAD=filename+cmp.ext + buffer-then-encrypt 64MB 上限；CBC/ChaCha20 stub；KMS 拆 T-0086；**安全敏感**）| feat | F06/backup+security | P1 | done | Claude | L | T-0071 ✅, T-0074 ✅, T-0072 ✅ | R-102 / `prd/T-0075-backup-encryption-aes256gcm.md` | sprint-07 | 2026-04-29 |
| T-0085 | backup 加密 AES-256-CBC + ChaCha20-Poly1305 算法实施 — CBC+HMAC-SHA256 (encrypt-then-MAC) 防 padding-oracle + ChaCha20-Poly1305 AEAD + OENC v1 algoByte 'C'/'P' + KEK 共用 + HKDF MAC key（mirror T-0077 模式；T-0075 followup）| feat | F06/backup+security | P3 | done | Claude | M | T-0075 ✅ | R-102 / `prd/T-0085-backup-encryption-cbc-chacha20.md` | sprint-08 | 2026-04-30 |
| T-0086 | backup KMSKeyProvider skeleton + mock（**收紧 scope**：真 AWS/Vault/HSM 适配器 carve T-0091；KEKWrapper + envelope kek_id 留 T-0087；T-0075 KeyProvider 接口预留点 closed）| feat | F06/backup+security | P2 | done | Claude | M | T-0075 ✅ | R-102 / `prd/T-0086-backup-kms-key-provider-skeleton.md` | sprint-08 | 2026-04-30 |
| T-0087 | backup KEK 旋转 core — envelope OENC v2 加 kek_id + KeyProvider 多版本接口 + EnvKeyProvider 多版本 env（**收紧 scope**：CLI 批量 re-encrypt 工具 carve T-0092；KEKWrapper 留 T-0091 真 KMS 集成）| feat | F06/backup+security | P2 | done | Claude | L | T-0075 ✅, T-0086 ✅ | R-102 / `prd/T-0087-backup-kek-rotation-core.md` | sprint-08 | 2026-04-30 |
| T-0092 | backup KEK rotation re-encrypt CLI 工具 — 独立二进制 `cmd/backup-reencrypt` + Reencryptor 服务（walk bucket + 解 envelope + decrypt + re-encrypt + atomic overwrite + 并发 [1,32] + dry-run + metrics）| feat | F06/backup+security+ops | P2 | done | Claude | M | T-0087 ✅ | R-102 / `prd/T-0092-backup-reencrypt-cli.md` | sprint-08 | 2026-04-30 |
| T-0032 | backup FTP 连接测试端点实现 — POST /backup/ftp-configs/:id/test 替换硬编码 stub；两层探测（TCP reachability for all protocols + FTP USER/PASS auth probe via stdlib net/textproto for protocol="ftp"）+ FTPTestResult 9 字段 JSON envelope + 5s timeout + 0 新依赖；SFTP/FTPS 真 auth probe carve T-0093 | feat | F06/backup | P2 | done | Claude | S | — | R-204 / `prd/T-0032-backup-ftp-connection-test.md` | sprint-08 | 2026-04-30 |
| T-0029 | RF 控制走 Carrier 适配器（去 LTE 硬编码）— Carrier 接口 +RFControlPath(tech) + 3 carrier 适配实施 + DeviceService.SetRFSwitch 重构走适配器查询 | ref | device | P2 | done | Claude | M | — | R-201 / commit `2be78a6a` | sprint-08 | 2026-04-30 |
| T-0031 | CAPTCHA 图形生成实现 — 替换 math text "12+34=?"（OCR/LLM 易破解）为 PNG 5-char alphanumeric 噪图（5×7 bitmap font 内嵌 / 31-char keyspace 排除 confusables / crypto/rand 全程 / 0 新依赖）；schema Question→Image data URL | feat | admin | P2 | done | Claude | S | — | R-203 / commit `ff983102` | sprint-08 | 2026-04-30 |
| T-0028 | syslog 远程转发 UDP/TCP — RFC 3164 wire format + UDP fire-and-forget + TCP 持久连接 + lazy reconnect on send failure + Forwarder.Send/Close API + 17 severity mapping + newline sanitize + 0 新依赖；consumer integration 待 future（service.go 7 测试 contract 不动）| feat | F06/syslog | P1 | done | Claude | M | — | R-105 / commit `db0da5d1` | sprint-08 | 2026-04-30 |
| T-0088 | 前端 BackupPolicy encryption Tag 条件化 + alert_severity Select 3 选 1（AES-256-GCM 时 green "已生效"；其它 warning；FE-only encryption_ready 派生避免 backend API 扩展）| feat | frontend | P3 | done | Claude | S | T-0075 ✅, T-0084 ✅ | R-102 / `prd/T-0088-frontend-backup-encryption-tag-severity-select.md` | sprint-08 | 2026-04-29 |
| T-0089 | backup decrypt 并发 semaphore 防 DoS — buffered channel + timeout-then-reject + 503 Retry-After + 3 metric + env var OMC_BACKUP_DECRYPT_CONCURRENCY 默认 8（闭环 T-0075 review M-4 资源耗尽漏洞）| feat | F06/backup+ops | P3 | done | Claude | S | T-0075 ✅ | T-0075 review M-4 / `prd/T-0089-backup-decrypt-semaphore.md` | sprint-08 | 2026-04-29 |
| T-0072 | 备份恢复后端 MVP（TR-069 Download(FileType=3) 路径 + download handler 流式解压 + restore_tasks 表 + POST /backup/restore；FE 拆 T-0078 / task↔path 链路拆 T-0079）| feat | F06/backup | P1 | done | Claude | M | T-0071 ✅, T-0074 ✅, T-0077 ✅ | R-102 / `prd/T-0072-backup-restore-backend-mvp.md` | sprint-07 | 2026-04-29 |
| T-0078 | 前端 RestoreData wholesale rewrite + 3 hook + 26 i18n key（**MVP 手动路径输入**；filemanager 路径架构不匹配 → 文件浏览器拆 T-0081 / T-0079 后做）| feat | frontend | P1 | done | Claude | L | T-0072 ✅ | R-102 / `prd/T-0078-frontend-restoredata-rewrite.md` | sprint-07 | 2026-04-29 |
| T-0079 | backup_task → file_path 链路回填（EventBus pub-sub + filename embed taskID8 + DB-layer CAS first-write-wins + restore_by_task_id 新 endpoint）| feat | F06/backup | P2 | done | Claude | M | T-0072 ✅, T-0007 ✅ | R-102 / `prd/T-0079-backup-task-filepath-linkage.md` | sprint-07 | 2026-04-29 |
| T-0080 | migration 000038 重复 hotfix（rename `000038_upgrade_tasks_firmware_id_nullable.sql` → `000049_*.sql` 让 goose 可解析；pre-existing 历史遗留，T-0072 review-agent 发现）| fix | infra/migration | P0 | done | Claude | S | — | CLAUDE.md §5.5 / T-0072 review finding | sprint-07 | 2026-04-29 |
| T-0027 | 拓扑自动分组规则引擎激活（三路径全闭环：手工 ApplyRule + cron @hourly + device.registered EventBus；A4 SQL 守护；6 metric + 7 log key + FE 来源列；R-104 关闭）| feat | F06/topology | P1 | done | Claude | M | — | R-104 关闭 / `prd/F06-topology-auto-grouping.md` / `docs/review-report/20260506/verify-T-0027.md` | sprint-09 | 2026-05-06 |

**说明**：
- T-0009 是外部凭据申请，不编码但走流水线（作为前置项，保证 T-0014 不被卡）。
- T-0010 是 XL 任务，按模块切分实际执行（F04 通知 → transfer → F08），分阶段推进但登记为一条。
- T-0006 E2E 是贯穿型任务，不拆成 7 条，每 Sprint 回顾时更新累计数；Sprint 不达标时在 Sprint 回顾里登记。

### 3.1 累计型任务进度（S4 核销读此处）

> **规则**：下游 Task.Deps 形如 `T-xxxx@累计≥N` 时，S4 verify 读本表的 `Progress` 判定达标。
> 上游每 Sprint 回顾 / 合入新 E2E 用例时更新此表。详见设计 §11.7.1。

| Task | Progress | 目标 | 下游依赖 | 下次更新 |
|------|----------|------|----------|---------|
| T-0006（E2E 累计用例） | **累计 549** ✅（T-0057 修真 bug 后实跑 PASS / claim 129；W1.6 27 + W2D 99 + W2.A.4 4 + Sprint 0-9 framework 段恢复） | 累计 ≥200 ✅ | T-0025 要求累计 ≥150 ✅ **解锁** | Wave 4 GA 前 RC 冻结再核（2026-08-03 前） |

---

## 4. Triaged — 已分诊、未排期（等 Sprint Planning 挑选）

| ID | Title | Type | Domain | Prio | Est | Deps | Risk | Notes |
|----|-------|------|--------|------|-----|------|------|-------|
| T-0030 | F10 互操作用例库扩充 | feat | F10 | P2 | L | — | R-202 | GA 级质量补强 |
| T-0093 | backup SFTP / FTPS 真 auth probe 实施（pkg/sftp + crypto/ssh + crypto/tls）— T-0032 carve-out；不依赖运维凭据可独立开发 | feat | F06/backup | P2 | M | T-0032 ✅ | R-204 / T-0032 carve-out | trigger：实际部署有 SFTP/FTPS 配置需求时启动 |
| T-0035 | 前端多皮肤架构（Phase 3-7：v2 皮肤脚手架 → 18 模块补齐 → 双皮肤部署） | feat | frontend | P2 | XL | T-0035-P1 | — | 方案 `frontend-multi-skin-plan-20260422.md`；Phase 1（`@omc/frontend-core` 抽取 + workspaces）已完成；Phase 3+ 需 Sprint 规划 |
| T-0091 | backup 真 KMS 适配器实施（AWS KMS / Vault Transit / HSM 二选一）— KMSClient interface 已 ready (T-0086)，本任务做 SDK 集成 + DI wiring + observability metrics | feat | F06/backup+security | P2 | M-L | T-0086 ✅ | R-102 / T-0086 carve-out | trigger：运维侧选定厂商 + 凭据准备；KEKWrapper refactor 与 T-0087 KEK 旋转一并做；新增 metric `omc_backup_kms_call_total{op,result}` + `omc_backup_kms_call_duration_seconds{op}` |
| T-0090 | MML 控制台公/私命令新增页面 UX 整改 — ① 操作类型差异化（MOD→修改值入口）② 删 3 字段（参数配置/所属分类/适用产品类型）③ 命令编码 input→textarea ④ textarea 必须自定义、不允许选已有命令 ⑤ 私有命令页面参考公有页面（功能一致）⑥ 私有命令按当前管理员所属分组过滤（RBAC）⑦ 公有命令不按管理员过滤 | feat | frontend+F06/mml+admin | P2 | XL | — (T-0094 已 rejected as misdiagnosis 2026-04-30) | R-NEW（productTypes 删除破坏性 + RBAC 私有命令分组隔离 双重风险） | @chenbo01 提需求；**2026-04-30 B 方案二次扩展**：原 ①②③ 保留 + 新加 ④（textarea 自定义限制）+ ⑤⑥⑦（私有/公有命令 RBAC 差异化）；用户重叠检测拍板 B（不覆盖原版而是扩展）；**Est L → XL，强制 S2 拆**；ULTRATHINK 三层核查（Go struct + migrations + FE 表单）+ B 扩展后子项风险阶梯横跨 4 档：①③④⑤ 纯 FE 低风险（操作类型差异化 + textarea + 自定义限制 + 私有页面复用公有 UI）；② 中耦合（参数配置 UI section 删后端接空 map 兼容；categoryGroup 清理由 T-0094 决策）；②④ productTypes 删 = 破坏性 migration（drop column + 历史数据评估 + Down 段回滚预案）；⑥ RBAC = backend 改动（mml_custom_command 查询按当前 admin 的 role_device_groups 过滤 private 命令；query 重写 + service 层 admin context 注入）；S2 强建议**拆 4 段**：T-0090a（S, FE-only ① ③ ④ + ② UI section 删除）+ T-0090b（M, ref + migration drop productTypes）+ T-0090c（M-L, backend RBAC private 查询过滤 + admin context 注入）+ T-0090d（S, FE 私有命令页面创建 复用公有组件）；Domain 加 admin（⑥ 涉 RBAC）；候选 Sprint=GA 规划期 |
| T-0096 | MML script 页面「更新」弹窗取消产品类型字段（应与 /mml/console「保存脚本」弹窗一致） | ref | frontend+F06/mml | P3 | S | T-0090 | R-NEW（共享 T-0090 productTypes 删除风险） | Triage 2026-04-30 走 B 方案 standalone（不 fold 进 T-0090a — 因 page scope 不同 script vs console + T-0090 体量已 L 不宜再扩 + 独立 PR 更清晰）；语义**反向** deps T-0090："与 console 一致"意味着 console 决定 productTypes 去留 → script 跟随（若 T-0090 决定保留 productTypes，T-0096 也保留；命名"取消"但执行随 T-0090）；建议与 T-0090a 同 Sprint 紧随做省 cross-PR 协调 |
| T-0097 | MML console「保存脚本」弹窗确认按钮 API 结果反馈（成功/失败提示） | bug | frontend+F06/mml | P2 | S | — | R-NEW（toast util 场景失效面） | Triage 2026-04-30 走 C 方案 triaged + **pre-pick reproduce 待办**：静态核查 `ScriptTaskDrawer.tsx:232` toast.success + `:242` toast.error 已在 → bug 真实性存疑；pick 前需 user 提供 reproduce 步骤；4 候选成因优先级（高→低）：① mock 模式（useMock=true 时 mockService 不抛错 → toast.error 永不触发，最可能解释"看不到任何提示"）② axios 拦截器吞 401/403 错误码 ③ 用户混淆"保存脚本"指其他按钮 ④ toast util 场景失效（line 190 注释提示 to-do-list #2 曾遇）；建议 user 实测：先在 dev real-API 模式（VITE_USE_MOCK=false）尝试触发失败 + 看浏览器 console 是否报错 |
| T-0098 | 参数模型 / KPI 指标库 / 告警库 数据字典平台化（XML 源文件 → 启动期 loader → DB schema → 双层缓存 → REST API → 产品管理 UI；本次提交仅交付 S2 整合设计稿 + XML 数据资产入库；**2026-05-07 v3 设计稿重组 + 三大业务流程图 + v1 实施计划落地** — 33 子任务 / 5 Phase / 10 决策待 PgM 拍板，**2026-05-07 D1-D10 全采纳推荐 + 36 子任务（T-0098-P1-01 .. T-0098-P5-06）拆出登记 §4.1 + 12 R-T0098-* 风险同步 risk-register + P1 6 条升格 planned/wave-3/Claude（前戏完成可开车）**，详见 `docs/project/参数-KPI-告警-整合-实施计划.md` + §10 changelog；**新方案**取代 `omcgo/docs/param-model-delivery/参数模型重新设计方案.md` + `omcgo/docs/prd/product/产品模型*.md` 旧设计） | feat | F02+F03+F04 | P2 | in_dev | Claude | XL | — | `docs/design/参数-KPI-告警-整合设计方案.md` + `docs/project/参数-KPI-告警-整合-实施计划.md` + `AI承诺对峙清单.md` W3 | wave-3 | 2026-05-07 |

### 4.1 T-0098 拆分子任务

> 36 条子任务（T-0098-P1-01 .. T-0098-P5-06，2026-05-07 D1-D10 全采纳推荐后批量登记）已全量拆出。
> 完整 sub-task 表（schema 与 §3 Active 主表对齐 13 列） → [`backlog/subtasks/T-0098-data-dict.md`](backlog/subtasks/T-0098-data-dict.md)
> sprint planning 时从该文件挑选；本主表保留 umbrella 行 T-0098（已升 wave-3/Claude）。
>
> **2026-05-07 前戏完成升格**：P1-01..P1-06 6 条已 **State→planned / Sprint=wave-3 / Owner=Claude**；wave-batched 模式（dev-pipeline §C.1）准入开发，Skip S0/S1；可直接 `/dev-pipeline pick T-0098-P1-01` 开车。**commit 形态**：6 commit + 单 PR（每 commit footer 各挂 `Backlog: T-0098-P1-0N`）；P2-01..P5-06 30 条仍 triaged 等下次 sprint planning。

---

## 5. Proposed — 待 Triage

| ID | Title | Proposed By | Created | Notes |
|----|-------|-------------|---------|-------|
| — | （当前为空，新想法请在此行之上追加） | — | — | 下周一 Triage 会议判决 |

---

## 6. Done — 近 7 天速览

> 本季度（2026Q2）完整 Closing Evidence 归档 → [`backlog/done/2026Q2.md`](backlog/done/2026Q2.md)
> 维护规则：S7 关闭新任务时同时追加到 `backlog/done/<当季>.md`，本表仅滚动近 5-10 条。

| ID | Title | Domain | Closed | 一句话摘要 |
|----|-------|--------|--------|----------|
| **T-0098-P2-02** | ParamRegistry — Registry + Translator + L1+L2 缓存 + discovered→default 降级 | F02 | 2026-05-07 | **参数轨脊柱 / 一次解锁 6 条下游**；6 实现 + 4 测试 ~1860 LOC；registry 60-100%/translator 91-100%/metrics 100% 单测；6 metric + 5 log 全 grep；productGetter 接口注入解耦；param_registry.use_new dual-stack flag 默认 false |
| **T-0098-P2-01** | ProductRegistry — productClass 路由 + L1+L2 缓存 + 三引用校验 | F02 | 2026-05-07 | **P2 wave 入口 1**；6 实现 + 3 测试 ~1280 LOC；registry 87-95%/cache 100%/metrics 100% 单测；4 metric 全 grep；ModuleGraph "productregistry" Depends ["dictload"] |
| **T-0098-P1-06** | 4 域字典 Loader + provider 接线 + 修复 entry_type 列宽 | infra+F02+F03+F04 | 2026-05-07 | **P1 wave 收官**；4 包 12 文件 ~1500 LOC；DoD 12 项行数全过；000060 forward fix entry_type VARCHAR(8)→16 |
| T-0098-P1-05 | KPI 表名对齐文档（D1=A docs only） | F03 | 2026-05-07 | 设计 §2.3 D1=A 命名对齐表 17 行；0 schema 变更；不再写 000060_kpi_rename |
| T-0098-P1-04 | 迁移 000059 alarm 字典（severity_levels + 4 种子 + alarm_definitions + alarms_active.is_unknown） | F04 | 2026-05-07 | 2 新表 + 4 索引；severity 4 行种子（31001-31004）；旧 alarm_libraries 留 P5-06 DROP（D2=B）|
| T-0098-P1-03 | 迁移 000058 param 字典（4 表 + 跨域 FK 收口） | F02 | 2026-05-07 | 4 新表 + 8 索引；DO 块 ADD CONSTRAINT 补齐 products.param_model_id FK；首跑触发 §5.5.1 教训 |
| T-0098-P1-02 | 迁移 000057 products + product_class_patterns + devices.product_id | F02 | 2026-05-07 | 2 新表 + 5 索引（含 sort_order 部分唯一）；devices 分区表无 FK；param_model_id 留位 |
| T-0098-P1-01 | 共享字典装载基础设施（dictloader + DictLoaderConfig） | infra | 2026-05-07 | 4 实现 + 4 测试 + appconfig 接入；coverage 98.4%；wave-3 P1 首发；Loader 接口 ready for P1-06 |
| T-0027 | 拓扑自动分组规则引擎激活 | F06/topology | 2026-05-06 | 三路径全闭环（手工 + cron + EventBus）+ R-104 关闭 |
| T-0083 | backup 多设备 orphan reaper | F06/backup | 2026-04-30 | list-prefix scan + 1000/run cap + @weekly |

（其余历史 done 见归档；T-0094 misdiagnosis 见 §8 Rejected）

---

## 7. Deferred — 判决延后到 GA 阶段

> **判决依据**：`milestone/2026Q2-to-RC.md §2 明确不做` + `§2 未达成项（GA 阶段再解决）`

| ID | Title | Domain | Prio | Reason | Review At |
|----|-------|--------|------|--------|-----------|
| T-0033 | 多级聚合与趋势分析（PM 站点/区域/网络级聚合 + 根因分析） | F03 / F05 | P2 | milestone §2.未达成项 列为 GA；与 R-205 关联 | GA 规划期 |
| T-0034 | F10 运营商标准测试套件对齐 | F10 | P2 | milestone §2.Out of Scope；RC 窗口不做 | GA 规划期 |

---

## 8. Rejected — 判决本 milestone 不做

> **判决依据**：`milestone/2026Q2-to-RC.md §2 明确不做`

| ID | Title | Reason |
|----|-------|--------|
| T-0035 | F08 CORBA / MTOSI / TMF 多协议支持 | 本 RC 窗口仅做 SNMP Trap；CORBA/MTOSI/TMF 按商用运营商实际要求再启动 |
| T-0036 | 分布式事务 / 跨机房容错 / 自动 failover | 超出当前模块化单体架构假设；待 100 万基站扩展期重新评估 |
| T-0037 | GA 级安全合规（密钥轮换 / DLP / 第三方审计） | 超出 RC 目标；由后续 GA 专项处理 |
| T-0094 | MML category_group DB 列缺失（误诊） | **misdiagnosis**：backlog 描述称"migrations/000014 + 000026 从未创建此列"事实错误。逐文件核查：`000020_mml_category_group_and_executor.sql` Up L3 早就 `ALTER TABLE mml_templates ADD COLUMN IF NOT EXISTS category_group VARCHAR(50)` + L4 `CREATE INDEX idx_mml_templates_scope_group`；`000026` Up L28 RENAME 表（PG 不删列、不改 category_group 名）。**真库验证 2026-04-30**（docker compose exec postgres `\d mml_custom_command`）：`category_group character varying(50) 可空 + 索引 idx_mml_templates_scope_group` 全部就位 ✅。详见 `docs/review-report/20260430/verify-T-0094.md` §5 final result。**联动解锁**：T-0090 Notes 子项 ②"categoryGroup 清理由 T-0094 决策"答案锁定 = 列已存在功能正常无需清理；T-0090 deps T-0094 ✅ → 改为 — |

---

## 9. 依赖关系图（关键路径）

```
T-0010 (NATS)──┬──▶ T-0011 (Webhook) ──┐
               ├──▶ T-0012 (worker重试) │
               └───────────────────────┤
                                       │
T-0009 (短信凭据) ────▶ T-0014 (短信渠道)│
                                       │── 并行推进 ──▶
T-0013 (SNMP骨架) ────▶ T-0017 (联调) ──▶ T-0020 (推送可靠性)
                                       │
T-0018 (灰度) ────────▶ T-0021 (回滚)   │
           └──────────▶ T-0019 (前端Software)

                                       ↓ 全部 done 后
                                 T-0023 (压测基线)
                                       ↓
                                 T-0024 (Gate 演练)
                                       ↓
                                 T-0025 (RC 冻结) + T-0026 (Runbook)
```

**关键路径**：T-0010 → T-0011 / T-0012 / T-0013 → T-0017 → T-0020 → T-0023 → T-0024 → T-0025。
**瓶颈**：T-0010（NATS，XL 任务，涉三模块）— 任何延期会连带影响 Sprint-02 ~ 05 多个下游，PgM 每周复盘必看。

---

## 10. 变更日志（近 5 条速览）

> 完整变更历史 → [`backlog/changelog.md`](backlog/changelog.md)
> 维护规则：每次状态/字段变更追加到 changelog.md；本表仅保留近 5 条速览，由 PgM 周一 Triage 时滚动更新。

| 日期 | 动作 | 条目 | 一句话 |
|------|------|------|--------|
| 2026-05-07 | **P2-02 ParamRegistry done** S2..S7 wave-batched 全过 | T-0098-P2-02 | 6 实现 + 4 测试 ~1860 LOC（含 ~655 LOC 测试）；registry 60-100% / translator 91-100% / metrics 100% 单测覆盖；6 Prometheus metric + 5 类 log 全 grep 命中；Provider ModuleGraph "paramregistry" Depends ["dictload","productregistry"]；param_registry.use_new dual-stack flag 默认 false；productGetter 接口注入解耦；**一次落地解锁 P2-03..P2-08 6 条下游**；详见 verify-T-0098-P2-02.md；planned 16→15 / done 85→86 |
| 2026-05-07 | **P2-01 ProductRegistry done** S2..S7 wave-batched 全过 | T-0098-P2-01 | 6 实现 + 3 测试 ~1280 LOC（含 ~515 LOC 测试）；registry 87-95% / cache 100% / metrics 100% 单测覆盖；4 Prometheus metric + 3 类 log 全 grep 命中；Provider ModuleGraph "productregistry" Depends ["dictload"]；详见 verify-T-0098-P2-01.md；planned 17→16 / done 84→85 |
| 2026-05-07 | **P2 前戏完成升格 + 双入口准入** | T-0098-P2-01..P2-11 11 sub-task | State→planned / Sprint=wave-3 / Owner=Claude；P2-01 (F02 ProductRegistry) + P2-09 (F03 KPI loader) 两个独立入口可并发；其余 9 条因路径集中 `internal/provision/` 串行；wave-batched §C.1 准入 Skip S0/S1；triaged 37→26 / planned 6→17 |
| 2026-05-07 | **P1 wave 收官** S2..S7 wave-batched 全过 | T-0098-P1-02..P1-06 5 commit | P1-02 products schema → P1-03 param 字典 + 跨域 FK → P1-04 alarm 字典 + 4 种子 → P1-05 KPI 命名 D1=A docs → P1-06 4 Loader + provider + 修复 entry_type 列宽；**DoD 12 项行数全过**（9+4781+2001 / 15+29 / 4+442 / 1764 / 6254）；6 sub-task 全部 done；wave-3 P1 阶段封箱 |
| 2026-05-07 | S2..S7 wave-batched 全过 | T-0098-P1-01 dictloader 框架 | 4 实现 + 4 测试 + appconfig 接入；coverage 98.4%；planned → done；wave-3 P1 首发 |

---

## 11. 附录：与既有制品的映射

| 既有条目 | Backlog 对应 | 关系 |
|---------|------------|------|
| `milestone/2026Q2-to-RC.md §2.P0#1` | T-0007, T-0011, T-0014 | 一个 milestone 条目拆出多个 Task（按通道） |
| `milestone §2.P0#2` | T-0006 | 一对一（贯穿型） |
| `milestone §2.P0#3` | T-0010 | 一对一 |
| `milestone §2.P0#4` | T-0002（已关闭） | 一对一 |
| `milestone §2.P0#5` | T-0013, T-0017, T-0020 | 一个 milestone 条目拆出多个 Task（按阶段） |
| `risk-register.md` R-001 | T-0007, T-0009, T-0011, T-0014 | 一个 Risk 拆出多个 mitigation Task |
| `risk-register.md` R-005 | T-0002（done） | 一对一 |
| `risk-register.md` R-108 | T-0003（done） | 一对一 |
| `prd/F04-alarm-notification.md` | T-0007, T-0011, T-0014 | 一个 PRD 拆出多个 Task |
| `prd/infra-event-bus.md`（待产出） | T-0010 | S0 立项阶段产出 PRD |
| `prd/F08-oss-protocol.md`（待产出） | T-0013, T-0017, T-0020 | S0 立项阶段产出 PRD |

**原则**：本表是**唯一任务清单**；milestone / sprint / risk-register / PRD 各自的"待办列表"将**逐步迁移为引用**本表条目，不再手工维护重复。

---

## 12. Bootstrap 遗留

初版 Bootstrap（2026-04-20）的待办 + 维护节奏 → [`backlog/bootstrap.md`](backlog/bootstrap.md)
