# OMC 项目流程控制体系设计（Process Control System）

> **文档日期**：2026-04-20  
> **背景**：基于 `docs/archive/reports/implementation-completeness-report-20260420.md` 暴露的流程短板（E2E 0 用例、功能卡 70-85%、5 个 P0 无人守），引入 PM/PgM/QA 角色 + 流程制品 + 流水线门控的三层体系。  
> **定位**：指导性架构文档，实施细节下沉到各专项制品（dod / release-gate / risk-register / prd / milestone / sprint）。

---

## 1. 设计目标

解决三类流程缺失：

1. **对焦缺失**：需求范围不清 → 功能开发卡在 70-85% 无法收尾
2. **守门缺失**：Definition of Done 未定义 → 声称 452 用例实际 0 无人察觉
3. **协调缺失**：依赖/排期/风险无登记册 → 5 个 P0 短板无 owner、无时间盒

**不是为了堆文档而堆文档**：每一份制品都对应一个具体守门场景；如果没有场景，不产出。

---

## 2. 非目标（明确不做什么）

- ❌ **不引入专职人力 PM/PgM**（规模未到，人月 ROI 不如补 E2E）
- ❌ **不做重型 Scrum/SAFe**（避免仪式化）
- ❌ **不做看板工具强依赖**（GitHub Issues 足够）
- ❌ **不追求 100% 流程合规**（允许 hotfix 走快速通道，但需补 postmortem）

---

## 3. 三层架构总览

```
┌─────────────────────────────────────────────────────────────────┐
│  L1 角色层（AI 角色，0 人力成本）                                │
│  - 产品经理（PM）        - 项目经理（PgM）      - QA/发布经理    │
│  - 激活时机：按"工作阶段"                                        │
│  - 落地位置：CLAUDE.md §16 新增 3 节                             │
└─────────────────────────────────────────────────────────────────┘
                              ⬇ 使用
┌─────────────────────────────────────────────────────────────────┐
│  L2 流程制品层（markdown 文档，AI 生成 + 人工审）                │
│  docs/                                                           │
│   ├── prd/{F01-F10}-*.md     — 按功能域的 PRD                   │
│   ├── milestone/YYYY-QN.md   — 季度/里程碑计划                  │
│   ├── sprint/sprint-NN.md    — 2 周一个冲刺                     │
│   ├── dod.md                 — Definition of Done               │
│   ├── release-gate.md        — 发布门控                         │
│   └── risk-register.md       — 风险登记册                       │
└─────────────────────────────────────────────────────────────────┘
                              ⬇ 执行
┌─────────────────────────────────────────────────────────────────┐
│  L3 流水线门控层（硬约束，不可绕过）                             │
│  .github/                                                        │
│   ├── pull_request_template.md — PR 模板内嵌 DoD 清单           │
│   ├── ISSUE_TEMPLATE/          — feature/bug/tech-debt          │
│   └── workflows/               — CI workflow（后续落地）        │
│  scripts/                                                        │
│   └── check-migrations.sh      — 迁移编号连续性检查             │
│  branch protection：main 需过 CI + 1 审查 + DoD 勾选            │
└─────────────────────────────────────────────────────────────────┘
```

三层必须配套：缺 L1 → 没有专家视角守门；缺 L2 → 没有"做什么"与"算完成"的依据；缺 L3 → 依赖人自觉，必然失守。

---

## 4. L1 角色层设计

### 4.1 现状

`CLAUDE.md §16` 已有 9 个 Expert Personas（架构/Go/TR069/电信/数据/前端/测试/安全/运维），按**文件路径**激活。优势：编码期自动应用领域视角。短板：只覆盖"编码"，没覆盖"决策/规划/发布"。

### 4.2 新增 3 个流程角色

| 角色 | 核心职责 | 激活时机 | 产出物 |
|------|---------|---------|--------|
| **16.10 产品经理（PM）** | 需求定义、用户故事、验收标准、运营商差异矩阵、优先级 | 讨论新需求 / 写 PRD / 范围争议 / 功能验收 | `docs/project/prd/*.md` |
| **16.11 项目经理（PgM）** | 排期、里程碑、依赖图、风险登记、进度同步 | 季度/冲刺规划 / 依赖冲突 / 风险评估 / 进度评审 | `docs/project/milestone/*.md`、`docs/project/sprint/*.md`、`docs/project/risk-register.md` |
| **16.12 QA / 发布经理** | DoD 守门、E2E 用例基线、发布 Gate 核查、回归 | PR review / 发布前 / 测试计划 / 回归评估 | `docs/project/dod.md`、`docs/project/release-gate.md`、CI 阈值 |

### 4.3 激活规则补丁（追加到 §16 激活规则节）

```
提 "需求/功能/范围"        → PM + 架构师
提 "排期/计划/冲刺"        → PgM
提 "发布/RC/GA/回归"       → QA/发布经理 + 运维
提 "风险/阻塞/依赖"        → PgM
docs/project/prd/**               → PM
docs/project/milestone/** sprint/** → PgM
docs/project/dod.md release-gate.md → QA/发布经理
```

原有"按文件路径激活"规则保留不变；新角色增量叠加。

### 4.4 与 9 个领域专家的协作

- PM 定义**做什么** → 领域专家定义**怎么做**
- PgM 定义**何时做、谁做、怎么协同** → 角色间不重复
- QA/发布经理定义**算完成** → 其他角色都要过这道门

---

## 5. L2 流程制品层设计

### 5.1 PRD（产品需求文档）

**路径**：`docs/project/prd/F{NN}-{slug}.md`  
**粒度**：一个功能域一份（F01-F10），子功能（如 F04-alarm-notification）独立成文  
**触发**：引入新功能或重大范围变更时由 PM 产出

**必含七要素**：
1. 业务背景（为什么做）
2. 用户故事（As a... I want... So that...）
3. 验收标准（Given/When/Then，可测的）
4. 运营商差异（CMCC/CTCC/CUCC 各自要求）
5. 非目标（明确不做）
6. 依赖（阻塞项 + 依赖项）
7. 度量（上线后看什么指标证明成功）

**强制约束**：没有 PRD 不得启动开发。

### 5.2 里程碑（Milestone）

**路径**：`docs/project/milestone/YYYY-Q{N}-{codename}.md`  
**粒度**：季度或一个重大发布窗口  
**首份**：`2026Q2-to-RC.md`（14 周 RC 冲刺）

**结构**：
- 里程碑目标（1-3 句话）
- 入选范围（bulleted，带优先级 P0/P1/P2）
- 退出标准（什么情况算达成）
- Sprint 拆解（时间表）
- 风险与缓解（指向 risk-register）

### 5.3 Sprint（冲刺）

**路径**：`docs/project/sprint/sprint-{NN}.md`  
**粒度**：2 周一个  
**生命周期**：规划（周一）→ 执行 → 回顾（第二周周五）

**结构**：
- Sprint 目标（1 句话）
- 承诺项（带 issue 链接）
- 产出（每日/周产出）
- 阻塞 & 风险
- 回顾（What went well / What didn't / Action items）

### 5.4 Definition of Done（DoD）

**路径**：`docs/project/dod.md`  
**性质**：硬约束清单，PR 合入必须每项打勾

**通用 DoD**：
- [ ] 代码通过 `go build ./...` + `go test ./...`
- [ ] 前端通过 `npx tsc --noEmit` + `npm run lint`（如涉及）
- [ ] 新增/修改代码有单元测试（覆盖率不降）
- [ ] E2E 用例已更新（新端点必有）
- [ ] CLAUDE.md 相关章节已更新（如涉及约定变更）
- [ ] 迁移文件编号连续（`scripts/check-migrations.sh` 通过）
- [ ] 无 TODO/FIXME/panic 未关联 issue
- [ ] PR 说明填写了"Why"（不只是"What"）

**模块特定 DoD**：各领域专家在对应章节追加（如 ACS 必须含协议合规检查、前端必须含类型映射）。

### 5.5 Release Gate（发布门控）

**路径**：`docs/project/release-gate.md`  
**触发**：任何 RC/GA/hotfix 发布前

**门控项**：
- [ ] 所有 P0 风险已关闭或有明确缓解
- [ ] 迁移脚本经 staging 环境验证
- [ ] 回滚脚本/步骤已演练
- [ ] 冒烟测试集（~20 核心用例）100% 通过
- [ ] Prometheus 关键告警规则已配置
- [ ] Runbook 已就绪（故障处理/常见问题）
- [ ] 运营商验收用例（如涉及）已过
- [ ] 性能基线无回退（KPI 对比上一版本）

### 5.6 Risk Register（风险登记册）

**路径**：`docs/project/risk-register.md`  
**性质**：活文档，每 sprint 复盘时更新

**字段**：
- ID（R-NNN）
- 描述
- 影响等级（P0/P1/P2）
- 发生概率（高/中/低）
- Owner
- 缓解措施
- 状态（Open/Mitigating/Closed）
- 下次复盘时间

**初始化**：5 个 P0 + 次级短板全部登记。

---

## 6. L3 流水线门控层设计

### 6.1 Pull Request 模板

**文件**：`.github/pull_request_template.md`

强制填写：
- 关联 Issue
- 变更摘要（Why / What）
- DoD 勾选清单（引用 `docs/project/dod.md`）
- 测试证据（单测/E2E/手工步骤）
- 回滚计划（如涉及迁移或不兼容变更）

### 6.2 Issue 模板

**文件**：`.github/ISSUE_TEMPLATE/{feature,bug,tech-debt}.md`

- feature：关联 PRD、验收标准、优先级、工作量估计
- bug：复现步骤、期望 vs 实际、影响面、相关日志
- tech-debt：现状、理想态、ROI、推迟风险

### 6.3 迁移检查脚本

**文件**：`scripts/check-migrations.sh`

检查：
- 编号连续性（当前存在 000010/000015-000018 跳跃）
- `up/down` 配对
- 命名规范（`000NNN_description.sql`）

**接入**：PR CI 必过；本地 `make check` 也调用。

### 6.4 CI Workflow（后续落地）

**文件**：`.github/workflows/ci.yml`（预留，初期手动触发）

阶段：
1. `build`：Go build + 前端 tsc
2. `unit-test`：go test + npm test（如有）
3. `lint`：golangci-lint + eslint
4. `migrate-check`：调用 `check-migrations.sh`
5. `e2e-subset`：核心 ~20 用例（PR 触发）
6. `coverage`：阈值初始 60%，每月 +5% 到 80%

### 6.5 Branch Protection

`main` 分支规则：
- 需 CI 全过
- 需 PR 至少 1 审查
- 禁止 force push
- 禁止绕过 hooks（与 CLAUDE.md §10 一致）

---

## 7. 与 5 个 P0 短板的映射

| P0 短板 | PM 产出 | PgM 产出 | QA/发布 产出 | 首批 Sprint |
|---------|---------|---------|-------------|------------|
| 告警通知链路 | `prd/F04-alarm-notification.md` | Sprint 1-2 | 加冒烟用例、邮件模板测试 | Sprint 1 |
| E2E 用例 = 0 | — | 分批规划（每 sprint +20） | 用例基线 + CI 阈值 | Sprint 1-14（贯穿）|
| OSS 协议栈 | `prd/F08-oss-protocol.md` | 独立 6-10 周投入评估 | 协议合规测试 | Sprint 3-8 |
| NATS JetStream | `prd/infra-event-bus.md` | 依赖链：影响 F04/F08/transfer | 事件回放测试 | Sprint 2-4 |
| 迁移版本跳跃 | — | 一次性修复 | `check-migrations.sh` 接入 CI | Sprint 1 |

---

## 8. 落地路线（4 周启动）

| 周 | 产出 | 负责角色 | 验收 |
|---|------|---------|------|
| **Week 1**（本周） | 扩展 CLAUDE.md + `dod.md` + `release-gate.md` + `risk-register.md` + `prd/F04-alarm-notification.md` + `milestone/2026Q2-to-RC.md` + PR/Issue 模板 + 迁移检查脚本 | 架构师 + PM + QA | 本文档全部制品到位 |
| **Week 2** | 4 个剩余 P0 的 PRD；第一个 Sprint 规划（sprint-01.md） | PM + PgM | Sprint 启动 |
| **Week 3** | CI workflow + branch protection；第一个 Sprint 执行 | QA + PgM | CI 绿；Sprint 按计划推进 |
| **Week 4** | Sprint-01 回顾 + 第一次 Release Gate 演练 | PgM + QA/发布 | 流程闭环跑通 |
| **持续** | 每 2 周一个 Sprint + 每月 DoD 审计 + 每季度 risk-register 复盘 | PgM 驱动 | — |

---

## 9. 度量（流程本身的 KPI）

流程不是目的，目的是缩短"缺陷到发现"的距离。度量以下指标：

| 指标 | 基线 | 目标 |
|------|------|------|
| E2E 用例数 | 0 | Sprint N 后累计 ≥ 20N |
| 未关联 Issue 的 PR 比例 | 未知 | < 10% |
| DoD 所有项勾选的 PR 比例 | 未知 | > 95% |
| Release Gate 未通过次数 | 未知 | 每次发布记录 |
| P0 风险从登记到关闭的平均天数 | 未知 | < 14 天 |
| Sprint 承诺完成率 | 未知 | > 80% |

每月第一个周五自动（或手动）生成度量报告，纳入 `docs/project/milestone/` 归档。

---

## 10. 演进与退出

**演进**：
- 规模扩大（>5 人团队或 >3 运营商并行）时，考虑引入专职 PM/PgM
- 进入 GA 后，Release Gate 需加 SLA 监控、容量测试、安全扫描
- 多地域部署后，补充"灰度发布策略"章节到 release-gate.md

**退出触发条件**（如发现体系不合身）：
- 流程制品长期为空或一份 PRD 写 < 30 分钟（说明模板过重）
- DoD 勾选率 < 50%（说明门控不被尊重）
- 连续 2 个 Sprint 承诺完成率 < 60%（说明规划失准）

**应对**：季度复盘时调整，不追求一次到位。

---

## 11. 与现有约定的关系

| 现有文件 | 关系 |
|---------|------|
| `CLAUDE.md §8 开发规范` | **保留**，是编码细则；新角色章节不重复 |
| `CLAUDE.md §9 开发流程` | **补强**：新增角色后，"实施步骤 1-6" 在 step 1 之前增加 "step 0: 确认 PRD 存在" |
| `CLAUDE.md §10 质量关卡` | **保留**，是 DoD 通用项的来源，在 `dod.md` 中引用 |
| `CLAUDE.md §11 决策框架` | **保留**，PM 做范围取舍时使用 |
| `CLAUDE.md §16 专家角色` | **扩展**：新增 16.10/16.11/16.12 三节 |
| `.claude/commands/{commit,review,e2e}.md` | **保留**，Skill 不变，但 commit/review 内部检查项会引用 `dod.md` |

---

## 12. 附录：Git 操作策略（Claude 行为约束）

配合流程体系，Claude 在本仓库执行 git 操作的策略：

| 操作 | 策略 |
|------|------|
| `git status` / `git diff` / `git log` / `git show` | 自动执行（`.claude/settings.json` allow） |
| `git add` / `git commit` | 自动执行（`.claude/settings.json` allow），但提交消息必须符合 `CLAUDE.md §8.1` Conventional Commits 中文规范 |
| `git push` / `git push --force` / `git push --tags` | **严禁自动执行**。必须由用户明确发出 "推送" / "push" / "上传" / "同步远端" 等指令后才能进行（`.claude/settings.json` ask + CLAUDE.md §8.1 补丁） |
| `git rebase -i` / `git reset --hard` / `git branch -D` | 需用户明确指令（全局默认行为保留） |

这条策略在 `CLAUDE.md §8.1` 与 `.claude/settings.json` 中双重落地，避免误操作污染远端。

---

**文档状态**：v1.0 草案，2026-04-20 落盘。  
**审阅人**：架构师 + PM + QA（任命后）。  
**下次更新**：Sprint-01 回顾后（Week 4）。
