# OMC 需求池（Backlog）— 任务状态表

> **性质**：活文档。项目**唯一任务清单**。所有新工作先入池再进流水线。
> **守护人**：项目经理（`CLAUDE.md §16.11`）+ 产品经理（`§16.10`）+ QA/发布经理（`§16.12`）轮值
> **设计依据**：`docs/project/dev-pipeline-design-20260420.md §11` L0 Backlog 层
> **初始化数据**：反向索引自 `docs/project/milestone/2026Q2-to-RC.md` + `risk-register.md` + 近期 commit
> **最近更新**：2026-04-20

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
| Total tasks | 37 | — | 初始化 |
| `done` | 5 | — | 含 R-005 / R-108 关闭 + 流程体系搭建 + L0 Backlog/流水线落地 |
| `in_dev` | 1 | — | T-0006 贯穿 Sprint-01..07 |
| `planned` | 20 | — | Sprint-01 ~ 07 全量切分 |
| `triaged` | 6 | — | P1/P2，暂未排期 |
| `blocked` | 0 | ≤ 3 | — |
| `proposed` 积压天数 | 0 | ≤ 7 | — |
| **P0 风险关闭数** | **1 / 5** | 5 / 5 | R-005 已关；R-001/002/003/004 Open |
| E2E 累计用例 | 0 | 200 | T-0006 贯穿推进 |
| Sprint 承诺完成率 | — | > 75% | 待 Sprint-01 首次回顾 |

**健康度警报**：当前无。

---

## 3. Active — Planned + In-flight（排序：Sprint 升序，同 Sprint 内 Prio 升序）

| ID | Title | Type | Domain | Prio | State | Owner | Est | Deps | Risk/PRD | Sprint | Updated |
|----|-------|------|--------|------|-------|-------|-----|------|----------|--------|---------|
| T-0006 | E2E 用例补齐（贯穿 01→07 累计 ≥200） | td | infra | P0 | in_dev | QA | XL | — | R-002 | sprint-01..07 | 2026-04-20 |
| T-0007 | F04 告警邮件通道 | feat | F04 | P0 | planned | 电信+Go | M | — | R-001 / `prd/F04-alarm-notification.md` | sprint-01 | 2026-04-20 |
| T-0008 | Prometheus/Grafana/AlertManager 容器编排 + 基础 dashboard | feat | ops | P1 | planned | 运维 | M | — | R-107 | sprint-01 | 2026-04-20 |
| T-0009 | 短信服务商凭据申请启动（外部动作） | td | F04 | P0 | planned | PM | S | — | R-001 | sprint-01 | 2026-04-20 |
| T-0010 | NATS JetStream 事件总线改造 | feat | infra | P0 | planned | 架构+运维 | XL | — | R-004 / `prd/infra-event-bus.md`（待产出） | sprint-02..04 | 2026-04-20 |
| T-0011 | F04 告警 Webhook 通道 | feat | F04 | P0 | planned | 电信+Go | M | T-0010 | R-001 / `prd/F04-alarm-notification.md` | sprint-02 | 2026-04-20 |
| T-0012 | worker 进程重试 / 死信队列 | feat | infra | P1 | planned | 架构+运维 | L | T-0010 | R-106 | sprint-02..04 | 2026-04-20 |
| T-0013 | F08 SNMP Trap 骨架 | feat | F08 | P0 | planned | PM+架构 | L | — | R-003 / `prd/F08-oss-protocol.md`（待产出） | sprint-03 | 2026-04-20 |
| T-0014 | F04 告警短信通道 | feat | F04 | P0 | planned | 电信+Go | M | T-0009 | R-001 / `prd/F04-alarm-notification.md` | sprint-03 | 2026-04-20 |
| T-0015 | License 容量 / 过期拦截 | feat | F06/license | P1 | planned | 电信 | M | — | R-103 | sprint-03 | 2026-04-20 |
| T-0016 | 前端 Backup 业务逻辑补齐 | feat | frontend | P1 | planned | 前端 | M | — | R-102 | sprint-03 | 2026-04-20 |
| T-0017 | F08 SNMP Trap 联调（staging ≥1 家运营商） | feat | F08 | P0 | planned | PM+架构 | L | T-0013 | R-003 | sprint-04 | 2026-04-20 |
| T-0018 | Software 灰度升级策略 | feat | F06/software | P1 | planned | 电信 | L | — | R-101 | sprint-04..05 | 2026-04-20 |
| T-0019 | 前端 Software 业务逻辑 | feat | frontend | P1 | planned | 前端 | M | T-0018 | R-102 | sprint-04 | 2026-04-20 |
| T-0020 | F08 推送可靠性（重试/去重/幂等） | feat | F08 | P0 | planned | PM+架构 | M | T-0017 | R-003 | sprint-05 | 2026-04-20 |
| T-0021 | Software 回滚能力 | feat | F06/software | P1 | planned | 电信 | M | T-0018 | R-101 | sprint-05 | 2026-04-20 |
| T-0022 | 前端 Topology / Report 补完 | feat | frontend | P1 | planned | 前端 | M | — | — | sprint-05 | 2026-04-20 |
| T-0023 | 压测基线（10K 设备 / 3K 并发）对标 | td | infra | P0 | planned | 架构+运维 | L | T-0010,T-0017,T-0020 | — | sprint-06 | 2026-04-20 |
| T-0024 | Release Gate 完整演练 1 次（staging） | proc | process | P0 | planned | QA | M | T-0023 | `release-gate.md` | sprint-06 | 2026-04-20 |
| T-0025 | RC 冻结 + 冒烟用例集（~20 条） | td | infra | P0 | planned | QA | M | T-0006@累计≥150 | — | sprint-07 | 2026-04-20 |
| T-0026 | Runbook ≥ 5 场景 | docs | ops | P0 | planned | 运维 | M | — | `release-gate.md §3.4` | sprint-07 | 2026-04-20 |

**说明**：
- T-0009 是外部凭据申请，不编码但走流水线（作为前置项，保证 T-0014 不被卡）。
- T-0010 是 XL 任务，按模块切分实际执行（F04 通知 → transfer → F08），分阶段推进但登记为一条。
- T-0006 E2E 是贯穿型任务，不拆成 7 条，每 Sprint 回顾时更新累计数；Sprint 不达标时在 Sprint 回顾里登记。

### 3.1 累计型任务进度（S4 核销读此处）

> **规则**：下游 Task.Deps 形如 `T-xxxx@累计≥N` 时，S4 verify 读本表的 `Progress` 判定达标。
> 上游每 Sprint 回顾 / 合入新 E2E 用例时更新此表。详见设计 §11.7.1。

| Task | Progress | 目标 | 下游依赖 | 下次更新 |
|------|----------|------|----------|---------|
| T-0006（E2E 累计用例） | **累计 0** | 累计 ≥200 | T-0025 要求累计 ≥150 | Sprint-01 回顾（2026-05-04） |

---

## 4. Triaged — 已分诊、未排期（等 Sprint Planning 挑选）

| ID | Title | Type | Domain | Prio | Est | Deps | Risk | Notes |
|----|-------|------|--------|------|-----|------|------|-------|
| T-0027 | 拓扑自动分组规则引擎激活 | feat | F06/topology | P1 | M | — | R-104 | 下次复盘 Sprint-04；当前手工分组凑合 |
| T-0028 | syslog 远程转发（UDP/TCP） | feat | F06/syslog | P1 | M | — | R-105 | 按需实现；与统一日志平台对接时再拉起 |
| T-0029 | RF 控制走 Carrier 适配器（去 LTE 硬编码） | ref | device | P2 | M | — | R-201 | 触发点：接入 5G NR 或 CUCC 差异化时 |
| T-0030 | F10 互操作用例库扩充 | feat | F10 | P2 | L | — | R-202 | GA 级质量补强 |
| T-0031 | CAPTCHA 图形生成实现 | feat | admin | P2 | S | — | R-203 | 当前仅端点占位 |
| T-0032 | FTP 连接测试端点实现 | feat | backup | P2 | S | — | R-204 | 当前返回 "not implemented" |
| T-0035 | 前端多皮肤架构（Phase 3-7：v2 皮肤脚手架 → 18 模块补齐 → 双皮肤部署） | feat | frontend | P2 | XL | T-0035-P1 | — | 方案 `frontend-multi-skin-plan-20260422.md`；Phase 1（`@omc/frontend-core` 抽取 + workspaces）已完成；Phase 3+ 需 Sprint 规划 |

---

## 5. Proposed — 待 Triage

| ID | Title | Proposed By | Created | Notes |
|----|-------|-------------|---------|-------|
| — | （当前为空） | — | — | 新想法请在此行之上追加，下周一 Triage 会议判决 |

---

## 6. Done — 最近 30 天

| ID | Title | Type | Domain | Closed | Closing Evidence |
|----|-------|------|--------|--------|------------------|
| T-0001 | 建立 PM/PgM/QA 三角色 + 流程制品体系 + 流水线门控 | proc | process | 2026-04-20 | commit `b6da1e17` + `docs/project/process-design-20260420.md` + `dod.md` + `release-gate.md` + `risk-register.md` |
| T-0002 | 补 5 个 `reserved_placeholder` 迁移（闭合版本号跳跃） | td | infra | 2026-04-20 | commit `6a6560e1`；`scripts/check-migrations.sh` 通过；**关闭 R-005** |
| T-0003 | 修正 `check-migrations.sh` Down 空判定阈值（撤销 R-108 误报） | bug | infra | 2026-04-20 | commit `36907a36`；**关闭 R-108**（误报） |
| T-0004 | 重构 Claude git 权限（软约束行为准则 + 硬约束 settings.json） | proc | process | 2026-04-20 | commit `7d212ce2`；`.claude/settings.json` 分层 allow/ask/deny |
| T-0005 | 开发流水线 skill 设计 + L0 Backlog | proc | process | 2026-04-20 | commit `eba378d0`（L0 + skill + 设计文档 + CLAUDE.md）+ commit `64607009`（既有制品联通：/commit 四元组 / DoD / Gate / Risk / Milestone）+ commit `1dae3b31`（S7 回写）；设计 `docs/project/dev-pipeline-design-20260420.md`；skill `.claude/commands/dev-pipeline.md`；**首次 dogfooding**：本三连 commit 自身全部走四元组 footer |
| T-0035-P1 | 前端多皮肤架构 Phase 1：抽取 `@omc/frontend-core` + 建立 npm workspaces + `@core/*` 路径别名 | ref | frontend | 2026-04-22 | 方案 `docs/project/frontend-multi-skin-plan-20260422.md`；`omcmb/frontend-core/src/{services,hooks/api,store,types,i18n,mock}`；workspace-level lint（`omcmb/eslint.config.js`）；`tsc --noEmit` / `npm run build` 通过；test/lint 前置失败与迁移无关（baseline 一致） |

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

## 10. 变更日志

| 日期 | 动作 | 条目 | 说明 |
|------|------|------|------|
| 2026-04-20 | 初始化 | 全部 37 条 | 反向索引自 milestone + risk-register + 近期 commit |
| 2026-04-20 | done | T-0001 | 流程体系搭建完成（commit `b6da1e17`） |
| 2026-04-20 | done | T-0002 | R-005 关闭 |
| 2026-04-20 | done | T-0003 | R-108 关闭（误报） |
| 2026-04-20 | done | T-0004 | Claude git 权限分层 |
| 2026-04-20 | in_dev | T-0005 | 流水线 skill 设计进行中 |
| 2026-04-20 | in_dev | T-0006 | E2E 补齐正式启动，Sprint-01 目标首批 20 条 |
| 2026-04-20 | schema 调整 | T-0025 | Deps 由 `T-0006` 改为 `T-0006@累计≥150`（累计型依赖语法，设计 §11.7.1） |
| 2026-04-20 | Owner 补齐 | T-0007..T-0026 | 按 risk-register 角色映射填入（电信/Go/PM/架构/运维/前端/QA），Sprint Planning 时替换为人名 |
| 2026-04-20 | done | T-0005 | S7 回写：commit `eba378d0` + `64607009` + `1dae3b31`（rebase 后 hash）；L0 Backlog + 流水线 skill + 既有制品联通全部完成，首次 dogfooding 成功 |
| 2026-04-22 | 登记 + done | T-0035-P1 | 前端多皮肤架构 Phase 1：抽取 `@omc/frontend-core`（6 业务层目录 `git mv`，141 文件 `@/→@core/*`），建立 npm workspaces + workspace-level ESLint；`tsc`/`build` 通过 |
| 2026-04-22 | 登记 | T-0035 | 前端多皮肤架构 Phase 3-7（v2 皮肤）母任务，Triaged 等 Sprint 规划 |

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

## 12. 待办（Bootstrap 遗留）

初版仅把 P0/P1 主干登入，以下为后续 7 天内完善事项：

- [ ] 为 T-0007 / T-0010 / T-0013（P0 且复杂）建 `docs/project/backlog/T-NNNN-*.md` 详情文件
- [x] ~~补齐所有 Active 条目的 `Owner` 字段~~ — 2026-04-20 已填**角色级**占位（电信/Go/PM/架构/运维/前端/QA），Sprint Planning 时由 PgM 替换为具体人名
- [ ] 所有 `待产出` PRD（infra-event-bus / F08-oss-protocol）启动 S0
- [ ] `sprint-01.md` 从 `TEMPLATE.md` 拉一版初稿，引用本表 T-0005~T-0009
- [ ] 每周一 09:30 Triage 例会时段确认
- [ ] `/dev-pipeline status` skill 子命令实现（自动读本表输出仪表盘）
- [ ] 累计型上游 Task（当前仅 T-0006）每 Sprint 回顾更新 `Notes: Progress: 累计 NN`（供下游 S4 核销用，规则见设计 §11.7.1）

---

**当前版本**：v1.2（2026-04-20 Bootstrap Day 1-3；补 Owner 角色占位 + 累计型依赖豁免 + T-0005 S7 回写关闭）
**维护节奏**：每日（状态回写）· 每周一（Triage）· 每 2 周（Sprint Planning + 仪表盘刷新）· 每季度（深度清理）
