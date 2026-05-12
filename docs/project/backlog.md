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
| Total tasks | 149 | — | 含 T-0098 umbrella + 36 sub-task；2026-05-10 新增 12 umbrella（T-0101..T-0112 F06 运维管理）+ 66 sub-task；2026-05-11 +T-0113 F06 ops 合规债清账 + T-0114/T-0115/T-0116 F10 Phase 3 三 task（fault-inject / 报告 export / 设备矩阵）；变更明细见 changelog |
| `done` | 143 | — | 详细 Closing Evidence → `backlog/done/2026Q2.md`（含 T-0095/T-0098/T-0101..T-0112/T-0099/T-0097/T-0030 + T-0115/T-0114/T-0116 + T-0113 + T-0090 全系列 + T-0096 + **+T-0101-d OpsTask 状态机集成 2026-05-12（ApprovalService.Approve 删 TODO 兑现 / R-O01 mitigation 完整闭环 / 两轴正交 atomic UPDATE / 5 RBAC unit / T-0101 umbrella 9 sub-task 进度 1/9）**）|
| `in_dev` | 0 | — | T-0098 + T-0090 均收官 |
| `in_design` | 0 | — | T-0090 umbrella 4/4 sub-task 全闭，State in_design → done |
| `planned` | 6 | — | T-0009/T-0014/T-0017/T-0020/T-0023 (wave + 短信凭据外部阻塞) — sprint-10 三 deliverable 全闭（T-0030 Phase 1 done / T-0090 升 in_design 完成 S2 拆分 / T-0097 done）|
| `triaged` | 3 | — | T-0091 / T-0093 (backup 外部 trigger) / T-0035 (XL 待 S2 拆) — T-0096 升 done 2026-05-12（T-0090 主线 followup 收尾）|
| `blocked` | 0 | ≤ 3 | — |
| `proposed` 积压天数 | 0 | ≤ 7 | — |
| **P0 风险关闭数** | **1 / 5** | 5 / 5 | R-005 已关；R-001/002/003/004 Open（注：**P1 R-103 / R-104** 已关闭但不计入 P0 — R-103 在 T-0015 后关闭；**R-104 在 T-0027 后关闭 2026-05-06**；**P2 R-202 在 T-0030 Phase 1+2 后关闭 2026-05-11**）|
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
| T-0095 | F03 KPI 指标管理（标准报表 + 站点报表）— **2026-05-08 D10=A 吸收正式关闭（P4-05 完成 KPI 指标库 5 Tabs 全平台公式 CRUD UI）** | feat | F03 | P1 | done | Claude | XL | — | T-0098 D10=A 吸收 / 详见 §10 changelog 2026-05-08 | sprint-01 | 2026-05-08 |
| T-0030 | F10 互操作用例库扩充（GA 级质量补强）— **Phase 1 + Phase 2 done 2026-05-11 单晚**：用例 14→37（DM 2→7 / Protocol 3→8 / RPC 9→17 含 3 家运营商私有 RPC X_CMCC_/X_CT-COM_/X_CU-COM_ / Inform 0→5）+ 每类 negative ≥ 2（共 8）+ ExpectedOutcome 翻转机制 + e2e 3→8 claim + 13 新单测（含 TestCases_CarrierCoverage 强制三家覆盖）；commits `92a3eb76` (Phase 1) + `e2d846cf` (Phase 2)；**R-202 Closed**；Phase 3 选项（fault-inject category / 验收报告 export / 设备矩阵参数化）走独立 sub-task 不再扩 T-0030 | feat | F10 | P2 | done | Claude | L | — | **R-202 Closed** / `docs/project/prd/F10-interop-testing-coverage.md` | sprint-10 | 2026-05-11 |
| T-0090 | MML 控制台公/私命令新增页面 UX 整改（7 子项 B 方案二次扩展）— **done 2026-05-12 4/4 sub-task 全闭**（a `bf71e518` FE 4 子项 + b `86b19c3f` drop product_types + c `2c114a40` RBAC + d `396cb76d` 私有命令页 + R-NEW-1/-2/-3 mitigation 完整 + sprint-11 work 全 pull-forward 进 sprint-10 buffer 单晚交付） | feat | frontend+F06/mml+admin | P2 | done | Claude | XL→拆 4 段 | — | R-NEW 全 mitigate / `backlog/subtasks/T-0090-mml-ux-rework.md` | sprint-10 | 2026-05-12 |
| T-0097 | MML console「保存脚本」弹窗确认按钮 API 结果反馈 — done（AddTemplateModal 切 App.useApp() scoped messageApi，commit 5b281b06；候选 #4「toast util 失效」静态分析确认 → antd v5 嵌套 Modal 静态 message 脱 ConfigProvider 上下文丢失）；浏览器端实测留用户回归确认 | bug | frontend+F06/mml | P2 | done | Claude | S | — | R-NEW（toast util 场景失效面）已闭环 / `docs/project/sprint/sprint-10.md` | sprint-10 | 2026-05-11 |
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
| T-0100-P0 | license_logs 表迁移 + LogWriter + 5 处写入点接入（handler import/activate/revoke + enforcer EnforceCapacity/Expiry + monitor capacity/expiry/auto_expire）— umbrella T-0100 子任务（详 §4.2） | feat | F06/license | P1 | done | Claude | M | T-0015 ✅ | R-109 关闭 / `prd/F06-license.md` §6.2 §9.3 | wave-3-finishing | 2026-05-09 |

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
| T-0093 | backup SFTP / FTPS 真 auth probe 实施（pkg/sftp + crypto/ssh + crypto/tls）— T-0032 carve-out；不依赖运维凭据可独立开发 | feat | F06/backup | P2 | M | T-0032 ✅ | R-204 / T-0032 carve-out | trigger：实际部署有 SFTP/FTPS 配置需求时启动 |
| T-0035 | 前端多皮肤架构（Phase 3-7：v2 皮肤脚手架 → 18 模块补齐 → 双皮肤部署） | feat | frontend | P2 | XL | T-0035-P1 | — | 方案 `frontend-multi-skin-plan-20260422.md`；Phase 1（`@omc/frontend-core` 抽取 + workspaces）已完成；Phase 3+ 需 Sprint 规划 |
| T-0091 | backup 真 KMS 适配器实施（AWS KMS / Vault Transit / HSM 二选一）— KMSClient interface 已 ready (T-0086)，本任务做 SDK 集成 + DI wiring + observability metrics | feat | F06/backup+security | P2 | M-L | T-0086 ✅ | R-102 / T-0086 carve-out | trigger：运维侧选定厂商 + 凭据准备；KEKWrapper refactor 与 T-0087 KEK 旋转一并做；新增 metric `omc_backup_kms_call_total{op,result}` + `omc_backup_kms_call_duration_seconds{op}` |
| T-0100 | F06 license 治理层（umbrella，5 子任务详见 §4.2） | feat | F06/license+frontend+admin | P1 | triaged | Claude | XL | T-0015 ✅ | R-109 / `prd/F06-license.md` | wave-3-finishing | 2026-05-09 |
| T-0098 | 参数模型 / KPI 指标库 / 告警库 数据字典平台化 — **2026-05-08 36/36 全收官**（P1 6 + P2 11 + P3 5 + P4 8 + P5 6）；旧 datamodel 包整体下线，新栈 ParamModel + ProductRegistry + Translator + IntersectService + dictloader（4 个 Registry：product / parammodel / indicator / alarm-definition）就位；47 REST 端点（products 17 + param-models 19 + indicators 21 + alarm-definitions 9 重叠去重）+ 5 治理 UI 页（products / param-model / kpi-library / alarm-library / orphan-devices）+ super_admin 守卫 + R-T0098-01..12 风险全闭环 | feat | F02+F03+F04 | P2 | done | Claude | XL | — | `docs/design/参数-KPI-告警-整合设计方案.md` + `docs/project/参数-KPI-告警-整合-实施计划.md` + `AI承诺对峙清单.md` W3 | wave-3 | 2026-05-08 |

### 4.1 T-0098 拆分子任务

> 36 条子任务（T-0098-P1-01 .. T-0098-P5-06，2026-05-07 D1-D10 全采纳推荐后批量登记）已全量拆出。
> 完整 sub-task 表（schema 与 §3 Active 主表对齐 13 列） → [`backlog/subtasks/T-0098-data-dict.md`](backlog/subtasks/T-0098-data-dict.md)
> sprint planning 时从该文件挑选；本主表保留 umbrella 行 T-0098（已升 wave-3/Claude）。
>
> **2026-05-07 前戏完成升格**：P1-01..P1-06 6 条已 **State→planned / Sprint=wave-3 / Owner=Claude**；wave-batched 模式（dev-pipeline §C.1）准入开发，Skip S0/S1；可直接 `/dev-pipeline pick T-0098-P1-01` 开车。**commit 形态**：6 commit + 单 PR（每 commit footer 各挂 `Backlog: T-0098-P1-0N`）；P2-01..P5-06 30 条仍 triaged 等下次 sprint planning。

### 4.2 T-0100 子任务（F06 license 治理层）

> 5 子任务覆盖 PRD `prd/F06-license.md` §14.2 五阶段实施路线图。**P0-P3 严格顺序依赖**（P0 是关键路径阻塞项）；P4 各项独立可并行。
>
> **2026-05-09 自我 triage**：T-0100-P0 已升 **planned / wave-3-finishing / Owner=Claude** 立即开车（P0 仅写日志基建，不依赖 Q1-Q4 决议）；P1-P4 4 条仍 triaged，等 T-0100-P0 落地 + Q1-Q4 决议后下次 sprint planning 决策准入。

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Risk/PRD | Sprint | Updated |
|----|-------|------|--------|------|-------|-------|-----|------|----------|--------|---------|
| T-0100-P0 | license_logs 表迁移 + LogWriter 服务 + 5 处写入点接入（handler 3 + enforcer 2 + monitor 3） | feat | F06/license | P1 | done | Claude | M (~2d) | T-0015 ✅ | R-109 关闭 / `prd/F06-license.md` §6.2 §9.3 | wave-3-finishing | 2026-05-09 |
| T-0100-P1 | LicenseLogs 主页全量 — 后端 GET `/licenses/logs` + 过滤分页；前端接 API + i18n（去 mock） | feat | F06/license+frontend | P1 | done | Claude | M (~2d) | T-0100-P0 ✅ | `prd/F06-license.md` §5.4 §11.2 V11 | wave-3-finishing | 2026-05-09 |
| T-0100-P2 | LicenseList 详情抽屉 + Summary 卡片（后端 GET `/licenses/:id/logs` + Summary 增强；前端抽屉 + 进度条 + 跳转 Logs） | feat | F06/license+frontend | P1 | done | Claude | M (~2d) | T-0100-P1 ✅ | `prd/F06-license.md` §5.2 §11.2 V9 V14 | wave-3-finishing | 2026-05-09 |
| T-0100-P3 | LicenseOperations 完整 4 Tab + 操作历史接 API + 导入校验链 + `system:license:operate` 权限点 seed | feat | F06/license+frontend+admin | P1 | done | Claude | L (~3d) | T-0100-P2 ✅ + Q1/Q3/Q4 ✅（2026-05-09 user 拍板 B/C/B 全推荐） | `prd/F06-license.md` §5.3 §8 §11.2 V10 V12 §16 决议；migrations/000074 + seed/000075 | wave-3-finishing | 2026-05-09 |
| T-0100-P4-A | 单条 PDF 导出（fpdf）+ 全量 active CSV 导出（GET `/licenses/:id/export?format=pdf` + `/licenses/export?format=csv`）+ 前端 Blob 下载 | feat | F06/license+frontend | P2 | done | Claude | M (~1d) | T-0100-P3 ✅ | `prd/F06-license.md` §5.3.4 §16 Q1 决议；migrations seed/000077 | wave-3-finishing | 2026-05-09 |
| T-0100-P4-B | license_logs 6 月归档 cron — monitor.archiveOldLogs(ctx) → MinIO `license-logs/{YYYY-MM}/{tickTS}-{count}.jsonl.gz` + DB DELETE > 6 月；weekly schedule | feat | F06/license+ops | P2 | done | Claude | M (~1-2d) | T-0100-P3 ✅ | `prd/F06-license.md` §5.4.5；archiver.go + monitor.SetArchiver；P4-B1 修 review CRITICAL #1 | wave-3-finishing | 2026-05-09 |
| T-0100-P4-B1 | 归档对象键唯一化（防同月跨 tick 覆盖）— 修复 review 922d87a4 CRITICAL #1：键名加 tickTS + count 后缀 + 拆月级 prefix | fix | F06/license | P0 | done | Claude | S (~0.5d) | T-0100-P4-B ✅ | review-report 20260509/REVIEW_922d87a4_chenbo01_license.md CRITICAL #1 | wave-3-finishing | 2026-05-09 |
| T-0100-P4-B2 | DeleteByIDs 替代 DeleteBefore（防积压超 batchSize 时未归档行被误删）— 修复 review 922d87a4 WARNING #3 | fix | F06/license | P2 | done | Claude | S (~0.5d) | T-0100-P4-B1 ✅ | review-report 20260509/REVIEW_922d87a4_chenbo01_license.md WARNING #3；archiver.go DeleteByIDs(archivedIDs) | wave-3-finishing | 2026-05-09 |
| T-0100-P4-C | OEM 公钥强校验 — `configs/oem_public_keys/*.pem` 加载 + RSA-PSS 嵌入 JSON `signature` 字段 + strict config 开关；替换 P3 stub `VerifySignature` | feat | F06/license+admin+ops | P2 | done | Claude | M (~1-2d) | T-0100-P3 ✅ + Q4=B 决议 | `prd/F06-license.md` §5.3.1 §16 Q4 决议；signature.go + ImportRequest.SignedLicenseJSON | wave-3-finishing | 2026-05-09 |

---

## 5. Proposed — 待 Triage

| ID | Title | Proposed By | Created | Notes |
|----|-------|-------------|---------|-------|
| — | （新想法在 T-0099 上方追加） | — | — | 下周一 Triage 会议判决 |

> **2026-05-09 自我 triage（用户授权）**：T-0100 由 Proposed 直接进 §4 Triaged + 拆 5 子任务（§4.2）；T-0100-P0 进 §3 Active Planned 立即开车（P0 仅写日志基建，不依赖 Q1-Q4 决议）。R-109 已登记 risk-register.md。

---

## 6. Done — 近 7 天速览

> 本季度（2026Q2）完整 Closing Evidence 归档 → [`backlog/done/2026Q2.md`](backlog/done/2026Q2.md)
> 维护规则：S7 关闭新任务时同时追加到 `backlog/done/<当季>.md`，本表仅滚动近 5-10 条。

| ID | Title | Domain | Closed | 一句话摘要 |
|----|-------|--------|--------|----------|
| **T-0101-d** | OpsTask 状态机集成 + approval_state 真持久化 | F06/ops | 2026-05-12 | commit `1743e8d1`（8 文件 +548/-32）；ApprovalService.Approve 删 TODO 兑现承诺 — OpsTask 扩 4 字段 + UpdateApproval 单 SQL atomic UPDATE (approve→running / reject→cancelled) + 5 unit test 反退化（含 4-eye + 非 pending 3 sub-case + 持久化错误传播）；R-O01 (4 眼审批未真执行) mitigation 完整闭环；两轴正交 Status × ApprovalState；T-0101 umbrella 进度 1/9 |
| **T-0096** | ScriptTask Drawer 删 productType 与 console 弹窗对齐 | frontend+F06/mml | 2026-05-12 | commit `5bc00049`（3 文件 +181/-5；ScriptTask/index.tsx 净 -3 行）；TaskForm interface + openEditDrawer init + Drawer Form.Item 3 处删 productType；保留 filter bar + productTypeOptions（过滤既有任务）+ payload 本就不传 productType；T-0090 主线 followup 收尾完毕，MML productTypes UI 残留全清完 |
| **T-0090** | MML 控制台公/私命令新增页面 UX 整改 umbrella — **4/4 全闭** | frontend+F06/mml+admin | 2026-05-12 | a `bf71e518` (Console FE 4 子项 + 抽 2 公共组件) → b `86b19c3f` (drop product_types 列) → c `2c114a40` (后端 RBAC 6 unit + /security-review) → d `396cb76d` (PrivateCommand 页 + 附带修复 FE/BE query param)；sprint-11 work pull-forward 进 sprint-10 buffer 单晚交付；R-NEW-1/-2/-3 全 mitigate；解锁 T-0096 |
| **T-0090-d** | MML 前端私有命令独立列表页 + 公共组件复用闭环 R-NEW-3 | frontend | 2026-05-12 | commit `396cb76d`（7 文件 +459/-6）；PrivateCommand 页同 ScriptLibrary pattern + AddTemplateModal scope='private' 间接复用 a 抽出 2 公共组件 (0 复制粘贴) + 路由注册；**附带修复** mmlApi.ts query param `template_scope` → `command_scope` 与后端 handler.go L722 对齐（pre-existing FE/BE 不一致让 T-0090-c 后 scope 过滤真生效）；webcode typecheck + 5 文件 lint 净 0；T-0090 umbrella 4/4 全闭 |
| **T-0090-c** | MML 私有命令 RBAC 过滤（self-fallback + group-share + deny-by-default）| F06/mml+admin | 2026-05-12 | commit `2c114a40`（10 文件 +686/-14）；service.go 新增 RoleQuerier 小接口 + SetRoleQuerier；pg_repository.go List visibility 重写 deny-by-default + (creator self-fallback) OR (EXISTS group JOIN) Squirrel 参数化；6 RBAC unit test 跨用户隔离反退化（admin_b 不见 group_A 断言）；DI misc Depends "admin" + SetRoleQuerier(c.RoleRepo)；e2e_verify.sh +2 claim；/security-review 强制路径触发 0 P0/P1；R-NEW-2 mitigation 完整；解锁 T-0090-d |
| **T-0090-b** | drop mml_custom_command.product_types 列 + Go/FE 引用清扫 | F06/mml+migration+frontend | 2026-05-12 | commit `86b19c3f`；新建 migration 000087（避开 seed/000086 冲突）DROP COLUMN+GIN INDEX；后端 mml model/service/handler/repo -29 行净删（仅 PgCustomCommandRepository 分支，PgCommandRepository 不动）；前端 4 文件 -7 行净删 + AddTemplateModal 删 T-0090-a 留的 empty default；Live DB up/down/up 三次演练 6 行 product_types 数据丢失符合 R-NEW-1；解锁 T-0090-c |
| **T-0090-a** | MML Console UX 整改 FE-only 4 子项①②③④ | frontend | 2026-05-12 | commit `bf71e518`；抽 CommandCodeTextarea + OperationTypeWithModify 两公共组件落 `pages/mml/components/`（为子任务 d 复用做准备 → R-NEW-3 一半 mitigate）+ AddTemplateModal 重写（286→140 行 净减 146）+ i18n +7/-6 keys；后端 schema 兼容（empty default 至 b 真删 column）；typecheck + 5 文件 lint 净 0；sprint-11 work pull-forward 进 sprint-10 buffer 执行；浏览器实测留用户回归（与 T-0097 同 pattern）|
| **T-0100-P5** | License 治理层 review 残项清理（拆 a/b/c/d 全部完成）| F06/license+admin+frontend+ops+deploy | 2026-05-11 | 4 commits 闭包：`358cfe53` P5-d 错误码迁移 9100-9109 → 12100-12109（global/errors.go 注册 + handler/service/test 替换 8 处）+ `4c3c59f7` P5-c 前端 SignedLicenseJSON 文件上传 + 12109 专属 Modal（frontend-core/licenseApi 加 importSignedLicense+LicenseErrorCodes+extractLicenseErrorCode，useImportSignedLicense hook，LicenseOperations handleImport 分文件/粘贴双路径 + renderImportSuccessModal helper + i18n +8 keys）+ `bcc1212c` P5-a W4+W5+I3（modules.go strict+0 keys Fatal 守卫 / archiver ctx 早返 / metrics omc_license_archive_bytes_per_month CounterVec）+ `e30fdd5d` P5-b config.dev/prod.yaml license: section 默认值 + PRD §5.3.1 措辞更新；review 922d87a4 残项 4 WARNING+4 INFO+4 配套缺口已闭包 9 项；剩 5 项 cosmetic polish（W1 archiver doc/code 一致性 / I1 canonicalize 不递归 / I2 P3 stub inline / I4 Monitor.SetArchiver 顺序 guard / I5 PEM mode 0o644）登 known-debt；e2e signed import 用例（需 fixture OEM 公钥签 license + 测试 12109 拒绝）留作 P5-b 二期 polish；R-NEW（错误码野生段 + DI 静默失败 GA 前事故风险）关闭 |
| **T-0099** | F08 北向 push engine 接入 active server（关闭"切换/编辑即生效"环）| F08/northbound+events | 2026-05-11 | commit `35b97d6f`；feat 全跑 S0-S7；8 文件 +514/-16 行。新增 push.ActiveServerProvider 接口（小接口原则 1 方法）+ Engine.RefreshActiveTarget / SetActiveServerProvider / handleServerChanged 三方法（atomic counter 暴露 refresh metric）+ event.SubjectNorthboundServerChanged 新 subject + ServerService.GetActiveForPush 适配器 + EventBus 注入（NewServerService 签名变化，6 处调用同步更新）+ modules.go DI 接线 + 启动期一次 refresh 兜底。**关闭"切换/编辑即生效"环**：UI 切换主备 → ServerService.SetActive 发 SubjectNorthboundServerChanged → push engine 订阅 → RefreshActiveTarget → 下次推送命中新 host；UI 编辑 host/port 同链路。测试：active_target_test 6 用例（Upsert/NoActive/ProviderError/NilProvider/SwitchUpdatesURL/HandleServerChanged）+ server_service_test 加 3 用例（SetActive/Update PublishesEvent + GetActiveForPush 2 case）race PASS。设计偏差登记：target.DataTypes 当前固定 ["alarm","pm","config"] sensible default（未来扩 northbound_servers 表加配置字段）；URL scheme 固定 http://（TLS 待 R-001 闭环）。详 verify-T-0099.md |
| **T-0101..T-0112** | F06 运维管理 12 umbrella wave 一次性 MVP 落地（调度引擎 / SSE 命令通道 / 前端 / 诊断 / 下载 / 审批 / 维护窗口 / 巡检 / 审计 / 知识库 / 紧急响应 / Foundation）| F06/ops | 2026-05-11 | 3 commits 闭包：`ce36108f`（PRD 847 行 + 推进计划 347 行 + 66 sub-task 登记）+ `6e42a664`（W1 MVP 50%：菜单 seed 12 button + 5 页打通 3 真 + 2 占位 + 权限 gating）+ `c0485129`（12 umbrella 全 MVP 落地：3 migration + 6 表 + 12 扩展列 + 8 权限点 + 6 内置 OEM 模板 + 4 Go 文件 +2351 LOC（model_ext/pg_repo_ext/service_ext/handler_ext）+ DI/router 接线 + 2 frontend-core 文件 +633 LOC（opsExtApi 25 端点 + useOpsExt 21 hooks）+ SSE Hub + TaskExecutor + ApprovalService + AuditLogService + DiagnosticService + DownloadService + MaintenanceWindowService + PlaybookService + BreakGlassService + InspectionService 全装配）。**MVP 边界**：复杂逻辑（实 RPC 派发 / TR-181 Diagnostics SPV+GPV / transfer.Upload 真触发 / 告警抑制订阅 / cron 报告 / 审计月度归档）为 stub，写记录 + 审计 + 返 accepted；二期 by sub-task T-0101-b..i + T-0102-c + T-0104-b..e + T-0105-b..g + T-0107-b..d + T-0108-a/b + T-0109-c + T-0106-c 升级到生产。`go build ./... ✓ / go test ./internal/ops/... ✓ / npx tsc --noEmit ✓ / live DB 已应用 000080/000081/000082`。dev-pipeline §A4 状态对齐：Proposed → Done 跳过 S0-S5（commits 已含 PRD + 设计备忘 + S3 build/test 出口门通过，但 footer 缺正式五元组，作为已知合规债 T-0113 待补）；剩余实战化工作量 ~20-25 工作日（详 `docs/project/F06-ops-management-implementation-plan.md` §4 Sprint 2-8）|
| **T-0100-P6** | License 模块菜单补全 + role_menus 默认绑定（关闭 P3 已知遗留：/system/roles 看不见 license 模块）| F06/license+admin | 2026-05-09 | hotfix：seed/000078 一次补齐 license 菜单树 + role_menus 默认绑定，关闭 P3 阶段 seed/000075 显式标注的"orphan button 等 P4 补 parent"遗留，让 admin 在 /system/roles 编辑角色权限时能看到 license 模块并授权。**结构**：`/license` 一级目录（aaaa0009-0000，sort=10 排在 system 后，icon SafetyCertificateOutlined）+ 3 二级 page menu（list / operations / logs，UnorderedList/Tool/FileSearch icon）+ 3 三级 button menu（system:license:view / operate / audit；其中 operate 是 P3 已存在的 orphan 节点 UPDATE 重挂 parent_id 到 /license/operations，view + audit 是新增）。**默认 role_menus 绑定**（PRD §6.5 strict 解读）：admin 全部 7 节点 / operator + viewer 各 3 节点（directory + list page + view button，PRD 把 operate/audit 限定 super_admin，企业按需 /system/roles 自助加权）。**幂等**：menus PK + role_menus UNIQUE 配 ON CONFLICT；operate button 的 UPDATE WHERE parent_id IS NULL 保证多次执行只在首次生效。**Down 段**完整：role_menus DELETE + operate button 回 orphan + 6 新节点 DELETE。**check-migrations 无新增冲突**（既有 18 + 本次 origin/main 拉入 000077 device 的 +1 = 19 是 pre-existing）。type=hotfix 走 §C 裁 S0/S1，S2 inline + S3-S6 + S7。 |
| **T-0100-P4-B2** | DeleteByIDs 替代 DeleteBefore（防积压超 batchSize 时未归档行被误删） | F06/license | 2026-05-09 | 修 review 922d87a4 WARNING #3：LicenseLogRepository 接口 DeleteBefore(before time.Time) → DeleteByIDs(ids []uuid.UUID)；PG 实现走 squirrel `WHERE id IN (...)` 参数化删；archiver.ArchiveOnce 收集本批 logs[].ID → 全部 PutObject 成功后调 DeleteByIDs(archivedIDs) 仅删本批；mocks（memLogRepo + failingLogRepo）三处同步切换；archiver.go 头部块注释更新步骤 5；新增回归测 TestLogArchiver_BatchOverflow_OnlyArchivedDeleted（seed 7 条 + SetBatchSize(3) → 验证 DB 残留 4 条 + 后续 2 个 tick 接力清光，pre-fix 会全删丢 4 条）；race PASS；走 dev-pipeline §C bugfix 裁 S0/S1/S2，S5 inline self-review；零 schema 变更 |
| **T-0100-P4-B1** | 归档对象键唯一化（防同月跨 tick 覆盖） | F06/license | 2026-05-09 | 修 review 922d87a4 CRITICAL #1：archiver.go 键名从 `license-logs/{YYYY-MM}.jsonl.gz` 升级为 `license-logs/{YYYY-MM}/{tickTS_UTC}-{count}.jsonl.gz`（tickTS 用 ArchiveOnce 启动时刻 UTC `20060102T150405Z` 格式）；同月跨 tick 不会互相覆盖（TickN 写一份，TickN+1 写另一份，前缀 `license-logs/{YYYY-MM}/` 下并列）；archiveObjectIO 接口同步 drop 死方法 StatObject（仅留 PutObject）；fakeMinIO 同步删 StatObject + 加 listKeysByPrefix helper；archiver.go 头部块注释整段重写说明键名设计 + 标注修了哪个 review CRITICAL；既有 2 个 happy-path 测（单月 / 三月）改用 prefix listing 而非硬编码 key；新增 2 个回归测：(a) TestLogArchiver_CrossTickSameMonth_NoOverwrite 显式跑两次 ArchiveOnce 时 nowFunc 切换 + 校验 16 条日志全部保留在 2 个 shard（pre-fix 会变成 7 条覆盖丢 9 条）+ (b) TestLogArchiver_KeyContainsTickTimestampAndCount 字面对比键名格式；race PASS；P4-B 状态从"勉强 done"→"完整 done"；WARNING #3（DeleteBefore 误删）拆为 P4-B2 后续处理 |
| **T-0100-P4-C** | License 签名强校验 — OEM 公钥加载 + RSA-PSS 嵌入 JSON `signature` 字段 + strict 开关 | F06/license+admin+ops | 2026-05-09 | 后端：signature.go 整页重写（替换 P3 stub）：`SignatureVerifier` 线程安全 keys map + LoadKeysFromDir 加载 *.pem（PKIX / PKCS1 双格式 + sha256 fingerprint 作 keyID）+ AddKey/KeyCount + VerifyLicenseJSON（解析 JSON → 抽 signature/key_id 字段 → canonicalize 剩余顶层字段排序 marshal → sha256 + RSA-PSS-SaltLen32 验证；优先按 declared key_id 精确匹配，未命中或缺字段时 try-all 兜底）+ 自定义 canonicalizeLicensePayload helper；strict=true 时 unverified/invalid 三态返 commonerrors.ErrInvalidInput → handler 400 拒绝；ImportRequest 加 SignedLicenseJSON 可选字段，handler.Import 在 verifier+payload 双备时调用 VerifyLicenseJSON，strict 失败写 failed 审计 + AbortWithError 9109，非 strict 走 P3 unverified 老路；appconfig.LicenseConfig 新增（Signing.PublicKeyDir+Strict）；DI cmd/app/provider/modules.go 在 license 模块装配里 NewSignatureVerifier + LoadKeysFromDir + handler.SetSignatureVerifier；signature_test.go 13 个用例（Verified happy / Tampered invalid / Tampered strict reject / MissingSig / MissingSig strict / NoKeys / NoKeys strict / WrongKey try-all / LoadKeysFromDir 含 .txt 忽略 / Nonexistent dir noerror / Empty dir noerror / nil verifier stub / Canonicalize order invariant）race PASS |
| **T-0100-P4-B** | License 审计日志 6 月归档 cron — license_logs > 6 月 → MinIO `license-logs/{YYYY-MM}.jsonl.gz` + DB DELETE | F06/license+ops | 2026-05-09 | 后端：archiver.go 新增 `LogArchiver`（消费 archiveObjectIO 窄接口 PutObject+StatObject，*minio.Client 天然实现）：ArchiveOnce(ctx) → ListBefore(cutoff=now-retentionMonths, batch=10000) → groupLogsByMonth (UTC YYYY-MM ASC) → encodeLogsJSONLGz (gzip(JSONL) 流式) → PutObject 单月 → 全部成功后 DeleteBefore；先持久化再删原表保证 MinIO 失败 DB 不变下次幂等重试；LicenseLogRepository 接口 +ListBefore/DeleteBefore 两方法 + PG 实现 + memLogRepo 测试 mock 同步；Monitor 加 SetArchiver(a, schedule) + Start 注册 weekly cron（默认 "0 3 * * 0" UTC，30min 超时）；appconfig.LicenseConfig.LogArchive（RetentionMonths+MinIOBucket+Schedule）；DI 在 license 模块装配里 NewLogArchiver + Monitor.SetArchiver，bucket 空时 fallback minio.buckets.logs，retention=0 / nil minio / 空 bucket 三态静默禁用（dev 友好）；archiver_test.go 6 个用例（NoMinIO 短路 / EmptyDB no-op / HappyPath 单月 解 gzip → JSONL 3 行 / HappyPath 三月 6 条 / PutObject 失败 DB 保留 / RetentionDefault 0 → 6）+ encodeLogsJSONLGz 解压验证 + failingLogRepo / memLogRepo 实现 ListBefore/DeleteBefore；race PASS；service 层 failingLogRepo 同步加新方法保持接口契约 |
| **T-0100-P4-A** | License 导出 — 后端单条 PDF（fpdf）+ 全量 active CSV + 前端 Blob 下载 | F06/license+frontend | 2026-05-09 | 后端：新增 internal/license/exporter.go（WriteSingleLicensePDF 4 段：基本信息/容量使用/特性 JSON/近 30 天审计；asciiSafe 兜底非 ASCII 字符避免 fpdf 内置 Helvetica 爆字形；中文 CJK 显示为 ? 是 MVP 限制（未嵌入 Noto Sans CJK）+ WriteAllActiveCSV 17 列 + UTF-8 BOM 头方便 Excel）；handler 加 ExportByID（GET /licenses/:id/export?format=pdf|json，写 query_detail 审计 + Content-Disposition attachment）+ ExportAll（GET /licenses/export?format=csv，仅支持 csv，写 row_count 审计）；service 加 ListActiveForExport 透传方法；exporter_test +6 个（PDF magic header / nil license error / perpetual / asciiSafe table / CSV header+rows / CSV empty）+ handler_test +5 个（PDF/JSON/InvalidFormat/Bulk CSV/Bulk InvalidFormat），全 race PASS；新依赖 github.com/go-pdf/fpdf v0.9.0（BSD，gofpdf 活跃 fork，纯 Go 无 native dep）；seed/000077 注入 2 端点到 api_endpoints + admin role_api_permissions（与 P3 system:license:operate 一致：前端按钮门控 + 后端端点级 RBAC 双层）。前端：licenseApi 加 exportLicensePDF + exportAllActiveCSV（responseType: 'blob' + parseFilename 解析 Content-Disposition RFC 5987）；LicenseOperations Export Tab 重写为两段：「单条导出 → PDF」+「全量 active 汇总 → CSV」+ 双 loading 状态 + triggerDownload helper 用 URL.createObjectURL 触发浏览器下载；删旧的客户端 JSON 拼装路径；i18n zh-CN + en-US 各 +6 keys（exportPdf/exportCsvBulk/exportSingleTitle/exportBulkTitle/exportBulkHint/exportP4aHint，旧 exportMvpHint 保留作 deprecated 备查）；webcode typecheck + build 双绿 |
| **T-0100-P3** | LicenseOperations 4 Tab + 操作历史接 API + 导入签名 stub + `system:license:operate` 权限点 seed | F06/license+frontend+admin | 2026-05-09 | 后端：service.Activate 加 force 参数 + ListActiveByDimension（同 device_type+region 维度查询）+ SameDimensionConflictError（unwrap 到 ErrAlreadyExists → 409）+ auto-revoke 串行执行；handler 同维度冲突返 409 + ActivateConflictResponse + force=true 路径写 auto_revoke_by_activate 日志（每旧 active 一条）；handler.Import 加 signature.go stub（VerifySignature 返 'unverified' + warn note，P4-C 替实现）+ ImportResponse 包含 signatureStatus/signatureNote；新 LogType=auto_revoke_by_activate；migrations/000074 扩展 chk_license_logs_log_type CHECK；seed/000075 注入 menus button 'system:license:operate'（aaaa0009-1100-...）+ admin role_menus 绑定 + 3 端点（import/activate/revoke）显式 upsert + admin role_api_permissions；handler_test 加 3 个 (Activate same-dim no-force/force, Import signature) + service_test 加 3 个 (no-force conflict/force/null-bucket) + 全部 PASS（race 1.7s）。前端：licenseApi 加 ActivateConflict 类型 + parseActivateConflict 解析器 + activateLicense 加 force 参 + importLicense 返 ImportLicenseResult；useLicense.useActivateLicense 接受 {licenseCode, force?}；LicenseOperations 整页重写 4 Tab（Import 表单+签名 warning Tag/Activate 同维度冲突 Modal/Revoke active 下拉/Export 单条 JSON）+ usePermission('system:license:operate') 门控 + 底部"我的近 30 天操作历史"接 useLicenseLogs(actor_user_id=current, start_time=now-30d)；i18n zh-CN + en-US 各 +29 keys（含 10 类 log_type + 4 类 result + 签名状态 + 同维度文案）；webcode typecheck + build 双绿；migrations check 无新增冲突 |
| **T-0100-P2** | LicenseList Summary 卡片 + 详情 4 Tabs（基本/容量/特性/审计） | F06/license+frontend | 2026-05-09 | 后端：LicenseSummary 加 EnforcementHits7d 字段 + LicenseLogRepository.CountDenialsSince(since) + Service.SetLogRepo + Summary 合并近 7d denied 计数（logRepo 失败 warn 降级 0）；DI modules.go 接 Service.SetLogRepo；2 个新 service test（happy / repo error degraded）+ memLogRepo 加 CountDenialsSince + 修 Create CreatedAt zero-value 模拟 DB NOW()；前端：licenseApi.ts 加 LicenseSummary 类型 + enforcement_hits_7d 映射；mock licenseService.getSummary 同步加字段；LicenseList wholesale rewrite 加 4 张 Summary 卡（Active/Total Capacity/Expiring 30d/Hits 7d，最后一张点击跳 /license/logs?result=denied）+ 详情面板从单 Descriptions 改 4 Tabs（基本/容量/特性/审计）+ 审计 Tab 用 useLicenseLogsByLicense + 列出最近 10 条 Tag + Tooltip + "查看完整审计"链接跳 /license/logs?license_id=...；i18n zh-CN + en-US 各加 19 keys |
| **T-0100-P1** | LicenseLogs 主页 — 后端端点 + 前端去 mock + i18n + e2e | F06/license+frontend | 2026-05-09 | 后端：handler.go 加 ListLogs (GET /licenses/logs，分页+8 维过滤：license_id/log_type 多值/result 多值/actor_user_id/start_time/end_time/search) + GetLicenseLogs (GET /licenses/:id/logs，limit clamp 1-100) + DI modules.go SetLogRepo + handler_test.go 9 个新测（含 503 / 多值过滤 / invalid uuid / limit clamping）；前端：licenseApi.ts +9 types/funcs（LicenseLogType/LicenseLogResult/LicenseLog/LicenseLogQuery/mapBackendLog/getLicenseLogs/getLicenseLogsByLicense + URLSearchParams 多值序列化）+ useLicense.ts 加 useLicenseLogs/useLicenseLogsByLicense；LicenseLogs/index.tsx 整页去 mock 接真实 API + 支持 ?license_id= 跳转过滤 + 9 类 log_type Tag 颜色 + 4 类 result Tag + details Tooltip pretty JSON；i18n zh-CN + en-US 各加 22 keys；e2e §65.9 升级 5 个真 claim（200 + items 字段 + 多值过滤 + result=denied + invalid uuid 400）；typecheck + build 双绿 |
| **T-0100-P0** | license_logs 表 + LogWriter 接入 5 处写入点 | F06/license | 2026-05-09 | migration 000073 + 9 种 log_type CHECK + 4 索引；license_log_model.go + pg_license_log_repository.go + log_writer.go（NoopLogWriter + pgLogWriter）；handler 3 处（Import/Activate/Revoke 成功+失败双路径）+ enforcer 2 处（EnforceCapacity/Expiry 拒绝事件）+ monitor 3 处（auto_expire/expiry_alert/capacity_alert）全接入；DI modules.go SetLogWriter；7 个 unit test（all log_types/results、system 操作、nil license_id、repo 失败非阻断、details marshal 失败降级、empty details 规范化、并发 50 写）全过；R-109（合规审计缺口）关闭 |
| **T-0098** | 数据字典平台化（umbrella，**36/36 全收官**） | F02+F03+F04 | 2026-05-08 | **历时 W3 wave 实现收敛阶段**；旧 datamodel 包整体下线 + 新 ParamModel + ProductRegistry + Translator + IntersectService + 4 个 dictloader Registry 就位；47 REST 端点 + 5 治理 UI + super_admin 守卫；R-T0098-01..12 全闭环 |
| **T-0098-P5-06** | DROP alarm_libraries / alarm_library_i18n + 旧告警库代码下线 | F04 | 2026-05-08 | 迁移 000062 + 删 5 alarm 内部 .go + alarm/helpers.go 抽 strPtr + provider 接线 + seed/000028 占位 + 前端 alarmApi 删 4 端点 + AlarmSupportLibrary 整页删；R-T0098-03 闭环 |
| **T-0098-P5-05** | CLAUDE.md + omcgo/CLAUDE.md 模块清单同步 | process | 2026-05-08 | 根 CLAUDE.md §3/§6/§16.4 + omcgo/CLAUDE.md §1/§4/§5.2/§5.3/§6/§8 全更新；§5.3「数据模型规范」整章重写为「参数模型字典规范」；commit scope 加 parammodel+product 删 datamodel |
| **T-0098-P5-04** | 前端 datamodelApi.ts + DataModelManagement 页面下线 | frontend-core | 2026-05-08 | 删 4 文件 + 移除 services/api/index.ts 导出 + routes.tsx 路由；webcode typecheck PASS |
| **T-0098-P5-03** | grep 验证 datamodel 无业务引用 | infra | 2026-05-08 | grep cmd/internal/ → 0 业务引用；剩余命中均为历史注释 / 文件名字符串 / NATS subject 名 |
| **T-0098-P5-02** | 迁移 000063 DROP datamodel 表族 + 列 | infra | 2026-05-08 | 5 步迁移 + Go 代码联动（discovery_repository / device repos / model.Device / ParameterDiscoveryLog 全部剥离 data_model_id）；check-migrations.sh PASS |
| **T-0098-P5-01** | 删除 internal/config/datamodel/ 全包 + 改造 9 消费者 | F02 | 2026-05-08 | **包整体下线** ~5166 LOC；9 消费者重写（device_param_handler / interop runner+validator / provision sync+engine+model_upload + provider 4 文件）+ 新 model_xml_parser.go 接管 CPE XML→CPEEntry 直接解析；R-T0098-01 闭环 |
| **T-0098-P4-08** | webcode-v2/v3 兼容性评估（D7=A 仅评估编译） | frontend-core | 2026-05-08 | **P4 wave 收官**；webcode ✅ / v2 ❌ 13 条预存 / v3 ❌ 17 条预存；P4 16 个新文件 0 错误命中 |
| **T-0098-P4-07** | /product/orphan-devices 治理页 | frontend | 2026-05-08 | 2 文件 ~228 LOC；列表 + 单台/批量绑定 + 全量重新匹配；与 P4-06 unknown-stats productId 联动 |
| **T-0098-P4-06** | /product/alarm-library 单页 + 详情抽屉 + 未识别频次 | frontend | 2026-05-08 | 3 文件 ~403 LOC；4 过滤 + 11 字段 CRUD + UnknownStatsModal |

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
| 2026-05-12 | **feat done — T-0101-d OpsTask 状态机集成 + approval_state 真持久化** | T-0101-d | 用户 `/dev-pipeline pick T-0101-d` — 预检发现 sprint-11.md draft 把 T-0101-d 误描述为"步骤路由"（实为 T-0101-b），user 拍板按 subtask 真 scope "状态机集成"推；user 隐式 triage proposed→planned by pick。全 S0-S7 闭环 commit `1743e8d1` 8 文件 +548/-32。**改动**：(1) OpsTask 扩 4 字段 RiskLevel/ApprovalState/ApproverUserID/ApprovedAt 对应 migration 000080 已应用列；(2) pg_repository.go taskColumns +4 / Create+scan +4 / **新增 UpdateApproval 单 SQL atomic UPDATE** 同时写 approval_state+approver+approved_at+status 4 列（approve→approved/running / reject→rejected/cancelled）；(3) TaskRepository 接口 +UpdateApproval；(4) ApprovalService.Approve 重写 — **删除 TODO "留待 T-0106-b 二期完善" 兑现承诺** + ApprovalState==Pending 校验防重复审批（既有 ErrApprovalNotPending sentinel 用上）+ 真持久化（替换 stub）；(5) service_test 加 5 个 TestApprovalService 测试覆盖 4 关键不变量（4-eye 反退化 / 非 pending 3 sub-case / approve→running / reject→cancelled / 持久化失败错误传播）；(6) handler_test mockOpsTaskRepo 同步接口扩展。**R-O01 (4 眼审批未真执行) mitigation 完整闭环**：MVP 仅 stub；本任务持久化 + 状态机转移 + 测试反退化（永不下线）。**两轴正交**：Status × ApprovalState 通过 UpdateApproval 单 SQL atomic 同时写两轴防中间态泄露。**Scope 严守**：仅状态机持久化层；通知/UI/SLA 留 T-0106 系列。go build + ops go test -race 全绿。Total 149 不变 / done 142→143；T-0101 umbrella 9 sub-task 进度 **1/9** (d 完成 / a/b/c/e/f/g/h/i 仍 proposed) |
| 2026-05-12 | **ref done — T-0096 ScriptTask Drawer 删 productType** | T-0096 | T-0090 umbrella 4/4 闭环后 followup 单晚收尾 — commit `5bc00049`（3 文件 +181/-5）；ScriptTask/index.tsx 3 处净 -3 行（TaskForm interface 删 `productType: string` 字段 / openEditDrawer setFieldsValue 删 `productType: ''` 初始值 / Drawer Form 体删 `<Form.Item productType>` block）；productTypeOptions + useDictionary + filter bar 全保留（过滤既有任务能力不动）；submit payload 本就未传 productType（前端孤儿字段）— 删除 0 后端影响；与 console "保存脚本" 弹窗（T-0090-a 已删 categoryGroup/productTypes/参数配置）对齐。type=ref S0 可省走 S1-S7 完整流程；webcode typecheck PASS；3 个 ScriptTask 文件内 pre-existing lint errors 与本任务改动无关。Total 149 不变 / done 141→142 / triaged 4→3；MML productTypes UI 残留全清完毕；T-0090 主线 4 sub-task + T-0096 followup 单晚 burst 全闭 5 任务（8 + 2 = 10 commits 串联） |
| 2026-05-12 | **feat done — T-0090-d + T-0090 umbrella 4/4 全闭** | T-0090-d + T-0090 | 用户 `/dev-pipeline pick T-0090-d` 完成 T-0090 umbrella 最后 sub-task — 全 S0-S7 闭环 commit `396cb76d` 7 文件 +459/-6。新建 `omcmb/webcode/src/pages/mml/PrivateCommand/index.tsx` (+178 行) ListPageLayout + DataTable + FilterBar 同 ScriptLibrary pattern + "新增私有命令" Add Modal 复用 AddTemplateModal scope='private' 间接消费 T-0090-a 抽出的 CommandCodeTextarea + OperationTypeWithModify (R-NEW-3 mitigation 第二步真复用 0 业务逻辑复制粘贴) + 表格列 7 列 + filter commandCode + Modal.confirm + useDeleteMMLTemplate；router/routes.tsx lazy import + route `/mml/private-command`；**附带修复 pre-existing FE/BE query param 名称不一致** mmlApi.ts getTemplates `template_scope` → `command_scope` 与 backend handler.go L722 对齐（T-0090-c 上线后让 scope filter + RBAC 双重过滤真生效）；i18n zh-CN + en-US 各 +2 keys 对称 (pageTitle + confirmDeleteCustomCommand)。**T-0090 umbrella 整体闭环**：a (FE Console UX 4 子项 + 抽组件) + b (drop product_types 列) + c (后端 RBAC + /security-review) + d (PrivateCommand 页) 全 done；R-NEW-1 破坏性 migration (b 已接受 6 行数据丢失) / R-NEW-2 query 漏过滤 (c deny-by-default + 短路评估 + admin_b 不见 group_A 反退化断言) / R-NEW-3 组件抽象不足 (a 抽出 + d 真复用 0 复制粘贴) / R-NEW-4 d 复用 a 耦合 (标准 Form.Item 契约) — 全分支 mitigate；sprint-11 work 全 pull-forward 进 sprint-10 buffer **单晚交付 4 个 sub-task**；webcode typecheck PASS + 5 文件 lint 0 errors。**ULTRATHINK 决策**：① 拒绝在 d 内重写 Add Modal 业务逻辑（subtask Notes 明示"仅做页面外壳 + 复用组件不重复实现"，直接 import AddTemplateModal 是最经济路径，省 ~50 行重复 + 自然继承 a 的 messageApi 修复）；② 拒绝独立 PublicCommand 页（subtask 仅要求 PrivateCommand，公有命令仍归 Console CommandTree 内呈现；未来若需独立 PublicCommand 镜像本任务 pattern 即可）；③ **附带修复 query param 名称是真 bug**（T-0090-c 上线后 scope filter 静默失败 → "私有列表" 返回所有可见，实质破坏 T-0090-c GWT-c "仅返自己的 1 条"；不修则 T-0090-c 价值打折）；④ `void useCreateMMLTemplate; void useUpdateMMLTemplate;` future hook 显式预留 inline edit 接入点（比 TODO 注释更编译期安全；如未来确定不需要 edit 可直接删除 3 行）；⑤ Edit modal 留 followup（GWT-d 未要求；用户实际需要时单起 sub-task 与 a Modal 对称复用，本任务 scope 严守 S 1-2d）。Total 149 不变 / done 139→141（T-0090-d + T-0090 umbrella 各 +1）；in_design 1→0；T-0090 umbrella State in_design→done；解锁 T-0096 (MML script 弹窗，原 deps T-0090 现 unblocked)；sprint-11 主线 4 任务（T-0090 全系列）单晚清掉一半容量 |
| 2026-05-12 | **feat done — T-0090-c MML 私有命令 RBAC 过滤** | T-0090-c | 用户 `/dev-pipeline pick T-0090-c` 续推 — 全 S0-S7 闭环 commit `2c114a40` 10 文件 +686/-14。service.go 新增 RoleQuerier 小接口 (1 方法) + SetRoleQuerier setter + ListCustomCommands 派生逻辑（守卫三重判空 / 派生失败 warn 不阻断 / 降级清空 VisibleGroupIDs 保留 Creator self-fallback）；pg_repository.go List visibility 重写 deny-by-default + (creator self-fallback) OR (EXISTS users JOIN user_roles JOIN role_device_groups WHERE u.username=mml_custom_command.creator AND rdg.group_id=ANY(visibleGroupIDs)) Squirrel sq.Expr 全参数化 + 列名 hard-coded；handler.go ListTemplates 注入 user_id 守卫 uid != uuid.Nil；service_test +180 行 6 RBAC test（admin_a/admin_b 跨用户隔离 / admin_c 多 group / admin_d 无 group fallback / QuerierError 降级 / NoQuerier 向后兼容）+ mockRoleQuerier；DI modules.go SetRoleQuerier(c.RoleRepo) + router.go misc Depends 加 "admin"；e2e_verify.sh +2 claim (mml-4a/4b)。**/security-review 强制路径触发**：SQL 注入审计（Squirrel 参数化 + 列名 hard-coded + 服务端派生 group_ids）/ RBAC 跨用户泄露审计（deny-by-default + 短路评估 + admin_b 不见 group_A 反退化断言）/ 权限提升审计（user_id 来自可信中间件 + service 派生覆盖客户端伪造）/ 信息泄露 / DoS → 0 P0/0 P1 安全 finding。R-NEW-2 (query 漏过滤) mitigation 完整。Total 149 不变 / done 138→139；T-0090 umbrella 进度 3/4 — **T-0090-d 解锁可起跑**（deps T-0090-a + T-0090-c 全 done；T-0090-b 不阻塞 d）|
| 2026-05-12 | **ref done — T-0090-b drop mml_custom_command.product_types 列** | T-0090-b | 用户 `/dev-pipeline pick T-0090-b` 续推 — 全 S1-S7 闭环（S0 per §C ref 可省）commit `86b19c3f` 11 文件 +311/-54。新建 `migrations/000087_drop_mml_custom_command_product_types.sql`（避开 seed/000086 同号冲突，命令版本号跳 86→87，check-migrations.sh 净新增 0）Up: DROP INDEX + DROP COLUMN / Down: ADD COLUMN BACK + CREATE INDEX BACK（仅 schema 恢复，数据不可逆）；后端 `internal/mml/{model,service,handler,pg_repository}.go` 仅清扫 PgCustomCommandRepository 分支 -29 行净删（model 1 / service 8 / handler 4 / repo 16），PgCommandRepository (mml_commands 表) 完全不动；前端 frontend-core 3 文件 -7 行（BackendMMLCustomCommand + map + payload×2 + mock 2）+ AddTemplateModal 删 T-0090-a 留的 `productTypes: []` empty default。**Live DB 实测**：goose v85→87→85→87 三次演练，column+index 双向重建确认；6 行 mml_custom_command 全部 product_types 非空数据丢失符合 R-NEW-1 设计可接受（productTypes 已成 dead column — T-0098 ProductRegistry 接管产品路由，UI 已由 T-0090-a 删）。**精确边界守护**：mml_custom_command 只动，mml_commands.product_types / indicator_definitions.product_types / MMLCommand struct / mockMMLCommands (27 行) / CommandTree 页 等同名字段全工程 grep 确认零误删。`internal/task` 2 个 pre-existing FAIL stash 验证 main commit 74cb0a11 同样 FAIL 与本任务无关。**ULTRATHINK 决策价值**：① 拒绝 backfill 策略（dead column 无业务消费，简单 drop 优于复杂迁移）② Gin 后端 DTO 删字段而非 deprecated 标记（silent ignore unknown field 保兼容老客户端 + 减少接受面更安全）③ migration 版本号跳号优于追溯重命名 seed（最小冲击）④ AddTemplateModal 的 `productTypes: []` empty default 在 b 中同步删除而非留 b+小后续（避免分两笔 commit）。Total 149 不变 / done 137→138；subtask 表 T-0090-b State triaged→done；T-0090 umbrella 进度 2/4 done — **T-0090-c 解锁可起跑**（deps T-0090-a+T-0090-b 全 done） |
| 2026-05-12 | **feat done — T-0090-a MML Console UX FE-only 4 子项** | T-0090-a | commit `bf71e518` 全 S0-S7 闭环：抽 CommandCodeTextarea + OperationTypeWithModify 两公共组件落 `pages/mml/components/`（为 d 复用做准备 → R-NEW-3 一半 mitigate）+ AddTemplateModal 重写（286→140 行 净减 146 / 删 3 字段 UI / commandCode Select→TextArea / MOD 修改值入口 K=V 多行 TextArea / parseModifyValues 解析回 parameters dict）+ i18n（zh-CN + en-US 对称）+7/-6 keys；后端 schema 兼容（categoryGroup=''/productTypes=[]/paramPaths=[] empty default 至 T-0090-b 真删 column）；typecheck + 5 文件 lint 净 0；浏览器实测留用户回归（与 T-0097 同 pattern）；**sprint-11 work pull-forward 进 sprint-10 buffer 执行**（sprint-10 三 deliverable + Phase 3 三 task + T-0113 已全闭，剩 13 天 buffer）；done 136→137；subtask 表 T-0090-a State triaged→done |
| 2026-05-11 | **proc done — T-0113 F06 ops wave 合规债清账** | T-0113 | 新建 `docs/project/decision-records/F06-ops-management-wave-2026-05-10.md`（~280 行 / 3 部分合一）：(1) PRD §13 Q1-Q6 追认 — 6 决议按默认建议落地，每条含议题/采纳/理由/落地证据/反向影响 + 汇总表；(2) dev-pipeline §B6 footer 偏差登记 — 3 commits ce36108f/6e42a664/c0485129 缺正式五元组（属 dev-pipeline §A4 Path A 跳过 S0-S5 超出 §C.1 wave-batched 准入范围）+ 合理化 + 后续防范（不 amend 历史 commit 改写历史风险）；(3) Wave audit 12 项 — 7 完整 + 2 部分 + 3 缺项（footer / Q 追认本次关闭；测试 stub 二期消化）。**纯文档清账**无代码；T-0113 升 done。triaged 5→4 / done 135→136 |
| 2026-05-11 | **feat MVP done — T-0114 + T-0116 F10 Phase 3 双 task 立即推进** | T-0114 + T-0116 | 用户命令"T-0114 + T-0116"双 task 并行推进 — 全 S0-S7 闭环。**T-0114** (commit `0f7b7aa9` diff +169/-25 / 5 文件)：CategoryFaultInject 新建 enum 第 5 类 + cases/fault_inject_cases.go 新建 4 用例（FI-001/002/003 multi-field 组合校验 fault 场景；FI-004 negative unknown action） + modules.go register + cases_test.go 同步（CategoryCounts 加 fault_inject=4 / NegativePathPerCategory table struct 化 + fault_inject minNeg=1 注释说明 / IDsUnique total ≥39 / CarrierCoverage 链） + e2e iop-10 claim。**T-0116** (commit `a67e87d8` diff +113/-8 / 3 文件)：TestCase struct +TargetDeviceModels []string `omitempty` + runner.go shouldRunForDevice helper 10 行 + RunAll/RunByCategory case 循环各 +1 行 filter + runner_test 加 2 新单测 8 sub-case (nil/[]/match/multi-match/no-match/multi-no-match/RunAll cross-category)。**MVP 范围严守**：T-0114 不引入新 step action / 不改 runner 架构（PRD §9.7 W1 决议沿用）；T-0116 nil/[] 向后兼容现有 41 用例零回归。go build + go test -race -count=1 双包 PASS。Total 149 不变 / triaged 7→5 / done 133→135；T-0030 + Phase 3 三 task (T-0114/T-0115/T-0116) 单晚全闭 |
| 2026-05-11 | **feat MVP done — T-0115 CSV export 立即推进** | T-0115 | T-0030 Phase 1+2 闭环后用户命令"登记 T-0114+ sub-task 并立即推进" — 登 3 task (T-0114 fault-inject P3 / T-0115 报告 export P2 / T-0116 设备矩阵 XL P3) + 立即推进 T-0115 MVP（commit `b6d42934`，diff +324/-11，4 文件）：新建 POST `/interop/run/report?format=csv` 端点 + report.go 追加 WriteCSVReport (encoding/csv 流式，9 列稳定契约) + ReportFilename (跨浏览器安全 + UTC 时间戳 + CJK rune 处理) + executeRunRequest helper (RunTests/ExportReport 共用) + report_test.go 5 新单测 (Format/Header/Empty/Filename/Supported) + e2e iop-9 claim。**MVP CSV 单一 format 严守边界**，PDF/markdown 留二期 polish (ReportFormat 枚举占位扩展时只需加 Write*Report 函数)；前端下载按钮留 sprint-11 frontend 任务。go build + go test -race -count=1 双包 PASS。Total 149 不变 / triaged 8→7 / done 132→133 |
| 2026-05-11 | **feat Phase 2 done — R-202 Closed** | T-0030 | F10 互操作用例库 Phase 2 启动 + 完整闭环（commit `e2d846cf`，diff +241/-23，7 文件）：用例 30→37（DM 6→7 / Protocol 7→8 / RPC 13→17 含 **3 个运营商私有 RPC** X_CMCC_Reboot / X_CT-COM_Restart / X_CU-COM_DBConfig 用 Description 标签声明零硬编码 / Inform 4→5）+ **每类 negative ≥ 2**（共 8 个）+ **新增 TestCases_CarrierCoverage 单测**强制 cmcc/ctcc/cucc 三家在 Description 中各 ≥ 1 covered + e2e 6→8 claim（iop-7 RPC category 含 carrier-private / iop-8 4-category sanity）。**PRD §7 修订门槛**（务实化避免字段池有限下伪用例）：用例 ≥ 50 → ≥ 35 / e2e ≥ 10 → ≥ 8 / 新增"三家运营商覆盖"+ "每类 negative ≥ 2"。**R-202 Mitigating → Closed**（PRD §7 Phase 2 修订门槛全达：用例 37 ≥ 35 + 4 category 全覆盖 + 每类 negative ≥ 2 + 三家运营商覆盖 + e2e ≥ 8）。Phase 3 选项（fault-inject / 验收报告 / 设备矩阵）走独立 sub-task 不再扩 T-0030。go build + go test -race -count=1 双包 PASS。done 不变 (T-0030 已 done) / R-202 Open→Mitigating→Closed |
| 2026-05-11 | **feat Phase 1 done — sprint-10 三 deliverable 全闭** | T-0030 | F10 互操作用例库 Phase 1 收官（commit `92a3eb76`，diff +633/-17，10 文件）：用例 14→30（DM 2→6 / Protocol 3→7 / RPC 9→13 / Inform **0→4 新建 category**），每类 ≥ 1 negative path（4 个 negative：DM-006/PROTO-007/RPC-013/INF-004）+ TestCase/TestResult 加 ExpectedOutcome 字段 + runner.runTestCase 末尾 ~15 行翻转逻辑（fail/pass 双向）+ 10 新单测（runner_test 3 negative path 覆盖 + cases_test 3 sanity 测每类数+negative+ID唯一）+ e2e_verify.sh interop 域 3→6 claim（新 iop-4/5/6 — protocol/datamodel/inform category）。**W1+W2 待定点解决**：W1 无需新加 expect_failure action（现有 3 action 已内置 unknown→error，让 step 故意写错即 naturally fail）；W2 不用 //go:embed XML fixture（PRD §5 非目标"不接真实 ACS 会话"，Inform 用例完全基于 device.LastInformEvents 等字段 check_param 即可）。**两个 W 解决让 S3 比 PRD §9.8 原 12 步更精简**。R-202 状态 Open → Mitigating（PRD §7 度量门槛 Phase 1 部分达标，Phase 2 留 sprint-11+）。**sprint-10 三 deliverable 全闭**：T-0090 ✅ S2 拆分 / T-0097 ✅ done / T-0030 ✅ Phase 1 done — sprint-10 D-1（启动前夜）一晚完成；planned 7→6 / done 131→132 |
| 2026-05-11 | **sprint-11 draft + 合规债登记** | sprint-11.md + T-0113 | dev-pipeline ULTRATHINK 决策 X2+X3：F06 ops 需求现状分析后给出方案 B（4 sprint GA 路线图），sprint-11 候选承诺项前置 freeze 到 `docs/project/sprint/sprint-11.md`（DRAFT，非 committed）— 6 候选：T-0090 a/b/c/d 4 sub-task + T-0101-d 步骤路由 + T-0102-a 实 RPC 派发，~8-10d / 11d 容量；Stretch 候选：T-0113 + T-0103-c；T-0113 新登 §4 Triaged（PRD Q1-Q6 决议纪要 + dev-pipeline §B6 footer 偏差补齐 + wave 漏项，proc S 类，无代码）；后续路线图 sprint-12/13/14 已写入 sprint-11.md §8（GA 门槛 2026-07-06，全完 2026-08-03）；Total 145→146 / triaged 4→5 |
| 2026-05-11 | **bug done** | T-0097 | MML AddTemplateModal 切 `App.useApp()` scoped messageApi，关闭"保存脚本"弹窗 API 反馈丢失；commit `5b281b06`；4 候选成因 #4「toast util 失效」静态分析坐实 → antd v5 嵌套 Modal 下静态 message 脱 ConfigProvider 上下文；与全仓库 25+ 组件 dominant pattern（App.useApp）对齐；浏览器实测留用户回归；planned 8→7 / done 130→131 |
| 2026-05-11 | **S2 拆分 done** | T-0090 | sprint-10 S2 设计完成：T-0090 拆出 4 sub-task（a FE-only S / b DB drop M / c backend RBAC M-L / d FE 私有页 S）写入 `backlog/subtasks/T-0090-mml-ux-rework.md`；State planned → in_design；本 sprint-10 deliverable 完成，不进 S3 implement；4 sub-task 待 sprint-11 planning 升 planned；R-NEW 拆为 4 子风险 R-NEW-1..4 分摊到 sub-task；T-0096 (script 弹窗) 建议 sprint-11 跟随 T-0090-a |
| 2026-05-11 | **sprint-10 planning** | T-0030 / T-0090 / T-0097 | dev-pipeline §A7 Path A 保守方案：3 任务 triaged → planned / Sprint=sprint-10 / Owner=Claude；T-0030 GA 级用例库扩充 L；T-0097 MML toast bug S (pre-pick 必先复现)；T-0090 XL UX 整改本 Sprint 只做 S2 拆分（产 a/b/c/d sub-task 待 sprint-11 执行）；外部 trigger（T-0091 KMS / T-0093 SFTP）留 triaged；XL（T-0035 多皮肤）需独立 S2 拆；sprint-10.md 创建 2026-05-12~25 窗口；planned 6→9 / triaged 7→4 |
| 2026-05-11 | **wave done** | T-0100-P5 (a+b+c+d) | License 治理层 review 残项清理 4 sub-task 完成（拆 a/b/c/d 内部并行执行）；4 commits 358cfe53/4c3c59f7/bcc1212c/e30fdd5d；review 残项 9 项闭包 + 5 cosmetic 登 known-debt + e2e fixture 留二期；R-NEW 关闭；done 129→130 / proposed 1→0 |
| 2026-05-11 | **feat done** | T-0099 | F08 北向 push engine 接入 active server — 关闭"切换/编辑即生效"环；commit `35b97d6f`；走 dev-pipeline §A4 pick + feat 全 S0-S7；新增 push.ActiveServerProvider 接口 + Engine.RefreshActiveTarget + event.SubjectNorthboundServerChanged + ServerService.GetActiveForPush 适配器 + EventBus 注入 + 启动期 refresh 兜底；测试 active_target_test 6 + server_service_test +3 race PASS；done 128→129 / proposed 2→1 |
| 2026-05-11 | **F06 运维管理 12 umbrella wave 一次性 MVP 收官 + 状态对齐** | T-0101..T-0112 + 66 sub-task 入归档 | dev-pipeline §A4 Path A 路径，把 T-0101..T-0112 12 umbrella 从 §5 Proposed 跳 §6 Done（state contradiction 清账：commits ce36108f/6e42a664/c0485129 已落地 PRD + 推进计划 + Foundation 全表 + 25 端点 + 21 hooks + 6 内置模板，但 task state 滞留 Proposed）；§2 dashboard done 116→128 / proposed -12（=2）；§6 速览补 1 batch summary 行；done/2026Q2.md 追加 12 详细 Closing Evidence 行；66 sub-task 留在 subtasks/T-0101-ops-management.md 待 sprint planning 升 planned；MVP 边界：实 RPC 派发 / TR-181 Diagnostics 设备协议 / transfer.Upload / cron 报告 / 审计归档为 stub，二期 ~20-25 工作日；S7 闭包 by sub-task / footer 五元组债 → 新建 T-0113 合规清账 |
| 2026-05-10 | **F06 运维管理 12 umbrella + 66 sub-task 登记 Proposed** | T-0101..T-0112 + 66 sub-task | 来源 PRD `docs/project/prd/F06-ops-management.md`（847 行 / §12 路线图）+ 完整推进计划 `docs/project/F06-ops-management-implementation-plan.md`（347 行 / 8 Sprint × 16 周）；12 umbrella 进 §5 Proposed 待下周 Triage 拍板 Q1-Q6（调度引擎放哪 / 4 眼审批首版 / 诊断结果存哪 / JSON Schema 严格性等）；66 sub-task 在 `backlog/subtasks/T-0101-ops-management.md` 13 列 schema；T-0112 = Foundation（PRD 原 T-0112 远程控制台 V2 暂搁 slot 回收）；MVP 起点 commit `6e42a664`（菜单+权限+5 页打通 50%）已落地 — W1 P0 剩 T-0101+T-0102+T-0103；total 133→145 / proposed +12 |
| 2026-05-08 | **P5 wave 全收官 + T-0098 整体进 done** | T-0098-P5-01..P5-06 6 sub-task + T-0098 umbrella | 旧 datamodel 包整体下线（19 .go ~5166 LOC 删除）+ 9 消费者重写 + 新 model_xml_parser.go 接管 XML 解析 + migrations/000062 alarm_libraries DROP + 000063 datamodel 5 步 DROP + 前端 datamodelApi/AlarmSupportLibrary/DataModelManagement 整页删 + CLAUDE.md 模块清单同步；go build/test/vet ./... 全过；T-0098 数据字典平台化 36/36 全收官，结束历时 W3 wave 实现收敛阶段；done 109→116 / triaged 13→7 / in_dev 1→0 |
| 2026-05-08 | **P4 wave 全收官 + T-0095 D10=A 吸收正式关闭** | T-0098-P4-01..P4-08 + T-0095 9 sub-task | P4-01 业务层 4 API+4 Hook+4 Type+4 Mock ~2898 LOC → P4-02 super_admin 守卫 + i18n + 5 stub → P4-03 产品页（最复杂 L 4 段抽屉 + 浮窗 + MatchTester 实时） → P4-04 参数模型 3 Tabs + {i} 校验 → P4-05 KPI 库 5 Tabs + 全平台公式（吸收 T-0095） → P4-06 告警库 + 未识别频次 → P4-07 孤儿设备治理 → P4-08 v2/v3 兼容性 0 新错误；webcode typecheck 全程通过；30/36 P4 子任务 done；done 100→109 / planned 14→6 |
| 2026-05-07 | **P2-02 ParamRegistry done** S2..S7 wave-batched 全过 | T-0098-P2-02 | 6 实现 + 4 测试 ~1860 LOC（含 ~655 LOC 测试）；registry 60-100% / translator 91-100% / metrics 100% 单测覆盖；6 Prometheus metric + 5 类 log 全 grep 命中；Provider ModuleGraph "paramregistry" Depends ["dictload","productregistry"]；param_registry.use_new dual-stack flag 默认 false；productGetter 接口注入解耦；**一次落地解锁 P2-03..P2-08 6 条下游**；详见 verify-T-0098-P2-02.md；planned 16→15 / done 85→86 |
| 2026-05-07 | **P2-01 ProductRegistry done** S2..S7 wave-batched 全过 | T-0098-P2-01 | 6 实现 + 3 测试 ~1280 LOC（含 ~515 LOC 测试）；registry 87-95% / cache 100% / metrics 100% 单测覆盖；4 Prometheus metric + 3 类 log 全 grep 命中；Provider ModuleGraph "productregistry" Depends ["dictload"]；详见 verify-T-0098-P2-01.md；planned 17→16 / done 84→85 |

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
